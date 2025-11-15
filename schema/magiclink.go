package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// MagicLink stores passwordless email tokens
// Updated for V2 architecture: App → Environment → Organization
type MagicLink struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:magic_links,alias:ml"`

	ID             xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	Email          string    `json:"email" bun:"email,notnull"`
	Token          string    `json:"token" bun:"token,notnull"`
	AppID          xid.ID    `json:"appID" bun:"app_id,notnull,type:varchar(20)"`                     // Platform app (required)
	OrganizationID *xid.ID   `json:"organizationID,omitempty" bun:"organization_id,type:varchar(20)"` // User-created org (optional)
	EnvironmentID  *xid.ID   `json:"environmentID,omitempty" bun:"environment_id,type:varchar(20)"`   // Optional environment context
	ExpiresAt      time.Time `json:"expiresAt" bun:"expires_at,notnull"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}
