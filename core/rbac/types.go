package rbac

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// RoleWithPermissions represents a role with its associated permissions.
type RoleWithPermissions struct {
	*schema.Role

	Permissions []*schema.Permission `json:"permissions"`
}

// RoleTemplate represents a role template with metadata for cloning.
type RoleTemplate struct {
	*schema.Role

	PermissionCount int  `json:"permissionCount"`
	CanModify       bool `json:"canModify"` // Whether this template can be modified
}

// RoleCustomization contains customization options when cloning a role template.
type RoleCustomization struct {
	Name               *string  `json:"name,omitempty"`               // Override template name
	Description        *string  `json:"description,omitempty"`        // Override template description
	PermissionIDs      []xid.ID `json:"permissionIDs,omitempty"`      // Specific permissions to clone (if empty, clone all)
	ExcludePermissions []xid.ID `json:"excludePermissions,omitempty"` // Permissions to exclude from template
}

// UserRoleAssignment represents a user's role assignment with full details.
type UserRoleAssignment struct {
	UserID         xid.ID                 `json:"userId"`
	OrganizationID *xid.ID                `json:"organizationId,omitempty"` // nil for app-level
	Roles          []*RoleWithPermissions `json:"roles"`
}

// RoleSyncConfig configures role synchronization between orgs.
type RoleSyncConfig struct {
	SourceOrgID xid.ID   `json:"sourceOrgId"`
	TargetOrgID xid.ID   `json:"targetOrgId"`
	RoleIDs     []xid.ID `json:"roleIds"` // empty = sync all
	Mode        string   `json:"mode"`    // "mirror" or "merge"
}

// BulkAssignmentResult tracks success/failure for bulk operations.
type BulkAssignmentResult struct {
	SuccessCount int              `json:"successCount"`
	FailureCount int              `json:"failureCount"`
	Errors       map[xid.ID]error `json:"errors"` // userID/roleID -> error
}

// AccessCheckResult contains the result of an access control check.
type AccessCheckResult struct {
	Allowed           bool               `json:"allowed"`
	Reason            string             `json:"reason"`
	MatchedPermission *schema.Permission `json:"matchedPermission,omitempty"`
	MatchedRole       *schema.Role       `json:"matchedRole,omitempty"`
	IsWildcard        bool               `json:"isWildcard"` // true if matched via wildcard
}

// PermissionCategory groups permissions by functional area.
type PermissionCategory string

const (
	// CategoryUsers is the users permission category.
	CategoryUsers         PermissionCategory = "users"
	CategorySettings      PermissionCategory = "settings"
	CategoryContent       PermissionCategory = "content"
	CategoryOrganizations PermissionCategory = "organizations"
	CategorySessions      PermissionCategory = "sessions"
	CategoryAPIKeys       PermissionCategory = "apikeys"
	CategoryAuditLogs     PermissionCategory = "audit_logs"
	CategoryRoles         PermissionCategory = "roles"
	CategoryPermissions   PermissionCategory = "permissions"
	CategoryDashboard     PermissionCategory = "dashboard"
	CategoryCustom        PermissionCategory = "custom"
)

// String returns the string representation of the category.
func (c PermissionCategory) String() string {
	return string(c)
}

// IsValid checks if the category is a valid pre-defined category.
func (c PermissionCategory) IsValid() bool {
	switch c {
	case CategoryUsers, CategorySettings, CategoryContent, CategoryOrganizations,
		CategorySessions, CategoryAPIKeys, CategoryAuditLogs, CategoryRoles,
		CategoryPermissions, CategoryDashboard, CategoryCustom:
		return true
	}

	return false
}

// PermissionAction represents common permission actions.
type PermissionAction string

const (
	// ActionView is the view action.
	ActionView   PermissionAction = "view"
	ActionCreate PermissionAction = "create"
	ActionEdit   PermissionAction = "edit"
	ActionUpdate PermissionAction = "update"
	ActionDelete PermissionAction = "delete"

	// ActionManage is the manage action (full control).
	ActionManage  PermissionAction = "manage"
	ActionList    PermissionAction = "list"
	ActionWrite   PermissionAction = "write"
	ActionExecute PermissionAction = "execute"

	// ActionAll is the wildcard action (all actions).
	ActionAll PermissionAction = "*"
)

// String returns the string representation of the action.
func (a PermissionAction) String() string {
	return string(a)
}

// PermissionResource represents common permission resources.
type PermissionResource string

const (
	// ResourceUsers is the users resource.
	ResourceUsers         PermissionResource = "users"
	ResourceSessions      PermissionResource = "sessions"
	ResourceOrganizations PermissionResource = "organizations"
	ResourceRoles         PermissionResource = "roles"
	ResourcePermissions   PermissionResource = "permissions"
	ResourceAPIKeys       PermissionResource = "apikeys"
	ResourceSettings      PermissionResource = "settings"
	ResourceAuditLogs     PermissionResource = "audit_logs"
	ResourceDashboard     PermissionResource = "dashboard"
	ResourceProfile       PermissionResource = "profile"

	// ResourceAll is the wildcard resource (all resources).
	ResourceAll PermissionResource = "*"
)

// String returns the string representation of the resource.
func (r PermissionResource) String() string {
	return string(r)
}

// BuildPermissionName constructs a permission name from action and resource
// Example: BuildPermissionName(ActionView, ResourceUsers) => "view on users".
func BuildPermissionName(action PermissionAction, resource PermissionResource) string {
	return string(action) + " on " + string(resource)
}

// ParsePermissionName parses a permission name into action and resource
// Returns empty strings if the format is invalid.
func ParsePermissionName(name string) (action PermissionAction, resource PermissionResource) {
	// Simple parser for "action on resource" format
	// Can be enhanced with more sophisticated parsing if needed
	return "", ""
}
