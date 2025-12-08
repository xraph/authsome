package social

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter provides rate limiting for OAuth endpoints
type RateLimiter struct {
	redis  *redis.Client
	limits map[string]RateLimit
}

// RateLimit defines rate limiting parameters
type RateLimit struct {
	Requests int
	Window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis: redisClient,
		limits: map[string]RateLimit{
			"oauth_signin":   {Requests: 10, Window: time.Minute}, // 10 signin attempts per minute
			"oauth_callback": {Requests: 20, Window: time.Minute}, // 20 callbacks per minute
			"oauth_link":     {Requests: 5, Window: time.Minute},  // 5 link attempts per minute
			"oauth_unlink":   {Requests: 5, Window: time.Minute},  // 5 unlink attempts per minute
		},
	}
}

// Allow checks if a request is allowed under rate limits
func (r *RateLimiter) Allow(ctx context.Context, action, key string) error {
	if r.redis == nil {
		// No rate limiting if Redis is not available
		return nil
	}

	limit, ok := r.limits[action]
	if !ok {
		// Default limit if action not configured
		limit = RateLimit{Requests: 10, Window: time.Minute}
	}

	redisKey := fmt.Sprintf("authsome:social:ratelimit:%s:%s", action, key)

	// Use Redis INCR with EXPIRE for simple rate limiting
	pipe := r.redis.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, limit.Window)

	if _, err := pipe.Exec(ctx); err != nil {
		// If Redis fails, allow the request (fail open)
		return nil
	}

	count := incr.Val()
	if count > int64(limit.Requests) {
		return fmt.Errorf("rate limit exceeded: %d requests in %v", limit.Requests, limit.Window)
	}

	return nil
}

// SetLimit allows customizing rate limits
func (r *RateLimiter) SetLimit(action string, requests int, window time.Duration) {
	r.limits[action] = RateLimit{
		Requests: requests,
		Window:   window,
	}
}
