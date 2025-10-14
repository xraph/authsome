package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Invitation represents the organization invitation table
type Invitation struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:invitations,alias:inv"`

	ID             xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID     `json:"organizationID" bun:"organization_id,notnull,type:varchar(20)"`
	Email          string     `json:"email" bun:"email,notnull"`
	Role           string     `json:"role" bun:"role,notnull"`
	InviterID      xid.ID     `json:"inviterID" bun:"inviter_id,notnull,type:varchar(20)"`
	Token          string     `json:"token" bun:"token,notnull,unique"`
	ExpiresAt      time.Time  `json:"expiresAt" bun:"expires_at,notnull"`
	AcceptedAt     *time.Time `json:"acceptedAt" bun:"accepted_at"`
	Status         string     `json:"status" bun:"status,notnull"`

	// Relations
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
}
