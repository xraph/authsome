package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// AccountLockout tracks locked user accounts due to failed login attempts
type AccountLockout struct {
	bun.BaseModel `bun:"table:account_lockouts,alias:al"`

	ID          xid.ID    `bun:"type:varchar(20),pk" json:"id"`
	UserID      xid.ID    `bun:"type:varchar(20),notnull" json:"user_id"`
	LockedUntil time.Time `bun:"type:timestamptz,notnull" json:"locked_until"`
	Reason      string    `bun:"type:varchar(255)" json:"reason"`
	CreatedAt   time.Time `bun:"type:timestamptz,notnull,default:current_timestamp" json:"created_at"`
}

