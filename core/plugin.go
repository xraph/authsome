package core

import (
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/forge"
)

// Plugin defines the interface for authentication plugins
//
// Plugins receive the Auth instance during Init, which provides access to:
// - Database: auth.GetDB()
// - Service Registry: auth.GetServiceRegistry()
// - Forge App: auth.GetForgeApp()
// - DI Container: auth.GetForgeApp().Container()
//
// Plugins can resolve services from the DI container using the helper functions
// in the authsome package (e.g., authsome.ResolveUserService, authsome.ResolveAuditService)
type Plugin interface {
	// ID returns the unique plugin identifier
	ID() string

	// Init initializes the plugin with the auth instance
	// The auth parameter will be an *authsome.Auth instance
	// Use type assertion: auth.(*authsome.Auth) or use interface methods
	Init(auth Authsome) error

	// RegisterRoutes registers plugin routes with the router
	// Routes are scoped to the auth base path (e.g., /api/auth)
	RegisterRoutes(router forge.Router) error

	// RegisterHooks registers plugin hooks with the hook registry
	// Hooks allow plugins to intercept auth lifecycle events
	RegisterHooks(hooks *hooks.HookRegistry) error

	// RegisterServiceDecorators allows plugins to replace core services with decorated versions
	// This enables plugins to enhance or modify core functionality
	RegisterServiceDecorators(services *registry.ServiceRegistry) error

	// Migrate runs plugin migrations
	// Create database tables and indexes needed by the plugin
	Migrate() error
}

// Optional Plugin Interfaces
// Plugins can optionally implement these interfaces to enable additional functionality

// PluginWithRoles is an optional interface that plugins can implement to register roles
// in the role bootstrap system. Roles registered here will be automatically bootstrapped
// to the platform organization during server startup.
//
// Example:
//
//	func (p *MyPlugin) RegisterRoles(registry *rbac.RoleRegistry) error {
//	    return registry.RegisterRole(&rbac.RoleDefinition{
//	        Name:        "custom_role",
//	        Description: "Custom Role",
//	        Permissions: []string{"view on custom_resource"},
//	    })
//	}
type PluginWithRoles interface {
	Plugin
	RegisterRoles(registry rbac.RoleRegistryInterface) error // registry is *rbac.RoleRegistry
}

// PluginWithDashboardExtension is an optional interface that plugins can implement
// to extend the dashboard plugin with custom navigation items, routes, and pages.
//
// This allows plugins to add their own screens to the dashboard without modifying
// the dashboard plugin code. The dashboard extension is registered during plugin
// initialization and provides:
// - Navigation items (main nav, settings, user dropdown)
// - Custom routes under /dashboard/app/:appId/
// - Settings sections
// - Dashboard widgets
//
// Example:
//
//	import "github.com/xraph/authsome/core/ui"
//
//	func (p *MyPlugin) DashboardExtension() ui.DashboardExtension {
//	    return &MyDashboardExtension{service: p.service}
//	}
type PluginWithDashboardExtension interface {
	Plugin
	// DashboardExtension returns a dashboard extension instance
	// The extension must implement the ui.DashboardExtension interface
	DashboardExtension() ui.DashboardExtension
}

type PluginRegistry interface {
	Register(p Plugin) error
	Get(id string) (Plugin, bool)
	List() []Plugin
}

// PluginDependencies defines optional interface for plugins to declare their dependencies
// Plugins implementing this interface will have their dependencies validated before initialization
// Dependencies are declared by plugin ID and must be registered before the dependent plugin
//
// Example:
//
//	func (p *DashboardPlugin) Dependencies() []string {
//	    return []string{"multiapp"} // Dashboard requires multiapp plugin
//	}
type PluginWithDependencies interface {
	Plugin
	// Dependencies returns a list of plugin IDs that must be initialized before this plugin
	Dependencies() []string
}
