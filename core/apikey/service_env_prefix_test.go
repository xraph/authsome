package apikey

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// mockEnvironmentRepository implements EnvironmentRepository for testing
type mockEnvironmentRepository struct {
	environments map[xid.ID]*schema.Environment
}

func (m *mockEnvironmentRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Environment, error) {
	if env, ok := m.environments[id]; ok {
		return env, nil
	}
	return nil, fmt.Errorf("environment not found")
}

// mockAPIKeyRepository implements Repository for testing
type mockAPIKeyRepository struct {
	keys map[xid.ID]*schema.APIKey
}

func (m *mockAPIKeyRepository) CreateAPIKey(ctx context.Context, key *schema.APIKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *mockAPIKeyRepository) FindAPIKeyByID(ctx context.Context, id xid.ID) (*schema.APIKey, error) {
	if key, ok := m.keys[id]; ok {
		return key, nil
	}
	return nil, ErrAPIKeyNotFound
}

func (m *mockAPIKeyRepository) FindAPIKeyByPrefix(ctx context.Context, prefix string) (*schema.APIKey, error) {
	for _, key := range m.keys {
		if key.Prefix == prefix {
			return key, nil
		}
	}
	return nil, ErrAPIKeyNotFound
}

func (m *mockAPIKeyRepository) ListAPIKeys(ctx context.Context, filter *ListAPIKeysFilter) (*pagination.PageResponse[*schema.APIKey], error) {
	return nil, nil
}

func (m *mockAPIKeyRepository) UpdateAPIKey(ctx context.Context, key *schema.APIKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *mockAPIKeyRepository) UpdateAPIKeyUsage(ctx context.Context, id xid.ID, ip, userAgent string) error {
	return nil
}

func (m *mockAPIKeyRepository) DeactivateAPIKey(ctx context.Context, id xid.ID) error {
	if key, ok := m.keys[id]; ok {
		key.Active = false
		return nil
	}
	return ErrAPIKeyNotFound
}

func (m *mockAPIKeyRepository) DeleteAPIKey(ctx context.Context, id xid.ID) error {
	delete(m.keys, id)
	return nil
}

func (m *mockAPIKeyRepository) CountAPIKeys(ctx context.Context, appID xid.ID, envID *xid.ID, orgID *xid.ID, userID *xid.ID) (int, error) {
	return len(m.keys), nil
}

func (m *mockAPIKeyRepository) CleanupExpiredAPIKeys(ctx context.Context) (int, error) {
	return 0, nil
}

func TestGeneratePrefix_EnvironmentTypes(t *testing.T) {
	// Setup
	appID := xid.New()
	userID := xid.New()
	devEnvID := xid.New()
	prodEnvID := xid.New()
	stagingEnvID := xid.New()
	previewEnvID := xid.New()
	testEnvID := xid.New()

	// Create mock environment repository
	envRepo := &mockEnvironmentRepository{
		environments: map[xid.ID]*schema.Environment{
			devEnvID: {
				ID:   devEnvID,
				Type: schema.EnvironmentTypeDevelopment,
				Name: "Development",
			},
			prodEnvID: {
				ID:   prodEnvID,
				Type: schema.EnvironmentTypeProduction,
				Name: "Production",
			},
			stagingEnvID: {
				ID:   stagingEnvID,
				Type: schema.EnvironmentTypeStaging,
				Name: "Staging",
			},
			previewEnvID: {
				ID:   previewEnvID,
				Type: schema.EnvironmentTypePreview,
				Name: "Preview",
			},
			testEnvID: {
				ID:   testEnvID,
				Type: schema.EnvironmentTypeTest,
				Name: "Test",
			},
		},
	}

	// Create service with mock repositories
	apiKeyRepo := &mockAPIKeyRepository{keys: make(map[xid.ID]*schema.APIKey)}
	cfg := Config{
		DefaultRateLimit: 1000,
		MaxRateLimit:     10000,
		DefaultExpiry:    365 * 24 * time.Hour,
		MaxKeysPerUser:   10,
		MaxKeysPerOrg:    100,
		KeyLength:        32,
	}

	service := NewService(apiKeyRepo, nil, cfg)
	service.SetEnvironmentRepository(envRepo)

	tests := []struct {
		name           string
		envID          xid.ID
		keyType        KeyType
		expectedPrefix string
	}{
		{
			name:           "Development environment - publishable key",
			envID:          devEnvID,
			keyType:        KeyTypePublishable,
			expectedPrefix: "pk_dev_",
		},
		{
			name:           "Production environment - publishable key",
			envID:          prodEnvID,
			keyType:        KeyTypePublishable,
			expectedPrefix: "pk_prod_",
		},
		{
			name:           "Staging environment - publishable key",
			envID:          stagingEnvID,
			keyType:        KeyTypePublishable,
			expectedPrefix: "pk_staging_",
		},
		{
			name:           "Preview environment - publishable key",
			envID:          previewEnvID,
			keyType:        KeyTypePublishable,
			expectedPrefix: "pk_preview_",
		},
		{
			name:           "Test environment - publishable key",
			envID:          testEnvID,
			keyType:        KeyTypePublishable,
			expectedPrefix: "pk_test_",
		},
		{
			name:           "Development environment - secret key",
			envID:          devEnvID,
			keyType:        KeyTypeSecret,
			expectedPrefix: "sk_dev_",
		},
		{
			name:           "Production environment - secret key",
			envID:          prodEnvID,
			keyType:        KeyTypeSecret,
			expectedPrefix: "sk_prod_",
		},
		{
			name:           "Development environment - restricted key",
			envID:          devEnvID,
			keyType:        KeyTypeRestricted,
			expectedPrefix: "rk_dev_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create API key
			req := &CreateAPIKeyRequest{
				AppID:         appID,
				EnvironmentID: tt.envID,
				UserID:        userID,
				Name:          "Test Key",
				Description:   "Testing environment prefix",
				KeyType:       tt.keyType,
				Scopes:        []string{"app:identify"},
			}

			key, err := service.CreateAPIKey(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, key)

			// Verify prefix
			assert.Contains(t, key.Key, tt.expectedPrefix, 
				"Key should have prefix %s, got: %s", tt.expectedPrefix, key.Key)

			// Verify prefix format: {type}_{env}_{random}
			// The key should be in format: prefix.secret
			// And prefix should be: {type}_{env}_{random}
			parts := splitKeyParts(key.Key)
			assert.Equal(t, 2, len(parts), "Key should have 2 parts (prefix.secret)")
			
			prefix := parts[0]
			assert.True(t, len(prefix) > len(tt.expectedPrefix), 
				"Prefix should include random suffix")
			assert.True(t, startsWith(prefix, tt.expectedPrefix),
				"Prefix should start with %s, got: %s", tt.expectedPrefix, prefix)
		})
	}
}

func TestGeneratePrefix_Caching(t *testing.T) {
	// Setup
	appID := xid.New()
	userID := xid.New()
	devEnvID := xid.New()

	envRepo := &mockEnvironmentRepository{
		environments: map[xid.ID]*schema.Environment{
			devEnvID: {
				ID:   devEnvID,
				Type: schema.EnvironmentTypeDevelopment,
				Name: "Development",
			},
		},
	}

	apiKeyRepo := &mockAPIKeyRepository{keys: make(map[xid.ID]*schema.APIKey)}
	cfg := Config{
		DefaultRateLimit: 1000,
		MaxRateLimit:     10000,
		DefaultExpiry:    365 * 24 * time.Hour,
		MaxKeysPerUser:   10,
		MaxKeysPerOrg:    100,
		KeyLength:        32,
	}

	service := NewService(apiKeyRepo, nil, cfg)
	service.SetEnvironmentRepository(envRepo)

	// Create first key (cache miss)
	req1 := &CreateAPIKeyRequest{
		AppID:         appID,
		EnvironmentID: devEnvID,
		UserID:        userID,
		Name:          "Test Key 1",
		KeyType:       KeyTypePublishable,
		Scopes:        []string{"app:identify"},
	}

	key1, err := service.CreateAPIKey(context.Background(), req1)
	require.NoError(t, err)
	require.NotNil(t, key1)

	// Create second key (cache hit)
	req2 := &CreateAPIKeyRequest{
		AppID:         appID,
		EnvironmentID: devEnvID,
		UserID:        userID,
		Name:          "Test Key 2",
		KeyType:       KeyTypeSecret,
		Scopes:        []string{"admin:full"},
	}

	key2, err := service.CreateAPIKey(context.Background(), req2)
	require.NoError(t, err)
	require.NotNil(t, key2)

	// Both keys should have dev prefix
	assert.Contains(t, key1.Key, "pk_dev_")
	assert.Contains(t, key2.Key, "sk_dev_")

	// Verify cache is populated
	assert.Equal(t, 1, len(service.envCache), "Cache should have 1 entry")
	assert.Equal(t, "dev", service.envCache[devEnvID], "Cache should map devEnvID to 'dev'")
}

func TestGeneratePrefix_ErrorHandling(t *testing.T) {
	// Setup service without environment repository
	apiKeyRepo := &mockAPIKeyRepository{keys: make(map[xid.ID]*schema.APIKey)}
	cfg := Config{
		DefaultRateLimit: 1000,
		MaxRateLimit:     10000,
		DefaultExpiry:    365 * 24 * time.Hour,
		MaxKeysPerUser:   10,
		MaxKeysPerOrg:    100,
		KeyLength:        32,
	}

	service := NewService(apiKeyRepo, nil, cfg)
	// Note: NOT setting environment repository

	appID := xid.New()
	userID := xid.New()
	envID := xid.New()

	req := &CreateAPIKeyRequest{
		AppID:         appID,
		EnvironmentID: envID,
		UserID:        userID,
		Name:          "Test Key",
		KeyType:       KeyTypePublishable,
		Scopes:        []string{"app:identify"},
	}

	// Should fail because environment repository is not configured
	_, err := service.CreateAPIKey(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "environment repository not configured")
}

// Helper functions

func splitKeyParts(key string) []string {
	parts := []string{}
	current := ""
	for _, c := range key {
		if c == '.' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func startsWith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

