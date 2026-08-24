package oauth2provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

// newWellKnownRouter registers RegisterRootRoutes on its own router
// instance. It must be its own instance rather than reusing newTestRouter's
// router: RegisterRoutes already mirrors the OIDC discovery document onto
// the grouped router in standalone mode, so calling RegisterRootRoutes on
// that same router panics on a duplicate route. Using a dedicated router
// here is also what proves RegisterRootRoutes itself does not error.
func newWellKnownRouter(t *testing.T, p *oauth2provider.Plugin) http.Handler {
	t.Helper()
	r := forge.NewRouter()
	require.NoError(t, p.RegisterRootRoutes(r))
	return r
}

func getJSON(t *testing.T, router http.Handler, path string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		return rec.Code, nil
	}
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return rec.Code, got
}

func TestMetadata_AuthServerDocument(t *testing.T) {
	p, _, _, _ := newRegistrationFixture(t, true)
	router := newWellKnownRouter(t, p)

	code, got := getJSON(t, router, "/.well-known/oauth-authorization-server")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "https://auth.example.com", got["issuer"])
	assert.Equal(t, "https://auth.example.com/v1/oauth/token", got["token_endpoint"])
	assert.Equal(t, "https://auth.example.com/v1/oauth/register", got["registration_endpoint"])
}

// Advertising an endpoint that 404s sends a client down a dead path and
// produces a worse error than not advertising it.
func TestMetadata_NoRegistrationEndpointWhenDisabled(t *testing.T) {
	p, _, _, _ := newRegistrationFixture(t, false)
	router := newWellKnownRouter(t, p)

	code, got := getJSON(t, router, "/.well-known/oauth-authorization-server")
	require.Equal(t, http.StatusOK, code)
	_, present := got["registration_endpoint"]
	assert.False(t, present)
}

// One builder feeds both documents, so they cannot drift apart.
func TestMetadata_OIDCAndAuthServerAgree(t *testing.T) {
	p, _, _, _ := newRegistrationFixture(t, true)
	router := newWellKnownRouter(t, p)

	_, as := getJSON(t, router, "/.well-known/oauth-authorization-server")
	_, oidc := getJSON(t, router, "/.well-known/openid-configuration")

	for _, k := range []string{
		"issuer", "authorization_endpoint", "token_endpoint", "jwks_uri",
		"registration_endpoint", "grant_types_supported",
		"code_challenge_methods_supported",
	} {
		assert.Equal(t, as[k], oidc[k], "field %q differs between the two documents", k)
	}
}

func TestMetadata_ProtectedResourceDocument(t *testing.T) {
	p, _, _, _ := newRegistrationFixture(t, true)
	router := newWellKnownRouter(t, p)

	code, got := getJSON(t, router, "/.well-known/oauth-protected-resource")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "https://auth.example.com", got["resource"])
	assert.Equal(t, []any{"https://auth.example.com"}, got["authorization_servers"])
	assert.Equal(t, []any{"header"}, got["bearer_methods_supported"])
}

func TestMetadata_ConfiguredExtraResource(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{
		Issuer:              "https://auth.example.com",
		DynamicRegistration: true,
		ProtectedResources: map[string]oauth2provider.ProtectedResource{
			"mcp": {
				Resource:        "https://mcp.example.com",
				ScopesSupported: []string{"openid", "profile"},
			},
		},
	})
	p.SetOAuth2Store(oauth2provider.NewMemoryStore())
	router := newWellKnownRouter(t, p)

	code, got := getJSON(t, router, "/.well-known/oauth-protected-resource/mcp")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "https://mcp.example.com", got["resource"])
	assert.Equal(t, []any{"https://auth.example.com"}, got["authorization_servers"])
	assert.Equal(t, []any{"openid", "profile"}, got["scopes_supported"])
}

func TestMetadata_UnknownExtraResourceIs404(t *testing.T) {
	p, _, _, _ := newRegistrationFixture(t, true)
	router := newWellKnownRouter(t, p)

	code, _ := getJSON(t, router, "/.well-known/oauth-protected-resource/nope")
	assert.Equal(t, http.StatusNotFound, code)
}
