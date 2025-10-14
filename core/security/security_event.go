package security

import (
	"time"

	"github.com/rs/xid"
)

// SecurityEvent represents a logged security event
type SecurityEvent struct {
	ID        xid.ID    `json:"id"`
	UserID    *xid.ID   `json:"userId"`
	Type      string    `json:"type"`
	IPAddress string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
	Geo       string    `json:"geo"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
