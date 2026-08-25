package oauth2provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// postTokenForm drives POST /v1/oauth/token with an
// application/x-www-form-urlencoded body, which is the encoding RFC 6749
// §4.1.3 actually mandates. Every conformant OAuth2 client library sends this
// rather than JSON.
func postTokenForm(t *testing.T, mux forge.Router, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{}
	for k, v := range body {
		form.Set(k, v)
	}
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// ── form-encoded token requests (RFC 6749 §4.1.3) ────────────

func TestTokenExchange_AcceptsFormEncodedBody(t *testing.T) {
	_, _, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	rec := postTokenForm(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
		"redirect_uri":  registeredURI,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "Bearer", resp.TokenType)
}

// The token endpoint is one of two form-encoded endpoints that carry a client
// secret. Revocation is the other, so it gets the same coverage.
func TestRevoke_AcceptsFormEncodedBody(t *testing.T) {
	_, _, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	rec := postTokenForm(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
		"redirect_uri":  registeredURI,
	})
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var token struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &token))

	form := url.Values{"token": {token.AccessToken}}
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	revoked := httptest.NewRecorder()
	mux.ServeHTTP(revoked, req)

	assert.Equal(t, http.StatusOK, revoked.Code, "body: %s", revoked.Body.String())
}

// The device authorization endpoint is form-encoded too, and unlike the token
// endpoint it has no JSON callers to have masked the gap.
func TestDeviceAuthorize_AcceptsFormEncodedBody(t *testing.T) {
	_, st, mux := newFixture(t)

	const deviceClientID = "device-client"
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:         id.NewOAuth2ClientID(),
		AppID:      id.NewAppID(),
		ClientID:   deviceClientID,
		Name:       "Device",
		Scopes:     []string{"openid", "profile"},
		GrantTypes: []string{"urn:ietf:params:oauth:grant-type:device_code"},
		Public:     true,
	}))

	form := url.Values{"client_id": {deviceClientID}, "scope": {"openid profile"}}
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/device/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var resp struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.DeviceCode)
	assert.NotEmpty(t, resp.UserCode)
}

// A charset parameter on the Content-Type is common in the wild and must not
// change how the body is read.
func TestTokenExchange_AcceptsFormContentTypeWithCharset(t *testing.T) {
	_, _, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {confidentialID},
		"client_secret": {confidentialSecret},
		"redirect_uri":  {registeredURI},
	}
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// RFC 6749 §4.1.3 puts token parameters in the body. Binding from the merged
// query-plus-body set instead would let a URL supply the client secret, so the
// query string must stay inert here.
func TestTokenExchange_IgnoresCredentialsInQueryString(t *testing.T) {
	_, _, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	q := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {confidentialID},
		"client_secret": {confidentialSecret},
		"redirect_uri":  {registeredURI},
	}
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/token?"+q.Encode(), strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"query string must not stand in for the request body: %s", rec.Body.String())
}

// JSON bodies still bind, even though every field now carries a form tag too.
func TestTokenExchange_StillAcceptsJSONBody(t *testing.T) {
	_, _, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	rec := postToken(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
		"redirect_uri":  registeredURI,
	})

	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}

// Clients that send a charset alongside a JSON Content-Type are common, and
// the token endpoint has to accept them.
func TestTokenExchange_AcceptsJSONContentTypeWithCharset(t *testing.T) {
	_, _, mux := newFixture(t)

	code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

	body, err := json.Marshal(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
		"redirect_uri":  registeredURI,
	})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
}
