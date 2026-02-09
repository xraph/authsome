package rbac

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// ServiceInterface defines the contract for RBAC service operations.
// This allows plugins to decorate the service with additional behavior.
type ServiceInterface interface {
	// ====== Policy Management ======

	// AddPolicy adds a policy to the in-memory policy store
	AddPolicy(p *Policy)

	// AddExpression parses and adds a policy expression to the store
	AddExpression(expression string) error

	// Allowed checks whether any registered policy allows the context
	Allowed(ctx *Context) bool

	// AllowedWithRoles checks policies against a subject plus assigned roles
	AllowedWithRoles(ctx *Context, roles []string) bool

	// LoadPolicies loads and parses all stored policy expressions from a repository
	LoadPolicies(ctx context.Context, repo PolicyRepository) error

	// ====== Role Template Management ======

	// GetRoleTemplates gets all role templates for an app and environment
	GetRoleTemplates(ctx context.Context, appID, envID xid.ID) ([]*schema.Role, error)

	// GetRoleTemplate gets a single role template by ID
	GetRoleTemplate(ctx context.Context, roleID xid.ID) (*schema.Role, error)

	// GetRoleTemplateWithPermissions gets a role template with its permissions loaded
	GetRoleTemplateWithPermissions(ctx context.Context, roleID xid.ID) (*RoleWithPermissions, error)

	// CreateRoleTemplate creates a new role template for an app
	CreateRoleTemplate(ctx context.Context, appID, envID xid.ID, name, displayName, description string, isOwnerRole bool, permissionIDs []xid.ID) (*schema.Role, error)

	// UpdateRoleTemplate updates an existing role template
	UpdateRoleTemplate(ctx context.Context, roleID xid.ID, name, displayName, description string, isOwnerRole bool, permissionIDs []xid.ID) (*schema.Role, error)

	// DeleteRoleTemplate deletes a role template
	DeleteRoleTemplate(ctx context.Context, roleID xid.ID) error

	// GetOwnerRole gets the role marked as the owner role for an app and environment
	GetOwnerRole(ctx context.Context, appID, envID xid.ID) (*schema.Role, error)

	// ====== Organization Role Management ======

	// BootstrapOrgRoles clones selected role templates for a new organization
	BootstrapOrgRoles(ctx context.Context, orgID, appID, envID xid.ID, templateIDs []xid.ID, customizations map[xid.ID]*RoleCustomization) error

	// GetOrgRoles gets all roles specific to an organization and environment
	GetOrgRoles(ctx context.Context, orgID, envID xid.ID) ([]*schema.Role, error)

	// GetOrgRoleWithPermissions gets a role with its permissions loaded
	GetOrgRoleWithPermissions(ctx context.Context, roleID xid.ID) (*RoleWithPermissions, error)

	// UpdateOrgRole updates an organization-specific role
	UpdateOrgRole(ctx context.Context, roleID xid.ID, name, displayName, description string, permissionIDs []xid.ID) error

	// DeleteOrgRole deletes an organization-specific role
	DeleteOrgRole(ctx context.Context, roleID xid.ID) error

	// AssignOwnerRole assigns the owner role to a user in an organization
	AssignOwnerRole(ctx context.Context, userID xid.ID, orgID xid.ID, envID xid.ID) error

	// ====== Permission Management ======

	// GetAppPermissions gets all app-level permissions
	GetAppPermissions(ctx context.Context, appID xid.ID) ([]*schema.Permission, error)

	// GetOrgPermissions gets all org-specific permissions
	GetOrgPermissions(ctx context.Context, orgID xid.ID) ([]*schema.Permission, error)

	// GetPermission gets a permission by ID
	GetPermission(ctx context.Context, permissionID xid.ID) (*schema.Permission, error)

	// GetPermissionsByCategory gets permissions by category
	GetPermissionsByCategory(ctx context.Context, category string, appID xid.ID) ([]*schema.Permission, error)

	// CreateCustomPermission creates a custom permission for an organization
	CreateCustomPermission(ctx context.Context, name, description, category string, orgID xid.ID) (*schema.Permission, error)

	// ====== Role-Permission Management ======

	// AssignPermissionsToRole assigns permissions to a role
	AssignPermissionsToRole(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error

	// RemovePermissionsFromRole removes permissions from a role
	RemovePermissionsFromRole(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error

	// GetRolePermissions gets all permissions for a role
	GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error)

	// ====== Role Assignment Methods ======

	// AssignRoleToUser assigns a single role to a user in an organization
	AssignRoleToUser(ctx context.Context, userID, roleID, orgID xid.ID) error

	// AssignRolesToUser assigns multiple roles to a user in an organization
	AssignRolesToUser(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error

	// AssignRoleToUsers assigns a single role to multiple users in an organization
	AssignRoleToUsers(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (*BulkAssignmentResult, error)

	// AssignAppLevelRole assigns a role at app-level (not org-scoped)
	AssignAppLevelRole(ctx context.Context, userID, roleID, appID xid.ID) error

	// ====== Role Unassignment Methods ======

	// UnassignRoleFromUser removes a single role from a user in an organization
	UnassignRoleFromUser(ctx context.Context, userID, roleID, orgID xid.ID) error

	// UnassignRolesFromUser removes multiple roles from a user in an organization
	UnassignRolesFromUser(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error

	// UnassignRoleFromUsers removes a single role from multiple users in an organization
	UnassignRoleFromUsers(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (*BulkAssignmentResult, error)

	// ClearUserRolesInOrg removes all roles from a user in an organization
	ClearUserRolesInOrg(ctx context.Context, userID, orgID xid.ID) error

	// ClearUserRolesInApp removes all roles from a user in an app
	ClearUserRolesInApp(ctx context.Context, userID, appID xid.ID) error

	// ====== Role Transfer Methods ======

	// TransferUserRoles moves roles from one org to another
	TransferUserRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error

	// CopyUserRoles duplicates roles from one org to another
	CopyUserRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error

	// ReplaceUserRoles atomically replaces all user roles in an org with a new set
	ReplaceUserRoles(ctx context.Context, userID, orgID xid.ID, newRoleIDs []xid.ID) error

	// SyncRolesBetweenOrgs synchronizes roles between organizations
	SyncRolesBetweenOrgs(ctx context.Context, userID xid.ID, config *RoleSyncConfig) error

	// ====== Role Listing Methods ======

	// GetUserRolesInOrg gets all roles (with permissions) for a specific user in an organization
	GetUserRolesInOrg(ctx context.Context, userID, orgID, envID xid.ID) ([]*RoleWithPermissions, error)

	// GetUserRolesInApp gets all roles (with permissions) for a specific user across all orgs in an app
	GetUserRolesInApp(ctx context.Context, userID, appID, envID xid.ID) ([]*RoleWithPermissions, error)

	// ListAllUserRolesInOrg lists all user-role assignments with permissions in an organization (admin view)
	ListAllUserRolesInOrg(ctx context.Context, orgID, envID xid.ID) ([]*UserRoleAssignment, error)

	// ListAllUserRolesInApp lists all user-role assignments with permissions across all orgs in an app (admin view)
	ListAllUserRolesInApp(ctx context.Context, appID, envID xid.ID) ([]*UserRoleAssignment, error)

	// ====== Access Control Methods ======

	// CheckUserAccessInOrg checks if a user can perform an action on a resource in an organization
	CheckUserAccessInOrg(ctx context.Context, userID, orgID, envID xid.ID, action, resource string, cachedRoles []*RoleWithPermissions) (*AccessCheckResult, error)

	// CheckUserAccessInApp checks if a user can perform an action on a resource at app level
	CheckUserAccessInApp(ctx context.Context, userID, appID, envID xid.ID, action, resource string, cachedRoles []*RoleWithPermissions) (*AccessCheckResult, error)

	// ====== Repository Configuration ======

	// SetRepositories sets the repository dependencies
	SetRepositories(
		roleRepo RoleRepository,
		permissionRepo PermissionRepository,
		rolePermissionRepo RolePermissionRepository,
		userRoleRepo UserRoleRepository,
	)
}

// Ensure Service implements ServiceInterface.
var _ ServiceInterface = (*Service)(nil)
