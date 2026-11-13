package mfa_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	mfa "github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// mfaChallengeLimit is the MFA-challenge limit newMFARateLimited sets.
const mfaChallengeLimit = 3

// newMFARateLimited builds an engine with rate limiting enabled at
// mfaChallengeLimit, mounts the MFA plugin routes, and returns the handler.
func newMFARateLimited(t *testing.T) http.Handler {
	t.Helper()
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.Enabled = true
	cfg.RateLimit.MFAChallengeLimit = mfaChallengeLimit
	cfg.RateLimit.WindowSeconds = 60

	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	p := mfa.New()
	eng, err := authsome.NewEngine(authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(memory.New()),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(cfg),
		authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()),
		authsome.WithPlugin(p),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })

	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	return router.Handler()
}

// TestMFAVerify_RateLimited pins that the MFA code-verification endpoint is
// throttled: past the MFAChallengeLimit, further attempts return 429. Without
// this an attacker can brute-force the 6-digit code space unthrottled.
func TestMFAVerify_RateLimited(t *testing.T) {
	const limit = mfaChallengeLimit
	h := newMFARateLimited(t)

	var lastCode int
	for i := 0; i < limit+1; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/mfa/verify", nil)
		req.RemoteAddr = "203.0.113.7:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		lastCode = rec.Code
	}

	require.Equal(t, http.StatusTooManyRequests, lastCode,
		"the attempt beyond the MFA challenge limit must be rate limited (429)")
}

// TestMFASMSVerify_RateLimited pins the same protection on the SMS verify route.
func TestMFASMSVerify_RateLimited(t *testing.T) {
	const limit = mfaChallengeLimit
	h := newMFARateLimited(t)

	var lastCode int
	for i := 0; i < limit+1; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/mfa/sms/verify", nil)
		req.RemoteAddr = "203.0.113.8:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		lastCode = rec.Code
	}

	require.Equal(t, http.StatusTooManyRequests, lastCode,
		"the attempt beyond the limit on SMS verify must be rate limited (429)")
}

// TestMFARecoveryRegenerate_RateLimited pins that regenerate, which accepts a
// TOTP code, is throttled like the other code-submission routes.
func TestMFARecoveryRegenerate_RateLimited(t *testing.T) {
	const limit = mfaChallengeLimit
	h := newMFARateLimited(t)

	var lastCode int
	for i := 0; i < limit+1; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/mfa/recovery/regenerate", nil)
		req.RemoteAddr = "203.0.113.9:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		lastCode = rec.Code
	}

	require.Equal(t, http.StatusTooManyRequests, lastCode,
		"the attempt beyond the MFA challenge limit must be rate limited (429)")
}

// TestMFAEnrollAndDisable_RateLimited pins the same protection on the two
// routes that take a TOTP or recovery code as step-up: enroll (a second
// factor) and DELETE /enrollment (turning MFA off). Without it a stolen
// session can guess the code that removes the second factor.
func TestMFAEnrollAndDisable_RateLimited(t *testing.T) {
	const limit = mfaChallengeLimit
	for _, c := range []struct{ method, path, addr string }{
		{"POST", "/v1/mfa/enroll", "203.0.113.10:5555"},
		{"DELETE", "/v1/mfa/enrollment", "203.0.113.11:5555"},
	} {
		h := newMFARateLimited(t)
		var lastCode int
		for i := 0; i < limit+1; i++ {
			req := httptest.NewRequestWithContext(context.Background(), c.method, c.path, nil)
			req.RemoteAddr = c.addr
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			lastCode = rec.Code
		}
		require.Equal(t, http.StatusTooManyRequests, lastCode,
			"%s %s: the attempt beyond the MFA challenge limit must be rate limited (429)", c.method, c.path)
	}
}

// The MFA code routes have their own per-address budget. Sign-up, email
// verification and sign-in spend the engine's shared one, and a person who
// enrolls a factor straight after signing up must not find their first
// code refused because of them.
func TestMFA_ownBucketSeparateFromSignIn(t *testing.T) {
	const limit = 3
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.Enabled = true
	cfg.RateLimit.MFAChallengeLimit = limit
	cfg.RateLimit.WindowSeconds = 60

	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	p := mfa.New()
	eng, err := authsome.NewEngine(authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(memory.New()),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(cfg),
		authsome.WithRateLimiter(ratelimit.NewMemoryLimiter()),
		authsome.WithPlugin(p),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })

	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	// A stand-in for sign-in: a route on the engine's own limiter.
	require.NoError(t, router.POST("/v1/signin", func(c forge.Context) error { return c.NoContent(http.StatusOK) },
		eng.RateLimitOptions(limit)...))
	h := router.Handler()
	send := func(method, path string) int {
		req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
		req.RemoteAddr = "203.0.113.12:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	for i := 0; i < limit; i++ {
		require.Equal(t, http.StatusOK, send("POST", "/v1/signin"))
	}
	require.Equal(t, http.StatusTooManyRequests, send("POST", "/v1/signin"), "the sign-in budget is spent")
	require.NotEqual(t, http.StatusTooManyRequests, send("POST", "/v1/mfa/verify"),
		"MFA codes have their own budget")
}
