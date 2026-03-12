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
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Mock strategy authenticator
// ──────────────────────────────────────────────────

type mockStrategyAuth struct {
	authenticateFn func(ctx context.Context, r *http.Request) (*strategy.Result, error)
}

func (m *mockStrategyAuth) Authenticate(ctx context.Context, r *http.Request) (*strategy.Result, error) {
	if m.authenticateFn != nil {
		return m.authenticateFn(ctx, r)
	}
	return nil, strategy.NotApplicableError{}
}

// ──────────────────────────────────────────────────
// Test fixtures helper
// ──────────────────────────────────────────────────

func newTestFixtures() (id.UserID, id.AppID, id.SessionID, *session.Session, *user.User) {
	userID := id.NewUserID()
	appID := id.NewAppID()
	sessID := id.NewSessionID()

	sess := &session.Session{
		ID:     sessID,
		AppID:  appID,
		UserID: userID,
		Token:  "valid-token",
	}
	u := &user.User{
		ID:        userID,
		AppID:     appID,
		Email:     "test@example.com",
		FirstName: "Test User",
	}

	return userID, appID, sessID, sess, u
}

// ──────────────────────────────────────────────────
// AuthMiddlewareWithStrategies tests
// ──────────────────────────────────────────────────

func TestStrategyMiddleware_ValidBearerSession(t *testing.T) {
	testUserID, testAppID, testSessID, testSession, testUser := newTestFixtures()

	mw := middleware.AuthMiddlewareWithStrategies(
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
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				t.Fatal("strategy should not be called when session resolves")
				return nil, nil
			},
		},
		log.NewNoopLogger(),
	)

	var (
		gotUser   *user.User
		gotSessID id.SessionID
		gotUserID id.UserID
		gotAppID  id.AppID
		gotScope  forge.Scope
		gotMethod string
		userOK    bool
		sessIDOK  bool
		userIDOK  bool
		appIDOK   bool
		scopeOK   bool
		methodOK  bool
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotSessID, sessIDOK = middleware.SessionIDFrom(ctx.Context())
		gotUserID, userIDOK = middleware.UserIDFrom(ctx.Context())
		gotAppID, appIDOK = middleware.AppIDFrom(ctx.Context())
		gotScope, scopeOK = forge.ScopeFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.True(t, userOK, "user should be set in context")
	assert.Equal(t, testUser.Email, gotUser.Email)
	assert.True(t, sessIDOK)
	assert.Equal(t, testSessID, gotSessID)
	assert.True(t, userIDOK)
	assert.Equal(t, testUserID, gotUserID)
	assert.True(t, appIDOK)
	assert.Equal(t, testAppID, gotAppID)
	assert.True(t, scopeOK, "forge.Scope should be set")
	assert.Equal(t, testAppID.String(), gotScope.AppID())
	assert.True(t, methodOK, "auth method should be set")
	assert.Equal(t, "session", gotMethod)
}

func TestStrategyMiddleware_InvalidBearerFallsBackToStrategy(t *testing.T) {
	strategyUserID := id.NewUserID()
	strategyAppID := id.NewAppID()
	strategySessID := id.NewSessionID()

	strategyUser := &user.User{
		ID:        strategyUserID,
		AppID:     strategyAppID,
		Email:     "strategy@example.com",
		FirstName: "Strategy User",
	}
	strategySess := &session.Session{
		ID:     strategySessID,
		AppID:  strategyAppID,
		UserID: strategyUserID,
	}

	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("session not found")
		},
		func(_ string) (*user.User, error) {
			t.Fatal("user resolver should not be called for session path")
			return nil, nil
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return &strategy.Result{
					User:    strategyUser,
					Session: strategySess,
				}, nil
			},
		},
		log.NewNoopLogger(),
	)

	var (
		gotUser   *user.User
		gotMethod string
		userOK    bool
		methodOK  bool
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.True(t, userOK, "user should be set from strategy result")
	assert.Equal(t, strategyUser.Email, gotUser.Email)
	assert.Equal(t, strategyUserID, gotUser.ID)
	assert.True(t, methodOK)
	assert.Equal(t, "strategy", gotMethod)
}

func TestStrategyMiddleware_NoAuthHeaders(t *testing.T) {
	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			t.Fatal("session resolver should not be called")
			return nil, nil
		},
		func(_ string) (*user.User, error) {
			t.Fatal("user resolver should not be called")
			return nil, nil
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return nil, strategy.NotApplicableError{}
			},
		},
		log.NewNoopLogger(),
	)

	var (
		hasUser   bool
		hasMethod bool
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, hasUser = middleware.UserFrom(ctx.Context())
		_, hasMethod = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, hasUser, "no user should be in context without auth headers")
	assert.False(t, hasMethod, "no auth method should be set")
}

func TestStrategyMiddleware_APIKeyOnly(t *testing.T) {
	strategyUserID := id.NewUserID()
	strategyAppID := id.NewAppID()
	strategySessID := id.NewSessionID()

	strategyUser := &user.User{
		ID:        strategyUserID,
		AppID:     strategyAppID,
		Email:     "apikey@example.com",
		FirstName: "API Key User",
	}
	strategySess := &session.Session{
		ID:     strategySessID,
		AppID:  strategyAppID,
		UserID: strategyUserID,
	}

	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			t.Fatal("session resolver should not be called without bearer token")
			return nil, nil
		},
		func(_ string) (*user.User, error) {
			t.Fatal("user resolver should not be called")
			return nil, nil
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, r *http.Request) (*strategy.Result, error) {
				// Simulate strategy reading X-API-Key header.
				if r.Header.Get("X-API-Key") == "my-api-key" {
					return &strategy.Result{
						User:    strategyUser,
						Session: strategySess,
					}, nil
				}
				return nil, strategy.NotApplicableError{}
			},
		},
		log.NewNoopLogger(),
	)

	var (
		gotUser  *user.User
		gotAppID id.AppID
		userOK   bool
		appIDOK  bool
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotAppID, appIDOK = middleware.AppIDFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "my-api-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.True(t, userOK, "user should be set from strategy result")
	assert.Equal(t, strategyUser.Email, gotUser.Email)
	assert.True(t, appIDOK)
	assert.Equal(t, strategyAppID, gotAppID)
}

func TestStrategyMiddleware_BearerWithAskPrefix(t *testing.T) {
	strategyUserID := id.NewUserID()
	strategyAppID := id.NewAppID()
	strategySessID := id.NewSessionID()

	strategyUser := &user.User{
		ID:        strategyUserID,
		AppID:     strategyAppID,
		Email:     "askprefix@example.com",
		FirstName: "Ask Prefix User",
	}
	strategySess := &session.Session{
		ID:     strategySessID,
		AppID:  strategyAppID,
		UserID: strategyUserID,
	}

	sessionResolverCalled := false

	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			sessionResolverCalled = true
			return nil, errors.New("should not be called")
		},
		func(_ string) (*user.User, error) {
			return nil, errors.New("not found")
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return &strategy.Result{
					User:    strategyUser,
					Session: strategySess,
				}, nil
			},
		},
		log.NewNoopLogger(),
	)

	var (
		gotUser   *user.User
		gotMethod string
		userOK    bool
		methodOK  bool
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ask_some_api_key_value")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, sessionResolverCalled, "session resolver should be skipped for ask_ prefix tokens")
	require.True(t, userOK, "user should be set from strategy result")
	assert.Equal(t, strategyUser.Email, gotUser.Email)
	assert.True(t, methodOK)
	assert.Equal(t, "strategy", gotMethod)
}

func TestStrategyMiddleware_StrategyNotApplicable(t *testing.T) {
	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			t.Fatal("session resolver should not be called")
			return nil, nil
		},
		func(_ string) (*user.User, error) {
			t.Fatal("user resolver should not be called")
			return nil, nil
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return nil, strategy.NotApplicableError{}
			},
		},
		log.NewNoopLogger(),
	)

	var hasUser bool

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, hasUser = middleware.UserFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "some-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, hasUser, "user should not be set when strategy is not applicable")
}

func TestStrategyMiddleware_StrategyError(t *testing.T) {
	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			t.Fatal("session resolver should not be called")
			return nil, nil
		},
		func(_ string) (*user.User, error) {
			t.Fatal("user resolver should not be called")
			return nil, nil
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return nil, errors.New("database connection lost")
			},
		},
		log.NewNoopLogger(),
	)

	var hasUser bool

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, hasUser = middleware.UserFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "some-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, hasUser, "user should not be set when strategy returns an error")
}

func TestStrategyMiddleware_RequireAuth_NoAuth(t *testing.T) {
	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("invalid")
		},
		func(_ string) (*user.User, error) {
			return nil, errors.New("not found")
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return nil, strategy.NotApplicableError{}
			},
		},
		log.NewNoopLogger(),
	)

	var called bool

	router := forge.NewRouter()
	router.Use(mw)
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "handler should not be called without authentication")
	assert.Contains(t, rec.Body.String(), "authentication required")
}

func TestStrategyMiddleware_RequireAuth_WithStrategy(t *testing.T) {
	strategyUserID := id.NewUserID()
	strategyAppID := id.NewAppID()
	strategySessID := id.NewSessionID()

	strategyUser := &user.User{
		ID:        strategyUserID,
		AppID:     strategyAppID,
		Email:     "protected@example.com",
		FirstName: "Protected User",
	}
	strategySess := &session.Session{
		ID:     strategySessID,
		AppID:  strategyAppID,
		UserID: strategyUserID,
	}

	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("invalid")
		},
		func(_ string) (*user.User, error) {
			return nil, errors.New("not found")
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return &strategy.Result{
					User:    strategyUser,
					Session: strategySess,
				}, nil
			},
		},
		log.NewNoopLogger(),
	)

	var called bool

	router := forge.NewRouter()
	router.Use(mw)
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", "my-api-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "handler should be called with valid strategy auth")
}

func TestStrategyMiddleware_StrategyResultSetsScope(t *testing.T) {
	strategyUserID := id.NewUserID()
	strategyAppID := id.NewAppID()
	strategyOrgID := id.NewOrgID()
	strategySessID := id.NewSessionID()

	strategyUser := &user.User{
		ID:        strategyUserID,
		AppID:     strategyAppID,
		Email:     "org-scope@example.com",
		FirstName: "Org Scope User",
	}
	strategySess := &session.Session{
		ID:     strategySessID,
		AppID:  strategyAppID,
		UserID: strategyUserID,
		OrgID:  strategyOrgID,
	}

	mw := middleware.AuthMiddlewareWithStrategies(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("invalid")
		},
		func(_ string) (*user.User, error) {
			return nil, errors.New("not found")
		},
		&mockStrategyAuth{
			authenticateFn: func(_ context.Context, _ *http.Request) (*strategy.Result, error) {
				return &strategy.Result{
					User:    strategyUser,
					Session: strategySess,
				}, nil
			},
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

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "org-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// forge.Scope should be set with org-level scope.
	require.True(t, scopeOK, "forge.Scope should be set")
	assert.Equal(t, strategyAppID.String(), gotScope.AppID())
	assert.Equal(t, strategyOrgID.String(), gotScope.OrgID())

	// Traditional OrgID should also be set.
	require.True(t, orgIDOK, "OrgID should be set from strategy session")
	assert.Equal(t, strategyOrgID, gotOrgID)
}
