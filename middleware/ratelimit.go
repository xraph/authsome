package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/ratelimit"
)

// RateLimitConfig configures the rate limit middleware for a specific endpoint.
type RateLimitConfig struct {
	// Limit is the maximum number of requests per window.
	Limit int

	// Window is the sliding window duration.
	Window time.Duration

	// KeyFunc extracts the rate limit key from the request (default: client IP).
	KeyFunc func(ctx forge.Context) string
}

// RateLimit returns a middleware that enforces rate limits using the given limiter.
func RateLimit(limiter ratelimit.Limiter, cfg RateLimitConfig) forge.Middleware {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(ctx forge.Context) string {
			// Check proxy headers first, then fall back to RemoteAddr.
			if xff := ctx.Request().Header.Get("X-Forwarded-For"); xff != "" {
				return xff
			}
			if xri := ctx.Request().Header.Get("X-Real-IP"); xri != "" {
				return xri
			}
			return ctx.Request().RemoteAddr
		}
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			key := cfg.KeyFunc(ctx)

			allowed, err := limiter.Allow(ctx.Context(), key, cfg.Limit, cfg.Window)
			if err != nil {
				// On error, allow the request through (fail open)
				return next(ctx)
			}

			if !allowed {
				remaining, _ := limiter.Remaining(ctx.Context(), key, cfg.Limit, cfg.Window) //nolint:errcheck // best-effort rate check
				retryAfter := int(cfg.Window.Seconds())

				ctx.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
				ctx.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				ctx.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))

				return ctx.JSON(http.StatusTooManyRequests, map[string]any{
					"error": "rate limit exceeded, try again in " + strconv.Itoa(retryAfter) + " seconds",
					"code":  http.StatusTooManyRequests,
				})
			}

			return next(ctx)
		}
	}
}
