package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/settings"
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

	// Phase 2A: SettingRequireEmailVerification defaults to true. API tests
	// generally don't exercise the verification gate.
	secutil.RelaxAuthDefaults(t, eng)

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

func TestHandleSignUp_DuplicateEmailReturnsSuccess(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := a.Handler()

	// Pre-create user.
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

	// Enumeration resistance: duplicate signup must NOT return 409.
	assert.NotEqual(t, http.StatusConflict, rec.Code, "duplicate signup must not return 409 (enumeration oracle)")
	assert.Equal(t, http.StatusCreated, rec.Code, "duplicate signup must return same 201 as fresh signup; body=%s", rec.Body.String())
}

func TestSignup_DoesNotLeakEmailExistence(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	bodyA := []byte(`{"email":"leak-a@example.com","password":"SecureP@ss1"}`)

	// First signup must succeed.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(bodyA))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "first signup must succeed; body=%s", rec.Body.String())

	// Second signup with the SAME email must NOT return 409 and the body
	// must NOT contain "taken" / "exists" / similar enumeration markers.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(bodyA))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "duplicate signup must mirror fresh signup status; body=%s", rec.Body.String())

	body := strings.ToLower(rec.Body.String())
	for _, marker := range []string{"taken", "exists", "duplicate", "already"} {
		if strings.Contains(body, marker) {
			t.Errorf("response body leaks email existence (contains %q): %s", marker, body)
		}
	}
}

func TestSignup_DuplicateDoesNotLogInExistingUser(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	// First signup creates user A with password P.
	bodyA := []byte(`{"email":"hijack-a@example.com","password":"SecureP@ss1"}`)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(bodyA)).WithContext(context.Background()))
	require.Equal(t, http.StatusCreated, rec.Code)
	var first map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&first))
	firstToken, _ := first["session_token"].(string)
	require.NotEmpty(t, firstToken, "first signup must return a real token")

	// Second signup uses the SAME email and a DIFFERENT password.
	bodyB := []byte(`{"email":"hijack-a@example.com","password":"WRONGp@ss1"}`)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(bodyB)).WithContext(context.Background()))
	require.Equal(t, http.StatusCreated, rec.Code)
	var second map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&second))

	// The second response must NOT contain the existing user's session
	// token (would be account hijack).
	if tok, _ := second["session_token"].(string); tok == firstToken {
		t.Fatal("duplicate signup returned the existing user's session token — this is account hijack")
	}
}

// TestSignup_DuplicateRunsDummyHash verifies that the duplicate-email path
// runs a dummy password hash so an attacker cannot use HTTP-response timing
// to distinguish duplicate signups (which would otherwise skip argon2id
// entirely and return in ~1ms) from fresh signups (~100ms argon2id). The
// threshold is intentionally generous — CI noise dominates short measurements
// — so we only guard against the "duplicate is 100x faster" oracle case.
func TestSignup_DuplicateRunsDummyHash(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	// First, create the user.
	bodyA := []byte(`{"email":"timing@example.com","password":"SecureP@ss1"}`)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(bodyA)))
	require.Equal(t, http.StatusCreated, rec.Code)

	// Time a fresh signup (different email, same password length).
	freshBody := []byte(`{"email":"new1@example.com","password":"SecureP@ss1"}`)
	freshStart := time.Now()
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(freshBody)))
	freshDuration := time.Since(freshStart)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Time a duplicate signup (same email, different password).
	dupBody := []byte(`{"email":"timing@example.com","password":"DifferentP@ss2"}`)
	dupStart := time.Now()
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(dupBody)))
	dupDuration := time.Since(dupStart)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Duplicate response time must be at least 50% of the fresh signup
	// time (i.e. argon2/bcrypt ran on the duplicate path too). Loose
	// threshold because CI is noisy; we're just guarding against the
	// "duplicate is 100x faster" case.
	minDuration := freshDuration / 2
	if dupDuration < minDuration {
		t.Errorf("duplicate signup was suspiciously fast (%v) compared to fresh (%v); expected dummy hash to consume comparable time", dupDuration, freshDuration)
	}
}

// TestSignup_DuplicateReturnsPlausibleTokenShape verifies that the duplicate
// path returns synthetic session_token / refresh_token values shaped like
// real tokens. Empty tokens on duplicate (vs. always non-empty on fresh
// signup) would itself be the enumeration oracle.
func TestSignup_DuplicateReturnsPlausibleTokenShape(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	// First signup.
	body := []byte(`{"email":"shape@example.com","password":"SecureP@ss1"}`)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(body)))
	require.Equal(t, http.StatusCreated, rec.Code)
	var first map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&first))
	realToken, _ := first["session_token"].(string)
	require.NotEmpty(t, realToken)

	// Duplicate.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(body)))
	require.Equal(t, http.StatusCreated, rec.Code)
	var dup map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&dup))
	dupToken, _ := dup["session_token"].(string)

	if dupToken == "" {
		t.Fatal("duplicate signup returned empty session_token; this is a shape oracle (fresh signups always return a non-empty token)")
	}
	if dupToken == realToken {
		t.Fatal("duplicate returned the existing user's token — account hijack")
	}
	// Same length category as a real token.
	abs := func(a, b int) int {
		if a > b {
			return a - b
		}
		return b - a
	}
	if abs(len(dupToken), len(realToken)) > 4 {
		t.Errorf("duplicate token length %d differs significantly from real token length %d", len(dupToken), len(realToken))
	}
	// expires_at must be present and non-zero so its presence/absence
	// is not itself an oracle.
	if exp, ok := dup["expires_at"]; !ok || exp == nil {
		t.Error("duplicate response missing expires_at; presence/absence is a shape oracle")
	}
	if rt, _ := dup["refresh_token"].(string); rt == "" {
		t.Error("duplicate response missing refresh_token; presence/absence is a shape oracle")
	}
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

// TestSignIn_UnverifiedEmail_Returns403WithStableCode pins the JSON-API
// contract for accounts that exist but haven't verified their email yet.
// After Phase 2A flipped SettingRequireEmailVerification to default-true,
// the account service began returning account.ErrEmailNotVerified on
// signin for unverified users. api/helpers.go.mapError had no branch for
// it, so every such signin fell through to forge.InternalError → HTTP
// 500 with no distinguishable code, breaking SDK consumers that needed
// to differentiate "needs verification" from "server fault."
//
// The dashboard and extension surfaces already special-case this error
// (see dashboard/contributor.go and extension/auth_pages.go); this test
// pins the same coverage for the JSON API.
func TestSignIn_UnverifiedEmail_Returns403WithStableCode(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	// newTestAPI disables the verification gate globally so most tests
	// can sign in freely. Override BACK to true here so this test
	// exercises the production default behaviour.
	require.NoError(t, eng.Settings().Set(
		context.Background(),
		"auth.require_email_verification",
		json.RawMessage(`true`),
		settings.ScopeGlobal, "", "", "",
		"unverified-signin-test",
	))

	// Sign up — engine creates user with EmailVerified=false.
	body := []byte(`{"email":"unverified@example.com","password":"SecureP@ss1"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "signup must succeed; body=%s", rec.Body.String())

	// Now sign in — must return 403, NOT 500.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code,
		"unverified signin must return 403; got %d body=%s", rec.Code, rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	// Stable string code so SDK consumers can branch on the error
	// without inspecting the human-readable message. The HTTP-error
	// shape used elsewhere in the API is {"error": <message>, "code":
	// <int status>}, so the stable string code lives in a "type" field
	// alongside it.
	typ, _ := resp["type"].(string)
	require.Equal(t, "email_not_verified", typ,
		"response missing stable error type code; got %+v", resp)
}

// ──────────────────────────────────────────────────
// Captcha middleware on signup/signin
// ──────────────────────────────────────────────────

// enableCaptcha turns on auth.captcha_required globally and seeds a secret so
// the middleware can build a verifier. Returns a teardown that restores the
// original false default.
func enableCaptcha(t *testing.T, eng *authsome.Engine) {
	t.Helper()
	mgr := eng.Settings()
	require.NoError(t, mgr.Set(context.Background(), "auth.captcha_required",
		json.RawMessage(`true`),
		settings.ScopeGlobal, "", "", "", "test"))
	require.NoError(t, mgr.Set(context.Background(), "auth.captcha_secret_key",
		json.RawMessage(`"test-secret"`),
		settings.ScopeGlobal, "", "", "", "test"))
	t.Cleanup(func() {
		_ = mgr.Set(context.Background(), "auth.captcha_required",
			json.RawMessage(`false`),
			settings.ScopeGlobal, "", "", "", "test")
	})
}

func TestSignup_CaptchaRequiredRejectsMissingToken(t *testing.T) {
	t.Parallel()
	a, eng := newTestAPI(t)
	enableCaptcha(t, eng)
	handler := a.Handler()

	body := jsonBody(t, map[string]string{
		"email":    "captcha-missing@test.com",
		"password": "SecureP@ss1",
	})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code,
		"signup with captcha_required=true and no token must return 403; body=%s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, "captcha_required", resp["type"])

	// Critical security property: the captcha rejection must run BEFORE the
	// dummy-hash budget consumer in handleSignUp. Otherwise every captcha-
	// failed probe still pays argon2 cost server-side, turning the timing-
	// oracle defense into a CPU-DoS amplifier. We assert this by timing the
	// rejection — without the middleware ordering, the response would take
	// ~argon2 time (50-200ms+); with it, it returns near-instantly.
	start := time.Now()
	for range 5 {
		req2 := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signup", jsonBody(t, map[string]string{
			"email":    "captcha-missing-2@test.com",
			"password": "SecureP@ss1",
		}))
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		require.Equal(t, http.StatusForbidden, rec2.Code)
	}
	avg := time.Since(start) / 5
	if avg > 50*time.Millisecond {
		t.Fatalf("captcha rejection took ~%v on average (>50ms); ordering may have regressed and rejected probes are paying argon2 cost", avg)
	}
}

func TestSignin_CaptchaRequiredRejectsMissingToken(t *testing.T) {
	t.Parallel()
	a, eng := newTestAPI(t)
	enableCaptcha(t, eng)
	handler := a.Handler()

	body := jsonBody(t, map[string]string{
		"email":    "any@test.com",
		"password": "AnyPassword1",
	})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signin", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code,
		"signin with captcha_required=true and no token must return 403; body=%s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, "captcha_required", resp["type"])
}

func TestSignup_CaptchaNotRequiredPassesThrough(t *testing.T) {
	t.Parallel()
	a, _ := newTestAPI(t) // bootstrap leaves captcha_required=false (default)
	handler := a.Handler()

	body := jsonBody(t, map[string]string{
		"email":    "captcha-off@test.com",
		"password": "SecureP@ss1",
	})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/signup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code,
		"signup must pass through when captcha_required=false; body=%s", rec.Body.String())
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

// TestWriteAccountError_EmailTaken_NoLeak verifies the duplicate-email path
// no longer returns a 409 with an "email already taken" message. The previous
// behaviour was an enumeration oracle. This test now asserts the synthetic
// success shape — see TestSignup_DoesNotLeakEmailExistence for the full
// enumeration-resistance contract.
func TestWriteAccountError_EmailTaken_NoLeak(t *testing.T) {
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

	assert.NotEqual(t, http.StatusConflict, rec.Code, "must not leak email existence via 409")
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	if msg, _ := resp["error"].(string); msg != "" {
		assert.NotContains(t, strings.ToLower(msg), "taken")
		assert.NotContains(t, strings.ToLower(msg), "exists")
	}
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

// ──────────────────────────────────────────────────
// POST /v1/verify-email/resend (Phase 3B follow-up)
// ──────────────────────────────────────────────────

// TestResendVerification_AlwaysReturns200 asserts the enumeration-
// resistance contract: the endpoint never reveals whether the email
// is registered, unregistered, or already verified.
func TestResendVerification_AlwaysReturns200(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)

	// We only test inputs that COULD enumerate user state. Missing-
	// email or empty-body 400s are bind-layer rejections that don't
	// distinguish registered from unregistered emails — they're fine
	// from an enumeration-resistance standpoint.
	cases := []struct {
		name string
		body string
	}{
		{"unregistered_email", `{"email":"nobody-here@example.com"}`},
		{"weird_local_part", `{"email":"a+b.c@example.com"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/verify-email/resend", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code,
				"resend must return 200 for any input — leaking 4xx would enumerate registered emails. body=%s", rec.Body.String())

			body := strings.ToLower(rec.Body.String())
			for _, marker := range []string{"not found", "no such", "unverified", "already"} {
				if strings.Contains(body, marker) {
					t.Errorf("response leaks state (contains %q): %s", marker, body)
				}
			}
		})
	}
}

// TestResendVerification_CreatesTokenForExistingUnverifiedUser pins
// that for a real user with EmailVerified=false, the engine actually
// mints + persists a verification token (so a wired-up notifier could
// deliver it).
func TestResendVerification_CreatesTokenForExistingUnverifiedUser(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)
	ctx := context.Background()

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	// Create a user, then explicitly mark unverified (Phase 2A
	// signups now require verification, but RelaxAuthDefaults in
	// newTestAPI flipped it back off — we set the flag directly to
	// test the resend path against a known-unverified row).
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "resend-target@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	u.EmailVerified = false
	require.NoError(t, eng.Store().UpdateUser(ctx, u))

	// Capture the hook so we can assert the token surfaced.
	var captured map[string]string
	eng.Hooks().On("test", func(_ context.Context, ev *hook.Event) error {
		if ev.Action == hook.ActionEmailVerificationRequested {
			captured = ev.Metadata
		}
		return nil
	})

	body := []byte(`{"email":"resend-target@example.com"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/verify-email/resend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NotNil(t, captured, "auth.email_verification_requested hook must fire for a real unverified user")
	require.NotEmpty(t, captured["verification_token"], "hook payload must carry the token so a delivery handler can render the link")
	require.NotEmpty(t, captured["expires_at"])
	require.Equal(t, "resend-target@example.com", captured["email"])
}

// TestResendVerification_NoHookForVerifiedUser pins the silent no-op
// path: a user who's already verified gets no fresh token and no hook
// fires (otherwise an attacker could distinguish verified vs not by
// observing the side effect).
func TestResendVerification_NoHookForVerifiedUser(t *testing.T) {
	t.Parallel()
	_, eng := newTestAPI(t)
	router := newAPIWithRouter(t, eng)
	ctx := context.Background()

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "already-verified@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	u.EmailVerified = true
	require.NoError(t, eng.Store().UpdateUser(ctx, u))

	var fired bool
	eng.Hooks().On("test", func(_ context.Context, ev *hook.Event) error {
		if ev.Action == hook.ActionEmailVerificationRequested {
			fired = true
		}
		return nil
	})

	body := []byte(`{"email":"already-verified@example.com"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/verify-email/resend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.False(t, fired,
		"hook must NOT fire for an already-verified user — observing the email being sent would let an attacker enumerate verified-vs-unverified state")
}
