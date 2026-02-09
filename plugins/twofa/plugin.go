package twofa

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the plugins.Plugin interface for Two-Factor Authentication.
type Plugin struct {
	service       *Service
	db            *bun.DB
	logger        forge.Logger
	config        Config
	defaultConfig Config
	authInst      core.Authsome
}

// Config holds the 2FA plugin configuration.
type Config struct {
	// TOTPIssuer is the issuer name shown in authenticator apps
	TOTPIssuer string `json:"totpIssuer"`
	// TOTPPeriod is the TOTP time period in seconds
	TOTPPeriod int `json:"totpPeriod"`
	// TOTPDigits is the number of digits in TOTP code
	TOTPDigits int `json:"totpDigits"`
	// BackupCodeCount is the number of backup codes to generate
	BackupCodeCount int `json:"backupCodeCount"`
	// BackupCodeLength is the length of each backup code
	BackupCodeLength int `json:"backupCodeLength"`
	// OTPExpiryMinutes is the OTP expiry time in minutes
	OTPExpiryMinutes int `json:"otpExpiryMinutes"`
	// MaxOTPAttempts is the maximum failed OTP attempts before lockout
	MaxOTPAttempts int `json:"maxOtpAttempts"`
	// TrustedDeviceDays is the number of days a device remains trusted
	TrustedDeviceDays int `json:"trustedDeviceDays"`
	// RequireFor2FA forces 2FA for all users
	RequireFor2FA bool `json:"requireFor2FA"`
}

// DefaultConfig returns the default 2FA plugin configuration.
func DefaultConfig() Config {
	return Config{
		TOTPIssuer:        "AuthSome",
		TOTPPeriod:        30,
		TOTPDigits:        6,
		BackupCodeCount:   10,
		BackupCodeLength:  8,
		OTPExpiryMinutes:  5,
		MaxOTPAttempts:    5,
		TrustedDeviceDays: 30,
		RequireFor2FA:     false,
	}
}

// PluginOption is a functional option for configuring the 2FA plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithTOTPIssuer sets the TOTP issuer name.
func WithTOTPIssuer(issuer string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TOTPIssuer = issuer
	}
}

// WithTOTPPeriod sets the TOTP time period.
func WithTOTPPeriod(period int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TOTPPeriod = period
	}
}

// WithBackupCodeCount sets the number of backup codes.
func WithBackupCodeCount(count int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.BackupCodeCount = count
	}
}

// WithBackupCodeLength sets the backup code length.
func WithBackupCodeLength(length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.BackupCodeLength = length
	}
}

// WithOTPExpiryMinutes sets the OTP expiry time.
func WithOTPExpiryMinutes(minutes int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.OTPExpiryMinutes = minutes
	}
}

// WithMaxOTPAttempts sets the max OTP attempts.
func WithMaxOTPAttempts(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxOTPAttempts = max
	}
}

// WithTrustedDeviceDays sets the trusted device duration.
func WithTrustedDeviceDays(days int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TrustedDeviceDays = days
	}
}

// WithRequireFor2FA sets whether 2FA is required for all users.
func WithRequireFor2FA(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireFor2FA = required
	}
}

// NewPlugin creates a new 2FA plugin instance with optional configuration.
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Plugin) ID() string { return "twofa" }

func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return errs.InternalServerErrorWithMessage("twofa plugin requires auth instance")
	}

	// Store auth instance for middleware access
	p.authInst = authInst

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return errs.InternalServerErrorWithMessage("database not available for twofa plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return errs.InternalServerErrorWithMessage("forge app not available for twofa plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "twofa"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.twofa", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind 2FA config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models for 2FA
	p.db.RegisterModel((*schema.TwoFASecret)(nil))
	p.db.RegisterModel((*schema.BackupCode)(nil))
	p.db.RegisterModel((*schema.TrustedDevice)(nil))
	p.db.RegisterModel((*schema.OTPCode)(nil))

	// Wire repository-backed service with config
	p.service = NewService(authInst.Repository().TwoFA(), p.config)

	p.logger.Info("2FA plugin initialized",
		forge.F("totp_issuer", p.config.TOTPIssuer),
		forge.F("backup_code_count", p.config.BackupCodeCount),
		forge.F("trusted_device_days", p.config.TrustedDeviceDays),
		forge.F("require_for_2fa", p.config.RequireFor2FA))

	return nil
}

// RegisterRoutes registers 2FA endpoints under the auth base.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Router is already scoped to the correct basePath by the auth mount
	h := NewHandler(p.service)

	// Get authentication middleware for API key validation
	authMw := p.authInst.AuthMiddleware()

	// Wrap handler with middleware if available
	wrapHandler := func(handler func(forge.Context) error) func(forge.Context) error {
		if authMw != nil {
			return authMw(handler)
		}

		return handler
	}

	router.POST("/2fa/enable", wrapHandler(h.Enable),
		forge.WithName("twofa.enable"),
		forge.WithSummary("Enable 2FA"),
		forge.WithDescription("Enables two-factor authentication for a user. Returns TOTP URI for QR code generation"),
		forge.WithResponseSchema(200, "2FA enabled", TwoFAEnableResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "TOTP"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/verify", wrapHandler(h.Verify),
		forge.WithName("twofa.verify"),
		forge.WithSummary("Verify 2FA code"),
		forge.WithDescription("Verifies a 2FA code (TOTP or backup code) for authentication. Optionally marks device as trusted"),
		forge.WithResponseSchema(200, "Code verified", TwoFAStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid code", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "Verification"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/disable", wrapHandler(h.Disable),
		forge.WithName("twofa.disable"),
		forge.WithSummary("Disable 2FA"),
		forge.WithDescription("Disables two-factor authentication for a user"),
		forge.WithResponseSchema(200, "2FA disabled", TwoFAStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "Management"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/generate-backup-codes", wrapHandler(h.GenerateBackupCodes),
		forge.WithName("twofa.backupcodes.generate"),
		forge.WithSummary("Generate backup codes"),
		forge.WithDescription("Generates new backup codes for 2FA recovery. Previous codes are invalidated"),
		forge.WithResponseSchema(200, "Backup codes generated", TwoFABackupCodesResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "BackupCodes"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/send-otp", wrapHandler(h.SendOTP),
		forge.WithName("twofa.sendotp"),
		forge.WithSummary("Send OTP code"),
		forge.WithDescription("Sends a one-time password via email or SMS for 2FA. Returns code in dev mode"),
		forge.WithResponseSchema(200, "OTP sent", TwoFASendOTPResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "OTP"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/status", wrapHandler(h.Status),
		forge.WithName("twofa.status"),
		forge.WithSummary("Get 2FA status"),
		forge.WithDescription("Retrieves 2FA status for a user including enabled state, method, and trusted device status"),
		forge.WithResponseSchema(200, "Status retrieved", TwoFAStatusDetailResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "Status"),
		forge.WithValidation(true),
	)

	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()
	// Create required tables if not exist
	if _, err := p.db.NewCreateTable().Model((*schema.TwoFASecret)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}

	if _, err := p.db.NewCreateTable().Model((*schema.BackupCode)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}

	if _, err := p.db.NewCreateTable().Model((*schema.TrustedDevice)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}

	if _, err := p.db.NewCreateTable().Model((*schema.OTPCode)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}

	return nil
}

// Response types for 2FA routes.
type TwoFAErrorResponse struct {
	Error string `example:"Error message" json:"error"`
}

type TwoFAEnableResponse struct {
	Status  string `example:"2fa_enabled"                                                                      json:"status"`
	TOTPURI string `example:"otpauth://totp/AuthSome:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=AuthSome" json:"totp_uri,omitempty"`
}

type TwoFABackupCodesResponse struct {
	Codes []string `example:"12345678,87654321" json:"codes"`
}

type TwoFASendOTPResponse struct {
	Status string `example:"otp_sent" json:"status"`
	Code   string `example:"123456"   json:"code,omitempty"`
}

type TwoFAStatusDetailResponse struct {
	Enabled bool   `example:"true"  json:"enabled"`
	Method  string `example:"totp"  json:"method,omitempty"`
	Trusted bool   `example:"false" json:"trusted,omitempty"`
}
