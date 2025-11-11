package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Member represents the organization member table
type Member struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:members,alias:m"`

	ID             xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID    `json:"organizationID" bun:"organization_id,notnull,type:varchar(20)"`
	UserID         xid.ID    `json:"userID" bun:"user_id,notnull,type:varchar(20)"`
	Role           string    `json:"role" bun:"role,notnull"`
	Status         string    `json:"status" bun:"status,notnull,default:'active'"`                        // active, suspended, pending
	JoinedAt       time.Time `json:"joinedAt" bun:"joined_at,nullzero,notnull,default:current_timestamp"` // when the member joined

	// Relations
	Organization *Organization `json:"organization" bun:"rel:belongs-to,join:organization_id=id"`
	User         *User         `json:"user" bun:"rel:belongs-to,join:user_id=id"`
}
