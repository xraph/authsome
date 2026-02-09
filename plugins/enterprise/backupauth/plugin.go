package backupauth

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

const (
	PluginID      = "backupauth"
	PluginName    = "Backup Authentication & Recovery"
	PluginVersion = "1.0.0"
)

// Plugin implements the AuthSome plugin interface for backup authentication.
type Plugin struct {
	service       *Service
	config        *Config
	handler       *Handler
	repo          Repository
	providers     ProviderRegistry
	db            *bun.DB
	cleanupTicker *time.Ticker
	cleanupDone   chan bool
}

// NewPlugin creates a new backup authentication plugin instance.
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the unique plugin identifier.
func (p *Plugin) ID() string {
	return PluginID
}

// Name returns the human-readable plugin name.
func (p *Plugin) Name() string {
	return PluginName
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return PluginVersion
}

// Description returns the plugin description.
func (p *Plugin) Description() string {
	return "Enterprise backup authentication and account recovery with multiple verification methods including recovery codes, security questions, trusted contacts, email/SMS verification, video verification, and document upload"
}

// Init initializes the plugin with dependencies from AuthSome.
func (p *Plugin) Init(auth any) error {
	// Type assert to get the auth instance with required methods
	authInstance, ok := auth.(interface {
		GetDB() *bun.DB
		GetForgeApp() forge.App
		GetServiceRegistry() *registry.ServiceRegistry
	})
	if !ok {
		return errs.InternalServerErrorWithMessage("invalid auth instance type")
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

// RegisterRoutes registers HTTP routes for the plugin.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled || p.handler == nil {
		return nil
	}

	// Recovery session routes (public - no auth required for recovery)
	router.POST("/recovery/start", p.handler.StartRecovery,
		forge.WithName("backupauth.recovery.start"), forge.WithSummary("Start account recovery"), forge.WithDescription("Initialize account recovery process"),
		forge.WithResponseSchema(200, "Recovery started", BackupAuthRecoveryResponse{}), forge.WithTags("Backup Auth", "Recovery"), forge.WithValidation(true))
	router.POST("/recovery/continue", p.handler.ContinueRecovery,
		forge.WithName("backupauth.recovery.continue"), forge.WithSummary("Continue recovery"), forge.WithDescription("Continue multi-step recovery process"),
		forge.WithResponseSchema(200, "Recovery continued", BackupAuthRecoveryResponse{}), forge.WithTags("Backup Auth", "Recovery"), forge.WithValidation(true))
	router.POST("/recovery/complete", p.handler.CompleteRecovery,
		forge.WithName("backupauth.recovery.complete"), forge.WithSummary("Complete recovery"), forge.WithDescription("Complete account recovery and restore access"),
		forge.WithResponseSchema(200, "Recovery completed", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Recovery"), forge.WithValidation(true))
	router.POST("/recovery/cancel", p.handler.CancelRecovery,
		forge.WithName("backupauth.recovery.cancel"), forge.WithSummary("Cancel recovery"), forge.WithDescription("Cancel ongoing recovery process"),
		forge.WithResponseSchema(200, "Recovery cancelled", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Recovery"))

	// Recovery codes (authenticated)
	router.POST("/recovery-codes/generate", p.handler.GenerateRecoveryCodes,
		forge.WithName("backupauth.codes.generate"), forge.WithSummary("Generate recovery codes"), forge.WithDescription("Generate backup recovery codes"),
		forge.WithResponseSchema(200, "Codes generated", BackupAuthCodesResponse{}), forge.WithTags("Backup Auth", "Codes"), forge.WithValidation(true))
	router.POST("/recovery-codes/verify", p.handler.VerifyRecoveryCode,
		forge.WithName("backupauth.codes.verify"), forge.WithSummary("Verify recovery code"), forge.WithDescription("Verify a recovery code for authentication"),
		forge.WithResponseSchema(200, "Code verified", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Codes"), forge.WithValidation(true))

	// Security questions (authenticated for setup, public for recovery)
	router.POST("/security-questions/setup", p.handler.SetupSecurityQuestions,
		forge.WithName("backupauth.questions.setup"), forge.WithSummary("Setup security questions"), forge.WithDescription("Configure security questions for recovery"),
		forge.WithResponseSchema(200, "Questions setup", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Security Questions"), forge.WithValidation(true))
	router.POST("/security-questions/get", p.handler.GetSecurityQuestions,
		forge.WithName("backupauth.questions.get"), forge.WithSummary("Get security questions"), forge.WithDescription("Retrieve user's security questions"),
		forge.WithResponseSchema(200, "Questions retrieved", BackupAuthQuestionsResponse{}), forge.WithTags("Backup Auth", "Security Questions"))
	router.POST("/security-questions/verify", p.handler.VerifySecurityAnswers,
		forge.WithName("backupauth.questions.verify"), forge.WithSummary("Verify security answers"), forge.WithDescription("Verify answers to security questions"),
		forge.WithResponseSchema(200, "Answers verified", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Security Questions"), forge.WithValidation(true))

	// Trusted contacts (authenticated)
	router.POST("/trusted-contacts/add", p.handler.AddTrustedContact,
		forge.WithName("backupauth.contacts.add"), forge.WithSummary("Add trusted contact"), forge.WithDescription("Add a trusted contact for account recovery"),
		forge.WithResponseSchema(200, "Contact added", BackupAuthContactResponse{}), forge.WithTags("Backup Auth", "Trusted Contacts"), forge.WithValidation(true))
	router.GET("/trusted-contacts", p.handler.ListTrustedContacts,
		forge.WithName("backupauth.contacts.list"), forge.WithSummary("List trusted contacts"), forge.WithDescription("List all trusted contacts"),
		forge.WithResponseSchema(200, "Contacts retrieved", BackupAuthContactsResponse{}), forge.WithTags("Backup Auth", "Trusted Contacts"))
	router.POST("/trusted-contacts/verify", p.handler.VerifyTrustedContact,
		forge.WithName("backupauth.contacts.verify"), forge.WithSummary("Verify trusted contact"), forge.WithDescription("Verify identity through trusted contact"),
		forge.WithResponseSchema(200, "Contact verified", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Trusted Contacts"), forge.WithValidation(true))
	router.POST("/trusted-contacts/request-verification", p.handler.RequestTrustedContactVerification,
		forge.WithName("backupauth.contacts.request"), forge.WithSummary("Request contact verification"), forge.WithDescription("Request verification from trusted contact"),
		forge.WithResponseSchema(200, "Verification requested", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Trusted Contacts"), forge.WithValidation(true))
	router.DELETE("/trusted-contacts/:id", p.handler.RemoveTrustedContact,
		forge.WithName("backupauth.contacts.remove"), forge.WithSummary("Remove trusted contact"), forge.WithDescription("Remove a trusted contact"),
		forge.WithResponseSchema(200, "Contact removed", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Trusted Contacts"))

	// Email/SMS verification (public)
	router.POST("/verification/send", p.handler.SendVerificationCode,
		forge.WithName("backupauth.verification.send"), forge.WithSummary("Send verification code"), forge.WithDescription("Send verification code via email/SMS"),
		forge.WithResponseSchema(200, "Code sent", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Verification"), forge.WithValidation(true))
	router.POST("/verification/verify", p.handler.VerifyCode,
		forge.WithName("backupauth.verification.verify"), forge.WithSummary("Verify code"), forge.WithDescription("Verify email/SMS verification code"),
		forge.WithResponseSchema(200, "Code verified", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Verification"), forge.WithValidation(true))

	// Video verification (public for scheduling, admin for completion)
	router.POST("/video/schedule", p.handler.ScheduleVideoSession,
		forge.WithName("backupauth.video.schedule"), forge.WithSummary("Schedule video session"), forge.WithDescription("Schedule a video verification session"),
		forge.WithResponseSchema(200, "Session scheduled", BackupAuthVideoResponse{}), forge.WithTags("Backup Auth", "Video"), forge.WithValidation(true))
	router.POST("/video/start", p.handler.StartVideoSession,
		forge.WithName("backupauth.video.start"), forge.WithSummary("Start video session"), forge.WithDescription("Start a scheduled video verification session"),
		forge.WithResponseSchema(200, "Session started", BackupAuthVideoResponse{}), forge.WithTags("Backup Auth", "Video"))
	router.POST("/video/complete", p.handler.CompleteVideoSession,
		forge.WithName("backupauth.video.complete"), forge.WithSummary("Complete video session"), forge.WithDescription("Complete video verification (admin only)"),
		forge.WithResponseSchema(200, "Session completed", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Video", "Admin"))

	// Document verification (public for upload, admin for review)
	router.POST("/documents/upload", p.handler.UploadDocument,
		forge.WithName("backupauth.documents.upload"), forge.WithSummary("Upload document"), forge.WithDescription("Upload identity document for verification"),
		forge.WithResponseSchema(200, "Document uploaded", BackupAuthDocumentResponse{}), forge.WithTags("Backup Auth", "Documents"), forge.WithValidation(true))
	router.GET("/documents/:id", p.handler.GetDocumentVerification,
		forge.WithName("backupauth.documents.get"), forge.WithSummary("Get document status"), forge.WithDescription("Get document verification status"),
		forge.WithResponseSchema(200, "Document status", BackupAuthDocumentResponse{}), forge.WithTags("Backup Auth", "Documents"))
	router.POST("/documents/:id/review", p.handler.ReviewDocument,
		forge.WithName("backupauth.documents.review"), forge.WithSummary("Review document"), forge.WithDescription("Review uploaded document (admin only)"),
		forge.WithResponseSchema(200, "Document reviewed", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Documents", "Admin"))

	// Admin routes
	adminGroup := router.Group("/admin")
	{
		adminGroup.GET("/sessions", p.handler.ListRecoverySessions,
			forge.WithName("backupauth.admin.sessions"), forge.WithSummary("List recovery sessions"), forge.WithDescription("List all recovery sessions"),
			forge.WithResponseSchema(200, "Sessions retrieved", BackupAuthSessionsResponse{}), forge.WithTags("Backup Auth", "Admin"))
		adminGroup.POST("/sessions/:id/approve", p.handler.ApproveRecovery,
			forge.WithName("backupauth.admin.approve"), forge.WithSummary("Approve recovery"), forge.WithDescription("Approve a recovery request"),
			forge.WithResponseSchema(200, "Recovery approved", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Admin"))
		adminGroup.POST("/sessions/:id/reject", p.handler.RejectRecovery,
			forge.WithName("backupauth.admin.reject"), forge.WithSummary("Reject recovery"), forge.WithDescription("Reject a recovery request"),
			forge.WithResponseSchema(200, "Recovery rejected", BackupAuthStatusResponse{}), forge.WithTags("Backup Auth", "Admin"))
		adminGroup.GET("/stats", p.handler.GetRecoveryStats,
			forge.WithName("backupauth.admin.stats"), forge.WithSummary("Get recovery stats"), forge.WithDescription("Get recovery statistics"),
			forge.WithResponseSchema(200, "Stats retrieved", BackupAuthStatsResponse{}), forge.WithTags("Backup Auth", "Admin"))
		adminGroup.GET("/config", p.handler.GetRecoveryConfig,
			forge.WithName("backupauth.admin.config"), forge.WithSummary("Get recovery config"), forge.WithDescription("Get recovery configuration"),
			forge.WithResponseSchema(200, "Config retrieved", BackupAuthConfigResponse{}), forge.WithTags("Backup Auth", "Admin"))
		adminGroup.PUT("/config", p.handler.UpdateRecoveryConfig,
			forge.WithName("backupauth.admin.config.update"), forge.WithSummary("Update recovery config"), forge.WithDescription("Update recovery configuration"),
			forge.WithResponseSchema(200, "Config updated", BackupAuthConfigResponse{}), forge.WithTags("Backup Auth", "Admin"), forge.WithValidation(true))
	}

	// Health check
	if err := router.GET("/health", p.handler.HealthCheck); err != nil { return err }

	return nil
}

// RegisterHooks registers plugin hooks with the hook registry.
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if !p.config.Enabled || p.service == nil {
		return nil
	}

	// TODO: Register hooks for user lifecycle events
	// hookRegistry.RegisterAfterUserCreate(p.onUserCreated)
	// hookRegistry.RegisterBeforeUserDelete(p.onBeforeUserDelete)

	return nil
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions.
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Backup auth plugin doesn't decorate core services
	// It provides its own service that's accessed via the plugin
	return nil
}

// Migrate performs database migrations.
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()

	// Create tables
	models := []any{
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

// startBackgroundTasks starts background tasks for the plugin.
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

// runSessionCleanup expires old recovery sessions.
func (p *Plugin) runSessionCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	count, err := p.repo.ExpireRecoverySessions(ctx, time.Now())
	if err != nil {
		// Log error (would use proper logger in production)
		return
	}

	if count > 0 {
	}
}

// Shutdown gracefully shuts down the plugin.
func (p *Plugin) Shutdown(ctx context.Context) error {
	// Stop background tasks
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
		close(p.cleanupDone)
	}

	return nil
}

// Health checks plugin health.
func (p *Plugin) Health(ctx context.Context) error {
	if p.service == nil {
		return errs.InternalServerErrorWithMessage("service not initialized")
	}

	// Check if database is accessible (simple query)
	// TODO: Implement actual health check query

	return nil
}

// Service returns the backup auth service for programmatic access (optional public method).
func (p *Plugin) Service() *Service {
	return p.service
}

// DTOs for backupauth routes.
type BackupAuthStatusResponse struct {
	Status string `example:"success" json:"status"`
}

type BackupAuthRecoveryResponse struct {
	SessionID string `example:"session_123" json:"session_id"`
}

type BackupAuthCodesResponse struct {
	Codes []string `example:"[\"code1\",\"code2\"]" json:"codes"`
}

type BackupAuthQuestionsResponse struct {
	Questions []string `json:"questions"`
}

type BackupAuthContactResponse struct {
	ID string `example:"contact_123" json:"id"`
}

type BackupAuthContactsResponse struct {
	Contacts []any `json:"contacts"`
}

type BackupAuthVideoResponse struct {
	SessionID string `example:"video_123" json:"session_id"`
}

type BackupAuthDocumentResponse struct {
	ID string `example:"doc_123" json:"id"`
}

type BackupAuthSessionsResponse struct {
	Sessions []any `json:"sessions"`
}

type BackupAuthStatsResponse struct {
	Stats any `json:"stats"`
}

type BackupAuthConfigResponse struct {
	Config any `json:"config"`
}

// SetProviders allows setting custom providers.
func (p *Plugin) SetProviders(providers ProviderRegistry) {
	p.providers = providers
	if p.service != nil {
		p.service.providers = providers
	}
}

// SetEmailProvider sets a custom email provider.
func (p *Plugin) SetEmailProvider(provider EmailProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetEmailProvider(provider)
	}
}

// SetSMSProvider sets a custom SMS provider.
func (p *Plugin) SetSMSProvider(provider SMSProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetSMSProvider(provider)
	}
}

// SetVideoProvider sets a custom video verification provider.
func (p *Plugin) SetVideoProvider(provider VideoProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetVideoProvider(provider)
	}
}

// SetDocumentProvider sets a custom document verification provider.
func (p *Plugin) SetDocumentProvider(provider DocumentProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetDocumentProvider(provider)
	}
}

// SetNotificationProvider sets a custom notification provider.
func (p *Plugin) SetNotificationProvider(provider NotificationProvider) {
	if registry, ok := p.providers.(*DefaultProviderRegistry); ok {
		registry.SetNotificationProvider(provider)
	}
}
