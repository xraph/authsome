package agentauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

// bridgingRegistry is a minimal auth.Registry test double standing in for
// the real DI-container-supplied registry plugin.SessionGuard/AdminGuard
// wrap via authprovider.RegistryMiddleware. It exists because, verified by
// hand against forge@v1.9.10, the registry's own Middleware(...) method
// (extensions/auth/registry.go) calls provider.Authenticate and sets
// "auth_context" on success but does NOT call authprovider.BridgeToContext —
// only authprovider.SessionProvider's own Middleware() method does that, and
// nothing in this codebase calls that method; production instead relies on
// Engine.AuthMiddleware(), a SEPARATE global middleware applied outside any
// plugin's own route group, to populate middleware.WithSession/WithUserID
// before a request ever reaches a plugin group's middleware at all.
//
// That global middleware is exactly the kind of thing item 3's fix must not
// depend on: RegisterRoutes is agentauth's own contract, and its group's
// middleware ordering has to be correct on its own terms, not conditioned on
// some other engine-level middleware happening to run first. So this test
// double's Middleware method calls the real, unmodified
// authprovider.BridgeToContext itself — mimicking exactly what a
// session-establishing step must do, positioned (via plugin.SessionGuard,
// which RegisterRoutes wraps with denyAgentPrincipal appended after it, not
// before) ahead of denyAgentPrincipal in the SAME real group-middleware
// slice RegisterRoutes builds. Every other piece here — plugin.SessionGuard,
// plugin.AdminGuard, RegisterRoutes' own append(...) calls, denyAgentPrincipal
// itself, authprovider.BridgeToContext — is the real, unmodified production
// code; only the auth-provider resolution step is a double.
type bridgingRegistry struct {
	providers map[string]auth.AuthProvider
	authz     auth.Authorizer
}

func newBridgingRegistry() *bridgingRegistry {
	return &bridgingRegistry{providers: make(map[string]auth.AuthProvider)}
}

var _ auth.Registry = (*bridgingRegistry)(nil)

func (r *bridgingRegistry) Register(p auth.AuthProvider) error {
	r.providers[p.Name()] = p
	return nil
}

func (r *bridgingRegistry) Unregister(name string) error {
	delete(r.providers, name)
	return nil
}

func (r *bridgingRegistry) Get(name string) (auth.AuthProvider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, errors.New("auth provider not found")
	}
	return p, nil
}

func (r *bridgingRegistry) Has(name string) bool {
	_, ok := r.providers[name]
	return ok
}

func (r *bridgingRegistry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Middleware is the piece under test's real dependency: this is what
// plugin.SessionGuard(engine)/AdminGuard(engine, ...) wrap via
// authprovider.RegistryMiddleware and mount as the session-establishing
// step of the group's middleware chain, ahead of denyAgentPrincipal.
func (r *bridgingRegistry) Middleware(providerNames ...string) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			for _, name := range providerNames {
				p, ok := r.providers[name]
				if !ok {
					continue
				}
				authCtx, err := p.Authenticate(ctx.Context(), ctx.Request())
				if err != nil {
					continue
				}
				if data, ok := authCtx.Data.(*authprovider.SessionData); ok {
					authprovider.BridgeToContext(ctx, data)
				}
				return next(ctx)
			}
			return ctx.JSON(http.StatusUnauthorized, map[string]any{
				"error": "authentication required", "code": http.StatusUnauthorized,
			})
		}
	}
}

func (r *bridgingRegistry) MiddlewareAnd(providerNames ...string) forge.Middleware {
	return r.Middleware(providerNames...)
}

func (r *bridgingRegistry) MiddlewareWithScopes(providerName string, _ ...string) forge.Middleware {
	return r.Middleware(providerName)
}

func (r *bridgingRegistry) MiddlewareWithRequirement(_ auth.Requirement) forge.Middleware {
	return func(next forge.Handler) forge.Handler { return next }
}

func (r *bridgingRegistry) OpenAPISchemes() map[string]auth.SecurityScheme { return nil }

func (r *bridgingRegistry) SetAuthorizer(a auth.Authorizer) {
	if a != nil {
		r.authz = a
	}
}

func (r *bridgingRegistry) Authorizer() auth.Authorizer { return r.authz }

// registryAuthEngine overrides stubEngine.AuthRegistry() to return a real
// (non-nil) registry, so plugin.SessionGuard(engine)/AdminGuard(engine, ...)
// actually contribute middleware — every other test in this package uses a
// stubEngine whose AuthRegistry() is nil, under which SessionGuard/AdminGuard
// contribute NO middleware at all, which is exactly why the original
// guard-verify for this item could not have caught the ordering bug: with no
// session-establishing middleware in the chain at all, there was no ordering
// to get wrong.
type registryAuthEngine struct {
	*stubEngine
	reg auth.Registry
}

func (e *registryAuthEngine) AuthRegistry() auth.Registry { return e.reg }

// bearerAuthProvider is an auth.AuthProvider test double resolving a bearer
// token straight to a pre-built *authprovider.SessionData, standing in for
// what a real token-backed provider (database lookup, JWT parse, etc.) would
// do. Its only job here is to hand a session to bridgingRegistry.Middleware,
// which does the real bridging.
type bearerAuthProvider struct {
	tokenToSession map[string]*authprovider.SessionData
}

func (p *bearerAuthProvider) Name() string                       { return "session" }
func (p *bearerAuthProvider) Type() auth.SecuritySchemeType      { return auth.SecurityTypeHTTP }
func (p *bearerAuthProvider) OpenAPIScheme() auth.SecurityScheme { return auth.SecurityScheme{} }
func (p *bearerAuthProvider) Middleware() forge.Middleware {
	return func(next forge.Handler) forge.Handler { return next }
}

func (p *bearerAuthProvider) Authenticate(_ context.Context, r *http.Request) (*auth.AuthContext, error) {
	authz := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(authz) <= len(prefix) || authz[:len(prefix)] != prefix {
		return nil, errors.New("missing bearer token")
	}
	token := authz[len(prefix):]
	data, ok := p.tokenToSession[token]
	if !ok {
		return nil, errors.New("unknown token")
	}
	return &auth.AuthContext{Subject: data.Session.UserID.String(), ProviderName: "session", Data: data}, nil
}

// newOrderingTestPlugin builds an agentauth plugin wired to the plugin's own
// real RegisterRoutes, with a real (non-nil) auth.Registry backing
// plugin.SessionGuard/AdminGuard, so the group middleware chain those
// functions build — and denyAgentPrincipal's real position in it — is
// exercised exactly as production constructs it.
func newOrderingTestPlugin(t *testing.T, tokenToSession map[string]*authprovider.SessionData) (*agentauth.MemoryStore, forge.Router) {
	t.Helper()

	reg := newBridgingRegistry()
	require.NoError(t, reg.Register(&bearerAuthProvider{tokenToSession: tokenToSession}))

	agentStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(agentStore))
	eng := &registryAuthEngine{
		stubEngine: &stubEngine{
			store:    memory.New(),
			registry: plugin.NewRegistry(log.NewNoopLogger()),
			bus:      hook.NewBus(log.NewNoopLogger()),
			logger:   log.NewNoopLogger(),
		},
		reg: reg,
	}
	require.NoError(t, p.OnInit(context.Background(), eng))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return agentStore, mux
}

// bearerRequest issues an HTTP request carrying token as an
// Authorization: Bearer header — a credential the session-establishing
// middleware must resolve DURING the request, as opposed to a session
// smuggled into the Go context ahead of the router (see the comment atop
// agent_principal_guard_test.go's withAgentPrincipalSession).
func bearerRequest(t *testing.T, mux forge.Router, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	reqBody := bytes.NewReader(nil)
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequestWithContext(t.Context(), method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// The property the whole item exists for, driven through the real
// SessionGuard/AdminGuard middleware and RegisterRoutes' own group-middleware
// construction: an agent-principal session, resolved DURING the request by a
// middleware occupying SessionGuard's real position in the chain, must be
// refused on both of agentauth's own route groups — not merely refused when
// a test hands denyAgentPrincipal an already-resolved session for free.
//
// This is the test the review demanded: it fails if denyAgentPrincipal is
// mounted before SessionGuard/AdminGuard in the group's middleware slice
// (forge wraps middleware outermost-first — see the comment on
// denyAgentPrincipal in middleware.go — so index 0 runs first and would see
// no session yet) and passes only with it mounted after. Confirmed by hand:
// moving plugin.go's two append(...) calls back to
// append([]forge.Middleware{denyAgentPrincipal()}, ...) turns both
// assertions below into 200/handler-reached, reproducing the reviewer's own
// scratch-router proof; restoring the fix turns them back into
// 403/handler-not-reached.
func TestAdminRoutes_RealAuthChain_AgentPrincipalSessionIsRefused(t *testing.T) {
	appID, orgID := id.NewAppID(), id.NewOrgID()
	delegate := &user.User{ID: id.NewUserID(), AppID: appID, Email: "owner@example.com"}
	agentSess := &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: delegate.ID, OrgID: orgID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	}
	_, mux := newOrderingTestPlugin(t, map[string]*authprovider.SessionData{
		"agent-bearer-token": {Session: agentSess, User: delegate},
	})

	w := bearerRequest(t, mux, http.MethodPut, "/v1/admin/agents/policy", "agent-bearer-token",
		map[string]any{"org_id": orgID.String(), "mode": "open"})

	assert.Equal(t, http.StatusForbidden, w.Code,
		"an agent-principal session resolved during the request must never reach the policy endpoint; body: %s", w.Body.String())
}

func TestMeRoutes_RealAuthChain_AgentPrincipalSessionIsRefused(t *testing.T) {
	appID, orgID := id.NewAppID(), id.NewOrgID()
	delegate := &user.User{ID: id.NewUserID(), AppID: appID, Email: "owner@example.com"}
	agentID, grantID := id.NewAgentID(), id.NewAgentGrantID()
	agentSess := &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: delegate.ID, OrgID: orgID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: agentID, GrantID: grantID,
	}
	agentStore, mux := newOrderingTestPlugin(t, map[string]*authprovider.SessionData{
		"agent-bearer-token": {Session: agentSess, User: delegate},
	})
	// The grant the agent's own session claims to be acting under, owned by
	// the delegating human — reachability, if the deny were bypassed, must
	// resolve to a real 200 on a real grant, not an incidental 404.
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), &agentauth.AgentGrant{
		ID: grantID, AppID: appID, AgentID: agentID, UserID: delegate.ID, OrgID: orgID,
		Scopes: []string{"invoices:read"}, ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	w := bearerRequest(t, mux, http.MethodDelete, "/v1/me/agents/"+grantID.String(), "agent-bearer-token", nil)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"an agent-principal session resolved during the request must never reach the revoke endpoint; body: %s", w.Body.String())
}

// A human-principal session, resolved through the exact same real chain,
// must still reach the handler — this rules out denyAgentPrincipal (or a
// mistake in the harness above) simply denying everything regardless of
// PrincipalKind, which would make the two tests above pass for the wrong
// reason.
func TestAdminRoutes_RealAuthChain_HumanPrincipalSessionReachesHandler(t *testing.T) {
	appID, orgID := id.NewAppID(), id.NewOrgID()
	delegate := &user.User{ID: id.NewUserID(), AppID: appID, Email: "owner@example.com"}
	humanSess := &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: delegate.ID, OrgID: orgID,
		PrincipalKind: session.PrincipalKindUser,
	}
	_, mux := newOrderingTestPlugin(t, map[string]*authprovider.SessionData{
		"human-bearer-token": {Session: humanSess, User: delegate},
	})

	w := bearerRequest(t, mux, http.MethodPut, "/v1/admin/agents/policy", "human-bearer-token",
		map[string]any{"org_id": orgID.String(), "mode": "open"})

	assert.Equal(t, http.StatusOK, w.Code,
		"a human-principal session must still reach the handler; body: %s", w.Body.String())
}
