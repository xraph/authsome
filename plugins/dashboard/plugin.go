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
	},
		forge.WithName("dashboard.ping"),
		forge.WithSummary("Dashboard health check"),
		forge.WithDescription("Verify that the dashboard plugin is loaded and working"),
		forge.WithResponseSchema(200, "Dashboard is working", DashboardPingResponse{}),
		forge.WithTags("Dashboard", "Health"),
	)

	// Public routes (no auth middleware) - these must be accessible without authentication
	router.GET("/dashboard/login", p.handler.ServeLogin,
		forge.WithName("dashboard.login.page"),
		forge.WithSummary("Login page"),
		forge.WithDescription("Render the admin dashboard login page"),
		forge.WithResponseSchema(200, "Login page HTML", DashboardHTMLResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
	)

	router.POST("/dashboard/login", p.handler.HandleLogin,
		forge.WithName("dashboard.login.submit"),
		forge.WithSummary("Process login"),
		forge.WithDescription("Authenticate admin user and create dashboard session"),
		forge.WithResponseSchema(200, "Login successful", DashboardLoginResponse{}),
		forge.WithResponseSchema(401, "Invalid credentials", DashboardErrorResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
		forge.WithValidation(true),
	)

	router.GET("/dashboard/signup", p.handler.ServeSignup,
		forge.WithName("dashboard.signup.page"),
		forge.WithSummary("Signup page"),
		forge.WithDescription("Render the admin dashboard signup page"),
		forge.WithResponseSchema(200, "Signup page HTML", DashboardHTMLResponse{}),
		forge.WithTags("Dashboard", "Authentication"),
	)

	router.POST("/dashboard/signup", p.handler.HandleSignup,
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
