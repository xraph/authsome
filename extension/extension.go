// Package extension adapts AuthSome as a Forge extension.
package extension

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"
	dashboard "github.com/xraph/forge/extensions/dashboard"
	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contributor"
	"github.com/xraph/forge/extensions/dashboard/ui/shell"
	"github.com/xraph/vessel"

	fuibridge "github.com/xraph/forgeui/bridge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/bridge/chronicleadapter"
	"github.com/xraph/authsome/bridge/dispatchadapter"
	"github.com/xraph/authsome/bridge/heraldadapter"
	"github.com/xraph/authsome/bridge/maileradapter"
	"github.com/xraph/authsome/bridge/relayadapter"
	authdash "github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/lockout"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/ratelimit"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	mongostore "github.com/xraph/authsome/store/mongo"
	pgstore "github.com/xraph/authsome/store/postgres"
	sqlitestore "github.com/xraph/authsome/store/sqlite"

	"github.com/xraph/grove"

	"github.com/xraph/chronicle"
	dispatchengine "github.com/xraph/dispatch/engine"
	"github.com/xraph/herald"
	"github.com/xraph/keysmith"
	"github.com/xraph/ledger"
	"github.com/xraph/relay"
	"github.com/xraph/vault"
	"github.com/xraph/warden"
)

// ExtensionName is the name registered with Forge.
const ExtensionName = "authsome"

// ExtensionDescription is the human-readable description.
const ExtensionDescription = "Pluggable authentication engine for identity, sessions, and multi-tenancy"

// ExtensionVersion is the semantic version.
const ExtensionVersion = "0.5.0"

// Ensure Extension implements forge.Extension, forge.MiddlewareExtension,
// dashboard.DashboardAware, dashboard.DashboardAuthAware, and
// dashboard.DashboardFooterContributor at compile time.
var (
	_ forge.Extension                      = (*Extension)(nil)
	_ forge.MiddlewareExtension            = (*Extension)(nil)
	_ dashboard.DashboardAware             = (*Extension)(nil)
	_ dashboard.DashboardAuthAware         = (*Extension)(nil)
	_ dashboard.DashboardFooterContributor = (*Extension)(nil)
	_ dashboard.BridgeAware                = (*Extension)(nil)
)

// Extension adapts AuthSome as a Forge extension.
type Extension struct {
	*forge.BaseExtension

	config     Config
	engine     *authsome.Engine
	apiHandler *api.API
	logger     log.Logger
	opts       []authsome.Option
	plugins    []plugin.Plugin
	useGrove   bool
	clientMode bool               // true when operating as a remote client
	client     *authclient.Client // non-nil in client mode
}

// New creates an AuthSome Forge extension with the given options.
func New(opts ...ExtOption) *Extension {
	e := &Extension{
		BaseExtension: forge.NewBaseExtension(ExtensionName, ExtensionVersion, ExtensionDescription),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Engine returns the underlying authsome engine (nil until Register is called).
func (e *Extension) Engine() *authsome.Engine { return e.engine }

// Register implements forge.Extension. It loads configuration, auto-discovers
// optional forgery bridges from the DI container, builds the engine, and
// registers the engine in the container for other extensions to use.
func (e *Extension) Register(fapp forge.App) error {
	if err := e.BaseExtension.Register(fapp); err != nil {
		return err
	}

	if err := e.loadConfiguration(); err != nil {
		return err
	}

	// Client mode: delegate all auth to a remote authsome server.
	// No local engine, no database, no migrations.
	if e.config.ClientMode {
		return e.initClientMode(fapp)
	}

	if err := e.init(fapp); err != nil {
		return err
	}

	// Register the engine in the DI container for other extensions.
	if err := vessel.Provide(fapp.Container(), func() (*authsome.Engine, error) {
		return e.engine, nil
	}); err != nil {
		return fmt.Errorf("authsome: register engine in container: %w", err)
	}

	return nil
}

// initClientMode sets up the extension as a remote client. No engine or
// database connection is created. An authclient.Client is registered in
// the DI container for downstream extensions to use.
func (e *Extension) initClientMode(fapp forge.App) error {
	if e.config.PortalURL == "" {
		return fmt.Errorf("authsome: client mode requires portal_url")
	}

	logger := e.logger
	if logger == nil {
		logger = e.Logger()
	}
	if logger == nil {
		logger = log.NewNoopLogger()
	}

	opts := []authclient.Option{
		authclient.WithSessionCookies(),
	}
	if e.config.ServiceAPIKey != "" {
		opts = append(opts, authclient.WithAPIKey(e.config.ServiceAPIKey))
	}

	e.client = authclient.NewClient(e.config.PortalURL, opts...)
	e.clientMode = true

	// Register the client in the DI container so workspace and other
	// extensions can inject it for org/membership operations.
	if err := vessel.Provide(fapp.Container(), func() (*authclient.Client, error) {
		return e.client, nil
	}); err != nil {
		return fmt.Errorf("authsome: register client in container: %w", err)
	}

	logger.Info("authsome: client mode enabled",
		log.String("portal_url", e.config.PortalURL),
	)

	return nil
}

// init builds the engine with auto-discovered dependencies.
func (e *Extension) init(fapp forge.App) error {
	logger := e.logger
	if logger == nil {
		logger = e.Logger()
	}
	if logger == nil {
		logger = log.NewNoopLogger()
	}

	opts := make([]authsome.Option, 0, len(e.opts)+10)
	opts = append(opts, e.opts...)
	opts = append(opts, authsome.WithLogger(logger)) //nolint:gocritic // can't combine with variadic spread above

	// Map extension config to engine config.
	opts = append(opts, authsome.WithConfig(e.buildEngineConfig()))

	// Register plugins
	for _, p := range e.plugins {
		opts = append(opts, authsome.WithPlugin(p))
	}

	// ── Resolve store from grove DI ──
	var driverName string
	if e.useGrove {
		// Explicit grove database configured -- resolve named or default DB.
		groveDB, err := e.resolveGroveDB(fapp)
		if err != nil {
			return fmt.Errorf("authsome: %w", err)
		}
		s, err := e.buildStoreFromGroveDB(groveDB)
		if err != nil {
			return err
		}
		driverName = groveDB.Driver().Name()
		opts = append(opts, authsome.WithStore(s), authsome.WithDriverName(driverName), authsome.WithDB(groveDB))
		e.Logger().Info("authsome: resolved grove.DB from container (driver=" + driverName + ")")
	} else if db, err := vessel.Inject[*grove.DB](fapp.Container()); err == nil {
		// Backward-compatible silent auto-discovery of default grove.DB.
		var s store.Store
		driverName = db.Driver().Name()
		switch driverName {
		case "pg":
			s = pgstore.New(db)
		case "sqlite":
			s = sqlitestore.New(db)
		case "mongo":
			s = mongostore.New(db)
		default:
			return fmt.Errorf("authsome: unsupported grove driver %q", driverName)
		}
		opts = append(opts, authsome.WithStore(s), authsome.WithDriverName(driverName), authsome.WithDB(db))
		e.Logger().Info("authsome: auto-discovered grove.DB from container (driver=" + driverName + ")")
	}

	// ── Auto-discover Chronicle (optional) ──
	if emitter, err := vessel.Inject[chronicle.Emitter](fapp.Container()); err == nil {
		opts = append(opts, authsome.WithChronicle(chronicleadapter.New(emitter)))
		e.Logger().Info("authsome: auto-discovered chronicle emitter")
	} else {
		// Fallback to slog audit stub
		opts = append(opts, authsome.WithChronicle(bridge.NewSlogChronicle(logger)))
	}

	// ── Auto-discover Warden (required authorization engine) ──
	wardenEng, err := vessel.Inject[*warden.Engine](fapp.Container())
	if err != nil {
		return fmt.Errorf("authsome: warden extension is required but not found in DI container; register warden before authsome: %w", err)
	}
	opts = append(opts, authsome.WithWarden(wardenEng))
	e.Logger().Info("authsome: auto-discovered warden engine (required)")

	// ── Auto-discover Keysmith (first-class key management engine) ──
	if ksEng, ksErr := vessel.Inject[*keysmith.Engine](fapp.Container()); ksErr == nil {
		opts = append(opts, authsome.WithKeysmith(ksEng))
		e.Logger().Info("authsome: auto-discovered keysmith engine (first-class)")
	}

	// ── Auto-discover Relay (optional) ──
	if relayInst, relayErr := vessel.Inject[*relay.Relay](fapp.Container()); relayErr == nil {
		opts = append(opts, authsome.WithEventRelay(relayadapter.New(relayInst)))
		e.Logger().Info("authsome: auto-discovered relay")
	} else {
		opts = append(opts, authsome.WithEventRelay(bridge.NewNoopRelay(logger)))
	}

	// ── Auto-discover Vault (optional) ──
	if _, vaultErr := vessel.Inject[*vault.Vault](fapp.Container()); vaultErr == nil {
		// Vault extension detected but services are not yet individually
		// accessible. Use NoopVault until vault exposes sub-services.
		opts = append(opts, authsome.WithVault(bridge.NewNoopVault()))
		e.Logger().Info("authsome: auto-discovered vault (noop bridge until services available)")
	} else {
		opts = append(opts, authsome.WithVault(bridge.NewNoopVault()))
	}

	// ── Auto-discover Dispatch (optional) ──
	if dispatchEng, dispatchErr := vessel.Inject[*dispatchengine.Engine](fapp.Container()); dispatchErr == nil {
		opts = append(opts, authsome.WithDispatcher(dispatchadapter.New(dispatchEng)))
		e.Logger().Info("authsome: auto-discovered dispatch engine")
	} else {
		opts = append(opts, authsome.WithDispatcher(bridge.NewNoopDispatcher()))
	}

	// ── Auto-discover Ledger (optional) ──
	if ledgerEng, ledgerErr := vessel.Inject[*ledger.Ledger](fapp.Container()); ledgerErr == nil {
		opts = append(opts, authsome.WithLedgerEngine(ledgerEng))
		e.Logger().Info("authsome: auto-discovered ledger")
	} else {
		opts = append(opts, authsome.WithLedger(bridge.NewNoopLedger()))
	}

	// ── Auto-discover Herald (optional) ──
	if heraldEng, heraldErr := vessel.Inject[*herald.Herald](fapp.Container()); heraldErr == nil {
		opts = append(opts, authsome.WithHerald(heraldadapter.New(heraldEng)))
		e.Logger().Info("authsome: auto-discovered herald notification engine")
	} else {
		opts = append(opts, authsome.WithHerald(bridge.NewNoopHerald(logger)))
	}

	// ── Auto-configure mailer from config ──
	switch e.config.Mailer.Provider {
	case "resend":
		opts = append(opts, authsome.WithMailer(
			maileradapter.NewResendMailer(e.config.Mailer.Resend.APIKey, e.config.Mailer.Resend.From),
		))
		e.Logger().Info("authsome: mailer configured (resend)")
	case "smtp":
		smtpCfg := e.config.Mailer.SMTP
		smtpOpts := []maileradapter.SMTPOption{}
		if smtpCfg.TLS {
			smtpOpts = append(smtpOpts, maileradapter.WithSMTPTLS(true))
		}
		opts = append(opts, authsome.WithMailer(
			maileradapter.NewSMTPMailer(smtpCfg.Host, smtpCfg.Port, smtpCfg.Username, smtpCfg.Password, smtpCfg.From, smtpOpts...),
		))
		e.Logger().Info("authsome: mailer configured (smtp)")
	default:
		opts = append(opts, authsome.WithMailer(bridge.NewNoopMailer(logger)))
	}

	// ── Auto-configure rate limiter (in-memory fallback) ──
	if e.config.RateLimit.Enabled {
		opts = append(opts, authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()))
		e.Logger().Info("authsome: rate limiting enabled (in-memory)")
	}

	// ── Auto-configure account lockout (in-memory fallback) ──
	if e.config.Lockout.Enabled {
		opts = append(opts, authsome.WithLockoutTracker(
			lockout.NewMemoryTracker(
				lockout.WithMaxAttempts(e.config.Lockout.MaxAttempts),
				lockout.WithLockoutDuration(e.config.Lockout.LockoutDuration()),
				lockout.WithResetAfter(e.config.Lockout.ResetAfter()),
			),
		))
		e.Logger().Info("authsome: account lockout enabled (in-memory)")
	}

	// ── Per-app session configuration from YAML ──
	for appIDStr, appCfg := range e.config.Apps {
		appID, parseErr := id.ParseAppID(appIDStr)
		if parseErr != nil {
			e.Logger().Warn("authsome: invalid app ID in per-app config, skipping",
				forge.F("app_id", appIDStr),
				forge.F("error", parseErr.Error()),
			)
			continue
		}
		cfg := &appsessionconfig.Config{
			ID:          id.NewAppSessionConfigID(),
			AppID:       appID,
			TokenFormat: appCfg.Session.TokenFormat,
		}
		if appCfg.Session.TokenTTL != 0 {
			secs := int(appCfg.Session.TokenTTL.Seconds())
			cfg.TokenTTLSeconds = &secs
		}
		if appCfg.Session.RefreshTokenTTL != 0 {
			secs := int(appCfg.Session.RefreshTokenTTL.Seconds())
			cfg.RefreshTokenTTLSeconds = &secs
		}
		cfg.MaxActiveSessions = appCfg.Session.MaxActiveSessions
		cfg.RotateRefreshToken = appCfg.Session.RotateRefreshToken
		cfg.BindToIP = appCfg.Session.BindToIP
		cfg.BindToDevice = appCfg.Session.BindToDevice
		opts = append(opts, authsome.WithAppSessionConfig(cfg))
	}

	// ── Bootstrap configuration ──
	// Bootstrap is enabled by default when using the Forge extension,
	// unless explicitly disabled via config.
	if e.config.Bootstrap.Enabled == nil || *e.config.Bootstrap.Enabled {
		bootstrapOpts := e.buildBootstrapOptions()
		opts = append(opts, authsome.WithBootstrap(bootstrapOpts...))
		e.Logger().Info("authsome: bootstrap enabled")
	}

	eng, err := authsome.NewEngine(opts...)
	if err != nil {
		return fmt.Errorf("authsome: create engine: %w", err)
	}
	e.engine = eng

	// Run migrations eagerly so that tables exist before any route handler
	// or dashboard auth page can access the database. This prevents
	// "relation does not exist" errors in PostgreSQL when the dashboard
	// loads before Start() completes. Idempotent — Start() will skip if
	// already applied.
	if err := eng.EnsureMigrated(context.Background()); err != nil {
		return fmt.Errorf("authsome: eager migration: %w", err)
	}

	// Create API handler and register routes on the Forge router.
	if !e.config.DisableRoutes {
		basePath := e.config.BasePath
		if basePath == "" {
			basePath = "/authsome"
		}

		// Pass the forge auth registry to the engine so it can register
		// the "session" auth provider and expose the registry to plugins.
		if container := fapp.Container(); container != nil {
			func() {
				defer func() { recover() }() //nolint:errcheck // forge.Must panics if not found
				registry := forge.Must[auth.Registry](container, auth.RegistryKey)
				eng.SetAuthRegistry(registry)
			}()
		}

		// Initialize plugins BEFORE route registration so that OnInit
		// runs first. Plugins may set up middleware, stores, or other
		// dependencies that RegisterRoutes relies on.
		eng.InitPlugins(context.Background())

		e.apiHandler = api.New(eng, fapp.Router())

		router := fapp.Router()
		if router != nil {
			groupedRouter := router.Group(basePath)
			if err := e.apiHandler.RegisterRoutes(groupedRouter); err != nil {
				return fmt.Errorf("authsome: register forge routes: %w", err)
			}

			// Register plugin routes on the grouped router.
			for _, rp := range eng.Plugins().RouteProviders() {
				if err := rp.RegisterRoutes(groupedRouter); err != nil {
					return fmt.Errorf("authsome: register plugin routes (%T): %w", rp, err)
				}
			}

			// Expose the dashboard contributor over HTTP so other Forge apps
			// can consume it as a remote contributor without needing the full
			// dashboard extension installed locally.
			if err := e.registerContributorProtocol(groupedRouter); err != nil {
				return err
			}

			e.Logger().Info("authsome: registered routes on forge router")
		} else {
			e.Logger().Warn("authsome: forge router not available, API routes not registered")
		}
	}

	return nil
}

// Start begins the authsome engine and runs auto-migration if enabled.
func (e *Extension) Start(ctx context.Context) error {
	if e.clientMode {
		// Client mode: verify Portal is reachable.
		if _, err := e.client.GetHealth(ctx); err != nil {
			e.Logger().Warn("authsome: portal health check failed (may not be running yet)",
				log.String("portal_url", e.config.PortalURL),
				log.String("error", err.Error()),
			)
		}
		e.MarkStarted()
		return nil
	}
	if e.engine == nil {
		return errors.New("authsome: extension not initialized")
	}

	// Late ledger binding. If ledger wasn't resolvable at Register time
	// (DI registration order is not guaranteed across extensions), retry
	// discovery now and push the result into the engine and any plugins
	// that captured a ledger reference during OnInit. Without this,
	// plugins like subscription silently operate on a nil ledger store
	// even when a ledger extension is present in the app.
	if e.engine.LedgerEngine() == nil {
		if ledgerEng, err := vessel.Inject[*ledger.Ledger](e.App().Container()); err == nil && ledgerEng != nil {
			e.engine.SetLedgerEngine(ledgerEng)
			e.engine.RebindLedgerOnPlugins()
			e.Logger().Info("authsome: late-bound ledger engine at start")
		}
	}

	if err := e.engine.Start(ctx); err != nil {
		return err
	}
	e.MarkStarted()
	return nil
}

// Stop gracefully shuts down the authsome engine.
func (e *Extension) Stop(ctx context.Context) error {
	if e.clientMode {
		e.MarkStopped()
		return nil
	}
	if e.engine == nil {
		e.MarkStopped()
		return nil
	}
	err := e.engine.Stop(ctx)
	e.MarkStopped()
	return err
}

// Health implements forge.Extension.
func (e *Extension) Health(ctx context.Context) error {
	if e.clientMode {
		_, err := e.client.GetHealth(ctx)
		return err
	}
	if e.engine == nil {
		return errors.New("authsome: extension not initialized")
	}
	return e.engine.Store().Ping(ctx)
}

// Handler returns the HTTP handler for standalone use outside Forge.
// When used with Forge, routes are registered via the Forge router instead.
func (e *Extension) Handler() http.Handler {
	if e.apiHandler != nil {
		return e.apiHandler.Handler()
	}
	return http.NotFoundHandler()
}

// Middlewares implements forge.MiddlewareExtension. It returns the auth
// middleware, session activity extension, and auto-refresh middleware
// so Forge auto-applies them globally to all routes.
func (e *Extension) Middlewares() []forge.Middleware {
	if e.clientMode {
		// Client mode: use remote token validation via introspect.
		// No session activity or auto-refresh — those are Portal's responsibility.
		return []forge.Middleware{
			middleware.ClientAuthMiddleware(e.client, e.Logger()),
		}
	}
	if e.engine == nil {
		return nil
	}
	return []forge.Middleware{
		e.AuthMiddleware(),
		e.sessionActivityMiddleware(),
		e.autoRefreshMiddleware(),
	}
}

// AuthMiddleware returns the authentication middleware that resolves sessions
// and users from the Authorization header. When JWT is configured, JWT tokens
// are validated stateless. When strategies are registered (e.g. API key plugin),
// it uses layered auth that falls back to the strategy registry when session
// resolution fails. This is the forge.Scope producer -- it resolves the
// authenticated identity and sets AppID/OrgID on context for all downstream
// extensions.
//
// A cookie-to-header bridge is applied so that dashboard JavaScript fetch()
// requests (which send the HttpOnly auth_token cookie but no Authorization
// header) are authenticated by the same session-resolution pipeline.
//
// The middleware is built and cached by the engine during InitPlugins(),
// ensuring the extension and all plugins share the exact same middleware.
func (e *Extension) AuthMiddleware() forge.Middleware {
	return e.engine.AuthMiddleware()
}

// autoRefreshMiddleware returns a middleware that transparently refreshes
// near-expiry access tokens based on the auto-refresh settings.
func (e *Extension) autoRefreshMiddleware() forge.Middleware {
	refresher := func(ctx context.Context, refreshToken string) (*session.Session, error) {
		return e.engine.Refresh(ctx, refreshToken)
	}
	return middleware.AutoRefreshMiddleware(
		refresher,
		func(ctx context.Context) middleware.AutoRefreshConfig {
			mgr := e.engine.Settings()
			if mgr == nil {
				return middleware.AutoRefreshConfig{}
			}

			opts := settings.ResolveOpts{}
			if appID, ok := middleware.AppIDFrom(ctx); ok {
				opts.AppID = appID.String()
			}

			enabled, err := settings.Get(ctx, mgr, authsome.SettingAutoRefreshEnabled, opts)
			if err != nil || !enabled {
				return middleware.AutoRefreshConfig{}
			}

			thresholdSec, err := settings.Get(ctx, mgr, authsome.SettingAutoRefreshThresholdSeconds, opts)
			if err != nil || thresholdSec <= 0 {
				thresholdSec = 300
			}

			exposeRefresh, _ := settings.Get(ctx, mgr, authsome.SettingAutoRefreshExposeRefreshToken, opts) //nolint:errcheck // best-effort

			return middleware.AutoRefreshConfig{
				Enabled:            true,
				Threshold:          time.Duration(thresholdSec) * time.Second,
				ExposeRefreshToken: exposeRefresh,
			}
		},
		e.Logger(),
		e.cookieSetter(),
	)
}

// sessionActivityMiddleware returns a middleware that extends session expiry
// on each authenticated request (sliding session window).
func (e *Extension) sessionActivityMiddleware() forge.Middleware {
	return middleware.SessionActivityMiddleware(
		e.engine.Store().TouchSession,
		func(ctx context.Context) middleware.SessionActivityConfig {
			mgr := e.engine.Settings()
			if mgr == nil {
				return middleware.SessionActivityConfig{}
			}

			opts := settings.ResolveOpts{}
			if appID, ok := middleware.AppIDFrom(ctx); ok {
				opts.AppID = appID.String()
			}

			enabled, err := settings.Get(ctx, mgr, authsome.SettingExtendOnActivity, opts)
			if err != nil || !enabled {
				return middleware.SessionActivityConfig{}
			}

			timeoutSec, err := settings.Get(ctx, mgr, authsome.SettingInactivityTimeoutSeconds, opts)
			if err != nil || timeoutSec <= 0 {
				timeoutSec = 1800
			}

			return middleware.SessionActivityConfig{
				Enabled:           true,
				InactivityTimeout: time.Duration(timeoutSec) * time.Second,
			}
		},
		e.Logger(),
		e.cookieSetter(),
	)
}

// cookieSetter returns a CookieSetter callback that re-sets the session
// cookie using the engine's dynamic cookie configuration from settings.
func (e *Extension) cookieSetter() middleware.CookieSetter {
	return func(ctx forge.Context, token string, maxAge int) {
		mgr := e.engine.Settings()
		if mgr == nil {
			return
		}
		goCtx := ctx.Context()
		opts := settings.ResolveOpts{}
		if appID, ok := middleware.AppIDFrom(goCtx); ok {
			opts.AppID = appID.String()
		}

		name, _ := settings.Get(goCtx, mgr, authsome.SettingCookieName, opts) //nolint:errcheck // best-effort
		if name == "" {
			name = "authsome_session_token"
		}
		domain, _ := settings.Get(goCtx, mgr, authsome.SettingCookieDomain, opts) //nolint:errcheck // best-effort
		path, _ := settings.Get(goCtx, mgr, authsome.SettingCookiePath, opts)     //nolint:errcheck // best-effort
		if path == "" {
			path = "/"
		}
		secureSetting, _ := settings.Get(goCtx, mgr, authsome.SettingCookieSecure, opts) //nolint:errcheck // best-effort
		httpOnly, _ := settings.Get(goCtx, mgr, authsome.SettingCookieHTTPOnly, opts)    //nolint:errcheck // best-effort
		sameSiteStr, _ := settings.Get(goCtx, mgr, authsome.SettingCookieSameSite, opts) //nolint:errcheck // best-effort

		r := ctx.Request()
		isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		secure := secureSetting && isHTTPS

		sameSite := http.SameSiteLaxMode
		switch sameSiteStr {
		case "strict":
			sameSite = http.SameSiteStrictMode
		case "none":
			sameSite = http.SameSiteNoneMode
		}

		http.SetCookie(ctx.Response(), &http.Cookie{
			Name:     name,
			Value:    token,
			Path:     path,
			Domain:   domain,
			MaxAge:   maxAge,
			HttpOnly: httpOnly,
			Secure:   secure,
			SameSite: sameSite,
		})
	}
}

// DashboardContributor implements dashboard.DashboardAware. It returns a
// LocalContributor that renders authsome pages, widgets, and settings in the
// Forge dashboard using templ + ForgeUI.
//
// In client mode no engine is available locally, so we return a thin stub
// contributor that publishes the authsome manifest (so the icon appears in
// the app grid) and redirects any in-process render to the remote authsome
// dashboard. The full UI is rendered by the remote service.
func (e *Extension) DashboardContributor() contributor.LocalContributor {
	if e.clientMode {
		return newProxyContributor(e.config.PortalURL, e.config.ServiceAPIKey)
	}
	if e.engine == nil {
		return nil
	}
	return authdash.New(
		authdash.NewManifest(e.engine, e.plugins),
		e.engine,
		e.plugins,
	)
}

// RegisterDashboardAuth implements dashboard.DashboardAuthAware. It registers
// authsome as the dashboard's auth page provider, auth checker, and tenant
// resolver. Called automatically by the dashboard during Start() via discovery.
func (e *Extension) RegisterDashboardAuth(dashExt *dashboard.Extension) {
	basePath := dashExt.ForgeUIApp().Config().BasePath

	switch {
	case e.clientMode:
		if e.client == nil {
			e.Logger().Warn("authsome: client mode enabled but client is nil; skipping dashboard auth registration")
			return
		}
		dashExt.SetAuthPageProvider(&clientAuthPages{
			client:   e.client,
			basePath: basePath,
		})
		dashExt.SetAuthChecker(&clientAuthChecker{client: e.client})
	default:
		if e.engine == nil {
			e.Logger().Warn("authsome: engine not initialised; skipping dashboard auth registration")
			return
		}
		dashExt.SetAuthPageProvider(&authPages{engine: e.engine, basePath: basePath})
		dashExt.SetAuthChecker(&authChecker{engine: e.engine})
	}

	dashExt.SetTenantResolver(dashauth.ScopeTenantResolver{})
	dashExt.EnableAuth()
	if e.clientMode {
		e.Logger().Info("authsome: registered as dashboard auth provider (client mode)")
	} else {
		e.Logger().Info("authsome: registered as dashboard auth provider")
	}
}

// DashboardUserDropdownActions implements dashboard.DashboardFooterContributor.
// It contributes user-related actions (Profile, Security) to the sidebar footer
// user dropdown menu.
func (e *Extension) DashboardUserDropdownActions(basePath string) []shell.UserDropdownAction {
	return []shell.UserDropdownAction{
		{Label: "Profile", Icon: "user", Href: basePath + "/ext/authsome/pages/profile"},
		{Label: "Security", Icon: "shield", Href: basePath + "/ext/authsome/pages/security"},
	}
}

// RegisterDashboardBridge implements dashboard.BridgeAware. It registers
// authsome bridge functions for Go↔JS communication in the dashboard.
func (e *Extension) RegisterDashboardBridge(b *fuibridge.Bridge) error {
	return b.Register("authsome.createApp", e.bridgeCreateApp,
		fuibridge.WithDescription("Create a new application with default environments and roles"),
	)
}

// createAppInput holds parameters for the authsome.createApp bridge function.
type createAppInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Logo string `json:"logo,omitempty"`
}

// createAppOutput holds the result of a successful app creation.
type createAppOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// bridgeCreateApp handles the authsome.createApp bridge function call.
func (e *Extension) bridgeCreateApp(ctx fuibridge.Context, input createAppInput) (*createAppOutput, error) {
	if input.Name == "" || input.Slug == "" {
		return nil, fuibridge.NewError(fuibridge.ErrCodeBadRequest, "Name and slug are required")
	}

	existing, err := e.engine.GetAppBySlug(ctx.Context(), input.Slug)
	if err == nil && existing != nil {
		return nil, fuibridge.NewError(fuibridge.ErrCodeBadRequest, fmt.Sprintf("Slug %q is already in use", input.Slug))
	}

	a := &app.App{
		Name: input.Name,
		Slug: input.Slug,
		Logo: input.Logo,
	}

	if err := e.engine.CreateApp(ctx.Context(), a); err != nil {
		return nil, fuibridge.NewError(fuibridge.ErrCodeInternal, fmt.Sprintf("Failed to create app: %v", err))
	}

	return &createAppOutput{
		ID:   a.ID.String(),
		Name: a.Name,
		Slug: a.Slug,
	}, nil
}

// --- Config Loading (mirrors grove extension pattern) ---

// loadConfiguration loads config from YAML files or programmatic sources.
func (e *Extension) loadConfiguration() error {
	programmaticConfig := e.config

	// Try loading from config file.
	fileConfig, configLoaded := e.tryLoadFromConfigFile()

	if !configLoaded {
		if programmaticConfig.RequireConfig {
			return errors.New("authsome: configuration is required but not found in config files; " +
				"ensure 'extensions.authsome' or 'authsome' key exists in your config")
		}

		// Use programmatic config merged with defaults.
		e.config = e.mergeWithDefaults(programmaticConfig)
	} else {
		// Config loaded from YAML -- merge with programmatic options.
		e.config = e.mergeConfigurations(fileConfig, programmaticConfig)
	}

	// Enable grove resolution if YAML config specifies a grove database.
	if e.config.GroveDatabase != "" {
		e.useGrove = true
	}

	e.Logger().Debug("authsome: configuration loaded",
		forge.F("disable_routes", e.config.DisableRoutes),
		forge.F("disable_migrate", e.config.DisableMigrate),
		forge.F("base_path", e.config.BasePath),
		forge.F("grove_database", e.config.GroveDatabase),
		forge.F("debug", e.config.Debug),
	)

	return nil
}

// tryLoadFromConfigFile attempts to load config from YAML files.
func (e *Extension) tryLoadFromConfigFile() (Config, bool) {
	cm := e.App().Config()
	var cfg Config

	// Try "extensions.authsome" first (namespaced pattern).
	if cm.IsSet("extensions.authsome") {
		if err := cm.Bind("extensions.authsome", &cfg); err == nil {
			e.Logger().Debug("authsome: loaded config from file",
				forge.F("key", "extensions.authsome"),
			)
			return cfg, true
		}
		e.Logger().Warn("authsome: failed to bind extensions.authsome config",
			forge.F("error", "bind failed"),
		)
	}

	// Try legacy "authsome" key.
	if cm.IsSet("authsome") {
		if err := cm.Bind("authsome", &cfg); err == nil {
			e.Logger().Debug("authsome: loaded config from file",
				forge.F("key", "authsome"),
			)
			return cfg, true
		}
		e.Logger().Warn("authsome: failed to bind authsome config",
			forge.F("error", "bind failed"),
		)
	}

	return Config{}, false
}

// mergeWithDefaults fills zero-valued fields with defaults.
func (e *Extension) mergeWithDefaults(cfg Config) Config {
	defaults := DefaultConfig()

	if cfg.BasePath == "" {
		cfg.BasePath = defaults.BasePath
	}

	// Session defaults.
	if cfg.Session.TokenTTL == 0 {
		cfg.Session.TokenTTL = defaults.Session.TokenTTL
	}
	if cfg.Session.RefreshTokenTTL == 0 {
		cfg.Session.RefreshTokenTTL = defaults.Session.RefreshTokenTTL
	}

	// Password defaults.
	if cfg.Password.MinLength == 0 {
		cfg.Password.MinLength = defaults.Password.MinLength
	}
	if cfg.Password.BcryptCost == 0 {
		cfg.Password.BcryptCost = defaults.Password.BcryptCost
	}
	// Bool defaults for password: use defaults only when both are false
	// (i.e., nothing was explicitly set).
	if !cfg.Password.RequireUppercase && defaults.Password.RequireUppercase {
		cfg.Password.RequireUppercase = defaults.Password.RequireUppercase
	}
	if !cfg.Password.RequireLowercase && defaults.Password.RequireLowercase {
		cfg.Password.RequireLowercase = defaults.Password.RequireLowercase
	}
	if !cfg.Password.RequireDigit && defaults.Password.RequireDigit {
		cfg.Password.RequireDigit = defaults.Password.RequireDigit
	}

	return cfg
}

// mergeConfigurations merges YAML config with programmatic options.
// YAML config takes precedence for most fields; programmatic bool flags fill gaps.
func (e *Extension) mergeConfigurations(yamlConfig, programmaticConfig Config) Config {
	// Programmatic bool flags override when true.
	if programmaticConfig.DisableRoutes {
		yamlConfig.DisableRoutes = true
	}
	if programmaticConfig.DisableMigrate {
		yamlConfig.DisableMigrate = true
	}
	if programmaticConfig.Debug {
		yamlConfig.Debug = true
	}

	// String fields: YAML takes precedence.
	if yamlConfig.BasePath == "" && programmaticConfig.BasePath != "" {
		yamlConfig.BasePath = programmaticConfig.BasePath
	}
	if yamlConfig.GroveDatabase == "" && programmaticConfig.GroveDatabase != "" {
		yamlConfig.GroveDatabase = programmaticConfig.GroveDatabase
	}

	// Duration/int fields: YAML takes precedence, programmatic fills gaps.
	if yamlConfig.Session.TokenTTL == 0 && programmaticConfig.Session.TokenTTL != 0 {
		yamlConfig.Session.TokenTTL = programmaticConfig.Session.TokenTTL
	}
	if yamlConfig.Session.RefreshTokenTTL == 0 && programmaticConfig.Session.RefreshTokenTTL != 0 {
		yamlConfig.Session.RefreshTokenTTL = programmaticConfig.Session.RefreshTokenTTL
	}
	if yamlConfig.Session.MaxActiveSessions == 0 && programmaticConfig.Session.MaxActiveSessions != 0 {
		yamlConfig.Session.MaxActiveSessions = programmaticConfig.Session.MaxActiveSessions
	}
	if yamlConfig.Session.RotateRefreshToken == nil && programmaticConfig.Session.RotateRefreshToken != nil {
		yamlConfig.Session.RotateRefreshToken = programmaticConfig.Session.RotateRefreshToken
	}

	// Password fields: YAML takes precedence, programmatic fills gaps.
	if yamlConfig.Password.MinLength == 0 && programmaticConfig.Password.MinLength != 0 {
		yamlConfig.Password.MinLength = programmaticConfig.Password.MinLength
	}
	if yamlConfig.Password.BcryptCost == 0 && programmaticConfig.Password.BcryptCost != 0 {
		yamlConfig.Password.BcryptCost = programmaticConfig.Password.BcryptCost
	}
	if programmaticConfig.Password.RequireUppercase {
		yamlConfig.Password.RequireUppercase = true
	}
	if programmaticConfig.Password.RequireLowercase {
		yamlConfig.Password.RequireLowercase = true
	}
	if programmaticConfig.Password.RequireDigit {
		yamlConfig.Password.RequireDigit = true
	}
	if programmaticConfig.Password.RequireSpecial {
		yamlConfig.Password.RequireSpecial = true
	}

	// Fill remaining zeros with defaults.
	return e.mergeWithDefaults(yamlConfig)
}

// buildEngineConfig maps the extension config to the core authsome.Config.
func (e *Extension) buildEngineConfig() authsome.Config {
	cfg := authsome.DefaultConfig()

	if e.config.BasePath != "" {
		cfg.BasePath = e.config.BasePath
	}
	cfg.Debug = e.config.Debug
	cfg.DisableRoutes = e.config.DisableRoutes
	cfg.DisableMigrate = e.config.DisableMigrate

	// Session
	if e.config.Session.TokenTTL != 0 {
		cfg.Session.TokenTTL = e.config.Session.TokenTTL
	}
	if e.config.Session.RefreshTokenTTL != 0 {
		cfg.Session.RefreshTokenTTL = e.config.Session.RefreshTokenTTL
	}
	if e.config.Session.MaxActiveSessions != 0 {
		cfg.Session.MaxActiveSessions = e.config.Session.MaxActiveSessions
	}
	if e.config.Session.RotateRefreshToken != nil {
		cfg.Session.RotateRefreshToken = e.config.Session.RotateRefreshToken
	}

	// Password
	if e.config.Password.MinLength != 0 {
		cfg.Password.MinLength = e.config.Password.MinLength
	}
	if e.config.Password.BcryptCost != 0 {
		cfg.Password.BcryptCost = e.config.Password.BcryptCost
	}
	if e.config.Password.RequireUppercase {
		cfg.Password.RequireUppercase = true
	}
	if e.config.Password.RequireLowercase {
		cfg.Password.RequireLowercase = true
	}
	if e.config.Password.RequireDigit {
		cfg.Password.RequireDigit = true
	}
	if e.config.Password.RequireSpecial {
		cfg.Password.RequireSpecial = true
	}
	if e.config.Password.Algorithm != "" {
		cfg.Password.Algorithm = e.config.Password.Algorithm
	}
	if e.config.Password.CheckBreached {
		cfg.Password.CheckBreached = true
	}
	if e.config.Password.Argon2.Memory != 0 {
		cfg.Password.Argon2.Memory = e.config.Password.Argon2.Memory
	}
	if e.config.Password.Argon2.Iterations != 0 {
		cfg.Password.Argon2.Iterations = e.config.Password.Argon2.Iterations
	}
	if e.config.Password.Argon2.Parallelism != 0 {
		cfg.Password.Argon2.Parallelism = e.config.Password.Argon2.Parallelism
	}
	if e.config.Password.Argon2.SaltLength != 0 {
		cfg.Password.Argon2.SaltLength = e.config.Password.Argon2.SaltLength
	}
	if e.config.Password.Argon2.KeyLength != 0 {
		cfg.Password.Argon2.KeyLength = e.config.Password.Argon2.KeyLength
	}

	// Rate limit
	if e.config.RateLimit.Enabled {
		cfg.RateLimit.Enabled = true
	}
	if e.config.RateLimit.SignInLimit != 0 {
		cfg.RateLimit.SignInLimit = e.config.RateLimit.SignInLimit
	}
	if e.config.RateLimit.SignUpLimit != 0 {
		cfg.RateLimit.SignUpLimit = e.config.RateLimit.SignUpLimit
	}
	if e.config.RateLimit.ForgotPasswordLimit != 0 {
		cfg.RateLimit.ForgotPasswordLimit = e.config.RateLimit.ForgotPasswordLimit
	}
	if e.config.RateLimit.MFAChallengeLimit != 0 {
		cfg.RateLimit.MFAChallengeLimit = e.config.RateLimit.MFAChallengeLimit
	}
	if e.config.RateLimit.WindowSeconds != 0 {
		cfg.RateLimit.WindowSeconds = e.config.RateLimit.WindowSeconds
	}

	// Lockout
	if e.config.Lockout.Enabled {
		cfg.Lockout.Enabled = true
	}
	if e.config.Lockout.MaxAttempts != 0 {
		cfg.Lockout.MaxAttempts = e.config.Lockout.MaxAttempts
	}
	if e.config.Lockout.LockoutDurationSeconds != 0 {
		cfg.Lockout.LockoutDurationSeconds = e.config.Lockout.LockoutDurationSeconds
	}
	if e.config.Lockout.ResetAfterSeconds != 0 {
		cfg.Lockout.ResetAfterSeconds = e.config.Lockout.ResetAfterSeconds
	}

	return cfg
}

// resolveGroveDB resolves a *grove.DB from the DI container.
// If GroveDatabase is set, it looks up the named DB; otherwise it uses the default.
func (e *Extension) resolveGroveDB(fapp forge.App) (*grove.DB, error) {
	if e.config.GroveDatabase != "" {
		db, err := vessel.InjectNamed[*grove.DB](fapp.Container(), e.config.GroveDatabase)
		if err != nil {
			return nil, fmt.Errorf("grove database %q not found in container: %w", e.config.GroveDatabase, err)
		}
		return db, nil
	}
	db, err := vessel.Inject[*grove.DB](fapp.Container())
	if err != nil {
		return nil, fmt.Errorf("default grove database not found in container: %w", err)
	}
	return db, nil
}

// buildStoreFromGroveDB constructs the appropriate store backend
// based on the grove driver type (pg, sqlite, mongo).
func (e *Extension) buildStoreFromGroveDB(db *grove.DB) (store.Store, error) {
	driverName := db.Driver().Name()
	switch driverName {
	case "pg":
		return pgstore.New(db), nil
	case "sqlite":
		return sqlitestore.New(db), nil
	case "mongo":
		return mongostore.New(db), nil
	default:
		return nil, fmt.Errorf("authsome: unsupported grove driver %q", driverName)
	}
}

// buildBootstrapOptions converts YAML bootstrap config to authsome.BootstrapOption values.
func (e *Extension) buildBootstrapOptions() []authsome.BootstrapOption {
	cfg := e.config.Bootstrap
	var opts []authsome.BootstrapOption

	if cfg.AppName != "" {
		opts = append(opts, authsome.WithBootstrapAppName(cfg.AppName))
	}
	if cfg.AppSlug != "" {
		opts = append(opts, authsome.WithBootstrapAppSlug(cfg.AppSlug))
	}
	if cfg.AppLogo != "" {
		opts = append(opts, authsome.WithBootstrapAppLogo(cfg.AppLogo))
	}
	if cfg.SkipDefaultEnvs {
		opts = append(opts, authsome.WithSkipDefaultEnvs())
	}
	if cfg.SkipDefaultRoles {
		opts = append(opts, authsome.WithSkipDefaultRoles())
	}

	// Override environments from YAML.
	if len(cfg.Environments) > 0 {
		envs := make([]authsome.BootstrapEnv, len(cfg.Environments))
		for i, env := range cfg.Environments {
			envs[i] = authsome.BootstrapEnv{
				Name:      env.Name,
				Slug:      env.Slug,
				Type:      environment.Type(env.Type),
				IsDefault: env.IsDefault,
			}
		}
		opts = append(opts, authsome.WithBootstrapEnvs(envs))
	}

	// Override roles from YAML.
	if len(cfg.Roles) > 0 {
		roles := make([]authsome.BootstrapRole, len(cfg.Roles))
		for i, role := range cfg.Roles {
			roles[i] = authsome.BootstrapRole{
				Name:        role.Name,
				Slug:        role.Slug,
				Description: role.Description,
			}
		}
		opts = append(opts, authsome.WithBootstrapRoles(roles))
	}

	return opts
}
