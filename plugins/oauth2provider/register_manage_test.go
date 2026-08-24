package oauth2provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// registerOne registers a client and returns its id and registration token.
func registerOne(t *testing.T, router http.Handler) (clientID, regToken string) {
	t.Helper()
	rec := postRegister(t, router, `{
		"client_name": "Managed",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"token_endpoint_auth_method": "none",
		"scope": "openid profile"
	}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got["client_id"].(string), got["registration_access_token"].(string)
}

func manageReq(t *testing.T, router http.Handler, method, clientID, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, "/v1/oauth/register/"+clientID, nil)
	} else {
		r = httptest.NewRequest(method, "/v1/oauth/register/"+clientID, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, r)
	return rec
}

func TestManage_ReadReturnsRegistration(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodGet, clientID, token, "")
	require.Equal(t, http.StatusOK, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, clientID, got["client_id"])
	assert.Equal(t, "Managed", got["client_name"])
	// The token is never re-issued on a read.
	assert.Empty(t, got["registration_access_token"])
}

func TestManage_WrongTokenIs401(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, _ := registerOne(t, router)

	rec := manageReq(t, router, http.MethodGet, clientID, "not-the-token", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "Bearer")
}

func TestManage_MissingTokenIs401(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, _ := registerOne(t, router)
	rec := manageReq(t, router, http.MethodGet, clientID, "", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// A token is scoped to the client it was issued for. Presenting client A's
// token against client B must fail even though the token itself is valid.
func TestManage_TokenFromAnotherClientIs401(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	_, tokenA := registerOne(t, router)
	clientB, _ := registerOne(t, router)

	rec := manageReq(t, router, http.MethodGet, clientB, tokenA, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// Admin-created clients have no registration token hash, so they are not
// reachable over 7592 even if the client_id is known.
func TestManage_AdminCreatedClientIsUnreachable(t *testing.T) {
	_, st, router, appID := newRegistrationFixture(t, true)
	require.NoError(t, st.CreateClient(t.Context(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     "admin-made",
		Name:         "Admin",
		RedirectURIs: []string{"https://app.example.com/cb"},
		GrantTypes:   []string{"authorization_code"},
	}))

	rec := manageReq(t, router, http.MethodGet, "admin-made", "anything", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestManage_UpdateChangesRedirectURIs(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"client_name": "Renamed",
		"redirect_uris": ["http://127.0.0.1:9500/cb"],
		"token_endpoint_auth_method": "none"
	}`)
	require.Equal(t, http.StatusOK, rec.Code)

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", stored.Name)
	assert.Equal(t, []string{"http://127.0.0.1:9500/cb"}, stored.RedirectURIs)
}

// The whole point of running updates through the same pipeline: an update
// must not be able to buy a capability registration refused.
func TestManage_UpdateCannotWidenGrants(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"grant_types": ["authorization_code", "client_credentials"]
	}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.NotContains(t, stored.GrantTypes, "client_credentials")
}

func TestManage_UpdateCannotWidenScopes(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"scope": "openid admin:all"
	}`)
	require.Equal(t, http.StatusOK, rec.Code)

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.NotContains(t, stored.Scopes, "admin:all")
}

func TestManage_UpdateRejectsMismatchedClientID(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "some-other-id",
		"redirect_uris": ["http://127.0.0.1:9000/cb"]
	}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])
}

func TestManage_Delete(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodDelete, clientID, token, "")
	require.Equal(t, http.StatusNoContent, rec.Code)

	_, err := st.GetClient(t.Context(), clientID)
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)
}

// Turning registration off closes the door to new clients. It must not
// strand the ones that came in while it was open: an operator still needs
// DELETE, and a client still needs to see that it was revoked.
func TestManage_StillWorksWhenRegistrationDisabled(t *testing.T) {
	p, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	p.SetDynamicRegistrationForTest(false)

	rec := manageReq(t, router, http.MethodGet, clientID, token, "")
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = manageReq(t, router, http.MethodDelete, clientID, token, "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	_, err := st.GetClient(t.Context(), clientID)
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)
}
