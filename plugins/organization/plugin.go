package organization

import (
	"context"
	"fmt"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/store"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin                = (*Plugin)(nil)
	_ plugin.OnInit                = (*Plugin)(nil)
	_ plugin.RouteProvider         = (*Plugin)(nil)
	_ plugin.DataExportContributor = (*Plugin)(nil)
)

// Config configures the organization plugin.
type Config struct {
	// PathPrefix is the HTTP path prefix for organization routes.
	// Defaults to the engine's BasePath.
	PathPrefix string
}

// Plugin is the organization management plugin.
type Plugin struct {
	engine       plugin.Engine
	config       Config
	store        store.Store
	plugins      *plugin.Registry
	hooks        *hook.Bus
	relay        bridge.EventRelay
	chronicle    bridge.Chronicle
	logger       log.Logger
	basePath     string
	defaultAppID string
	permChecker  plugin.PermissionChecker
}

// New creates a new organization plugin with optional configuration.
// The store is resolved automatically from the engine during OnInit.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "organization" }

// SetStore allows direct store injection for testing.
func (p *Plugin) SetStore(s store.Store) { p.store = s }

// OnInit captures engine capabilities for use by the plugin's service layer.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.store = engine.Store()
	p.plugins = engine.Plugins()
	p.hooks = engine.Hooks()
	p.relay = engine.Relay()
	p.chronicle = engine.Chronicle()
	p.logger = engine.Logger()

	p.basePath = "/v1"
	p.defaultAppID = engine.DefaultAppID()

	if pc, ok := engine.(plugin.PermissionChecker); ok {
		p.permChecker = pc
	}

	if p.config.PathPrefix == "" {
		p.config.PathPrefix = p.basePath
	}

	return nil
}

// ExportUserData returns the user's organization data for GDPR export.
func (p *Plugin) ExportUserData(ctx context.Context, userID id.UserID) (label string, data any, err error) {
	orgs, err := p.store.ListUserOrganizations(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("organization: export user data: %w", err)
	}
	if len(orgs) == 0 {
		return "organizations", nil, nil
	}
	return "organizations", orgs, nil
}
