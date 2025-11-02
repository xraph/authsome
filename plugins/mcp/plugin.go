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
	config Config
	db     *bun.DB
	server *Server
	auth   interface{} // Will be *authsome.Auth

	// Service references for resource/tool handlers
	serviceRegistry *registry.ServiceRegistry
}

// NewPlugin creates a new MCP plugin
func NewPlugin(config Config) *Plugin {
	return &Plugin{
		config: config,
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "mcp"
}

// Init initializes the plugin with auth instance
func (p *Plugin) Init(auth interface{}) error {
	p.auth = auth

	// Extract database from auth instance
	type dbGetter interface {
		GetDB() *bun.DB
		GetServiceRegistry() *registry.ServiceRegistry
	}

	if dbAuth, ok := auth.(dbGetter); ok {
		p.db = dbAuth.GetDB()
		p.serviceRegistry = dbAuth.GetServiceRegistry()
	} else {
		return fmt.Errorf("auth instance does not implement required methods")
	}

	if p.db == nil {
		return fmt.Errorf("database not available")
	}

	// Initialize MCP server based on transport
	var err error
	p.server, err = NewServer(p.config, p)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

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
