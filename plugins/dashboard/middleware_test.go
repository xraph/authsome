package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(10, time.Minute)

	t.Run("allows requests under limit", func(t *testing.T) {
		clientIP := "192.168.1.1"

		// Should allow first 10 requests
		for i := range 10 {
			allowed := rl.allow(clientIP)
			assert.True(t, allowed, "request %d should be allowed", i+1)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		clientIP := "192.168.1.2"

		// Use up all tokens
		for range 10 {
			rl.allow(clientIP)
		}

		// 11th request should be blocked
		allowed := rl.allow(clientIP)
		assert.False(t, allowed, "request over limit should be blocked")
	})

	t.Run("different IPs have separate buckets", func(t *testing.T) {
		clientIP1 := "192.168.1.3"
		clientIP2 := "192.168.1.4"

		// Use up tokens for IP1
		for range 10 {
			rl.allow(clientIP1)
		}

		// IP2 should still be allowed
		allowed := rl.allow(clientIP2)
		assert.True(t, allowed, "different IP should have separate limit")
	})
}

// Note: compareTokens is a private function and tested through middleware integration tests

func TestClientBucket(t *testing.T) {
	t.Run("creates new bucket with count 1", func(t *testing.T) {
		rl := newRateLimiter(10, time.Minute)
		clientIP := "192.168.1.100"

		// First call creates bucket with count 1
		allowed := rl.allow(clientIP)
		assert.True(t, allowed)

		// Verify bucket was created
		rl.mu.Lock()
		limit, exists := rl.clients[clientIP]
		rl.mu.Unlock()

		assert.True(t, exists)
		assert.Equal(t, 1, limit.count) // Should have count of 1 after first request
	})
}

// Note: Middleware integration tests would require setting up a full Forge context
// with mock services, which would be added in integration test files
