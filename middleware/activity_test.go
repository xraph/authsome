package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// ──────────────────────────────────────────���───────
// SessionActivityMiddleware tests
// ──────────────────────────────────────────────────

func TestSessionActivity_Disabled(t *testing.T) {
	var touched int32

	mw := middleware.SessionActivityMiddleware(
		func(_ context.Context, _ id.SessionID, _, _ time.Time) error {
			atomic.AddInt32(&touched, 1)
			return nil
		},
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{Enabled: false}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:    id.NewSessionID(),
		Token: "test-token",
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
	assert.Equal(t, int32(0), atomic.LoadInt32(&touched), "toucher should not be called when disabled")
}

func TestSessionActivity_ExtendExpiry(t *testing.T) {
	var touchedSessionID id.SessionID

	mw := middleware.SessionActivityMiddleware(
		func(_ context.Context, sessID id.SessionID, _, _ time.Time) error {
			touchedSessionID = sessID
			return nil
		},
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{
				Enabled:           true,
				InactivityTimeout: 30 * time.Minute,
			}
		},
		log.NewNoopLogger(),
	)

	sessID := id.NewSessionID()
	sess := &session.Session{
		ID:             sessID,
		Token:          "test-token",
		LastActivityAt: time.Now().Add(-2 * time.Minute), // 2 min ago — past throttle interval
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
	assert.Equal(t, sessID, touchedSessionID, "toucher should be called with the session ID")
}

func TestSessionActivity_ThrottleTouchInterval(t *testing.T) {
	var touched int32

	mw := middleware.SessionActivityMiddleware(
		func(_ context.Context, _ id.SessionID, _, _ time.Time) error {
			atomic.AddInt32(&touched, 1)
			return nil
		},
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{
				Enabled:           true,
				InactivityTimeout: 30 * time.Minute,
			}
		},
		log.NewNoopLogger(),
	)

	sess := &session.Session{
		ID:             id.NewSessionID(),
		Token:          "test-token",
		LastActivityAt: time.Now().Add(-10 * time.Second), // 10s ago — within throttle
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
	assert.Equal(t, int32(0), atomic.LoadInt32(&touched), "toucher should be throttled for recent activity")
}

func TestSessionActivity_CookieSetter_Called(t *testing.T) {
	var cookieToken string
	var cookieMaxAge int

	setter := func(_ forge.Context, token string, maxAge int) {
		cookieToken = token
		cookieMaxAge = maxAge
	}

	mw := middleware.SessionActivityMiddleware(
		func(_ context.Context, _ id.SessionID, _, _ time.Time) error {
			return nil
		},
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{
				Enabled:           true,
				InactivityTimeout: 30 * time.Minute,
			}
		},
		log.NewNoopLogger(),
		setter,
	)

	sess := &session.Session{
		ID:             id.NewSessionID(),
		Token:          "my-session-token",
		LastActivityAt: time.Now().Add(-2 * time.Minute),
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
	assert.Equal(t, "my-session-token", cookieToken, "cookie setter should be called with session token")
	assert.Equal(t, 1800, cookieMaxAge, "cookie MaxAge should match inactivity timeout")
}

func TestSessionActivity_NoCookieSetter_NoPanic(t *testing.T) {
	mw := middleware.SessionActivityMiddleware(
		func(_ context.Context, _ id.SessionID, _, _ time.Time) error {
			return nil
		},
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{
				Enabled:           true,
				InactivityTimeout: 30 * time.Minute,
			}
		},
		log.NewNoopLogger(),
		// no cookie setter — backward compatibility
	)

	sess := &session.Session{
		ID:             id.NewSessionID(),
		Token:          "test-token",
		LastActivityAt: time.Now().Add(-2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic.
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSessionActivity_NoSession_NoOp(t *testing.T) {
	var touched int32

	mw := middleware.SessionActivityMiddleware(
		func(_ context.Context, _ id.SessionID, _, _ time.Time) error {
			atomic.AddInt32(&touched, 1)
			return nil
		},
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{Enabled: true}
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
	assert.Equal(t, int32(0), atomic.LoadInt32(&touched))
}

// injectSession is a test middleware that puts a session on context.
func injectSession(sess *session.Session) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			goCtx := middleware.WithSession(ctx.Context(), sess)
			goCtx = middleware.WithSessionID(goCtx, sess.ID)
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	}
}
