package stepup

import "time"

// SecurityLevel represents the strength of authentication required
type SecurityLevel string

const (
	SecurityLevelNone     SecurityLevel = "none"     // No step-up required
	SecurityLevelLow      SecurityLevel = "low"      // Recent login (within session)
	SecurityLevelMedium   SecurityLevel = "medium"   // Re-auth within time window
	SecurityLevelHigh     SecurityLevel = "high"     // Strong re-auth (password + 2FA)
	SecurityLevelCritical SecurityLevel = "critical" // Biometric or hardware token
)

// VerificationMethod represents acceptable verification methods
type VerificationMethod string

const (
	MethodPassword   VerificationMethod = "password"    // Password verification
	MethodTOTP       VerificationMethod = "totp"        // Time-based OTP
	MethodSMS        VerificationMethod = "sms"         // SMS code
	MethodEmail      VerificationMethod = "email"       // Email code
	MethodBiometric  VerificationMethod = "biometric"   // Biometric verification
	MethodWebAuthn   VerificationMethod = "webauthn"    // WebAuthn/FIDO2
	MethodBackupCode VerificationMethod = "backup_code" // Backup codes
)

// Config holds the step-up authentication plugin configuration
type Config struct {
	// Global settings
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Time windows for different security levels
	MediumAuthWindow   time.Duration `json:"medium_auth_window" yaml:"medium_auth_window"`     // e.g., 15 minutes
	HighAuthWindow     time.Duration `json:"high_auth_window" yaml:"high_auth_window"`         // e.g., 5 minutes
	CriticalAuthWindow time.Duration `json:"critical_auth_window" yaml:"critical_auth_window"` // e.g., immediate

	// Verification method requirements per security level
	LowMethods      []VerificationMethod `json:"low_methods" yaml:"low_methods"`
	MediumMethods   []VerificationMethod `json:"medium_methods" yaml:"medium_methods"`
	HighMethods     []VerificationMethod `json:"high_methods" yaml:"high_methods"`
	CriticalMethods []VerificationMethod `json:"critical_methods" yaml:"critical_methods"`

	// Rule configuration
	RouteRules     []RouteRule     `json:"route_rules" yaml:"route_rules"`
	AmountRules    []AmountRule    `json:"amount_rules" yaml:"amount_rules"`
	ResourceRules  []ResourceRule  `json:"resource_rules" yaml:"resource_rules"`
	TimeBasedRules []TimeBasedRule `json:"time_based_rules" yaml:"time_based_rules"`
	ContextRules   []ContextRule   `json:"context_rules" yaml:"context_rules"`

	// Grace periods and remembering
	RememberStepUp   bool          `json:"remember_step_up" yaml:"remember_step_up"`   // Remember step-up per device
	RememberDuration time.Duration `json:"remember_duration" yaml:"remember_duration"` // How long to remember
	GracePeriod      time.Duration `json:"grace_period" yaml:"grace_period"`           // Grace period after step-up

	// User experience
	AllowDegradation bool   `json:"allow_degradation" yaml:"allow_degradation"` // Allow graceful degradation
	PromptMessage    string `json:"prompt_message" yaml:"prompt_message"`       // Custom prompt message
	RedirectURL      string `json:"redirect_url" yaml:"redirect_url"`           // Where to redirect for step-up

	// Risk-based adaptive requirements
	RiskBasedEnabled    bool    `json:"risk_based_enabled" yaml:"risk_based_enabled"`       // Enable adaptive security
	RiskThresholdLow    float64 `json:"risk_threshold_low" yaml:"risk_threshold_low"`       // Risk score for low security
	RiskThresholdMedium float64 `json:"risk_threshold_medium" yaml:"risk_threshold_medium"` // Risk score for medium security
	RiskThresholdHigh   float64 `json:"risk_threshold_high" yaml:"risk_threshold_high"`     // Risk score for high security

	// Organization-scoped overrides
	EnableOrgOverrides bool `json:"enable_org_overrides" yaml:"enable_org_overrides"`

	// Audit and monitoring
	AuditEnabled bool     `json:"audit_enabled" yaml:"audit_enabled"`
	AuditEvents  []string `json:"audit_events" yaml:"audit_events"` // Events to audit

	// Webhook notifications
	WebhookEnabled bool     `json:"webhook_enabled" yaml:"webhook_enabled"`
	WebhookEvents  []string `json:"webhook_events" yaml:"webhook_events"` // Events to webhook
}

// RouteRule defines step-up requirements based on route patterns
type RouteRule struct {
	Pattern       string        `json:"pattern" yaml:"pattern"`               // Route pattern (supports wildcards)
	Method        string        `json:"method" yaml:"method"`                 // HTTP method (GET, POST, etc.)
	SecurityLevel SecurityLevel `json:"security_level" yaml:"security_level"` // Required security level
	Description   string        `json:"description" yaml:"description"`       // Human-readable description
	OrgID         string        `json:"org_id" yaml:"org_id"`                 // Organization-specific (empty for global)
}

// AmountRule defines step-up requirements based on monetary amounts
type AmountRule struct {
	MinAmount     float64       `json:"min_amount" yaml:"min_amount"`         // Minimum amount threshold
	MaxAmount     float64       `json:"max_amount" yaml:"max_amount"`         // Maximum amount threshold (0 for unlimited)
	Currency      string        `json:"currency" yaml:"currency"`             // Currency code (USD, EUR, etc.)
	SecurityLevel SecurityLevel `json:"security_level" yaml:"security_level"` // Required security level
	Description   string        `json:"description" yaml:"description"`       // Human-readable description
	OrgID         string        `json:"org_id" yaml:"org_id"`                 // Organization-specific
}

// ResourceRule defines step-up requirements based on resource types
type ResourceRule struct {
	ResourceType  string        `json:"resource_type" yaml:"resource_type"`   // Resource type (user, payment, etc.)
	Action        string        `json:"action" yaml:"action"`                 // Action (read, update, delete)
	SecurityLevel SecurityLevel `json:"security_level" yaml:"security_level"` // Required security level
	Sensitivity   string        `json:"sensitivity" yaml:"sensitivity"`       // Sensitivity classification
	Description   string        `json:"description" yaml:"description"`       // Human-readable description
	OrgID         string        `json:"org_id" yaml:"org_id"`                 // Organization-specific
}

// TimeBasedRule defines step-up requirements based on time elapsed
type TimeBasedRule struct {
	Operation     string        `json:"operation" yaml:"operation"`           // Operation name
	MaxAge        time.Duration `json:"max_age" yaml:"max_age"`               // Maximum age of authentication
	SecurityLevel SecurityLevel `json:"security_level" yaml:"security_level"` // Required security level
	Description   string        `json:"description" yaml:"description"`       // Human-readable description
	OrgID         string        `json:"org_id" yaml:"org_id"`                 // Organization-specific
}

// ContextRule defines step-up requirements based on contextual factors
type ContextRule struct {
	Name          string        `json:"name" yaml:"name"`                     // Rule name
	Condition     string        `json:"condition" yaml:"condition"`           // Condition expression (CEL or simple)
	SecurityLevel SecurityLevel `json:"security_level" yaml:"security_level"` // Required security level
	Description   string        `json:"description" yaml:"description"`       // Human-readable description
	OrgID         string        `json:"org_id" yaml:"org_id"`                 // Organization-specific
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,

		// Time windows
		MediumAuthWindow:   15 * time.Minute,
		HighAuthWindow:     5 * time.Minute,
		CriticalAuthWindow: 0, // Immediate

		// Verification methods per level
		LowMethods:      []VerificationMethod{MethodPassword},
		MediumMethods:   []VerificationMethod{MethodPassword},
		HighMethods:     []VerificationMethod{MethodPassword, MethodTOTP},
		CriticalMethods: []VerificationMethod{MethodPassword, MethodWebAuthn},

		// Default route rules
		RouteRules: []RouteRule{
			{
				Pattern:       "/api/user/email",
				Method:        "PUT",
				SecurityLevel: SecurityLevelMedium,
				Description:   "Changing email requires re-authentication",
			},
			{
				Pattern:       "/api/user/password",
				Method:        "PUT",
				SecurityLevel: SecurityLevelHigh,
				Description:   "Changing password requires strong authentication",
			},
			{
				Pattern:       "/api/payment/*",
				Method:        "POST",
				SecurityLevel: SecurityLevelMedium,
				Description:   "Payment operations require verification",
			},
		},

		// Default amount rules
		AmountRules: []AmountRule{
			{
				MinAmount:     0,
				MaxAmount:     1000,
				Currency:      "USD",
				SecurityLevel: SecurityLevelMedium,
				Description:   "Amounts under $1,000 require medium security",
			},
			{
				MinAmount:     1000,
				MaxAmount:     10000,
				Currency:      "USD",
				SecurityLevel: SecurityLevelHigh,
				Description:   "Amounts $1,000-$10,000 require high security",
			},
			{
				MinAmount:     10000,
				MaxAmount:     0, // Unlimited
				Currency:      "USD",
				SecurityLevel: SecurityLevelCritical,
				Description:   "Amounts over $10,000 require critical security",
			},
		},

		// Default resource rules
		ResourceRules: []ResourceRule{
			{
				ResourceType:  "user",
				Action:        "delete",
				SecurityLevel: SecurityLevelHigh,
				Sensitivity:   "high",
				Description:   "Deleting user account requires high security",
			},
			{
				ResourceType:  "settings",
				Action:        "update",
				SecurityLevel: SecurityLevelMedium,
				Sensitivity:   "medium",
				Description:   "Updating security settings requires verification",
			},
		},

		// Grace periods
		RememberStepUp:   true,
		RememberDuration: 24 * time.Hour,
		GracePeriod:      30 * time.Second,

		// User experience
		AllowDegradation: true,
		PromptMessage:    "This action requires additional verification for your security.",
		RedirectURL:      "/auth/stepup",

		// Risk-based settings
		RiskBasedEnabled:    true,
		RiskThresholdLow:    0.3,
		RiskThresholdMedium: 0.6,
		RiskThresholdHigh:   0.8,

		// Multi-tenancy
		EnableOrgOverrides: true,

		// Audit
		AuditEnabled: true,
		AuditEvents: []string{
			"stepup.required",
			"stepup.initiated",
			"stepup.verified",
			"stepup.failed",
			"stepup.bypassed",
		},

		// Webhooks
		WebhookEnabled: false,
		WebhookEvents: []string{
			"stepup.required",
			"stepup.verified",
			"stepup.failed",
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Set defaults if values are zero
	if c.MediumAuthWindow == 0 {
		c.MediumAuthWindow = 15 * time.Minute
	}
	if c.HighAuthWindow == 0 {
		c.HighAuthWindow = 5 * time.Minute
	}
	if c.RememberDuration == 0 {
		c.RememberDuration = 24 * time.Hour
	}
	if c.GracePeriod == 0 {
		c.GracePeriod = 30 * time.Second
	}

	// Validate risk thresholds
	if c.RiskBasedEnabled {
		if c.RiskThresholdLow < 0 || c.RiskThresholdLow > 1 {
			c.RiskThresholdLow = 0.3
		}
		if c.RiskThresholdMedium < c.RiskThresholdLow || c.RiskThresholdMedium > 1 {
			c.RiskThresholdMedium = 0.6
		}
		if c.RiskThresholdHigh < c.RiskThresholdMedium || c.RiskThresholdHigh > 1 {
			c.RiskThresholdHigh = 0.8
		}
	}

	return nil
}
