package social

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_SetLimit(t *testing.T) {
	limiter := NewRateLimiter(nil)

	// Set custom limit
	limiter.SetLimit("test_action", 100, 5*time.Minute)

	// Verify limit was set
	limit, ok := limiter.limits["test_action"]
	assert.True(t, ok)
	assert.Equal(t, 100, limit.Requests)
	assert.Equal(t, 5*time.Minute, limit.Window)
}

func TestRateLimiter_AllowWithoutRedis(t *testing.T) {
	// Rate limiter without Redis should allow all requests
	limiter := NewRateLimiter(nil)
	ctx := context.Background()

	// Should allow unlimited requests
	for range 1000 {
		err := limiter.Allow(ctx, "oauth_signin", "test-key")
		assert.NoError(t, err)
	}
}

func TestRateLimiter_DefaultLimits(t *testing.T) {
	limiter := NewRateLimiter(nil)

	// Check default limits
	assert.Equal(t, 10, limiter.limits["oauth_signin"].Requests)
	assert.Equal(t, time.Minute, limiter.limits["oauth_signin"].Window)

	assert.Equal(t, 20, limiter.limits["oauth_callback"].Requests)
	assert.Equal(t, time.Minute, limiter.limits["oauth_callback"].Window)

	assert.Equal(t, 5, limiter.limits["oauth_link"].Requests)
	assert.Equal(t, time.Minute, limiter.limits["oauth_link"].Window)

	assert.Equal(t, 5, limiter.limits["oauth_unlink"].Requests)
	assert.Equal(t, time.Minute, limiter.limits["oauth_unlink"].Window)
}
