package plugins

import (
	"fmt"

	"github.com/xraph/authsome/core"
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
type Plugin = core.Plugin

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
type PluginWithRoles = core.PluginWithRoles

// Registry manages registered plugins
type Registry struct {
	plugins map[string]Plugin
}

type PluginRegistry = core.PluginRegistry

// NewRegistry creates a new plugin registry
func NewRegistry() PluginRegistry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

// Register registers a plugin
func (r *Registry) Register(p Plugin) error {
	if _, exists := r.plugins[p.ID()]; exists {
		return fmt.Errorf("plugin %s already registered", p.ID())
	}
	r.plugins[p.ID()] = p
	return nil
}

// Get retrieves a plugin by ID
func (r *Registry) Get(id string) (Plugin, bool) {
	p, exists := r.plugins[id]
	return p, exists
}

// List returns all registered plugins
func (r *Registry) List() []Plugin {
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}
