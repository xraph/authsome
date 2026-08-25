package oauth2provider_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"

	"golang.org/x/crypto/bcrypt"
)

// registerRawClient adds a confidential client with credentials given verbatim,
// so a test can pick an id or secret that needs encoding on the wire.
func registerRawClient(t *testing.T, st oauth2provider.Store, clientID, secret string) {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        id.NewAppID(),
		ClientID:     clientID,
		ClientSecret: string(hashed),
		Name:         "Raw",
		RedirectURIs: []string{registeredURI},
		Scopes:       []string{"openid", "profile"},
		GrantTypes:   []string{"client_credentials"},
	}))
}

// postTokenAuth posts a token request carrying a raw Authorization header, so a
// test controls exactly what encoding reaches the server.
func postTokenAuth(t *testing.T, mux forge.Router, authHeader string, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/token", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// basicHeader builds the header the way a well-behaved client does: percent
// encode each half first (RFC 6749 §2.3.1), then base64 the joined pair.
func basicHeader(clientID, secret string) string {
	pair := url.QueryEscape(clientID) + ":" + url.QueryEscape(secret)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(pair))
}

// rawBasicHeader base64s the pair with no encoding at all, the way a naive
// client library does.
func rawBasicHeader(clientID, secret string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"+secret))
}

// ── client_secret_basic across every grant (RFC 6749 §2.3.1) ────────────

// Discovery advertises client_secret_basic, so the authorization_code grant has
// to honour it and not just client_secret_post.
func TestTokenExchange_AcceptsBasicAuth(t *testing.T) {
	_, _, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	rec := postTokenAuth(t, mux, basicHeader(confidentialID, confidentialSecret), map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": registeredURI,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
}

func TestClientCredentials_AcceptsBasicAuth(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := postTokenAuth(t, mux, basicHeader(confidentialID, confidentialSecret), map[string]string{
		"grant_type": "client_credentials",
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
}

func TestDeviceCodeGrant_AcceptsBasicAuth(t *testing.T) {
	_, st, mux := newFixture(t)
	registerDeviceClients(t, st)

	deviceCode := startDeviceFlow(t, mux, deviceConfID)
	approveDeviceCode(t, st, deviceCode)

	rec := postTokenAuth(t, mux, basicHeader(deviceConfID, deviceConfSecret), map[string]string{
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		"device_code": deviceCode,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
}

func TestTokenEndpoint_RejectsWrongSecretInBasicAuth(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := postTokenAuth(t, mux, basicHeader(confidentialID, "not-the-secret"), map[string]string{
		"grant_type": "client_credentials",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
}

func TestTokenEndpoint_RejectsMalformedBasicAuth(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := postTokenAuth(t, mux, "Basic !!!not-base64!!!", map[string]string{
		"grant_type": "client_credentials",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "invalid_client", oauthErrorCode(t, rec))
}

// ── one authentication method per request (RFC 6749 §2.3) ───────────────

// Two sets of credentials in one request is a client bug, and guessing which
// one to trust is how an attacker who controls only the body gets to override
// the header. Refuse the request instead.
func TestTokenEndpoint_RejectsBasicAndBodySecretTogether(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := postTokenAuth(t, mux, basicHeader(confidentialID, confidentialSecret), map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "invalid_request", oauthErrorCode(t, rec))
}

// A body client_id that agrees with the header is common and harmless. One
// that disagrees means the request cannot say who it is claiming to be.
func TestTokenEndpoint_RejectsBasicClientIDConflictingWithBody(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := postTokenAuth(t, mux, basicHeader(confidentialID, confidentialSecret), map[string]string{
		"grant_type": "client_credentials",
		"client_id":  publicID,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "invalid_request", oauthErrorCode(t, rec))
}

func TestTokenEndpoint_AcceptsBasicWithAgreeingBodyClientID(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := postTokenAuth(t, mux, basicHeader(confidentialID, confidentialSecret), map[string]string{
		"grant_type": "client_credentials",
		"client_id":  confidentialID,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// ── credential encoding on the wire ─────────────────────────────────────

// A colon is the field separator, so a client_id containing one has to arrive
// percent encoded or the split lands in the wrong place. Decoding is what makes
// that id usable at all.
func TestTokenEndpoint_BasicAuthDecodesPercentEncodedCredentials(t *testing.T) {
	_, st, mux := newFixture(t)
	registerRawClient(t, st, "svc:billing", "p@ss:word/with%stuff")

	rec := postTokenAuth(t, mux, basicHeader("svc:billing", "p@ss:word/with%stuff"), map[string]string{
		"grant_type": "client_credentials",
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// A plus sign in a secret is ordinary (base64 secrets are full of them) and
// plenty of client libraries base64 the pair without encoding it first. Under
// form decoding that plus would silently become a space and the compare would
// fail, so decoding stops at percent escapes.
func TestTokenEndpoint_BasicAuthKeepsPlusSignInSecret(t *testing.T) {
	_, st, mux := newFixture(t)
	registerRawClient(t, st, "svc-plus", "ab+cd/ef+gh=")

	rec := postTokenAuth(t, mux, rawBasicHeader("svc-plus", "ab+cd/ef+gh="), map[string]string{
		"grant_type": "client_credentials",
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// ── the existing body method keeps working ──────────────────────────────

func TestTokenEndpoint_BodyCredentialsStillWorkWithoutBasicHeader(t *testing.T) {
	_, _, mux := newFixture(t)

	rec := postToken(t, mux, map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}
