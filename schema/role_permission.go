package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	bun.BaseModel `bun:"table:role_permissions,alias:rp"`

	ID           xid.ID    `bun:"id,pk,type:varchar(20)"`
	RoleID       xid.ID    `bun:"role_id,notnull,type:varchar(20)"`
	PermissionID xid.ID    `bun:"permission_id,notnull,type:varchar(20)"`
	CreatedAt    time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Role       *Role       `bun:"rel:belongs-to,join:role_id=id"`
	Permission *Permission `bun:"rel:belongs-to,join:permission_id=id"`
}

