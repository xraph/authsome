package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// UserRole maps users to roles within an app
type UserRole struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:user_roles,alias:ur"`

	ID     xid.ID `bun:"id,pk,type:varchar(20)"`
	UserID xid.ID `bun:"user_id,notnull,type:varchar(20)"`
	RoleID xid.ID `bun:"role_id,notnull,type:varchar(20)"`
	AppID  xid.ID `bun:"organization_id,notnull,type:varchar(20)"` // Column still named organization_id for migration compatibility

	// Relations
	User *User `bun:"rel:belongs-to,join:user_id=id"`
	Role *Role `bun:"rel:belongs-to,join:role_id=id"`
}
