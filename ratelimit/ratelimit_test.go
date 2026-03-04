package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/ratelimit"
)

func TestMemoryLimiter_Allow(t *testing.T) {
	limiter := ratelimit.NewMemoryLimiter()
	ctx := context.Background()
	key := "test-key"
	limit := 3
	window := 1 * time.Second

	// First 3 should be allowed
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(ctx, key, limit, window)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	// 4th should be denied
	allowed, err := limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.False(t, allowed, "request 4 should be denied")
}

func TestMemoryLimiter_Remaining(t *testing.T) {
	limiter := ratelimit.NewMemoryLimiter()
	ctx := context.Background()
	key := "remaining-key"
	limit := 5
	window := 10 * time.Second

	remaining, err := limiter.Remaining(ctx, key, limit, window)
	require.NoError(t, err)
	assert.Equal(t, 5, remaining)

	_, _ = limiter.Allow(ctx, key, limit, window)
	_, _ = limiter.Allow(ctx, key, limit, window)

	remaining, err = limiter.Remaining(ctx, key, limit, window)
	require.NoError(t, err)
	assert.Equal(t, 3, remaining)
}

func TestMemoryLimiter_WindowExpiry(t *testing.T) {
	limiter := ratelimit.NewMemoryLimiter()
	ctx := context.Background()
	key := "expiry-key"
	limit := 2
	window := 50 * time.Millisecond

	// Use up the limit
	_, _ = limiter.Allow(ctx, key, limit, window)
	_, _ = limiter.Allow(ctx, key, limit, window)

	allowed, err := limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.False(t, allowed)

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, key, limit, window)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestMemoryLimiter_DifferentKeys(t *testing.T) {
	limiter := ratelimit.NewMemoryLimiter()
	ctx := context.Background()
	limit := 1
	window := 10 * time.Second

	allowed, err := limiter.Allow(ctx, "key-a", limit, window)
	require.NoError(t, err)
	assert.True(t, allowed)

	// key-a is exhausted
	allowed, err = limiter.Allow(ctx, "key-a", limit, window)
	require.NoError(t, err)
	assert.False(t, allowed)

	// key-b is independent
	allowed, err = limiter.Allow(ctx, "key-b", limit, window)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestNoopLimiter(t *testing.T) {
	limiter := ratelimit.NewNoopLimiter()
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		allowed, err := limiter.Allow(ctx, "key", 1, time.Second)
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	remaining, err := limiter.Remaining(ctx, "key", 5, time.Second)
	require.NoError(t, err)
	assert.Equal(t, 5, remaining)
}
