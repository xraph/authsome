package rbac

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// PolicyRepository provides access to stored policy expressions
type PolicyRepository interface {
	// ListAll returns all stored policy expressions
	ListAll(ctx context.Context) ([]string, error)
	// Create stores a new policy expression
	Create(ctx context.Context, expression string) error
}

// RoleRepository handles role operations for RBAC
type RoleRepository interface {
	Create(ctx context.Context, role *schema.Role) error
	Update(ctx context.Context, role *schema.Role) error
	Delete(ctx context.Context, roleID xid.ID) error
	FindByID(ctx context.Context, roleID xid.ID) (*schema.Role, error)
	FindByNameAndApp(ctx context.Context, name string, appID xid.ID) (*schema.Role, error)
	ListByOrg(ctx context.Context, orgID *string) ([]schema.Role, error)
	
	// Template operations
	GetRoleTemplates(ctx context.Context, appID xid.ID) ([]*schema.Role, error)
	GetOwnerRole(ctx context.Context, appID xid.ID) (*schema.Role, error)
	
	// Organization-scoped roles
	GetOrgRoles(ctx context.Context, orgID xid.ID) ([]*schema.Role, error)
	GetOrgRoleWithPermissions(ctx context.Context, roleID xid.ID) (*schema.Role, error)
	
	// Role cloning
	CloneRole(ctx context.Context, templateID xid.ID, orgID xid.ID, customName *string) (*schema.Role, error)
}

// PermissionRepository handles permission operations for RBAC
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

// RolePermissionRepository handles role-permission relationships
type RolePermissionRepository interface {
	AssignPermission(ctx context.Context, roleID, permissionID xid.ID) error
	UnassignPermission(ctx context.Context, roleID, permissionID xid.ID) error
	GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*schema.Permission, error)
	GetPermissionRoles(ctx context.Context, permissionID xid.ID) ([]*schema.Role, error)
	ReplaceRolePermissions(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error
}

// UserRoleRepository handles user-role assignments for RBAC
type UserRoleRepository interface {
	Assign(ctx context.Context, userID, roleID, orgID xid.ID) error
	Unassign(ctx context.Context, userID, roleID, orgID xid.ID) error
	ListRolesForUser(ctx context.Context, userID xid.ID, orgID *xid.ID) ([]schema.Role, error)
}
