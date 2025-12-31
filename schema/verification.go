package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Verification represents email/phone verification tokens
type Verification struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:verifications,alias:v"`

	ID        xid.ID     `bun:"id,pk,type:varchar(20)"`
	AppID     xid.ID     `bun:"app_id,notnull,type:varchar(20)"`
	UserID    xid.ID     `bun:"user_id,notnull,type:varchar(20)"`
	Token     string     `bun:"token,notnull,unique"`
	Code      string     `bun:"code"`         // 6-digit numeric code for mobile-friendly verification
	Type      string     `bun:"type,notnull"` // email, phone, password_reset
	ExpiresAt time.Time  `bun:"expires_at,notnull"`
	Used      bool       `bun:"used,notnull,default:false"`
	UsedAt    *time.Time `bun:"used_at"`

	// Relations
	App  *App  `bun:"rel:belongs-to,join:app_id=id"`
	User *User `bun:"rel:belongs-to,join:user_id=id"`
}
