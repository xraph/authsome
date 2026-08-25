package oauth2provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// newTestRouter builds a forge router with RegisterRoutes exercised,
// mirroring how the real engine wires the plugin's grouped endpoints. It
// returns the router that carries /v1/oauth/register, which is what every
// case here needs.
//
// It does not exercise RegisterRootRoutes: in standalone mode RegisterRoutes
// already mirrors the discovery document onto this same router (see
// plugin.go), so calling RegisterRootRoutes here too would panic on a
// duplicate route. RegisterRootRoutes is exercised on its own router
// instance instead, by newWellKnownRouter in metadata_test.go.
func newTestRouter(t *testing.T, p *oauth2provider.Plugin) http.Handler {
	t.Helper()
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	return router
}

// newRegistrationFixture builds a plugin wired to an in-memory store with
// dynamic registration either enabled or disabled, using a freshly minted
// app as the registration fallback. The plugin is part of the returned
// tuple so metadata_test.go can build its own root router from it via
// newWellKnownRouter, alongside the store, grouped router, and app ID the
// registration tests here use directly.
func newRegistrationFixture(t *testing.T, enabled bool) (*oauth2provider.Plugin, oauth2provider.Store, http.Handler, id.AppID) {
	t.Helper()
	appID := id.NewAppID()
	p := oauth2provider.New(oauth2provider.Config{
		Issuer:              "https://auth.example.com",
		DynamicRegistration: enabled,
		RegistrationAppID:   appID.String(),
	})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	router := newTestRouter(t, p)
	return p, st, router, appID
}

func postRegister(t *testing.T, router http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRegister_HappyPathPublicClient(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)

	rec := postRegister(t, router, `{
		"client_name": "MCP CLI",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"token_endpoint_auth_method": "none",
		"grant_types": ["authorization_code", "refresh_token"],
		"scope": "openid profile email"
	}`)

	require.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.NotEmpty(t, got["client_id"])
	assert.NotEmpty(t, got["registration_access_token"])
	assert.Contains(t, got["registration_client_uri"], got["client_id"])
	// A public client gets no secret.
	assert.Empty(t, got["client_secret"])
	assert.Equal(t, "openid profile email", got["scope"])
}

func TestRegister_ConfidentialClientGetsSecretOnce(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)

	rec := postRegister(t, router, `{
		"client_name": "Server",
		"redirect_uris": ["https://app.example.com/cb"],
		"token_endpoint_auth_method": "client_secret_post"
	}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	secret, _ := got["client_secret"].(string)
	require.NotEmpty(t, secret)

	stored, err := st.GetClient(t.Context(), got["client_id"].(string))
	require.NoError(t, err)
	// Stored as a bcrypt hash of exactly this secret, never in the clear.
	// A plain NotEqual would also pass for a reversed string or any other
	// non-matching garbage, so verify the hash actually opens with it.
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.ClientSecret), []byte(secret)))
	assert.NotEmpty(t, stored.RegistrationTokenHash)
	assert.True(t, stored.DynamicallyRegistered)
}

func TestRegister_DisabledReturns404(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, false)
	rec := postRegister(t, router, `{"redirect_uris":["https://app.example.com/cb"]}`)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRegister_RejectsClientCredentials(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"grant_types": ["client_credentials"]
	}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])
	assert.Contains(t, got["error_description"], "client_credentials")
}

func TestRegister_DropsScopesOutsideAllowlist(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"scope": "openid admin:all email"
	}`)

	require.Equal(t, http.StatusCreated, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "openid email", got["scope"])
}

func TestRegister_RejectsBadRedirectURI(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{"redirect_uris":["http://evil.example.com/cb"]}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_redirect_uri", got["error"])
}

func TestRegister_RequiresRedirectURIs(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{"client_name":"No URIs"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// checkRegistrationSizeCaps used to leave redirect URI length, scope length
// and grant_types count completely uncapped, so a body within forge's body
// limit could still write an unboundedly large field into the stored client.
// These four cases each exercise one of the caps added to close that gap.

func TestRegister_RejectsOversizedRedirectURI(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	longURI := "https://app.example.com/" + strings.Repeat("a", 2048)
	rec := postRegister(t, router, `{"redirect_uris":["`+longURI+`"]}`)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])
	assert.Contains(t, got["error_description"], "redirect_uris")
}

func TestRegister_RejectsOversizedScope(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	longScope := strings.Repeat("openid ", 200) // > 1024 bytes
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"scope": "`+longScope+`"
	}`)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])
	assert.Contains(t, got["error_description"], "scope")
}

func TestRegister_RejectsTooManyGrantTypes(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	grants := make([]string, 0, 11)
	for i := 0; i < 11; i++ {
		grants = append(grants, `"authorization_code"`)
	}
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"grant_types": [`+strings.Join(grants, ",")+`]
	}`)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])
	assert.Contains(t, got["error_description"], "grant_types")
}

// A body with a scope repeated many times used to write one slice entry per
// repetition into the stored client's scopes column.
func TestRegister_DedupsDuplicatedScope(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"scope": "openid openid openid email"
	}`)

	require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "openid email", got["scope"])

	stored, err := st.GetClient(t.Context(), got["client_id"].(string))
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "email"}, stored.Scopes)
}

// With no publishable key on the request and no configured fallback, there
// is no app to attach the client to and the request must be refused rather
// than pooled into somebody's tenant.
func TestRegister_NoAppResolvesTo403(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{
		Issuer:              "https://auth.example.com",
		DynamicRegistration: true,
		// RegistrationAppID deliberately unset.
	})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	router := newTestRouter(t, p)

	rec := postRegister(t, router, `{"redirect_uris":["https://app.example.com/cb"]}`)
	require.Equal(t, http.StatusForbidden, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "access_denied", got["error"])
}

func TestRegister_RoundTripsInformationalMetadata(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"client_uri": "https://example.com",
		"software_id": "mcp-cli",
		"software_version": "2.1.0",
		"contacts": ["ops@example.com"]
	}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	// Response-side: this is what clientInfoResponse's str() closure and
	// the contacts type switch actually produce over the wire, decoded
	// back out of real JSON (encoding/json turns a JSON array into
	// []interface{}, not []string).
	assert.Equal(t, "mcp-cli", got["software_id"])
	assert.Equal(t, "https://example.com", got["client_uri"])
	assert.Equal(t, []any{"ops@example.com"}, got["contacts"])

	stored, err := st.GetClient(t.Context(), got["client_id"].(string))
	require.NoError(t, err)
	assert.Equal(t, "mcp-cli", stored.Metadata["software_id"])
	assert.Equal(t, "https://example.com", stored.Metadata["client_uri"])
}

// A Plugin built the way newRegistrationFixture builds it never calls
// OnInit, so it has no engine and no engine-provided rate limiter.
// registrationLimiter must still fall back to a process-local one and the
// route must still attach the middleware — the earlier code silently
// skipped attaching it whenever p.engine was nil, which described exactly
// this fixture and, more importantly, every stock deployment that never
// turns on extension.Config.RateLimit.
//
// The limit is set low rather than left at the real 10/hour default so
// this test doesn't need ten rounds of DefaultCost bcrypt hashing; the
// point is proving the middleware is wired up at all, not exercising a
// particular configured cap.
func TestRegister_RateLimitsWithoutEngineLimiter(t *testing.T) {
	appID := id.NewAppID()
	p := oauth2provider.New(oauth2provider.Config{
		Issuer:                "https://auth.example.com",
		DynamicRegistration:   true,
		RegistrationAppID:     appID.String(),
		RegistrationRateLimit: oauth2provider.RateLimit{Limit: 3, Window: time.Minute},
	})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	router := newTestRouter(t, p)

	// A public client skips the client-secret bcrypt hash, leaving one
	// hash (the registration access token) per request.
	const body = `{"redirect_uris":["https://app.example.com/cb"],"token_endpoint_auth_method":"none"}`
	for i := 1; i <= 3; i++ {
		rec := postRegister(t, router, body)
		require.Equal(t, http.StatusCreated, rec.Code, "request %d is within the limit", i)
	}

	rec := postRegister(t, router, body)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "the request past the limit must be rejected")
}

// ── tenancy: publishable key resolution ────────────────

// stubAppResolver satisfies middleware.AppResolver with a fixed key->app
// map, standing in for Engine.ResolveAppByPublicKey so these tests can wire
// a real middleware.PublishableKeyMiddleware onto the test router instead
// of stamping middleware.WithAppID onto the context by hand.
type stubAppResolver struct {
	byKey map[string]*app.App
}

func (r stubAppResolver) ResolveAppByPublicKey(_ context.Context, key string) (*app.App, error) {
	a, ok := r.byKey[key]
	if !ok {
		return nil, errors.New("stubAppResolver: unknown publishable key")
	}
	return a, nil
}

// newTenantRouter builds a router carrying middleware.PublishableKeyMiddleware
// ahead of the plugin's routes, mirroring how api.RegisterRoutes wires it in
// production (see api/api.go), so a request with a resolvable
// X-Publishable-Key header reaches the handler with an app already on the
// context.
func newTenantRouter(t *testing.T, p *oauth2provider.Plugin, resolver middleware.AppResolver) http.Handler {
	t.Helper()
	router := forge.NewRouter()
	router.Use(middleware.PublishableKeyMiddleware(resolver, nil))
	require.NoError(t, p.RegisterRoutes(router))
	return router
}

func postRegisterWithKey(t *testing.T, router http.Handler, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, key)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// resolveRegistrationAppID checks middleware.AppIDFrom before falling back
// to Config.RegistrationAppID. This is the branch the rest of this file
// never exercises, because every other fixture sets RegistrationAppID and
// sends no key: a publishable key on the request must select its own app
// even when the plugin has no configured fallback at all.
func TestRegister_PublishableKeySelectsApp(t *testing.T) {
	appA := &app.App{ID: id.NewAppID(), Name: "App A"}
	// Only the ID is read; govet flags an unused Name write.
	appFallback := &app.App{ID: id.NewAppID()}
	resolver := stubAppResolver{byKey: map[string]*app.App{"pk_test_a": appA}}

	// RegistrationAppID is set to a THIRD app on purpose. With it unset, the
	// fallback branch is empty and skipped, so the key would still win even
	// if resolveRegistrationAppID consulted config first: the test would
	// pass against an inverted precedence and pin nothing. Giving the
	// fallback a real value is what makes the two branches compete.
	p := oauth2provider.New(oauth2provider.Config{
		Issuer:              "https://auth.example.com",
		DynamicRegistration: true,
		RegistrationAppID:   appFallback.ID.String(),
	})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	router := newTenantRouter(t, p, resolver)

	rec := postRegisterWithKey(t, router, "pk_test_a",
		`{"redirect_uris":["https://app.example.com/cb"]}`)
	require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	stored, err := st.GetClient(t.Context(), got["client_id"].(string))
	require.NoError(t, err)
	assert.Equal(t, appA.ID, stored.AppID)
	assert.NotEqual(t, appFallback.ID, stored.AppID,
		"the publishable key must outrank Config.RegistrationAppID, not merely be consulted when it is empty")
}

// The cross-tenant boundary: a client registered under app A's publishable
// key must never be attached to, or visible under, app B. Before this test
// the four tenancy cases the design called for (context AppID, config
// fallback, publishable key selection, and this boundary) had no coverage
// for the last two at all.
func TestRegister_PublishableKeyDoesNotCrossTenants(t *testing.T) {
	appA := &app.App{ID: id.NewAppID(), Name: "App A"}
	appB := &app.App{ID: id.NewAppID(), Name: "App B"}
	// Only the ID is read; govet flags an unused Name write.
	appFallback := &app.App{ID: id.NewAppID()}
	resolver := stubAppResolver{byKey: map[string]*app.App{
		"pk_test_a": appA,
		"pk_test_b": appB,
	}}

	// A third app on RegistrationAppID, for the reason spelled out in
	// TestRegister_PublishableKeySelectsApp: without it the fallback branch
	// is empty and an inverted precedence would still land on app A.
	p := oauth2provider.New(oauth2provider.Config{
		Issuer:              "https://auth.example.com",
		DynamicRegistration: true,
		RegistrationAppID:   appFallback.ID.String(),
	})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	router := newTenantRouter(t, p, resolver)

	rec := postRegisterWithKey(t, router, "pk_test_a",
		`{"redirect_uris":["https://app.example.com/cb"]}`)
	require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	stored, err := st.GetClient(t.Context(), got["client_id"].(string))
	require.NoError(t, err)
	assert.Equal(t, appA.ID, stored.AppID)
	assert.NotEqual(t, appB.ID, stored.AppID)
	assert.NotEqual(t, appFallback.ID, stored.AppID,
		"the publishable key must outrank Config.RegistrationAppID")

	bClients, err := st.ListClients(t.Context(), appB.ID)
	require.NoError(t, err)
	assert.Empty(t, bClients, "a client registered under app A's key must not be visible under app B")
}
