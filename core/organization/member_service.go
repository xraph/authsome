package organization

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
)

// MemberService handles member aggregate operations
type MemberService struct {
	repo    MemberRepository
	orgRepo OrganizationRepository // For validation (org exists)
	config  Config
	rbacSvc *rbac.Service
}

// NewMemberService creates a new member service
func NewMemberService(repo MemberRepository, orgRepo OrganizationRepository, cfg Config, rbacSvc *rbac.Service) *MemberService {
	return &MemberService{
		repo:    repo,
		orgRepo: orgRepo,
		config:  cfg,
		rbacSvc: rbacSvc,
	}
}

// AddMember adds a user as a member of an organization
func (s *MemberService) AddMember(ctx context.Context, orgID, userID xid.ID, role string) (*Member, error) {
	// Validate role
	if err := validateRole(role); err != nil {
		return nil, err
	}

	// Check if user is already a member
	existing, err := s.repo.FindByUserAndOrg(ctx, userID, orgID)
	if err == nil && existing != nil {
		return nil, MemberAlreadyExists(userID.String())
	}

	// Check member limit
	count, err := s.repo.CountByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}
	if count >= s.config.MaxMembersPerOrganization {
		return nil, MaxMembersReached(s.config.MaxMembersPerOrganization)
	}

	now := time.Now().UTC()
	member := &Member{
		ID:             xid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		Status:         StatusActive,
		JoinedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return member, nil
}

// FindMemberByID retrieves a member by ID
func (s *MemberService) FindMemberByID(ctx context.Context, id xid.ID) (*Member, error) {
	member, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, MemberNotFound()
	}
	return member, nil
}

// FindMember retrieves a member by organization ID and user ID
func (s *MemberService) FindMember(ctx context.Context, orgID, userID xid.ID) (*Member, error) {
	member, err := s.repo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return nil, MemberNotFound()
	}
	return member, nil
}

// ListMembers lists members in an organization with pagination and filtering
func (s *MemberService) ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*Member], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	return s.repo.ListByOrganization(ctx, filter)
}

// UpdateMember updates a member
func (s *MemberService) UpdateMember(ctx context.Context, id xid.ID, req *UpdateMemberRequest, updaterUserID xid.ID) (*Member, error) {
	member, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, MemberNotFound()
	}

	// Verify updater is admin or owner
	isAdmin, err := s.IsAdmin(ctx, member.OrganizationID, updaterUserID)
	if err != nil || !isAdmin {
		return nil, NotAdmin()
	}

	// Cannot change owner role
	if member.Role == RoleOwner {
		return nil, CannotRemoveOwner()
	}

	// Update fields
	if req.Role != nil {
		if err := validateRole(*req.Role); err != nil {
			return nil, err
		}
		member.Role = *req.Role
	}
	if req.Status != nil {
		if err := validateStatus(*req.Status); err != nil {
			return nil, err
		}
		member.Status = *req.Status
	}
	member.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	return member, nil
}

// RemoveMember removes a member from an organization
func (s *MemberService) RemoveMember(ctx context.Context, id, removerUserID xid.ID) error {
	member, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return MemberNotFound()
	}

	// Cannot remove owner
	if member.Role == RoleOwner {
		return CannotRemoveOwner()
	}

	// Verify remover is admin or owner
	isAdmin, err := s.IsAdmin(ctx, member.OrganizationID, removerUserID)
	if err != nil || !isAdmin {
		return NotAdmin()
	}

	return s.repo.Delete(ctx, id)
}

// GetUserMemberships returns all organizations a user is a member of
func (s *MemberService) GetUserMemberships(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Member], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	return s.repo.ListByUser(ctx, userID, filter)
}

// RemoveUserFromAllOrganizations removes a user from all organizations they belong to
func (s *MemberService) RemoveUserFromAllOrganizations(ctx context.Context, userID xid.ID) error {
	// Get all memberships with a large limit
	memberships, err := s.repo.ListByUser(ctx, userID, &pagination.PaginationParams{
		Page:  1,
		Limit: 1000,
	})
	if err != nil {
		return fmt.Errorf("failed to get user memberships: %w", err)
	}

	// Delete each membership
	for _, membership := range memberships.Data {
		if err := s.repo.DeleteByUserAndOrg(ctx, userID, membership.OrganizationID); err != nil {
			return fmt.Errorf("failed to remove membership: %w", err)
		}
	}

	return nil
}

// IsMember checks if a user is a member of an organization
func (s *MemberService) IsMember(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	member, err := s.repo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return false, nil // User is not a member (or error occurred)
	}
	return member != nil && member.Status == StatusActive, nil
}

// IsOwner checks if a user is the owner of an organization
func (s *MemberService) IsOwner(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	member, err := s.repo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return false, nil
	}
	return member != nil && member.Role == RoleOwner && member.Status == StatusActive, nil
}

// IsAdmin checks if a user is an admin or owner of an organization
func (s *MemberService) IsAdmin(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	member, err := s.repo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return false, nil
	}
	return member != nil && (member.Role == RoleOwner || member.Role == RoleAdmin) && member.Status == StatusActive, nil
}

// RequireOwner checks if a user is the owner of an organization and returns an error if not
func (s *MemberService) RequireOwner(ctx context.Context, orgID, userID xid.ID) error {
	isOwner, err := s.IsOwner(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return NotOwner()
	}
	return nil
}

// RequireAdmin checks if a user is an admin or owner of an organization and returns an error if not
func (s *MemberService) RequireAdmin(ctx context.Context, orgID, userID xid.ID) error {
	isAdmin, err := s.IsAdmin(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return NotAdmin()
	}
	return nil
}

// Type assertion to ensure MemberService implements MemberOperations
var _ MemberOperations = (*MemberService)(nil)
