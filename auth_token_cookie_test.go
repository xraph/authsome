package authsome_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// cookieProbe signs a user up and returns a function that reports whether a
// request carrying only the given cookie is signed in.
func cookieProbe(t *testing.T, eng *authsome.Engine) func(cookie string) bool {
	t.Helper()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: appID, Email: "cookie@example.com", Password: "SecureP@ss123", FirstName: "Cookie",
	})
	require.NoError(t, err)

	var ok bool
	router := forge.NewRouter()
	router.Use(eng.AuthMiddleware())
	require.NoError(t, router.GET("/probe", func(fctx forge.Context) error {
		_, ok = middleware.UserIDFrom(fctx.Context())
		return fctx.NoContent(http.StatusOK)
	}))
	return func(cookie string) bool {
		ok = false
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/probe", nil)
		req.AddCookie(&http.Cookie{Name: cookie, Value: sess.Token})
		router.ServeHTTP(httptest.NewRecorder(), req)
		return ok
	}
}

// auth_token is a generic name, and cookies are shared by every port on a
// host. Reading it unasked let another app's cookie on localhost sign
// requests in here.
func TestAuthTokenCookie_IgnoredUnlessAccepted(t *testing.T) {
	eng, _ := newTestEngine(t)
	signedIn := cookieProbe(t, eng)

	assert.False(t, signedIn("auth_token"), "auth_token must not authenticate by default")
	assert.True(t, signedIn("authsome_session_token"), "the configured session cookie still authenticates")
}

func TestAuthTokenCookie_AcceptedFromConfig(t *testing.T) {
	cfg := testEngineConfig()
	cfg.Session.AuthTokenCookie = true
	eng, _ := newTestEngine(t, authsome.WithConfig(cfg))
	signedIn := cookieProbe(t, eng)

	assert.True(t, signedIn("auth_token"))
}

// The dashboard registers after the middleware is built, so turning it on
// later must reach a middleware that is already mounted.
func TestAuthTokenCookie_AcceptedOnceTheDashboardAsks(t *testing.T) {
	eng, _ := newTestEngine(t)
	signedIn := cookieProbe(t, eng)
	require.False(t, signedIn("auth_token"))

	eng.AcceptAuthTokenCookie()
	assert.True(t, signedIn("auth_token"))
}
