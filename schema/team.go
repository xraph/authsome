package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Team represents the team table
type Team struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:teams,alias:t"`

	ID             xid.ID `bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID `bun:"organization_id,notnull,type:varchar(20)"`
	Name           string `bun:"name,notnull"`
	Description    string `bun:"description"`

	// Relations
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
}
