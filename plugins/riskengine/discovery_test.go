package riskengine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"
	"github.com/xraph/grove"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// registryEngine is the smallest plugin.Engine that carries a real plugin
// registry, which is all OnInit needs to find contributors.
type registryEngine struct {
	registry *plugin.Registry
}

var _ plugin.Engine = (*registryEngine)(nil)

func (e *registryEngine) Plugins() *plugin.Registry { return e.registry }
func (e *registryEngine) Plugin(name string) plugin.Plugin {
	if e.registry == nil {
		return nil
	}
	return e.registry.Plugin(name)
}

func (*registryEngine) Store() store.Store                          { return nil }
func (*registryEngine) DB() *grove.DB                               { return nil }
func (*registryEngine) Hooks() *hook.Bus                            { return nil }
func (*registryEngine) Logger() log.Logger                          { return log.NewNoopLogger() }
func (*registryEngine) Settings() *settings.Manager                 { return nil }
func (*registryEngine) Chronicle() bridge.Chronicle                 { return nil }
func (*registryEngine) Relay() bridge.EventRelay                    { return nil }
func (*registryEngine) Herald() bridge.Herald                       { return nil }
func (*registryEngine) Mailer() bridge.Mailer                       { return nil }
func (*registryEngine) SMSSender() bridge.SMSSender                 { return nil }
func (*registryEngine) Ledger() bridge.Ledger                       { return nil }
func (*registryEngine) TokenEncryptor() bridge.Encryptor            { return nil }
func (*registryEngine) CeremonyStore() ceremony.Store               { return nil }
func (*registryEngine) APIKeyStore() apikey.Store                   { return nil }
func (*registryEngine) SecurityEvents() securityevent.Store         { return nil }
func (*registryEngine) AuthMiddleware() forge.Middleware            { return nil }
func (*registryEngine) AuthRegistry() auth.Registry                 { return nil }
func (*registryEngine) PlatformAppID() id.AppID                     { return id.Nil }
func (*registryEngine) DefaultAppID() string                        { return "" }
func (*registryEngine) BasePath() string                            { return "" }
func (*registryEngine) TokenFormatForApp(string) tokenformat.Format { return nil }

func (*registryEngine) SessionConfigForApp(context.Context, id.AppID, ...id.EnvironmentID) account.SessionConfig {
	return account.SessionConfig{}
}
func (*registryEngine) ResolveSessionByToken(string) (*session.Session, error) { return nil, nil }
func (*registryEngine) ResolveUser(string) (*user.User, error)                 { return nil, nil }
func (*registryEngine) GetUser(context.Context, id.UserID) (*user.User, error) { return nil, nil }
func (*registryEngine) EnsureDefaultRole(context.Context, id.AppID, id.UserID) {}
func (*registryEngine) ResolvePrincipal(context.Context, principal.Ref) (*principal.Principal, error) {
	return nil, principal.ErrNotFound
}
func (*registryEngine) PrincipalStore() principal.Store { return nil }
func (*registryEngine) Can(context.Context, principal.Ref, principal.Chain, string, string) (bool, error) {
	return false, nil
}

// notAContributor is registered alongside the real one to prove the sweep
// discriminates on the interface rather than adopting everything it finds.
type notAContributor struct{}

func (*notAContributor) Name() string { return "bystander" }

// The bug this pins: a plugin could implement RiskContributor, be registered
// with the engine, and never be called, because the only ways into the
// contributor list were New, NewWithConfig and AddContributor -- all of which
// need the host application to know that this particular plugin happens to
// score risk and to wire it by hand. Everything the contributor did shipped
// as dead code.
func TestOnInit_AdoptsContributorsFromThePluginRegistry(t *testing.T) {
	contributor := &mockContributor{name: "durable-signals", score: 80, weight: 2.0}

	registry := plugin.NewRegistry(log.NewNoopLogger())
	registry.Register(&notAContributor{})
	registry.Register(contributor)

	p := New()
	require.NoError(t, p.OnInit(context.Background(), &registryEngine{registry: registry}))

	require.Len(t, p.contributors, 1, "exactly the plugin implementing RiskContributor")
	assert.Equal(t, "durable-signals", p.contributors[0].Name())

	// And it is actually consulted on the sign-in path, which is the thing
	// that was broken.
	err := p.OnBeforeSignIn(context.Background(), &account.SignInRequest{
		AppID: id.NewAppID(), Email: "target@corp.com",
	})
	require.NoError(t, err)
	require.NotNil(t, p.lastAssessment)
	require.Len(t, p.lastAssessment.Signals, 1)
	assert.Equal(t, "durable-signals", p.lastAssessment.Signals[0].Source)
	assert.Equal(t, 80, p.lastAssessment.OverallScore)
}

// A host that already wired a contributor explicitly must not end up with it
// twice -- the same signal counted twice is the same signal weighted twice.
func TestOnInit_DoesNotAdoptAContributorTwice(t *testing.T) {
	contributor := &mockContributor{name: "durable-signals", score: 80, weight: 2.0}

	registry := plugin.NewRegistry(log.NewNoopLogger())
	registry.Register(contributor)

	p := New(contributor)
	require.NoError(t, p.OnInit(context.Background(), &registryEngine{registry: registry}))
	assert.Len(t, p.contributors, 1)

	// The same goes for AddContributor called after discovery.
	p.AddContributor(contributor)
	assert.Len(t, p.contributors, 1)
}

// An engine with no registry at all (a stub, a host mid-construction) must
// not panic the sweep.
func TestOnInit_ToleratesAnEngineWithNoRegistry(t *testing.T) {
	p := New()
	require.NoError(t, p.OnInit(context.Background(), &registryEngine{registry: nil}))
	assert.Empty(t, p.contributors)
}
