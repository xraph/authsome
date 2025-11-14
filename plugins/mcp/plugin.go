package mcp

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/forge"
)

// Plugin implements the MCP (Model Context Protocol) server
// Exposes AuthSome data and operations to AI assistants
type Plugin struct {
	config          Config
	defaultConfig   Config
	db              *bun.DB
	server          *Server
	auth            interface{} // Will be *authsome.Auth
	logger          forge.Logger
	serviceRegistry *registry.ServiceRegistry
}

// PluginOption is a functional option for configuring the MCP plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithTransport sets the MCP transport type
func WithTransport(transport string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Transport = transport
	}
}

// WithEnabled sets whether MCP is enabled
func WithEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Enabled = enabled
	}
}

// WithPort sets the HTTP port for MCP server
func WithPort(port int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Port = port
	}
}

// WithExposeSecrets sets whether to expose secrets (dev only)
func WithExposeSecrets(expose bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.ExposeSecrets = expose
	}
}

// NewPlugin creates a new MCP plugin with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: Config{
			Enabled:       true,
			Transport:     "stdio",
			Port:          0,
			ExposeSecrets: false,
		},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	// Initialize with default config
	p.config = p.defaultConfig

	return p
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "mcp"
}

// Init initializes the plugin with auth instance
func (p *Plugin) Init(auth interface{}) error {
	p.auth = auth

	// Extract database and forge app from auth instance
	type authInstance interface {
		GetDB() *bun.DB
		GetServiceRegistry() *registry.ServiceRegistry
		GetForgeApp() forge.App
	}

	authInst, ok := auth.(authInstance)
	if !ok {
		return fmt.Errorf("mcp plugin requires auth instance with required methods")
	}

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for mcp plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for mcp plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "mcp"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.mcp", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind MCP config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	p.serviceRegistry = authInst.GetServiceRegistry()

	if !p.config.Enabled {
		p.logger.Info("MCP plugin initialized but disabled")
		return nil
	}

	// Initialize MCP server based on transport
	var err error
	p.server, err = NewServer(p.config, p)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	p.logger.Info("MCP plugin initialized",
		forge.F("enabled", p.config.Enabled),
		forge.F("transport", p.config.Transport),
		forge.F("port", p.config.Port))

	return nil
}

// RegisterRoutes registers HTTP endpoints if HTTP transport is enabled
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.config.Transport != TransportHTTP {
		return nil // No routes needed for stdio transport
	}

	// TODO: Register HTTP MCP endpoint
	// router.POST("/api/mcp", p.handleMCPRequest)

	return nil
}

// RegisterHooks registers plugin hooks (none needed for MCP)
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	return nil // MCP is read-mostly, no hooks needed
}

// RegisterServiceDecorators allows service decoration (none needed for MCP)
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	return nil // MCP doesn't modify core services
}

// Migrate runs database migrations (none needed for MCP)
func (p *Plugin) Migrate() error {
	return nil // MCP doesn't require database schema
}

// Start starts the MCP server (call after Initialize)
func (p *Plugin) Start(ctx context.Context) error {
	if !p.config.Enabled {
		return nil
	}

	return p.server.Start(ctx)
}

// Stop gracefully stops the MCP server
func (p *Plugin) Stop(ctx context.Context) error {
	if p.server != nil {
		return p.server.Stop(ctx)
	}
	return nil
}

// GetServer returns the underlying MCP server (for CLI/testing)
func (p *Plugin) GetServer() *Server {
	return p.server
}
