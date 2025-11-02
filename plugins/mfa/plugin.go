package mfa

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/plugins/emailotp"
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
	config          *Config

	// Dependencies from other plugins
	twofaService    *twofa.Service
	emailOTPService *emailotp.Service
	phoneService    *phone.Service
	// passkeyService  *passkey.Service // Uncomment when passkey is stable
}

// NewPlugin creates a new MFA plugin
func NewPlugin() *Plugin {
	return &Plugin{
		config: DefaultConfig(),
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "mfa"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(auth interface{}) error {
	// Extract database from auth instance
	type authInterface interface {
		GetDB() *bun.DB
		GetServiceRegistry() *registry.ServiceRegistry
	}

	authInstance, ok := auth.(authInterface)
	if !ok {
		return fmt.Errorf("invalid auth instance type")
	}

	p.db = authInstance.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available")
	}

	// Initialize adapter registry
	p.adapterRegistry = NewFactorAdapterRegistry()

	// Initialize MFA repository
	mfaRepo := repo.NewMFARepository(p.db)

	// Initialize dependent plugin services
	p.initializeDependentServices(p.db)

	// Register factor adapters
	p.registerFactorAdapters()

	// Create MFA service
	p.service = NewService(mfaRepo, p.adapterRegistry, p.config)

	fmt.Println("[MFA] Plugin initialized successfully")
	fmt.Printf("[MFA] Registered %d factor adapters\n", len(p.adapterRegistry.List()))

	return nil
}

// initializeDependentServices initializes services from other plugins
func (p *Plugin) initializeDependentServices(db *bun.DB) {
	// Initialize twofa service (for TOTP and backup codes)
	twofaRepo := repo.NewTwoFARepository(db)
	p.twofaService = twofa.NewService(twofaRepo)

	// Initialize emailotp service (for email factor)
	// Note: In production, these would be injected rather than created here
	// For now, create minimal instances
	emailOTPRepo := repo.NewEmailOTPRepository(db)
	p.emailOTPService = emailotp.NewService(
		emailOTPRepo,
		nil, // userService - would need proper injection
		nil, // authService
		nil, // auditService
		nil, // emailProvider
		emailotp.Config{
			OTPLength:     6,
			ExpiryMinutes: 5 * 60, // 5 minutes
			MaxAttempts:   5,
			DevExposeOTP:  true,
		},
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
	if p.config.Email.Enabled && p.emailOTPService != nil {
		emailAdapter := NewEmailFactorAdapter(p.emailOTPService, true)
		p.adapterRegistry.Register(emailAdapter)
	}

	// Register SMS adapter
	if p.config.SMS.Enabled && p.phoneService != nil {
		smsAdapter := NewSMSFactorAdapter(p.phoneService, true)
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

	fmt.Println("[MFA] Routes registered successfully")
	fmt.Println("[MFA] Available endpoints:")
	fmt.Println("  - POST /mfa/factors/enroll")
	fmt.Println("  - GET  /mfa/factors")
	fmt.Println("  - POST /mfa/challenge")
	fmt.Println("  - POST /mfa/verify")
	fmt.Println("  - GET  /mfa/status")
	fmt.Println("  - GET  /mfa/ping (test endpoint)")

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
