package oauth2provider_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// newUserInfoRouter builds a plugin router the way it is actually wired in
// production (extension/extension.go:530-536): ResourceMetadataChallenge is
// registered with router.Use before the plugin's own routes are registered
// on that same router. /v1/oauth/userinfo is a route handler (handleUserInfo
// in plugin.go), not middleware, so it is exactly the shape
// middleware.ResourceMetadataChallenge cannot see a 401 through: forge
// converts a route handler's returned error into a written response inside
// its own handler-conversion wrapper before the error ever reaches an
// enclosing middleware's next() call. engineMetaURL is deliberately
// different from the URL the plugin itself derives from its issuer, so a
// test can tell which of the two set the header actually observed.
func newUserInfoRouter(t *testing.T, engineMetaURL string) http.Handler {
	t.Helper()
	p := oauth2provider.New(oauth2provider.Config{
		Issuer: "https://auth.example.com",
	})
	p.SetOAuth2Store(oauth2provider.NewMemoryStore())

	r := forge.NewRouter()
	r.Use(middleware.ResourceMetadataChallenge(engineMetaURL))
	require.NoError(t, p.RegisterRoutes(r))
	return r
}

func TestUserInfo_MissingAuthCarriesResourceMetadataHint(t *testing.T) {
	router := newUserInfoRouter(t, "https://engine-configured.example.com/should-not-appear")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/oauth/userinfo", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"),
		`resource_metadata="https://auth.example.com/.well-known/oauth-protected-resource"`)
}

// TestUserInfo_ResourceMetadataChallengeDoesNotOverwriteHandlerHeader proves
// the do-not-overwrite invariant in the real router-plus-route-handler
// topology, not a middleware stand-in. ResourceMetadataChallenge is wired
// with its own, different URL; the response must carry only
// handleUserInfo's own header, never the middleware's value, and never both
// (no duplication, no second WWW-Authenticate line). In this topology that
// is not really a "the middleware chose not to overwrite" story: the
// middleware's next() call reports no error at all for this route (see
// newUserInfoRouter's comment), so it never even reaches its own
// overwrite-guard logic. What this test actually pins down is the observable
// behavior a reviewer or a client would see: the handler's header wins,
// unconditionally, in the real wiring.
func TestUserInfo_ResourceMetadataChallengeDoesNotOverwriteHandlerHeader(t *testing.T) {
	const engineMetaURL = "https://engine-configured.example.com/should-not-appear"
	router := newUserInfoRouter(t, engineMetaURL)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/oauth/userinfo", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	got := rec.Header().Values("WWW-Authenticate")
	require.Len(t, got, 1, "exactly one WWW-Authenticate header, no duplication from the middleware")
	assert.Equal(t,
		`Bearer resource_metadata="https://auth.example.com/.well-known/oauth-protected-resource"`,
		got[0])
	assert.NotContains(t, got[0], "engine-configured.example.com")
}
