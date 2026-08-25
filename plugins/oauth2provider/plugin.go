// Package oauth2provider implements an OAuth2 authorization server plugin for AuthSome.
// It supports Authorization Code + PKCE (RFC 7636), Client Credentials grants,
// token revocation, OIDC userinfo, and OpenID Connect discovery.
package oauth2provider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apitypes"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/tokenformat"

	"github.com/xraph/grove/migrate"

	"golang.org/x/crypto/bcrypt"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.RouteProvider     = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
)

// Config configures the OAuth2 provider plugin.
type Config struct {
	// Issuer is the OAuth2 issuer URL (e.g. "https://auth.example.com").
	Issuer string

	// AuthCodeTTL is the lifetime of authorization codes (default: 10 minutes).
	AuthCodeTTL time.Duration

	// AccessTokenTTL is the lifetime of access tokens (default: 1 hour).
	AccessTokenTTL time.Duration

	// DeviceCodeTTL is the lifetime of device authorization codes (default: 10 minutes).
	DeviceCodeTTL time.Duration

	// DeviceCodeInterval is the minimum polling interval in seconds (default: 5).
	DeviceCodeInterval int

	// VerificationURI is the customizable user verification URL for the device flow.
	// If empty, defaults to "{issuer}/v1/oauth/device".
	// Set this to a custom URL (e.g. "https://myapp.com/device") when using
	// an external UI like authsome-ui to host the verification page.
	VerificationURI string

	// ConsentGate, if set, is consulted before every authorization code is
	// issued. See SetConsentGate for post-construction wiring.
	ConsentGate ConsentGate
}

// Plugin is the OAuth2 provider plugin.
type Plugin struct {
	config      Config
	store       store.Store
	oauth2Store Store
	logger      log.Logger
	engine      plugin.Engine
	consentGate ConsentGate
}

// New creates a new OAuth2 provider plugin.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.AuthCodeTTL == 0 {
		c.AuthCodeTTL = 10 * time.Minute
	}
	if c.AccessTokenTTL == 0 {
		c.AccessTokenTTL = time.Hour
	}
	if c.DeviceCodeTTL == 0 {
		c.DeviceCodeTTL = 10 * time.Minute
	}
	if c.DeviceCodeInterval == 0 {
		c.DeviceCodeInterval = 5
	}
	return &Plugin{config: c, consentGate: c.ConsentGate}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "oauth2provider" }

// OnInit captures dependencies from the engine.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.store = engine.Store()
	p.logger = engine.Logger()
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	// Create a persistent OAuth2 store from the engine's database reference.
	// Falls back to in-memory if no database is available.
	if p.oauth2Store == nil {
		if db := engine.DB(); db != nil {
			switch db.Driver().Name() {
			case "pg":
				p.oauth2Store = NewPostgresStore(db)
			case "sqlite":
				p.oauth2Store = NewSqliteStore(db)
			case "mongo":
				p.oauth2Store = NewMongoStore(db)
			}
		}
	}
	if p.oauth2Store == nil {
		p.oauth2Store = NewMemoryStore()
	}

	return nil
}

// MigrationGroups returns the OAuth2 migration groups for the given driver.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg", "postgres":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite", "sqlite3":
		return []*migrate.Group{SqliteMigrations}
	default:
		return nil
	}
}

// SetStore allows direct store injection for testing.
func (p *Plugin) SetStore(s store.Store) { p.store = s }

// SetOAuth2Store allows direct OAuth2 store injection for testing.
func (p *Plugin) SetOAuth2Store(s Store) { p.oauth2Store = s }

// RegisterRoutes registers OAuth2 provider HTTP endpoints.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Public OAuth2 endpoints
	g := router.Group("/v1/oauth", forge.WithGroupTags("OAuth2"))

	if err := g.GET("/authorize", p.handleAuthorize,
		forge.WithSummary("OAuth2 Authorization"),
		forge.WithDescription("Authorization endpoint for the OAuth2 authorization code flow."),
		forge.WithOperationID("oauth2Authorize"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/token", p.handleToken,
		forge.WithSummary("OAuth2 Token"),
		forge.WithDescription("Token endpoint for exchanging authorization codes or client credentials for access tokens."),
		forge.WithOperationID("oauth2Token"),
		forge.WithResponseSchema(http.StatusOK, "Token response", TokenResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/revoke", p.handleRevoke,
		forge.WithSummary("Revoke token"),
		forge.WithDescription("Revokes an access or refresh token."),
		forge.WithOperationID("oauth2Revoke"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/userinfo", p.handleUserInfo,
		forge.WithSummary("OIDC UserInfo"),
		forge.WithDescription("Returns claims about the authenticated user."),
		forge.WithOperationID("oauth2UserInfo"),
		forge.WithResponseSchema(http.StatusOK, "UserInfo", UserInfo{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Device Authorization Grant (RFC 8628) — public endpoint
	if err := g.POST("/device/authorize", p.handleDeviceAuthorize,
		forge.WithSummary("Device Authorization"),
		forge.WithDescription("Device authorization endpoint (RFC 8628). Returns a device_code and user_code for device/CLI authentication."),
		forge.WithOperationID("oauth2DeviceAuthorize"),
		forge.WithResponseSchema(http.StatusOK, "Device authorization response", DeviceAuthResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Authenticated OAuth2 endpoints — require a logged-in user.
	// The extension's global AuthMiddleware populates the user context;
	// RequireAuthMiddleware blocks if no user was resolved.
	// SessionGuard rather than reaching through p.engine directly: RegisterRoutes
	// would nil-deref when called before OnInit, which also made the plugin
	// impossible to exercise in a test without a full engine.
	authG := router.Group("/v1/oauth",
		forge.WithGroupTags("OAuth2"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
	)

	if err := authG.POST("/device/complete", p.handleDeviceComplete,
		forge.WithSummary("Complete device authorization"),
		forge.WithDescription("Approve or deny a device authorization request. Requires authenticated user. Used by external verification UIs (e.g. authsome-ui)."),
		forge.WithOperationID("oauth2DeviceComplete"),
		forge.WithResponseSchema(http.StatusOK, "Device completion response", DeviceCompleteResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Well-known OIDC discovery
	if err := router.GET("/.well-known/openid-configuration", p.handleDiscovery,
		forge.WithSummary("OpenID Connect Discovery"),
		forge.WithOperationID("oidcDiscovery"),
		forge.WithTags("OAuth2"),
	); err != nil {
		return err
	}

	// Admin endpoints for client management. Gated behind a session plus
	// manage/oauth2_client — a caller who can mint clients can register an
	// arbitrary redirect_uri and harvest codes for any user.
	admin := router.Group("/v1/admin/oauth",
		forge.WithGroupTags("OAuth2 Admin"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.AdminGuard(p.engine, "manage", "oauth2_client")...),
	)

	if err := admin.POST("/clients", p.handleCreateClient,
		forge.WithSummary("Create OAuth2 client"),
		forge.WithOperationID("createOAuth2Client"),
		forge.WithRequestSchema(CreateClientRequest{}),
		forge.WithResponseSchema(http.StatusCreated, "Client created", CreateClientResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := admin.GET("/clients", p.handleListClients,
		forge.WithSummary("List OAuth2 clients"),
		forge.WithOperationID("listOAuth2Clients"),
		forge.WithResponseSchema(http.StatusOK, "Clients", ListClientsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return admin.DELETE("/clients/:clientId", p.handleDeleteClient,
		forge.WithSummary("Delete OAuth2 client"),
		forge.WithOperationID("deleteOAuth2Client"),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// AuthorizeRequest is the OAuth2 authorization request.
type AuthorizeRequest struct {
	ResponseType string `query:"response_type"`
	ClientID     string `query:"client_id"`
	// Optional per RFC 6749 §4.1.1 — when omitted the client must have exactly
	// one registered URI, which resolveRedirectURI selects. Marking it
	// required here rejected such requests at the binder, before the handler
	// could apply that rule.
	RedirectURI         string `query:"redirect_uri,omitempty"`
	Scope               string `query:"scope,omitempty"`
	State               string `query:"state,omitempty"`
	CodeChallenge       string `query:"code_challenge,omitempty"`
	CodeChallengeMethod string `query:"code_challenge_method,omitempty"`
}

// TokenRequest is the OAuth2 token request.
type TokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type"`
	Code         string `json:"code,omitempty" form:"code"`
	RedirectURI  string `json:"redirect_uri,omitempty" form:"redirect_uri"`
	ClientID     string `json:"client_id,omitempty" form:"client_id"`
	ClientSecret string `json:"client_secret,omitempty" form:"client_secret"`
	CodeVerifier string `json:"code_verifier,omitempty" form:"code_verifier"`
	DeviceCode   string `json:"device_code,omitempty" form:"device_code"`
}

// RevokeRequest is the OAuth2 revocation request.
type RevokeRequest struct {
	Token         string `json:"token" form:"token"`
	TokenTypeHint string `json:"token_type_hint,omitempty" form:"token_type_hint"`
}

// UserInfoRequest is empty (user is determined from bearer token).
type UserInfoRequest struct{}

// DiscoveryRequest is empty.
type DiscoveryRequest struct{}

// DiscoveryResponse is the OpenID Connect discovery document.
type DiscoveryResponse struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	DeviceAuthorizationEndpoint       string   `json:"device_authorization_endpoint"`
	JWKSURI                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

// CreateClientRequest is the admin request to create an OAuth2 client.
type CreateClientRequest struct {
	AppID        string   `json:"app_id"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes,omitempty"`
	GrantTypes   []string `json:"grant_types,omitempty"`
	Public       bool     `json:"public,omitempty"`
}

// CreateClientResponse is returned when an OAuth2 client is created.
type CreateClientResponse struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"` // Only returned once at creation
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grant_types"`
	Public       bool     `json:"public"`
}

// ListClientsRequest is the request to list OAuth2 clients.
type ListClientsRequest struct {
	AppID string `query:"app_id"`
}

// ListClientsResponse is the response listing OAuth2 clients.
type ListClientsResponse struct {
	Clients []*OAuth2Client `json:"clients"`
}

// DeleteClientRequest deletes a client by internal ID.
type DeleteClientRequest struct {
	ClientID string `param:"clientId"`
}

// DeleteClientResponse is the response after deleting a client.
type DeleteClientResponse struct {
	Status string `json:"status"`
}

// DeviceAuthRequest is the device authorization request (RFC 8628 Section 3.1).
type DeviceAuthRequest struct {
	ClientID string `json:"client_id" form:"client_id"`
	Scope    string `json:"scope,omitempty" form:"scope,omitempty"`
}

// DeviceAuthResponse is the device authorization response (RFC 8628 Section 3.2).
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// DeviceCompleteRequest allows an authenticated user to approve or deny a device code.
type DeviceCompleteRequest struct {
	UserCode string `json:"user_code" form:"user_code"`
	Action   string `json:"action" form:"action"` // "approve" or "deny"
}

// DeviceCompleteResponse is the response after completing device authorization.
type DeviceCompleteResponse struct {
	Status string `json:"status"` // "authorized" or "denied"
}

// OAuth2Error is an RFC 6749 / RFC 8628 error response.
type OAuth2Error struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// OAuth2HTTPError wraps OAuth2 error codes as a framework-compatible HTTP error.
// This allows RFC 8628 device flow errors to flow through the Forge framework's
// error handler while maintaining the OAuth2 JSON error response format.
type OAuth2HTTPError struct {
	httpStatus  int
	oauthError  string
	description string
}

func (e *OAuth2HTTPError) Error() string   { return e.description }
func (e *OAuth2HTTPError) StatusCode() int { return e.httpStatus }
func (e *OAuth2HTTPError) ResponseBody() any {
	return &OAuth2Error{
		Error:            e.oauthError,
		ErrorDescription: e.description,
	}
}

// newOAuth2Error creates an OAuth2HTTPError for device flow error responses.
func newOAuth2Error(status int, errorCode, description string) *OAuth2HTTPError { //nolint:unparam // status kept for API flexibility
	return &OAuth2HTTPError{
		httpStatus:  status,
		oauthError:  errorCode,
		description: description,
	}
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleAuthorize(ctx forge.Context, req *AuthorizeRequest) (*apitypes.Empty, error) {
	if req.ResponseType != "code" {
		return nil, forge.BadRequest("unsupported response_type; use 'code'")
	}
	if req.ClientID == "" {
		return nil, forge.BadRequest("client_id required")
	}

	// Validate client.
	client, err := p.oauth2Store.GetClient(ctx.Context(), req.ClientID)
	if err != nil {
		return nil, forge.BadRequest("invalid client_id")
	}

	// Resolve the redirect URI. Omitting it is only legal when exactly one is
	// registered, in which case that one is used — echoing back the empty
	// string would produce a relative redirect that never reaches the client.
	redirectURI, err := p.resolveRedirectURI(client, req.RedirectURI)
	if err != nil {
		return nil, err
	}

	// The client must be registered for this grant. GrantTypes was recorded at
	// registration but never enforced, so any client could run any flow.
	if !clientAllowsGrant(client, "authorization_code") {
		return nil, forge.BadRequest("client is not authorized for the authorization_code grant")
	}

	// Confine the request to the scopes the client was registered with.
	// Without this the code carries whatever the caller asked for, so a client
	// limited to "profile" could mint a token for any scope it names.
	scopes, err := resolveScopes(client, req.Scope)
	if err != nil {
		return nil, err
	}

	// PKCE is mandatory for public clients (RFC 8252 §8.1): they hold no
	// secret, so an attacker who intercepts the code on the redirect — a
	// custom-scheme hijack, a shared browser — can redeem it without one.
	// Enforcing only when a challenge happens to be present lets a client opt
	// out of its own protection by simply omitting it.
	if client.Public {
		if req.CodeChallenge == "" {
			return nil, forge.BadRequest("code_challenge required (PKCE) for public clients")
		}
		// "plain" leaves the verifier recoverable anywhere the challenge is
		// observable — the authorize URL lands in history, logs, and Referer.
		if req.CodeChallengeMethod != "" && req.CodeChallengeMethod != "S256" {
			return nil, forge.BadRequest("code_challenge_method must be S256 for public clients")
		}
	}

	// Require authenticated user.
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required to authorize")
	}

	// Give a registered gate (e.g. an agent-delegation policy) a chance to
	// veto the authorization before a code is issued. orgID is whatever the
	// session carries; it may be the zero value when the session has none.
	orgID, _ := middleware.OrgIDFrom(ctx.Context())
	if gateErr := p.EvaluateConsent(ctx.Context(), req.ClientID, userID, orgID, client.AppID, scopes); gateErr != nil {
		return nil, gateErr
	}

	// Generate authorization code.
	codeStr, err := generateSecureToken(32)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate auth code: %w", err))
	}

	authCode := &AuthorizationCode{
		ID:                  id.NewAuthCodeID(),
		Code:                codeStr,
		ClientID:            req.ClientID,
		UserID:              userID,
		AppID:               client.AppID,
		RedirectURI:         redirectURI,
		Scopes:              scopes,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(p.config.AuthCodeTTL),
		CreatedAt:           time.Now(),
	}

	if createErr := p.oauth2Store.CreateAuthCode(ctx.Context(), authCode); createErr != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: store auth code: %w", createErr))
	}

	redirectURL, err := buildRedirect(redirectURI, codeStr, req.State)
	if err != nil {
		return nil, err
	}
	return nil, ctx.Redirect(http.StatusFound, redirectURL)
}

// buildRedirect appends the code and state to the client's redirect URI.
//
// Concatenating "?code=..." breaks a registered URI that already carries a
// query string, and an unescaped state can inject further parameters, so both
// values go through net/url.
func buildRedirect(redirectURI, code, state string) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", forge.BadRequest("invalid redirect_uri")
	}
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// clientAllowsGrant reports whether the client is registered for grantType.
// An empty GrantTypes list is treated as authorization_code only, matching the
// default applied at registration.
func clientAllowsGrant(client *OAuth2Client, grantType string) bool {
	if len(client.GrantTypes) == 0 {
		return grantType == "authorization_code"
	}
	for _, g := range client.GrantTypes {
		if g == grantType {
			return true
		}
	}
	return false
}

// resolveScopes intersects the requested scopes with the client's registered
// set, rejecting anything outside it (RFC 6749 §4.1.2.1 invalid_scope). An
// empty request yields the client's full registered set.
func resolveScopes(client *OAuth2Client, requested string) ([]string, error) {
	fields := strings.Fields(requested)
	if len(fields) == 0 {
		return append([]string(nil), client.Scopes...), nil
	}
	allowed := make(map[string]struct{}, len(client.Scopes))
	for _, s := range client.Scopes {
		allowed[s] = struct{}{}
	}
	out := make([]string, 0, len(fields))
	for _, s := range fields {
		if _, ok := allowed[s]; !ok {
			return nil, forge.BadRequest(fmt.Sprintf("scope %q is not registered for this client", s))
		}
		out = append(out, s)
	}
	return out, nil
}

func (p *Plugin) handleToken(ctx forge.Context, req *TokenRequest) (*TokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return p.handleAuthorizationCodeGrant(ctx, req)
	case "client_credentials":
		return p.handleClientCredentialsGrant(ctx, req)
	case "urn:ietf:params:oauth:grant-type:device_code":
		return p.handleDeviceCodeGrant(ctx, req)
	default:
		return nil, forge.BadRequest("unsupported grant_type")
	}
}

func (p *Plugin) handleAuthorizationCodeGrant(ctx forge.Context, req *TokenRequest) (*TokenResponse, error) {
	if req.Code == "" {
		return nil, forge.BadRequest("code required")
	}

	// Look up the authorization code.
	authCode, err := p.oauth2Store.GetAuthCode(ctx.Context(), req.Code)
	if err != nil {
		return nil, forge.BadRequest("invalid authorization code")
	}
	if authCode.Consumed {
		return nil, forge.BadRequest("authorization code already used")
	}
	if time.Now().After(authCode.ExpiresAt) {
		return nil, forge.BadRequest("authorization code expired")
	}

	// Validate client.
	client, err := p.oauth2Store.GetClient(ctx.Context(), authCode.ClientID)
	if err != nil {
		return nil, forge.BadRequest("invalid client")
	}

	// The presented client_id must be the one the code was issued to.
	// Otherwise a client can redeem a code minted for a different client.
	if req.ClientID != "" && req.ClientID != authCode.ClientID {
		return nil, forge.BadRequest("client_id does not match the authorization code")
	}

	// Validate client authentication (confidential clients).
	if !client.Public {
		if req.ClientSecret == "" {
			return nil, forge.Unauthorized("client_secret required for confidential clients")
		}
		if cmpErr := bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(req.ClientSecret)); cmpErr != nil {
			return nil, forge.Unauthorized("invalid client_secret")
		}
	}

	// The redirect_uri must match the one the code was issued against
	// (RFC 6749 §4.1.3). Skipping this breaks the binding between the leg that
	// obtained the code and the leg that redeems it, so a code leaked to a
	// different registered URI — or to an attacker who got one registered —
	// can still be exchanged here.
	if authCode.RedirectURI != "" && req.RedirectURI != authCode.RedirectURI {
		return nil, forge.BadRequest("redirect_uri does not match the authorization request")
	}

	// PKCE verification (RFC 7636). Public clients are required to have bound
	// a challenge at /authorize; re-assert it here so a code minted before
	// that rule existed, or through any other path, still cannot skip it.
	if client.Public && authCode.CodeChallenge == "" {
		return nil, forge.BadRequest("authorization code is not PKCE-bound")
	}
	if authCode.CodeChallenge != "" {
		if req.CodeVerifier == "" {
			return nil, forge.BadRequest("code_verifier required (PKCE)")
		}
		if !verifyPKCE(authCode.CodeChallenge, authCode.CodeChallengeMethod, req.CodeVerifier) {
			return nil, forge.BadRequest("invalid code_verifier")
		}
	}

	// Consume the code as a compare-and-set. Checking Consumed above and
	// writing here are two statements: without the store enforcing
	// consumed=false, two concurrent requests both pass the check and both get
	// tokens from one code.
	consumed, err := p.oauth2Store.ConsumeAuthCode(ctx.Context(), req.Code)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: consume auth code: %w", err))
	}
	if !consumed {
		return nil, forge.BadRequest("authorization code already used")
	}

	// Generate tokens.
	return p.issueTokens(ctx.Context(), client, authCode.UserID, authCode.AppID, authCode.Scopes)
}

func (p *Plugin) handleClientCredentialsGrant(ctx forge.Context, req *TokenRequest) (*TokenResponse, error) {
	if req.ClientID == "" || req.ClientSecret == "" {
		return nil, forge.BadRequest("client_id and client_secret required")
	}

	client, err := p.oauth2Store.GetClient(ctx.Context(), req.ClientID)
	if err != nil {
		return nil, forge.Unauthorized("invalid client")
	}
	if client.Public {
		return nil, forge.BadRequest("client_credentials grant not allowed for public clients")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(req.ClientSecret)); err != nil {
		return nil, forge.Unauthorized("invalid client_secret")
	}

	// Client credentials: no user, just issue an app-level token.
	return p.issueClientToken(ctx.Context(), client)
}

func (p *Plugin) handleRevoke(ctx forge.Context, req *RevokeRequest) (*apitypes.Empty, error) {
	if req.Token == "" {
		return nil, forge.BadRequest("token required")
	}

	// Try to revoke as a session token by looking it up first.
	sess, err := p.store.GetSessionByToken(ctx.Context(), req.Token)
	if err == nil {
		if delErr := p.store.DeleteSession(ctx.Context(), sess.ID); delErr != nil {
			p.logger.Debug("oauth2: failed to delete session",
				log.String("error", delErr.Error()),
			)
		}
	}

	// RFC 7009: always return 200 regardless of whether the token was found.
	return nil, ctx.JSON(http.StatusOK, map[string]string{"status": "revoked"})
}

func (p *Plugin) handleUserInfo(ctx forge.Context, _ *UserInfoRequest) (*UserInfo, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	u, err := p.store.GetUser(ctx.Context(), userID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: get user: %w", err))
	}

	return &UserInfo{
		Sub:           u.ID.String(),
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Name:          u.Name(),
		Phone:         u.Phone,
	}, nil
}

func (p *Plugin) handleDiscovery(_ forge.Context, _ *DiscoveryRequest) (*DiscoveryResponse, error) {
	issuer := p.config.Issuer
	if issuer == "" {
		issuer = "https://localhost"
	}

	return &DiscoveryResponse{
		Issuer:                            issuer,
		AuthorizationEndpoint:             issuer + "/v1/oauth/authorize",
		TokenEndpoint:                     issuer + "/v1/oauth/token",
		UserinfoEndpoint:                  issuer + "/v1/oauth/userinfo",
		RevocationEndpoint:                issuer + "/v1/oauth/revoke",
		DeviceAuthorizationEndpoint:       issuer + "/v1/oauth/device/authorize",
		JWKSURI:                           issuer + "/.well-known/jwks.json",
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "client_credentials", "urn:ietf:params:oauth:grant-type:device_code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{"RS256", "ES256"},
		ScopesSupported:                   []string{"openid", "profile", "email", "phone"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_post", "client_secret_basic"},
		CodeChallengeMethodsSupported:     []string{"S256", "plain"},
	}, nil
}

// ──────────────────────────────────────────────────
// Admin Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleCreateClient(ctx forge.Context, req *CreateClientRequest) (*CreateClientResponse, error) {
	if req.Name == "" {
		return nil, forge.BadRequest("name required")
	}
	if req.AppID == "" {
		return nil, forge.BadRequest("app_id required")
	}
	if len(req.RedirectURIs) == 0 && !req.Public {
		return nil, forge.BadRequest("redirect_uris required for confidential clients")
	}

	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	// Generate client credentials.
	clientIDStr, err := generateSecureToken(16)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate client_id: %w", err))
	}

	var rawSecret string
	var hashedSecret string
	if !req.Public {
		rawSecret, err = generateSecureToken(32)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: generate client_secret: %w", err))
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: hash client_secret: %w", err))
		}
		hashedSecret = string(hash)
	}

	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	client := &OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		Name:         req.Name,
		ClientID:     clientIDStr,
		ClientSecret: hashedSecret,
		RedirectURIs: req.RedirectURIs,
		Scopes:       scopes,
		GrantTypes:   grantTypes,
		Public:       req.Public,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := p.oauth2Store.CreateClient(ctx.Context(), client); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: create client: %w", err))
	}

	resp := &CreateClientResponse{
		ID:           client.ID.String(),
		ClientID:     client.ClientID,
		Name:         client.Name,
		RedirectURIs: client.RedirectURIs,
		Scopes:       client.Scopes,
		GrantTypes:   client.GrantTypes,
		Public:       client.Public,
	}
	// Only return the raw secret once.
	if rawSecret != "" {
		resp.ClientSecret = rawSecret
	}

	return resp, nil
}

func (p *Plugin) handleListClients(ctx forge.Context, req *ListClientsRequest) (*ListClientsResponse, error) {
	if req.AppID == "" {
		return nil, forge.BadRequest("app_id query parameter required")
	}

	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	clients, err := p.oauth2Store.ListClients(ctx.Context(), appID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: list clients: %w", err))
	}
	if clients == nil {
		clients = []*OAuth2Client{}
	}

	return &ListClientsResponse{Clients: clients}, nil
}

func (p *Plugin) handleDeleteClient(ctx forge.Context, req *DeleteClientRequest) (*DeleteClientResponse, error) {
	if req.ClientID == "" {
		return nil, forge.BadRequest("client ID required")
	}

	clientID, err := id.ParseOAuth2ClientID(req.ClientID)
	if err != nil {
		return nil, forge.BadRequest("invalid client ID")
	}

	if err := p.oauth2Store.DeleteClient(ctx.Context(), clientID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: delete client: %w", err))
	}

	return &DeleteClientResponse{Status: "deleted"}, nil
}

// ──────────────────────────────────────────────────
// Token Issuance
// ──────────────────────────────────────────────────

func (p *Plugin) issueTokens(ctx context.Context, _ *OAuth2Client, userID id.UserID, appID id.AppID, scopes []string) (*TokenResponse, error) {
	// Resolve session config for the app.
	sessCfg := account.SessionConfig{
		TokenTTL:        p.config.AccessTokenTTL,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}
	if p.engine != nil {
		sessCfg = p.engine.SessionConfigForApp(ctx, appID)
	}

	sess, err := account.NewSession(appID, userID, sessCfg)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: create session: %w", err))
	}

	// Bind session to the app's default environment so the FK constraint
	// on authsome_sessions.env_id is satisfied.
	if env, envErr := p.store.GetDefaultEnvironment(ctx, appID); envErr == nil && env != nil {
		sess.EnvID = env.ID
	}

	// If token format is JWT, generate a JWT access token.
	if p.engine != nil {
		tokFmt := p.engine.TokenFormatForApp(appID.String())
		if tokFmt.Name() == "jwt" {
			jwtToken, err := tokFmt.GenerateAccessToken(tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     appID.String(),
				SessionID: sess.ID.String(),
				Scopes:    scopes,
				IssuedAt:  sess.CreatedAt,
				ExpiresAt: sess.ExpiresAt,
			})
			if err != nil {
				return nil, forge.InternalError(fmt.Errorf("oauth2: generate JWT: %w", err))
			}
			sess.Token = jwtToken
		}
	}

	if err := p.store.CreateSession(ctx, sess); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: save session: %w", err))
	}

	return &TokenResponse{
		AccessToken:  sess.Token,
		TokenType:    "Bearer",
		ExpiresIn:    int(time.Until(sess.ExpiresAt).Seconds()),
		RefreshToken: sess.RefreshToken,
		Scope:        strings.Join(scopes, " "),
	}, nil
}

func (p *Plugin) issueClientToken(ctx context.Context, client *OAuth2Client) (*TokenResponse, error) {
	// Client credentials: create a session with no user.
	sessCfg := account.SessionConfig{
		TokenTTL:        p.config.AccessTokenTTL,
		RefreshTokenTTL: 0, // No refresh token for client credentials.
	}

	// Use an empty user ID for machine-to-machine tokens.
	sess, err := account.NewSession(client.AppID, id.Nil, sessCfg)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: create client session: %w", err))
	}

	// Bind session to the app's default environment so the FK constraint
	// on authsome_sessions.env_id is satisfied.
	if env, envErr := p.store.GetDefaultEnvironment(ctx, client.AppID); envErr == nil && env != nil {
		sess.EnvID = env.ID
	}

	if err := p.store.CreateSession(ctx, sess); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: save client session: %w", err))
	}

	return &TokenResponse{
		AccessToken: sess.Token,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(sess.ExpiresAt).Seconds()),
		Scope:       strings.Join(client.Scopes, " "),
	}, nil
}

// ──────────────────────────────────────────────────
// Device Authorization Grant (RFC 8628)
// ──────────────────────────────────────────────────

// deviceCodeGrantType is the full IANA grant type for device authorization.
const deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"

func (p *Plugin) handleDeviceAuthorize(ctx forge.Context, req *DeviceAuthRequest) (*DeviceAuthResponse, error) {
	if req.ClientID == "" {
		return nil, forge.BadRequest("client_id required")
	}

	// Validate client.
	client, err := p.oauth2Store.GetClient(ctx.Context(), req.ClientID)
	if err != nil {
		return nil, forge.BadRequest("invalid client_id")
	}

	// Check that client supports device_code grant type.
	if !p.clientSupportsGrant(client, deviceCodeGrantType) && !p.clientSupportsGrant(client, "device_code") {
		return nil, forge.BadRequest("client does not support device authorization grant")
	}

	// Generate device code (256-bit, hex-encoded).
	deviceCodeStr, err := generateSecureToken(32)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate device_code: %w", err))
	}

	// Generate human-readable user code (XXXX-XXXX format).
	userCodeStr, err := generateUserCode()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate user_code: %w", err))
	}

	// Compute verification URI.
	verificationURI := p.config.VerificationURI
	if verificationURI == "" {
		issuer := p.config.Issuer
		if issuer == "" {
			issuer = "https://localhost"
		}
		verificationURI = issuer + "/v1/oauth/device"
	}

	scopes := strings.Fields(req.Scope)

	dc := &DeviceCode{
		ID:              id.NewDeviceCodeID(),
		DeviceCode:      deviceCodeStr,
		UserCode:        userCodeStr,
		ClientID:        req.ClientID,
		AppID:           client.AppID,
		Scopes:          scopes,
		VerificationURI: verificationURI,
		ExpiresAt:       time.Now().Add(p.config.DeviceCodeTTL),
		Interval:        p.config.DeviceCodeInterval,
		Status:          DeviceCodeStatusPending,
		CreatedAt:       time.Now(),
	}

	if err := p.oauth2Store.CreateDeviceCode(ctx.Context(), dc); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: store device code: %w", err))
	}

	resp := &DeviceAuthResponse{
		DeviceCode:              deviceCodeStr,
		UserCode:                userCodeStr,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURI + "?code=" + userCodeStr,
		ExpiresIn:               int(p.config.DeviceCodeTTL.Seconds()),
		Interval:                p.config.DeviceCodeInterval,
	}

	return resp, nil
}

func (p *Plugin) handleDeviceCodeGrant(ctx forge.Context, req *TokenRequest) (*TokenResponse, error) {
	if req.DeviceCode == "" {
		return nil, forge.BadRequest("device_code required")
	}
	if req.ClientID == "" {
		return nil, forge.BadRequest("client_id required")
	}

	// Look up the device code.
	dc, err := p.oauth2Store.GetDeviceCodeByDeviceCode(ctx.Context(), req.DeviceCode)
	if err != nil {
		return nil, newOAuth2Error(http.StatusBadRequest, "invalid_grant", "invalid device_code")
	}

	// Validate client_id matches.
	if dc.ClientID != req.ClientID {
		return nil, newOAuth2Error(http.StatusBadRequest, "invalid_grant", "client_id mismatch")
	}

	// Check expiration.
	if time.Now().After(dc.ExpiresAt) {
		return nil, newOAuth2Error(http.StatusBadRequest, "expired_token", "the device code has expired")
	}

	// RFC 8628 Section 3.5: enforce polling interval (slow_down).
	now := time.Now()
	if !dc.LastPolledAt.IsZero() {
		minNextPoll := dc.LastPolledAt.Add(time.Duration(dc.Interval) * time.Second)
		if now.Before(minNextPoll) {
			// Client is polling too fast. Per RFC 8628, increase the interval by 5 seconds.
			dc.Interval += 5
			dc.LastPolledAt = now
			_ = p.oauth2Store.UpdateDeviceCode(ctx.Context(), dc) //nolint:errcheck // best-effort update
			return nil, newOAuth2Error(http.StatusBadRequest, "slow_down", "polling too frequently, please slow down")
		}
	}
	// Record this poll timestamp.
	dc.LastPolledAt = now

	// Check status.
	switch dc.Status {
	case DeviceCodeStatusPending:
		// Persist the updated LastPolledAt.
		_ = p.oauth2Store.UpdateDeviceCode(ctx.Context(), dc) //nolint:errcheck // best-effort update
		// RFC 8628 Section 3.5: authorization_pending is expected during polling.
		return nil, newOAuth2Error(http.StatusBadRequest, "authorization_pending", "the user has not yet completed authorization")

	case DeviceCodeStatusDenied:
		return nil, newOAuth2Error(http.StatusBadRequest, "access_denied", "the user denied the authorization request")

	case DeviceCodeStatusAuthorized:
		// Success! Issue tokens.
		client, err := p.oauth2Store.GetClient(ctx.Context(), dc.ClientID)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: get client for device code: %w", err))
		}

		// Mark as consumed before issuing tokens (one-time use).
		// If this update fails, do NOT issue tokens to prevent double-use.
		dc.Status = DeviceCodeStatusConsumed
		if err := p.oauth2Store.UpdateDeviceCode(ctx.Context(), dc); err != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: consume device code: %w", err))
		}

		return p.issueTokens(ctx.Context(), client, dc.UserID, dc.AppID, dc.Scopes)

	default:
		return nil, newOAuth2Error(http.StatusBadRequest, "invalid_grant", "unexpected device code status")
	}
}

func (p *Plugin) handleDeviceComplete(ctx forge.Context, req *DeviceCompleteRequest) (*DeviceCompleteResponse, error) {
	if req.UserCode == "" {
		return nil, forge.BadRequest("user_code required")
	}
	if req.Action != "approve" && req.Action != "deny" {
		return nil, forge.BadRequest("action must be 'approve' or 'deny'")
	}

	// Require authenticated user (resolved by the auth middleware on this group).
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required to complete device authorization")
	}

	// Normalize user code: accept both "ABCDEFGH" and "ABCD-EFGH" formats.
	// Stored format is XXXX-XXXX, so insert dash if missing.
	userCode := strings.ToUpper(strings.ReplaceAll(req.UserCode, "-", ""))
	if len(userCode) == 8 {
		userCode = userCode[:4] + "-" + userCode[4:]
	}

	// Look up device code by user code.
	dc, err := p.oauth2Store.GetDeviceCodeByUserCode(ctx.Context(), userCode)
	if err != nil {
		return nil, forge.NotFound("invalid or expired user code")
	}

	// Check expiration.
	if time.Now().After(dc.ExpiresAt) {
		return nil, forge.BadRequest("device code expired")
	}

	// Must be in pending state.
	if dc.Status != DeviceCodeStatusPending {
		return nil, forge.BadRequest("device code already " + dc.Status)
	}

	// Apply the user's decision.
	if req.Action == "approve" {
		// Same veto point as the authorization-code flow: a gate refusal
		// must block the approval, not be undone after the fact.
		orgID, _ := middleware.OrgIDFrom(ctx.Context())
		if gateErr := p.EvaluateConsent(ctx.Context(), dc.ClientID, userID, orgID, dc.AppID, dc.Scopes); gateErr != nil {
			return nil, gateErr
		}
		dc.Status = DeviceCodeStatusAuthorized
		dc.UserID = userID
	} else {
		dc.Status = DeviceCodeStatusDenied
	}

	if err := p.oauth2Store.UpdateDeviceCode(ctx.Context(), dc); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: update device code: %w", err))
	}

	return &DeviceCompleteResponse{Status: dc.Status}, nil
}

// clientSupportsGrant checks if a client has the given grant type registered.
func (p *Plugin) clientSupportsGrant(client *OAuth2Client, grantType string) bool {
	for _, gt := range client.GrantTypes {
		if gt == grantType {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// resolveRedirectURI returns the redirect URI this authorization will use.
//
// An omitted redirect_uri is only acceptable when the client registered
// exactly one, and resolves to that one. Anything supplied must match a
// registered entry exactly — no prefix or subpath matching, which would let a
// caller redirect the code to an attacker-controlled path on the same host.
func (p *Plugin) resolveRedirectURI(client *OAuth2Client, uri string) (string, error) {
	if uri == "" {
		if len(client.RedirectURIs) == 1 {
			return client.RedirectURIs[0], nil
		}
		return "", forge.BadRequest("redirect_uri required")
	}
	for _, u := range client.RedirectURIs {
		if u == uri {
			return uri, nil
		}
	}
	return "", forge.BadRequest("invalid redirect_uri")
}

// verifyPKCE validates a PKCE code_verifier against a code_challenge.
func verifyPKCE(challenge, method, verifier string) bool {
	switch method {
	case "S256", "":
		h := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(h[:])
		return computed == challenge
	case "plain":
		return verifier == challenge
	default:
		return false
	}
}

// generateSecureToken creates a cryptographically random hex token.
func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// userCodeCharset is a set of unambiguous consonant-like characters for user codes.
// Excludes vowels (to avoid words), and ambiguous chars (I, L, O, 0, 1).
const userCodeCharset = "BCDFGHJKMNPQRSTVWXYZ"

// generateUserCode creates a human-readable user code in XXXX-XXXX format.
// Uses rejection sampling to avoid modulo bias.
func generateUserCode() (string, error) {
	const n = len(userCodeCharset) // 20
	// Accept only random byte values below the largest multiple of n that fits in a byte.
	// This ensures uniform distribution across the charset.
	const maxAcceptable = (256 / n) * n // 240

	chars := make([]byte, 0, 8)
	buf := make([]byte, 1)
	for len(chars) < 8 {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		if int(buf[0]) < maxAcceptable {
			chars = append(chars, userCodeCharset[int(buf[0])%n])
		}
	}

	// Format: XXXX-XXXX
	return string(chars[:4]) + "-" + string(chars[4:]), nil
}
