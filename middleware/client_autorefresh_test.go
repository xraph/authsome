package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	authclient "github.com/xraph/authsome/sdk/go"
)

// ──────────────────────────────────────────────────
// Client-mode session refresh
//
// A client-mode service holds no session store and never sees a refresh token,
// so it keeps a browser's cookie session alive by forwarding the browser's own
// Cookie header to the identity server's cookie-first /v1/refresh, then
// replaying that server's Set-Cookie back to the browser.
// ──────────────────────────────────────────────────

const refreshCookieName = "authsome_session_token"

// refreshStub is a fake identity server: it introspects one token with a
// caller-chosen expiry and rotates it on /v1/refresh.
type refreshStub struct {
	server *httptest.Server

	// refreshCalls counts /v1/refresh hits.
	refreshCalls atomic.Int32
	// lastCookie records the Cookie header the refresh call arrived with.
	lastCookie atomic.Value
	// refreshStatus, when non-zero, is returned by /v1/refresh instead of
	// rotating.
	refreshStatus atomic.Int32
}

// newRefreshStub stands up the fake identity server. expiresIn is how far in
// the future the introspected token expires; pass 0 to report no expiry at all.
func newRefreshStub(t *testing.T, token string, expiresIn time.Duration) *refreshStub {
	t.Helper()

	stub := &refreshStub{}
	userID := id.NewUserID().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/introspect", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		resp := authclient.IntrospectResponse{}
		if body.Token == token {
			resp.Active = true
			resp.UserID = userID
			resp.AppID = id.NewAppID().String()
			resp.SessionID = id.NewSessionID().String()
			resp.User = &authclient.IntrospectUser{ID: userID, Email: "refresh@test.com"}
			if expiresIn != 0 {
				resp.ExpiresAt = time.Now().Add(expiresIn).Format(time.RFC3339)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	})
	mux.HandleFunc("/v1/refresh", func(w http.ResponseWriter, r *http.Request) {
		stub.refreshCalls.Add(1)
		stub.lastCookie.Store(r.Header.Get("Cookie"))

		if code := stub.refreshStatus.Load(); code != 0 {
			w.WriteHeader(int(code))
			_, _ = w.Write([]byte(`{"error":"invalid refresh"}`))
			return
		}

		// Mirror the real handler: rotate, and set the session cookie with the
		// attributes this server resolved from its own settings.
		http.SetCookie(w, &http.Cookie{
			Name:     refreshCookieName,
			Value:    "rotated-token",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   3600,
		})
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{
			"session_token": "rotated-token",
			"refresh_token": "rotated-refresh",
			"expires_at":    time.Now().Add(time.Hour).Format(time.RFC3339),
		}))
	})

	stub.server = httptest.NewServer(mux)
	t.Cleanup(stub.server.Close)
	return stub
}

func (s *refreshStub) cookieHeader() string {
	v, _ := s.lastCookie.Load().(string)
	return v
}

// refreshRouter wires ClientAuthMiddleware + ClientAutoRefreshMiddleware
// against the stub, exactly as Extension.Middlewares does in client mode.
func refreshRouter(stub *refreshStub, threshold time.Duration) forge.Router {
	client := authclient.NewClient(stub.server.URL)
	router := forge.NewRouter()
	router.Use(middleware.ClientAuthMiddleware(client, log.NewNoopLogger()))
	router.Use(middleware.ClientAutoRefreshMiddleware(
		middleware.NewClientSessionRefresher(client),
		middleware.AutoRefreshConfig{Enabled: true, Threshold: threshold},
		log.NewNoopLogger(),
	))
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})
	return router
}

// testResponse is the part of a response these tests assert on. The helpers
// hand back one of these rather than the *http.Response itself so the body is
// closed where it is read, instead of leaving every caller holding one open.
type testResponse struct {
	StatusCode int
	Header     http.Header
	Cookies    []*http.Cookie
}

// serveRequest issues GET /test against router over a real connection, letting
// decorate attach whatever credential the caller is exercising.
//
// Deliberately not an httptest.ResponseRecorder: a recorder's Header() map goes
// on accepting writes after WriteHeader has already snapshotted it, so a header
// this middleware sets too late still reads back fine. Forge streams the
// response straight to the connection, so only a real server can tell whether a
// header was actually delivered.
func serveRequest(t *testing.T, router forge.Router, decorate func(*http.Request)) testResponse {
	t.Helper()

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	require.NoError(t, err)
	if decorate != nil {
		decorate(req)
	}

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	return testResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Cookies:    resp.Cookies(),
	}
}

// cookieRequest presents token in the session cookie, the way a browser does.
func cookieRequest(t *testing.T, router forge.Router, token string) testResponse {
	t.Helper()
	return serveRequest(t, router, func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: token})
	})
}

// bearerRequest is cookieRequest's Authorization-header counterpart.
func bearerRequest(t *testing.T, router forge.Router, token string) testResponse {
	t.Helper()
	return serveRequest(t, router, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	})
}

// anonRequest issues GET /test with no credential at all.
func anonRequest(t *testing.T, router forge.Router) testResponse {
	t.Helper()
	return serveRequest(t, router, nil)
}

// TestClientAutoRefresh_RotatesNearExpiry is the fix this middleware exists
// for: a cookie session inside the refresh window gets rotated, and the
// identity server's own Set-Cookie reaches the browser.
func TestClientAutoRefresh_RotatesNearExpiry(t *testing.T) {
	stub := newRefreshStub(t, "near-expiry-token", 2*time.Minute)
	resp := cookieRequest(t, refreshRouter(stub, 5*time.Minute), "near-expiry-token")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(1), stub.refreshCalls.Load(), "expected one refresh")
	assert.Equal(t, "rotated-token", resp.Header.Get("X-Auth-Token"))
	assert.NotEmpty(t, resp.Header.Get("X-Auth-Token-Expires-At"))

	// The rotated cookie is replayed verbatim, so its attributes are whatever
	// the identity server chose rather than anything this service guessed.
	cookies := resp.Cookies
	require.Len(t, cookies, 1, "rotated cookie must reach the browser")
	assert.Equal(t, refreshCookieName, cookies[0].Name)
	assert.Equal(t, "rotated-token", cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, 3600, cookies[0].MaxAge)
}

// The browser's Cookie header is what authorises the rotation — /v1/refresh is
// cookie-first, and this service has no refresh token to offer instead.
func TestClientAutoRefresh_ForwardsBrowserCookie(t *testing.T) {
	stub := newRefreshStub(t, "near-expiry-token", 2*time.Minute)
	cookieRequest(t, refreshRouter(stub, 5*time.Minute), "near-expiry-token")

	assert.Contains(t, stub.cookieHeader(), refreshCookieName+"=near-expiry-token")
}

// Refresh must not fire on every request, only inside the window.
func TestClientAutoRefresh_SkipsWhenFarFromExpiry(t *testing.T) {
	stub := newRefreshStub(t, "fresh-token", 50*time.Minute)
	resp := cookieRequest(t, refreshRouter(stub, 5*time.Minute), "fresh-token")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Zero(t, stub.refreshCalls.Load(), "no refresh expected far from expiry")
	assert.Empty(t, resp.Header.Get("X-Auth-Token"))
}

// The window comes from the configured threshold rather than a constant baked
// into the middleware: the same 50-minute session left alone above is rotated
// once the threshold is widened past its expiry.
func TestClientAutoRefresh_HonoursConfiguredThreshold(t *testing.T) {
	stub := newRefreshStub(t, "fresh-token", 50*time.Minute)
	resp := cookieRequest(t, refreshRouter(stub, 60*time.Minute), "fresh-token")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(1), stub.refreshCalls.Load(), "a wider threshold must bring this session into the window")
	assert.Equal(t, "rotated-token", resp.Header.Get("X-Auth-Token"))
}

// An introspection response with no expiry gives no basis to decide, so the
// session is left alone rather than rotated on every single request.
func TestClientAutoRefresh_SkipsWhenExpiryUnknown(t *testing.T) {
	stub := newRefreshStub(t, "no-expiry-token", 0)
	resp := cookieRequest(t, refreshRouter(stub, 5*time.Minute), "no-expiry-token")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Zero(t, stub.refreshCalls.Load(), "no refresh expected without a known expiry")
}

// An API client presenting a Bearer token owns its own refresh token; rotating
// underneath it would revoke the one still in its hand.
func TestClientAutoRefresh_SkipsBearerCredential(t *testing.T) {
	stub := newRefreshStub(t, "near-expiry-token", 2*time.Minute)
	resp := bearerRequest(t, refreshRouter(stub, 5*time.Minute), "near-expiry-token")

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Zero(t, stub.refreshCalls.Load(), "bearer credentials must not be rotated for")
}

// A failed rotation is non-fatal: the handler's response still goes out, and
// the browser keeps working until the original expiry.
func TestClientAutoRefresh_RefreshFailureIsNonFatal(t *testing.T) {
	stub := newRefreshStub(t, "near-expiry-token", 2*time.Minute)
	stub.refreshStatus.Store(http.StatusUnauthorized)

	resp := cookieRequest(t, refreshRouter(stub, 5*time.Minute), "near-expiry-token")

	require.Equal(t, http.StatusOK, resp.StatusCode, "refresh failure must not change the response")
	assert.Equal(t, int32(1), stub.refreshCalls.Load())
	assert.Empty(t, resp.Header.Get("X-Auth-Token"))
	assert.Empty(t, resp.Cookies)
}

// An unauthenticated request has no session to rotate.
func TestClientAutoRefresh_SkipsUnauthenticated(t *testing.T) {
	stub := newRefreshStub(t, "near-expiry-token", 2*time.Minute)
	resp := anonRequest(t, refreshRouter(stub, 5*time.Minute))

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Zero(t, stub.refreshCalls.Load())
}

// The refresh token stays server-side unless explicitly opted into, matching
// the session.auto_refresh_expose_refresh_token default of false.
func TestClientAutoRefresh_WithholdsRefreshTokenByDefault(t *testing.T) {
	stub := newRefreshStub(t, "near-expiry-token", 2*time.Minute)
	resp := cookieRequest(t, refreshRouter(stub, 5*time.Minute), "near-expiry-token")

	assert.Empty(t, resp.Header.Get("X-Auth-Refresh-Token"))
}

// RefreshTokensWithCookies refuses an empty Cookie header before spending a
// round-trip to be told the same thing.
func TestRefreshTokensWithCookies_RequiresCookies(t *testing.T) {
	stub := newRefreshStub(t, "tok", time.Minute)
	client := authclient.NewClient(stub.server.URL)

	_, err := client.RefreshTokensWithCookies(context.Background(), "  ")

	require.Error(t, err)
	assert.Zero(t, stub.refreshCalls.Load(), "no HTTP call expected")
}
