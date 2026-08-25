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
