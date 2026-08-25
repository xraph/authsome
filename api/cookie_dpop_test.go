package api_test

import (
	"context"
	"crypto/ecdsa"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
)

// ──────────────────────────────────────────────────
// Bound sessions over cookie transport
//
// The engine bridges a session cookie into "Authorization: Bearer <token>"
// before the inner middleware runs, so extractCredential never saw a cookie in
// a real deployment and every bound cookie session failed on wrong_scheme.
// The spec said that could not happen because browser sign-in never binds.
// handleSignIn and handleSignUp bind through dpopBindingForRequest now, so
// under mode=required every first-party sign-in produced exactly the session
// that could never be used again.
//
// RFC 9449 section 7.1 is a rule about the Authorization header. A cookie is
// not one, so the scheme rule has nothing to match and what remains is the
// proof, which is required here as strictly as anywhere else.
// ──────────────────────────────────────────────────

const dpopMeEndpoint = "http://example.com/v1/me"

// dpopCookieName is the session cookie the engine writes when no per-app
// override is configured, and the one buildAuthMiddleware falls back to.
const dpopCookieName = "authsome_session_token"

// meProof mints a proof for GET /v1/me carrying ath over the session token.
func meProof(t *testing.T, key *ecdsa.PrivateKey, token string) string {
	t.Helper()

	claims := dpoptest.ValidClaims(http.MethodGet, dpopMeEndpoint)
	claims["ath"] = dpop.AccessTokenHash(token)
	return dpoptest.MintProof(t, key, "ES256", claims)
}

// getMe builds the GET /v1/me a browser makes, carrying the session in the
// cookie unless authHeader says otherwise.
func getMe(t *testing.T, token, authHeader, proof string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/me", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader+" "+token)
	} else {
		req.AddCookie(&http.Cookie{Name: dpopCookieName, Value: token})
	}
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	return req
}

// signUpBound creates a session bound to key, or unbound when key is nil.
func signUpBound(t *testing.T, eng *authsome.Engine, email string, key *ecdsa.PrivateKey) string {
	t.Helper()

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	jkt := ""
	if key != nil {
		jkt = dpoptest.Thumbprint(t, key)
	}
	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  "SecureP@ss1",
		FirstName: "Cookie",
		DPoPJKT:   jkt,
	})
	require.NoError(t, err)
	return sess.Token
}

// TestCookieTransport_UnboundSessionIsUnaffected is the migration guard. Every
// setting at its default, no binding, cookie transport: unchanged.
func TestCookieTransport_UnboundSessionIsUnaffected(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := refreshChainHandler(t, a, eng)

	token := signUpBound(t, eng, "cookie-unbound@example.com", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, getMe(t, token, "", ""))

	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// TestCookieTransport_BoundSessionWithProofSucceeds is the finding. Before the
// bridge recorded what it did, this was a hard 401 no client could recover
// from.
func TestCookieTransport_BoundSessionWithProofSucceeds(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := refreshChainHandler(t, a, eng)

	key := dpopSignInKey(t)
	token := signUpBound(t, eng, "cookie-bound@example.com", key)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, getMe(t, token, "", meProof(t, key, token)))

	assert.Equal(t, http.StatusOK, rec.Code,
		"a bound session presented by cookie with a valid proof must be honoured; body: %s", rec.Body.String())
}

// TestCookieTransport_BoundSessionWithoutProofIsRefused: honouring the cookie
// transport is not the same as exempting it. No proof, no session.
func TestCookieTransport_BoundSessionWithoutProofIsRefused(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := refreshChainHandler(t, a, eng)

	key := dpopSignInKey(t)
	token := signUpBound(t, eng, "cookie-bound-noproof@example.com", key)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, getMe(t, token, "", ""))

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
}

// TestCookieTransport_BoundSessionWithProofFromAnotherKeyIsRefused pins the
// key comparison on the cookie path, not just the presence of a DPoP header.
func TestCookieTransport_BoundSessionWithProofFromAnotherKeyIsRefused(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := refreshChainHandler(t, a, eng)

	bound := dpopSignInKey(t)
	attacker := dpopSignInKey(t)
	token := signUpBound(t, eng, "cookie-bound-wrongkey@example.com", bound)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, getMe(t, token, "", meProof(t, attacker, token)))

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
}

// TestCookieTransport_BearerHeaderStaysStrict is the half of section 7.1 that
// does not move. The same token, the same valid proof, presented in a real
// Authorization header under the wrong scheme, is still refused. A fix that
// relaxed the scheme rule outright rather than scoping it to credentials that
// arrived in that header would pass this by accident, so it presents a proof
// that would otherwise succeed.
func TestCookieTransport_BearerHeaderStaysStrict(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := refreshChainHandler(t, a, eng)

	key := dpopSignInKey(t)
	token := signUpBound(t, eng, "cookie-bound-bearer@example.com", key)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, getMe(t, token, "Bearer", meProof(t, key, token)))

	assert.Equal(t, http.StatusUnauthorized, rec.Code,
		"RFC 9449 section 7.1 still governs the Authorization header; body: %s", rec.Body.String())
}

// TestCookieTransport_DPoPSchemeStillWorks: the header path a native client
// uses is untouched by any of this.
func TestCookieTransport_DPoPSchemeStillWorks(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := refreshChainHandler(t, a, eng)

	key := dpopSignInKey(t)
	token := signUpBound(t, eng, "cookie-bound-dpopscheme@example.com", key)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, getMe(t, token, "DPoP", meProof(t, key, token)))

	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}
