package audit

import (
	"time"

	"github.com/rs/xid"
)

// Event represents an audit trail record
type Event struct {
	ID        xid.ID    `json:"id"`
	UserID    *xid.ID   `json:"userId"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	IPAddress string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
	Metadata  string    `json:"metadata"` // optional contextual metadata (JSON string or plain)
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
