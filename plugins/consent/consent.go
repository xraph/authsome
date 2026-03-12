package consent

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
)

// Consent represents a user's consent record for a specific purpose.
type Consent struct {
	ID        id.ConsentID `json:"id"`
	UserID    id.UserID    `json:"user_id"`
	AppID     id.AppID     `json:"app_id"`
	Purpose   string       `json:"purpose"`              // e.g. "marketing", "analytics", "essential"
	Granted   bool         `json:"granted"`              // true = granted, false = revoked
	Version   string       `json:"version"`              // policy version this consent applies to
	IPAddress string       `json:"ip_address"`           // IP at time of consent action
	GrantedAt time.Time    `json:"granted_at"`           // when consent was granted
	RevokedAt *time.Time   `json:"revoked_at,omitempty"` // when consent was revoked (nil if active)
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// Query holds filters for listing consent records.
type Query struct {
	UserID  id.UserID `json:"user_id"`
	AppID   id.AppID  `json:"app_id"`
	Purpose string    `json:"purpose,omitempty"`
	Cursor  string    `json:"cursor,omitempty"`
	Limit   int       `json:"limit,omitempty"`
}

// Store persists and queries consent records.
type Store interface {
	// GrantConsent records or updates a consent grant for a specific purpose.
	// If a consent record already exists for the user+app+purpose combination,
	// it is updated rather than duplicated.
	GrantConsent(ctx context.Context, c *Consent) error

	// RevokeConsent marks a consent as revoked by setting RevokedAt and Granted=false.
	RevokeConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) error

	// GetConsent returns a specific consent record by user, app, and purpose.
	GetConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) (*Consent, error)

	// ListConsents returns all consent records matching the query.
	ListConsents(ctx context.Context, q *Query) ([]*Consent, string, error)
}
