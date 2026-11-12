package organization

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

	// InvitationURLTemplate is the link an invitation email carries, with
	// {token} standing in for the invitation token, for example
	// "https://app.example.com/accept-invite?token={token}". Empty (the
	// default) means the plugin sends no invitation email, which was the
	// behaviour before this option existed. Mail goes through the plugin
	// registered as "notification"; without one nothing is sent either.
	InvitationURLTemplate string

	// RoleRanks gives role names other than owner, admin and member a rank
	// for the checks that compare roles: who may read or change an
	// organization, grant a role, or remove a member. Keys are lower case
	// names; values are on the built-in scale, member 1, admin 2, owner 3,
	// so {"builder": 1} lets a builder do what a member can and nothing
	// more. A name not listed ranks 0, below member, which is how every
	// custom role was treated before this option existed. The built-in names
	// cannot be changed here. Be careful with 2 or 3: a role ranked as admin
	// passes every admin-level check in this plugin.
	RoleRanks map[string]int
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
	skipLogged   sync.Once
}

// New creates a new organization plugin with optional configuration.
// The store is resolved automatically from the engine during OnInit.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	// Keys are matched trimmed and lower case, so store them that way. The
	// caller's map is copied, not changed.
	if len(c.RoleRanks) > 0 {
		ranks := make(map[string]int, len(c.RoleRanks))
		for name, rank := range c.RoleRanks {
			ranks[strings.ToLower(strings.TrimSpace(name))] = rank
		}
		c.RoleRanks = ranks
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
