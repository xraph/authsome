package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/ratelimit"
)

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	limiter := ratelimit.NewMemoryLimiter()
	mw := middleware.RateLimit(limiter, middleware.RateLimitConfig{
		Limit:  3,
		Window: 10 * time.Second,
	})

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimit_DeniesOverLimit(t *testing.T) {
	limiter := ratelimit.NewMemoryLimiter()
	mw := middleware.RateLimit(limiter, middleware.RateLimitConfig{
		Limit:  2,
		Window: 10 * time.Second,
	})

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	// Use up the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Should be denied
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Should have Retry-After header
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimit_NoopAllowsAll(t *testing.T) {
	limiter := ratelimit.NewNoopLimiter()
	mw := middleware.RateLimit(limiter, middleware.RateLimitConfig{
		Limit:  1,
		Window: 10 * time.Second,
	})

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
