package social

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
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

	// SessionTokenTTL is the lifetime of sessions created via social sign-in (default: 1 hour).
	SessionTokenTTL time.Duration

	// SessionRefreshTTL is the lifetime of refresh tokens (default: 30 days).
	SessionRefreshTTL time.Duration
}

// sessionConfigResolver resolves per-app session configuration.
type sessionConfigResolver interface {
	SessionConfigForApp(ctx context.Context, appID id.AppID) account.SessionConfig
}

// Plugin is the social OAuth authentication plugin.
type Plugin struct {
	config        Config
	providers     map[string]Provider
	store         store.Store // Core authsome store (for users/sessions)
	oauthStore    Store       // OAuth-specific store (for connections)
	appID         string
	sessionConfig sessionConfigResolver
	settingsMgr   *settings.Manager

	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	hooks       *hook.Bus
	logger      log.Logger
	ceremonies  ceremony.Store
	roleEnsurer roleEnsurer
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "social", SettingSessionTokenTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "social", SettingSessionRefreshTTLSeconds); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "social", SettingSocialProviders)
}

// roleEnsurer assigns a default Warden role to a newly created user.
type roleEnsurer interface {
	EnsureDefaultRole(ctx context.Context, appID id.AppID, userID id.UserID)
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
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type storeGetter interface {
		Store() store.Store
	}
	if sg, ok := engine.(storeGetter); ok {
		p.store = sg.Store()
	}

	type chronicleGetter interface {
		Chronicle() bridge.Chronicle
	}
	if cg, ok := engine.(chronicleGetter); ok {
		p.chronicle = cg.Chronicle()
	}

	type relayGetter interface {
		Relay() bridge.EventRelay
	}
	if rg, ok := engine.(relayGetter); ok {
		p.relay = rg.Relay()
	}

	type hooksGetter interface {
		Hooks() *hook.Bus
	}
	if hg, ok := engine.(hooksGetter); ok {
		p.hooks = hg.Hooks()
	}

	type loggerGetter interface {
		Logger() log.Logger
	}
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}

	type ceremonyGetter interface {
		CeremonyStore() ceremony.Store
	}
	if cg, ok := engine.(ceremonyGetter); ok {
		p.ceremonies = cg.CeremonyStore()
	}
	if p.ceremonies == nil {
		p.ceremonies = ceremony.NewMemory()
	}

	if scr, ok := engine.(sessionConfigResolver); ok {
		p.sessionConfig = scr
	}

	type settingsGetter interface {
		Settings() *settings.Manager
	}
	if sg, ok := engine.(settingsGetter); ok {
		p.settingsMgr = sg.Settings()
	}

	if re, ok := engine.(roleEnsurer); ok {
		p.roleEnsurer = re
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
func (p *Plugin) RegisterRoutes(r any) error {
	router, ok := r.(forge.Router)
	if !ok {
		return fmt.Errorf("social: expected forge.Router, got %T", r)
	}

	g := router.Group("/v1/auth/social", forge.WithGroupTags("Social OAuth"))

	if err := g.POST("/:provider", p.handleStart,
		forge.WithSummary("Start OAuth flow"),
		forge.WithOperationID("startOAuth"),
		forge.WithResponseSchema(http.StatusOK, "OAuth authorization URL", StartResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.GET("/:provider/callback", p.handleCallback,
		forge.WithSummary("OAuth callback"),
		forge.WithOperationID("oauthCallback"),
		forge.WithResponseSchema(http.StatusOK, "Authentication result", CallbackResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// StartRequest contains the path parameter for starting OAuth.
type StartRequest struct {
	Provider string `path:"provider"`
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
type CallbackResponse struct {
	User         any    `json:"user"`
	SessionToken string `json:"session_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    any    `json:"expires_at"`
	Provider     string `json:"provider"`
	IsNewUser    bool   `json:"is_new_user"`
}

// ──────────────────────────────────────────────────
// Forge Handlers
// ──────────────────────────────────────────────────

// handleStart initiates the OAuth flow by returning the authorization URL.
func (p *Plugin) handleStart(ctx forge.Context, req *StartRequest) (*StartResponse, error) {
	provider, ok := p.resolveProvider(ctx.Context(), req.Provider)
	if !ok {
		return nil, forge.BadRequest(fmt.Sprintf("unsupported provider: %s", req.Provider))
	}

	state, err := generateState()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate state: %w", err))
	}

	// Store the state for CSRF protection
	stateData, _ := json.Marshal(map[string]string{"provider": req.Provider})             //nolint:errcheck // marshaling known types
	_ = p.ceremonies.Set(ctx.Context(), "social:state:"+state, stateData, 10*time.Minute) //nolint:errcheck // best-effort cache

	cfg := provider.OAuth2Config()
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return &StartResponse{AuthURL: authURL}, nil
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

	stateData, err := p.ceremonies.Get(ctx.Context(), "social:state:"+req.State)
	if err != nil {
		return nil, forge.BadRequest("invalid state parameter")
	}
	_ = p.ceremonies.Delete(ctx.Context(), "social:state:"+req.State) //nolint:errcheck // best-effort cleanup
	var stateInfo map[string]string
	if unmarshalErr := json.Unmarshal(stateData, &stateInfo); unmarshalErr != nil || stateInfo["provider"] != req.Provider {
		return nil, forge.BadRequest("invalid state parameter")
	}

	// Check for error from provider
	if req.Error != "" {
		return nil, forge.BadRequest(fmt.Sprintf("provider error: %s", req.Error))
	}

	// Exchange code for token
	if req.Code == "" {
		return nil, forge.BadRequest("missing code parameter")
	}

	cfg := provider.OAuth2Config()
	token, err := cfg.Exchange(ctx.Context(), req.Code)
	if err != nil {
		return nil, forge.BadRequest("failed to exchange code")
	}

	// Fetch user profile from provider
	providerUser, err := provider.FetchUser(ctx.Context(), token)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to fetch user from provider: %w", err))
	}

	appIDStr := p.appID
	appID, err := id.ParseAppID(appIDStr)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("invalid app_id configuration: %w", err))
	}

	goCtx := ctx.Context()

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
					ID:        id.NewUserID(),
					AppID:     appID,
					Email:     strings.ToLower(providerUser.Email),
					FirstName: providerUser.FirstName,
					LastName:  providerUser.LastName,
					Image:     providerUser.AvatarURL,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				if createErr := p.store.CreateUser(goCtx, u); createErr != nil {
					return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", createErr))
				}
				if p.roleEnsurer != nil {
					p.roleEnsurer.EnsureDefaultRole(goCtx, appID, u.ID)
				}
			}
		} else {
			// No email from provider — create user without email
			u = &user.User{
				ID:        id.NewUserID(),
				AppID:     appID,
				FirstName: providerUser.FirstName,
				LastName:  providerUser.LastName,
				Image:     providerUser.AvatarURL,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if createErr := p.store.CreateUser(goCtx, u); createErr != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", createErr))
			}
			if p.roleEnsurer != nil {
				p.roleEnsurer.EnsureDefaultRole(goCtx, appID, u.ID)
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

	// Resolve per-app session config, falling back to plugin config.
	sessCfg := account.SessionConfig{
		TokenTTL:        p.config.SessionTokenTTL,
		RefreshTokenTTL: p.config.SessionRefreshTTL,
	}
	if p.sessionConfig != nil {
		sessCfg = p.sessionConfig.SessionConfigForApp(goCtx, appID)
	}
	sess, err := account.NewSession(appID, u.ID, sessCfg)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create session: %w", err))
	}

	if err := p.store.CreateSession(goCtx, sess); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to store session: %w", err))
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

	// Set httpOnly session cookie for browser-based flows.
	p.setSessionCookie(ctx, sess.Token, int(sessCfg.TokenTTL.Seconds()))

	return &CallbackResponse{
		User:         u,
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt,
		Provider:     req.Provider,
		IsNewUser:    isNewUser,
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

// setSessionCookie sets an httpOnly session cookie on the response.
// It uses the default cookie name "authsome_session_token" and auto-detects
// HTTPS for the Secure flag. For advanced cookie configuration (custom name,
// domain, sameSite), the core API handlers use the dynamic settings system.
func (p *Plugin) setSessionCookie(ctx forge.Context, token string, maxAge int) {
	r := ctx.Request()
	isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(ctx.Response(), &http.Cookie{
		Name:     "authsome_session_token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteLaxMode,
	})
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
