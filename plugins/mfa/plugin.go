package mfa

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/emailotp"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/plugins/twofa"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the plugins.Plugin interface for Multi-Factor Authentication
type Plugin struct {
	db              *bun.DB
	service         *Service
	adapterRegistry *FactorAdapterRegistry
	notifAdapter    *notificationPlugin.Adapter
	config          *Config
	defaultConfig   *Config
	logger          forge.Logger

	// Dependencies from other plugins
	twofaService    *twofa.Service
	emailOTPService *emailotp.Service
	phoneService    *phone.Service
	// passkeyService  *passkey.Service // Uncomment when passkey is stable
}

// PluginOption is a functional option for configuring the MFA plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg *Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithEnabled sets whether MFA is enabled
func WithEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Enabled = enabled
	}
}

// WithRequireForAllUsers sets whether MFA is required for all users
func WithRequireForAllUsers(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireForAllUsers = required
	}
}

// WithGracePeriodDays sets the grace period in days
func WithGracePeriodDays(days int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.GracePeriodDays = days
	}
}

// WithTOTP sets the TOTP configuration
func WithTOTP(enabled bool, issuer string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TOTP.Enabled = enabled
		p.defaultConfig.TOTP.Issuer = issuer
	}
}

// WithSMS sets the SMS configuration
func WithSMS(enabled bool, codeLength, expiryMinutes int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SMS.Enabled = enabled
		p.defaultConfig.SMS.CodeLength = codeLength
		p.defaultConfig.SMS.CodeExpiryMinutes = expiryMinutes
	}
}

// WithEmail sets the email configuration
func WithEmail(enabled bool, codeLength, expiryMinutes int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Email.Enabled = enabled
		p.defaultConfig.Email.CodeLength = codeLength
		p.defaultConfig.Email.CodeExpiryMinutes = expiryMinutes
	}
}

// WithBackupCodes sets the backup codes configuration
func WithBackupCodes(enabled bool, count, length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.BackupCodes.Enabled = enabled
		p.defaultConfig.BackupCodes.Count = count
		p.defaultConfig.BackupCodes.Length = length
	}
}

// WithAdaptiveMFA sets the adaptive MFA configuration
func WithAdaptiveMFA(enabled bool, threshold float64) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AdaptiveMFA.Enabled = enabled
		p.defaultConfig.AdaptiveMFA.RiskThreshold = threshold
	}
}

// NewPlugin creates a new MFA plugin with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	// Set config to default config
	p.config = p.defaultConfig

	return p
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "mfa"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(authInstance core.Authsome) error {
	if authInstance == nil {
		return errs.InternalServerError("auth instance is nil", nil)
	}

	p.db = authInstance.GetDB()
	if p.db == nil {
		return errs.InternalServerError("database not available", nil)
	}

	// Get Forge app and config manager
	forgeApp := authInstance.GetForgeApp()
	if forgeApp != nil {
		configManager := forgeApp.Config()
		p.logger = forgeApp.Logger()

		// Bind configuration using Forge ConfigManager with provided defaults
		if err := configManager.BindWithDefault("auth.mfa", p.config, p.defaultConfig); err != nil {
			// Log but don't fail - use defaults
			p.logger.Warn("failed to bind config", forge.F("error", err.Error()))
			p.config = p.defaultConfig
		}
	} else {
		// Fallback to default config if no Forge app
		p.config = p.defaultConfig
	}

	// Get notification adapter from service registry
	svcRegistry := authInstance.GetServiceRegistry()
	if svcRegistry != nil {
		if adapter, exists := svcRegistry.Get("notification.adapter"); exists {
			if typedAdapter, ok := adapter.(*notificationPlugin.Adapter); ok {
				p.notifAdapter = typedAdapter
				p.logger.Info("retrieved notification adapter from service registry")
			} else {
				p.logger.Warn("notification adapter type assertion failed")
			}
		} else {
			p.logger.Info("notification adapter not available in service registry (graceful degradation)")
		}
	}

	// Initialize adapter registry
	p.adapterRegistry = NewFactorAdapterRegistry()

	// Initialize MFA repository
	mfaRepo := repo.NewMFARepository(p.db)

	// Initialize dependent plugin services
	p.initializeDependentServices(p.db, authInstance)

	// Register factor adapters
	p.registerFactorAdapters()

	// Create MFA service
	p.service = NewService(mfaRepo, p.adapterRegistry, p.notifAdapter, p.config)

	p.logger.Info("plugin initialized successfully")
	p.logger.Info("registered factor adapters", forge.F("count", len(p.adapterRegistry.List())))

	return nil
}

// initializeDependentServices initializes services from other plugins
func (p *Plugin) initializeDependentServices(db *bun.DB, authInstance core.Authsome) {
	// Initialize twofa service (for TOTP and backup codes)
	twofaRepo := repo.NewTwoFARepository(db)
	p.twofaService = twofa.NewService(twofaRepo, twofa.Config{
		TOTPIssuer:       "AuthSome",
		BackupCodeCount:  10,
		BackupCodeLength: 8,
		TOTPPeriod:       30,
		TOTPDigits:       6,
	})

	// Initialize emailotp service (for email factor)
	// Note: In production, these would be injected rather than created here
	// For now, create minimal instances
	emailOTPRepo := repo.NewEmailOTPRepository(db)
	p.emailOTPService = emailotp.NewService(
		emailOTPRepo,
		authInstance.GetServiceRegistry().UserService(),
		authInstance.GetServiceRegistry().SessionService(),
		authInstance.GetServiceRegistry().AuditService(),
		nil, // notifAdapter
		emailotp.Config{
			OTPLength:     6,
			ExpiryMinutes: 5 * 60, // 5 minutes
			MaxAttempts:   5,
			DevExposeOTP:  true,
		},
		nil, // logger
	)

	// Initialize phone service (for SMS factor)
	phoneRepo := repo.NewPhoneRepository(db)
	p.phoneService = phone.NewService(
		phoneRepo,
		nil, // userService
		nil, // authService
		nil, // auditService
		nil, // smsProvider
		phone.Config{
			CodeLength:    6,
			ExpiryMinutes: 5 * 60,
			MaxAttempts:   5,
			DevExposeCode: true,
		},
	)

	// passkey service would be initialized here when stable
}

// registerFactorAdapters registers all available factor adapters
func (p *Plugin) registerFactorAdapters() {
	// Register TOTP adapter
	if p.config.TOTP.Enabled && p.twofaService != nil {
		totpAdapter := NewTOTPFactorAdapter(p.twofaService, true)
		p.adapterRegistry.Register(totpAdapter)
	}

	// Register backup code adapter
	if p.config.BackupCodes.Enabled && p.twofaService != nil {
		backupAdapter := NewBackupCodeFactorAdapter(p.twofaService, true)
		p.adapterRegistry.Register(backupAdapter)
	}

	// Register email adapter
	if p.config.Email.Enabled {
		emailAdapter := NewEmailFactorAdapter(p.emailOTPService, p.notifAdapter, true)
		p.adapterRegistry.Register(emailAdapter)
	}

	// Register SMS adapter
	if p.config.SMS.Enabled {
		smsAdapter := NewSMSFactorAdapter(p.phoneService, p.notifAdapter, true)
		p.adapterRegistry.Register(smsAdapter)
	}

	// Register WebAuthn adapter (when passkey plugin is stable)
	// if p.config.WebAuthn.Enabled && p.passkeyService != nil {
	//     webauthnAdapter := NewWebAuthnFactorAdapter(p.passkeyService, true)
	//     p.adapterRegistry.Register(webauthnAdapter)
	// }
}

// RegisterRoutes registers MFA endpoints
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized, Init must be called first")
	}

	// Create handler
	handler := NewHandler(p.service)

	// Register main MFA routes
	RegisterRoutes(router, handler)

	// Add a test endpoint to verify MFA is loaded
	router.GET("/mfa/ping", func(c forge.Context) error {
		return c.JSON(200, map[string]interface{}{
			"plugin":            "mfa",
			"version":           "1.0.0",
			"enabled":           p.config.Enabled,
			"available_factors": p.adapterRegistry.GetAvailable(),
		})
	})

	return nil
}

// RegisterHooks registers MFA-related hooks
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error {
	// MFA can register hooks for:
	// - After user creation: suggest MFA enrollment
	// - Before sensitive operations: require step-up auth
	// - After failed login: increase MFA requirement

	// TODO: Implement hooks when needed
	return nil
}

// RegisterServiceDecorators allows MFA to enhance core services
func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error {
	// MFA could decorate:
	// - AuthService: add MFA checks after password verification
	// - SessionService: add MFA session validation

	// TODO: Implement decorators when needed
	return nil
}

// Migrate creates required database tables
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()

	// Create MFA tables
	tables := []interface{}{
		(*schema.MFAFactor)(nil),
		(*schema.MFAChallenge)(nil),
		(*schema.MFASession)(nil),
		(*schema.MFATrustedDevice)(nil),
		(*schema.MFAPolicy)(nil),
		(*schema.MFAAttempt)(nil),
		(*schema.MFARiskAssessment)(nil),
	}

	for _, table := range tables {
		if _, err := p.db.NewCreateTable().
			Model(table).
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}
	}

	// Create indexes for performance
	if err := p.createIndexes(ctx); err != nil {
		return err
	}

	return nil
}

// createIndexes creates database indexes for MFA tables
func (p *Plugin) createIndexes(ctx context.Context) error {
	// Index on user_id for fast lookups
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_mfa_factors_user_id ON mfa_factors(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_factors_user_status ON mfa_factors(user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_challenges_session_id ON mfa_challenges(session_id)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_challenges_user_id ON mfa_challenges(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_sessions_user_id ON mfa_sessions(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_sessions_token ON mfa_sessions(session_token)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_trusted_devices_user_id ON mfa_trusted_devices(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_trusted_devices_device_id ON mfa_trusted_devices(user_id, device_id)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_attempts_user_id ON mfa_attempts(user_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_mfa_risk_user_id ON mfa_risk_assessments(user_id, created_at)",
	}

	for _, indexSQL := range indexes {
		if _, err := p.db.ExecContext(ctx, indexSQL); err != nil {
			// Log but don't fail - indexes might already exist
			// In production, use proper logging
			_ = err
		}
	}

	return nil
}

// Service returns the MFA service (for use by middleware and other components)
func (p *Plugin) Service() *Service {
	return p.service
}

// Config returns the plugin configuration
func (p *Plugin) Config() *Config {
	return p.config
}

// WithConfig sets custom configuration
func (p *Plugin) WithConfig(config *Config) *Plugin {
	p.config = config
	return p
}
