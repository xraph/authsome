package authsome

import "time"

// Config holds all configuration for the AuthSome engine.
type Config struct {
	// AppID is the forge.Scope app identity.
	AppID string `json:"app_id"`

	// BasePath is the URL prefix for all auth routes (default: "/authsome").
	BasePath string `json:"base_path"`

	// Session configuration
	Session SessionConfig `json:"session"`

	// Password policy
	Password PasswordConfig `json:"password"`

	// Rate limiting per endpoint
	RateLimit RateLimitConfig `json:"rate_limit"`

	// Account lockout after failed attempts
	Lockout LockoutConfig `json:"lockout"`

	// DriverName is the grove driver name ("pg", "sqlite", "mongo", "memory").
	// Set automatically by the extension based on the detected grove driver.
	DriverName string `json:"driver_name"`

	// Debug enables verbose logging of all operations.
	Debug bool `json:"debug"`

	// DisableRoutes prevents automatic route registration.
	DisableRoutes bool `json:"disable_routes"`

	// DisableMigrate prevents automatic database migration on Start.
	DisableMigrate bool `json:"disable_migrate"`
}

// SessionConfig configures session behavior.
type SessionConfig struct {
	// TokenTTL is the lifetime of session tokens (default: 1h).
	TokenTTL time.Duration `json:"token_ttl"`

	// RefreshTokenTTL is the lifetime of refresh tokens (default: 30d).
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl"`

	// MaxActiveSessions is the maximum number of concurrent sessions per user (0 = unlimited).
	MaxActiveSessions int `json:"max_active_sessions"`

	// RotateRefreshToken controls whether refresh operations issue a new
	// refresh token (invalidating the old one). Default: true.
	// When false, the same refresh token is reused until it expires.
	RotateRefreshToken *bool `json:"rotate_refresh_token"`

	// BindToIP rejects requests when the client IP differs from the
	// IP recorded at session creation. Default: false.
	BindToIP bool `json:"bind_to_ip"`

	// BindToDevice rejects requests when the User-Agent differs from
	// the one recorded at session creation. Default: false.
	BindToDevice bool `json:"bind_to_device"`

	// CleanupInterval is the interval at which expired sessions are
	// automatically purged. 0 = disabled.
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// ShouldRotateRefreshToken returns whether refresh token rotation is enabled.
// Defaults to true when not explicitly configured.
func (c SessionConfig) ShouldRotateRefreshToken() bool {
	if c.RotateRefreshToken == nil {
		return true
	}
	return *c.RotateRefreshToken
}

// PasswordConfig configures password validation.
type PasswordConfig struct {
	// MinLength is the minimum password length (default: 8).
	MinLength int `json:"min_length"`

	// RequireUppercase requires at least one uppercase letter.
	RequireUppercase bool `json:"require_uppercase"`

	// RequireLowercase requires at least one lowercase letter.
	RequireLowercase bool `json:"require_lowercase"`

	// RequireDigit requires at least one digit.
	RequireDigit bool `json:"require_digit"`

	// RequireSpecial requires at least one special character.
	RequireSpecial bool `json:"require_special"`

	// BcryptCost is the bcrypt cost factor (default: 12).
	BcryptCost int `json:"bcrypt_cost"`

	// Algorithm is the password hashing algorithm ("bcrypt" or "argon2id").
	// Default: "bcrypt". When changed, existing hashes are auto-rehashed on
	// next successful login.
	Algorithm string `json:"algorithm"`

	// Argon2 holds parameters for Argon2id hashing (used when Algorithm = "argon2id").
	Argon2 Argon2Config `json:"argon2"`

	// CheckBreached enables HaveIBeenPwned breach checking on signup and
	// password change. Default: false.
	CheckBreached bool `json:"check_breached"`

	// HistoryCount is the number of previous passwords to remember. When set,
	// users cannot reuse any of their last N passwords. 0 = disabled.
	HistoryCount int `json:"history_count"`

	// MaxAgeDays forces password rotation after this many days. When a user
	// signs in with an expired password, the engine returns ErrPasswordExpired.
	// 0 = no expiry.
	MaxAgeDays int `json:"max_age_days"`
}

// Argon2Config holds Argon2id parameters.
type Argon2Config struct {
	Memory      uint32 `json:"memory"`      // KiB (default: 65536 = 64 MiB)
	Iterations  uint32 `json:"iterations"`  // Time cost (default: 3)
	Parallelism uint8  `json:"parallelism"` // Threads (default: 2)
	SaltLength  uint32 `json:"salt_length"` // Bytes (default: 16)
	KeyLength   uint32 `json:"key_length"`  // Bytes (default: 32)
}

// RateLimitConfig configures per-endpoint rate limits.
type RateLimitConfig struct {
	// SignInLimit is the max sign-in attempts per window (default: 5).
	SignInLimit int `json:"signin_limit"`

	// SignUpLimit is the max sign-up attempts per window (default: 3).
	SignUpLimit int `json:"signup_limit"`

	// RefreshLimit is the max token refresh attempts per window (default: 10).
	RefreshLimit int `json:"refresh_limit"`

	// IntrospectLimit is the max token introspection attempts per window (default: 20).
	IntrospectLimit int `json:"introspect_limit"`

	// ForgotPasswordLimit is the max forgot-password attempts per window (default: 3).
	ForgotPasswordLimit int `json:"forgot_password_limit"`

	// MFAChallengeLimit is the max MFA challenge attempts per window (default: 5).
	MFAChallengeLimit int `json:"mfa_challenge_limit"`

	// WindowSeconds is the sliding window duration in seconds (default: 60).
	WindowSeconds int `json:"window_seconds"`

	// Enabled enables rate limiting (default: false).
	Enabled bool `json:"enabled"`
}

// Window returns the rate limit window as a time.Duration.
func (c RateLimitConfig) Window() time.Duration {
	if c.WindowSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.WindowSeconds) * time.Second
}

// LockoutConfig configures account lockout after failed authentication attempts.
type LockoutConfig struct {
	// MaxAttempts is the number of failed attempts before lockout (default: 5).
	MaxAttempts int `json:"max_attempts"`

	// LockoutDurationSeconds is the lockout duration in seconds (default: 900 = 15 min).
	LockoutDurationSeconds int `json:"lockout_duration_seconds"`

	// ResetAfterSeconds resets the failure count after this many seconds of no failures (default: 3600 = 1h).
	ResetAfterSeconds int `json:"reset_after_seconds"`

	// Enabled enables account lockout (default: false).
	Enabled bool `json:"enabled"`
}

// LockoutDuration returns the lockout duration as a time.Duration.
func (c LockoutConfig) LockoutDuration() time.Duration {
	if c.LockoutDurationSeconds <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(c.LockoutDurationSeconds) * time.Second
}

// ResetAfter returns the failure count reset period as a time.Duration.
func (c LockoutConfig) ResetAfter() time.Duration {
	if c.ResetAfterSeconds <= 0 {
		return 1 * time.Hour
	}
	return time.Duration(c.ResetAfterSeconds) * time.Second
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BasePath: "/authsome",
		Session: SessionConfig{
			TokenTTL:        1 * time.Hour,
			RefreshTokenTTL: 30 * 24 * time.Hour,
		},
		Password: PasswordConfig{
			MinLength:        8,
			RequireUppercase: true,
			RequireLowercase: true,
			RequireDigit:     true,
			BcryptCost:       12,
		},
		RateLimit: RateLimitConfig{
			SignInLimit:         5,
			SignUpLimit:         3,
			RefreshLimit:        10,
			IntrospectLimit:     20,
			ForgotPasswordLimit: 3,
			MFAChallengeLimit:   5,
			WindowSeconds:       60,
		},
		Lockout: LockoutConfig{
			MaxAttempts:            5,
			LockoutDurationSeconds: 900,  // 15 minutes
			ResetAfterSeconds:      3600, // 1 hour
		},
	}
}
