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

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/authz"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/lockout"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"

	"github.com/xraph/keysmith"
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

	// Dynamic settings manager (optional).
	settingsMgr *settings.Manager

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

	// Register authsome resource types with Warden (idempotent).
	if e.warden_ != nil {
		e.registerWardenResourceTypes(ctx)
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
		if err := e.warden_.Store().CreateResourceType(ctx, rt); err != nil {
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

// Settings returns the dynamic settings manager (may be nil).
func (e *Engine) Settings() *settings.Manager { return e.settingsMgr }

// ──────────────────────────────────────────────────
// Client configuration types
// ──────────────────────────────────────────────────

// ClientConfigResponse is the public client-facing configuration returned
// by the /client-config endpoint. It tells the frontend SDK which auth
// methods are enabled, available providers, branding, etc.
type ClientConfigResponse struct {
	Version          string                      `json:"version"`
	AppID            string                      `json:"app_id"`
	Branding         *ClientConfigBranding       `json:"branding,omitempty"`
	Password         *ClientConfigToggle         `json:"password,omitempty"`
	Social           *ClientConfigSocial         `json:"social,omitempty"`
	Passkey          *ClientConfigToggle         `json:"passkey,omitempty"`
	MFA              *ClientConfigMFA            `json:"mfa,omitempty"`
	MagicLink        *ClientConfigToggle         `json:"magiclink,omitempty"`
	SSO              *ClientConfigSSO            `json:"sso,omitempty"`
	SupportedPlugins []string                    `json:"supported_plugins"`
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
	Enabled     bool                          `json:"enabled"`
	Connections []ClientConfigSSOConnection   `json:"connections"`
}

// ClientConfigSSOConnection represents an SSO connection/provider.
type ClientConfigSSOConnection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

func providerDisplayName(id string) string {
	if name, ok := providerDisplayNames[id]; ok {
		return name
	}
	if len(id) == 0 {
		return id
	}
	return strings.ToUpper(id[:1]) + id[1:]
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
	var (
		passwordEnabled  bool
		socialEnabled    bool
		passkeyEnabled   bool
		mfaEnabled       bool
		magicLinkEnabled bool
		ssoEnabled       bool
		socialProviders  []ClientConfigSocialProvider
		ssoConnections   []ClientConfigSSOConnection
		mfaMethods       []string
		pluginNames      []string
	)

	for _, p := range e.plugins.Plugins() {
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
		}
	}

	// Load per-app overrides and apply on top of plugin defaults.
	if overrides, err := e.store.GetAppClientConfig(ctx, appID); err == nil {
		applyClientConfigOverrides(resp, overrides,
			&passwordEnabled, &socialEnabled, &passkeyEnabled,
			&mfaEnabled, &magicLinkEnabled, &ssoEnabled,
			&socialProviders, &ssoConnections, &mfaMethods,
		)
	}

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

	if pluginNames == nil {
		pluginNames = []string{}
	}
	resp.SupportedPlugins = pluginNames

	return resp
}

// applyClientConfigOverrides applies per-app overrides to the client config.
func applyClientConfigOverrides(
	resp *ClientConfigResponse,
	cfg *appclientconfig.Config,
	passwordEnabled, socialEnabled, passkeyEnabled *bool,
	mfaEnabled, magicLinkEnabled, ssoEnabled *bool,
	socialProviders *[]ClientConfigSocialProvider,
	ssoConnections *[]ClientConfigSSOConnection,
	mfaMethods *[]string,
) {
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
	if e.keysmith_ != nil {
		return apikey.NewKeymithStore(e.keysmith_)
	}
	return e.store.(apikey.Store)
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
	if e.keysmith_ != nil {
		ks := apikey.NewKeymithStore(e.keysmith_)
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
