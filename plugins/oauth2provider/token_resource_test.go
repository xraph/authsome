package oauth2provider_test

import (
	"bytes"
	"context"
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
	authstore "github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"

	"golang.org/x/crypto/bcrypt"
)

// tokenFixture is like newFixture but also hands back the account store. An
// opaque access token (the fixture runs no JWT engine) carries no claims of
// its own, so the session's Audience field is the only place a test can read
// back what a grant actually issued.
func tokenFixture(t *testing.T) (oauth2provider.Store, authstore.Store, forge.Router) {
	t.Helper()
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	acct := memory.New()
	p.SetStore(acct)

	hashed, err := bcrypt.GenerateFromPassword([]byte(confidentialSecret), bcrypt.MinCost)
	require.NoError(t, err)

	appID := id.NewAppID()
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     confidentialID,
		ClientSecret: string(hashed),
		Name:         "Confidential",
		RedirectURIs: []string{registeredURI, otherURI},
		Scopes:       []string{"openid", "profile"},
		GrantTypes:   []string{"authorization_code", "client_credentials", "urn:ietf:params:oauth:grant-type:device_code"},
	}))
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     publicID,
		Name:         "Public",
		RedirectURIs: []string{registeredURI},
		Scopes:       []string{"openid", "profile"},
		GrantTypes:   []string{"authorization_code"},
		Public:       true,
	}))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return st, acct, mux
}

// postTokenJSON posts an arbitrary JSON body to /v1/oauth/token. Unlike
// postToken (map[string]string), this accepts the "resource" field as a
// []string. The body is JSON, not form-encoded, so resourceParams finds
// nothing in the query or the form, so any resource value here can only
// have reached the grant through TokenRequest.Resource's encoding/json tag.
func postTokenJSON(t *testing.T, mux forge.Router, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/token", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// postDeviceAuthorize posts a JSON body carrying client_id to
// /v1/oauth/device/authorize, with the resource indicators on the query
// string. DeviceAuthRequest has no field for "resource": it is read
// straight off the request by resourceParams, which checks the query string
// on every method, so putting it there exercises that path without touching
// the JSON-bound client_id. Repeating the parameter is how RFC 8707 asks for
// more than one resource.
func postDeviceAuthorize(t *testing.T, mux forge.Router, clientID string, resources ...string) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(map[string]string{"client_id": clientID})
	require.NoError(t, err)
	q := url.Values{}
	for _, r := range resources {
		q.Add("resource", r)
	}
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/oauth/device/authorize?"+q.Encode(), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeTokenResponse(t *testing.T, rec *httptest.ResponseRecorder) oauth2provider.TokenResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var resp oauth2provider.TokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	return resp
}

func audienceFor(t *testing.T, acct authstore.Store, accessToken string) []string {
	t.Helper()
	sess, err := acct.GetSessionByToken(context.Background(), accessToken)
	require.NoError(t, err)
	return sess.Audience
}

func TestTokenResource(t *testing.T) {
	// Case 1: authorize with two resources, redeem with one. The token's
	// audience must be exactly the one requested at the token endpoint, not
	// the full granted set. That is what proves narrowing (not just
	// pass-through) is happening.
	t.Run("redemption narrows to the resource requested at the token endpoint", func(t *testing.T) {
		st, acct, mux := tokenFixture(t)
		grantResources(t, st, resAPI, resFiles)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)
		q.Add("resource", resFiles)
		code := codeFrom(t, authorize(t, mux, q))

		rec := postTokenJSON(t, mux, map[string]any{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     confidentialID,
			"client_secret": confidentialSecret,
			"redirect_uri":  registeredURI,
			"resource":      []string{resAPI},
		})

		resp := decodeTokenResponse(t, rec)
		assert.Equal(t, []string{resAPI}, audienceFor(t, acct, resp.AccessToken))
	})

	// Case 2: authorize with one resource, redeem asking for a different one.
	// Must be refused, and refused specifically because it was never granted
	// (narrowResources' rejection message), not because it is unregistered
	// with the client, which is resolveResources' rejection message. The two
	// checks share the invalid_target code, so only the wording tells them
	// apart.
	t.Run("redemption cannot widen past what the code was granted", func(t *testing.T) {
		st, _, mux := tokenFixture(t)
		grantResources(t, st, resAPI, resFiles)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)
		code := codeFrom(t, authorize(t, mux, q))

		rec := postTokenJSON(t, mux, map[string]any{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     confidentialID,
			"client_secret": confidentialSecret,
			"redirect_uri":  registeredURI,
			"resource":      []string{resFiles},
		})

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_target")
		assert.Contains(t, rec.Body.String(), "was not granted by this authorization",
			"must fail via narrowResources' rejection, not resolveResources'")
		assert.NotContains(t, rec.Body.String(), "access_token")
	})

	// Case 3: authorize with two resources, redeem with no resource field.
	// The token must carry both. Omission inherits the whole granted set
	// rather than clearing it.
	t.Run("omitting resource at redemption inherits the whole granted set", func(t *testing.T) {
		st, acct, mux := tokenFixture(t)
		grantResources(t, st, resAPI, resFiles)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)
		q.Add("resource", resFiles)
		code := codeFrom(t, authorize(t, mux, q))

		rec := postToken(t, mux, map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     confidentialID,
			"client_secret": confidentialSecret,
			"redirect_uri":  registeredURI,
		})

		resp := decodeTokenResponse(t, rec)
		assert.Equal(t, []string{resAPI, resFiles}, audienceFor(t, acct, resp.AccessToken))
	})

	// Case 4: client credentials against a registered resource. There is no
	// prior authorization to narrow here, so this grant must validate against
	// the client's own allowlist (resolveResources), and the session must
	// carry the result.
	t.Run("client credentials carries a registered resource", func(t *testing.T) {
		st, acct, mux := tokenFixture(t)
		grantResources(t, st, resAPI)

		rec := postTokenJSON(t, mux, map[string]any{
			"grant_type":    "client_credentials",
			"client_id":     confidentialID,
			"client_secret": confidentialSecret,
			"resource":      []string{resAPI},
		})

		resp := decodeTokenResponse(t, rec)
		assert.Equal(t, []string{resAPI}, audienceFor(t, acct, resp.AccessToken))
	})

	// Discrimination companion to case 4: client credentials has no prior
	// grant to narrow against, so an unregistered resource must be refused by
	// the allowlist rule (resolveResources' wording), not the narrowing rule
	// (narrowResources' wording). A client-credentials path wrongly wired
	// through narrowResources(nil, requested) would also reject the request,
	// but with the wrong message. This is what catches that.
	t.Run("client credentials refuses an unregistered resource via the allowlist rule", func(t *testing.T) {
		st, _, mux := tokenFixture(t)
		grantResources(t, st, resAPI)

		rec := postTokenJSON(t, mux, map[string]any{
			"grant_type":    "client_credentials",
			"client_id":     confidentialID,
			"client_secret": confidentialSecret,
			"resource":      []string{resOther},
		})

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_target")
		assert.Contains(t, rec.Body.String(), "is not registered for this client",
			"must fail via resolveResources' rejection, not narrowResources'")
	})

	// Case 5: device authorize with a resource, then poll after approval. The
	// polled token must carry it. Approval is applied directly on the store
	// rather than through /device/complete, which sits behind session
	// middleware this fixture does not wire up. The polling path under test
	// is the token endpoint, not the completion endpoint.
	t.Run("device flow carries the resource through to the polled token", func(t *testing.T) {
		st, acct, mux := tokenFixture(t)
		grantResources(t, st, resAPI)

		authRec := postDeviceAuthorize(t, mux, confidentialID, resAPI)
		require.Equal(t, http.StatusOK, authRec.Code, "body: %s", authRec.Body.String())
		var authResp oauth2provider.DeviceAuthResponse
		require.NoError(t, json.Unmarshal(authRec.Body.Bytes(), &authResp))
		require.NotEmpty(t, authResp.DeviceCode)

		dc, err := st.GetDeviceCodeByDeviceCode(context.Background(), authResp.DeviceCode)
		require.NoError(t, err)
		assert.Equal(t, []string{resAPI}, dc.Resources,
			"the device code must carry the resource resolved at /device/authorize")

		dc.Status = oauth2provider.DeviceCodeStatusAuthorized
		dc.UserID = id.NewUserID()
		require.NoError(t, st.UpdateDeviceCode(context.Background(), dc))

		rec := postToken(t, mux, map[string]string{
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
			"device_code": authResp.DeviceCode,
			"client_id":   confidentialID,
		})

		resp := decodeTokenResponse(t, rec)
		assert.Equal(t, []string{resAPI}, audienceFor(t, acct, resp.AccessToken))
	})

	// Case 5a: the device grant must narrow the same way the authorization
	// code grant does. Authorize for two resources, redeem asking for one, and
	// the token must carry only that one. Case 5 redeems with no resource at
	// all, so it only ever exercises the pass-through branch: delete the
	// narrowing call from the device grant entirely and case 5 stays green.
	t.Run("device redemption narrows to the resource requested at the token endpoint", func(t *testing.T) {
		st, acct, mux := tokenFixture(t)
		grantResources(t, st, resAPI, resFiles)

		authRec := postDeviceAuthorize(t, mux, confidentialID, resAPI, resFiles)
		require.Equal(t, http.StatusOK, authRec.Code, "body: %s", authRec.Body.String())
		var authResp oauth2provider.DeviceAuthResponse
		require.NoError(t, json.Unmarshal(authRec.Body.Bytes(), &authResp))

		dc, err := st.GetDeviceCodeByDeviceCode(context.Background(), authResp.DeviceCode)
		require.NoError(t, err)
		require.Equal(t, []string{resAPI, resFiles}, dc.Resources,
			"the device code must carry both resources resolved at /device/authorize")

		dc.Status = oauth2provider.DeviceCodeStatusAuthorized
		dc.UserID = id.NewUserID()
		require.NoError(t, st.UpdateDeviceCode(context.Background(), dc))

		rec := postTokenJSON(t, mux, map[string]any{
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
			"device_code": authResp.DeviceCode,
			"client_id":   confidentialID,
			"resource":    []string{resFiles},
		})

		resp := decodeTokenResponse(t, rec)
		assert.Equal(t, []string{resFiles}, audienceFor(t, acct, resp.AccessToken),
			"the device grant widened the token back to everything the code was granted")
	})

	// Case 5b: the widening half. A device redemption naming a resource the
	// user never approved must be refused, and refused by the narrowing rule
	// rather than the client allowlist rule. resOther is not registered with
	// the client either, so a device grant wrongly wired through
	// resolveResources would also reject it, with the wrong message. Only the
	// wording tells the two apart, since both answer invalid_target.
	t.Run("device redemption cannot widen past what the device code was granted", func(t *testing.T) {
		st, _, mux := tokenFixture(t)
		grantResources(t, st, resAPI, resFiles)

		authRec := postDeviceAuthorize(t, mux, confidentialID, resAPI)
		require.Equal(t, http.StatusOK, authRec.Code, "body: %s", authRec.Body.String())
		var authResp oauth2provider.DeviceAuthResponse
		require.NoError(t, json.Unmarshal(authRec.Body.Bytes(), &authResp))

		dc, err := st.GetDeviceCodeByDeviceCode(context.Background(), authResp.DeviceCode)
		require.NoError(t, err)
		dc.Status = oauth2provider.DeviceCodeStatusAuthorized
		dc.UserID = id.NewUserID()
		require.NoError(t, st.UpdateDeviceCode(context.Background(), dc))

		rec := postTokenJSON(t, mux, map[string]any{
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
			"device_code": authResp.DeviceCode,
			"client_id":   confidentialID,
			"resource":    []string{resFiles},
		})

		assert.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), "invalid_target")
		assert.Contains(t, rec.Body.String(), "was not granted by this authorization",
			"must fail via narrowResources' rejection, not resolveResources'")
		assert.NotContains(t, rec.Body.String(), "access_token")
	})

	// Case 6: a JSON token request carrying "resource": [...] must work. The
	// form-encoded case is served by resourceParams reading the raw request;
	// this body is JSON with no query string, so resourceParams finds
	// nothing and the only way narrowResources ever sees the value is through
	// TokenRequest.Resource's json tag decoding it.
	t.Run("a JSON resource array binds through encoding/json, not resourceParams", func(t *testing.T) {
		st, acct, mux := tokenFixture(t)
		grantResources(t, st, resAPI, resFiles)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)
		q.Add("resource", resFiles)
		code := codeFrom(t, authorize(t, mux, q))

		rec := postTokenJSON(t, mux, map[string]any{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     confidentialID,
			"client_secret": confidentialSecret,
			"redirect_uri":  registeredURI,
			"resource":      []string{resFiles},
		})

		resp := decodeTokenResponse(t, rec)
		assert.Equal(t, []string{resFiles}, audienceFor(t, acct, resp.AccessToken))
	})
}
