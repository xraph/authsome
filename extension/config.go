package extension

import (
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/middleware"
	"github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/plugins"
)

// Config holds the configuration for the AuthSome extension
type Config struct {
	// DisableOpenAPI disables the OpenAPI documentation
	DisableOpenAPI bool `yaml:"disableOpenAPI" json:"disableOpenAPI"`

	// BasePath is the base path where auth routes are mounted
	BasePath string `yaml:"basePath" json:"basePath"`

	// Database configuration - mutually exclusive options
	// Database is a direct database connection (takes precedence)
	Database interface{} `yaml:"-" json:"-"`
	// DatabaseName is the name of the database to use from DatabaseManager
	DatabaseName string `yaml:"databaseName" json:"databaseName"`

	// CORS configuration
	CORSEnabled    bool     `yaml:"corsEnabled" json:"corsEnabled"`
	TrustedOrigins []string `yaml:"trustedOrigins" json:"trustedOrigins"`

	// Secret for signing tokens
	Secret string `yaml:"secret" json:"secret"`

	// RBACEnforce enables handler-level RBAC enforcement
	RBACEnforce bool `yaml:"rbacEnforce" json:"rbacEnforce"`

	// SecurityConfig for IP/country restrictions
	SecurityConfig *security.Config `yaml:"security" json:"security"`

	// RateLimitConfig for rate limiting
	RateLimitConfig *ratelimit.Config `yaml:"rateLimit" json:"rateLimit"`

	// RateLimitStorage is the storage backend for rate limiting
	RateLimitStorage ratelimit.Storage `yaml:"-" json:"-"`

	// GeoIPProvider for country-based restrictions
	GeoIPProvider security.GeoIPProvider `yaml:"-" json:"-"`

	// SessionCookie configures cookie-based session management
	SessionCookie *session.CookieConfig `yaml:"sessionCookie" json:"sessionCookie"`

	// AuthMiddlewareConfig configures the authentication middleware behavior
	AuthMiddlewareConfig *middleware.AuthMiddlewareConfig `yaml:"authMiddleware" json:"authMiddleware"`

	// Plugins to register with AuthSome
	Plugins []plugins.Plugin `yaml:"-" json:"-"`

	// RequireConfig determines if configuration must be loaded from file
	RequireConfig bool `yaml:"-" json:"-"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		BasePath:       "/api/auth",
		CORSEnabled:    false, // Disabled by default - let Forge handle it
		RBACEnforce:    false,
		RequireConfig:  false,
		DisableOpenAPI: false,
	}
}

// ConfigOption is a functional option for configuring the extension
type ConfigOption func(*Config)

// WithBasePath sets the base path for routes
func WithBasePath(path string) ConfigOption {
	return func(c *Config) {
		c.BasePath = path
	}
}

// WithDatabase sets a direct database connection
func WithDatabase(db *bun.DB) ConfigOption {
	return func(c *Config) {
		c.Database = db
	}
}

// WithDatabaseName sets the database name to use from DatabaseManager
func WithDatabaseName(name string) ConfigOption {
	return func(c *Config) {
		c.DatabaseName = name
	}
}

// WithCORSEnabled enables or disables CORS middleware
func WithCORSEnabled(enabled bool) ConfigOption {
	return func(c *Config) {
		c.CORSEnabled = enabled
	}
}

// WithTrustedOrigins sets trusted origins for CORS and auto-enables CORS if origins provided
func WithTrustedOrigins(origins []string) ConfigOption {
	return func(c *Config) {
		c.TrustedOrigins = origins
		// Auto-enable CORS if origins are provided
		if len(origins) > 0 {
			c.CORSEnabled = true
		}
	}
}

// WithSecret sets the secret for token signing
func WithSecret(secret string) ConfigOption {
	return func(c *Config) {
		c.Secret = secret
	}
}

// WithRBACEnforcement enables/disables RBAC enforcement
func WithRBACEnforcement(enabled bool) ConfigOption {
	return func(c *Config) {
		c.RBACEnforce = enabled
	}
}

// WithSecurityConfig sets security configuration
func WithSecurityConfig(config security.Config) ConfigOption {
	return func(c *Config) {
		c.SecurityConfig = &config
	}
}

// WithRateLimitConfig sets rate limit configuration
func WithRateLimitConfig(config ratelimit.Config) ConfigOption {
	return func(c *Config) {
		c.RateLimitConfig = &config
	}
}

// WithRateLimitStorage sets the rate limit storage backend
func WithRateLimitStorage(storage ratelimit.Storage) ConfigOption {
	return func(c *Config) {
		c.RateLimitStorage = storage
	}
}

// WithGeoIPProvider sets the GeoIP provider
func WithGeoIPProvider(provider security.GeoIPProvider) ConfigOption {
	return func(c *Config) {
		c.GeoIPProvider = provider
	}
}

// WithPlugins sets the plugins to register
func WithPlugins(plugins ...plugins.Plugin) ConfigOption {
	return func(c *Config) {
		c.Plugins = append(c.Plugins, plugins...)
	}
}

// WithRequireConfig sets whether configuration must be loaded from file
func WithRequireConfig(require bool) ConfigOption {
	return func(c *Config) {
		c.RequireConfig = require
	}
}

// WithConfig sets the entire configuration
func WithConfig(config Config) ConfigOption {
	return func(c *Config) {
		*c = config
	}
}

func WithDisableOpenAPI(disable bool) ConfigOption {
	return func(c *Config) {
		c.DisableOpenAPI = disable
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
func WithGlobalCookieConfig(config session.CookieConfig) ConfigOption {
	return func(c *Config) {
		c.SessionCookie = &config
	}
}

// WithSessionCookieEnabled enables or disables cookie-based session management globally
// When enabled, authentication responses will automatically set secure HTTP cookies
func WithSessionCookieEnabled(enabled bool) ConfigOption {
	return func(c *Config) {
		if c.SessionCookie == nil {
			c.SessionCookie = &session.CookieConfig{}
		}
		c.SessionCookie.Enabled = enabled
	}
}

// WithSessionCookieName sets the session cookie name
// Default: "authsome_session"
func WithSessionCookieName(name string) ConfigOption {
	return func(c *Config) {
		if c.SessionCookie == nil {
			c.SessionCookie = &session.CookieConfig{}
		}
		c.SessionCookie.Name = name
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
func WithAuthMiddlewareConfig(config middleware.AuthMiddlewareConfig) ConfigOption {
	return func(c *Config) {
		c.AuthMiddlewareConfig = &config
	}
}
