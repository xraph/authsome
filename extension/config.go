package extension

import (
	"github.com/uptrace/bun"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/plugins"
)

// Config holds the configuration for the AuthSome extension
type Config struct {
	// Mode is the operation mode (Standalone or SaaS)
	Mode authsome.Mode `yaml:"mode" json:"mode"`

	// BasePath is the base path where auth routes are mounted
	BasePath string `yaml:"basePath" json:"basePath"`

	// Database configuration - mutually exclusive options
	// Database is a direct database connection (takes precedence)
	Database interface{} `yaml:"-" json:"-"`
	// DatabaseName is the name of the database to use from DatabaseManager
	DatabaseName string `yaml:"databaseName" json:"databaseName"`

	// TrustedOrigins for CORS
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

	// Plugins to register with AuthSome
	Plugins []plugins.Plugin `yaml:"-" json:"-"`

	// RequireConfig determines if configuration must be loaded from file
	RequireConfig bool `yaml:"-" json:"-"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		Mode:          authsome.ModeStandalone,
		BasePath:      "/api/auth",
		RBACEnforce:   false,
		RequireConfig: false,
	}
}

// ConfigOption is a functional option for configuring the extension
type ConfigOption func(*Config)

// WithMode sets the operation mode
func WithMode(mode authsome.Mode) ConfigOption {
	return func(c *Config) {
		c.Mode = mode
	}
}

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

// WithTrustedOrigins sets trusted origins for CORS
func WithTrustedOrigins(origins []string) ConfigOption {
	return func(c *Config) {
		c.TrustedOrigins = origins
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
