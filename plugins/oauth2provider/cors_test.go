package oauth2provider_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store/memory"
)

const spaOrigin = "https://spa.example.com"

func corsRequest(t *testing.T, h http.Handler, method, path, origin string) *httptest.ResponseRecorder {
	t.Helper()
	var body *strings.Reader
	if method == http.MethodPost {
		body = strings.NewReader("grant_type=authorization_code")
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequestWithContext(t.Context(), method, path, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if method == http.MethodOptions {
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func corsRouter(t *testing.T, cfg oauth2provider.Config) http.Handler {
	t.Helper()
	if cfg.Issuer == "" {
		cfg.Issuer = "https://auth.example.com"
	}
	_, _, mux := newFixtureWithConfig(t, cfg)
	return mux
}

func TestCORS_BrowserEndpoints(t *testing.T) {
	h := corsRouter(t, oauth2provider.Config{})

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/v1/oauth/jwks"},
		{http.MethodGet, "/.well-known/openid-configuration"},
		{http.MethodPost, "/v1/oauth/token"},
		{http.MethodPost, "/v1/oauth/revoke"},
		{http.MethodGet, "/v1/oauth/userinfo"},
		{http.MethodPost, "/v1/oauth/userinfo"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := corsRequest(t, h, tc.method, tc.path, spaOrigin)
			assert.Equal(t, spaOrigin, rec.Header().Get("Access-Control-Allow-Origin"),
				"status %d: the SPA can read the answer, errors included", rec.Code)
			assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
			assert.Contains(t, rec.Header().Values("Vary"), "Origin")
			assert.Contains(t, rec.Header().Get("Access-Control-Expose-Headers"), "WWW-Authenticate")
		})
	}
}

func TestCORS_Preflight(t *testing.T) {
	h := corsRouter(t, oauth2provider.Config{})
	for _, path := range []string{
		"/v1/oauth/token", "/v1/oauth/revoke", "/v1/oauth/userinfo", "/v1/oauth/jwks", "/v1/oauth/register",
		"/.well-known/openid-configuration", "/apps/acme/.well-known/openid-configuration", "/apps/acme/v1/oauth/jwks",
	} {
		t.Run(path, func(t *testing.T) {
			rec := corsRequest(t, h, http.MethodOptions, path, spaOrigin)
			require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
			assert.Equal(t, spaOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
			assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
			assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
			assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "DPoP")
			assert.Equal(t, "600", rec.Header().Get("Access-Control-Max-Age"))
		})
	}
}

func TestCORS_RootDiscoveryDocuments(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(oauth2provider.NewMemoryStore())
	p.SetStore(memory.New())
	h := newWellKnownRouter(t, p)
	for _, path := range []string{
		"/.well-known/openid-configuration", "/.well-known/oauth-authorization-server",
		"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp",
		"/apps/acme/.well-known/openid-configuration", "/apps/acme/.well-known/oauth-authorization-server",
	} {
		t.Run(path, func(t *testing.T) {
			assert.Equal(t, http.StatusNoContent, corsRequest(t, h, http.MethodOptions, path, spaOrigin).Code)
			rec := corsRequest(t, h, http.MethodGet, path, spaOrigin)
			assert.Equal(t, spaOrigin, rec.Header().Get("Access-Control-Allow-Origin"), "status %d", rec.Code)
		})
	}
}

func TestCORS_NavigationsGetNone(t *testing.T) {
	h := corsRouter(t, oauth2provider.Config{})
	q := baseAuthorizeQuery(confidentialID)
	rec := corsRequest(t, h, http.MethodGet, "/v1/oauth/authorize?"+q.Encode(), spaOrigin)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "authorize uses the session cookie and is a navigation")
}

func TestCORS_NoOriginNoHeaders(t *testing.T) {
	h := corsRouter(t, oauth2provider.Config{})
	rec := corsRequest(t, h, http.MethodGet, "/v1/oauth/jwks", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_OriginList(t *testing.T) {
	h := corsRouter(t, oauth2provider.Config{CORSOrigins: []string{"https://SPA.example.com:443/", "http://localhost:5173"}})

	rec := corsRequest(t, h, http.MethodGet, "/v1/oauth/jwks", spaOrigin)
	assert.Equal(t, spaOrigin, rec.Header().Get("Access-Control-Allow-Origin"), "case and default port don't matter")
	rec = corsRequest(t, h, http.MethodGet, "/v1/oauth/jwks", "http://localhost:5173")
	assert.Equal(t, "http://localhost:5173", rec.Header().Get("Access-Control-Allow-Origin"))

	rec = corsRequest(t, h, http.MethodGet, "/v1/oauth/jwks", "://not an origin")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "an origin that won't parse matches nothing")

	rec = corsRequest(t, h, http.MethodGet, "/v1/oauth/jwks", "https://evil.example.com")
	assert.Equal(t, http.StatusOK, rec.Code, "the request itself still works")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "but the browser won't hand it over")

	rec = corsRequest(t, h, http.MethodOptions, "/v1/oauth/token", "https://evil.example.com")
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Methods"))

	star := corsRouter(t, oauth2provider.Config{CORSOrigins: []string{"*"}})
	rec = corsRequest(t, star, http.MethodGet, "/v1/oauth/jwks", "https://anything.example.com")
	assert.Equal(t, "https://anything.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_Disabled(t *testing.T) {
	h := corsRouter(t, oauth2provider.Config{DisableCORS: true})
	rec := corsRequest(t, h, http.MethodGet, "/v1/oauth/jwks", spaOrigin)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEqual(t, http.StatusNoContent, corsRequest(t, h, http.MethodOptions, "/v1/oauth/token", spaOrigin).Code)
}

// A host-wide CORS middleware that allows credentials for its own
// frontends must not leak that onto these endpoints: echoing an origin and
// allowing credentials is what lets another site read a cookie-backed
// answer.
func TestCORS_StripsHostCredentials(t *testing.T) {
	inner := corsRouter(t, oauth2provider.Config{})
	host := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Origin", "https://host-frontend.example.com")
		inner.ServeHTTP(w, r)
	})
	rec := corsRequest(t, host, http.MethodGet, "/v1/oauth/jwks", spaOrigin)
	assert.Equal(t, spaOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}
