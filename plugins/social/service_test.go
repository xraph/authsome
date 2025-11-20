package social

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/plugins/social/providers"
	"github.com/xraph/authsome/schema"
)

// MockSocialAccountRepository is a mock implementation of SocialAccountRepository
type MockSocialAccountRepository struct {
	mock.Mock
}

func (m *MockSocialAccountRepository) Create(ctx context.Context, account *schema.SocialAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockSocialAccountRepository) FindByProviderUserID(ctx context.Context, provider, providerUserID string) (*schema.SocialAccount, error) {
	args := m.Called(ctx, provider, providerUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.SocialAccount), args.Error(1)
}

func (m *MockSocialAccountRepository) FindByUserID(ctx context.Context, userID xid.ID) ([]*schema.SocialAccount, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.SocialAccount), args.Error(1)
}

func (m *MockSocialAccountRepository) FindByUserAndProvider(ctx context.Context, userID xid.ID, provider string) (*schema.SocialAccount, error) {
	args := m.Called(ctx, userID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.SocialAccount), args.Error(1)
}

func (m *MockSocialAccountRepository) Update(ctx context.Context, account *schema.SocialAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockSocialAccountRepository) Delete(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSocialAccountRepository) DeleteByUserAndProvider(ctx context.Context, userID xid.ID, provider string) error {
	args := m.Called(ctx, userID, provider)
	return args.Error(0)
}

func (m *MockSocialAccountRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SocialAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.SocialAccount), args.Error(1)
}

func (m *MockSocialAccountRepository) FindByProviderAndProviderID(ctx context.Context, provider, providerID string, appID xid.ID, userOrganizationID *xid.ID) (*schema.SocialAccount, error) {
	args := m.Called(ctx, provider, providerID, appID, userOrganizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.SocialAccount), args.Error(1)
}

func (m *MockSocialAccountRepository) FindByUser(ctx context.Context, userID xid.ID) ([]*schema.SocialAccount, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.SocialAccount), args.Error(1)
}

func (m *MockSocialAccountRepository) Unlink(ctx context.Context, userID xid.ID, provider string) error {
	args := m.Called(ctx, userID, provider)
	return args.Error(0)
}

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *schema.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id xid.ID) (*schema.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*schema.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *schema.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*schema.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}

func (m *MockUserRepository) ListUsers(ctx context.Context, limit, offset int) ([]*schema.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.User), args.Error(1)
}

func (m *MockUserRepository) CountUsers(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockStateStore is a mock implementation of StateStore
type MockStateStore struct {
	store map[string]*OAuthState
}

func NewMockStateStore() *MockStateStore {
	return &MockStateStore{
		store: make(map[string]*OAuthState),
	}
}

func (m *MockStateStore) Set(ctx context.Context, key string, state *OAuthState, ttl time.Duration) error {
	m.store[key] = state
	return nil
}

func (m *MockStateStore) Get(ctx context.Context, key string) (*OAuthState, error) {
	state, ok := m.store[key]
	if !ok {
		return nil, fmt.Errorf("state not found")
	}
	return state, nil
}

func (m *MockStateStore) Delete(ctx context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func TestService_ListProviders(t *testing.T) {
	// Create config with providers
	config := Config{
		BaseURL:            "http://localhost:3000",
		AutoCreateUser:     true,
		TrustEmailVerified: true,
	}

	mockSocialRepo := &MockSocialAccountRepository{}
	mockStateStore := NewMockStateStore()
	mockAudit := &audit.Service{}

	// Create service without user service for this test
	service := &Service{
		config:     config,
		providers:  make(map[string]providers.Provider),
		socialRepo: mockSocialRepo,
		stateStore: mockStateStore,
		audit:      mockAudit,
	}

	// Initially no providers configured
	providersList := service.ListProviders()
	assert.Empty(t, providersList)
}

func TestStateStore_SetGetDelete(t *testing.T) {
	store := NewMockStateStore()
	ctx := context.Background()

	appID := xid.New()
	state := &OAuthState{
		Provider:    "google",
		AppID:       appID,
		CreatedAt:   time.Now(),
		ExtraScopes: []string{"email", "profile"},
	}

	// Test Set
	err := store.Set(ctx, "test-state-key", state, 15*time.Minute)
	assert.NoError(t, err)

	// Test Get
	retrieved, err := store.Get(ctx, "test-state-key")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "google", retrieved.Provider)
	assert.Equal(t, appID, retrieved.AppID)
	assert.Equal(t, 2, len(retrieved.ExtraScopes))

	// Test Delete
	err = store.Delete(ctx, "test-state-key")
	assert.NoError(t, err)

	// Verify deleted
	_, err = store.Get(ctx, "test-state-key")
	assert.Error(t, err)
}

func TestMemoryStateStore_Expiration(t *testing.T) {
	store := NewMemoryStateStore()
	ctx := context.Background()

	appID := xid.New()
	
	// Set state with very short TTL
	state := &OAuthState{
		Provider:  "google",
		AppID:     appID,
		CreatedAt: time.Now(),
	}

	// Set state with TTL of 10 milliseconds
	err := store.Set(ctx, "test-key", state, 10*time.Millisecond)
	assert.NoError(t, err)

	// Immediately get - should succeed
	retrieved, err := store.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Wait for expiration
	time.Sleep(50 * time.Millisecond)

	// Try to get - should fail due to expiration
	_, err = store.Get(ctx, "test-key")
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "expired")
	}
}

func TestConfig_Providers(t *testing.T) {
	// Test zero value
	var emptyConfig ProvidersConfig
	assert.Nil(t, emptyConfig.Google)
	assert.Nil(t, emptyConfig.GitHub)

	// Test with Google provider
	configWithGoogle := ProvidersConfig{
		Google: &providers.ProviderConfig{
			ClientID: "test-client-id",
			Enabled:  true,
		},
	}
	assert.NotNil(t, configWithGoogle.Google)
	assert.Equal(t, "test-client-id", configWithGoogle.Google.ClientID)

	// Test with GitHub provider
	configWithGitHub := ProvidersConfig{
		GitHub: &providers.ProviderConfig{
			ClientID: "test-client-id",
			Enabled:  true,
		},
	}
	assert.NotNil(t, configWithGitHub.GitHub)
	assert.Equal(t, "test-client-id", configWithGitHub.GitHub.ClientID)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "http://localhost:3000", config.BaseURL)
	assert.True(t, config.AllowAccountLinking)
	assert.True(t, config.AutoCreateUser)
	assert.False(t, config.RequireEmailVerified)
	assert.False(t, config.StateStorage.UseRedis)
	assert.Equal(t, "localhost:6379", config.StateStorage.RedisAddr)
	assert.Equal(t, 15*time.Minute, config.StateStorage.StateTTL)
}

