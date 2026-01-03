package impersonation

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

const (
	PluginID      = "impersonation"
	PluginName    = "User Impersonation"
	PluginVersion = "1.0.0"
)

// Plugin implements the AuthSome plugin interface for impersonation
type Plugin struct {
	config      Config
	service     *impersonation.Service
	handler     *Handler
	middleware  *ImpersonationMiddleware
	stopCleanup chan struct{}
}

// NewPlugin creates a new impersonation plugin instance
func NewPlugin() *Plugin {
	return &Plugin{
		config:      DefaultConfig(),
		stopCleanup: make(chan struct{}),
	}
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
	return "Secure admin-to-user impersonation with audit logging, RBAC, and time limits for troubleshooting and support"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("impersonation plugin requires Auth instance")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	serviceRegistry := authInst.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}

	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Get required services from registry
	userSvcInterface := serviceRegistry.UserService()
	if userSvcInterface == nil {
		return fmt.Errorf("user service not found in registry")
	}
	userSvc, ok := userSvcInterface.(user.ServiceInterface)
	if !ok {
		return fmt.Errorf("invalid user service type")
	}

	sessionSvcInterface := serviceRegistry.SessionService()
	if sessionSvcInterface == nil {
		return fmt.Errorf("session service not found in registry")
	}
	sessionSvc, ok := sessionSvcInterface.(session.ServiceInterface)
	if !ok {
		return fmt.Errorf("invalid session service type")
	}

	auditSvc := serviceRegistry.AuditService()
	if auditSvc == nil {
		return fmt.Errorf("audit service not found in registry")
	}

	rbacSvc := serviceRegistry.RBACService()
	// RBAC is optional, can be nil

	// Initialize repository
	repo := repository.NewImpersonationRepository(db)

	// Convert plugin config to service config
	serviceConfig := impersonation.Config{
		DefaultDurationMinutes: p.config.DefaultDurationMinutes,
		MaxDurationMinutes:     p.config.MaxDurationMinutes,
		MinDurationMinutes:     p.config.MinDurationMinutes,
		RequireReason:          p.config.RequireReason,
		RequireTicket:          p.config.RequireTicket,
		MinReasonLength:        p.config.MinReasonLength,
		RequirePermission:      p.config.RequirePermission,
		ImpersonatePermission:  p.config.ImpersonatePermission,
		AuditAllActions:        p.config.AuditAllActions,
		AutoCleanupEnabled:     p.config.AutoCleanupEnabled,
		CleanupIntervalMinutes: int(p.config.CleanupInterval.Minutes()),
	}

	// Initialize service
	p.service = impersonation.NewService(
		repo,
		userSvc,
		sessionSvc,
		auditSvc,
		rbacSvc,
		serviceConfig,
	)

	// Initialize handler
	p.handler = NewHandler(p.service, p.config)

	// Initialize middleware
	p.middleware = NewMiddleware(p.service, p.config)

	// Start cleanup goroutine if enabled
	if p.config.AutoCleanupEnabled {
		go p.runCleanupTask()
	}


	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("handler not initialized; call Init first")
	}

	// Create API group for impersonation
	api := router.Group("/impersonation")

	// Impersonation management endpoints
	api.POST("/start", p.handler.StartImpersonation,
		forge.WithName("impersonation.start"),
		forge.WithSummary("Start impersonation"),
		forge.WithDescription("Begin impersonating another user. Requires admin privileges and creates an audit trail."),
		forge.WithResponseSchema(200, "Impersonation started", ImpersonationStartResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ImpersonationErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", ImpersonationErrorResponse{}),
		forge.WithTags("Impersonation"),
		forge.WithValidation(true),
	)

	api.POST("/end", p.handler.EndImpersonation,
		forge.WithName("impersonation.end"),
		forge.WithSummary("End impersonation"),
		forge.WithDescription("End the current impersonation session and restore the original user context"),
		forge.WithResponseSchema(200, "Impersonation ended", ImpersonationEndResponse{}),
		forge.WithResponseSchema(400, "Invalid request or no active impersonation", ImpersonationErrorResponse{}),
		forge.WithTags("Impersonation"),
	)

	api.GET("/:id", p.handler.GetImpersonation,
		forge.WithName("impersonation.get"),
		forge.WithSummary("Get impersonation details"),
		forge.WithDescription("Retrieve details of a specific impersonation session"),
		forge.WithResponseSchema(200, "Impersonation retrieved", ImpersonationSession{}),
		forge.WithResponseSchema(404, "Impersonation not found", ImpersonationErrorResponse{}),
		forge.WithTags("Impersonation"),
	)

	api.GET("/", p.handler.ListImpersonations,
		forge.WithName("impersonation.list"),
		forge.WithSummary("List impersonations"),
		forge.WithDescription("List all impersonation sessions (active and historical) with pagination support"),
		forge.WithResponseSchema(200, "Impersonations retrieved", ImpersonationListResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ImpersonationErrorResponse{}),
		forge.WithTags("Impersonation"),
	)

	api.POST("/verify", p.handler.VerifyImpersonation,
		forge.WithName("impersonation.verify"),
		forge.WithSummary("Verify impersonation"),
		forge.WithDescription("Verify if the current session is an active impersonation"),
		forge.WithResponseSchema(200, "Verification result", ImpersonationVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ImpersonationErrorResponse{}),
		forge.WithTags("Impersonation"),
	)

	// Audit endpoints
	api.GET("/audit", p.handler.ListAuditEvents,
		forge.WithName("impersonation.audit.list"),
		forge.WithSummary("List impersonation audit events"),
		forge.WithDescription("Retrieve audit logs of all impersonation activities for compliance and security monitoring"),
		forge.WithResponseSchema(200, "Audit events retrieved", ImpersonationAuditResponse{}),
		forge.WithResponseSchema(500, "Internal server error", ImpersonationErrorResponse{}),
		forge.WithTags("Impersonation", "Audit"),
	)

	return nil
}

// DTOs for impersonation routes (placeholder types)
type ImpersonationErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type ImpersonationStartResponse struct {
	SessionID      string `json:"session_id"`
	ImpersonatorID string `json:"impersonator_id"`
	TargetUserID   string `json:"target_user_id"`
	StartedAt      string `json:"started_at"`
}

type ImpersonationEndResponse struct {
	Status  string `json:"status" example:"success"`
	EndedAt string `json:"ended_at"`
}

type ImpersonationSession struct{}
type ImpersonationListResponse []interface{}
type ImpersonationVerifyResponse struct {
	IsImpersonating bool   `json:"is_impersonating"`
	ImpersonatorID  string `json:"impersonator_id,omitempty"`
	TargetUserID    string `json:"target_user_id,omitempty"`
}
type ImpersonationAuditResponse []interface{}

// RegisterHooks registers lifecycle hooks
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	// Register hooks for impersonation events
	// For example, you could hook into user login to prevent login during active impersonation
	// Or hook into session creation to mark impersonation sessions

	// TODO: Implement hooks when hook system is ready
	return nil
}

// RegisterServiceDecorators registers service decorators
func (p *Plugin) RegisterServiceDecorators(serviceRegistry *registry.ServiceRegistry) error {
	// Impersonation plugin doesn't need to decorate core services
	return nil
}

// Migrate runs database migrations for the plugin
func (p *Plugin) Migrate() error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}

	// Migrations will be handled by the main AuthSome migration system
	// The schema is already defined in schema/impersonation.go

	return nil
}

// Shutdown gracefully shuts down the plugin
func (p *Plugin) Shutdown(ctx context.Context) error {
	// Stop cleanup goroutine
	if p.config.AutoCleanupEnabled {
		close(p.stopCleanup)
	}

	return nil
}

// Health checks plugin health
func (p *Plugin) Health(ctx context.Context) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}

	// Check if we can query the database
	// We'll do a simple list query with a dummy app ID
	filter := &impersonation.ListSessionsFilter{
		AppID: xid.NilID(), // dummy app ID for health check
	}
	filter.Limit = 1

	_, err := p.service.List(ctx, filter)
	// It's ok if we get errors - as long as we can query the DB
	// (empty results or permission errors are fine for health check)
	return err
}

// GetService returns the impersonation service for programmatic access
func (p *Plugin) GetService() *impersonation.Service {
	return p.service
}

// GetMiddleware returns the impersonation middleware
func (p *Plugin) GetMiddleware() *ImpersonationMiddleware {
	return p.middleware
}

// runCleanupTask runs a periodic cleanup of expired impersonation sessions
func (p *Plugin) runCleanupTask() {
	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupExpiredSessions()
		case <-p.stopCleanup:

			return
		}
	}
}

// cleanupExpiredSessions expires old impersonation sessions
func (p *Plugin) cleanupExpiredSessions() {
	ctx := context.Background()
	count, err := p.service.ExpireSessions(ctx)
	if err != nil {
		return
	}

	if count > 0 {
	}
}
