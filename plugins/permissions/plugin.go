package permissions

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/permissions/handlers"
	"github.com/xraph/authsome/plugins/permissions/storage"
	"github.com/xraph/forge"
)

const (
	PluginID      = "permissions"
	PluginName    = "Advanced Permissions"
	PluginVersion = "1.0.0"
)

// Plugin implements the AuthSome plugin interface for advanced permissions
type Plugin struct {
	config  *Config
	service *Service
	handler *Handler
}

// NewPlugin creates a new permissions plugin instance
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
	return "Enterprise-grade permissions system with ABAC, dynamic resources, and CEL policy language"
}

// Init initializes the plugin (stub - to be fully implemented in Week 2-4)
func (p *Plugin) Init(auth interface{}) error {
	// Load configuration with defaults
	p.config = DefaultConfig()

	// Initialize service (stub)
	p.service = &Service{
		config: p.config,
	}

	// Initialize handler
	p.handler = NewHandler(p.service)

	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// API group for permissions
	api := router.Group("/api/permissions")

	// Policy management
	api.POST("/policies", p.handler.CreatePolicy,
		forge.WithName("permissions.policies.create"),
		forge.WithSummary("Create permission policy"),
		forge.WithDescription("Create a new ABAC permission policy using CEL expression language"),
		forge.WithRequestSchema(handlers.CreatePolicyRequest{}),
		forge.WithResponseSchema(200, "Policy created", handlers.PolicyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Policies"),
		forge.WithValidation(true),
	)

	api.GET("/policies", p.handler.ListPolicies,
		forge.WithName("permissions.policies.list"),
		forge.WithSummary("List permission policies"),
		forge.WithDescription("List all permission policies for the organization"),
		forge.WithResponseSchema(200, "Policies retrieved", handlers.PoliciesListResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Policies"),
	)

	api.GET("/policies/:id", p.handler.GetPolicy,
		forge.WithName("permissions.policies.get"),
		forge.WithSummary("Get permission policy"),
		forge.WithDescription("Retrieve a specific permission policy by ID"),
		forge.WithResponseSchema(200, "Policy retrieved", handlers.PolicyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Policy not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Policies"),
	)

	api.PUT("/policies/:id", p.handler.UpdatePolicy,
		forge.WithName("permissions.policies.update"),
		forge.WithSummary("Update permission policy"),
		forge.WithDescription("Update an existing permission policy"),
		forge.WithRequestSchema(handlers.UpdatePolicyRequest{}),
		forge.WithResponseSchema(200, "Policy updated", handlers.PolicyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Policy not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Policies"),
		forge.WithValidation(true),
	)

	api.DELETE("/policies/:id", p.handler.DeletePolicy,
		forge.WithName("permissions.policies.delete"),
		forge.WithSummary("Delete permission policy"),
		forge.WithDescription("Delete a permission policy"),
		forge.WithResponseSchema(200, "Policy deleted", handlers.StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Policy not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Policies"),
	)

	api.POST("/policies/validate", p.handler.ValidatePolicy,
		forge.WithName("permissions.policies.validate"),
		forge.WithSummary("Validate policy"),
		forge.WithDescription("Validate a policy's CEL expression syntax without creating it"),
		forge.WithRequestSchema(handlers.ValidatePolicyRequest{}),
		forge.WithResponseSchema(200, "Policy valid", handlers.ValidatePolicyResponse{}),
		forge.WithResponseSchema(400, "Policy invalid", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Policies"),
		forge.WithValidation(true),
	)

	api.POST("/policies/test", p.handler.TestPolicy,
		forge.WithName("permissions.policies.test"),
		forge.WithSummary("Test policy"),
		forge.WithDescription("Test a policy against sample data to verify its behavior"),
		forge.WithRequestSchema(handlers.TestPolicyRequest{}),
		forge.WithResponseSchema(200, "Policy tested", handlers.TestPolicyResponse{}),
		forge.WithResponseSchema(400, "Invalid test request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Policies"),
		forge.WithValidation(true),
	)

	// Resource management
	api.POST("/resources", p.handler.CreateResource,
		forge.WithName("permissions.resources.create"),
		forge.WithSummary("Create resource"),
		forge.WithDescription("Register a new resource type for permission management"),
		forge.WithRequestSchema(handlers.CreateResourceRequest{}),
		forge.WithResponseSchema(200, "Resource created", handlers.ResourceResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Resources"),
		forge.WithValidation(true),
	)

	api.GET("/resources", p.handler.ListResources,
		forge.WithName("permissions.resources.list"),
		forge.WithSummary("List resources"),
		forge.WithDescription("List all registered resource types"),
		forge.WithResponseSchema(200, "Resources retrieved", handlers.ResourcesListResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Resources"),
	)

	api.GET("/resources/:id", p.handler.GetResource,
		forge.WithName("permissions.resources.get"),
		forge.WithSummary("Get resource"),
		forge.WithDescription("Retrieve a specific resource type by ID"),
		forge.WithResponseSchema(200, "Resource retrieved", handlers.ResourceResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Resource not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Resources"),
	)

	api.DELETE("/resources/:id", p.handler.DeleteResource,
		forge.WithName("permissions.resources.delete"),
		forge.WithSummary("Delete resource"),
		forge.WithDescription("Delete a resource type"),
		forge.WithResponseSchema(200, "Resource deleted", handlers.StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Resource not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Resources"),
	)

	// Action management
	api.POST("/actions", p.handler.CreateAction,
		forge.WithName("permissions.actions.create"),
		forge.WithSummary("Create action"),
		forge.WithDescription("Register a new action type for permission policies"),
		forge.WithRequestSchema(handlers.CreateActionRequest{}),
		forge.WithResponseSchema(200, "Action created", handlers.ActionResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Actions"),
		forge.WithValidation(true),
	)

	api.GET("/actions", p.handler.ListActions,
		forge.WithName("permissions.actions.list"),
		forge.WithSummary("List actions"),
		forge.WithDescription("List all registered action types"),
		forge.WithResponseSchema(200, "Actions retrieved", handlers.ActionsListResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Actions"),
	)

	api.DELETE("/actions/:id", p.handler.DeleteAction,
		forge.WithName("permissions.actions.delete"),
		forge.WithSummary("Delete action"),
		forge.WithDescription("Delete an action type"),
		forge.WithResponseSchema(200, "Action deleted", handlers.StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Action not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Actions"),
	)

	// Namespace management
	api.POST("/namespaces", p.handler.CreateNamespace,
		forge.WithName("permissions.namespaces.create"),
		forge.WithSummary("Create namespace"),
		forge.WithDescription("Create a new namespace for organizing permissions"),
		forge.WithRequestSchema(handlers.CreateNamespaceRequest{}),
		forge.WithResponseSchema(200, "Namespace created", handlers.NamespaceResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Namespaces"),
		forge.WithValidation(true),
	)

	api.GET("/namespaces", p.handler.ListNamespaces,
		forge.WithName("permissions.namespaces.list"),
		forge.WithSummary("List namespaces"),
		forge.WithDescription("List all permission namespaces"),
		forge.WithResponseSchema(200, "Namespaces retrieved", handlers.NamespacesListResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Namespaces"),
	)

	api.GET("/namespaces/:id", p.handler.GetNamespace,
		forge.WithName("permissions.namespaces.get"),
		forge.WithSummary("Get namespace"),
		forge.WithDescription("Retrieve a specific namespace by ID"),
		forge.WithResponseSchema(200, "Namespace retrieved", handlers.NamespaceResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Namespace not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Namespaces"),
	)

	api.PUT("/namespaces/:id", p.handler.UpdateNamespace,
		forge.WithName("permissions.namespaces.update"),
		forge.WithSummary("Update namespace"),
		forge.WithDescription("Update a namespace"),
		forge.WithRequestSchema(handlers.UpdateNamespaceRequest{}),
		forge.WithResponseSchema(200, "Namespace updated", handlers.NamespaceResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Namespace not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Namespaces"),
		forge.WithValidation(true),
	)

	api.DELETE("/namespaces/:id", p.handler.DeleteNamespace,
		forge.WithName("permissions.namespaces.delete"),
		forge.WithSummary("Delete namespace"),
		forge.WithDescription("Delete a namespace"),
		forge.WithResponseSchema(200, "Namespace deleted", handlers.StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Namespace not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Namespaces"),
	)

	// Evaluation endpoint (primary authorization check)
	api.POST("/evaluate", p.handler.Evaluate,
		forge.WithName("permissions.evaluate"),
		forge.WithSummary("Evaluate permission"),
		forge.WithDescription("Evaluate whether a user has permission to perform an action on a resource"),
		forge.WithRequestSchema(handlers.EvaluateRequest{}),
		forge.WithResponseSchema(200, "Permission evaluated", handlers.EvaluateResponse{}),
		forge.WithResponseSchema(400, "Invalid evaluation request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Evaluation"),
		forge.WithValidation(true),
	)

	api.POST("/evaluate/batch", p.handler.EvaluateBatch,
		forge.WithName("permissions.evaluate.batch"),
		forge.WithSummary("Batch evaluate permissions"),
		forge.WithDescription("Evaluate multiple permission checks in a single request for efficiency"),
		forge.WithRequestSchema(handlers.BatchEvaluateRequest{}),
		forge.WithResponseSchema(200, "Permissions evaluated", handlers.BatchEvaluateResponse{}),
		forge.WithResponseSchema(400, "Invalid evaluation request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Evaluation"),
		forge.WithValidation(true),
	)

	// Policy templates
	api.GET("/templates", p.handler.ListTemplates,
		forge.WithName("permissions.templates.list"),
		forge.WithSummary("List policy templates"),
		forge.WithDescription("List available policy templates for common permission patterns"),
		forge.WithResponseSchema(200, "Templates retrieved", handlers.TemplatesListResponse{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Templates"),
	)

	api.GET("/templates/:id", p.handler.GetTemplate,
		forge.WithName("permissions.templates.get"),
		forge.WithSummary("Get policy template"),
		forge.WithDescription("Retrieve a specific policy template by ID"),
		forge.WithResponseSchema(200, "Template retrieved", handlers.TemplateResponse{}),
		forge.WithResponseSchema(404, "Template not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Templates"),
	)

	api.POST("/templates/:id/instantiate", p.handler.InstantiateTemplate,
		forge.WithName("permissions.templates.instantiate"),
		forge.WithSummary("Instantiate policy template"),
		forge.WithDescription("Create a new policy from a template with custom parameters"),
		forge.WithRequestSchema(handlers.InstantiateTemplateRequest{}),
		forge.WithResponseSchema(200, "Template instantiated", handlers.PolicyResponse{}),
		forge.WithResponseSchema(400, "Invalid template parameters", errs.AuthsomeError{}),
		forge.WithResponseSchema(404, "Template not found", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Templates"),
		forge.WithValidation(true),
	)

	// Migration from RBAC
	api.POST("/migrate/rbac", p.handler.MigrateFromRBAC,
		forge.WithName("permissions.migrate.rbac"),
		forge.WithSummary("Migrate from RBAC"),
		forge.WithDescription("Migrate existing RBAC policies to the advanced permissions system"),
		forge.WithRequestSchema(handlers.MigrateRBACRequest{}),
		forge.WithResponseSchema(200, "Migration started", handlers.MigrationResponse{}),
		forge.WithResponseSchema(400, "Invalid migration request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Migration"),
		forge.WithValidation(true),
	)

	api.GET("/migrate/rbac/status", p.handler.GetMigrationStatus,
		forge.WithName("permissions.migrate.rbac.status"),
		forge.WithSummary("Get RBAC migration status"),
		forge.WithDescription("Check the status of an ongoing RBAC to permissions migration"),
		forge.WithResponseSchema(200, "Migration status retrieved", handlers.MigrationStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Migration"),
	)

	// Audit & reporting
	api.GET("/audit", p.handler.GetAuditLog,
		forge.WithName("permissions.audit.log"),
		forge.WithSummary("Get permission audit log"),
		forge.WithDescription("Retrieve audit logs of permission evaluations and policy changes"),
		forge.WithResponseSchema(200, "Audit log retrieved", handlers.AuditLogResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Audit"),
	)

	api.GET("/analytics", p.handler.GetAnalytics,
		forge.WithName("permissions.analytics"),
		forge.WithSummary("Get permission analytics"),
		forge.WithDescription("Retrieve analytics and metrics about permission usage and patterns"),
		forge.WithResponseSchema(200, "Analytics retrieved", handlers.AnalyticsResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
		forge.WithTags("Permissions", "Analytics"),
	)

	return nil
}


// RegisterHooks registers lifecycle hooks (stub - hooks not yet defined in core)
func (p *Plugin) RegisterHooks(hooks interface{}) error {
	// TODO: Implement when hook registry is available
	return nil
}

// Migrate runs database migrations for the plugin
func (p *Plugin) Migrate(ctx context.Context) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}
	return p.service.Migrate(ctx)
}

// Service returns the permissions service (for programmatic access)
func (p *Plugin) Service() *Service {
	return p.service
}

// Shutdown gracefully shuts down the plugin
func (p *Plugin) Shutdown(ctx context.Context) error {
	if p.service != nil {
		return p.service.Shutdown(ctx)
	}
	return nil
}

// Health checks plugin health
func (p *Plugin) Health(ctx context.Context) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}
	return p.service.Health(ctx)
}

// Suppress unused warnings for variables that will be used in future implementations
var _ = storage.NewRepository
var _ = storage.NewMemoryCache
var _ = storage.NewRedisCache
