package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	"github.com/xraph/authsome/organization"
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

	// PublicBaseURL is the externally-reachable base URL of this SSO service
	// (e.g. "https://auth.muono.cloud"). SAML SP EntityID / ACS URL derive from
	// it when a connection doesn't override them. Required for SAML.
	PublicBaseURL string

	// AllowedReturnOrigins is the allowlist of scheme://host[:port] origins the
	// browser SSO landing may redirect back to after ACS. Guards against open
	// redirect. localhost/127.0.0.1 are always allowed (dev). The first entry is
	// used to derive a default `/sso/callback` landing when none is supplied.
	AllowedReturnOrigins []string
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

	// Wire the connection store from the engine's database. Without this the
	// admin CRUD and DB-managed provider resolution have no backing store and
	// every SSO connection is invisible. Mirrors the oauth2provider plugin.
	// An explicit SetSSOStore (e.g. in tests) takes precedence.
	if p.ssoStore == nil {
		if db := engine.DB(); db != nil {
			switch db.Driver().Name() {
			case "pg":
				p.ssoStore = NewPostgresStore(db)
			case "sqlite":
				p.ssoStore = NewSqliteStore(db)
			case "mongo":
				p.ssoStore = NewMongoStore(db)
			}
		}
	}
	if p.ssoStore == nil {
		p.ssoStore = NewMemoryStore()
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

// errProviderNotFound signals that no code- or DB-configured provider matched.
// Distinguished from a build error so callers can return 400 vs 500.
var errProviderNotFound = errors.New("unsupported SSO provider")

// resolveProvider looks up a provider by name — first from code-configured
// providers, then from database-managed SSO connections. A build failure
// (e.g. unreachable SAML metadata) surfaces as a non-sentinel error. The
// resolved *Connection is returned too (nil for code-configured providers) so
// the callback path can enroll the user into the connection's org.
func (p *Plugin) resolveProvider(ctx context.Context, appID id.AppID, name string) (Provider, *Connection, error) {
	// Check code-configured providers first.
	if prov, ok := p.providers[name]; ok {
		return prov, nil, nil
	}

	// Fall back to DB-managed connections, scoped to the resolved app.
	if p.ssoStore == nil {
		return nil, nil, errProviderNotFound
	}
	conn, err := p.ssoStore.GetConnectionByProvider(ctx, appID, name)
	if err != nil || conn == nil || !conn.Active {
		return nil, nil, errProviderNotFound
	}
	prov, err := p.connectionToProvider(conn)
	if err != nil {
		return nil, nil, err
	}
	return prov, conn, nil
}

// requestAppID resolves the app this request targets: the per-request app set by
// the publishable-key middleware (workspace-app logins) when present, else the
// statically-configured p.appID (platform app / code-configured providers). This
// is what lets a login hitting a workspace app resolve THAT app's connections.
func (p *Plugin) requestAppID(ctx forge.Context) (id.AppID, error) {
	if appID, ok := middleware.AppIDFrom(ctx.Context()); ok {
		return appID, nil
	}
	return id.ParseAppID(p.appID)
}

// connectionByID loads an active connection by its id. Used by the browser
// landing paths (ACS/metadata), which carry a `?connection=` param instead of a
// publishable key so the app can be recovered without the pub-key middleware.
func (p *Plugin) connectionByID(ctx context.Context, connID string) (*Connection, error) {
	if p.ssoStore == nil {
		return nil, errProviderNotFound
	}
	parsed, err := id.ParseSSOConnectionID(connID)
	if err != nil {
		return nil, errProviderNotFound
	}
	conn, err := p.ssoStore.GetConnection(ctx, parsed)
	if err != nil || conn == nil || !conn.Active {
		return nil, errProviderNotFound
	}
	return conn, nil
}

// connectionToProvider creates a Provider from a stored Connection.
func (p *Plugin) connectionToProvider(conn *Connection) (Provider, error) {
	switch conn.Protocol {
	case "oidc":
		return NewOIDCProvider(OIDCConfig{
			Name:         conn.Provider,
			Issuer:       conn.Issuer,
			ClientID:     conn.ClientID,
			ClientSecret: conn.ClientSecret,
			// The IdP redirects the browser here with ?code&state after auth. It
			// must be set (empty redirect_uri is rejected by real IdPs) and must
			// exactly match what's registered with the IdP — see oidcRedirectURLFor.
			RedirectURL: p.oidcRedirectURLFor(conn),
		}), nil
	case "saml":
		return NewSAMLProvider(SAMLConfig{
			Name:              conn.Provider,
			IDPMetadataXML:    conn.IDPMetadataXML,
			MetadataURL:       conn.MetadataURL,
			IDPSSOURL:         conn.IDPSSOURL,
			IDPCertificatePEM: conn.IDPCertificate,
			EntityID:          p.entityIDFor(conn),
			ACSURL:            p.acsURLFor(conn),
			SPCertificatePEM:  conn.SPCertificate,
			SPPrivateKeyPEM:   conn.SPPrivateKey,
			SignRequests:      conn.SignRequests,
			AttributeMap:      conn.AttributeMappings,
		})
	default:
		return nil, fmt.Errorf("sso: unsupported protocol %q", conn.Protocol)
	}
}

// publicBaseURL returns the configured base URL without a trailing slash.
func (p *Plugin) publicBaseURL() string {
	return strings.TrimRight(p.config.PublicBaseURL, "/")
}

// acsURLFor returns the SP ACS URL for a connection: the stored override, or
// the canonical {base}/v1/sso/{provider}/acs.
func (p *Plugin) acsURLFor(conn *Connection) string {
	if v := strings.TrimSpace(conn.ACSURL); v != "" {
		return v
	}
	// Embed the connection id so the IdP's top-level POST (which carries no
	// publishable key) still lets ACS recover the connection and its app.
	return p.publicBaseURL() + "/v1/sso/" + conn.Provider + "/acs?connection=" + conn.ID.String()
}

// oidcRedirectURLFor returns the OIDC redirect_uri for a connection: the browser
// landing the IdP sends ?code&state to. Connection-scoped (like the ACS URL) so
// the GET handler can recover the connection and its app without a publishable
// key. This must be registered as an allowed redirect URI with the IdP.
func (p *Plugin) oidcRedirectURLFor(conn *Connection) string {
	return p.publicBaseURL() + "/v1/sso/" + conn.Provider + "/callback?connection=" + conn.ID.String()
}

// entityIDFor returns the SP EntityID for a connection: the stored override, or
// the canonical {base}/v1/sso/{provider}/metadata.
func (p *Plugin) entityIDFor(conn *Connection) string {
	if v := strings.TrimSpace(conn.EntityID); v != "" {
		return v
	}
	// Doubles as the SP metadata URL the IdP fetches; connection-scoped so it
	// resolves without a publishable key. Valid as a URI EntityID too.
	return p.publicBaseURL() + "/v1/sso/" + conn.Provider + "/metadata?connection=" + conn.ID.String()
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

	// Domain-routed login: caller submits an email, we resolve the IdP from its
	// domain. Enables "type your work email → get sent to your IdP".
	if err := g.POST("/login", p.handleLoginByDomain,
		forge.WithSummary("Start SSO login by email domain"),
		forge.WithOperationID("startSSOLoginByDomain"),
		forge.WithRequestSchema(LoginByDomainRequest{}),
		forge.WithResponseSchema(http.StatusOK, "SSO login URL", LoginResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// SP metadata: IdPs fetch this to auto-configure the Service Provider.
	// Raw handler because it emits application/samlmetadata+xml, not JSON.
	if err := g.GET("/:provider/metadata", p.handleSPMetadata,
		forge.WithSummary("SAML SP metadata"),
		forge.WithOperationID("ssoSPMetadata"),
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

	// OIDC browser redirect landing: the IdP redirects the browser here (GET) with
	// ?code&state after auth. Raw handler because it exchanges the code, mints a
	// one-time code, and 302-redirects to the frontend return URL — mirroring the
	// SAML ACS — so the same frontend /sso/callback + /exchange path serves both
	// protocols. Distinct from the POST /callback above (a JSON API variant).
	if err := g.GET("/:provider/callback", p.handleOIDCRedirect,
		forge.WithSummary("SSO OIDC redirect landing"),
		forge.WithOperationID("ssoOIDCRedirect"),
	); err != nil {
		return err
	}

	// SAML ACS: the IdP HTTP-POSTs the assertion here as a top-level browser
	// navigation (no publishable key). Raw handler because it 302-redirects the
	// browser to the frontend return URL with a one-time code, rather than
	// returning a JSON body the browser would render as a page.
	if err := g.POST("/:provider/acs", p.handleACS,
		forge.WithSummary("SSO SAML ACS endpoint"),
		forge.WithOperationID("ssoACS"),
	); err != nil {
		return err
	}

	// Exchange a one-time code (minted by ACS) for the session tokens. Called by
	// the frontend landing page with its publishable key, so the app can be
	// verified against the code's bound app.
	if err := g.POST("/exchange", p.handleExchange,
		forge.WithSummary("Exchange SSO one-time code for a session"),
		forge.WithOperationID("ssoExchange"),
		forge.WithRequestSchema(ExchangeRequest{}),
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
	if err := admin.POST("/connections", p.handleAdminCreateConnection,
		forge.WithSummary("Create SSO connection (admin)"),
		forge.WithDescription("Registers an OIDC or SAML SSO connection on a target App. Used by platform-admin clients to provision per-tenant IdPs."),
		forge.WithOperationID("ssoAdminCreateConnection"),
		forge.WithRequestSchema(AdminCreateConnectionRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Connection created", AdminCreateConnectionResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := admin.GET("/connections", p.handleAdminListConnections,
		forge.WithSummary("List SSO connections for an app (admin)"),
		forge.WithDescription("Returns every SSO connection registered on the target App. Filter by ?app_id=app_..."),
		forge.WithOperationID("ssoAdminListConnections"),
		forge.WithRequestSchema(AdminListConnectionsRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Connections", AdminListConnectionsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := admin.GET("/connections/:connectionId", p.handleAdminGetConnection,
		forge.WithSummary("Get SSO connection (admin)"),
		forge.WithOperationID("ssoAdminGetConnection"),
		forge.WithRequestSchema(AdminConnectionPathRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Connection", Connection{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := admin.PUT("/connections/:connectionId", p.handleAdminUpdateConnection,
		forge.WithSummary("Update SSO connection (admin)"),
		forge.WithDescription("Modify an existing SSO connection. Empty fields leave the stored value unchanged. The app_id and protocol are immutable."),
		forge.WithOperationID("ssoAdminUpdateConnection"),
		forge.WithRequestSchema(AdminUpdateConnectionRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Connection updated", Connection{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return admin.DELETE("/connections/:connectionId", p.handleAdminDeleteConnection,
		forge.WithSummary("Delete SSO connection (admin)"),
		forge.WithOperationID("ssoAdminDeleteConnection"),
		forge.WithRequestSchema(AdminConnectionPathRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Deleted", AdminDeleteConnectionResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// LoginRequest contains the path parameter for starting SSO.
type LoginRequest struct {
	Provider  string `path:"provider"`
	ReturnURL string `json:"return_url,omitempty" query:"return_url,omitempty" description:"Frontend URL to land on after login (must be allowlisted)"`
}

// ssoState is the CSRF/state ceremony payload. It also carries the resolved app
// and the frontend return URL so the browser-landing callback/ACS (which lack a
// publishable key) can recover both.
type ssoState struct {
	Provider  string `json:"provider"`
	AppID     string `json:"app_id,omitempty"`
	ReturnURL string `json:"return_url,omitempty"`
	// RequestID is the SAML AuthnRequest ID minted at login, matched against the
	// assertion's InResponseTo at the ACS. Empty for OIDC.
	RequestID string `json:"request_id,omitempty"`
}

// requestIDProvider is implemented by SAML providers that expose the AuthnRequest
// ID generated when building the login URL, so it can be persisted in the state
// ceremony and validated at the ACS.
type requestIDProvider interface {
	LoginURLWithRequestID(state string) (loginURL, requestID string, err error)
}

// ExchangeRequest is the body for POST /v1/sso/exchange.
type ExchangeRequest struct {
	Code string `json:"code" description:"One-time code issued by the ACS redirect"`
}

// otcPayload is the session handoff stashed under a one-time code by ACS and
// redeemed once (app-bound) at /v1/sso/exchange.
type otcPayload struct {
	User         json.RawMessage `json:"user,omitempty"`
	SessionToken string          `json:"session_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresAt    string          `json:"expires_at,omitempty"`
	Provider     string          `json:"provider,omitempty"`
	IsNewUser    bool            `json:"is_new_user,omitempty"`
	AppID        string          `json:"app_id"`
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
	appID, err := p.requestAppID(ctx)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("invalid app_id configuration: %w", err))
	}
	provider, _, err := p.resolveProvider(ctx.Context(), appID, req.Provider)
	if err != nil {
		return nil, providerResolveError(req.Provider, err)
	}
	return p.startLogin(ctx.Context(), appID, provider, req.Provider, req.ReturnURL)
}

// startLogin generates a CSRF state (carrying the app + return URL), caches it,
// and returns the IdP login URL. Shared by provider-name and email-domain entry
// points. The return URL is validated here (login is publishable-key-authed), so
// the opaque state token that round-trips through the IdP can't be tampered with.
func (p *Plugin) startLogin(ctx context.Context, appID id.AppID, provider Provider, providerName, returnURL string) (*LoginResponse, error) {
	if returnURL != "" && !p.isAllowedReturnURL(returnURL) {
		return nil, forge.BadRequest("return_url is not allowed")
	}

	state, err := generateState()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate state: %w", err))
	}

	// Build the IdP login URL. SAML providers also return the AuthnRequest ID so
	// we can persist it in the state ceremony and match the assertion's
	// InResponseTo at the ACS.
	var loginURL, requestID string
	if rp, ok := provider.(requestIDProvider); ok {
		loginURL, requestID, err = rp.LoginURLWithRequestID(state)
	} else {
		loginURL, err = provider.LoginURL(state)
	}
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to get login URL: %w", err))
	}

	stateData, _ := json.Marshal(ssoState{Provider: providerName, AppID: appID.String(), ReturnURL: returnURL, RequestID: requestID}) //nolint:errcheck // best-effort cache
	_ = p.ceremonies.Set(ctx, "sso:state:"+state, stateData, 10*time.Minute)                                                          //nolint:errcheck // best-effort cache

	return &LoginResponse{
		LoginURL: loginURL,
		State:    state,
	}, nil
}

// providerResolveError maps a resolveProvider failure to the right HTTP status:
// 400 for an unknown provider, 500 for a build/config failure.
func providerResolveError(name string, err error) error {
	if errors.Is(err, errProviderNotFound) {
		return forge.BadRequest(fmt.Sprintf("unsupported SSO provider: %s", name))
	}
	return forge.InternalError(fmt.Errorf("sso: resolve provider %q: %w", name, err))
}

// LoginByDomainRequest carries an email whose domain selects the IdP connection.
type LoginByDomainRequest struct {
	Email     string `json:"email" description:"Work email; its domain routes to the matching SSO connection"`
	ReturnURL string `json:"return_url,omitempty" description:"Frontend URL to land on after login (must be allowlisted)"`
}

// handleLoginByDomain resolves the SSO connection from the email's domain and
// starts the login flow, so users never have to know their provider's name.
func (p *Plugin) handleLoginByDomain(ctx forge.Context, req *LoginByDomainRequest) (*LoginResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	at := strings.LastIndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return nil, forge.BadRequest("a valid email is required")
	}
	if p.ssoStore == nil {
		return nil, forge.BadRequest("no SSO connection for this domain")
	}
	appID, err := p.requestAppID(ctx)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("invalid app_id configuration: %w", err))
	}
	conn, err := p.ssoStore.GetConnectionByDomain(ctx.Context(), appID, email[at+1:])
	if err != nil || conn == nil || !conn.Active {
		return nil, forge.BadRequest("no SSO connection for this domain")
	}
	provider, err := p.connectionToProvider(conn)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("sso: build provider: %w", err))
	}
	return p.startLogin(ctx.Context(), appID, provider, conn.Provider, req.ReturnURL)
}

// handleSPMetadata serves the SAML SP metadata XML for an IdP to consume. Raw
// handler: the body is application/samlmetadata+xml, not the usual JSON envelope.
func (p *Plugin) handleSPMetadata(ctx forge.Context) error {
	name := ctx.Param("provider")
	var provider Provider
	// The IdP fetches this without a publishable key, so resolve by the
	// connection-scoped `?connection=` param when present; else fall back to
	// provider-name resolution under the request/platform app.
	if connID := ctx.Request().URL.Query().Get("connection"); connID != "" {
		conn, err := p.connectionByID(ctx.Context(), connID)
		if err != nil {
			return providerResolveError(name, err)
		}
		if provider, err = p.connectionToProvider(conn); err != nil {
			return forge.InternalError(fmt.Errorf("sso: build provider: %w", err))
		}
	} else {
		appID, err := p.requestAppID(ctx)
		if err != nil {
			return forge.InternalError(fmt.Errorf("invalid app_id configuration: %w", err))
		}
		if provider, _, err = p.resolveProvider(ctx.Context(), appID, name); err != nil {
			return providerResolveError(name, err)
		}
	}
	mp, ok := provider.(SAMLMetadataProvider)
	if !ok {
		return forge.BadRequest("provider does not expose SAML metadata")
	}
	xmlBytes, contentType, err := mp.Metadata()
	if err != nil {
		return forge.InternalError(fmt.Errorf("sso: metadata: %w", err))
	}
	ctx.Response().Header().Set("Content-Type", contentType)
	ctx.Response().WriteHeader(http.StatusOK)
	_, werr := ctx.Response().Write(xmlBytes)
	return werr
}

// handleCallback processes the OIDC callback.
func (p *Plugin) handleCallback(ctx forge.Context, req *CallbackRequest) (*CallbackResponse, error) {
	if req.State == "" {
		return nil, forge.BadRequest("missing state parameter")
	}

	// The OIDC callback is a browser redirect with no publishable key; recover
	// the app (and validate CSRF) from the state ceremony stashed at login.
	st, err := p.loadState(ctx.Context(), req.State, req.Provider)
	if err != nil {
		return nil, forge.BadRequest(err.Error())
	}
	appID, err := p.appIDFromState(st)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("invalid app_id configuration: %w", err))
	}

	provider, conn, err := p.resolveProvider(ctx.Context(), appID, req.Provider)
	if err != nil {
		return nil, providerResolveError(req.Provider, err)
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

	return p.authenticateUser(ctx, appID, provider, conn, params)
}

// handleOIDCRedirect is the browser landing for the OIDC authorization-code
// flow. The IdP redirects here (GET) with ?code&state; this validates the CSRF
// state, exchanges the code for the user, mints a one-time code, and 302s to the
// frontend return URL — the same handoff the SAML ACS uses, so one frontend
// callback serves both protocols.
func (p *Plugin) handleOIDCRedirect(ctx forge.Context) error {
	r := ctx.Request()
	name := ctx.Param("provider")
	q := r.URL.Query()
	code := q.Get("code")
	state := q.Get("state")
	connID := q.Get("connection")
	providerErr := q.Get("error")

	// Validate the CSRF state and recover the frontend return URL. OIDC is always
	// SP-initiated, so a missing/invalid state is a hard failure (loadState also
	// consumes it — single-use).
	st, serr := p.loadState(ctx.Context(), state, name)
	returnURL := ""
	if st != nil {
		returnURL = st.ReturnURL
	}

	fail := func(reason string) error {
		http.Redirect(ctx.Response(), r, p.errorRedirect(returnURL, reason), http.StatusFound)
		return nil
	}

	if serr != nil {
		return fail("invalid_state")
	}
	if providerErr != "" {
		return fail(providerErr)
	}

	// Resolve the connection (and its app) from `?connection=`; fall back to
	// provider-name under the request app for legacy/platform links.
	var conn *Connection
	var err error
	if connID != "" {
		conn, err = p.connectionByID(ctx.Context(), connID)
	} else {
		var appID id.AppID
		if appID, err = p.requestAppID(ctx); err == nil {
			_, conn, err = p.resolveProvider(ctx.Context(), appID, name)
		}
	}
	if err != nil || conn == nil {
		return fail("provider_not_found")
	}
	if code == "" {
		return fail("missing_code")
	}

	provider, perr := p.connectionToProvider(conn)
	if perr != nil {
		return fail("provider_error")
	}

	params := map[string]string{"code": code, "state": state}
	result, aerr := p.authenticateUser(ctx, conn.AppID, provider, conn, params)
	if aerr != nil {
		if p.logger != nil {
			p.logger.Warn("sso: oidc callback authentication failed", log.String("error", aerr.Error()))
		}
		return fail("auth_failed")
	}

	otc, cerr := p.mintOTC(ctx.Context(), conn.AppID, result)
	if cerr != nil {
		return fail("handoff_failed")
	}
	http.Redirect(ctx.Response(), r, p.successRedirect(returnURL, otc), http.StatusFound)
	return nil
}

// handleACS processes the SAML Assertion Consumer Service POST. It is a raw
// handler: the IdP posts here as a top-level browser navigation (no publishable
// key), so we recover the connection/app from the connection-scoped ACS URL,
// provision the user, mint a one-time code carrying the session, and 302-redirect
// the browser to the frontend return URL. On any failure we redirect with an
// `?sso_error=` marker rather than leaking the assertion or an error body.
func (p *Plugin) handleACS(ctx forge.Context) error {
	r := ctx.Request()
	name := ctx.Param("provider")
	samlResponse := r.FormValue("SAMLResponse")
	relayState := r.FormValue("RelayState")
	connID := r.URL.Query().Get("connection")

	// Recover the frontend return URL + the AuthnRequest ID from the state
	// ceremony (SP-initiated). Both were validated + stored at login, so they're
	// trusted here. Absent for IdP-initiated flows → return URL falls back to the
	// configured default, and no request ID means the assertion must be
	// IdP-initiated (gated by AllowIDPInitiated).
	returnURL := ""
	requestID := ""
	if relayState != "" {
		if st, serr := p.loadState(ctx.Context(), relayState, name); serr == nil {
			returnURL = st.ReturnURL
			requestID = st.RequestID
		}
	}

	fail := func(reason string) error {
		http.Redirect(ctx.Response(), r, p.errorRedirect(returnURL, reason), http.StatusFound)
		return nil
	}

	// Resolve the connection (and its app) from `?connection=`; fall back to
	// provider-name under the request app for legacy/platform links.
	var conn *Connection
	var err error
	if connID != "" {
		conn, err = p.connectionByID(ctx.Context(), connID)
	} else {
		var appID id.AppID
		if appID, err = p.requestAppID(ctx); err == nil {
			_, conn, err = p.resolveProvider(ctx.Context(), appID, name)
		}
	}
	if err != nil || conn == nil {
		return fail("provider_not_found")
	}
	if samlResponse == "" {
		return fail("missing_response")
	}

	provider, perr := p.connectionToProvider(conn)
	if perr != nil {
		return fail("provider_error")
	}

	params := map[string]string{"SAMLResponse": samlResponse, "RelayState": relayState, "request_id": requestID}
	result, aerr := p.authenticateUser(ctx, conn.AppID, provider, conn, params)
	if aerr != nil {
		if p.logger != nil {
			p.logger.Warn("sso: acs authentication failed", log.String("error", aerr.Error()))
		}
		return fail("auth_failed")
	}

	code, cerr := p.mintOTC(ctx.Context(), conn.AppID, result)
	if cerr != nil {
		return fail("handoff_failed")
	}
	http.Redirect(ctx.Response(), r, p.successRedirect(returnURL, code), http.StatusFound)
	return nil
}

// authenticateUser processes an SSO identity and creates/links a user.
// errUnverifiedSSOLink signals that an SSO email matched a pre-existing local
// account whose email has never been verified. Linking in that case is an
// account-takeover vector, so the caller refuses.
var errUnverifiedSSOLink = errors.New("sso: refusing to link to an unverified pre-existing account")

// linkableExistingUser resolves an existing local account for an SSO email and
// verifies it is safe to link to. It returns:
//   - (user, nil) when a matching account exists and the matched email is verified
//   - (nil, nil)  when no account matches (the caller creates a fresh user)
//   - (nil, errUnverifiedSSOLink) when a match exists but its email is unverified
//   - (nil, err)  on an unexpected store error
//
// This mirrors the social plugin's verified-email linking rule so SSO logins
// cannot be captured by a pre-registered unverified account.
func (p *Plugin) linkableExistingUser(ctx context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*user.User, error) {
	u, err := p.store.GetUserByAnyEmail(ctx, appID, envID, email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	rec, recErr := p.store.GetUserEmailRecord(ctx, appID, envID, email)
	if recErr != nil || rec == nil || !rec.Verified {
		return nil, errUnverifiedSSOLink
	}
	return u, nil
}

func (p *Plugin) authenticateUser(ctx forge.Context, appID id.AppID, provider Provider, conn *Connection, params map[string]string) (*CallbackResponse, error) {
	ssoUser, err := provider.HandleCallback(ctx.Context(), params)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("sso: callback failed: %w", err))
	}

	goCtx := ctx.Context()

	// Resolve the app's default environment so the user and its email row are
	// env-scoped consistently with password/social sign-ups.
	var envID id.EnvironmentID
	if env, _ := p.store.GetDefaultEnvironment(goCtx, appID); env != nil { //nolint:errcheck // best-effort env lookup
		envID = env.ID
	}

	// Find or create user by email. Match across all of an account's emails so
	// a verified SSO email links to the existing account instead of duplicating.
	var u *user.User
	isNew := false

	if ssoUser.Email != "" {
		email := strings.ToLower(ssoUser.Email)
		u, err = p.linkableExistingUser(goCtx, appID, envID, email)
		if errors.Is(err, errUnverifiedSSOLink) {
			// A pre-existing account owns this email but has never verified it.
			// Linking here would let an attacker who pre-registered the
			// victim's email capture the victim's SSO login, so refuse.
			return nil, forge.NewHTTPError(http.StatusConflict,
				"an account with this email already exists but is not verified; verify it before signing in with SSO")
		}
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("sso: resolve existing user: %w", err))
		}
		if u == nil {
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
				EnvID:         envID,
				Email:         strings.ToLower(ssoUser.Email),
				EmailVerified: true, // SSO-authenticated emails are verified
				FirstName:     ssoUser.FirstName,
				LastName:      ssoUser.LastName,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			if createErr := p.store.CreateUserWithPrimaryEmail(goCtx, u, user.NewPrimaryEmail(u, "sso")); createErr != nil {
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

	// Org-level SSO: enroll the user into the connection's org so signing in via
	// the IdP adds them to the org (the login-based alternative to an invite).
	// Idempotent and best-effort — a membership hiccup must not fail the login.
	if conn != nil && conn.OrgID.Prefix() != "" {
		p.ensureOrgMembership(goCtx, conn.OrgID, u.ID)
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

// ensureOrgMembership adds the user to the org as a member if not already one.
// Idempotent and best-effort: a lookup/insert failure is logged, never fatal to
// the login (the user still gets a session; membership can be reconciled later).
func (p *Plugin) ensureOrgMembership(ctx context.Context, orgID id.OrgID, userID id.UserID) {
	if members, err := p.store.ListMembers(ctx, orgID); err == nil {
		for _, m := range members {
			if m != nil && m.UserID == userID {
				return // already a member
			}
		}
	}
	now := time.Now()
	m := &organization.Member{
		ID:        id.NewMemberID(),
		OrgID:     orgID,
		UserID:    userID,
		Role:      organization.RoleMember,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := p.store.CreateMember(ctx, m); err != nil && p.logger != nil {
		p.logger.Warn("sso: enroll user into org failed",
			log.String("org_id", orgID.String()),
			log.String("user_id", userID.String()),
			log.String("error", err.Error()))
	}
}

// loadState validates the CSRF state token (single-use), enforces the provider
// match, and returns the stashed ceremony payload (app + return URL) so the
// browser-landing paths can recover context they can't get from a publishable key.
func (p *Plugin) loadState(ctx context.Context, state, providerName string) (*ssoState, error) {
	stateData, err := p.ceremonies.Get(ctx, "sso:state:"+state)
	if err != nil {
		return nil, fmt.Errorf("invalid state parameter")
	}
	_ = p.ceremonies.Delete(ctx, "sso:state:"+state) //nolint:errcheck // best-effort cleanup
	var s ssoState
	if err := json.Unmarshal(stateData, &s); err != nil || s.Provider != providerName {
		return nil, fmt.Errorf("invalid state parameter")
	}
	return &s, nil
}

// appIDFromState resolves the app carried in the state payload, falling back to
// the static platform app when absent (legacy states / code-configured flows).
func (p *Plugin) appIDFromState(s *ssoState) (id.AppID, error) {
	if s != nil && strings.TrimSpace(s.AppID) != "" {
		return id.ParseAppID(s.AppID)
	}
	return id.ParseAppID(p.appID)
}

// mintOTC stashes the freshly-issued session under a single-use one-time code
// (short TTL, app-bound) for the frontend to redeem at /v1/sso/exchange.
func (p *Plugin) mintOTC(ctx context.Context, appID id.AppID, r *CallbackResponse) (string, error) {
	code, err := generateState()
	if err != nil {
		return "", err
	}
	expires := ""
	if t, ok := r.ExpiresAt.(time.Time); ok {
		expires = t.Format(time.RFC3339)
	}
	var userJSON json.RawMessage
	if b, merr := json.Marshal(r.User); merr == nil {
		userJSON = b
	}
	// #nosec G101 -- SessionToken/RefreshToken carry freshly-minted runtime session values, not hardcoded credentials. The payload is stashed single-use under a 60s app-bound OTC (mintOTC) so the browser redeems it at /v1/sso/exchange instead of receiving tokens in a redirect URL.
	payload, err := json.Marshal(otcPayload{
		User:         userJSON,
		SessionToken: r.SessionToken,
		RefreshToken: r.RefreshToken,
		ExpiresAt:    expires,
		Provider:     r.Provider,
		IsNewUser:    r.IsNewUser,
		AppID:        appID.String(),
	})
	if err != nil {
		return "", err
	}
	if err := p.ceremonies.Set(ctx, "sso:otc:"+code, payload, 60*time.Second); err != nil {
		return "", err
	}
	return code, nil
}

// handleExchange redeems a one-time code (single-use) for the session tokens.
// The caller presents its publishable key; the code is bound to the app it was
// minted for, so it can't be redeemed against a different app.
func (p *Plugin) handleExchange(ctx forge.Context, req *ExchangeRequest) (*CallbackResponse, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, forge.BadRequest("code is required")
	}
	raw, err := p.ceremonies.Get(ctx.Context(), "sso:otc:"+code)
	if err != nil {
		return nil, forge.BadRequest("invalid or expired code")
	}
	_ = p.ceremonies.Delete(ctx.Context(), "sso:otc:"+code) //nolint:errcheck // single-use
	var pl otcPayload
	if err := json.Unmarshal(raw, &pl); err != nil {
		return nil, forge.BadRequest("invalid code")
	}
	// App-bind: reject redemption under a different app's publishable key.
	if appID, ok := middleware.AppIDFrom(ctx.Context()); ok && appID.String() != pl.AppID {
		return nil, forge.BadRequest("code is not valid for this app")
	}
	var u any
	if len(pl.User) > 0 {
		u = pl.User
	}
	return &CallbackResponse{
		User:         u,
		SessionToken: pl.SessionToken,
		RefreshToken: pl.RefreshToken,
		ExpiresAt:    pl.ExpiresAt,
		Provider:     pl.Provider,
		IsNewUser:    pl.IsNewUser,
	}, nil
}

// isAllowedReturnURL guards the browser landing against open redirect. localhost
// is always allowed (dev); otherwise the URL must be https and its origin must
// exactly match a configured AllowedReturnOrigins entry.
func (p *Plugin) isAllowedReturnURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	if host := u.Hostname(); host == "localhost" || host == "127.0.0.1" {
		return true
	}
	if u.Scheme != "https" {
		return false
	}
	origin := u.Scheme + "://" + u.Host
	for _, a := range p.config.AllowedReturnOrigins {
		if strings.EqualFold(strings.TrimRight(a, "/"), origin) {
			return true
		}
	}
	return false
}

// defaultReturnURL returns a validated return URL, or a safe default landing
// derived from the first allowlisted origin (or the public base) otherwise.
func (p *Plugin) defaultReturnURL(returnURL string) string {
	if returnURL != "" && p.isAllowedReturnURL(returnURL) {
		return returnURL
	}
	if len(p.config.AllowedReturnOrigins) > 0 {
		return strings.TrimRight(p.config.AllowedReturnOrigins[0], "/") + "/sso/callback"
	}
	return p.publicBaseURL() + "/sso/callback"
}

func (p *Plugin) successRedirect(returnURL, code string) string {
	return appendQuery(p.defaultReturnURL(returnURL), "code", code)
}

func (p *Plugin) errorRedirect(returnURL, reason string) string {
	return appendQuery(p.defaultReturnURL(returnURL), "sso_error", reason)
}

func appendQuery(base, key, value string) string {
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + key + "=" + url.QueryEscape(value)
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
// Programmatic API
// ──────────────────────────────────────────────────

// SSOStore returns the connection store so an embedding host app can read/manage
// connections directly (e.g. list by domain, toggle active) behind its own
// authorization. Returns nil before OnInit has wired a store.
func (p *Plugin) SSOStore() Store { return p.ssoStore }

// CreateConnectionInput describes an SSO connection to provision. It is the
// transport-agnostic form of AdminCreateConnectionRequest so host apps that
// expose their own tenant-scoped admin surface (instead of the platform-admin
// HTTP route) can create connections after their own authorization check.
type CreateConnectionInput struct {
	AppID    id.AppID
	OrgID    id.OrgID // zero value = app-wide (no org scope)
	Provider string
	Protocol string // "oidc" | "saml"
	Domain   string

	// OIDC
	Issuer       string
	ClientID     string
	ClientSecret string

	// SAML — supply one IdP source: MetadataURL, IDPMetadataXML, or
	// IDPSSOURL + IDPCertificate.
	MetadataURL       string
	IDPMetadataXML    string
	IDPSSOURL         string
	IDPCertificate    string
	EntityID          string
	ACSURL            string
	SignRequests      bool
	AttributeMappings map[string]string
}

// CreateConnection provisions an SSO connection: it resolves the app's default
// environment (env_id is NOT NULL), validates the protocol-specific fields, and
// for SAML derives the SP EntityID/ACS URL from PublicBaseURL and generates a
// self-signed SP keypair when none is supplied. This is the shared core behind
// handleAdminCreateConnection; host apps call it directly behind their own
// authorization instead of the platform-admin HTTP route.
func (p *Plugin) CreateConnection(ctx context.Context, in CreateConnectionInput) (*Connection, error) {
	if p.ssoStore == nil {
		return nil, forge.InternalError(fmt.Errorf("sso plugin: store not wired"))
	}
	if in.AppID.IsNil() {
		return nil, forge.BadRequest("app_id is required")
	}
	if strings.TrimSpace(in.Provider) == "" || strings.TrimSpace(in.Protocol) == "" || strings.TrimSpace(in.Domain) == "" {
		return nil, forge.BadRequest("provider, protocol, and domain are required")
	}
	if in.Protocol != "oidc" && in.Protocol != "saml" {
		return nil, forge.BadRequest("protocol must be 'oidc' or 'saml'")
	}

	// Scope the connection to the app's default environment. The
	// authsome_sso_connections.env_id column is NOT NULL with an FK to
	// authsome_environments, so a connection must carry a valid env.
	env, err := p.store.GetDefaultEnvironment(ctx, in.AppID)
	if err != nil || env == nil {
		return nil, forge.InternalError(fmt.Errorf("sso: resolve default environment for app: %w", err))
	}

	now := time.Now()
	conn := &Connection{
		ID:        id.NewSSOConnectionID(),
		AppID:     in.AppID,
		EnvID:     env.ID.String(),
		OrgID:     in.OrgID,
		Provider:  in.Provider,
		Protocol:  in.Protocol,
		Domain:    in.Domain,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	switch in.Protocol {
	case "oidc":
		if strings.TrimSpace(in.Issuer) == "" || strings.TrimSpace(in.ClientID) == "" {
			return nil, forge.BadRequest("OIDC connections require issuer and client_id")
		}
		conn.Issuer = in.Issuer
		conn.ClientID = in.ClientID
		conn.ClientSecret = in.ClientSecret
	case "saml":
		if err := applySAMLCreate(conn, in); err != nil {
			return nil, err
		}
		// Persist the effective EntityID/ACS URL (derived from PublicBaseURL when
		// not overridden) so admins can read them back to configure their IdP.
		conn.EntityID = p.entityIDFor(conn)
		conn.ACSURL = p.acsURLFor(conn)
		// Generate a self-signed SP keypair when the admin supplies none, so
		// signed AuthnRequests and SP metadata work out of the box.
		certPEM, keyPEM, kerr := generateSPKeypair(conn.EntityID)
		if kerr != nil {
			return nil, forge.InternalError(fmt.Errorf("sso: generate SP keypair: %w", kerr))
		}
		conn.SPCertificate = certPEM
		conn.SPPrivateKey = keyPEM
	}

	if err := p.ssoStore.CreateConnection(ctx, conn); err != nil {
		return nil, forge.InternalError(fmt.Errorf("sso: create connection: %w", err))
	}
	return conn, nil
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
	MetadataURL  string `json:"metadata_url,omitempty" description:"SAML metadata URL (one IdP source for saml)"`

	// SAML IdP configuration. Supply exactly one IdP source: metadata_url,
	// idp_metadata_xml, or idp_sso_url + idp_certificate.
	IDPMetadataXML    string            `json:"idp_metadata_xml,omitempty" description:"Pasted SAML IdP metadata XML"`
	IDPSSOURL         string            `json:"idp_sso_url,omitempty" description:"SAML IdP SSO (redirect) URL"`
	IDPCertificate    string            `json:"idp_certificate,omitempty" description:"SAML IdP signing certificate (PEM)"`
	EntityID          string            `json:"entity_id,omitempty" description:"SP EntityID override; defaults to the metadata URL"`
	ACSURL            string            `json:"acs_url,omitempty" description:"SP ACS URL override; defaults to the canonical /acs route"`
	SignRequests      bool              `json:"sign_requests,omitempty" description:"Sign outbound AuthnRequests with the SP key"`
	AttributeMappings map[string]string `json:"attribute_mappings,omitempty" description:"SAML attribute name → user field (email|first_name|last_name|groups)"`
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
	if strings.TrimSpace(req.AppID) == "" {
		return nil, forge.BadRequest("app_id is required")
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

	conn, err := p.CreateConnection(ctx.Context(), CreateConnectionInput{
		AppID:             appID,
		OrgID:             orgID,
		Provider:          req.Provider,
		Protocol:          req.Protocol,
		Domain:            req.Domain,
		Issuer:            req.Issuer,
		ClientID:          req.ClientID,
		ClientSecret:      req.ClientSecret,
		MetadataURL:       req.MetadataURL,
		IDPMetadataXML:    req.IDPMetadataXML,
		IDPSSOURL:         req.IDPSSOURL,
		IDPCertificate:    req.IDPCertificate,
		EntityID:          req.EntityID,
		ACSURL:            req.ACSURL,
		SignRequests:      req.SignRequests,
		AttributeMappings: req.AttributeMappings,
	})
	if err != nil {
		return nil, err
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

// applySAMLCreate validates the IdP source and copies the SAML fields from the
// create input onto the connection.
func applySAMLCreate(conn *Connection, in CreateConnectionInput) error {
	hasMetaURL := strings.TrimSpace(in.MetadataURL) != ""
	hasMetaXML := strings.TrimSpace(in.IDPMetadataXML) != ""
	hasCertURL := strings.TrimSpace(in.IDPSSOURL) != "" && strings.TrimSpace(in.IDPCertificate) != ""
	if !hasMetaURL && !hasMetaXML && !hasCertURL {
		return forge.BadRequest("SAML connections require one IdP source: metadata_url, idp_metadata_xml, or idp_sso_url + idp_certificate")
	}
	conn.MetadataURL = in.MetadataURL
	conn.IDPMetadataXML = in.IDPMetadataXML
	conn.IDPSSOURL = in.IDPSSOURL
	conn.IDPCertificate = in.IDPCertificate
	conn.EntityID = in.EntityID
	conn.ACSURL = in.ACSURL
	conn.SignRequests = in.SignRequests
	conn.AttributeMappings = in.AttributeMappings
	return nil
}

// AdminListConnectionsRequest binds the query for GET /v1/admin/sso/connections.
type AdminListConnectionsRequest struct {
	AppID string `query:"app_id" description:"App identifier; required"`
}

// AdminListConnectionsResponse is the listing response. Returns []
// instead of a paged envelope because per-app connection counts are
// expected to stay small (a few IdPs at most).
type AdminListConnectionsResponse struct {
	Connections []*Connection `json:"connections"`
}

// AdminConnectionPathRequest binds :connectionId for GET / DELETE.
type AdminConnectionPathRequest struct {
	ConnectionID string `path:"connectionId" description:"Connection identifier"`
}

// AdminUpdateConnectionRequest binds the body for PUT /connections/:id.
// Empty/zero values are treated as "leave unchanged" so callers can
// PATCH a single field without re-sending the entire connection.
type AdminUpdateConnectionRequest struct {
	ConnectionID string `path:"connectionId" description:"Connection identifier"`

	Provider     string `json:"provider,omitempty" description:"Stable provider name"`
	Domain       string `json:"domain,omitempty" description:"Email-domain or IdP host"`
	Issuer       string `json:"issuer,omitempty" description:"OIDC issuer URL"`
	ClientID     string `json:"client_id,omitempty" description:"OIDC client ID"`
	ClientSecret string `json:"client_secret,omitempty" description:"OIDC client secret; pass empty string to leave unchanged"`
	MetadataURL  string `json:"metadata_url,omitempty" description:"SAML metadata URL"`

	// SAML fields. Empty strings leave the stored value unchanged; send
	// attribute_mappings/sign_requests to replace them wholesale.
	IDPMetadataXML    string            `json:"idp_metadata_xml,omitempty" description:"Pasted SAML IdP metadata XML"`
	IDPSSOURL         string            `json:"idp_sso_url,omitempty" description:"SAML IdP SSO URL"`
	IDPCertificate    string            `json:"idp_certificate,omitempty" description:"SAML IdP signing certificate (PEM)"`
	EntityID          string            `json:"entity_id,omitempty" description:"SP EntityID override"`
	ACSURL            string            `json:"acs_url,omitempty" description:"SP ACS URL override"`
	SignRequests      *bool             `json:"sign_requests,omitempty" description:"Toggle signing outbound AuthnRequests"`
	AttributeMappings map[string]string `json:"attribute_mappings,omitempty" description:"Replace the SAML attribute → user field map"`

	Active *bool `json:"active,omitempty" description:"Activate/deactivate the connection without deleting it"`
}

// AdminDeleteConnectionResponse mirrors the StatusResponse shape used
// elsewhere in the admin surface so the SDK can model deletes
// uniformly.
type AdminDeleteConnectionResponse struct {
	Status string `json:"status"`
}

// handleAdminListConnections returns every connection registered on
// the target App. Connections do not include ClientSecret (the
// Connection struct's `json:"-"` tag strips it) so this endpoint is
// safe to surface to authenticated workspace admins.
func (p *Plugin) handleAdminListConnections(ctx forge.Context, req *AdminListConnectionsRequest) (*AdminListConnectionsResponse, error) {
	if p.ssoStore == nil {
		return nil, forge.InternalError(fmt.Errorf("sso plugin: store not wired"))
	}
	if strings.TrimSpace(req.AppID) == "" {
		return nil, forge.BadRequest("app_id is required")
	}
	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid app_id: %v", err))
	}
	conns, err := p.ssoStore.ListConnections(ctx.Context(), appID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("sso: list connections: %w", err))
	}
	if conns == nil {
		conns = []*Connection{}
	}
	return &AdminListConnectionsResponse{Connections: conns}, nil
}

// handleAdminGetConnection returns a single connection by ID.
func (p *Plugin) handleAdminGetConnection(ctx forge.Context, req *AdminConnectionPathRequest) (*Connection, error) {
	if p.ssoStore == nil {
		return nil, forge.InternalError(fmt.Errorf("sso plugin: store not wired"))
	}
	connID, err := id.ParseSSOConnectionID(req.ConnectionID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid connection_id: %v", err))
	}
	conn, err := p.ssoStore.GetConnection(ctx.Context(), connID)
	if err != nil {
		return nil, forge.NotFound("sso connection not found")
	}
	return conn, nil
}

// handleAdminUpdateConnection modifies an existing connection. The
// app_id and protocol are immutable — moving a connection between
// apps would orphan the inbound /v1/sso/:provider routes; switching
// protocols would orphan the OIDC vs SAML field set.
func (p *Plugin) handleAdminUpdateConnection(ctx forge.Context, req *AdminUpdateConnectionRequest) (*Connection, error) {
	if p.ssoStore == nil {
		return nil, forge.InternalError(fmt.Errorf("sso plugin: store not wired"))
	}
	connID, err := id.ParseSSOConnectionID(req.ConnectionID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid connection_id: %v", err))
	}
	conn, err := p.ssoStore.GetConnection(ctx.Context(), connID)
	if err != nil {
		return nil, forge.NotFound("sso connection not found")
	}

	if v := strings.TrimSpace(req.Provider); v != "" {
		conn.Provider = v
	}
	if v := strings.TrimSpace(req.Domain); v != "" {
		conn.Domain = v
	}
	if v := strings.TrimSpace(req.Issuer); v != "" {
		conn.Issuer = v
	}
	if v := strings.TrimSpace(req.ClientID); v != "" {
		conn.ClientID = v
	}
	if req.ClientSecret != "" {
		conn.ClientSecret = req.ClientSecret
	}
	if v := strings.TrimSpace(req.MetadataURL); v != "" {
		conn.MetadataURL = v
	}
	if v := strings.TrimSpace(req.IDPMetadataXML); v != "" {
		conn.IDPMetadataXML = v
	}
	if v := strings.TrimSpace(req.IDPSSOURL); v != "" {
		conn.IDPSSOURL = v
	}
	if v := strings.TrimSpace(req.IDPCertificate); v != "" {
		conn.IDPCertificate = v
	}
	if v := strings.TrimSpace(req.EntityID); v != "" {
		conn.EntityID = v
	}
	if v := strings.TrimSpace(req.ACSURL); v != "" {
		conn.ACSURL = v
	}
	if req.SignRequests != nil {
		conn.SignRequests = *req.SignRequests
	}
	if req.AttributeMappings != nil {
		conn.AttributeMappings = req.AttributeMappings
	}
	if req.Active != nil {
		conn.Active = *req.Active
	}
	conn.UpdatedAt = time.Now()

	if err := p.ssoStore.UpdateConnection(ctx.Context(), conn); err != nil {
		return nil, forge.InternalError(fmt.Errorf("sso: update connection: %w", err))
	}
	return conn, nil
}

// handleAdminDeleteConnection hard-deletes the connection. Use the
// PUT endpoint with `active=false` if a soft-disable is preferred.
func (p *Plugin) handleAdminDeleteConnection(ctx forge.Context, req *AdminConnectionPathRequest) (*AdminDeleteConnectionResponse, error) {
	if p.ssoStore == nil {
		return nil, forge.InternalError(fmt.Errorf("sso plugin: store not wired"))
	}
	connID, err := id.ParseSSOConnectionID(req.ConnectionID)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid connection_id: %v", err))
	}
	if err := p.ssoStore.DeleteConnection(ctx.Context(), connID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("sso: delete connection: %w", err))
	}
	return &AdminDeleteConnectionResponse{Status: "deleted"}, nil
}
