package oauth2provider_test

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// reauthFixture has a login page, so a request that needs a fresh sign-in
// has somewhere to go.
func reauthFixture(t *testing.T, cfg oauth2provider.Config) (forge.Router, id.AppID) {
	t.Helper()
	if cfg.Issuer == "" {
		cfg.Issuer = "https://auth.example.com"
	}
	_, st, mux := newFixtureWithConfig(t, cfg)
	c, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	return mux, c.AppID
}

func reauthQuery() url.Values {
	q := baseAuthorizeQuery(confidentialID)
	q.Set("scope", "openid")
	q.Set("state", "st-1")
	return q
}

// signedInFor authorizes as the fixture user with a session created ago.
func signedInFor(mux http.Handler, appID id.AppID, ago time.Duration, q url.Values) (loc *url.URL, code int) {
	r := authorizeWith(sessionCtx(fixtureUserID, appID, time.Now().Add(-ago)), mux, q)
	l, _ := url.Parse(r.Header().Get("Location"))
	return l, r.Code
}

func TestAuthorize_MaxAge(t *testing.T) {
	mux, appID := reauthFixture(t, oauth2provider.Config{LoginURL: "/sign-in"})

	t.Run("a recent enough sign-in gets a code", func(t *testing.T) {
		q := reauthQuery()
		q.Set("max_age", "600")
		loc, code := signedInFor(mux, appID, time.Minute, q)
		require.Equal(t, http.StatusFound, code)
		assert.NotEmpty(t, loc.Query().Get("code"))
	})

	t.Run("an older sign-in goes back to the login page", func(t *testing.T) {
		q := reauthQuery()
		q.Set("max_age", "60")
		loc, code := signedInFor(mux, appID, time.Hour, q)
		require.Equal(t, http.StatusFound, code)
		assert.Equal(t, "/sign-in", loc.Path)
		assert.Equal(t, "/v1/oauth/authorize?"+q.Encode(), loc.Query().Get("redirect"),
			"max_age is met by the new session, so the request comes back unchanged")
	})

	t.Run("max_age=0 always asks", func(t *testing.T) {
		q := reauthQuery()
		q.Set("max_age", "0")
		loc, _ := signedInFor(mux, appID, 2*time.Second, q)
		assert.Equal(t, "/sign-in", loc.Path)
	})

	t.Run("with prompt=none the client hears login_required", func(t *testing.T) {
		q := reauthQuery()
		q.Set("max_age", "60")
		q.Set("prompt", "none")
		loc, _ := signedInFor(mux, appID, time.Hour, q)
		assert.Equal(t, "app.example.com", loc.Host)
		assert.Equal(t, "login_required", loc.Query().Get("error"))
		assert.Equal(t, "st-1", loc.Query().Get("state"))
	})

	for _, bad := range []string{"-1", "soon", "1.5"} {
		t.Run("rejects max_age "+bad, func(t *testing.T) {
			q := reauthQuery()
			q.Set("max_age", bad)
			loc, _ := signedInFor(mux, appID, time.Minute, q)
			assert.Equal(t, "invalid_request", loc.Query().Get("error"))
			assert.Equal(t, "st-1", loc.Query().Get("state"))
		})
	}
}

func TestAuthorize_PromptLogin(t *testing.T) {
	mux, appID := reauthFixture(t, oauth2provider.Config{LoginURL: "/sign-in"})

	q := reauthQuery()
	q.Set("prompt", "login consent")
	before := time.Now().Unix()
	loc, code := signedInFor(mux, appID, time.Minute, q)
	require.Equal(t, http.StatusFound, code)
	require.Equal(t, "/sign-in", loc.Path)

	back, err := url.Parse(loc.Query().Get("redirect"))
	require.NoError(t, err)
	bq := back.Query()
	assert.Equal(t, "consent", bq.Get("prompt"), "login is dropped, the rest of the prompt stays")
	after, err := strconv.ParseInt(bq.Get("login_after"), 10, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, after, before)

	// Coming back with the session that was already there is not a new
	// sign-in, so it goes round again.
	bq.Set("prompt", "")
	loc, _ = signedInFor(mux, appID, time.Minute, bq)
	assert.Equal(t, "/sign-in", loc.Path)

	// A session made after the trip gets its code. No loop.
	r := authorizeWith(sessionCtx(fixtureUserID, appID, time.Unix(after, 0).Add(time.Second)), mux, bq)
	require.Equal(t, http.StatusFound, r.Code, r.Body.String())
	loc, err = url.Parse(r.Header().Get("Location"))
	require.NoError(t, err)
	assert.NotEmpty(t, loc.Query().Get("code"))
}

func TestAuthorize_PromptLoginSignedOutComesBackOnce(t *testing.T) {
	mux, appID := reauthFixture(t, oauth2provider.Config{LoginURL: "/sign-in"})
	q := reauthQuery()
	q.Set("prompt", "login")

	loc := location(t, authorizeSignedOut(t, mux, q))
	back, err := url.Parse(loc.Query().Get("redirect"))
	require.NoError(t, err)
	assert.Empty(t, back.Query().Get("prompt"))
	require.NotEmpty(t, back.Query().Get("login_after"))

	loc2, code := signedInFor(mux, appID, -time.Second, back.Query())
	require.Equal(t, http.StatusFound, code)
	assert.NotEmpty(t, loc2.Query().Get("code"), "the sign-in that just happened satisfies prompt=login")
}

func TestAuthorize_PromptLoginWithoutALoginPage(t *testing.T) {
	mux, appID := reauthFixture(t, oauth2provider.Config{})
	q := reauthQuery()
	q.Set("prompt", "login")
	loc, _ := signedInFor(mux, appID, time.Minute, q)
	assert.Equal(t, "app.example.com", loc.Host)
	assert.Equal(t, "login_required", loc.Query().Get("error"), "nowhere to sign in again, so the client is told")
}

func TestAuthorize_PromptNoneCombined(t *testing.T) {
	mux, appID := reauthFixture(t, oauth2provider.Config{LoginURL: "/sign-in"})
	q := reauthQuery()
	q.Set("prompt", "none login")
	loc, _ := signedInFor(mux, appID, time.Minute, q)
	assert.Equal(t, "invalid_request", loc.Query().Get("error"))
}

func TestAuthorize_PromptNoneNeedsConsent(t *testing.T) {
	_, mux, appID := newConsentFixture(t)
	q := consentQuery("openid")
	q.Set("prompt", "none")
	r := authorizeWith(sessionCtx(fixtureUserID, appID, time.Now()), mux, q)
	require.Equal(t, http.StatusFound, r.Code, r.Body.String())
	loc, err := url.Parse(r.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "app.example.com", loc.Host)
	assert.Equal(t, "consent_required", loc.Query().Get("error"))
	assert.Equal(t, "st-1", loc.Query().Get("state"))
}

func TestAuthorize_LoginAfterAlone(t *testing.T) {
	// login_after can only make a request stricter: a client that sets it
	// to the past gains nothing, and one set to the future sends the user
	// to sign in.
	mux, appID := reauthFixture(t, oauth2provider.Config{LoginURL: "/sign-in"})
	q := reauthQuery()
	q.Set("login_after", "0")
	loc, _ := signedInFor(mux, appID, time.Hour, q)
	assert.NotEmpty(t, loc.Query().Get("code"))

	q.Set("login_after", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
	loc, _ = signedInFor(mux, appID, time.Hour, q)
	assert.Equal(t, "/sign-in", loc.Path)

	q.Set("login_after", "garbage")
	loc, _ = signedInFor(mux, appID, time.Hour, q)
	assert.NotEmpty(t, loc.Query().Get("code"), "an unreadable value adds nothing")
}

func TestAuthorize_ReauthWithABrokenLoginPage(t *testing.T) {
	// OnInit refuses this config; a plugin that skipped it gets an error
	// rather than a redirect to somewhere nobody configured.
	mux, appID := reauthFixture(t, oauth2provider.Config{LoginURL: "sign-in"})
	q := reauthQuery()
	q.Set("prompt", "login")
	_, code := signedInFor(mux, appID, time.Minute, q)
	assert.NotEqual(t, http.StatusFound, code)
}

func TestAuthorize_ErrorToAnUnparsableRedirectURI(t *testing.T) {
	// Only reachable for a URI stored before registration validated them.
	_, st, mux := newFixtureWithConfig(t, oauth2provider.Config{Issuer: "https://auth.example.com"})
	c, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	c.RedirectURIs = []string{"%zz"}
	require.NoError(t, st.UpdateClient(context.Background(), c))

	q := baseAuthorizeQuery(confidentialID)
	q.Set("redirect_uri", "%zz")
	q.Set("prompt", "none login")
	rec := authorizeWith(sessionCtx(fixtureUserID, c.AppID, time.Now()), mux, q)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}
