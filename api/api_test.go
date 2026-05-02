package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// newAPIWithRouter builds an API + Forge router for tests that drive the full
// request pipeline (rather than calling handlers directly). Returns the
// router's http.Handler so callers can ServeHTTP against it.
func newAPIWithRouter(t *testing.T, eng *authsome.Engine) http.Handler {
	t.Helper()
	rootRouter := forge.NewRouter()
	a := api.New(eng, rootRouter)
	require.NoError(t, a.RegisterRoutes(rootRouter))
	return rootRouter.Handler()
}

// testAppIDStr is a valid TypeID string for tests.
const testAppIDStr = "aapp_01jf0000000000000000000000"

func newTestAPI(t *testing.T) (*api.API, *authsome.Engine) {
	t.Helper()
	s := memory.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	a := api.New(eng)
	return a, eng
}

func signUp(t *testing.T, eng *authsome.Engine, email, password string) (*json.RawMessage, string, string) { //nolint:unparam // test helper with configurable password
	t.Helper()
	ctx := context.Background()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	u, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  password,
		FirstName: "Test User",
	})
	require.NoError(t, err)

	raw, _ := json.Marshal(u)
	rm := json.RawMessage(raw)
	return &rm, sess.Token, sess.RefreshToken
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// ──────────────────────────────────────────────────
// Manifest endpoint
// ──────────────────────────────────────────────────

func TestHandleManifest(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/.well-known/authsome/manifest", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "0.5.0", body["version"])
	assert.NotEmpty(t, body["base_path"])
	assert.NotEmpty(t, body["endpoints"])
}

// TestHandleManifest_GroupedMount verifies that the well-known
// manifest is reachable under the grouped router's basepath in
// addition to the root mount. This mirrors the production wiring
// in extension.Register where api.New is given the root router and
// RegisterRoutes is called with a /authsome-prefixed group; SDK
// clients whose baseURL includes the mount prefix (e.g.
// http://host:7902/authsome) need the manifest at
// <baseURL>/.well-known/authsome/manifest, which is the grouped
// path. Without the mirror, every SDK platform-app-id discovery
// 404s in production but passes in this file's existing
// TestHandleManifest because that test doesn't exercise grouping.
func TestHandleManifest_GroupedMount(t *testing.T) {
	s := memory.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))

	rootRouter := forge.NewRouter()
	a := api.New(eng, rootRouter)
	groupedRouter := rootRouter.Group("/authsome")
	require.NoError(t, a.RegisterRoutes(groupedRouter))

	handler := rootRouter.Handler()

	// Grouped path: what SDK clients hit when baseURL=http://.../authsome.
	// This is THE regression case — without the api.go mirror, this 404s
	// in production and every SDK lazy platform-app-id discovery silently
	// fails, leading to 401s on every admin call from a service-account
	// authclient. We assert 200 + manifest shape (version/base_path
	// /endpoints); platform_app_id population is exercised by bootstrap
	// tests, not here.
	t.Run("grouped /authsome/.well-known/authsome/manifest", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/authsome/.well-known/authsome/manifest", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "manifest should be reachable under grouped mount; body=%s", rec.Body.String())
		var body map[string]any
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.NotEmpty(t, body["version"], "manifest should carry version field")
		assert.NotEmpty(t, body["endpoints"], "manifest should carry endpoints array")
	})

	// Root path: still works, browsers/operators/IdPs depend on this.
	t.Run("root /.well-known/authsome/manifest", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/.well-known/authsome/manifest", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "manifest should still be reachable at root; body=%s", rec.Body.String())
	})
}

// ──────────────────────────────────────────────────
// Health endpoint
// ──────────────────────────────────────────────────

func TestHandleHealth(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "healthy", body["status"])
}

// ──────────────────────────────────────────────────
// SignUp endpoint
// ──────────────────────────────────────────────────

func TestHandleSignUp_Success(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	body := jsonBody(t, map[string]string{
		"email":    "signup@test.com",
		"password": "SecureP@ss1",
		"name":     "Sign Up User",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["user"])
	assert.NotEmpty(t, resp["session_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

func TestHandleSignUp_InvalidBody(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signup", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSignUp_WeakPassword(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	body := jsonBody(t, map[string]string{
		"email":    "weak@test.com",
		"password": "short",
		"name":     "Weak",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSignUp_DuplicateEmail(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	// Pre-create user
	signUp(t, eng, "dupe@test.com", "SecureP@ss1")

	body := jsonBody(t, map[string]string{
		"email":    "dupe@test.com",
		"password": "SecureP@ss1",
		"name":     "Dupe",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// ──────────────────────────────────────────────────
// SignIn endpoint
// ──────────────────────────────────────────────────

func TestHandleSignIn_Success(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	signUp(t, eng, "signin@test.com", "SecureP@ss1")

	body := jsonBody(t, map[string]string{
		"email":    "signin@test.com",
		"password": "SecureP@ss1",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signin", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["user"])
	assert.NotEmpty(t, resp["session_token"])
}

func TestHandleSignIn_WrongPassword(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	signUp(t, eng, "wrong@test.com", "SecureP@ss1")

	body := jsonBody(t, map[string]string{
		"email":    "wrong@test.com",
		"password": "WrongPassword1",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signin", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ──────────────────────────────────────────────────
// SignOut endpoint
// ──────────────────────────────────────────────────

func TestHandleSignOut_Success(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	_, token, _ := signUp(t, eng, "signout@test.com", "SecureP@ss1")

	// Resolve the session to get the session ID
	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signout", nil)
	// Put session ID in context (normally done by auth middleware)
	ctx := middleware.WithSessionID(req.Context(), sess.ID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleSignOut_Unauthenticated(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signout", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ──────────────────────────────────────────────────
// Refresh endpoint
// ──────────────────────────────────────────────────

func TestHandleRefresh_Success(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	_, _, refreshToken := signUp(t, eng, "refresh@test.com", "SecureP@ss1")

	body := jsonBody(t, map[string]string{
		"refresh_token": refreshToken,
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["session_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

func TestHandleRefresh_MissingToken(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	body := jsonBody(t, map[string]string{})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ──────────────────────────────────────────────────
// GetMe endpoint
// ──────────────────────────────────────────────────

func TestHandleGetMe_Success(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	_, token, _ := signUp(t, eng, "me@test.com", "SecureP@ss1")

	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/me", nil)
	ctx := middleware.WithUserID(req.Context(), sess.UserID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "me@test.com", resp["email"])
}

func TestHandleGetMe_Unauthenticated(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ──────────────────────────────────────────────────
// UpdateMe endpoint
// ──────────────────────────────────────────────────

func TestHandleUpdateMe_Success(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	_, token, _ := signUp(t, eng, "update@test.com", "SecureP@ss1")

	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{
		"first_name": "Updated Name",
	})

	req := httptest.NewRequestWithContext(context.Background(), "PATCH", "/v1/me", body)
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.WithUserID(req.Context(), sess.UserID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", resp["first_name"])
}

// ──────────────────────────────────────────────────
// Sessions endpoint
// ──────────────────────────────────────────────────

func TestHandleListSessions_Success(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	_, token, _ := signUp(t, eng, "sessions@test.com", "SecureP@ss1")

	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/sessions", nil)
	ctx := middleware.WithUserID(req.Context(), sess.UserID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	sessions, ok := resp["sessions"].([]any)
	require.True(t, ok)
	assert.Len(t, sessions, 1)
}

func TestHandleListSessions_Unauthenticated(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ──────────────────────────────────────────────────
// RevokeSession endpoint
// ──────────────────────────────────────────────────

func TestHandleRevokeSession_Success(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	_, token, _ := signUp(t, eng, "revoke@test.com", "SecureP@ss1")

	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/sessions/"+sess.ID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleRevokeSession_InvalidID(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/sessions/invalid-id", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ──────────────────────────────────────────────────
// Error response helpers
// ──────────────────────────────────────────────────

func TestWriteAccountError_EmailTaken(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	signUp(t, eng, "taken@test.com", "SecureP@ss1")

	body := jsonBody(t, map[string]string{
		"email":    "taken@test.com",
		"password": "SecureP@ss1",
		"name":     "Taken",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "email already taken", resp["error"])
}

func TestIntrospect_APIKey_ValidSecretKey(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)
	ctx := context.Background()

	publicKey, secretKey, secretHash, publicPrefix, secretPrefix, err := apikey.GenerateKeyPair()
	require.NoError(t, err)

	appID, _ := id.ParseAppID(testAppIDStr)
	store := eng.APIKeyStore()
	require.NotNil(t, store, "engine must expose APIKeyStore for this test")

	keyRow := &apikey.APIKey{
		ID:              id.NewAPIKeyID(),
		AppID:           appID,
		UserID:          id.NewUserID(),
		Name:            "test key",
		KeyHash:         secretHash,
		KeyPrefix:       secretPrefix,
		PublicKey:       publicKey,
		PublicKeyPrefix: publicPrefix,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	require.NoError(t, store.CreateAPIKey(ctx, keyRow))

	body, _ := json.Marshal(map[string]string{"token": secretKey})
	req := httptest.NewRequest(http.MethodPost, "/v1/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, true, resp["active"], "secret API key must introspect as active")
	assert.Equal(t, keyRow.AppID.String(), resp["app_id"])
	assert.Equal(t, keyRow.UserID.String(), resp["user_id"])
}

func TestIntrospect_APIKey_PublicKeyRejected(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	publicKey, _, _, _, _, err := apikey.GenerateKeyPair()
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"token": publicKey})
	req := httptest.NewRequest(http.MethodPost, "/v1/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Public keys are not authentication tokens. Must report inactive.
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, false, resp["active"], "public key must NOT introspect as active")
}

func TestIntrospect_APIKey_RevokedReturnsInactive(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)
	ctx := context.Background()

	_, secretKey, secretHash, _, secretPrefix, err := apikey.GenerateKeyPair()
	require.NoError(t, err)
	appID, _ := id.ParseAppID(testAppIDStr)

	keyRow := &apikey.APIKey{
		ID:        id.NewAPIKeyID(),
		AppID:     appID,
		UserID:    id.NewUserID(),
		Name:      "revoked key",
		KeyHash:   secretHash,
		KeyPrefix: secretPrefix,
		Revoked:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, eng.APIKeyStore().CreateAPIKey(ctx, keyRow))

	body, _ := json.Marshal(map[string]string{"token": secretKey})
	req := httptest.NewRequest(http.MethodPost, "/v1/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, false, resp["active"], "revoked key must NOT introspect as active")
}
