package sharedsignals

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.OnShutdown        = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
	_ plugin.SettingsProvider  = (*Plugin)(nil)
	_ plugin.RouteProvider     = (*Plugin)(nil)
)

func intPtr(v int) *int { return &v }

// logString is a tiny alias so call sites do not repeat the import path of
// the logging package in every field.
func logString(key, value string) log.Field { return log.String(key, value) }

// ──────────────────────────────────────────────────
// Dynamic settings
// ──────────────────────────────────────────────────

var (
	// SettingEnabled turns the receiver on and off without deleting streams.
	SettingEnabled = settings.Define("sharedsignals.enabled", true,
		settings.WithDisplayName("Shared Signals Enabled"),
		settings.WithDescription("Accept inbound CAEP events from configured streams"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Turn the receiver off to stop acting on inbound events without deleting any stream"),
		settings.WithOrder(10),
	)

	// SettingSignalTTLHours is how long a received signal keeps affecting risk.
	SettingSignalTTLHours = settings.Define("sharedsignals.signal_ttl_hours", 24,
		settings.WithDisplayName("Signal TTL (hours)"),
		settings.WithDescription("How long a received CAEP signal keeps influencing the risk score"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(1), Max: intPtr(720)}),
		settings.WithHelpText("Signals decay to zero over this window. Default: 24"),
		settings.WithOrder(20),
	)

	// SettingMaxActionsPerHour is the circuit breaker default for new streams.
	SettingMaxActionsPerHour = settings.Define("sharedsignals.max_actions_per_hour", 100,
		settings.WithDisplayName("Max Actions Per Hour"),
		settings.WithDescription("Actions one stream may take in an hour before it is paused"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(1), Max: intPtr(100000)}),
		settings.WithHelpText("A transmitter that crosses this is paused and an alert is raised"),
		settings.WithOrder(30),
	)

	// SettingCAEPSignalWeight is the weight the risk engine gives our signals.
	SettingCAEPSignalWeight = settings.Define("sharedsignals.risk_weight", 2,
		settings.WithDisplayName("Risk Weight"),
		settings.WithDescription("Weight the risk engine applies to CAEP signals"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(1), Max: intPtr(10)}),
		settings.WithHelpText("A CAEP event is higher confidence than an IP heuristic. Default: 2"),
		settings.WithOrder(40),
	)

	// SettingMaxRiskScore caps what a CAEP signal contributes to the score.
	SettingMaxRiskScore = settings.Define("sharedsignals.max_risk_score", 84,
		settings.WithDisplayName("Max Risk Score"),
		settings.WithDescription("Highest score a Shared Signals event may contribute"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(1), Max: intPtr(100)}),
		settings.WithHelpText("84 keeps a confirmed compromise in the challenge band. Raise to 100 to block the sign-in outright"),
		settings.WithOrder(50),
	)
)

// Config configures the plugin. Every field has a sane default.
type Config struct {
	// Audience is the aud value inbound SETs must carry. Streams inherit it
	// at creation when they do not set their own.
	Audience string

	// SignalTTL is how long a stored signal stays active.
	SignalTTL time.Duration

	// MaxActionsPerHour is the circuit breaker default for new streams.
	MaxActionsPerHour int

	// ClockSkew is the tolerance for an iat slightly in the future.
	ClockSkew time.Duration

	// MaxSETAge is how far in the past an iat may be. Transmitters retry for
	// a long time, so this is generous.
	MaxSETAge time.Duration

	// MaxBodyBytes bounds the push request body.
	MaxBodyBytes int64

	// MaxRiskScore caps the score this plugin hands to the risk engine.
	//
	// severityFor scores a session-revoked event at 100, which is honest: an
	// identity provider that watched an account be taken over is as certain as
	// a signal gets, and that number is what the stored signal keeps for
	// forensics. But a single contributor's score becomes the whole composite,
	// and riskengine blocks at 85 by default, so handing 100 straight through
	// refused the sign-in outright rather than stepping it up. A user with
	// correct credentials and a correct second factor could not get in.
	//
	// The default of 84 sits one below that threshold, so a CAEP signal
	// challenges. Raise it to 100 if you would rather a confirmed compromise
	// bar the door; the number is a policy statement, not a measurement.
	MaxRiskScore int

	// KeyRefreshInterval is how often cached JWKS entries are re-fetched in
	// the background. A value of zero takes the default; a negative value
	// turns the ticker off entirely, for an embedder that does not want a
	// background goroutine and accepts that key freshness then depends
	// wholly on inbound traffic.
	KeyRefreshInterval time.Duration
}

func (c *Config) defaults() {
	if c.SignalTTL == 0 {
		c.SignalTTL = 24 * time.Hour
	}
	if c.MaxActionsPerHour == 0 {
		c.MaxActionsPerHour = 100
	}
	if c.ClockSkew == 0 {
		c.ClockSkew = 5 * time.Minute
	}
	if c.MaxSETAge == 0 {
		c.MaxSETAge = 24 * time.Hour
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = 64 * 1024
	}
	if c.MaxRiskScore == 0 {
		c.MaxRiskScore = 84
	}
	if c.KeyRefreshInterval == 0 {
		// Four times inside the JWKS client's one-hour MaxKeyAge, so an
		// entry is re-confirmed well before ordinary traffic would ever find
		// it stale, and roughly fifty times inside its twelve-hour hard
		// limit. That ratio is the point: hitting the hard limit cannot be a
		// blip or an unlucky gap in traffic, it can only be an endpoint that
		// has been unreachable for half a day.
		c.KeyRefreshInterval = 15 * time.Minute
	}
}

// Plugin receives Shared Signals events and turns them into session
// revocations and durable risk signals.
type Plugin struct {
	config Config

	store     Store
	authStore store.Store
	jwks      *jwksclient.Client

	engine      plugin.Engine
	revoker     plugin.SessionRevoker
	logger      log.Logger
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	hooks       *hook.Bus
	settingsMgr *settings.Manager
	refresher   *refresher
}

// New builds the plugin. Config is optional.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "sharedsignals" }

// SetStore overrides the backing store. Tests use it; production wiring goes
// through OnInit.
func (p *Plugin) SetStore(s Store) { p.store = s }

// OnInit captures engine references and picks a store for the driver in use.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.authStore = engine.Store()
	p.chronicle = engine.Chronicle()
	p.relay = engine.Relay()
	p.hooks = engine.Hooks()
	p.settingsMgr = engine.Settings()

	p.logger = engine.Logger()
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	// Revoking through the engine keeps the AfterSessionRevoke hooks, the
	// hook bus and the outbound relay firing. An engine that cannot revoke
	// leaves us able to record signals but not to act, which the receiver
	// reports rather than hiding.
	if r, ok := engine.(plugin.SessionRevoker); ok {
		p.revoker = r
	}

	if p.store == nil {
		if db := engine.DB(); db != nil {
			driver := db.Driver().Name()
			switch driver {
			case "pg":
				p.store = NewPostgresStore(db)
			case "sqlite":
				p.store = NewSqliteStore(db)
			case "mongo":
				p.store = NewMongoStore(db)
			default:
				// The host has a database, we just do not know how to talk
				// to it, and the fall-through below is about to hand this
				// plugin a process-local store. For most plugins that is a
				// degraded cache. For this one it is three security
				// controls going soft at once, so it is an error, not a
				// debug line.
				p.logger.Error(
					"sharedsignals: unrecognised database driver, falling back to the in-memory store",
					logString("driver", driver),
					logString("impact",
						"the replay guard, the circuit breaker counter and the signal history "+
							"are process-local and will not survive a restart"),
				)
			}
		}
	}
	if p.store == nil {
		p.store = NewMemoryStore()
	}

	p.jwks = jwksclient.New(jwksclient.Options{})

	if p.config.KeyRefreshInterval > 0 {
		p.refresher = newRefresher(p.config.KeyRefreshInterval, p.refreshKeys)
		p.refresher.start()
	}

	return nil
}

// refreshKeys is one background round: re-fetch every JWKS URI the client
// has cached and report the ones that did not answer.
//
// The failures are logged at Warn rather than swallowed because nothing else
// in the process can see them. An inbound token failing verification is
// visible in the receiver's own error path, but an endpoint that has quietly
// stopped answering the ticker produces no traffic at all -- and the first
// symptom, twelve hours later, is a stream that starts refusing SETs.
func (p *Plugin) refreshKeys(ctx context.Context) {
	for _, failure := range p.jwks.RefreshAll(ctx) {
		p.logger.Warn("sharedsignals: background JWKS refresh failed",
			logString("jwks_uri", failure.URI),
			logString("error", failure.Err.Error()),
		)
	}
}

// OnShutdown stops the background JWKS refresh and waits for any round
// already in flight, so the process does not exit with an outbound fetch
// still open against an IdP.
func (p *Plugin) OnShutdown(_ context.Context) error {
	if p.refresher != nil {
		p.refresher.stop()
	}
	return nil
}

// MigrationGroups implements plugin.MigrationProvider.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg", "postgres", "postgresql":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite", "sqlite3":
		return []*migrate.Group{SqliteMigrations}
	case "mongo", "mongodb":
		return []*migrate.Group{MongoMigrations}
	default:
		return nil
	}
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "sharedsignals", SettingEnabled); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "sharedsignals", SettingSignalTTLHours); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "sharedsignals", SettingMaxActionsPerHour); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "sharedsignals", SettingCAEPSignalWeight); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "sharedsignals", SettingMaxRiskScore)
}

// RegisterRoutes implements plugin.RouteProvider. The receiver endpoint lands
// in receiver.go and the admin CRUD in admin.go.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if err := p.registerReceiverRoutes(router); err != nil {
		return err
	}
	return p.registerAdminRoutes(router)
}
