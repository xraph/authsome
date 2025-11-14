package app

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/schema"
)

// MemberService handles member aggregate operations
type MemberService struct {
	repo    MemberRepository
	appRepo AppRepository // For validation (app exists)
	config  Config
	rbacSvc *rbac.Service
}

// NewMemberService creates a new member service
func NewMemberService(repo MemberRepository, appRepo AppRepository, cfg Config, rbacSvc *rbac.Service) *MemberService {
	return &MemberService{
		repo:    repo,
		appRepo: appRepo,
		config:  cfg,
		rbacSvc: rbacSvc,
	}
}

// CreateMember adds a new member to an app
func (s *MemberService) CreateMember(ctx context.Context, member *Member) (*Member, error) {
	if member.ID.IsNil() {
		member.ID = xid.New()
	}
	if member.Status == "" {
		member.Status = MemberStatusActive
	}
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()
	err := s.repo.CreateMember(ctx, member.ToSchema())
	if err != nil {
		return nil, err
	}
	memberSchema, err := s.repo.FindMemberByID(ctx, member.ID)
	if err != nil {
		return nil, err
	}
	return FromSchemaMember(memberSchema), nil
}

// FindMemberByID finds a member by ID
func (s *MemberService) FindMemberByID(ctx context.Context, id xid.ID) (*Member, error) {
	memberSchema, err := s.repo.FindMemberByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return FromSchemaMember(memberSchema), nil
}

// FindMember finds a member by appID and userID
func (s *MemberService) FindMember(ctx context.Context, appID, userID xid.ID) (*Member, error) {
	memberSchema, err := s.repo.FindMember(ctx, appID, userID)
	if err != nil {
		return nil, err
	}
	return FromSchemaMember(memberSchema), nil
}

// ListMembers lists members in an app with pagination and filtering
func (s *MemberService) ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*Member], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	response, err := s.repo.ListMembers(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	// Convert schema members to DTOs
	members := FromSchemaMembers(response.Data)
	return &pagination.PageResponse[*Member]{
		Data:       members,
		Pagination: response.Pagination,
	}, nil
}

// GetUserMemberships retrieves all apps where the user is a member
func (s *MemberService) GetUserMemberships(ctx context.Context, userID xid.ID) ([]*Member, error) {
	schemaMembers, err := s.repo.ListMembersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return FromSchemaMembers(schemaMembers), nil
}

// UpdateMember updates a member
func (s *MemberService) UpdateMember(ctx context.Context, member *Member) error {
	member.UpdatedAt = time.Now()
	return s.repo.UpdateMember(ctx, member.ToSchema())
}

// DeleteMember deletes a member by ID
func (s *MemberService) DeleteMember(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteMember(ctx, id)
}

// CountMembers returns total number of members in an app
func (s *MemberService) CountMembers(ctx context.Context, appID xid.ID) (int, error) {
	return s.repo.CountMembers(ctx, appID)
}

// IsUserMember checks if a user is an active member of an app
func (s *MemberService) IsUserMember(ctx context.Context, appID, userID xid.ID) (bool, error) {
	member, err := s.repo.FindMember(ctx, appID, userID)
	if err != nil {
		return false, nil // User is not a member (or error occurred)
	}
	return member != nil && member.Status == MemberStatusActive, nil
}

// IsOwner checks if a user is the owner of an app
// This is a convenience method that checks the member role directly
func (s *MemberService) IsOwner(ctx context.Context, appID, userID xid.ID) (bool, error) {
	member, err := s.repo.FindMember(ctx, appID, userID)
	if err != nil {
		return false, nil // User is not a member
	}
	return member != nil && member.Role == schema.MemberRoleOwner && member.Status == schema.MemberStatusActive, nil
}

// IsAdmin checks if a user is an admin or owner of an app
// This is a convenience method that checks the member role directly
func (s *MemberService) IsAdmin(ctx context.Context, appID, userID xid.ID) (bool, error) {
	member, err := s.repo.FindMember(ctx, appID, userID)
	if err != nil {
		return false, nil // User is not a member
	}
	if member == nil {
		return false, nil
	}
	isAdminOrOwner := (member.Role == schema.MemberRoleOwner || member.Role == schema.MemberRoleAdmin) &&
		member.Status == schema.MemberStatusActive
	return isAdminOrOwner, nil
}

// RequireOwner checks if a user is the owner of an app and returns an error if not
// For RBAC-based checks, use CheckPermission/RequirePermission from rbac.go instead with specific actions
func (s *MemberService) RequireOwner(ctx context.Context, appID, userID xid.ID) error {
	isOwner, err := s.IsOwner(ctx, appID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return NotOwner()
	}
	return nil
}

// RequireAdmin checks if a user is an admin or owner of an app and returns an error if not
// For RBAC-based checks, use CheckPermission/RequirePermission from rbac.go instead with specific actions
func (s *MemberService) RequireAdmin(ctx context.Context, appID, userID xid.ID) error {
	isAdmin, err := s.IsAdmin(ctx, appID, userID)
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
