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
	FindByNameAndApp(ctx context.Context, name string, appID xid.ID) (*schema.Role, error)
	ListByOrg(ctx context.Context, orgID *string) ([]schema.Role, error)
}

// UserRoleRepository handles user-role assignments for RBAC
type UserRoleRepository interface {
	Assign(ctx context.Context, userID, roleID, orgID xid.ID) error
	Unassign(ctx context.Context, userID, roleID, orgID xid.ID) error
	ListRolesForUser(ctx context.Context, userID xid.ID, orgID *xid.ID) ([]schema.Role, error)
}
