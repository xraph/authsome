package idverification

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the identity verification plugin.
type Plugin struct {
	config         Config
	service        *Service
	handler        *Handler
	middleware     *Middleware
	repo           Repository
	db             *bun.DB
	auditService   *audit.Service
	webhookService *webhook.Service
}

// NewPlugin creates a new identity verification plugin.
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin ID.
func (p *Plugin) ID() string {
	return "idverification"
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "Identity Verification (KYC)"
}

// Description returns the plugin description.
func (p *Plugin) Description() string {
	return "Enterprise-grade identity verification and KYC compliance with support for multiple providers (Onfido, Jumio, Stripe Identity)"
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Init initializes the plugin.
func (p *Plugin) Init(container any) error {
	// Type assert the container to get dependencies
	// This assumes a DI container pattern
	deps, ok := container.(map[string]any)
	if !ok {
		return errs.BadRequest("invalid container type")
	}

	// Get database
	db, ok := deps["db"].(*bun.DB)
	if !ok {
		return errs.InternalServerErrorWithMessage("database not found in container")
	}

	p.db = db

	// Get config manager
	configManager, ok := deps["config"].(forge.ConfigManager)
	if !ok {
		return errs.InternalServerErrorWithMessage("config manager not found in container")
	}

	// config configuration
	var config Config
	if err := configManager.Bind("auth.idverification", &config); err != nil {
		// config default config if not found
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	p.config = config

	// Get optional services
	if auditSvc, ok := deps["audit"].(*audit.Service); ok {
		p.auditService = auditSvc
	}

	if webhookSvc, ok := deps["webhook"].(*webhook.Service); ok {
		p.webhookService = webhookSvc
	}

	// Create repository
	p.repo = repository.NewIdentityVerificationRepository(db)

	// Create service
	service, err := NewService(p.repo, config, p.auditService, p.webhookService)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	p.service = service

	// Create handler
	p.handler = NewHandler(service)

	// Create middleware
	p.middleware = NewMiddleware(service)

	return nil
}

// RegisterRoutes registers the plugin routes.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled {
		return nil
	}

	// Public routes (require authentication)
	verificationGroup := router.Group("/verification")
	{
		// Session management
		if err := verificationGroup.POST("/sessions", p.handler.CreateVerificationSession,
			forge.WithName("idverification.sessions.create"),
			forge.WithSummary("Create verification session"),
			forge.WithDescription("Creates a new identity verification session with the specified provider (Onfido, Jumio, Stripe Identity)"),
			forge.WithResponseSchema(201, "Session created", IDVerificationSessionResponse{}),
			forge.WithResponseSchema(400, "Invalid request", IDVerificationErrorResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "Sessions"),
			forge.WithValidation(true),
		); err != nil {
			return err
		}
		if err := verificationGroup.GET("/sessions/:id", p.handler.GetVerificationSession,
			forge.WithName("idverification.sessions.get"),
			forge.WithSummary("Get verification session"),
			forge.WithDescription("Retrieves details of a specific verification session by ID"),
			forge.WithResponseSchema(200, "Session retrieved", IDVerificationSessionResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithResponseSchema(404, "Session not found", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "Sessions"),
		); err != nil {
			return err
		}

		// User verifications
		if err := verificationGroup.GET("/me", p.handler.GetUserVerifications,
			forge.WithName("idverification.user.verifications"),
			forge.WithSummary("Get user verifications"),
			forge.WithDescription("Retrieves all identity verifications for the current authenticated user"),
			forge.WithResponseSchema(200, "Verifications retrieved", IDVerificationListResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "User"),
		); err != nil {
			return err
		}
		if err := verificationGroup.GET("/me/status", p.handler.GetUserVerificationStatus,
			forge.WithName("idverification.user.status"),
			forge.WithSummary("Get user verification status"),
			forge.WithDescription("Retrieves the current verification status for the authenticated user"),
			forge.WithResponseSchema(200, "Status retrieved", IDVerificationStatusResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "User"),
		); err != nil {
			return err
		}
		if err := verificationGroup.POST("/me/reverify", p.handler.RequestReverification,
			forge.WithName("idverification.user.reverify"),
			forge.WithSummary("Request reverification"),
			forge.WithDescription("Requests a new identity verification for the current user"),
			forge.WithResponseSchema(200, "Reverification requested", IDVerificationSessionResponse{}),
			forge.WithResponseSchema(400, "Invalid request", IDVerificationErrorResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "User"),
			forge.WithValidation(true),
		); err != nil {
			return err
		}

		// Single verification
		if err := verificationGroup.GET("/:id", p.handler.GetVerification,
			forge.WithName("idverification.get"),
			forge.WithSummary("Get verification"),
			forge.WithDescription("Retrieves details of a specific identity verification by ID"),
			forge.WithResponseSchema(200, "Verification retrieved", IDVerificationResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithResponseSchema(404, "Verification not found", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification"),
		); err != nil {
			return err
		}

		// Webhook endpoint (no auth required, verified by signature)
		if err := verificationGroup.POST("/webhook/:provider", p.handler.HandleWebhook,
			forge.WithName("idverification.webhook"),
			forge.WithSummary("Handle provider webhook"),
			forge.WithDescription("Receives webhook events from identity verification providers (Onfido, Jumio, Stripe Identity). Signature verified"),
			forge.WithResponseSchema(200, "Webhook processed", IDVerificationWebhookResponse{}),
			forge.WithResponseSchema(400, "Invalid webhook", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "Webhooks"),
		); err != nil {
			return err
		}
	}

	// Admin routes (require admin role)
	adminGroup := router.Group("/verification/admin")
	{
		if err := adminGroup.POST("/users/:userId/block", p.handler.AdminBlockUser,
			forge.WithName("idverification.admin.users.block"),
			forge.WithSummary("Block user"),
			forge.WithDescription("Blocks a user from identity verification (admin only)"),
			forge.WithResponseSchema(200, "User blocked", IDVerificationStatusResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithResponseSchema(403, "Insufficient privileges", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "Admin"),
		); err != nil {
			return err
		}
		if err := adminGroup.POST("/users/:userId/unblock", p.handler.AdminUnblockUser,
			forge.WithName("idverification.admin.users.unblock"),
			forge.WithSummary("Unblock user"),
			forge.WithDescription("Unblocks a user for identity verification (admin only)"),
			forge.WithResponseSchema(200, "User unblocked", IDVerificationStatusResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithResponseSchema(403, "Insufficient privileges", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "Admin"),
		); err != nil {
			return err
		}
		if err := adminGroup.GET("/users/:userId/status", p.handler.AdminGetUserVerificationStatus,
			forge.WithName("idverification.admin.users.status"),
			forge.WithSummary("Get user verification status (admin)"),
			forge.WithDescription("Retrieves verification status for a specific user (admin only)"),
			forge.WithResponseSchema(200, "Status retrieved", IDVerificationStatusResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithResponseSchema(403, "Insufficient privileges", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "Admin"),
		); err != nil {
			return err
		}
		if err := adminGroup.GET("/users/:userId/verifications", p.handler.AdminGetUserVerifications,
			forge.WithName("idverification.admin.users.verifications"),
			forge.WithSummary("Get user verifications (admin)"),
			forge.WithDescription("Retrieves all verifications for a specific user (admin only)"),
			forge.WithResponseSchema(200, "Verifications retrieved", IDVerificationListResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", IDVerificationErrorResponse{}),
			forge.WithResponseSchema(403, "Insufficient privileges", IDVerificationErrorResponse{}),
			forge.WithTags("IdentityVerification", "Admin"),
		); err != nil {
			return err
		}
	}

	return nil
}

// Migrate runs database migrations for the plugin.
func (p *Plugin) Migrate() error {
	ctx := context.Background()

	// Create identity_verifications table
	if _, err := p.db.NewCreateTable().
		Model((*schema.IdentityVerification)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create identity_verifications table: %w", err)
	}

	// Create indexes for identity_verifications
	indexes := []struct {
		name    string
		columns []string
	}{
		{"idx_iv_user_id", []string{"user_id"}},
		{"idx_iv_organization_id", []string{"organization_id"}},
		{"idx_iv_provider", []string{"provider"}},
		{"idx_iv_provider_check_id", []string{"provider_check_id"}},
		{"idx_iv_status", []string{"status"}},
		{"idx_iv_verification_type", []string{"verification_type"}},
		{"idx_iv_created_at", []string{"created_at"}},
		{"idx_iv_expires_at", []string{"expires_at"}},
	}

	for _, idx := range indexes {
		if _, err := p.db.NewCreateIndex().
			Model((*schema.IdentityVerification)(nil)).
			Index(idx.name).
			Column(idx.columns...).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	// Create identity_verification_documents table
	if _, err := p.db.NewCreateTable().
		Model((*schema.IdentityVerificationDocument)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create identity_verification_documents table: %w", err)
	}

	// Create indexes for identity_verification_documents
	docIndexes := []struct {
		name    string
		columns []string
	}{
		{"idx_ivd_verification_id", []string{"verification_id"}},
		{"idx_ivd_created_at", []string{"created_at"}},
		{"idx_ivd_retain_until", []string{"retain_until"}},
	}

	for _, idx := range docIndexes {
		if _, err := p.db.NewCreateIndex().
			Model((*schema.IdentityVerificationDocument)(nil)).
			Index(idx.name).
			Column(idx.columns...).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	// Create identity_verification_sessions table
	if _, err := p.db.NewCreateTable().
		Model((*schema.IdentityVerificationSession)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create identity_verification_sessions table: %w", err)
	}

	// Create indexes for identity_verification_sessions
	sessionIndexes := []struct {
		name    string
		columns []string
	}{
		{"idx_ivs_user_id", []string{"user_id"}},
		{"idx_ivs_organization_id", []string{"organization_id"}},
		{"idx_ivs_status", []string{"status"}},
		{"idx_ivs_expires_at", []string{"expires_at"}},
	}

	for _, idx := range sessionIndexes {
		if _, err := p.db.NewCreateIndex().
			Model((*schema.IdentityVerificationSession)(nil)).
			Index(idx.name).
			Column(idx.columns...).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	// Create user_verification_status table
	if _, err := p.db.NewCreateTable().
		Model((*schema.UserVerificationStatus)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create user_verification_status table: %w", err)
	}

	// Create indexes for user_verification_status
	statusIndexes := []struct {
		name    string
		columns []string
	}{
		{"idx_uvs_user_id", []string{"user_id"}},
		{"idx_uvs_organization_id", []string{"organization_id"}},
		{"idx_uvs_verification_level", []string{"verification_level"}},
		{"idx_uvs_is_verified", []string{"is_verified"}},
		{"idx_uvs_is_blocked", []string{"is_blocked"}},
		{"idx_uvs_requires_reverification", []string{"requires_reverification"}},
	}

	for _, idx := range statusIndexes {
		if _, err := p.db.NewCreateIndex().
			Model((*schema.UserVerificationStatus)(nil)).
			Index(idx.name).
			Column(idx.columns...).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	return nil
}

// GetService returns the verification service.
func (p *Plugin) GetService() *Service {
	return p.service
}

// GetConfig returns the plugin configuration.
func (p *Plugin) GetConfig() Config {
	return p.config
}

// IsEnabled returns whether the plugin is enabled.
func (p *Plugin) IsEnabled() bool {
	return p.config.Enabled
}

// GetHandler returns the HTTP handler.
func (p *Plugin) GetHandler() *Handler {
	return p.handler
}

// GetMiddleware returns the verification middleware.
func (p *Plugin) GetMiddleware() *Middleware {
	return p.middleware
}

// Middleware returns the LoadVerificationStatus middleware function
// Middleware is a convenience method for registering the middleware with Forge.
func (p *Plugin) Middleware() func(next func(forge.Context) error) func(forge.Context) error {
	if p.middleware == nil {
		return func(next func(forge.Context) error) func(forge.Context) error {
			return next
		}
	}

	return p.middleware.LoadVerificationStatus
}

// IDVerificationErrorResponse types for identity verification routes.
type IDVerificationErrorResponse struct {
	Error string `example:"Error message" json:"error"`
}

type IDVerificationSessionResponse struct {
	Session any `json:"session"`
}

type IDVerificationResponse struct {
	Verification any `json:"verification"`
}

type IDVerificationListResponse struct {
	Verifications []any `json:"verifications"`
}

type IDVerificationStatusResponse struct {
	Status any `json:"status"`
}

type IDVerificationWebhookResponse struct {
	Status string `example:"processed" json:"status"`
}
