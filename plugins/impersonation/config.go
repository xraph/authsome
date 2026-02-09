package impersonation

import (
	"time"
)

// Config holds the impersonation plugin configuration.
type Config struct {
	// Time limits
	DefaultDurationMinutes int `json:"default_duration_minutes" yaml:"default_duration_minutes"`
	MaxDurationMinutes     int `json:"max_duration_minutes"     yaml:"max_duration_minutes"`
	MinDurationMinutes     int `json:"min_duration_minutes"     yaml:"min_duration_minutes"`

	// Security
	RequireReason   bool `json:"require_reason"    yaml:"require_reason"`
	RequireTicket   bool `json:"require_ticket"    yaml:"require_ticket"`
	MinReasonLength int  `json:"min_reason_length" yaml:"min_reason_length"`

	// RBAC
	RequirePermission     bool   `json:"require_permission"     yaml:"require_permission"`
	ImpersonatePermission string `json:"impersonate_permission" yaml:"impersonate_permission"`

	// Audit
	AuditAllActions bool `json:"audit_all_actions" yaml:"audit_all_actions"`

	// Auto-cleanup
	AutoCleanupEnabled bool          `json:"auto_cleanup_enabled" yaml:"auto_cleanup_enabled"`
	CleanupInterval    time.Duration `json:"cleanup_interval"     yaml:"cleanup_interval"`

	// UI Indicator
	ShowIndicator    bool   `json:"show_indicator"    yaml:"show_indicator"` // Show banner in UI
	IndicatorMessage string `json:"indicator_message" yaml:"indicator_message"`

	// Webhooks
	WebhookOnStart bool     `json:"webhook_on_start" yaml:"webhook_on_start"`
	WebhookOnEnd   bool     `json:"webhook_on_end"   yaml:"webhook_on_end"`
	WebhookURLs    []string `json:"webhook_urls"     yaml:"webhook_urls"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		// Time limits
		DefaultDurationMinutes: 30,
		MaxDurationMinutes:     480, // 8 hours
		MinDurationMinutes:     1,

		// Security
		RequireReason:   true,
		RequireTicket:   false,
		MinReasonLength: 10,

		// RBAC
		RequirePermission:     true,
		ImpersonatePermission: "impersonate:user",

		// Audit
		AuditAllActions: true,

		// Auto-cleanup
		AutoCleanupEnabled: true,
		CleanupInterval:    15 * time.Minute,

		// UI Indicator
		ShowIndicator:    true,
		IndicatorMessage: "⚠️ You are currently impersonating another user",

		// Webhooks
		WebhookOnStart: true,
		WebhookOnEnd:   true,
		WebhookURLs:    []string{},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Set defaults if zero values
	if c.DefaultDurationMinutes == 0 {
		c.DefaultDurationMinutes = 30
	}

	if c.MaxDurationMinutes == 0 {
		c.MaxDurationMinutes = 480
	}

	if c.MinDurationMinutes == 0 {
		c.MinDurationMinutes = 1
	}

	if c.MinReasonLength == 0 {
		c.MinReasonLength = 10
	}

	if c.ImpersonatePermission == "" {
		c.ImpersonatePermission = "impersonate:user"
	}

	if c.CleanupInterval == 0 {
		c.CleanupInterval = 15 * time.Minute
	}

	if c.IndicatorMessage == "" {
		c.IndicatorMessage = "⚠️ You are currently impersonating another user"
	}

	return nil
}
