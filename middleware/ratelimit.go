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

	// FailOpen lets the request through when the limiter returns an error.
	// Off by default: a limiter that cannot answer refuses with 503, so an
	// outage of the shared store does not become an unlimited window.
	FailOpen bool

	// OnError is called with the limiter's error, for a metric or a log.
	OnError func(err error)

	// OnReject is called when a request is refused for exceeding the limit,
	// and OnFailOpen when a limiter error let one through. Both are for
	// metrics.
	OnReject   func()
	OnFailOpen func()
}

// RateLimit returns a middleware that enforces rate limits using the given limiter.
func RateLimit(limiter ratelimit.Limiter, cfg RateLimitConfig) forge.Middleware {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(ctx forge.Context) string {
			// Trusted-proxy-aware client IP: forwarding headers are honored only
			// when the peer is a trusted proxy, so a direct client cannot mint a
			// fresh rate-limit bucket per request by spoofing X-Forwarded-For.
			return ClientIP(ctx.Request())
		}
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			key := cfg.KeyFunc(ctx)

			allowed, err := limiter.Allow(ctx.Context(), key, cfg.Limit, cfg.Window)
			if err != nil {
				if cfg.OnError != nil {
					cfg.OnError(err)
				}
				if cfg.FailOpen {
					if cfg.OnFailOpen != nil {
						cfg.OnFailOpen()
					}
					return next(ctx)
				}
				ctx.Response().Header().Set("Retry-After", "5")
				return forge.NewHTTPError(http.StatusServiceUnavailable,
					"rate limiting is temporarily unavailable, try again shortly")
			}

			if !allowed {
				if cfg.OnReject != nil {
					cfg.OnReject()
				}
				remaining, _ := limiter.Remaining(ctx.Context(), key, cfg.Limit, cfg.Window) //nolint:errcheck // best-effort rate check
				retryAfter := int(cfg.Window.Seconds())

				ctx.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
				ctx.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				ctx.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))

				return forge.NewHTTPError(http.StatusTooManyRequests,
					"rate limit exceeded, try again in "+strconv.Itoa(retryAfter)+" seconds")
			}

			return next(ctx)
		}
	}
}
