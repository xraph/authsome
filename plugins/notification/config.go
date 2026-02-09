package notification

import (
	"time"
)

// Config holds the notification plugin configuration.
type Config struct {
	// AddDefaultTemplates automatically adds default templates on startup
	AddDefaultTemplates bool `json:"add_default_templates" yaml:"add_default_templates"`

	// DefaultLanguage is the default language for templates
	DefaultLanguage string `json:"default_language" yaml:"default_language"`

	// AllowAppOverrides allows apps to override default templates in SaaS mode
	AllowAppOverrides bool `json:"allow_app_overrides" yaml:"allow_app_overrides"`

	// AutoPopulateTemplates creates default templates for new apps
	AutoPopulateTemplates bool `json:"auto_populate_templates" yaml:"auto_populate_templates"`

	// AllowTemplateReset enables template reset functionality
	AllowTemplateReset bool `json:"allow_template_reset" yaml:"allow_template_reset"`

	// AutoSendWelcome automatically sends welcome email on user signup
	// DEPRECATED: Use AutoSend.Auth.Welcome instead
	AutoSendWelcome bool `json:"auto_send_welcome" yaml:"auto_send_welcome"`

	// AutoSend configuration for automatic notification sending
	AutoSend AutoSendConfig `json:"auto_send" yaml:"auto_send"`

	// AppName is the default application name used in notifications
	// If empty, will use the App name from the database
	AppName string `json:"app_name" yaml:"app_name"`

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

	// Async configuration for notification processing
	Async AsyncConfig `json:"async" yaml:"async"`
}

// AsyncConfig controls asynchronous notification processing.
type AsyncConfig struct {
	// Enabled enables async processing for non-critical notifications
	Enabled bool `json:"enabled" yaml:"enabled"`
	// WorkerPoolSize is the number of workers for async processing
	WorkerPoolSize int `json:"worker_pool_size" yaml:"worker_pool_size"`
	// QueueSize is the buffer size for async queues
	QueueSize int `json:"queue_size" yaml:"queue_size"`
	// RetryEnabled enables retry for failed notifications
	RetryEnabled bool `json:"retry_enabled" yaml:"retry_enabled"`
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `json:"max_retries" yaml:"max_retries"`
	// RetryBackoff are the delays between retries (e.g., ["1m", "5m", "15m"])
	RetryBackoff []string `json:"retry_backoff" yaml:"retry_backoff"`
	// PersistFailures persists permanently failed notifications to DB
	PersistFailures bool `json:"persist_failures" yaml:"persist_failures"`
}

// AutoSendConfig controls automatic notification sending for lifecycle events.
type AutoSendConfig struct {
	Auth         AuthAutoSendConfig         `json:"auth"         yaml:"auth"`
	Organization OrganizationAutoSendConfig `json:"organization" yaml:"organization"`
	Session      SessionAutoSendConfig      `json:"session"      yaml:"session"`
	Account      AccountAutoSendConfig      `json:"account"      yaml:"account"`
}

// AuthAutoSendConfig controls authentication-related notifications.
type AuthAutoSendConfig struct {
	Welcome           bool `json:"welcome"            yaml:"welcome"`
	VerificationEmail bool `json:"verification_email" yaml:"verification_email"`
	MagicLink         bool `json:"magic_link"         yaml:"magic_link"`
	EmailOTP          bool `json:"email_otp"          yaml:"email_otp"`
	MFACode           bool `json:"mfa_code"           yaml:"mfa_code"`
	PasswordReset     bool `json:"password_reset"     yaml:"password_reset"`
}

// OrganizationAutoSendConfig controls organization-related notifications.
type OrganizationAutoSendConfig struct {
	Invite        bool `json:"invite"         yaml:"invite"`
	MemberAdded   bool `json:"member_added"   yaml:"member_added"`
	MemberRemoved bool `json:"member_removed" yaml:"member_removed"`
	RoleChanged   bool `json:"role_changed"   yaml:"role_changed"`
	Transfer      bool `json:"transfer"       yaml:"transfer"`
	Deleted       bool `json:"deleted"        yaml:"deleted"`
	MemberLeft    bool `json:"member_left"    yaml:"member_left"`
}

// SessionAutoSendConfig controls session/device security notifications.
type SessionAutoSendConfig struct {
	NewDevice       bool `json:"new_device"       yaml:"new_device"`
	NewLocation     bool `json:"new_location"     yaml:"new_location"`
	SuspiciousLogin bool `json:"suspicious_login" yaml:"suspicious_login"`
	DeviceRemoved   bool `json:"device_removed"   yaml:"device_removed"`
	AllRevoked      bool `json:"all_revoked"      yaml:"all_revoked"`
}

// AccountAutoSendConfig controls account lifecycle notifications.
type AccountAutoSendConfig struct {
	EmailChangeRequest bool `json:"email_change_request" yaml:"email_change_request"`
	EmailChanged       bool `json:"email_changed"        yaml:"email_changed"`
	PasswordChanged    bool `json:"password_changed"     yaml:"password_changed"`
	UsernameChanged    bool `json:"username_changed"     yaml:"username_changed"`
	Deleted            bool `json:"deleted"              yaml:"deleted"`
	Suspended          bool `json:"suspended"            yaml:"suspended"`
	Reactivated        bool `json:"reactivated"          yaml:"reactivated"`
}

// RateLimit defines rate limiting configuration.
type RateLimit struct {
	MaxRequests int           `json:"max_requests" yaml:"max_requests"`
	Window      time.Duration `json:"window"       yaml:"window"`
}

// ProvidersConfig holds provider configurations.
type ProvidersConfig struct {
	Email EmailProviderConfig `json:"email"         yaml:"email"`
	SMS   *SMSProviderConfig  `json:"sms,omitempty" yaml:"sms,omitempty"` // Optional SMS provider
}

// EmailProviderConfig holds email provider configuration.
type EmailProviderConfig struct {
	Provider string         `json:"provider"  yaml:"provider"` // smtp, sendgrid, ses, etc.
	From     string         `json:"from"      yaml:"from"`
	FromName string         `json:"from_name" yaml:"from_name"`
	ReplyTo  string         `json:"reply_to"  yaml:"reply_to"`
	Config   map[string]any `json:"config"    yaml:"config"`
}

// SMSProviderConfig holds SMS provider configuration.
type SMSProviderConfig struct {
	Provider string         `json:"provider" yaml:"provider"` // twilio, vonage, aws-sns, etc.
	From     string         `json:"from"     yaml:"from"`
	Config   map[string]any `json:"config"   yaml:"config"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		AddDefaultTemplates:   true,
		DefaultLanguage:       "en",
		AllowAppOverrides:     false,
		AutoPopulateTemplates: true,
		AllowTemplateReset:    true,
		AutoSendWelcome:       true, // DEPRECATED: use AutoSend.Auth.Welcome
		AutoSend: AutoSendConfig{
			Auth: AuthAutoSendConfig{
				Welcome:           true,  // Send welcome email on signup
				VerificationEmail: false, // Let emailverification plugin control this
				MagicLink:         false, // Let magiclink plugin control this
				EmailOTP:          false, // Let emailotp plugin control this
				MFACode:           false, // Let mfa plugin control this
				PasswordReset:     false, // Let password reset handler control this
			},
			Organization: OrganizationAutoSendConfig{
				Invite:        true, // Send invitation emails
				MemberAdded:   true, // Notify on member addition
				MemberRemoved: true, // Notify on member removal
				RoleChanged:   true, // Notify on role changes
				Transfer:      true, // Notify on ownership transfer
				Deleted:       true, // Notify on organization deletion
				MemberLeft:    true, // Notify when member leaves
			},
			Session: SessionAutoSendConfig{
				NewDevice:       true, // Notify on new device login
				NewLocation:     true, // Notify on new location login
				SuspiciousLogin: true, // Notify on suspicious activity
				DeviceRemoved:   true, // Notify when device removed
				AllRevoked:      true, // Notify on mass signout
			},
			Account: AccountAutoSendConfig{
				EmailChangeRequest: true, // Send confirmation for email change
				EmailChanged:       true, // Notify on email change completion
				PasswordChanged:    true, // Notify on password change
				UsernameChanged:    true, // Notify on username change
				Deleted:            true, // Notify on account deletion
				Suspended:          true, // Notify on account suspension
				Reactivated:        true, // Notify on account reactivation
			},
		},
		AppName:       "", // Empty means use app name from database
		RetryAttempts: 3,
		RetryDelay:    5 * time.Minute,
		CleanupAfter:  30 * 24 * time.Hour, // 30 days
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
			// SMS is optional - not configured by default
			SMS: nil,
		},
		Async: AsyncConfig{
			Enabled:         true,                        // Enable async by default
			WorkerPoolSize:  5,                           // 5 workers per priority
			QueueSize:       1000,                        // Buffer 1000 notifications
			RetryEnabled:    true,                        // Enable retry by default
			MaxRetries:      3,                           // 3 retry attempts
			RetryBackoff:    []string{"1m", "5m", "15m"}, // Exponential backoff
			PersistFailures: true,                        // Persist failed to DB
		},
	}
}

// Validate validates the configuration.
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
