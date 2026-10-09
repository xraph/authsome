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

// The sliding window never moves a session past its absolute lifetime.
func TestSessionActivity_ClampedToAbsoluteLifetime(t *testing.T) {
	var touchedExpiry time.Time
	mw := middleware.SessionActivityMiddleware(
		func(_ context.Context, _ id.SessionID, _, expiresAt time.Time) error {
			touchedExpiry = expiresAt
			return nil
		},
		func(_ context.Context) middleware.SessionActivityConfig {
			return middleware.SessionActivityConfig{
				Enabled:           true,
				InactivityTimeout: 30 * time.Minute,
				AbsoluteLifetime:  time.Hour,
			}
		},
		log.NewNoopLogger(),
	)

	created := time.Now().Add(-50 * time.Minute)
	sess := &session.Session{
		ID: id.NewSessionID(), Token: "tok", CreatedAt: created,
		LastActivityAt: time.Now().Add(-2 * time.Minute),
	}

	router := forge.NewRouter()
	router.Use(injectSession(sess))
	router.Use(mw)
	require.NoError(t, router.GET("/test", func(ctx forge.Context) error { return ctx.NoContent(http.StatusOK) }))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, touchedExpiry.Equal(created.Add(time.Hour)), "the window stops at created_at + lifetime, got %v", touchedExpiry)
	assert.True(t, sess.ExpiresAt.Equal(created.Add(time.Hour)), "the context copy matches the row")
}
