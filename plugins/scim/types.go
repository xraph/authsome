package scim

import (
	"time"

	"github.com/xraph/authsome/id"
)

// SCIMConfig represents a SCIM 2.0 provisioning configuration.
// Each config provides a unique SCIM endpoint with its own bearer tokens,
// and can be scoped to an application or a specific organization.
type SCIMConfig struct {
	ID          id.SCIMConfigID   `json:"id"`
	AppID       id.AppID          `json:"app_id"`
	OrgID       id.OrgID          `json:"org_id,omitempty"` // empty = app-level
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	AutoCreate  bool              `json:"auto_create"`  // auto-create users on SCIM push
	AutoSuspend bool              `json:"auto_suspend"` // auto-suspend on SCIM deactivate
	GroupSync   bool              `json:"group_sync"`   // sync SCIM Groups to teams
	DefaultRole string            `json:"default_role"` // default org role for provisioned users
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SCIMToken represents a bearer token for authenticating SCIM API requests.
type SCIMToken struct {
	ID         id.SCIMTokenID `json:"id"`
	ConfigID   id.SCIMConfigID `json:"config_id"`
	Name       string          `json:"name"`
	TokenHash  string          `json:"-"`
	LastUsedAt *time.Time      `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time      `json:"expires_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// IsExpired reports whether the token has expired.
func (t *SCIMToken) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}

// SCIMProvisionLog records a single SCIM provisioning action for audit purposes.
type SCIMProvisionLog struct {
	ID           id.SCIMLogID `json:"id"`
	ConfigID     id.SCIMConfigID `json:"config_id"`
	Action       string          `json:"action"`        // "create_user", "update_user", "delete_user", "create_group", etc.
	ResourceType string          `json:"resource_type"` // "User" or "Group"
	ExternalID   string          `json:"external_id"`   // SCIM externalId
	InternalID   string          `json:"internal_id"`   // authsome user/team ID
	Status       string          `json:"status"`        // "success", "error", "skipped"
	Detail       string          `json:"detail"`        // error message or summary
	CreatedAt    time.Time       `json:"created_at"`
}

// Provision log statuses.
const (
	LogStatusSuccess = "success"
	LogStatusError   = "error"
	LogStatusSkipped = "skipped"
)

// Provision log actions.
const (
	ActionCreateUser  = "create_user"
	ActionUpdateUser  = "update_user"
	ActionDeleteUser  = "delete_user"
	ActionSuspendUser = "suspend_user"
	ActionCreateGroup = "create_group"
	ActionUpdateGroup = "update_group"
	ActionDeleteGroup = "delete_group"
	ActionAddMember   = "add_member"
	ActionRemoveMember = "remove_member"
)
