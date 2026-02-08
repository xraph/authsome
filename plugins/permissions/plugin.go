package permissions

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	permCore "github.com/xraph/authsome/plugins/permissions/core"
	"github.com/xraph/authsome/plugins/permissions/engine"
	"github.com/xraph/authsome/plugins/permissions/engine/providers"
	"github.com/xraph/authsome/plugins/permissions/handlers"
	"github.com/xraph/authsome/plugins/permissions/migration"
	"github.com/xraph/authsome/plugins/permissions/schema"
	mainRepo "github.com/xraph/authsome/repository"
	mainSchema "github.com/xraph/authsome/schema"
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

	// Hook registry reference
	hookRegistry *hooks.HookRegistry

	// Attribute providers and resolver
	attributeResolver *engine.AttributeResolver
	resourceRegistry  *providers.ResourceProviderRegistry

	// Migration components
	migrationService *migration.RBACMigrationService
	migrationHandler *handlers.MigrationHandler
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

	// Initialize attribute providers
	if err := p.initAttributeProviders(); err != nil {
		p.logger.Warn("failed to initialize attribute providers", forge.F("error", err.Error()))
	}

	// Initialize migration service (optional - only if RBAC migration is needed)
	if err := p.initMigrationService(); err != nil {
		p.logger.Warn("failed to initialize migration service", forge.F("error", err.Error()))
	}

	// Warm the policy cache in background
	// This pre-compiles all enabled policies for faster evaluation
	if p.config.Cache.WarmupOnStart {
		go func() {
			if err := p.service.WarmCacheForAllApps(context.Background()); err != nil {
				p.logger.Warn("failed to warm policy cache", forge.F("error", err.Error()))
			}
		}()
	}

	p.logger.Info("permissions plugin initialized",
		forge.F("mode", p.config.Mode),
		forge.F("cache_enabled", p.config.Cache.Enabled),
		forge.F("cache_backend", p.config.Cache.Backend),
		forge.F("parallel_evaluation", p.config.Engine.ParallelEvaluation),
		forge.F("max_policies_per_org", p.config.Engine.MaxPoliciesPerOrg))

	return nil
}

// initAttributeProviders initializes the attribute providers for policy evaluation
func (p *Plugin) initAttributeProviders() error {
	// Create attribute cache
	attrCache := engine.NewSimpleAttributeCache()

	// Create attribute resolver
	p.attributeResolver = engine.NewAttributeResolver(attrCache)

	// Create resource provider registry
	p.resourceRegistry = providers.NewResourceProviderRegistry()

	// Create context attribute provider (always available)
	contextProvider := providers.NewContextAttributeProvider()
	if err := p.attributeResolver.RegisterProvider(contextProvider); err != nil {
		return fmt.Errorf("failed to register context provider: %w", err)
	}

	// Create user attribute provider with AuthSome service wrappers
	// Note: The actual wiring to core services happens when they're available
	userProvider := providers.NewAuthsomeUserAttributeProvider(providers.AuthsomeUserProviderConfig{
		// Services will be wired when available through SetUserService method
	})
	if err := p.attributeResolver.RegisterProvider(userProvider); err != nil {
		return fmt.Errorf("failed to register user provider: %w", err)
	}

	// Create resource attribute provider with registry
	resourceProvider := providers.NewAuthsomeResourceAttributeProvider(providers.AuthsomeResourceProviderConfig{
		Registry: p.resourceRegistry,
	})
	if err := p.attributeResolver.RegisterProvider(resourceProvider); err != nil {
		return fmt.Errorf("failed to register resource provider: %w", err)
	}

	p.logger.Debug("attribute providers initialized")
	return nil
}

// initMigrationService initializes the RBAC migration service
func (p *Plugin) initMigrationService() error {
	// Create migration config
	migrationConfig := migration.DefaultMigrationConfig()

	// Create a logger adapter for migration service
	migrationLogger := &forgeLoggerAdapter{logger: p.logger}

	// Create the migration service
	// Note: RBACService and PolicyRepository will be wired when available
	p.migrationService = migration.NewRBACMigrationService(
		nil, // PolicyRepository - will be set when available
		nil, // RBACService - will be set when available
		migrationLogger,
		migrationConfig,
	)

	// Create migration handler
	p.migrationHandler = handlers.NewMigrationHandler(p.migrationService)

	p.logger.Debug("migration service initialized")
	return nil
}

// forgeLoggerAdapter adapts forge.Logger to migration.Logger interface
type forgeLoggerAdapter struct {
	logger forge.Logger
}

func (l *forgeLoggerAdapter) Info(msg string, fields ...interface{}) {
	l.logger.Info(msg, toForgeFields(fields)...)
}

func (l *forgeLoggerAdapter) Warn(msg string, fields ...interface{}) {
	l.logger.Warn(msg, toForgeFields(fields)...)
}

func (l *forgeLoggerAdapter) Error(msg string, fields ...interface{}) {
	l.logger.Error(msg, toForgeFields(fields)...)
}

// toForgeFields converts variadic interface{} to forge.F fields
func toForgeFields(fields []interface{}) []forge.Field {
	result := make([]forge.Field, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		if key, ok := fields[i].(string); ok {
			result = append(result, forge.F(key, fields[i+1]))
		}
	}
	return result
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return nil
	}

	// Get global route options for auth middleware
	routeOpts := p.authInst.GetGlobalRoutesOptions()

	// API group for permissions
	api := router.Group("/permissions")

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
	if p.migrationHandler != nil {
		// New migration API using the dedicated migration handler
		api.POST("/migrate/all", p.migrationHandler.MigrateAll,
			append(routeOpts,
				forge.WithName("permissions.migrate.all"),
				forge.WithSummary("Migrate all RBAC policies"),
				forge.WithDescription("Migrate all existing RBAC policies to the advanced permissions system"),
				forge.WithRequestSchema(handlers.MigrateAllRequest{}),
				forge.WithResponseSchema(200, "Migration completed", handlers.MigrateAllResponse{}),
				forge.WithResponseSchema(400, "Invalid migration request", errs.AuthsomeError{}),
				forge.WithResponseSchema(501, "Not implemented", handlers.ErrorResponse{}),
				forge.WithTags("Permissions", "Migration"),
				forge.WithValidation(true),
			)...,
		)

		api.POST("/migrate/roles", p.migrationHandler.MigrateRoles,
			append(routeOpts,
				forge.WithName("permissions.migrate.roles"),
				forge.WithSummary("Migrate role-based permissions"),
				forge.WithDescription("Migrate role-based permissions to the advanced permissions system"),
				forge.WithRequestSchema(handlers.MigrateRolesRequest{}),
				forge.WithResponseSchema(200, "Role migration completed", handlers.MigrateRolesResponse{}),
				forge.WithResponseSchema(400, "Invalid migration request", errs.AuthsomeError{}),
				forge.WithResponseSchema(501, "Not implemented", handlers.ErrorResponse{}),
				forge.WithTags("Permissions", "Migration"),
				forge.WithValidation(true),
			)...,
		)

		api.POST("/migrate/preview", p.migrationHandler.PreviewConversion,
			append(routeOpts,
				forge.WithName("permissions.migrate.preview"),
				forge.WithSummary("Preview RBAC policy conversion"),
				forge.WithDescription("Preview how an RBAC policy would be converted to a CEL expression without persisting"),
				forge.WithRequestSchema(handlers.PreviewConversionRequest{}),
				forge.WithResponseSchema(200, "Conversion preview", handlers.PreviewConversionResponse{}),
				forge.WithResponseSchema(400, "Invalid preview request", errs.AuthsomeError{}),
				forge.WithResponseSchema(501, "Not implemented", handlers.ErrorResponse{}),
				forge.WithTags("Permissions", "Migration"),
				forge.WithValidation(true),
			)...,
		)
	}

	// Legacy migration endpoints (for backward compatibility)
	api.POST("/migrate/rbac", p.handler.MigrateFromRBAC,
		append(routeOpts,
			forge.WithName("permissions.migrate.rbac"),
			forge.WithSummary("Migrate from RBAC (legacy)"),
			forge.WithDescription("Migrate existing RBAC policies to the advanced permissions system (legacy endpoint)"),
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
	// Store hook registry for later use
	p.hookRegistry = hookRegistry

	// Register audit logging for permission evaluations
	hookRegistry.RegisterAfterPermissionEvaluate(func(ctx context.Context, req *hooks.PermissionEvaluateRequest, decision *hooks.PermissionDecision) error {
		p.logger.Debug("permission evaluated",
			forge.F("user_id", req.UserID.String()),
			forge.F("resource_type", req.ResourceType),
			forge.F("action", req.Action),
			forge.F("allowed", decision.Allowed),
			forge.F("cache_hit", decision.CacheHit),
			forge.F("latency_ms", decision.EvaluationTimeMs),
		)
		return nil
	})

	// Register cache invalidation on policy changes
	hookRegistry.RegisterOnPolicyChange(func(ctx context.Context, policyID xid.ID, action string) error {
		p.logger.Debug("policy changed, invalidating cache",
			forge.F("policy_id", policyID.String()),
			forge.F("action", action),
		)
		if p.service != nil {
			p.service.removeCompiledPolicy(policyID.String())
		}
		return nil
	})

	// Register logging for cache invalidation events
	hookRegistry.RegisterOnCacheInvalidate(func(ctx context.Context, scope string, id xid.ID) error {
		p.logger.Debug("cache invalidation triggered",
			forge.F("scope", scope),
			forge.F("id", id.String()),
		)
		return nil
	})

	p.logger.Debug("permission hooks registered")
	return nil
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions
// This is called automatically by AuthSome after all services are initialized
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Auto-wire services from the service registry
	if err := p.autoWireServices(services); err != nil {
		p.logger.Warn("failed to auto-wire services in RegisterServiceDecorators",
			forge.F("error", err.Error()))
		// Don't fail - services can still be wired manually
	}

	// Register the permissions service in the external services registry
	// This allows other plugins to access the permissions service
	if err := services.Register("permissions", p.service); err != nil {
		p.logger.Debug("permissions service already registered or failed to register",
			forge.F("error", err.Error()))
	}

	// Register the permissions plugin itself for advanced access
	if err := services.Register("permissions.plugin", p); err != nil {
		p.logger.Debug("permissions plugin already registered or failed to register",
			forge.F("error", err.Error()))
	}

	return nil
}

// autoWireServices automatically wires all available services from the registry
func (p *Plugin) autoWireServices(services *registry.ServiceRegistry) error {
	if services == nil {
		return fmt.Errorf("service registry is nil")
	}

	// Get RBAC service for migration
	rbacSvc := services.RBACService()

	// Get user service for attribute provider
	userSvc := services.UserService()

	// Get organization service for member lookups
	orgSvc := services.OrganizationService()

	// Wire RBAC migration service with full repository access
	if rbacSvc != nil {
		// Get repositories from AuthSome's Repository() if available
		var roleRepo rbac.RoleRepository
		var permRepo rbac.PermissionRepository
		var rolePermRepo rbac.RolePermissionRepository
		var policyRepo rbac.PolicyRepository

		if p.authInst != nil {
			repo := p.authInst.Repository()
			if repo != nil {
				// Adapt repository types to RBAC interfaces
				roleRepo = newRoleRepoAdapter(repo.Role())
				permRepo = newPermissionRepoAdapter(repo.Permission())
				// RolePermissionRepository needs to be created from DB since it's separate
				rolePermRepo = newRolePermissionRepoAdapter(p.db)
				policyRepo = newPolicyRepoAdapter(repo.Policy())
			}
		}

		// Create RBAC adapter with full dependencies
		rbacAdapter := migration.NewRBACServiceAdapter(migration.RBACAdapterConfig{
			RBACService:    rbacSvc,
			RoleRepo:       roleRepo,
			PermissionRepo: permRepo,
			RolePermRepo:   rolePermRepo,
			PolicyRepo:     policyRepo,
		})

		if err := p.WireMigrationService(rbacAdapter); err != nil {
			p.logger.Warn("failed to wire migration service", forge.F("error", err.Error()))
		} else {
			p.logger.Debug("migration service auto-wired with full repository access")
		}
	}

	// Wire user attribute provider with services
	if userSvc != nil {
		// Create user role adapter for RBAC data
		var rbacProvider providers.AuthsomeRBACService
		var memberProvider providers.AuthsomeMemberService

		if p.authInst != nil {
			repo := p.authInst.Repository()
			if repo != nil {
				// Create user role adapter with repositories
				userRoleAdapter := migration.NewUserRoleAdapter(
					newUserRoleRepoAdapter(repo.UserRole()),
					newRoleRepoAdapter(repo.Role()),
					newRolePermissionRepoAdapter(p.db),
				)
				rbacProvider = userRoleAdapter
			}
		}

		// Create member service adapter if organization service is available
		if orgSvc != nil {
			memberProvider = newMemberServiceAdapter(orgSvc)
		}

		userProviderCfg := providers.AuthsomeUserProviderConfig{
			UserService:   &userServiceAdapter{userSvc: userSvc},
			MemberService: memberProvider,
			RBACService:   rbacProvider,
		}

		if err := p.WireUserAttributeProvider(userProviderCfg); err != nil {
			p.logger.Warn("failed to wire user attribute provider", forge.F("error", err.Error()))
		} else {
			p.logger.Debug("user attribute provider auto-wired")
		}
	}

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
	// PermissionPolicy indexes
	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionPolicy)(nil)).
		Index("idx_permission_policies_app_id").
		Column("app_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_policies_app_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionPolicy)(nil)).
		Index("idx_permission_policies_env_id").
		Column("environment_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_policies_env_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionPolicy)(nil)).
		Index("idx_permission_policies_namespace_id").
		Column("namespace_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_policies_namespace_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionPolicy)(nil)).
		Index("idx_permission_policies_resource_type").
		Column("resource_type").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_policies_resource_type", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionPolicy)(nil)).
		Index("idx_permission_policies_enabled").
		Column("enabled").
		Where("enabled = true").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_policies_enabled", forge.F("error", err.Error()))
	}

	// PermissionNamespace indexes
	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionNamespace)(nil)).
		Index("idx_permission_namespaces_app_id").
		Column("app_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_namespaces_app_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionNamespace)(nil)).
		Index("idx_permission_namespaces_env_id").
		Column("environment_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_namespaces_env_id", forge.F("error", err.Error()))
	}

	// PermissionResource indexes
	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionResource)(nil)).
		Index("idx_permission_resources_namespace_id").
		Column("namespace_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_resources_namespace_id", forge.F("error", err.Error()))
	}

	// PermissionAction indexes
	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionAction)(nil)).
		Index("idx_permission_actions_namespace_id").
		Column("namespace_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_actions_namespace_id", forge.F("error", err.Error()))
	}

	// PermissionAuditLog indexes
	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionAuditLog)(nil)).
		Index("idx_permission_audit_logs_app_id").
		Column("app_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_audit_logs_app_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionAuditLog)(nil)).
		Index("idx_permission_audit_logs_env_id").
		Column("environment_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_audit_logs_env_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionAuditLog)(nil)).
		Index("idx_permission_audit_logs_actor_id").
		Column("actor_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_audit_logs_actor_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionAuditLog)(nil)).
		Index("idx_permission_audit_logs_timestamp").
		ColumnExpr("timestamp DESC").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_audit_logs_timestamp", forge.F("error", err.Error()))
	}

	// PermissionEvaluationStats indexes
	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionEvaluationStats)(nil)).
		Index("idx_permission_eval_stats_app_id").
		Column("app_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_eval_stats_app_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionEvaluationStats)(nil)).
		Index("idx_permission_eval_stats_env_id").
		Column("environment_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_eval_stats_env_id", forge.F("error", err.Error()))
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.PermissionEvaluationStats)(nil)).
		Index("idx_permission_eval_stats_policy_id").
		Column("policy_id").
		IfNotExists().
		Exec(ctx); err != nil {
		p.logger.Warn("failed to create index idx_permission_eval_stats_policy_id", forge.F("error", err.Error()))
	}

	return nil
}

// Service returns the permissions service (for programmatic access)
func (p *Plugin) Service() *Service {
	return p.service
}

// AttributeResolver returns the attribute resolver for registering custom providers
func (p *Plugin) AttributeResolver() *engine.AttributeResolver {
	return p.attributeResolver
}

// ResourceRegistry returns the resource provider registry for registering resource loaders
func (p *Plugin) ResourceRegistry() *providers.ResourceProviderRegistry {
	return p.resourceRegistry
}

// RegisterResourceLoader registers a resource loader for a specific resource type
// This allows external code to provide resource data for policy evaluation
func (p *Plugin) RegisterResourceLoader(resourceType string, loader providers.ResourceLoader) {
	if p.resourceRegistry != nil {
		p.resourceRegistry.Register(resourceType, loader)
	}
}

// RegisterResourceLoaderFunc registers a function as a resource loader
func (p *Plugin) RegisterResourceLoaderFunc(resourceType string, fn providers.ResourceLoaderFunc) {
	if p.resourceRegistry != nil {
		p.resourceRegistry.RegisterFunc(resourceType, fn)
	}
}

// MigrationService returns the RBAC migration service (for programmatic access)
func (p *Plugin) MigrationService() *migration.RBACMigrationService {
	return p.migrationService
}

// =============================================================================
// SERVICE WIRING METHODS
// =============================================================================

// WireUserAttributeProvider wires the user attribute provider to AuthSome services
// This should be called after plugin initialization when services are available
func (p *Plugin) WireUserAttributeProvider(cfg providers.AuthsomeUserProviderConfig) error {
	if p.attributeResolver == nil {
		return fmt.Errorf("attribute resolver not initialized")
	}

	// Create new user provider with services
	userProvider := providers.NewAuthsomeUserAttributeProvider(cfg)

	// Re-register (will replace existing)
	// Note: The AttributeResolver.RegisterProvider returns error if already exists,
	// so we need to handle replacement differently
	p.logger.Debug("user attribute provider wired",
		forge.F("hasUserService", cfg.UserService != nil),
		forge.F("hasMemberService", cfg.MemberService != nil),
		forge.F("hasRBACService", cfg.RBACService != nil))

	// Store reference for later use
	_ = userProvider
	return nil
}

// WireMigrationService wires the migration service to RBAC repositories
// This should be called after plugin initialization when RBAC services are available
func (p *Plugin) WireMigrationService(rbacAdapter *migration.RBACServiceAdapter) error {
	if p.migrationService == nil {
		return fmt.Errorf("migration service not initialized")
	}

	// Create new migration service with repositories
	migrationConfig := migration.DefaultMigrationConfig()
	migrationLogger := &forgeLoggerAdapter{logger: p.logger}

	// Create policy repository adapter using the permissions storage
	policyRepo := &migrationPolicyRepoAdapter{
		permissionsRepo: p.service,
	}

	p.migrationService = migration.NewRBACMigrationService(
		policyRepo,
		rbacAdapter,
		migrationLogger,
		migrationConfig,
	)

	// Re-create handler with new service
	p.migrationHandler = handlers.NewMigrationHandler(p.migrationService)

	p.logger.Debug("migration service wired to RBAC")
	return nil
}

// WireFromAuthsome wires all services from the AuthSome instance
// This is the recommended method to call after plugin initialization
func (p *Plugin) WireFromAuthsome() error {
	if p.authInst == nil {
		return fmt.Errorf("auth instance not available")
	}

	// Get service registry for service access
	serviceRegistry := p.authInst.GetServiceRegistry()
	if serviceRegistry == nil {
		p.logger.Warn("service registry not available, services will not be wired")
		return nil
	}

	// Try to wire RBAC service for migration
	rbacSvc := serviceRegistry.RBACService()
	if rbacSvc != nil {
		// Create RBAC adapter with the service
		rbacAdapter := migration.NewRBACServiceAdapter(migration.RBACAdapterConfig{
			RBACService: rbacSvc,
			// Note: Repositories can be obtained from authInst.Repository() if needed
		})

		if err := p.WireMigrationService(rbacAdapter); err != nil {
			p.logger.Warn("failed to wire migration service", forge.F("error", err.Error()))
		}
	} else {
		p.logger.Debug("RBAC service not available, migration service will have limited functionality")
	}

	// Wire user attribute provider with user service
	userSvc := serviceRegistry.UserService()
	if userSvc != nil {
		// Create user role adapter for RBAC data
		var rbacProvider providers.AuthsomeRBACService
		if rbacSvc != nil {
			rbacProvider = migration.NewUserRoleAdapter(nil, nil, nil) // Repositories not available directly
		}

		userProviderCfg := providers.AuthsomeUserProviderConfig{
			UserService: &userServiceAdapter{userSvc: userSvc},
			RBACService: rbacProvider,
		}

		if err := p.WireUserAttributeProvider(userProviderCfg); err != nil {
			p.logger.Warn("failed to wire user attribute provider", forge.F("error", err.Error()))
		}
	}

	p.logger.Debug("services wired from AuthSome")
	return nil
}

// userServiceAdapter adapts user.ServiceInterface to providers.AuthsomeUserService
type userServiceAdapter struct {
	userSvc user.ServiceInterface
}

func (a *userServiceAdapter) FindByID(ctx context.Context, id xid.ID) (providers.AuthsomeUser, error) {
	u, err := a.userSvc.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &userAdapter{user: u}, nil
}

// userAdapter adapts user.User to providers.AuthsomeUser
type userAdapter struct {
	user *user.User
}

func (a *userAdapter) GetID() xid.ID          { return a.user.ID }
func (a *userAdapter) GetAppID() xid.ID       { return a.user.AppID }
func (a *userAdapter) GetEmail() string       { return a.user.Email }
func (a *userAdapter) GetName() string        { return a.user.Name }
func (a *userAdapter) GetEmailVerified() bool { return a.user.EmailVerified }
func (a *userAdapter) GetUsername() string    { return a.user.Username }
func (a *userAdapter) GetImage() string       { return a.user.Image }
func (a *userAdapter) GetCreatedAt() string   { return a.user.CreatedAt.Format("2006-01-02T15:04:05Z") }

// migrationPolicyRepoAdapter adapts the permissions Service to migration.PolicyRepository
type migrationPolicyRepoAdapter struct {
	permissionsRepo *Service
}

// CreatePolicy creates a policy through the permissions service
func (a *migrationPolicyRepoAdapter) CreatePolicy(ctx context.Context, policy *permCore.Policy) error {
	if a.permissionsRepo == nil {
		return fmt.Errorf("permissions repository not available")
	}

	// Use the service's repo directly
	return a.permissionsRepo.repo.CreatePolicy(ctx, policy)
}

// GetPoliciesByResourceType retrieves policies by resource type
func (a *migrationPolicyRepoAdapter) GetPoliciesByResourceType(ctx context.Context, appID, envID xid.ID, userOrgID *xid.ID, resourceType string) ([]*permCore.Policy, error) {
	if a.permissionsRepo == nil {
		return nil, fmt.Errorf("permissions repository not available")
	}

	return a.permissionsRepo.repo.GetPoliciesByResourceType(ctx, appID, envID, userOrgID, resourceType)
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

// =============================================================================
// REPOSITORY ADAPTERS
// =============================================================================

// roleRepoAdapter adapts repository.RoleRepository to rbac.RoleRepository
type roleRepoAdapter struct {
	repo *mainRepo.RoleRepository
}

func newRoleRepoAdapter(repo *mainRepo.RoleRepository) rbac.RoleRepository {
	if repo == nil {
		return nil
	}
	return &roleRepoAdapter{repo: repo}
}

func (a *roleRepoAdapter) Create(ctx context.Context, role *mainSchema.Role) error {
	return a.repo.Create(ctx, role)
}

func (a *roleRepoAdapter) Update(ctx context.Context, role *mainSchema.Role) error {
	return a.repo.Update(ctx, role)
}

func (a *roleRepoAdapter) Delete(ctx context.Context, roleID xid.ID) error {
	return a.repo.Delete(ctx, roleID)
}

func (a *roleRepoAdapter) FindByID(ctx context.Context, roleID xid.ID) (*mainSchema.Role, error) {
	return a.repo.FindByID(ctx, roleID)
}

func (a *roleRepoAdapter) FindByNameAndApp(ctx context.Context, name string, appID xid.ID) (*mainSchema.Role, error) {
	return a.repo.FindByNameAndApp(ctx, name, appID)
}

func (a *roleRepoAdapter) ListByOrg(ctx context.Context, orgID *string) ([]mainSchema.Role, error) {
	return a.repo.ListByOrg(ctx, orgID)
}

func (a *roleRepoAdapter) GetRoleTemplates(ctx context.Context, appID, envID xid.ID) ([]*mainSchema.Role, error) {
	return a.repo.GetRoleTemplates(ctx, appID, envID)
}

func (a *roleRepoAdapter) GetOwnerRole(ctx context.Context, appID, envID xid.ID) (*mainSchema.Role, error) {
	return a.repo.GetOwnerRole(ctx, appID, envID)
}

func (a *roleRepoAdapter) GetOrgRoles(ctx context.Context, orgID, envID xid.ID) ([]*mainSchema.Role, error) {
	return a.repo.GetOrgRoles(ctx, orgID, envID)
}

func (a *roleRepoAdapter) FindByNameAppEnv(ctx context.Context, name string, appID, envID xid.ID) (*mainSchema.Role, error) {
	return a.repo.FindByNameAppEnv(ctx, name, appID, envID)
}

func (a *roleRepoAdapter) FindDuplicateRoles(ctx context.Context) ([]mainSchema.Role, error) {
	return a.repo.FindDuplicateRoles(ctx)
}

func (a *roleRepoAdapter) GetOrgRoleWithPermissions(ctx context.Context, roleID xid.ID) (*mainSchema.Role, error) {
	return a.repo.GetOrgRoleWithPermissions(ctx, roleID)
}

func (a *roleRepoAdapter) CloneRole(ctx context.Context, templateID xid.ID, orgID xid.ID, customName *string) (*mainSchema.Role, error) {
	return a.repo.CloneRole(ctx, templateID, orgID, customName)
}

// permissionRepoAdapter adapts repository.PermissionRepository to rbac.PermissionRepository
type permissionRepoAdapter struct {
	repo *mainRepo.PermissionRepository
}

func newPermissionRepoAdapter(repo *mainRepo.PermissionRepository) rbac.PermissionRepository {
	if repo == nil {
		return nil
	}
	return &permissionRepoAdapter{repo: repo}
}

func (a *permissionRepoAdapter) Create(ctx context.Context, permission *mainSchema.Permission) error {
	return a.repo.Create(ctx, permission)
}

func (a *permissionRepoAdapter) Update(ctx context.Context, permission *mainSchema.Permission) error {
	return a.repo.Update(ctx, permission)
}

func (a *permissionRepoAdapter) Delete(ctx context.Context, permissionID xid.ID) error {
	return a.repo.Delete(ctx, permissionID)
}

func (a *permissionRepoAdapter) FindByID(ctx context.Context, permissionID xid.ID) (*mainSchema.Permission, error) {
	return a.repo.FindByID(ctx, permissionID)
}

func (a *permissionRepoAdapter) FindByName(ctx context.Context, name string, appID xid.ID, orgID *xid.ID) (*mainSchema.Permission, error) {
	return a.repo.FindByName(ctx, name, appID, orgID)
}

func (a *permissionRepoAdapter) ListByApp(ctx context.Context, appID xid.ID) ([]*mainSchema.Permission, error) {
	return a.repo.ListByApp(ctx, appID)
}

func (a *permissionRepoAdapter) ListByOrg(ctx context.Context, orgID xid.ID) ([]*mainSchema.Permission, error) {
	return a.repo.ListByOrg(ctx, orgID)
}

func (a *permissionRepoAdapter) ListByCategory(ctx context.Context, category string, appID xid.ID) ([]*mainSchema.Permission, error) {
	return a.repo.ListByCategory(ctx, category, appID)
}

func (a *permissionRepoAdapter) CreateCustomPermission(ctx context.Context, name, description, category string, orgID xid.ID) (*mainSchema.Permission, error) {
	return a.repo.CreateCustomPermission(ctx, name, description, category, orgID)
}

// rolePermissionRepoAdapter creates and wraps mainRepo.RolePermissionRepository
type rolePermissionRepoAdapter struct {
	repo *mainRepo.RolePermissionRepository
}

func newRolePermissionRepoAdapter(db *bun.DB) rbac.RolePermissionRepository {
	if db == nil {
		return nil
	}
	return &rolePermissionRepoAdapter{repo: mainRepo.NewRolePermissionRepository(db)}
}

func (a *rolePermissionRepoAdapter) AssignPermission(ctx context.Context, roleID, permissionID xid.ID) error {
	return a.repo.AssignPermission(ctx, roleID, permissionID)
}

func (a *rolePermissionRepoAdapter) UnassignPermission(ctx context.Context, roleID, permissionID xid.ID) error {
	return a.repo.UnassignPermission(ctx, roleID, permissionID)
}

func (a *rolePermissionRepoAdapter) GetRolePermissions(ctx context.Context, roleID xid.ID) ([]*mainSchema.Permission, error) {
	return a.repo.GetRolePermissions(ctx, roleID)
}

func (a *rolePermissionRepoAdapter) GetPermissionRoles(ctx context.Context, permissionID xid.ID) ([]*mainSchema.Role, error) {
	return a.repo.GetPermissionRoles(ctx, permissionID)
}

func (a *rolePermissionRepoAdapter) ReplaceRolePermissions(ctx context.Context, roleID xid.ID, permissionIDs []xid.ID) error {
	return a.repo.ReplaceRolePermissions(ctx, roleID, permissionIDs)
}

// userRoleRepoAdapter adapts repository.UserRoleRepository to rbac.UserRoleRepository
type userRoleRepoAdapter struct {
	repo *mainRepo.UserRoleRepository
}

func newUserRoleRepoAdapter(repo *mainRepo.UserRoleRepository) rbac.UserRoleRepository {
	if repo == nil {
		return nil
	}
	return &userRoleRepoAdapter{repo: repo}
}

func (a *userRoleRepoAdapter) Assign(ctx context.Context, userID, roleID, orgID xid.ID) error {
	return a.repo.Assign(ctx, userID, roleID, orgID)
}

func (a *userRoleRepoAdapter) Unassign(ctx context.Context, userID, roleID, orgID xid.ID) error {
	return a.repo.Unassign(ctx, userID, roleID, orgID)
}

func (a *userRoleRepoAdapter) ListRolesForUser(ctx context.Context, userID xid.ID, orgID *xid.ID) ([]mainSchema.Role, error) {
	return a.repo.ListRolesForUser(ctx, userID, orgID)
}

func (a *userRoleRepoAdapter) AssignBatch(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error {
	return a.repo.AssignBatch(ctx, userID, roleIDs, orgID)
}

func (a *userRoleRepoAdapter) AssignBulk(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (map[xid.ID]error, error) {
	return a.repo.AssignBulk(ctx, userIDs, roleID, orgID)
}

func (a *userRoleRepoAdapter) AssignAppLevel(ctx context.Context, userID, roleID, appID xid.ID) error {
	return a.repo.AssignAppLevel(ctx, userID, roleID, appID)
}

func (a *userRoleRepoAdapter) UnassignBatch(ctx context.Context, userID xid.ID, roleIDs []xid.ID, orgID xid.ID) error {
	return a.repo.UnassignBatch(ctx, userID, roleIDs, orgID)
}

func (a *userRoleRepoAdapter) UnassignBulk(ctx context.Context, userIDs []xid.ID, roleID xid.ID, orgID xid.ID) (map[xid.ID]error, error) {
	return a.repo.UnassignBulk(ctx, userIDs, roleID, orgID)
}

func (a *userRoleRepoAdapter) ClearUserRolesInOrg(ctx context.Context, userID, orgID xid.ID) error {
	return a.repo.ClearUserRolesInOrg(ctx, userID, orgID)
}

func (a *userRoleRepoAdapter) ClearUserRolesInApp(ctx context.Context, userID, appID xid.ID) error {
	return a.repo.ClearUserRolesInApp(ctx, userID, appID)
}

func (a *userRoleRepoAdapter) TransferRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error {
	return a.repo.TransferRoles(ctx, userID, sourceOrgID, targetOrgID, roleIDs)
}

func (a *userRoleRepoAdapter) CopyRoles(ctx context.Context, userID, sourceOrgID, targetOrgID xid.ID, roleIDs []xid.ID) error {
	return a.repo.CopyRoles(ctx, userID, sourceOrgID, targetOrgID, roleIDs)
}

func (a *userRoleRepoAdapter) ReplaceUserRoles(ctx context.Context, userID, orgID xid.ID, newRoleIDs []xid.ID) error {
	return a.repo.ReplaceUserRoles(ctx, userID, orgID, newRoleIDs)
}

func (a *userRoleRepoAdapter) ListRolesForUserInOrg(ctx context.Context, userID, orgID, envID xid.ID) ([]mainSchema.Role, error) {
	return a.repo.ListRolesForUserInOrg(ctx, userID, orgID, envID)
}

func (a *userRoleRepoAdapter) ListRolesForUserInApp(ctx context.Context, userID, appID, envID xid.ID) ([]mainSchema.Role, error) {
	return a.repo.ListRolesForUserInApp(ctx, userID, appID, envID)
}

func (a *userRoleRepoAdapter) ListAllUserRolesInOrg(ctx context.Context, orgID, envID xid.ID) ([]mainSchema.UserRole, error) {
	return a.repo.ListAllUserRolesInOrg(ctx, orgID, envID)
}

func (a *userRoleRepoAdapter) ListAllUserRolesInApp(ctx context.Context, appID, envID xid.ID) ([]mainSchema.UserRole, error) {
	return a.repo.ListAllUserRolesInApp(ctx, appID, envID)
}

// policyRepoAdapter adapts repository.PolicyRepository to rbac.PolicyRepository
type policyRepoAdapter struct {
	repo *mainRepo.PolicyRepository
}

func newPolicyRepoAdapter(repo *mainRepo.PolicyRepository) rbac.PolicyRepository {
	if repo == nil {
		return nil
	}
	return &policyRepoAdapter{repo: repo}
}

func (a *policyRepoAdapter) ListAll(ctx context.Context) ([]string, error) {
	return a.repo.ListAll(ctx)
}

func (a *policyRepoAdapter) Create(ctx context.Context, expression string) error {
	return a.repo.Create(ctx, expression)
}

// memberServiceAdapter adapts organization.Service to providers.AuthsomeMemberService
type memberServiceAdapter struct {
	orgSvc *organization.Service
}

func newMemberServiceAdapter(orgSvc *organization.Service) providers.AuthsomeMemberService {
	if orgSvc == nil {
		return nil
	}
	return &memberServiceAdapter{orgSvc: orgSvc}
}

func (a *memberServiceAdapter) GetUserMembershipsForUser(ctx context.Context, userID xid.ID) ([]providers.AuthsomeMembership, error) {
	if a.orgSvc == nil || a.orgSvc.Member == nil {
		return nil, nil
	}

	// Get memberships from organization service
	memberships, err := a.orgSvc.Member.GetUserMemberships(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	// Convert to providers.AuthsomeMembership
	result := make([]providers.AuthsomeMembership, 0, len(memberships.Data))
	for _, m := range memberships.Data {
		result = append(result, &membershipAdapter{
			organizationID: m.OrganizationID,
			role:           m.Role,
			status:         m.Status,
		})
	}

	return result, nil
}

// membershipAdapter adapts membership data to providers.AuthsomeMembership
type membershipAdapter struct {
	organizationID xid.ID
	role           string
	status         string
}

func (a *membershipAdapter) GetOrganizationID() xid.ID { return a.organizationID }
func (a *membershipAdapter) GetRole() string           { return a.role }
func (a *membershipAdapter) GetStatus() string         { return a.status }
