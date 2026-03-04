// Package securityevent defines queryable security event types for audit
// and compliance. Unlike the ephemeral hook bus, security events are persisted
// and queryable (e.g., "show all failed logins for user X in the last hour").
package securityevent

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
)

// Event represents a persisted security event.
type Event struct {
	ID        string            `json:"id"`
	AppID     id.AppID          `json:"app_id"`
	UserID    id.UserID         `json:"user_id,omitempty"`
	Action    string            `json:"action"`
	IPAddress string            `json:"ip_address,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Outcome   string            `json:"outcome"` // "success" or "failure"
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// Query holds filters for querying security events.
type Query struct {
	AppID  id.AppID  `json:"app_id"`
	UserID id.UserID `json:"user_id,omitempty"`
	Action string    `json:"action,omitempty"`
	Since  time.Time `json:"since,omitempty"`
	Until  time.Time `json:"until,omitempty"`
	Cursor string    `json:"cursor,omitempty"`
	Limit  int       `json:"limit,omitempty"`
}

// Store persists and queries security events.
type Store interface {
	// RecordSecurityEvent persists a security event.
	RecordSecurityEvent(ctx context.Context, event *Event) error

	// QuerySecurityEvents returns events matching the query and a cursor for
	// the next page. An empty cursor means no more results.
	QuerySecurityEvents(ctx context.Context, q *Query) ([]*Event, string, error)
}
