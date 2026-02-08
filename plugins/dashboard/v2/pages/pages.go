package pages

import (
	"fmt"

	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/plugins/dashboard/v2/layouts"
	"github.com/xraph/authsome/plugins/dashboard/v2/services"
	"github.com/xraph/forgeui"
	"github.com/xraph/forgeui/router"
)

// ExtensionRegistry is an interface to avoid import cycles
type ExtensionRegistry interface {
	List() []ui.DashboardExtension
	GetDashboardWidgets() []ui.DashboardWidget
}

type PagesManager struct {
	fuiApp            *forgeui.App
	baseUIPath        string
	services          *services.Services
	extensionRegistry ExtensionRegistry
}

func NewPagesManager(fuiApp *forgeui.App, baseUIPath string) *PagesManager {
	return &PagesManager{
		fuiApp:     fuiApp,
		baseUIPath: baseUIPath,
	}
}

func (p *PagesManager) RegisterPages() error {
	if err := p.registerAuthPages(); err != nil {
		return err
	}

	if err := p.registerDashboardPages(); err != nil {
		return err
	}

	// Register extension pages as ForgeUI pages
	if p.extensionRegistry != nil {
		if err := p.registerExtensionPages(); err != nil {
			return fmt.Errorf("failed to register extension pages: %w", err)
		}

		// Register settings pages as routes
		if err := p.registerExtensionSettingsPages(); err != nil {
			return fmt.Errorf("failed to register extension settings pages: %w", err)
		}
	}

	return nil
}

func (p *PagesManager) registerAuthPages() error {
	authGroup := p.fuiApp.Group("/auth").
		Middleware(p.services.AuthlessMiddleware).
		Layout(layouts.LayoutAuthless)

	authGroup.Page("/login").Handler(p.LoginPage).Register()
	authGroup.Page("/login").Method("POST").Handler(p.HandleLogin).Register()
	authGroup.Page("/signup").Handler(p.RegisterPage).Register()
	authGroup.Page("/signup").Method("POST").Handler(p.HandleSignup).Register()
	authGroup.Page("/forgot-password").Handler(p.ForgotPasswordPage).Register()
	authGroup.Page("/reset-password").Handler(p.ResetPasswordPage).Register()
	authGroup.Page("/logout").Handler(p.LogoutPage).Register()

	return nil
}

func (p *PagesManager) registerDashboardPages() error {

	protectedGroup := p.fuiApp.Group("").
		Middleware(p.services.AuthMiddleware).
		Layout(layouts.LayoutApp)

	dashboardGroup := p.fuiApp.Group("/app").
		Middleware(p.services.AuthMiddleware).
		Middleware(p.services.AppContextMiddleware).
		Layout(layouts.LayoutDashboard)

	// Settings group with horizontal navigation layout
	settingsGroup := p.fuiApp.Group("/app").
		Middleware(p.services.AuthMiddleware).
		Middleware(p.services.AppContextMiddleware).
		Layout(layouts.LayoutSettings)

	// Dashboard index - app selection
	protectedGroup.Page("/").Handler(p.DashboardIndexPage).Register()

	// Create app page - redirects to index with modal open
	protectedGroup.Page("/apps/new").Handler(p.CreateAppPage).Register()

	// User profile page - available to all authenticated users
	protectedGroup.Page("/profile").Handler(p.ProfilePage).Register()

	// App-specific dashboard
	dashboardGroup.Page("/:appId").Handler(p.DashboardHomePage).Register()

	// Users management
	dashboardGroup.Page("/:appId/users").Handler(p.UsersListPage).Register()
	dashboardGroup.Page("/:appId/users/:userId").Handler(p.UserDetailPage).Register()
	dashboardGroup.Page("/:appId/users/:userId/edit").Handler(p.UserEditPage).Register()

	// Sessions management
	dashboardGroup.Page("/:appId/sessions").Handler(p.SessionsListPage).Register()

	// Organizations
	dashboardGroup.Page("/:appId/organizations").Handler(p.OrganizationsListPage).Register()
	dashboardGroup.Page("/:appId/organizations/:orgId").Handler(p.OrganizationDetailPage).Register()
	dashboardGroup.Page("/:appId/organizations/:orgId/edit").Handler(p.OrganizationEditPage).Register()

	// Settings - use settingsGroup for horizontal tab navigation
	settingsGroup.Page("/:appId/settings").Handler(p.SettingsGeneralPage).Register()
	settingsGroup.Page("/:appId/settings/general").Handler(p.SettingsGeneralPage).Register()
	settingsGroup.Page("/:appId/settings/security").Handler(p.SettingsSecurityPage).Register()
	// Note: /settings/api-keys is handled by the apikey plugin extension

	// Platform management (admin only)
	protectedGroup.Page("/platform/apps").Handler(p.AppsManagementPage).Register()
	protectedGroup.Page("/platform/plugins").Handler(p.PluginsManagementPage).Register()
	protectedGroup.Page("/platform/config").Handler(p.ConfigViewerPage).Register()

	// Environments (if multiapp enabled)
	dashboardGroup.Page("/:appId/environments").Handler(p.EnvironmentsManagementPage).Register()
	dashboardGroup.Page("/:appId/environments/:envId").Handler(p.EnvironmentDetailPage).Register()

	// Audit logs
	dashboardGroup.Page("/:appId/audit").Handler(p.AuditLogViewerPage).Register()

	// Plugins management
	dashboardGroup.Page("/:appId/plugins").Handler(p.PluginsPage).Register()

	// Error page
	protectedGroup.Page("/error").Handler(p.ErrorPage).Register()

	return nil
}

// registerExtensionPages registers all extension routes as ForgeUI pages
func (p *PagesManager) registerExtensionPages() error {
	extensions := p.extensionRegistry.List()

	for _, ext := range extensions {
		routes := ext.Routes()

		for _, route := range routes {
			if err := p.registerExtensionRoute(ext, route); err != nil {
				return fmt.Errorf("failed to register %s route %s: %w",
					ext.ExtensionID(), route.Path, err)
			}
		}
	}

	return nil
}

// registerExtensionRoute registers a single extension route as a ForgeUI page
func (p *PagesManager) registerExtensionRoute(ext ui.DashboardExtension, route ui.Route) error {
	// Safety check
	if p.services == nil {
		return fmt.Errorf("services not initialized - cannot register routes")
	}

	// Determine group based on route requirements
	var group *router.Group

	// Check if this is a settings route (path starts with /settings/)
	isSettingsRoute := len(route.Path) >= 10 && route.Path[:10] == "/settings/"

	if route.RequireAdmin {
		// Admin routes: auth + app context + appropriate layout
		if isSettingsRoute {
			// Settings routes get settings layout with horizontal tabs
			group = p.fuiApp.Group("/app").
				Middleware(p.services.AuthMiddleware).
				Middleware(p.services.AppContextMiddleware).
				Layout(layouts.LayoutSettings)
		} else {
			// Regular admin routes get dashboard layout
			group = p.fuiApp.Group("/app").
				Middleware(p.services.AuthMiddleware).
				Middleware(p.services.AppContextMiddleware).
				Layout(layouts.LayoutDashboard)
		}
	} else if route.RequireAuth {
		// Authenticated routes: auth + app context + app layout
		group = p.fuiApp.Group("/app").
			Middleware(p.services.AuthMiddleware).
			Middleware(p.services.AppContextMiddleware).
			Layout(layouts.LayoutApp)
	} else {
		// Public routes: authless layout
		group = p.fuiApp.Group("/app").
			Middleware(p.services.AuthlessMiddleware).
			Layout(layouts.LayoutAuthless)
	}

	// Build full path: /:appId + route.Path
	fullPath := "/:appId" + route.Path

	// Register based on method
	method := route.Method
	if method == "" {
		method = "GET" // Default to GET if not specified
	}

	// Register the page - matches dashboard's own pattern exactly
	switch method {
	case "GET":
		group.Page(fullPath).Handler(route.Handler).Register()
	case "POST":
		// POST routes for form submissions
		group.Page(fullPath).Method("POST").Handler(route.Handler).Register()
	case "PUT", "DELETE", "PATCH":
		// Other HTTP methods
		group.Page(fullPath).Method(method).Handler(route.Handler).Register()
	default:
		return fmt.Errorf("unsupported method %s for ForgeUI route", method)
	}

	return nil
}

// registerExtensionSettingsPages registers settings pages from extensions as ForgeUI routes
// This automatically creates routes for all SettingsPages() declared by extensions
// that don't have explicit routes defined in Routes()
func (p *PagesManager) registerExtensionSettingsPages() error {
	extensions := p.extensionRegistry.List()

	for _, ext := range extensions {
		settingsPages := ext.SettingsPages()
		for _, page := range settingsPages {
			// Find the matching route from ext.Routes()
			routes := ext.Routes()
			var routeExists bool

			// Look for a route that matches the settings page path
			expectedPath := "/settings/" + page.Path
			for _, route := range routes {
				if route.Path == expectedPath {
					routeExists = true
					break
				}
			}

			if routeExists {
				// Route already registered via Routes(), skip
				continue
			}

			// Settings page metadata exists but no route handler
			// This is expected behavior - SettingsPages() is for navigation metadata
			// Actual route handlers should be defined in Routes()
		}
	}

	return nil
}
