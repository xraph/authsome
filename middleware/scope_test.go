package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// An OAuth2-issued session reaches a handler only through a route tagged
// with RequireScope, and only when it holds the scopes the route names.

func oauthMiddlewareFixture(t *testing.T, sess *session.Session) forge.Router {
	t.Helper()
	resolveSession := func(_ string) (*session.Session, error) { return sess, nil }
	resolveUser := func(_ string) (*user.User, error) {
		return &user.User{ID: sess.UserID, AppID: sess.AppID, Email: "oauth@example.com"}, nil
	}
	router := forge.NewRouter()
	router.Use(middleware.AuthMiddleware(resolveSession, resolveUser, log.NewNoopLogger()))
	whoami := func(ctx forge.Context) error {
		if _, ok := middleware.SessionFrom(ctx.Context()); !ok {
			return ctx.NoContent(http.StatusUnauthorized)
		}
		if _, ok := middleware.UserIDFrom(ctx.Context()); !ok {
			return ctx.NoContent(http.StatusUnauthorized)
		}
		return ctx.NoContent(http.StatusOK)
	}
	require.NoError(t, router.GET("/untagged", whoami))
	require.NoError(t, router.GET("/any", whoami, forge.WithMiddleware(middleware.RequireScope())))
	require.NoError(t, router.GET("/read", whoami, forge.WithMiddleware(middleware.RequireScope("read"))))
	require.NoError(t, router.GET("/admin", whoami, forge.WithMiddleware(middleware.RequireScope("read", "admin"))))
	return router
}

func get(router forge.Router, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRequireScope_ParksOAuthSessionsUntilARouteAdmitsThem(t *testing.T) {
	sess := &session.Session{ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: id.NewUserID(), Token: "tok", ClientID: "acme", Scopes: []string{"read"}}
	router := oauthMiddlewareFixture(t, sess)

	assert.Equal(t, http.StatusUnauthorized, get(router, "/untagged").Code, "an untagged route never sees an OAuth2 session")
	assert.Equal(t, http.StatusOK, get(router, "/any").Code, "a route that accepts any OAuth2 token admits it")
	assert.Equal(t, http.StatusOK, get(router, "/read").Code, "a held scope admits the session with its user")
	admin := get(router, "/admin")
	assert.Equal(t, http.StatusForbidden, admin.Code, "a missing scope is refused")
	assert.Contains(t, admin.Header().Get("WWW-Authenticate"), "insufficient_scope")
}

func TestRequireScope_OrdinarySessionsAreUntouched(t *testing.T) {
	sess := &session.Session{ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: id.NewUserID(), Token: "tok"}
	router := oauthMiddlewareFixture(t, sess)
	assert.Equal(t, http.StatusOK, get(router, "/untagged").Code, "a login session is installed as before")
	assert.Equal(t, http.StatusOK, get(router, "/admin").Code, "RequireScope does not gate login sessions; the route's own rules do")
}
