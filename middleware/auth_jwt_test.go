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
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Mock JWT validator
// ──────────────────────────────────────────────────

type mockJWTValidator struct {
	claims *tokenformat.TokenClaims
	err    error
}

func (m *mockJWTValidator) ValidateJWT(_ string) (*tokenformat.TokenClaims, error) {
	return m.claims, m.err
}

// ──────────────────────────────────────────────────
// JWT session checker tests (C1/C2 fix)
// ──────────────────────────────────────────────────

func TestJWTAuth_SessionChecker_Disabled_PassesThrough(t *testing.T) {
	testUserID := id.NewUserID()
	testAppID := id.NewAppID()
	testSessID := id.NewSessionID()

	validator := &mockJWTValidator{
		claims: &tokenformat.TokenClaims{
			UserID:    testUserID.String(),
			AppID:     testAppID.String(),
			SessionID: testSessID.String(),
		},
	}

	mw := middleware.AuthMiddlewareWithJWT(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("not found")
		},
		func(userIDStr string) (*user.User, error) {
			if userIDStr == testUserID.String() {
				return &user.User{ID: testUserID, AppID: testAppID, Email: "jwt@test.com"}, nil
			}
			return nil, errors.New("not found")
		},
		nil, // no strategy auth
		validator,
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{
			JWTSessionChecker: func(_ string) (*session.Session, error) {
				// nil, nil = feature disabled
				return nil, nil
			},
		},
	)

	var gotUser bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, gotUser = middleware.UserFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Use a token that looks like a JWT (two dots)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer header.payload.signature")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, gotUser, "JWT auth should pass through when session check is disabled")
}

func TestJWTAuth_SessionChecker_SessionNotFound_Rejects(t *testing.T) {
	testUserID := id.NewUserID()
	testAppID := id.NewAppID()
	testSessID := id.NewSessionID()

	validator := &mockJWTValidator{
		claims: &tokenformat.TokenClaims{
			UserID:    testUserID.String(),
			AppID:     testAppID.String(),
			SessionID: testSessID.String(),
		},
	}

	mw := middleware.AuthMiddlewareWithJWT(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("not found")
		},
		func(userIDStr string) (*user.User, error) {
			if userIDStr == testUserID.String() {
				return &user.User{ID: testUserID}, nil
			}
			return nil, errors.New("not found")
		},
		nil,
		validator,
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{
			JWTSessionChecker: func(_ string) (*session.Session, error) {
				// Session not found — revoked
				return nil, errors.New("session not found")
			},
		},
	)

	var gotUser bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, gotUser = middleware.UserFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer header.payload.signature")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code) // middleware passes through, RequireAuth would reject
	assert.False(t, gotUser, "JWT auth should fail when session is revoked")
}

func TestJWTAuth_SessionChecker_IPMismatch_Rejects(t *testing.T) {
	testUserID := id.NewUserID()
	testAppID := id.NewAppID()
	testSessID := id.NewSessionID()

	validator := &mockJWTValidator{
		claims: &tokenformat.TokenClaims{
			UserID:    testUserID.String(),
			AppID:     testAppID.String(),
			SessionID: testSessID.String(),
		},
	}

	mw := middleware.AuthMiddlewareWithJWT(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("not found")
		},
		func(_ string) (*user.User, error) {
			return &user.User{ID: testUserID}, nil
		},
		nil,
		validator,
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{
			BindToIP: true,
			JWTSessionChecker: func(_ string) (*session.Session, error) {
				return &session.Session{
					ID:        testSessID,
					IPAddress: "10.0.0.1",
				}, nil
			},
		},
	)

	var gotUser bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, gotUser = middleware.UserFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer header.payload.signature")
	req.RemoteAddr = "192.168.1.1:12345" // different IP
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, gotUser, "JWT auth should fail when IP doesn't match session")
}

func TestJWTAuth_SessionChecker_DeviceMismatch_Rejects(t *testing.T) {
	testUserID := id.NewUserID()
	testAppID := id.NewAppID()
	testSessID := id.NewSessionID()

	validator := &mockJWTValidator{
		claims: &tokenformat.TokenClaims{
			UserID:    testUserID.String(),
			AppID:     testAppID.String(),
			SessionID: testSessID.String(),
		},
	}

	mw := middleware.AuthMiddlewareWithJWT(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("not found")
		},
		func(_ string) (*user.User, error) {
			return &user.User{ID: testUserID}, nil
		},
		nil,
		validator,
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{
			BindToDevice: true,
			JWTSessionChecker: func(_ string) (*session.Session, error) {
				return &session.Session{
					ID:        testSessID,
					UserAgent: "OriginalBrowser/1.0",
				}, nil
			},
		},
	)

	var gotUser bool
	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		_, gotUser = middleware.UserFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer header.payload.signature")
	req.Header.Set("User-Agent", "DifferentBrowser/2.0")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, gotUser, "JWT auth should fail when device doesn't match session")
}

func TestJWTAuth_SessionChecker_Matches_Allows(t *testing.T) {
	testUserID := id.NewUserID()
	testAppID := id.NewAppID()
	testSessID := id.NewSessionID()

	validator := &mockJWTValidator{
		claims: &tokenformat.TokenClaims{
			UserID:    testUserID.String(),
			AppID:     testAppID.String(),
			SessionID: testSessID.String(),
		},
	}

	mw := middleware.AuthMiddlewareWithJWT(
		func(_ string) (*session.Session, error) {
			return nil, errors.New("not found")
		},
		func(userIDStr string) (*user.User, error) {
			if userIDStr == testUserID.String() {
				return &user.User{ID: testUserID, AppID: testAppID, Email: "jwt@test.com"}, nil
			}
			return nil, errors.New("not found")
		},
		nil,
		validator,
		log.NewNoopLogger(),
		middleware.SessionBindingConfig{
			BindToIP:     true,
			BindToDevice: true,
			JWTSessionChecker: func(_ string) (*session.Session, error) {
				return &session.Session{
					ID:        testSessID,
					IPAddress: "10.0.0.1",
					UserAgent: "TestBrowser/1.0",
				}, nil
			},
		},
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

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer header.payload.signature")
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.True(t, userOK, "JWT auth should succeed when session exists and binding matches")
	assert.Equal(t, "jwt@test.com", gotUser.Email)
	assert.True(t, methodOK)
	assert.Equal(t, "jwt", gotMethod)
}
