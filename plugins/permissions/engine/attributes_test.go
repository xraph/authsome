package engine

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/plugins/permissions/engine/providers"
)

func TestAttributeResolver_RegisterProvider(t *testing.T) {
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Create a mock user provider
	userService := providers.NewMockUserService()
	userProvider := providers.NewUserAttributeProvider(userService)
	
	// Test successful registration
	err := resolver.RegisterProvider(userProvider)
	assert.NoError(t, err)
	
	// Test duplicate registration
	err = resolver.RegisterProvider(userProvider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	
	// Test nil provider
	err = resolver.RegisterProvider(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestAttributeResolver_GetProvider(t *testing.T) {
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Test getting non-existent provider
	_, err := resolver.GetProvider("user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	
	// Register and get provider
	userService := providers.NewMockUserService()
	userProvider := providers.NewUserAttributeProvider(userService)
	err = resolver.RegisterProvider(userProvider)
	require.NoError(t, err)
	
	provider, err := resolver.GetProvider("user")
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "user", provider.Name())
}

func TestAttributeResolver_Resolve_User(t *testing.T) {
	// Setup
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Create mock user service with test data
	userService := providers.NewMockUserService()
	userService.AddUser(&providers.User{
		ID:         "user_123",
		Email:      "alice@example.com",
		Name:       "Alice Smith",
		Roles:      []string{"admin", "developer"},
		Groups:     []string{"engineering", "leadership"},
		OrgID:      "org_456",
		Department: "Engineering",
		Active:     true,
	})
	
	userProvider := providers.NewUserAttributeProvider(userService)
	err := resolver.RegisterProvider(userProvider)
	require.NoError(t, err)
	
	// Test resolution
	ctx := context.Background()
	attrs, err := resolver.Resolve(ctx, "user", "user_123")
	require.NoError(t, err)
	require.NotNil(t, attrs)
	
	// Verify attributes
	assert.Equal(t, "user_123", attrs["id"])
	assert.Equal(t, "alice@example.com", attrs["email"])
	assert.Equal(t, "Alice Smith", attrs["name"])
	assert.Equal(t, []string{"admin", "developer"}, attrs["roles"])
	assert.Equal(t, []string{"engineering", "leadership"}, attrs["groups"])
	assert.Equal(t, "org_456", attrs["org_id"])
	assert.Equal(t, "Engineering", attrs["department"])
	assert.Equal(t, true, attrs["active"])
	
	// Test caching - second call should be faster (from cache)
	attrs2, err := resolver.Resolve(ctx, "user", "user_123")
	require.NoError(t, err)
	assert.Equal(t, attrs, attrs2)
}

func TestAttributeResolver_ResolveBatch_Users(t *testing.T) {
	// Setup
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Create mock user service with multiple users
	userService := providers.NewMockUserService()
	userService.AddUser(&providers.User{
		ID:    "user_1",
		Name:  "Alice",
		Roles: []string{"admin"},
	})
	userService.AddUser(&providers.User{
		ID:    "user_2",
		Name:  "Bob",
		Roles: []string{"developer"},
	})
	userService.AddUser(&providers.User{
		ID:    "user_3",
		Name:  "Charlie",
		Roles: []string{"viewer"},
	})
	
	userProvider := providers.NewUserAttributeProvider(userService)
	err := resolver.RegisterProvider(userProvider)
	require.NoError(t, err)
	
	// Test batch resolution
	ctx := context.Background()
	result, err := resolver.ResolveBatch(ctx, "user", []string{"user_1", "user_2", "user_3"})
	require.NoError(t, err)
	require.Len(t, result, 3)
	
	// Verify each user
	assert.Equal(t, "Alice", result["user_1"]["name"])
	assert.Equal(t, "Bob", result["user_2"]["name"])
	assert.Equal(t, "Charlie", result["user_3"]["name"])
}

func TestAttributeResolver_Resolve_Resource(t *testing.T) {
	// Setup
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Create mock resource service with test data
	resourceService := providers.NewMockResourceService()
	resourceService.AddResource(&providers.Resource{
		ID:           "doc_123",
		Type:         "document",
		Name:         "Q4 Strategy",
		Owner:        "user_456",
		OrgID:        "org_789",
		TeamID:       "team_abc",
		Visibility:   "private",
		Status:       "active",
		Tags:         []string{"strategy", "confidential"},
		Confidential: "internal",
	})
	
	resourceProvider := providers.NewResourceAttributeProvider(resourceService)
	err := resolver.RegisterProvider(resourceProvider)
	require.NoError(t, err)
	
	// Test resolution (key format: "type:id")
	ctx := context.Background()
	attrs, err := resolver.Resolve(ctx, "resource", "document:doc_123")
	require.NoError(t, err)
	require.NotNil(t, attrs)
	
	// Verify attributes
	assert.Equal(t, "doc_123", attrs["id"])
	assert.Equal(t, "document", attrs["type"])
	assert.Equal(t, "Q4 Strategy", attrs["name"])
	assert.Equal(t, "user_456", attrs["owner"])
	assert.Equal(t, "org_789", attrs["org_id"])
	assert.Equal(t, "team_abc", attrs["team_id"])
	assert.Equal(t, "private", attrs["visibility"])
	assert.Equal(t, "active", attrs["status"])
	assert.Equal(t, []string{"strategy", "confidential"}, attrs["tags"])
	assert.Equal(t, "internal", attrs["confidential"])
}

func TestAttributeResolver_Resolve_Context(t *testing.T) {
	// Setup
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Register context provider
	contextProvider := providers.NewContextAttributeProvider()
	err := resolver.RegisterProvider(contextProvider)
	require.NoError(t, err)
	
	// Test resolution
	ctx := context.Background()
	attrs, err := resolver.Resolve(ctx, "context", "current")
	require.NoError(t, err)
	require.NotNil(t, attrs)
	
	// Verify time-based attributes are present
	assert.Contains(t, attrs, "timestamp")
	assert.Contains(t, attrs, "hour")
	assert.Contains(t, attrs, "day_of_week")
	assert.Contains(t, attrs, "is_weekday")
	assert.Contains(t, attrs, "is_weekend")
}

func TestAttributeResolver_ResolveMultiple(t *testing.T) {
	// Setup
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Register providers
	userService := providers.NewMockUserService()
	userService.AddUser(&providers.User{
		ID:    "user_1",
		Name:  "Alice",
		Roles: []string{"admin"},
	})
	userProvider := providers.NewUserAttributeProvider(userService)
	resolver.RegisterProvider(userProvider)
	
	resourceService := providers.NewMockResourceService()
	resourceService.AddResource(&providers.Resource{
		ID:    "doc_1",
		Type:  "document",
		Owner: "user_1",
	})
	resourceProvider := providers.NewResourceAttributeProvider(resourceService)
	resolver.RegisterProvider(resourceProvider)
	
	// Test resolving multiple attributes from different providers
	ctx := context.Background()
	requests := []AttributeRequest{
		{Provider: "user", Key: "user_1"},
		{Provider: "resource", Key: "document:doc_1"},
	}
	
	result, err := resolver.ResolveMultiple(ctx, requests)
	require.NoError(t, err)
	require.Len(t, result, 2)
	
	// Verify both results
	assert.Contains(t, result, "user:user_1")
	assert.Contains(t, result, "resource:document:doc_1")
	assert.Equal(t, "Alice", result["user:user_1"]["name"])
	assert.Equal(t, "user_1", result["resource:document:doc_1"]["owner"])
}

func TestAttributeResolver_CacheHitAndMiss(t *testing.T) {
	// Setup
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Create mock user service
	userService := providers.NewMockUserService()
	userService.AddUser(&providers.User{
		ID:   "user_123",
		Name: "Alice",
	})
	userProvider := providers.NewUserAttributeProvider(userService)
	resolver.RegisterProvider(userProvider)
	
	ctx := context.Background()
	
	// First call - should hit provider (cache miss)
	attrs1, err := resolver.Resolve(ctx, "user", "user_123")
	require.NoError(t, err)
	require.NotNil(t, attrs1)
	
	// Second call - should hit cache
	attrs2, err := resolver.Resolve(ctx, "user", "user_123")
	require.NoError(t, err)
	require.NotNil(t, attrs2)
	assert.Equal(t, attrs1, attrs2)
	
	// Clear cache
	err = resolver.ClearCacheKey(ctx, "user", "user_123")
	require.NoError(t, err)
	
	// Third call - should hit provider again (cache cleared)
	attrs3, err := resolver.Resolve(ctx, "user", "user_123")
	require.NoError(t, err)
	require.NotNil(t, attrs3)
	assert.Equal(t, attrs1, attrs3)
}

func TestAttributeResolver_ClearCache(t *testing.T) {
	// Setup
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	// Create mock user service
	userService := providers.NewMockUserService()
	userService.AddUser(&providers.User{ID: "user_1", Name: "Alice"})
	userService.AddUser(&providers.User{ID: "user_2", Name: "Bob"})
	userProvider := providers.NewUserAttributeProvider(userService)
	resolver.RegisterProvider(userProvider)
	
	ctx := context.Background()
	
	// Populate cache
	resolver.Resolve(ctx, "user", "user_1")
	resolver.Resolve(ctx, "user", "user_2")
	
	// Clear all cache
	err := resolver.ClearCache(ctx)
	assert.NoError(t, err)
	
	// Verify cache is cleared (next resolve should hit provider)
	attrs, err := resolver.Resolve(ctx, "user", "user_1")
	require.NoError(t, err)
	assert.Equal(t, "Alice", attrs["name"])
}

func TestAttributeResolver_ErrorHandling(t *testing.T) {
	attrCache := NewSimpleAttributeCache()
	resolver := NewAttributeResolver(attrCache)
	
	userService := providers.NewMockUserService()
	userProvider := providers.NewUserAttributeProvider(userService)
	resolver.RegisterProvider(userProvider)
	
	ctx := context.Background()
	
	// Test non-existent user
	_, err := resolver.Resolve(ctx, "user", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
	
	// Test non-existent provider
	_, err = resolver.Resolve(ctx, "nonexistent_provider", "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

