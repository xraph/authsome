package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

func TestAuthMiddlewareSetsRequestInfoForAnonymousRequests(t *testing.T) {
	mw := middleware.AuthMiddleware(
		func(string) (*session.Session, error) { return nil, errors.New("no session") },
		func(string) (*user.User, error) { return nil, errors.New("no user") },
		log.NewNoopLogger(),
	)

	var seen hook.RequestInfo
	var ok bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		seen, ok = middleware.RequestInfoFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.RemoteAddr = "203.0.113.9:5555"
	req.Header.Set("User-Agent", "curl/8.0")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, ok, "request info must be set even for anonymous requests")
	assert.Equal(t, "203.0.113.9", seen.IP)
	assert.Equal(t, "curl/8.0", seen.UserAgent)
	assert.Empty(t, seen.SessionID)
}

func TestAuthMiddlewareAddsSessionToRequestInfo(t *testing.T) {
	userID := id.NewUserID()
	appID := id.NewAppID()
	sessID := id.NewSessionID()
	sess := &session.Session{ID: sessID, AppID: appID, UserID: userID, Token: "valid-token"}
	u := &user.User{ID: userID, AppID: appID, Email: "a@example.com"}

	mw := middleware.AuthMiddleware(
		func(token string) (*session.Session, error) {
			if token == "valid-token" {
				return sess, nil
			}
			return nil, errors.New("invalid")
		},
		func(string) (*user.User, error) { return u, nil },
		log.NewNoopLogger(),
	)

	var seen hook.RequestInfo
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		seen, _ = middleware.RequestInfoFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, sessID.String(), seen.SessionID)
}
