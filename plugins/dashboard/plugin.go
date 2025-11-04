package dashboard

import (
	"embed"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins"
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
}

// NewPlugin creates a new dashboard plugin instance
func NewPlugin() *Plugin {
	return &Plugin{}
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
		GetBasePath() string
		GetPluginRegistry() *plugins.Registry
	}

	authInstance, ok := dep.(authInstanceInterface)
	if !ok {
		return fmt.Errorf("dashboard plugin requires auth instance with GetDB, GetServiceRegistry, GetBasePath, and GetPluginRegistry methods")
	}

	serviceRegistry := authInstance.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}

	// Get database for repository initialization
	db := authInstance.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	// Get base path (e.g., "/api/auth")
	p.basePath = authInstance.GetBasePath()
	if p.basePath == "" {
		p.basePath = ""
	}

	// Get plugin registry to check which plugins are enabled
	fmt.Printf("[Dashboard] ========== PLUGIN DETECTION START ==========\n")
	p.enabledPlugins = make(map[string]bool)

	pluginRegistry := authInstance.GetPluginRegistry()
	if pluginRegistry != nil {
		pluginList := pluginRegistry.List()
		fmt.Printf("[Dashboard] ✅ Plugin registry found, detected %d plugins\n", len(pluginList))
		for _, plugin := range pluginList {
			pluginID := plugin.ID()
			p.enabledPlugins[pluginID] = true
			fmt.Printf("[Dashboard]    ✓ Enabled plugin: %s\n", pluginID)
		}
	} else {
		fmt.Printf("[Dashboard] ⚠️  Plugin registry is nil\n")
	}

	fmt.Printf("[Dashboard] Final enabled plugins count: %d\n", len(p.enabledPlugins))
	fmt.Printf("[Dashboard] Final enabled plugins map: %v\n", p.enabledPlugins)
	fmt.Printf("[Dashboard] ========== PLUGIN DETECTION END ==========\n")

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
		if orgSvc, ok := orgSvcInterface.(*organization.Service); ok {
			p.orgService = orgSvc
			p.isSaaSMode = serviceRegistry.IsMultiTenant()
		}
	}

	// Initialize Permission Checker
	userRoleRepo := repository.NewUserRoleRepository(db)
	p.permChecker = NewPermissionChecker(p.rbacSvc, userRoleRepo)
	fmt.Println("[Dashboard] ✅ Permission checker initialized")

	// Initialize CSRF Protector
	csrfProtector, err := NewCSRFProtector()
	if err != nil {
		return fmt.Errorf("failed to initialize CSRF protector: %w", err)
	}
	p.csrfProtector = csrfProtector
	fmt.Println("[Dashboard] ✅ CSRF protector initialized")

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
	)

	fmt.Println("[Dashboard] ✅ Handler initialized with RBAC and CSRF protection")

	return nil
}

// RegisterRoutes registers the dashboard routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("dashboard handler not initialized; call Init first")
	}

	fmt.Printf("[Dashboard] Registering routes with basePath: %s\n", p.basePath)

	// Create middleware chain
	chain := func(h func(forge.Context) error) func(forge.Context) error {
		return p.RequireAuth()(p.RequireAdmin()(p.AuditLog()(p.RateLimit()(h))))
	}

	// Test route without middleware
	router.GET("/dashboard/ping", func(c forge.Context) error {
		return c.JSON(200, map[string]string{"message": "Dashboard plugin is working!"})
	})

	// Public routes (no auth middleware) - these must be accessible without authentication
	publicRoutes := []struct {
		method  string
		path    string
		handler func(forge.Context) error
	}{
		{"GET", "/dashboard/login", p.handler.ServeLogin},
		{"POST", "/dashboard/login", p.handler.HandleLogin},
		{"GET", "/dashboard/signup", p.handler.ServeSignup},
		{"POST", "/dashboard/signup", p.handler.HandleSignup},
		{"POST", "/dashboard/logout", p.handler.HandleLogout},
		{"GET", "/dashboard/logout", p.handler.HandleLogout}, // Support GET for convenience
	}

	for _, route := range publicRoutes {
		switch route.method {
		case "GET":
			router.GET(route.path, route.handler)
		case "POST":
			router.POST(route.path, route.handler)
		}
	}

	// Dashboard pages (with auth middleware)
	router.GET("/dashboard/", chain(p.handler.ServeDashboard))
	router.GET("/dashboard/users", chain(p.handler.ServeUsers))
	router.GET("/dashboard/users/:id", chain(p.handler.ServeUserDetail))
	router.GET("/dashboard/users/:id/edit", chain(p.handler.ServeUserEdit))
	router.POST("/dashboard/users/:id/edit", chain(p.handler.HandleUserEdit))
	router.POST("/dashboard/users/:id/delete", chain(p.handler.HandleUserDelete))
	router.GET("/dashboard/sessions", chain(p.handler.ServeSessions))
	router.POST("/dashboard/sessions/:id/revoke", chain(p.handler.HandleRevokeSession))
	router.POST("/dashboard/sessions/revoke-user", chain(p.handler.HandleRevokeUserSessions))
	router.GET("/dashboard/settings", chain(p.handler.ServeSettings))

	// Static assets (no auth required)
	router.GET("/dashboard/static/*", p.handler.ServeStatic)

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
