package schema

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SCIMProvider represents a SCIM identity provider configuration.
type SCIMProvider struct {
	bun.BaseModel `bun:"table:scim_providers"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)"           json:"id"`
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)"  json:"app_id"`
	EnvironmentID  xid.ID  `bun:"environment_id,type:varchar(20)"  json:"environment_id"`
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organization_id,omitempty"`

	Name      string `bun:"name,notnull"      json:"name"`
	Type      string `bun:"type,notnull"      json:"type"`      // "okta", "azure_ad", "onelogin", "google_workspace", "custom"
	Direction string `bun:"direction,notnull" json:"direction"` // "inbound", "outbound", "bidirectional"

	// Inbound configuration (IdP → AuthSome)
	BaseURL    *string `bun:"base_url"    json:"base_url,omitempty"`
	AuthMethod string  `bun:"auth_method" json:"auth_method"` // "bearer", "oauth2"

	// Outbound configuration (AuthSome → External)
	TargetURL  *string `bun:"target_url"             json:"target_url,omitempty"`
	TargetAuth *string `bun:"target_auth,type:jsonb" json:"-"` // Encrypted auth credentials

	Status         string     `bun:"status"           json:"status"` // "active", "inactive", "error"
	LastSyncAt     *time.Time `bun:"last_sync_at"     json:"last_sync_at,omitempty"`
	LastSyncStatus string     `bun:"last_sync_status" json:"last_sync_status,omitempty"`

	Config   map[string]any `bun:"config,type:jsonb"   json:"config,omitempty"`
	Metadata map[string]any `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// BeforeInsert hook to set ID and timestamps.
func (p *SCIMProvider) BeforeInsert(ctx context.Context, db *bun.DB) error {
	if p.ID.IsNil() {
		p.ID = xid.New()
	}

	now := time.Now()
	p.CreatedAt = now

	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = "active"
	}

	return nil
}

// BeforeUpdate hook to update timestamp.
func (p *SCIMProvider) BeforeUpdate(ctx context.Context, db *bun.DB) error {
	p.UpdatedAt = time.Now()

	return nil
}
