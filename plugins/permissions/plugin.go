package permissions

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/permissions/handlers"
	"github.com/xraph/authsome/plugins/permissions/schema"
	"github.com/xraph/forge"
)

const (
	PluginID      = "permissions"
	PluginName    = "Advanced Permissions"
	PluginVersion = "1.0.0"
)

// Plugin implements the AuthSome plugin interface for advanced permissions
// V2 Architecture: App → Environment → Organization
type Plugin struct {
	config        *Config
	defaultConfig *Config
	service       *Service
	handler       *Handler
	db            *bun.DB
	logger        forge.Logger
	authInst      core.Authsome
}

// PluginOption is a functional option for configuring the permissions plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg *Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithMode sets the evaluation mode
func WithMode(mode string) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Mode = mode
	}
}

// WithCacheBackend sets the cache backend
func WithCacheBackend(backend string) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Cache.Backend = backend
	}
}

// WithCacheEnabled sets whether caching is enabled
func WithCacheEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Cache.Enabled = enabled
	}
}

// WithParallelEvaluation sets whether parallel evaluation is enabled
func WithParallelEvaluation(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Engine.ParallelEvaluation = enabled
	}
}

// WithMaxPoliciesPerOrg sets the maximum policies per organization
func WithMaxPoliciesPerOrg(max int) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Engine.MaxPoliciesPerOrg = max
	}
}

// WithMetricsEnabled sets whether metrics are enabled
func WithMetricsEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Performance.EnableMetrics = enabled
	}
}

// NewPlugin creates a new permissions plugin instance
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		defaultConfig: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
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

// Init initializes the plugin with dependencies
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("permissions plugin requires auth instance")
	}

	p.authInst = authInst

	// Get database
	p.db = authInst.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for permissions plugin")
	}

	// Get Forge app for config and logger
	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for permissions plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", PluginID))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	p.config = DefaultConfig()
	if err := configManager.BindWithDefault("auth.permissions", p.config, *p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind permissions config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("invalid permissions config: %w", err)
	}

	// Register Bun models for permissions plugin
	p.db.RegisterModel((*schema.PermissionPolicy)(nil))
	p.db.RegisterModel((*schema.PermissionNamespace)(nil))
	p.db.RegisterModel((*schema.PermissionResource)(nil))
	p.db.RegisterModel((*schema.PermissionAction)(nil))
	p.db.RegisterModel((*schema.PermissionAuditLog)(nil))
	p.db.RegisterModel((*schema.PermissionEvaluationStats)(nil))

	// Create service with all dependencies
	p.service = NewService(p.db, p.config, p.logger)

	// Initialize handler
	p.handler = NewHandler(p.service)

	p.logger.Info("permissions plugin initialized",
		forge.F("mode", p.config.Mode),
		forge.F("cache_enabled", p.config.Cache.Enabled),
		forge.F("cache_backend", p.config.Cache.Backend),
		forge.F("parallel_evaluation", p.config.Engine.ParallelEvaluation),
		forge.F("max_policies_per_org", p.config.Engine.MaxPoliciesPerOrg))

	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return nil
	}

	// Get global route options for auth middleware
	routeOpts := p.authInst.GetGlobalRoutesOptions()

	// API group for permissions
	api := router.Group("/api/permissions")

	// Policy management
	api.POST("/policies", p.handler.CreatePolicy,
		append(routeOpts,
			forge.WithName("permissions.policies.create"),
			forge.WithSummary("Create permission policy"),
			forge.WithDescription("Create a new ABAC permission policy using CEL expression language"),
			forge.WithRequestSchema(handlers.CreatePolicyRequest{}),
			forge.WithResponseSchema(200, "Policy created", handlers.PolicyResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Policies"),
			forge.WithValidation(true),
		)...,
	)

	api.GET("/policies", p.handler.ListPolicies,
		append(routeOpts,
			forge.WithName("permissions.policies.list"),
			forge.WithSummary("List permission policies"),
			forge.WithDescription("List all permission policies for the environment/organization"),
			forge.WithResponseSchema(200, "Policies retrieved", handlers.PoliciesListResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Policies"),
		)...,
	)

	api.GET("/policies/:id", p.handler.GetPolicy,
		append(routeOpts,
			forge.WithName("permissions.policies.get"),
			forge.WithSummary("Get permission policy"),
			forge.WithDescription("Retrieve a specific permission policy by ID"),
			forge.WithResponseSchema(200, "Policy retrieved", handlers.PolicyResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(404, "Policy not found", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Policies"),
		)...,
	)

	api.PUT("/policies/:id", p.handler.UpdatePolicy,
		append(routeOpts,
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
		)...,
	)

	api.DELETE("/policies/:id", p.handler.DeletePolicy,
		append(routeOpts,
			forge.WithName("permissions.policies.delete"),
			forge.WithSummary("Delete permission policy"),
			forge.WithDescription("Delete a permission policy"),
			forge.WithResponseSchema(200, "Policy deleted", handlers.StatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(404, "Policy not found", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Policies"),
		)...,
	)

	api.POST("/policies/validate", p.handler.ValidatePolicy,
		append(routeOpts,
			forge.WithName("permissions.policies.validate"),
			forge.WithSummary("Validate policy"),
			forge.WithDescription("Validate a policy's CEL expression syntax without creating it"),
			forge.WithRequestSchema(handlers.ValidatePolicyRequest{}),
			forge.WithResponseSchema(200, "Policy valid", handlers.ValidatePolicyResponse{}),
			forge.WithResponseSchema(400, "Policy invalid", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Policies"),
			forge.WithValidation(true),
		)...,
	)

	api.POST("/policies/test", p.handler.TestPolicy,
		append(routeOpts,
			forge.WithName("permissions.policies.test"),
			forge.WithSummary("Test policy"),
			forge.WithDescription("Test a policy against sample data to verify its behavior"),
			forge.WithRequestSchema(handlers.TestPolicyRequest{}),
			forge.WithResponseSchema(200, "Policy tested", handlers.TestPolicyResponse{}),
			forge.WithResponseSchema(400, "Invalid test request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Policies"),
			forge.WithValidation(true),
		)...,
	)

	// Resource management
	api.POST("/resources", p.handler.CreateResource,
		append(routeOpts,
			forge.WithName("permissions.resources.create"),
			forge.WithSummary("Create resource"),
			forge.WithDescription("Register a new resource type for permission management"),
			forge.WithRequestSchema(handlers.CreateResourceRequest{}),
			forge.WithResponseSchema(200, "Resource created", handlers.ResourceResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Resources"),
			forge.WithValidation(true),
		)...,
	)

	api.GET("/resources", p.handler.ListResources,
		append(routeOpts,
			forge.WithName("permissions.resources.list"),
			forge.WithSummary("List resources"),
			forge.WithDescription("List all registered resource types"),
			forge.WithResponseSchema(200, "Resources retrieved", handlers.ResourcesListResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Resources"),
		)...,
	)

	api.GET("/resources/:id", p.handler.GetResource,
		append(routeOpts,
			forge.WithName("permissions.resources.get"),
			forge.WithSummary("Get resource"),
			forge.WithDescription("Retrieve a specific resource type by ID"),
			forge.WithResponseSchema(200, "Resource retrieved", handlers.ResourceResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(404, "Resource not found", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Resources"),
		)...,
	)

	api.DELETE("/resources/:id", p.handler.DeleteResource,
		append(routeOpts,
			forge.WithName("permissions.resources.delete"),
			forge.WithSummary("Delete resource"),
			forge.WithDescription("Delete a resource type"),
			forge.WithResponseSchema(200, "Resource deleted", handlers.StatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(404, "Resource not found", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Resources"),
		)...,
	)

	// Action management
	api.POST("/actions", p.handler.CreateAction,
		append(routeOpts,
			forge.WithName("permissions.actions.create"),
			forge.WithSummary("Create action"),
			forge.WithDescription("Register a new action type for permission policies"),
			forge.WithRequestSchema(handlers.CreateActionRequest{}),
			forge.WithResponseSchema(200, "Action created", handlers.ActionResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Actions"),
			forge.WithValidation(true),
		)...,
	)

	api.GET("/actions", p.handler.ListActions,
		append(routeOpts,
			forge.WithName("permissions.actions.list"),
			forge.WithSummary("List actions"),
			forge.WithDescription("List all registered action types"),
			forge.WithResponseSchema(200, "Actions retrieved", handlers.ActionsListResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Actions"),
		)...,
	)

	api.DELETE("/actions/:id", p.handler.DeleteAction,
		append(routeOpts,
			forge.WithName("permissions.actions.delete"),
			forge.WithSummary("Delete action"),
			forge.WithDescription("Delete an action type"),
			forge.WithResponseSchema(200, "Action deleted", handlers.StatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(404, "Action not found", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Actions"),
		)...,
	)

	// Namespace management
	api.POST("/namespaces", p.handler.CreateNamespace,
		append(routeOpts,
			forge.WithName("permissions.namespaces.create"),
			forge.WithSummary("Create namespace"),
			forge.WithDescription("Create a new namespace for organizing permissions"),
			forge.WithRequestSchema(handlers.CreateNamespaceRequest{}),
			forge.WithResponseSchema(200, "Namespace created", handlers.NamespaceResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Namespaces"),
			forge.WithValidation(true),
		)...,
	)

	api.GET("/namespaces", p.handler.ListNamespaces,
		append(routeOpts,
			forge.WithName("permissions.namespaces.list"),
			forge.WithSummary("List namespaces"),
			forge.WithDescription("List all permission namespaces"),
			forge.WithResponseSchema(200, "Namespaces retrieved", handlers.NamespacesListResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Namespaces"),
		)...,
	)

	api.GET("/namespaces/:id", p.handler.GetNamespace,
		append(routeOpts,
			forge.WithName("permissions.namespaces.get"),
			forge.WithSummary("Get namespace"),
			forge.WithDescription("Retrieve a specific namespace by ID"),
			forge.WithResponseSchema(200, "Namespace retrieved", handlers.NamespaceResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(404, "Namespace not found", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Namespaces"),
		)...,
	)

	api.PUT("/namespaces/:id", p.handler.UpdateNamespace,
		append(routeOpts,
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
		)...,
	)

	api.DELETE("/namespaces/:id", p.handler.DeleteNamespace,
		append(routeOpts,
			forge.WithName("permissions.namespaces.delete"),
			forge.WithSummary("Delete namespace"),
			forge.WithDescription("Delete a namespace"),
			forge.WithResponseSchema(200, "Namespace deleted", handlers.StatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(404, "Namespace not found", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Namespaces"),
		)...,
	)

	// Evaluation endpoint (primary authorization check)
	api.POST("/evaluate", p.handler.Evaluate,
		append(routeOpts,
			forge.WithName("permissions.evaluate"),
			forge.WithSummary("Evaluate permission"),
			forge.WithDescription("Evaluate whether a user has permission to perform an action on a resource"),
			forge.WithRequestSchema(handlers.EvaluateRequest{}),
			forge.WithResponseSchema(200, "Permission evaluated", handlers.EvaluateResponse{}),
			forge.WithResponseSchema(400, "Invalid evaluation request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Evaluation"),
			forge.WithValidation(true),
		)...,
	)

	api.POST("/evaluate/batch", p.handler.EvaluateBatch,
		append(routeOpts,
			forge.WithName("permissions.evaluate.batch"),
			forge.WithSummary("Batch evaluate permissions"),
			forge.WithDescription("Evaluate multiple permission checks in a single request for efficiency"),
			forge.WithRequestSchema(handlers.BatchEvaluateRequest{}),
			forge.WithResponseSchema(200, "Permissions evaluated", handlers.BatchEvaluateResponse{}),
			forge.WithResponseSchema(400, "Invalid evaluation request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Evaluation"),
			forge.WithValidation(true),
		)...,
	)

	// Policy templates
	api.GET("/templates", p.handler.ListTemplates,
		append(routeOpts,
			forge.WithName("permissions.templates.list"),
			forge.WithSummary("List policy templates"),
			forge.WithDescription("List available policy templates for common permission patterns"),
			forge.WithResponseSchema(200, "Templates retrieved", handlers.TemplatesListResponse{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Templates"),
		)...,
	)

	api.GET("/templates/:id", p.handler.GetTemplate,
		append(routeOpts,
			forge.WithName("permissions.templates.get"),
			forge.WithSummary("Get policy template"),
			forge.WithDescription("Retrieve a specific policy template by ID"),
			forge.WithResponseSchema(200, "Template retrieved", handlers.TemplateResponse{}),
			forge.WithResponseSchema(404, "Template not found", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Templates"),
		)...,
	)

	api.POST("/templates/:id/instantiate", p.handler.InstantiateTemplate,
		append(routeOpts,
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
		)...,
	)

	// Migration from RBAC
	api.POST("/migrate/rbac", p.handler.MigrateFromRBAC,
		append(routeOpts,
			forge.WithName("permissions.migrate.rbac"),
			forge.WithSummary("Migrate from RBAC"),
			forge.WithDescription("Migrate existing RBAC policies to the advanced permissions system"),
			forge.WithRequestSchema(handlers.MigrateRBACRequest{}),
			forge.WithResponseSchema(200, "Migration started", handlers.MigrationResponse{}),
			forge.WithResponseSchema(400, "Invalid migration request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Migration"),
			forge.WithValidation(true),
		)...,
	)

	api.GET("/migrate/rbac/status", p.handler.GetMigrationStatus,
		append(routeOpts,
			forge.WithName("permissions.migrate.rbac.status"),
			forge.WithSummary("Get RBAC migration status"),
			forge.WithDescription("Check the status of an ongoing RBAC to permissions migration"),
			forge.WithResponseSchema(200, "Migration status retrieved", handlers.MigrationStatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Migration"),
		)...,
	)

	// Audit & reporting
	api.GET("/audit", p.handler.GetAuditLog,
		append(routeOpts,
			forge.WithName("permissions.audit.log"),
			forge.WithSummary("Get permission audit log"),
			forge.WithDescription("Retrieve audit logs of permission evaluations and policy changes"),
			forge.WithResponseSchema(200, "Audit log retrieved", handlers.AuditLogResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Audit"),
		)...,
	)

	api.GET("/analytics", p.handler.GetAnalytics,
		append(routeOpts,
			forge.WithName("permissions.analytics"),
			forge.WithSummary("Get permission analytics"),
			forge.WithDescription("Retrieve analytics and metrics about permission usage and patterns"),
			forge.WithResponseSchema(200, "Analytics retrieved", handlers.AnalyticsResponse{}),
			forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
			forge.WithResponseSchema(501, "Not implemented", handlers.MessageResponse{}),
			forge.WithTags("Permissions", "Analytics"),
		)...,
	)

	return nil
}

// RegisterHooks registers lifecycle hooks
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	// TODO: Register permission-related hooks in future phase
	// - BeforePermissionEvaluate
	// - AfterPermissionEvaluate
	// - OnPolicyChange
	// - OnCacheInvalidate
	return nil
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// The permissions plugin doesn't decorate core services
	// It provides its own independent permission system
	return nil
}

// Migrate runs database migrations for the plugin
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return fmt.Errorf("database not initialized")
	}

	ctx := context.Background()

	// Create tables
	models := []interface{}{
		(*schema.PermissionPolicy)(nil),
		(*schema.PermissionNamespace)(nil),
		(*schema.PermissionResource)(nil),
		(*schema.PermissionAction)(nil),
		(*schema.PermissionAuditLog)(nil),
		(*schema.PermissionEvaluationStats)(nil),
	}

	for _, model := range models {
		_, err := p.db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create table for %T: %w", model, err)
		}
	}

	// Create indexes
	if err := p.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	if p.logger != nil {
		p.logger.Info("permissions plugin migrations completed")
	}

	return nil
}

// createIndexes creates database indexes for optimal performance
func (p *Plugin) createIndexes(ctx context.Context) error {
	indexes := []string{
		// PermissionPolicy indexes
		`CREATE INDEX IF NOT EXISTS idx_permission_policies_app_id ON permission_policies(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_policies_env_id ON permission_policies(environment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_policies_namespace_id ON permission_policies(namespace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_policies_resource_type ON permission_policies(resource_type)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_policies_enabled ON permission_policies(enabled) WHERE enabled = true`,

		// PermissionNamespace indexes
		`CREATE INDEX IF NOT EXISTS idx_permission_namespaces_app_id ON permission_namespaces(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_namespaces_env_id ON permission_namespaces(environment_id)`,

		// PermissionResource indexes
		`CREATE INDEX IF NOT EXISTS idx_permission_resources_namespace_id ON permission_resources(namespace_id)`,

		// PermissionAction indexes
		`CREATE INDEX IF NOT EXISTS idx_permission_actions_namespace_id ON permission_actions(namespace_id)`,

		// PermissionAuditLog indexes
		`CREATE INDEX IF NOT EXISTS idx_permission_audit_logs_app_id ON permission_audit_logs(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_audit_logs_env_id ON permission_audit_logs(environment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_audit_logs_actor_id ON permission_audit_logs(actor_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_audit_logs_timestamp ON permission_audit_logs(timestamp DESC)`,

		// PermissionEvaluationStats indexes
		`CREATE INDEX IF NOT EXISTS idx_permission_eval_stats_app_id ON permission_evaluation_stats(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_eval_stats_env_id ON permission_evaluation_stats(environment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permission_eval_stats_policy_id ON permission_evaluation_stats(policy_id)`,
	}

	for _, idx := range indexes {
		if _, err := p.db.ExecContext(ctx, idx); err != nil {
			// Log but don't fail - some databases may not support all index syntax
			if p.logger != nil {
				p.logger.Warn("failed to create index", forge.F("error", err.Error()))
			}
		}
	}

	return nil
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
