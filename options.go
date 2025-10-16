package authsome

import (
	rl "github.com/xraph/authsome/core/ratelimit"
	sec "github.com/xraph/authsome/core/security"
)

// Option is a function that configures Auth
type Option func(*Auth)

// WithMode sets the operation mode
func WithMode(mode Mode) Option {
	return func(a *Auth) {
		a.config.Mode = mode
	}
}

// WithForgeConfig sets the Forge config manager (type deferred)
func WithForgeConfig(cfg interface{}) Option {
	return func(a *Auth) {
		a.forgeConfig = cfg
	}
}

// WithDatabase sets the database connection
func WithDatabase(db interface{}) Option {
	return func(a *Auth) {
		a.db = db
	}
}

// WithBasePath sets the base path for routes
func WithBasePath(path string) Option {
	return func(a *Auth) {
		a.config.BasePath = path
	}
}

// WithTrustedOrigins sets trusted origins for CORS
func WithTrustedOrigins(origins []string) Option {
	return func(a *Auth) {
		a.config.TrustedOrigins = origins
	}
}

// WithSecret sets the secret for token signing
func WithSecret(secret string) Option {
	return func(a *Auth) {
		a.config.Secret = secret
	}
}

// WithSecurityConfig sets security service configuration (IP rules, country rules)
// Pass lists like IPWhitelist/IPBlacklist; Enabled true to enforce checks
func WithSecurityConfig(cfg sec.Config) Option {
	return func(a *Auth) {
		a.securityConfig = cfg
	}
}

// WithRateLimitConfig sets rate limit configuration (enabled, default rule, per-path rules)
func WithRateLimitConfig(cfg rl.Config) Option {
	return func(a *Auth) {
		a.rateLimitConfig = cfg
	}
}

// WithRateLimitStorage sets the rate limit storage backend (memory or redis)
func WithRateLimitStorage(storage rl.Storage) Option {
	return func(a *Auth) {
		a.rateLimitStorage = storage
	}
}

// WithGeoIPProvider sets a GeoIP provider for country-based restrictions
func WithGeoIPProvider(provider sec.GeoIPProvider) Option {
	return func(a *Auth) {
		a.geoipProvider = provider
	}
}

// WithRBACEnforcement enables/disables handler-level RBAC enforcement
func WithRBACEnforcement(enabled bool) Option {
	return func(a *Auth) {
		a.config.RBACEnforce = enabled
	}
}
