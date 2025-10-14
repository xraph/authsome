package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Member represents the organization member table
type Member struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:members,alias:m"`

	ID             xid.ID `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID `json:"organizationID" bun:"organization_id,notnull,type:varchar(20)"`
	UserID         xid.ID `json:"userID" bun:"user_id,notnull,type:varchar(20)"`
	Role           string `json:"role" bun:"role,notnull"`

	// Relations
	Organization *Organization `json:"organization" bun:"rel:belongs-to,join:organization_id=id"`
	User         *User         `json:"user" bun:"rel:belongs-to,join:user_id=id"`
}
