package schema

import (
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"time"
)

// MagicLink stores passwordless email tokens
type MagicLink struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:magic_links,alias:ml"`

	ID        xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	Email     string    `json:"email" bun:"email,notnull"`
	Token     string    `json:"token" bun:"token,notnull"`
	ExpiresAt time.Time `json:"expiresAt" bun:"expires_at,notnull"`
}
