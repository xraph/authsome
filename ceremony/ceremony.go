// Package ceremony provides a store interface for short-lived ceremony
// sessions used by authentication plugins (passkey, social, SSO, MFA).
//
// Ceremony data is ephemeral (seconds to minutes) and keyed by a unique
// identifier with automatic expiration. This abstraction replaces in-memory
// sync.Map/map usage in plugins, enabling multi-instance deployments
// via database-backed implementations.
package ceremony

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when the requested ceremony key does not exist
// or has expired.
var ErrNotFound = errors.New("ceremony: key not found or expired")

// Store persists short-lived ceremony/state data with automatic expiration.
type Store interface {
	// Set stores data under key with a TTL. Overwrites if the key exists.
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error

	// Get retrieves data by key. Returns ErrNotFound if the key is absent
	// or has expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete removes data by key. It is idempotent — deleting a
	// non-existent key returns nil.
	Delete(ctx context.Context, key string) error
}
