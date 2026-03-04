package extension

import "time"

// Config holds the AuthSome extension configuration.
// Fields can be set programmatically via ExtOption functions or loaded from
// YAML configuration files (under "extensions.authsome" or "authsome" keys).
type Config struct {
	// DisableRoutes prevents HTTP route registration.
	DisableRoutes bool `json:"disable_routes" mapstructure:"disable_routes" yaml:"disable_routes"`

	// DisableMigrate prevents auto-migration on start.
	DisableMigrate bool `json:"disable_migrate" mapstructure:"disable_migrate" yaml:"disable_migrate"`

	// BasePath is the URL prefix for all auth routes (default: "/v1/auth").
	BasePath string `json:"base_path" mapstructure:"base_path" yaml:"base_path"`

	// Debug enables verbose logging of all operations.
	Debug bool `json:"debug" mapstructure:"debug" yaml:"debug"`

	// Session configuration.
	Session SessionConfig `json:"session" mapstructure:"session" yaml:"session"`

	// Password policy configuration.
	Password PasswordConfig `json:"password" mapstructure:"password" yaml:"password"`

	// Rate limiting per endpoint.
	RateLimit RateLimitConfig `json:"rate_limit" mapstructure:"rate_limit" yaml:"rate_limit"`

	// Account lockout after failed attempts.
	Lockout LockoutConfig `json:"lockout" mapstructure:"lockout" yaml:"lockout"`

	// Mailer configuration for transactional email delivery.
	Mailer MailerConfig `json:"mailer" mapstructure:"mailer" yaml:"mailer"`

	// GroveDatabase is the name of a grove.DB registered in the DI container.
	// When set, the extension resolves this named database and auto-constructs
	// the appropriate store based on the driver type (pg/sqlite/mongo).
	// When empty and WithGroveDatabase was called, the default (unnamed) DB is used.
	GroveDatabase string `json:"grove_database" mapstructure:"grove_database" yaml:"grove_database"`

	// Apps holds per-app session configuration overrides.
	// Keys are app IDs (e.g., "mobile-app", "web-dashboard").
	Apps map[string]AppSessionConfigYAML `json:"apps" mapstructure:"apps" yaml:"apps"`

	// Bootstrap configures automatic platform app setup on first start.
	Bootstrap BootstrapYAMLConfig `json:"bootstrap" mapstructure:"bootstrap" yaml:"bootstrap"`

	// RequireConfig requires config to be present in YAML files.
	// If true and no config is found, Register returns an error.
	RequireConfig bool `json:"-" yaml:"-"`
}

// BootstrapYAMLConfig holds bootstrap settings loadable from YAML.
type BootstrapYAMLConfig struct {
	// Enabled controls whether bootstrap runs on start. Default: true when using extension.
	Enabled *bool `json:"enabled" mapstructure:"enabled" yaml:"enabled"`

	// AppName is the platform app display name (default: "Platform").
	AppName string `json:"app_name" mapstructure:"app_name" yaml:"app_name"`

	// AppSlug is the platform app slug (default: "platform").
	AppSlug string `json:"app_slug" mapstructure:"app_slug" yaml:"app_slug"`

	// AppLogo is an optional logo URL for the platform app.
	AppLogo string `json:"app_logo" mapstructure:"app_logo" yaml:"app_logo"`

	// SkipDefaultEnvs disables automatic environment creation.
	SkipDefaultEnvs bool `json:"skip_default_envs" mapstructure:"skip_default_envs" yaml:"skip_default_envs"`

	// SkipDefaultRoles disables automatic role creation.
	SkipDefaultRoles bool `json:"skip_default_roles" mapstructure:"skip_default_roles" yaml:"skip_default_roles"`

	// Environments overrides the default environments to create.
	Environments []BootstrapEnvYAML `json:"environments" mapstructure:"environments" yaml:"environments"`

	// Roles overrides the default roles to create.
	Roles []BootstrapRoleYAML `json:"roles" mapstructure:"roles" yaml:"roles"`
}

// BootstrapEnvYAML describes an environment in YAML config.
type BootstrapEnvYAML struct {
	Name      string `json:"name" mapstructure:"name" yaml:"name"`
	Slug      string `json:"slug" mapstructure:"slug" yaml:"slug"`
	Type      string `json:"type" mapstructure:"type" yaml:"type"` // development, staging, production
	IsDefault bool   `json:"is_default" mapstructure:"is_default" yaml:"is_default"`
}

// BootstrapRoleYAML describes a role in YAML config.
type BootstrapRoleYAML struct {
	Name        string `json:"name" mapstructure:"name" yaml:"name"`
	Slug        string `json:"slug" mapstructure:"slug" yaml:"slug"`
	Description string `json:"description" mapstructure:"description" yaml:"description"`
}

// AppSessionConfigYAML holds per-app session config in YAML format.
type AppSessionConfigYAML struct {
	Session AppSessionYAML `json:"session" mapstructure:"session" yaml:"session"`
}

// AppSessionYAML holds session config overrides for a specific app.
type AppSessionYAML struct {
	TokenTTL           time.Duration `json:"token_ttl" mapstructure:"token_ttl" yaml:"token_ttl"`
	RefreshTokenTTL    time.Duration `json:"refresh_token_ttl" mapstructure:"refresh_token_ttl" yaml:"refresh_token_ttl"`
	MaxActiveSessions  *int          `json:"max_active_sessions" mapstructure:"max_active_sessions" yaml:"max_active_sessions"`
	RotateRefreshToken *bool         `json:"rotate_refresh_token" mapstructure:"rotate_refresh_token" yaml:"rotate_refresh_token"`
	BindToIP           *bool         `json:"bind_to_ip" mapstructure:"bind_to_ip" yaml:"bind_to_ip"`
	BindToDevice       *bool         `json:"bind_to_device" mapstructure:"bind_to_device" yaml:"bind_to_device"`
	TokenFormat        string        `json:"token_format" mapstructure:"token_format" yaml:"token_format"`
}

// MailerConfig configures the transactional email provider.
type MailerConfig struct {
	// Provider is the email provider ("resend", "smtp", or "noop"). Default: "noop".
	Provider string `json:"provider" mapstructure:"provider" yaml:"provider"`

	// Resend configuration (used when Provider = "resend").
	Resend ResendConfig `json:"resend" mapstructure:"resend" yaml:"resend"`

	// SMTP configuration (used when Provider = "smtp").
	SMTP SMTPConfig `json:"smtp" mapstructure:"smtp" yaml:"smtp"`
}

// ResendConfig holds Resend API credentials.
type ResendConfig struct {
	APIKey string `json:"api_key" mapstructure:"api_key" yaml:"api_key"`
	From   string `json:"from" mapstructure:"from" yaml:"from"`
}

// SMTPConfig holds SMTP server credentials.
type SMTPConfig struct {
	Host     string `json:"host" mapstructure:"host" yaml:"host"`
	Port     string `json:"port" mapstructure:"port" yaml:"port"`
	Username string `json:"username" mapstructure:"username" yaml:"username"`
	Password string `json:"password" mapstructure:"password" yaml:"password"`
	From     string `json:"from" mapstructure:"from" yaml:"from"`
	TLS      bool   `json:"tls" mapstructure:"tls" yaml:"tls"`
}

// SessionConfig configures session behavior.
type SessionConfig struct {
	// TokenTTL is the lifetime of session tokens (default: 1h).
	TokenTTL time.Duration `json:"token_ttl" mapstructure:"token_ttl" yaml:"token_ttl"`

	// RefreshTokenTTL is the lifetime of refresh tokens (default: 30d).
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl" mapstructure:"refresh_token_ttl" yaml:"refresh_token_ttl"`

	// MaxActiveSessions is the maximum number of concurrent sessions per user (0 = unlimited).
	MaxActiveSessions int `json:"max_active_sessions" mapstructure:"max_active_sessions" yaml:"max_active_sessions"`

	// RotateRefreshToken controls whether refresh operations issue a new
	// refresh token (invalidating the old one). Default: true.
	RotateRefreshToken *bool `json:"rotate_refresh_token" mapstructure:"rotate_refresh_token" yaml:"rotate_refresh_token"`
}

// PasswordConfig configures password validation.
type PasswordConfig struct {
	// MinLength is the minimum password length (default: 8).
	MinLength int `json:"min_length" mapstructure:"min_length" yaml:"min_length"`

	// RequireUppercase requires at least one uppercase letter.
	RequireUppercase bool `json:"require_uppercase" mapstructure:"require_uppercase" yaml:"require_uppercase"`

	// RequireLowercase requires at least one lowercase letter.
	RequireLowercase bool `json:"require_lowercase" mapstructure:"require_lowercase" yaml:"require_lowercase"`

	// RequireDigit requires at least one digit.
	RequireDigit bool `json:"require_digit" mapstructure:"require_digit" yaml:"require_digit"`

	// RequireSpecial requires at least one special character.
	RequireSpecial bool `json:"require_special" mapstructure:"require_special" yaml:"require_special"`

	// BcryptCost is the bcrypt cost factor (default: 12).
	BcryptCost int `json:"bcrypt_cost" mapstructure:"bcrypt_cost" yaml:"bcrypt_cost"`

	// Algorithm is the password hashing algorithm ("bcrypt" or "argon2id"). Default: "bcrypt".
	Algorithm string `json:"algorithm" mapstructure:"algorithm" yaml:"algorithm"`

	// Argon2 holds parameters for Argon2id hashing.
	Argon2 Argon2Config `json:"argon2" mapstructure:"argon2" yaml:"argon2"`

	// CheckBreached enables HaveIBeenPwned breach checking. Default: false.
	CheckBreached bool `json:"check_breached" mapstructure:"check_breached" yaml:"check_breached"`
}

// Argon2Config holds Argon2id parameters for the extension config.
type Argon2Config struct {
	Memory      uint32 `json:"memory" mapstructure:"memory" yaml:"memory"`
	Iterations  uint32 `json:"iterations" mapstructure:"iterations" yaml:"iterations"`
	Parallelism uint8  `json:"parallelism" mapstructure:"parallelism" yaml:"parallelism"`
	SaltLength  uint32 `json:"salt_length" mapstructure:"salt_length" yaml:"salt_length"`
	KeyLength   uint32 `json:"key_length" mapstructure:"key_length" yaml:"key_length"`
}

// RateLimitConfig configures per-endpoint rate limits.
type RateLimitConfig struct {
	// Enabled enables rate limiting.
	Enabled bool `json:"enabled" mapstructure:"enabled" yaml:"enabled"`

	// SignInLimit is the max sign-in attempts per window (default: 5).
	SignInLimit int `json:"signin_limit" mapstructure:"signin_limit" yaml:"signin_limit"`

	// SignUpLimit is the max sign-up attempts per window (default: 3).
	SignUpLimit int `json:"signup_limit" mapstructure:"signup_limit" yaml:"signup_limit"`

	// ForgotPasswordLimit is the max forgot-password attempts per window (default: 3).
	ForgotPasswordLimit int `json:"forgot_password_limit" mapstructure:"forgot_password_limit" yaml:"forgot_password_limit"`

	// MFAChallengeLimit is the max MFA challenge attempts per window (default: 5).
	MFAChallengeLimit int `json:"mfa_challenge_limit" mapstructure:"mfa_challenge_limit" yaml:"mfa_challenge_limit"`

	// WindowSeconds is the sliding window duration in seconds (default: 60).
	WindowSeconds int `json:"window_seconds" mapstructure:"window_seconds" yaml:"window_seconds"`
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
	// Enabled enables account lockout.
	Enabled bool `json:"enabled" mapstructure:"enabled" yaml:"enabled"`

	// MaxAttempts is the number of failed attempts before lockout (default: 5).
	MaxAttempts int `json:"max_attempts" mapstructure:"max_attempts" yaml:"max_attempts"`

	// LockoutDurationSeconds is the lockout duration in seconds (default: 900 = 15 min).
	LockoutDurationSeconds int `json:"lockout_duration_seconds" mapstructure:"lockout_duration_seconds" yaml:"lockout_duration_seconds"`

	// ResetAfterSeconds resets the failure count after this many seconds of no failures (default: 3600 = 1h).
	ResetAfterSeconds int `json:"reset_after_seconds" mapstructure:"reset_after_seconds" yaml:"reset_after_seconds"`
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
		BasePath: "/v1/auth",
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
	}
}
