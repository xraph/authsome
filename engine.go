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
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/lockout"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/strategy"

	"github.com/xraph/keysmith"
	"github.com/xraph/warden"
)

// Engine is the core AuthSome orchestrator. It holds only interfaces —
// no concrete implementations — for maximum pluggability.
type Engine struct {
	config Config
	store  store.Store
	logger log.Logger

	// Plugin system
	plugins           *plugin.Registry
	hooks             *hook.Bus
	strategies        *strategy.Registry
	pendingPlugins        []plugin.Plugin
	pendingStrategies     []pendingStrategy
	pendingAppSessCfgs    []*appsessionconfig.Config

	// First-class authorization engine (optional, replaces bridge for RBAC)
	warden_ *warden.Engine

	// First-class key management engine (optional, replaces bridge for API keys)
	keysmith_ *keysmith.Engine

	// Optional forgery bridges (injected, not required)
	chronicle  bridge.Chronicle
	authorizer bridge.Authorizer
	keyManager bridge.KeyManager
	relay      bridge.EventRelay
	mailer     bridge.Mailer
	sms        bridge.SMSSender
	herald_    bridge.Herald
	vault      bridge.Vault
	dispatcher bridge.Dispatcher
	ledger     bridge.Ledger
	metrics    bridge.MetricsCollector

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

	// Bootstrap configuration (nil = disabled).
	bootstrapCfg  *BootstrapConfig
	platformAppID id.AppID

	started bool
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
	if e.warden_ == nil {
		return nil, errors.New("authsome: warden engine is required (use WithWarden option)")
	}

	// Initialize subsystems
	e.plugins = plugin.NewRegistry(e.logger)
	e.hooks = hook.NewBus(e.logger)
	e.strategies = strategy.NewRegistry(e.logger)

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

// Start initializes the engine, runs migrations, and starts plugins.
func (e *Engine) Start(ctx context.Context) error {
	if e.started {
		return nil
	}

	// Run store migrations (core + plugin groups together).
	if !e.config.DisableMigrate {
		extraGroups := e.plugins.CollectMigrationGroups(e.config.DriverName)
		if err := e.store.Migrate(ctx, extraGroups...); err != nil {
			return err
		}
	}

	// Bootstrap platform app (if configured).
	if e.bootstrapCfg != nil {
		if err := e.bootstrap(ctx); err != nil {
			return err
		}
	}

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

	// Initialize plugins
	e.plugins.EmitOnInit(ctx, e)

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

	// Register webhook event catalog with relay
	if e.relay != nil {
		if err := e.relay.RegisterEventTypes(ctx, bridge.WebhookEventCatalog()); err != nil {
			e.logger.Warn("authsome: failed to register webhook event catalog",
				log.String("error", err.Error()),
			)
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

// ──────────────────────────────────────────────────
// Accessors
// ──────────────────────────────────────────────────

// Store returns the persistence backend.
func (e *Engine) Store() store.Store { return e.store }

// Plugins returns the plugin registry.
func (e *Engine) Plugins() *plugin.Registry { return e.plugins }

// Hooks returns the global event bus.
func (e *Engine) Hooks() *hook.Bus { return e.hooks }

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
func (e *Engine) Herald() bridge.Herald { return e.herald_ }

// Vault returns the secrets/feature-flag/config bridge (may be nil).
func (e *Engine) Vault() bridge.Vault { return e.vault }

// Dispatcher returns the job queue bridge (may be nil).
func (e *Engine) Dispatcher() bridge.Dispatcher { return e.dispatcher }

// Ledger returns the billing/metering bridge (may be nil).
func (e *Engine) Ledger() bridge.Ledger { return e.ledger }

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
func (e *Engine) Warden() *warden.Engine { return e.warden_ }

// Keysmith returns the first-class key management engine (may be nil).
func (e *Engine) Keysmith() *keysmith.Engine { return e.keysmith_ }

// APIKeyStore returns the API key store. When Keysmith is available,
// operations delegate to Keysmith's full key management engine (gaining
// rate limiting, policy enforcement, key rotation, scope management, and
// usage tracking). Otherwise, it returns the composite store's API key methods.
func (e *Engine) APIKeyStore() apikey.Store {
	if e.keysmith_ != nil {
		return apikey.NewKeymithStore(e.keysmith_)
	}
	return e.store.(apikey.Store)
}

// ListUserRoleSlugs returns the slugs of all roles assigned to a user.
// This satisfies the middleware.RoleChecker interface.
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
