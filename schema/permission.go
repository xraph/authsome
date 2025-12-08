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
	AppID          *xid.ID `json:"appID" bun:"app_id,type:varchar(20)"`                   // App-scoped permissions
	OrganizationID *xid.ID `json:"organizationID" bun:"organization_id,type:varchar(20)"` // Org-scoped permissions (NULL = app-level)
	Name           string  `json:"name" bun:"name,notnull"`
	Description    string  `json:"description" bun:"description"`
	IsCustom       bool    `json:"isCustom" bun:"is_custom,notnull,default:false"` // Distinguishes custom from pre-defined permissions
	Category       string  `json:"category" bun:"category"`                        // Groups permissions: "users", "settings", "content", etc.

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Roles        []Role        `bun:"m2m:role_permissions,join:Permission=Role"`
}
