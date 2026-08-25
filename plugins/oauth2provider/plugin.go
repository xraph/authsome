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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apitypes"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/tokenformat"

	"github.com/xraph/grove/migrate"

	"golang.org/x/crypto/bcrypt"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.RouteProvider     = (*Plugin)(nil)
	_ plugin.RootRouteProvider = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
)

// RateLimit bounds how often an endpoint may be called.
type RateLimit struct {
	Limit  int
	Window time.Duration
}

// ProtectedResource describes one RFC 9728 protected resource.
type ProtectedResource struct {
	Resource        string
	ScopesSupported []string
}

// defaultDynamicRegistrationScopes is the default Config.DynamicRegistrationScopes
// allowlist, applied in New when the embedder sets none. Both discovery
// documents advertise scopes_supported straight from
// p.config.DynamicRegistrationScopes (metadata.go), so this default is also,
// transitively, what a stock deployment advertises there. One list feeding
// both means a client that follows discovery never asks for a scope
// registration silently drops, and discovery never advertises a scope
// registration would refuse by default.
var defaultDynamicRegistrationScopes = []string{"openid", "profile", "email", "offline_access"}

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

	// DynamicRegistration enables RFC 7591 registration. Off by default:
	// an upgrade must never open a public registration endpoint on its own.
	DynamicRegistration bool

	// RegistrationAppID is the app dynamic clients belong to when the
	// request carries no resolvable publishable key. Leave empty on a
	// multi-tenant deployment so an unkeyed request is refused rather
	// than pooled into somebody's app.
	RegistrationAppID string

	// DynamicRegistrationScopes is the allowlist a dynamic client's
	// requested scopes are intersected against. Defaults to openid,
	// profile, email and offline_access.
	DynamicRegistrationScopes []string

	// RegistrationRateLimit caps POST /register per client IP.
	// Defaults to 10 per hour.
	RegistrationRateLimit RateLimit

	// ProtectedResources declares additional RFC 9728 resource identifiers,
	// keyed by the path suffix they are served under. AuthSome always
	// describes itself at the unsuffixed path regardless of this map.
	ProtectedResources map[string]ProtectedResource
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

	// regLimiter is a process-local fallback used for POST /register when
	// the engine has no rate limiter configured (extension.Config.RateLimit
	// defaults off, and most embedders never set one either). See
	// registrationLimiter.
	regLimiter  ratelimit.Limiter
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
	if len(c.DynamicRegistrationScopes) == 0 {
		c.DynamicRegistrationScopes = append([]string(nil), defaultDynamicRegistrationScopes...)
	}
	if c.RegistrationRateLimit.Limit == 0 {
		c.RegistrationRateLimit = RateLimit{Limit: 10, Window: time.Hour}
	}

	p := &Plugin{config: c, logger: log.NewNoopLogger(), consentGate: c.ConsentGate}
	return p
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

	// POST /register is unauthenticated and does two bcrypt hashes per
	// request, so it must never run unlimited. Most deployments never turn
	// on extension.Config.RateLimit (it defaults off), which would
	// otherwise leave engine.RateLimiter() nil and the endpoint's rate
	// limit middleware silently unattached. Fall back to a process-local
	// limiter instead — constructed once, here, not per request or per
	// route registration.
	if engine.RateLimiter() == nil {
		p.regLimiter = ratelimit.NewMemoryLimiter()
		p.logger.Warn("oauth2: no engine rate limiter configured; " +
			"dynamic client registration falls back to a process-local limiter, " +
			"so the cap is per-replica behind a load balancer")
	}

	return nil
}

// registrationLimiter returns the limiter POST /register should use. It
// prefers the engine's shared limiter and falls back to a process-local one
// otherwise — built once in OnInit, or lazily here for tests that construct
// a Plugin without calling OnInit at all, so the fallback never leaves the
// endpoint unlimited.
func (p *Plugin) registrationLimiter() ratelimit.Limiter {
	if p.engine != nil {
		if rl := p.engine.RateLimiter(); rl != nil {
			return rl
		}
	}
	if p.regLimiter == nil {
		p.regLimiter = ratelimit.NewMemoryLimiter()
	}
	return p.regLimiter
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

// SetDynamicRegistrationForTest toggles RFC 7591 registration after
// construction. Tests use it to prove the 7592 routes keep working once
// registration itself is closed.
func (p *Plugin) SetDynamicRegistrationForTest(enabled bool) {
	p.config.DynamicRegistration = enabled
}

// RegisterRoutes registers OAuth2 provider HTTP endpoints.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Public OAuth2 endpoints
	g := router.Group("/v1/oauth", forge.WithGroupTags("OAuth2"))

	if err := g.GET("/authorize", p.handleAuthorize,
		forge.WithSummary("OAuth2 Authorization"),
		forge.WithDescription("Authorization endpoint for the OAuth2 authorization code flow."),
		forge.WithOperationID("oauth2Authorize"),
		forge.WithQuerySchema(resourceQuery{}),
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

	// RFC 7591 dynamic client registration. Unauthenticated by design, so
	// it is rate limited by IP and returns 404 unless explicitly enabled.
	// The limiter always comes from registrationLimiter, never skipped:
	// most deployments never configure engine.RateLimiter(), and an
	// endpoint that mints credentials must not go unlimited just because
	// nobody opted into the shared limiter.
	regOpts := []forge.RouteOption{
		forge.WithSummary("Register OAuth2 client"),
		forge.WithDescription("RFC 7591 dynamic client registration."),
		forge.WithOperationID("oauth2RegisterClient"),
		forge.WithRequestSchema(RegisterClientRequest{}),
		forge.WithResponseSchema(http.StatusCreated, "Client registered", RegisterClientResponse{}),
		forge.WithErrorResponses(),
		forge.WithMiddleware(middleware.RateLimit(p.registrationLimiter(), middleware.RateLimitConfig{
			Limit:  p.config.RegistrationRateLimit.Limit,
			Window: p.config.RegistrationRateLimit.Window,
		})),
	}
	if err := g.POST("/register", p.handleRegisterClient, regOpts...); err != nil {
		return err
	}

	// RFC 7592 registration management. Registered unconditionally: turning
	// registration off must not strand clients that already registered.
	// The limiter always comes from registrationLimiter, for the same
	// reason POST /register does above — an engine-less test or a stock
	// deployment that never configures a shared limiter must still get
	// one, rather than these routes going unlimited.
	//
	// The key combines the caller's IP with the path's client_id rather
	// than either alone. Supplying a KeyFunc replaces
	// middleware.RateLimit's default IP-only key, it does not add to it —
	// so a client_id-only key (what this used to be) dropped IP throttling
	// entirely: one host varying the path segment got a fresh, unthrottled
	// bucket per client_id it guessed, and worse, anyone who learned a
	// real client_id could burn that specific bucket and lock its
	// legitimate owner out of GET/PUT/DELETE for the window. Keying on
	// both restores a real per-caller budget and fixes that lockout: an
	// attacker's requests now land in their own IP's bucket instead of
	// the victim's.
	//
	// What this key does NOT do: throttle enumeration across client_ids.
	// One IP still gets a fresh 6x budget for every distinct client_id it
	// tries, by design — client_id is 16 random bytes, not a secret worth
	// building a global-per-IP limiter around here, and bcrypt's own cost
	// already taxes each guess in authenticateRegistration. Do not read
	// this key as an anti-enumeration control; it only fixes who pays for
	// a given client_id's budget, not how many client_ids one caller may
	// try.
	//
	// middleware.ClientIP is the same trusted-proxy-aware helper the
	// default KeyFunc uses; reading a forwarding header here directly
	// would let a direct, untrusted client mint a fresh bucket per
	// request by spoofing it.
	manageOpts := func(extra ...forge.RouteOption) []forge.RouteOption {
		base := make([]forge.RouteOption, 0, 2+len(extra))
		base = append(base,
			forge.WithErrorResponses(),
			forge.WithMiddleware(middleware.RateLimit(p.registrationLimiter(), middleware.RateLimitConfig{
				Limit:  p.config.RegistrationRateLimit.Limit * 6,
				Window: p.config.RegistrationRateLimit.Window,
				KeyFunc: func(c forge.Context) string {
					return "oauth2-reg-manage:" + middleware.ClientIP(c.Request()) + ":" + c.Param("clientId")
				},
			})),
		)

		return append(base, extra...)
	}

	if err := g.GET("/register/:clientId", p.handleReadRegistration, manageOpts(
		forge.WithSummary("Read OAuth2 client registration"),
		forge.WithOperationID("oauth2ReadRegistration"),
		forge.WithResponseSchema(http.StatusOK, "Client information", RegisterClientResponse{}),
	)...); err != nil {
		return err
	}

	if err := g.PUT("/register/:clientId", p.handleUpdateRegistration, manageOpts(
		forge.WithSummary("Update OAuth2 client registration"),
		forge.WithOperationID("oauth2UpdateRegistration"),
		forge.WithRequestSchema(UpdateRegistrationRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Client information", RegisterClientResponse{}),
	)...); err != nil {
		return err
	}

	if err := g.DELETE("/register/:clientId", p.handleDeleteRegistration, manageOpts(
		forge.WithSummary("Delete OAuth2 client registration"),
		forge.WithOperationID("oauth2DeleteRegistration"),
		forge.WithNoContentResponse(),
	)...); err != nil {
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

	// Mirror onto the grouped router so an SDK client whose base URL
	// includes the mount prefix still resolves discovery.
	if err := router.GET("/.well-known/openid-configuration", p.handleDiscovery,
		forge.WithSummary("OpenID Connect Discovery"),
		forge.WithOperationID("oidcDiscoveryPrefixed"),
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

// RegisterRootRoutes registers discovery documents at the origin root.
// These cannot live on the grouped router: a client that only knows the
// host fetches https://host/.well-known/... with no prefix.
func (p *Plugin) RegisterRootRoutes(router forge.Router) error {
	return p.registerWellKnown(router)
}

// registerWellKnown mounts the discovery documents on the given router.
// Only the OIDC document is also mirrored onto the grouped router (see
// RegisterRoutes); the RFC 8414 and RFC 9728 documents are not, since a
// prefixed copy of either is undiscoverable and would only add duplicate
// OpenAPI entries.
func (p *Plugin) registerWellKnown(router forge.Router) error {
	if err := router.GET("/.well-known/openid-configuration", p.handleDiscovery,
		forge.WithSummary("OpenID Connect Discovery"),
		forge.WithOperationID("oidcDiscovery"),
		forge.WithTags("OAuth2"),
	); err != nil {
		return err
	}

	if err := router.GET("/.well-known/oauth-authorization-server", p.handleAuthServerMetadata,
		forge.WithSummary("OAuth2 Authorization Server Metadata"),
		forge.WithDescription("RFC 8414 authorization server metadata."),
		forge.WithOperationID("oauth2AuthServerMetadata"),
		forge.WithResponseSchema(http.StatusOK, "Metadata", AuthServerMetadata{}),
		forge.WithTags("OAuth2"),
	); err != nil {
		return err
	}

	if err := router.GET("/.well-known/oauth-protected-resource", p.handleProtectedResourceMetadata,
		forge.WithSummary("OAuth2 Protected Resource Metadata"),
		forge.WithDescription("RFC 9728 protected resource metadata."),
		forge.WithOperationID("oauth2ProtectedResourceMetadata"),
		forge.WithResponseSchema(http.StatusOK, "Metadata", ProtectedResourceMetadata{}),
		forge.WithTags("OAuth2"),
	); err != nil {
		return err
	}

	return router.GET("/.well-known/oauth-protected-resource/:resourcePath", p.handleScopedProtectedResourceMetadata,
		forge.WithSummary("OAuth2 Protected Resource Metadata (scoped)"),
		forge.WithOperationID("oauth2ScopedProtectedResourceMetadata"),
		forge.WithResponseSchema(http.StatusOK, "Metadata", ProtectedResourceMetadata{}),
		forge.WithTags("OAuth2"),
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
	// No resource field. go-utils' bindFormParam gained multi-value binding in
	// v1.1.7, but bindQueryParam still reads one value through c.Query, so a
	// []string query field silently keeps the first value and drops the rest.
	// That is worse than the error it used to raise, so the authorization
	// endpoint reads the raw query; see resourceParams.
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
	// RFC 8707, repeatable. RFC 6749 section 4.1.3 sends token-endpoint
	// parameters as a form body, and a JSON body is accepted too, so the field
	// carries both tags.
	Resource []string `json:"resource,omitempty" form:"resource,omitempty"`
}

// RevokeRequest is the OAuth2 revocation request.
type RevokeRequest struct {
	Token         string `json:"token" form:"token"`
	TokenTypeHint string `json:"token_type_hint,omitempty" form:"token_type_hint"`
	ClientID      string `json:"client_id,omitempty" form:"client_id"`
	ClientSecret  string `json:"client_secret,omitempty" form:"client_secret"`
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
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	JWKSURI                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	// ResourceIndicatorsSupported advertises RFC 8707.
	//
	// This name is not registered. RFC 8707 registers the `resource`
	// parameter and the `invalid_target` error and defines no discovery
	// metadata at all, and the RFC 8414 IANA registry has no entry for it.
	// It is the convention that came out of the MCP ecosystem and it is what
	// clients look for, so do not read it as standardised.
	ResourceIndicatorsSupported bool `json:"resource_indicators_supported"`

	// DPoPSigningAlgValuesSupported lists the JWS algorithms accepted in DPoP
	// proofs (RFC 9449 section 5.1).
	DPoPSigningAlgValuesSupported []string `json:"dpop_signing_alg_values_supported,omitempty"`
}

// CreateClientRequest is the admin request to create an OAuth2 client.
type CreateClientRequest struct {
	AppID        string   `json:"app_id"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes,omitempty"`
	Resources    []string `json:"resources,omitempty"`
	GrantTypes   []string `json:"grant_types,omitempty"`
	Public       bool     `json:"public,omitempty"`
	DPoPMode     string   `json:"dpop_mode,omitempty"`
}

// CreateClientResponse is returned when an OAuth2 client is created.
type CreateClientResponse struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"` // Only returned once at creation
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	Resources    []string `json:"resources"`
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
//
// The tag is `path`, not `param`: forge binds path parameters from the former
// (internal/router/openapi_request_schema.go). Under `param` the field never
// bound, so request validation rejected every call with "ClientID: field is
// required" and the handler never ran.
type DeleteClientRequest struct {
	ClientID string `path:"clientId"`
}

// DeleteClientResponse is the response after deleting a client.
type DeleteClientResponse struct {
	Status string `json:"status"`
}

// DeviceAuthRequest is the device authorization request (RFC 8628 Section 3.1).
type DeviceAuthRequest struct {
	ClientID string `json:"client_id" form:"client_id"`
	Scope    string `json:"scope,omitempty" form:"scope,omitempty"`
	// RFC 8707, repeatable. RFC 8628 section 3.1 makes this endpoint
	// form-encoded, the same as the token endpoint, and bindFormParam fills a
	// []string from every occurrence. The document describes the field from
	// these tags, so this endpoint needs no separate declaration the way
	// /authorize does.
	Resource []string `json:"resource,omitempty" form:"resource,omitempty"`
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
func newOAuth2Error(status int, errorCode, description string) *OAuth2HTTPError {
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

	// RFC 8707. Read off the raw query: the struct binder still collapses a
	// repeated query parameter to its first value; see resourceParams.
	resources, err := resolveResources(client, resourceParams(ctx.Request()))
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

	// Require authenticated user. No resource_metadata hint here, unlike
	// handleUserInfo: this 401 means the end user is not signed in, not
	// that a client failed to authenticate, and the response goes to a
	// browser in the middle of a redirect, not to a machine parsing a
	// WWW-Authenticate header. A login redirect is the correct response;
	// a discovery hint would be read by nobody. See
	// middleware.ResourceMetadataChallenge's doc comment for the other two
	// endpoints in this group and why each is or isn't covered.
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
		Resources:           resources,
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

// authMethodForPublic maps the admin surface's Public bool onto the RFC 7591
// token_endpoint_auth_method that is the source of truth everywhere else.
func authMethodForPublic(public bool) string {
	if public {
		return "none"
	}
	return "client_secret_basic"
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

// basicAuthScheme is the RFC 7617 scheme name, matched case-insensitively.
const basicAuthScheme = "basic"

// parseBasicClientAuth pulls client credentials out of an HTTP Basic
// Authorization header (RFC 6749 §2.3.1).
//
// present tells the caller whether the header carried Basic credentials at all,
// which is not the same question as whether they parsed. A header in some other
// scheme belongs to somebody else, most likely a bearer token the auth
// middleware ignores on this public endpoint, so it reports absent and the body
// credentials stand. A header that says Basic and then does not decode is a
// failed authentication attempt and has to be answered as one.
func parseBasicClientAuth(r *http.Request) (clientID, clientSecret string, present bool, err error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", "", false, nil
	}
	scheme, encoded, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, basicAuthScheme) {
		return "", "", false, nil
	}

	raw, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if decodeErr != nil {
		return "", "", true, newOAuth2Error(http.StatusUnauthorized, "invalid_client", "malformed Basic authorization header")
	}
	rawID, rawSecret, found := strings.Cut(string(raw), ":")
	if !found {
		return "", "", true, newOAuth2Error(http.StatusUnauthorized, "invalid_client", "malformed Basic authorization header")
	}

	return unescapeCredential(rawID), unescapeCredential(rawSecret), true, nil
}

// unescapeCredential undoes the encoding RFC 6749 §2.3.1 asks a client to apply
// to each half of the pair before base64. Without it a client_id containing a
// colon can never authenticate, because the colon it had to escape stays
// escaped and no longer matches what is registered.
//
// This stops at percent escapes and leaves a plus sign alone, which is a
// deliberate narrowing of what the RFC describes. The RFC names
// application/x-www-form-urlencoded, where a plus decodes to a space. Secrets
// with a plus in them are everywhere (any base64 secret is a candidate) and
// plenty of client libraries base64 the pair without encoding it first, so full
// form decoding would quietly turn those into spaces and fail the compare with
// nothing in the logs to explain it. A space inside a credential is rare enough
// that the trade lands the right way round. Anything this server issues is hex,
// so neither case arises for credentials it minted itself.
//
// A value that is not valid percent encoding is passed through untouched rather
// than rejected, which is what lets the naive clients above work.
func unescapeCredential(v string) string {
	decoded, err := url.PathUnescape(v)
	if err != nil {
		return v
	}
	return decoded
}

// applyBasicClientAuth folds Basic credentials into the token request so every
// grant below sees them the same way it sees client_secret_post. Discovery has
// advertised both methods since the endpoint existed; only the body one was
// ever read, so a client that followed the discovery document could not
// authenticate at all.
func applyBasicClientAuth(r *http.Request, reqClientID, reqClientSecret *string) error {
	clientID, clientSecret, present, err := parseBasicClientAuth(r)
	if err != nil {
		return err
	}
	if !present {
		return nil
	}

	// RFC 6749 §2.3: a client must not use more than one authentication method
	// in a request. Picking a winner is the dangerous reading. Whichever side
	// loses, an attacker who can write only that side gets to decide which
	// credential the server checks, so the request is refused instead.
	if *reqClientSecret != "" {
		return newOAuth2Error(http.StatusBadRequest, "invalid_request",
			"use either the Authorization header or client_secret in the body, not both")
	}
	// A body client_id that agrees with the header is ordinary and plenty of
	// libraries send it. One that disagrees leaves the request unable to say
	// who it claims to be.
	if *reqClientID != "" && *reqClientID != clientID {
		return newOAuth2Error(http.StatusBadRequest, "invalid_request",
			"client_id does not match the Authorization header")
	}

	*reqClientID = clientID
	*reqClientSecret = clientSecret
	return nil
}

// authenticateClient resolves a client and verifies whatever credential it is
// expected to hold. A confidential client must prove its secret. A public
// client has none by definition, so being registered is all there is to check,
// and callers must not treat that as proof of identity.
func (p *Plugin) authenticateClient(ctx context.Context, clientID, clientSecret string) (*OAuth2Client, error) {
	client, err := p.oauth2Store.GetClient(ctx, clientID)
	if err != nil {
		return nil, newOAuth2Error(http.StatusUnauthorized, "invalid_client", "invalid client")
	}
	if !client.Public {
		if clientSecret == "" {
			return nil, newOAuth2Error(http.StatusUnauthorized, "invalid_client", "client_secret required for confidential clients")
		}
		if cmpErr := bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(clientSecret)); cmpErr != nil {
			return nil, newOAuth2Error(http.StatusUnauthorized, "invalid_client", "invalid client_secret")
		}
	}
	return client, nil
}

// callerOwnsSession reports whether the principal behind caller is entitled to
// revoke target: either it is literally the same session, or both belong to the
// same principal.
func callerOwnsSession(caller, target *session.Session) bool {
	if caller == nil || target == nil {
		return false
	}
	if caller.ID == target.ID {
		return true
	}
	subject := caller.Subject()
	// A caller with no resolvable subject matches nothing. Without this guard
	// two zero-valued sessions would compare equal to each other.
	if subject.ID == "" {
		return false
	}
	return subject == target.Subject()
}

// handleToken and the grant handlers it dispatches to (below) return several
// 401s on client-authentication failure (invalid client_secret, unknown
// client_id). Those are not given a resource_metadata hint, deliberately: a
// client calling the token endpoint already holds the authorization server's
// URL, since it just made this request to it. RFC 9728 discovery exists for
// a client that does not yet know where the server is, which is the
// situation on a protected resource such as /v1/oauth/userinfo, not on the
// authorization server's own endpoints. middleware.ResourceMetadataChallenge
// would not add the header here even if it were wanted: these 401s are
// returned from a route handler, and forge converts a route handler's
// returned error into a written response before any enclosing middleware's
// next() call sees it, the same reason handleUserInfo below sets its own
// header instead of relying on that middleware.
func (p *Plugin) handleToken(ctx forge.Context, req *TokenRequest) (*TokenResponse, error) {
	// Runs before the grant switch so all three grants authenticate the same
	// way, and so a malformed header is answered once rather than three times.
	if err := applyBasicClientAuth(ctx.Request(), &req.ClientID, &req.ClientSecret); err != nil {
		return nil, err
	}

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

	// Resolve DPoP binding while the code is still redeemable. This must
	// happen before ConsumeAuthCode: a nonce challenge answers 400
	// use_dpop_nonce and expects the client to retry the same token request
	// carrying the nonce (RFC 9449 §8.2), but the code is single-use. If it
	// were already consumed by the time bindDPoP ran, that mandatory retry
	// would hit "authorization code already used" instead, and a
	// nonce-required app could never issue a bound token from this grant.
	jkt, err := p.bindDPoP(ctx, client, authCode.AppID)
	if err != nil {
		return nil, err
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

	// The code carries what the user authorized; the token request may
	// narrow it further but never widen it.
	resources, err := narrowResources(authCode.Resources, req.Resource)
	if err != nil {
		return nil, err
	}

	// Generate tokens.
	return p.issueTokens(ctx, client, authCode.UserID, authCode.AppID, authCode.Scopes, resources, jkt)
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
	if !clientAllowsGrant(client, "client_credentials") {
		return nil, forge.BadRequest("client is not registered for the client_credentials grant")
	}

	if cmpErr := bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(req.ClientSecret)); cmpErr != nil {
		return nil, forge.Unauthorized("invalid client_secret")
	}

	// There is no prior authorization to narrow against here, so the
	// client's own allowlist is the only bound.
	resources, err := resolveResources(client, req.Resource)
	if err != nil {
		return nil, err
	}

	// Client credentials: no user, just issue an app-level token. No
	// single-use artifact is consumed on this grant, so resolving the
	// binding immediately before issuing is safe (a nonce challenge can be
	// retried with the exact same client_id/client_secret).
	jkt, err := p.bindDPoP(ctx, client, client.AppID)
	if err != nil {
		return nil, err
	}
	return p.issueClientToken(ctx, client, resources, jkt)
}

func (p *Plugin) handleRevoke(ctx forge.Context, req *RevokeRequest) (*apitypes.Empty, error) {
	if req.Token == "" {
		return nil, forge.BadRequest("token required")
	}

	if err := applyBasicClientAuth(ctx.Request(), &req.ClientID, &req.ClientSecret); err != nil {
		return nil, err
	}

	// RFC 7009 §2.1 wants the caller authenticated, and this endpoint used to
	// ask for nothing at all: it revoked whatever token it was handed, for
	// whoever sent one. Two kinds of caller legitimately arrive here and they
	// prove themselves differently, so both are accepted and one of them is
	// required.
	//
	// An OAuth2 client authenticates with its credentials, by header or by
	// body, the same way it does at /token.
	//
	// A signed-in principal authenticates with the bearer token the auth
	// middleware already resolved. That is the path every generated SDK takes:
	// they send a session and no client credentials, so demanding credentials
	// from everyone would break revocation across all of them.
	var authenticatedClient bool
	if req.ClientID != "" || req.ClientSecret != "" {
		if _, err := p.authenticateClient(ctx.Context(), req.ClientID, req.ClientSecret); err != nil {
			return nil, err
		}
		authenticatedClient = true
	}
	caller, hasCaller := middleware.SessionFrom(ctx.Context())
	if !authenticatedClient && !hasCaller {
		return nil, newOAuth2Error(http.StatusUnauthorized, "invalid_client", "client authentication required")
	}

	// RFC 7009 §2.2: an unknown token is a successful revocation. Answering
	// otherwise turns this endpoint into a token validity oracle.
	sess, err := p.store.GetSessionByToken(ctx.Context(), req.Token)
	if err != nil {
		return nil, ctx.JSON(http.StatusOK, map[string]string{"status": "revoked"})
	}

	// A principal may only revoke its own sessions. Refusing out loud would
	// leak that the token exists and belongs to somebody else, so a token the
	// caller does not own draws the same answer an unknown one draws: 200,
	// and nothing revoked.
	//
	// A client that authenticated is not held to this, and that is the half of
	// RFC 7009 §2.1 still missing. Checking that a token was issued to the
	// requesting client needs the session to record which client minted it,
	// and session.Session has no such field. Until it does, an authenticated
	// client can revoke any token it holds a copy of. Narrower than the
	// anonymous hole it replaces, and left tracked rather than fixed here
	// because that column reaches all four store backends.
	if !authenticatedClient && !callerOwnsSession(caller, sess) {
		p.logger.Debug("oauth2: revoke ignored for a token the caller does not own",
			log.String("session_id", sess.ID.String()),
		)
		return nil, ctx.JSON(http.StatusOK, map[string]string{"status": "revoked"})
	}

	if delErr := p.store.DeleteSession(ctx.Context(), sess.ID); delErr != nil {
		p.logger.Debug("oauth2: failed to delete session",
			log.String("error", delErr.Error()),
		)
	}

	return nil, ctx.JSON(http.StatusOK, map[string]string{"status": "revoked"})
}

func (p *Plugin) handleUserInfo(ctx forge.Context, _ *UserInfoRequest) (*UserInfo, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		// AuthMiddleware only soft-resolves the bearer token (it calls next
		// on a missing or invalid one rather than rejecting), so this is the
		// actual point of enforcement for the OIDC protected resource, and
		// the 401 originates here rather than in middleware.
		// middleware.ResourceMetadataChallenge cannot see it: it is
		// registered as router middleware, but forge converts a route
		// handler's returned error into a written response inside its own
		// handler-conversion wrapper before any enclosing middleware's
		// next() call returns, so the error never reaches the chain. Set
		// the hint directly, the same way authenticateRegistration does in
		// register.go, and only if the header is not already set.
		if ctx.Response().Header().Get("WWW-Authenticate") == "" {
			ctx.SetHeader("WWW-Authenticate",
				`Bearer resource_metadata="`+p.issuerURL()+`/.well-known/oauth-protected-resource"`)
		}
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

// handleDiscovery derives the OIDC discovery document from
// buildAuthServerMetadata by copying each field across by hand.
// AuthServerMetadata and DiscoveryResponse are field-identical but distinct
// struct types, so a field added to one is not automatically added to the
// other; TestMetadata_OIDCAndAuthServerAgree is what actually catches that.
func (p *Plugin) handleDiscovery(_ forge.Context, _ *DiscoveryRequest) (*DiscoveryResponse, error) {
	m := p.buildAuthServerMetadata()
	return &DiscoveryResponse{
		Issuer:                            m.Issuer,
		AuthorizationEndpoint:             m.AuthorizationEndpoint,
		TokenEndpoint:                     m.TokenEndpoint,
		UserinfoEndpoint:                  m.UserinfoEndpoint,
		RevocationEndpoint:                m.RevocationEndpoint,
		DeviceAuthorizationEndpoint:       m.DeviceAuthorizationEndpoint,
		RegistrationEndpoint:              m.RegistrationEndpoint,
		JWKSURI:                           m.JWKSURI,
		ResponseTypesSupported:            m.ResponseTypesSupported,
		GrantTypesSupported:               m.GrantTypesSupported,
		SubjectTypesSupported:             m.SubjectTypesSupported,
		IDTokenSigningAlgValuesSupported:  m.IDTokenSigningAlgValuesSupported,
		ScopesSupported:                   m.ScopesSupported,
		TokenEndpointAuthMethodsSupported: m.TokenEndpointAuthMethodsSupported,
		CodeChallengeMethodsSupported:     m.CodeChallengeMethodsSupported,
		ResourceIndicatorsSupported:       m.ResourceIndicatorsSupported,
		DPoPSigningAlgValuesSupported:     m.DPoPSigningAlgValuesSupported,
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

	// manage:oauth2_client says the caller may administer clients. It does not
	// say whose. A client carries redirect URIs and a secret, so minting one
	// under an app_id the caller supplied would hand them the authorization
	// code flow of an app they have no claim to.
	appID, err := plugin.ScopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
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

	// Validate the allowlist at registration so a malformed entry is caught
	// here rather than turning every later authorize request into an opaque
	// invalid_target. This shares the exact rule resolveResources applies at
	// request time, so the two can never disagree about what counts as valid.
	for _, raw := range req.Resources {
		if msg := resourceURISyntaxError(raw); msg != "" {
			return nil, forge.BadRequest(msg)
		}
	}

	// Normalise through ParseMode so an unrecognised value is stored as a
	// known one rather than sitting in the column waiting to be misread.
	// Empty stays empty, which means inherit.
	dpopMode := ""
	if req.DPoPMode != "" {
		dpopMode = string(dpop.ParseMode(req.DPoPMode))
	}

	client := &OAuth2Client{
		ID:                      id.NewOAuth2ClientID(),
		AppID:                   appID,
		Name:                    req.Name,
		ClientID:                clientIDStr,
		ClientSecret:            hashedSecret,
		RedirectURIs:            req.RedirectURIs,
		Scopes:                  scopes,
		Resources:               req.Resources,
		GrantTypes:              grantTypes,
		Public:                  req.Public,
		TokenEndpointAuthMethod: authMethodForPublic(req.Public),
		DPoPMode:                dpopMode,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
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
		Resources:    client.Resources,
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

	appID, err := plugin.ScopedAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
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

	// Load before deleting. The tenancy check needs the row's AppID and
	// DeleteClient takes only an id, so there is no way to make this one call.
	// A mismatch answers 404, not 403, so the route cannot be used to probe
	// for the existence of another app's clients.
	client, err := p.oauth2Store.GetClientByID(ctx.Context(), clientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			return nil, forge.NotFound("oauth2 client not found")
		}
		return nil, forge.InternalError(fmt.Errorf("oauth2: load client: %w", err))
	}
	if err := plugin.AssertAppScope(ctx, client.AppID); err != nil {
		return nil, err
	}

	if err := p.oauth2Store.DeleteClient(ctx.Context(), clientID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: delete client: %w", err))
	}

	return &DeleteClientResponse{Status: "deleted"}, nil
}

// ──────────────────────────────────────────────────
// Token Issuance
// ──────────────────────────────────────────────────

// bindDPoP resolves the effective DPoP mode and, if a proof was presented,
// validates it and returns the thumbprint to stamp on the session.
//
// Returns an empty string when the token should be issued unbound. At this
// point there is no token yet, so Expectation carries no AccessToken and no
// ExpectedJKT: the key is learned here rather than checked.
func (p *Plugin) bindDPoP(ctx forge.Context, client *OAuth2Client, appID id.AppID) (string, error) {
	if p.engine == nil {
		return "", nil
	}

	mode := p.engine.DPoPModeForApp(ctx.Context(), appID)
	if client != nil {
		mode = dpop.MaxMode(mode, dpop.ParseMode(client.DPoPMode))
	}
	if mode == dpop.ModeOff {
		return "", nil
	}

	raw := ctx.Request().Header.Get("DPoP")
	if raw == "" {
		if mode == dpop.ModeRequired {
			return "", newOAuth2Error(http.StatusBadRequest, "invalid_dpop_proof",
				"this client must present a DPoP proof")
		}
		return "", nil // optional, and the client did not prove
	}

	proof, err := dpop.Parse(raw)
	if err != nil {
		return "", newOAuth2Error(http.StatusBadRequest, "invalid_dpop_proof", err.Error())
	}

	nonceRequired := p.engine.DPoPNonceRequiredForApp(ctx.Context(), appID)

	err = p.engine.DPoPValidator().Validate(ctx.Context(), proof, dpop.Expectation{
		Method:        ctx.Request().Method,
		URL:           middleware.RequestURL(ctx.Request()),
		NonceRequired: nonceRequired,
	})
	if err != nil {
		if errors.Is(err, dpop.ErrNonceRequired) || errors.Is(err, dpop.ErrNonceMismatch) {
			// RFC 9449 section 8: the token endpoint answers with 400 and a
			// fresh nonce, not a 401 challenge.
			if signer := p.engine.DPoPNonceSigner(); signer != nil {
				ctx.Response().Header().Set("DPoP-Nonce", signer.Issue(proof.JKT))
			}
			return "", newOAuth2Error(http.StatusBadRequest, "use_dpop_nonce",
				"a server-provided nonce is required")
		}
		return "", newOAuth2Error(http.StatusBadRequest, "invalid_dpop_proof", err.Error())
	}

	return proof.JKT, nil
}

// issueTokens mints a session and its token response. jkt is the DPoP
// thumbprint to bind to, resolved by the caller via bindDPoP before any
// single-use grant artifact (an authorization code, a device code) is
// consumed. Discovering it in here instead would run bindDPoP after the
// code is already burned, so a use_dpop_nonce challenge could never be
// retried: the retry the RFC requires would arrive against a code the
// store already rejects as used. See handleAuthorizationCodeGrant and
// handleDeviceCodeGrant for where jkt is actually resolved.
func (p *Plugin) issueTokens(ctx forge.Context, _ *OAuth2Client, userID id.UserID, appID id.AppID, scopes, resources []string, jkt string) (*TokenResponse, error) {
	// Resolve session config for the app.
	sessCfg := account.SessionConfig{
		TokenTTL:        p.config.AccessTokenTTL,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}
	if p.engine != nil {
		sessCfg = p.engine.SessionConfigForApp(ctx.Context(), appID)
	}

	sess, err := account.NewSession(appID, userID, sessCfg)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: create session: %w", err))
	}
	sess.DPoPJKT = jkt

	// Stamp the granted scopes. Without this they exist only in the JWT claims
	// and the response body, so an opaque token loses them and token exchange
	// has no subject-side ceiling to narrow against.
	sess.Scopes = scopes

	// Bind session to the app's default environment so the FK constraint
	// on authsome_sessions.env_id is satisfied.
	if env, envErr := p.store.GetDefaultEnvironment(ctx.Context(), appID); envErr == nil && env != nil {
		sess.EnvID = env.ID
	}

	// The opaque half of the audience. An opaque access token carries no
	// claims, so this is the only place introspection and the middleware
	// audience check can read it from.
	sess.Audience = resources

	// If token format is JWT, generate a JWT access token.
	if p.engine != nil {
		tokFmt := p.engine.TokenFormatForApp(appID.String())
		if tokFmt.Name() == "jwt" {
			jwtToken, err := tokFmt.GenerateAccessToken(tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     appID.String(),
				SessionID: sess.ID.String(),
				Scopes:    scopes,
				Audience:  resources,
				DPoPJKT:   jkt,
				IssuedAt:  sess.CreatedAt,
				ExpiresAt: sess.ExpiresAt,
			})
			if err != nil {
				return nil, forge.InternalError(fmt.Errorf("oauth2: generate JWT: %w", err))
			}
			sess.Token = jwtToken
		}
	}

	if err := p.store.CreateSession(ctx.Context(), sess); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: save session: %w", err))
	}

	tokenType := "Bearer"
	if jkt != "" {
		tokenType = "DPoP"
	}

	return &TokenResponse{
		AccessToken:  sess.Token,
		TokenType:    tokenType,
		ExpiresIn:    int(time.Until(sess.ExpiresAt).Seconds()),
		RefreshToken: sess.RefreshToken,
		Scope:        strings.Join(scopes, " "),
	}, nil
}

// issueClientToken mints a client-credentials session. jkt is the DPoP
// thumbprint to bind to, resolved by the caller. Client credentials has no
// single-use artifact to burn, so the caller resolves it immediately before
// calling rather than needing the reordering issueTokens' callers require;
// the shape is kept consistent with issueTokens regardless.
func (p *Plugin) issueClientToken(ctx forge.Context, client *OAuth2Client, resources []string, jkt string) (*TokenResponse, error) {
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
	sess.DPoPJKT = jkt

	// A client-credentials token carries the client's registered scopes.
	sess.Scopes = client.Scopes

	// Bind session to the app's default environment so the FK constraint
	// on authsome_sessions.env_id is satisfied.
	if env, envErr := p.store.GetDefaultEnvironment(ctx.Context(), client.AppID); envErr == nil && env != nil {
		sess.EnvID = env.ID
	}

	sess.Audience = resources

	if err := p.store.CreateSession(ctx.Context(), sess); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: save client session: %w", err))
	}

	tokenType := "Bearer"
	if jkt != "" {
		tokenType = "DPoP"
	}

	return &TokenResponse{
		AccessToken: sess.Token,
		TokenType:   tokenType,
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
		verificationURI = p.issuerURL() + "/v1/oauth/device"
	}

	scopes := strings.Fields(req.Scope)

	// There is no prior authorization to narrow against at this endpoint
	// either, so the client's own allowlist bounds what may be requested,
	// same as client credentials.
	resources, err := resolveResources(client, req.Resource)
	if err != nil {
		return nil, err
	}

	dc := &DeviceCode{
		ID:              id.NewDeviceCodeID(),
		DeviceCode:      deviceCodeStr,
		UserCode:        userCodeStr,
		ClientID:        req.ClientID,
		AppID:           client.AppID,
		Scopes:          scopes,
		Resources:       resources,
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

	// Authenticate the client before touching the device code at all.
	//
	// RFC 8628 §3.4 routes the token request through RFC 6749 §3.2.1, so a
	// confidential client must present its secret here exactly as it does for
	// authorization_code and client_credentials. Skip that and the device code
	// becomes the only secret in the exchange, which is not a job it can do: it
	// rides in every poll, so it settles into access logs and proxies, and the
	// client_id sitting beside it is public by definition.
	//
	// This runs ahead of the expiry and slow_down checks rather than after
	// them, for two reasons. An unauthenticated caller should not get an oracle
	// that separates a live code (authorization_pending, slow_down) from an
	// expired or invented one (expired_token, invalid_grant). And the slow_down
	// branch writes: it pushes the code's polling interval up by five seconds
	// and stamps LastPolledAt, so leaving it reachable without credentials
	// hands anyone who picked up a leaked device code a way to ratchet the real
	// device's interval until the code times out. The price is a bcrypt compare
	// on every poll, including the too-fast ones, and that is the same price
	// the sibling grants already pay.
	client, err := p.authenticateClient(ctx.Context(), req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
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
		// Success! Issue tokens. The client was loaded and authenticated at the
		// top of this handler, and dc.ClientID was checked against it.

		// Resolve DPoP binding while the device code is still redeemable, for
		// the same reason as the authorization_code grant: a use_dpop_nonce
		// challenge must be retryable, and marking the code consumed first
		// would make the retry fail with "invalid_grant" instead.
		jkt, err := p.bindDPoP(ctx, client, dc.AppID)
		if err != nil {
			return nil, err
		}

		// Mark as consumed before issuing tokens (one-time use).
		// If this update fails, do NOT issue tokens to prevent double-use.
		dc.Status = DeviceCodeStatusConsumed
		if err := p.oauth2Store.UpdateDeviceCode(ctx.Context(), dc); err != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: consume device code: %w", err))
		}

		resources, resErr := narrowResources(dc.Resources, req.Resource)
		if resErr != nil {
			return nil, resErr
		}

		return p.issueTokens(ctx, client, dc.UserID, dc.AppID, dc.Scopes, resources, jkt)

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
