package apikey

import (
	"time"
)

// Config holds the API key plugin configuration.
type Config struct {
	// Service configuration
	DefaultRateLimit int           `json:"default_rate_limit" yaml:"default_rate_limit"`
	MaxRateLimit     int           `json:"max_rate_limit"     yaml:"max_rate_limit"`
	DefaultExpiry    time.Duration `json:"default_expiry"     yaml:"default_expiry"`
	MaxKeysPerUser   int           `json:"max_keys_per_user"  yaml:"max_keys_per_user"`
	MaxKeysPerOrg    int           `json:"max_keys_per_org"   yaml:"max_keys_per_org"`
	KeyLength        int           `json:"key_length"         yaml:"key_length"`

	// Authentication configuration
	AllowQueryParam bool `json:"allow_query_param" yaml:"allow_query_param"` // Allow API key in query params (not recommended for production)

	// Rate limiting configuration
	RateLimiting RateLimitConfig `json:"rate_limiting" yaml:"rate_limiting"`

	// IP whitelisting
	IPWhitelisting IPWhitelistConfig `json:"ip_whitelisting" yaml:"ip_whitelisting"`

	// Webhook notifications
	Webhooks WebhookConfig `json:"webhooks" yaml:"webhooks"`

	// Cleanup scheduler
	AutoCleanup AutoCleanupConfig `json:"auto_cleanup" yaml:"auto_cleanup"`
}

// RateLimitConfig configures rate limiting behavior.
type RateLimitConfig struct {
	Enabled bool          `json:"enabled" yaml:"enabled"`
	Window  time.Duration `json:"window"  yaml:"window"` // Time window for rate limiting
}

// IPWhitelistConfig configures IP whitelisting.
type IPWhitelistConfig struct {
	Enabled    bool `json:"enabled"     yaml:"enabled"`
	StrictMode bool `json:"strict_mode" yaml:"strict_mode"` // Reject if IP not in allowlist
}

// WebhookConfig configures webhook notifications for API key events.
type WebhookConfig struct {
	Enabled           bool     `json:"enabled"              yaml:"enabled"`
	NotifyOnCreated   bool     `json:"notify_on_created"    yaml:"notify_on_created"`
	NotifyOnRotated   bool     `json:"notify_on_rotated"    yaml:"notify_on_rotated"`
	NotifyOnDeleted   bool     `json:"notify_on_deleted"    yaml:"notify_on_deleted"`
	NotifyOnRateLimit bool     `json:"notify_on_rate_limit" yaml:"notify_on_rate_limit"`
	NotifyOnExpiring  bool     `json:"notify_on_expiring"   yaml:"notify_on_expiring"` // Notify N days before expiry
	ExpiryWarningDays int      `json:"expiry_warning_days"  yaml:"expiry_warning_days"`
	WebhookURLs       []string `json:"webhook_urls"         yaml:"webhook_urls"`
}

// AutoCleanupConfig configures automatic cleanup of expired keys.
type AutoCleanupConfig struct {
	Enabled  bool          `json:"enabled"  yaml:"enabled"`
	Interval time.Duration `json:"interval" yaml:"interval"` // How often to run cleanup
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		// Service defaults
		DefaultRateLimit: 1000,
		MaxRateLimit:     10000,
		DefaultExpiry:    365 * 24 * time.Hour, // 1 year
		MaxKeysPerUser:   10,
		MaxKeysPerOrg:    100,
		KeyLength:        32,

		// Authentication defaults
		AllowQueryParam: false, // Disabled by default for security

		// Rate limiting defaults
		RateLimiting: RateLimitConfig{
			Enabled: true,
			Window:  time.Hour, // 1 hour window
		},

		// IP whitelisting defaults
		IPWhitelisting: IPWhitelistConfig{
			Enabled:    false,
			StrictMode: false,
		},

		// Webhook defaults
		Webhooks: WebhookConfig{
			Enabled:           false,
			NotifyOnCreated:   true,
			NotifyOnRotated:   true,
			NotifyOnDeleted:   true,
			NotifyOnRateLimit: true,
			NotifyOnExpiring:  true,
			ExpiryWarningDays: 7,
			WebhookURLs:       []string{},
		},

		// Auto cleanup defaults
		AutoCleanup: AutoCleanupConfig{
			Enabled:  true,
			Interval: 24 * time.Hour, // Run daily
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Set defaults if values are zero
	if c.DefaultRateLimit == 0 {
		c.DefaultRateLimit = 1000
	}

	if c.MaxRateLimit == 0 {
		c.MaxRateLimit = 10000
	}

	if c.DefaultExpiry == 0 {
		c.DefaultExpiry = 365 * 24 * time.Hour
	}

	if c.MaxKeysPerUser == 0 {
		c.MaxKeysPerUser = 10
	}

	if c.MaxKeysPerOrg == 0 {
		c.MaxKeysPerOrg = 100
	}

	if c.KeyLength == 0 {
		c.KeyLength = 32
	}

	if c.RateLimiting.Window == 0 {
		c.RateLimiting.Window = time.Hour
	}

	if c.Webhooks.ExpiryWarningDays == 0 {
		c.Webhooks.ExpiryWarningDays = 7
	}

	if c.AutoCleanup.Interval == 0 {
		c.AutoCleanup.Interval = 24 * time.Hour
	}

	return nil
}
