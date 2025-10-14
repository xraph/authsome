package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Organization represents the organization table
type Organization struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:organizations,alias:o"`

	ID       xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	Name     string                 `json:"name" bun:"name,notnull"`
	Slug     string                 `json:"slug" bun:"slug,notnull,unique"`
	Logo     string                 `json:"logo" bun:"logo"`
	Metadata map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
}
