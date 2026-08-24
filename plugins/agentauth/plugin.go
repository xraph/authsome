package agentauth

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/plugin"
)

// defaultGrantTTL caps how long a delegation lives before the user has to
// consent again.
const defaultGrantTTL = 90 * 24 * time.Hour

// Compile-time interface checks.
var (
	_ plugin.Plugin = (*Plugin)(nil)
	_ plugin.OnInit = (*Plugin)(nil)
)

// Plugin is the delegated agent identity plugin.
type Plugin struct {
	engine      plugin.Engine
	store       Store
	scopes      *ScopeRegistry
	hooks       *hook.Bus
	chronicle   bridge.Chronicle
	logger      log.Logger
	permChecker plugin.PermissionChecker
	grantTTL    time.Duration
	basePath    string
}

// Option configures the plugin at construction.
type Option func(*Plugin)

// WithScope maps a delegation scope onto a Warden permission. A scope with no
// mapping is rejected at consent time.
func WithScope(scope string, p Permission) Option {
	return func(pl *Plugin) { pl.scopes.Register(scope, p) }
}

// WithStore injects a persistent store. Without it the plugin uses an
// in-memory store.
func WithStore(s Store) Option {
	return func(pl *Plugin) { pl.store = s }
}

// WithDefaultGrantTTL sets the fallback cap on grant lifetime. An org policy
// with a shorter MaxGrantTTL still wins.
func WithDefaultGrantTTL(d time.Duration) Option {
	return func(pl *Plugin) { pl.grantTTL = d }
}

// New creates the agentauth plugin.
func New(opts ...Option) *Plugin {
	p := &Plugin{
		store:    NewMemoryStore(),
		scopes:   NewScopeRegistry(),
		grantTTL: defaultGrantTTL,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "agentauth" }

// Scopes returns the delegation scope registry.
func (p *Plugin) Scopes() *ScopeRegistry { return p.scopes }

// Store returns the plugin's store.
func (p *Plugin) Store() Store { return p.store }

// DefaultGrantTTL returns the fallback cap on grant lifetime.
func (p *Plugin) DefaultGrantTTL() time.Duration { return p.grantTTL }

// OnInit captures engine capabilities.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.hooks = engine.Hooks()
	p.chronicle = engine.Chronicle()
	p.logger = engine.Logger()
	p.basePath = engine.BasePath()
	if p.basePath == "" {
		p.basePath = "/v1"
	}
	if pc, ok := engine.(plugin.PermissionChecker); ok {
		p.permChecker = pc
	}
	return nil
}
