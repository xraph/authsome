package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/ratelimit"
)

// The routes that consume a guessable secret are throttled like sign-in.
// The fixture relaxes limits for the rest of the suite, so these tests put a
// real limiter back before the routes are built.

func limitedAPI(t *testing.T) http.Handler {
	t.Helper()
	a, eng := newTestAPI(t)
	eng.SetRateLimiter(ratelimit.NewMemoryLimiter())
	return withTestKey(a.Handler())
}

func postJSONRaw(t *testing.T, h http.Handler, path, body string, decorate func(*http.Request) *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if decorate != nil {
		req = decorate(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestResetPassword_IsRateLimited(t *testing.T) {
	h := limitedAPI(t)
	body := `{"token":"not-a-real-token","new_password":"An0therStr0ng!Pass"}`
	var last int
	for i := 0; i < 6; i++ {
		last = postJSONRaw(t, h, "/v1/reset-password", body, nil).Code
		if i < 5 {
			require.NotEqual(t, http.StatusTooManyRequests, last, "request %d is within the budget", i+1)
		}
	}
	assert.Equal(t, http.StatusTooManyRequests, last, "the sixth reset attempt in a window is refused")
}

func TestChangePassword_IsRateLimited(t *testing.T) {
	a, eng := newTestAPI(t)
	eng.SetRateLimiter(ratelimit.NewMemoryLimiter())
	h := withTestKey(a.Handler())
	_, token, _ := signUp(t, eng, "limited-change@test.com", "SecureP@ss1")
	uid := userIDFor(t, eng, token)

	body := `{"current_password":"wrong-password","new_password":"An0therStr0ng!Pass"}`
	var last int
	for i := 0; i < 6; i++ {
		last = postJSONRaw(t, h, "/v1/change-password", body, func(r *http.Request) *http.Request {
			return asAdmin(t, r, eng, uid)
		}).Code
		if i < 5 {
			require.NotEqual(t, http.StatusTooManyRequests, last, "request %d is within the budget", i+1)
		}
	}
	assert.Equal(t, http.StatusTooManyRequests, last, "the sixth change attempt in a window is refused")
}
