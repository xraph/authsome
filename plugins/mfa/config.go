package mfa

// Config holds MFA plugin configuration
type Config struct {
	// Global settings
	Enabled            bool `json:"enabled" default:"true"`
	RequireForAllUsers bool `json:"require_for_all_users" default:"false"`
	GracePeriodDays    int  `json:"grace_period_days" default:"7"`

	// Factor settings
	AllowedFactorTypes  []FactorType `json:"allowed_factor_types"`
	RequiredFactorCount int          `json:"required_factor_count" default:"1"`

	// TOTP settings
	TOTP TOTPConfig `json:"totp"`

	// SMS settings
	SMS SMSConfig `json:"sms"`

	// Email settings
	Email EmailConfig `json:"email"`

	// WebAuthn settings
	WebAuthn WebAuthnConfig `json:"webauthn"`

	// Backup codes settings
	BackupCodes BackupCodesConfig `json:"backup_codes"`

	// Trusted device settings
	TrustedDevices TrustedDevicesConfig `json:"trusted_devices"`

	// Challenge settings
	ChallengeExpiryMinutes int `json:"challenge_expiry_minutes" default:"5"`
	MaxAttempts            int `json:"max_attempts" default:"3"`

	// Rate limiting
	RateLimit RateLimitConfig `json:"rate_limit"`

	// Adaptive MFA
	AdaptiveMFA AdaptiveMFAConfig `json:"adaptive_mfa"`

	// Session settings
	SessionExpiryMinutes int `json:"session_expiry_minutes" default:"15"`
}

// TOTPConfig configures TOTP (Google Authenticator) settings
type TOTPConfig struct {
	Enabled    bool   `json:"enabled" default:"true"`
	Issuer     string `json:"issuer" default:"AuthSome"`
	Period     int    `json:"period" default:"30"` // Seconds
	Digits     int    `json:"digits" default:"6"`
	Algorithm  string `json:"algorithm" default:"SHA1"` // SHA1, SHA256, SHA512
	WindowSize int    `json:"window_size" default:"1"`  // Past/future periods to accept
}

// SMSConfig configures SMS verification settings
type SMSConfig struct {
	Enabled           bool             `json:"enabled" default:"true"`
	Provider          string           `json:"provider"` // "twilio", "vonage", etc.
	CodeLength        int              `json:"code_length" default:"6"`
	CodeExpiryMinutes int              `json:"code_expiry_minutes" default:"5"`
	TemplateID        string           `json:"template_id"`
	RateLimit         *RateLimitConfig `json:"rate_limit,omitempty"`
}

// EmailConfig configures email verification settings
type EmailConfig struct {
	Enabled           bool             `json:"enabled" default:"true"`
	Provider          string           `json:"provider"` // Email provider
	CodeLength        int              `json:"code_length" default:"6"`
	CodeExpiryMinutes int              `json:"code_expiry_minutes" default:"10"`
	TemplateID        string           `json:"template_id"`
	RateLimit         *RateLimitConfig `json:"rate_limit,omitempty"`
}

// WebAuthnConfig configures WebAuthn/FIDO2 settings
type WebAuthnConfig struct {
	Enabled                bool     `json:"enabled" default:"true"`
	RPDisplayName          string   `json:"rp_display_name" default:"AuthSome"`
	RPID                   string   `json:"rp_id"`                                 // e.g., "example.com"
	RPOrigins              []string `json:"rp_origins"`                            // Allowed origins
	AttestationPreference  string   `json:"attestation_preference" default:"none"` // none, indirect, direct
	AuthenticatorSelection struct {
		RequireResidentKey     bool   `json:"require_resident_key" default:"false"`
		ResidentKeyRequirement string `json:"resident_key_requirement" default:"preferred"` // discouraged, preferred, required
		UserVerification       string `json:"user_verification" default:"preferred"`        // discouraged, preferred, required
	} `json:"authenticator_selection"`
	Timeout int `json:"timeout" default:"60000"` // Milliseconds
}

// BackupCodesConfig configures backup recovery codes
type BackupCodesConfig struct {
	Enabled    bool   `json:"enabled" default:"true"`
	Count      int    `json:"count" default:"10"`
	Length     int    `json:"length" default:"8"`
	Format     string `json:"format" default:"XXXX-XXXX"` // Code format
	AllowReuse bool   `json:"allow_reuse" default:"false"`
}

// TrustedDevicesConfig configures trusted device settings
type TrustedDevicesConfig struct {
	Enabled           bool `json:"enabled" default:"true"`
	DefaultExpiryDays int  `json:"default_expiry_days" default:"30"`
	MaxExpiryDays     int  `json:"max_expiry_days" default:"90"`
	MaxDevicesPerUser int  `json:"max_devices_per_user" default:"5"`
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	Enabled        bool `json:"enabled" default:"true"`
	MaxAttempts    int  `json:"max_attempts" default:"5"`
	WindowMinutes  int  `json:"window_minutes" default:"15"`
	LockoutMinutes int  `json:"lockout_minutes" default:"30"`
}

// AdaptiveMFAConfig configures risk-based authentication
type AdaptiveMFAConfig struct {
	Enabled                bool    `json:"enabled" default:"false"`
	RiskThreshold          float64 `json:"risk_threshold" default:"50.0"` // 0-100
	FactorLocationChange   bool    `json:"factor_location_change" default:"true"`
	FactorNewDevice        bool    `json:"factor_new_device" default:"true"`
	FactorVelocity         bool    `json:"factor_velocity" default:"true"`
	FactorIPReputation     bool    `json:"factor_ip_reputation" default:"false"`
	RequireStepUpThreshold float64 `json:"require_step_up_threshold" default:"75.0"`
	LocationChangeRisk     float64 `json:"location_change_risk" default:"30.0"`
	NewDeviceRisk          float64 `json:"new_device_risk" default:"40.0"`
	VelocityRisk           float64 `json:"velocity_risk" default:"50.0"`
}

// DefaultConfig returns default MFA configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:             true,
		RequireForAllUsers:  false,
		GracePeriodDays:     7,
		AllowedFactorTypes:  []FactorType{FactorTypeTOTP, FactorTypeSMS, FactorTypeEmail, FactorTypeWebAuthn, FactorTypeBackup},
		RequiredFactorCount: 1,
		TOTP: TOTPConfig{
			Enabled:    true,
			Issuer:     "AuthSome",
			Period:     30,
			Digits:     6,
			Algorithm:  "SHA1",
			WindowSize: 1,
		},
		SMS: SMSConfig{
			Enabled:           true,
			CodeLength:        6,
			CodeExpiryMinutes: 5,
		},
		Email: EmailConfig{
			Enabled:           true,
			CodeLength:        6,
			CodeExpiryMinutes: 10,
		},
		WebAuthn: WebAuthnConfig{
			Enabled:               true,
			RPDisplayName:         "AuthSome",
			AttestationPreference: "none",
			Timeout:               60000,
		},
		BackupCodes: BackupCodesConfig{
			Enabled:    true,
			Count:      10,
			Length:     8,
			Format:     "XXXX-XXXX",
			AllowReuse: false,
		},
		TrustedDevices: TrustedDevicesConfig{
			Enabled:           true,
			DefaultExpiryDays: 30,
			MaxExpiryDays:     90,
			MaxDevicesPerUser: 5,
		},
		ChallengeExpiryMinutes: 5,
		MaxAttempts:            3,
		RateLimit: RateLimitConfig{
			Enabled:        true,
			MaxAttempts:    5,
			WindowMinutes:  15,
			LockoutMinutes: 30,
		},
		AdaptiveMFA: AdaptiveMFAConfig{
			Enabled:                false,
			RiskThreshold:          50.0,
			FactorLocationChange:   true,
			FactorNewDevice:        true,
			FactorVelocity:         true,
			FactorIPReputation:     false,
			RequireStepUpThreshold: 75.0,
			LocationChangeRisk:     30.0,
			NewDeviceRisk:          40.0,
			VelocityRisk:           50.0,
		},
		SessionExpiryMinutes: 15,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.RequiredFactorCount < 0 {
		c.RequiredFactorCount = 1
	}

	if c.RequiredFactorCount > len(c.AllowedFactorTypes) {
		c.RequiredFactorCount = len(c.AllowedFactorTypes)
	}

	if c.GracePeriodDays < 0 {
		c.GracePeriodDays = 0
	}

	if c.ChallengeExpiryMinutes <= 0 {
		c.ChallengeExpiryMinutes = 5
	}

	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}

	if c.SessionExpiryMinutes <= 0 {
		c.SessionExpiryMinutes = 15
	}

	return nil
}

// IsFactorAllowed checks if a factor type is allowed
func (c *Config) IsFactorAllowed(factorType FactorType) bool {
	for _, allowed := range c.AllowedFactorTypes {
		if allowed == factorType {
			return true
		}
	}
	return false
}

// GetFactorConfig returns configuration for a specific factor type
func (c *Config) GetFactorConfig(factorType FactorType) interface{} {
	switch factorType {
	case FactorTypeTOTP:
		return &c.TOTP
	case FactorTypeSMS:
		return &c.SMS
	case FactorTypeEmail:
		return &c.Email
	case FactorTypeWebAuthn:
		return &c.WebAuthn
	case FactorTypeBackup:
		return &c.BackupCodes
	default:
		return nil
	}
}
