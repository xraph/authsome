package authsome

import (
	rl "github.com/xraph/authsome/core/ratelimit"
	sec "github.com/xraph/authsome/core/security"
	"github.com/xraph/forge"
	forgedb "github.com/xraph/forge/extensions/database"
)

// Option is a function that configures Auth
type Option func(*Auth)

// WithMode sets the operation mode
func WithMode(mode Mode) Option {
	return func(a *Auth) {
		a.config.Mode = mode
	}
}

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
