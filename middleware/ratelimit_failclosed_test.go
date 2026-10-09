package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
)

type brokenLimiter struct{}

func (brokenLimiter) Allow(context.Context, string, int, time.Duration) (bool, error) {
	return false, errors.New("store down")
}

func (brokenLimiter) Remaining(context.Context, string, int, time.Duration) (int, error) {
	return 0, errors.New("store down")
}

func serveLimited(t *testing.T, cfg middleware.RateLimitConfig) *httptest.ResponseRecorder {
	t.Helper()
	router := forge.NewRouter()
	router.Use(middleware.RateLimit(brokenLimiter{}, cfg))
	require.NoError(t, router.GET("/test", func(ctx forge.Context) error { return ctx.NoContent(http.StatusOK) }))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// A limiter that cannot answer must not become an unlimited window.
func TestRateLimit_FailsClosedByDefault(t *testing.T) {
	var seen error
	rec := serveLimited(t, middleware.RateLimitConfig{
		Limit: 3, Window: time.Minute,
		OnError: func(err error) { seen = err },
	})
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.Error(t, seen, "the error is reported for a metric")
}

func TestRateLimit_FailOpenIsExplicit(t *testing.T) {
	rec := serveLimited(t, middleware.RateLimitConfig{Limit: 3, Window: time.Minute, FailOpen: true})
	assert.Equal(t, http.StatusOK, rec.Code)
}
