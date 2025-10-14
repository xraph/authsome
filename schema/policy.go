package schema

import (
    "github.com/rs/xid"
    "github.com/uptrace/bun"
)

// Policy stores RBAC policy expressions
type Policy struct {
    AuditableModel `bun:",inline"`
    bun.BaseModel  `bun:"table:policies,alias:pol"`

    ID         xid.ID  `bun:"id,pk,type:varchar(20)"`
    Expression string  `bun:"expression,notnull"`
}