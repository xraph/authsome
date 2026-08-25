package sharedsignals

import (
	"context"

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

// stubEngine is the smallest thing that satisfies plugin.Engine. It exists so
// OnInit can be tested without standing up a real engine.
type stubEngine struct{}

var _ plugin.Engine = stubEngine{}

func (stubEngine) Store() store.Store                          { return nil }
func (stubEngine) DB() *grove.DB                               { return nil }
func (stubEngine) Plugins() *plugin.Registry                   { return nil }
func (stubEngine) Plugin(string) plugin.Plugin                 { return nil }
func (stubEngine) Hooks() *hook.Bus                            { return nil }
func (stubEngine) Logger() log.Logger                          { return log.NewNoopLogger() }
func (stubEngine) Settings() *settings.Manager                 { return nil }
func (stubEngine) Chronicle() bridge.Chronicle                 { return nil }
func (stubEngine) Relay() bridge.EventRelay                    { return nil }
func (stubEngine) Herald() bridge.Herald                       { return nil }
func (stubEngine) Mailer() bridge.Mailer                       { return nil }
func (stubEngine) SMSSender() bridge.SMSSender                 { return nil }
func (stubEngine) Ledger() bridge.Ledger                       { return nil }
func (stubEngine) TokenEncryptor() bridge.Encryptor            { return nil }
func (stubEngine) CeremonyStore() ceremony.Store               { return nil }
func (stubEngine) APIKeyStore() apikey.Store                   { return nil }
func (stubEngine) SecurityEvents() securityevent.Store         { return nil }
func (stubEngine) AuthMiddleware() forge.Middleware            { return nil }
func (stubEngine) AuthRegistry() auth.Registry                 { return nil }
func (stubEngine) PlatformAppID() id.AppID                     { return id.Nil }
func (stubEngine) DefaultAppID() string                        { return "" }
func (stubEngine) BasePath() string                            { return "" }
func (stubEngine) TokenFormatForApp(string) tokenformat.Format { return nil }

func (stubEngine) SessionConfigForApp(context.Context, id.AppID, ...id.EnvironmentID) account.SessionConfig {
	return account.SessionConfig{}
}
func (stubEngine) ResolveSessionByToken(string) (*session.Session, error) { return nil, nil }
func (stubEngine) ResolveUser(string) (*user.User, error)                 { return nil, nil }
func (stubEngine) GetUser(context.Context, id.UserID) (*user.User, error) { return nil, nil }
func (stubEngine) EnsureDefaultRole(context.Context, id.AppID, id.UserID) {}
func (stubEngine) ResolvePrincipal(context.Context, principal.Ref) (*principal.Principal, error) {
	return nil, principal.ErrNotFound
}
func (stubEngine) PrincipalStore() principal.Store { return nil }

// Can denies unconditionally, which is correct only because bare stubEngine
// has no HasPermission to disagree with: it does not satisfy
// plugin.PermissionChecker at all, so nothing can reach it through both
// questions at once and get two different answers.
//
// A double that DOES answer both must keep them consistent. fakeAuthEngine
// (admin_test.go) embeds this type and adds a scripted HasPermission, so it
// overrides this method rather than inheriting it. Leaving it inherited was a
// real bug: middleware.RequirePermission reaches for Can when a caller carries
// an actor chain, and a stub Can under a scripted HasPermission denies exactly
// the requests the script was written to allow.
func (stubEngine) Can(context.Context, principal.Ref, principal.Chain, string, string) (bool, error) {
	return false, nil
}
