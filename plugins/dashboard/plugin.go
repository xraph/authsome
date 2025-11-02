package dashboard

import (
	"embed"
	"fmt"
	"html/template"

	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

//go:embed templates/* static/css/* static/js/*.js
var assets embed.FS

// Plugin implements the dashboard plugin for AuthSome
type Plugin struct {
	handler        *Handler
	templates      *template.Template
	userSvc        *user.Service
	sessionSvc     *session.Service
	auditSvc       *audit.Service
	rbacSvc        *rbac.Service
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
	// Try to extract service registry using reflection-like approach
	type serviceRegistryGetter interface {
		GetServiceRegistry() *registry.ServiceRegistry
	}
	type basePathGetter interface {
		GetBasePath() string
	}
	type pluginRegistryGetter interface {
		GetPluginRegistry() interface{}
	}

	// Get service registry
	srGetter, ok := dep.(serviceRegistryGetter)
	if !ok {
		return fmt.Errorf("dashboard plugin requires auth instance with GetServiceRegistry")
	}

	serviceRegistry := srGetter.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}

	// Get base path (e.g., "/api/auth")
	if bpGetter, ok := dep.(basePathGetter); ok {
		p.basePath = bpGetter.GetBasePath()
		if p.basePath == "" {
			p.basePath = ""
		}
	}

	// Get plugin registry to check which plugins are enabled
	p.enabledPlugins = make(map[string]bool)
	if prGetter, ok := dep.(pluginRegistryGetter); ok {
		pluginRegistry := prGetter.GetPluginRegistry()

		// Build map of enabled plugins for easy template access
		if pluginRegistry != nil {
			// Use type assertion to access List() method
			type pluginRegistryInterface interface {
				List() []interface{}
			}
			if reg, ok := pluginRegistry.(pluginRegistryInterface); ok {
				plugins := reg.List()
				for _, plugin := range plugins {
					// Use type assertion to get ID
					type pluginIDInterface interface {
						ID() string
					}
					if plg, ok := plugin.(pluginIDInterface); ok {
						p.enabledPlugins[plg.ID()] = true
					}
				}
			}
		}
	}

	fmt.Printf("[Dashboard] Detected %d enabled plugins\n", len(p.enabledPlugins))

	// Get required services from registry using specific getters
	userSvcInterface := serviceRegistry.UserService()
	if userSvcInterface == nil {
		return fmt.Errorf("user service not found in registry")
	}
	p.userSvc, ok = userSvcInterface.(*user.Service)
	if !ok {
		return fmt.Errorf("invalid user service type")
	}

	sessionSvcInterface := serviceRegistry.SessionService()
	if sessionSvcInterface == nil {
		return fmt.Errorf("session service not found in registry")
	}
	p.sessionSvc, ok = sessionSvcInterface.(*session.Service)
	if !ok {
		return fmt.Errorf("invalid session service type")
	}

	p.auditSvc = serviceRegistry.AuditService()
	if p.auditSvc == nil {
		return fmt.Errorf("audit service not found in registry")
	}

	p.rbacSvc = serviceRegistry.RBACService()
	if p.rbacSvc == nil {
		return fmt.Errorf("rbac service not found in registry")
	}

	// Parse templates from embedded filesystem
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(assets, "templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}
	p.templates = tmpl

	// Initialize handler with services, templates, base path, and enabled plugins
	p.handler = NewHandler(p.templates, assets, p.userSvc, p.sessionSvc, p.auditSvc, p.rbacSvc, p.basePath, p.enabledPlugins)

	fmt.Println("[Dashboard] Initialized with RBAC and CSRF protection")

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

// templateFuncs returns template helper functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
		"dec": func(i int) int {
			return i - 1
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"len": func(s interface{}) int {
			switch v := s.(type) {
			case []interface{}:
				return len(v)
			case []string:
				return len(v)
			default:
				return 0
			}
		},
		"slice": func(s string, start, end int) string {
			if start < 0 || start >= len(s) {
				return ""
			}
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"upper": func(s string) string {
			if len(s) == 0 {
				return ""
			}
			return string(s[0] - 32)
		},
		"formatDate": func(t interface{}) string {
			// TODO: Implement proper date formatting
			return fmt.Sprintf("%v", t)
		},
	}
}
