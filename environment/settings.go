package environment

import (
	"time"

	"github.com/xraph/authsome/account"
)

// Settings holds per-environment configuration overrides.
// All fields are pointers so nil means "inherit from the App-level config".
// Stored as JSONB for schema flexibility.
type Settings struct {
	// Auth behavior overrides.
	SkipEmailVerification *bool `json:"skip_email_verification,omitempty"`
	AllowTestCredentials  *bool `json:"allow_test_credentials,omitempty"`

	// Session overrides.
	TokenTTLSeconds        *int `json:"token_ttl_seconds,omitempty"`
	RefreshTokenTTLSeconds *int `json:"refresh_token_ttl_seconds,omitempty"`

	// Password policy overrides.
	PasswordMinLength *int  `json:"password_min_length,omitempty"`
	CheckBreached     *bool `json:"check_breached,omitempty"`

	// Rate limit overrides.
	RateLimitEnabled    *bool `json:"rate_limit_enabled,omitempty"`
	SignInRateLimit     *int  `json:"signin_rate_limit,omitempty"`
	SignUpRateLimit     *int  `json:"signup_rate_limit,omitempty"`
	RateLimitWindowSecs *int  `json:"rate_limit_window_seconds,omitempty"`

	// Lockout overrides.
	LockoutEnabled     *bool `json:"lockout_enabled,omitempty"`
	LockoutMaxAttempts *int  `json:"lockout_max_attempts,omitempty"`

	// Webhook URL override (for dev/staging webhook receivers).
	WebhookURLOverride string `json:"webhook_url_override,omitempty"`

	// Social/OAuth provider overrides (different client IDs per environment).
	OAuthOverrides map[string]OAuthProviderOverride `json:"oauth_overrides,omitempty"`
}

// OAuthProviderOverride allows per-environment OAuth credentials.
type OAuthProviderOverride struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	RedirectURL  string `json:"redirect_url,omitempty"`
}

// DefaultSettingsForType returns the default Settings for a given environment type.
func DefaultSettingsForType(t Type) *Settings {
	switch t {
	case TypeDevelopment:
		minLen := 4
		return &Settings{
			SkipEmailVerification: boolPtr(true),
			AllowTestCredentials:  boolPtr(true),
			PasswordMinLength:     &minLen,
			RateLimitEnabled:      boolPtr(false),
			LockoutEnabled:        boolPtr(false),
		}
	case TypeStaging:
		return &Settings{
			RateLimitEnabled: boolPtr(true),
			LockoutEnabled:   boolPtr(true),
		}
	case TypeProduction:
		return &Settings{
			RateLimitEnabled: boolPtr(true),
			LockoutEnabled:   boolPtr(true),
			CheckBreached:    boolPtr(true),
		}
	default:
		return nil
	}
}

// MergeSettings merges override settings on top of base settings.
// Non-nil fields in override take precedence. Returns a new Settings.
func MergeSettings(base, override *Settings) *Settings {
	if base == nil && override == nil {
		return nil
	}

	result := &Settings{}

	// Start with base values.
	if base != nil {
		*result = *base
		// Deep copy the map.
		if base.OAuthOverrides != nil {
			result.OAuthOverrides = make(map[string]OAuthProviderOverride, len(base.OAuthOverrides))
			for k, v := range base.OAuthOverrides {
				result.OAuthOverrides[k] = v
			}
		}
	}

	if override == nil {
		return result
	}

	// Apply overrides for non-nil fields.
	if override.SkipEmailVerification != nil {
		result.SkipEmailVerification = override.SkipEmailVerification
	}
	if override.AllowTestCredentials != nil {
		result.AllowTestCredentials = override.AllowTestCredentials
	}
	if override.TokenTTLSeconds != nil {
		result.TokenTTLSeconds = override.TokenTTLSeconds
	}
	if override.RefreshTokenTTLSeconds != nil {
		result.RefreshTokenTTLSeconds = override.RefreshTokenTTLSeconds
	}
	if override.PasswordMinLength != nil {
		result.PasswordMinLength = override.PasswordMinLength
	}
	if override.CheckBreached != nil {
		result.CheckBreached = override.CheckBreached
	}
	if override.RateLimitEnabled != nil {
		result.RateLimitEnabled = override.RateLimitEnabled
	}
	if override.SignInRateLimit != nil {
		result.SignInRateLimit = override.SignInRateLimit
	}
	if override.SignUpRateLimit != nil {
		result.SignUpRateLimit = override.SignUpRateLimit
	}
	if override.RateLimitWindowSecs != nil {
		result.RateLimitWindowSecs = override.RateLimitWindowSecs
	}
	if override.LockoutEnabled != nil {
		result.LockoutEnabled = override.LockoutEnabled
	}
	if override.LockoutMaxAttempts != nil {
		result.LockoutMaxAttempts = override.LockoutMaxAttempts
	}
	if override.WebhookURLOverride != "" {
		result.WebhookURLOverride = override.WebhookURLOverride
	}
	if override.OAuthOverrides != nil {
		if result.OAuthOverrides == nil {
			result.OAuthOverrides = make(map[string]OAuthProviderOverride)
		}
		for k, v := range override.OAuthOverrides {
			result.OAuthOverrides[k] = v
		}
	}

	return result
}

// SkipEmailVerificationEnabled returns whether email verification should be skipped.
func (s *Settings) SkipEmailVerificationEnabled() bool {
	if s == nil || s.SkipEmailVerification == nil {
		return false
	}
	return *s.SkipEmailVerification
}

// AllowTestCredentialsEnabled returns whether test credentials are allowed.
func (s *Settings) AllowTestCredentialsEnabled() bool {
	if s == nil || s.AllowTestCredentials == nil {
		return false
	}
	return *s.AllowTestCredentials
}

// IsRateLimitEnabled returns whether rate limiting is enabled.
func (s *Settings) IsRateLimitEnabled() bool {
	if s == nil || s.RateLimitEnabled == nil {
		return false
	}
	return *s.RateLimitEnabled
}

// IsLockoutEnabled returns whether account lockout is enabled.
func (s *Settings) IsLockoutEnabled() bool {
	if s == nil || s.LockoutEnabled == nil {
		return false
	}
	return *s.LockoutEnabled
}

// IsBreachCheckEnabled returns whether password breach checking is enabled.
func (s *Settings) IsBreachCheckEnabled() bool {
	if s == nil || s.CheckBreached == nil {
		return false
	}
	return *s.CheckBreached
}

// ApplySessionOverrides applies session-related overrides from environment
// settings onto the given session config. Only non-nil fields are applied.
func (s *Settings) ApplySessionOverrides(cfg *account.SessionConfig) {
	if s == nil {
		return
	}
	if s.TokenTTLSeconds != nil {
		cfg.TokenTTL = time.Duration(*s.TokenTTLSeconds) * time.Second
	}
	if s.RefreshTokenTTLSeconds != nil {
		cfg.RefreshTokenTTL = time.Duration(*s.RefreshTokenTTLSeconds) * time.Second
	}
}

func boolPtr(b bool) *bool { return &b }
