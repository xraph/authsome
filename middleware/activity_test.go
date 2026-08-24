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

// TestSessionActivity_AgentSession_NotExtended is the middleware/activity.go
// half of the "an agent session never outlives its grant" invariant, the
// third of three guards alongside Engine.Refresh's agent-principal refusal
// (service_test.go: TestRefresh_RefusesAgentPrincipalSession) and
// roleStampingStore's shouldRestamp exclusion
// (engine_session_roles_test.go: TestRotateSessionSkipsAgents). An agent
// session's ExpiresAt must survive the activity middleware completely
// unchanged, and the toucher (the store write) must never be called for it.
func TestSessionActivity_AgentSession_NotExtended(t *testing.T) {
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

	originalExpiresAt := time.Now().Add(10 * time.Minute) // a grant-clamped agent TTL
	sess := &session.Session{
		ID:             id.NewSessionID(),
		Token:          "agent-test-token",
		PrincipalKind:  session.PrincipalKindAgent,
		AgentID:        id.NewAgentID(),
		GrantID:        id.NewAgentGrantID(),
		LastActivityAt: time.Now().Add(-2 * time.Minute), // past throttle interval
		ExpiresAt:      originalExpiresAt,
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
	assert.Equal(t, int32(0), atomic.LoadInt32(&touched), "an agent session must never be touched by the sliding window")
	assert.True(t, sess.ExpiresAt.Equal(originalExpiresAt), "an agent session's ExpiresAt must survive the activity middleware unchanged")
}

// TestSessionActivity_HumanSession_StillExtended is the control for the test
// above: it proves the agent exclusion did not disable the sliding window
// for everyone. A human session under the identical middleware, config and
// throttle window must still be extended exactly as before.
func TestSessionActivity_HumanSession_StillExtended(t *testing.T) {
	var touchedSessionID id.SessionID
	var touched int32

	mw := middleware.SessionActivityMiddleware(
		func(_ context.Context, sessID id.SessionID, _, _ time.Time) error {
			touchedSessionID = sessID
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

	sessID := id.NewSessionID()
	originalExpiresAt := time.Now().Add(5 * time.Minute)
	sess := &session.Session{
		ID:             sessID,
		Token:          "human-test-token",
		LastActivityAt: time.Now().Add(-2 * time.Minute), // past throttle interval
		ExpiresAt:      originalExpiresAt,
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
	assert.Equal(t, int32(1), atomic.LoadInt32(&touched), "a human session must still be touched by the sliding window")
	assert.Equal(t, sessID, touchedSessionID)
	assert.True(t, sess.ExpiresAt.After(originalExpiresAt), "a human session's ExpiresAt must still be extended")
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
