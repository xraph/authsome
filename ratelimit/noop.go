package ratelimit

import (
	"context"
	"time"
)

// NoopLimiter is a rate limiter that always allows requests.
type NoopLimiter struct{}

// NewNoopLimiter creates a no-op rate limiter.
func NewNoopLimiter() *NoopLimiter { return &NoopLimiter{} }

var _ Limiter = (*NoopLimiter)(nil)

// Allow always returns true.
func (*NoopLimiter) Allow(context.Context, string, int, time.Duration) (bool, error) {
	return true, nil
}

// Remaining always returns the full limit.
func (*NoopLimiter) Remaining(_ context.Context, _ string, limit int, _ time.Duration) (int, error) {
	return limit, nil
}
