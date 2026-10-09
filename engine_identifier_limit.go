package authsome

import (
	"context"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
)

// AllowIdentifier counts an attempt against an identifier (an email, a
// username, a phone number) within an app and scope, and returns
// account.ErrRateLimited once the limit for the window is spent.
//
// The route limiters key on the client address, which a distributed
// attacker rotates. This second limiter keys on the target instead, so one
// account cannot be hammered from many addresses. The identifier is hashed
// before it becomes a key, so the shared store never holds an address in
// the clear. A zero limit, a disabled limiter or no limiter at all lets the
// attempt through; a limiter error follows the fail-open setting.
func (e *Engine) AllowIdentifier(ctx context.Context, scope string, appID id.AppID, identifier string, limit int) error {
	if e.rateLimiter == nil || !e.config.RateLimit.Enabled || limit <= 0 || identifier == "" {
		return nil
	}
	key := "id:" + scope + ":" + appID.String() + ":" + hashIdentifier(identifier)
	allowed, err := e.rateLimiter.Allow(ctx, key, limit, e.config.RateLimit.Window())
	if err != nil {
		e.logger.Warn("authsome: identifier rate limiter error",
			log.String("scope", scope),
			log.String("error", err.Error()),
		)
		if e.metrics != nil {
			e.metrics.IncrementCounter("ratelimit.error", appID.String())
		}
		if e.config.RateLimit.FailOpen {
			e.count("control.degraded", appID.String())
			return nil
		}
		return account.ErrRateLimited
	}
	if !allowed {
		e.count("identifier_limit.rejected", appID.String())
		return account.ErrRateLimited
	}
	return nil
}

// PluginIdentifierLimit is AllowIdentifier for a plugin that holds the
// engine as plugin.Engine. pick selects the limit from the rate-limit
// config. It returns nil when the engine is not the concrete *Engine.
func PluginIdentifierLimit(ctx context.Context, engine any, scope string, appID id.AppID, identifier string, pick func(RateLimitConfig) int) error {
	eng, ok := engine.(*Engine)
	if !ok || eng == nil {
		return nil
	}
	return eng.AllowIdentifier(ctx, scope, appID, identifier, pick(eng.Config().RateLimit))
}
