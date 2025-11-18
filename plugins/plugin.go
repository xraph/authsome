package plugins

import (
	"fmt"
	"strings"

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

// PluginWithDependencies is an optional interface that plugins can implement to declare
// their dependencies. Plugins with dependencies will be automatically initialized after
// their dependencies using topological sort.
//
// Example:
//
//	func (p *DashboardPlugin) Dependencies() []string {
//	    return []string{"multiapp"} // Dashboard requires multiapp plugin
//	}
type PluginWithDependencies = core.PluginWithDependencies

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

// List returns all registered plugins in registration order
func (r *Registry) List() []Plugin {
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// ListSorted returns all registered plugins sorted by dependencies using topological sort
// Plugins with dependencies will be placed after their dependencies
// Returns an error if circular dependencies are detected or if a dependency is missing
func (r *Registry) ListSorted() ([]Plugin, error) {
	// Build dependency graph
	graph := make(map[string][]string) // plugin ID -> list of plugins that depend on it
	inDegree := make(map[string]int)   // plugin ID -> number of dependencies
	pluginMap := make(map[string]Plugin)

	// Initialize all plugins in the graph
	for id, plugin := range r.plugins {
		pluginMap[id] = plugin
		if _, exists := inDegree[id]; !exists {
			inDegree[id] = 0
		}
		if _, exists := graph[id]; !exists {
			graph[id] = []string{}
		}
	}

	// Build edges based on dependencies
	for id, plugin := range r.plugins {
		// Check if plugin implements PluginWithDependencies
		if depPlugin, ok := plugin.(core.PluginWithDependencies); ok {
			deps := depPlugin.Dependencies()
			inDegree[id] = len(deps)

			for _, depID := range deps {
				// Validate dependency exists
				if _, exists := r.plugins[depID]; !exists {
					return nil, fmt.Errorf("plugin '%s' depends on '%s' which is not registered", id, depID)
				}

				// Add edge: depID -> id (depID must come before id)
				graph[depID] = append(graph[depID], id)
			}
		}
	}

	// Kahn's algorithm for topological sort with cycle detection
	queue := make([]string, 0)
	result := make([]Plugin, 0, len(r.plugins))

	// Find all plugins with no dependencies (in-degree = 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Process queue
	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, pluginMap[current])

		// Reduce in-degree for all dependent plugins
		for _, dependent := range graph[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check for circular dependencies
	if len(result) != len(r.plugins) {
		// Find plugins involved in cycle
		cyclePlugins := make([]string, 0)
		for id, degree := range inDegree {
			if degree > 0 {
				cyclePlugins = append(cyclePlugins, id)
			}
		}
		return nil, fmt.Errorf("circular dependency detected among plugins: %s", strings.Join(cyclePlugins, ", "))
	}

	return result, nil
}

// ValidateDependencies checks if all plugin dependencies are satisfied
// Returns an error describing any issues found
func (r *Registry) ValidateDependencies() error {
	for id, plugin := range r.plugins {
		// Check if plugin implements PluginWithDependencies
		if depPlugin, ok := plugin.(core.PluginWithDependencies); ok {
			deps := depPlugin.Dependencies()
			for _, depID := range deps {
				if _, exists := r.plugins[depID]; !exists {
					return fmt.Errorf("plugin '%s' depends on '%s' which is not registered", id, depID)
				}
			}
		}
	}

	// Check for circular dependencies by attempting sort
	_, err := r.ListSorted()
	if err != nil {
		return err
	}

	return nil
}
