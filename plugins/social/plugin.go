package social

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	log "github.com/xraph/go-utils/log"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
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
)

// Config configures the social OAuth plugin.
type Config struct {
	// Providers is the list of enabled OAuth providers.
	Providers []Provider

	// SessionTokenTTL is the lifetime of sessions created via social sign-in (default: 1 hour).
	SessionTokenTTL time.Duration

	// SessionRefreshTTL is the lifetime of refresh tokens (default: 30 days).
	SessionRefreshTTL time.Duration
}

// Plugin is the social OAuth authentication plugin.
// sessionConfigResolver resolves per-app session configuration.
type sessionConfigResolver interface {
	SessionConfigForApp(ctx context.Context, appID id.AppID) account.SessionConfig
}

type Plugin struct {
	config        Config
	providers     map[string]Provider
	store         store.Store // Core authsome store (for users/sessions)
	oauthStore    Store       // OAuth-specific store (for connections)
	appID         string
	sessionConfig sessionConfigResolver

	chronicle  bridge.Chronicle
	relay      bridge.EventRelay
	hooks      *hook.Bus
	logger     log.Logger
	ceremonies ceremony.Store
}

// New creates a new social OAuth plugin.
func New(cfg Config) *Plugin {
	if cfg.SessionTokenTTL == 0 {
		cfg.SessionTokenTTL = 1 * time.Hour
	}
	if cfg.SessionRefreshTTL == 0 {
		cfg.SessionRefreshTTL = 30 * 24 * time.Hour
	}

	providers := make(map[string]Provider, len(cfg.Providers))
	for _, p := range cfg.Providers {
		providers[p.Name()] = p
	}

	return &Plugin{
		config:     cfg,
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

// Providers returns the list of configured provider names.
func (p *Plugin) Providers() []string {
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	return names
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
	provider, ok := p.providers[req.Provider]
	if !ok {
		return nil, forge.BadRequest(fmt.Sprintf("unsupported provider: %s", req.Provider))
	}

	state, err := generateState()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate state: %w", err))
	}

	// Store the state for CSRF protection
	stateData, _ := json.Marshal(map[string]string{"provider": req.Provider})
	_ = p.ceremonies.Set(ctx.Context(), "social:state:"+state, stateData, 10*time.Minute)

	cfg := provider.OAuth2Config()
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return &StartResponse{AuthURL: authURL}, nil
}

// handleCallback processes the OAuth callback, exchanges the code for a token,
// fetches the user profile, and either links to an existing user or creates a new one.
func (p *Plugin) handleCallback(ctx forge.Context, req *CallbackRequest) (*CallbackResponse, error) {
	provider, ok := p.providers[req.Provider]
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
	_ = p.ceremonies.Delete(ctx.Context(), "social:state:"+req.State)
	var stateInfo map[string]string
	if err := json.Unmarshal(stateData, &stateInfo); err != nil || stateInfo["provider"] != req.Provider {
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
		conn, err := p.oauthStore.GetOAuthConnection(goCtx, req.Provider, providerUser.ProviderUserID)
		if err == nil {
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
				if err := p.store.CreateUser(goCtx, u); err != nil {
					return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", err))
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
			if err := p.store.CreateUser(goCtx, u); err != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", err))
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
			if err := p.oauthStore.CreateOAuthConnection(goCtx, conn); err != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to store oauth connection: %w", err))
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
func (p *Plugin) CanUnlink(_ context.Context, _ id.UserID, provider string) bool {
	_, ok := p.providers[provider]
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
// Helpers
// ──────────────────────────────────────────────────

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// audit records an audit event via Chronicle (nil-safe).
func (p *Plugin) audit(ctx context.Context, action, resource, resourceID, actorID, tenant, outcome string) {
	if p.chronicle == nil {
		return
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
		Outcome:    outcome,
		Severity:   bridge.SeverityInfo,
	})
}

// relayEvent sends a webhook event to EventRelay (nil-safe).
func (p *Plugin) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if p.relay == nil {
		return
	}
	_ = p.relay.Send(ctx, &bridge.WebhookEvent{
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
