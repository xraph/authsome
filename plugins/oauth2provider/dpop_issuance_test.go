package oauth2provider_test

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

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/tokenformat"

	"golang.org/x/crypto/bcrypt"
)

// dpopIssuanceJWTKey is a fixed HMAC key for tests that need the engine to
// have a derivable DPoP nonce secret (Engine.NonceSecret reads it off any
// registered HMAC JWT format, regardless of which app key it's registered
// under) and/or need a specific app minting JWT-format access tokens.
var dpopIssuanceJWTKey = []byte("test-jwt-hmac-signing-key-at-least-32-bytes!!")

// ──────────────────────────────────────────────────
// Fixture: a plugin wired to a real engine.
//
// bindDPoP needs the engine accessors (DPoPModeForApp, DPoPValidator, ...),
// so unlike newFixture in authcode_test.go (which never sets p.engine), this
// fixture goes through secutil.NewTestEngine so p.engine is non-nil and the
// app-scoped dpop.mode setting actually resolves.
// ──────────────────────────────────────────────────

const (
	dpopIssuanceClientID    = "dpop-issuance-client"
	dpopIssuanceSecret      = "dpop-issuance-secret-value"
	dpopIssuanceRedirectURI = "https://app.example.com/cb"
	dpopTokenEndpoint       = "http://example.com/v1/oauth/token"
)

// dpopFixtureOpts configures newDPoPFixtureOpts. The zero value is the plain
// case: app mode off, client mode inherit, no nonce requirement, opaque
// tokens.
type dpopFixtureOpts struct {
	appMode       string
	clientMode    string
	nonceRequired bool
	jwtFormat     bool // mint the fixture's own app on JWT-format access tokens
}

// newDPoPFixtureOpts registers a confidential client (no PKCE to worry
// about) under a fresh app, applies the requested app-scoped settings, and
// returns the engine (for store lookups), router (for driving HTTP
// requests), and the app's ID (for tests that need to inspect it directly).
func newDPoPFixtureOpts(t *testing.T, opts dpopFixtureOpts) (*authsome.Engine, forge.Router, id.AppID) {
	t.Helper()

	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)

	appID := id.NewAppID()

	engOpts := []authsome.Option{authsome.WithPlugin(p)}
	if opts.nonceRequired || opts.jwtFormat {
		// A nonce signer needs a derivable HMAC secret (Engine.NonceSecret
		// reads it off any registered HMAC JWT format). When the fixture
		// also wants this specific app minting JWT tokens, register the
		// format under the app's own key so both concerns share one format;
		// otherwise register it under an unrelated key purely to give the
		// engine a secret to derive from, without changing this app's token
		// format away from opaque.
		jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
			SigningMethod: jwt.SigningMethodHS256,
			SigningKey:    dpopIssuanceJWTKey,
			VerifyKey:     dpopIssuanceJWTKey,
		})
		require.NoError(t, err)
		key := "dpop-issuance-nonce-secret-only"
		if opts.jwtFormat {
			key = appID.String()
		}
		engOpts = append(engOpts, authsome.WithJWTFormat(key, jwtFmt))
	}

	eng := secutil.NewTestEngine(t, engOpts...)

	hashed, err := bcrypt.GenerateFromPassword([]byte(dpopIssuanceSecret), bcrypt.MinCost)
	require.NoError(t, err)

	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     dpopIssuanceClientID,
		ClientSecret: string(hashed),
		Name:         "DPoP issuance client",
		RedirectURIs: []string{dpopIssuanceRedirectURI},
		Scopes:       []string{"openid"},
		GrantTypes:   []string{"authorization_code"},
		DPoPMode:     opts.clientMode,
	}))

	mgr := eng.Settings()
	require.NotNil(t, mgr)
	if opts.appMode != "" {
		raw, err := json.Marshal(opts.appMode)
		require.NoError(t, err)
		require.NoError(t, mgr.Set(context.Background(), "dpop.mode", raw,
			settings.ScopeApp, appID.String(), appID.String(), "", "test"))
	}
	if opts.nonceRequired {
		raw, err := json.Marshal(true)
		require.NoError(t, err)
		require.NoError(t, mgr.Set(context.Background(), "dpop.nonce_required", raw,
			settings.ScopeApp, appID.String(), appID.String(), "", "test"))
	}

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	return eng, mux, appID
}

// newDPoPFixture is the common case: app/client mode only, opaque tokens, no
// nonce requirement.
func newDPoPFixture(t *testing.T, appMode, clientMode string) (*authsome.Engine, forge.Router) {
	t.Helper()
	eng, mux, _ := newDPoPFixtureOpts(t, dpopFixtureOpts{appMode: appMode, clientMode: clientMode})
	return eng, mux
}

// dpopIssuanceCode drives GET /v1/oauth/authorize for the fixture's
// confidential client and returns the resulting authorization code.
func dpopIssuanceCode(t *testing.T, mux forge.Router) string {
	t.Helper()
	return codeFrom(t, authorize(t, mux, baseAuthorizeQuery(dpopIssuanceClientID)))
}

// postDPoPToken exchanges an authorization code at /v1/oauth/token,
// optionally attaching a DPoP proof header.
func postDPoPToken(t *testing.T, mux forge.Router, code, dpopHeader string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     dpopIssuanceClientID,
		"client_secret": dpopIssuanceSecret,
		"redirect_uri":  dpopIssuanceRedirectURI,
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/token", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if dpopHeader != "" {
		req.Header.Set("DPoP", dpopHeader)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// ──────────────────────────────────────────────────
// Proof-minting helpers, copied from middleware/auth_dpop_test.go (itself
// copied from dpop/proof_test.go) rather than exported across packages.
// ──────────────────────────────────────────────────

func dpopIssuanceKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

func dpopIssuanceThumbprint(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""
	jkt, err := jwkutil.Thumbprint(j)
	require.NoError(t, err)
	return jkt
}

// dpopIssuanceProof mints a proof bound to POST against the token endpoint.
func dpopIssuanceProof(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	return dpopIssuanceProofWithNonce(t, key, "")
}

// dpopIssuanceProofWithNonce mints a proof bound to POST against the token
// endpoint, optionally carrying a server-issued nonce (RFC 9449 §8). Each
// call gets a fresh jti from a monotonic counter rather than a
// wall-clock-derived one: two proofs minted in the same test within the same
// nanosecond (the retry-after-challenge tests mint two back to back) must
// not collide, or the second would look like a replay of the first.
func dpopIssuanceProofWithNonce(t *testing.T, key *ecdsa.PrivateKey, nonce string) string {
	t.Helper()

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": j}
	hb, err := json.Marshal(header)
	require.NoError(t, err)

	claims := map[string]any{
		"jti": fmt.Sprintf("proof-%d", dpopIssuanceNextJTI()),
		"htm": http.MethodPost,
		"htu": dpopTokenEndpoint,
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

// dpopIssuanceNextJTI hands out a unique counter value per test process,
// used to build jti claims that cannot collide within a single test.
var dpopIssuanceJTICounter int64

func dpopIssuanceNextJTI() int64 {
	dpopIssuanceJTICounter++
	return dpopIssuanceJTICounter
}

// ──────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────

// TestIssue_OptionalWithoutProofStaysBearer: the migration path. A client
// that knows nothing about DPoP keeps working unchanged.
func TestIssue_OptionalWithoutProofStaysBearer(t *testing.T) {
	eng, mux := newDPoPFixture(t, "optional", "")
	code := dpopIssuanceCode(t, mux)

	rec := postDPoPToken(t, mux, code, "")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp oauth2provider.TokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Bearer", resp.TokenType)
	require.NotEmpty(t, resp.AccessToken)

	sess, err := eng.Store().GetSessionByToken(context.Background(), resp.AccessToken)
	require.NoError(t, err)
	assert.Empty(t, sess.DPoPJKT)
}

// TestIssue_OptionalWithProofBinds: a client that proves possession of a key
// gets bound, and the response says so via token_type.
func TestIssue_OptionalWithProofBinds(t *testing.T) {
	eng, mux := newDPoPFixture(t, "optional", "")
	code := dpopIssuanceCode(t, mux)

	key := dpopIssuanceKey(t)
	proof := dpopIssuanceProof(t, key)
	wantJKT := dpopIssuanceThumbprint(t, key)

	rec := postDPoPToken(t, mux, code, proof)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp oauth2provider.TokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "DPoP", resp.TokenType)
	require.NotEmpty(t, resp.AccessToken)

	sess, err := eng.Store().GetSessionByToken(context.Background(), resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, wantJKT, sess.DPoPJKT)
}

// TestIssue_RequiredWithoutProofIsRejected: an app that mandates DPoP must
// refuse to issue an unbound token.
func TestIssue_RequiredWithoutProofIsRejected(t *testing.T) {
	_, mux := newDPoPFixture(t, "required", "")
	code := dpopIssuanceCode(t, mux)

	rec := postDPoPToken(t, mux, code, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())

	var oauthErr oauth2provider.OAuth2Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &oauthErr))
	assert.Equal(t, "invalid_dpop_proof", oauthErr.Error)
}

// TestIssue_ClientModeCannotWeakenApp is the monotonic rule at issuance: a
// per-client value must never bring the app's mandate down.
func TestIssue_ClientModeCannotWeakenApp(t *testing.T) {
	_, mux := newDPoPFixture(t, "required", "off")
	code := dpopIssuanceCode(t, mux)

	rec := postDPoPToken(t, mux, code, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())

	var oauthErr oauth2provider.OAuth2Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &oauthErr))
	assert.Equal(t, "invalid_dpop_proof", oauthErr.Error)
}

// TestIssue_ClientModeCanStrengthenApp: a client may demand more than the
// app requires.
func TestIssue_ClientModeCanStrengthenApp(t *testing.T) {
	_, mux := newDPoPFixture(t, "optional", "required")
	code := dpopIssuanceCode(t, mux)

	rec := postDPoPToken(t, mux, code, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())

	var oauthErr oauth2provider.OAuth2Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &oauthErr))
	assert.Equal(t, "invalid_dpop_proof", oauthErr.Error)
}

// TestIssue_NonceChallengeCanBeRetried is the property that used to be
// impossible: bindDPoP ran after ConsumeAuthCode, so the first attempt's
// use_dpop_nonce 400 burned the code, and the RFC-mandated retry (§8.2)
// always failed with "authorization code already used." bindDPoP now runs
// before the code is consumed, so the same code survives a nonce challenge
// and the retry succeeds.
func TestIssue_NonceChallengeCanBeRetried(t *testing.T) {
	eng, mux, _ := newDPoPFixtureOpts(t, dpopFixtureOpts{appMode: "optional", nonceRequired: true})
	code := dpopIssuanceCode(t, mux)

	key := dpopIssuanceKey(t)
	wantJKT := dpopIssuanceThumbprint(t, key)

	// First attempt: a valid proof, but no nonce. The app demands one.
	first := postDPoPToken(t, mux, code, dpopIssuanceProof(t, key))
	require.Equal(t, http.StatusBadRequest, first.Code, "body: %s", first.Body.String())

	var oauthErr oauth2provider.OAuth2Error
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &oauthErr))
	assert.Equal(t, "use_dpop_nonce", oauthErr.Error)

	nonce := first.Header().Get("DPoP-Nonce")
	require.NotEmpty(t, nonce, "the challenge must hand back a nonce to retry with")

	// Retry against the SAME code, now carrying the nonce. This is only
	// possible if the first attempt did not consume the code.
	second := postDPoPToken(t, mux, code, dpopIssuanceProofWithNonce(t, key, nonce))
	require.Equal(t, http.StatusOK, second.Code, "body: %s", second.Body.String())

	var resp oauth2provider.TokenResponse
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &resp))
	assert.Equal(t, "DPoP", resp.TokenType)

	sess, err := eng.Store().GetSessionByToken(context.Background(), resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, wantJKT, sess.DPoPJKT)
}

// TestIssue_JWTFormatCarriesCnfClaim guards the reason issueTokens threads
// jkt into the JWT claims at mint time instead of assigning sess.DPoPJKT
// after the fact: a JWT-format access token is validated statelessly from
// its own cnf claim (middleware.tryJWTAuth), not from the session row, so a
// binding that only lands in the DB is invisible to enforcement for these
// apps. Deleting the DPoPJKT claims line and running just this test should
// fail; see the mutation-check note in the task report.
func TestIssue_JWTFormatCarriesCnfClaim(t *testing.T) {
	_, mux, appID := newDPoPFixtureOpts(t, dpopFixtureOpts{appMode: "optional", jwtFormat: true})
	code := dpopIssuanceCode(t, mux)

	key := dpopIssuanceKey(t)
	wantJKT := dpopIssuanceThumbprint(t, key)

	rec := postDPoPToken(t, mux, code, dpopIssuanceProof(t, key))
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp oauth2provider.TokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "DPoP", resp.TokenType)

	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    dpopIssuanceJWTKey,
		VerifyKey:     dpopIssuanceJWTKey,
	})
	require.NoError(t, err)

	claims, err := jwtFmt.ValidateAccessToken(resp.AccessToken)
	require.NoError(t, err, "the access token must be a JWT this key can validate")
	assert.Equal(t, appID.String(), claims.AppID)
	assert.Equal(t, wantJKT, claims.DPoPJKT,
		"the JWT's own cnf.jkt claim must carry the thumbprint, not just the session row")
}
