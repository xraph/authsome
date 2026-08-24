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

	"golang.org/x/crypto/bcrypt"
)

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

// newDPoPFixture registers a confidential client (no PKCE to worry about)
// under a fresh app, sets the app's dpop.mode setting, and returns the
// engine (for store lookups) and router (for driving HTTP requests).
func newDPoPFixture(t *testing.T, appMode, clientMode string) (*authsome.Engine, forge.Router) {
	t.Helper()

	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)

	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))

	appID := id.NewAppID()

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
		DPoPMode:     clientMode,
	}))

	if appMode != "" {
		mgr := eng.Settings()
		require.NotNil(t, mgr)
		raw, err := json.Marshal(appMode)
		require.NoError(t, err)
		require.NoError(t, mgr.Set(context.Background(), "dpop.mode", raw,
			settings.ScopeApp, appID.String(), appID.String(), "", "test"))
	}

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

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

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": j}
	hb, err := json.Marshal(header)
	require.NoError(t, err)

	claims := map[string]any{
		"jti": fmt.Sprintf("proof-%d", time.Now().UnixNano()),
		"htm": http.MethodPost,
		"htu": dpopTokenEndpoint,
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
