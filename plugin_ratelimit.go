package authsome

import (
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
)

// PluginRateLimit builds rate-limit route options from the engine's shared
// limiter, letting a plugin cap a code- or token-submission endpoint the same
// way the core API caps its own.
//
// Plugins own their route registration, so anything that accepts a guessable
// secret — a 6-digit OTP, an emailed code — has to opt into this explicitly or
// it ships unthrottled. `pick` selects which per-endpoint cap applies, e.g.
//
//	authsome.PluginRateLimit(p.engine, func(c authsome.RateLimitConfig) int {
//	    return c.VerifyEmailLimit
//	})
//
// Returns nil — meaning no middleware, not a closed door — when the host isn't
// the concrete *Engine (minimal test wiring), when rate limiting is disabled,
// or when the chosen cap is non-positive. Callers must therefore treat this as
// defence in depth and still bound attempts on the credential itself; a
// limiter that silently no-ops is the whole reason a per-challenge attempt
// counter is not optional.
func PluginRateLimit(engine any, pick func(RateLimitConfig) int) []forge.RouteOption {
	eng, ok := engine.(*Engine)
	if !ok || eng == nil {
		return nil
	}
	rl := eng.RateLimiter()
	cfg := eng.Config().RateLimit
	if rl == nil || !cfg.Enabled {
		return nil
	}
	limit := pick(cfg)
	if limit <= 0 {
		return nil
	}
	return []forge.RouteOption{
		forge.WithMiddleware(middleware.RateLimit(rl, middleware.RateLimitConfig{
			Limit:  limit,
			Window: cfg.Window(),
		})),
	}
}
