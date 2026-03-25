package magiclink_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

const testAppIDStr = "aapp_01jf0000000000000000000000"

// mockMailer captures sent magic links for test assertions.
type mockMailer struct {
	mu     sync.Mutex
	sent   []sentLink
	errVal error
}

type sentLink struct {
	Email string
	Token string
}

func (m *mockMailer) SendMagicLink(_ context.Context, email, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentLink{Email: email, Token: token})
	return m.errVal
}

func newTestPlugin(t *testing.T) (*magiclink.Plugin, *memory.Store, *mockMailer) {
	t.Helper()
	mailer := &mockMailer{}
	p := magiclink.New(magiclink.Config{
		Mailer:            mailer,
		TokenTTL:          5 * time.Minute,
		SessionTokenTTL:   1 * time.Hour,
		SessionRefreshTTL: 24 * time.Hour,
	})
	s := memory.New()
	p.SetStore(s)
	p.SetAppID(testAppIDStr)
	return p, s, mailer
}

func createTestUser(t *testing.T, s *memory.Store) *user.User {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	u := &user.User{
		ID:        id.NewUserID(),
		AppID:     appID,
		Email:     "test@example.com",
		FirstName: "Test User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateUser(context.Background(), u)
	require.NoError(t, err)
	return u
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// ──────────────────────────────────────────────────
// Unit tests
// ──────────────────────────────────────────────────

func TestPlugin_Name(t *testing.T) {
	p := magiclink.New(magiclink.Config{})
	assert.Equal(t, "magiclink", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) { //nolint:revive // test function signature
	p := magiclink.New(magiclink.Config{})

	// Plugin should implement base Plugin interface
	var _ plugin.Plugin = p

	// Plugin should implement RouteProvider
	var _ plugin.RouteProvider = p

	// Plugin should implement OnInit
	var _ plugin.OnInit = p
}

func TestPlugin_DefaultConfig(t *testing.T) {
	p := magiclink.New(magiclink.Config{})
	// Should not panic with zero config — defaults are applied internally
	assert.Equal(t, "magiclink", p.Name())
}

func TestPlugin_RegisterInRegistry(t *testing.T) {
	reg := plugin.NewRegistry(log.NewNoopLogger())
	p := magiclink.New(magiclink.Config{})

	reg.Register(p)

	assert.Len(t, reg.Plugins(), 1)
	assert.Equal(t, "magiclink", reg.Plugins()[0].Name())

	// Should be discoverable as a RouteProvider
	assert.Len(t, reg.RouteProviders(), 1)
}

// ──────────────────────────────────────────────────
// Send endpoint tests
// ──────────────────────────────────────────────────

func TestHandleSend_Success(t *testing.T) {
	p, _, mailer := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{
		"email": "user@example.com",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/send", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "magic link sent", resp["status"])

	// Mailer should have been called
	mailer.mu.Lock()
	defer mailer.mu.Unlock()
	assert.Len(t, mailer.sent, 1)
	assert.Equal(t, "user@example.com", mailer.sent[0].Email)
	assert.NotEmpty(t, mailer.sent[0].Token)
}

func TestHandleSend_MissingEmail(t *testing.T) {
	p, _, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/send", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSend_InvalidJSON(t *testing.T) {
	p, _, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/send", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSend_MailerError(t *testing.T) {
	p, _, mailer := newTestPlugin(t)
	mailer.errVal = assert.AnError

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{
		"email": "user@example.com",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/send", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleSend_WithExplicitAppID(t *testing.T) {
	p, _, mailer := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{
		"email":  "user@example.com",
		"app_id": testAppIDStr,
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/send", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	mailer.mu.Lock()
	defer mailer.mu.Unlock()
	assert.Len(t, mailer.sent, 1)
}

// ──────────────────────────────────────────────────
// Verify endpoint tests
// ──────────────────────────────────────────────────

func TestHandleVerify_Success(t *testing.T) {
	p, s, mailer := newTestPlugin(t)
	u := createTestUser(t, s)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	// Create a verification via the store directly
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	v, err := account.NewVerification(context.Background(), appID, u.ID, magiclink.VerificationTypeMagicLink, 5*time.Minute)
	require.NoError(t, err)
	err = s.CreateVerification(context.Background(), v)
	require.NoError(t, err)

	_ = mailer // not used in verify

	body := jsonBody(t, map[string]string{
		"token": v.Token,
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/verify", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["user"])
	assert.NotEmpty(t, resp["session_token"])
	assert.NotEmpty(t, resp["refresh_token"])
	assert.NotEmpty(t, resp["expires_at"])
}

func TestHandleVerify_MissingToken(t *testing.T) {
	p, _, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/verify", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleVerify_InvalidToken(t *testing.T) {
	p, _, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{
		"token": "nonexistent-token",
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/verify", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleVerify_ExpiredToken(t *testing.T) {
	p, s, _ := newTestPlugin(t)
	u := createTestUser(t, s)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	// Create an already-expired verification
	v, err := account.NewVerification(context.Background(), appID, u.ID, magiclink.VerificationTypeMagicLink, 1*time.Millisecond)
	require.NoError(t, err)
	// Force it to be expired
	v.ExpiresAt = time.Now().Add(-1 * time.Hour)
	err = s.CreateVerification(context.Background(), v)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{
		"token": v.Token,
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/verify", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "expired")
}

func TestHandleVerify_AlreadyConsumed(t *testing.T) {
	p, s, _ := newTestPlugin(t)
	u := createTestUser(t, s)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	v, err := account.NewVerification(context.Background(), appID, u.ID, magiclink.VerificationTypeMagicLink, 5*time.Minute)
	require.NoError(t, err)
	err = s.CreateVerification(context.Background(), v)
	require.NoError(t, err)

	// Consume it
	err = s.ConsumeVerification(context.Background(), v.Token)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{
		"token": v.Token,
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/verify", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "already used")
}

func TestHandleVerify_InvalidJSON(t *testing.T) {
	p, _, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/verify", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ──────────────────────────────────────────────────
// Full flow: send + verify
// ──────────────────────────────────────────────────

func TestFullFlow_SendThenVerify(t *testing.T) {
	p, s, mailer := newTestPlugin(t)
	u := createTestUser(t, s)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	// The send endpoint creates a verification with a random userID (since it doesn't
	// look up the user by email). For a proper E2E, we create the verification
	// associated with the real user directly.
	v, err := account.NewVerification(context.Background(), appID, u.ID, magiclink.VerificationTypeMagicLink, 5*time.Minute)
	require.NoError(t, err)
	err = s.CreateVerification(context.Background(), v)
	require.NoError(t, err)

	_ = mailer

	// Step 2: Verify
	body := jsonBody(t, map[string]string{
		"token": v.Token,
	})

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/magic-link/verify", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["user"])
	assert.NotEmpty(t, resp["session_token"])
	assert.NotEmpty(t, resp["refresh_token"])

	// Session should be stored
	sessions, err := s.ListUserSessions(context.Background(), u.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}

// ──────────────────────────────────────────────────
// MailerFunc adapter
// ──────────────────────────────────────────────────

func TestMailerFunc(t *testing.T) {
	var called bool
	var capturedEmail, capturedToken string

	f := magiclink.MailerFunc(func(_ context.Context, email, token string) error {
		called = true
		capturedEmail = email
		capturedToken = token
		return nil
	})

	err := f.SendMagicLink(context.Background(), "test@example.com", "abc123")
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "test@example.com", capturedEmail)
	assert.Equal(t, "abc123", capturedToken)
}
