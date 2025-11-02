package backupauth

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/forge"
)

const (
	PluginID      = "backupauth"
	PluginName    = "Backup Authentication & Recovery"
	PluginVersion = "1.0.0"
)

// Plugin implements the AuthSome plugin interface for backup authentication
type Plugin struct {
	service          *Service
	config           *Config
	handler          *Handler
	repo             Repository
	providers        ProviderRegistry
	db               *bun.DB
	cleanupTicker    *time.Ticker
	cleanupDone      chan bool
}

// NewPlugin creates a new backup authentication plugin instance
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the unique plugin identifier
func (p *Plugin) ID() string {
	return PluginID
}

// Name returns the human-readable plugin name
func (p *Plugin) Name() string {
	return PluginName
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return PluginVersion
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Enterprise backup authentication and account recovery with multiple verification methods including recovery codes, security questions, trusted contacts, email/SMS verification, video verification, and document upload"
}

// Init initializes the plugin with dependencies from AuthSome
func (p *Plugin) Init(auth interface{}) error {
	// Type assert to get the auth instance with required methods
	authInstance, ok := auth.(interface {
		GetDB() *bun.DB
		GetForgeApp() forge.App
		GetServiceRegistry() *registry.ServiceRegistry
	})
	if !ok {
		return fmt.Errorf("invalid auth instance type")
	}

	p.db = authInstance.GetDB()
	forgeApp := authInstance.GetForgeApp()
	configManager := forgeApp.Config()

	// Load configuration from Forge config manager
	var config Config
	if err := configManager.Bind("auth.backupauth", &config); err != nil {
		// Use defaults if binding fails
		config = *DefaultConfig()
	}
	config.Validate() // Ensure defaults are set
	p.config = &config

	if !p.config.Enabled {
		return nil
	}

	// Initialize provider registry
	p.providers = NewDefaultProviderRegistry()

	// Initialize repository
	p.repo = NewBunRepository(p.db)

	// Initialize service
	p.service = NewService(
		p.repo,
		p.config,
		p.providers,
	)

	// Initialize handler
	p.handler = NewHandler(p.service)

	// Start background tasks
	p.startBackgroundTasks()

	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled || p.handler == nil {
		return nil
	}

	// Recovery session routes (public - no auth required for recovery)
	router.POST("/recovery/start", p.handler.StartRecovery)
	router.POST("/recovery/continue", p.handler.ContinueRecovery)
	router.POST("/recovery/complete", p.handler.CompleteRecovery)
	router.POST("/recovery/cancel", p.handler.CancelRecovery)

	// Recovery codes (authenticated)
	router.POST("/recovery-codes/generate", p.handler.GenerateRecoveryCodes)
	router.POST("/recovery-codes/verify", p.handler.VerifyRecoveryCode)

	// Security questions (authenticated for setup, public for recovery)
	router.POST("/security-questions/setup", p.handler.SetupSecurityQuestions)
	router.POST("/security-questions/get", p.handler.GetSecurityQuestions)
	router.POST("/security-questions/verify", p.handler.VerifySecurityAnswers)

	// Trusted contacts (authenticated)
	router.POST("/trusted-contacts/add", p.handler.AddTrustedContact)
	router.GET("/trusted-contacts", p.handler.ListTrustedContacts)
	router.POST("/trusted-contacts/verify", p.handler.VerifyTrustedContact)
	router.POST("/trusted-contacts/request-verification", p.handler.RequestTrustedContactVerification)
	router.DELETE("/trusted-contacts/:id", p.handler.RemoveTrustedContact)

	// Email/SMS verification (public)
	router.POST("/verification/send", p.handler.SendVerificationCode)
	router.POST("/verification/verify", p.handler.VerifyCode)

	// Video verification (public for scheduling, admin for completion)
	router.POST("/video/schedule", p.handler.ScheduleVideoSession)
	router.POST("/video/start", p.handler.StartVideoSession)
	router.POST("/video/complete", p.handler.CompleteVideoSession) // Admin only

	// Document verification (public for upload, admin for review)
	router.POST("/documents/upload", p.handler.UploadDocument)
	router.GET("/documents/:id", p.handler.GetDocumentVerification)
	router.POST("/documents/:id/review", p.handler.ReviewDocument) // Admin only

	// Admin routes
	adminGroup := router.Group("/admin")
	{
		adminGroup.GET("/sessions", p.handler.ListRecoverySessions)
		adminGroup.POST("/sessions/:id/approve", p.handler.ApproveRecovery)
		adminGroup.POST("/sessions/:id/reject", p.handler.RejectRecovery)
		adminGroup.GET("/stats", p.handler.GetRecoveryStats)
		adminGroup.GET("/config", p.handler.GetRecoveryConfig)
		adminGroup.PUT("/config", p.handler.UpdateRecoveryConfig)
	}

	// Health check
	router.GET("/health", p.handler.HealthCheck)

	return nil
}

// RegisterHooks registers plugin hooks with the hook registry
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if !p.config.Enabled || p.service == nil {
		return nil
	}

	// TODO: Register hooks for user lifecycle events
	// hookRegistry.RegisterAfterUserCreate(p.onUserCreated)
	// hookRegistry.RegisterBeforeUserDelete(p.onBeforeUserDelete)

	return nil
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Backup auth plugin doesn't decorate core services
	// It provides its own service that's accessed via the plugin
	return nil
}

// Migrate performs database migrations
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()

	// Create tables
	models := []interface{}{
		(*SecurityQuestion)(nil),
		(*TrustedContact)(nil),
		(*RecoverySession)(nil),
		(*VideoVerificationSession)(nil),
		(*DocumentVerification)(nil),
		(*RecoveryAttemptLog)(nil),
		(*RecoveryConfiguration)(nil),
		(*RecoveryCodeUsage)(nil),
	}

	for _, model := range models {
		_, err := p.db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// startBackgroundTasks starts background tasks for the plugin
func (p *Plugin) startBackgroundTasks() {
	if !p.config.Enabled {
		return
	}

	// Start session cleanup
	if p.config.MultiStepRecovery.SessionExpiry > 0 {
		p.cleanupTicker = time.NewTicker(1 * time.Hour)
		p.cleanupDone = make(chan bool)

		go func() {
			for {
				select {
				case <-p.cleanupTicker.C:
					p.runSessionCleanup()
				case <-p.cleanupDone:
					return
				}
			}
		}()
	}
}

// runSessionCleanup expires old recovery sessions
func (p *Plugin) runSessionCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	count, err := p.repo.ExpireRecoverySessions(ctx, time.Now())
	if err != nil {
		// Log error (would use proper logger in production)
		fmt.Printf("[Backup Auth Plugin] Failed to expire sessions: %v\n", err)
		return
	}

	if count > 0 {
		fmt.Printf("[Backup Auth Plugin] Expired %d recovery session(s)\n", count)
	}
}

// Shutdown gracefully shuts down the plugin
func (p *Plugin) Shutdown(ctx context.Context) error {
	// Stop background tasks
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
		close(p.cleanupDone)
	}

	return nil
}

// Health checks plugin health
func (p *Plugin) Health(ctx context.Context) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}

	// Check if database is accessible (simple query)
	// TODO: Implement actual health check query

	return nil
}

// Service returns the backup auth service for programmatic access (optional public method)
func (p *Plugin) Service() *Service {
	return p.service
}

// SetProviders allows setting custom providers
func (p *Plugin) SetProviders(providers ProviderRegistry) {
	p.providers = providers
	if p.service != nil {
		p.service.providers = providers
	}
}

// SetEmailProvider sets a custom email provider
func (p *Plugin) SetEmailProvider(provider EmailProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetEmailProvider(provider)
	}
}

// SetSMSProvider sets a custom SMS provider
func (p *Plugin) SetSMSProvider(provider SMSProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetSMSProvider(provider)
	}
}

// SetVideoProvider sets a custom video verification provider
func (p *Plugin) SetVideoProvider(provider VideoProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetVideoProvider(provider)
	}
}

// SetDocumentProvider sets a custom document verification provider
func (p *Plugin) SetDocumentProvider(provider DocumentProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetDocumentProvider(provider)
	}
}

// SetNotificationProvider sets a custom notification provider
func (p *Plugin) SetNotificationProvider(provider NotificationProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetNotificationProvider(provider)
	}
}

