package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// ──────────────────────────────────────────────────
// Response delivery
//
// Forge hands middleware the raw http.ResponseWriter and streams the response
// to the connection as the handler writes it. Anything a middleware sets after
// next() returns has already missed the wire.
//
// Every other test in this package asserts through httptest.ResponseRecorder,
// which cannot detect that: WriteHeader snapshots the header map into
// snapHeader, but Header() keeps handing back the live map, so a header set far
// too late still reads back perfectly. That is what let AutoRefreshMiddleware
// and SessionActivityMiddleware ship for as long as they did while their
// rotated token and extended cookie Max-Age never reached any browser.
//
// The tests below go over a real connection, which is the only thing that
// proves delivery.
// ──────────────────────────────────────────────────

// serveOverTheWire runs one authenticated GET against a real server and returns
// the response as a client saw it.
func serveOverTheWire(t *testing.T, router forge.Router) *http.Response {
	t.Helper()

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// deliveryCookieSetter is the CookieSetter both middlewares are given, standing
// in for Extension.cookieSetter.
func deliveryCookieSetter(ctx forge.Context, token string, maxAge int) {
	http.SetCookie(ctx.Response(), &http.Cookie{
		Name:     "authsome_session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   maxAge,
	})
}

// TestAutoRefresh_DeliversRotatedCookieAndHeaders is the regression guard for
// the post-handler bug: a rotated session has to reach the client, or the
// browser goes on presenting a token the store no longer holds.
func TestAutoRefresh_DeliversRotatedCookieAndHeaders(t *testing.T) {
	sess := &session.Session{
		ID:           id.NewSessionID(),
		UserID:       id.NewUserID(),
		Token:        "test-token",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute), // inside the window
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			return &session.Session{
				ID:           sess.ID,
				Token:        "rotated-token",
				RefreshToken: "rotated-refresh",
				ExpiresAt:    time.Now().Add(time.Hour),
			}, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: true, Threshold: 5 * time.Minute}
		},
		log.NewNoopLogger(),
		deliveryCookieSetter,
	))
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	resp := serveOverTheWire(t, router)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "rotated-token", resp.Header.Get("X-Auth-Token"),
		"the rotated access token must reach the client")
	assert.NotEmpty(t, resp.Header.Get("X-Auth-Token-Expires-At"))

	cookies := resp.Cookies()
	require.Len(t, cookies, 1, "the rotated session cookie must reach the browser")
	assert.Equal(t, "rotated-token", cookies[0].Value)
}

// The sliding window's whole purpose is that the browser holds its cookie for
// as long as the server holds the session. That only works if the extended
// Max-Age is actually delivered.
func TestSessionActivity_DeliversExtendedCookie(t *testing.T) {
	sess := &session.Session{
		ID:     id.NewSessionID(),
		UserID: id.NewUserID(),
		Token:  "test-token",
		// Zero LastActivityAt so the write throttle does not skip the touch.
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(middleware.SessionActivityMiddleware(
		func(_ context.Context, _ id.SessionID, _, _ time.Time) error { return nil },
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{
				Enabled:           true,
				InactivityTimeout: 7 * 24 * time.Hour,
			}
		},
		log.NewNoopLogger(),
		deliveryCookieSetter,
	))
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	resp := serveOverTheWire(t, router)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookies := resp.Cookies()
	require.Len(t, cookies, 1, "the extended session cookie must reach the browser")
	assert.Equal(t, "test-token", cookies[0].Value)
	assert.Equal(t, int((7 * 24 * time.Hour).Seconds()), cookies[0].MaxAge,
		"Max-Age must carry the extended lifetime, not the token TTL")
}

// Ordering guard. Both middlewares now work before the handler, so they run in
// the same pass — and the sliding window rewrites the very field auto-refresh
// reads. Extended first, every session sits permanently outside the refresh
// window and rotation quietly stops happening. Extension.Middlewares orders
// auto-refresh ahead of the sliding window to prevent that; this pins the
// property that ordering exists to protect.
func TestAutoRefresh_SeesTokenExpiryNotTheSlidingExtension(t *testing.T) {
	sess := &session.Session{
		ID:           id.NewSessionID(),
		UserID:       id.NewUserID(),
		Token:        "test-token",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute), // inside the window
	}

	refreshed := false
	router := forge.NewRouter()
	router.Use(injectSession(sess))
	// Same order as Extension.Middlewares: auto-refresh, then the window.
	router.Use(middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ middleware.RefreshRequest) (*session.Session, error) {
			refreshed = true
			return &session.Session{ID: sess.ID, Token: "rotated-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: true, Threshold: 5 * time.Minute}
		},
		log.NewNoopLogger(),
	))
	router.Use(middleware.SessionActivityMiddleware(
		func(_ context.Context, _ id.SessionID, _, _ time.Time) error { return nil },
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{Enabled: true, InactivityTimeout: 7 * 24 * time.Hour}
		},
		log.NewNoopLogger(),
	))
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	serveOverTheWire(t, router)

	assert.True(t, refreshed,
		"auto-refresh must evaluate the token's own expiry, before the sliding window rewrites it")
}
