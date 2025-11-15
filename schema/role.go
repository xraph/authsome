package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Role table
type Role struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:roles,alias:r"`

	ID          xid.ID  `bun:"id,pk,type:varchar(20)"`
	AppID       *xid.ID `bun:"app_id,type:varchar(20)"` // App-scoped roles
	Name        string  `bun:"name,notnull,unique:role_app_name"`
	Description string  `bun:"description"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}
