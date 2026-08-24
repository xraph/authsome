package oauth2provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

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
// Admin-created clients have DynamicallyRegistered == false AND an empty
// RegistrationTokenHash. authenticateRegistration checks both, and the two
// cases below pull them apart so each half of that check is actually
// exercised: a single case setting both at once would still pass with
// either condition alone removed.

// A dynamically registered client with no token hash yet (the state a
// client would be in only via a direct store write, since the real
// registration path always sets one) must not authenticate no matter what
// token is presented.
func TestManage_DynamicClientWithEmptyHashIsUnreachable(t *testing.T) {
	_, st, router, appID := newRegistrationFixture(t, true)
	require.NoError(t, st.CreateClient(t.Context(), &oauth2provider.OAuth2Client{
		ID:                    id.NewOAuth2ClientID(),
		AppID:                 appID,
		ClientID:              "dynamic-no-hash",
		Name:                  "Dynamic without a hash",
		RedirectURIs:          []string{"https://app.example.com/cb"},
		GrantTypes:            []string{"authorization_code"},
		DynamicallyRegistered: true,
		RegistrationTokenHash: "",
	}))

	rec := manageReq(t, router, http.MethodGet, "dynamic-no-hash", "anything", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// An admin-created client that happens to carry a valid registration token
// hash (never possible through this codebase's own admin path today, but
// nothing stops a future admin surface or a migration from setting one)
// must still be unreachable, because DynamicallyRegistered is false. The
// token presented here is the one the hash actually matches, so this
// proves the DynamicallyRegistered gate independently of the hash check.
func TestManage_HasHashButNotDynamicallyRegisteredIsUnreachable(t *testing.T) {
	_, st, router, appID := newRegistrationFixture(t, true)
	const token = "a-token-that-hashes-correctly"
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, st.CreateClient(t.Context(), &oauth2provider.OAuth2Client{
		ID:                    id.NewOAuth2ClientID(),
		AppID:                 appID,
		ClientID:              "admin-made",
		Name:                  "Admin",
		RedirectURIs:          []string{"https://app.example.com/cb"},
		GrantTypes:            []string{"authorization_code"},
		DynamicallyRegistered: false,
		RegistrationTokenHash: string(hash),
	}))

	rec := manageReq(t, router, http.MethodGet, "admin-made", token, "")
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

// UpdateRegistrationRequest.ClientID is path-bound with no json tag
// (json:"-"), specifically so a body field cannot win a case-insensitive
// match against it and overwrite the path value once BindRequest decodes
// the body on top. This proves the attack the tag closes: a caller who
// does not hold the victim's token, hitting an arbitrary path, cannot
// smuggle the victim's client_id into the body and have it authenticate or
// mutate as that client.
func TestManage_UpdateIgnoresClientIDSmuggledInBody(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	victimID, _ := registerOne(t, router)

	body, err := json.Marshal(map[string]any{
		"client_id":     victimID,
		"redirect_uris": []string{"http://127.0.0.1:9999/cb"},
	})
	require.NoError(t, err)

	rec := manageReq(t, router, http.MethodPut, "some-junk-path-segment", "not-the-victims-token", string(body))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	stored, err := st.GetClient(t.Context(), victimID)
	require.NoError(t, err)
	assert.Equal(t, []string{"http://127.0.0.1:9000/cb"}, stored.RedirectURIs,
		"the victim's record must be untouched by a request addressed to a different path")
}

// PUT runs the same registration in place, so it is the same full-record
// replace on every backend as POST is. It must be capped the same way.
func TestManage_UpdateRejectsOversizedClientName(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	body, err := json.Marshal(map[string]any{
		"client_id":     clientID,
		"client_name":   strings.Repeat("a", 257),
		"redirect_uris": []string{"http://127.0.0.1:9000/cb"},
	})
	require.NoError(t, err)

	rec := manageReq(t, router, http.MethodPut, clientID, token, string(body))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.Equal(t, "Managed", stored.Name, "the rejected update must not have written anything")
}

func TestManage_UpdateRejectsTooManyRedirectURIs(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	uris := make([]string, 21)
	for i := range uris {
		uris[i] = fmt.Sprintf("https://app%d.example.com/cb", i)
	}
	body, err := json.Marshal(map[string]any{
		"client_id":     clientID,
		"redirect_uris": uris,
	})
	require.NoError(t, err)

	rec := manageReq(t, router, http.MethodPut, clientID, token, string(body))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.Equal(t, []string{"http://127.0.0.1:9000/cb"}, stored.RedirectURIs,
		"the rejected update must not have written anything")
}

func TestManage_UpdateRejectsTooManyContacts(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	contacts := make([]string, 11)
	for i := range contacts {
		contacts[i] = fmt.Sprintf("ops%d@example.com", i)
	}
	body, err := json.Marshal(map[string]any{
		"client_id":     clientID,
		"redirect_uris": []string{"http://127.0.0.1:9000/cb"},
		"contacts":      contacts,
	})
	require.NoError(t, err)

	rec := manageReq(t, router, http.MethodPut, clientID, token, string(body))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])
}

// Omitting scope resets to the full allowlist, not a merge with whatever
// the client held before. Pinned deliberately so this reads as an
// intentional replacement-semantics choice, not a bug to "fix" later.
func TestManage_UpdateOmittingScopeResetsToAllowlist(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"token_endpoint_auth_method": "none",
		"scope": "openid"
	}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var reg map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &reg))
	clientID := reg["client_id"].(string)
	token := reg["registration_access_token"].(string)
	require.Equal(t, "openid", reg["scope"])

	rec = manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"token_endpoint_auth_method": "none"
	}`)
	require.Equal(t, http.StatusOK, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "openid profile email offline_access", got["scope"],
		"an omitted scope resets to the full allowlist, per RFC 7592 replacement semantics")
}

// A public client has no client_secret hash, and update never mints one.
// Letting it switch to a confidential auth method would leave it unable to
// authenticate forever, with no signal at the time it happened.
func TestManage_UpdateRejectsPublicToConfidentialTransition(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router) // registered with token_endpoint_auth_method: none

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"token_endpoint_auth_method": "client_secret_basic"
	}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.Equal(t, "none", stored.TokenEndpointAuthMethod)
	assert.True(t, stored.Public)
}

// Every other 401 case in this file goes through GET. This one, and the
// DELETE case below it, prove PUT and DELETE independently check the
// token rather than inheriting coverage from GET by accident.
func TestManage_UpdateWrongTokenIs401AndDoesNotMutate(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, _ := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, "not-the-token", `{
		"client_id": "`+clientID+`",
		"client_name": "Hijacked",
		"redirect_uris": ["http://127.0.0.1:6666/cb"]
	}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.Equal(t, "Managed", stored.Name)
	assert.Equal(t, []string{"http://127.0.0.1:9000/cb"}, stored.RedirectURIs)
}

func TestManage_DeleteWrongTokenIs401AndClientSurvives(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, _ := registerOne(t, router)

	rec := manageReq(t, router, http.MethodDelete, clientID, "not-the-token", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	_, err := st.GetClient(t.Context(), clientID)
	assert.NoError(t, err, "a rejected delete must not have removed the client")
}

// RFC 7592 permits rotating the registration access token on update; this
// server deliberately does not, because rotation strands any client that
// fails to persist the new value. The original token presented at
// registration must keep working after a PUT, and the PUT response itself
// must not hand back credentials a plain update never touches.
func TestManage_UpdateDoesNotRotateTokenOrLeakCredentials(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	putRec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"client_name": "Renamed",
		"redirect_uris": ["http://127.0.0.1:9500/cb"],
		"token_endpoint_auth_method": "none"
	}`)
	require.Equal(t, http.StatusOK, putRec.Code)

	var putResp map[string]any
	require.NoError(t, json.Unmarshal(putRec.Body.Bytes(), &putResp))
	assert.Empty(t, putResp["registration_access_token"])
	assert.Empty(t, putResp["client_secret"])

	getRec := manageReq(t, router, http.MethodGet, clientID, token, "")
	assert.Equal(t, http.StatusOK, getRec.Code, "the original registration token must still work after a PUT")
}

// UpdateClient is a full-record replace on every backend. A handler that
// fetches the stored client and mutates it in place (as this one does)
// keeps everything it does not explicitly touch; a handler that
// constructed a fresh OAuth2Client from the request would silently wipe
// these fields instead.
func TestManage_UpdatePreservesFieldsItDoesNotTouch(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	before, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"client_name": "Renamed",
		"redirect_uris": ["http://127.0.0.1:9500/cb"],
		"token_endpoint_auth_method": "none"
	}`)
	require.Equal(t, http.StatusOK, rec.Code)

	after, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.Equal(t, before.RegistrationTokenHash, after.RegistrationTokenHash)
	assert.Equal(t, before.ClientSecret, after.ClientSecret)
	assert.Equal(t, before.CreatedAt, after.CreatedAt)
	assert.Equal(t, before.AppID, after.AppID)
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

	rec = manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"client_name": "Still Managed",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"token_endpoint_auth_method": "none"
	}`)
	require.Equal(t, http.StatusOK, rec.Code)
	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.Equal(t, "Still Managed", stored.Name)

	rec = manageReq(t, router, http.MethodDelete, clientID, token, "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	_, err = st.GetClient(t.Context(), clientID)
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)
}
