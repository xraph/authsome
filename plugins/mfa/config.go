package mfa

import "slices"

// Config holds MFA plugin configuration.
type Config struct {
	// Global settings
	Enabled            bool `default:"true"  json:"enabled"`
	RequireForAllUsers bool `default:"false" json:"require_for_all_users"`
	GracePeriodDays    int  `default:"7"     json:"grace_period_days"`

	// Factor settings
	AllowedFactorTypes  []FactorType `json:"allowed_factor_types"`
	RequiredFactorCount int          `default:"1"                 json:"required_factor_count"`

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
	ChallengeExpiryMinutes int `default:"5" json:"challenge_expiry_minutes"`
	MaxAttempts            int `default:"3" json:"max_attempts"`

	// Rate limiting
	RateLimit RateLimitConfig `json:"rate_limit"`

	// Adaptive MFA
	AdaptiveMFA AdaptiveMFAConfig `json:"adaptive_mfa"`

	// Session settings
	SessionExpiryMinutes int `default:"15" json:"session_expiry_minutes"`
}

// TOTPConfig configures TOTP (Google Authenticator) settings.
type TOTPConfig struct {
	Enabled    bool   `default:"true"     json:"enabled"`
	Issuer     string `default:"AuthSome" json:"issuer"`
	Period     int    `default:"30"       json:"period"` // Seconds
	Digits     int    `default:"6"        json:"digits"`
	Algorithm  string `default:"SHA1"     json:"algorithm"`   // SHA1, SHA256, SHA512
	WindowSize int    `default:"1"        json:"window_size"` // Past/future periods to accept
}

// SMSConfig configures SMS verification settings.
type SMSConfig struct {
	Enabled           bool             `default:"true"              json:"enabled"`
	Provider          string           `json:"provider"` // "twilio", "vonage", etc.
	CodeLength        int              `default:"6"                 json:"code_length"`
	CodeExpiryMinutes int              `default:"5"                 json:"code_expiry_minutes"`
	TemplateID        string           `json:"template_id"`
	RateLimit         *RateLimitConfig `json:"rate_limit,omitempty"`
}

// EmailConfig configures email verification settings.
type EmailConfig struct {
	Enabled           bool             `default:"true"              json:"enabled"`
	Provider          string           `json:"provider"` // Email provider
	CodeLength        int              `default:"6"                 json:"code_length"`
	CodeExpiryMinutes int              `default:"10"                json:"code_expiry_minutes"`
	TemplateID        string           `json:"template_id"`
	RateLimit         *RateLimitConfig `json:"rate_limit,omitempty"`
}

// WebAuthnConfig configures WebAuthn/FIDO2 settings.
type WebAuthnConfig struct {
	Enabled                bool     `default:"true"     json:"enabled"`
	RPDisplayName          string   `default:"AuthSome" json:"rp_display_name"`
	RPID                   string   `json:"rp_id"`                                     // e.g., "example.com"
	RPOrigins              []string `json:"rp_origins"`                                // Allowed origins
	AttestationPreference  string   `default:"none"     json:"attestation_preference"` // none, indirect, direct
	AuthenticatorSelection struct {
		RequireResidentKey     bool   `default:"false"     json:"require_resident_key"`
		ResidentKeyRequirement string `default:"preferred" json:"resident_key_requirement"` // discouraged, preferred, required
		UserVerification       string `default:"preferred" json:"user_verification"`        // discouraged, preferred, required
	} `json:"authenticator_selection"`
	Timeout int `default:"60000" json:"timeout"` // Milliseconds
}

// BackupCodesConfig configures backup recovery codes.
type BackupCodesConfig struct {
	Enabled    bool   `default:"true"      json:"enabled"`
	Count      int    `default:"10"        json:"count"`
	Length     int    `default:"8"         json:"length"`
	Format     string `default:"XXXX-XXXX" json:"format"` // Code format
	AllowReuse bool   `default:"false"     json:"allow_reuse"`
}

// TrustedDevicesConfig configures trusted device settings.
type TrustedDevicesConfig struct {
	Enabled           bool `default:"true" json:"enabled"`
	DefaultExpiryDays int  `default:"30"   json:"default_expiry_days"`
	MaxExpiryDays     int  `default:"90"   json:"max_expiry_days"`
	MaxDevicesPerUser int  `default:"5"    json:"max_devices_per_user"`
}

// RateLimitConfig configures rate limiting.
type RateLimitConfig struct {
	Enabled        bool `default:"true" json:"enabled"`
	MaxAttempts    int  `default:"5"    json:"max_attempts"`
	WindowMinutes  int  `default:"15"   json:"window_minutes"`
	LockoutMinutes int  `default:"30"   json:"lockout_minutes"`
}

// AdaptiveMFAConfig configures risk-based authentication.
type AdaptiveMFAConfig struct {
	Enabled                bool    `default:"false" json:"enabled"`
	RiskThreshold          float64 `default:"50.0"  json:"risk_threshold"` // 0-100
	FactorLocationChange   bool    `default:"true"  json:"factor_location_change"`
	FactorNewDevice        bool    `default:"true"  json:"factor_new_device"`
	FactorVelocity         bool    `default:"true"  json:"factor_velocity"`
	FactorIPReputation     bool    `default:"false" json:"factor_ip_reputation"`
	RequireStepUpThreshold float64 `default:"75.0"  json:"require_step_up_threshold"`
	LocationChangeRisk     float64 `default:"30.0"  json:"location_change_risk"`
	NewDeviceRisk          float64 `default:"40.0"  json:"new_device_risk"`
	VelocityRisk           float64 `default:"50.0"  json:"velocity_risk"`
}

// DefaultConfig returns default MFA configuration.
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

// Validate validates the configuration.
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

// IsFactorAllowed checks if a factor type is allowed.
func (c *Config) IsFactorAllowed(factorType FactorType) bool {
	return slices.Contains(c.AllowedFactorTypes, factorType)
}

// GetFactorConfig returns configuration for a specific factor type.
func (c *Config) GetFactorConfig(factorType FactorType) any {
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
