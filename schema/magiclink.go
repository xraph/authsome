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

	ID                 xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	Email              string    `json:"email" bun:"email,notnull"`
	Token              string    `json:"token" bun:"token,notnull"`
	AppID              xid.ID    `json:"appId" bun:"app_id,notnull,type:varchar(20)"`                              // Platform app (required)
	UserOrganizationID *xid.ID   `json:"userOrganizationId,omitempty" bun:"user_organization_id,type:varchar(20)"` // User-created org (optional)
	ExpiresAt          time.Time `json:"expiresAt" bun:"expires_at,notnull"`
}
