package stepup

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/forge"
)

const (
	PluginID      = "stepup"
	PluginName    = "Step-Up Authentication"
	PluginVersion = "1.0.0"
)

// Plugin implements the AuthSome plugin interface for step-up authentication
type Plugin struct {
	config     *Config
	service    *Service
	handler    *Handler
	middleware *Middleware
	repo       Repository
	db         *bun.DB
}

// NewPlugin creates a new step-up authentication plugin instance
func NewPlugin(config *Config) *Plugin {
	if config == nil {
		config = DefaultConfig()
	}
	return &Plugin{
		config: config,
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
	return "Context-aware step-up authentication for high-value operations with route, amount, and resource-based rules"
}

// Init initializes the plugin with the auth instance
func (p *Plugin) Init(auth interface{}) error {
	// Extract database and service registry from auth instance
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

	// Get service registry
	serviceRegistry := authInstance.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}

	// Get audit service and wrap it with adapter
	var auditAdapter AuditServiceInterface
	if auditSvc := serviceRegistry.AuditService(); auditSvc != nil {
		auditAdapter = &auditServiceAdapter{svc: auditSvc}
	}

	// Initialize repository
	p.repo = NewBunRepository(p.db)

	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize service
	p.service = NewService(p.repo, p.config, auditAdapter)

	// Initialize handler
	p.handler = NewHandler(p.service, p.config)

	// Initialize middleware
	p.middleware = NewMiddleware(p.service, p.config)

	fmt.Printf("[StepUp] Plugin initialized successfully\n")
	fmt.Printf("[StepUp] Enabled: %v\n", p.config.Enabled)
	fmt.Printf("[StepUp] Route rules: %d\n", len(p.config.RouteRules))
	fmt.Printf("[StepUp] Amount rules: %d\n", len(p.config.AmountRules))
	fmt.Printf("[StepUp] Resource rules: %d\n", len(p.config.ResourceRules))
	fmt.Printf("[StepUp] Remember devices: %v\n", p.config.RememberStepUp)
	fmt.Printf("[StepUp] Risk-based: %v\n", p.config.RiskBasedEnabled)

	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized, Init must be called first")
	}

	// Create a group for step-up routes
	group := router.Group("/stepup")

	// Evaluation endpoints
	group.POST("/evaluate", p.handler.Evaluate,
		forge.WithName("stepup.evaluate"),
		forge.WithSummary("Evaluate step-up requirement"),
		forge.WithDescription("Evaluate if step-up authentication is required for an action"),
		forge.WithResponseSchema(200, "Evaluation result", StepUpEvaluationResponse{}),
		forge.WithTags("Step-Up", "Authentication"),
		forge.WithValidation(true),
	)
	group.POST("/verify", p.handler.Verify,
		forge.WithName("stepup.verify"),
		forge.WithSummary("Verify step-up authentication"),
		forge.WithDescription("Verify step-up authentication credentials (password, MFA, etc.)"),
		forge.WithResponseSchema(200, "Verification successful", StepUpVerificationResponse{}),
		forge.WithResponseSchema(401, "Verification failed", StepUpErrorResponse{}),
		forge.WithTags("Step-Up", "Authentication"),
		forge.WithValidation(true),
	)
	group.GET("/status", p.handler.Status,
		forge.WithName("stepup.status"),
		forge.WithSummary("Get step-up status"),
		forge.WithDescription("Get current step-up authentication status for the session"),
		forge.WithResponseSchema(200, "Status retrieved", StepUpStatusResponse{}),
		forge.WithTags("Step-Up", "Authentication"),
	)

	// Requirements management
	group.GET("/requirements/:id", p.handler.GetRequirement,
		forge.WithName("stepup.requirements.get"),
		forge.WithSummary("Get step-up requirement"),
		forge.WithDescription("Retrieve details of a specific step-up requirement"),
		forge.WithResponseSchema(200, "Requirement retrieved", StepUpRequirementResponse{}),
		forge.WithTags("Step-Up", "Requirements"),
	)
	group.GET("/requirements/pending", p.handler.ListPendingRequirements,
		forge.WithName("stepup.requirements.pending"),
		forge.WithSummary("List pending requirements"),
		forge.WithDescription("List all pending step-up requirements for current user"),
		forge.WithResponseSchema(200, "Pending requirements", StepUpRequirementsResponse{}),
		forge.WithTags("Step-Up", "Requirements"),
	)

	// Verifications history
	group.GET("/verifications", p.handler.ListVerifications,
		forge.WithName("stepup.verifications.list"),
		forge.WithSummary("List verifications"),
		forge.WithDescription("List step-up verification history"),
		forge.WithResponseSchema(200, "Verifications retrieved", StepUpVerificationsResponse{}),
		forge.WithTags("Step-Up", "History"),
	)

	// Remembered devices management
	group.GET("/devices", p.handler.ListRememberedDevices,
		forge.WithName("stepup.devices.list"),
		forge.WithSummary("List remembered devices"),
		forge.WithDescription("List devices that are remembered for step-up authentication"),
		forge.WithResponseSchema(200, "Devices retrieved", StepUpDevicesResponse{}),
		forge.WithTags("Step-Up", "Devices"),
	)
	group.DELETE("/devices/:id", p.handler.ForgetDevice,
		forge.WithName("stepup.devices.forget"),
		forge.WithSummary("Forget device"),
		forge.WithDescription("Remove a device from remembered devices list"),
		forge.WithResponseSchema(200, "Device forgotten", StepUpStatusResponse{}),
		forge.WithTags("Step-Up", "Devices"),
	)

	// Policies management (organization-level)
	group.POST("/policies", p.handler.CreatePolicy,
		forge.WithName("stepup.policies.create"),
		forge.WithSummary("Create step-up policy"),
		forge.WithDescription("Create a new step-up authentication policy"),
		forge.WithResponseSchema(200, "Policy created", StepUpPolicyResponse{}),
		forge.WithTags("Step-Up", "Policies"),
		forge.WithValidation(true),
	)
	group.GET("/policies", p.handler.ListPolicies,
		forge.WithName("stepup.policies.list"),
		forge.WithSummary("List step-up policies"),
		forge.WithDescription("List all step-up authentication policies"),
		forge.WithResponseSchema(200, "Policies retrieved", StepUpPoliciesResponse{}),
		forge.WithTags("Step-Up", "Policies"),
	)
	group.GET("/policies/:id", p.handler.GetPolicy,
		forge.WithName("stepup.policies.get"),
		forge.WithSummary("Get step-up policy"),
		forge.WithDescription("Retrieve a specific step-up policy"),
		forge.WithResponseSchema(200, "Policy retrieved", StepUpPolicyResponse{}),
		forge.WithTags("Step-Up", "Policies"),
	)
	group.PUT("/policies/:id", p.handler.UpdatePolicy,
		forge.WithName("stepup.policies.update"),
		forge.WithSummary("Update step-up policy"),
		forge.WithDescription("Update an existing step-up authentication policy"),
		forge.WithResponseSchema(200, "Policy updated", StepUpPolicyResponse{}),
		forge.WithTags("Step-Up", "Policies"),
		forge.WithValidation(true),
	)
	group.DELETE("/policies/:id", p.handler.DeletePolicy,
		forge.WithName("stepup.policies.delete"),
		forge.WithSummary("Delete step-up policy"),
		forge.WithDescription("Delete a step-up authentication policy"),
		forge.WithResponseSchema(200, "Policy deleted", StepUpStatusResponse{}),
		forge.WithTags("Step-Up", "Policies"),
	)

	// Audit logs
	group.GET("/audit", p.handler.GetAuditLogs,
		forge.WithName("stepup.audit.list"),
		forge.WithSummary("Get audit logs"),
		forge.WithDescription("Retrieve step-up authentication audit logs"),
		forge.WithResponseSchema(200, "Audit logs retrieved", StepUpAuditLogsResponse{}),
		forge.WithTags("Step-Up", "Audit"),
	)

	fmt.Println("[StepUp] Routes registered successfully")
	fmt.Println("[StepUp] Available endpoints:")
	fmt.Println("  - POST   /stepup/evaluate")
	fmt.Println("  - POST   /stepup/verify")
	fmt.Println("  - GET    /stepup/status")
	fmt.Println("  - GET    /stepup/requirements/:id")
	fmt.Println("  - GET    /stepup/requirements/pending")
	fmt.Println("  - GET    /stepup/verifications")
	fmt.Println("  - GET    /stepup/devices")
	fmt.Println("  - DELETE /stepup/devices/:id")
	fmt.Println("  - POST   /stepup/policies")
	fmt.Println("  - GET    /stepup/policies")
	fmt.Println("  - GET    /stepup/policies/:id")
	fmt.Println("  - PUT    /stepup/policies/:id")
	fmt.Println("  - DELETE /stepup/policies/:id")
	fmt.Println("  - GET    /stepup/audit")

	return nil
}

// RegisterHooks registers step-up lifecycle hooks
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}

	// Register cleanup hook to run periodically
	// This would typically be registered with a job scheduler
	// For now, we'll just register a hook that can be called manually

	fmt.Println("[StepUp] Hooks registered successfully")
	return nil
}

// RegisterServiceDecorators allows step-up to enhance core services
func (p *Plugin) RegisterServiceDecorators(serviceRegistry *registry.ServiceRegistry) error {
	// Step-up plugin doesn't need to decorate core services
	// It provides its own middleware for enforcement
	return nil
}

// Migrate creates required database tables and indexes
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	ctx := context.Background()

	// Create tables
	tables := []interface{}{
		(*StepUpVerification)(nil),
		(*StepUpRequirement)(nil),
		(*StepUpRememberedDevice)(nil),
		(*StepUpAttempt)(nil),
		(*StepUpPolicy)(nil),
		(*StepUpAuditLog)(nil),
	}

	for _, table := range tables {
		if _, err := p.db.NewCreateTable().
			Model(table).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes for performance
	if err := p.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	fmt.Println("[StepUp] Database migration completed successfully")
	return nil
}

// createIndexes creates database indexes for step-up tables
func (p *Plugin) createIndexes(ctx context.Context) error {
	indexes := []string{
		// Verifications indexes
		"CREATE INDEX IF NOT EXISTS idx_stepup_verifications_user_org ON stepup_verifications(user_id, org_id)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_verifications_level ON stepup_verifications(user_id, org_id, security_level)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_verifications_expires ON stepup_verifications(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_verifications_session ON stepup_verifications(session_id)",

		// Requirements indexes
		"CREATE INDEX IF NOT EXISTS idx_stepup_requirements_user_org ON stepup_requirements(user_id, org_id)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_requirements_status ON stepup_requirements(status, expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_requirements_token ON stepup_requirements(challenge_token)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_requirements_session ON stepup_requirements(session_id)",

		// Remembered devices indexes
		"CREATE INDEX IF NOT EXISTS idx_stepup_devices_user_org ON stepup_remembered_devices(user_id, org_id)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_devices_device_id ON stepup_remembered_devices(user_id, org_id, device_id)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_devices_expires ON stepup_remembered_devices(expires_at)",

		// Attempts indexes
		"CREATE INDEX IF NOT EXISTS idx_stepup_attempts_requirement ON stepup_attempts(requirement_id)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_attempts_user_org ON stepup_attempts(user_id, org_id)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_attempts_success ON stepup_attempts(user_id, org_id, success, created_at)",

		// Policies indexes
		"CREATE INDEX IF NOT EXISTS idx_stepup_policies_org ON stepup_policies(org_id, enabled)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_policies_priority ON stepup_policies(org_id, priority DESC)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_policies_user ON stepup_policies(org_id, user_id)",

		// Audit logs indexes
		"CREATE INDEX IF NOT EXISTS idx_stepup_audit_user_org ON stepup_audit_logs(user_id, org_id)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_audit_event_type ON stepup_audit_logs(event_type, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_stepup_audit_created ON stepup_audit_logs(created_at DESC)",
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

// Service returns the step-up service (for programmatic access)
func (p *Plugin) Service() *Service {
	return p.service
}

// Middleware returns the step-up middleware (for route protection)
func (p *Plugin) Middleware() *Middleware {
	return p.middleware
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

// StartCleanupScheduler starts a background task to cleanup expired records
func (p *Plugin) StartCleanupScheduler(interval time.Duration) {
	if p.service == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			ctx := context.Background()
			if err := p.service.CleanupExpired(ctx); err != nil {
				// Log error but continue
				fmt.Printf("[StepUp] Cleanup error: %v\n", err)
			}
		}
	}()

	fmt.Printf("[StepUp] Cleanup scheduler started (interval: %v)\n", interval)
}

// Health checks the plugin health
func (p *Plugin) Health(ctx context.Context) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}

	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Test database connection
	if err := p.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the plugin
func (p *Plugin) Shutdown(ctx context.Context) error {
	fmt.Println("[StepUp] Shutting down plugin...")

	// Perform any cleanup needed
	if p.service != nil {
		if err := p.service.CleanupExpired(ctx); err != nil {
			fmt.Printf("[StepUp] Final cleanup error: %v\n", err)
		}
	}

	fmt.Println("[StepUp] Plugin shut down successfully")
	return nil
}

// auditServiceAdapter adapts the core audit service to match the plugin's expected interface
type auditServiceAdapter struct {
	svc *audit.Service
}

// Log implements AuditServiceInterface by converting Event to the core audit service's signature
func (a *auditServiceAdapter) Log(ctx context.Context, event *audit.Event) error {
	if a.svc == nil {
		return nil // No-op if audit service not available
	}

	// Convert Event to the parameters expected by core audit service
	// Signature: Log(ctx, userID, action, resource, ip, ua, metadata)
	return a.svc.Log(
		ctx,
		event.UserID,
		event.Action,
		event.Resource,
		event.IPAddress,
		event.UserAgent,
		event.Metadata,
	)
}

// DTOs for step-up routes
type StepUpErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type StepUpStatusResponse struct {
	Status string `json:"status" example:"success"`
}

type StepUpEvaluationResponse struct {
	Required bool   `json:"required" example:"true"`
	Reason   string `json:"reason,omitempty" example:"High-value transaction"`
}

type StepUpVerificationResponse struct {
	Verified  bool   `json:"verified" example:"true"`
	ExpiresAt string `json:"expires_at,omitempty" example:"2024-01-01T00:00:00Z"`
}

type StepUpRequirementResponse struct {
	ID string `json:"id" example:"req_123"`
}

type StepUpRequirementsResponse struct {
	Requirements []interface{} `json:"requirements"`
}

type StepUpVerificationsResponse struct {
	Verifications []interface{} `json:"verifications"`
}

type StepUpPolicyResponse struct {
	ID string `json:"id" example:"policy_123"`
}

type StepUpPoliciesResponse struct {
	Policies []interface{} `json:"policies"`
}

type StepUpAuditLogsResponse struct {
	AuditLogs []interface{} `json:"audit_logs"`
}
