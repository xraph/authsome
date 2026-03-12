package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
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
)

// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.RouteProvider     = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
	_ plugin.SettingsProvider  = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingSessionTokenTTLSeconds controls the session token lifetime for SSO sign-in.
	SettingSessionTokenTTLSeconds = settings.Define("sso.session_token_ttl_seconds", 3600,
		settings.WithDisplayName("Session Token TTL (seconds)"),
		settings.WithDescription("Lifetime of sessions created via SSO sign-in in seconds"),
		settings.WithCategory("SSO"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(300), Max: intPtr(86400)}),
		settings.WithHelpText("How long sessions created via SSO remain valid. Default: 3600 (1 hour)"),
		settings.WithOrder(10),
	)

	// SettingSessionRefreshTTLSeconds controls the refresh token lifetime for SSO sessions.
	SettingSessionRefreshTTLSeconds = settings.Define("sso.session_refresh_ttl_seconds", 2592000,
		settings.WithDisplayName("Refresh Token TTL (seconds)"),
		settings.WithDescription("Lifetime of refresh tokens for SSO sessions in seconds"),
		settings.WithCategory("SSO"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(3600), Max: intPtr(7776000)}),
		settings.WithHelpText("How long refresh tokens remain valid. Default: 2592000 (30 days)"),
		settings.WithOrder(20),
	)
)

func intPtr(v int) *int { return &v }

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "sso", SettingSessionTokenTTLSeconds); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "sso", SettingSessionRefreshTTLSeconds)
}

// Config configures the SSO plugin.
type Config struct {
	// Providers is the list of configured SSO providers.
	Providers []Provider

	// SessionTokenTTL is the lifetime of sessions created via SSO sign-in (default: 1 hour).
	SessionTokenTTL time.Duration

	// SessionRefreshTTL is the lifetime of refresh tokens (default: 30 days).
	SessionRefreshTTL time.Duration
}

// Plugin is the SSO authentication plugin.
// sessionConfigResolver resolves per-app session configuration.
type sessionConfigResolver interface {
	SessionConfigForApp(ctx context.Context, appID id.AppID) account.SessionConfig
}

type Plugin struct {
	config        Config
	providers     map[string]Provider
	store         store.Store // Core authsome store (for users/sessions)
	ssoStore      Store       // SSO-specific store (for connections)
	appID         string
	sessionConfig sessionConfigResolver

	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	hooks       *hook.Bus
	logger      log.Logger
	ceremonies  ceremony.Store
	roleEnsurer roleEnsurer
}

// roleEnsurer assigns a default Warden role to a newly created user.
type roleEnsurer interface {
	EnsureDefaultRole(ctx context.Context, appID id.AppID, userID id.UserID)
}

// New creates a new SSO plugin.
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
func (p *Plugin) Name() string { return "sso" }

// Connections returns the list of configured SSO connection names for client config.
func (p *Plugin) Connections() []string {
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	return names
}

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

// SetSSOStore sets the SSO connection store.
func (p *Plugin) SetSSOStore(s Store) {
	p.ssoStore = s
}

// SetAppID sets the default app ID.
func (p *Plugin) SetAppID(appID string) {
	p.appID = appID
}

// Providers returns the list of configured provider names.
func (p *Plugin) ProviderNames() []string {
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	return names
}

// resolveProvider looks up a provider by name — first from code-configured
// providers, then from database-managed SSO connections.
func (p *Plugin) resolveProvider(ctx context.Context, name string) (Provider, bool) {
	// Check code-configured providers first.
	if prov, ok := p.providers[name]; ok {
		return prov, true
	}

	// Fall back to DB-managed connections.
	if p.ssoStore == nil {
		return nil, false
	}
	appID, err := id.ParseAppID(p.appID)
	if err != nil {
		return nil, false
	}
	conn, err := p.ssoStore.GetSSOConnectionByProvider(ctx, appID, name)
	if err != nil || conn == nil || !conn.Active {
		return nil, false
	}
	return p.connectionToProvider(conn), true
}

// connectionToProvider creates a Provider from a stored SSOConnection.
func (p *Plugin) connectionToProvider(conn *SSOConnection) Provider {
	switch conn.Protocol {
	case "oidc":
		return NewOIDCProvider(OIDCConfig{
			Name:         conn.Provider,
			Issuer:       conn.Issuer,
			ClientID:     conn.ClientID,
			ClientSecret: conn.ClientSecret,
		})
	case "saml":
		return NewSAMLProvider(SAMLConfig{
			Name:        conn.Provider,
			MetadataURL: conn.MetadataURL,
		})
	default:
		return nil
	}
}

// RegisterRoutes registers SSO HTTP endpoints on a forge.Router.
func (p *Plugin) RegisterRoutes(r any) error {
	router, ok := r.(forge.Router)
	if !ok {
		return fmt.Errorf("sso: expected forge.Router, got %T", r)
	}

	g := router.Group("/v1/auth/sso", forge.WithGroupTags("SSO"))

	if err := g.POST("/:provider/login", p.handleLogin,
		forge.WithSummary("Start SSO login flow"),
		forge.WithOperationID("startSSOLogin"),
		forge.WithResponseSchema(http.StatusOK, "SSO login URL", LoginResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/:provider/callback", p.handleCallback,
		forge.WithSummary("SSO callback (OIDC)"),
		forge.WithOperationID("ssoCallback"),
		forge.WithResponseSchema(http.StatusOK, "Authentication result", CallbackResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.POST("/:provider/acs", p.handleACS,
		forge.WithSummary("SSO SAML ACS endpoint"),
		forge.WithOperationID("ssoACS"),
		forge.WithResponseSchema(http.StatusOK, "Authentication result", CallbackResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// LoginRequest contains the path parameter for starting SSO.
type LoginRequest struct {
	Provider string `path:"provider"`
}

// LoginResponse is returned when the SSO flow is initiated.
type LoginResponse struct {
	LoginURL string `json:"login_url"`
	State    string `json:"state"`
}

// CallbackRequest contains the parameters for the OIDC callback.
type CallbackRequest struct {
	Provider string `path:"provider"`
	State    string `json:"state" query:"state,omitempty"`
	Code     string `json:"code" query:"code,omitempty"`
	Error    string `json:"error,omitempty" query:"error,omitempty"`
}

// ACSRequest contains the SAML Assertion Consumer Service parameters.
type ACSRequest struct {
	Provider     string `path:"provider"`
	SAMLResponse string `json:"SAMLResponse" form:"SAMLResponse"`
	RelayState   string `json:"RelayState" form:"RelayState"`
}

// CallbackResponse is returned on successful SSO authentication.
type CallbackResponse struct {
	User         any    `json:"user"`
	SessionToken string `json:"session_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    any    `json:"expires_at"`
	Provider     string `json:"provider"`
	IsNewUser    bool   `json:"is_new_user"`
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

// handleLogin initiates the SSO flow by returning the IdP login URL.
func (p *Plugin) handleLogin(ctx forge.Context, req *LoginRequest) (*LoginResponse, error) {
	provider, ok := p.resolveProvider(ctx.Context(), req.Provider)
	if !ok {
		return nil, forge.BadRequest(fmt.Sprintf("unsupported SSO provider: %s", req.Provider))
	}

	state, err := generateState()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate state: %w", err))
	}

	// Store the state for CSRF protection
	stateData, _ := json.Marshal(map[string]string{"provider": req.Provider})
	_ = p.ceremonies.Set(ctx.Context(), "sso:state:"+state, stateData, 10*time.Minute)

	loginURL, err := provider.LoginURL(state)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to get login URL: %w", err))
	}

	return &LoginResponse{
		LoginURL: loginURL,
		State:    state,
	}, nil
}

// handleCallback processes the OIDC callback.
func (p *Plugin) handleCallback(ctx forge.Context, req *CallbackRequest) (*CallbackResponse, error) {
	provider, ok := p.resolveProvider(ctx.Context(), req.Provider)
	if !ok {
		return nil, forge.BadRequest(fmt.Sprintf("unsupported SSO provider: %s", req.Provider))
	}

	if req.State == "" {
		return nil, forge.BadRequest("missing state parameter")
	}

	if err := p.validateState(ctx.Context(), req.State, req.Provider); err != nil {
		return nil, forge.BadRequest(err.Error())
	}

	if req.Error != "" {
		return nil, forge.BadRequest(fmt.Sprintf("provider error: %s", req.Error))
	}

	if req.Code == "" {
		return nil, forge.BadRequest("missing code parameter")
	}

	params := map[string]string{
		"code":  req.Code,
		"state": req.State,
	}

	return p.authenticateUser(ctx, provider, params)
}

// handleACS processes the SAML Assertion Consumer Service callback.
func (p *Plugin) handleACS(ctx forge.Context, req *ACSRequest) (*CallbackResponse, error) {
	provider, ok := p.resolveProvider(ctx.Context(), req.Provider)
	if !ok {
		return nil, forge.BadRequest(fmt.Sprintf("unsupported SSO provider: %s", req.Provider))
	}

	if req.SAMLResponse == "" {
		return nil, forge.BadRequest("missing SAMLResponse")
	}

	if req.RelayState != "" {
		if err := p.validateState(ctx.Context(), req.RelayState, req.Provider); err != nil {
			return nil, forge.BadRequest(err.Error())
		}
	}

	params := map[string]string{
		"SAMLResponse": req.SAMLResponse,
		"RelayState":   req.RelayState,
	}

	return p.authenticateUser(ctx, provider, params)
}

// authenticateUser processes an SSO identity and creates/links a user.
func (p *Plugin) authenticateUser(ctx forge.Context, provider Provider, params map[string]string) (*CallbackResponse, error) {
	ssoUser, err := provider.HandleCallback(ctx.Context(), params)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("sso: callback failed: %w", err))
	}

	appIDStr := p.appID
	appID, err := id.ParseAppID(appIDStr)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("invalid app_id configuration: %w", err))
	}

	goCtx := ctx.Context()

	// Find or create user by email.
	var u *user.User
	isNew := false

	if ssoUser.Email != "" {
		u, err = p.store.GetUserByEmail(goCtx, appID, strings.ToLower(ssoUser.Email))
		if err != nil {
			// No existing user -- create one.
			u = &user.User{
				ID:            id.NewUserID(),
				AppID:         appID,
				Email:         strings.ToLower(ssoUser.Email),
				EmailVerified: true, // SSO-authenticated emails are verified
				FirstName:     ssoUser.FirstName,
				LastName:      ssoUser.LastName,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			if err := p.store.CreateUser(goCtx, u); err != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", err))
			}
			if p.roleEnsurer != nil {
				p.roleEnsurer.EnsureDefaultRole(goCtx, appID, u.ID)
			}
			isNew = true
		}
	} else {
		return nil, forge.BadRequest("SSO provider did not return an email address")
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

	eventType := "auth.sso.signin"
	hookAction := hook.ActionSSOSignIn
	if isNew {
		eventType = "auth.sso.signup"
		hookAction = hook.ActionSSOSignUp
	}
	p.relayEvent(ctx.Context(), eventType, "", map[string]string{"user_id": u.ID.String(), "provider": provider.Name()})
	p.emitHook(ctx.Context(), hookAction, hook.ResourceUser, u.ID.String(), u.ID.String(), "")

	return &CallbackResponse{
		User:         u,
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt,
		Provider:     provider.Name(),
		IsNewUser:    isNew,
	}, nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func (p *Plugin) validateState(ctx context.Context, state, providerName string) error {
	stateData, err := p.ceremonies.Get(ctx, "sso:state:"+state)
	if err != nil {
		return fmt.Errorf("invalid state parameter")
	}
	_ = p.ceremonies.Delete(ctx, "sso:state:"+state)
	var stateInfo map[string]string
	if err := json.Unmarshal(stateData, &stateInfo); err != nil || stateInfo["provider"] != providerName {
		return fmt.Errorf("invalid state parameter")
	}
	return nil
}

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
		Category:   "auth",
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
