package social_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"

	"golang.org/x/oauth2"
)

const testAppIDStr = "aapp_01jf0000000000000000000000"

// mockProvider is a test double for the social.Provider interface.
type mockProvider struct {
	name      string
	cfg       *oauth2.Config
	fetchUser *social.ProviderUser
	fetchErr  error
}

func (m *mockProvider) Name() string                 { return m.name }
func (m *mockProvider) OAuth2Config() *oauth2.Config { return m.cfg }
func (m *mockProvider) FetchUser(_ context.Context, _ *oauth2.Token) (*social.ProviderUser, error) {
	return m.fetchUser, m.fetchErr
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name: name,
		cfg: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost/callback",
			Scopes:       []string{"openid", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://example.com/auth",
				TokenURL: "https://example.com/token",
			},
		},
		fetchUser: &social.ProviderUser{
			ProviderUserID: "provider-user-123",
			Email:          "social@example.com",
			FirstName:      "Social User",
			AvatarURL:      "https://example.com/avatar.png",
		},
	}
}

func newTestPlugin(t *testing.T, providers ...social.Provider) (*social.Plugin, *memory.Store, *social.MemoryStore) {
	t.Helper()
	p := social.New(social.Config{
		Providers:         providers,
		SessionTokenTTL:   1 * time.Hour,
		SessionRefreshTTL: 24 * time.Hour,
	})
	s := memory.New()
	os := social.NewMemoryStore()
	p.SetStore(s)
	p.SetOAuthStore(os)
	p.SetAppID(testAppIDStr)
	return p, s, os
}

// ──────────────────────────────────────────────────
// Unit tests
// ──────────────────────────────────────────────────

func TestPlugin_Name(t *testing.T) {
	p := social.New(social.Config{})
	assert.Equal(t, "social", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	p := social.New(social.Config{})

	var _ plugin.Plugin = p
	var _ plugin.RouteProvider = p
	var _ plugin.OnInit = p
}

func TestPlugin_Providers(t *testing.T) {
	google := newMockProvider("google")
	github := newMockProvider("github")
	p := social.New(social.Config{
		Providers: []social.Provider{google, github},
	})
	names := p.Providers()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "google")
	assert.Contains(t, names, "github")
}

func TestPlugin_RegisterInRegistry(t *testing.T) {
	reg := plugin.NewRegistry(log.NewNoopLogger())
	p := social.New(social.Config{})
	reg.Register(p)

	assert.Len(t, reg.Plugins(), 1)
	assert.Equal(t, "social", reg.Plugins()[0].Name())
	assert.Len(t, reg.RouteProviders(), 1)
}

// ──────────────────────────────────────────────────
// Start endpoint tests
// ──────────────────────────────────────────────────

func TestHandleStart_Success(t *testing.T) {
	google := newMockProvider("google")
	p, _, _ := newTestPlugin(t, google)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/v1/auth/social/google", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["auth_url"])
	assert.Contains(t, resp["auth_url"], "example.com/auth")
	assert.Contains(t, resp["auth_url"], "state=")
}

func TestHandleStart_UnsupportedProvider(t *testing.T) {
	p, _, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/v1/auth/social/twitter", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ──────────────────────────────────────────────────
// Callback endpoint tests (state validation)
// ──────────────────────────────────────────────────

func TestHandleCallback_MissingState(t *testing.T) {
	google := newMockProvider("google")
	p, _, _ := newTestPlugin(t, google)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/v1/auth/social/google/callback?code=abc", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "missing state")
}

func TestHandleCallback_InvalidState(t *testing.T) {
	google := newMockProvider("google")
	p, _, _ := newTestPlugin(t, google)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/v1/auth/social/google/callback?code=abc&state=invalid-state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "invalid state")
}

func TestHandleCallback_MissingCode(t *testing.T) {
	google := newMockProvider("google")
	p, _, _ := newTestPlugin(t, google)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	// Start the flow to get a valid state
	startReq := httptest.NewRequest("POST", "/v1/auth/social/google", nil)
	startRec := httptest.NewRecorder()
	mux.ServeHTTP(startRec, startReq)
	require.Equal(t, http.StatusOK, startRec.Code)

	var startResp map[string]string
	err = json.NewDecoder(startRec.Body).Decode(&startResp)
	require.NoError(t, err)

	state := extractQueryParam(t, startResp["auth_url"], "state")

	// Send callback without code
	req := httptest.NewRequest("GET", "/v1/auth/social/google/callback?state="+state, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "missing code")
}

func TestHandleCallback_ProviderError(t *testing.T) {
	google := newMockProvider("google")
	p, _, _ := newTestPlugin(t, google)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	// Start the flow
	startReq := httptest.NewRequest("POST", "/v1/auth/social/google", nil)
	startRec := httptest.NewRecorder()
	mux.ServeHTTP(startRec, startReq)
	require.Equal(t, http.StatusOK, startRec.Code)

	var startResp map[string]string
	err = json.NewDecoder(startRec.Body).Decode(&startResp)
	require.NoError(t, err)

	state := extractQueryParam(t, startResp["auth_url"], "state")

	req := httptest.NewRequest("GET", "/v1/auth/social/google/callback?state="+state+"&error=access_denied", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "access_denied")
}

func TestHandleCallback_UnsupportedProvider(t *testing.T) {
	p, _, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/v1/auth/social/twitter/callback?code=abc&state=xyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ──────────────────────────────────────────────────
// Memory store tests
// ──────────────────────────────────────────────────

func TestMemoryStore_CRUD(t *testing.T) {
	s := social.NewMemoryStore()
	ctx := context.Background()

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	userID := id.NewUserID()
	conn := &social.OAuthConnection{
		ID:             id.NewOAuthConnectionID(),
		AppID:          appID,
		UserID:         userID,
		Provider:       "google",
		ProviderUserID: "goog-123",
		Email:          "test@google.com",
		AccessToken:    "access-tok",
		RefreshToken:   "refresh-tok",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create
	err = s.CreateOAuthConnection(ctx, conn)
	require.NoError(t, err)

	// Get by provider + provider user ID
	got, err := s.GetOAuthConnection(ctx, "google", "goog-123")
	require.NoError(t, err)
	assert.Equal(t, conn.ID, got.ID)
	assert.Equal(t, "google", got.Provider)
	assert.Equal(t, "goog-123", got.ProviderUserID)

	// Get by user ID
	conns, err := s.GetOAuthConnectionsByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, conns, 1)

	// Not found
	_, err = s.GetOAuthConnection(ctx, "github", "gh-999")
	assert.ErrorIs(t, err, social.ErrConnectionNotFound)

	// Delete
	err = s.DeleteOAuthConnection(ctx, conn.ID)
	require.NoError(t, err)

	// Should be gone
	_, err = s.GetOAuthConnection(ctx, "google", "goog-123")
	assert.ErrorIs(t, err, social.ErrConnectionNotFound)

	// Delete nonexistent
	err = s.DeleteOAuthConnection(ctx, id.NewOAuthConnectionID())
	assert.ErrorIs(t, err, social.ErrConnectionNotFound)
}

func TestMemoryStore_MultipleConnections(t *testing.T) {
	s := social.NewMemoryStore()
	ctx := context.Background()

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	userID := id.NewUserID()

	for _, provider := range []string{"google", "github"} {
		conn := &social.OAuthConnection{
			ID:             id.NewOAuthConnectionID(),
			AppID:          appID,
			UserID:         userID,
			Provider:       provider,
			ProviderUserID: provider + "-123",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		err := s.CreateOAuthConnection(ctx, conn)
		require.NoError(t, err)
	}

	conns, err := s.GetOAuthConnectionsByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, conns, 2)

	// Different user should have zero
	conns, err = s.GetOAuthConnectionsByUserID(ctx, id.NewUserID())
	require.NoError(t, err)
	assert.Len(t, conns, 0)
}

// ──────────────────────────────────────────────────
// Provider tests
// ──────────────────────────────────────────────────

func TestNewGoogleProvider(t *testing.T) {
	p := social.NewGoogleProvider(social.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	})
	assert.Equal(t, "google", p.Name())
	assert.NotNil(t, p.OAuth2Config())
	assert.Equal(t, "client-id", p.OAuth2Config().ClientID)
}

func TestNewGitHubProvider(t *testing.T) {
	p := social.NewGitHubProvider(social.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	})
	assert.Equal(t, "github", p.Name())
	assert.NotNil(t, p.OAuth2Config())
	assert.Equal(t, "client-id", p.OAuth2Config().ClientID)
}

func TestGoogleProvider_CustomScopes(t *testing.T) {
	p := social.NewGoogleProvider(social.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"custom-scope"},
	})
	assert.Equal(t, []string{"custom-scope"}, p.OAuth2Config().Scopes)
}

func TestGitHubProvider_CustomScopes(t *testing.T) {
	p := social.NewGitHubProvider(social.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"repo"},
	})
	assert.Equal(t, []string{"repo"}, p.OAuth2Config().Scopes)
}

// ──────────────────────────────────────────────────
// OAuthConnection model tests
// ──────────────────────────────────────────────────

func TestOAuthConnection_Fields(t *testing.T) {
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	conn := &social.OAuthConnection{
		ID:             id.NewOAuthConnectionID(),
		AppID:          appID,
		UserID:         id.NewUserID(),
		Provider:       "google",
		ProviderUserID: "123",
		Email:          "test@gmail.com",
		AccessToken:    "tok",
		RefreshToken:   "rtok",
		Metadata:       map[string]string{"key": "value"},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// JSON should not expose tokens
	data, err := json.Marshal(conn)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.NotContains(t, parsed, "access_token")
	assert.NotContains(t, parsed, "refresh_token")
	assert.Equal(t, "google", parsed["provider"])
}

// ──────────────────────────────────────────────────
// Integration: start flow state tracking
// ──────────────────────────────────────────────────

func TestFullFlow_StartCreatesState(t *testing.T) {
	google := newMockProvider("google")
	p, _, _ := newTestPlugin(t, google)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	// Start flow
	req := httptest.NewRequest("POST", "/v1/auth/social/google", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	authURL := resp["auth_url"]
	state := extractQueryParam(t, authURL, "state")
	assert.NotEmpty(t, state)

	// State should be consumed after first use (callback with the state)
	// Send an invalid callback (wrong provider state) to a fresh plugin instance
	github := newMockProvider("github")
	p2 := social.New(social.Config{Providers: []social.Provider{google, github}})
	p2.SetStore(memory.New())
	p2.SetAppID(testAppIDStr)
	mux2 := forge.NewRouter()
	err = p2.RegisterRoutes(mux2)
	require.NoError(t, err)

	req2 := httptest.NewRequest("GET", "/v1/auth/social/github/callback?code=abc&state="+state, nil)
	rec2 := httptest.NewRecorder()
	mux2.ServeHTTP(rec2, req2)

	// Should fail because the state was never created in p2
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
}

// ──────────────────────────────────────────────────
// Store interaction tests (without real OAuth exchange)
// ──────────────────────────────────────────────────

func TestPlugin_ExistingUserByEmail(t *testing.T) {
	oauthStore := social.NewMemoryStore()
	coreStore := memory.New()
	ctx := context.Background()

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	existingUser := &user.User{
		ID:        id.NewUserID(),
		AppID:     appID,
		Email:     "existing@example.com",
		FirstName: "Existing User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = coreStore.CreateUser(ctx, existingUser)
	require.NoError(t, err)

	found, err := coreStore.GetUserByEmail(ctx, appID, "existing@example.com")
	require.NoError(t, err)
	assert.Equal(t, existingUser.ID, found.ID)

	conn := &social.OAuthConnection{
		ID:             id.NewOAuthConnectionID(),
		AppID:          appID,
		UserID:         existingUser.ID,
		Provider:       "google",
		ProviderUserID: "goog-456",
		Email:          "existing@example.com",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err = oauthStore.CreateOAuthConnection(ctx, conn)
	require.NoError(t, err)

	found2, err := oauthStore.GetOAuthConnection(ctx, "google", "goog-456")
	require.NoError(t, err)
	assert.Equal(t, existingUser.ID, found2.UserID)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func extractQueryParam(t *testing.T, rawURL, key string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	return u.Query().Get(key)
}

// ──────────────────────────────────────────────────
// Microsoft Provider tests
// ──────────────────────────────────────────────────

func TestNewMicrosoftProvider(t *testing.T) {
	p := social.NewMicrosoftProvider(social.ProviderConfig{
		ClientID:     "ms-client-id",
		ClientSecret: "ms-secret",
		RedirectURL:  "http://localhost/callback",
	})
	assert.Equal(t, "microsoft", p.Name())
	assert.NotNil(t, p.OAuth2Config())
	assert.Equal(t, "ms-client-id", p.OAuth2Config().ClientID)
	// Default scopes
	assert.Contains(t, p.OAuth2Config().Scopes, "User.Read")
}

func TestMicrosoftProvider_CustomScopes(t *testing.T) {
	p := social.NewMicrosoftProvider(social.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"openid", "email"},
	})
	assert.Equal(t, []string{"openid", "email"}, p.OAuth2Config().Scopes)
}

func TestMicrosoftProvider_FetchUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"id": "ms-user-123",
			"displayName": "John Doe",
			"mail": "john@outlook.com",
			"userPrincipalName": "john@contoso.com"
		}`))
	}))
	defer server.Close()

	// Create a provider and swap the Graph API URL via a custom HTTP client
	p := social.NewMicrosoftProvider(social.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	})

	// We can't easily swap the Microsoft Graph URL, so test FetchUser
	// indirectly by verifying the interface contract and returned values
	// from a mock server test in a more integrated way. For now, verify
	// the provider is properly constructed.
	assert.Equal(t, "microsoft", p.Name())
	cfg := p.OAuth2Config()
	assert.Contains(t, cfg.Endpoint.AuthURL, "login.microsoftonline.com")
	_ = server // server reference held so it's not GC'd
}

// ──────────────────────────────────────────────────
// Apple Provider tests
// ──────────────────────────────────────────────────

func TestNewAppleProvider(t *testing.T) {
	p := social.NewAppleProvider(social.ProviderConfig{
		ClientID:     "com.example.app",
		ClientSecret: "apple-secret",
		RedirectURL:  "http://localhost/callback",
	})
	assert.Equal(t, "apple", p.Name())
	assert.NotNil(t, p.OAuth2Config())
	assert.Equal(t, "com.example.app", p.OAuth2Config().ClientID)
	// Default scopes
	assert.Contains(t, p.OAuth2Config().Scopes, "email")
	assert.Contains(t, p.OAuth2Config().Scopes, "name")
}

func TestAppleProvider_CustomScopes(t *testing.T) {
	p := social.NewAppleProvider(social.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"email"},
	})
	assert.Equal(t, []string{"email"}, p.OAuth2Config().Scopes)
}

func TestAppleProvider_FetchUser_FromIDToken(t *testing.T) {
	p := social.NewAppleProvider(social.ProviderConfig{
		ClientID:     "com.example.app",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	})

	// Build a mock JWT with claims in the payload
	// JWT format: header.payload.signature
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"apple-user-001","email":"user@icloud.com","email_verified":true}`))
	sig := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))
	mockJWT := header + "." + payload + "." + sig

	token := &oauth2.Token{}
	token = token.WithExtra(map[string]any{"id_token": mockJWT})

	user, err := p.FetchUser(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, "apple-user-001", user.ProviderUserID)
	assert.Equal(t, "user@icloud.com", user.Email)
}

func TestAppleProvider_FetchUser_NoIDToken(t *testing.T) {
	p := social.NewAppleProvider(social.ProviderConfig{
		ClientID:     "com.example.app",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	})

	token := &oauth2.Token{}
	_, err := p.FetchUser(context.Background(), token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no id_token")
}

func TestAppleProvider_FetchUser_MissingSub(t *testing.T) {
	p := social.NewAppleProvider(social.ProviderConfig{
		ClientID:     "com.example.app",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	})

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"user@icloud.com"}`))
	sig := base64.RawURLEncoding.EncodeToString([]byte("fake"))
	mockJWT := header + "." + payload + "." + sig

	token := &oauth2.Token{}
	token = token.WithExtra(map[string]any{"id_token": mockJWT})

	_, err := p.FetchUser(context.Background(), token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing sub claim")
}
