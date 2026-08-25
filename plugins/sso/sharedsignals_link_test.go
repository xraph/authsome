package sso

import (
	"context"
	"testing"

	"github.com/xraph/authsome/ratelimit"

	log "github.com/xraph/go-utils/log"
	"github.com/xraph/grove"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	"github.com/stretchr/testify/assert"

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

type recordingLinker struct {
	calls   int
	issuer  string
	subject string
	userID  id.UserID
	err     error
}

func (r *recordingLinker) Name() string { return "sharedsignals" }

func (r *recordingLinker) LinkSubject(_ context.Context, _ id.AppID, _ id.EnvironmentID,
	issuer, subject string, userID id.UserID, _ string) error {
	r.calls++
	r.issuer, r.subject, r.userID = issuer, subject, userID
	return r.err
}

// stubEngineWithPlugin is the smallest thing that satisfies plugin.Engine,
// with Plugin(name) returning a caller-supplied plugin for one name. sso has
// no existing engine test double (routes_test.go only exercises RegisterRoutes
// against a bare router), so this follows the same shape as the stubEngine
// built for the sharedsignals plugin's own OnInit tests.
type stubEngineWithPlugin struct {
	name string
	p    plugin.Plugin
}

var _ plugin.Engine = stubEngineWithPlugin{}

func (s stubEngineWithPlugin) Store() store.Store        { return nil }
func (s stubEngineWithPlugin) DB() *grove.DB             { return nil }
func (s stubEngineWithPlugin) Plugins() *plugin.Registry { return nil }
func (s stubEngineWithPlugin) Plugin(name string) plugin.Plugin {
	if name == s.name {
		return s.p
	}
	return nil
}
func (s stubEngineWithPlugin) Hooks() *hook.Bus                            { return nil }
func (s stubEngineWithPlugin) Logger() log.Logger                          { return log.NewNoopLogger() }
func (s stubEngineWithPlugin) Settings() *settings.Manager                 { return nil }
func (s stubEngineWithPlugin) Chronicle() bridge.Chronicle                 { return nil }
func (s stubEngineWithPlugin) Relay() bridge.EventRelay                    { return nil }
func (s stubEngineWithPlugin) Herald() bridge.Herald                       { return nil }
func (s stubEngineWithPlugin) Mailer() bridge.Mailer                       { return nil }
func (s stubEngineWithPlugin) SMSSender() bridge.SMSSender                 { return nil }
func (s stubEngineWithPlugin) Ledger() bridge.Ledger                       { return nil }
func (s stubEngineWithPlugin) TokenEncryptor() bridge.Encryptor            { return nil }
func (s stubEngineWithPlugin) CeremonyStore() ceremony.Store               { return nil }
func (s stubEngineWithPlugin) APIKeyStore() apikey.Store                   { return nil }
func (s stubEngineWithPlugin) SecurityEvents() securityevent.Store         { return nil }
func (s stubEngineWithPlugin) AuthMiddleware() forge.Middleware            { return nil }
func (s stubEngineWithPlugin) AuthRegistry() auth.Registry                 { return nil }
func (s stubEngineWithPlugin) PlatformAppID() id.AppID                     { return id.Nil }
func (s stubEngineWithPlugin) DefaultAppID() string                        { return "" }
func (s stubEngineWithPlugin) BasePath() string                            { return "" }
func (s stubEngineWithPlugin) TokenFormatForApp(string) tokenformat.Format { return nil }

func (s stubEngineWithPlugin) SessionConfigForApp(context.Context, id.AppID, ...id.EnvironmentID) account.SessionConfig {
	return account.SessionConfig{}
}
func (s stubEngineWithPlugin) ResolveSessionByToken(string) (*session.Session, error) {
	return nil, nil
}
func (s stubEngineWithPlugin) ResolveUser(string) (*user.User, error) { return nil, nil }
func (s stubEngineWithPlugin) GetUser(context.Context, id.UserID) (*user.User, error) {
	return nil, nil
}
func (s stubEngineWithPlugin) EnsureDefaultRole(context.Context, id.AppID, id.UserID) {}
func (s stubEngineWithPlugin) ResolvePrincipal(context.Context, principal.Ref) (*principal.Principal, error) {
	return nil, principal.ErrNotFound
}
func (s stubEngineWithPlugin) PrincipalStore() principal.Store { return nil }
func (s stubEngineWithPlugin) Can(context.Context, principal.Ref, principal.Chain, string, string) (bool, error) {
	return false, nil
}

// newSSOPluginWithPlugin builds an sso Plugin whose engine returns p under
// "sharedsignals" and nil for every other name.
func newSSOPluginWithPlugin(t *testing.T, p plugin.Plugin) *Plugin {
	t.Helper()
	pl := New()
	pl.engine = stubEngineWithPlugin{name: "sharedsignals", p: p}
	pl.logger = log.NewNoopLogger()
	return pl
}

func TestLinkSharedSignalsSubject_SkipsWhenPluginAbsent(_ *testing.T) {
	p := &Plugin{}
	// No engine means no plugin registry; this must not panic.
	p.linkSharedSignalsSubject(context.Background(),
		&user.User{ID: id.NewUserID()}, "https://i", "u1")
}

func TestLinkSharedSignalsSubject_SkipsEmptyValues(t *testing.T) {
	r := &recordingLinker{}
	p := newSSOPluginWithPlugin(t, r)
	u := &user.User{ID: id.NewUserID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID()}

	p.linkSharedSignalsSubject(context.Background(), u, "", "u1")
	p.linkSharedSignalsSubject(context.Background(), u, "https://i", "")
	assert.Zero(t, r.calls)
}

func TestLinkSharedSignalsSubject_RecordsLink(t *testing.T) {
	r := &recordingLinker{}
	p := newSSOPluginWithPlugin(t, r)
	u := &user.User{ID: id.NewUserID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID()}

	p.linkSharedSignalsSubject(context.Background(), u, "https://org.okta.com", "okta-user-1")
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, "https://org.okta.com", r.issuer)
	assert.Equal(t, "okta-user-1", r.subject)
	assert.Equal(t, u.ID, r.userID)
}

func (s stubEngineWithPlugin) RateLimiter() ratelimit.Limiter { return nil }
