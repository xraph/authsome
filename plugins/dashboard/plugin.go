package dashboard

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui/schema"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	bridgepkg "github.com/xraph/authsome/plugins/dashboard/bridge"
	"github.com/xraph/authsome/plugins/dashboard/layouts"
	"github.com/xraph/authsome/plugins/dashboard/pages"
	"github.com/xraph/authsome/plugins/dashboard/services"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui"
	"github.com/xraph/forgeui/bridge"
	"github.com/xraph/forgeui/theme"
)

//go:embed static/*/**
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
	envSvc            environment.EnvironmentService
	isMultiAppMode    bool
	permChecker       *PermissionChecker
	csrfProtector     *CSRFProtector
	basePath          string
	staticPath        string
	baseUIPath        string
	enabledPlugins    map[string]bool
	config            Config
	defaultConfig     Config
	platformOrgID     xid.ID // Platform organization ID for context injection
	db                *bun.DB
	serviceRegistry   *registry.ServiceRegistry // Store for checking multitenancy service after all plugins init
	extensionRegistry *ExtensionRegistry        // Registry for dashboard extensions from other plugins
	pluginRegistry    core.PluginRegistry       // Reference to plugin registry for discovering extensions

	fuiApp        *forgeui.App
	fuiBridge     *bridge.Bridge
	pagesManager  *pages.PagesManager
	layoutManager *layouts.LayoutManager
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
			p.log.Debug("enabled plugin", forge.F("plugin", pluginID))
		}
	} else {
		p.log.Warn("plugin registry is nil")
	}

	p.initializeForgeUI()
	// Layout manager already initialized with correct isMultiAppMode in initializeForgeUI()
	p.log.Debug("layout manager configured", forge.F("isMultiAppMode", p.isMultiAppMode))

	// Initialize extension registry
	p.extensionRegistry = NewExtensionRegistry()
	p.log.Debug("dashboard extension registry initialized")

	// Register default settings schema sections
	if err := schema.RegisterDefaultSections(); err != nil {
		p.log.Warn("failed to register default settings sections", forge.F("error", err.Error()))
		// Don't fail initialization - settings will work with reduced functionality
	} else {
		p.log.Debug("default settings schema sections registered")
	}

	// Extension discovery is deferred to RegisterRoutes() to ensure all plugins are fully initialized

	// Set extension registry on layout manager if it exists
	if p.layoutManager != nil {
		p.layoutManager.SetExtensionRegistry(p.extensionRegistry)
	}

	// Set extension registry on pages manager if it exists
	if p.pagesManager != nil {
		p.pagesManager.SetExtensionRegistry(p.extensionRegistry)
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

	// Get App service (required for multi-app support)
	p.appService = serviceRegistry.AppService()
	if p.appService == nil {
		return fmt.Errorf("app service not found in registry")
	}

	// Get Environment service (optional - only available with multiapp plugin)
	// If not available, dashboard will operate without environment management features
	p.envSvc = serviceRegistry.EnvironmentService()
	envService := p.envSvc
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
	p.log.Debug("setting up default RBAC policies")
	if err := SetupDefaultPolicies(p.rbacSvc); err != nil {
		return fmt.Errorf("failed to setup default policies: %w", err)
	}
	p.log.Debug("default RBAC policies configured")

	services := services.NewServices(
		p.baseUIPath,
		p.sessionSvc,
		p.userSvc,
		nil, // p.authSvc
		p.appService,
		nil, // p.orgService
		p.rbacSvc,
		p.apikeyService,
		nil, // p.formsSvc
		p.auditSvc,
		nil, // p.webhookSvc
		p.envSvc,
	)

	p.pagesManager.SetServices(services)

	// Initialize Bridge Manager with existing bridge (preserves CSRF and other settings)
	p.log.Debug("initializing bridge manager")
	bridgeManager := bridgepkg.NewBridgeManager(
		p.fuiBridge, // Use existing bridge from fuiApp (has CSRF disabled)
		services,
		p.log,
		p.baseUIPath,
		p.userSvc,
		p.sessionSvc,
		p.appService,
		p.orgService,
		p.rbacSvc,
		p.apikeyService,
		p.auditSvc,
		p.envSvc,
		p.enabledPlugins,
		p.extensionRegistry, // Pass extension registry for widget access
	)

	// Register all bridge functions with the existing bridge
	if err := bridgeManager.RegisterFunctions(); err != nil {
		return fmt.Errorf("failed to register bridge functions: %w", err)
	}

	// Bridge instance remains the same (p.fuiBridge) - just added functions to it
	p.log.Debug("bridge manager initialized and functions registered")

	// Set bridge reference on extension registry for function registration
	p.extensionRegistry.SetBridge(p.fuiBridge)
	p.log.Debug("bridge reference set on extension registry")

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
		configManager,
	)

	// Store service registry for later access in RegisterRoutes
	p.serviceRegistry = serviceRegistry

	return nil
}

func (p *Plugin) setBasePaths() {
	p.baseUIPath = p.basePath + "/ui"
	p.staticPath = p.baseUIPath + "/static"
}

func (p *Plugin) initializeForgeUI() {
	// Initialize ForgeUI with modern API
	lightTheme := theme.DefaultLight()
	darkTheme := theme.DefaultDark()

	p.setBasePaths()

	// Strip the "static/" prefix so files are accessible at their correct paths
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}

	// Initialize ForgeUI App with asset management
	fuiApp := forgeui.New(
		forgeui.WithDebug(true),
		// forgeui.WithAssetPublicDir("static"),
		// forgeui.WithAssets("static"),
		forgeui.WithEmbedFS(staticFS),
		forgeui.WithBasePath(p.baseUIPath),
		forgeui.WithBridge(
			bridge.WithTimeout(30*time.Second),
			bridge.WithCSRF(false),
		),
		forgeui.WithThemes(&lightTheme, &darkTheme),
		forgeui.WithDefaultLayout(layouts.LayoutRoot),
	)

	p.fuiApp = fuiApp
	p.fuiBridge = fuiApp.Bridge()

	p.pagesManager = pages.NewPagesManager(fuiApp, p.baseUIPath)
	// Note: isMultiAppMode will be set later in Init() after checking plugins
	p.layoutManager = layouts.NewLayoutManager(fuiApp, p.baseUIPath, p.isMultiAppMode, p.enabledPlugins)
}

// RegisterRoles implements the PluginWithRoles optional interface
// This is called automatically during server initialization to register dashboard roles
func (p *Plugin) RegisterRoles(registry interface{}) error {
	roleRegistry, ok := registry.(*rbac.RoleRegistry)
	if !ok {
		return fmt.Errorf("invalid role registry type")
	}

	// Dashboard plugin extends/modifies the default roles with additional permissions
	// Note: Default roles (superadmin, owner, admin, member) are already registered by core
	// We extend them with dashboard-specific permissions
	if err := RegisterDashboardRoles(roleRegistry); err != nil {
		return fmt.Errorf("failed to register dashboard roles: %w", err)
	}

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

// ForgeUIApp returns the ForgeUI App instance
func (p *Plugin) ForgeUIApp() *forgeui.App {
	return p.fuiApp
}

// ForgeUIBridge returns the ForgeUI Bridge instance
func (p *Plugin) ForgeUIBridge() *bridge.Bridge {
	return p.fuiBridge
}

func (p *Plugin) registerForgeUIRoutes(router forge.Router) error {
	// Serve static files through asset pipeline
	// In development mode: no fingerprinting, moderate caching
	// In production mode: automatic fingerprinting, immutable caching
	router.GET("/static/*", p.fuiApp.Assets.Handler())

	return nil
}

// RegisterRoutes registers the dashboard routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("dashboard handler not initialized; call Init first")
	}

	// Pass extension registry to handler and vice versa (for extension rendering)
	p.handler.extensionRegistry = p.extensionRegistry
	p.extensionRegistry.SetHandler(p.handler)

	// Create middleware chain with platform org context
	// Platform org context is injected FIRST, then environment context, then other middleware
	// Note: chain no longer used - extension routes registered via ForgeUI
	_ = func(h func(forge.Context) error) func(forge.Context) error {
		return p.PlatformOrgContext()(p.EnvironmentContext()(p.RequireAuth()(p.RequireAdmin()(p.AuditLog()(p.RateLimit()(h))))))
	}

	// Chain for public auth routes: use app context (multi-app) without auth
	// Note: authlessChain no longer used - extension routes registered via ForgeUI
	_ = func(h func(forge.Context) error) func(forge.Context) error {
		return p.AppContext()(h)
	}

	// Discover and register dashboard extensions from other plugins
	// This happens in RegisterRoutes() after all plugins have completed Init()
	if p.pluginRegistry != nil {
		p.log.Debug("discovering dashboard extensions from plugins...")
		pluginList := p.pluginRegistry.List()

		for _, plugin := range pluginList {
			// Check if plugin implements PluginWithDashboardExtension
			if extPlugin, ok := plugin.(core.PluginWithDashboardExtension); ok {
				ext := extPlugin.DashboardExtension()
				// Lazy initialization ensures extension is never nil
				if ext == nil {
					p.log.Warn("plugin returned nil dashboard extension",
						forge.F("plugin", plugin.ID()))
					continue
				}

				if err := p.extensionRegistry.Register(ext); err != nil {
					p.log.Error("failed to register dashboard extension",
						forge.F("plugin", plugin.ID()),
						forge.F("error", err.Error()))
					continue
				}
				p.log.Info("registered dashboard extension",
					forge.F("plugin", plugin.ID()),
					forge.F("extension", ext.ExtensionID()))
			}
		}
		p.log.Debug("dashboard extension discovery complete",
			forge.F("extensions", len(p.extensionRegistry.List())))
	}

	// Register bridge functions from extensions
	p.log.Debug("registering bridge functions from extensions...")
	extList := p.extensionRegistry.List()
	p.log.Debug("extension list for bridge registration",
		forge.F("count", len(extList)))

	for _, ext := range extList {
		extID := ext.ExtensionID()
		functions := ext.BridgeFunctions()
		p.log.Debug("registering bridge functions for extension",
			forge.F("extension", extID),
			forge.F("functionCount", len(functions)))

		if err := p.extensionRegistry.RegisterBridgeFunctions(ext, p.log); err != nil {
			p.log.Error("failed to register extension bridge functions",
				forge.F("extension", extID),
				forge.F("error", err.Error()))
			// Continue - don't fail entire startup
		} else {
			p.log.Info("successfully registered bridge functions",
				forge.F("extension", extID),
				forge.F("functionCount", len(functions)))
		}
	}
	p.log.Debug("extension bridge functions registered")

	if err := p.layoutManager.RegisterLayouts(); err != nil {
		return fmt.Errorf("failed to register layouts: %w", err)
	}

	if err := p.pagesManager.RegisterPages(); err != nil {
		return fmt.Errorf("failed to register pages: %w", err)
	}

	// Register ForgeUI handler for all HTTP methods (GET, POST, etc.)
	// This handles static assets, bridge API calls, and all ForgeUI routes
	stripPrefix := p.baseUIPath
	uiRouter := router.Group("/ui")

	// Register bridge endpoint FIRST (before wildcard routes) to ensure it's matched
	// CRITICAL: This must be registered BEFORE the wildcard /ui/* routes below
	// Bridge endpoints need to be accessible without dashboard auth middleware
	// Use our custom middleware to enrich the bridge context with user, app, and environment IDs
	bridgeHTTPHandler := p.BridgeContextMiddleware()
	uiRouter.POST("/api/bridge", func(c forge.Context) error {
		bridgeHTTPHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	// Register wildcard ForgeUI handler AFTER specific routes
	// This catches all other /ui/* requests
	// CRITICAL: Skip /api/bridge requests - they're handled by our custom middleware above
	fuiHandler := func(c forge.Context) error {
		path := c.Request().URL.Path
		// Skip bridge requests - they should be handled by our middleware
		if strings.HasSuffix(path, "/api/bridge") || strings.Contains(path, "/api/bridge") {
			p.log.Debug("⚠️ fuiHandler skipping bridge request - should be handled by middleware",
				forge.F("path", path))
			return c.String(http.StatusNotFound, "Bridge endpoint not found in wildcard handler")
		}
		http.StripPrefix(stripPrefix, p.fuiApp.Handler()).ServeHTTP(c.Response(), c.Request())
		return nil
	}

	// Register for all methods that ForgeUI needs
	// These are registered AFTER /api/bridge so they don't intercept it
	router.GET("/ui/*", fuiHandler)
	router.POST("/ui/*", fuiHandler)
	router.PUT("/ui/*", fuiHandler)
	router.DELETE("/ui/*", fuiHandler)
	router.PATCH("/ui/*", fuiHandler)

	if err := p.registerForgeUIRoutes(uiRouter); err != nil {
		return fmt.Errorf("failed to register ForgeUI routes: %w", err)
	}

	// Test route without middleware
	uiRouter.GET("/ping", func(c forge.Context) error {
		return c.JSON(200, map[string]string{"message": "Dashboard plugin is working!"})
	})

	// All page routes are now handled by ForgeUI in pages/pages.go
	// Only keep static asset serving here

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

	// Extension routes are now registered as ForgeUI pages in PagesManager.registerExtensionPages()
	// This provides unified registration, consistent layouts, and better integration with ForgeUI

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
	RedirectURL string `json:"redirect_url" example:"/ui/"`
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
