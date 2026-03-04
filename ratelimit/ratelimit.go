// Package ratelimit provides rate limiting for AuthSome endpoints.
package ratelimit

import (
	"context"
	"time"
)

// Limiter is the interface for rate limiting.
type Limiter interface {
	// Allow checks if a request identified by key is allowed within the given
	// limit and window. Returns true if allowed, false if rate limited.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)

	// Remaining returns the number of requests remaining for the given key.
	Remaining(ctx context.Context, key string, limit int, window time.Duration) (int, error)
}
