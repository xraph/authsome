package scim

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/forge"
)

const (
	PluginID      = "scim"
	PluginName    = "SCIM 2.0 Provisioning"
	PluginVersion = "1.0.0"
)

// Plugin implements the SCIM 2.0 provisioning plugin for enterprise identity providers
type Plugin struct {
	config  *Config
	service *Service
	handler *Handler
	
	// Dependencies
	db             *bun.DB
	userService    *user.Service
	orgService     *organization.Service
	auditService   *audit.Service
	webhookService *webhook.Service
}

// NewPlugin creates a new SCIM plugin instance
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the unique plugin identifier
func (p *Plugin) ID() string {
	return PluginID
}

// Name returns the human-readable plugin name (optional, for dashboard display)
func (p *Plugin) Name() string {
	return PluginName
}

// Version returns the plugin version (optional, for compatibility checks)
func (p *Plugin) Version() string {
	return PluginVersion
}

// Description returns the plugin description (optional, for documentation)
func (p *Plugin) Description() string {
	return "Enterprise-grade SCIM 2.0 provisioning for automated user/group sync with Okta, Azure AD, OneLogin, and other identity providers"
}

// Init initializes the plugin with dependencies from the Auth instance
func (p *Plugin) Init(auth interface{}) error {
	// Extract service registry using interface methods
	type serviceRegistryGetter interface {
		GetServiceRegistry() *registry.ServiceRegistry
		GetDB() *bun.DB
	}
	
	srGetter, ok := auth.(serviceRegistryGetter)
	if !ok {
		return fmt.Errorf("SCIM plugin requires auth instance with GetServiceRegistry and GetDB")
	}
	
	serviceRegistry := srGetter.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}
	
	p.db = srGetter.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available")
	}
	
	// Get required services from registry
	userSvcInterface := serviceRegistry.UserService()
	if userSvcInterface == nil {
		return fmt.Errorf("user service not found in registry")
	}
	var convOk bool
	p.userService, convOk = userSvcInterface.(*user.Service)
	if !convOk {
		return fmt.Errorf("invalid user service type")
	}
	
	// Get organization service (required for multi-tenancy)
	orgSvcInterface := serviceRegistry.OrganizationService()
	if orgSvcInterface == nil {
		return fmt.Errorf("organization service not found in registry")
	}
	p.orgService, convOk = orgSvcInterface.(*organization.Service)
	if !convOk {
		return fmt.Errorf("invalid organization service type")
	}
	
	// Get audit service
	p.auditService = serviceRegistry.AuditService()
	if p.auditService == nil {
		return fmt.Errorf("audit service not found in registry")
	}
	
	// Get webhook service
	p.webhookService = serviceRegistry.WebhookService()
	if p.webhookService == nil {
		return fmt.Errorf("webhook service not found in registry")
	}
	
	// Load configuration
	p.config = DefaultConfig()
	
	// TODO: Load from config manager when registry supports ConfigManager()
	// For now, use default configuration
	// Future: Implement config loading like:
	// if configManager := serviceRegistry.ConfigManager(); configManager != nil {
	//     var cfg Config
	//     if err := configManager.Bind("auth.plugins.scim", &cfg); err == nil {
	//         p.config = cfg
	//     }
	// }
	
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("invalid SCIM config: %w", err)
	}
	
	// Initialize repository
	repo := NewRepository(p.db)
	
	// Initialize service
	p.service = NewService(ServiceConfig{
		Config:         p.config,
		Repository:     repo,
		UserService:    p.userService,
		OrgService:     p.orgService,
		AuditService:   p.auditService,
		WebhookService: p.webhookService,
	})
	
	// Initialize handler
	p.handler = NewHandler(p.service, p.config)
	
	fmt.Println("[SCIM] Plugin initialized successfully")
	return nil
}

// RegisterRoutes registers SCIM 2.0 compliant HTTP routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("SCIM handler not initialized; call Init first")
	}
	
	// Create middleware chain for SCIM endpoints (auth, org resolution, rate limiting)
	scimChain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.AuthMiddleware()(p.OrgResolutionMiddleware()(p.RateLimitMiddleware()(h)))
	}
	
	// SCIM 2.0 base path as per RFC 7644
	scimGroup := router.Group("/scim/v2")
	
	// Service Provider Configuration (RFC 7643 Section 5)
	scimGroup.GET("/ServiceProviderConfig", scimChain(p.handler.GetServiceProviderConfig))
	
	// Resource Types (RFC 7643 Section 6)
	scimGroup.GET("/ResourceTypes", scimChain(p.handler.GetResourceTypes))
	scimGroup.GET("/ResourceTypes/:id", scimChain(p.handler.GetResourceType))
	
	// Schemas (RFC 7643 Section 7)
	scimGroup.GET("/Schemas", scimChain(p.handler.GetSchemas))
	scimGroup.GET("/Schemas/:id", scimChain(p.handler.GetSchema))
	
	// Users endpoint (RFC 7644 Section 3)
	scimGroup.POST("/Users", scimChain(p.handler.CreateUser))
	scimGroup.GET("/Users", scimChain(p.handler.ListUsers))
	scimGroup.GET("/Users/:id", scimChain(p.handler.GetUser))
	scimGroup.PUT("/Users/:id", scimChain(p.handler.ReplaceUser))
	scimGroup.PATCH("/Users/:id", scimChain(p.handler.UpdateUser))
	scimGroup.DELETE("/Users/:id", scimChain(p.handler.DeleteUser))
	
	// Groups endpoint (RFC 7644 Section 3)
	scimGroup.POST("/Groups", scimChain(p.handler.CreateGroup))
	scimGroup.GET("/Groups", scimChain(p.handler.ListGroups))
	scimGroup.GET("/Groups/:id", scimChain(p.handler.GetGroup))
	scimGroup.PUT("/Groups/:id", scimChain(p.handler.ReplaceGroup))
	scimGroup.PATCH("/Groups/:id", scimChain(p.handler.UpdateGroup))
	scimGroup.DELETE("/Groups/:id", scimChain(p.handler.DeleteGroup))
	
	// Bulk operations (RFC 7644 Section 3.7)
	scimGroup.POST("/Bulk", scimChain(p.handler.BulkOperation))
	
	// Search endpoint (RFC 7644 Section 3.4.3)
	scimGroup.POST("/.search", scimChain(p.handler.Search))
	
	// Create admin middleware chain (SCIM auth + admin check)
	adminChain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.AuthMiddleware()(p.OrgResolutionMiddleware()(p.RequireAdminMiddleware()(h)))
	}
	
	// Custom endpoints for provisioning management (non-standard)
	adminGroup := router.Group("/api/scim-admin")
	
	// Token management
	adminGroup.POST("/tokens", adminChain(p.handler.CreateProvisioningToken))
	adminGroup.GET("/tokens", adminChain(p.handler.ListProvisioningTokens))
	adminGroup.DELETE("/tokens/:id", adminChain(p.handler.RevokeProvisioningToken))
	
	// Attribute mapping configuration
	adminGroup.GET("/mappings", adminChain(p.handler.GetAttributeMappings))
	adminGroup.PUT("/mappings", adminChain(p.handler.UpdateAttributeMappings))
	
	// Provisioning logs and audit
	adminGroup.GET("/logs", adminChain(p.handler.GetProvisioningLogs))
	adminGroup.GET("/stats", adminChain(p.handler.GetProvisioningStats))
	
	fmt.Println("[SCIM] Routes registered successfully")
	fmt.Println("  - POST   /scim/v2/Users (create user)")
	fmt.Println("  - GET    /scim/v2/Users (list users)")
	fmt.Println("  - GET    /scim/v2/Users/:id (get user)")
	fmt.Println("  - PUT    /scim/v2/Users/:id (replace user)")
	fmt.Println("  - PATCH  /scim/v2/Users/:id (update user)")
	fmt.Println("  - DELETE /scim/v2/Users/:id (delete user)")
	fmt.Println("  - POST   /scim/v2/Groups (create group)")
	fmt.Println("  - GET    /scim/v2/Groups (list groups)")
	fmt.Println("  - POST   /scim/v2/Bulk (bulk operations)")
	
	return nil
}

// RegisterHooks registers lifecycle hooks for SCIM events
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	// Hook into user lifecycle to send provisioning webhooks
	hooks.RegisterAfterUserCreate(p.handleUserCreated)
	hooks.RegisterAfterUserUpdate(p.handleUserUpdated)
	hooks.RegisterAfterUserDelete(p.handleUserDeleted)
	
	// Hook into organization creation to set up default SCIM config
	hooks.RegisterAfterOrganizationCreate(p.handleOrganizationCreated)
	
	return nil
}

// RegisterServiceDecorators allows SCIM plugin to enhance core services
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// SCIM plugin doesn't need to decorate core services
	// It operates alongside them
	return nil
}

// Migrate runs database migrations for SCIM-specific tables
func (p *Plugin) Migrate() error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}
	
	ctx := context.Background()
	return p.service.Migrate(ctx)
}

// Hook handlers

func (p *Plugin) handleUserCreated(ctx context.Context, u *user.User) error {
	// Send provisioning webhook when user is created via SCIM
	// Note: In production, you'd want to check if this user was SCIM-provisioned
	// by querying the SCIM repository for a matching external_id
	return p.service.SendProvisioningWebhook(ctx, "user.created", map[string]interface{}{
		"user_id": u.ID.String(),
		"email":   u.Email,
		"source":  "scim",
	})
}

func (p *Plugin) handleUserUpdated(ctx context.Context, u *user.User) error {
	// Send provisioning webhook when user is updated via SCIM
	return p.service.SendProvisioningWebhook(ctx, "user.updated", map[string]interface{}{
		"user_id": u.ID.String(),
		"email":   u.Email,
		"source":  "scim",
	})
}

func (p *Plugin) handleUserDeleted(ctx context.Context, userID xid.ID) error {
	// Send provisioning webhook when user is deleted via SCIM
	return p.service.SendProvisioningWebhook(ctx, "user.deleted", map[string]interface{}{
		"user_id": userID.String(),
		"source":  "scim",
	})
}

func (p *Plugin) handleOrganizationCreated(ctx context.Context, org interface{}) error {
	// Initialize default SCIM configuration for new organization
	// The org parameter is of type interface{} to match the hook signature
	// In production, you'd extract the org ID and initialize SCIM config
	// For now, we just return nil to avoid errors
	return nil
}

// Service returns the SCIM service for programmatic access
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

