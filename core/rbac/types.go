package rbac

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// RoleWithPermissions represents a role with its associated permissions
type RoleWithPermissions struct {
	*schema.Role
	Permissions []*schema.Permission `json:"permissions"`
}

// RoleTemplate represents a role template with metadata for cloning
type RoleTemplate struct {
	*schema.Role
	PermissionCount int  `json:"permissionCount"`
	CanModify       bool `json:"canModify"` // Whether this template can be modified
}

// RoleCustomization contains customization options when cloning a role template
type RoleCustomization struct {
	Name              *string   `json:"name,omitempty"`              // Override template name
	Description       *string   `json:"description,omitempty"`       // Override template description
	PermissionIDs     []xid.ID  `json:"permissionIDs,omitempty"`     // Specific permissions to clone (if empty, clone all)
	ExcludePermissions []xid.ID `json:"excludePermissions,omitempty"` // Permissions to exclude from template
}

// PermissionCategory groups permissions by functional area
type PermissionCategory string

const (
	// Pre-defined permission categories
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

// String returns the string representation of the category
func (c PermissionCategory) String() string {
	return string(c)
}

// IsValid checks if the category is a valid pre-defined category
func (c PermissionCategory) IsValid() bool {
	switch c {
	case CategoryUsers, CategorySettings, CategoryContent, CategoryOrganizations,
		CategorySessions, CategoryAPIKeys, CategoryAuditLogs, CategoryRoles,
		CategoryPermissions, CategoryDashboard, CategoryCustom:
		return true
	}
	return false
}

// PermissionAction represents common permission actions
type PermissionAction string

const (
	// CRUD actions
	ActionView   PermissionAction = "view"
	ActionCreate PermissionAction = "create"
	ActionEdit   PermissionAction = "edit"
	ActionUpdate PermissionAction = "update"
	ActionDelete PermissionAction = "delete"
	
	// Management actions
	ActionManage   PermissionAction = "manage"   // Full control
	ActionList     PermissionAction = "list"     // List/index
	ActionRead     PermissionAction = "read"     // Read-only
	ActionWrite    PermissionAction = "write"    // Write access
	ActionExecute  PermissionAction = "execute"  // Execute/run
	
	// Special actions
	ActionAll      PermissionAction = "*"        // Wildcard - all actions
)

// String returns the string representation of the action
func (a PermissionAction) String() string {
	return string(a)
}

// PermissionResource represents common permission resources
type PermissionResource string

const (
	// Core resources
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
	
	// Wildcard
	ResourceAll           PermissionResource = "*"
)

// String returns the string representation of the resource
func (r PermissionResource) String() string {
	return string(r)
}

// BuildPermissionName constructs a permission name from action and resource
// Example: BuildPermissionName(ActionView, ResourceUsers) => "view on users"
func BuildPermissionName(action PermissionAction, resource PermissionResource) string {
	return string(action) + " on " + string(resource)
}

// ParsePermissionName parses a permission name into action and resource
// Returns empty strings if the format is invalid
func ParsePermissionName(name string) (action PermissionAction, resource PermissionResource) {
	// Simple parser for "action on resource" format
	// Can be enhanced with more sophisticated parsing if needed
	return "", ""
}

