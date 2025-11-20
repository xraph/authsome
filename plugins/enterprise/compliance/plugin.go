package compliance

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/responses"
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
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("compliance plugin requires Auth instance")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available")
	}

	configManager := forgeApp.Config()
	serviceRegistry := authInst.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}

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

	// Get app service (interface type, may be nil if multi-app plugin not loaded)
	appSvc := serviceRegistry.AppService()

	// Initialize repository
	repo := NewBunRepository(db)

	// Create adapters for services
	auditAdapter := NewAuditServiceAdapter(auditSvc)
	userAdapter := NewUserServiceAdapter(userSvc)
	appAdapter := NewAppServiceAdapter(appSvc)
	emailAdapter := NewEmailServiceAdapter(notificationSvc)

	// Initialize service
	p.service = NewService(
		repo,
		p.config,
		auditAdapter,
		userAdapter,
		appAdapter,
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
		complianceGroup.POST("/profiles", p.handler.CreateProfile,
			forge.WithName("compliance.profiles.create"),
			forge.WithSummary("Create compliance profile"),
			forge.WithDescription("Create a new compliance profile for an organization"),
			forge.WithRequestSchema(CreateProfileRequest{}),
			forge.WithResponseSchema(200, "Profile created", ComplianceProfileResponse{}),
			forge.WithTags("Compliance"),
			forge.WithValidation(true),
		)
		complianceGroup.POST("/profiles/from-template", p.handler.CreateProfileFromTemplate,
			forge.WithName("compliance.profiles.create.template"),
			forge.WithSummary("Create profile from template"),
			forge.WithDescription("Create a compliance profile from a predefined template (GDPR, HIPAA, SOC2, etc.)"),
			forge.WithRequestSchema(CreateProfileFromTemplateRequest{}),
			forge.WithResponseSchema(200, "Profile created", ComplianceProfileResponse{}),
			forge.WithTags("Compliance"),
			forge.WithValidation(true),
		)
		complianceGroup.GET("/profiles/:id", p.handler.GetProfile,
			forge.WithName("compliance.profiles.get"),
			forge.WithSummary("Get compliance profile"),
			forge.WithDescription("Retrieve a specific compliance profile by ID"),
			forge.WithResponseSchema(200, "Profile retrieved", ComplianceProfileResponse{}),
			forge.WithTags("Compliance"),
		)
		complianceGroup.GET("/apps/:appId/profile", p.handler.GetAppProfile,
			forge.WithName("compliance.profiles.org"),
			forge.WithSummary("Get organization profile"),
			forge.WithDescription("Get the compliance profile for a specific organization"),
			forge.WithResponseSchema(200, "Profile retrieved", ComplianceProfileResponse{}),
			forge.WithTags("Compliance", "Organizations"),
		)
		complianceGroup.PUT("/profiles/:id", p.handler.UpdateProfile,
			forge.WithName("compliance.profiles.update"),
			forge.WithSummary("Update compliance profile"),
			forge.WithDescription("Update an existing compliance profile"),
			forge.WithRequestSchema(UpdateProfileRequest{}),
			forge.WithResponseSchema(200, "Profile updated", ComplianceProfileResponse{}),
			forge.WithTags("Compliance"),
			forge.WithValidation(true),
		)
		complianceGroup.DELETE("/profiles/:id", p.handler.DeleteProfile,
			forge.WithName("compliance.profiles.delete"),
			forge.WithSummary("Delete compliance profile"),
			forge.WithDescription("Delete a compliance profile"),
			forge.WithResponseSchema(200, "Profile deleted", ComplianceStatusResponse{}),
			forge.WithTags("Compliance"),
		)

		// Status and Dashboard
		complianceGroup.GET("/apps/:appId/status", p.handler.GetComplianceStatus,
			forge.WithName("compliance.status"),
			forge.WithSummary("Get compliance status"),
			forge.WithDescription("Get overall compliance status for an organization"),
			forge.WithResponseSchema(200, "Status retrieved", ComplianceStatusDetailsResponse{}),
			forge.WithTags("Compliance", "Organizations"),
		)
		complianceGroup.GET("/apps/:appId/dashboard", p.handler.GetDashboard,
			forge.WithName("compliance.dashboard"),
			forge.WithSummary("Get compliance dashboard"),
			forge.WithDescription("Get compliance dashboard metrics and overview"),
			forge.WithResponseSchema(200, "Dashboard retrieved", ComplianceDashboardResponse{}),
			forge.WithTags("Compliance", "Organizations"),
		)

		// Checks
		complianceGroup.POST("/profiles/:profileId/checks", p.handler.RunCheck,
			forge.WithName("compliance.checks.run"),
			forge.WithSummary("Run compliance check"),
			forge.WithDescription("Execute a compliance check for a profile"),
			forge.WithRequestSchema(RunCheckRequest{}),
			forge.WithResponseSchema(200, "Check started", ComplianceCheckResponse{}),
			forge.WithTags("Compliance", "Checks"),
			forge.WithValidation(true),
		)
		complianceGroup.GET("/profiles/:profileId/checks", p.handler.ListChecks,
			forge.WithName("compliance.checks.list"),
			forge.WithSummary("List compliance checks"),
			forge.WithDescription("List all compliance checks for a profile"),
			forge.WithResponseSchema(200, "Checks retrieved", ComplianceChecksResponse{}),
			forge.WithTags("Compliance", "Checks"),
		)
		complianceGroup.GET("/checks/:id", p.handler.GetCheck,
			forge.WithName("compliance.checks.get"),
			forge.WithSummary("Get compliance check"),
			forge.WithDescription("Retrieve details of a specific compliance check"),
			forge.WithResponseSchema(200, "Check retrieved", ComplianceCheckResponse{}),
			forge.WithTags("Compliance", "Checks"),
		)

		// Violations
		complianceGroup.GET("/apps/:appId/violations", p.handler.ListViolations,
			forge.WithName("compliance.violations.list"),
			forge.WithSummary("List compliance violations"),
			forge.WithDescription("List all compliance violations for an organization"),
			forge.WithResponseSchema(200, "Violations retrieved", ComplianceViolationsResponse{}),
			forge.WithTags("Compliance", "Violations"),
		)
		complianceGroup.GET("/violations/:id", p.handler.GetViolation,
			forge.WithName("compliance.violations.get"),
			forge.WithSummary("Get compliance violation"),
			forge.WithDescription("Retrieve details of a specific compliance violation"),
			forge.WithResponseSchema(200, "Violation retrieved", ComplianceViolationResponse{}),
			forge.WithTags("Compliance", "Violations"),
		)
		complianceGroup.PUT("/violations/:id/resolve", p.handler.ResolveViolation,
			forge.WithName("compliance.violations.resolve"),
			forge.WithSummary("Resolve compliance violation"),
			forge.WithDescription("Mark a compliance violation as resolved"),
			forge.WithRequestSchema(ResolveViolationRequest{}),
			forge.WithResponseSchema(200, "Violation resolved", ComplianceStatusResponse{}),
			forge.WithTags("Compliance", "Violations"),
			forge.WithValidation(true),
		)

		// Reports
		complianceGroup.POST("/apps/:appId/reports", p.handler.GenerateReport,
			forge.WithName("compliance.reports.generate"),
			forge.WithSummary("Generate compliance report"),
			forge.WithDescription("Generate a compliance report for an organization"),
			forge.WithRequestSchema(GenerateReportRequest{}),
			forge.WithResponseSchema(200, "Report generated", ComplianceReportResponse{}),
			forge.WithTags("Compliance", "Reports"),
			forge.WithValidation(true),
		)
		complianceGroup.GET("/apps/:appId/reports", p.handler.ListReports,
			forge.WithName("compliance.reports.list"),
			forge.WithSummary("List compliance reports"),
			forge.WithDescription("List all compliance reports for an organization"),
			forge.WithResponseSchema(200, "Reports retrieved", ComplianceReportsResponse{}),
			forge.WithTags("Compliance", "Reports"),
		)
		complianceGroup.GET("/reports/:id", p.handler.GetReport,
			forge.WithName("compliance.reports.get"),
			forge.WithSummary("Get compliance report"),
			forge.WithDescription("Retrieve a specific compliance report"),
			forge.WithResponseSchema(200, "Report retrieved", ComplianceReportResponse{}),
			forge.WithTags("Compliance", "Reports"),
		)
		complianceGroup.GET("/reports/:id/download", p.handler.DownloadReport,
			forge.WithName("compliance.reports.download"),
			forge.WithSummary("Download compliance report"),
			forge.WithDescription("Download a compliance report file (PDF, CSV, JSON)"),
			forge.WithResponseSchema(200, "Report file", ComplianceReportFileResponse{}),
			forge.WithTags("Compliance", "Reports"),
		)

		// Evidence
		complianceGroup.POST("/apps/:appId/evidence", p.handler.CreateEvidence,
			forge.WithName("compliance.evidence.create"),
			forge.WithSummary("Create evidence record"),
			forge.WithDescription("Create a new compliance evidence record"),
			forge.WithRequestSchema(CreateEvidenceRequest{}),
			forge.WithResponseSchema(200, "Evidence created", ComplianceEvidenceResponse{}),
			forge.WithTags("Compliance", "Evidence"),
			forge.WithValidation(true),
		)
		complianceGroup.GET("/apps/:appId/evidence", p.handler.ListEvidence,
			forge.WithName("compliance.evidence.list"),
			forge.WithSummary("List evidence records"),
			forge.WithDescription("List all compliance evidence records for an organization"),
			forge.WithResponseSchema(200, "Evidence retrieved", ComplianceEvidencesResponse{}),
			forge.WithTags("Compliance", "Evidence"),
		)
		complianceGroup.GET("/evidence/:id", p.handler.GetEvidence,
			forge.WithName("compliance.evidence.get"),
			forge.WithSummary("Get evidence record"),
			forge.WithDescription("Retrieve a specific compliance evidence record"),
			forge.WithResponseSchema(200, "Evidence retrieved", ComplianceEvidenceResponse{}),
			forge.WithTags("Compliance", "Evidence"),
		)
		complianceGroup.DELETE("/evidence/:id", p.handler.DeleteEvidence,
			forge.WithName("compliance.evidence.delete"),
			forge.WithSummary("Delete evidence record"),
			forge.WithDescription("Delete a compliance evidence record"),
			forge.WithResponseSchema(200, "Evidence deleted", ComplianceStatusResponse{}),
			forge.WithTags("Compliance", "Evidence"),
		)

		// Policies
		complianceGroup.POST("/apps/:appId/policies", p.handler.CreatePolicy,
			forge.WithName("compliance.policies.create"),
			forge.WithSummary("Create compliance policy"),
			forge.WithDescription("Create a new compliance policy document"),
			forge.WithRequestSchema(CreatePolicyRequest{}),
			forge.WithResponseSchema(200, "Policy created", CompliancePolicyResponse{}),
			forge.WithTags("Compliance", "Policies"),
			forge.WithValidation(true),
		)
		complianceGroup.GET("/apps/:appId/policies", p.handler.ListPolicies,
			forge.WithName("compliance.policies.list"),
			forge.WithSummary("List compliance policies"),
			forge.WithDescription("List all compliance policies for an organization"),
			forge.WithResponseSchema(200, "Policies retrieved", CompliancePoliciesResponse{}),
			forge.WithTags("Compliance", "Policies"),
		)
		complianceGroup.GET("/policies/:id", p.handler.GetPolicy,
			forge.WithName("compliance.policies.get"),
			forge.WithSummary("Get compliance policy"),
			forge.WithDescription("Retrieve a specific compliance policy"),
			forge.WithResponseSchema(200, "Policy retrieved", CompliancePolicyResponse{}),
			forge.WithTags("Compliance", "Policies"),
		)
		complianceGroup.PUT("/policies/:id", p.handler.UpdatePolicy,
			forge.WithName("compliance.policies.update"),
			forge.WithSummary("Update compliance policy"),
			forge.WithDescription("Update an existing compliance policy"),
			forge.WithRequestSchema(UpdatePolicyRequest{}),
			forge.WithResponseSchema(200, "Policy updated", CompliancePolicyResponse{}),
			forge.WithTags("Compliance", "Policies"),
			forge.WithValidation(true),
		)
		complianceGroup.DELETE("/policies/:id", p.handler.DeletePolicy,
			forge.WithName("compliance.policies.delete"),
			forge.WithSummary("Delete compliance policy"),
			forge.WithDescription("Delete a compliance policy"),
			forge.WithResponseSchema(200, "Policy deleted", ComplianceStatusResponse{}),
			forge.WithTags("Compliance", "Policies"),
		)

		// Training
		complianceGroup.POST("/apps/:appId/training", p.handler.CreateTraining,
			forge.WithName("compliance.training.create"),
			forge.WithSummary("Create training module"),
			forge.WithDescription("Create a compliance training module"),
			forge.WithRequestSchema(CreateTrainingRequest{}),
			forge.WithResponseSchema(200, "Training created", ComplianceTrainingResponse{}),
			forge.WithTags("Compliance", "Training"),
			forge.WithValidation(true),
		)
		complianceGroup.GET("/apps/:appId/training", p.handler.ListTraining,
			forge.WithName("compliance.training.list"),
			forge.WithSummary("List training modules"),
			forge.WithDescription("List all compliance training modules"),
			forge.WithResponseSchema(200, "Training retrieved", ComplianceTrainingsResponse{}),
			forge.WithTags("Compliance", "Training"),
		)
		complianceGroup.GET("/users/:userId/training", p.handler.GetUserTraining,
			forge.WithName("compliance.training.user"),
			forge.WithSummary("Get user training status"),
			forge.WithDescription("Get compliance training status for a user"),
			forge.WithResponseSchema(200, "Training status retrieved", ComplianceUserTrainingResponse{}),
			forge.WithTags("Compliance", "Training"),
		)
		complianceGroup.PUT("/training/:id/complete", p.handler.CompleteTraining,
			forge.WithName("compliance.training.complete"),
			forge.WithRequestSchema(CompleteTrainingRequest{}),
			forge.WithSummary("Complete training"),
			forge.WithDescription("Mark a training module as completed"),
			forge.WithResponseSchema(200, "Training completed", ComplianceStatusResponse{}),
			forge.WithTags("Compliance", "Training"),
			forge.WithValidation(true),
		)

		// Templates
		complianceGroup.GET("/templates", p.handler.ListTemplates,
			forge.WithName("compliance.templates.list"),
			forge.WithSummary("List compliance templates"),
			forge.WithDescription("List available compliance templates (GDPR, HIPAA, SOC2, PCI-DSS, etc.)"),
			forge.WithResponseSchema(200, "Templates retrieved", ComplianceTemplatesResponse{}),
			forge.WithTags("Compliance", "Templates"),
		)
		complianceGroup.GET("/templates/:standard", p.handler.GetTemplate,
			forge.WithName("compliance.templates.get"),
			forge.WithSummary("Get compliance template"),
			forge.WithDescription("Retrieve a specific compliance template by standard name"),
			forge.WithResponseSchema(200, "Template retrieved", ComplianceTemplateResponse{}),
			forge.WithTags("Compliance", "Templates"),
		)
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

func (p *Plugin) onAfterSignIn(ctx context.Context, response *responses.AuthResponse) error {
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

// DTOs for compliance routes
type ComplianceStatusResponse struct {
	Status string `json:"status" example:"success"`
}

type ComplianceProfileResponse struct {
	ID string `json:"id" example:"profile_123"`
}

type ComplianceStatusDetailsResponse struct {
	Status string `json:"status" example:"compliant"`
}

type ComplianceDashboardResponse struct {
	Metrics interface{} `json:"metrics"`
}

type ComplianceCheckResponse struct {
	ID string `json:"id" example:"check_123"`
}

type ComplianceChecksResponse struct {
	Checks []interface{} `json:"checks"`
}

type ComplianceViolationsResponse struct {
	Violations []interface{} `json:"violations"`
}

type ComplianceViolationResponse struct {
	ID string `json:"id" example:"violation_123"`
}

type ComplianceReportResponse struct {
	ID string `json:"id" example:"report_123"`
}

type ComplianceReportsResponse struct {
	Reports []interface{} `json:"reports"`
}

type ComplianceReportFileResponse struct {
	ContentType string `json:"content_type" example:"application/pdf"`
	Data        []byte `json:"data"`
}

type ComplianceEvidenceResponse struct {
	ID string `json:"id" example:"evidence_123"`
}

type ComplianceEvidencesResponse struct {
	Evidence []interface{} `json:"evidence"`
}

type CompliancePolicyResponse struct {
	ID string `json:"id" example:"policy_123"`
}

type CompliancePoliciesResponse struct {
	Policies []interface{} `json:"policies"`
}

type ComplianceTrainingResponse struct {
	ID string `json:"id" example:"training_123"`
}

type ComplianceTrainingsResponse struct {
	Training []interface{} `json:"training"`
}

type ComplianceUserTrainingResponse struct {
	UserID string `json:"user_id" example:"user_123"`
}

type ComplianceTemplatesResponse struct {
	Templates []interface{} `json:"templates"`
}

type ComplianceTemplateResponse struct {
	Standard string `json:"standard" example:"GDPR"`
}
