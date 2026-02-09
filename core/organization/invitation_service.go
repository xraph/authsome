package organization

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/internal/utils"
	"github.com/xraph/authsome/schema"
)

// InvitationService handles invitation lifecycle operations.
type InvitationService struct {
	repo       InvitationRepository
	memberRepo MemberRepository       // For creating members (cross-aggregate)
	orgRepo    OrganizationRepository // For validation
	config     Config
	rbacSvc    *rbac.Service
	roleRepo   rbac.RoleRepository // For RBAC role validation
}

// NewInvitationService creates a new invitation service.
func NewInvitationService(repo InvitationRepository, memberRepo MemberRepository, orgRepo OrganizationRepository, cfg Config, rbacSvc *rbac.Service, roleRepo rbac.RoleRepository) *InvitationService {
	return &InvitationService{
		repo:       repo,
		memberRepo: memberRepo,
		orgRepo:    orgRepo,
		config:     cfg,
		rbacSvc:    rbacSvc,
		roleRepo:   roleRepo,
	}
}

// validateRoleAgainstRBAC validates a role name or ID against RBAC role definitions
// Supports both role IDs (20 chars alphanumeric) and role names
// Falls back to static validation if RBAC repository is not available
//
// Validation rules:
// - Always allows hardcoded roles (owner, admin, member)
// - If AllowAppLevelRoles=true: Allows app-level roles (organization_id IS NULL) and org-specific roles
// - If AllowAppLevelRoles=false: Only allows hardcoded roles.
func (s *InvitationService) validateRoleAgainstRBAC(ctx context.Context, appID xid.ID, roleInput string) error {
	if s.roleRepo == nil {
		// Fallback to static validation if no RBAC role repository
		return validateRole(roleInput)
	}

	// Check if roleInput is a hardcoded role name (always allowed)
	if roleInput == RoleOwner || roleInput == RoleAdmin || roleInput == RoleMember {
		return nil
	}

	// Detect if input is a role ID (20 chars alphanumeric) or a role name
	isRoleID := len(roleInput) == 20 && isAlphanumeric(roleInput)

	var (
		role *schema.Role
		err  error
	)

	if isRoleID {
		// Try to find role by ID
		roleID, parseErr := xid.FromString(roleInput)
		if parseErr != nil {
			return InvalidRole(roleInput)
		}

		role, err = s.roleRepo.FindByID(ctx, roleID)
		if err != nil {
			return InvalidRole(roleInput)
		}

		// Verify role belongs to this app
		if role.AppID == nil || *role.AppID != appID {
			return InvalidRole(roleInput)
		}
	} else {
		// Try to find role by name within the app
		role, err = s.roleRepo.FindByNameAndApp(ctx, roleInput, appID)
		if err != nil {
			return InvalidRole(roleInput)
		}
	}

	// Check if it's a hardcoded role by name (always allowed)
	if role.Name == RoleOwner || role.Name == RoleAdmin || role.Name == RoleMember {
		return nil
	}

	// Apply scoping rules based on AllowAppLevelRoles config
	if s.config.AllowAppLevelRoles {
		// Allow app-level roles (organization_id IS NULL) or org-specific roles
		// All roles at this point are valid since they belong to the app
		return nil
	}

	// If AllowAppLevelRoles is false, only hardcoded roles are allowed
	// We've already checked hardcoded roles above, so reject all custom roles
	return InvalidRoleWithHint(roleInput, "custom roles disabled - enable AllowAppLevelRoles config to use app-level RBAC roles")
}

// InviteMember creates an invitation for a user to join an organization.
func (s *InvitationService) InviteMember(ctx context.Context, orgID xid.ID, req *InviteMemberRequest, inviterUserID xid.ID) (*Invitation, error) {
	// Get organization to access appID for role validation
	org, err := s.orgRepo.FindByID(ctx, orgID)
	if err != nil {
		return nil, OrganizationNotFound()
	}

	// Validate role against RBAC definitions
	if err := s.validateRoleAgainstRBAC(ctx, org.AppID, req.Role); err != nil {
		return nil, err
	}

	// Verify inviter is admin or owner
	inviter, err := s.memberRepo.FindByUserAndOrg(ctx, inviterUserID, orgID)
	if err != nil || inviter == nil {
		return nil, NotAdmin()
	}

	if inviter.Role != RoleOwner && inviter.Role != RoleAdmin {
		return nil, NotAdmin()
	}

	// Generate secure token
	token, err := utils.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	now := time.Now().UTC()
	invitation := &Invitation{
		ID:             xid.New(),
		OrganizationID: orgID,
		Email:          req.Email,
		Role:           req.Role,
		Token:          token,
		Status:         InvitationStatusPending,
		InviterID:      inviterUserID,
		ExpiresAt:      now.Add(time.Duration(s.config.InvitationExpiryHours) * time.Hour),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, invitation); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	return invitation, nil
}

// FindInvitationByID retrieves an invitation by its ID.
func (s *InvitationService) FindInvitationByID(ctx context.Context, id xid.ID) (*Invitation, error) {
	invitation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, InvitationNotFound()
	}

	return invitation, nil
}

// FindInvitationByToken retrieves an invitation by its token.
func (s *InvitationService) FindInvitationByToken(ctx context.Context, token string) (*Invitation, error) {
	invitation, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, InvitationNotFound()
	}

	// Check if expired
	if err := validateInvitationExpiry(invitation.ExpiresAt); err != nil {
		invitation.Status = InvitationStatusExpired
		invitation.UpdatedAt = time.Now().UTC()
		_ = s.repo.Update(ctx, invitation)

		return nil, err
	}

	return invitation, nil
}

// ListInvitations retrieves a paginated list of invitations for an organization.
func (s *InvitationService) ListInvitations(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*Invitation], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	return s.repo.ListByOrganization(ctx, filter)
}

// AcceptInvitation accepts an invitation and adds the user to the organization
// This is a cross-aggregate operation: it updates the invitation and creates a member.
func (s *InvitationService) AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*Member, error) {
	invitation, err := s.FindInvitationByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Validate invitation status
	if err := validateInvitationPending(invitation.Status); err != nil {
		return nil, err
	}

	// Check if user is already a member
	existing, err := s.memberRepo.FindByUserAndOrg(ctx, userID, invitation.OrganizationID)
	if err == nil && existing != nil {
		return nil, MemberAlreadyExists(userID.String())
	}

	// Add user as member (cross-aggregate operation)
	now := time.Now().UTC()
	member := &Member{
		ID:             xid.New(),
		OrganizationID: invitation.OrganizationID,
		UserID:         userID,
		Role:           invitation.Role,
		Status:         StatusActive,
		JoinedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	// Update invitation status
	acceptedAt := now
	invitation.Status = InvitationStatusAccepted
	invitation.AcceptedAt = &acceptedAt
	invitation.UpdatedAt = now

	if err := s.repo.Update(ctx, invitation); err != nil {
		// Member was created but invitation update failed - log this inconsistency
		// In a real system, you might want to use a transaction or saga pattern
		return member, fmt.Errorf("member created but failed to update invitation: %w", err)
	}

	return member, nil
}

// DeclineInvitation declines an invitation.
func (s *InvitationService) DeclineInvitation(ctx context.Context, token string) error {
	invitation, err := s.FindInvitationByToken(ctx, token)
	if err != nil {
		return err
	}

	invitation.Status = InvitationStatusDeclined
	invitation.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, invitation)
}

// CancelInvitation cancels a pending invitation (admin/owner only).
func (s *InvitationService) CancelInvitation(ctx context.Context, id, cancellerUserID xid.ID) error {
	invitation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return InvitationNotFound()
	}

	// Verify canceller is admin or owner
	member, err := s.memberRepo.FindByUserAndOrg(ctx, cancellerUserID, invitation.OrganizationID)
	if err != nil || member == nil {
		return NotAdmin()
	}

	if member.Role != RoleOwner && member.Role != RoleAdmin {
		return NotAdmin()
	}

	// Can only cancel pending invitations
	if invitation.Status != InvitationStatusPending {
		return InvitationInvalidStatus(InvitationStatusPending, invitation.Status)
	}

	invitation.Status = InvitationStatusCancelled
	invitation.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, invitation)
}

// ResendInvitation resends an invitation with a new token and updated expiry.
func (s *InvitationService) ResendInvitation(ctx context.Context, id, resenderUserID xid.ID) (*Invitation, error) {
	invitation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, InvitationNotFound()
	}

	// Verify resender is admin or owner
	member, err := s.memberRepo.FindByUserAndOrg(ctx, resenderUserID, invitation.OrganizationID)
	if err != nil || member == nil {
		return nil, NotAdmin()
	}

	if member.Role != RoleOwner && member.Role != RoleAdmin {
		return nil, NotAdmin()
	}

	// Can only resend pending or expired invitations
	if invitation.Status != InvitationStatusPending && invitation.Status != InvitationStatusExpired {
		return nil, InvitationInvalidStatus("pending or expired", invitation.Status)
	}

	// Generate new token
	token, err := utils.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	now := time.Now().UTC()
	invitation.Token = token
	invitation.Status = InvitationStatusPending
	invitation.ExpiresAt = now.Add(time.Duration(s.config.InvitationExpiryHours) * time.Hour)
	invitation.AcceptedAt = nil
	invitation.UpdatedAt = now

	if err := s.repo.Update(ctx, invitation); err != nil {
		return nil, fmt.Errorf("failed to update invitation: %w", err)
	}

	return invitation, nil
}

// CleanupExpiredInvitations removes all expired invitations.
func (s *InvitationService) CleanupExpiredInvitations(ctx context.Context) (int, error) {
	return s.repo.DeleteExpired(ctx)
}

// Type assertion to ensure InvitationService implements InvitationOperations.
var _ InvitationOperations = (*InvitationService)(nil)
