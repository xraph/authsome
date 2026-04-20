// Package authsome provides a composable authentication engine for the Forge ecosystem.
//
// AuthSome v0.5.0 is an interface-driven, store-abstracted authentication engine
// with a type-cached plugin registry, global hook bus, strategy-based authentication,
// and optional bridges to forgery extensions (Chronicle, Warden, Keysmith, Relay).
package authsome

import (
	"context"
	"errors"
	"fmt"
	"strings"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/authz"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/bridge/ledgeradapter"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/lockout"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"

	"github.com/xraph/grove"
	"github.com/xraph/keysmith"
	xledger "github.com/xraph/ledger"
	ledgerstore "github.com/xraph/ledger/store"
	"github.com/xraph/warden"
	wardenid "github.com/xraph/warden/id"
	"github.com/xraph/warden/resourcetype"
)

// Engine is the core AuthSome orchestrator. It holds only interfaces —
// no concrete implementations — for maximum pluggability.
type Engine struct {
	config Config
	store  store.Store
	logger log.Logger

	// Plugin system
	plugins            *plugin.Registry
	hooks              *hook.Bus
	strategies         *strategy.Registry
	pendingPlugins     []plugin.Plugin
	pendingStrategies  []pendingStrategy
	pendingAppSessCfgs []*appsessionconfig.Config

	// First-class authorization engine (optional, replaces bridge for RBAC)
	wardenEng *warden.Engine

	// First-class key management engine (optional, replaces bridge for API keys)
	keysmithEng *keysmith.Engine

	// First-class billing engine (optional, provides store access for subscription plugin)
	ledgerEng *xledger.Ledger

	// Optional forgery bridges (injected, not required)
	chronicle    bridge.Chronicle
	authorizer   bridge.Authorizer
	keyManager   bridge.KeyManager
	relay        bridge.EventRelay
	mailer       bridge.Mailer
	sms          bridge.SMSSender
	heraldBridge bridge.Herald
	vault        bridge.Vault
	dispatcher   bridge.Dispatcher
	ledger       bridge.Ledger
	metrics      bridge.MetricsCollector

	// Ceremony state store for multi-instance auth ceremonies
	ceremonyStore ceremony.Store

	// Token format (default: opaque; per-app overrides via AppSessionConfig.TokenFormat)
	defaultTokenFormat tokenformat.Format
	jwtFormats         map[string]*tokenformat.JWT // keyed by app ID

	// Security
	rateLimiter     ratelimit.Limiter
	lockout         lockout.Tracker
	passwordHistory account.PasswordHistoryStore
	securityEvents  securityevent.Store

	// Dynamic settings manager (optional).
	settingsMgr *settings.Manager

	// Bootstrap configuration (nil = disabled).
	bootstrapCfg  *BootstrapConfig
	platformAppID id.AppID

	// Database reference for plugins that need direct database access
	// to create their own persistent stores.
	db *grove.DB

	// Cached auth middleware — built once during InitPlugins() with full
	// capability detection (JWT + strategies + cookie bridge).
	authMiddleware forge.Middleware

	// Forge auth provider registry — plugins register their providers here.
	authRegistry auth.Registry

	pluginsInitialized bool
	started            bool
}

// NewEngine creates a new AuthSome engine with the given options.
func NewEngine(opts ...Option) (*Engine, error) {
	e := &Engine{
		config: DefaultConfig(),
		logger: log.NewNoopLogger(),
	}

	for _, opt := range opts {
		opt(e)
	}

	if e.store == nil {
		return nil, errors.New("authsome: store is required")
	}
	if e.wardenEng == nil {
		return nil, errors.New("authsome: warden engine is required (use WithWarden option)")
	}

	// Initialize subsystems
	e.plugins = plugin.NewRegistry(e.logger)
	e.hooks = hook.NewBus(e.logger)
	e.strategies = strategy.NewRegistry(e.logger)

	// Initialize the dynamic settings manager.
	e.settingsMgr = settings.NewManager(e.store, e.logger)

	// Register core session settings.
	if err := registerCoreSessionSettings(e.settingsMgr); err != nil {
		return nil, fmt.Errorf("authsome: failed to register core session settings: %w", err)
	}

	// Register pending plugins
	for _, p := range e.pendingPlugins {
		e.plugins.Register(p)
	}
	e.pendingPlugins = nil

	// Register pending strategies
	for _, ps := range e.pendingStrategies {
		e.strategies.Register(ps.strategy, ps.priority)
	}
	e.pendingStrategies = nil

	return e, nil
}

// ErrNotStarted is returned when a service method is called before Start().
var ErrNotStarted = errors.New("authsome: engine not started, please wait for initialization to complete")

// EnsureMigrated runs store migrations (core + plugin groups) if not disabled.
// This is idempotent — safe to call multiple times. Use this to ensure tables
// exist before the engine is fully started (e.g. during extension init).
func (e *Engine) EnsureMigrated(ctx context.Context) error {
	if e.config.DisableMigrate {
		return nil
	}
	extraGroups := e.plugins.CollectMigrationGroups(e.config.DriverName)
	return e.store.Migrate(ctx, extraGroups...)
}

// requireStarted returns ErrNotStarted if the engine has not been started.
func (e *Engine) requireStarted() error {
	if !e.started {
		return ErrNotStarted
	}
	return nil
}

// buildAuthMiddleware constructs the fully-configured auth middleware with
// capability detection (JWT, strategies) and the cookie-to-header bridge.
// Called once during InitPlugins so both the extension and plugins share
// the same middleware instance.
func (e *Engine) buildAuthMiddleware() {
	var inner forge.Middleware

	bindCfg := middleware.SessionBindingConfig{
		CookieNameResolver: e.resolveSessionCookieName,
		JWTSessionChecker:  e.jwtSessionChecker,
	}

	switch {
	case e.HasJWT():
		inner = middleware.AuthMiddlewareWithJWT(
			e.ResolveSessionByToken,
			e.ResolveUser,
			e.Strategies(),
			e,
			e.Logger(),
			bindCfg,
		)
	case e.HasStrategies():
		inner = middleware.AuthMiddlewareWithStrategies(
			e.ResolveSessionByToken,
			e.ResolveUser,
			e.Strategies(),
			e.Logger(),
			bindCfg,
		)
	default:
		inner = middleware.AuthMiddleware(
			e.ResolveSessionByToken,
			e.ResolveUser,
			e.Logger(),
			bindCfg,
		)
	}

	// Wrap with cookie-to-header bridge: when no Authorization header is
	// present, read the auth_token cookie (set during browser login) or
	// the dynamic session cookie and inject it as a Bearer token.
	e.authMiddleware = func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			r := ctx.Request()
			if r.Header.Get("Authorization") == "" {
				if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
					r.Header.Set("Authorization", "Bearer "+cookie.Value)
				} else {
					// Fall back to the dynamic session cookie name from settings.
					cookieName := e.resolveSessionCookieName(ctx.Context())
					if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
						r.Header.Set("Authorization", "Bearer "+cookie.Value)
					}
				}
			}
			return inner(next)(ctx)
		}
	}
}

// AuthMiddleware returns the engine's fully-configured non-blocking
// authentication middleware (cookie bridge + JWT + strategies). Populates
// user context when a valid token is present, passes through otherwise.
// Applied globally by the extension.
func (e *Engine) AuthMiddleware() forge.Middleware { return e.authMiddleware }

// AuthRegistry returns the forge auth provider registry. Plugins can register
// their own auth providers and create middleware via registry.Middleware().
func (e *Engine) AuthRegistry() auth.Registry { return e.authRegistry }

// SetAuthRegistry sets the forge auth provider registry. Called by the
// extension after obtaining the registry from the DI container.
func (e *Engine) SetAuthRegistry(r auth.Registry) { e.authRegistry = r }

// registerSessionAuthProvider registers the core "session" auth provider
// with the forge auth registry. This must be called before plugin OnInit
// so plugins can reference "session" in their auth declarations.
func (e *Engine) registerSessionAuthProvider() {
	if e.authRegistry == nil {
		e.logger.Debug("authsome: no auth registry available, skipping session provider registration")
		return
	}
	provider := authprovider.NewSessionProvider(e.ResolveSessionByToken, e.ResolveUser, e.logger, e.resolveSessionCookieName)
	if err := e.authRegistry.Register(provider); err != nil {
		e.logger.Warn("authsome: failed to register session auth provider",
			log.String("error", err.Error()),
		)
	} else {
		e.logger.Info("authsome: registered 'session' auth provider with forge auth registry")
	}
}

// resolveSessionCookieName returns the session cookie name for the given
// context by reading from dynamic settings. Falls back to the default
// "authsome_session_token" when settings are unavailable or unset.
func (e *Engine) resolveSessionCookieName(ctx context.Context) string {
	mgr := e.Settings()
	if mgr == nil {
		return "authsome_session_token"
	}
	opts := settings.ResolveOpts{}
	if appID, ok := middleware.AppIDFrom(ctx); ok {
		opts.AppID = appID.String()
	}
	name, err := settings.Get(ctx, mgr, SettingCookieName, opts)
	if err != nil || name == "" {
		return "authsome_session_token"
	}
	return name
}

// jwtSessionChecker checks whether a JWT's session ID still exists in the
// store. This enables JWT revocation — revoked sessions are rejected even if
// the JWT signature is valid. The SettingJWTRequireActiveSession setting
// controls whether this check is active; when disabled, a non-nil sentinel
// session is returned to skip binding checks.
func (e *Engine) jwtSessionChecker(sessionIDStr string) (*session.Session, error) {
	ctx := context.Background()

	// Check if the feature is enabled via dynamic settings.
	mgr := e.Settings()
	if mgr != nil {
		enabled, _ := settings.Get(ctx, mgr, SettingJWTRequireActiveSession, settings.ResolveOpts{}) //nolint:errcheck // best-effort
		if !enabled {
			return nil, nil //nolint:nilnil // nil,nil signals "feature disabled, skip check"
		}
	}

	sessID, err := id.ParseSessionID(sessionIDStr)
	if err != nil {
		return nil, fmt.Errorf("authsome: invalid session ID in JWT: %w", err)
	}
	return e.store.GetSession(ctx, sessID)
}

// ensureAuthRegistry creates a local in-memory auth registry if no external
// one was provided (e.g. forge auth extension not registered in the app).
// This ensures AuthRegistry() never returns nil and plugins can always
// register providers and create middleware.
func (e *Engine) ensureAuthRegistry() {
	if e.authRegistry != nil {
		return
	}
	e.authRegistry = auth.NewRegistry(nil, e.logger)
	e.logger.Debug("authsome: created local auth registry (forge auth extension not available)")
}

// InitPlugins builds the auth middleware, registers the session auth provider,
// and calls OnInit on all registered plugins. Idempotent — safe to call
// multiple times. The extension calls this before route registration so
// plugins can set up dependencies that RegisterRoutes relies on.
func (e *Engine) InitPlugins(ctx context.Context) {
	if e.pluginsInitialized {
		return
	}
	e.buildAuthMiddleware()
	e.ensureAuthRegistry()
	e.registerSessionAuthProvider()
	e.plugins.EmitOnInit(ctx, e)
	e.pluginsInitialized = true
}

// Start initializes the engine, runs migrations, and starts plugins.
func (e *Engine) Start(ctx context.Context) error {
	if e.started {
		return nil
	}

	// Run store migrations (idempotent — may have been called earlier via EnsureMigrated).
	if err := e.EnsureMigrated(ctx); err != nil {
		return err
	}

	// Register webhook event catalog with relay (before bootstrap so
	// events emitted during bootstrap are recognized).
	if e.relay != nil {
		if err := e.relay.RegisterEventTypes(ctx, bridge.WebhookEventCatalog()); err != nil {
			e.logger.Warn("authsome: failed to register webhook event catalog",
				log.String("error", err.Error()),
			)
		}
	}

	// Initialize plugins (idempotent — may have been called earlier by
	// the extension before route registration).
	e.InitPlugins(ctx)

	// Seed per-app session configs (from code options or YAML config).
	for _, cfg := range e.pendingAppSessCfgs {
		if err := e.store.SetAppSessionConfig(ctx, cfg); err != nil {
			e.logger.Warn("authsome: failed to seed app session config",
				log.String("app_id", cfg.AppID.String()),
				log.String("error", err.Error()),
			)
		}
	}
	e.pendingAppSessCfgs = nil

	// Bootstrap platform app (if configured).
	if e.bootstrapCfg != nil {
		if err := e.bootstrap(ctx); err != nil {
			return err
		}
	}

	// Register authsome resource types with Warden (idempotent).
	// This must run AFTER bootstrap so that platformAppID is set and
	// resource types are created with the correct tenant ID.
	if e.wardenEng != nil {
		e.registerWardenResourceTypes(ctx)
	}

	// Distribute the resolved app ID to plugins that need it.
	// This must happen after bootstrap (which may create the platform app).
	if appID := e.config.AppID; appID != "" {
		for _, p := range e.plugins.Plugins() {
			type appIDSetter interface {
				SetAppID(string)
			}
			if s, ok := p.(appIDSetter); ok {
				s.SetAppID(appID)
			}
		}
	}

	// Auto-register strategies from plugins implementing StrategyProvider.
	for _, p := range e.plugins.Plugins() {
		if sp, ok := p.(plugin.StrategyProvider); ok {
			e.strategies.Register(sp.Strategy(), sp.StrategyPriority())
			e.logger.Info("authsome: auto-registered strategy from plugin",
				log.String("plugin", p.Name()),
				log.String("strategy", sp.Strategy().Name()),
			)
		}
	}

	// Auto-register settings from plugins implementing SettingsProvider.
	for _, p := range e.plugins.Plugins() {
		if sp, ok := p.(plugin.SettingsProvider); ok {
			if err := sp.DeclareSettings(e.settingsMgr); err != nil {
				e.logger.Warn("authsome: failed to register plugin settings",
					log.String("plugin", p.Name()),
					log.String("error", err.Error()),
				)
			}
		}
	}

	// Register metrics collector as a hook handler
	if e.metrics != nil {
		e.hooks.On("metrics", func(_ context.Context, event *hook.Event) error {
			outcome := "success"
			if event.Err != nil {
				outcome = "failure"
			}
			e.metrics.RecordEvent(event.Action, event.Resource, outcome, event.Tenant, 0)
			return nil
		})
	}

	// Register security event recorder as a hook handler
	if e.securityEvents != nil {
		e.hooks.On("security_events", func(ctx context.Context, event *hook.Event) error {
			outcome := "success"
			if event.Err != nil {
				outcome = "failure"
			}
			return e.securityEvents.RecordSecurityEvent(ctx, &securityevent.Event{
				Action:    event.Action,
				Outcome:   outcome,
				Metadata:  event.Metadata,
				CreatedAt: event.Timestamp,
			})
		})
	}

	e.started = true
	return nil
}

// Health checks the health of the engine by pinging its store.
func (e *Engine) Health(ctx context.Context) error {
	return e.store.Ping(ctx)
}

// Stop gracefully shuts down the engine and all plugins.
func (e *Engine) Stop(ctx context.Context) error {
	if !e.started {
		return nil
	}
	e.plugins.EmitOnShutdown(ctx)
	e.started = false
	return nil
}

// registerWardenResourceTypes registers authsome's default resource type
// schemas with the Warden store. The operation is idempotent — duplicate
// errors are silently ignored.
func (e *Engine) registerWardenResourceTypes(ctx context.Context) {
	tenantID := e.config.AppID
	if !e.platformAppID.IsNil() {
		tenantID = e.platformAppID.String()
	}

	for _, schema := range authz.DefaultSchemas() {
		rt := &resourcetype.ResourceType{
			ID:          wardenid.NewResourceTypeID(),
			TenantID:    tenantID,
			AppID:       tenantID,
			Name:        schema.Name,
			Description: schema.Description,
			Relations:   schema.Relations,
			Permissions: schema.Permissions,
		}
		if err := e.wardenEng.Store().CreateResourceType(ctx, rt); err != nil {
			// Idempotent — ignore duplicate/already-exists errors.
			e.logger.Debug("authsome: resource type may already exist",
				log.String("name", schema.Name),
				log.String("error", err.Error()),
			)
		}
	}
	e.logger.Info("authsome: warden resource types registered",
		log.Int("count", len(authz.DefaultSchemas())),
	)
}

// ──────────────────────────────────────────────────
// Accessors
// ──────────────────────────────────────────────────

// Compile-time check: *Engine must satisfy plugin.Engine.
var _ plugin.Engine = (*Engine)(nil)

// Store returns the persistence backend.
func (e *Engine) Store() store.Store { return e.store }

// GetUser returns a user by ID. This method allows the Engine to satisfy
// narrow interfaces (e.g. notification plugin's userLookup) without
// exposing the full store.
func (e *Engine) GetUser(ctx context.Context, userID id.UserID) (*user.User, error) {
	return e.store.GetUser(ctx, userID)
}

// Plugins returns the plugin registry.
func (e *Engine) Plugins() *plugin.Registry { return e.plugins }

// Plugin returns a registered plugin by name, or nil if not found.
func (e *Engine) Plugin(name string) plugin.Plugin { return e.plugins.Plugin(name) }

// Hooks returns the global event bus.
func (e *Engine) Hooks() *hook.Bus { return e.hooks }

// DefaultAppID returns the configured app ID string.
func (e *Engine) DefaultAppID() string { return e.config.AppID }

// BasePath returns the URL prefix for auth routes.
func (e *Engine) BasePath() string { return e.config.BasePath }

// Strategies returns the strategy registry.
func (e *Engine) Strategies() *strategy.Registry { return e.strategies }

// HasStrategies returns true if any authentication strategies are registered.
func (e *Engine) HasStrategies() bool { return len(e.strategies.Strategies()) > 0 }

// Logger returns the engine logger.
func (e *Engine) Logger() log.Logger { return e.logger }

// Config returns the engine configuration.
func (e *Engine) Config() Config { return e.config }

// SessionConfigForApp returns the resolved session config for an app,
// applying per-app and optional per-environment overrides on top of the
// global defaults. Plugins should use this instead of hardcoding their
// own session config.
func (e *Engine) SessionConfigForApp(ctx context.Context, appID id.AppID, envIDs ...id.EnvironmentID) account.SessionConfig {
	return e.sessionConfigForApp(ctx, appID, envIDs...)
}

// Chronicle returns the audit trail bridge (may be nil).
func (e *Engine) Chronicle() bridge.Chronicle { return e.chronicle }

// Authorizer returns the authorization bridge (may be nil).
func (e *Engine) Authorizer() bridge.Authorizer { return e.authorizer }

// KeyManager returns the key management bridge (may be nil).
func (e *Engine) KeyManager() bridge.KeyManager { return e.keyManager }

// Relay returns the event relay bridge (may be nil).
func (e *Engine) Relay() bridge.EventRelay { return e.relay }

// Mailer returns the email bridge (may be nil).
func (e *Engine) Mailer() bridge.Mailer { return e.mailer }

// SMSSender returns the SMS bridge (may be nil).
func (e *Engine) SMSSender() bridge.SMSSender { return e.sms }

// Herald returns the Herald notification bridge (may be nil).
func (e *Engine) Herald() bridge.Herald { return e.heraldBridge }

// Vault returns the secrets/feature-flag/config bridge (may be nil).
func (e *Engine) Vault() bridge.Vault { return e.vault }

// Dispatcher returns the job queue bridge (may be nil).
func (e *Engine) Dispatcher() bridge.Dispatcher { return e.dispatcher }

// Ledger returns the billing/metering bridge (may be nil).
func (e *Engine) Ledger() bridge.Ledger { return e.ledger }

// LedgerEngine returns the first-class ledger billing engine (may be nil).
func (e *Engine) LedgerEngine() *xledger.Ledger { return e.ledgerEng }

// LedgerStore returns the underlying ledger store for direct query access
// (e.g. plan/feature/subscription listing). Returns nil if no ledger engine
// was provided via WithLedgerEngine.
func (e *Engine) LedgerStore() ledgerstore.Store {
	if e.ledgerEng != nil {
		return e.ledgerEng.Store()
	}
	return nil
}

// SetLedgerEngine wires a ledger engine into the authsome engine after
// construction. This is the late-binding counterpart to WithLedgerEngine,
// intended for hosts that cannot resolve the ledger at engine-build time
// (typical when both authsome and ledger live in a DI container whose
// registration order is not guaranteed).
//
// It also updates the bridge.Ledger adapter so hook-based consumers see a
// real ledger instead of the noop. Callers are still responsible for
// propagating the new ledger to plugins that captured their own reference
// during OnInit — see Engine.rebindLedgerOnPlugins.
func (e *Engine) SetLedgerEngine(l *xledger.Ledger) {
	e.ledgerEng = l
	if l != nil {
		e.ledger = ledgeradapter.New(l)
	}
}

// ledgerRebindable is the optional interface implemented by plugins that
// captured a ledger reference during OnInit and need to be notified when
// the engine's ledger is rebound after construction.
type ledgerRebindable interface {
	SetLedger(*xledger.Ledger)
	SetLedgerStore(ledgerstore.Store)
}

// RebindLedgerOnPlugins pushes the engine's current ledger engine and store
// into any plugin that implements SetLedger / SetLedgerStore. Safe to call
// multiple times; no-op when the engine has no ledger.
func (e *Engine) RebindLedgerOnPlugins() {
	if e.ledgerEng == nil || e.plugins == nil {
		return
	}
	store := e.ledgerEng.Store()
	for _, p := range e.plugins.Plugins() {
		if rb, ok := p.(ledgerRebindable); ok {
			rb.SetLedger(e.ledgerEng)
			rb.SetLedgerStore(store)
		}
	}
}

// MetricsCollector returns the metrics collector bridge (may be nil).
func (e *Engine) MetricsCollector() bridge.MetricsCollector { return e.metrics }

// CeremonyStore returns the ceremony state store for short-lived auth
// ceremony sessions. Returns an in-memory store if none was configured.
func (e *Engine) CeremonyStore() ceremony.Store {
	if e.ceremonyStore != nil {
		return e.ceremonyStore
	}
	return ceremony.NewMemory()
}

// RateLimiter returns the rate limiter (may be nil).
func (e *Engine) RateLimiter() ratelimit.Limiter { return e.rateLimiter }

// Lockout returns the account lockout tracker (may be nil).
func (e *Engine) Lockout() lockout.Tracker { return e.lockout }

// PasswordHistory returns the password history store (may be nil).
func (e *Engine) PasswordHistory() account.PasswordHistoryStore { return e.passwordHistory }

// SecurityEvents returns the security event store (may be nil).
func (e *Engine) SecurityEvents() securityevent.Store { return e.securityEvents }

// Warden returns the first-class authorization engine (may be nil).
func (e *Engine) Warden() *warden.Engine { return e.wardenEng }

// Keysmith returns the first-class key management engine (may be nil).
func (e *Engine) Keysmith() *keysmith.Engine { return e.keysmithEng }

// Settings returns the dynamic settings manager (may be nil).
func (e *Engine) Settings() *settings.Manager { return e.settingsMgr }

// DB returns the database handle so plugins can create their own persistent
// stores. Returns nil if not set.
func (e *Engine) DB() *grove.DB { return e.db }

// ──────────────────────────────────────────────────
// Client configuration types
// ──────────────────────────────────────────────────

// ClientConfigResponse is the public client-facing configuration returned
// by the /client-config endpoint. It tells the frontend SDK which auth
// methods are enabled, available providers, branding, etc.
type ClientConfigResponse struct {
	Version             string                         `json:"version"`
	AppID               string                         `json:"app_id"`
	SignupEnabled       bool                           `json:"signup_enabled"`
	Branding            *ClientConfigBranding          `json:"branding,omitempty"`
	Password            *ClientConfigToggle            `json:"password,omitempty"`
	Social              *ClientConfigSocial            `json:"social,omitempty"`
	Passkey             *ClientConfigToggle            `json:"passkey,omitempty"`
	MFA                 *ClientConfigMFA               `json:"mfa,omitempty"`
	MagicLink           *ClientConfigToggle            `json:"magiclink,omitempty"`
	SSO                 *ClientConfigSSO               `json:"sso,omitempty"`
	EmailVerification   *ClientConfigEmailVerification `json:"email_verification,omitempty"`
	DeviceAuthorization *ClientConfigToggle            `json:"device_authorization,omitempty"`
	Waitlist            *ClientConfigToggle            `json:"waitlist,omitempty"`
	SupportedPlugins    []string                       `json:"supported_plugins"`
	SignupFields        []ClientConfigSignupField      `json:"signup_fields,omitempty"`
}

// ClientConfigEmailVerification represents email verification settings.
type ClientConfigEmailVerification struct {
	Enabled  bool `json:"enabled"`
	Required bool `json:"required"`
}

// ClientConfigBranding holds app branding information.
type ClientConfigBranding struct {
	AppName string `json:"app_name,omitempty"`
	LogoURL string `json:"logo_url,omitempty"`
}

// ClientConfigToggle represents a simple enabled/disabled auth method.
type ClientConfigToggle struct {
	Enabled bool `json:"enabled"`
}

// ClientConfigSocial represents the social auth configuration.
type ClientConfigSocial struct {
	Enabled   bool                         `json:"enabled"`
	Providers []ClientConfigSocialProvider `json:"providers"`
}

// ClientConfigSocialProvider represents a social login provider.
type ClientConfigSocialProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ClientConfigMFA represents the MFA configuration.
type ClientConfigMFA struct {
	Enabled bool     `json:"enabled"`
	Methods []string `json:"methods"`
}

// ClientConfigSSO represents the SSO configuration.
type ClientConfigSSO struct {
	Enabled     bool                        `json:"enabled"`
	Connections []ClientConfigSSOConnection `json:"connections"`
}

// ClientConfigSSOConnection represents an SSO connection/provider.
type ClientConfigSSOConnection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ClientConfigSignupField describes a custom signup form field.
type ClientConfigSignupField struct {
	Key         string                       `json:"key"`
	Label       string                       `json:"label"`
	Type        string                       `json:"type"`
	Placeholder string                       `json:"placeholder,omitempty"`
	Description string                       `json:"description,omitempty"`
	Options     []ClientConfigSelectOption   `json:"options,omitempty"`
	Default     string                       `json:"default,omitempty"`
	Validation  *ClientConfigFieldValidation `json:"validation,omitempty"`
	Order       int                          `json:"order"`
}

// ClientConfigSelectOption represents a select/radio option.
type ClientConfigSelectOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ClientConfigFieldValidation defines validation rules for a signup field.
type ClientConfigFieldValidation struct {
	Required bool   `json:"required,omitempty"`
	MinLen   int    `json:"min_len,omitempty"`
	MaxLen   int    `json:"max_len,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	Min      *int   `json:"min,omitempty"`
	Max      *int   `json:"max,omitempty"`
}

// socialProviderLister is implemented by the social plugin to expose providers.
type socialProviderLister interface {
	Providers() []string
}

// ssoConnectionLister is implemented by the SSO plugin to expose connections.
type ssoConnectionLister interface {
	Connections() []string
}

// providerDisplayNames maps lowercase provider IDs to proper display names
// for the client-config response. The frontend renders these as button labels.
var providerDisplayNames = map[string]string{
	"google": "Google", "github": "GitHub", "apple": "Apple",
	"microsoft": "Microsoft", "facebook": "Facebook", "twitter": "Twitter",
	"discord": "Discord", "slack": "Slack", "linkedin": "LinkedIn",
	"gitlab": "GitLab", "bitbucket": "Bitbucket", "spotify": "Spotify",
	"twitch": "Twitch", "dropbox": "Dropbox", "amazon": "Amazon",
	"yahoo": "Yahoo", "line": "Line", "instagram": "Instagram",
	"pinterest": "Pinterest", "patreon": "Patreon", "strava": "Strava",
	"zoom": "Zoom", "okta": "Okta", "azure-ad": "Azure AD",
	"auth0": "Auth0", "onelogin": "OneLogin", "ping": "Ping Identity",
}

func providerDisplayName(providerID string) string {
	if name, ok := providerDisplayNames[providerID]; ok {
		return name
	}
	if providerID == "" {
		return providerID
	}
	return strings.ToUpper(providerID[:1]) + providerID[1:]
}

// ClientConfig returns the merged client-facing configuration for an app.
// This aggregates plugin-level defaults with per-app overrides.
func (e *Engine) ClientConfig(ctx context.Context, appID id.AppID) *ClientConfigResponse {
	resp := &ClientConfigResponse{
		Version: "1",
		AppID:   appID.String(),
	}

	// Load app record for branding defaults.
	if a, err := e.store.GetApp(ctx, appID); err == nil {
		resp.Branding = &ClientConfigBranding{
			AppName: a.Name,
			LogoURL: a.Logo,
		}
	}

	// Detect enabled auth methods from registered plugins.
	allPlugins := e.plugins.Plugins()
	var (
		passwordEnabled  bool
		socialEnabled    bool
		passkeyEnabled   bool
		mfaEnabled       bool
		magicLinkEnabled bool
		ssoEnabled       bool
		waitlistEnabled  bool
		socialProviders  []ClientConfigSocialProvider
		ssoConnections   []ClientConfigSSOConnection
		mfaMethods       []string
		pluginNames      = make([]string, 0, len(allPlugins))
	)

	for _, p := range allPlugins {
		name := p.Name()
		pluginNames = append(pluginNames, name)

		switch name {
		case "password":
			passwordEnabled = true
		case "social":
			socialEnabled = true
			if lister, ok := p.(socialProviderLister); ok {
				for _, prov := range lister.Providers() {
					socialProviders = append(socialProviders, ClientConfigSocialProvider{
						ID:   prov,
						Name: providerDisplayName(prov),
					})
				}
			}
		case "passkey":
			passkeyEnabled = true
		case "mfa":
			mfaEnabled = true
			mfaMethods = []string{"totp"}
		case "magiclink":
			magicLinkEnabled = true
		case "sso":
			ssoEnabled = true
			if lister, ok := p.(ssoConnectionLister); ok {
				for _, name := range lister.Connections() {
					ssoConnections = append(ssoConnections, ClientConfigSSOConnection{
						ID:   name,
						Name: providerDisplayName(name),
					})
				}
			}
		case "waitlist":
			waitlistEnabled = true
		}
	}

	// Load per-app overrides and apply on top of plugin defaults.
	if overrides, err := e.store.GetAppClientConfig(ctx, appID); err == nil {
		applyClientConfigOverrides(resp, overrides,
			&passwordEnabled, &socialEnabled, &passkeyEnabled,
			&mfaEnabled, &magicLinkEnabled, &ssoEnabled,
			&waitlistEnabled,
			&socialProviders, &ssoConnections, &mfaMethods,
		)
	}

	// Signup is enabled by default unless explicitly disabled.
	resp.SignupEnabled = true

	// Build response sections.
	resp.Password = &ClientConfigToggle{Enabled: passwordEnabled}
	resp.Passkey = &ClientConfigToggle{Enabled: passkeyEnabled}
	resp.MagicLink = &ClientConfigToggle{Enabled: magicLinkEnabled}

	if socialProviders == nil {
		socialProviders = []ClientConfigSocialProvider{}
	}
	resp.Social = &ClientConfigSocial{
		Enabled:   socialEnabled,
		Providers: socialProviders,
	}

	if mfaMethods == nil {
		mfaMethods = []string{}
	}
	resp.MFA = &ClientConfigMFA{
		Enabled: mfaEnabled,
		Methods: mfaMethods,
	}

	if ssoConnections == nil {
		ssoConnections = []ClientConfigSSOConnection{}
	}
	resp.SSO = &ClientConfigSSO{
		Enabled:     ssoEnabled,
		Connections: ssoConnections,
	}

	resp.Waitlist = &ClientConfigToggle{Enabled: waitlistEnabled}

	if pluginNames == nil {
		pluginNames = []string{}
	}
	resp.SupportedPlugins = pluginNames

	// Load active signup form config and expose fields.
	if fc, err := e.store.GetFormConfig(ctx, appID, formconfig.FormTypeSignup); err == nil && fc != nil && fc.Active && len(fc.Fields) > 0 {
		fields := make([]ClientConfigSignupField, 0, len(fc.Fields))
		for _, f := range fc.Fields {
			sf := ClientConfigSignupField{
				Key:         f.Key,
				Label:       f.Label,
				Type:        string(f.Type),
				Placeholder: f.Placeholder,
				Description: f.Description,
				Default:     f.Default,
				Order:       f.Order,
			}
			if len(f.Options) > 0 {
				opts := make([]ClientConfigSelectOption, 0, len(f.Options))
				for _, o := range f.Options {
					opts = append(opts, ClientConfigSelectOption{Label: o.Label, Value: o.Value})
				}
				sf.Options = opts
			}
			if f.Validation != (formconfig.Validation{}) {
				sf.Validation = &ClientConfigFieldValidation{
					Required: f.Validation.Required,
					MinLen:   f.Validation.MinLen,
					MaxLen:   f.Validation.MaxLen,
					Pattern:  f.Validation.Pattern,
					Min:      f.Validation.Min,
					Max:      f.Validation.Max,
				}
			}
			fields = append(fields, sf)
		}
		resp.SignupFields = fields
	}

	// Email verification: enabled when password plugin is registered,
	// required based on environment settings (default: required in production).
	if passwordEnabled {
		emailVerifRequired := true
		if env, _ := e.GetDefaultEnvironment(ctx, appID); env != nil && env.Settings != nil { //nolint:errcheck // best-effort env lookup
			if env.Settings.SkipEmailVerification != nil && *env.Settings.SkipEmailVerification {
				emailVerifRequired = false
			}
		}
		resp.EmailVerification = &ClientConfigEmailVerification{
			Enabled:  true,
			Required: emailVerifRequired,
		}
	}

	// Device authorization: enabled when oauth2provider plugin is registered.
	for _, name := range pluginNames {
		if name == "oauth2provider" {
			resp.DeviceAuthorization = &ClientConfigToggle{Enabled: true}
			break
		}
	}

	return resp
}

// applyClientConfigOverrides applies per-app overrides to the client config.
func applyClientConfigOverrides(
	resp *ClientConfigResponse,
	cfg *appclientconfig.Config,
	passwordEnabled, socialEnabled, passkeyEnabled *bool,
	mfaEnabled, magicLinkEnabled, ssoEnabled *bool,
	waitlistEnabled *bool,
	socialProviders *[]ClientConfigSocialProvider,
	_ *[]ClientConfigSSOConnection,
	mfaMethods *[]string,
) {
	if cfg.SignupEnabled != nil {
		resp.SignupEnabled = *cfg.SignupEnabled
	}
	if cfg.PasswordEnabled != nil {
		*passwordEnabled = *cfg.PasswordEnabled
	}
	if cfg.SocialEnabled != nil {
		*socialEnabled = *cfg.SocialEnabled
	}
	if cfg.PasskeyEnabled != nil {
		*passkeyEnabled = *cfg.PasskeyEnabled
	}
	if cfg.MFAEnabled != nil {
		*mfaEnabled = *cfg.MFAEnabled
	}
	if cfg.MagicLinkEnabled != nil {
		*magicLinkEnabled = *cfg.MagicLinkEnabled
	}
	if cfg.SSOEnabled != nil {
		*ssoEnabled = *cfg.SSOEnabled
	}
	if cfg.WaitlistEnabled != nil {
		*waitlistEnabled = *cfg.WaitlistEnabled
	}

	// Filter social providers to per-app whitelist.
	if len(cfg.SocialProviders) > 0 {
		allowed := make(map[string]bool, len(cfg.SocialProviders))
		for _, name := range cfg.SocialProviders {
			allowed[name] = true
		}
		filtered := make([]ClientConfigSocialProvider, 0, len(cfg.SocialProviders))
		for _, p := range *socialProviders {
			if allowed[p.ID] {
				filtered = append(filtered, p)
			}
		}
		*socialProviders = filtered
	}

	// Filter MFA methods to per-app whitelist.
	if len(cfg.MFAMethods) > 0 {
		*mfaMethods = cfg.MFAMethods
	}

	// Branding overrides.
	if cfg.AppName != nil || cfg.LogoURL != nil {
		if resp.Branding == nil {
			resp.Branding = &ClientConfigBranding{}
		}
		if cfg.AppName != nil {
			resp.Branding.AppName = *cfg.AppName
		}
		if cfg.LogoURL != nil {
			resp.Branding.LogoURL = *cfg.LogoURL
		}
	}
}

// APIKeyStore returns the API key store. When Keysmith is available,
// operations delegate to Keysmith's full key management engine (gaining
// rate limiting, policy enforcement, key rotation, scope management, and
// usage tracking). Otherwise, it returns the composite store's API key methods.
func (e *Engine) APIKeyStore() apikey.Store {
	if e.keysmithEng != nil {
		return apikey.NewKeymithStore(e.keysmithEng)
	}
	return e.store.(apikey.Store) //nolint:errcheck // type assertion
}

// ResolveAppByPublicKey resolves a publishable key (pk_live_...) to an app.
// It first checks the apps table publishable_key column (fast, indexed), then
// falls back to searching Keysmith API key metadata for keys generated via
// the API key plugin.
func (e *Engine) ResolveAppByPublicKey(ctx context.Context, publicKey string) (*app.App, error) {
	// Fast path: apps table publishable_key column.
	if a, err := e.store.GetAppByPublishableKey(ctx, publicKey); err == nil {
		return a, nil
	}

	// Fallback: search Keysmith metadata.
	if e.keysmithEng != nil {
		ks := apikey.NewKeymithStore(e.keysmithEng)
		ak, err := ks.FindByPublicKey(ctx, publicKey)
		if err != nil {
			return nil, fmt.Errorf("resolve publishable key: %w", err)
		}
		return e.store.GetApp(ctx, ak.AppID)
	}

	return nil, errors.New("publishable key not found")
}

// ListUserRoleSlugs returns the slugs of all roles assigned to a user.
// This satisfies the middleware.RoleChecker interface (legacy; prefer PermissionChecker).
func (e *Engine) ListUserRoleSlugs(ctx context.Context, userID id.UserID) ([]string, error) {
	var roles []*rbac.Role
	var err error
	if appID := e.PlatformAppID(); !appID.IsNil() {
		roles, err = e.rbacStore().ListUserRolesForApp(ctx, appID.String(), userID.String())
	} else {
		roles, err = e.rbacStore().ListUserRoles(ctx, userID.String())
	}
	if err != nil {
		return nil, err
	}
	slugs := make([]string, len(roles))
	for i, r := range roles {
		slugs[i] = r.Slug
	}
	return slugs, nil
}

// TokenFormatForApp returns the token format for a specific app.
// Resolution order: per-app JWT format → default format → opaque.
func (e *Engine) TokenFormatForApp(appID string) tokenformat.Format {
	if e.jwtFormats != nil {
		if jwtFmt, ok := e.jwtFormats[appID]; ok {
			return jwtFmt
		}
	}
	if e.defaultTokenFormat != nil {
		return e.defaultTokenFormat
	}
	return tokenformat.Opaque{}
}

// JWTFormats returns all registered JWT formats (for JWKS endpoint).
func (e *Engine) JWTFormats() map[string]*tokenformat.JWT { return e.jwtFormats }

// ValidateJWT tries all registered JWT formats to validate a token.
// This implements middleware.JWTValidator.
func (e *Engine) ValidateJWT(token string) (*tokenformat.TokenClaims, error) {
	// Try each registered JWT format.
	for _, jwtFmt := range e.jwtFormats {
		claims, err := jwtFmt.ValidateAccessToken(token)
		if err == nil {
			return claims, nil
		}
	}
	// Try the default format if it's JWT.
	if e.defaultTokenFormat != nil && e.defaultTokenFormat.Name() == "jwt" {
		return e.defaultTokenFormat.ValidateAccessToken(token)
	}
	return nil, tokenformat.ErrInvalidToken
}

// HasJWT returns true if any JWT format is configured.
func (e *Engine) HasJWT() bool {
	if len(e.jwtFormats) > 0 {
		return true
	}
	return e.defaultTokenFormat != nil && e.defaultTokenFormat.Name() == "jwt"
}

// DefaultTokenFormat returns the default token format (may be nil).
func (e *Engine) DefaultTokenFormat() tokenformat.Format { return e.defaultTokenFormat }

// ──────────────────────────────────────────────────
// Account Linking
// ──────────────────────────────────────────────────

// ListAuthMethods aggregates linked auth methods from all plugins that
// implement AuthMethodContributor. Returns an empty slice if no plugins
// contribute methods.
func (e *Engine) ListAuthMethods(ctx context.Context, userID id.UserID) ([]*plugin.AuthMethod, error) {
	var methods []*plugin.AuthMethod
	for _, p := range e.plugins.Plugins() {
		if contributor, ok := p.(plugin.AuthMethodContributor); ok {
			pm, err := contributor.ListUserAuthMethods(ctx, userID)
			if err != nil {
				e.logger.Warn("authsome: list auth methods failed",
					log.String("plugin", p.Name()),
					log.String("error", err.Error()),
				)
				continue
			}
			methods = append(methods, pm...)
		}
	}
	return methods, nil
}

// UnlinkAuthMethod removes an auth method from a user. It returns an error
// if the method is the last one (cannot unlink all methods) or the plugin
// does not support unlinking.
func (e *Engine) UnlinkAuthMethod(ctx context.Context, userID id.UserID, provider string) error {
	// Safety: count total auth methods to ensure at least one remains.
	allMethods, err := e.ListAuthMethods(ctx, userID)
	if err != nil {
		return fmt.Errorf("authsome: list auth methods: %w", err)
	}
	if len(allMethods) <= 1 {
		return errors.New("authsome: cannot unlink last authentication method")
	}

	// Find the plugin that owns this provider.
	for _, p := range e.plugins.Plugins() {
		if unlinker, ok := p.(plugin.AuthMethodUnlinker); ok {
			if unlinker.CanUnlink(ctx, userID, provider) {
				return unlinker.UnlinkAuthMethod(ctx, userID, provider)
			}
		}
	}

	return fmt.Errorf("authsome: no plugin can unlink provider %q", provider)
}

// ──────────────────────────────────────────────────
// Metrics
// ──────────────────────────────────────────────────

// Metrics returns observable metrics for the engine.
type Metrics struct {
	PluginsLoaded int `json:"plugins_loaded"`
	Strategies    int `json:"strategies"`
}

// Metrics returns the current engine metrics.
func (e *Engine) Metrics() Metrics {
	return Metrics{
		PluginsLoaded: len(e.plugins.Plugins()),
		Strategies:    len(e.strategies.Strategies()),
	}
}
