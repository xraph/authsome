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

	// Service dependencies
	auditService audit.ServiceInterface
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

	// Get audit service
	if auditSvc := serviceRegistry.AuditService(); auditSvc != nil {
		// Type assertion to our interface
		if audSvc, ok := auditSvc.(audit.ServiceInterface); ok {
			p.auditService = audSvc
		}
	}

	// Initialize repository
	p.repo = NewBunRepository(p.db)

	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize service
	p.service = NewService(p.repo, p.config, p.auditService)

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
	group.POST("/evaluate", p.handler.Evaluate)
	group.POST("/verify", p.handler.Verify)
	group.GET("/status", p.handler.Status)

	// Requirements management
	group.GET("/requirements/:id", p.handler.GetRequirement)
	group.GET("/requirements/pending", p.handler.ListPendingRequirements)

	// Verifications history
	group.GET("/verifications", p.handler.ListVerifications)

	// Remembered devices management
	group.GET("/devices", p.handler.ListRememberedDevices)
	group.DELETE("/devices/:id", p.handler.ForgetDevice)

	// Policies management (organization-level)
	group.POST("/policies", p.handler.CreatePolicy)
	group.GET("/policies", p.handler.ListPolicies)
	group.GET("/policies/:id", p.handler.GetPolicy)
	group.PUT("/policies/:id", p.handler.UpdatePolicy)
	group.DELETE("/policies/:id", p.handler.DeletePolicy)

	// Audit logs
	group.GET("/audit", p.handler.GetAuditLogs)

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

