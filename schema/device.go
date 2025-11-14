package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Device represents the devices table
type Device struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:devices,alias:d"`

	ID          xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	UserID      xid.ID    `json:"userID" bun:"user_id,notnull,type:varchar(20)"`
	Fingerprint string    `json:"fingerprint" bun:"fingerprint,notnull,unique"`
	UserAgent   string    `json:"userAgent" bun:"user_agent"`
	IPAddress   string    `json:"ipAddress" bun:"ip_address"`
	LastActive  time.Time `json:"lastActive" bun:"last_active,notnull"`

	// Relations
	User *User `bun:"rel:belongs-to,join:user_id=id"`
}
