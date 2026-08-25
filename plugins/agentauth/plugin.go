package agentauth

import (
	"context"
	"net/http"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/oauth2provider"

	"github.com/xraph/grove/migrate"
)

// defaultGrantTTL caps how long a delegation lives before the user has to
// consent again.
const defaultGrantTTL = 90 * 24 * time.Hour

// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.RouteProvider     = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
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
	cache       *grantCache
}

// Option configures the plugin at construction.
type Option func(*Plugin)

// WithScope maps a delegation scope onto a Warden permission. A scope with no
// mapping is rejected at consent time.
func WithScope(scope string, p Permission) Option {
	return func(pl *Plugin) { pl.scopes.Register(scope, p) }
}

// WithStore injects a persistent store. Without it the plugin uses an
// in-memory store. A nil s is a no-op rather than installing a nil store:
// Store is an interface, so pl.store = nil would leave every store call in
// the plugin (Authorize's grant read included, at the very first thing an
// agent request does) calling a method on a nil interface value, which
// panics rather than erroring. New() already sets a working default before
// any Option runs, so there is nothing useful a nil argument here could mean
// other than "leave it alone".
func WithStore(s Store) Option {
	return func(pl *Plugin) {
		if s == nil {
			return
		}
		pl.store = s
	}
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
		cache:    newGrantCache(grantCacheTTL),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "agentauth" }

// MigrationGroups returns the agentauth migration groups for the given driver.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg", "postgres":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite", "sqlite3":
		return []*migrate.Group{SqliteMigrations}
	case "mongo", "mongodb":
		return []*migrate.Group{MongoMigrations}
	default:
		return nil
	}
}

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
	// The router plugins register on is already grouped at the app's base
	// path (extension/extension.go groups it before handing plugins the
	// router), so p.basePath must carry only the version segment, not the
	// base path again. engine.BasePath() defaults to "/authsome", and using
	// it here would have doubled up to "/authsome/authsome/me/agents".
	// plugins/consent/plugin.go hardcodes "/v1" for the same reason; this
	// matches it.
	p.basePath = "/v1"
	if pc, ok := engine.(plugin.PermissionChecker); ok {
		p.permChecker = pc
	}
	p.registerConsentGate(engine)
	return nil
}

// consentGateSetter is implemented by oauth2provider.Plugin. Asserting
// against this interface rather than the concrete type keeps agentauth from
// importing more of that package's surface than this one call needs, and
// keeps the wiring resilient to a test double or future re-implementation
// that satisfies the same shape without being oauth2provider.Plugin itself.
type consentGateSetter interface {
	SetConsentGate(oauth2provider.ConsentGate)
}

// registerConsentGate wires agentauth as oauth2provider's ConsentGate when
// an "oauth2provider" plugin is registered on the same engine, per the
// design spec ("agentauth registers itself as the gate during OnInit") and
// per oauth2provider's own consent_gate.go doc comment, which already
// asserts this happens.
//
// Without this call the plugin enforces nothing at the point that matters
// most: a host that follows the RegisterRoutes/RouteProvider wiring alone,
// with no code of its own naming ConsentGate, would see an agent run the
// ordinary OAuth2 flow and receive a plain human session — no PrincipalKind,
// no grant, that user's full role set — because nothing ever told
// oauth2provider a gate exists to consult.
//
// A missing oauth2provider plugin, or one that doesn't expose
// SetConsentGate, is a no-op rather than an error: agentauth must remain
// usable in a host that has not installed oauth2provider yet, or that wires
// the gate some other way (see doc.go's host integration contract, point 1).
func (p *Plugin) registerConsentGate(engine plugin.Engine) {
	op := engine.Plugins().Plugin("oauth2provider")
	if op == nil {
		p.logger.Debug("agentauth: no oauth2provider plugin registered; consent gate not wired")
		return
	}
	setter, ok := op.(consentGateSetter)
	if !ok {
		p.logger.Debug("agentauth: registered oauth2provider plugin does not expose SetConsentGate; consent gate not wired")
		return
	}
	setter.SetConsentGate(p)
}

// RegisterRoutes registers the user and admin surfaces: a place for a user
// to see and revoke which agents are acting for them, and an admin surface
// for registering and blocking agents org-wide.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	me := router.Group(p.basePath+"/me/agents",
		forge.WithGroupTags("agentauth"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(append([]forge.Middleware{denyAgentPrincipal()}, plugin.SessionGuard(p.engine)...)...),
	)
	if err := me.GET("", p.handleListMyGrants,
		forge.WithSummary("List agents acting on my behalf"),
		forge.WithResponseSchema(http.StatusOK, "Active delegations", ListGrantsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := me.DELETE("/:id", p.handleRevokeMyGrant,
		forge.WithSummary("Revoke an agent's delegation"),
		forge.WithResponseSchema(http.StatusOK, "Grant revoked", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	admin := router.Group(p.basePath+"/admin/agents",
		forge.WithGroupTags("agentauth-admin"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(append([]forge.Middleware{denyAgentPrincipal()}, plugin.AdminGuard(p.engine, "read", "agent")...)...),
	)
	if err := admin.GET("", p.handleListAgents,
		forge.WithSummary("List registered agents"),
		forge.WithResponseSchema(http.StatusOK, "Registered agents", ListAgentsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := admin.POST("", p.handleRegisterAgent,
		append([]forge.RouteOption{
			forge.WithSummary("Register an agent"),
			forge.WithRequestSchema(RegisterAgentRequest{}),
			forge.WithResponseSchema(http.StatusCreated, "Agent registered", Agent{}),
			forge.WithErrorResponses(),
		}, plugin.PermissionRouteOptions(p.engine, "write", "agent")...)...,
	); err != nil {
		return err
	}
	if err := admin.PATCH("/:id/status", p.handleSetAgentStatus,
		append([]forge.RouteOption{
			forge.WithSummary("Approve or block an agent"),
			forge.WithRequestSchema(SetAgentStatusRequest{}),
			forge.WithResponseSchema(http.StatusOK, "Status updated", StatusResponse{}),
			forge.WithErrorResponses(),
		}, plugin.PermissionRouteOptions(p.engine, "write", "agent")...)...,
	); err != nil {
		return err
	}
	return admin.PUT("/policy", p.handlePutOrgPolicy,
		append([]forge.RouteOption{
			forge.WithSummary("Set the org's agent delegation policy"),
			forge.WithRequestSchema(PutOrgPolicyRequest{}),
			forge.WithResponseSchema(http.StatusOK, "Policy set", OrgAgentPolicy{}),
			forge.WithErrorResponses(),
		}, plugin.PermissionRouteOptions(p.engine, "write", "agent")...)...,
	)
}
