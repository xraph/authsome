package authsome

import (
	"github.com/xraph/authsome/core/middleware"
	rl "github.com/xraph/authsome/core/ratelimit"
	sec "github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/forge"
	forgedb "github.com/xraph/forge/extensions/database"
)

// Option is a function that configures Auth
type Option func(*Auth)

// WithForgeApp sets the Forge application instance
func WithForgeApp(app forge.App) Option {
	return func(a *Auth) {
		a.forgeApp = app
	}
}

// WithDatabase sets the database connection directly (backwards compatible)
// For new applications, consider using WithDatabaseManager with Forge's database extension
func WithDatabase(db interface{}) Option {
	return func(a *Auth) {
		a.db = db
	}
}

// WithDatabaseManager uses Forge's database extension DatabaseManager
// This is the recommended approach when using Forge's database extension
// The database will be resolved from the manager using the default or specified name
func WithDatabaseManager(manager *forgedb.DatabaseManager, dbName ...string) Option {
	return func(a *Auth) {
		// Resolve database name (default if not specified)
		name := "default"
		if len(dbName) > 0 && dbName[0] != "" {
			name = dbName[0]
		}

		// Get BunDB from manager
		// This will be done lazily in Initialize() to ensure manager is ready
		a.config.DatabaseManagerName = name
		a.config.DatabaseManager = manager
	}
}

// WithDatabaseFromForge resolves the database from Forge's DI container
// This automatically uses the database extension if registered
func WithDatabaseFromForge() Option {
	return func(a *Auth) {
		a.config.UseForgeDI = true
	}
}

// WithBasePath sets the base path for routes
func WithBasePath(path string) Option {
	return func(a *Auth) {
		a.config.BasePath = path
	}
}

// WithExcludeFromSchemas sets whether to exclude the extension from schemas
func WithGlobalRoutesOptions(opts ...forge.RouteOption) Option {
	return func(a *Auth) {
		a.globalRoutesOptions = append(a.globalRoutesOptions, opts...)
	}
}

// WithGlobalGroupRoutesOptions sets the global group routes options
func WithGlobalGroupRoutesOptions(opts ...forge.GroupOption) Option {
	return func(a *Auth) {
		a.globalGroupRoutesOptions = append(a.globalGroupRoutesOptions, opts...)
	}
}

// WithCORSEnabled enables or disables CORS middleware
// When enabled, uses TrustedOrigins for allowed origins
// Default: false (disabled - let Forge or your app handle CORS)
func WithCORSEnabled(enabled bool) Option {
	return func(a *Auth) {
		a.config.CORSEnabled = enabled
	}
}

// WithTrustedOrigins sets trusted origins for CORS
// Setting origins does NOT automatically enable CORS - use WithCORSEnabled(true)
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

// WithDatabaseSchema sets the PostgreSQL schema for AuthSome tables
// This allows organizational separation of auth tables from application tables
// Example: WithDatabaseSchema("auth") creates tables in the "auth" schema
// Default: "" (uses database default, typically "public")
// Note: Schema must be valid SQL identifier; will be created if it doesn't exist
func WithDatabaseSchema(schema string) Option {
	return func(a *Auth) {
		a.config.DatabaseSchema = schema
	}
}

// WithGlobalCookieConfig sets the global cookie configuration for session management
// This configuration applies to all apps unless overridden at the app level
// Example:
//
//	WithGlobalCookieConfig(session.CookieConfig{
//	    Enabled:  true,
//	    Name:     "my_session",
//	    HttpOnly: true,
//	    SameSite: "Lax",
//	})
func WithGlobalCookieConfig(config session.CookieConfig) Option {
	return func(a *Auth) {
		a.config.SessionCookie = config
	}
}

// WithSessionCookieEnabled enables or disables cookie-based session management globally
// When enabled, authentication responses will automatically set secure HTTP cookies
func WithSessionCookieEnabled(enabled bool) Option {
	return func(a *Auth) {
		a.config.SessionCookie.Enabled = enabled
	}
}

// WithSessionCookieName sets the session cookie name
// Default: "authsome_session"
func WithSessionCookieName(name string) Option {
	return func(a *Auth) {
		a.config.SessionCookie.Name = name
		// Also set the deprecated field for backward compatibility
		a.config.SessionCookieName = name
	}
}

// WithAuthMiddlewareConfig sets the authentication middleware configuration
// This controls how the global authentication middleware behaves, including:
// - Session cookie name
// - Optional authentication (allow unauthenticated requests)
// - API key authentication settings
// - Context resolution (app/environment from headers or API key)
//
// Example:
//
//	WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
//	    SessionCookieName:   "my_session",
//	    Optional:            true,
//	    AllowAPIKeyInQuery:  false, // Security best practice
//	    AllowSessionInQuery: false, // Security best practice
//	    Context: middleware.ContextConfig{
//	        AutoDetectFromAPIKey: true,
//	        AutoDetectFromConfig: true,
//	    },
//	})
func WithAuthMiddlewareConfig(config middleware.AuthMiddlewareConfig) Option {
	return func(a *Auth) {
		a.authMiddlewareConfig = config
	}
}
