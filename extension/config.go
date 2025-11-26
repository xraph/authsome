package extension

import (
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/middleware"
	"github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/validator"
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

	// SessionConfig configures session behavior (TTL, sliding window, refresh tokens)
	SessionConfig *session.Config `yaml:"sessionConfig" json:"sessionConfig"`

	// UserConfig configures user service behavior (password requirements, etc.)
	UserConfig *user.Config `yaml:"userConfig" json:"userConfig"`

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

// WithSessionCookieMaxAge sets the cookie MaxAge in seconds
// This controls how long the browser keeps the cookie
// If not set, defaults to session TTL (24 hours)
//
// Example:
//
//	extension.WithSessionCookieMaxAge(3600)  // 1 hour
//	extension.WithSessionCookieMaxAge(86400) // 24 hours
func WithSessionCookieMaxAge(seconds int) ConfigOption {
	return func(c *Config) {
		if c.SessionCookie == nil {
			c.SessionCookie = &session.CookieConfig{}
		}
		c.SessionCookie.MaxAge = &seconds
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

// WithSessionConfig sets the full session configuration
// This controls session behavior including TTL, sliding window, and refresh tokens
//
// Example:
//
//	extension.WithSessionConfig(session.Config{
//	    DefaultTTL:           24 * time.Hour,
//	    RememberTTL:          7 * 24 * time.Hour,
//	    EnableSlidingWindow:  true,
//	    SlidingRenewalAfter:  5 * time.Minute,
//	    EnableRefreshTokens:  true,
//	    RefreshTokenTTL:      30 * 24 * time.Hour,
//	    AccessTokenTTL:       15 * time.Minute,
//	})
func WithSessionConfig(config session.Config) ConfigOption {
	return func(c *Config) {
		c.SessionConfig = &config
	}
}

// WithSlidingWindowSessions enables automatic session renewal on each request
// When enabled, sessions are extended whenever the user makes a request
// The renewalThreshold determines how often to actually update the database (default: 5 minutes)
// This prevents logging out active users while minimizing database writes
//
// Example:
//
//	extension.WithSlidingWindowSessions(true, 5*time.Minute)
func WithSlidingWindowSessions(enabled bool, renewalThreshold ...time.Duration) ConfigOption {
	return func(c *Config) {
		if c.SessionConfig == nil {
			c.SessionConfig = &session.Config{}
		}
		c.SessionConfig.EnableSlidingWindow = enabled
		if len(renewalThreshold) > 0 {
			c.SessionConfig.SlidingRenewalAfter = renewalThreshold[0]
		}
	}
}

// WithRefreshTokens enables the refresh token pattern
// Short-lived access tokens are issued with long-lived refresh tokens
// Clients must explicitly refresh when access token expires
//
// Example:
//
//	extension.WithRefreshTokens(true, 15*time.Minute, 30*24*time.Hour)
//	// 15 min access tokens, 30 day refresh tokens
func WithRefreshTokens(enabled bool, accessTTL, refreshTTL time.Duration) ConfigOption {
	return func(c *Config) {
		if c.SessionConfig == nil {
			c.SessionConfig = &session.Config{}
		}
		c.SessionConfig.EnableRefreshTokens = enabled
		if accessTTL > 0 {
			c.SessionConfig.AccessTokenTTL = accessTTL
		}
		if refreshTTL > 0 {
			c.SessionConfig.RefreshTokenTTL = refreshTTL
		}
	}
}

// WithSessionTTL sets the default and "remember me" session TTL
//
// Example:
//
//	extension.WithSessionTTL(24*time.Hour, 7*24*time.Hour)
func WithSessionTTL(defaultTTL, rememberTTL time.Duration) ConfigOption {
	return func(c *Config) {
		if c.SessionConfig == nil {
			c.SessionConfig = &session.Config{}
		}
		if defaultTTL > 0 {
			c.SessionConfig.DefaultTTL = defaultTTL
		}
		if rememberTTL > 0 {
			c.SessionConfig.RememberTTL = rememberTTL
		}
	}
}

// WithUserConfig sets the full user configuration
// This controls user service behavior including password requirements
//
// Example:
//
//	extension.WithUserConfig(user.Config{
//	    PasswordRequirements: validator.PasswordRequirements{
//	        MinLength:      12,
//	        RequireUpper:   true,
//	        RequireLower:   true,
//	        RequireNumber:  true,
//	        RequireSpecial: true,
//	    },
//	})
func WithUserConfig(config user.Config) ConfigOption {
	return func(c *Config) {
		c.UserConfig = &config
	}
}

// WithPasswordRequirements sets the password requirements
// This controls password validation for user registration and password changes
//
// Example:
//
//	extension.WithPasswordRequirements(validator.PasswordRequirements{
//	    MinLength:      12,
//	    RequireUpper:   true,
//	    RequireLower:   true,
//	    RequireNumber:  true,
//	    RequireSpecial: true,
//	})
func WithPasswordRequirements(reqs validator.PasswordRequirements) ConfigOption {
	return func(c *Config) {
		if c.UserConfig == nil {
			c.UserConfig = &user.Config{}
		}
		c.UserConfig.PasswordRequirements = reqs
	}
}

// WithPasswordPolicy is a convenience function to set common password policies
// Predefined policies: "weak", "medium", "strong", "enterprise"
//
// Example:
//
//	extension.WithPasswordPolicy("strong")
func WithPasswordPolicy(policy string) ConfigOption {
	return func(c *Config) {
		if c.UserConfig == nil {
			c.UserConfig = &user.Config{}
		}

		switch policy {
		case "weak":
			c.UserConfig.PasswordRequirements = validator.PasswordRequirements{
				MinLength:      6,
				RequireUpper:   false,
				RequireLower:   false,
				RequireNumber:  false,
				RequireSpecial: false,
			}
		case "medium":
			c.UserConfig.PasswordRequirements = validator.PasswordRequirements{
				MinLength:      8,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  false,
				RequireSpecial: false,
			}
		case "strong":
			c.UserConfig.PasswordRequirements = validator.PasswordRequirements{
				MinLength:      10,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  true,
				RequireSpecial: true,
			}
		case "enterprise":
			c.UserConfig.PasswordRequirements = validator.PasswordRequirements{
				MinLength:      14,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  true,
				RequireSpecial: true,
			}
		default:
			// Default to medium
			c.UserConfig.PasswordRequirements = validator.PasswordRequirements{
				MinLength:      8,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  false,
				RequireSpecial: false,
			}
		}
	}
}

// WithMinPasswordLength sets the minimum password length
//
// Example:
//
//	extension.WithMinPasswordLength(12)
func WithMinPasswordLength(length int) ConfigOption {
	return func(c *Config) {
		if c.UserConfig == nil {
			c.UserConfig = &user.Config{}
		}
		c.UserConfig.PasswordRequirements.MinLength = length
	}
}
