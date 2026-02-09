package apikey

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// RoleRepository interface for RBAC operations on API keys
// This is implemented by repository.APIKeyRoleRepository.
type RoleRepository interface {
	// Role assignment
	AssignRole(ctx context.Context, apiKeyID, roleID xid.ID, orgID *xid.ID, createdBy *xid.ID) error
	UnassignRole(ctx context.Context, apiKeyID, roleID xid.ID, orgID *xid.ID) error
	BulkAssignRoles(ctx context.Context, apiKeyID xid.ID, roleIDs []xid.ID, orgID *xid.ID, createdBy *xid.ID) error
	BulkUnassignRoles(ctx context.Context, apiKeyID xid.ID, roleIDs []xid.ID, orgID *xid.ID) error
	ReplaceRoles(ctx context.Context, apiKeyID xid.ID, roleIDs []xid.ID, orgID *xid.ID, createdBy *xid.ID) error

	// Role queries
	GetRoles(ctx context.Context, apiKeyID xid.ID, orgID *xid.ID) ([]*schema.Role, error)
	HasRole(ctx context.Context, apiKeyID, roleID xid.ID, orgID *xid.ID) (bool, error)
	GetAPIKeysWithRole(ctx context.Context, roleID xid.ID, orgID *xid.ID) ([]*schema.APIKey, error)

	// Permission queries
	GetPermissions(ctx context.Context, apiKeyID xid.ID, orgID *xid.ID) ([]*schema.Permission, error)

	// Creator permissions (for delegation)
	GetCreatorPermissions(ctx context.Context, creatorID xid.ID, orgID *xid.ID) ([]*schema.Permission, error)
	GetCreatorRoles(ctx context.Context, creatorID xid.ID, orgID *xid.ID) ([]*schema.Role, error)
}

// Role represents an RBAC role (simplified DTO).
type Role struct {
	ID          xid.ID `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Permission represents an RBAC permission (simplified DTO).
type Permission struct {
	ID       xid.ID `json:"id"`
	Action   string `json:"action"`           // e.g., "view", "edit", "delete", "*"
	Resource string `json:"resource"`         // e.g., "users", "posts", "*"
	Source   string `json:"source,omitempty"` // "key", "creator", "impersonation"
}

// EffectivePermissions represents all permissions that apply to an API key.
type EffectivePermissions struct {
	Scopes               []string      `json:"scopes"`                      // Legacy scope strings
	Permissions          []*Permission `json:"permissions"`                 // RBAC permissions
	DelegatedFromCreator bool          `json:"delegatedFromCreator"`        // If creator permissions are included
	ImpersonatingUser    *xid.ID       `json:"impersonatingUser,omitempty"` // If impersonating a user
}

// HasPermission checks if effective permissions include a specific action/resource.
func (ep *EffectivePermissions) HasPermission(action, resource string) bool {
	for _, perm := range ep.Permissions {
		if matchPermission(perm, action, resource) {
			return true
		}
	}

	return false
}

// HasScope checks if effective permissions include a specific scope.
func (ep *EffectivePermissions) HasScope(scope string) bool {
	for _, s := range ep.Scopes {
		if s == scope || s == "admin:full" {
			return true
		}
	}

	return false
}

// CanAccess checks if effective permissions allow access (scopes OR RBAC).
func (ep *EffectivePermissions) CanAccess(action, resource string) bool {
	// Check scopes first (legacy)
	scopeString := resource + ":" + action
	if ep.HasScope(scopeString) {
		return true
	}

	// Check RBAC permissions
	return ep.HasPermission(action, resource)
}

func matchPermission(perm *Permission, action, resource string) bool {
	// Wildcard matching
	if perm.Action == "*" && perm.Resource == "*" {
		return true // Full admin
	}

	if perm.Action == "*" && perm.Resource == resource {
		return true // All actions on resource
	}

	if perm.Action == action && perm.Resource == "*" {
		return true // Specific action on all resources
	}

	if perm.Action == action && perm.Resource == resource {
		return true // Exact match
	}

	return false
}
