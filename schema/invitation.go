package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Invitation represents the app invitation table
type Invitation struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:invitations,alias:inv"`

	ID         xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID      xid.ID     `json:"appID" bun:"organization_id,notnull,type:varchar(20)"` // Column still named organization_id for migration compatibility
	Email      string     `json:"email" bun:"email,notnull"`
	Role       string     `json:"role" bun:"role,notnull"`
	InviterID  xid.ID     `json:"inviterID" bun:"inviter_id,notnull,type:varchar(20)"`
	Token      string     `json:"token" bun:"token,notnull,unique"`
	ExpiresAt  time.Time  `json:"expiresAt" bun:"expires_at,notnull"`
	AcceptedAt *time.Time `json:"acceptedAt" bun:"accepted_at"`
	Status     string     `json:"status" bun:"status,notnull"`

	// Relations
	App *App `bun:"rel:belongs-to,join:organization_id=id"`
}
