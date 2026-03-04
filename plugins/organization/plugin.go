package organization

import (
	"context"
	"fmt"
	log "github.com/xraph/go-utils/log"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/store"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin              = (*Plugin)(nil)
	_ plugin.OnInit              = (*Plugin)(nil)
	_ plugin.RouteProvider       = (*Plugin)(nil)
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
	config       Config
	store        store.Store
	plugins      *plugin.Registry
	hooks        *hook.Bus
	relay        bridge.EventRelay
	chronicle    bridge.Chronicle
	logger       log.Logger
	basePath     string
	defaultAppID string
	roleChecker  middleware.RoleChecker
}

// New creates a new organization plugin with the given store and optional configuration.
func New(s store.Store, cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &Plugin{config: c, store: s}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "organization" }

// OnInit captures engine capabilities for use by the plugin's service layer.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	e, ok := engine.(*authsome.Engine)
	if !ok {
		return fmt.Errorf("organization: expected *authsome.Engine, got %T", engine)
	}

	p.plugins = e.Plugins()
	p.hooks = e.Hooks()
	p.relay = e.Relay()
	p.chronicle = e.Chronicle()
	p.logger = e.Logger()
	p.basePath = e.Config().BasePath
	p.defaultAppID = e.Config().AppID
	p.roleChecker = e

	if p.config.PathPrefix == "" {
		p.config.PathPrefix = p.basePath
	}

	return nil
}

// ExportUserData returns the user's organization data for GDPR export.
func (p *Plugin) ExportUserData(ctx context.Context, userID id.UserID) (string, any, error) {
	orgs, err := p.store.ListUserOrganizations(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("organization: export user data: %w", err)
	}
	if len(orgs) == 0 {
		return "organizations", nil, nil
	}
	return "organizations", orgs, nil
}
