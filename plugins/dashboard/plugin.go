package dashboard

import (
	"context"
	"embed"
	"fmt"
	"net/http"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

//go:embed static/css/* static/js/*.js
var assets embed.FS

// Plugin implements the dashboard plugin for AuthSome
type Plugin struct {
	log               forge.Logger
	handler           *Handler
	userSvc           user.ServiceInterface
	sessionSvc        session.ServiceInterface
	auditSvc          *audit.Service
	rbacSvc           *rbac.Service
	apikeyService     *apikey.Service
	appService        app.Service
	orgService        *organization.Service
	isMultiAppMode    bool
	permChecker       *PermissionChecker
	csrfProtector     *CSRFProtector
	basePath          string
	enabledPlugins    map[string]bool
	config            Config
	defaultConfig     Config
	platformOrgID     xid.ID // Platform organization ID for context injection
	db                *bun.DB
	serviceRegistry   *registry.ServiceRegistry // Store for checking multitenancy service after all plugins init
	extensionRegistry *ExtensionRegistry        // Registry for dashboard extensions from other plugins
	pluginRegistry    core.PluginRegistry       // Reference to plugin registry for discovering extensions
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

// Dependencies declares the plugin dependencies
// Dashboard requires multiapp plugin for environment management
func (p *Plugin) Dependencies() []string {
	return []string{"multiapp"}
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
	p.pluginRegistry = authInstance.GetPluginRegistry()

	if p.pluginRegistry != nil {
		pluginList := p.pluginRegistry.List()
		for _, plugin := range pluginList {
			pluginID := plugin.ID()
			p.enabledPlugins[pluginID] = true

			if pluginID == "multiapp" {
				p.isMultiAppMode = true
			}
			p.log.Info("enabled plugin", forge.F("plugin", pluginID))
		}
	} else {
		p.log.Warn("plugin registry is nil")
	}

	// Initialize extension registry
	p.extensionRegistry = NewExtensionRegistry()
	p.log.Info("dashboard extension registry initialized")

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

	// Get App service (required for multi-app support)
	p.appService = serviceRegistry.AppService()
	if p.appService == nil {
		return fmt.Errorf("app service not found in registry")
	}

	// Get Environment service (optional - only available with multiapp plugin)
	// If not available, dashboard will operate without environment management features
	envService := serviceRegistry.EnvironmentService()
	if envService == nil {
		p.log.Warn("environment service not available - dashboard will operate without environment management")
		// Continue initialization - environment service is optional
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
		envService,
		db,
		p.isMultiAppMode,
		p.basePath,
		p.enabledPlugins,
		hookRegistry,
	)

	// Store service registry for later access in RegisterRoutes
	p.serviceRegistry = serviceRegistry

	// Note: Dashboard extensions will be registered in RegisterRoutes()
	// after all plugins have been initialized. This ensures plugins can
	// access their services when creating dashboard extensions.

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

			// Note: Platform org context injection removed
			// Handlers now explicitly use platformOrgID for operations
			return next(c)
		}
	}
}

// AppContext middleware injects app context into dashboard requests for authless routes
func (p *Plugin) AppContext() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Note: App context injection removed
			// Handlers now extract appID from URL path parameters (/dashboard/app/:appId/)
			// App ID from headers or subdomains is ignored in favor of URL-based routing
			return next(c)
		}
	}
}

// RegisterRoutes registers the dashboard routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("dashboard handler not initialized; call Init first")
	}

	// Discover and register dashboard extensions from other plugins
	if p.pluginRegistry != nil {
		p.log.Info("discovering dashboard extensions from plugins...")
		pluginList := p.pluginRegistry.List()
		for _, plugin := range pluginList {
			// Check if plugin implements PluginWithDashboardExtension
			if extPlugin, ok := plugin.(core.PluginWithDashboardExtension); ok {
				extensionInterface := extPlugin.DashboardExtension()
				if extensionInterface == nil {
					p.log.Warn("plugin returned nil dashboard extension",
						forge.F("plugin", plugin.ID()))
					continue
				}

				// Type assert to ui.DashboardExtension
				if ext, ok := extensionInterface.(ui.DashboardExtension); ok {
					if err := p.extensionRegistry.Register(ext); err != nil {
						p.log.Error("failed to register dashboard extension",
							forge.F("plugin", plugin.ID()),
							forge.F("error", err.Error()))
						continue
					}
					p.log.Info("registered dashboard extension",
						forge.F("plugin", plugin.ID()),
						forge.F("extension", ext.ExtensionID()))
				} else {
					p.log.Warn("dashboard extension does not implement DashboardExtension interface",
						forge.F("plugin", plugin.ID()))
				}
			}
		}
		p.log.Info("dashboard extension discovery complete",
			forge.F("extensions", len(p.extensionRegistry.List())))
	}

	// Pass extension registry to handler and vice versa (for extension rendering)
	p.handler.extensionRegistry = p.extensionRegistry
	p.extensionRegistry.SetHandler(p.handler)

	// Create middleware chain with platform org context
	// Platform org context is injected FIRST, then environment context, then other middleware
	chain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.PlatformOrgContext()(p.EnvironmentContext()(p.RequireAuth()(p.RequireAdmin()(p.AuditLog()(p.RateLimit()(h))))))
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

	// Environment switcher
	router.POST("/dashboard/app/:appId/environment/switch", chain(p.handler.HandleEnvironmentSwitch),
		forge.WithName("dashboard.app.environment.switch"),
		forge.WithSummary("Switch environment"),
		forge.WithDescription("Switch the current selected environment for the app"),
		forge.WithResponseSchema(302, "Redirects to referrer", nil),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App or environment not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Environments"),
		forge.WithValidation(true),
	)

	// Environment management
	router.GET("/dashboard/app/:appId/environments", chain(p.handler.ServeEnvironments),
		forge.WithName("dashboard.app.environments"),
		forge.WithSummary("App environments page"),
		forge.WithDescription("Render the environments list page showing all environments for the app"),
		forge.WithResponseSchema(200, "Environments page HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Environments"),
	)

	router.GET("/dashboard/app/:appId/environments/create", chain(p.handler.ServeEnvironmentCreate),
		forge.WithName("dashboard.app.environments.create"),
		forge.WithSummary("Create environment page"),
		forge.WithDescription("Render the create environment form page"),
		forge.WithResponseSchema(200, "Create environment page HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Environments"),
	)

	router.POST("/dashboard/app/:appId/environments/create", chain(p.handler.HandleEnvironmentCreate),
		forge.WithName("dashboard.app.environments.create.post"),
		forge.WithSummary("Create environment"),
		forge.WithDescription("Create a new environment for the app"),
		forge.WithResponseSchema(302, "Redirects to environment detail", nil),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Environments"),
		forge.WithValidation(true),
	)

	router.GET("/dashboard/app/:appId/environments/:envId", chain(p.handler.ServeEnvironmentDetail),
		forge.WithName("dashboard.app.environments.detail"),
		forge.WithSummary("Environment detail page"),
		forge.WithDescription("Render the environment detail page showing environment information"),
		forge.WithResponseSchema(200, "Environment detail page HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App or environment not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Environments"),
	)

	router.GET("/dashboard/app/:appId/environments/:envId/edit", chain(p.handler.ServeEnvironmentEdit),
		forge.WithName("dashboard.app.environments.edit"),
		forge.WithSummary("Edit environment page"),
		forge.WithDescription("Render the edit environment form page"),
		forge.WithResponseSchema(200, "Edit environment page HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App or environment not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Environments"),
	)

	router.POST("/dashboard/app/:appId/environments/:envId/edit", chain(p.handler.HandleEnvironmentEdit),
		forge.WithName("dashboard.app.environments.edit.post"),
		forge.WithSummary("Update environment"),
		forge.WithDescription("Update an existing environment's information"),
		forge.WithResponseSchema(302, "Redirects to environment detail", nil),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App or environment not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Environments"),
		forge.WithValidation(true),
	)

	router.POST("/dashboard/app/:appId/environments/:envId/delete", chain(p.handler.HandleEnvironmentDelete),
		forge.WithName("dashboard.app.environments.delete"),
		forge.WithSummary("Delete environment"),
		forge.WithDescription("Delete an existing environment from the app"),
		forge.WithResponseSchema(302, "Redirects to environments list", nil),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App or environment not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Environments"),
		forge.WithValidation(true),
	)

	// Settings routes - redirect base to general
	router.GET("/dashboard/app/:appId/settings", func(c forge.Context) error {
		appID := c.Param("appId")
		return c.Redirect(http.StatusFound, fmt.Sprintf("%s/dashboard/app/%s/settings/general", p.basePath, appID))
	},
		forge.WithName("dashboard.app.settings.redirect"),
		forge.WithSummary("Settings redirect"),
		forge.WithDescription("Redirects to general settings page"),
		forge.WithResponseSchema(302, "Redirect to general settings", DashboardHTMLResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Settings"),
	)

	router.GET("/dashboard/app/:appId/settings/general", chain(p.handler.ServeSettingsGeneral),
		forge.WithName("dashboard.app.settings.general"),
		forge.WithSummary("General settings page"),
		forge.WithDescription("Render the general dashboard settings page"),
		forge.WithResponseSchema(200, "General settings page HTML", DashboardHTMLResponse{}),
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

	// // Organization management routes (user-created organizations within app)
	// router.GET("/dashboard/app/:appId/organizations", chain(p.handler.ServeOrganizations),
	// 	forge.WithName("dashboard.app.organizations.list"),
	// 	forge.WithSummary("List organizations"),
	// 	forge.WithDescription("Render the organizations management page with all user-created organizations in the app"),
	// 	forge.WithResponseSchema(200, "Organizations list HTML", DashboardHTMLResponse{}),
	// 	forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
	// 	forge.WithTags("Dashboard", "Admin", "Apps", "Organizations"),
	// )

	// router.GET("/dashboard/app/:appId/organizations/create", chain(p.handler.ServeOrganizationCreate),
	// 	forge.WithName("dashboard.app.organizations.create.page"),
	// 	forge.WithSummary("Create organization page"),
	// 	forge.WithDescription("Render the organization creation form"),
	// 	forge.WithResponseSchema(200, "Organization create form HTML", DashboardHTMLResponse{}),
	// 	forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
	// 	forge.WithTags("Dashboard", "Admin", "Apps", "Organizations"),
	// )

	// router.POST("/dashboard/app/:appId/organizations/create", chain(p.handler.HandleOrganizationCreate),
	// 	forge.WithName("dashboard.app.organizations.create.submit"),
	// 	forge.WithSummary("Create organization"),
	// 	forge.WithDescription("Process organization creation form and create new organization"),
	// 	forge.WithResponseSchema(200, "Organization created", DashboardStatusResponse{}),
	// 	forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
	// 	forge.WithTags("Dashboard", "Admin", "Apps", "Organizations"),
	// 	forge.WithValidation(true),
	// )

	// router.GET("/dashboard/app/:appId/organizations/:orgId", chain(p.handler.ServeOrganizationDetail),
	// 	forge.WithName("dashboard.app.organizations.detail"),
	// 	forge.WithSummary("Organization detail"),
	// 	forge.WithDescription("Render detailed view of a specific organization in the app"),
	// 	forge.WithResponseSchema(200, "Organization detail HTML", DashboardHTMLResponse{}),
	// 	forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(404, "Organization or app not found", DashboardErrorResponse{}),
	// 	forge.WithTags("Dashboard", "Admin", "Apps", "Organizations"),
	// )

	// router.GET("/dashboard/app/:appId/organizations/:orgId/edit", chain(p.handler.ServeOrganizationEdit),
	// 	forge.WithName("dashboard.app.organizations.edit.page"),
	// 	forge.WithSummary("Edit organization page"),
	// 	forge.WithDescription("Render the organization edit form"),
	// 	forge.WithResponseSchema(200, "Organization edit form HTML", DashboardHTMLResponse{}),
	// 	forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(404, "Organization or app not found", DashboardErrorResponse{}),
	// 	forge.WithTags("Dashboard", "Admin", "Apps", "Organizations"),
	// )

	// router.POST("/dashboard/app/:appId/organizations/:orgId/edit", chain(p.handler.HandleOrganizationEdit),
	// 	forge.WithName("dashboard.app.organizations.edit.submit"),
	// 	forge.WithSummary("Update organization"),
	// 	forge.WithDescription("Process organization edit form and update organization information"),
	// 	forge.WithResponseSchema(200, "Organization updated", DashboardStatusResponse{}),
	// 	forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(404, "Organization or app not found", DashboardErrorResponse{}),
	// 	forge.WithTags("Dashboard", "Admin", "Apps", "Organizations"),
	// 	forge.WithValidation(true),
	// )

	// router.POST("/dashboard/app/:appId/organizations/:orgId/delete", chain(p.handler.HandleOrganizationDelete),
	// 	forge.WithName("dashboard.app.organizations.delete"),
	// 	forge.WithSummary("Delete organization"),
	// 	forge.WithDescription("Delete an organization (requires admin privileges)"),
	// 	forge.WithResponseSchema(200, "Organization deleted", DashboardStatusResponse{}),
	// 	forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(403, "Insufficient privileges", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(404, "Organization or app not found", DashboardErrorResponse{}),
	// 	forge.WithTags("Dashboard", "Admin", "Apps", "Organizations"),
	// )

	// // App Management routes (platform apps management - admin only)
	// // Create is only available when multiapp plugin is enabled
	// router.GET("/dashboard/app/:appId/apps-management", chain(p.handler.ServeAppsManagement),
	// 	forge.WithName("dashboard.app.apps-management.list"),
	// 	forge.WithSummary("List platform apps"),
	// 	forge.WithDescription("Render the apps management page with all platform apps (admin only)"),
	// 	forge.WithResponseSchema(200, "Apps management list HTML", DashboardHTMLResponse{}),
	// 	forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
	// 	forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
	// 	forge.WithTags("Dashboard", "Admin", "Apps", "Management"),
	// )

	router.GET("/dashboard/app/:appId/apps-management/create", chain(p.handler.ServeAppMgmtCreate),
		forge.WithName("dashboard.app.apps-management.create.page"),
		forge.WithSummary("Create app page"),
		forge.WithDescription("Render the app creation form (requires multiapp plugin)"),
		forge.WithResponseSchema(200, "App create form HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(403, "Multiapp plugin not enabled", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Management"),
	)

	router.POST("/dashboard/app/:appId/apps-management/create", chain(p.handler.HandleAppMgmtCreate),
		forge.WithName("dashboard.app.apps-management.create.submit"),
		forge.WithSummary("Create app"),
		forge.WithDescription("Process app creation form (requires multiapp plugin)"),
		forge.WithResponseSchema(200, "App created", DashboardStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(403, "Multiapp plugin not enabled", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Management"),
		forge.WithValidation(true),
	)

	router.GET("/dashboard/app/:appId/apps-management/:targetAppId", chain(p.handler.ServeAppMgmtDetail),
		forge.WithName("dashboard.app.apps-management.detail"),
		forge.WithSummary("App detail"),
		forge.WithDescription("Render detailed view of a specific platform app"),
		forge.WithResponseSchema(200, "App detail HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Management"),
	)

	router.GET("/dashboard/app/:appId/apps-management/:targetAppId/edit", chain(p.handler.ServeAppMgmtEdit),
		forge.WithName("dashboard.app.apps-management.edit.page"),
		forge.WithSummary("Edit app page"),
		forge.WithDescription("Render the app edit form"),
		forge.WithResponseSchema(200, "App edit form HTML", DashboardHTMLResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Management"),
	)

	router.POST("/dashboard/app/:appId/apps-management/:targetAppId/edit", chain(p.handler.HandleAppMgmtEdit),
		forge.WithName("dashboard.app.apps-management.edit.submit"),
		forge.WithSummary("Update app"),
		forge.WithDescription("Process app edit form and update app information"),
		forge.WithResponseSchema(200, "App updated", DashboardStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", DashboardErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Management"),
		forge.WithValidation(true),
	)

	router.POST("/dashboard/app/:appId/apps-management/:targetAppId/delete", chain(p.handler.HandleAppMgmtDelete),
		forge.WithName("dashboard.app.apps-management.delete"),
		forge.WithSummary("Delete app"),
		forge.WithDescription("Delete a platform app (requires admin privileges, cannot delete platform app)"),
		forge.WithResponseSchema(200, "App deleted", DashboardStatusResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", DashboardErrorResponse{}),
		forge.WithResponseSchema(403, "Insufficient privileges or platform app", DashboardErrorResponse{}),
		forge.WithResponseSchema(404, "App not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Admin", "Apps", "Management"),
	)

	// Static assets (no auth required)
	router.GET("/dashboard/static/*", p.handler.ServeStatic,
		forge.WithName("dashboard.static"),
		forge.WithSummary("Static assets"),
		forge.WithDescription("Serve static assets (CSS, JS, images) for the dashboard"),
		forge.WithResponseSchema(200, "Static file", DashboardStaticResponse{}),
		forge.WithResponseSchema(404, "File not found", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Assets"),
	)

	// Register routes from dashboard extensions
	extensionRoutes := p.extensionRegistry.GetAllRoutes()
	p.log.Info("registering extension routes",
		forge.F("count", len(extensionRoutes)),
		forge.F("basePath", p.basePath))

	for _, route := range extensionRoutes {
		// Build full path with app context
		fullPath := "/dashboard/app/:appId" + route.Path

		// Build middleware chain based on route requirements
		var handler func(forge.Context) error
		if handlerFunc, ok := route.Handler.(func(forge.Context) error); ok {
			if route.RequireAuth && route.RequireAdmin {
				handler = chain(handlerFunc)
			} else if route.RequireAuth {
				// Auth but not admin - create custom chain with environment context
				handler = p.PlatformOrgContext()(p.EnvironmentContext()(p.RequireAuth()(p.AuditLog()(p.RateLimit()(handlerFunc)))))
			} else {
				// No auth required
				handler = authlessChain(handlerFunc)
			}
		} else {
			p.log.Warn("extension route handler is not a valid function",
				forge.F("path", fullPath),
				forge.F("name", route.Name),
				forge.F("handler_type", fmt.Sprintf("%T", route.Handler)))
			continue
		}

		// Register route based on method
		opts := []forge.RouteOption{
			forge.WithName(route.Name),
			forge.WithSummary(route.Summary),
			forge.WithDescription(route.Description),
		}
		if len(route.Tags) > 0 {
			opts = append(opts, forge.WithTags(route.Tags...))
		}

		switch route.Method {
		case "GET":
			router.GET(fullPath, handler, opts...)
		case "POST":
			router.POST(fullPath, handler, opts...)
		case "PUT":
			router.PUT(fullPath, handler, opts...)
		case "DELETE":
			router.DELETE(fullPath, handler, opts...)
		case "PATCH":
			router.PATCH(fullPath, handler, opts...)
		default:
			p.log.Warn("unsupported HTTP method for extension route",
				forge.F("method", route.Method),
				forge.F("path", fullPath))
		}

		// Log at Info level to ensure visibility
		p.log.Info("registered extension route",
			forge.F("method", route.Method),
			forge.F("path", fullPath),
			forge.F("name", route.Name),
			forge.F("requireAuth", route.RequireAuth),
			forge.F("requireAdmin", route.RequireAdmin))
	}

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
