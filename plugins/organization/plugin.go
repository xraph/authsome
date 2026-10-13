package organization

import (
	"context"
	"fmt"
	"time"

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

	// InvitationTTL is how long an invitation can be accepted after it is
	// created. Zero means DefaultInvitationTTL (seven days). A negative value
	// turns expiry off, so invitations never lapse, which was the behaviour
	// before this option existed.
	InvitationTTL time.Duration
}

// DefaultInvitationTTL is how long an invitation stays acceptable when
// Config.InvitationTTL is left at zero.
const DefaultInvitationTTL = 7 * 24 * time.Hour

// invitationTTL resolves Config.InvitationTTL: the default for zero, and
// zero (no expiry) for a negative value.
func (c Config) invitationTTL() time.Duration {
	switch {
	case c.InvitationTTL == 0:
		return DefaultInvitationTTL
	case c.InvitationTTL < 0:
		return 0
	default:
		return c.InvitationTTL
	}
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
	p.chronicle = bridge.NewBusChronicle(engine.Hooks())
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
