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
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store/memory"
)

// A third-party client gets a code only after the signed-in user approved
// its scopes on the consent page; the approval is remembered per client and
// can be widened, forced with prompt=consent, listed and revoked.

const thirdPartyID = "third-party"

func newConsentFixture(t *testing.T) (oauth2provider.Store, forge.Router, id.AppID) {
	t.Helper()
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	p.SetStore(memory.New())
	appID := id.NewAppID()
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: thirdPartyID, Name: "Acme Reports",
		RedirectURIs: []string{registeredURI}, Scopes: []string{"openid", "profile", "email"},
		GrantTypes: []string{"authorization_code"}, Public: true, TokenEndpointAuthMethod: "none",
	}))
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: "own-app", Name: "Own",
		RedirectURIs: []string{registeredURI}, Scopes: []string{"openid"}, GrantTypes: []string{"authorization_code"},
		Public: true, TokenEndpointAuthMethod: "none", FirstParty: true,
	}))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return st, mux, appID
}

func authorizeAs(t *testing.T, mux forge.Router, userID id.UserID, q url.Values) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(context.Background(), userID)
	req := httptest.NewRequestWithContext(ctx, "GET", "/v1/oauth/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func consentQuery(scope string) url.Values {
	q := baseAuthorizeQuery(thirdPartyID)
	q.Set("scope", scope)
	q.Set("state", "st-1")
	q.Set("code_challenge", s256("verifier-verifier-verifier-verifier-1234"))
	q.Set("code_challenge_method", "S256")
	return q
}

// consentIDFrom asserts the authorize response is a redirect to the consent
// page and returns its cid.
func consentIDFrom(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "/v1/oauth/consent", loc.Path, "a third-party request lands on the consent page, not on the client: %s", loc)
	require.NotEmpty(t, loc.Query().Get("cid"))
	return loc.Query().Get("cid")
}

func getConsent(t *testing.T, mux forge.Router, userID id.UserID, cid string, accept string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(context.Background(), userID)
	req := httptest.NewRequestWithContext(ctx, "GET", "/v1/oauth/consent?cid="+cid, nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decide(t *testing.T, mux forge.Router, userID id.UserID, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(context.Background(), userID)
	req := httptest.NewRequestWithContext(ctx, "POST", "/v1/oauth/consent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func csrfFromPage(t *testing.T, body string) string {
	t.Helper()
	const marker = `name="csrf" value="`
	i := strings.Index(body, marker)
	require.GreaterOrEqual(t, i, 0, "the page carries the csrf token")
	rest := body[i+len(marker):]
	return rest[:strings.Index(rest, `"`)]
}

func TestConsent_ThirdPartyClientNeedsApproval(t *testing.T) {
	st, mux, appID := newConsentFixture(t)
	user := id.NewUserID()

	cid := consentIDFrom(t, authorizeAs(t, mux, user, consentQuery("openid profile")))

	page := getConsent(t, mux, user, cid, "")
	require.Equal(t, http.StatusOK, page.Code, page.Body.String())
	assert.Contains(t, page.Header().Get("Content-Type"), "text/html")
	assert.Equal(t, "no-store", page.Header().Get("Cache-Control"))
	assert.Equal(t, "DENY", page.Header().Get("X-Frame-Options"))
	body := page.Body.String()
	assert.Contains(t, body, "Acme Reports")
	assert.Contains(t, body, "<code>openid</code>")
	assert.Contains(t, body, "<code>profile</code>")
	csrf := csrfFromPage(t, body)

	// A wrong token is refused and the request stays pending.
	bad := decide(t, mux, user, url.Values{"cid": {cid}, "csrf": {"nope"}, "decision": {"approve"}})
	assert.Equal(t, http.StatusForbidden, bad.Code)
	_, err := st.GetGrant(context.Background(), appID, user, thirdPartyID)
	assert.ErrorIs(t, err, oauth2provider.ErrGrantNotFound, "a refused decision records nothing")

	approved := decide(t, mux, user, url.Values{"cid": {cid}, "csrf": {csrf}, "decision": {"approve"}})
	require.Equal(t, http.StatusFound, approved.Code, approved.Body.String())
	loc, err := url.Parse(approved.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, registeredURI, loc.Scheme+"://"+loc.Host+loc.Path)
	assert.NotEmpty(t, loc.Query().Get("code"), "approval issues the code")
	assert.Equal(t, "st-1", loc.Query().Get("state"))
	g, err := st.GetGrant(context.Background(), appID, user, thirdPartyID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"openid", "profile"}, g.Scopes)

	// The consent id is single use.
	again := decide(t, mux, user, url.Values{"cid": {cid}, "csrf": {csrf}, "decision": {"approve"}})
	assert.Equal(t, http.StatusBadRequest, again.Code)

	// A covered request now issues a code straight away.
	direct := authorizeAs(t, mux, user, consentQuery("openid"))
	assert.NotEmpty(t, codeFrom(t, direct))

	// Wider scopes, or prompt=consent, bring the page back.
	consentIDFrom(t, authorizeAs(t, mux, user, consentQuery("openid profile email")))
	forced := consentQuery("openid")
	forced.Set("prompt", "consent")
	consentIDFrom(t, authorizeAs(t, mux, user, forced))

	// Another user's approval is their own.
	consentIDFrom(t, authorizeAs(t, mux, id.NewUserID(), consentQuery("openid")))
}

func TestConsent_DenySendsAccessDenied(t *testing.T) {
	st, mux, appID := newConsentFixture(t)
	user := id.NewUserID()
	cid := consentIDFrom(t, authorizeAs(t, mux, user, consentQuery("openid")))
	csrf := csrfFromPage(t, getConsent(t, mux, user, cid, "").Body.String())

	denied := decide(t, mux, user, url.Values{"cid": {cid}, "csrf": {csrf}, "decision": {"deny"}})
	require.Equal(t, http.StatusFound, denied.Code)
	loc, err := url.Parse(denied.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "access_denied", loc.Query().Get("error"))
	assert.Equal(t, "st-1", loc.Query().Get("state"))
	assert.Empty(t, loc.Query().Get("code"))
	_, err = st.GetGrant(context.Background(), appID, user, thirdPartyID)
	assert.ErrorIs(t, err, oauth2provider.ErrGrantNotFound)
}

func TestConsent_BelongsToTheUserWhoStartedIt(t *testing.T) {
	_, mux, _ := newConsentFixture(t)
	owner, other := id.NewUserID(), id.NewUserID()
	cid := consentIDFrom(t, authorizeAs(t, mux, owner, consentQuery("openid")))
	assert.Equal(t, http.StatusForbidden, getConsent(t, mux, other, cid, "").Code)
	assert.Equal(t, http.StatusForbidden, decide(t, mux, other, url.Values{"cid": {cid}, "csrf": {"x"}, "decision": {"approve"}}).Code)
	assert.Equal(t, http.StatusOK, getConsent(t, mux, owner, cid, "").Code, "the owner still can")
}

func TestConsent_FirstPartyClientSkipsThePage(t *testing.T) {
	_, mux, _ := newConsentFixture(t)
	q := baseAuthorizeQuery("own-app")
	q.Set("scope", "openid")
	q.Set("code_challenge", s256("verifier-verifier-verifier-verifier-1234"))
	q.Set("code_challenge_method", "S256")
	assert.NotEmpty(t, codeFrom(t, authorizeAs(t, mux, id.NewUserID(), q)))
}

func TestConsent_JSONFrontEnd(t *testing.T) {
	st, mux, appID := newConsentFixture(t)
	user := id.NewUserID()
	cid := consentIDFrom(t, authorizeAs(t, mux, user, consentQuery("openid email")))

	shown := getConsent(t, mux, user, cid, "application/json")
	require.Equal(t, http.StatusOK, shown.Code, shown.Body.String())
	var view oauth2provider.ConsentView
	require.NoError(t, json.Unmarshal(shown.Body.Bytes(), &view))
	assert.Equal(t, "Acme Reports", view.ClientName)
	assert.Equal(t, []string{"openid", "email"}, view.Scopes)
	require.NotEmpty(t, view.CSRF)

	body, _ := json.Marshal(oauth2provider.ConsentDecision{CID: cid, CSRF: view.CSRF, Decision: "approve"})
	ctx := middleware.WithUserID(context.Background(), user)
	req := httptest.NewRequestWithContext(ctx, "POST", "/v1/oauth/consent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result oauth2provider.ConsentResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Contains(t, result.RedirectURL, "code=")
	g, err := st.GetGrant(context.Background(), appID, user, thirdPartyID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"openid", "email"}, g.Scopes)
}

func TestConsent_GrantsAreListedAndRevocable(t *testing.T) {
	st, mux, appID := newConsentFixture(t)
	user := id.NewUserID()
	require.NoError(t, st.UpsertGrant(context.Background(), &oauth2provider.Grant{AppID: appID, UserID: user, ClientID: thirdPartyID, Scopes: []string{"openid"}}))

	ctx := middleware.WithAppID(middleware.WithUserID(context.Background(), user), appID)
	list := httptest.NewRequestWithContext(ctx, "GET", "/v1/me/oauth/grants", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, list)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var out oauth2provider.ListGrantsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out.Grants, 1)
	assert.Equal(t, "Acme Reports", out.Grants[0].ClientName)
	assert.Equal(t, []string{"openid"}, out.Grants[0].Scopes)

	del := httptest.NewRequestWithContext(ctx, "DELETE", "/v1/me/oauth/grants/"+thirdPartyID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, del)
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	_, err := st.GetGrant(context.Background(), appID, user, thirdPartyID)
	assert.ErrorIs(t, err, oauth2provider.ErrGrantNotFound)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, del)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// With the grant gone, the next request shows the page again.
	consentIDFrom(t, authorizeAs(t, mux, user, consentQuery("openid")))
}
