package schema

import (
    "github.com/rs/xid"
    "github.com/uptrace/bun"
)

// Role table
type Role struct {
    AuditableModel `bun:",inline"`
    bun.BaseModel  `bun:"table:roles,alias:r"`

    ID             xid.ID  `bun:"id,pk,type:varchar(20)"`
    OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)"`
    Name           string  `bun:"name,notnull,unique:role_org_name"`
    Description    string  `bun:"description"`
}