package oauth2provider_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every token endpoint error carries an RFC 6749 section 5.2 code, so a
// client library can tell "ask the user again" (invalid_grant) from "your
// credentials are wrong" (invalid_client) without parsing prose.
func TestTokenErrors_UseRFC6749Codes(t *testing.T) {
	_, _, mux := newFixture(t)
	code := func() string { return codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID))) }

	for _, tc := range []struct {
		name   string
		body   map[string]string
		status int
		code   string
	}{
		{"missing grant_type", map[string]string{}, http.StatusBadRequest, "invalid_request"},
		{"unknown grant_type", map[string]string{"grant_type": "password"}, http.StatusBadRequest, "unsupported_grant_type"},
		{"missing code", map[string]string{"grant_type": "authorization_code"}, http.StatusBadRequest, "invalid_request"},
		{"unknown code", map[string]string{"grant_type": "authorization_code", "code": "nope"}, http.StatusBadRequest, "invalid_grant"},
		{"code for another client", map[string]string{
			"grant_type": "authorization_code", "code": code(), "client_id": publicID, "redirect_uri": registeredURI,
		}, http.StatusBadRequest, "invalid_grant"},
		{"no secret", map[string]string{
			"grant_type": "authorization_code", "code": code(), "client_id": confidentialID, "redirect_uri": registeredURI,
		}, http.StatusUnauthorized, "invalid_client"},
		{"wrong secret", map[string]string{
			"grant_type": "authorization_code", "code": code(), "client_id": confidentialID,
			"client_secret": "wrong", "redirect_uri": registeredURI,
		}, http.StatusUnauthorized, "invalid_client"},
		{"wrong redirect_uri", map[string]string{
			"grant_type": "authorization_code", "code": code(), "client_id": confidentialID,
			"client_secret": confidentialSecret, "redirect_uri": otherURI,
		}, http.StatusBadRequest, "invalid_grant"},
		{"client credentials without credentials", map[string]string{"grant_type": "client_credentials"}, http.StatusUnauthorized, "invalid_client"},
		{"client credentials for an unknown client", map[string]string{
			"grant_type": "client_credentials", "client_id": "nobody", "client_secret": "x",
		}, http.StatusUnauthorized, "invalid_client"},
		{"client credentials not registered", map[string]string{
			"grant_type": "client_credentials", "client_id": confidentialID, "client_secret": confidentialSecret,
		}, http.StatusBadRequest, "unauthorized_client"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := postToken(t, mux, tc.body)
			assert.Equal(t, tc.status, rec.Code, rec.Body.String())
			assert.Equal(t, tc.code, oauthErrorCode(t, rec))
			assert.Empty(t, rec.Header().Get("WWW-Authenticate"), "no Basic challenge when Basic wasn't used")
		})
	}
}

func TestTokenErrors_CodeReuseIsInvalidGrant(t *testing.T) {
	_, _, mux := newFixture(t)
	c := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))
	body := map[string]string{
		"grant_type": "authorization_code", "code": c, "client_id": confidentialID,
		"client_secret": confidentialSecret, "redirect_uri": registeredURI,
	}
	require.Equal(t, http.StatusOK, postToken(t, mux, body).Code)
	rec := postToken(t, mux, body)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_grant", oauthErrorCode(t, rec))
}

func TestTokenErrors_PKCE(t *testing.T) {
	_, _, mux := newFixture(t)
	q := baseAuthorizeQuery(publicID)
	q.Set("code_challenge", s256(oidcVerifier))
	q.Set("code_challenge_method", "S256")
	c := codeFrom(t, authorize(t, mux, q))

	base := map[string]string{"grant_type": "authorization_code", "code": c, "client_id": publicID, "redirect_uri": registeredURI}
	rec := postToken(t, mux, base)
	assert.Equal(t, "invalid_request", oauthErrorCode(t, rec), "a missing verifier is a malformed request")

	base["code_verifier"] = "the-wrong-verifier-the-wrong-verifier-123"
	rec = postToken(t, mux, base)
	assert.Equal(t, "invalid_grant", oauthErrorCode(t, rec), "a wrong one is a failed grant")
}

// A client that authenticated with Basic and failed gets the challenge for
// that scheme (RFC 6749 section 5.2).
func TestTokenErrors_BasicGetsAChallenge(t *testing.T) {
	_, _, mux := newFixture(t)
	form := url.Values{"grant_type": {"client_credentials"}}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(confidentialID+":wrong")))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	assert.Equal(t, "invalid_client", oauthErrorCode(t, rec))
	assert.Equal(t, `Basic realm="oauth2"`, rec.Header().Get("WWW-Authenticate"))
}

func TestRevokeErrors_MissingToken(t *testing.T) {
	_, _, mux := newFixture(t)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/oauth/revoke", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_request", oauthErrorCode(t, rec))
}

// Before the client and redirect URI check out, nothing is redirected:
// the error is answered directly, in the same body shape.
func TestAuthorizeErrors_BeforeTheRedirectURIIsTrusted(t *testing.T) {
	_, _, mux := newFixture(t)
	for name, tc := range map[string]struct {
		q    url.Values
		code string
	}{
		"no client_id":   {url.Values{"response_type": {"code"}}, "invalid_request"},
		"unknown client": {baseAuthorizeQuery("nobody"), "invalid_client"},
		"unregistered redirect": {func() url.Values {
			q := baseAuthorizeQuery(confidentialID)
			q.Set("redirect_uri", "https://evil.example.com/cb")
			return q
		}(), "invalid_request"},
		"ambiguous missing redirect": {func() url.Values { q := baseAuthorizeQuery(confidentialID); q.Del("redirect_uri"); return q }(), "invalid_request"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := authorize(t, mux, tc.q)
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.Empty(t, rec.Header().Get("Location"), "never redirected to an unverified URI")
			assert.Equal(t, tc.code, oauthErrorCode(t, rec))
		})
	}
}

func TestAuthorizeErrors_MissingResponseType(t *testing.T) {
	_, _, mux := newFixture(t)
	q := baseAuthorizeQuery(confidentialID)
	q.Del("response_type")
	code, _ := redirectedError(t, authorize(t, mux, q))
	assert.Equal(t, "invalid_request", code)
}

func TestAuthorizeErrors_UnsupportedResponseType(t *testing.T) {
	_, _, mux := newFixture(t)
	q := baseAuthorizeQuery(confidentialID)
	q.Set("response_type", "token")
	q.Set("state", "st-7")
	rec := authorize(t, mux, q)
	code, _ := redirectedError(t, rec)
	assert.Equal(t, "unsupported_response_type", code)
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "st-7", loc.Query().Get("state"))
}
