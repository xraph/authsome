package rbac

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// PolicyRepository provides access to stored policy expressions.
type PolicyRepository interface {
	// ListAll returns all stored policy expressions
	ListAll(ctx context.Context) ([]string, error)
	// Create stores a new policy expression
	Create(ctx context.Context, expression string) error
}

// RoleRepository handles role operations for RBAC.
type RoleRepository interface {
	Create(ctx context.Context, role *schema.Role) error
	Update(ctx context.Context, role *schema.Role) error
	Delete(ctx context.Context, roleID xid.ID) error
	FindByID(ctx context.Context, roleID xid.ID) (*schema.Role, error)
	FindByNameAndApp(ctx context.Context, name string, appID xid.ID) (*schema.Role, error)
	FindByNameAppEnv(ctx context.Context, name string, appID, envID xid.ID) (*schema.Role, error)
	ListByOrg(ctx context.Context, orgID *string) ([]schema.Role, error)

	// Template operations
	GetRoleTemplates(ctx context.Context, appID, envID xid.ID) ([]*schema.Role, error)
	GetOwnerRole(ctx context.Context, appID, envID xid.ID) (*schema.Role, error)

	// Organization-scoped roles
	GetOrgRoles(ctx context.Context, orgID, envID xid.ID) ([]*schema.Role, error)
	GetOrgRoleWithPermissions(ctx context.Context, roleID xid.ID) (*schema.Role, error)

	// Role cloning
	CloneRole(ctx context.Context, templateID xid.ID, orgID xid.ID, customName *string) (*schema.Role, error)

	// Migration helpers
	FindDuplicateRoles(ctx context.Context) ([]schema.Role, error)
}

// PermissionRepository handles permission operations for RBAC.
type PermissionRepository interface {
	Create(ctx context.Context, permission *schema.Permission) error
	Update(ctx context.Context, permission *schema.Permission) error
	Delete(ctx context.Context, permissionID xid.ID) error
	FindByID(ctx context.Context, permissionID xid.ID) (*schema.Permission, error)
	FindByName(ctx context.Context, name string, appID xid.ID, orgID *xid.ID) (*schema.Permission, error)
	ListByApp(ctx context.Context, appID xid.ID) ([]*schema.Permission, error)
	ListByOrg(ctx context.Context, orgID xid.ID) ([]*schema.Permission, error)
	ListByCategory(ctx context.Context, category string, appID xid.ID) ([]*schema.Permission, error)

	// Custom permissions
	CreateCustomPermission(ctx context.Context, name, description, category string, orgID xid.ID) (*schema.Permission, error)
}

// RolePermissionRepository handles role-permission relationships.
type RolePermissionRepository interface {
	AssignPermission(ctx context.Context, roleID, permissionID xid.ID) error
	UnassignPermission(ctx context.Context, roleID, permissionID xid.ID) error
	GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error)
	GetPermissionRoles(ctx context.Context, permissionID xid.ID) ([]*schema.Role, error)
	ReplaceRolePermissions(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error
}

// UserRoleRepository handles user-role assignments for RBAC.
type UserRoleRepository interface {
	// Single assignment (legacy)
	Assign(ctx context.Context, userID, roleID, orgID xid.ID) error
	Unassign(ctx context.Context, userID, roleID, orgID xid.ID) error
	ListRolesForUser(ctx context.Context, userID xid.ID, orgID *xid.ID) ([]schema.Role, error)

	// ====== Assignment Methods ======
	// AssignBatch assigns multiple roles to a single user in an organization
	AssignBatch(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error
	// AssignBulk assigns a single role to multiple users in an organization
	AssignBulk(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (map[xid.ID]error, error)
	// AssignAppLevel assigns a role at app-level (not org-scoped)
	AssignAppLevel(ctx context.Context, userID, roleID, appID xid.ID) error

	// ====== Unassignment Methods ======
	// UnassignBatch removes multiple roles from a single user in an organization
	UnassignBatch(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error
	// UnassignBulk removes a single role from multiple users in an organization
	UnassignBulk(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (map[xid.ID]error, error)
	// ClearUserRolesInOrg removes all roles from a user in an organization
	ClearUserRolesInOrg(ctx context.Context, userID, orgID xid.ID) error
	// ClearUserRolesInApp removes all roles from a user in an app
	ClearUserRolesInApp(ctx context.Context, userID, appID xid.ID) error

	// ====== Transfer/Move Methods ======
	// TransferRoles moves roles from one org to another (delete + insert)
	TransferRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error
	// CopyRoles duplicates roles from one org to another (insert only)
	CopyRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error
	// ReplaceUserRoles atomically replaces all user roles in an org with a new set
	ReplaceUserRoles(ctx context.Context, userID, orgID xid.ID, newRoleIDs []xid.ID) error

	// ====== Listing Methods ======
	// ListRolesForUserInOrg gets roles for a specific user in an organization with environment filter
	ListRolesForUserInOrg(ctx context.Context, userID, orgID, envID xid.ID) ([]schema.Role, error)
	// ListRolesForUserInApp gets roles for a specific user across all orgs in an app with environment filter
	ListRolesForUserInApp(ctx context.Context, userID, appID, envID xid.ID) ([]schema.Role, error)
	// ListAllUserRolesInOrg lists all user-role assignments in an organization (admin view)
	ListAllUserRolesInOrg(ctx context.Context, orgID, envID xid.ID) ([]schema.UserRole, error)
	// ListAllUserRolesInApp lists all user-role assignments in an app across all orgs (admin view)
	ListAllUserRolesInApp(ctx context.Context, appID, envID xid.ID) ([]schema.UserRole, error)
}
