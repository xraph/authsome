// Package lockout provides account lockout tracking for AuthSome.
package lockout

import (
	"context"
	"time"
)

// Tracker tracks failed authentication attempts and lockouts.
type Tracker interface {
	// RecordFailure records a failed authentication attempt. Returns the
	// current failure count for the key.
	RecordFailure(ctx context.Context, key string) (attempts int, err error)

	// IsLocked checks if the key is currently locked out. Returns true if
	// locked, along with the time the lockout expires.
	IsLocked(ctx context.Context, key string) (locked bool, until time.Time, err error)

	// Reset clears the failure count for a key (called on successful auth).
	Reset(ctx context.Context, key string) error
}
