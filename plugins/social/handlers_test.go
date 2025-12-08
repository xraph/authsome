package social

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/social/providers"
	"github.com/xraph/authsome/schema"
)

func TestSignInRequest_JSON(t *testing.T) {
	req := SignInRequest{
		Provider:    "google",
		Scopes:      []string{"email", "profile"},
		RedirectURL: "https://example.com/callback",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var decoded SignInRequest
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "google", decoded.Provider)
	assert.Equal(t, 2, len(decoded.Scopes))
}

func TestLinkAccountRequest_JSON(t *testing.T) {
	req := LinkAccountRequest{
		Provider: "github",
		Scopes:   []string{"user:email"},
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var decoded LinkAccountRequest
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "github", decoded.Provider)
	assert.Equal(t, 1, len(decoded.Scopes))
}

func TestAuthURLResponse_JSON(t *testing.T) {
	resp := AuthURLResponse{
		URL: "https://accounts.google.com/o/oauth2/v2/auth?client_id=123",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded AuthURLResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Contains(t, decoded.URL, "accounts.google.com")
}

func TestCallbackDataResponse_JSON(t *testing.T) {
	userID := xid.New()

	resp := CallbackDataResponse{
		User: user.FromSchemaUser(&schema.User{
			ID:    userID,
			Email: "test@example.com",
		}),
		IsNewUser: true,
		Action:    "signup",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded CallbackDataResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, userID, decoded.User.ID)
	assert.True(t, decoded.IsNewUser)
	assert.Equal(t, "signup", decoded.Action)
}

func TestProvidersResponse_JSON(t *testing.T) {
	resp := ProvidersResponse{
		Providers: []string{"google", "github", "facebook"},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded ProvidersResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(decoded.Providers))
	assert.Contains(t, decoded.Providers, "google")
}

func TestHandler_ListProviders(t *testing.T) {
	// Setup
	config := Config{
		BaseURL:        "http://localhost:3000",
		AutoCreateUser: true,
	}

	mockSocialRepo := &MockSocialAccountRepository{}
	mockStateStore := NewMockStateStore()
	mockAudit := &audit.Service{}

	// Note: For this test, we'll create service without user service
	// since we're only testing provider listing which doesn't need it
	service := &Service{
		config:     config,
		providers:  make(map[string]providers.Provider),
		socialRepo: mockSocialRepo,
		stateStore: mockStateStore,
		audit:      mockAudit,
	}

	_ = NewHandler(service, nil)

	providers := service.ListProviders()
	assert.NotNil(t, providers)
	assert.IsType(t, []string{}, providers)
}

func TestRateLimiter_Allow(t *testing.T) {
	// Test rate limiter with nil Redis client (should allow all requests)
	limiter := NewRateLimiter(nil)
	ctx := context.Background()

	err := limiter.Allow(ctx, "oauth_signin", "127.0.0.1")
	assert.NoError(t, err)

	// Test setting custom limits
	limiter.SetLimit("custom_action", 5, 1*time.Minute)
	limit, ok := limiter.limits["custom_action"]
	assert.True(t, ok)
	assert.Equal(t, 5, limit.Requests)
	assert.Equal(t, 1*time.Minute, limit.Window)
}
