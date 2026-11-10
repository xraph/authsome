package oauth2provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

const byeURI = "https://app.example.com/bye"

// logoutFixture is the OIDC fixture with a post-logout URI on the public
// client and a real session for its user.
type logoutFixture struct {
	*oidcFixture
	sess *session.Session
}

func newLogoutFixture(t *testing.T, cfg oauth2provider.Config) *logoutFixture {
	t.Helper()
	f := newOIDCFixture(t, cfg, true)
	ctx := context.Background()
	c, err := f.st.GetClient(ctx, publicID)
	require.NoError(t, err)
	c.PostLogoutRedirectURIs = []string{byeURI}
	require.NoError(t, f.st.UpdateClient(ctx, c))
	return &logoutFixture{oidcFixture: f, sess: f.newSession(t, f.user)}
}

func (f *oidcFixture) newSession(t *testing.T, u *user.User) *session.Session {
	t.Helper()
	res, err := f.eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{User: u, AppID: f.appID, AuthMethod: "password"})
	require.NoError(t, err)
	return res.Session
}

// signedIn is a request context carrying sess.
func signedIn(sess *session.Session) context.Context {
	ctx := middleware.WithSession(context.Background(), sess)
	ctx = middleware.WithSessionID(ctx, sess.ID)
	ctx = middleware.WithAppID(ctx, sess.AppID)
	return middleware.WithUserID(ctx, sess.UserID)
}

func (f *logoutFixture) idToken(t *testing.T) string {
	t.Helper()
	body := decodeTokenBody(t, f.codeGrant(t, publicID, "openid", "", time.Now()))
	tok, _ := body["id_token"].(string)
	require.NotEmpty(t, tok)
	return tok
}

func (f *logoutFixture) logout(ctx context.Context, q url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/oauth/logout?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

func (f *logoutFixture) sessionAlive(t *testing.T) bool {
	t.Helper()
	_, err := f.eng.Store().GetSession(context.Background(), f.sess.ID)
	return err == nil
}

var hiddenField = regexp.MustCompile(`name="(lid|csrf)" value="([^"]+)"`)

// confirmForm reads the lid and csrf off a confirmation page.
func confirmForm(t *testing.T, rec *httptest.ResponseRecorder) (lid, csrf string) {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	for _, m := range hiddenField.FindAllStringSubmatch(rec.Body.String(), -1) {
		if m[1] == "lid" {
			lid = m[2]
		} else {
			csrf = m[2]
		}
	}
	require.NotEmpty(t, lid, rec.Body.String())
	require.NotEmpty(t, csrf)
	return lid, csrf
}

func (f *logoutFixture) confirm(ctx context.Context, lid, csrf, decision string) *httptest.ResponseRecorder {
	form := url.Values{"lid": {lid}, "csrf": {csrf}, "decision": {decision}}
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/oauth/logout/confirm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

func TestLogout_HintForTheSignedInUserEndsTheSession(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	rec := f.logout(signedIn(f.sess), url.Values{
		"id_token_hint": {f.idToken(t)}, "post_logout_redirect_uri": {byeURI}, "state": {"s-9"},
	})
	require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
	assert.Equal(t, byeURI+"?state=s-9", rec.Header().Get("Location"))
	assert.False(t, f.sessionAlive(t), "the session is gone")

	var cleared bool
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge < 0 {
			cleared = true
		}
	}
	assert.True(t, cleared, "the session cookie is cleared")
}

func TestLogout_ExpiredHintStillCounts(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{IDTokenTTL: time.Second})
	hint := f.idToken(t)
	time.Sleep(1100 * time.Millisecond)
	rec := f.logout(signedIn(f.sess), url.Values{"id_token_hint": {hint}})
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "signed out")
	assert.False(t, f.sessionAlive(t))
}

func TestLogout_WithoutAHintAsksFirst(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	q := url.Values{"client_id": {publicID}, "post_logout_redirect_uri": {byeURI}}
	ctx := signedIn(f.sess)

	page := f.logout(ctx, q)
	assert.Contains(t, page.Body.String(), "Public is asking to sign you out")
	assert.True(t, f.sessionAlive(t), "nothing ends before the user says so")
	lid, csrf := confirmForm(t, page)

	assert.Equal(t, http.StatusForbidden, f.confirm(ctx, lid, "wrong", "logout").Code)
	assert.True(t, f.sessionAlive(t))

	rec := f.confirm(ctx, lid, csrf, "logout")
	require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
	assert.Equal(t, byeURI, rec.Header().Get("Location"))
	assert.False(t, f.sessionAlive(t))

	assert.Equal(t, http.StatusBadRequest, f.confirm(ctx, lid, csrf, "logout").Code, "single use")
}

func TestLogout_StayingSignedIn(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	ctx := signedIn(f.sess)
	lid, csrf := confirmForm(t, f.logout(ctx, url.Values{}))
	rec := f.confirm(ctx, lid, csrf, "stay")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "still signed in")
	assert.True(t, f.sessionAlive(t))
}

func TestLogout_ConfirmFromAnotherSessionEndsNothing(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	lid, csrf := confirmForm(t, f.logout(signedIn(f.sess), url.Values{}))
	other := f.newSession(t, f.user)
	rec := f.confirm(signedIn(other), lid, csrf, "logout")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, f.sessionAlive(t), "the session the page was shown for is untouched")
	_, err := f.eng.Store().GetSession(context.Background(), other.ID)
	assert.NoError(t, err, "and so is the one that posted")
}

func TestLogout_HintForSomeoneElseAsksFirst(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	hint := f.idToken(t)
	now := time.Now()
	other := &user.User{ID: id.NewUserID(), AppID: f.appID, Email: "other@example.com", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, f.eng.Store().CreateUser(context.Background(), other))
	otherSess := f.newSession(t, other)

	confirmForm(t, f.logout(signedIn(otherSess), url.Values{"id_token_hint": {hint}}))
	_, err := f.eng.Store().GetSession(context.Background(), otherSess.ID)
	assert.NoError(t, err)
}

func TestLogout_SignedOut(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	rec := f.logout(context.Background(), url.Values{"client_id": {publicID}, "post_logout_redirect_uri": {byeURI}, "state": {"x"}})
	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, byeURI+"?state=x", rec.Header().Get("Location"))

	rec = f.logout(context.Background(), url.Values{})
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "signed out")
}

func TestLogout_PostForm(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	form := url.Values{"id_token_hint": {f.idToken(t)}, "post_logout_redirect_uri": {byeURI}}
	req := httptest.NewRequestWithContext(signedIn(f.sess), http.MethodPost, "/v1/oauth/logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
	assert.False(t, f.sessionAlive(t))
}

func TestLogout_Refusals(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	hint := f.idToken(t)
	parts := strings.Split(hint, ".")
	tampered := parts[0] + "." + parts[1] + "." + strings.Repeat("A", len(parts[2]))

	for name, q := range map[string]url.Values{
		"unregistered uri":         {"client_id": {publicID}, "post_logout_redirect_uri": {"https://evil.example.com/"}},
		"uri with no client":       {"post_logout_redirect_uri": {byeURI}},
		"unknown client":           {"client_id": {"nobody"}},
		"client and hint disagree": {"client_id": {noOpenIDID}, "id_token_hint": {hint}},
		"tampered hint":            {"id_token_hint": {tampered}},
		"garbage hint":             {"id_token_hint": {"not-a-jwt"}},
		"oversized hint":           {"id_token_hint": {strings.Repeat("a", 17<<10)}},
	} {
		t.Run(name, func(t *testing.T) {
			rec := f.logout(signedIn(f.sess), q)
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.True(t, f.sessionAlive(t))
		})
	}
}

func TestDiscovery_AdvertisesEndSession(t *testing.T) {
	f := newLogoutFixture(t, oauth2provider.Config{})
	rec := getPath(t, f.mux, "/.well-known/openid-configuration")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"end_session_endpoint":"`+oidcIssuer+`/v1/oauth/logout"`)
}

func TestPostLogoutRedirectURIs_AreValidated(t *testing.T) {
	p, _, _ := newFixture(t)
	ctx := context.Background()
	appID := id.NewAppID()

	_, err := p.CreateClientFor(ctx, appID, oauth2provider.CreateClientRequest{
		Name: "Bad", Public: true, PostLogoutRedirectURIs: []string{"javascript:alert(1)"},
	})
	assert.Error(t, err)

	created, err := p.CreateClientFor(ctx, appID, oauth2provider.CreateClientRequest{
		Name: "Good", Public: true, PostLogoutRedirectURIs: []string{byeURI},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{byeURI}, created.PostLogoutRedirectURIs)

	bad := []string{"https://app.example.com/bye#frag"}
	_, err = p.UpdateClientFor(ctx, appID, created.ID, oauth2provider.UpdateClientRequest{PostLogoutRedirectURIs: &bad})
	assert.Error(t, err)

	cleared := []string{}
	c, err := p.UpdateClientFor(ctx, appID, created.ID, oauth2provider.UpdateClientRequest{PostLogoutRedirectURIs: &cleared})
	require.NoError(t, err)
	assert.Empty(t, c.PostLogoutRedirectURIs)

	tooMany := make([]string, 21)
	for i := range tooMany {
		tooMany[i] = byeURI
	}
	_, err = p.UpdateClientFor(ctx, appID, created.ID, oauth2provider.UpdateClientRequest{PostLogoutRedirectURIs: &tooMany})
	assert.Error(t, err)
	long := []string{"https://app.example.com/" + strings.Repeat("a", 3000)}
	_, err = p.UpdateClientFor(ctx, appID, created.ID, oauth2provider.UpdateClientRequest{PostLogoutRedirectURIs: &long})
	assert.Error(t, err)
}

func TestRegister_PostLogoutRedirectURIs(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{"client_name":"RP","redirect_uris":["https://rp.example.com/cb"],
		"post_logout_redirect_uris":["https://rp.example.com/bye"],"token_endpoint_auth_method":"none"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"post_logout_redirect_uris":["https://rp.example.com/bye"]`)

	rec = postRegister(t, router, `{"redirect_uris":["https://rp.example.com/cb"],
		"post_logout_redirect_uris":["http://rp.example.com/bye"],"token_endpoint_auth_method":"none"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "plain http off loopback is refused here too")
}
