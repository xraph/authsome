package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Permission table
type Permission struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:permissions,alias:perm"`

	ID             xid.ID  `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID *xid.ID `json:"organizationID" bun:"organization_id,type:varchar(20)"`
	Name           string  `json:"name" bun:"name,notnull,unique:perm_org_name"`
	Description    string  `json:"description" bun:"description"`
}
