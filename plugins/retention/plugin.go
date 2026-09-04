package retention

import (
	"context"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin                = (*Plugin)(nil)
	_ plugin.OnInit                = (*Plugin)(nil)
	_ plugin.OnShutdown            = (*Plugin)(nil)
	_ plugin.MigrationProvider     = (*Plugin)(nil)
	_ plugin.SettingsProvider      = (*Plugin)(nil)
	_ plugin.DataExportContributor = (*Plugin)(nil)
)

// The four hook interface checks (AfterSignUp, AfterSignIn, AfterSignOut,
// AfterUserUpdate) live in hooks.go, added in Task 7, not here. A
// compile-time assertion for a method that does not exist yet would stop
// this task from building at all.

// ──────────────────────────────────────────────────
// Dynamic settings
// ──────────────────────────────────────────────────

var (
	// SettingEnabled turns delivery off without dropping queued work.
	//
	// The worker reads it per app immediately before delivering, so
	// flipping it off during a CRM incident stops the sends and leaves
	// the backlog claimable. See deliveryEnabled.
	SettingEnabled = settings.Define("retention.enabled", true,
		settings.WithDisplayName("CRM Retention Sync Enabled"),
		settings.WithDescription("Mirror signup and login activity into the configured CRM"),
		settings.WithCategory("Retention"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithOrder(10),
	)

	// SettingRequireConsent gates delivery on an active consent grant.
	//
	// Defaults to false on purpose. Fail-closed reads as the responsible
	// default, and would be right if purposes were a fixed enum. They are
	// free text, so fail-closed means a fresh install watches the plugin do
	// nothing and files a bug.
	SettingRequireConsent = settings.Define("retention.require_consent", false,
		settings.WithDisplayName("Require Consent Before Sync"),
		settings.WithDescription("Only send to the CRM when the user has an active grant for the purpose below"),
		settings.WithCategory("Retention"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Needs the consent plugin registered. With this on, a user with no grant is never sent."),
		settings.WithOrder(20),
	)

	// SettingConsentPurpose is the purpose string the gate checks.
	SettingConsentPurpose = settings.Define("retention.consent_purpose", "marketing",
		settings.WithDisplayName("Consent Purpose"),
		settings.WithDescription("The consent purpose that authorises CRM sync"),
		settings.WithCategory("Retention"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithOrder(30),
	)
)

// ProviderConfig configures one CRM destination. Type selects the
// implementation: "hubspot" for the vendor provider, "generic" for the
// config-driven REST one.
type ProviderConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Token       string            `json:"token,omitempty"`
	AuthType    string            `json:"auth_type,omitempty"`    // "bearer" (default) or "header"
	AuthHeader  string            `json:"auth_header,omitempty"`  // used when AuthType is "header"
	BaseURL     string            `json:"base_url,omitempty"`     // overridden in tests
	ContactURL  string            `json:"contact_url,omitempty"`  // generic only
	ActivityURL string            `json:"activity_url,omitempty"` // generic only; empty means no CapActivities
	FieldMap    map[string]string `json:"field_map,omitempty"`    // generic only
}

// Config is the plugin's static configuration. Everything here has a working
// default except Providers, and an empty Providers list makes every hook a
// no-op rather than an error.
type Config struct {
	Providers    []ProviderConfig `json:"providers"`
	TickInterval time.Duration    `json:"tick_interval"` // default 30s
	Lease        time.Duration    `json:"lease"`         // default 2m
	BaseBackoff  time.Duration    `json:"base_backoff"`  // default 5s
	BatchSize    int              `json:"batch_size"`    // default 50
	MaxAttempts  int              `json:"max_attempts"`  // default 8
}

// defaults fills the zero values, matching the Config.defaults() convention in
// plugins/sharedsignals/plugin.go.
func (c *Config) defaults() {
	if c.TickInterval <= 0 {
		c.TickInterval = 30 * time.Second
	}
	if c.Lease <= 0 {
		c.Lease = 2 * time.Minute
	}
	if c.BaseBackoff <= 0 {
		c.BaseBackoff = 5 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 8
	}
}

// Plugin mirrors signup/login activity into a CRM. See the package doc in
// provider.go for the outbox/worker shape.
type Plugin struct {
	config Config

	store     Store
	providers map[string]Provider

	engine      plugin.Engine
	logger      log.Logger
	settingsMgr *settings.Manager

	worker *worker

	// consent is the resolved consent plugin, if one is registered. Nil means
	// no consent plugin is available; allowSend treats that as "cannot
	// evaluate the gate" and refuses to send whenever the gate is on.
	consent consentChecker

	// consentPolicy resolves the consent gate for one app at delivery time.
	//
	// A function field rather than two cached booleans, because
	// retention.require_consent and retention.consent_purpose are ScopeApp:
	// one process serves apps that disagree about them, and a value read once
	// during OnInit would hand the first app's answer to every other app.
	// They are dynamic settings too, so enabling the gate should bite without
	// a restart. Tests stub this directly, the same way workerDeps.AllowSend
	// is stubbed.
	consentPolicy func(ctx context.Context, appID id.AppID) (require bool, purpose string)

	// enabledPolicy resolves retention.enabled for one app at delivery
	// time. A function field for the same reasons consentPolicy is one:
	// the setting is ScopeApp and dynamic, so a value read once during
	// OnInit would hand the first app's answer to every other app and
	// would ignore the operator flipping the switch. Nil means enabled.
	enabledPolicy func(ctx context.Context, appID id.AppID) bool
}

// New builds the plugin. Config is optional.
//
// logger is set to a no-op here, not left nil, because Task 7's hooks log an
// enqueue failure and can run before OnInit in tests. A nil logger there
// would turn a swallowed store error into a panic inside a login.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{
		config: c,
		logger: log.NewNoopLogger(),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "retention" }

// SetStore overrides the backing store. Tests use it; production wiring goes
// through OnInit.
func (p *Plugin) SetStore(s Store) { p.store = s }

// RegisterProvider adds or replaces a CRM provider by name. Tests use it to
// inject fakes; production wiring builds providers from Config in OnInit.
func (p *Plugin) RegisterProvider(pr Provider) {
	if p.providers == nil {
		p.providers = make(map[string]Provider)
	}
	p.providers[pr.Name()] = pr
}

// OnInit captures engine references, picks a store for the driver in use,
// builds the configured providers and starts the delivery worker.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.settingsMgr = engine.Settings()

	p.logger = engine.Logger()
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
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
				p.logger.Warn(
					"retention: unrecognised database driver, falling back to the in-memory store",
					log.String("driver", driver),
					log.String("impact",
						"a pending backlog is process-local and dies with the process; "+
							"nothing enqueued between now and a restart will ever reach the CRM"),
				)
			}
		}
	}
	if p.store == nil {
		p.store = NewMemoryStore()
	}

	// The SQL and Mongo stores can hit a row they cannot map, and
	// skipping it silently would leave nothing to grep for. MemoryStore
	// has no mapping step and so does not implement this.
	if sl, ok := p.store.(interface{ SetLogger(log.Logger) }); ok {
		sl.SetLogger(p.logger)
	}

	built, err := buildProviders(p.config.Providers)
	if err != nil {
		return fmt.Errorf("retention: %w", err)
	}
	if p.providers == nil {
		p.providers = make(map[string]Provider, len(built))
	}
	for name, pr := range built {
		p.providers[name] = pr
	}

	if cp := engine.Plugin("consent"); cp != nil {
		if checker, ok := cp.(consentChecker); ok {
			p.consent = checker
		}
	}

	// consentPolicy is left nil when there is no settings manager (the stub
	// engine used in tests returns nil), matching require_consent's default
	// of false: allowSend treats a nil policy as "gate off".
	if p.settingsMgr != nil {
		p.consentPolicy = newConsentPolicy(p.settingsMgr, p.logger)
		p.enabledPolicy = newEnabledPolicy(p.settingsMgr, p.logger)
	}

	p.worker = newWorker(workerDeps{
		Store:       p.store,
		Providers:   p.providers,
		Logger:      p.logger,
		Interval:    p.config.TickInterval,
		Lease:       p.config.Lease,
		BatchSize:   p.config.BatchSize,
		MaxAttempts: p.config.MaxAttempts,
		BaseBackoff: p.config.BaseBackoff,
		LoadContact: p.loadContact,
		Enabled:     p.deliveryEnabled,
		AllowSend:   p.allowSend,
	})
	p.worker.start()

	return nil
}

// loadContact reloads the user at delivery time rather than trusting the
// enqueued payload, so a changed email or a since-deleted user is honoured
// instead of a stale snapshot from enqueue time.
func (p *Plugin) loadContact(ctx context.Context, j *Job) (*Contact, error) {
	u, err := p.engine.GetUser(ctx, j.UserID)
	if err != nil {
		// Our engine and its store, not the CRM. A user store that is
		// briefly unreachable must not permanently dead-letter the job.
		return nil, localError(fmt.Sprintf("retention: load user %s", j.UserID), err)
	}
	if u == nil {
		return nil, fmt.Errorf("retention: user %s not found", j.UserID)
	}
	return &Contact{
		UserID:    u.ID,
		AppID:     u.AppID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}, nil
}

// newEnabledPolicy builds the enabledPolicy closure against a live settings
// manager, the same shape as newConsentPolicy so both are stubbable and
// both resolve per app rather than once at startup.
//
// An unreadable setting resolves to enabled, the opposite of the consent
// gate, and deliberately so. Consent gates PII leaving the building, where
// uncertainty has to block. This gates a feature: a settings-store outage
// must not silently stop a working integration, which is a failure nobody
// would think to look for.
//
// mgr and logger must not be nil; OnInit only calls this once it has both.
func newEnabledPolicy(mgr *settings.Manager, logger log.Logger) func(ctx context.Context, appID id.AppID) bool {
	return func(ctx context.Context, appID id.AppID) bool {
		opts := settings.ResolveOpts{AppID: appID.String()}
		enabled, err := settings.Get(ctx, mgr, SettingEnabled, opts)
		if err != nil {
			logger.Warn("retention: enabled setting unreadable, delivering anyway",
				log.String("app_id", appID.String()),
				log.String("error", err.Error()))
			return true
		}
		return enabled
	}
}

// deliveryEnabled is the worker's Enabled. A nil policy (no settings
// manager, which is what the stub engine in tests gives us) means enabled,
// matching the setting's registered default of true.
func (p *Plugin) deliveryEnabled(ctx context.Context, j *Job) bool {
	if p.enabledPolicy == nil {
		return true
	}
	return p.enabledPolicy(ctx, j.AppID)
}

// buildProviders maps each ProviderConfig onto an implementation by Type,
// returning an error for an unknown type so a typo in config fails at
// startup rather than dead-lettering every job later.
func buildProviders(cfgs []ProviderConfig) (map[string]Provider, error) {
	out := make(map[string]Provider, len(cfgs))
	for _, c := range cfgs {
		switch c.Type {
		case "generic":
			p, err := NewGenericProvider(c)
			if err != nil {
				return nil, err
			}
			out[c.Name] = p
		case "hubspot":
			p, err := NewHubSpotProvider(c)
			if err != nil {
				return nil, err
			}
			out[c.Name] = p
		default:
			return nil, fmt.Errorf("unknown provider type %q for provider %q", c.Type, c.Name)
		}
	}
	return out, nil
}

// OnShutdown stops the delivery worker. Safe to call on a plugin whose
// OnInit never ran, since worker is then nil.
func (p *Plugin) OnShutdown(_ context.Context) error {
	if p.worker != nil {
		p.worker.stop()
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
	if err := settings.RegisterTyped(m, "retention", SettingEnabled); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "retention", SettingRequireConsent); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "retention", SettingConsentPurpose)
}
