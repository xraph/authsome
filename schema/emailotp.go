package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// EmailOTP stores OTP codes for email-based authentication
type EmailOTP struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:email_otps,alias:eotp"`

	ID        xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID     xid.ID    `json:"appID" bun:"app_id,notnull,type:varchar(20)"`
	Email     string    `json:"email" bun:"email,notnull"`
	OTP       string    `json:"otp" bun:"otp,notnull"`
	ExpiresAt time.Time `json:"expiresAt" bun:"expires_at,notnull"`
	Attempts  int       `json:"attempts" bun:"attempts,notnull,default:0"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}
