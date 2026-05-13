package social

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"

	"github.com/xraph/grove/migrate"

	"golang.org/x/oauth2"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin                = (*Plugin)(nil)
	_ plugin.RouteProvider         = (*Plugin)(nil)
	_ plugin.OnInit                = (*Plugin)(nil)
	_ plugin.MigrationProvider     = (*Plugin)(nil)
	_ plugin.AuthMethodContributor = (*Plugin)(nil)
	_ plugin.AuthMethodUnlinker    = (*Plugin)(nil)
	_ plugin.SettingsProvider      = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingSessionTokenTTLSeconds controls the session token lifetime for social sign-in.
	SettingSessionTokenTTLSeconds = settings.Define("social.session_token_ttl_seconds", 3600,
		settings.WithDisplayName("Session Token TTL (seconds)"),
		settings.WithDescription("Lifetime of sessions created via social sign-in in seconds"),
		settings.WithCategory("Social OAuth"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(300), Max: intPtr(86400)}),
		settings.WithHelpText("How long sessions created via social login remain valid. Default: 3600 (1 hour)"),
		settings.WithOrder(10),
	)

	// SettingSessionRefreshTTLSeconds controls the refresh token lifetime for social sessions.
	SettingSessionRefreshTTLSeconds = settings.Define("social.session_refresh_ttl_seconds", 2592000,
		settings.WithDisplayName("Refresh Token TTL (seconds)"),
		settings.WithDescription("Lifetime of refresh tokens for social sign-in sessions in seconds"),
		settings.WithCategory("Social OAuth"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(3600), Max: intPtr(7776000)}),
		settings.WithHelpText("How long refresh tokens remain valid. Default: 2592000 (30 days)"),
		settings.WithOrder(20),
	)
)

// ProviderSetting represents a social provider configured via the dashboard.
type ProviderSetting struct {
	Name         string   `json:"name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      bool     `json:"enabled"`
}

// SettingSocialProviders stores dashboard-configured social providers.
var SettingSocialProviders = settings.Define("social.providers", []ProviderSetting{},
	settings.WithDisplayName("Social Providers"),
	settings.WithDescription("Social OAuth providers configured via dashboard"),
	settings.WithCategory("Social OAuth"),
	settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
	settings.WithSensitive(),
	settings.WithOrder(30),
	settings.WithObjectFields(
		settings.ObjectFieldDef{
			Key:         "name",
			DisplayName: "Provider",
			InputType:   formconfig.FieldSelect,
			Required:    true,
			Options: []formconfig.SelectOption{
				{Label: "Google", Value: "google"},
				{Label: "GitHub", Value: "github"},
				{Label: "Apple", Value: "apple"},
				{Label: "Microsoft", Value: "microsoft"},
				{Label: "Facebook", Value: "facebook"},
				{Label: "LinkedIn", Value: "linkedin"},
				{Label: "Discord", Value: "discord"},
				{Label: "Slack", Value: "slack"},
				{Label: "Twitter", Value: "twitter"},
				{Label: "Spotify", Value: "spotify"},
				{Label: "Twitch", Value: "twitch"},
				{Label: "GitLab", Value: "gitlab"},
				{Label: "Bitbucket", Value: "bitbucket"},
				{Label: "Dropbox", Value: "dropbox"},
				{Label: "Yahoo", Value: "yahoo"},
				{Label: "Amazon", Value: "amazon"},
				{Label: "Zoom", Value: "zoom"},
				{Label: "Pinterest", Value: "pinterest"},
				{Label: "Strava", Value: "strava"},
				{Label: "Patreon", Value: "patreon"},
				{Label: "Instagram", Value: "instagram"},
				{Label: "Line", Value: "line"},
			},
		},
		settings.ObjectFieldDef{
			Key:         "client_id",
			DisplayName: "Client ID",
			InputType:   formconfig.FieldText,
			Required:    true,
			Placeholder: "OAuth client ID",
		},
		settings.ObjectFieldDef{
			Key:         "client_secret",
			DisplayName: "Client Secret",
			InputType:   formconfig.FieldText,
			Required:    true,
			Sensitive:   true,
			Placeholder: "OAuth client secret",
		},
		settings.ObjectFieldDef{
			Key:         "redirect_url",
			DisplayName: "Redirect URL",
			InputType:   formconfig.FieldURL,
			Placeholder: "https://example.com/auth/callback",
			HelpText:    "The OAuth callback URL registered with the provider",
		},
		settings.ObjectFieldDef{
			Key:         "scopes",
			DisplayName: "Scopes",
			InputType:   formconfig.FieldTextarea,
			Placeholder: "openid\nprofile\nemail",
			HelpText:    "One scope per line",
		},
		settings.ObjectFieldDef{
			Key:         "enabled",
			DisplayName: "Enabled",
			InputType:   formconfig.FieldSwitch,
		},
	),
)

func intPtr(v int) *int { return &v }

// Config configures the social OAuth plugin.
type Config struct {
	// Providers is the list of enabled OAuth providers.
	Providers []Provider

	// Domain is the external base URL of the server (e.g. "https://api.example.com").
	// When a provider does not specify a RedirectURL, the callback URL is
	// auto-generated as {Domain}/v1/social/{provider}/callback.
	// If empty, the callback URL is derived from the incoming request's Host header.
	Domain string

	// SessionTokenTTL is the lifetime of sessions created via social sign-in (default: 1 hour).
	SessionTokenTTL time.Duration

	// SessionRefreshTTL is the lifetime of refresh tokens (default: 30 days).
	SessionRefreshTTL time.Duration
}

// Plugin is the social OAuth authentication plugin.
type Plugin struct {
	config      Config
	providers   map[string]Provider
	store       store.Store // Core authsome store (for users/sessions)
	oauthStore  Store       // OAuth-specific store (for connections)
	appID       string
	engine      plugin.Engine
	settingsMgr *settings.Manager

	chronicle  bridge.Chronicle
	relay      bridge.EventRelay
	hooks      *hook.Bus
	logger     log.Logger
	ceremonies ceremony.Store
	basePath   string // authsome route prefix (e.g. "/authsome", "/api/identity")
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "social", SettingSessionTokenTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "social", SettingSessionRefreshTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "social", SettingSocialProviders); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "social", SettingAllowedFrontendURLs)
}

// New creates a new social OAuth plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.SessionTokenTTL == 0 {
		c.SessionTokenTTL = 1 * time.Hour
	}
	if c.SessionRefreshTTL == 0 {
		c.SessionRefreshTTL = 30 * 24 * time.Hour
	}

	providers := make(map[string]Provider, len(c.Providers))
	for _, p := range c.Providers {
		providers[p.Name()] = p
	}

	return &Plugin{
		config:     c,
		providers:  providers,
		ceremonies: ceremony.NewMemory(),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "social" }

// OnInit captures the store reference and bridges from the engine.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.store = engine.Store()
	p.chronicle = engine.Chronicle()
	p.relay = engine.Relay()
	p.hooks = engine.Hooks()
	p.logger = engine.Logger()
	p.ceremonies = engine.CeremonyStore()
	if p.ceremonies == nil {
		p.ceremonies = ceremony.NewMemory()
	}
	p.settingsMgr = engine.Settings()
	p.basePath = engine.BasePath()

	// If an OAuth store has been wired (via SetOAuthStore) and is not
	// already encrypted, transparently wrap it so access/refresh tokens
	// are encrypted at rest. Skipping when the engine's encryptor is a
	// Noop is fine — wrapping is still semantically a passthrough.
	if p.oauthStore != nil {
		if _, already := p.oauthStore.(*EncryptedStore); !already {
			p.oauthStore = NewEncryptedStore(p.oauthStore, engine.TokenEncryptor())
		}
	}

	return nil
}

// MigrationGroups implements plugin.MigrationProvider.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite":
		return []*migrate.Group{SqliteMigrations}
	default:
		return nil
	}
}

// SetStore sets the core store for testing.
func (p *Plugin) SetStore(s store.Store) {
	p.store = s
}

// SetOAuthStore sets the OAuth connection store.
func (p *Plugin) SetOAuthStore(s Store) {
	p.oauthStore = s
}

// SetAppID sets the default app ID.
func (p *Plugin) SetAppID(appID string) {
	p.appID = appID
}

// Providers returns the list of all active provider names (code + DB).
func (p *Plugin) Providers() []string {
	return p.allProviderNames(context.Background())
}

// resolveProvider resolves a provider by name. Code-configured providers
// take precedence over dashboard-configured ones.
func (p *Plugin) resolveProvider(ctx context.Context, name string) (Provider, bool) {
	// Code providers always win.
	if prov, ok := p.providers[name]; ok {
		return prov, true
	}
	// Check DB-configured providers.
	dbProviders := p.loadDBProviderSettings(ctx)
	for _, s := range dbProviders {
		if s.Name == name && s.Enabled {
			prov := providerFromSetting(s)
			if prov != nil {
				return prov, true
			}
		}
	}
	return nil, false
}

// loadDBProviderSettings reads dynamic providers from the settings store.
func (p *Plugin) loadDBProviderSettings(ctx context.Context) []ProviderSetting {
	if p.settingsMgr == nil {
		return nil
	}
	providers, err := settings.Get(ctx, p.settingsMgr, SettingSocialProviders, settings.ResolveOpts{})
	if err != nil {
		return nil
	}
	return providers
}

// saveDBProviderSettings writes dynamic providers to the settings store.
func (p *Plugin) saveDBProviderSettings(ctx context.Context, providers []ProviderSetting) error {
	if p.settingsMgr == nil {
		return fmt.Errorf("social: settings manager not available")
	}
	raw, err := json.Marshal(providers)
	if err != nil {
		return err
	}
	return p.settingsMgr.Set(ctx, SettingSocialProviders.Def.Key, raw,
		settings.ScopeGlobal, "", "", "", "dashboard")
}

// providerFromSetting creates a Provider from a ProviderSetting.
func providerFromSetting(s ProviderSetting) Provider {
	cfg := ProviderConfig{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecret,
		RedirectURL:  s.RedirectURL,
		Scopes:       s.Scopes,
	}
	switch strings.ToLower(s.Name) {
	case "google":
		return NewGoogleProvider(cfg)
	case "github":
		return NewGitHubProvider(cfg)
	case "apple":
		return NewAppleProvider(cfg)
	case "microsoft":
		return NewMicrosoftProvider(cfg)
	case "facebook":
		return NewFacebookProvider(cfg)
	case "linkedin":
		return NewLinkedInProvider(cfg)
	case "discord":
		return NewDiscordProvider(cfg)
	case "slack":
		return NewSlackProvider(cfg)
	case "twitter":
		return NewTwitterProvider(cfg)
	case "spotify":
		return NewSpotifyProvider(cfg)
	case "twitch":
		return NewTwitchProvider(cfg)
	case "gitlab":
		return NewGitLabProvider(cfg)
	case "bitbucket":
		return NewBitbucketProvider(cfg)
	case "dropbox":
		return NewDropboxProvider(cfg)
	case "yahoo":
		return NewYahooProvider(cfg)
	case "amazon":
		return NewAmazonProvider(cfg)
	case "zoom":
		return NewZoomProvider(cfg)
	case "pinterest":
		return NewPinterestProvider(cfg)
	case "strava":
		return NewStravaProvider(cfg)
	case "patreon":
		return NewPatreonProvider(cfg)
	case "instagram":
		return NewInstagramProvider(cfg)
	case "line":
		return NewLineProvider(cfg)
	default:
		return nil
	}
}

// allProviderNames returns names of all providers (code + enabled DB).
func (p *Plugin) allProviderNames(ctx context.Context) []string {
	names := make(map[string]struct{})
	for name := range p.providers {
		names[name] = struct{}{}
	}
	dbProviders := p.loadDBProviderSettings(ctx)
	for _, s := range dbProviders {
		if s.Enabled {
			names[s.Name] = struct{}{}
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// RegisterRoutes registers social OAuth HTTP endpoints on a forge.Router.
//
// Both endpoints get the same per-IP rate limit as the JSON sign-in
// endpoint (default 5/window). Without this, an attacker can amplify
// requests against the upstream OAuth provider — the start endpoint
// generates state cookies and ceremony entries, the callback exchanges
// tokens and hits the provider's siteverify-equivalent endpoint.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	g := router.Group("/v1/social", forge.WithGroupTags("Social OAuth"))

	startOpts := []forge.RouteOption{
		forge.WithSummary("Start OAuth flow"),
		forge.WithOperationID("startOAuth"),
		forge.WithResponseSchema(http.StatusOK, "OAuth authorization URL", StartResponse{}),
		forge.WithErrorResponses(),
	}
	startOpts = append(startOpts, p.rateLimitOpts(rateLimitForStart)...)
	if err := g.POST("/:provider", p.handleStart, startOpts...); err != nil {
		return err
	}

	callbackOpts := []forge.RouteOption{
		forge.WithSummary("OAuth callback"),
		forge.WithOperationID("oauthCallback"),
		forge.WithResponseSchema(http.StatusOK, "Authentication result", CallbackResponse{}),
		forge.WithErrorResponses(),
	}
	callbackOpts = append(callbackOpts, p.rateLimitOpts(rateLimitForCallback)...)
	if err := g.GET("/:provider/callback", p.handleCallback, callbackOpts...); err != nil {
		return err
	}

	admin := router.Group("/v1/admin/social", forge.WithGroupTags("Social OAuth Admin"))
	if err := admin.GET("/providers", p.handleAdminListProviders,
		forge.WithSummary("List social providers (admin)"),
		forge.WithDescription("Returns the social providers configured at the resolved scope. When app_id is supplied, providers are merged from global + app overrides. Client secrets are masked."),
		forge.WithOperationID("socialAdminListProviders"),
		forge.WithRequestSchema(AdminListProvidersRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Providers", AdminListProvidersResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := admin.GET("/providers/catalog", p.handleAdminCatalog,
		forge.WithSummary("List supported social providers"),
		forge.WithDescription("Returns every provider this build of authsome can speak to (the static catalog). Use this to populate a 'pick a provider' UI before configuring credentials."),
		forge.WithOperationID("socialAdminCatalog"),
		forge.WithResponseSchema(http.StatusOK, "Catalog", AdminCatalogResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	if err := admin.PUT("/providers/:provider", p.handleAdminUpsertProvider,
		forge.WithSummary("Configure a social provider (admin)"),
		forge.WithDescription("Upserts the per-app provider configuration. Pass app_id to scope per-app; omit for global. Replaces any existing entry for the same provider name."),
		forge.WithOperationID("socialAdminUpsertProvider"),
		forge.WithRequestSchema(AdminUpsertProviderRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Provider stored", AdminProviderResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
	return admin.DELETE("/providers/:provider", p.handleAdminDeleteProvider,
		forge.WithSummary("Delete a social provider (admin)"),
		forge.WithOperationID("socialAdminDeleteProvider"),
		forge.WithRequestSchema(AdminDeleteProviderRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Deleted", AdminDeleteProviderResponse{}),
		forge.WithErrorResponses(),
	)
}

// rateLimitTarget selects which configured limit to apply.
type rateLimitTarget int

const (
	// rateLimitForStart caps OAuth-flow initiation; uses the engine's
	// configured SignUpLimit (default 3/window) so a bot can't farm
	// state-token allocations against the ceremony store.
	rateLimitForStart rateLimitTarget = iota
	// rateLimitForCallback caps the redirect-back endpoint. Uses the
	// SignInLimit (default 5/window) — slightly more generous because
	// browsers may retry on transient network errors during the OAuth
	// bounce.
	rateLimitForCallback
)

// rateLimitOpts returns a forge route option applying per-IP rate limits
// to the social endpoint, or nil when rate limiting is disabled or the
// engine isn't the concrete *authsome.Engine (e.g. test wiring).
func (p *Plugin) rateLimitOpts(target rateLimitTarget) []forge.RouteOption {
	eng, ok := p.engine.(*authsome.Engine)
	if !ok || eng == nil {
		return nil
	}
	rl := eng.RateLimiter()
	cfg := eng.Config().RateLimit
	if rl == nil || !cfg.Enabled {
		return nil
	}
	limit := cfg.SignUpLimit
	if target == rateLimitForCallback {
		limit = cfg.SignInLimit
	}
	if limit <= 0 {
		return nil
	}
	return []forge.RouteOption{
		forge.WithMiddleware(middleware.RateLimit(rl, middleware.RateLimitConfig{
			Limit:  limit,
			Window: cfg.Window(),
		})),
	}
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// StartRequest contains the path parameter for starting OAuth.
//
// FrontendURL is the originating SPA's root (e.g. "https://app.example.com")
// for split-origin deployments where the auth service runs on a different
// host than the frontend. It serves two purposes:
//  1. Trusted origin for validating RedirectURL when the request's Origin/Referer
//     headers can't be relied on (CORS, server-to-server, popup contexts).
//  2. Fallback redirect target when RedirectURL is empty or the auth flow
//     fails before a redirect target can be resolved.
//
// RedirectURL is the post-auth destination — where to send the browser after
// a successful login/signup. If empty, callers will fall back to FrontendURL.
type StartRequest struct {
	Provider    string `path:"provider"`
	FrontendURL string `json:"frontend_url,omitempty" query:"frontend_url,omitempty"`
	RedirectURL string `json:"redirect_url,omitempty" query:"redirect_url,omitempty"`
}

// StartResponse is returned when the OAuth flow is initiated.
type StartResponse struct {
	AuthURL string `json:"auth_url"`
}

// CallbackRequest contains the path and query parameters for the OAuth callback.
type CallbackRequest struct {
	Provider string `path:"provider"`
	State    string `query:"state,omitempty"`
	Code     string `query:"code,omitempty"`
	Error    string `query:"error,omitempty"`
}

// CallbackResponse is returned on successful OAuth authentication.
//
// RedirectURL and FrontendURL echo the values stashed in the OAuth state
// during handleStart so non-browser callers (mobile apps, native flows) can
// route the user without having to track them separately.
type CallbackResponse struct {
	User         any    `json:"user"`
	SessionToken string `json:"session_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    any    `json:"expires_at"`
	Provider     string `json:"provider"`
	IsNewUser    bool   `json:"is_new_user"`
	RedirectURL  string `json:"redirect_url,omitempty"`
	FrontendURL  string `json:"frontend_url,omitempty"`
}

// ──────────────────────────────────────────────────
// Forge Handlers
// ──────────────────────────────────────────────────

// handleStart initiates the OAuth flow by returning the authorization URL.
// resolveAppID returns the app ID from the request scope, falling back to
// the plugin-level default. This allows multi-app deployments where each
// app can initiate its own social login flow.
func (p *Plugin) resolveAppID(ctx forge.Context) (id.AppID, error) {
	if scopeAppID := forge.AppIDFrom(ctx.Context()); scopeAppID != "" {
		return id.ParseAppID(scopeAppID)
	}
	if p.appID != "" {
		return id.ParseAppID(p.appID)
	}
	return id.AppID{}, fmt.Errorf("no app_id available")
}

func (p *Plugin) handleStart(ctx forge.Context, req *StartRequest) (*StartResponse, error) {
	provider, ok := p.resolveProvider(ctx.Context(), req.Provider)
	if !ok {
		return nil, forge.BadRequest(fmt.Sprintf("unsupported provider: %s", req.Provider))
	}

	appID, err := p.resolveAppID(ctx)
	if err != nil {
		return nil, forge.BadRequest("unable to determine app_id for social login")
	}

	// Resolve the default environment so the callback knows which env
	// the user should be created in (supports multi-app / multi-env).
	var envIDStr string
	if env, _ := p.store.GetDefaultEnvironment(ctx.Context(), appID); env != nil { //nolint:errcheck // best-effort env lookup
		envIDStr = env.ID.String()
	}

	state, err := generateState()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate state: %w", err))
	}

	// Validate the redirect URL. The trust authority is gated by the
	// per-app auth.allowed_frontend_urls allowlist:
	//   1. Caller-supplied frontend_url, IF its host is on the allowlist
	//      (for split-origin deployments where the SPA and auth service
	//      live on different hosts).
	//   2. Origin / Referer header, IF its host is on the allowlist.
	// Falling off both → safeRedirect can only accept relative paths.
	// frontend_url itself must also be an absolute http(s) URL with no
	// embedded credentials; relative paths are rejected because we'll use
	// it as a fallback redirect target.
	safeFrontend := sanitizeFrontendURL(req.FrontendURL)
	if safeFrontend != "" && !isAllowedOrigin(ctx.Context(), p.settingsMgr, appID, safeFrontend) {
		safeFrontend = ""
	}
	originForRedirect := safeFrontend
	if originForRedirect == "" {
		candidate := ctx.Request().Header.Get("Origin")
		if candidate == "" {
			candidate = ctx.Request().Header.Get("Referer")
		}
		if candidate != "" && isAllowedOrigin(ctx.Context(), p.settingsMgr, appID, candidate) {
			originForRedirect = candidate
		}
	}
	safeRedirect := sanitizeRedirectURL(req.RedirectURL, originForRedirect)

	// Store the state for CSRF protection, including app and env IDs so the
	// callback can resolve them without relying on global defaults.
	// Resolve the OAuth callback URL. If the provider has no RedirectURL
	// configured, auto-generate one from Config.Domain or the request host.
	cfg := provider.OAuth2Config()
	callbackURL := p.resolveCallbackURL(ctx.Request(), cfg.RedirectURL, req.Provider)

	// PKCE (RFC 7636): generate a per-flow verifier, derive its S256
	// challenge, send the challenge in the auth URL, store the verifier
	// in state, and present it on the token exchange. Closes the
	// authorization-code-interception attack class — even if an attacker
	// captures the redirect-back code (logs, browser history, hostile
	// proxy), without the verifier they can't exchange it.
	pkceVerifier, err := generateState()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate PKCE verifier: %w", err))
	}

	// OIDC nonce (OpenID Connect Core §3.1.2.1): per-flow random value
	// echoed back in the ID token's `nonce` claim. Verifying the claim
	// matches what we stashed in state defeats ID-token replay. We send
	// the nonce on every flow; providers that don't issue ID tokens
	// silently ignore it.
	oidcNonce, err := generateState()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate OIDC nonce: %w", err))
	}

	stateInfo := map[string]string{
		"provider":      req.Provider,
		"app_id":        appID.String(),
		"env_id":        envIDStr,
		"frontend_url":  safeFrontend,
		"redirect_url":  safeRedirect,
		"callback_url":  callbackURL,
		"pkce_verifier": pkceVerifier,
		"oidc_nonce":    oidcNonce,
	}
	stateData, _ := json.Marshal(stateInfo) //nolint:errcheck // marshaling known types
	// Namespace the state key by app so a state minted for app A can't
	// be replayed against app B's callback even if both share the same
	// ceremony store. Closes the cross-tenant state-confusion attack
	// surfaced in the Phase 1 audit.
	_ = p.ceremonies.Set(ctx.Context(), socialStateKey(appID, state), stateData, 10*time.Minute) //nolint:errcheck // best-effort cache

	// Clone the config so we don't mutate the provider's stored config.
	authCfg := *cfg
	authCfg.RedirectURL = callbackURL
	authURL := authCfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", pkceChallengeS256(pkceVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("nonce", oidcNonce),
	)

	return &StartResponse{AuthURL: authURL}, nil
}

// socialStateKey is the ceremony-store key under which an OAuth state
// envelope lives. Namespaced by app to prevent cross-tenant state replay
// when the ceremony store is shared.
func socialStateKey(appID id.AppID, state string) string {
	return "social:state:" + appID.String() + ":" + state
}

// pkceChallengeS256 returns the RFC 7636 S256 code_challenge for verifier.
// base64url-no-padding(SHA256(verifier)).
func pkceChallengeS256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// loadOAuthState fetches the state envelope, trying the namespaced key
// (Phase 2D) first and falling back to the legacy unnamespaced key for
// states that were minted before the rollout. Returns the raw envelope
// plus the appID under which it was found (empty when the legacy key
// hit, so the caller can fall back to stateInfo["app_id"]).
func (p *Plugin) loadOAuthState(ctx context.Context, appID id.AppID, state string) ([]byte, string, error) {
	if !appID.IsNil() {
		if data, err := p.ceremonies.Get(ctx, socialStateKey(appID, state)); err == nil {
			_ = p.ceremonies.Delete(ctx, socialStateKey(appID, state)) //nolint:errcheck // best-effort
			return data, appID.String(), nil
		}
	}
	// Legacy key shape — for in-flight states minted before namespacing.
	// Eligible for removal one TTL window after Phase 2D rolls.
	legacyKey := "social:state:" + state
	data, err := p.ceremonies.Get(ctx, legacyKey)
	if err != nil {
		return nil, "", err
	}
	_ = p.ceremonies.Delete(ctx, legacyKey) //nolint:errcheck // best-effort
	return data, "", nil
}

// verifyOIDCNonce decodes the ID token's payload (no signature check —
// signature verification is a separate hardening item) and returns true
// iff the `nonce` claim equals the value stashed in state during
// handleStart.
//
// Returns true when no ID token is present (provider isn't OIDC) so
// non-OIDC providers (Twitter, GitHub legacy) keep working — those flows
// don't carry a nonce in their callback shape.
func verifyOIDCNonce(token *oauth2.Token, expectedNonce string) bool {
	if expectedNonce == "" {
		return true
	}
	idToken, _ := token.Extra("id_token").(string)
	if idToken == "" {
		// Provider didn't return an ID token; non-OIDC flow.
		return true
	}
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some providers pad — try standard decoding too.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return false
		}
	}
	var claims struct {
		Nonce string `json:"nonce"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}
	return claims.Nonce == expectedNonce
}

// handleCallback processes the OAuth callback, exchanges the code for a token,
// fetches the user profile, and either links to an existing user or creates a new one.
func (p *Plugin) handleCallback(ctx forge.Context, req *CallbackRequest) (*CallbackResponse, error) {
	provider, ok := p.resolveProvider(ctx.Context(), req.Provider)
	if !ok {
		return nil, forge.BadRequest(fmt.Sprintf("unsupported provider: %s", req.Provider))
	}

	// Validate state parameter
	if req.State == "" {
		return nil, forge.BadRequest("missing state parameter")
	}

	// State is stored under a per-app key (Phase 2D); resolve the app
	// from the request's scope first, falling back to the plugin-level
	// default. We probe both the namespaced and legacy key shape so
	// in-flight states minted before this change still complete during
	// the rollout window.
	candidateAppID, _ := p.resolveAppID(ctx) //nolint:errcheck // ok if zero — legacy probe handles it
	stateData, stateAppID, err := p.loadOAuthState(ctx.Context(), candidateAppID, req.State)
	if err != nil {
		return nil, forge.BadRequest("invalid state parameter")
	}
	var stateInfo map[string]string
	if unmarshalErr := json.Unmarshal(stateData, &stateInfo); unmarshalErr != nil || stateInfo["provider"] != req.Provider {
		return nil, forge.BadRequest("invalid state parameter")
	}

	// Resolve the app ID from the state (set during handleStart).
	appIDStr := stateInfo["app_id"]
	if appIDStr == "" {
		appIDStr = stateAppID
	}
	if appIDStr == "" {
		// Fallback for states created before app_id was stored.
		appIDStr = p.appID
	}
	appID, err := id.ParseAppID(appIDStr)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("invalid app_id in OAuth state: %w", err))
	}

	// Check for error from provider
	if req.Error != "" {
		return p.callbackError(ctx, stateInfo, fmt.Sprintf("provider error: %s", req.Error))
	}

	// Exchange code for token
	if req.Code == "" {
		return p.callbackError(ctx, stateInfo, "missing code parameter")
	}

	cfg := provider.OAuth2Config()

	// Use the callback URL stored during handleStart so the token exchange
	// sends the exact same redirect_uri the authorization request used.
	exchangeCfg := *cfg
	if cbURL := stateInfo["callback_url"]; cbURL != "" {
		exchangeCfg.RedirectURL = cbURL
	}
	exchangeOpts := []oauth2.AuthCodeOption{}
	if verifier := stateInfo["pkce_verifier"]; verifier != "" {
		// Present the PKCE code_verifier on token exchange (RFC 7636).
		// Providers that didn't accept the challenge ignore this field;
		// providers that did require it will reject mismatches.
		exchangeOpts = append(exchangeOpts, oauth2.SetAuthURLParam("code_verifier", verifier))
	}
	token, err := exchangeCfg.Exchange(ctx.Context(), req.Code, exchangeOpts...)
	if err != nil {
		return nil, forge.BadRequest("failed to exchange code")
	}

	// OIDC nonce verification (no-op for providers that don't issue
	// ID tokens or weren't given a nonce). Done before any further
	// processing so a tampered ID token can't reach the provider's
	// FetchUser path.
	if !verifyOIDCNonce(token, stateInfo["oidc_nonce"]) {
		return nil, forge.BadRequest("invalid id_token nonce")
	}

	// Fetch user profile from provider
	providerUser, err := provider.FetchUser(ctx.Context(), token)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to fetch user from provider: %w", err))
	}

	goCtx := ctx.Context()

	// Resolve env ID from the state (set during handleStart). Fall back to
	// the default environment when the state doesn't carry one.
	var envID id.EnvironmentID
	if eid := stateInfo["env_id"]; eid != "" {
		envID, _ = id.ParseEnvironmentID(eid) //nolint:errcheck // best-effort; zero value is safe
	}
	if envID.IsNil() {
		if env, _ := p.store.GetDefaultEnvironment(goCtx, appID); env != nil { //nolint:errcheck // best-effort env lookup
			envID = env.ID
		}
	}

	// Check if an OAuth connection already exists
	var u *user.User
	if p.oauthStore != nil {
		conn, connErr := p.oauthStore.GetOAuthConnection(goCtx, req.Provider, providerUser.ProviderUserID)
		if connErr == nil {
			// Existing connection — look up the user
			u, err = p.store.GetUser(goCtx, conn.UserID)
			if err != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to resolve user: %w", err))
			}

			// Update tokens
			conn.AccessToken = token.AccessToken
			conn.RefreshToken = token.RefreshToken
			conn.ExpiresAt = token.Expiry
			conn.UpdatedAt = time.Now()
		}
	}

	if u == nil {
		// Try to find user by email
		if providerUser.Email != "" {
			u, err = p.store.GetUserByEmail(goCtx, appID, strings.ToLower(providerUser.Email))
			if err != nil {
				// No existing user — create one
				u = &user.User{
					ID:            id.NewUserID(),
					AppID:         appID,
					EnvID:         envID,
					Email:         strings.ToLower(providerUser.Email),
					EmailVerified: true, // Social-authenticated emails are pre-verified by the provider.
					FirstName:     providerUser.FirstName,
					LastName:      providerUser.LastName,
					Image:         providerUser.AvatarURL,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				if createErr := p.store.CreateUser(goCtx, u); createErr != nil {
					return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", createErr))
				}
				if p.engine != nil {
					p.engine.EnsureDefaultRole(goCtx, appID, u.ID)
				}
			}
		} else {
			// No email from provider — create user without email
			u = &user.User{
				ID:        id.NewUserID(),
				AppID:     appID,
				EnvID:     envID,
				FirstName: providerUser.FirstName,
				LastName:  providerUser.LastName,
				Image:     providerUser.AvatarURL,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if createErr := p.store.CreateUser(goCtx, u); createErr != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", createErr))
			}
			if p.engine != nil {
				p.engine.EnsureDefaultRole(goCtx, appID, u.ID)
			}
		}

		// Create OAuth connection if we have a connection store
		if p.oauthStore != nil {
			conn := &OAuthConnection{
				ID:             id.NewOAuthConnectionID(),
				AppID:          appID,
				UserID:         u.ID,
				Provider:       req.Provider,
				ProviderUserID: providerUser.ProviderUserID,
				Email:          providerUser.Email,
				AccessToken:    token.AccessToken,
				RefreshToken:   token.RefreshToken,
				ExpiresAt:      token.Expiry,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if createErr := p.oauthStore.CreateOAuthConnection(goCtx, conn); createErr != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to store oauth connection: %w", createErr))
			}
		}
	}

	// Mint the session through Engine.IssueSession so the centralized
	// MFARequired gate fires for OAuth callbacks too. Falls back to a
	// direct mint if the engine isn't the concrete *authsome.Engine
	// (e.g. test wiring without a full engine).
	var sess *session.Session
	if eng, ok := p.engine.(*authsome.Engine); ok && eng != nil {
		result, issueErr := eng.IssueSession(goCtx, &authsome.IssueSessionRequest{
			User:       u,
			AppID:      appID,
			EnvID:      envID,
			AuthMethod: "social:" + req.Provider,
			IPAddress:  ctx.Request().RemoteAddr,
			UserAgent:  ctx.Request().UserAgent(),
		})
		if issueErr != nil {
			// *authsome.MFARequiredError implements forge's
			// StatusCode/ResponseBody so it renders as a 403 with the
			// ticket envelope; other errors fall through as 500.
			return nil, issueErr
		}
		sess = result.Session
	} else {
		// Fallback for tests that don't wire a full engine. Production
		// always reaches the IssueSession branch.
		sessCfg := account.SessionConfig{
			TokenTTL:        p.config.SessionTokenTTL,
			RefreshTokenTTL: p.config.SessionRefreshTTL,
		}
		if p.engine != nil {
			sessCfg = p.engine.SessionConfigForApp(goCtx, appID)
		}
		var newErr error
		sess, newErr = account.NewSession(appID, u.ID, sessCfg)
		if newErr != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to create session: %w", newErr))
		}
		sess.EnvID = envID
		if err := p.store.CreateSession(goCtx, sess); err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to store session: %w", err))
		}
	}

	isNewUser := u.CreatedAt.After(time.Now().Add(-5 * time.Second))
	eventType := "auth.social.signin"
	hookAction := hook.ActionSocialSignIn
	if isNewUser {
		eventType = "auth.social.signup"
		hookAction = hook.ActionSocialSignUp
	}
	p.relayEvent(ctx.Context(), eventType, "", map[string]string{"user_id": u.ID.String(), "provider": req.Provider})
	p.emitHook(ctx.Context(), hookAction, hook.ResourceUser, u.ID.String(), u.ID.String(), "")

	// Set httpOnly session cookie for browser-based flows. Cookie TTL
	// mirrors the session token's remaining lifetime.
	cookieTTL := int(time.Until(sess.ExpiresAt).Seconds())
	if cookieTTL < 0 {
		cookieTTL = 0
	}
	p.setSessionCookie(ctx, sess.Token, cookieTTL)

	// Browser-based OAuth callbacks arrive as GET redirects. Respond with a
	// small HTML page that closes the popup (or redirects to the stored
	// redirect URL for non-popup flows) so the parent window can pick up
	// the session cookie.
	if ctx.Request().Method == http.MethodGet {
		redirectTarget := stateInfo["redirect_url"]
		if redirectTarget == "" {
			redirectTarget = stateInfo["frontend_url"]
		}
		if redirectTarget == "" {
			redirectTarget = "/"
		}
		// The redirect target is interpolated into a JS string literal
		// (`window.location.href="<value>"`), so we must use JS-escaping
		// — html.EscapeString here would only stop </script> injection
		// and leaves backslash, quote, newline, and U+2028/U+2029
		// (which terminate JS string literals) unescaped. The sanitizer
		// already strips most dangerous bytes upstream, but defense in
		// depth: use template.JSEscapeString for the right context.
		escapedRedirect := template.JSEscapeString(redirectTarget)

		ctx.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
		ctx.Response().WriteHeader(http.StatusOK)
		_, _ = ctx.Response().Write([]byte(`<!DOCTYPE html><html><head><title>Login successful</title></head><body><script>` + //nolint:errcheck // best-effort HTML write
			// Always try to close the popup first. After cross-origin navigation
			// (e.g. through Google OAuth) window.opener may be null, so we attempt
			// window.close() unconditionally and fall back to a redirect after a
			// short delay to give the close a chance to fire.
			`try{window.close()}catch(e){}` +
			`setTimeout(function(){window.location.href="` + escapedRedirect + `"},300)` +
			`</script><p>Login successful. Redirecting&hellip;</p></body></html>`))
		return nil, nil
	}

	return &CallbackResponse{
		User:         u,
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt,
		Provider:     req.Provider,
		IsNewUser:    isNewUser,
		RedirectURL:  stateInfo["redirect_url"],
		FrontendURL:  stateInfo["frontend_url"],
	}, nil
}

// ──────────────────────────────────────────────────
// AuthMethodContributor / AuthMethodUnlinker
// ──────────────────────────────────────────────────

// ListUserAuthMethods implements plugin.AuthMethodContributor.
// It returns one entry per OAuth connection linked to the user.
func (p *Plugin) ListUserAuthMethods(ctx context.Context, userID id.UserID) ([]*plugin.AuthMethod, error) {
	if p.oauthStore == nil {
		return nil, nil
	}
	conns, err := p.oauthStore.GetOAuthConnectionsByUserID(ctx, userID)
	if err != nil {
		return nil, nil
	}
	methods := make([]*plugin.AuthMethod, 0, len(conns))
	for _, c := range conns {
		label := c.Provider
		if c.Email != "" {
			label = fmt.Sprintf("%s (%s)", c.Provider, c.Email)
		}
		methods = append(methods, &plugin.AuthMethod{
			Type:     "social:" + c.Provider,
			Provider: c.Provider,
			Label:    label,
			LinkedAt: c.CreatedAt.Format(time.RFC3339),
		})
	}
	return methods, nil
}

// CanUnlink implements plugin.AuthMethodUnlinker.
// It returns true if the given provider is one managed by this plugin.
func (p *Plugin) CanUnlink(ctx context.Context, _ id.UserID, provider string) bool {
	_, ok := p.resolveProvider(ctx, provider)
	return ok
}

// UnlinkAuthMethod implements plugin.AuthMethodUnlinker.
// It removes the OAuth connection for the given provider from the user's account.
func (p *Plugin) UnlinkAuthMethod(ctx context.Context, userID id.UserID, provider string) error {
	if p.oauthStore == nil {
		return fmt.Errorf("social: oauth store not available")
	}
	conns, err := p.oauthStore.GetOAuthConnectionsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("social: list connections: %w", err)
	}
	for _, c := range conns {
		if c.Provider == provider {
			return p.oauthStore.DeleteOAuthConnection(ctx, c.ID)
		}
	}
	return fmt.Errorf("social: no connection found for provider %q", provider)
}

// ──────────────────────────────────────────────────
// Cookie helpers
// ──────────────────────────────────────────────────
//
// (removed: resolveCookieSetting / resolveCookieSettingBool — replaced by
// authsome.SessionCookieTemplate which centralises cookie-attribute
// resolution and __Host- prefix handling across the engine, social, and
// dashboard auth pages.)

// setSessionCookie sets an httpOnly session cookie on the response.
// Resolves the full cookie configuration (name, domain, path, secure,
// http_only, same_site, and the __Host- prefix opt-in) via
// authsome.SessionCookieTemplate so the social plugin's cookie matches
// the engine's and dashboard's exactly.
func (p *Plugin) setSessionCookie(ctx forge.Context, token string, maxAge int) {
	goCtx := ctx.Context()
	mgr := p.engine.Settings()
	if mgr == nil {
		return
	}

	r := ctx.Request()
	isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	c := authsome.SessionCookieTemplate(goCtx, mgr, p.appID, isHTTPS)
	c.Value = token
	c.MaxAge = maxAge
	http.SetCookie(ctx.Response(), c)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// resolveCallbackURL returns the OAuth callback URL for a provider. If the
// provider already has a RedirectURL configured, it is returned unchanged.
// Otherwise the URL is constructed from:
//  1. Config.Domain (if set), e.g. "https://api.example.com"
//  2. The incoming request's scheme + host (fallback)
//
// The path includes the authsome base path (e.g. /api/identity/v1/social/{provider}/callback).
func (p *Plugin) resolveCallbackURL(r *http.Request, providerRedirectURL, providerName string) string {
	if providerRedirectURL != "" {
		return providerRedirectURL
	}

	base := strings.TrimRight(p.config.Domain, "/")
	if base == "" {
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}

	prefix := p.basePath
	if prefix == "" {
		prefix = "/authsome"
	}
	return base + prefix + "/v1/social/" + providerName + "/callback"
}

// callbackError handles a callback failure once the OAuth state is known. For
// browser-initiated GET callbacks it redirects the user back to the SPA with
// an `error` query parameter so the frontend can render an error UI; for
// non-browser callers it surfaces the error as a JSON 400. The fallback chain
// for the redirect target is: redirect_url → frontend_url → forge.BadRequest.
func (p *Plugin) callbackError(ctx forge.Context, stateInfo map[string]string, message string) (*CallbackResponse, error) {
	if ctx.Request().Method != http.MethodGet {
		return nil, forge.BadRequest(message)
	}
	target := stateInfo["redirect_url"]
	if target == "" {
		target = stateInfo["frontend_url"]
	}
	if target == "" {
		return nil, forge.BadRequest(message)
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, forge.BadRequest(message)
	}
	q := parsed.Query()
	q.Set("error", message)
	parsed.RawQuery = q.Encode()

	ctx.Response().Header().Set("Location", parsed.String())
	ctx.Response().WriteHeader(http.StatusFound)
	return nil, nil
}

// relayEvent sends a webhook event to EventRelay (nil-safe).
func (p *Plugin) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if p.relay == nil {
		return
	}
	_ = p.relay.Send(ctx, &bridge.WebhookEvent{ //nolint:errcheck // best-effort webhook
		Type:     eventType,
		TenantID: tenantID,
		Data:     data,
	})
}

// emitHook fires a global hook event (nil-safe).
func (p *Plugin) emitHook(ctx context.Context, action, resource, resourceID, actorID, tenant string) {
	if p.hooks == nil {
		return
	}
	p.hooks.Emit(ctx, &hook.Event{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
	})
}

// ──────────────────────────────────────────────────
// Admin: social provider management
// ──────────────────────────────────────────────────

// supportedProviderCatalog is the static list of providers this build
// can speak. Sourced from the SettingSocialProviders dropdown so the
// admin UI shows the same set the engine can actually wire.
var supportedProviderCatalog = []AdminCatalogProvider{
	{ID: "google", Name: "Google"},
	{ID: "github", Name: "GitHub"},
	{ID: "apple", Name: "Apple"},
	{ID: "microsoft", Name: "Microsoft"},
	{ID: "facebook", Name: "Facebook"},
	{ID: "linkedin", Name: "LinkedIn"},
	{ID: "discord", Name: "Discord"},
	{ID: "slack", Name: "Slack"},
	{ID: "twitter", Name: "Twitter"},
	{ID: "spotify", Name: "Spotify"},
	{ID: "twitch", Name: "Twitch"},
	{ID: "gitlab", Name: "GitLab"},
	{ID: "bitbucket", Name: "Bitbucket"},
	{ID: "dropbox", Name: "Dropbox"},
	{ID: "yahoo", Name: "Yahoo"},
	{ID: "amazon", Name: "Amazon"},
	{ID: "zoom", Name: "Zoom"},
	{ID: "pinterest", Name: "Pinterest"},
	{ID: "strava", Name: "Strava"},
	{ID: "patreon", Name: "Patreon"},
	{ID: "instagram", Name: "Instagram"},
	{ID: "line", Name: "Line"},
}

// AdminCatalogProvider is one entry in the supported-provider catalog.
type AdminCatalogProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AdminCatalogResponse is the response for GET /v1/admin/social/providers/catalog.
type AdminCatalogResponse struct {
	Providers []AdminCatalogProvider `json:"providers"`
}

// AdminProvider is the read-side shape for a configured provider.
// ClientSecret is masked (returns the literal "***" when set, empty
// when unset) so the admin UI can render "secret is configured" without
// echoing the value back.
type AdminProvider struct {
	Name         string   `json:"name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"`
	RedirectURL  string   `json:"redirect_url,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      bool     `json:"enabled"`
	HasSecret    bool     `json:"has_secret"`
}

// AdminListProvidersRequest binds the query for GET /v1/admin/social/providers.
type AdminListProvidersRequest struct {
	AppID string `query:"app_id" description:"App identifier; omit for global scope"`
}

// AdminListProvidersResponse is the listing response.
type AdminListProvidersResponse struct {
	Providers []AdminProvider `json:"providers"`
}

// AdminUpsertProviderRequest binds the path + body for PUT.
type AdminUpsertProviderRequest struct {
	Provider string `path:"provider" description:"Provider ID (google, github, ...)"`
	AppID    string `query:"app_id" description:"App identifier; omit for global scope"`

	ClientID     string   `json:"client_id"      description:"OAuth client ID"`
	ClientSecret string   `json:"client_secret"  description:"OAuth client secret. Pass empty string to leave the existing value unchanged."`
	RedirectURL  string   `json:"redirect_url,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

// AdminProviderResponse is the response after upsert.
type AdminProviderResponse struct {
	Provider AdminProvider `json:"provider"`
}

// AdminDeleteProviderRequest binds the path + query for DELETE.
type AdminDeleteProviderRequest struct {
	Provider string `path:"provider" description:"Provider ID"`
	AppID    string `query:"app_id"  description:"App identifier; omit for global scope"`
}

// AdminDeleteProviderResponse mirrors the StatusResponse shape.
type AdminDeleteProviderResponse struct {
	Status string `json:"status"`
}

// handleAdminListProviders returns the configured providers at the
// resolved scope. With ?app_id, returns the merged view (global +
// app overrides) so the UI shows what the public client-config
// endpoint would return.
func (p *Plugin) handleAdminListProviders(ctx forge.Context, req *AdminListProvidersRequest) (*AdminListProvidersResponse, error) {
	if p.settingsMgr == nil {
		return nil, forge.InternalError(fmt.Errorf("social: settings manager not wired"))
	}
	opts := settings.ResolveOpts{}
	if v := strings.TrimSpace(req.AppID); v != "" {
		opts.AppID = v
	}
	providers, err := settings.Get(ctx.Context(), p.settingsMgr, SettingSocialProviders, opts)
	if err != nil {
		// Cascade returns the default ([]) when no override exists; a
		// real error means the store is broken.
		return nil, forge.InternalError(fmt.Errorf("social: read providers: %w", err))
	}
	out := make([]AdminProvider, 0, len(providers))
	for _, prov := range providers {
		out = append(out, maskProvider(prov))
	}
	return &AdminListProvidersResponse{Providers: out}, nil
}

// handleAdminCatalog returns the static provider catalog.
func (p *Plugin) handleAdminCatalog(_ forge.Context, _ *struct{}) (*AdminCatalogResponse, error) {
	out := make([]AdminCatalogProvider, len(supportedProviderCatalog))
	copy(out, supportedProviderCatalog)
	return &AdminCatalogResponse{Providers: out}, nil
}

// handleAdminUpsertProvider replaces the entry for the named provider
// at the requested scope. When ?app_id is supplied, the provider list
// is read at App scope, mutated, and written back at App scope —
// global is left untouched. Empty client_secret leaves the existing
// stored secret unchanged so the UI can re-save without echoing it.
func (p *Plugin) handleAdminUpsertProvider(ctx forge.Context, req *AdminUpsertProviderRequest) (*AdminProviderResponse, error) {
	if p.settingsMgr == nil {
		return nil, forge.InternalError(fmt.Errorf("social: settings manager not wired"))
	}
	name := strings.ToLower(strings.TrimSpace(req.Provider))
	if name == "" {
		return nil, forge.BadRequest("provider is required")
	}
	if !catalogContains(name) {
		return nil, forge.BadRequest("unsupported provider; call GET /v1/admin/social/providers/catalog for the list")
	}
	if strings.TrimSpace(req.ClientID) == "" {
		return nil, forge.BadRequest("client_id is required")
	}

	scope, scopeID := scopeFor(req.AppID)
	current := p.readScope(ctx.Context(), scope, scopeID)

	// Preserve the existing secret when caller passes empty string.
	existingSecret := ""
	for _, prov := range current {
		if strings.EqualFold(prov.Name, name) {
			existingSecret = prov.ClientSecret
			break
		}
	}
	secret := req.ClientSecret
	if strings.TrimSpace(secret) == "" {
		secret = existingSecret
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	updated := ProviderSetting{
		Name:         name,
		ClientID:     req.ClientID,
		ClientSecret: secret,
		RedirectURL:  req.RedirectURL,
		Scopes:       req.Scopes,
		Enabled:      enabled,
	}

	out := replaceOrAppend(current, updated)
	if err := p.writeScope(ctx.Context(), scope, scopeID, out); err != nil {
		return nil, forge.InternalError(err)
	}
	return &AdminProviderResponse{Provider: maskProvider(updated)}, nil
}

// handleAdminDeleteProvider removes one provider entry from the list
// at the requested scope. Idempotent — no error when the entry is
// missing.
func (p *Plugin) handleAdminDeleteProvider(ctx forge.Context, req *AdminDeleteProviderRequest) (*AdminDeleteProviderResponse, error) {
	if p.settingsMgr == nil {
		return nil, forge.InternalError(fmt.Errorf("social: settings manager not wired"))
	}
	name := strings.ToLower(strings.TrimSpace(req.Provider))
	if name == "" {
		return nil, forge.BadRequest("provider is required")
	}
	scope, scopeID := scopeFor(req.AppID)
	current := p.readScope(ctx.Context(), scope, scopeID)
	out := make([]ProviderSetting, 0, len(current))
	for _, prov := range current {
		if strings.EqualFold(prov.Name, name) {
			continue
		}
		out = append(out, prov)
	}
	if err := p.writeScope(ctx.Context(), scope, scopeID, out); err != nil {
		return nil, forge.InternalError(err)
	}
	return &AdminDeleteProviderResponse{Status: "deleted"}, nil
}

// readScope reads ProviderSetting list at exactly the named scope.
// Returns the empty slice when nothing is stored — does NOT cascade
// up so writes only touch the scope the caller asked for.
func (p *Plugin) readScope(ctx context.Context, scope settings.Scope, scopeID string) []ProviderSetting {
	if p.settingsMgr == nil || p.settingsMgr.Store() == nil {
		return nil
	}
	s, err := p.settingsMgr.Store().GetSetting(ctx, SettingSocialProviders.Def.Key, scope, scopeID)
	if err != nil || s == nil || len(s.Value) == 0 {
		return nil
	}
	var out []ProviderSetting
	if err := json.Unmarshal(s.Value, &out); err != nil {
		return nil
	}
	return out
}

// writeScope serialises the provider list and writes it at the named scope.
func (p *Plugin) writeScope(ctx context.Context, scope settings.Scope, scopeID string, providers []ProviderSetting) error {
	if p.settingsMgr == nil {
		return fmt.Errorf("social: settings manager not wired")
	}
	raw, err := json.Marshal(providers)
	if err != nil {
		return err
	}
	appID := ""
	if scope == settings.ScopeApp {
		appID = scopeID
	}
	return p.settingsMgr.Set(ctx, SettingSocialProviders.Def.Key, raw, scope, scopeID, appID, "", "admin")
}

func scopeFor(appID string) (settings.Scope, string) {
	if v := strings.TrimSpace(appID); v != "" {
		return settings.ScopeApp, v
	}
	return settings.ScopeGlobal, ""
}

func catalogContains(name string) bool {
	for _, p := range supportedProviderCatalog {
		if p.ID == name {
			return true
		}
	}
	return false
}

func replaceOrAppend(list []ProviderSetting, entry ProviderSetting) []ProviderSetting {
	for i, prov := range list {
		if strings.EqualFold(prov.Name, entry.Name) {
			out := make([]ProviderSetting, len(list))
			copy(out, list)
			out[i] = entry
			return out
		}
	}
	return append(list, entry)
}

func maskProvider(p ProviderSetting) AdminProvider {
	out := AdminProvider{
		Name:        p.Name,
		ClientID:    p.ClientID,
		RedirectURL: p.RedirectURL,
		Scopes:      p.Scopes,
		Enabled:     p.Enabled,
		HasSecret:   strings.TrimSpace(p.ClientSecret) != "",
	}
	if out.HasSecret {
		out.ClientSecret = "***"
	}
	return out
}
