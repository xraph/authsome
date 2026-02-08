package organization

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
	repo     MemberRepository
	orgRepo  OrganizationRepository // For validation (org exists)
	config   Config
	rbacSvc  *rbac.Service
	roleRepo rbac.RoleRepository // For role validation against RBAC definitions
}

// NewMemberService creates a new member service
func NewMemberService(repo MemberRepository, orgRepo OrganizationRepository, cfg Config, rbacSvc *rbac.Service, roleRepo rbac.RoleRepository) *MemberService {
	// Apply defaults for any zero-valued config fields
	if cfg.MaxMembersPerOrganization == 0 {
		cfg.MaxMembersPerOrganization = DefaultConfig().MaxMembersPerOrganization
	}
	if cfg.MaxOrganizationsPerUser == 0 {
		cfg.MaxOrganizationsPerUser = DefaultConfig().MaxOrganizationsPerUser
	}
	if cfg.MaxTeamsPerOrganization == 0 {
		cfg.MaxTeamsPerOrganization = DefaultConfig().MaxTeamsPerOrganization
	}
	if cfg.InvitationExpiryHours == 0 {
		cfg.InvitationExpiryHours = DefaultConfig().InvitationExpiryHours
	}

	return &MemberService{
		repo:     repo,
		orgRepo:  orgRepo,
		config:   cfg,
		rbacSvc:  rbacSvc,
		roleRepo: roleRepo,
	}
}

// =============================================================================
// RBAC Validation Helpers
// =============================================================================

// validateRoleAgainstRBAC validates a role name or ID against RBAC role definitions
// Supports both role IDs (20 chars alphanumeric) and role names
// Falls back to static validation if RBAC repository is not available
//
// Validation rules:
// - Always allows hardcoded roles (owner, admin, member)
// - If AllowAppLevelRoles=true: Allows app-level roles (organization_id IS NULL) and org-specific roles
// - If AllowAppLevelRoles=false: Only allows hardcoded roles
func (s *MemberService) validateRoleAgainstRBAC(ctx context.Context, appID xid.ID, roleInput string) error {
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

	var role *schema.Role
	var err error

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

// checkPermissionByRole provides fallback permission checking based on role hierarchy
// Used when RBAC service is not available
func (s *MemberService) checkPermissionByRole(role, action, resource string) bool {
	// Simple hierarchy-based fallback: owner > admin > member
	switch role {
	case RoleOwner:
		// Owner can do everything
		return true
	case RoleAdmin:
		// Admin can do most things except ownership-level actions
		if action == "delete" && resource == "organization" {
			return false
		}
		return true
	case RoleMember:
		// Member has limited permissions
		return action == "view" || action == "read"
	default:
		return false
	}
}

// =============================================================================
// Member Operations
// =============================================================================

// AddMember adds a user as a member of an organization
func (s *MemberService) AddMember(ctx context.Context, orgID, userID xid.ID, role string) (*Member, error) {
	// Get organization to access appID for role validation
	org, err := s.orgRepo.FindByID(ctx, orgID)
	if err != nil {
		return nil, OrganizationNotFound()
	}

	// Validate role against RBAC definitions
	if err := s.validateRoleAgainstRBAC(ctx, org.AppID, role); err != nil {
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

	// Check permission to update members using RBAC
	contextVars := map[string]string{
		"target_role":    member.Role,
		"target_user_id": member.UserID.String(),
	}
	hasPermission, err := s.CheckPermissionWithContext(ctx, member.OrganizationID, updaterUserID, "update", "members", contextVars)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		// Fallback to legacy admin check if no RBAC policies configured
		isAdmin, err := s.IsAdmin(ctx, member.OrganizationID, updaterUserID)
		if err != nil || !isAdmin {
			return nil, PermissionDenied("update", "members")
		}
	}

	// Update fields
	if req.Role != nil {
		// Get organization to access appID for role validation
		org, err := s.orgRepo.FindByID(ctx, member.OrganizationID)
		if err != nil {
			return nil, OrganizationNotFound()
		}

		// Validate role against RBAC definitions
		if err := s.validateRoleAgainstRBAC(ctx, org.AppID, *req.Role); err != nil {
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

// UpdateMemberRole updates only the role of a member
func (s *MemberService) UpdateMemberRole(ctx context.Context, orgID, memberID xid.ID, newRole string, updaterUserID xid.ID) (*Member, error) {
	member, err := s.repo.FindByID(ctx, memberID)
	if err != nil {
		return nil, MemberNotFound()
	}

	// Verify member belongs to the specified organization
	if member.OrganizationID != orgID {
		return nil, MemberNotFound()
	}

	// Check permission to update member roles using RBAC
	// Pass target member's current role as context for conditional policies
	contextVars := map[string]string{
		"target_role":    member.Role,
		"target_user_id": member.UserID.String(),
		"new_role":       newRole,
	}
	hasPermission, err := s.CheckPermissionWithContext(ctx, orgID, updaterUserID, "update", "member_role", contextVars)
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		// Fallback to legacy admin check if no RBAC policies configured
		isAdmin, err := s.IsAdmin(ctx, orgID, updaterUserID)
		if err != nil || !isAdmin {
			return nil, PermissionDenied("update", "member_role")
		}
	}

	// Get organization to access appID for role validation
	org, err := s.orgRepo.FindByID(ctx, orgID)
	if err != nil {
		return nil, OrganizationNotFound()
	}

	// Validate new role against RBAC definitions
	if err := s.validateRoleAgainstRBAC(ctx, org.AppID, newRole); err != nil {
		return nil, err
	}

	member.Role = newRole
	member.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to update member role: %w", err)
	}

	return member, nil
}

// RemoveMember removes a member from an organization
func (s *MemberService) RemoveMember(ctx context.Context, id, removerUserID xid.ID) error {
	member, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return MemberNotFound()
	}

	// Check permission to remove members using RBAC
	contextVars := map[string]string{
		"target_role":    member.Role,
		"target_user_id": member.UserID.String(),
	}
	hasPermission, err := s.CheckPermissionWithContext(ctx, member.OrganizationID, removerUserID, "delete", "members", contextVars)
	if err != nil {
		return err
	}
	if !hasPermission {
		// Fallback to legacy admin check if no RBAC policies configured
		isAdmin, err := s.IsAdmin(ctx, member.OrganizationID, removerUserID)
		if err != nil || !isAdmin {
			return PermissionDenied("delete", "members")
		}
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

// =============================================================================
// RBAC Permission Methods
// =============================================================================

// CheckPermission checks if a user has permission to perform an action on a resource
// within an organization. Uses the member's role stored in organization_members as
// the single source of truth, and validates against RBAC policy definitions.
func (s *MemberService) CheckPermission(ctx context.Context, orgID, userID xid.ID, action, resource string) (bool, error) {
	// Get member's role from the member table (single source of truth)
	member, err := s.repo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil || member == nil || member.Status != StatusActive {
		return false, nil // Not a member or not active
	}

	// If RBAC service is not available, use fallback role hierarchy
	if s.rbacSvc == nil {
		return s.checkPermissionByRole(member.Role, action, resource), nil
	}

	// Use RBAC service to check permission with the member's role
	// The role stored in member table is passed to AllowedWithRoles
	allowed := s.rbacSvc.AllowedWithRoles(&rbac.Context{
		Subject:  fmt.Sprintf("user:%s", userID.String()),
		Action:   action,
		Resource: resource,
	}, []string{member.Role})

	return allowed, nil
}

// CheckPermissionWithContext checks permission with additional context variables
// for conditional permission evaluation (e.g., resource ownership)
func (s *MemberService) CheckPermissionWithContext(ctx context.Context, orgID, userID xid.ID, action, resource string, contextVars map[string]string) (bool, error) {
	// Get member's role from the member table (single source of truth)
	member, err := s.repo.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil || member == nil || member.Status != StatusActive {
		return false, nil // Not a member or not active
	}

	// If RBAC service is not available, use fallback role hierarchy
	if s.rbacSvc == nil {
		return s.checkPermissionByRole(member.Role, action, resource), nil
	}

	// Use RBAC service with context variables
	allowed := s.rbacSvc.AllowedWithRoles(&rbac.Context{
		Subject:  fmt.Sprintf("user:%s", userID.String()),
		Action:   action,
		Resource: resource,
		Vars:     contextVars,
	}, []string{member.Role})

	return allowed, nil
}

// RequirePermission checks if a user has permission and returns an error if denied
func (s *MemberService) RequirePermission(ctx context.Context, orgID, userID xid.ID, action, resource string) error {
	allowed, err := s.CheckPermission(ctx, orgID, userID, action, resource)
	if err != nil {
		return err
	}
	if !allowed {
		return PermissionDenied(action, resource)
	}
	return nil
}

// Type assertion to ensure MemberService implements MemberOperations
var _ MemberOperations = (*MemberService)(nil)
