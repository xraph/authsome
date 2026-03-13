package authsome_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// ──────────────────────────────────────────────────
// E2E Multi-Auth test helpers
// ──────────────────────────────────────────────────

// e2eEngineWithAPIKey creates an engine with the API key plugin registered.
// The memory.Store satisfies both store.Store and apikey.Store interfaces.
func e2eEngineWithAPIKey(t *testing.T) (*authsome.Engine, *memory.Store) {
	t.Helper()
	s := memory.New()
	akPlugin := apikeyPlugin.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID("aapp_01jf0000000000000000000000"),
		authsome.WithPlugin(akPlugin),
	)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })

	return eng, s
}

// e2eRouter sets up a forge router with the layered auth middleware.
func e2eRouter(eng *authsome.Engine) forge.Router {
	router := forge.NewRouter()
	mw := middleware.AuthMiddlewareWithStrategies(
		eng.ResolveSessionByToken,
		eng.ResolveUser,
		eng.Strategies(),
		eng.Logger(),
	)
	router.Use(mw)
	return router
}

// e2eCreateAPIKey creates an API key for a user in the store and returns the raw key and key entity.
func e2eCreateAPIKey(t *testing.T, store *memory.Store, appID id.AppID, userID id.UserID) (string, *apikey.APIKey) {
	t.Helper()
	raw, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)

	now := time.Now()
	key := &apikey.APIKey{
		ID:        id.NewAPIKeyID(),
		AppID:     appID,
		UserID:    userID,
		Name:      "test-key",
		KeyHash:   hash,
		KeyPrefix: prefix,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = store.CreateAPIKey(context.Background(), key)
	require.NoError(t, err)

	return raw, key
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — Bearer Session Auth
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_BearerSessionAuth(t *testing.T) {
	eng, _ := e2eEngineWithAPIKey(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Sign up a user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "session-auth@example.com",
		Password:  "SecureP@ss1",
		FirstName: "SessionUser",
	})
	require.NoError(t, err)

	// Step 2: Sign in to get a session token
	_, sess, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "session-auth@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, sess.Token)

	// Step 3: Set up router with layered middleware
	router := e2eRouter(eng)

	var (
		gotUser   *user.User
		gotMethod string
		userOK    bool
		methodOK  bool
	)

	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Step 4: Send request with session token in Bearer header
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Step 5: Verify
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, userOK, "user should be in context")
	assert.Equal(t, u.ID, gotUser.ID)
	assert.Equal(t, "session-auth@example.com", gotUser.Email)
	assert.True(t, methodOK, "auth method should be set")
	assert.Equal(t, "session", gotMethod)
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — API Key Auth
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_APIKeyAuth(t *testing.T) {
	eng, store := e2eEngineWithAPIKey(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Sign up a user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "apikey-auth@example.com",
		Password:  "SecureP@ss1",
		FirstName: "APIKeyUser",
	})
	require.NoError(t, err)

	// Step 2: Create API key in the store
	rawKey, _ := e2eCreateAPIKey(t, store, appID, u.ID)

	// Step 3: Set up router
	router := e2eRouter(eng)

	var (
		gotUser   *user.User
		gotMethod string
		userOK    bool
		methodOK  bool
	)

	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Step 4: Send request with API key in X-API-Key header
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-App-ID", appID.String())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Step 5: Verify
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, userOK, "user should be in context via API key strategy")
	assert.Equal(t, u.ID, gotUser.ID)
	assert.Equal(t, "apikey-auth@example.com", gotUser.Email)
	assert.True(t, methodOK, "auth method should be set")
	assert.Equal(t, "strategy", gotMethod)
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — Bearer Fails, API Key Succeeds
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_BearerFailsAPIKeySucceeds(t *testing.T) {
	eng, store := e2eEngineWithAPIKey(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Sign up a user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "fallback-auth@example.com",
		Password:  "SecureP@ss1",
		FirstName: "FallbackUser",
	})
	require.NoError(t, err)

	// Step 2: Create API key
	rawKey, _ := e2eCreateAPIKey(t, store, appID, u.ID)

	// Step 3: Set up router
	router := e2eRouter(eng)

	var (
		gotUser   *user.User
		gotMethod string
		userOK    bool
		methodOK  bool
	)

	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Step 4: Send request with invalid session token in Bearer + valid API key in X-API-Key
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-session-token-abc123")
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-App-ID", appID.String())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Step 5: Verify — API key strategy should have resolved the identity
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, userOK, "user should be in context via API key fallback")
	assert.Equal(t, u.ID, gotUser.ID)
	assert.True(t, methodOK, "auth method should be set")
	assert.Equal(t, "strategy", gotMethod)
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — Both Fail (unauthenticated)
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_BothFail(t *testing.T) {
	eng, _ := e2eEngineWithAPIKey(t)

	// Set up router
	router := e2eRouter(eng)

	var (
		userOK   bool
		methodOK bool
	)

	router.GET("/test", func(ctx forge.Context) error {
		_, userOK = middleware.UserFrom(ctx.Context())
		_, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Send request with invalid session token and no API key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-session-token-xyz")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Request should pass through unauthenticated (middleware does not block)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, userOK, "no user should be in context")
	assert.False(t, methodOK, "no auth method should be set")
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — RequireAuth Blocks Unauthenticated
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_RequireAuthBlocksUnauthenticated(t *testing.T) {
	eng, _ := e2eEngineWithAPIKey(t)

	// Set up router with layered auth + RequireAuth
	router := e2eRouter(eng)
	router.Use(middleware.RequireAuth())

	var called bool
	router.GET("/protected", func(ctx forge.Context) error {
		called = true
		return ctx.NoContent(http.StatusOK)
	})

	// Send request with no valid auth headers
	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Should be rejected with 401
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "handler should not be called without authentication")
	assert.Contains(t, rec.Body.String(), "authentication required")
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — API Key In X-API-Key Header
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_APIKeyInXAPIKeyHeader(t *testing.T) {
	eng, store := e2eEngineWithAPIKey(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Sign up a user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "xapikey-user@example.com",
		Password:  "SecureP@ss1",
		FirstName: "XAPIKeyUser",
	})
	require.NoError(t, err)

	// Step 2: Create API key
	rawKey, _ := e2eCreateAPIKey(t, store, appID, u.ID)

	// Step 3: Set up router
	router := e2eRouter(eng)

	var (
		gotUser   *user.User
		gotMethod string
		userOK    bool
		methodOK  bool
	)

	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Step 4: Send request with API key via X-API-Key header (not Bearer)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-App-ID", appID.String())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Step 5: Verify
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, userOK, "user should be in context via X-API-Key header")
	assert.Equal(t, u.ID, gotUser.ID)
	assert.Equal(t, "xapikey-user@example.com", gotUser.Email)
	assert.True(t, methodOK, "auth method should be set")
	assert.Equal(t, "strategy", gotMethod)
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — API Key With ask_ Prefix In Bearer
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_APIKeyWithAskPrefixInBearer(t *testing.T) {
	eng, store := e2eEngineWithAPIKey(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Sign up a user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "bearer-ask@example.com",
		Password:  "SecureP@ss1",
		FirstName: "BearerAskUser",
	})
	require.NoError(t, err)

	// Step 2: Create API key (raw key starts with "ask_")
	rawKey, _ := e2eCreateAPIKey(t, store, appID, u.ID)
	require.True(t, len(rawKey) > 4 && rawKey[:4] == "ask_", "raw key must start with ask_")

	// Step 3: Set up router
	router := e2eRouter(eng)

	var (
		gotUser   *user.User
		gotMethod string
		userOK    bool
		methodOK  bool
	)

	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Step 4: Send request with ask_ key in Authorization: Bearer header
	// The middleware should skip session resolution and go directly to strategy chain
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("X-App-ID", appID.String())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Step 5: Verify — routes to API key strategy (skips session)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, userOK, "user should be in context via API key strategy")
	assert.Equal(t, u.ID, gotUser.ID)
	assert.Equal(t, "bearer-ask@example.com", gotUser.Email)
	assert.True(t, methodOK, "auth method should be set")
	assert.Equal(t, "strategy", gotMethod)
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — Revoked API Key Rejected
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_RevokedAPIKeyRejected(t *testing.T) {
	eng, store := e2eEngineWithAPIKey(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Sign up a user
	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "revoked-key@example.com",
		Password:  "SecureP@ss1",
		FirstName: "RevokedKeyUser",
	})
	require.NoError(t, err)

	// Step 2: Create API key
	rawKey, key := e2eCreateAPIKey(t, store, appID, e2eResolveUserID(t, eng, "revoked-key@example.com", appID))

	// Step 3: Revoke the key
	key.Revoked = true
	key.UpdatedAt = time.Now()
	err = store.UpdateAPIKey(ctx, key)
	require.NoError(t, err)

	// Step 4: Set up router
	router := e2eRouter(eng)

	var (
		userOK   bool
		methodOK bool
	)

	router.GET("/test", func(ctx forge.Context) error {
		_, userOK = middleware.UserFrom(ctx.Context())
		_, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Step 5: Send request with the revoked API key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-App-ID", appID.String())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Step 6: Verify — strategy returns error, falls through to unauthenticated
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, userOK, "no user should be in context with revoked key")
	assert.False(t, methodOK, "no auth method should be set with revoked key")
}

// e2eResolveUserID is a helper that signs in and returns the user's ID.
func e2eResolveUserID(t *testing.T, eng *authsome.Engine, email string, appID id.AppID) id.UserID {
	t.Helper()
	u, _, err := eng.SignIn(context.Background(), &account.SignInRequest{
		AppID:    appID,
		Email:    email,
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	return u.ID
}

// ──────────────────────────────────────────────────
// E2E: Multi-Auth — Context Values Correct
// ──────────────────────────────────────────────────

func TestE2E_MultiAuth_ContextValuesCorrect(t *testing.T) {
	eng, store := e2eEngineWithAPIKey(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Sign up a user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "context-check@example.com",
		Password:  "SecureP@ss1",
		FirstName: "ContextUser",
	})
	require.NoError(t, err)

	// Step 2: Create API key
	rawKey, _ := e2eCreateAPIKey(t, store, appID, u.ID)

	// Step 3: Set up router
	router := e2eRouter(eng)

	var (
		gotUser      *user.User
		gotSession   *session.Session
		gotAppID     id.AppID
		gotUserID    id.UserID
		gotSessionID id.SessionID
		gotMethod    string
		userOK       bool
		sessionOK    bool
		appIDOK      bool
		userIDOK     bool
		sessionIDOK  bool
		methodOK     bool
	)

	router.GET("/test", func(ctx forge.Context) error {
		gotUser, userOK = middleware.UserFrom(ctx.Context())
		gotSession, sessionOK = middleware.SessionFrom(ctx.Context())
		gotAppID, appIDOK = middleware.AppIDFrom(ctx.Context())
		gotUserID, userIDOK = middleware.UserIDFrom(ctx.Context())
		gotSessionID, sessionIDOK = middleware.SessionIDFrom(ctx.Context())
		gotMethod, methodOK = middleware.AuthMethodFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	// Step 4: Send request with API key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", rawKey)
	req.Header.Set("X-App-ID", appID.String())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Step 5: Verify all context values
	assert.Equal(t, http.StatusOK, rec.Code)

	// User should be set
	assert.True(t, userOK, "user should be in context")
	assert.Equal(t, u.ID, gotUser.ID)
	assert.Equal(t, "context-check@example.com", gotUser.Email)
	assert.Equal(t, "ContextUser", gotUser.FirstName)

	// AppID should match
	assert.True(t, appIDOK, "app ID should be in context")
	assert.Equal(t, appID, gotAppID)

	// UserID should match the user
	assert.True(t, userIDOK, "user ID should be in context")
	assert.Equal(t, u.ID, gotUserID)

	// Session should be set (synthetic session from API key strategy)
	assert.True(t, sessionOK, "session should be in context (synthetic)")
	assert.False(t, gotSession.ID.IsNil(), "synthetic session should have a non-nil ID")
	assert.Equal(t, appID, gotSession.AppID)
	assert.Equal(t, u.ID, gotSession.UserID)

	// SessionID should be set
	assert.True(t, sessionIDOK, "session ID should be in context")
	assert.Equal(t, gotSession.ID, gotSessionID)

	// Auth method should be "strategy"
	assert.True(t, methodOK, "auth method should be set")
	assert.Equal(t, "strategy", gotMethod)
}
