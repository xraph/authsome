package session

import (
	"time"

	"github.com/rs/xid"
)

// Session represents a user session
type Session struct {
	ID        xid.ID    `json:"id"`
	Token     string    `json:"token"`
	UserID    xid.ID    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	IPAddress string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateSessionRequest represents the data to create a session
type CreateSessionRequest struct {
	UserID    xid.ID `json:"userId"`
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
	Remember  bool   `json:"remember"`
}
