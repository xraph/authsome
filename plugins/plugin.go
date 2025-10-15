package plugins

import (
    "fmt"

    "github.com/xraph/authsome/core/hooks"
    "github.com/xraph/authsome/core/registry"
)

// Plugin defines the interface for authentication plugins
type Plugin interface {
    // ID returns the unique plugin identifier
    ID() string

    // Init initializes the plugin with the auth instance
    Init(auth interface{}) error

    // RegisterRoutes registers plugin routes with the router
    RegisterRoutes(router interface{}) error

    // RegisterHooks registers plugin hooks with the hook registry
    RegisterHooks(hooks *hooks.HookRegistry) error

    // RegisterServiceDecorators allows plugins to replace core services with decorated versions
    RegisterServiceDecorators(services *registry.ServiceRegistry) error

    // Migrate runs plugin migrations
    Migrate() error
}

// Registry manages registered plugins
type Registry struct {
    plugins map[string]Plugin
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
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