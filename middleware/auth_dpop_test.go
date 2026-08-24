package middleware_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Proof-minting helpers, copied from dpop/proof_test.go rather than exported
// across packages.
// ──────────────────────────────────────────────────

func dpopTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

// dpopMintProof builds a signed DPoP proof. Fields left out of claims are
// simply absent, so a test can produce a proof missing exactly one claim.
func dpopMintProof(t *testing.T, key *ecdsa.PrivateKey, alg string, claims map[string]any) string { //nolint:unparam // alg mirrors dpop/proof_test.go's mintProof signature; every test here happens to use ES256
	t.Helper()

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	// A proof's jwk carries no use or alg member; strip them so the
	// thumbprint the server computes matches the one the client computed.
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": "dpop+jwt", "alg": alg, "jwk": j}
	hb, err := json.Marshal(header)
	require.NoError(t, err)
	cb, err := json.Marshal(claims)
	require.NoError(t, err)

	signing := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(cb)

	method := jwt.GetSigningMethod(alg)
	require.NotNil(t, method, "unknown alg %q", alg)
	sig, err := method.Sign(signing, key)
	require.NoError(t, err)

	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// dpopThumbprint computes the RFC 7638 thumbprint a proof minted with key
// will carry, so a test can bind a session to it up front.
func dpopThumbprint(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""
	jkt, err := jwkutil.Thumbprint(j)
	require.NoError(t, err)
	return jkt
}

// dpopValidClaims builds the mandatory claim set for a GET proof against htu.
// Every test in this file exercises the same GET /test route, so htm is fixed
// here rather than threaded through as a parameter nothing varies.
func dpopValidClaims(htu string) map[string]any { //nolint:unparam // htu is a real proof claim; every test here happens to target the same route
	return map[string]any{
		"jti": "proof-1",
		"htm": http.MethodGet,
		"htu": htu,
		"iat": time.Now().Unix(),
	}
}

// ──────────────────────────────────────────────────
// DPoP enforcement tests
// ──────────────────────────────────────────────────

// dpopTestSetup wires an AuthMiddleware whose SessionBindingConfig carries a
// real dpop.Validator, so enforcement runs exactly the way the engine would
// configure it.
func dpopTestSetup(t *testing.T, sess *session.Session, bind middleware.SessionBindingConfig) forge.Router {
	t.Helper()

	if bind.DPoPValidator == nil {
		bind.DPoPValidator = dpop.NewValidator(dpop.Config{})
	}

	u := &user.User{ID: sess.UserID, AppID: sess.AppID, Email: "dpop@test.com"}

	mw := middleware.AuthMiddleware(
		func(token string) (*session.Session, error) {
			if token == sess.Token {
				return sess, nil
			}
			return nil, errors.New("invalid")
		},
		func(userIDStr string) (*user.User, error) {
			if userIDStr == sess.UserID.String() {
				return u, nil
			}
			return nil, errors.New("not found")
		},
		log.NewNoopLogger(),
		bind,
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, ok := middleware.UserFrom(ctx.Context())
		if !ok {
			return forge.InternalError(errors.New("handler reached without user context"))
		}
		return ctx.NoContent(http.StatusOK)
	})

	return router
}

// TestDPoP_UnboundTokenIsUnaffected is the regression guard for the whole
// feature. A session with an empty DPoPJKT must behave exactly as it does
// today, with no proof, no DPoP scheme and no new header.
func TestDPoP_UnboundTokenIsUnaffected(t *testing.T) {
	sess := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  id.NewAppID(),
		UserID: id.NewUserID(),
		Token:  "unbound-token",
		// DPoPJKT left empty on purpose.
	}

	router := dpopTestSetup(t, sess, middleware.SessionBindingConfig{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "Bearer unbound-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestDPoP_BoundTokenWithoutProofIs401 is the core enforcement rule.
func TestDPoP_BoundTokenWithoutProofIs401(t *testing.T) {
	key := dpopTestKey(t)
	sess := &session.Session{
		ID:      id.NewSessionID(),
		AppID:   id.NewAppID(),
		UserID:  id.NewUserID(),
		Token:   "bound-token",
		DPoPJKT: dpopThumbprint(t, key),
	}

	router := dpopTestSetup(t, sess, middleware.SessionBindingConfig{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "DPoP bound-token")
	// Deliberately no DPoP header.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a bound token with no proof must be refused")
}

// TestDPoP_BoundTokenUnderBearerSchemeIs401: strict scheme matching. A bound
// token presented as Bearer is refused even with a valid proof alongside.
func TestDPoP_BoundTokenUnderBearerSchemeIs401(t *testing.T) {
	key := dpopTestKey(t)
	jkt := dpopThumbprint(t, key)
	sess := &session.Session{
		ID:      id.NewSessionID(),
		AppID:   id.NewAppID(),
		UserID:  id.NewUserID(),
		Token:   "bound-token",
		DPoPJKT: jkt,
	}

	router := dpopTestSetup(t, sess, middleware.SessionBindingConfig{})

	claims := dpopValidClaims("http://example.com/test")
	claims["ath"] = dpop.AccessTokenHash("bound-token")
	proof := dpopMintProof(t, key, "ES256", claims)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "Bearer bound-token")
	req.Header.Set("DPoP", proof)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a bound token presented as Bearer must be refused even with a valid proof")
}

// TestDPoP_WrongKeyIs401: a structurally valid proof signed by a different key.
func TestDPoP_WrongKeyIs401(t *testing.T) {
	keyA := dpopTestKey(t)
	keyB := dpopTestKey(t)

	sess := &session.Session{
		ID:      id.NewSessionID(),
		AppID:   id.NewAppID(),
		UserID:  id.NewUserID(),
		Token:   "bound-token",
		DPoPJKT: dpopThumbprint(t, keyA), // bound to A
	}

	router := dpopTestSetup(t, sess, middleware.SessionBindingConfig{})

	claims := dpopValidClaims("http://example.com/test")
	claims["ath"] = dpop.AccessTokenHash("bound-token")
	proof := dpopMintProof(t, keyB, "ES256", claims) // signed with B

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "DPoP bound-token")
	req.Header.Set("DPoP", proof)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a proof signed by the wrong key must be refused")
}

// TestDPoP_MissingATHIs401: without ath a proof captured on one endpoint
// replays against every other one.
func TestDPoP_MissingATHIs401(t *testing.T) {
	key := dpopTestKey(t)
	sess := &session.Session{
		ID:      id.NewSessionID(),
		AppID:   id.NewAppID(),
		UserID:  id.NewUserID(),
		Token:   "bound-token",
		DPoPJKT: dpopThumbprint(t, key),
	}

	router := dpopTestSetup(t, sess, middleware.SessionBindingConfig{})

	claims := dpopValidClaims("http://example.com/test")
	// ath deliberately omitted.
	proof := dpopMintProof(t, key, "ES256", claims)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "DPoP bound-token")
	req.Header.Set("DPoP", proof)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a proof with no ath must be refused")
}

func TestDPoP_ValidProofPasses(t *testing.T) {
	key := dpopTestKey(t)
	sess := &session.Session{
		ID:      id.NewSessionID(),
		AppID:   id.NewAppID(),
		UserID:  id.NewUserID(),
		Token:   "bound-token",
		DPoPJKT: dpopThumbprint(t, key),
	}

	router := dpopTestSetup(t, sess, middleware.SessionBindingConfig{})

	claims := dpopValidClaims("http://example.com/test")
	claims["ath"] = dpop.AccessTokenHash("bound-token")
	proof := dpopMintProof(t, key, "ES256", claims)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "DPoP bound-token")
	req.Header.Set("DPoP", proof)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "a correct key, htm, htu and ath must pass")
}

// TestDPoP_NonceChallenge: when a nonce is required and absent, the response
// carries a DPoP-Nonce header the client can retry with.
func TestDPoP_NonceChallenge(t *testing.T) {
	secret := []byte("this-is-a-32-byte-test-secret!!")
	signer, err := dpop.NewNonceSigner(secret, dpop.DefaultNonceTTL)
	require.NoError(t, err)

	key := dpopTestKey(t)
	sess := &session.Session{
		ID:      id.NewSessionID(),
		AppID:   id.NewAppID(),
		UserID:  id.NewUserID(),
		Token:   "bound-token",
		DPoPJKT: dpopThumbprint(t, key),
	}

	validator := dpop.NewValidator(dpop.Config{Nonce: signer})

	router := dpopTestSetup(t, sess, middleware.SessionBindingConfig{
		DPoPValidator:   validator,
		DPoPNonceSigner: signer,
		DPoPNonceRequired: func(_ context.Context, _ string) bool {
			return true
		},
	})

	claims := dpopValidClaims("http://example.com/test")
	claims["ath"] = dpop.AccessTokenHash("bound-token")
	// Nonce deliberately omitted.
	proof := dpopMintProof(t, key, "ES256", claims)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "DPoP bound-token")
	req.Header.Set("DPoP", proof)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), `error="use_dpop_nonce"`)
	assert.NotEmpty(t, rec.Header().Get("DPoP-Nonce"), "a nonce challenge must hand the client something to retry with")
}
