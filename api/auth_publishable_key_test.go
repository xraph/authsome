package api_test

// Tests for the X-Publishable-Key signup/signin routing fix.
//
// Before the fix, /v1/signup with a non-platform tenant's pk would still
// land the user in the platform app because the SDK did not send the key
// and the handler silently fell back to engine.Config().AppID. These
// tests pin the new behavior:
//
//   1. pk header → user lands in the resolved App (the bug repro).
//   2. Body app_id alone still works (server-to-server callers).
//   3. pk + matching app_id is fine; mismatched values fail closed.
//   4. Neither pk nor app_id → 400 "app context required".

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
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// tenantAppIDStr is the second app used to verify routing into a
// non-platform tenant. Distinct TypeID from testAppIDStr so a user
// landing in the wrong app shows up immediately as a string mismatch.
const tenantAppIDStr = "aapp_01jf0000000000000000000099"

// tenantPublishableKey is the publishable key seeded onto the tenant app.
const tenantPublishableKey = "pk_test_authsome_tenant_default"

// newMultiAppAPI builds a handler with two apps: the seeded platform app
// (testAppIDStr / testPublishableKey from api_test.go) and a separate
// tenant app (tenantAppIDStr / tenantPublishableKey). The returned handler
// is the bare http.Handler — callers attach the X-Publishable-Key header
// themselves so each test can drive a specific routing branch.
func newMultiAppAPI(t *testing.T) (http.Handler, *authsome.Engine) { //nolint:unparam // engine return retained for future tests
	t.Helper()
	s := memory.New()
	seedTestPlatformApp(t, s)

	tenantID, err := id.ParseAppID(tenantAppIDStr)
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, s.CreateApp(context.Background(), &app.App{
		ID:             tenantID,
		Name:           "Tenant",
		Slug:           "tenant",
		PublishableKey: tenantPublishableKey,
		IsPlatform:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}))

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
	secutil.RelaxAuthDefaults(t, eng)

	router := forge.NewRouter()
	a := api.New(eng, router)
	require.NoError(t, a.RegisterRoutes(router))
	return router.Handler(), eng
}

// signupBody helps build a /v1/signup body without polluting tests with
// boilerplate; signup field set is small but the marshal is repetitive.
func signupBody(t *testing.T, email, password, appID string) *bytes.Buffer { //nolint:unparam // password kept parameterised for future variants
	t.Helper()
	body := map[string]string{"email": email, "password": password}
	if appID != "" {
		body["app_id"] = appID
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// TestSignup_PublishableKeyResolvesNonPlatformApp pins the canonical bug
// repro: a frontend with the non-platform tenant's pk_* must NOT route
// the new user into the platform app.
func TestSignup_PublishableKeyResolvesNonPlatformApp(t *testing.T) {
	handler, _ := newMultiAppAPI(t)

	body := signupBody(t, "tenant-user@example.com", "SecureP@ss1", "")
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, tenantPublishableKey)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	user, _ := resp["user"].(map[string]any)
	require.NotNil(t, user, "response missing user: %v", resp)
	assert.Equal(t, tenantAppIDStr, user["app_id"],
		"user must land in the tenant app resolved from pk; got app_id=%v", user["app_id"])
	assert.NotEqual(t, testAppIDStr, user["app_id"],
		"user must NOT land in the platform app — that was the bug")
}

// TestSignup_BodyAppIDStillWorks pins backwards-compat for server-to-server
// callers (no pk, just an app_id in the body).
func TestSignup_BodyAppIDStillWorks(t *testing.T) {
	handler, _ := newMultiAppAPI(t)

	body := signupBody(t, "s2s-user@example.com", "SecureP@ss1", tenantAppIDStr)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	user, _ := resp["user"].(map[string]any)
	require.NotNil(t, user)
	assert.Equal(t, tenantAppIDStr, user["app_id"])
}

// TestSignup_PublishableKeyAndMatchingAppID pins that consistent values
// agree and route correctly — both can be sent (e.g. by a server that
// proxies a frontend request and re-asserts the app for safety).
func TestSignup_PublishableKeyAndMatchingAppID(t *testing.T) {
	handler, _ := newMultiAppAPI(t)

	body := signupBody(t, "consistent@example.com", "SecureP@ss1", tenantAppIDStr)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, tenantPublishableKey)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
}

// TestSignup_PublishableKeyAndMismatchedAppIDFailsClosed pins the
// fail-closed semantics: pk says one app, body says another. A consistent
// caller never produces this; an attacker probing for AppID confusion
// might. Reject.
func TestSignup_PublishableKeyAndMismatchedAppIDFailsClosed(t *testing.T) {
	handler, _ := newMultiAppAPI(t)

	body := signupBody(t, "mismatch@example.com", "SecureP@ss1", testAppIDStr)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, tenantPublishableKey)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code,
		"mismatched pk + app_id must reject; body=%s", rec.Body.String())
}

// TestSignup_NoAppContextRejectsWith400 pins the headline behavior change:
// without either a pk or a body app_id, signup no longer falls back to
// the platform app. Single-app installs must now send pk_* or app_id.
func TestSignup_NoAppContextRejectsWith400(t *testing.T) {
	handler, _ := newMultiAppAPI(t)

	body := signupBody(t, "no-context@example.com", "SecureP@ss1", "")
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code,
		"signup without app context must 400; body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "app context required",
		"error must name the missing-context cause")
}

// TestSignup_UnknownPublishableKeyRejectsWith400 pins that an unrecognized
// pk_* falls through to the resolver's "no app context" branch — the
// middleware quietly drops bad keys, then the handler 400s the same way
// as a missing key.
func TestSignup_UnknownPublishableKeyRejectsWith400(t *testing.T) {
	handler, _ := newMultiAppAPI(t)

	body := signupBody(t, "bad-key@example.com", "SecureP@ss1", "")
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, "pk_test_does_not_exist")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
}

// TestSignin_PublishableKeyRoutesToTenantApp confirms /v1/signin uses the
// same resolution path. A user signed up under the tenant pk must be
// signed in via the same pk; signing in with the platform pk must miss
// (different app's user pool).
func TestSignin_PublishableKeyRoutesToTenantApp(t *testing.T) {
	handler, _ := newMultiAppAPI(t)

	// Sign up via tenant pk.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signup",
		signupBody(t, "signin-tenant@example.com", "SecureP@ss1", ""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, tenantPublishableKey)
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "signup body=%s", rec.Body.String())

	// Signin via the same tenant pk → ok.
	rec = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signin",
		signupBody(t, "signin-tenant@example.com", "SecureP@ss1", ""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, tenantPublishableKey)
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "signin same-pk body=%s", rec.Body.String())

	// Signin via the platform pk → must NOT find the user. The endpoint
	// is anti-enumeration so the exact error is intentionally generic;
	// we only assert it's not a success.
	rec = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signin",
		signupBody(t, "signin-tenant@example.com", "SecureP@ss1", ""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, testPublishableKey)
	handler.ServeHTTP(rec, req)
	require.NotEqual(t, http.StatusOK, rec.Code,
		"signin with wrong-app pk must NOT succeed; body=%s", rec.Body.String())
}

// TestForgotPassword_RoutesByPublishableKey confirms forgot-password also
// honors the pk path. Endpoint is anti-enumeration (always 200) so the
// assertion is just "no 400 fallback when pk is supplied".
func TestForgotPassword_RoutesByPublishableKey(t *testing.T) {
	handler, _ := newMultiAppAPI(t)

	body, err := json.Marshal(map[string]string{"email": "anyone@example.com"})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/forgot-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.PublishableKeyHeader, tenantPublishableKey)
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code,
		"forgot-password with valid pk must return 200; body=%s", rec.Body.String())

	// Without a pk and without app_id → 400.
	rec = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/forgot-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"forgot-password without app context must 400; body=%s", rec.Body.String())
}
