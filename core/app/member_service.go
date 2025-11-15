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
	repo         MemberRepository
	appRepo      AppRepository           // For validation (app exists)
	roleRepo     rbac.RoleRepository     // Access to roles table for RBAC
	userRoleRepo rbac.UserRoleRepository // Access to user_roles table for RBAC
	config       Config
	rbacSvc      *rbac.Service
}

// NewMemberService creates a new member service
func NewMemberService(
	repo MemberRepository,
	appRepo AppRepository,
	roleRepo rbac.RoleRepository, // From core/rbac package
	userRoleRepo rbac.UserRoleRepository, // From core/rbac package
	cfg Config,
	rbacSvc *rbac.Service,
) *MemberService {
	return &MemberService{
		repo:         repo,
		appRepo:      appRepo,
		roleRepo:     roleRepo,
		userRoleRepo: userRoleRepo,
		config:       cfg,
		rbacSvc:      rbacSvc,
	}
}

// =============================================================================
// RBAC Synchronization Helpers
// =============================================================================

// getRoleIDByName looks up the Role ID from the roles table
func (s *MemberService) getRoleIDByName(ctx context.Context, appID xid.ID, roleName string) (xid.ID, error) {
	// Map member role names to RBAC role names
	rbacRoleName := s.mapMemberRoleToRBAC(roleName)

	// Query roles table
	role, err := s.roleRepo.FindByNameAndApp(ctx, rbacRoleName, appID)
	if err != nil {
		return xid.NilID(), fmt.Errorf("role %s not found for app %s: %w", rbacRoleName, appID, err)
	}
	return role.ID, nil
}

// mapMemberRoleToRBAC maps member role strings to RBAC role constants
func (s *MemberService) mapMemberRoleToRBAC(memberRole string) string {
	switch memberRole {
	case string(MemberRoleOwner):
		return rbac.RoleOwner // "owner"
	case string(MemberRoleAdmin):
		return rbac.RoleAdmin // "admin"
	case string(MemberRoleMember):
		return rbac.RoleMember // "member"
	default:
		return rbac.RoleMember // Default to member
	}
}

// syncRoleToRBAC creates/updates UserRole entry to match member role
func (s *MemberService) syncRoleToRBAC(ctx context.Context, userID, appID xid.ID, roleName string) error {
	// Get RBAC role ID
	roleID, err := s.getRoleIDByName(ctx, appID, roleName)
	if err != nil {
		return err
	}

	// Remove any existing role assignments for this user in this app
	// (A user should only have one role per app as a member)
	existingRoles, err := s.userRoleRepo.ListRolesForUser(ctx, userID, &appID)
	if err != nil {
		return fmt.Errorf("failed to check existing roles: %w", err)
	}

	for _, existingRole := range existingRoles {
		if err := s.userRoleRepo.Unassign(ctx, userID, existingRole.ID, appID); err != nil {
			return fmt.Errorf("failed to unassign old role: %w", err)
		}
	}

	// Assign new role
	return s.userRoleRepo.Assign(ctx, userID, roleID, appID)
}

// assignSuperAdminRole assigns the superadmin role to a user in addition to their member role
// This gives them full system access as the platform owner
func (s *MemberService) assignSuperAdminRole(ctx context.Context, userID, appID xid.ID) error {
	// Get superadmin role ID
	superAdminRole, err := s.roleRepo.FindByNameAndApp(ctx, rbac.RoleSuperAdmin, appID)
	if err != nil {
		return fmt.Errorf("superadmin role not found: %w", err)
	}

	// Assign superadmin role (in addition to owner role from syncRoleToRBAC)
	if err := s.userRoleRepo.Assign(ctx, userID, superAdminRole.ID, appID); err != nil {
		return fmt.Errorf("failed to assign superadmin role: %w", err)
	}

	return nil
}

// =============================================================================
// Member Operations
// =============================================================================

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

	// 🔥 FIRST USER DETECTION: Check if this is the first member in the app
	// The first member should always be promoted to owner
	memberCount, err := s.repo.CountMembers(ctx, member.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to check member count: %w", err)
	}

	isFirstMember := memberCount == 0
	if isFirstMember {
		fmt.Printf("[MemberService] First member detected in app %s - promoting to owner: %s\n",
			member.AppID, member.UserID)
		member.Role = MemberRoleOwner
	}

	// Create member record
	err = s.repo.CreateMember(ctx, member.ToSchema())
	if err != nil {
		return nil, fmt.Errorf("failed to create member: %w", err)
	}

	// 🔥 SYNC: Create corresponding UserRole entry in RBAC system
	if err := s.syncRoleToRBAC(ctx, member.UserID, member.AppID, string(member.Role)); err != nil {
		// Rollback member creation on failure
		s.repo.DeleteMember(ctx, member.ID)
		return nil, fmt.Errorf("failed to sync role to RBAC: %w", err)
	}

	// 🔥 SUPERADMIN PROMOTION: If this is the first member of the platform app,
	// also assign them the superadmin role for full system access
	if isFirstMember {
		platformApp, err := s.appRepo.GetPlatformApp(ctx)
		if err == nil && platformApp.ID == member.AppID {
			fmt.Printf("[MemberService] First member of platform app - promoting to superadmin: %s\n",
				member.UserID)
			if err := s.assignSuperAdminRole(ctx, member.UserID, member.AppID); err != nil {
				fmt.Printf("[MemberService] Warning: Failed to assign superadmin role: %v\n", err)
				// Don't fail - user is still owner which is highly privileged
			} else {
				fmt.Printf("[MemberService] Successfully assigned superadmin role\n")
			}
		}
	}

	// Retrieve created member
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

	// Get existing member to check if role changed
	existing, err := s.repo.FindMemberByID(ctx, member.ID)
	if err != nil {
		return fmt.Errorf("failed to find existing member: %w", err)
	}

	// Update member record
	if err := s.repo.UpdateMember(ctx, member.ToSchema()); err != nil {
		return err
	}

	// 🔥 SYNC: If role changed, update RBAC
	if existing.Role != member.Role {
		if err := s.syncRoleToRBAC(ctx, member.UserID, member.AppID, string(member.Role)); err != nil {
			return fmt.Errorf("failed to sync role change to RBAC: %w", err)
		}
	}

	return nil
}

// DeleteMember deletes a member by ID
func (s *MemberService) DeleteMember(ctx context.Context, id xid.ID) error {
	// Get member info before deletion
	member, err := s.repo.FindMemberByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find member: %w", err)
	}

	// Delete member record
	if err := s.repo.DeleteMember(ctx, id); err != nil {
		return err
	}

	// 🔥 SYNC: Remove RBAC roles for this user in this app
	roles, err := s.userRoleRepo.ListRolesForUser(ctx, member.UserID, &member.AppID)
	if err != nil {
		// Log but don't fail - member is already deleted
		fmt.Printf("Warning: failed to list roles for cleanup: %v\n", err)
		return nil
	}

	for _, role := range roles {
		if err := s.userRoleRepo.Unassign(ctx, member.UserID, role.ID, member.AppID); err != nil {
			fmt.Printf("Warning: failed to unassign role %s: %v\n", role.Name, err)
		}
	}

	return nil
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
