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

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/webhook"
)

// newBootstrappedAPI builds an API whose engine has run the bootstrap path, so
// the embedded warden roles are seeded and the first user to sign up is
// promoted to platform-owner (with real manage:* grants). Permission-gated
// route groups can only be exercised against a bootstrapped engine; the plain
// newTestAPI helper does not seed roles.
func newBootstrappedAPI(t *testing.T) (*api.API, *authsome.Engine) {
	t.Helper()
	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithBootstrap(),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)
	return api.New(eng), eng
}

// asAdmin injects both the user ID and the full user object onto the request
// context, mirroring what the identity-resolving auth middleware sets in
// production. Needed for route groups guarded by RequireAuth (which reads the
// full user) plus RequirePermission (which reads the user ID).
func asAdmin(t *testing.T, req *http.Request, eng *authsome.Engine, userID id.UserID) *http.Request {
	t.Helper()
	u, err := eng.AdminGetUser(context.Background(), userID)
	require.NoError(t, err)
	ctx := middleware.WithUserID(req.Context(), userID)
	ctx = middleware.WithUser(ctx, u)
	return req.WithContext(ctx)
}

// otherAppID returns a valid AppID distinct from the test platform app, for
// building cross-tenant fixtures.
func otherAppID(t *testing.T) id.AppID {
	t.Helper()
	appID, err := id.ParseAppID("aapp_01jf9999999999999999999999")
	require.NoError(t, err)
	return appID
}

// ──────────────────────────────────────────────────
// Webhooks — /v1/webhooks CRUD
// ──────────────────────────────────────────────────

func TestWebhookCreate_RequiresAuth(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := withTestKey(a.Handler())

	body := []byte(`{"url":"https://evil.example.com/hook","events":["user.created"]}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWebhookList_RequiresAuth(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := withTestKey(a.Handler())

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/webhooks", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWebhookGet_RequiresAuth(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	w := seedWebhook(t, eng, testAppIDStr)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/webhooks/"+w.ID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWebhookUpdate_RequiresAuth(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	w := seedWebhook(t, eng, testAppIDStr)
	body := []byte(`{"url":"https://evil.example.com/hook"}`)
	req := httptest.NewRequestWithContext(context.Background(), "PATCH", "/v1/webhooks/"+w.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	got, err := eng.GetWebhook(context.Background(), w.ID)
	require.NoError(t, err)
	assert.NotEqual(t, "https://evil.example.com/hook", got.URL, "unauthenticated caller must not rewrite the webhook URL")
}

func TestWebhookDelete_RequiresAuth(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	w := seedWebhook(t, eng, testAppIDStr)
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/webhooks/"+w.ID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWebhookGet_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	// Platform-owner (first user) is fully privileged in their own app, but a
	// webhook belonging to a *different* app must still be invisible to them.
	_, ownerToken, _ := signUp(t, eng, "wh-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreign := seedWebhook(t, eng, otherAppID(t).String())
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/webhooks/"+foreign.ID.String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "a webhook from another app must not be readable")
}

func TestWebhookDelete_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	_, ownerToken, _ := signUp(t, eng, "wh-del-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreign := seedWebhook(t, eng, otherAppID(t).String())
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/webhooks/"+foreign.ID.String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	_, err := eng.GetWebhook(context.Background(), foreign.ID)
	assert.NoError(t, err, "another app's webhook must survive the delete attempt")
}

func TestWebhookCreate_OwnerSucceeds(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	_, ownerToken, _ := signUp(t, eng, "wh-create-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	body := []byte(`{"url":"https://good.example.com/hook","events":["user.created"]}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var got webhook.Webhook
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, testAppIDStr, got.AppID.String(), "created webhook must be bound to the caller's app")
}

func TestWebhookCreate_ForbiddenWithoutPermission(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	// First user consumes the platform-owner slot; the second is a plain user.
	_, _, _ = signUp(t, eng, "wh-first-owner@test.com", "SecureP@ss1")
	_, regularToken, _ := signUp(t, eng, "wh-regular@test.com", "SecureP@ss1")
	regularID := userIDFor(t, eng, regularToken)

	body := []byte(`{"url":"https://good.example.com/hook","events":["user.created"]}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, regularID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "a non-admin user must not create webhooks; body=%s", rec.Body.String())
}

// ──────────────────────────────────────────────────
// Environments — /v1/environments CRUD
// ──────────────────────────────────────────────────

func TestEnvironmentCreate_RequiresAuth(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := withTestKey(a.Handler())

	body := []byte(`{"name":"Prod","slug":"prod","type":"production"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/environments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestEnvironmentList_RequiresAuth(t *testing.T) {
	a, _ := newTestAPI(t)
	handler := withTestKey(a.Handler())

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/environments", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestEnvironmentDelete_RequiresAuth(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	env := seedEnvironment(t, eng, testAppIDStr)
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/environments/"+env.ID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	_, err := eng.GetEnvironment(context.Background(), env.ID)
	assert.NoError(t, err, "environment must not be deleted by an unauthenticated caller")
}

func TestEnvironmentDelete_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	_, ownerToken, _ := signUp(t, eng, "env-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreign := seedEnvironment(t, eng, otherAppID(t).String())
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/environments/"+foreign.ID.String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	_, err := eng.GetEnvironment(context.Background(), foreign.ID)
	assert.NoError(t, err, "another app's environment must survive the delete attempt")
}

func TestEnvironmentCreate_OwnerSucceeds(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	_, ownerToken, _ := signUp(t, eng, "env-create-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	body := []byte(`{"name":"Staging","slug":"staging","type":"staging"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/environments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var got environment.Environment
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, testAppIDStr, got.AppID.String(), "created environment must be bound to the caller's app")
}

// ──────────────────────────────────────────────────
// Fixtures
// ──────────────────────────────────────────────────

func seedWebhook(t *testing.T, eng *authsome.Engine, appIDStr string) *webhook.Webhook {
	t.Helper()
	appID, err := id.ParseAppID(appIDStr)
	require.NoError(t, err)
	w := &webhook.Webhook{
		ID:     id.NewWebhookID(),
		AppID:  appID,
		URL:    "https://original.example.com/hook",
		Events: []string{"user.created"},
	}
	require.NoError(t, eng.Store().CreateWebhook(context.Background(), w))
	return w
}

func seedEnvironment(t *testing.T, eng *authsome.Engine, appIDStr string) *environment.Environment {
	t.Helper()
	appID, err := id.ParseAppID(appIDStr)
	require.NoError(t, err)
	now := time.Now()
	env := &environment.Environment{
		ID:        id.NewEnvironmentID(),
		AppID:     appID,
		Name:      "Original",
		Slug:      "original",
		Type:      environment.TypeDevelopment,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, eng.Store().CreateEnvironment(context.Background(), env))
	return env
}
