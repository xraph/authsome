package dashboard

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/interfaces"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	mtorg "github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

//go:embed static/css/* static/js/*.js
var assets embed.FS

// Plugin implements the dashboard plugin for AuthSome
type Plugin struct {
	log             forge.Logger
	handler         *Handler
	userSvc         user.ServiceInterface
	sessionSvc      session.ServiceInterface
	auditSvc        *audit.Service
	rbacSvc         *rbac.Service
	apikeyService   *apikey.Service
	appService      app.AppService
	orgService      organization.OrganizationService
	isMultiAppMode  bool
	permChecker     *PermissionChecker
	csrfProtector   *CSRFProtector
	basePath        string
	enabledPlugins  map[string]bool
	config          Config
	defaultConfig   Config
	platformOrgID   xid.ID // Platform organization ID for context injection
	db              *bun.DB
	serviceRegistry *registry.ServiceRegistry // Store for checking multitenancy service after all plugins init
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
func (p *Plugin) Init(authInstance core.Authsome) error {
	if authInstance == nil {
		return fmt.Errorf("dashboard plugin requires auth instance with GetDB, GetServiceRegistry, GetHookRegistry, GetBasePath, GetPluginRegistry, and GetForgeApp methods")
	}

	// Get Forge app and config manager
	forgeApp := authInstance.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available")
	}

	p.log = forgeApp.Logger().With(forge.F("plugin", "dashboard"))
	configManager := forgeApp.Config()

	// Bind plugin configuration using Forge ConfigManager with provided defaults
	if err := configManager.BindWithDefault("auth.dashboard", &p.config, p.defaultConfig); err != nil {
		// Log but don't fail - use defaults
		p.log.Warn("failed to bind dashboard config", forge.F("error", err.Error()))
		// Fall back to default config
		p.config = p.defaultConfig
	}

	serviceRegistry := authInstance.GetServiceRegistry()
	if serviceRegistry == nil {
		return errs.InternalServerError("service registry not available", nil)
	}

	hookRegistry := authInstance.GetHookRegistry()
	if hookRegistry == nil {
		return errs.InternalServerError("hook registry not available", nil)
	}

	// Get database for repository initialization
	db := authInstance.GetDB()
	if db == nil {
		return errs.InternalServerError("database not available", nil)
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
			if pluginID == "multitenancy" {
				p.isMultiAppMode = true
			}
			p.log.Info("enabled plugin", forge.F("plugin", pluginID))
		}
	} else {
		p.log.Warn("plugin registry is nil")
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
	if orgSvc := serviceRegistry.OrganizationService(); orgSvc != nil {
		p.orgService = orgSvc
		p.isMultiAppMode = serviceRegistry.IsMultiTenant()
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

	platformOrg, err := authInstance.GetServiceRegistry().AppService().GetPlatformApp(context.Background())
	if err != nil {
		p.log.Warn("could not find platform app", forge.F("error", err.Error()))
		p.log.Warn("dashboard will operate without platform app context")
	} else {
		p.platformOrgID = platformOrg.ID
		p.log.Info("platform app loaded", forge.F("id", p.platformOrgID.String()))
	}

	// Setup default RBAC policies for immediate use (backward compatibility)
	// The role bootstrap will ensure these are persisted
	p.log.Info("setting up default RBAC policies")
	if err := SetupDefaultPolicies(p.rbacSvc); err != nil {
		return fmt.Errorf("failed to setup default policies: %w", err)
	}
	p.log.Info("default RBAC policies configured")

	// Templates no longer needed - using gomponents
	// Initialize handler with services, base path, and enabled plugins
	p.handler = NewHandler(
		assets,
		p.appService,
		p.userSvc,
		p.sessionSvc,
		p.auditSvc,
		p.rbacSvc,
		p.apikeyService,
		p.orgService,
		db,
		p.isMultiAppMode,
		p.basePath,
		p.enabledPlugins,
		hookRegistry,
	)

	// Store service registry for later access in RegisterRoutes
	p.serviceRegistry = serviceRegistry

	// Note: Don't check for multitenancy service here during Init()
	// The multitenancy plugin may not have registered its service yet
	// We'll check and set it in RegisterRoutes() which is called after all plugins Init()

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

// AppContext middleware injects app context into dashboard requests for authless routes
func (p *Plugin) AppContext() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			r := c.Request()
			appIDStr := r.Header.Get("X-App-ID")

			host := r.Host
			if idx := strings.Index(host, ":"); idx > 0 {
				host = host[:idx]
			}
			if appIDStr == "" {
				if idx := strings.Index(host, "."); idx > 0 {
					sub := host[:idx]
					if sub != "www" && sub != "api" && sub != "app" {
						appIDStr = sub
					}
				}
			}

			if appIDStr != "" {
				if appID, err := xid.FromString(appIDStr); err == nil {
					ctx := interfaces.SetAppID(r.Context(), appID)
					rr := r.WithContext(ctx)
					*c.Request() = *rr
				}
			}

			return next(c)
		}
	}
}

// RegisterRoutes registers the dashboard routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("dashboard handler not initialized; call Init first")
	}

	// NOW check for multitenancy service (after all plugins have initialized)
	if p.serviceRegistry != nil {
		if orgSvcInterface := p.serviceRegistry.OrganizationService(); orgSvcInterface != nil {
			fmt.Printf("[Dashboard] RegisterRoutes: Checking organization service type: %T\n", orgSvcInterface)
			if mtOrgSvc, ok := orgSvcInterface.(*mtorg.Service); ok {
				p.handler.SetMultitenancyOrgService(mtOrgSvc)
				fmt.Printf("[Dashboard] ✅ Multitenancy organization service set in handler\n")
			} else {
				fmt.Printf("[Dashboard] ⚠️  Organization service is not multitenancy service (type: %T)\n", orgSvcInterface)
				fmt.Printf("[Dashboard] SaaS mode: %v, but multitenancy service not available\n", p.isMultiAppMode)
			}
		} else {
			fmt.Printf("[Dashboard] RegisterRoutes: No organization service found in registry\n")
		}
	}

	// Create middleware chain with platform org context
	// Platform org context is injected FIRST, then other middleware
	chain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.PlatformOrgContext()(p.RequireAuth()(p.RequireAdmin()(p.AuditLog()(p.RateLimit()(h)))))
	}

	// Chain for public auth routes: use app context (multi-app) without auth
	authlessChain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.AppContext()(h)
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

	// Dashboard index - shows app list or redirects to single app
	router.GET("/dashboard/", chain(p.handler.ServeAppsList),
		forge.WithName("dashboard.index"),
		forge.WithSummary("Dashboard index"),
		forge.WithDescription("Show app cards (multiapp mode) or redirect to default app (standalone)"),
		forge.WithResponseSchema(200, "App list or redirect", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Apps"),
	)

	// App-scoped dashboard pages (with auth middleware)
	// All routes now require appId in URL
	router.GET("/dashboard/app/:appId/", chain(p.handler.ServeDashboard),
		forge.WithName("dashboard.app.home"),
		forge.WithSummary("App dashboard home"),
		forge.WithDescription("Render the main admin dashboard page with app-specific statistics and overview"),
		forge.WithResponseSchema(200, "Dashboard home HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps"),
	)

	router.GET("/dashboard/app/:appId/users", chain(p.handler.ServeUsers),
		forge.WithName("dashboard.app.users.list"),
		forge.WithSummary("List app users"),
		forge.WithDescription("Render the user management page with list of all users in the app"),
		forge.WithResponseSchema(200, "Users list HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Users"),
	)

	router.GET("/dashboard/app/:appId/users/:id", chain(p.handler.ServeUserDetail),
		forge.WithName("dashboard.app.users.detail"),
		forge.WithSummary("User detail"),
		forge.WithDescription("Render detailed view of a specific user in the app"),
		forge.WithResponseSchema(200, "User detail HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "User or app not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Users"),
	)

	router.GET("/dashboard/app/:appId/users/:id/edit", chain(p.handler.ServeUserEdit),
		forge.WithName("dashboard.app.users.edit.page"),
		forge.WithSummary("Edit user page"),
		forge.WithDescription("Render the user edit form"),
		forge.WithResponseSchema(200, "User edit form HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "User or app not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Users"),
	)

	router.POST("/dashboard/app/:appId/users/:id/edit", chain(p.handler.HandleUserEdit),
		forge.WithName("dashboard.app.users.edit.submit"),
		forge.WithSummary("Update user"),
		forge.WithDescription("Process user edit form and update user information"),
		forge.WithResponseSchema(200, "User updated", DashboardStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "User or app not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Users"),
		forge.WithValidation(true),
	)

	router.POST("/dashboard/app/:appId/users/:id/delete", chain(p.handler.HandleUserDelete),
		forge.WithName("dashboard.app.users.delete"),
		forge.WithSummary("Delete user"),
		forge.WithDescription("Delete a user account (requires admin privileges)"),
		forge.WithResponseSchema(200, "User deleted", DashboardStatusResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "User or app not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Users"),
	)

	router.GET("/dashboard/app/:appId/sessions", chain(p.handler.ServeSessions),
		forge.WithName("dashboard.app.sessions.list"),
		forge.WithSummary("List app sessions"),
		forge.WithDescription("Render the session management page with all active sessions in the app"),
		forge.WithResponseSchema(200, "Sessions list HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Sessions"),
	)

	router.POST("/dashboard/app/:appId/sessions/:id/revoke", chain(p.handler.HandleRevokeSession),
		forge.WithName("dashboard.app.sessions.revoke"),
		forge.WithSummary("Revoke session"),
		forge.WithDescription("Revoke a specific user session by ID"),
		forge.WithResponseSchema(200, "Session revoked", DashboardStatusResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "Session or app not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Sessions"),
	)

	router.POST("/dashboard/app/:appId/sessions/revoke-user", chain(p.handler.HandleRevokeUserSessions),
		forge.WithName("dashboard.app.sessions.revoke.user"),
		forge.WithSummary("Revoke all user sessions"),
		forge.WithDescription("Revoke all sessions for a specific user in the app"),
		forge.WithResponseSchema(200, "User sessions revoked", DashboardStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Sessions"),
		forge.WithValidation(true),
	)

	router.GET("/dashboard/app/:appId/settings", chain(p.handler.ServeSettings),
		forge.WithName("dashboard.app.settings"),
		forge.WithSummary("App settings page"),
		forge.WithDescription("Render the dashboard settings and configuration page for the app"),
		forge.WithResponseSchema(200, "Settings page HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Settings"),
	)

	router.GET("/dashboard/app/:appId/plugins", chain(p.handler.ServePlugins),
		forge.WithName("dashboard.app.plugins"),
		forge.WithSummary("App plugins page"),
		forge.WithDescription("Render the plugins management page showing all available plugins and their status"),
		forge.WithResponseSchema(200, "Plugins page HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Plugins"),
	)

	// NOTE: Old app management routes removed
	// App creation/management is now done via multiapp plugin API
	// Dashboard index (/) shows app cards for selection

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
	fmt.Printf("  - GET  %s/dashboard/ (app list or redirect)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/app/:appId/ (app dashboard)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/app/:appId/users (list)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/app/:appId/users/:id (detail)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/app/:appId/sessions (list)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/app/:appId/settings (settings)\n", p.basePath)
	fmt.Printf("  - GET  %s/dashboard/app/:appId/plugins (plugins)\n", p.basePath)
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
