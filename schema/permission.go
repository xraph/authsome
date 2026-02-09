package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Permission table.
type Permission struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:permissions,alias:perm"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)"           json:"id"`
	AppID          *xid.ID `bun:"app_id,type:varchar(20)"          json:"appID"`          // App-scoped permissions
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationID"` // Org-scoped permissions (NULL = app-level)
	Name           string  `bun:"name,notnull"                     json:"name"`
	Description    string  `bun:"description"                      json:"description"`
	IsCustom       bool    `bun:"is_custom,notnull,default:false"  json:"isCustom"` // Distinguishes custom from pre-defined permissions
	Category       string  `bun:"category"                         json:"category"` // Groups permissions: "users", "settings", "content", etc.

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Roles        []Role        `bun:"m2m:role_permissions,join:Permission=Role"`
}
