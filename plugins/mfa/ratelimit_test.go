package mfa_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	mfa "github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/ratelimit"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// newMFARateLimited builds an engine with rate limiting enabled at the given
// MFA-challenge limit, mounts the MFA plugin routes, and returns the handler.
func newMFARateLimited(t *testing.T, limit int) http.Handler {
	t.Helper()
	cfg := authsome.DefaultConfig()
	cfg.AppID = "aapp_01jf0000000000000000000000"
	cfg.RateLimit.Enabled = true
	cfg.RateLimit.MFAChallengeLimit = limit
	cfg.RateLimit.WindowSeconds = 60

	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	p := mfa.New()
	eng, err := authsome.NewEngine(
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
	const limit = 3
	h := newMFARateLimited(t, limit)

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
	const limit = 3
	h := newMFARateLimited(t, limit)

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
