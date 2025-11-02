package plugins

import (
	"fmt"

	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
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
	Init(auth interface{}) error

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

// AuthInterface defines the interface that plugins can use to access Auth instance features
// This avoids direct coupling to the authsome package in plugin code
type AuthInterface interface {
	GetDB() interface{}                            // Returns *bun.DB
	GetForgeApp() interface{}                      // Returns forge.App
	GetServiceRegistry() *registry.ServiceRegistry // Returns service registry
	GetHookRegistry() *hooks.HookRegistry          // Returns hook registry
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
