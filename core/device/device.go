package device

import (
	"time"

	"github.com/rs/xid"
)

// Device represents a user device
type Device struct {
	ID          xid.ID    `json:"id"`
	UserID      xid.ID    `json:"userId"`
	Fingerprint string    `json:"fingerprint"`
	UserAgent   string    `json:"userAgent"`
	IPAddress   string    `json:"ipAddress"`
	LastActive  time.Time `json:"lastActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
