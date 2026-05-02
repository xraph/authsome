package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/captcha"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/settings"
)

// ──────────────────────────────────────────────────
// Test scaffolding
// ──────────────────────────────────────────────────

// captchaMemStore is a minimal in-memory settings.Store for tests.
type captchaMemStore struct {
	mu       sync.Mutex
	settings map[string]*settings.Setting
}

func newCaptchaMemStore() *captchaMemStore {
	return &captchaMemStore{settings: make(map[string]*settings.Setting)}
}

func captchaMemKey(key string, scope settings.Scope, scopeID string) string {
	return key + "|" + string(scope) + "|" + scopeID
}

func (s *captchaMemStore) GetSetting(_ context.Context, key string, scope settings.Scope, scopeID string) (*settings.Setting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.settings[captchaMemKey(key, scope, scopeID)]
	if !ok {
		return nil, settings.ErrNotFound
	}
	return st, nil
}

func (s *captchaMemStore) SetSetting(_ context.Context, st *settings.Setting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[captchaMemKey(st.Key, st.Scope, st.ScopeID)] = st
	return nil
}

func (s *captchaMemStore) DeleteSetting(_ context.Context, key string, scope settings.Scope, scopeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.settings, captchaMemKey(key, scope, scopeID))
	return nil
}

func (s *captchaMemStore) ListSettings(_ context.Context, _ settings.ListOpts) ([]*settings.Setting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*settings.Setting, 0, len(s.settings))
	for _, st := range s.settings {
		out = append(out, st)
	}
	return out, nil
}

func (s *captchaMemStore) ResolveSettings(_ context.Context, key string, opts settings.ResolveOpts) ([]*settings.Setting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*settings.Setting
	if st, ok := s.settings[captchaMemKey(key, settings.ScopeGlobal, "")]; ok {
		result = append(result, st)
	}
	if opts.AppID != "" {
		if st, ok := s.settings[captchaMemKey(key, settings.ScopeApp, opts.AppID)]; ok {
			result = append(result, st)
		}
	}
	if opts.OrgID != "" {
		if st, ok := s.settings[captchaMemKey(key, settings.ScopeOrg, opts.OrgID)]; ok {
			result = append(result, st)
		}
	}
	if opts.UserID != "" {
		if st, ok := s.settings[captchaMemKey(key, settings.ScopeUser, opts.UserID)]; ok {
			result = append(result, st)
		}
	}
	return result, nil
}

func (s *captchaMemStore) BatchResolve(ctx context.Context, keys []string, opts settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	out := make(map[string][]*settings.Setting, len(keys))
	for _, k := range keys {
		got, err := s.ResolveSettings(ctx, k, opts)
		if err != nil {
			return nil, err
		}
		out[k] = got
	}
	return out, nil
}

func (s *captchaMemStore) DeleteSettingsByNamespace(_ context.Context, _ string) error {
	return nil
}

// captchaTestSetup builds a settings.Manager pre-registered with the four
// captcha settings and seeds a global override toggling required.
type captchaTestEnv struct {
	manager *settings.Manager
	store   *captchaMemStore
}

func newCaptchaTestEnv(t *testing.T, required bool) *captchaTestEnv {
	t.Helper()
	store := newCaptchaMemStore()
	mgr := settings.NewManager(store, log.NewNoopLogger())

	// Register the four captcha settings using minimal definitions.
	// Real production code uses the typed defs in captcha_settings.go;
	// for middleware tests we only need the keys + defaults registered
	// against the Manager so Resolve doesn't return ErrUnknownKey.
	require.NoError(t, settings.RegisterTyped(mgr, "auth", settings.Define("auth.captcha_required", false, settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp))))
	require.NoError(t, settings.RegisterTyped(mgr, "auth", settings.Define("auth.captcha_provider", "turnstile", settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp))))
	require.NoError(t, settings.RegisterTyped(mgr, "auth", settings.Define("auth.captcha_secret_key", "", settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp))))
	require.NoError(t, settings.RegisterTyped(mgr, "auth", settings.Define("auth.captcha_site_key", "", settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp))))

	if required {
		raw, _ := json.Marshal(true)
		require.NoError(t, mgr.Set(context.Background(), "auth.captcha_required", raw, settings.ScopeGlobal, "", "", "", "test"))
		secret, _ := json.Marshal("test-secret")
		require.NoError(t, mgr.Set(context.Background(), "auth.captcha_secret_key", secret, settings.ScopeGlobal, "", "", "", "test"))
	}

	return &captchaTestEnv{manager: mgr, store: store}
}

func (e *captchaTestEnv) setSecret(t *testing.T, secret string) {
	t.Helper()
	raw, _ := json.Marshal(secret)
	require.NoError(t, e.manager.Set(context.Background(), "auth.captcha_secret_key", raw, settings.ScopeGlobal, "", "", "", "test"))
}

// stubVerifier is a captcha.Verifier whose Verify call delegates to a
// caller-provided function. Tracks call count for cache assertions.
type stubVerifier struct {
	id        string
	callCount int32
	fn        func(ctx context.Context, token, remoteIP, action string) (*captcha.Result, error)
}

func (s *stubVerifier) Verify(ctx context.Context, token, remoteIP, action string) (*captcha.Result, error) {
	atomic.AddInt32(&s.callCount, 1)
	return s.fn(ctx, token, remoteIP, action)
}

// captchaTestRouter wires the middleware against a forge router.
func captchaTestRouter(t *testing.T, opts middleware.CaptchaOptions, factoryRecord *[]string) http.Handler {
	t.Helper()
	if opts.VerifierFor != nil && factoryRecord != nil {
		// Wrap the supplied factory to record (provider, secret) calls.
		inner := opts.VerifierFor
		opts.VerifierFor = func(provider, secret string) (captcha.Verifier, error) {
			*factoryRecord = append(*factoryRecord, provider+"|"+secret)
			return inner(provider, secret)
		}
	}
	router := forge.NewRouter()
	router.Use(middleware.CaptchaMiddleware(opts))
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})
	router.POST("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})
	return router.Handler()
}

func decodeJSON(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.NewDecoder(body).Decode(&out))
	return out
}

// ──────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────

func TestCaptchaMiddleware_PassThroughWhenNotRequired(t *testing.T) {
	env := newCaptchaTestEnv(t, false)
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		t.Fatal("verifier must not be called when captcha_required=false")
		return nil, nil
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int32(0), atomic.LoadInt32(&stub.callCount))
}

func TestCaptchaMiddleware_RequiresTokenWhenRequired(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		t.Fatal("verifier must not be called when token is missing")
		return nil, nil
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeJSON(t, rec.Body)
	assert.Equal(t, "captcha_required", resp["type"])
}

func TestCaptchaMiddleware_AcceptsValidToken(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return &captcha.Result{Success: true}, nil
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int32(1), atomic.LoadInt32(&stub.callCount))
}

func TestCaptchaMiddleware_RejectsInvalidToken(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return nil, captcha.ErrInvalidToken
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "bad-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeJSON(t, rec.Body)
	assert.Equal(t, "captcha_invalid", resp["type"])
}

func TestCaptchaMiddleware_DuplicateTokenRejects(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return nil, captcha.ErrDuplicateToken
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "replayed-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := decodeJSON(t, rec.Body)
	assert.Equal(t, "captcha_invalid", resp["type"])
}

func TestCaptchaMiddleware_TransientFailureReturns503(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return nil, errors.Join(captcha.ErrTransientFailure, errors.New("upstream 500"))
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "tok")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	resp := decodeJSON(t, rec.Body)
	assert.Equal(t, "captcha_unavailable", resp["type"])
}

func TestCaptchaMiddleware_TokenFromHeaderOrForm(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	var sawToken string
	stub := &stubVerifier{fn: func(_ context.Context, token, _, _ string) (*captcha.Result, error) {
		sawToken = token
		return &captcha.Result{Success: true}, nil
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	// Form fallback (no header).
	body := strings.NewReader("captcha_token=form-tok")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "form-tok", sawToken, "form field should populate token when header absent")

	// Header takes precedence.
	body = strings.NewReader("captcha_token=form-tok")
	req = httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Captcha-Token", "header-tok")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "header-tok", sawToken, "header should take precedence over form field")
}

func TestCaptchaMiddleware_VerifierCachedAcrossRequests(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return &captcha.Result{Success: true}, nil
	}}
	var factoryCalls []string
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, &factoryCalls)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Captcha-Token", "t")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	assert.Equal(t, 1, len(factoryCalls), "factory should be called exactly once when (appID, provider, secret) is unchanged")
}

func TestCaptchaMiddleware_VerifierRebuiltOnSecretRotation(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return &captcha.Result{Success: true}, nil
	}}
	var factoryCalls []string
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, &factoryCalls)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "t")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, len(factoryCalls))

	env.setSecret(t, "new-secret-value")

	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "t")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	assert.Equal(t, 2, len(factoryCalls), "factory must be called again when the secret rotates")
}

type captureChronicle struct {
	mu     sync.Mutex
	events []*bridge.AuditEvent
}

func (c *captureChronicle) Record(_ context.Context, e *bridge.AuditEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	clone := *e
	c.events = append(c.events, &clone)
	return nil
}

func (c *captureChronicle) Events() []*bridge.AuditEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*bridge.AuditEvent, len(c.events))
	copy(out, c.events)
	return out
}

func TestCaptchaMiddleware_RecordsAuditEventOnFailure(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	ch := &captureChronicle{}
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return nil, captcha.ErrInvalidToken
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
		Chronicle:   ch,
		Action:      "signup",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "bad")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	events := ch.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "captcha.verify", events[0].Action)
	assert.Equal(t, "failure", events[0].Outcome)
	assert.Equal(t, bridge.SeverityWarning, events[0].Severity)
	assert.Equal(t, "signup", events[0].Metadata["action"])
}

func TestCaptchaMiddleware_RecordsAuditEventOnSuccess(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	ch := &captureChronicle{}
	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return &captcha.Result{Success: true}, nil
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
		Chronicle:   ch,
		Action:      "signin",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "ok")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	events := ch.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "captcha.verify", events[0].Action)
	assert.Equal(t, "success", events[0].Outcome)
	assert.Equal(t, bridge.SeverityInfo, events[0].Severity)
}

func TestCaptchaMiddleware_RemoteIPExtractedFromHeader(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	var sawIP string
	stub := &stubVerifier{fn: func(_ context.Context, _, ip, _ string) (*captcha.Result, error) {
		sawIP = ip
		return &captcha.Result{Success: true}, nil
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "t")
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "203.0.113.5", sawIP, "first XFF entry should be the original client")
}

func TestCaptchaMiddleware_NoActionPassesEmptyToVerifier(t *testing.T) {
	env := newCaptchaTestEnv(t, true)
	var sawAction string
	stub := &stubVerifier{fn: func(_ context.Context, _, _, action string) (*captcha.Result, error) {
		sawAction = action
		return &captcha.Result{Success: true}, nil
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings:    env.manager,
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "t")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", sawAction)
}

func TestCaptchaMiddleware_AppScopedSetting(t *testing.T) {
	env := newCaptchaTestEnv(t, false) // global default false
	// Override per-app to required=true.
	appID := id.NewAppID()
	raw, _ := json.Marshal(true)
	require.NoError(t, env.manager.Set(context.Background(), "auth.captcha_required", raw, settings.ScopeApp, appID.String(), appID.String(), "", "test"))
	secret, _ := json.Marshal("app-secret")
	require.NoError(t, env.manager.Set(context.Background(), "auth.captcha_secret_key", secret, settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	stub := &stubVerifier{fn: func(_ context.Context, _, _, _ string) (*captcha.Result, error) {
		return nil, captcha.ErrInvalidToken
	}}
	handler := captchaTestRouter(t, middleware.CaptchaOptions{
		Settings: env.manager,
		ResolveAppID: func(_ forge.Context) (id.AppID, bool) {
			return appID, true
		},
		VerifierFor: func(string, string) (captcha.Verifier, error) { return stub, nil },
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Captcha-Token", "t")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code, "per-app override should activate captcha gate")
}
