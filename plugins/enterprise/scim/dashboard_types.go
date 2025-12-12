package scim

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/enterprise/scim/schema"
)

// Dashboard-specific types

// SCIMToken represents a SCIM bearer token for authentication
type SCIMToken struct {
	ID             xid.ID     `json:"id"`
	AppID          xid.ID     `json:"app_id"`
	EnvironmentID  xid.ID     `json:"environment_id"`
	OrganizationID *xid.ID    `json:"organization_id,omitempty"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Token          string     `json:"token,omitempty"` // Only populated on creation
	Scopes         []string   `json:"scopes"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	UsageCount     int64      `json:"usage_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// SCIMProvider represents a SCIM identity provider (imported from schema)
type SCIMProvider = schema.SCIMProvider

// SCIMSyncEvent represents a sync event (imported from schema)
type SCIMSyncEvent = schema.SCIMSyncEvent
