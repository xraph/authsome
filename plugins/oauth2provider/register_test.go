package oauth2provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// newTestRouter builds a forge router with both RegisterRoutes and
// RegisterRootRoutes exercised, mirroring how the real engine wires the
// plugin. In standalone mode RegisterRoutes already mirrors the discovery
// document onto the grouped router (see plugin.go), so calling
// RegisterRootRoutes on that same router panics on a duplicate route.
// RegisterRootRoutes is therefore mounted on a second router instance; this
// test router returns the one that carries /v1/oauth/register, which is
// what every case here needs. Task 7's metadata tests reach the
// /.well-known/... paths through RegisterRootRoutes directly instead.
func newTestRouter(t *testing.T, p *oauth2provider.Plugin) http.Handler {
	t.Helper()
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	rootRouter := forge.NewRouter()
	require.NoError(t, p.RegisterRootRoutes(rootRouter))
	return router
}

// newRegistrationFixture builds a plugin wired to an in-memory store with
// dynamic registration either enabled or disabled, using a freshly minted
// app as the registration fallback. The plugin is part of the returned
// tuple (even though no case here needs it) because Task 7's metadata
// tests reuse this exact helper and do need it.
//
//nolint:unparam // shape shared with Task 7, see above
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
	req := httptest.NewRequest(http.MethodPost, "/v1/oauth/register", strings.NewReader(body))
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
	// Stored hashed, never in the clear.
	assert.NotEqual(t, secret, stored.ClientSecret)
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
	stored, err := st.GetClient(t.Context(), got["client_id"].(string))
	require.NoError(t, err)
	assert.Equal(t, "mcp-cli", stored.Metadata["software_id"])
	assert.Equal(t, "https://example.com", stored.Metadata["client_uri"])
}
