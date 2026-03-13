package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Context helpers tests
// ──────────────────────────────────────────────────

func TestWithUser_UserFrom(t *testing.T) {
	u := &user.User{
		ID:        id.NewUserID(),
		Email:     "test@example.com",
		FirstName: "Test User",
	}

	ctx := middleware.WithUser(context.Background(), u)
	got, ok := middleware.UserFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, u.Email, got.Email)
}

func TestUserFrom_Missing(t *testing.T) {
	_, ok := middleware.UserFrom(context.Background())
	assert.False(t, ok)
}

func TestWithSession_SessionFrom(t *testing.T) {
	sess := &session.Session{
		ID:    id.NewSessionID(),
		Token: "test-token",
	}

	ctx := middleware.WithSession(context.Background(), sess)
	got, ok := middleware.SessionFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, sess.ID, got.ID)
	assert.Equal(t, sess.Token, got.Token)
}

func TestSessionFrom_Missing(t *testing.T) {
	_, ok := middleware.SessionFrom(context.Background())
	assert.False(t, ok)
}

func TestWithAppID_AppIDFrom(t *testing.T) {
	appID := id.NewAppID()
	ctx := middleware.WithAppID(context.Background(), appID)
	got, ok := middleware.AppIDFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, appID, got)
}

func TestAppIDFrom_Missing(t *testing.T) {
	_, ok := middleware.AppIDFrom(context.Background())
	assert.False(t, ok)
}

func TestWithOrgID_OrgIDFrom(t *testing.T) {
	orgID := id.NewOrgID()
	ctx := middleware.WithOrgID(context.Background(), orgID)
	got, ok := middleware.OrgIDFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, orgID, got)
}

func TestOrgIDFrom_Missing(t *testing.T) {
	_, ok := middleware.OrgIDFrom(context.Background())
	assert.False(t, ok)
}

func TestWithUserID_UserIDFrom(t *testing.T) {
	userID := id.NewUserID()
	ctx := middleware.WithUserID(context.Background(), userID)
	got, ok := middleware.UserIDFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, userID, got)
}

func TestUserIDFrom_Missing(t *testing.T) {
	_, ok := middleware.UserIDFrom(context.Background())
	assert.False(t, ok)
}

func TestWithSessionID_SessionIDFrom(t *testing.T) {
	sessID := id.NewSessionID()
	ctx := middleware.WithSessionID(context.Background(), sessID)
	got, ok := middleware.SessionIDFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, sessID, got)
}

func TestSessionIDFrom_Missing(t *testing.T) {
	_, ok := middleware.SessionIDFrom(context.Background())
	assert.False(t, ok)
}

// ──────────────────────────────────────────────────
// AuthMiddleware tests
// ──────────────────────────────────────────────────

func TestAuthMiddleware_NoToken(t *testing.T) {
	mw := middleware.AuthMiddleware(
		func(_ string) (*session.Session, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
		func(_ string) (*user.User, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
		log.NewNoopLogger(),
	)

	var gotUser bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, gotUser = middleware.UserFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, gotUser, "no user should be in context without token")
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	testUserID := id.NewUserID()
	testAppID := id.NewAppID()
	testSessID := id.NewSessionID()

	testSession := &session.Session{
		ID:     testSessID,
		AppID:  testAppID,
		UserID: testUserID,
		Token:  "valid-token",
	}
	testUser := &user.User{
		ID:        testUserID,
		AppID:     testAppID,
		Email:     "test@test.com",
		FirstName: "Test",
	}

	mw := middleware.AuthMiddleware(
		func(token string) (*session.Session, error) {
			if token == "valid-token" {
				return testSession, nil
			}
			return nil, errors.New("invalid")
		},
		func(userIDStr string) (*user.User, error) {
			if userIDStr == testUserID.String() {
				return testUser, nil
			}
			return nil, errors.New("not found")
		},
		log.NewNoopLogger(),
	)

	var (
		gotUser   *user.User
		gotSessID id.SessionID
		gotUserID id.UserID
		gotAppID  id.AppID
		gotScope  forge.Scope
		userOK    bool
		sessIDOK  bool
		userIDOK  bool
		appIDOK   bool
		scopeOK   bool
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotSessID, sessIDOK = middleware.SessionIDFrom(ctx.Context())
		gotUserID, userIDOK = middleware.UserIDFrom(ctx.Context())
		gotAppID, appIDOK = middleware.AppIDFrom(ctx.Context())
		gotScope, scopeOK = forge.ScopeFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, userOK)
	assert.True(t, sessIDOK)
	assert.True(t, userIDOK)
	assert.True(t, appIDOK)
	assert.Equal(t, testUser.Email, gotUser.Email)
	assert.Equal(t, testSessID, gotSessID)
	assert.Equal(t, testUserID, gotUserID)
	assert.Equal(t, testAppID, gotAppID)

	// forge.Scope should be set with app-level scope (no org)
	assert.True(t, scopeOK, "forge.Scope should be set")
	assert.Equal(t, testAppID.String(), gotScope.AppID())
	assert.Equal(t, "", gotScope.OrgID())
}

func TestAuthMiddleware_ValidToken_OrgScope(t *testing.T) {
	testUserID := id.NewUserID()
	testAppID := id.NewAppID()
	testOrgID := id.NewOrgID()
	testSessID := id.NewSessionID()

	testSession := &session.Session{
		ID:     testSessID,
		AppID:  testAppID,
		UserID: testUserID,
		OrgID:  testOrgID,
		Token:  "org-token",
	}
	testUser := &user.User{
		ID:        testUserID,
		AppID:     testAppID,
		Email:     "org@test.com",
		FirstName: "Org User",
	}

	mw := middleware.AuthMiddleware(
		func(token string) (*session.Session, error) {
			if token == "org-token" {
				return testSession, nil
			}
			return nil, errors.New("invalid")
		},
		func(userIDStr string) (*user.User, error) {
			if userIDStr == testUserID.String() {
				return testUser, nil
			}
			return nil, errors.New("not found")
		},
		log.NewNoopLogger(),
	)

	var (
		gotScope forge.Scope
		gotOrgID id.OrgID
		scopeOK  bool
		orgIDOK  bool
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotScope, scopeOK = forge.ScopeFrom(ctx.Context())
		gotOrgID, orgIDOK = middleware.OrgIDFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer org-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// forge.Scope should be set with org-level scope
	assert.True(t, scopeOK, "forge.Scope should be set")
	assert.Equal(t, testAppID.String(), gotScope.AppID())
	assert.Equal(t, testOrgID.String(), gotScope.OrgID())

	// Traditional OrgID should also be set
	assert.True(t, orgIDOK)
	assert.Equal(t, testOrgID, gotOrgID)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	mw := middleware.AuthMiddleware(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("session not found")
		},
		func(_ string) (*user.User, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
		log.NewNoopLogger(),
	)

	var gotUser bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, gotUser = middleware.UserFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Should still call next handler, just without user context
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, gotUser)
}

func TestAuthMiddleware_NonBearerAuth(t *testing.T) {
	mw := middleware.AuthMiddleware(
		func(_ string) (*session.Session, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
		func(_ string) (*user.User, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
		log.NewNoopLogger(),
	)

	var called bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
}

func TestAuthMiddleware_UserResolveFails(t *testing.T) {
	testUserID := id.NewUserID()
	testAppID := id.NewAppID()

	testSession := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  testAppID,
		UserID: testUserID,
		Token:  "valid-token",
	}

	mw := middleware.AuthMiddleware(
		func(_ string) (*session.Session, error) {
			return testSession, nil
		},
		func(_ string) (*user.User, error) {
			return nil, errors.New("user not found")
		},
		log.NewNoopLogger(),
	)

	var (
		hasUser   bool
		hasSessID bool
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, hasUser = middleware.UserFrom(ctx.Context())
		_, hasSessID = middleware.SessionIDFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, hasUser, "user should not be set when resolve fails")
	assert.True(t, hasSessID, "session ID should still be set")
}

// ──────────────────────────────────────────────────
// RequireAuth tests
// ──────────────────────────────────────────────────

func TestRequireAuth_WithUser(t *testing.T) {
	var called bool

	router := forge.NewRouter()
	// Inject user into context before RequireAuth checks
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			goCtx := middleware.WithUser(ctx.Context(), &user.User{ID: id.NewUserID()})
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	})
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
}

func TestRequireAuth_WithoutUser(t *testing.T) {
	var called bool

	router := forge.NewRouter()
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "handler should not be called without authentication")
	assert.Contains(t, rec.Body.String(), "authentication required")
}
