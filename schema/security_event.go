package schema

import (
    "github.com/rs/xid"
    "github.com/uptrace/bun"
)

// SecurityEvent represents the security_events table
type SecurityEvent struct {
    AuditableModel `bun:",inline"`
    bun.BaseModel  `bun:"table:security_events,alias:se"`

    ID        xid.ID  `bun:"id,pk,type:varchar(20)"`
    UserID    *xid.ID `bun:"user_id,type:varchar(20)"`
    Type      string  `bun:"type,notnull"`
    IPAddress string  `bun:"ip_address"`
    UserAgent string  `bun:"user_agent"`
    Geo       string  `bun:"geo"`
}