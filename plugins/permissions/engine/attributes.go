package engine

import (
	"context"
	"fmt"
	"time"
)

// AttributeProvider fetches attributes from various sources (users, resources, etc.)
type AttributeProvider interface {
	// Name returns the provider name (e.g., "user", "resource", "request")
	Name() string

	// GetAttributes fetches attributes for a given entity
	// The key format depends on the provider (e.g., "user:123", "resource:doc_456")
	GetAttributes(ctx context.Context, key string) (map[string]interface{}, error)

	// GetBatchAttributes fetches attributes for multiple entities
	// Returns a map of key -> attributes
	GetBatchAttributes(ctx context.Context, keys []string) (map[string]map[string]interface{}, error)
}

// AttributeResolver coordinates multiple providers and handles caching
type AttributeResolver struct {
	providers map[string]AttributeProvider
	cache     AttributeCache
}

// AttributeCache defines the caching interface for attributes
// This is intentionally simple and can wrap any caching backend
type AttributeCache interface {
	Get(ctx context.Context, key string) (map[string]interface{}, bool)
	Set(ctx context.Context, key string, value map[string]interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// simpleAttributeCache is a simple in-memory implementation for testing
type simpleAttributeCache struct {
	data map[string]map[string]interface{}
}

// NewSimpleAttributeCache creates a simple in-memory attribute cache
func NewSimpleAttributeCache() AttributeCache {
	return &simpleAttributeCache{
		data: make(map[string]map[string]interface{}),
	}
}

func (c *simpleAttributeCache) Get(ctx context.Context, key string) (map[string]interface{}, bool) {
	val, ok := c.data[key]
	return val, ok
}

func (c *simpleAttributeCache) Set(ctx context.Context, key string, value map[string]interface{}, ttl time.Duration) error {
	c.data[key] = value
	return nil
}

func (c *simpleAttributeCache) Delete(ctx context.Context, key string) error {
	delete(c.data, key)
	return nil
}

func (c *simpleAttributeCache) Clear(ctx context.Context) error {
	c.data = make(map[string]map[string]interface{})
	return nil
}

// NewAttributeResolver creates a new attribute resolver
func NewAttributeResolver(cache AttributeCache) *AttributeResolver {
	return &AttributeResolver{
		providers: make(map[string]AttributeProvider),
		cache:     cache,
	}
}

// RegisterProvider registers an attribute provider
func (r *AttributeResolver) RegisterProvider(provider AttributeProvider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider '%s' already registered", name)
	}

	r.providers[name] = provider
	return nil
}

// GetProvider returns a registered provider by name
func (r *AttributeResolver) GetProvider(name string) (AttributeProvider, error) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}
	return provider, nil
}

// Resolve fetches attributes for a given provider and key
// It checks the cache first, then falls back to the provider
func (r *AttributeResolver) Resolve(ctx context.Context, providerName, key string) (map[string]interface{}, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", providerName, key)
	if r.cache != nil {
		if attrs, found := r.cache.Get(ctx, cacheKey); found {
			return attrs, nil
		}
	}

	// Get provider
	provider, err := r.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Fetch from provider
	attrs, err := provider.GetAttributes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes from provider '%s': %w", providerName, err)
	}

	// Cache the result
	if r.cache != nil {
		// Default TTL: 5 minutes for most attributes
		ttl := 5 * time.Minute
		if err := r.cache.Set(ctx, cacheKey, attrs, ttl); err != nil {
			// Log but don't fail the request if caching fails
			// In production, you'd want proper logging here
			_ = err
		}
	}

	return attrs, nil
}

// ResolveBatch fetches attributes for multiple entities from a provider
func (r *AttributeResolver) ResolveBatch(ctx context.Context, providerName string, keys []string) (map[string]map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]map[string]interface{}), nil
	}

	// Get provider
	provider, err := r.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Check which keys are in cache
	result := make(map[string]map[string]interface{})
	uncachedKeys := make([]string, 0, len(keys))

	if r.cache != nil {
		for _, key := range keys {
			cacheKey := fmt.Sprintf("%s:%s", providerName, key)
			if attrs, found := r.cache.Get(ctx, cacheKey); found {
				result[key] = attrs
			} else {
				uncachedKeys = append(uncachedKeys, key)
			}
		}
	} else {
		uncachedKeys = keys
	}

	// Fetch uncached keys from provider
	if len(uncachedKeys) > 0 {
		batchResult, err := provider.GetBatchAttributes(ctx, uncachedKeys)
		if err != nil {
			return nil, fmt.Errorf("failed to get batch attributes from provider '%s': %w", providerName, err)
		}

		// Merge with cached results and update cache
		for key, attrs := range batchResult {
			result[key] = attrs

			if r.cache != nil {
				cacheKey := fmt.Sprintf("%s:%s", providerName, key)
				ttl := 5 * time.Minute
				if err := r.cache.Set(ctx, cacheKey, attrs, ttl); err != nil {
					_ = err // Log but don't fail
				}
			}
		}
	}

	return result, nil
}

// EnrichEvaluationContext enriches an evaluation context with additional attributes
// This is called before policy evaluation to ensure all needed attributes are present
func (r *AttributeResolver) EnrichEvaluationContext(ctx context.Context, evalCtx *EvaluationContext) error {
	// For now, this is a no-op
	// In a full implementation, this would:
	// 1. Analyze which attributes are needed based on the policies
	// 2. Fetch missing attributes from providers
	// 3. Add them to the evaluation context

	// Example future implementation:
	// - If principal.id is set but principal.roles is missing, fetch from user provider
	// - If resource.type and resource.id are set but resource.owner is missing, fetch from resource provider
	// - Add request context attributes (IP, time, location) if not present

	return nil
}

// AttributeRequest represents a request for attributes
type AttributeRequest struct {
	Provider string
	Key      string
}

// ResolveMultiple fetches attributes for multiple requests in parallel
func (r *AttributeResolver) ResolveMultiple(ctx context.Context, requests []AttributeRequest) (map[string]map[string]interface{}, error) {
	// Group requests by provider for batch fetching
	providerRequests := make(map[string][]string)
	for _, req := range requests {
		providerRequests[req.Provider] = append(providerRequests[req.Provider], req.Key)
	}

	// Fetch from each provider
	result := make(map[string]map[string]interface{})
	for providerName, keys := range providerRequests {
		batchResult, err := r.ResolveBatch(ctx, providerName, keys)
		if err != nil {
			return nil, err
		}

		// Merge results with provider prefix
		for key, attrs := range batchResult {
			fullKey := fmt.Sprintf("%s:%s", providerName, key)
			result[fullKey] = attrs
		}
	}

	return result, nil
}

// ClearCache clears all cached attributes
func (r *AttributeResolver) ClearCache(ctx context.Context) error {
	if r.cache == nil {
		return nil
	}
	return r.cache.Clear(ctx)
}

// ClearCacheKey clears a specific cached attribute
func (r *AttributeResolver) ClearCacheKey(ctx context.Context, providerName, key string) error {
	if r.cache == nil {
		return nil
	}
	cacheKey := fmt.Sprintf("%s:%s", providerName, key)
	return r.cache.Delete(ctx, cacheKey)
}
