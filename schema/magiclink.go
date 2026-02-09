package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// MagicLink stores passwordless email tokens
// Updated for V2 architecture: App → Environment → Organization.
type MagicLink struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:magic_links,alias:ml"`

	ID             xid.ID    `bun:"id,pk,type:varchar(20)"           json:"id"`
	Email          string    `bun:"email,notnull"                    json:"email"`
	Token          string    `bun:"token,notnull"                    json:"token"`
	AppID          xid.ID    `bun:"app_id,notnull,type:varchar(20)"  json:"appID"`                    // Platform app (required)
	OrganizationID *xid.ID   `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"` // User-created org (optional)
	EnvironmentID  *xid.ID   `bun:"environment_id,type:varchar(20)"  json:"environmentID,omitempty"`  // Optional environment context
	ExpiresAt      time.Time `bun:"expires_at,notnull"               json:"expiresAt"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}
