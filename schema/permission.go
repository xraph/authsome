package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Permission table
type Permission struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:permissions,alias:perm"`

	ID          xid.ID  `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID       *xid.ID `json:"appID" bun:"app_id,type:varchar(20)"` // App-scoped permissions
	Name        string  `json:"name" bun:"name,notnull,unique:perm_app_name"`
	Description string  `json:"description" bun:"description"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}
