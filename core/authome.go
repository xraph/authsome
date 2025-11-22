package core

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Authsome defines the public API for the Auth instance
// This interface enables better testability and allows for alternative implementations
type Authsome interface {
	// Initialize initializes all core services
	Initialize(ctx context.Context) error

	// Mount mounts the auth routes to the Forge router
	Mount(router forge.Router, basePath string) error

	// RegisterPlugin registers a plugin
	RegisterPlugin(plugin Plugin) error

	// GetConfig returns the auth config
	GetConfig() Config

	// GetDB returns the database instance
	GetDB() *bun.DB

	// GetForgeApp returns the forge application instance
	GetForgeApp() forge.App

	// GetServiceRegistry returns the service registry for plugins
	GetServiceRegistry() *registry.ServiceRegistry

	// GetHookRegistry returns the hook registry for plugins
	GetHookRegistry() *hooks.HookRegistry

	// GetBasePath returns the base path for AuthSome routes
	GetBasePath() string

	// GetPluginRegistry returns the plugin registry
	GetPluginRegistry() PluginRegistry

	// Logger returns the logger for AuthSome
	Logger() forge.Logger

	// IsPluginEnabled checks if a plugin is registered and enabled
	IsPluginEnabled(pluginID string) bool

	// Repository returns the repository instance
	Repository() repository.Repository

	// AuthMiddleware returns the optional authentication middleware
	// This middleware populates the auth context with API key and/or session data
	AuthMiddleware() func(func(forge.Context) error) func(forge.Context) error
}
