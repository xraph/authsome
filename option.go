package authsome

import (
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/bridge/keysmithadapter"
	"github.com/xraph/authsome/bridge/wardenadapter"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/lockout"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/tokenformat"

	"github.com/xraph/keysmith"
	"github.com/xraph/warden"
)

// Option configures the AuthSome engine.
type Option func(*Engine)

// WithConfig sets the engine configuration.
func WithConfig(cfg Config) Option {
	return func(e *Engine) {
		e.config = cfg
	}
}

// WithStore sets the persistence backend.
func WithStore(s store.Store) Option {
	return func(e *Engine) {
		e.store = s
	}
}

// WithLogger sets the structured logger.
func WithLogger(logger log.Logger) Option {
	return func(e *Engine) {
		e.logger = logger
	}
}

// WithDebug enables verbose debug logging.
func WithDebug(debug bool) Option {
	return func(e *Engine) {
		e.config.Debug = debug
	}
}

// WithAppID sets the application identity used in forge.Scope.
func WithAppID(appID string) Option {
	return func(e *Engine) {
		e.config.AppID = appID
	}
}

// WithPlugin registers a plugin with the engine.
func WithPlugin(p plugin.Plugin) Option {
	return func(e *Engine) {
		e.pendingPlugins = append(e.pendingPlugins, p)
	}
}

// WithStrategy registers an authentication strategy with the given priority.
// Lower priority values are evaluated first.
func WithStrategy(s strategy.Strategy, priority int) Option {
	return func(e *Engine) {
		e.pendingStrategies = append(e.pendingStrategies, pendingStrategy{s, priority})
	}
}

// WithChronicle sets the audit trail bridge.
func WithChronicle(c bridge.Chronicle) Option {
	return func(e *Engine) {
		e.chronicle = c
	}
}

// WithAuthorizer sets the authorization bridge.
func WithAuthorizer(a bridge.Authorizer) Option {
	return func(e *Engine) {
		e.authorizer = a
	}
}

// WithWarden sets the required Warden authorization engine. All RBAC operations
// (roles, permissions, assignments, and permission checks) delegate to Warden's
// full RBAC+ReBAC+ABAC evaluation. Warden is required — NewEngine will return
// an error if this option is not provided.
// This also sets the bridge.Authorizer for backward compatibility.
func WithWarden(w *warden.Engine) Option {
	return func(e *Engine) {
		e.warden_ = w
		e.authorizer = wardenadapter.New(w)
	}
}

// WithKeysmith sets the first-class Keysmith key management engine. When set,
// API key operations delegate to Keysmith (gaining rate limiting, policy
// enforcement, key rotation with grace periods, scope management, usage
// tracking, and multi-tenant support).
// This also sets the bridge.KeyManager for backward compatibility.
func WithKeysmith(ks *keysmith.Engine) Option {
	return func(e *Engine) {
		e.keysmith_ = ks
		e.keyManager = keysmithadapter.New(ks)
	}
}

// WithKeyManager sets the key management bridge.
func WithKeyManager(km bridge.KeyManager) Option {
	return func(e *Engine) {
		e.keyManager = km
	}
}

// WithEventRelay sets the webhook/event relay bridge.
func WithEventRelay(r bridge.EventRelay) Option {
	return func(e *Engine) {
		e.relay = r
	}
}

// WithMailer sets the transactional email bridge.
func WithMailer(m bridge.Mailer) Option {
	return func(e *Engine) {
		e.mailer = m
	}
}

// WithSMSSender sets the SMS sending bridge for MFA verification codes.
func WithSMSSender(s bridge.SMSSender) Option {
	return func(e *Engine) {
		e.sms = s
	}
}

// WithHerald sets the unified notification system bridge.
// When configured, Herald replaces the separate Mailer and SMSSender bridges
// for notification delivery, providing multi-channel support with templates,
// scoped configuration, and user preference management.
func WithHerald(h bridge.Herald) Option {
	return func(e *Engine) {
		e.herald_ = h
	}
}

// WithVault sets the secrets, feature flags, and configuration bridge.
func WithVault(v bridge.Vault) Option {
	return func(e *Engine) {
		e.vault = v
	}
}

// WithDispatcher sets the background job queue bridge.
func WithDispatcher(d bridge.Dispatcher) Option {
	return func(e *Engine) {
		e.dispatcher = d
	}
}

// WithLedger sets the billing/metering bridge.
func WithLedger(l bridge.Ledger) Option {
	return func(e *Engine) {
		e.ledger = l
	}
}

// WithBasePath sets the URL prefix for all auth routes.
func WithBasePath(path string) Option {
	return func(e *Engine) {
		e.config.BasePath = path
	}
}

// WithDisableRoutes prevents automatic route registration.
func WithDisableRoutes() Option {
	return func(e *Engine) {
		e.config.DisableRoutes = true
	}
}

// WithDisableMigrate prevents automatic migration on Start.
func WithDisableMigrate() Option {
	return func(e *Engine) {
		e.config.DisableMigrate = true
	}
}

// WithDriverName sets the grove driver name for plugin migration discovery.
func WithDriverName(name string) Option {
	return func(e *Engine) {
		e.config.DriverName = name
	}
}

// WithRateLimiter sets the rate limiter for brute-force protection.
func WithRateLimiter(rl ratelimit.Limiter) Option {
	return func(e *Engine) {
		e.rateLimiter = rl
	}
}

// WithLockoutTracker sets the account lockout tracker.
func WithLockoutTracker(t lockout.Tracker) Option {
	return func(e *Engine) {
		e.lockout = t
	}
}

// WithLockoutConfig sets the lockout configuration.
func WithLockoutConfig(cfg LockoutConfig) Option {
	return func(e *Engine) {
		e.config.Lockout = cfg
	}
}

// WithMetrics sets the metrics collector for observability.
func WithMetrics(m bridge.MetricsCollector) Option {
	return func(e *Engine) {
		e.metrics = m
	}
}

// WithCeremonyStore sets the ceremony state store for short-lived auth
// ceremony sessions (passkey, social OAuth, SSO, MFA SMS challenges).
// When not set, plugins fall back to an in-memory store.
func WithCeremonyStore(s ceremony.Store) Option {
	return func(e *Engine) {
		e.ceremonyStore = s
	}
}

// WithPasswordHistory sets the password history store for preventing
// password reuse. Works in conjunction with PasswordConfig.HistoryCount.
func WithPasswordHistory(s account.PasswordHistoryStore) Option {
	return func(e *Engine) {
		e.passwordHistory = s
	}
}

// WithSecurityEvents sets the security event store for persisting and
// querying security-relevant events (failed logins, lockouts, etc.).
func WithSecurityEvents(s securityevent.Store) Option {
	return func(e *Engine) {
		e.securityEvents = s
	}
}

// WithAppSessionConfig registers a per-app session configuration override.
// The config is seeded into the store during engine Start and overrides the
// global session config for sessions created under the specified app.
func WithAppSessionConfig(cfg *appsessionconfig.Config) Option {
	return func(e *Engine) {
		e.pendingAppSessCfgs = append(e.pendingAppSessCfgs, cfg)
	}
}

// WithDefaultTokenFormat sets the default token format for access tokens.
// When not set, opaque tokens (64-char hex) are used.
func WithDefaultTokenFormat(f tokenformat.Format) Option {
	return func(e *Engine) {
		e.defaultTokenFormat = f
	}
}

// WithJWTFormat registers a JWT token format for a specific app.
// Access tokens for this app will be signed JWTs instead of opaque tokens.
func WithJWTFormat(appID string, jwtFmt *tokenformat.JWT) Option {
	return func(e *Engine) {
		if e.jwtFormats == nil {
			e.jwtFormats = make(map[string]*tokenformat.JWT)
		}
		e.jwtFormats[appID] = jwtFmt
	}
}

// WithBootstrap enables automatic platform app bootstrap with optional customization.
func WithBootstrap(opts ...BootstrapOption) Option {
	return func(e *Engine) {
		cfg := DefaultBootstrapConfig()
		for _, opt := range opts {
			opt(cfg)
		}
		e.bootstrapCfg = cfg
	}
}

type pendingStrategy struct {
	strategy strategy.Strategy
	priority int
}
