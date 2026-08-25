package api_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/tokenformat"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// dpopSignInEndpoint is the exact URL middleware.RequestURL reconstructs for
// an httptest request to POST /v1/signin: no TLS (http), default httptest
// host (example.com), and the route path.
const dpopSignInEndpoint = "http://example.com/v1/signin"

// dpopSignInJWTKey is a fixed HMAC key used only to give the engine a
// derivable DPoP nonce secret (Engine.NonceSecret reads it off any
// registered HMAC JWT format). The platform app itself stays on opaque
// tokens; this key is never used to mint an access token in these tests.
var dpopSignInJWTKey = []byte("test-jwt-hmac-signing-key-at-least-32-bytes!!")

// newTestAPIWithNonceSigner builds the same engine newTestAPI (api_test.go)
// builds, plus an HMAC JWT format registered under an unrelated key so
// Engine.DPoPNonceSigner is non-nil. newTestAPI itself takes no options, so
// this is a separate constructor rather than a change to a shared helper.
func newTestAPIWithNonceSigner(t *testing.T) (*api.API, *authsome.Engine) {
	t.Helper()

	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    dpopSignInJWTKey,
		VerifyKey:     dpopSignInJWTKey,
	})
	require.NoError(t, err)

	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithJWTFormat("dpop-nonce-secret-only", jwtFmt),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))

	secutil.RelaxAuthDefaults(t, eng)

	return api.New(eng), eng
}

// setSignInDPoPMode sets the platform app's dpop.mode setting so
// Engine.DPoPModeForApp resolves it for handleSignIn/handleSignUp's calls
// into dpopBindingForRequest.
func setSignInDPoPMode(t *testing.T, eng *authsome.Engine, mode string) {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	mgr := eng.Settings()
	require.NotNil(t, mgr)
	raw, err := json.Marshal(mode)
	require.NoError(t, err)
	require.NoError(t, mgr.Set(context.Background(), "dpop.mode", raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))
}

// setSignInNonceRequired sets the platform app's dpop.nonce_required setting.
func setSignInNonceRequired(t *testing.T, eng *authsome.Engine, required bool) {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	mgr := eng.Settings()
	require.NotNil(t, mgr)
	raw, err := json.Marshal(required)
	require.NoError(t, err)
	require.NoError(t, mgr.Set(context.Background(), "dpop.nonce_required", raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))
}

// ──────────────────────────────────────────────────
// Proof-minting helpers, copied from middleware/auth_dpop_test.go /
// plugins/oauth2provider/dpop_issuance_test.go rather than exported across
// packages.
// ──────────────────────────────────────────────────

func dpopSignInKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

func dpopSignInThumbprint(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""
	jkt, err := jwkutil.Thumbprint(j)
	require.NoError(t, err)
	return jkt
}

// dpopSignInProof mints a proof bound to POST against the signin endpoint,
// optionally carrying a server-issued nonce.
func dpopSignInProof(t *testing.T, key *ecdsa.PrivateKey, nonce string) string {
	t.Helper()

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": j}
	hb, err := json.Marshal(header)
	require.NoError(t, err)

	claims := map[string]any{
		"jti": fmt.Sprintf("signin-proof-%d", time.Now().UnixNano()),
		"htm": http.MethodPost,
		"htu": dpopSignInEndpoint,
		"iat": time.Now().Unix(),
	}
	if nonce != "" {
		claims["nonce"] = nonce
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

func postSignIn(email, password, dpopHeader string) *http.Request {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if dpopHeader != "" {
		req.Header.Set("DPoP", dpopHeader)
	}
	return req
}

// ──────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────

// TestSignIn_OptionalWithProofBinds is the first-party equivalent of the
// OAuth2 plugin's TestIssue_OptionalWithProofBinds: dpopBindingForRequest and
// the SignInRequest.DPoPJKT threading through Engine.SignIn -> IssueSession
// went entirely unexercised beyond the brief's own tests, despite carrying
// the security argument for going beyond the brief's file list.
func TestSignIn_OptionalWithProofBinds(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())
	setSignInDPoPMode(t, eng, "optional")

	signUp(t, eng, "dpop-signin-bound@test.com", "SecureP@ss1")

	key := dpopSignInKey(t)
	wantJKT := dpopSignInThumbprint(t, key)

	req := postSignIn("dpop-signin-bound@test.com", "SecureP@ss1", dpopSignInProof(t, key, ""))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	token, _ := resp["session_token"].(string)
	require.NotEmpty(t, token)

	sess, err := eng.Store().GetSessionByToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, wantJKT, sess.DPoPJKT)
}

// TestSignIn_OptionalWithoutProofStaysUnbound is the migration-path sibling:
// a sign-in from a client that presents no proof under mode optional must
// keep issuing an ordinary unbound session.
func TestSignIn_OptionalWithoutProofStaysUnbound(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())
	setSignInDPoPMode(t, eng, "optional")

	signUp(t, eng, "dpop-signin-unbound@test.com", "SecureP@ss1")

	req := postSignIn("dpop-signin-unbound@test.com", "SecureP@ss1", "")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	token, _ := resp["session_token"].(string)
	require.NotEmpty(t, token)

	sess, err := eng.Store().GetSessionByToken(context.Background(), token)
	require.NoError(t, err)
	assert.Empty(t, sess.DPoPJKT)
}

// TestSignIn_NonceChallengeIsRetryable pins the fix for Important 1: a
// pre-authentication client has no other endpoint to obtain a nonce from, so
// a nonce-required app must answer a missing nonce with a retryable 400
// use_dpop_nonce plus a fresh DPoP-Nonce header (matching the OAuth2 token
// endpoint), rather than collapsing into a generic "invalid DPoP proof" that
// gives the client nowhere to go. Before the fix this test's second attempt
// would have gotten "invalid DPoP proof" back for the very retry the first
// response is supposed to enable.
func TestSignIn_NonceChallengeIsRetryable(t *testing.T) {
	a, eng := newTestAPIWithNonceSigner(t)
	handler := withTestKey(a.Handler())
	setSignInDPoPMode(t, eng, "optional")
	setSignInNonceRequired(t, eng, true)

	signUp(t, eng, "dpop-signin-nonce@test.com", "SecureP@ss1")

	key := dpopSignInKey(t)
	wantJKT := dpopSignInThumbprint(t, key)

	// First attempt: a valid proof, but no nonce.
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, postSignIn("dpop-signin-nonce@test.com", "SecureP@ss1", dpopSignInProof(t, key, "")))
	require.Equal(t, http.StatusBadRequest, first.Code, "body: %s", first.Body.String())

	var errBody map[string]any
	require.NoError(t, json.NewDecoder(first.Body).Decode(&errBody))
	assert.Equal(t, "use_dpop_nonce", errBody["error"])

	nonce := first.Header().Get("DPoP-Nonce")
	require.NotEmpty(t, nonce, "the challenge must hand back a nonce to retry with")

	// Retry, now carrying the nonce. Sign-in has no single-use artifact to
	// burn, so nothing prevents this from succeeding.
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, postSignIn("dpop-signin-nonce@test.com", "SecureP@ss1", dpopSignInProof(t, key, nonce)))
	require.Equal(t, http.StatusOK, second.Code, "body: %s", second.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(second.Body).Decode(&resp))
	token, _ := resp["session_token"].(string)
	require.NotEmpty(t, token)

	sess, err := eng.Store().GetSessionByToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, wantJKT, sess.DPoPJKT)
}
