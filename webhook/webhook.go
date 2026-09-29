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
	ID     id.WebhookID     `json:"id"`
	AppID  id.AppID         `json:"app_id"`
	EnvID  id.EnvironmentID `json:"env_id"`
	URL    string           `json:"url"`
	Events []string         `json:"events"`
	// Secret is the signing secret in plaintext. It is set on the way out
	// of CreateWebhook and RotateWebhookSecret, shown to the caller once,
	// and never stored: a row that has a RelayEndpointID reads back with an
	// empty Secret. Rows written before webhooks became Relay endpoints
	// still carry their plaintext until they are adopted, and read it back
	// so adoption can hand the same secret to Relay.
	Secret string `json:"-"`
	// SecretHash is store.HashToken of the current secret, so a caller can
	// check which secret they hold without the store holding it.
	SecretHash string `json:"-"`
	// RelayEndpointID names the Relay endpoint that delivers for this
	// webhook. Empty on a row written before webhooks became endpoints.
	RelayEndpointID string    `json:"relay_endpoint_id,omitempty"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StoredSecret is what a store writes to the secret column: nothing once
// the webhook delivers through a Relay endpoint (Relay holds the secret),
// and the plaintext only for a row that has not been adopted yet, so the
// legacy shape is preserved until adoption rewrites it.
func StoredSecret(w *Webhook) string {
	if w.RelayEndpointID != "" {
		return ""
	}
	return w.Secret
}
