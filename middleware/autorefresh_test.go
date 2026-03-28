package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// ──────────────────────────────────────────────────
// AutoRefreshMiddleware tests
// ──────────────────────────────────────────────────

func TestAutoRefresh_Disabled(t *testing.T) {
	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ string) (*session.Session, error) {
			t.Fatal("refresher should not be called when disabled")
			return nil, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: false}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:        id.NewSessionID(),
		Token:     "original-token",
		ExpiresAt: time.Now().Add(2 * time.Minute), // near expiry
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("X-Auth-Token"), "no refresh header when disabled")
}

func TestAutoRefresh_NotNearExpiry(t *testing.T) {
	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ string) (*session.Session, error) {
			t.Fatal("refresher should not be called when token not near expiry")
			return nil, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:   true,
				Threshold: 5 * time.Minute,
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "original-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(30 * time.Minute), // far from expiry
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("X-Auth-Token"), "no refresh when not near expiry")
}

func TestAutoRefresh_NearExpiry_RefreshesAndSetsHeaders(t *testing.T) {
	refreshedSess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ string) (*session.Session, error) {
			return refreshedSess, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:            true,
				Threshold:          5 * time.Minute,
				ExposeRefreshToken: true,
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute), // within 5-min threshold
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "new-access-token", rec.Header().Get("X-Auth-Token"))
	assert.NotEmpty(t, rec.Header().Get("X-Auth-Token-Expires-At"))
	assert.Equal(t, "new-refresh-token", rec.Header().Get("X-Auth-Refresh-Token"))
}

func TestAutoRefresh_RefreshTokenNotExposedByDefault(t *testing.T) {
	refreshedSess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ string) (*session.Session, error) {
			return refreshedSess, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:            true,
				Threshold:          5 * time.Minute,
				ExposeRefreshToken: false, // default — don't expose
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "new-access-token", rec.Header().Get("X-Auth-Token"), "access token should be in header")
	assert.Empty(t, rec.Header().Get("X-Auth-Refresh-Token"), "refresh token should NOT be in header")
}

func TestAutoRefresh_CookieSetter_Called(t *testing.T) {
	var cookieToken string
	var cookieMaxAge int

	refreshedSess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "refreshed-token",
		RefreshToken: "refreshed-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	setter := func(_ forge.Context, token string, maxAge int) {
		cookieToken = token
		cookieMaxAge = maxAge
	}

	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ string) (*session.Session, error) {
			return refreshedSess, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:   true,
				Threshold: 5 * time.Minute,
			}
		},
		log.NewNoopLogger(),
		setter,
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "refreshed-token", cookieToken, "cookie setter should receive the new token")
	assert.Greater(t, cookieMaxAge, 0, "cookie MaxAge should be positive")
}

func TestAutoRefresh_RefreshFailure_NonFatal(t *testing.T) {
	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ string) (*session.Session, error) {
			return nil, errors.New("refresh failed")
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{
				Enabled:   true,
				Threshold: 5 * time.Minute,
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:           id.NewSessionID(),
		Token:        "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Response should still be OK even when refresh fails
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("X-Auth-Token"), "no new token header on refresh failure")
}

func TestAutoRefresh_NoSession_NoOp(t *testing.T) {
	mw := middleware.AutoRefreshMiddleware(
		func(_ context.Context, _ string) (*session.Session, error) {
			t.Fatal("refresher should not be called without a session")
			return nil, nil
		},
		func(_ context.Context) middleware.AutoRefreshConfig {
			return middleware.AutoRefreshConfig{Enabled: true, Threshold: 5 * time.Minute}
		},
		log.NewNoopLogger(),
	)

	router := forge.NewRouter()
	// No session injected
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
