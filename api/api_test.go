package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

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
	a := api.New(eng)
	return a, eng
}

func signUp(t *testing.T, eng *authsome.Engine, email, password string) (*json.RawMessage, string, string) {
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

	req := httptest.NewRequest("GET", "/.well-known/authsome/manifest", nil)
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

// ──────────────────────────────────────────────────
// Health endpoint
// ──────────────────────────────────────────────────

func TestHandleHealth(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequest("GET", "/v1/auth/health", nil)
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

	req := httptest.NewRequest("POST", "/v1/auth/signup", body)
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

	req := httptest.NewRequest("POST", "/v1/auth/signup", bytes.NewBufferString("not json"))
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

	req := httptest.NewRequest("POST", "/v1/auth/signup", body)
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

	req := httptest.NewRequest("POST", "/v1/auth/signup", body)
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

	req := httptest.NewRequest("POST", "/v1/auth/signin", body)
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

	req := httptest.NewRequest("POST", "/v1/auth/signin", body)
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

	req := httptest.NewRequest("POST", "/v1/auth/signout", nil)
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

	req := httptest.NewRequest("POST", "/v1/auth/signout", nil)
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

	req := httptest.NewRequest("POST", "/v1/auth/refresh", body)
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
	req := httptest.NewRequest("POST", "/v1/auth/refresh", body)
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

	req := httptest.NewRequest("GET", "/v1/auth/me", nil)
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

	req := httptest.NewRequest("GET", "/v1/auth/me", nil)
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

	req := httptest.NewRequest("PATCH", "/v1/auth/me", body)
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

	req := httptest.NewRequest("GET", "/v1/auth/sessions", nil)
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

	req := httptest.NewRequest("GET", "/v1/auth/sessions", nil)
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

	req := httptest.NewRequest("DELETE", "/v1/auth/sessions/"+sess.ID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleRevokeSession_InvalidID(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := a.Handler()

	req := httptest.NewRequest("DELETE", "/v1/auth/sessions/invalid-id", nil)
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

	req := httptest.NewRequest("POST", "/v1/auth/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "email already taken", resp["error"])
}
