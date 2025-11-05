package notification

import (
	"time"
)

// Config holds the notification plugin configuration
type Config struct {
	// AddDefaultTemplates automatically adds default templates on startup
	AddDefaultTemplates bool `json:"add_default_templates" yaml:"add_default_templates"`

	// DefaultLanguage is the default language for templates
	DefaultLanguage string `json:"default_language" yaml:"default_language"`

	// AllowOrgOverrides allows organizations to override default templates in SaaS mode
	AllowOrgOverrides bool `json:"allow_org_overrides" yaml:"allow_org_overrides"`

	// AutoSendWelcome automatically sends welcome email on user signup
	AutoSendWelcome bool `json:"auto_send_welcome" yaml:"auto_send_welcome"`

	// RetryAttempts is the number of retry attempts for failed notifications
	RetryAttempts int `json:"retry_attempts" yaml:"retry_attempts"`

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration `json:"retry_delay" yaml:"retry_delay"`

	// CleanupAfter is the duration after which old notifications are deleted
	CleanupAfter time.Duration `json:"cleanup_after" yaml:"cleanup_after"`

	// RateLimits defines rate limits for notification sending
	RateLimits map[string]RateLimit `json:"rate_limits" yaml:"rate_limits"`

	// Providers configuration for email and SMS
	Providers ProvidersConfig `json:"providers" yaml:"providers"`
}

// RateLimit defines rate limiting configuration
type RateLimit struct {
	MaxRequests int           `json:"max_requests" yaml:"max_requests"`
	Window      time.Duration `json:"window" yaml:"window"`
}

// ProvidersConfig holds provider configurations
type ProvidersConfig struct {
	Email EmailProviderConfig `json:"email" yaml:"email"`
	SMS   SMSProviderConfig   `json:"sms" yaml:"sms"`
}

// EmailProviderConfig holds email provider configuration
type EmailProviderConfig struct {
	Provider string                 `json:"provider" yaml:"provider"` // smtp, sendgrid, ses, etc.
	From     string                 `json:"from" yaml:"from"`
	FromName string                 `json:"from_name" yaml:"from_name"`
	ReplyTo  string                 `json:"reply_to" yaml:"reply_to"`
	Config   map[string]interface{} `json:"config" yaml:"config"`
}

// SMSProviderConfig holds SMS provider configuration
type SMSProviderConfig struct {
	Provider string                 `json:"provider" yaml:"provider"` // twilio, vonage, aws-sns, etc.
	From     string                 `json:"from" yaml:"from"`
	Config   map[string]interface{} `json:"config" yaml:"config"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		AddDefaultTemplates: true,
		DefaultLanguage:     "en",
		AllowOrgOverrides:   false,
		AutoSendWelcome:     true,
		RetryAttempts:       3,
		RetryDelay:          5 * time.Minute,
		CleanupAfter:        30 * 24 * time.Hour, // 30 days
		RateLimits: map[string]RateLimit{
			"email": {
				MaxRequests: 100,
				Window:      time.Hour,
			},
			"sms": {
				MaxRequests: 50,
				Window:      time.Hour,
			},
		},
		Providers: ProvidersConfig{
			Email: EmailProviderConfig{
				Provider: "smtp",
				From:     "noreply@example.com",
				FromName: "AuthSome",
			},
			SMS: SMSProviderConfig{
				Provider: "twilio",
			},
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.RetryAttempts < 0 {
		c.RetryAttempts = 3
	}
	if c.RetryDelay == 0 {
		c.RetryDelay = 5 * time.Minute
	}
	if c.CleanupAfter == 0 {
		c.CleanupAfter = 30 * 24 * time.Hour
	}
	if c.DefaultLanguage == "" {
		c.DefaultLanguage = "en"
	}
	return nil
}
