// Package webhook defines the webhook domain entity and its store interface.
// This is for authsome's own webhook subscriptions. Actual delivery is
// delegated to Relay via the bridge.EventRelay interface.
package webhook

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Webhook represents a registered webhook endpoint.
type Webhook struct {
	ID        id.WebhookID     `json:"id"`
	AppID     id.AppID         `json:"app_id"`
	EnvID     id.EnvironmentID `json:"env_id"`
	URL       string           `json:"url"`
	Events    []string         `json:"events"`
	Secret    string           `json:"-"`
	Active    bool             `json:"active"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
