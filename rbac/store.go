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

	// ErrSystemRoleImmutable refuses a change to a role declared
	// is_system, as warden's own contract and REST paths refuse it.
	ErrSystemRoleImmutable = errors.New("rbac: system role cannot be changed")
)

// Store persists RBAC data. All IDs are plain strings — the rbac package
// serves as a DTO facade. Implementations convert to/from their internal
// typed-ID formats as needed.
//
// Every by-ID operation takes the owning app. A call whose app does not own
// the row behaves as if the row did not exist, so holding another tenant's
// ID reads, changes and deletes nothing.
type Store interface {
	// Role CRUD
	CreateRole(ctx context.Context, r *Role) error
	GetRole(ctx context.Context, appID, roleID string) (*Role, error)
	// GetRoles loads several roles in one call; ids that do not resolve
	// are left out rather than failing the whole call.
	GetRoles(ctx context.Context, appID string, roleIDs []string) ([]*Role, error)
	GetRoleBySlug(ctx context.Context, appID string, slug string) (*Role, error)
	UpdateRole(ctx context.Context, r *Role) error
	DeleteRole(ctx context.Context, appID, roleID string) error
	ListRoles(ctx context.Context, appID string) ([]*Role, error)

	// Permission management
	AddPermission(ctx context.Context, appID string, p *Permission) error
	RemovePermission(ctx context.Context, appID, permID string) error
	ListRolePermissions(ctx context.Context, appID, roleID string) ([]*Permission, error)

	// Role assignment
	AssignUserRole(ctx context.Context, appID string, ur *UserRole) error
	UnassignUserRole(ctx context.Context, appID, userID, roleID string) error
	ListUserRoles(ctx context.Context, userID string) ([]*Role, error)
	ListUserRolesForApp(ctx context.Context, appID string, userID string) ([]*Role, error)

	// Hierarchy
	GetRoleChildren(ctx context.Context, appID, roleID string) ([]*Role, error)

	// Permission check — walks the parent chain for inherited permissions.
	HasPermission(ctx context.Context, userID string, action, resource string) (bool, error)
}
