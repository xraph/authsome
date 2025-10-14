package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"time"
)

// EmailOTP stores OTP codes for email-based authentication
type EmailOTP struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:email_otps,alias:eotp"`

	ID        xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	Email     string    `json:"email" bun:"email,notnull"`
	OTP       string    `json:"otp" bun:"otp,notnull"`
	ExpiresAt time.Time `json:"expiresAt" bun:"expires_at,notnull"`
	Attempts  int       `json:"attempts" bun:"attempts,notnull,default:0"`
}
