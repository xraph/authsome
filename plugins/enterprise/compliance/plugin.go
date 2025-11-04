package compliance

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

// Plugin implements the AuthSome plugin interface for compliance
type Plugin struct {
	service      *Service
	config       *Config
	policyEngine *PolicyEngine
	handler      *Handler
}

// NewPlugin creates a new compliance plugin
func NewPlugin() *Plugin {
	return &Plugin{
		config: DefaultConfig(),
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "compliance"
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "Enterprise Compliance & Audit"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Comprehensive compliance management for SOC 2, HIPAA, PCI-DSS, GDPR, and ISO 27001"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Init initializes the plugin with AuthSome dependencies
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

	db := authInstance.GetDB()
	forgeApp := authInstance.GetForgeApp()
	configManager := forgeApp.Config()
	serviceRegistry := authInstance.GetServiceRegistry()

	// Load configuration from Forge config manager
	var config Config
	if err := configManager.Bind("auth.compliance", &config); err != nil {
		// Use defaults if binding fails
		config = *DefaultConfig()
	}
	config.Validate() // Ensure defaults are set
	p.config = &config

	if !p.config.Enabled {
		return nil
	}

	// Get services from registry
	auditSvc := serviceRegistry.AuditService()
	notificationSvc := serviceRegistry.NotificationService()
	hookRegistry := serviceRegistry.HookRegistry()

	// Get user service (interface type)
	userSvc := serviceRegistry.UserService()

	// Get organization service (interface type, may be nil if multi-tenancy plugin not loaded)
	orgSvc := serviceRegistry.OrganizationService()

	// Initialize repository
	repo := NewBunRepository(db)

	// Create adapters for services
	auditAdapter := NewAuditServiceAdapter(auditSvc)
	userAdapter := NewUserServiceAdapter(userSvc)
	orgAdapter := NewOrganizationServiceAdapter(orgSvc)
	emailAdapter := NewEmailServiceAdapter(notificationSvc)

	// Initialize service
	p.service = NewService(
		repo,
		p.config,
		auditAdapter,
		userAdapter,
		orgAdapter,
		emailAdapter,
	)

	// Initialize policy engine
	p.policyEngine = NewPolicyEngine(p.service)

	// Initialize handler
	p.handler = NewHandler(p.service, p.policyEngine)

	// Register hooks
	if err := p.registerHooks(hookRegistry); err != nil {
		return fmt.Errorf("failed to register hooks: %w", err)
	}

	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled || p.handler == nil {
		return nil
	}

	// Compliance profile management
	complianceGroup := router.Group(p.config.Dashboard.Path)
	{
		// Profiles
		complianceGroup.POST("/profiles", p.handler.CreateProfile)
		complianceGroup.POST("/profiles/from-template", p.handler.CreateProfileFromTemplate)
		complianceGroup.GET("/profiles/:id", p.handler.GetProfile)
		complianceGroup.GET("/organizations/:orgId/profile", p.handler.GetOrganizationProfile)
		complianceGroup.PUT("/profiles/:id", p.handler.UpdateProfile)
		complianceGroup.DELETE("/profiles/:id", p.handler.DeleteProfile)

		// Status and Dashboard
		complianceGroup.GET("/organizations/:orgId/status", p.handler.GetComplianceStatus)
		complianceGroup.GET("/organizations/:orgId/dashboard", p.handler.GetDashboard)

		// Checks
		complianceGroup.POST("/profiles/:profileId/checks", p.handler.RunCheck)
		complianceGroup.GET("/profiles/:profileId/checks", p.handler.ListChecks)
		complianceGroup.GET("/checks/:id", p.handler.GetCheck)

		// Violations
		complianceGroup.GET("/organizations/:orgId/violations", p.handler.ListViolations)
		complianceGroup.GET("/violations/:id", p.handler.GetViolation)
		complianceGroup.PUT("/violations/:id/resolve", p.handler.ResolveViolation)

		// Reports
		complianceGroup.POST("/organizations/:orgId/reports", p.handler.GenerateReport)
		complianceGroup.GET("/organizations/:orgId/reports", p.handler.ListReports)
		complianceGroup.GET("/reports/:id", p.handler.GetReport)
		complianceGroup.GET("/reports/:id/download", p.handler.DownloadReport)

		// Evidence
		complianceGroup.POST("/organizations/:orgId/evidence", p.handler.CreateEvidence)
		complianceGroup.GET("/organizations/:orgId/evidence", p.handler.ListEvidence)
		complianceGroup.GET("/evidence/:id", p.handler.GetEvidence)
		complianceGroup.DELETE("/evidence/:id", p.handler.DeleteEvidence)

		// Policies
		complianceGroup.POST("/organizations/:orgId/policies", p.handler.CreatePolicy)
		complianceGroup.GET("/organizations/:orgId/policies", p.handler.ListPolicies)
		complianceGroup.GET("/policies/:id", p.handler.GetPolicy)
		complianceGroup.PUT("/policies/:id", p.handler.UpdatePolicy)
		complianceGroup.DELETE("/policies/:id", p.handler.DeletePolicy)

		// Training
		complianceGroup.POST("/organizations/:orgId/training", p.handler.CreateTraining)
		complianceGroup.GET("/organizations/:orgId/training", p.handler.ListTraining)
		complianceGroup.GET("/users/:userId/training", p.handler.GetUserTraining)
		complianceGroup.PUT("/training/:id/complete", p.handler.CompleteTraining)

		// Templates
		complianceGroup.GET("/templates", p.handler.ListTemplates)
		complianceGroup.GET("/templates/:standard", p.handler.GetTemplate)
	}

	return nil
}

// registerHooks registers lifecycle hooks
func (p *Plugin) registerHooks(hookRegistry *hooks.HookRegistry) error {
	// Register user lifecycle hooks
	hookRegistry.RegisterAfterUserCreate(p.onUserCreated)
	hookRegistry.RegisterAfterUserUpdate(p.onUserUpdated)

	// Register auth hooks
	hookRegistry.RegisterBeforeSignIn(p.onBeforeSignIn)
	hookRegistry.RegisterAfterSignIn(p.onAfterSignIn)

	// Register session hooks
	hookRegistry.RegisterAfterSessionCreate(p.onSessionCreated)

	// Organization hooks (when multi-tenancy plugin is available)
	hookRegistry.RegisterAfterOrganizationCreate(p.onOrganizationCreated)
	hookRegistry.RegisterAfterMemberAdd(p.onMemberAdded)

	return nil
}

// Hook Handlers (matching AuthSome's actual hook signatures)

func (p *Plugin) onUserCreated(ctx context.Context, u *user.User) error {
	if p.service == nil {
		return nil
	}

	// Check if MFA is required for this user's organization
	// TODO: This will work properly when multi-tenancy plugin is available
	// For now, we'll check against a default organization

	// Check for required training
	// Create training records if compliance profile exists
	// This is a best-effort check - don't block user creation

	return nil
}

func (p *Plugin) onUserUpdated(ctx context.Context, u *user.User) error {
	if p.service == nil {
		return nil
	}

	// Check if any compliance-related fields changed
	// Log audit event for tracking

	return nil
}

func (p *Plugin) onBeforeSignIn(ctx context.Context, req *auth.SignInRequest) error {
	if p.service == nil || p.policyEngine == nil {
		return nil
	}

	// Enforce password policy (if password is being validated)
	// Check if MFA is required
	// This hook can block sign-in if policies aren't met

	return nil
}

func (p *Plugin) onAfterSignIn(ctx context.Context, response *auth.AuthResponse) error {
	if p.service == nil {
		return nil
	}

	// Log successful sign-in for compliance audit trail
	// Check session policies

	return nil
}

func (p *Plugin) onSessionCreated(ctx context.Context, sess *session.Session) error {
	if p.service == nil || p.policyEngine == nil {
		return nil
	}

	// Enforce session policies (timeout, IP validation, etc.)
	// Log session creation for audit trail

	return nil
}

func (p *Plugin) onOrganizationCreated(ctx context.Context, org interface{}) error {
	if p.service == nil {
		return nil
	}

	// Extract organization ID from interface
	// Create default compliance profile if configured
	if p.config.DefaultStandard != "" {
		// TODO: Create default profile when multi-tenancy is available
	}

	return nil
}

func (p *Plugin) onMemberAdded(ctx context.Context, member interface{}) error {
	if p.service == nil {
		return nil
	}

	// When a user is added to an organization
	// Check if they need compliance training
	// Create required training records

	return nil
}

// RegisterHooks registers plugin hooks with the hook registry (implements Plugin interface)
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	return p.registerHooks(hookRegistry)
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Compliance plugin doesn't decorate core services
	// It provides its own service that's accessed via the plugin
	return nil
}

// Migrate performs database migrations
func (p *Plugin) Migrate() error {
	// Database migrations will be handled by migration system
	// The schema is defined in schema.go and will be registered in migrations/bun/
	return nil
}

// Service returns the compliance service for direct access (optional public method)
func (p *Plugin) Service() *Service {
	return p.service
}

// PolicyEngine returns the policy engine for direct access (optional public method)
func (p *Plugin) PolicyEngine() *PolicyEngine {
	return p.policyEngine
}
