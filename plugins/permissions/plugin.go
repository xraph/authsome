package permissions

import (
	"context"
	"fmt"

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
	api.POST("/policies", p.handler.CreatePolicy)
	api.GET("/policies", p.handler.ListPolicies)
	api.GET("/policies/:id", p.handler.GetPolicy)
	api.PUT("/policies/:id", p.handler.UpdatePolicy)
	api.DELETE("/policies/:id", p.handler.DeletePolicy)
	api.POST("/policies/validate", p.handler.ValidatePolicy)
	api.POST("/policies/test", p.handler.TestPolicy)

	// Resource management
	api.POST("/resources", p.handler.CreateResource)
	api.GET("/resources", p.handler.ListResources)
	api.GET("/resources/:id", p.handler.GetResource)
	api.DELETE("/resources/:id", p.handler.DeleteResource)

	// Action management
	api.POST("/actions", p.handler.CreateAction)
	api.GET("/actions", p.handler.ListActions)
	api.DELETE("/actions/:id", p.handler.DeleteAction)

	// Namespace management
	api.POST("/namespaces", p.handler.CreateNamespace)
	api.GET("/namespaces", p.handler.ListNamespaces)
	api.GET("/namespaces/:id", p.handler.GetNamespace)
	api.PUT("/namespaces/:id", p.handler.UpdateNamespace)
	api.DELETE("/namespaces/:id", p.handler.DeleteNamespace)

	// Evaluation endpoint (primary authorization check)
	api.POST("/evaluate", p.handler.Evaluate)
	api.POST("/evaluate/batch", p.handler.EvaluateBatch)

	// Policy templates
	api.GET("/templates", p.handler.ListTemplates)
	api.GET("/templates/:id", p.handler.GetTemplate)
	api.POST("/templates/:id/instantiate", p.handler.InstantiateTemplate)

	// Migration from RBAC
	api.POST("/migrate/rbac", p.handler.MigrateFromRBAC)
	api.GET("/migrate/rbac/status", p.handler.GetMigrationStatus)

	// Audit & reporting
	api.GET("/audit", p.handler.GetAuditLog)
	api.GET("/analytics", p.handler.GetAnalytics)

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
