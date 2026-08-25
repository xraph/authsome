package api_test

import (
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
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
)

// ──────────────────────────────────────────────────
// Refresh with a live access token in the header.
//
// The TypeScript client this branch ships sends Authorization: DPoP <token>
// on every call it has a token for, /refresh included. So a proactive refresh
// arrives with two things that both trigger RFC 9449 enforcement: the bound
// access token in the header, which the global auth middleware checks, and the
// proof, which Engine.Refresh checks again a few frames later.
//
// api/refresh_dpop_test.go covers /refresh without the header, which is what a
// reactive refresh after expiry looks like. That variant never met the double
// check, which is why the break showed up in production as an intermittent
// forced sign-out rather than as a red test.
// ──────────────────────────────────────────────────

// refreshChainHandler builds the API behind the engine's own auth middleware,
// which is how the extension serves it. newTestAPI's Handler() alone has no
// global auth middleware, so it cannot see this interaction at all.
func refreshChainHandler(t *testing.T, a *api.API, eng *authsome.Engine) http.Handler {
	t.Helper()

	router := forge.NewRouter()
	router.Use(eng.AuthMiddleware())
	require.NoError(t, a.RegisterRoutes(router))
	return withTestKey(router.Handler())
}

// refreshChainProof mints a proof for POST /v1/refresh carrying ath over the
// access token the request presents in the Authorization header, which is what
// the middleware's enforcement requires and what the shipped SDK sends.
func refreshChainProof(t *testing.T, key *ecdsa.PrivateKey, accessToken string) string {
	t.Helper()

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": j}
	hb, err := json.Marshal(header)
	require.NoError(t, err)

	claims := map[string]any{
		"jti": fmt.Sprintf("refresh-chain-proof-%d", time.Now().UnixNano()),
		"htm": http.MethodPost,
		"htu": dpopRefreshEndpoint,
		"iat": time.Now().Unix(),
		"ath": dpop.AccessTokenHash(accessToken),
	}
	cb, err := json.Marshal(claims)
	require.NoError(t, err)

	signing := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(cb)

	sig, err := jwt.GetSigningMethod("ES256").Sign(signing, key)
	require.NoError(t, err)

	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// TestRefresh_HTTP_ProactiveRefreshWithLiveAccessTokenSucceeds is
// manifestation (b) of the double-validation bug. The global middleware
// validated the proof and recorded its jti; Engine.Refresh then validated the
// same proof and found its own jti in the replay cache, so a client refreshing
// before expiry was signed out while a client refreshing after expiry was not.
func TestRefresh_HTTP_ProactiveRefreshWithLiveAccessTokenSucceeds(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := refreshChainHandler(t, a, eng)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	key := dpopSignInKey(t)
	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     "proactive-refresh@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Proactive",
		DPoPJKT:   dpopSignInThumbprint(t, key),
	})
	require.NoError(t, err)

	req := postRefresh(sess.RefreshToken, refreshChainProof(t, key, sess.Token))
	req.Header.Set("Authorization", "DPoP "+sess.Token)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"a refresh carrying a live bound access token and one proof must not be read as a replay; body: %s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.NotEmpty(t, resp["session_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

// TestRefresh_HTTP_ProactiveRefreshWithProofFromAnotherKeyIsRefused keeps the
// positive case above honest. The same request shape, a proof from a key the
// session is not bound to, and the request must still be refused.
func TestRefresh_HTTP_ProactiveRefreshWithProofFromAnotherKeyIsRefused(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := refreshChainHandler(t, a, eng)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	bound := dpopSignInKey(t)
	attacker := dpopSignInKey(t)
	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     "proactive-refresh-wrong-key@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Wrong Key",
		DPoPJKT:   dpopSignInThumbprint(t, bound),
	})
	require.NoError(t, err)

	req := postRefresh(sess.RefreshToken, refreshChainProof(t, attacker, sess.Token))
	req.Header.Set("Authorization", "DPoP "+sess.Token)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
}
