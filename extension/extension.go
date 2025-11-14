package extension

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/plugins"
	"github.com/xraph/forge"
)

// Extension implements the Forge extension interface for AuthSome
type Extension struct {
	*forge.BaseExtension
	config  Config
	auth    *authsome.Auth
	plugins []plugins.Plugin
}

// NewExtension creates a new AuthSome extension with optional configuration
func NewExtension(opts ...ConfigOption) forge.Extension {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}

	base := forge.NewBaseExtension(
		"authsome",
		"1.0.0",
		"Enterprise-grade authentication and authorization system",
	)

	return &Extension{
		BaseExtension: base,
		config:        config,
		plugins:       []plugins.Plugin{},
	}
}

// Register registers the extension with the Forge application
func (e *Extension) Register(app forge.App) error {
	if err := e.BaseExtension.Register(app); err != nil {
		return err
	}

	e.Logger().Info("registering authsome extension")

	// Load configuration from Forge config system if available
	finalConfig := e.config
	if err := e.LoadConfig("authsome", &finalConfig, e.config, DefaultConfig(), e.config.RequireConfig); err != nil {
		if e.config.RequireConfig {
			return fmt.Errorf("authsome: failed to load required config: %w", err)
		}
		e.Logger().Warn("authsome: using default/programmatic config", forge.F("error", err.Error()))
	}
	e.config = finalConfig

	// Build AuthSome options
	opts := []authsome.Option{
		authsome.WithForgeApp(app),
		authsome.WithBasePath(e.config.BasePath),
	}

	// Database configuration - try multiple sources
	if e.config.Database != nil {
		// Direct database provided
		opts = append(opts, authsome.WithDatabase(e.config.Database))
		e.Logger().Info("authsome: using provided database connection")
	} else if e.config.DatabaseName != "" {
		// Use specific database from DatabaseManager
		manager, err := authsome.ResolveDatabaseManager(app.Container())
		if err != nil {
			return fmt.Errorf("authsome: failed to resolve database manager: %w", err)
		}
		opts = append(opts, authsome.WithDatabaseManager(manager, e.config.DatabaseName))
		e.Logger().Info("authsome: using database from manager", forge.F("database", e.config.DatabaseName))
	} else {
		// Auto-resolve from Forge DI
		opts = append(opts, authsome.WithDatabaseFromForge())
		e.Logger().Info("authsome: auto-resolving database from Forge DI")
	}

	// Add optional configuration
	if len(e.config.TrustedOrigins) > 0 {
		opts = append(opts, authsome.WithTrustedOrigins(e.config.TrustedOrigins))
	}
	if e.config.Secret != "" {
		opts = append(opts, authsome.WithSecret(e.config.Secret))
	}
	if e.config.SecurityConfig != nil {
		opts = append(opts, authsome.WithSecurityConfig(*e.config.SecurityConfig))
	}
	if e.config.RateLimitConfig != nil {
		opts = append(opts, authsome.WithRateLimitConfig(*e.config.RateLimitConfig))
	}
	if e.config.RateLimitStorage != nil {
		opts = append(opts, authsome.WithRateLimitStorage(e.config.RateLimitStorage))
	}
	if e.config.GeoIPProvider != nil {
		opts = append(opts, authsome.WithGeoIPProvider(e.config.GeoIPProvider))
	}
	opts = append(opts, authsome.WithRBACEnforcement(e.config.RBACEnforce))

	// Create AuthSome instance
	e.auth = authsome.New(opts...)

	// Register plugins
	for _, plugin := range e.config.Plugins {
		if err := e.auth.RegisterPlugin(plugin); err != nil {
			return fmt.Errorf("authsome: failed to register plugin %s: %w", plugin.ID(), err)
		}
		e.Logger().Info("authsome: registered plugin", forge.F("plugin", plugin.ID()))
	}

	e.Logger().Info("authsome extension registered successfully")
	return nil
}

// Start starts the extension and initializes AuthSome
func (e *Extension) Start(ctx context.Context) error {
	e.Logger().Info("starting authsome extension")

	if e.auth == nil {
		return fmt.Errorf("authsome: not registered properly")
	}

	// Initialize AuthSome
	if err := e.auth.Initialize(ctx); err != nil {
		return fmt.Errorf("authsome: initialization failed: %w", err)
	}

	// Mount routes
	app := e.App()
	if app == nil {
		return fmt.Errorf("authsome: forge app not available")
	}

	if err := e.auth.Mount(app.Router(), e.config.BasePath); err != nil {
		return fmt.Errorf("authsome: failed to mount routes: %w", err)
	}

	e.MarkStarted()
	e.Logger().Info("authsome extension started successfully",
		forge.F("basePath", e.config.BasePath),
		forge.F("plugins", len(e.config.Plugins)),
	)

	return nil
}

// Stop stops the extension
func (e *Extension) Stop(ctx context.Context) error {
	e.Logger().Info("stopping authsome extension")

	// AuthSome doesn't require explicit shutdown currently
	// But we mark it as stopped for proper lifecycle management

	e.MarkStopped()
	e.Logger().Info("authsome extension stopped")
	return nil
}

// Health checks the extension health
func (e *Extension) Health(ctx context.Context) error {
	if e.auth == nil {
		return fmt.Errorf("authsome not initialized")
	}

	// AuthSome is healthy if it's initialized
	// Individual service health can be checked through their respective interfaces
	return nil
}

// Auth returns the AuthSome instance
// Use this to access AuthSome programmatically after extension is registered
func (e *Extension) Auth() *authsome.Auth {
	return e.auth
}

// RegisterPlugin registers a plugin before Start is called
func (e *Extension) RegisterPlugin(plugin plugins.Plugin) error {
	if e.auth != nil {
		// Already initialized, register directly
		return e.auth.RegisterPlugin(plugin)
	}
	// Not initialized yet, add to pending plugins
	e.config.Plugins = append(e.config.Plugins, plugin)
	return nil
}

// GetPluginRegistry returns the plugin registry for plugin detection
// This is used by the dashboard plugin to detect which plugins are enabled
func (e *Extension) GetPluginRegistry() plugins.PluginRegistry {
	if e.auth == nil {
		return nil
	}
	return e.auth.GetPluginRegistry()
}

// GetServiceRegistry returns the service registry
// This is used by plugins that need access to core services
func (e *Extension) GetServiceRegistry() *registry.ServiceRegistry {
	if e.auth == nil {
		return nil
	}
	return e.auth.GetServiceRegistry()
}

// GetBasePath returns the configured base path
// This is used by plugins to construct URLs
func (e *Extension) GetBasePath() string {
	if e.auth == nil {
		return e.config.BasePath
	}
	return e.auth.GetBasePath()
}

// GetDB returns the database instance
// This is used by plugins that need direct database access
func (e *Extension) GetDB() *bun.DB {
	if e.auth == nil {
		return nil
	}
	return e.auth.GetDB()
}
