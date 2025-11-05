package ratelimit

import (
	"context"
	"time"
)

// Storage abstracts the rate limit counter storage
type Storage interface {
	// Increment increases the counter for key within the window and returns the current count
	Increment(ctx context.Context, key string, window time.Duration) (int, error)
}
