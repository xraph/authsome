package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// PasswordHistory tracks user password history to prevent password reuse
type PasswordHistory struct {
	bun.BaseModel `bun:"table:password_histories,alias:ph"`

	ID           xid.ID    `bun:"type:varchar(20),pk" json:"id"`
	UserID       xid.ID    `bun:"type:varchar(20),notnull" json:"user_id"`
	PasswordHash string    `bun:"type:text,notnull" json:"password_hash"`
	CreatedAt    time.Time `bun:"type:timestamptz,notnull,default:current_timestamp" json:"created_at"`
}
