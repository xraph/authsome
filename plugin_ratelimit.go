package authsome

import (
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/ratelimit"
)

// RateLimitOptions returns the route options that throttle an endpoint to
// limit requests per window, keyed by client address, or nil when rate
// limiting is off or the limit is zero. Every route in the tree that carries
// a limit builds it here, so the fail-closed behaviour and the error metric
// are the same everywhere.
func (e *Engine) RateLimitOptions(limit int) []forge.RouteOption {
	rl := e.RateLimiter()
	cfg := e.Config().RateLimit
	if rl == nil || !cfg.Enabled || limit <= 0 {
		return nil
	}
	return []forge.RouteOption{
		forge.WithMiddleware(middleware.RateLimit(rl, e.RateLimitMiddlewareConfig(limit))),
	}
}

// RateLimitMiddleware returns the throttling middleware for limit requests
// per window, or nil when rate limiting is off or the limit is zero. Route
// groups that share one budget (SSO, SCIM) mount it as a group middleware.
func (e *Engine) RateLimitMiddleware(limit int) forge.Middleware {
	rl := e.RateLimiter()
	cfg := e.Config().RateLimit
	if rl == nil || !cfg.Enabled || limit <= 0 {
		return nil
	}
	return middleware.RateLimit(rl, e.RateLimitMiddlewareConfig(limit))
}

// PluginRateLimitMiddleware is RateLimitMiddleware for a plugin that holds
// the engine as plugin.Engine; nil when the engine is not the concrete
// *Engine or limiting is off.
func PluginRateLimitMiddleware(engine any, pick func(RateLimitConfig) int) forge.Middleware {
	eng, ok := engine.(*Engine)
	if !ok || eng == nil {
		return nil
	}
	return eng.RateLimitMiddleware(pick(eng.Config().RateLimit))
}

// RateLimitMiddlewareConfig builds the middleware configuration for limit
// requests per window: the configured window, the fail-open switch and a
// counter for limiter errors.
func (e *Engine) RateLimitMiddlewareConfig(limit int) middleware.RateLimitConfig {
	cfg := e.Config().RateLimit
	return middleware.RateLimitConfig{
		Limit:    limit,
		Window:   cfg.Window(),
		FailOpen: cfg.FailOpen,
		OnError: func(err error) {
			e.count("ratelimit.error", "")
			e.logger.Warn("authsome: rate limiter error", log.String("error", err.Error()))
		},
		OnReject:   func() { e.count("ratelimit.rejected", "") },
		OnFailOpen: func() { e.count("control.degraded", "") },
	}
}

// PluginLimiter hands a plugin the engine's limiter and rate-limit config.
// ok is false when the engine is not the concrete *Engine, the limiter is
// absent or rate limiting is off.
func PluginLimiter(engine any) (rl ratelimit.Limiter, cfg RateLimitConfig, ok bool) {
	eng, isEngine := engine.(*Engine)
	if !isEngine || eng == nil {
		return nil, RateLimitConfig{}, false
	}
	rl = eng.RateLimiter()
	cfg = eng.Config().RateLimit
	if rl == nil || !cfg.Enabled {
		return nil, cfg, false
	}
	return rl, cfg, true
}

// PluginRateLimit returns forge route options that rate-limit an endpoint
// using the engine's limiter, or nil when rate limiting is off. Plugins use
// it so their routes share the engine's window and behaviour.
//
// pick selects the limit from the engine's rate-limit config; a zero limit
// disables limiting for that route.
func PluginRateLimit(engine any, pick func(RateLimitConfig) int) []forge.RouteOption {
	eng, ok := engine.(*Engine)
	if !ok || eng == nil {
		return nil
	}
	return eng.RateLimitOptions(pick(eng.Config().RateLimit))
}
