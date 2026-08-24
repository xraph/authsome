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
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/agentauth"
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

func (e *stubEngine) Store() store.Store        { return e.store }
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
	p, grant := issuanceSetup(t, hooks)

	_, err := p.IssueAgentSession(context.Background(), grant)

	require.NoError(t, err)
	assert.Equal(t, 1, hooks.beforeSessionCreate, "BeforeSessionCreate must fire so riskengine can score agent traffic")
	assert.Equal(t, 1, hooks.afterSignIn, "AfterSessionCreate must fire so impossibletravel records the agent's location")
}

func TestIssueAgentSession_StampsAgentPrincipalAndKeepsUser(t *testing.T) {
	p, grant := issuanceSetup(t, &recordingHooks{})

	sess, err := p.IssueAgentSession(context.Background(), grant)

	require.NoError(t, err)
	assert.Equal(t, session.PrincipalKindAgent, sess.PrincipalKind)
	assert.Equal(t, grant.UserID.String(), sess.UserID.String(), "the delegating human stays on the session")
	assert.Equal(t, grant.AgentID.String(), sess.AgentID.String())
	assert.Equal(t, grant.ID.String(), sess.GrantID.String())
	assert.True(t, sess.ServiceAccountID.IsNil(), "an agent session is not a service account session")
}

// The session must never outlive the grant that authorized it.
func TestIssueAgentSession_NeverOutlivesGrant(t *testing.T) {
	p, grant := issuanceSetup(t, &recordingHooks{})
	grant.ExpiresAt = time.Now().Add(5 * time.Minute)

	sess, err := p.IssueAgentSession(context.Background(), grant)

	require.NoError(t, err)
	assert.False(t, sess.ExpiresAt.After(grant.ExpiresAt),
		"a session outliving its grant would resurrect a revoked delegation")
}

func TestIssueAgentSession_RefusesInactiveGrant(t *testing.T) {
	p, grant := issuanceSetup(t, &recordingHooks{})
	grant.ExpiresAt = time.Now().Add(-time.Minute)

	_, err := p.IssueAgentSession(context.Background(), grant)

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

// issuanceSetup builds a plugin wired to a stub plugin.Engine whose
// Plugins() returns a registry with the recording plugin on it, and whose
// Store() returns a memory store.
func issuanceSetup(t *testing.T, hooks *recordingHooks) (*agentauth.Plugin, *agentauth.AgentGrant) {
	t.Helper()
	reg := plugin.NewRegistry(log.NewNoopLogger())
	reg.Register(hooks)

	eng := &stubEngine{store: memory.New(), registry: reg, bus: hook.NewBus(log.NewNoopLogger()), logger: log.NewNoopLogger()}
	agentStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(agentStore))
	require.NoError(t, p.OnInit(context.Background(), eng))

	userID := id.NewUserID()
	require.NoError(t, eng.store.CreateUser(context.Background(), &user.User{
		ID: userID, AppID: id.NewAppID(), Email: "owner@example.com", CreatedAt: time.Now(),
	}))

	grant := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
		UserID: userID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, agentStore.CreateAgentGrant(context.Background(), grant))
	return p, grant
}
