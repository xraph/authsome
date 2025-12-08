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
	AppID          *xid.ID `bun:"app_id,type:varchar(20)"`          // App-scoped roles
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)"` // Org-scoped roles (NULL = app-level template)
	Name           string  `bun:"name,notnull"`
	Description    string  `bun:"description"`
	IsTemplate     bool    `bun:"is_template,notnull,default:false"`   // Marks roles as templates for cloning
	IsOwnerRole    bool    `bun:"is_owner_role,notnull,default:false"` // Marks the default owner role for new orgs
	TemplateID     *xid.ID `bun:"template_id,type:varchar(20)"`        // Tracks which template this role was cloned from

	// Relations
	App          *App          `bun:"rel:belongs-to,join:app_id=id"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Template     *Role         `bun:"rel:belongs-to,join:template_id=id"`
	Permissions  []Permission  `bun:"m2m:role_permissions,join:Role=Permission"`
}
