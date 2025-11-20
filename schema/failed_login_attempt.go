package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// FailedLoginAttempt records failed login attempts for account lockout functionality
type FailedLoginAttempt struct {
	bun.BaseModel `bun:"table:failed_login_attempts,alias:fla"`

	ID        xid.ID    `bun:"type:varchar(20),pk" json:"id"`
	Username  string    `bun:"type:varchar(255),notnull" json:"username"`
	AppID     xid.ID    `bun:"type:varchar(20),notnull" json:"app_id"`
	IP        string    `bun:"type:varchar(45)" json:"ip"`
	UserAgent string    `bun:"type:text" json:"user_agent"`
	AttemptAt time.Time `bun:"type:timestamptz,notnull,default:current_timestamp" json:"attempt_at"`
}

