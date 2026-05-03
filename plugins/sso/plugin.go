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

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
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
type Plugin struct {
	config    Config
	providers map[string]Provider
	store     store.Store // Core authsome store (for users/sessions)
	ssoStore  Store       // SSO-specific store (for connections)
	appID     string
	engine    plugin.Engine

	chronicle  bridge.Chronicle
	relay      bridge.EventRelay
	hooks      *hook.Bus
	logger     log.Logger
	ceremonies ceremony.Store
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "sso", SettingSessionTokenTTLSeconds); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "sso", SettingSessionRefreshTTLSeconds)
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
	conn, err := p.ssoStore.GetConnectionByProvider(ctx, appID, name)
	if err != nil || conn == nil || !conn.Active {
		return nil, false
	}
	return p.connectionToProvider(conn), true
}

// connectionToProvider creates a Provider from a stored Connection.
func (p *Plugin) connectionToProvider(conn *Connection) Provider {
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
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	g := router.Group("/v1/sso", forge.WithGroupTags("SSO"))

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

	if err := g.POST("/:provider/acs", p.handleACS,
		forge.WithSummary("SSO SAML ACS endpoint"),
		forge.WithOperationID("ssoACS"),
		forge.WithResponseSchema(http.StatusOK, "Authentication result", CallbackResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Admin: create SSO connection scoped to a target App. Used by
	// platform-admin clients (e.g. TwinOS studio) to register an
	// upstream IdP per workspace App at create time. Caller must
	// authenticate with a platform-admin API key.
	admin := router.Group("/v1/admin/sso", forge.WithGroupTags("SSO Admin"))
	return admin.POST("/connections", p.handleAdminCreateConnection,
		forge.WithSummary("Create SSO connection (admin)"),
		forge.WithDescription("Registers an OIDC or SAML SSO connection on a target App. Used by platform-admin clients to provision per-tenant IdPs."),
		forge.WithOperationID("ssoAdminCreateConnection"),
		forge.WithRequestSchema(AdminCreateConnectionRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Connection created", AdminCreateConnectionResponse{}),
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
	stateData, _ := json.Marshal(map[string]string{"provider": req.Provider})          //nolint:errcheck // best-effort cache
	_ = p.ceremonies.Set(ctx.Context(), "sso:state:"+state, stateData, 10*time.Minute) //nolint:errcheck // best-effort cache

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
			// No existing user -- create one. Prefer the upstream
			// IdP's user_id (sub claim) as the local user_id when
			// it parses as a valid Authsome UserID. This makes the
			// federated user's identity stable across Apps: if a
			// user authenticates from upstream App `studio` (where
			// they have user_id = ausr_X) into a workspace App via
			// federation, the local user_id is also ausr_X.
			//
			// Stable-across-Apps identity is what makes Warden
			// assignments + introspect lookups agree. Without this,
			// the saga would assign roles by the upstream user_id
			// while the workspace App's introspect returns a fresh
			// local user_id — guaranteed mismatch on every request.
			//
			// Falls back to a fresh local id when the upstream sub
			// doesn't parse (non-Authsome IdPs like Google / GitHub).
			localID := id.NewUserID()
			if ssoUser.ProviderUserID != "" {
				if parsed, parseErr := id.ParseUserID(ssoUser.ProviderUserID); parseErr == nil {
					localID = parsed
				}
			}
			u = &user.User{
				ID:            localID,
				AppID:         appID,
				Email:         strings.ToLower(ssoUser.Email),
				EmailVerified: true, // SSO-authenticated emails are verified
				FirstName:     ssoUser.FirstName,
				LastName:      ssoUser.LastName,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			if createErr := p.store.CreateUser(goCtx, u); createErr != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", createErr))
			}
			if p.engine != nil {
				p.engine.EnsureDefaultRole(goCtx, appID, u.ID)
			}
			isNew = true
		}
	} else {
		return nil, forge.BadRequest("SSO provider did not return an email address")
	}

	// Mint the session through Engine.IssueSession so the centralized
	// MFARequired gate fires for SAML/OIDC callbacks too.
	var sess *session.Session
	if eng, ok := p.engine.(*authsome.Engine); ok && eng != nil {
		result, issueErr := eng.IssueSession(goCtx, &authsome.IssueSessionRequest{
			User:       u,
			AppID:      appID,
			AuthMethod: "sso:" + provider.Name(),
			IPAddress:  ctx.Request().RemoteAddr,
			UserAgent:  ctx.Request().UserAgent(),
		})
		if issueErr != nil {
			return nil, issueErr
		}
		sess = result.Session
	} else {
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
		if storeErr := p.store.CreateSession(goCtx, sess); storeErr != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to store session: %w", storeErr))
		}
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
	_ = p.ceremonies.Delete(ctx, "sso:state:"+state) //nolint:errcheck // best-effort cleanup
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
// Admin endpoints
// ──────────────────────────────────────────────────

// AdminCreateConnectionRequest is the body for
// POST /v1/admin/sso/connections. Caller specifies the target App
// + the IdP details. Domain is required and must be unique within
// the App so the dispatch path (`/v1/sso/:provider/login`) can
// resolve the right connection.
type AdminCreateConnectionRequest struct {
	AppID        string `json:"app_id" description:"Target Application ID"`
	OrgID        string `json:"org_id,omitempty" description:"Optional Org scope inside the App"`
	Provider     string `json:"provider" description:"Stable name for this IdP (e.g. 'studio', 'okta')"`
	Protocol     string `json:"protocol" description:"oidc or saml"`
	Domain       string `json:"domain" description:"Email-domain or IdP host this connection covers"`
	Issuer       string `json:"issuer,omitempty" description:"OIDC issuer URL (required for oidc)"`
	ClientID     string `json:"client_id,omitempty" description:"OIDC client ID"`
	ClientSecret string `json:"client_secret,omitempty" description:"OIDC client secret (omit for public flows)"`
	MetadataURL  string `json:"metadata_url,omitempty" description:"SAML metadata URL (required for saml)"`
}

// AdminCreateConnectionResponse is the response from
// POST /v1/admin/sso/connections.
type AdminCreateConnectionResponse struct {
	ID       string `json:"id"`
	AppID    string `json:"app_id"`
	Provider string `json:"provider"`
	Protocol string `json:"protocol"`
	Domain   string `json:"domain"`
	Active   bool   `json:"active"`
}

// handleAdminCreateConnection registers an SSO connection on a
// target App. Mirrors the dashboard's connection-creation flow so
// the same store-level invariants apply.
func (p *Plugin) handleAdminCreateConnection(ctx forge.Context, req *AdminCreateConnectionRequest) (*AdminCreateConnectionResponse, error) {
	if p.ssoStore == nil {
		return nil, forge.InternalError(fmt.Errorf("sso plugin: store not wired"))
	}
	if strings.TrimSpace(req.AppID) == "" {
		return nil, forge.BadRequest("app_id is required")
	}
	if strings.TrimSpace(req.Provider) == "" || strings.TrimSpace(req.Protocol) == "" || strings.TrimSpace(req.Domain) == "" {
		return nil, forge.BadRequest("provider, protocol, and domain are required")
	}
	if req.Protocol != "oidc" && req.Protocol != "saml" {
		return nil, forge.BadRequest("protocol must be 'oidc' or 'saml'")
	}

	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid app_id: %v", err))
	}
	var orgID id.OrgID
	if strings.TrimSpace(req.OrgID) != "" {
		orgID, err = id.ParseOrgID(req.OrgID)
		if err != nil {
			return nil, forge.BadRequest(fmt.Sprintf("invalid org_id: %v", err))
		}
	}

	now := time.Now()
	conn := &Connection{
		ID:        id.NewSSOConnectionID(),
		AppID:     appID,
		OrgID:     orgID,
		Provider:  req.Provider,
		Protocol:  req.Protocol,
		Domain:    req.Domain,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	switch req.Protocol {
	case "oidc":
		if strings.TrimSpace(req.Issuer) == "" || strings.TrimSpace(req.ClientID) == "" {
			return nil, forge.BadRequest("OIDC connections require issuer and client_id")
		}
		conn.Issuer = req.Issuer
		conn.ClientID = req.ClientID
		conn.ClientSecret = req.ClientSecret
	case "saml":
		if strings.TrimSpace(req.MetadataURL) == "" {
			return nil, forge.BadRequest("SAML connections require metadata_url")
		}
		conn.MetadataURL = req.MetadataURL
	}

	if err := p.ssoStore.CreateConnection(ctx.Context(), conn); err != nil {
		return nil, forge.InternalError(fmt.Errorf("sso: create connection: %w", err))
	}

	return &AdminCreateConnectionResponse{
		ID:       conn.ID.String(),
		AppID:    conn.AppID.String(),
		Provider: conn.Provider,
		Protocol: conn.Protocol,
		Domain:   conn.Domain,
		Active:   conn.Active,
	}, nil
}
