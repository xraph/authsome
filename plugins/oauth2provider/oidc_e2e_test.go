package oauth2provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

// startOIDCServer serves a fixture's router over real HTTP with the plugin's
// issuer set to the server's own URL, which go-oidc requires: it fetches
// discovery and the JWKS over the network and insists the discovery issuer
// equals the URL it was given.
func startOIDCServer(t *testing.T) (*oidcFixture, string) {
	t.Helper()
	var h http.Handler
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}))
	base := "http://" + srv.Listener.Addr().String()
	f := newOIDCFixture(t, oauth2provider.Config{Issuer: base}, true)
	h = f.mux
	srv.Start()
	t.Cleanup(srv.Close)
	return f, srv.URL
}

// tokenRequest posts a form to the real server's token endpoint.
func tokenRequest(t *testing.T, base string, form url.Values) map[string]any {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		base+"/v1/oauth/token", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, http.StatusOK, resp.StatusCode, "token response: %v", body)
	return body
}

// runGoOIDC drives discovery, authorize, code exchange and refresh against
// issuer with go-oidc as the relying party.
func runGoOIDC(t *testing.T, f *oidcFixture, base, issuer string) {
	t.Helper()
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, issuer)
	require.NoError(t, err, "go-oidc must accept the discovery document")
	verifier := provider.Verifier(&oidc.Config{ClientID: publicID})

	q := baseAuthorizeQuery(publicID)
	q.Set("scope", "openid email")
	q.Set("nonce", "n-e2e")
	q.Set("code_challenge", s256(oidcVerifier))
	q.Set("code_challenge_method", "S256")
	code := codeFrom(t, authorizeWith(sessionCtx(f.user.ID, f.appID, time.Now()), f.mux, q))

	first := tokenRequest(t, base, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {registeredURI},
		"client_id": {publicID}, "code_verifier": {oidcVerifier},
	})
	raw, ok := first["id_token"].(string)
	require.True(t, ok, "id_token missing: %v", first)

	idToken, err := verifier.Verify(ctx, raw)
	require.NoError(t, err)
	assert.Equal(t, issuer, idToken.Issuer)
	assert.Equal(t, f.user.ID.String(), idToken.Subject)
	assert.Equal(t, "n-e2e", idToken.Nonce)
	assert.NoError(t, idToken.VerifyAccessToken(first["access_token"].(string)))
	var claims struct {
		Email string `json:"email"`
	}
	require.NoError(t, idToken.Claims(&claims))
	assert.Equal(t, "oidc@example.com", claims.Email)

	refresh, ok := first["refresh_token"].(string)
	require.True(t, ok, "refresh_token missing: %v", first)
	second := tokenRequest(t, base, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {refresh}, "client_id": {publicID},
	})
	raw2, ok := second["id_token"].(string)
	require.True(t, ok, "refresh id_token missing: %v", second)
	refreshed, err := verifier.Verify(ctx, raw2)
	require.NoError(t, err)
	assert.Equal(t, issuer, refreshed.Issuer)
	assert.Equal(t, f.user.ID.String(), refreshed.Subject)
	assert.Empty(t, refreshed.Nonce, "a refreshed id_token carries no nonce")
	assert.NoError(t, refreshed.VerifyAccessToken(second["access_token"].(string)))
}

func TestOIDC_GoOIDCVerifiesDeploymentIssuer(t *testing.T) {
	f, base := startOIDCServer(t)
	runGoOIDC(t, f, base, base)
}

func TestOIDC_GoOIDCVerifiesAppIssuer(t *testing.T) {
	f, base := startOIDCServer(t)
	setAppIssuer(t, f, f.appID, true)
	runGoOIDC(t, f, base, base+"/apps/"+platformSlug)
}
