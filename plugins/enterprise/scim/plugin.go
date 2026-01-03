package scim

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
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
	userService    user.ServiceInterface // Use interface to support decorated services
	orgService     interface{}           // Use interface{} to support both core and multitenancy org services
	auditService   *audit.Service
	webhookService *webhook.Service
	
	// Dashboard extension
	dashboardExt *DashboardExtension
	
	// Organization UI extension
	orgUIExt *OrganizationUIExtension
}

// PluginOption is a functional option for configuring the SCIM plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg *Config) PluginOption {
	return func(p *Plugin) {
		p.config = cfg
	}
}

// WithAuthMethod sets the authentication method (bearer or oauth2)
func WithAuthMethod(method string) PluginOption {
	return func(p *Plugin) {
		if p.config == nil {
			p.config = DefaultConfig()
		}
		p.config.AuthMethod = method
	}
}

// WithRateLimit configures rate limiting
func WithRateLimit(enabled bool, requestsPerMin, burstSize int) PluginOption {
	return func(p *Plugin) {
		if p.config == nil {
			p.config = DefaultConfig()
		}
		p.config.RateLimit.Enabled = enabled
		p.config.RateLimit.RequestsPerMin = requestsPerMin
		p.config.RateLimit.BurstSize = burstSize
	}
}

// WithUserProvisioning configures user provisioning behavior
func WithUserProvisioning(autoActivate, sendWelcomeEmail, preventDuplicates bool, defaultRole string) PluginOption {
	return func(p *Plugin) {
		if p.config == nil {
			p.config = DefaultConfig()
		}
		p.config.UserProvisioning.AutoActivate = autoActivate
		p.config.UserProvisioning.SendWelcomeEmail = sendWelcomeEmail
		p.config.UserProvisioning.PreventDuplicates = preventDuplicates
		p.config.UserProvisioning.DefaultRole = defaultRole
	}
}

// WithGroupSync configures group synchronization
func WithGroupSync(enabled, syncToTeams, syncToRoles, createMissing bool) PluginOption {
	return func(p *Plugin) {
		if p.config == nil {
			p.config = DefaultConfig()
		}
		p.config.GroupSync.Enabled = enabled
		p.config.GroupSync.SyncToTeams = syncToTeams
		p.config.GroupSync.SyncToRoles = syncToRoles
		p.config.GroupSync.CreateMissingGroups = createMissing
	}
}

// WithJITProvisioning configures Just-In-Time provisioning
func WithJITProvisioning(enabled, createOnFirstLogin, updateOnLogin bool) PluginOption {
	return func(p *Plugin) {
		if p.config == nil {
			p.config = DefaultConfig()
		}
		p.config.JITProvisioning.Enabled = enabled
		p.config.JITProvisioning.CreateOnFirstLogin = createOnFirstLogin
		p.config.JITProvisioning.UpdateOnLogin = updateOnLogin
	}
}

// WithWebhooks configures provisioning event webhooks
func WithWebhooks(enabled bool, urls []string, retryAttempts int) PluginOption {
	return func(p *Plugin) {
		if p.config == nil {
			p.config = DefaultConfig()
		}
		p.config.Webhooks.Enabled = enabled
		p.config.Webhooks.WebhookURLs = urls
		p.config.Webhooks.RetryAttempts = retryAttempts
	}
}

// WithBulkOperations configures bulk operation limits
func WithBulkOperations(enabled bool, maxOps, maxPayloadBytes int) PluginOption {
	return func(p *Plugin) {
		if p.config == nil {
			p.config = DefaultConfig()
		}
		p.config.BulkOperations.Enabled = enabled
		p.config.BulkOperations.MaxOperations = maxOps
		p.config.BulkOperations.MaxPayloadBytes = maxPayloadBytes
	}
}

// WithSecurity configures security settings
func WithSecurity(requireHTTPS, auditAll, maskSensitive bool, ipWhitelist []string) PluginOption {
	return func(p *Plugin) {
		if p.config == nil {
			p.config = DefaultConfig()
		}
		p.config.Security.RequireHTTPS = requireHTTPS
		p.config.Security.AuditAllOperations = auditAll
		p.config.Security.MaskSensitiveData = maskSensitive
		p.config.Security.IPWhitelist = ipWhitelist
	}
}

// NewPlugin creates a new SCIM plugin instance
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		config: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	// Initialize UI extensions early so they're available during plugin scanning
	// These don't depend on any runtime services, just the plugin reference
	p.orgUIExt = NewOrganizationUIExtension(p)
	p.dashboardExt = NewDashboardExtension(p)

	return p
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
func (p *Plugin) Init(auth core.Authsome) error {
	// Get service registry and database from auth instance
	serviceRegistry := auth.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}

	p.db = auth.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for SCIM plugin - ensure database is properly initialized before authsome")
	}

	// Get required services from registry
	p.userService = serviceRegistry.UserService()
	if p.userService == nil {
		return fmt.Errorf("user service not found in registry")
	}

	// Get organization/app service (required for SCIM)
	// Can be either:
	// - *app.Service (from multitenancy plugin) - App mode
	// - *organization.Service (from organization plugin) - Organization mode
	p.orgService = serviceRegistry.OrganizationService()
	if p.orgService == nil {
		return fmt.Errorf("organization or app service not found in registry - SCIM requires multitenancy or organization plugin")
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

	// UI extensions (orgUIExt and dashboardExt) are already initialized in NewPlugin()
	// to ensure they're available during plugin scanning, regardless of init order

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
	scimGroup.GET("/ServiceProviderConfig", scimChain(p.handler.GetServiceProviderConfig),
		forge.WithName("scim.serviceprovider.config"),
		forge.WithSummary("Get service provider configuration"),
		forge.WithDescription("Returns SCIM 2.0 service provider configuration including supported features, authentication schemes, and capabilities"),
		forge.WithResponseSchema(200, "Service provider configuration", ServiceProviderConfig{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("SCIM", "Configuration"),
	)

	// Resource Types (RFC 7643 Section 6)
	scimGroup.GET("/ResourceTypes", scimChain(p.handler.GetResourceTypes),
		forge.WithName("scim.resourcetypes.list"),
		forge.WithSummary("List resource types"),
		forge.WithDescription("Returns all supported SCIM resource types (User, Group) with their schemas and endpoints"),
		forge.WithResponseSchema(200, "List of resource types", ListResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("SCIM", "ResourceTypes"),
	)

	scimGroup.GET("/ResourceTypes/:id", scimChain(p.handler.GetResourceType),
		forge.WithName("scim.resourcetypes.get"),
		forge.WithSummary("Get resource type"),
		forge.WithDescription("Returns details for a specific SCIM resource type (User or Group)"),
		forge.WithResponseSchema(200, "Resource type details", ResourceType{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "Resource type not found", ErrorResponse{}),
		forge.WithTags("SCIM", "ResourceTypes"),
	)

	// Schemas (RFC 7643 Section 7)
	scimGroup.GET("/Schemas", scimChain(p.handler.GetSchemas),
		forge.WithName("scim.schemas.list"),
		forge.WithSummary("List schemas"),
		forge.WithDescription("Returns all supported SCIM schemas including core user schema, enterprise extension, and group schema"),
		forge.WithResponseSchema(200, "List of schemas", ListResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("SCIM", "Schemas"),
	)

	scimGroup.GET("/Schemas/:id", scimChain(p.handler.GetSchema),
		forge.WithName("scim.schemas.get"),
		forge.WithSummary("Get schema"),
		forge.WithDescription("Returns detailed schema definition for a specific SCIM schema ID"),
		forge.WithResponseSchema(200, "Schema definition", Schema{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "Schema not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Schemas"),
	)

	// Users endpoint (RFC 7644 Section 3)
	scimGroup.POST("/Users", scimChain(p.handler.CreateUser),
		forge.WithName("scim.users.create"),
		forge.WithSummary("Create user"),
		forge.WithDescription("Creates a new user via SCIM 2.0 provisioning. Supports core user attributes and enterprise extension"),
		forge.WithRequestSchema(SCIMUser{}),
		forge.WithResponseSchema(201, "User created", SCIMUser{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(409, "User already exists", ErrorResponse{}),
		forge.WithTags("SCIM", "Users"),
		forge.WithValidation(true),
	)

	scimGroup.GET("/Users", scimChain(p.handler.ListUsers),
		forge.WithName("scim.users.list"),
		forge.WithSummary("List users"),
		forge.WithDescription("Lists users with filtering, sorting, and pagination support. Supports SCIM filter syntax"),
		forge.WithResponseSchema(200, "List of users", ListResponse{}),
		forge.WithResponseSchema(400, "Invalid filter", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("SCIM", "Users"),
	)

	scimGroup.GET("/Users/:id", scimChain(p.handler.GetUser),
		forge.WithName("scim.users.get"),
		forge.WithSummary("Get user"),
		forge.WithDescription("Retrieves a specific user by SCIM ID with all attributes and extensions"),
		forge.WithResponseSchema(200, "User details", SCIMUser{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Users"),
	)

	scimGroup.PUT("/Users/:id", scimChain(p.handler.ReplaceUser),
		forge.WithName("scim.users.replace"),
		forge.WithSummary("Replace user"),
		forge.WithRequestSchema(SCIMUser{}),
		forge.WithDescription("Replaces all user attributes with the provided values (full update)"),
		forge.WithResponseSchema(200, "User updated", SCIMUser{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Users"),
		forge.WithValidation(true),
	)

	scimGroup.PATCH("/Users/:id", scimChain(p.handler.UpdateUser),
		forge.WithName("scim.users.update"),
		forge.WithSummary("Update user"),
		forge.WithRequestSchema(PatchOp{}),
		forge.WithDescription("Partially updates user attributes using SCIM PATCH operations (add, remove, replace)"),
		forge.WithResponseSchema(200, "User updated", SCIMUser{}),
		forge.WithResponseSchema(400, "Invalid patch operation", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Users"),
		forge.WithValidation(true),
	)

	scimGroup.DELETE("/Users/:id", scimChain(p.handler.DeleteUser),
		forge.WithName("scim.users.delete"),
		forge.WithSummary("Delete user"),
		forge.WithDescription("Deletes a user by SCIM ID. User is soft-deleted and can be restored if configured"),
		forge.WithResponseSchema(204, "User deleted", nil),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Users"),
	)

	// Groups endpoint (RFC 7644 Section 3)
	scimGroup.POST("/Groups", scimChain(p.handler.CreateGroup),
		forge.WithName("scim.groups.create"),
		forge.WithSummary("Create group"),
		forge.WithRequestSchema(SCIMGroup{}),
		forge.WithDescription("Creates a new group via SCIM 2.0 provisioning with optional member references"),
		forge.WithResponseSchema(201, "Group created", SCIMGroup{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(409, "Group already exists", ErrorResponse{}),
		forge.WithTags("SCIM", "Groups"),
		forge.WithValidation(true),
	)

	scimGroup.GET("/Groups", scimChain(p.handler.ListGroups),
		forge.WithName("scim.groups.list"),
		forge.WithSummary("List groups"),
		forge.WithDescription("Lists groups with filtering, sorting, and pagination support. Supports SCIM filter syntax"),
		forge.WithResponseSchema(200, "List of groups", ListResponse{}),
		forge.WithResponseSchema(400, "Invalid filter", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("SCIM", "Groups"),
	)

	scimGroup.GET("/Groups/:id", scimChain(p.handler.GetGroup),
		forge.WithName("scim.groups.get"),
		forge.WithSummary("Get group"),
		forge.WithDescription("Retrieves a specific group by SCIM ID with all members and attributes"),
		forge.WithResponseSchema(200, "Group details", SCIMGroup{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "Group not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Groups"),
	)

	scimGroup.PUT("/Groups/:id", scimChain(p.handler.ReplaceGroup),
		forge.WithName("scim.groups.replace"),
		forge.WithSummary("Replace group"),
		forge.WithRequestSchema(SCIMGroup{}),
		forge.WithDescription("Replaces all group attributes and members with the provided values (full update)"),
		forge.WithResponseSchema(200, "Group updated", SCIMGroup{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "Group not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Groups"),
		forge.WithValidation(true),
	)

	scimGroup.PATCH("/Groups/:id", scimChain(p.handler.UpdateGroup),
		forge.WithName("scim.groups.update"),
		forge.WithSummary("Update group"),
		forge.WithRequestSchema(PatchOp{}),
		forge.WithDescription("Partially updates group attributes and members using SCIM PATCH operations (add, remove, replace)"),
		forge.WithResponseSchema(200, "Group updated", SCIMGroup{}),
		forge.WithResponseSchema(400, "Invalid patch operation", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "Group not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Groups"),
		forge.WithValidation(true),
	)

	scimGroup.DELETE("/Groups/:id", scimChain(p.handler.DeleteGroup),
		forge.WithName("scim.groups.delete"),
		forge.WithSummary("Delete group"),
		forge.WithDescription("Deletes a group by SCIM ID. Group memberships are preserved but group is removed"),
		forge.WithResponseSchema(204, "Group deleted", nil),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "Group not found", ErrorResponse{}),
		forge.WithTags("SCIM", "Groups"),
	)

	// Bulk operations (RFC 7644 Section 3.7)
	scimGroup.POST("/Bulk", scimChain(p.handler.BulkOperation),
		forge.WithName("scim.bulk.operation"),
		forge.WithSummary("Bulk operations"),
		forge.WithRequestSchema(BulkRequest{}),
		forge.WithDescription("Performs multiple SCIM operations in a single request. Supports create, update, and delete operations on users and groups"),
		forge.WithResponseSchema(200, "Bulk operation results", BulkResponse{}),
		forge.WithResponseSchema(400, "Invalid bulk request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(413, "Request too large", ErrorResponse{}),
		forge.WithTags("SCIM", "Bulk"),
		forge.WithValidation(true),
	)

	// Search endpoint (RFC 7644 Section 3.4.3)
	scimGroup.POST("/.search", scimChain(p.handler.Search),
		forge.WithName("scim.search"),
		forge.WithSummary("Search resources"),
		forge.WithRequestSchema(SearchRequest{}),
		forge.WithDescription("Advanced search endpoint for users and groups with complex filter expressions and pagination"),
		forge.WithResponseSchema(200, "Search results", ListResponse{}),
		forge.WithResponseSchema(400, "Invalid search request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("SCIM", "Search"),
		forge.WithValidation(true),
	)

	// Create admin middleware chain (SCIM auth + admin check)
	adminChain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.AuthMiddleware()(p.OrgResolutionMiddleware()(p.RequireAdminMiddleware()(h)))
	}

	// Custom endpoints for provisioning management (non-standard)
	adminGroup := router.Group("/admin/scim")

	// Token management
	adminGroup.POST("/tokens", adminChain(p.handler.CreateProvisioningToken),
		forge.WithName("scim.admin.tokens.create"),
		forge.WithSummary("Create provisioning token"),
		forge.WithRequestSchema(CreateTokenRequest{}),
		forge.WithDescription("Creates a new SCIM provisioning token (Bearer token) for authenticating SCIM requests. Token is shown only once"),
		forge.WithResponseSchema(201, "Token created", SCIMTokenResponse{}),
		forge.WithResponseSchema(400, "Invalid request", SCIMErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", SCIMErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", SCIMErrorResponse{}),
		forge.WithTags("SCIM", "Admin", "Tokens"),
		forge.WithValidation(true),
	)

	adminGroup.GET("/tokens", adminChain(p.handler.ListProvisioningTokens),
		forge.WithName("scim.admin.tokens.list"),
		forge.WithSummary("List provisioning tokens"),
		forge.WithDescription("Lists all provisioning tokens for the organization with pagination. Token values are never returned"),
		forge.WithResponseSchema(200, "List of tokens", SCIMTokenListResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", SCIMErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", SCIMErrorResponse{}),
		forge.WithTags("SCIM", "Admin", "Tokens"),
	)

	adminGroup.DELETE("/tokens/:id", adminChain(p.handler.RevokeProvisioningToken),
		forge.WithName("scim.admin.tokens.revoke"),
		forge.WithSummary("Revoke provisioning token"),
		forge.WithDescription("Revokes a provisioning token by ID. Token can no longer be used for SCIM authentication"),
		forge.WithResponseSchema(200, "Token revoked", SCIMStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", SCIMErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", SCIMErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", SCIMErrorResponse{}),
		forge.WithResponseSchema(404, "Token not found", SCIMErrorResponse{}),
		forge.WithTags("SCIM", "Admin", "Tokens"),
	)

	// Attribute mapping configuration
	adminGroup.GET("/mappings", adminChain(p.handler.GetAttributeMappings),
		forge.WithName("scim.admin.mappings.get"),
		forge.WithSummary("Get attribute mappings"),
		forge.WithDescription("Retrieves custom attribute mappings for SCIM attributes to AuthSome fields"),
		forge.WithResponseSchema(200, "Attribute mappings", SCIMAttributeMappingsResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", SCIMErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", SCIMErrorResponse{}),
		forge.WithTags("SCIM", "Admin", "Mappings"),
	)

	adminGroup.PUT("/mappings", adminChain(p.handler.UpdateAttributeMappings),
		forge.WithName("scim.admin.mappings.update"),
		forge.WithSummary("Update attribute mappings"),
		forge.WithDescription("Updates custom attribute mappings for SCIM attributes to AuthSome fields. Used for custom field mapping"),
		forge.WithRequestSchema(UpdateAttributeMappingsRequest{}),
		forge.WithResponseSchema(200, "Mappings updated", SCIMStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", SCIMErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", SCIMErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", SCIMErrorResponse{}),
		forge.WithTags("SCIM", "Admin", "Mappings"),
		forge.WithValidation(true),
	)

	// Provisioning logs and audit
	adminGroup.GET("/logs", adminChain(p.handler.GetProvisioningLogs),
		forge.WithName("scim.admin.logs.get"),
		forge.WithSummary("Get provisioning logs"),
		forge.WithDescription("Retrieves provisioning operation logs with filtering by action, pagination, and detailed request/response data"),
		forge.WithResponseSchema(200, "Provisioning logs", SCIMLogsResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", SCIMErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", SCIMErrorResponse{}),
		forge.WithTags("SCIM", "Admin", "Logs"),
	)

	adminGroup.GET("/stats", adminChain(p.handler.GetProvisioningStats),
		forge.WithName("scim.admin.stats.get"),
		forge.WithSummary("Get provisioning statistics"),
		forge.WithDescription("Returns real-time SCIM provisioning metrics including request counts, error rates, and performance statistics"),
		forge.WithResponseSchema(200, "Provisioning statistics", SCIMStatsResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", SCIMErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", SCIMErrorResponse{}),
		forge.WithTags("SCIM", "Admin", "Stats"),
	)


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

// Response types for admin endpoints (for API documentation)

// SCIMErrorResponse represents an error response for admin endpoints
type SCIMErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

// SCIMStatusResponse represents a status response
type SCIMStatusResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// SCIMTokenResponse represents a token creation response
type SCIMTokenResponse struct {
	Token   string `json:"token" example:"scim_abc123"`
	ID      string `json:"id" example:"01HZ"`
	Name    string `json:"name" example:"Production SCIM Token"`
	Message string `json:"message" example:"Store this token securely"`
}

// SCIMTokenListResponse represents a list of tokens response
type SCIMTokenListResponse struct {
	Tokens []SCIMTokenInfo `json:"tokens"`
	Total  int             `json:"total" example:"5"`
	Limit  int             `json:"limit" example:"50"`
	Offset int             `json:"offset" example:"0"`
}

// SCIMTokenInfo represents token information (without sensitive data)
type SCIMTokenInfo struct {
	ID          string     `json:"id" example:"01HZ..."`
	Name        string     `json:"name" example:"Production SCIM Token"`
	Description string     `json:"description" example:"Token for Okta provisioning"`
	Scopes      []string   `json:"scopes" example:"users,groups"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
}

// SCIMAttributeMappingsResponse represents attribute mappings response
type SCIMAttributeMappingsResponse struct {
	Mappings map[string]string `json:"mappings" example:"userName:email,displayName:name"`
}

// SCIMLogsResponse represents provisioning logs response
type SCIMLogsResponse struct {
	Logs   []SCIMLogInfo `json:"logs"`
	Total  int           `json:"total" example:"100"`
	Limit  int           `json:"limit" example:"50"`
	Offset int           `json:"offset" example:"0"`
}

// SCIMLogInfo represents a single log entry
type SCIMLogInfo struct {
	ID           string    `json:"id" example:"01HZ..."`
	Operation    string    `json:"operation" example:"CREATE_USER"`
	ResourceType string    `json:"resource_type" example:"User"`
	ResourceID   string    `json:"resource_id" example:"01HZ..."`
	Method       string    `json:"method" example:"POST"`
	Path         string    `json:"path" example:"/scim/v2/Users"`
	StatusCode   int       `json:"status_code" example:"201"`
	Success      bool      `json:"success" example:"true"`
	ErrorMessage string    `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
	DurationMS   int       `json:"duration_ms" example:"45"`
}

// SCIMStatsResponse represents provisioning statistics response
type SCIMStatsResponse struct {
	SCIMMetrics map[string]interface{} `json:"scim_metrics"`
}

// DashboardExtension returns the dashboard extension for the SCIM plugin
// This allows the plugin to extend the dashboard with SCIM-specific UI
// This implements the PluginWithDashboardExtension interface
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	return p.dashboardExt
}

// Implement ui.OrganizationUIExtension interface by delegating to orgUIExt
// This allows the SCIM plugin to extend organization pages with SCIM-specific UI

func (p *Plugin) ExtensionID() string {
	if p.orgUIExt == nil {
		return ""
	}
	return p.orgUIExt.ExtensionID()
}

func (p *Plugin) OrganizationWidgets() []ui.OrganizationWidget {
	if p.orgUIExt == nil {
		return []ui.OrganizationWidget{}
	}
	return p.orgUIExt.OrganizationWidgets()
}

func (p *Plugin) OrganizationTabs() []ui.OrganizationTab {
	if p.orgUIExt == nil {
		return []ui.OrganizationTab{}
	}
	return p.orgUIExt.OrganizationTabs()
}

func (p *Plugin) OrganizationActions() []ui.OrganizationAction {
	if p.orgUIExt == nil {
		return []ui.OrganizationAction{}
	}
	return p.orgUIExt.OrganizationActions()
}

func (p *Plugin) OrganizationQuickLinks() []ui.OrganizationQuickLink {
	if p.orgUIExt == nil {
		return []ui.OrganizationQuickLink{}
	}
	return p.orgUIExt.OrganizationQuickLinks()
}

func (p *Plugin) OrganizationSettingsSections() []ui.OrganizationSettingsSection {
	if p.orgUIExt == nil {
		return []ui.OrganizationSettingsSection{}
	}
	return p.orgUIExt.OrganizationSettingsSections()
}
