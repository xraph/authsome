package rbac

import (
	"context"
	"errors"
)

// Store errors.
var (
	ErrRoleNotFound        = errors.New("rbac: role not found")
	ErrPermissionNotFound  = errors.New("rbac: permission not found")
	ErrRoleAlreadyAssigned = errors.New("rbac: role already assigned")
	ErrCyclicHierarchy     = errors.New("rbac: cyclic role hierarchy detected")
)

// Store persists RBAC data. All IDs are plain strings — the rbac package
// serves as a DTO facade. Implementations convert to/from their internal
// typed-ID formats as needed.
type Store interface {
	// Role CRUD
	CreateRole(ctx context.Context, r *Role) error
	GetRole(ctx context.Context, roleID string) (*Role, error)
	GetRoleBySlug(ctx context.Context, appID string, slug string) (*Role, error)
	UpdateRole(ctx context.Context, r *Role) error
	DeleteRole(ctx context.Context, roleID string) error
	ListRoles(ctx context.Context, appID string) ([]*Role, error)

	// Permission management
	AddPermission(ctx context.Context, p *Permission) error
	RemovePermission(ctx context.Context, permID string) error
	ListRolePermissions(ctx context.Context, roleID string) ([]*Permission, error)

	// Role assignment
	AssignUserRole(ctx context.Context, ur *UserRole) error
	UnassignUserRole(ctx context.Context, userID string, roleID string) error
	ListUserRoles(ctx context.Context, userID string) ([]*Role, error)
	ListUserRolesForApp(ctx context.Context, appID string, userID string) ([]*Role, error)

	// Hierarchy
	GetRoleChildren(ctx context.Context, roleID string) ([]*Role, error)

	// Permission check — walks the parent chain for inherited permissions.
	HasPermission(ctx context.Context, userID string, action, resource string) (bool, error)
}
