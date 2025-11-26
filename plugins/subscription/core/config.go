package core

import (
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
)

// Config holds the subscription plugin configuration
type Config struct {
	// General settings
	Enabled             bool `json:"enabled"`
	RequireSubscription bool `json:"requireSubscription"` // Require subscription to create org

	// Trial settings
	DefaultTrialDays  int      `json:"defaultTrialDays"`
	TrialAllowedPlans []string `json:"trialAllowedPlans"` // Plan slugs that allow trials

	// Grace period
	GracePeriodDays int `json:"gracePeriodDays"` // Days before suspending on failed payment

	// Seat management
	AutoSyncSeats bool `json:"autoSyncSeats"` // Auto-update quantity based on org members

	// Provider configuration
	Provider     string       `json:"provider"` // stripe, paddle, etc.
	StripeConfig StripeConfig `json:"stripe"`
	// Future providers:
	// PaddleConfig   PaddleConfig `json:"paddle"`

	// Webhook settings
	WebhookTolerance int `json:"webhookTolerance"` // Seconds tolerance for webhook timestamp

	// Metering settings
	UsageReportBatchSize int `json:"usageReportBatchSize"` // Batch size for usage reporting
	UsageReportInterval  int `json:"usageReportInterval"`  // Seconds between usage reports
}

// StripeConfig holds Stripe-specific configuration
type StripeConfig struct {
	SecretKey      string `json:"secretKey"`
	WebhookSecret  string `json:"webhookSecret"`
	PublishableKey string `json:"publishableKey"`
	APIVersion     string `json:"apiVersion"`     // Optional: specific API version
	ConnectAccount string `json:"connectAccount"` // Optional: for Stripe Connect
}

// DefaultConfig returns the default plugin configuration
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		RequireSubscription:  false,
		DefaultTrialDays:     14,
		TrialAllowedPlans:    []string{},
		GracePeriodDays:      7,
		AutoSyncSeats:        false,
		Provider:             "stripe",
		WebhookTolerance:     300, // 5 minutes
		UsageReportBatchSize: 100,
		UsageReportInterval:  3600, // 1 hour
		StripeConfig: StripeConfig{
			APIVersion: "2024-06-20",
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.DefaultTrialDays < 0 {
		return suberrors.ErrInvalidBillingInterval
	}
	if c.GracePeriodDays < 0 {
		return suberrors.ErrInvalidBillingInterval
	}
	if c.Provider == "stripe" && c.StripeConfig.SecretKey == "" {
		// Allow empty for development/testing
	}
	return nil
}

// IsStripeConfigured returns true if Stripe is properly configured
func (c *Config) IsStripeConfigured() bool {
	return c.Provider == "stripe" && c.StripeConfig.SecretKey != ""
}

// GetWebhookSecret returns the appropriate webhook secret based on provider
func (c *Config) GetWebhookSecret() string {
	switch c.Provider {
	case "stripe":
		return c.StripeConfig.WebhookSecret
	default:
		return ""
	}
}

