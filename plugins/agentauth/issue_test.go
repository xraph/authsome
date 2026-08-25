package agentauth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"

	"github.com/xraph/grove"
)

// recordingHooks is a plugin that records which lifecycle hooks fired during
// issuance. It is registered on a plugin.Registry, which is what carries the
// Emit* methods.
type recordingHooks struct {
	beforeSessionCreate int
	afterSignIn         int
}

func (h *recordingHooks) Name() string { return "recording-hooks" }

func (h *recordingHooks) OnBeforeSessionCreate(_ context.Context, _ *session.Session) error {
	h.beforeSessionCreate++
	return nil
}

func (h *recordingHooks) OnAfterSignIn(_ context.Context, _ *user.User, _ *session.Session) error {
	h.afterSignIn++
	return nil
}

// vetoingHooks refuses every session it sees, standing in for a risk plugin
// that has decided to block. It records the session's ID before returning
// the error, since the caller of IssueAgentSession never gets the session
// back on the veto path and a test still needs a way to ask "did that ID
// reach the store anyway".
type vetoingHooks struct {
	err           error
	lastSessionID id.SessionID
}

func (h *vetoingHooks) Name() string { return "vetoing-hooks" }

func (h *vetoingHooks) OnBeforeSessionCreate(_ context.Context, sess *session.Session) error {
	h.lastSessionID = sess.ID
	return h.err
}

// stubEngine is a minimal plugin.Engine for testing issuance. Store() and
// Plugins() return real, working values; GetUser() resolves against the
// same store CreateUser wrote to. Everything else returns a zero value: this
// task never exercises it.
type stubEngine struct {
	store    store.Store
	registry *plugin.Registry
	bus      *hook.Bus
	logger   log.Logger
}

var _ plugin.Engine = (*stubEngine)(nil)

func (e *stubEngine) Store() store.Store { return e.store }

// The plugin.Engine interface grew while this branch was out: DPoP
// proof-of-possession, principal resolution, chain-aware authorization, the
// rate limiter and the security-event store. Issuance touches none of them,
// so they stay inert here.
func (*stubEngine) DPoPValidator() *dpop.Validator                         { return nil }
func (*stubEngine) DPoPNonceSigner() *dpop.NonceSigner                     { return nil }
func (*stubEngine) DPoPModeForApp(context.Context, id.AppID) dpop.Mode     { return dpop.ModeOff }
func (*stubEngine) DPoPNonceRequiredForApp(context.Context, id.AppID) bool { return false }
func (*stubEngine) PrincipalStore() principal.Store                        { return nil }
func (*stubEngine) RateLimiter() ratelimit.Limiter                         { return nil }
func (*stubEngine) SecurityEvents() securityevent.Store                    { return nil }
func (*stubEngine) ResolvePrincipal(context.Context, principal.Ref) (*principal.Principal, error) {
	return nil, nil
}
func (*stubEngine) Can(context.Context, principal.Ref, principal.Chain, string, string) (bool, error) {
	return false, nil
}
func (e *stubEngine) DB() *grove.DB             { return nil }
func (e *stubEngine) Plugins() *plugin.Registry { return e.registry }
func (e *stubEngine) Plugin(_ string) plugin.Plugin {
	return nil
}
func (e *stubEngine) Hooks() *hook.Bus            { return e.bus }
func (e *stubEngine) Logger() log.Logger          { return e.logger }
func (e *stubEngine) Settings() *settings.Manager { return nil }
func (e *stubEngine) Chronicle() bridge.Chronicle { return nil }
func (e *stubEngine) Relay() bridge.EventRelay    { return nil }
func (e *stubEngine) Herald() bridge.Herald       { return nil }
func (e *stubEngine) Mailer() bridge.Mailer       { return nil }
func (e *stubEngine) SMSSender() bridge.SMSSender { return nil }
func (e *stubEngine) Ledger() bridge.Ledger       { return nil }
func (e *stubEngine) TokenEncryptor() bridge.Encryptor {
	return bridge.NoopEncryptor{}
}
func (e *stubEngine) SessionConfigForApp(_ context.Context, _ id.AppID, _ ...id.EnvironmentID) account.SessionConfig {
	return account.SessionConfig{}
}
func (e *stubEngine) TokenFormatForApp(_ string) tokenformat.Format { return nil }
func (e *stubEngine) CeremonyStore() ceremony.Store                 { return nil }
func (e *stubEngine) APIKeyStore() apikey.Store                     { return nil }
func (e *stubEngine) ResolveSessionByToken(_ string) (*session.Session, error) {
	return nil, errors.New("not implemented")
}
func (e *stubEngine) ResolveUser(_ string) (*user.User, error) {
	return nil, errors.New("not implemented")
}
func (e *stubEngine) GetUser(ctx context.Context, userID id.UserID) (*user.User, error) {
	return e.store.GetUser(ctx, userID)
}
func (e *stubEngine) EnsureDefaultRole(_ context.Context, _ id.AppID, _ id.UserID) {}
func (e *stubEngine) AuthMiddleware() forge.Middleware                             { return nil }
func (e *stubEngine) AuthRegistry() auth.Registry                                  { return nil }
func (e *stubEngine) PlatformAppID() id.AppID                                      { return id.AppID{} }
func (e *stubEngine) DefaultAppID() string                                         { return "" }
func (e *stubEngine) BasePath() string                                             { return "" }

// The risk plugins subscribe to BeforeSessionCreate and AfterSignIn. The API
// key plugin hand-builds a synthetic session at plugins/apikey/plugin.go:567
// and therefore fires neither, which is exactly why API key traffic is
// invisible to riskengine, impossibletravel and the rest today. This test
// stops somebody optimizing agent issuance into that same shape later and
// silently taking the risk plugins offline.
func TestIssueAgentSession_FiresRiskHooks(t *testing.T) {
	hooks := &recordingHooks{}
	p, grant, _ := issuanceSetup(t, hooks)

	_, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})

	require.NoError(t, err)
	assert.Equal(t, 1, hooks.beforeSessionCreate, "BeforeSessionCreate must fire so riskengine can score agent traffic")
	assert.Equal(t, 1, hooks.afterSignIn, "AfterSignIn must fire so impossibletravel records the agent's location")
}

func TestIssueAgentSession_StampsAgentPrincipalAndKeepsUser(t *testing.T) {
	p, grant, _ := issuanceSetup(t, &recordingHooks{})

	sess, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})

	require.NoError(t, err)
	assert.Equal(t, session.PrincipalKindAgent, sess.PrincipalKind)
	assert.Equal(t, grant.UserID.String(), sess.UserID.String(), "the delegating human stays on the session")
	assert.Equal(t, grant.AgentID.String(), sess.AgentID.String())
	assert.Equal(t, grant.ID.String(), sess.GrantID.String())
	assert.True(t, sess.ServiceAccountID.IsNil(), "an agent session is not a service account session")
}

// The session must never outlive the grant that authorized it.
func TestIssueAgentSession_NeverOutlivesGrant(t *testing.T) {
	p, grant, _ := issuanceSetup(t, &recordingHooks{})
	grant.ExpiresAt = time.Now().Add(5 * time.Minute)

	sess, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})

	require.NoError(t, err)
	assert.False(t, sess.ExpiresAt.After(grant.ExpiresAt),
		"a session outliving its grant would resurrect a revoked delegation")
	assert.False(t, sess.RefreshTokenExpiresAt.After(grant.ExpiresAt),
		"the refresh token is part of the same credential and must not outlive the grant either")
}

func TestIssueAgentSession_RefusesInactiveGrant(t *testing.T) {
	p, grant, _ := issuanceSetup(t, &recordingHooks{})
	grant.ExpiresAt = time.Now().Add(-time.Minute)

	_, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

// C1: a hand-built session with no Token and no RefreshToken is not a usable
// credential, and both columns carry a non-partial unique index on postgres
// and sqlite. Before the fix, both fields were left as "" on every session,
// so this insert would succeed once and then collide with "authsome:
// conflict" on the real backends (the memory store used here keys sessions
// only on ID, so it can't reproduce the collision itself — that is verified
// separately against a real postgres instance; see the report). What this
// test can and does prove without a real backend is the fix itself: two
// mints from the same grant must produce distinct, non-empty tokens.
func TestIssueAgentSession_TwoSessionsFromSameGrantGetDistinctTokens(t *testing.T) {
	p, grant, _ := issuanceSetup(t, &recordingHooks{})

	sess1, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})
	require.NoError(t, err)

	sess2, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})
	require.NoError(t, err)

	assert.NotEmpty(t, sess1.Token, "a session with no token is not a usable credential")
	assert.NotEmpty(t, sess1.RefreshToken)
	assert.NotEmpty(t, sess2.Token)
	assert.NotEmpty(t, sess2.RefreshToken)
	assert.NotEqual(t, sess1.Token, sess2.Token, "two sessions minted from the same grant must not share a token")
	assert.NotEqual(t, sess1.RefreshToken, sess2.RefreshToken)
}

// C2: authsome_sessions.env_id is NOT NULL with a foreign key to
// authsome_environments on postgres. A zero id.EnvironmentID stringifies to
// "" and fails that FK on the very first insert (sqlite and mongo enforce
// no such constraint, so this passed silently on both before the fix). The
// memory store used here doesn't enforce the FK either, so this only proves
// IssueAgentSession resolves and sets EnvID from the app's default
// environment; the FK itself is verified against a real postgres instance
// separately (see the report).
func TestIssueAgentSession_ResolvesDefaultEnvironment(t *testing.T) {
	p, grant, eng := issuanceSetup(t, &recordingHooks{})

	sess, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})
	require.NoError(t, err)

	env, envErr := eng.store.GetDefaultEnvironment(context.Background(), grant.AppID)
	require.NoError(t, envErr)
	assert.False(t, sess.EnvID.IsNil(), "EnvID must be set for the postgres FK to be satisfiable")
	assert.Equal(t, env.ID.String(), sess.EnvID.String())
}

// I1: agentauth.Authorize enforces the intersection of the grant's scope and
// the delegating human's own permission. Stamping the human's roles onto the
// session would let a role-gated route be satisfied straight off
// sess.Roles, bypassing that intersection without agentauth.Authorize ever
// running.
//
// This stub wires IssueAgentSession straight to the plain memory store, not
// the production roleStampingStore decorator (engine_session_roles.go), so
// it cannot exercise the actual production bypass by itself — that is
// TestCreateSessionSkipsAgents in the root package, added and guard-verified
// alongside this fix. This test instead guards IssueAgentSession's own
// contribution: it must never set sess.Roles itself.
func TestIssueAgentSession_CarriesNoStampedRoles(t *testing.T) {
	p, grant, _ := issuanceSetup(t, &recordingHooks{})

	sess, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})

	require.NoError(t, err)
	assert.Empty(t, sess.Roles, "an agent session must never carry the delegating human's roles")
}

// I3: without IPAddress and UserAgent, impossibletravel and geoip return
// immediately with nothing to score, so the hooks firing teaches those
// plugins nothing. IssueMeta is the caller-supplied path for that request
// context to reach the session.
func TestIssueAgentSession_CarriesRequestMetadata(t *testing.T) {
	p, grant, _ := issuanceSetup(t, &recordingHooks{})
	meta := agentauth.IssueMeta{
		IPAddress: "203.0.113.7",
		UserAgent: "test-agent/1.0",
		DeviceID:  id.NewDeviceID(),
	}

	sess, err := p.IssueAgentSession(context.Background(), grant, meta)

	require.NoError(t, err)
	assert.Equal(t, meta.IPAddress, sess.IPAddress)
	assert.Equal(t, meta.UserAgent, sess.UserAgent)
	assert.Equal(t, meta.DeviceID.String(), sess.DeviceID.String())
}

// I5: recordingHooks.OnBeforeSessionCreate always returns nil, so none of
// the tests above exercise the veto path at all — it has been decorative by
// luck rather than by design. This is the one that actually calls it and
// checks both halves: the error surfaces to the caller, and nothing was
// left behind in the store.
func TestIssueAgentSession_VetoBlocksPersistence(t *testing.T) {
	veto := &vetoingHooks{err: errors.New("blocked: high risk")}
	p, grant, eng := issuanceSetup(t, veto)

	sess, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})

	require.Error(t, err)
	assert.Nil(t, sess)
	require.False(t, veto.lastSessionID.IsNil(), "the vetoing hook must have seen the session before it was rejected")

	_, getErr := eng.store.GetSession(context.Background(), veto.lastSessionID)
	assert.ErrorIs(t, getErr, store.ErrNotFound, "a vetoed session must never be persisted")
}

// I6: IssueAgentSession is an exported entry point, so a nil grant or one
// missing an id Authorize binds sessions to is exactly what a careless or
// malicious caller can hand it. grant.IsActive would panic on a nil grant
// before any of this ran, and a grant with a nil UserID would mint a session
// with user_id = "" that Authorize can never legitimately match.
func TestIssueAgentSession_RejectsNilGrant(t *testing.T) {
	p, _, _ := issuanceSetup(t, &recordingHooks{})

	_, err := p.IssueAgentSession(context.Background(), nil, agentauth.IssueMeta{})

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

func TestIssueAgentSession_RejectsGrantMissingRequiredIDs(t *testing.T) {
	p, grant, _ := issuanceSetup(t, &recordingHooks{})

	cases := map[string]func(*agentauth.AgentGrant){
		"nil UserID":  func(g *agentauth.AgentGrant) { g.UserID = id.UserID{} },
		"nil AgentID": func(g *agentauth.AgentGrant) { g.AgentID = id.AgentID{} },
		"nil AppID":   func(g *agentauth.AgentGrant) { g.AppID = id.AppID{} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			g := *grant
			mutate(&g)

			_, err := p.IssueAgentSession(context.Background(), &g, agentauth.IssueMeta{})

			require.ErrorIs(t, err, agentauth.ErrGrantInactive)
		})
	}
}

// M3: LastUsedAt/UpdatedAt must be stamped on the grant only once
// UpdateAgentGrant has actually succeeded, and never left dangling on the
// caller's pointer when it hasn't.
func TestIssueAgentSession_StampsGrantLastUsedOnSuccess(t *testing.T) {
	p, grant, _ := issuanceSetup(t, &recordingHooks{})
	require.Nil(t, grant.LastUsedAt)
	originalUpdatedAt := grant.UpdatedAt

	_, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})

	require.NoError(t, err)
	require.NotNil(t, grant.LastUsedAt)
	assert.True(t, grant.UpdatedAt.After(originalUpdatedAt), "UpdatedAt must be bumped alongside LastUsedAt")
}

// issuanceSetup builds a plugin wired to a stub plugin.Engine whose
// Plugins() returns a registry with the given hook plugins registered, and
// whose Store() returns a memory store carrying a user, a matching default
// environment, and a live grant. Returns the stub engine too so tests can
// reach into the store directly (GetSession, GetDefaultEnvironment) for
// assertions IssueAgentSession's return value alone can't make.
func issuanceSetup(t *testing.T, hooks ...plugin.Plugin) (*agentauth.Plugin, *agentauth.AgentGrant, *stubEngine) {
	t.Helper()
	reg := plugin.NewRegistry(log.NewNoopLogger())
	for _, h := range hooks {
		reg.Register(h)
	}

	eng := &stubEngine{store: memory.New(), registry: reg, bus: hook.NewBus(log.NewNoopLogger()), logger: log.NewNoopLogger()}
	agentStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(agentStore))
	require.NoError(t, p.OnInit(context.Background(), eng))

	appID := id.NewAppID()
	userID := id.NewUserID()
	require.NoError(t, eng.store.CreateUser(context.Background(), &user.User{
		ID: userID, AppID: appID, Email: "owner@example.com", CreatedAt: time.Now(),
	}))

	// A default environment for the grant's app, so the EnvID resolution
	// IssueAgentSession does (C2) has something real to find, mirroring
	// what postgres' env_id foreign key requires in production.
	require.NoError(t, eng.store.CreateEnvironment(context.Background(), &environment.Environment{
		ID:        id.NewEnvironmentID(),
		AppID:     appID,
		Name:      "Production",
		Slug:      "production",
		Type:      environment.TypeProduction,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))

	grant := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: id.NewAgentID(),
		UserID: userID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), grant))
	return p, grant, eng
}
