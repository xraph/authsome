package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Device represents the devices table.
type Device struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:devices,alias:d"`

	ID          xid.ID    `bun:"id,pk,type:varchar(20)"           json:"id"`
	AppID       xid.ID    `bun:"app_id,notnull,type:varchar(20)"  json:"appID"`
	UserID      xid.ID    `bun:"user_id,notnull,type:varchar(20)" json:"userID"`
	Fingerprint string    `bun:"fingerprint,notnull,unique"       json:"fingerprint"`
	UserAgent   string    `bun:"user_agent"                       json:"userAgent"`
	IPAddress   string    `bun:"ip_address"                       json:"ipAddress"`
	LastActive  time.Time `bun:"last_active,notnull"              json:"lastActive"`

	// Relations
	App  *App  `bun:"rel:belongs-to,join:app_id=id"`
	User *User `bun:"rel:belongs-to,join:user_id=id"`
}
