package api_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
)

// dpopRefreshEndpoint is the exact URL middleware.RequestURL reconstructs for
// an httptest request to POST /v1/refresh: no TLS (http), default httptest
// host (example.com), and the route path.
const dpopRefreshEndpoint = "http://example.com/v1/refresh"

// dpopRefreshProof mints a proof bound to POST against dpopRefreshEndpoint.
// Reuses dpopSignInKey / dpopSignInThumbprint from dpop_signin_test.go
// (those helpers are not sign-in specific, just named after their first use).
func dpopRefreshProof(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": j}
	hb, err := json.Marshal(header)
	require.NoError(t, err)

	claims := map[string]any{
		"jti": fmt.Sprintf("refresh-proof-%d", time.Now().UnixNano()),
		"htm": http.MethodPost,
		"htu": dpopRefreshEndpoint,
		"iat": time.Now().Unix(),
	}
	cb, err := json.Marshal(claims)
	require.NoError(t, err)

	signing := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(cb)

	method := jwt.GetSigningMethod("ES256")
	require.NotNil(t, method)
	sig, err := method.Sign(signing, key)
	require.NoError(t, err)

	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// postRefresh builds a POST /v1/refresh request carrying the refresh token in
// the body (the pure-API-client path, no session cookie), optionally with a
// DPoP header.
func postRefresh(refreshToken, dpopHeader string) *http.Request {
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if dpopHeader != "" {
		req.Header.Set("DPoP", dpopHeader)
	}
	return req
}

// ──────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────

// TestRefresh_HTTP_BoundSessionWithValidProofSucceeds is the single point
// where this feature meets the wire: handleRefresh (api/auth_handlers.go)
// must actually read the DPoP header off the request and thread it, along
// with the method and reconstructed URL, into the RefreshOpts it hands to
// Engine.Refresh. Engine.Refresh's own enforcement is covered at the engine
// level (refresh_dpop_test.go in the root package); this test exists because
// deleting the three lines that populate DPoPProof/Method/RequestURL on the
// RefreshOpts literal is invisible to go build, go vet, and every other test
// in this package, and would silently lock every DPoP-bound client out of
// /refresh in production.
func TestRefresh_HTTP_BoundSessionWithValidProofSucceeds(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	key := dpopSignInKey(t)
	jkt := dpopSignInThumbprint(t, key)

	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     "http-refresh-bound@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Bound User",
		DPoPJKT:   jkt,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, postRefresh(sess.RefreshToken, dpopRefreshProof(t, key)))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.NotEmpty(t, resp["session_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

// TestRefresh_HTTP_BoundSessionWithoutProofIsRefused is the negative half of
// the same wiring check: a bound session refreshed over HTTP with no DPoP
// header at all must be refused, exactly as Engine.Refresh refuses it
// directly.
func TestRefresh_HTTP_BoundSessionWithoutProofIsRefused(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	key := dpopSignInKey(t)
	jkt := dpopSignInThumbprint(t, key)

	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     "http-refresh-noproof@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Bound User",
		DPoPJKT:   jkt,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, postRefresh(sess.RefreshToken, ""))
	assert.NotEqual(t, http.StatusOK, rec.Code,
		"a bound session refreshed over HTTP with no DPoP header must be refused, body: %s", rec.Body.String())
}
