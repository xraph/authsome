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
    UserID    xid.ID     `bun:"user_id,notnull,type:varchar(20)"`
    Token     string     `bun:"token,notnull,unique"`
    Type      string     `bun:"type,notnull"` // email, phone, password_reset
    ExpiresAt time.Time  `bun:"expires_at,notnull"`
    Used      bool       `bun:"used,notnull,default:false"`
    UsedAt    *time.Time `bun:"used_at"`

    // Relations
    User *User `bun:"rel:belongs-to,join:user_id=id"`
}
