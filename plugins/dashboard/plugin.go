package dashboard

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/forge"
)

//go:embed dist
var dashboardAssets embed.FS

// Plugin implements the AuthSome plugin interface for serving the dashboard SPA
type Plugin struct {
	handler *Handler
}

// NewPlugin creates a new dashboard plugin instance
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the unique identifier for this plugin
func (p *Plugin) ID() string {
	return "dashboard"
}

// Init initializes the plugin with the AuthSome instance
func (p *Plugin) Init(dep interface{}) error {
	// Initialize the handler with embedded assets using GetAssets for proper fallback
	assets := GetAssets()
	p.handler = NewHandler(assets)
	return nil
}

// RegisterRoutes registers the dashboard routes with the router
func (p *Plugin) RegisterRoutes(router interface{}) error {
	// Check if handler is initialized
	if p.handler == nil {
		return fmt.Errorf("dashboard handler not initialized")
	}

	fmt.Printf("[Dashboard] RegisterRoutes called with router type: %T\n", router)
	switch v := router.(type) {
	case *forge.App:
		// Create a group for dashboard routes using relative path
		grp := v.Group("dashboard")
		// Serve assets under dashboard scope
		grp.GET("/assets/*", p.handler.ServeDashboardAssets)
		// Serve the main dashboard page (SPA entry point)
		grp.GET("/", p.handler.ServeIndex)
		// Serve static assets (JS, CSS, images, etc.)
		grp.GET("/*", p.handler.ServeAssets)

		return nil
	case *forge.Group:
		// Router is already a group with the correct base path, use relative paths
		grp := v.Group("dashboard")
		// Serve assets under dashboard scope
		grp.GET("/assets/*", p.handler.ServeDashboardAssets)
		// Serve the main dashboard page (SPA entry point)
		grp.GET("/", p.handler.ServeIndex)
		// Serve static assets (JS, CSS, images, etc.)
		grp.GET("/*", p.handler.ServeAssets)

		return nil
	case *http.ServeMux:
		// Use pure http.ServeMux routing without Forge wrapper
		// Note: This case is for direct http.ServeMux usage, not for mounted scenarios

		// Serve assets under dashboard scope - use pattern that matches all paths under /dashboard/assets/
		v.HandleFunc("/dashboard/assets/", func(w http.ResponseWriter, r *http.Request) {
			p.handler.ServeDashboardAssetsHTTP(w, r)
		})

		// Serve the main dashboard page and other assets
		v.HandleFunc("/dashboard/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/dashboard/" {
				p.handler.ServeIndexHTTP(w, r)
			} else {
				p.handler.ServeAssetsHTTP(w, r)
			}
		})

		return nil
	default:
		return nil
	}
}

// RegisterHooks registers any hooks this plugin needs
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	// Dashboard plugin doesn't need hooks
	return nil
}

// RegisterServiceDecorators registers any service decorators
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Dashboard plugin doesn't need service decorators
	return nil
}

// Migrate runs any database migrations needed by this plugin
func (p *Plugin) Migrate() error {
	// Dashboard plugin doesn't need database migrations
	return nil
}

// GetAssets returns the embedded dashboard assets
func GetAssets() fs.FS {
	// Try to get the dist subdirectory first
	distFS, err := fs.Sub(dashboardAssets, "dist")
	if err != nil {
		// If dist subdirectory doesn't exist, try to use the root
		// Check if dashboardAssets has any files
		if entries, err := fs.ReadDir(dashboardAssets, "."); err != nil || len(entries) == 0 {
			// If no embedded assets, create a minimal in-memory filesystem
			return createFallbackFS()
		}
		return dashboardAssets
	}
	return distFS
}

// createFallbackFS creates a minimal in-memory filesystem with basic HTML
func createFallbackFS() fs.FS {
	// For now, return nil and handle it in the handler
	// In a real implementation, you'd create an in-memory FS
	return nil
}
