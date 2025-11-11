package dashboard

import (
	"context"
	"embed"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/interfaces"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins"
	mtorg "github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

//go:embed static/css/* static/js/*.js
var assets embed.FS

// Plugin implements the dashboard plugin for AuthSome
type Plugin struct {
	handler        *Handler
	userSvc        user.ServiceInterface
	sessionSvc     session.ServiceInterface
	auditSvc       *audit.Service
	rbacSvc        *rbac.Service
	apikeyService  *apikey.Service
	orgService     *organization.Service
	isSaaSMode     bool
	permChecker    *PermissionChecker
	csrfProtector  *CSRFProtector
	basePath       string
	enabledPlugins map[string]bool
	config         Config
	defaultConfig  Config
	platformOrgID  xid.ID // Platform organization ID for context injection
	db             *bun.DB
}

// Config holds the dashboard plugin configuration
type Config struct {
	// EnableSignup allows new users to sign up for dashboard access
	EnableSignup bool `json:"enableSignup"`

	// RequireEmailVerification requires email verification for new signups
	RequireEmailVerification bool `json:"requireEmailVerification"`

	// SessionDuration sets the duration for dashboard sessions in hours
	SessionDuration int `json:"sessionDuration"`

	// MaxLoginAttempts sets the maximum login attempts before lockout
	MaxLoginAttempts int `json:"maxLoginAttempts"`

	// LockoutDuration sets the lockout duration in minutes
	LockoutDuration int `json:"lockoutDuration"`

	// DefaultTheme sets the default theme (light, dark, auto)
	DefaultTheme string `json:"defaultTheme"`
}

// PluginOption is a functional option for configuring the dashboard plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithEnableSignup sets whether signup is enabled
func WithEnableSignup(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.EnableSignup = enabled
	}
}

// WithRequireEmailVerification sets whether email verification is required
func WithRequireEmailVerification(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireEmailVerification = required
	}
}

// WithSessionDuration sets the session duration in hours
func WithSessionDuration(hours int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SessionDuration = hours
	}
}

// WithMaxLoginAttempts sets the maximum login attempts
func WithMaxLoginAttempts(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxLoginAttempts = max
	}
}

// WithLockoutDuration sets the lockout duration in minutes
func WithLockoutDuration(minutes int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.LockoutDuration = minutes
	}
}

// WithDefaultTheme sets the default theme
func WithDefaultTheme(theme string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DefaultTheme = theme
	}
}

// NewPlugin creates a new dashboard plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: Config{
			EnableSignup:             true,
			RequireEmailVerification: false,
			SessionDuration:          24, // 24 hours
			MaxLoginAttempts:         5,
			LockoutDuration:          15, // 15 minutes
			DefaultTheme:             "auto",
		},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ID returns the unique identifier for this plugin
func (p *Plugin) ID() string {
	return "dashboard"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(dep interface{}) error {
	// Try to extract required interfaces from auth instance
	type authInstanceInterface interface {
		GetDB() *bun.DB
		GetServiceRegistry() *registry.ServiceRegistry
		GetHookRegistry() *hooks.HookRegistry
		GetBasePath() string
		GetPluginRegistry() *plugins.Registry
		GetForgeApp() forge.App
	}

	authInstance, ok := dep.(authInstanceInterface)
	if !ok {
		return fmt.Errorf("dashboard plugin requires auth instance with GetDB, GetServiceRegistry, GetHookRegistry, GetBasePath, GetPluginRegistry, and GetForgeApp methods")
	}

	// Get Forge app and config manager
	forgeApp := authInstance.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available")
	}
	configManager := forgeApp.Config()

	// Bind plugin configuration using Forge ConfigManager with provided defaults
	if err := configManager.BindWithDefault("auth.dashboard", &p.config, p.defaultConfig); err != nil {
		// Log but don't fail - use defaults
		fmt.Printf("[Dashboard] Warning: failed to bind dashboard config: %v\n", err)
		// Fall back to default config
		p.config = p.defaultConfig
	}

	serviceRegistry := authInstance.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}

	hookRegistry := authInstance.GetHookRegistry()
	if hookRegistry == nil {
		return fmt.Errorf("hook registry not available")
	}

	// Get database for repository initialization
	db := authInstance.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	p.db = db

	// Get base path (e.g., "/api/auth")
	p.basePath = authInstance.GetBasePath()
	if p.basePath == "" {
		p.basePath = ""
	}

	// Get plugin registry to check which plugins are enabled
	p.enabledPlugins = make(map[string]bool)

	pluginRegistry := authInstance.GetPluginRegistry()
	if pluginRegistry != nil {
		pluginList := pluginRegistry.List()
		for _, plugin := range pluginList {
			pluginID := plugin.ID()
			p.enabledPlugins[pluginID] = true
			fmt.Printf("[Dashboard]    ✓ Enabled plugin: %s\n", pluginID)
		}
	} else {
		fmt.Printf("[Dashboard] ⚠️  Plugin registry is nil\n")
	}

	// Get required services from registry using specific getters
	p.userSvc = serviceRegistry.UserService()
	if p.userSvc == nil {
		return fmt.Errorf("user service not found in registry")
	}

	p.sessionSvc = serviceRegistry.SessionService()
	if p.sessionSvc == nil {
		return fmt.Errorf("session service not found in registry")
	}

	p.auditSvc = serviceRegistry.AuditService()
	if p.auditSvc == nil {
		return fmt.Errorf("audit service not found in registry")
	}

	p.rbacSvc = serviceRegistry.RBACService()
	if p.rbacSvc == nil {
		return fmt.Errorf("rbac service not found in registry")
	}

	// Get API Key service if plugin is enabled
	p.apikeyService = serviceRegistry.APIKeyService()

	// Get Organization service and check if we're in SaaS mode
	if orgSvcInterface := serviceRegistry.OrganizationService(); orgSvcInterface != nil {
		// Try to get core organization service
		if orgSvc, ok := orgSvcInterface.(*organization.Service); ok {
			p.orgService = orgSvc
			p.isSaaSMode = serviceRegistry.IsMultiTenant()
		}

		// Try to get multitenancy organization service
		if _, ok := orgSvcInterface.(*mtorg.Service); ok {
			// Multitenancy service is available, will be set in handler
			p.isSaaSMode = serviceRegistry.IsMultiTenant()
		}
	}

	// Initialize Permission Checker
	userRoleRepo := repository.NewUserRoleRepository(db)
	p.permChecker = NewPermissionChecker(p.rbacSvc, userRoleRepo)

	// Initialize CSRF Protector
	csrfProtector, err := NewCSRFProtector()
	if err != nil {
		return fmt.Errorf("failed to initialize CSRF protector: %w", err)
	}
	p.csrfProtector = csrfProtector

	// Note: Role registration now happens via RegisterRoles() method (PluginWithRoles interface)
	// This is called automatically by the authsome initialization system

	// Get platform organization ID for context injection
	// Dashboard always operates in platform organization context
	var platformOrg struct {
		ID xid.ID `bun:"id"`
	}
	err = db.NewSelect().
		Table("organizations").
		Column("id").
		Where("is_platform = ?", true).
		Scan(context.Background(), &platformOrg)

	if err != nil {
		fmt.Printf("[Dashboard] Warning: Could not find platform organization: %v\n", err)
		fmt.Printf("[Dashboard] Dashboard will operate without platform org context\n")
	} else {
		p.platformOrgID = platformOrg.ID
		fmt.Printf("[Dashboard] ✅ Platform organization ID loaded: %s\n", p.platformOrgID.String())
	}

	// Setup default RBAC policies for immediate use (backward compatibility)
	// The role bootstrap will ensure these are persisted
	fmt.Printf("[Dashboard] Setting up default RBAC policies...\n")
	if err := SetupDefaultPolicies(p.rbacSvc); err != nil {
		return fmt.Errorf("failed to setup default policies: %w", err)
	}
	fmt.Printf("[Dashboard] ✅ Default RBAC policies configured\n")

	// Templates no longer needed - using gomponents
	// Initialize handler with services, base path, and enabled plugins
	p.handler = NewHandler(
		assets,
		p.userSvc,
		p.sessionSvc,
		p.auditSvc,
		p.rbacSvc,
		p.apikeyService,
		p.orgService,
		db,
		p.isSaaSMode,
		p.basePath,
		p.enabledPlugins,
		hookRegistry,
	)

	// Set multitenancy organization service if available
	if orgSvcInterface := serviceRegistry.OrganizationService(); orgSvcInterface != nil {
		if mtOrgSvc, ok := orgSvcInterface.(*mtorg.Service); ok {
			p.handler.SetMultitenancyOrgService(mtOrgSvc)
		}
	}

	return nil
}

// RegisterRoles implements the PluginWithRoles optional interface
// This is called automatically during server initialization to register dashboard roles
func (p *Plugin) RegisterRoles(registry interface{}) error {
	roleRegistry, ok := registry.(*rbac.RoleRegistry)
	if !ok {
		return fmt.Errorf("invalid role registry type")
	}

	fmt.Printf("[Dashboard] Registering dashboard roles in RoleRegistry...\n")

	// Dashboard plugin extends/modifies the default roles with additional permissions
	// Note: Default roles (superadmin, owner, admin, member) are already registered by core
	// We extend them with dashboard-specific permissions
	if err := RegisterDashboardRoles(roleRegistry); err != nil {
		return fmt.Errorf("failed to register dashboard roles: %w", err)
	}

	fmt.Printf("[Dashboard] ✅ Dashboard roles registered\n")
	return nil
}

// PlatformOrgContext middleware injects platform organization context into all dashboard requests
// Dashboard always operates in the context of the platform organization without requiring API keys
func (p *Plugin) PlatformOrgContext() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Skip if platform org ID not set
			if p.platformOrgID.IsNil() {
				return next(c)
			}

			// Inject platform organization ID into request context using the SetOrganizationID helper
			// This gives dashboard implicit platform-level access without needing API keys
			ctx := interfaces.SetOrganizationID(c.Request().Context(), p.platformOrgID)

			// Create new request with updated context
			r := c.Request().WithContext(ctx)

			// Store the new request - we need to use reflection or direct access
			// Since Forge Context wraps *http.Request, we update it directly
			*c.Request() = *r

			return next(c)
		}
	}
}

// RegisterRoutes registers the dashboard routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("dashboard handler not initialized; call Init first")
	}

	// Create middleware chain with platform org context
	// Platform org context is injected FIRST, then other middleware
	chain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.PlatformOrgContext()(p.RequireAuth()(p.RequireAdmin()(p.AuditLog()(p.RateLimit()(h)))))
	}

	// Chain for authenticated routes (login/signup) - with platform org context but no auth
	authlessChain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.PlatformOrgContext()(h)
	}

	// Test route without middleware
	router.GET("/dashboard/ping", func(c forge.Context) error {
		return c.JSON(200, map[string]string{"message": "Dashboard plugin is working!"})
	},
		forge.WithName("dashboard.ping"),
		forge.WithSummary("Dashboard health check"),
		forge.WithDescription("Verify that the dashboard plugin is loaded and working"),
		forge.WithResponseSchema(200, "Dashboard is working", DashboardPingResponse{}),
		forge.WithTags("Dashboard", "Health"),
	)

	// Public routes (no auth middleware) - with platform org context injection
	// These must be accessible without authentication but still need platform org context
	router.GET("/dashboard/login", authlessChain(p.handler.ServeLogin),
		forge.WithName("dashboard.login.page"),
		forge.WithSummary("Login page"),
		forge.WithDescription("Render the admin dashboard login page"),
		forge.WithResponseSchema(200, "Login page HTML", DashboardHTMLResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
	)

	router.POST("/dashboard/login", authlessChain(p.handler.HandleLogin),
		forge.WithName("dashboard.login.submit"),
		forge.WithSummary("Process login"),
		forge.WithDescription("Authenticate admin user and create dashboard session"),
		forge.WithResponseSchema(200, "Login successful", DashboardLoginResponse{}),
		forge.WithResponseSchema(401, "Invalid credentials", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
		forge.WithValidation(true),
	)

	router.GET("/dashboard/signup", authlessChain(p.handler.ServeSignup),
		forge.WithName("dashboard.signup.page"),
		forge.WithSummary("Signup page"),
		forge.WithDescription("Render the admin dashboard signup page"),
		forge.WithResponseSchema(200, "Signup page HTML", DashboardHTMLResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
	)

	router.POST("/dashboard/signup", authlessChain(p.handler.HandleSignup),
		forge.WithName("dashboard.signup.submit"),
		forge.WithSummary("Process signup"),
		forge.WithDescription("Register new admin user for dashboard access"),
		forge.WithResponseSchema(200, "Signup successful", DashboardLoginResponse{}),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
		forge.WithValidation(true),
	)

	router.POST("/dashboard/logout", p.handler.HandleLogout,
		forge.WithName("dashboard.logout"),
		forge.WithSummary("Logout"),
		forge.WithDescription("End dashboard session and logout admin user"),
		forge.WithResponseSchema(200, "Logout successful", DashboardStatusResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
	)

	router.GET("/dashboard/logout", p.handler.HandleLogout,
		forge.WithName("dashboard.logout.get"),
		forge.WithSummary("Logout (GET)"),
		forge.WithDescription("End dashboard session via GET request (for convenience)"),
		forge.WithResponseSchema(200, "Logout successful", DashboardStatusResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
	)

	// Dashboard pages (with auth middleware)
	router.GET("/dashboard/", chain(p.handler.ServeDashboard),
		forge.WithName("dashboard.home"),
		forge.WithSummary("Dashboard home"),
		forge.WithDescription("Render the main admin dashboard page with statistics and overview"),
		forge.WithResponseSchema(200, "Dashboard home HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin"),
	)

	router.GET("/dashboard/users", chain(p.handler.ServeUsers),
		forge.WithName("dashboard.users.list"),
		forge.WithSummary("List users"),
		forge.WithDescription("Render the user management page with list of all users"),
		forge.WithResponseSchema(200, "Users list HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Users"),
	)

	router.GET("/dashboard/users/:id", chain(p.handler.ServeUserDetail),
		forge.WithName("dashboard.users.detail"),
		forge.WithSummary("User detail"),
		forge.WithDescription("Render detailed view of a specific user"),
		forge.WithResponseSchema(200, "User detail HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Users"),
	)

	router.GET("/dashboard/users/:id/edit", chain(p.handler.ServeUserEdit),
		forge.WithName("dashboard.users.edit.page"),
		forge.WithSummary("Edit user page"),
		forge.WithDescription("Render the user edit form"),
		forge.WithResponseSchema(200, "User edit form HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Users"),
	)

	router.POST("/dashboard/users/:id/edit", chain(p.handler.HandleUserEdit),
		forge.WithName("dashboard.users.edit.submit"),
		forge.WithSummary("Update user"),
		forge.WithDescription("Process user edit form and update user information"),
		forge.WithResponseSchema(200, "User updated", DashboardStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Users"),
		forge.WithValidation(true),
	)

	router.POST("/dashboard/users/:id/delete", chain(p.handler.HandleUserDelete),
		forge.WithName("dashboard.users.delete"),
		forge.WithSummary("Delete user"),
		forge.WithDescription("Delete a user account (requires admin privileges)"),
		forge.WithResponseSchema(200, "User deleted", DashboardStatusResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Users"),
	)

	router.GET("/dashboard/sessions", chain(p.handler.ServeSessions),
		forge.WithName("dashboard.sessions.list"),
		forge.WithSummary("List sessions"),
		forge.WithDescription("Render the session management page with all active sessions"),
		forge.WithResponseSchema(200, "Sessions list HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Sessions"),
	)

	router.POST("/dashboard/sessions/:id/revoke", chain(p.handler.HandleRevokeSession),
		forge.WithName("dashboard.sessions.revoke"),
		forge.WithSummary("Revoke session"),
		forge.WithDescription("Revoke a specific user session by ID"),
		forge.WithResponseSchema(200, "Session revoked", DashboardStatusResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "Session not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Sessions"),
	)

	router.POST("/dashboard/sessions/revoke-user", chain(p.handler.HandleRevokeUserSessions),
		forge.WithName("dashboard.sessions.revoke.user"),
		forge.WithSummary("Revoke all user sessions"),
		forge.WithDescription("Revoke all sessions for a specific user"),
		forge.WithResponseSchema(200, "User sessions revoked", DashboardStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Sessions"),
		forge.WithValidation(true),
	)

	router.GET("/dashboard/settings", chain(p.handler.ServeSettings),
		forge.WithName("dashboard.settings"),
		forge.WithSummary("Settings page"),
		forge.WithDescription("Render the dashboard settings and configuration page"),
		forge.WithResponseSchema(200, "Settings page HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Settings"),
	)

	// Organization management routes (only available in SaaS mode)
	if p.isSaaSMode {
		router.GET("/dashboard/organizations", chain(p.handler.ServeOrganizations),
			forge.WithName("dashboard.organizations.list"),
			forge.WithSummary("List organizations"),
			forge.WithDescription("Render the organizations management page with list of all organizations"),
			forge.WithResponseSchema(200, "Organizations list HTML", DashboardHTMLResponse{}),
			forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
			forge.WithTags("Dashboard", "Admin", "Organizations"),
		)

		router.GET("/dashboard/organizations/create", chain(p.handler.ServeOrganizationCreate),
			forge.WithName("dashboard.organizations.create.page"),
			forge.WithSummary("Create organization page"),
			forge.WithDescription("Render the organization creation form"),
			forge.WithResponseSchema(200, "Create organization form HTML", DashboardHTMLResponse{}),
			forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
			forge.WithTags("Dashboard", "Admin", "Organizations"),
		)

		router.POST("/dashboard/organizations/create", chain(p.handler.HandleOrganizationCreate),
			forge.WithName("dashboard.organizations.create.submit"),
			forge.WithSummary("Create organization"),
			forge.WithDescription("Process organization creation form and create new organization"),
			forge.WithResponseSchema(200, "Organization created", DashboardStatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
			forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
			forge.WithTags("Dashboard", "Admin", "Organizations"),
			forge.WithValidation(true),
		)

		router.GET("/dashboard/organizations/:id", chain(p.handler.ServeOrganizationDetail),
			forge.WithName("dashboard.organizations.detail"),
			forge.WithSummary("Organization detail"),
			forge.WithDescription("Render detailed view of a specific organization"),
			forge.WithResponseSchema(200, "Organization detail HTML", DashboardHTMLResponse{}),
			forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
			forge.WithResponseSchema(404, "Organization not found", DashboardErrorResponse{}),
			forge.WithTags("Dashboard", "Admin", "Organizations"),
		)

		router.GET("/dashboard/organizations/:id/edit", chain(p.handler.ServeOrganizationEdit),
			forge.WithName("dashboard.organizations.edit.page"),
			forge.WithSummary("Edit organization page"),
			forge.WithDescription("Render the organization edit form"),
			forge.WithResponseSchema(200, "Edit organization form HTML", DashboardHTMLResponse{}),
			forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
			forge.WithResponseSchema(404, "Organization not found", DashboardErrorResponse{}),
			forge.WithTags("Dashboard", "Admin", "Organizations"),
		)

		router.POST("/dashboard/organizations/:id/edit", chain(p.handler.HandleOrganizationEdit),
			forge.WithName("dashboard.organizations.edit.submit"),
			forge.WithSummary("Update organization"),
			forge.WithDescription("Process organization edit form and update organization information"),
			forge.WithResponseSchema(200, "Organization updated", DashboardStatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
			forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
			forge.WithResponseSchema(404, "Organization not found", DashboardErrorResponse{}),
			forge.WithTags("Dashboard", "Admin", "Organizations"),
			forge.WithValidation(true),
		)

		router.POST("/dashboard/organizations/:id/delete", chain(p.handler.HandleOrganizationDelete),
			forge.WithName("dashboard.organizations.delete"),
			forge.WithSummary("Delete organization"),
			forge.WithDescription("Delete an organization (requires admin privileges)"),
			forge.WithResponseSchema(200, "Organization deleted", DashboardStatusResponse{}),
			forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
			forge.WithResponseSchema(403, "Insufficient privileges", DashboardErrorResponse{}),
			forge.WithResponseSchema(404, "Organization not found", DashboardErrorResponse{}),
			forge.WithTags("Dashboard", "Admin", "Organizations"),
		)
	}

	// Static assets (no auth required)
	router.GET("/dashboard/static/*", p.handler.ServeStatic,
		forge.WithName("dashboard.static"),
		forge.WithSummary("Static assets"),
		forge.WithDescription("Serve static assets (CSS, JS, images) for the dashboard"),
		forge.WithResponseSchema(200, "Static file", DashboardStaticResponse{}),
		forge.WithResponseSchema(404, "File not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Assets"),
	)

	// 404 catch-all for any unmatched dashboard routes (must be last)
	// Note: This won't catch routes that match above patterns
	// It's more for documenting the 404 handler
	// Actual 404s will be handled by the framework or custom middleware

	fmt.Printf("[Dashboard] Routes registered successfully\n")
	fmt.Printf("[Dashboard] Available endpoints:\n")
	fmt.Printf("  - GET  %s/dashboard/login (login page)\n", p.basePath)
	fmt.Printf("  - POST %s/dashboard/login (process login)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/signup (signup page)\n", p.basePath)
	fmt.Printf("  - POST %s/dashboard/signup (process signup)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/ (home)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/users (list)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/users/:id (detail)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/sessions (list)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/static/* (assets)\n", p.basePath)

	return nil
}

// DTOs for dashboard routes
type DashboardErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type DashboardStatusResponse struct {
	Status string `json:"status" example:"success"`
}

type DashboardHTMLResponse struct {
	HTML string `json:"html" example:"<html>...</html>"`
}

type DashboardPingResponse struct {
	Message string `json:"message" example:"Dashboard plugin is working!"`
}

type DashboardLoginResponse struct {
	RedirectURL string `json:"redirect_url" example:"/dashboard/"`
}

type DashboardStaticResponse struct {
	ContentType string `json:"content_type" example:"text/css"`
	Content     []byte `json:"content"`
}

// RegisterHooks registers hooks for the dashboard plugin
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	// Dashboard plugin doesn't need hooks
	return nil
}

// RegisterServiceDecorators registers service decorators
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Dashboard plugin doesn't need service decorators
	return nil
}

// Migrate runs database migrations for the dashboard plugin
func (p *Plugin) Migrate() error {
	// Dashboard plugin doesn't need database migrations
	return nil
}
