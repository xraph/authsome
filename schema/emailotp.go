package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// EmailOTP stores OTP codes for email-based authentication.
type EmailOTP struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:email_otps,alias:eotp"`

	ID        xid.ID    `bun:"id,pk,type:varchar(20)"          json:"id"`
	AppID     xid.ID    `bun:"app_id,notnull,type:varchar(20)" json:"appID"`
	Email     string    `bun:"email,notnull"                   json:"email"`
	OTP       string    `bun:"otp,notnull"                     json:"otp"`
	ExpiresAt time.Time `bun:"expires_at,notnull"              json:"expiresAt"`
	Attempts  int       `bun:"attempts,notnull,default:0"      json:"attempts"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}
