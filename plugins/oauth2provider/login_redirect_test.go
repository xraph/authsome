package oauth2provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/settings"
)

// newLoginFixture is newFixture with a chosen config.
func newLoginFixture(t *testing.T, cfg oauth2provider.Config) forge.Router {
	t.Helper()
	if cfg.Issuer == "" {
		cfg.Issuer = "https://auth.example.com"
	}
	_, _, mux := newFixtureWithConfig(t, cfg)
	return mux
}

// authorizeSignedOut drives GET /v1/oauth/authorize with no session.
func authorizeSignedOut(t *testing.T, mux forge.Router, q url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/oauth/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func location(t *testing.T, rec *httptest.ResponseRecorder) *url.URL {
	t.Helper()
	require.Equal(t, http.StatusFound, rec.Code, "body: %s", rec.Body.String())
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	return loc
}

func TestAuthorizeSignedOut(t *testing.T) {
	q := baseAuthorizeQuery(confidentialID)
	q.Set("scope", "openid")
	q.Set("state", "st-1")

	t.Run("no login URL keeps the 401", func(t *testing.T) {
		mux := newLoginFixture(t, oauth2provider.Config{})
		rec := authorizeSignedOut(t, mux, q)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("a login page on another origin gets the absolute authorize URL", func(t *testing.T) {
		mux := newLoginFixture(t, oauth2provider.Config{LoginURL: "https://login.example.com/sign-in?theme=dark"})
		loc := location(t, authorizeSignedOut(t, mux, q))

		assert.Equal(t, "login.example.com", loc.Host)
		assert.Equal(t, "/sign-in", loc.Path)
		assert.Equal(t, "dark", loc.Query().Get("theme"), "the login URL's own query survives")

		back, err := url.Parse(loc.Query().Get("redirect"))
		require.NoError(t, err)
		assert.Equal(t, "https", back.Scheme)
		assert.Equal(t, "auth.example.com", back.Host)
		assert.Equal(t, "/v1/oauth/authorize", back.Path)
		assert.Equal(t, q, back.Query(), "every authorize parameter comes back after sign-in")
	})

	t.Run("a login page on the issuer's origin gets a relative path", func(t *testing.T) {
		for _, login := range []string{"/sign-in", "https://auth.example.com/sign-in"} {
			mux := newLoginFixture(t, oauth2provider.Config{LoginURL: login})
			loc := location(t, authorizeSignedOut(t, mux, q))

			back := loc.Query().Get("redirect")
			assert.Equal(t, "/v1/oauth/authorize?"+q.Encode(), back, "login URL %q", login)
		}
	})

	t.Run("the return parameter name is configurable", func(t *testing.T) {
		mux := newLoginFixture(t, oauth2provider.Config{LoginURL: "/sign-in", LoginReturnParam: "return_to"})
		loc := location(t, authorizeSignedOut(t, mux, q))
		assert.NotEmpty(t, loc.Query().Get("return_to"))
		assert.Empty(t, loc.Query().Get("redirect"))
	})

	t.Run("prompt=none answers login_required at the client", func(t *testing.T) {
		pq := url.Values{}
		for k, v := range q {
			pq[k] = v
		}
		pq.Set("prompt", "none")
		for _, login := range []string{"", "/sign-in"} {
			mux := newLoginFixture(t, oauth2provider.Config{LoginURL: login})
			loc := location(t, authorizeSignedOut(t, mux, pq))

			assert.Equal(t, "app.example.com", loc.Host, "login URL %q", login)
			assert.Equal(t, "login_required", loc.Query().Get("error"))
			assert.Equal(t, "st-1", loc.Query().Get("state"))
		}
	})

	t.Run("a bad request is refused before any redirect", func(t *testing.T) {
		mux := newLoginFixture(t, oauth2provider.Config{LoginURL: "/sign-in"})
		bad := baseAuthorizeQuery(confidentialID)
		bad.Set("redirect_uri", "https://evil.example.com/cb")
		rec := authorizeSignedOut(t, mux, bad)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "an unregistered redirect_uri must not reach the login page")
	})
}

func TestNew_RejectsBadLoginURL(t *testing.T) {
	for _, bad := range []string{"sign-in", "javascript:alert(1)", "//evil.example.com/x", "ftp://x.example.com/"} {
		_, err := oauth2provider.ValidateLoginURL(bad)
		assert.Error(t, err, "login URL %q", bad)
	}
	for _, good := range []string{"", "/sign-in", "https://login.example.com/sign-in?x=1", "http://localhost:3000/login"} {
		_, err := oauth2provider.ValidateLoginURL(good)
		assert.NoError(t, err, "login URL %q", good)
	}
}

// setAppLoginURL sets the per-app login page the way the dashboard would.
func setAppLoginURL(t *testing.T, f *oidcFixture, login string) {
	t.Helper()
	raw, err := json.Marshal(login)
	require.NoError(t, err)
	require.NoError(t, f.eng.Settings().Set(context.Background(), "oauth2.login_url", raw,
		settings.ScopeApp, f.appID.String(), f.appID.String(), "", "test"))
}

func TestAuthorizeSignedOut_AppLoginURLWins(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{Issuer: oidcIssuer, LoginURL: "https://login.example.com/default"}, true)
	setAppLoginURL(t, f, "https://muono.example.com/login")

	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {publicID},
		"redirect_uri":          {registeredURI},
		"scope":                 {"openid"},
		"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"code_challenge_method": {"S256"},
	}
	loc := location(t, authorizeSignedOut(t, f.mux, q))
	assert.Equal(t, "muono.example.com", loc.Host)
	assert.Equal(t, "/login", loc.Path)
}

func TestAppLoginURLSetting_RejectsBadValues(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{Issuer: oidcIssuer}, true)
	raw, err := json.Marshal("javascript:alert(1)")
	require.NoError(t, err)
	err = f.eng.Settings().Set(context.Background(), "oauth2.login_url", raw,
		settings.ScopeApp, f.appID.String(), f.appID.String(), "", "test")
	assert.Error(t, err)
}
