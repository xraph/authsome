// Package rbac provides role-based access control for AuthSome.
//
// Roles contain named sets of permissions (action + resource pairs).
// Users are assigned roles, optionally scoped to an organization.
// The HasPermission check walks the user's role assignments to determine access.
//
// When Warden is available, all RBAC operations delegate to the Warden
// authorization engine. The types in this package serve as the API facade.
package rbac

import (
	"strings"
	"time"

	wardenassign "github.com/xraph/warden/assignment"
	wardenid "github.com/xraph/warden/id"
	wardenperm "github.com/xraph/warden/permission"
	wardenrole "github.com/xraph/warden/role"
)

// envIDToNamespace converts an authsome environment id (e.g. "aenv_01jf...")
// into a warden namespace segment. Warden namespace segments must match
// `^[a-z][a-z0-9-]{0,62}$`; TypeIDs only contain '_' as a separator, so a
// single replacement yields a valid segment and a stable per-env scope.
func envIDToNamespace(envID string) string {
	return strings.ReplaceAll(envID, "_", "-")
}

// Role represents a named collection of permissions.
type Role struct {
	ID          string    `json:"id"`
	AppID       string    `json:"app_id"`
	EnvID       string    `json:"env_id"`
	ParentID    string    `json:"parent_id,omitempty"` // Empty = root role (no parent)
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents a single action on a resource granted by a role.
type Permission struct {
	ID       string `json:"id"`
	RoleID   string `json:"role_id"`
	Action   string `json:"action"`   // e.g. "read", "write", "delete", "admin"
	Resource string `json:"resource"` // e.g. "user", "org", "document"
}

// UserRole represents a role assigned to a user, optionally scoped to an org.
type UserRole struct {
	UserID     string    `json:"user_id"`
	RoleID     string    `json:"role_id"`
	OrgID      string    `json:"org_id,omitempty"` // empty = global scope
	AssignedAt time.Time `json:"assigned_at"`
}

// ──────────────────────────────────────────────────
// Warden type conversions
// ──────────────────────────────────────────────────

// ToWardenRole converts an authsome RBAC Role to a warden Role.
func ToWardenRole(r *Role) *wardenrole.Role {
	wr := &wardenrole.Role{
		TenantID:    r.AppID,
		AppID:       r.AppID,
		Name:        r.Name,
		Slug:        r.Slug,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.ID != "" {
		if wid, err := convertToWardenRoleID(r.ID); err == nil {
			wr.ID = wid
		} else {
			wr.ID = wardenid.NewRoleID()
		}
	} else {
		wr.ID = wardenid.NewRoleID()
	}
	// Parent linkage: warden's role model now uses ParentSlug as a natural
	// key. Authsome still tracks ParentID externally; conversion of
	// ParentID → ParentSlug requires a store lookup that this pure helper
	// can't do. Bootstrap-time parent wiring lives in the warden DSL apply
	// path (bootstrap/warden) which speaks ParentSlug directly. Runtime
	// CRUD callers that need parent linkage should set the parent slug
	// after creation via a follow-up store update.
	// Scope env-owned roles into a per-environment warden namespace so
	// cloning a role with the same slug into a sibling env doesn't collide
	// with the source role's (tenant, ns="", slug) uniqueness key.
	// Bootstrap-seeded roles (no EnvID) stay at the tenant root / "platform"
	// namespace where GetRoleBySlug looks for them.
	if r.EnvID != "" {
		wr.NamespacePath = envIDToNamespace(r.EnvID)
		wr.Metadata = map[string]any{"env_id": r.EnvID}
	}
	return wr
}

// FromWardenRole converts a warden Role to an authsome RBAC Role.
func FromWardenRole(wr *wardenrole.Role) *Role {
	appID := wr.AppID
	if appID == "" {
		appID = wr.TenantID // backward compatibility
	}
	r := &Role{
		ID:          wr.ID.String(),
		AppID:       appID,
		Name:        wr.Name,
		Slug:        wr.Slug,
		Description: wr.Description,
		CreatedAt:   wr.CreatedAt,
		UpdatedAt:   wr.UpdatedAt,
	}
	// Warden surfaces parent linkage via ParentSlug now (was ParentID). We
	// store the slug into authsome's ParentID field as a best-effort hint
	// for callers; downstream code that needs the resolved parent role
	// must look it up by slug.
	r.ParentID = wr.ParentSlug
	// Restore EnvID from warden metadata if present.
	if envID, ok := wr.Metadata["env_id"].(string); ok {
		r.EnvID = envID
	}
	return r
}

// ToWardenPermission converts an authsome RBAC Permission to a warden Permission.
func ToWardenPermission(p *Permission, tenantID string) *wardenperm.Permission {
	wp := &wardenperm.Permission{
		TenantID: tenantID,
		AppID:    tenantID,
		// Warden DSL convention: <resource>:<action>. The check engine
		// matches by (action, resource) tuple, not by name, so this is a
		// cosmetic flip from authsome's old action:resource convention.
		Name:     p.Resource + ":" + p.Action,
		Action:   p.Action,
		Resource: p.Resource,
	}
	if p.ID != "" {
		if wid, err := convertToWardenPermissionID(p.ID); err == nil {
			wp.ID = wid
		} else {
			wp.ID = wardenid.NewPermissionID()
		}
	} else {
		wp.ID = wardenid.NewPermissionID()
	}
	return wp
}

// FromWardenPermission converts a warden Permission to an authsome RBAC Permission.
// The roleID is provided because warden permissions are not directly associated
// with a single role (they are attached via a join table).
func FromWardenPermission(wp *wardenperm.Permission, roleID string) *Permission {
	return &Permission{
		ID:       wp.ID.String(),
		RoleID:   roleID,
		Action:   wp.Action,
		Resource: wp.Resource,
	}
}

// ToWardenAssignment converts an authsome RBAC UserRole to a warden Assignment.
func ToWardenAssignment(ur *UserRole, tenantID string) *wardenassign.Assignment {
	return &wardenassign.Assignment{
		ID:          wardenid.NewAssignmentID(),
		TenantID:    tenantID,
		AppID:       tenantID,
		RoleID:      mustParseWardenRoleID(ur.RoleID),
		SubjectKind: "user",
		SubjectID:   ur.UserID,
		CreatedAt:   ur.AssignedAt,
	}
}

// mustParseWardenRoleID converts and parses a role ID to a warden role ID.
// Handles both authsome ("arol") and warden ("role") prefixes.
func mustParseWardenRoleID(s string) wardenid.RoleID {
	rid, err := convertToWardenRoleID(s)
	if err != nil {
		// Fallback: generate new ID if parse fails (shouldn't happen)
		return wardenid.NewRoleID()
	}
	return rid
}
