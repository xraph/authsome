package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/permissions/engine"
)

// Cache interface defined in interfaces.go

// =============================================================================
// MEMORY CACHE WITH LRU
// =============================================================================

// cacheEntry represents a cached item with expiration.
type cacheEntry struct {
	policy     *engine.CompiledPolicy
	expiration time.Time
	lastAccess time.Time
}

// MemoryCache is an in-memory LRU cache implementation
// V2 Architecture: App → Environment → Organization.
type MemoryCache struct {
	data       map[string]*cacheEntry
	mu         sync.RWMutex
	maxSize    int
	defaultTTL time.Duration

	// Stats
	hits      int64
	misses    int64
	evictions int64
}

// CacheConfig holds cache configuration.
type CacheConfig struct {
	MaxSize    int           `json:"maxSize"    yaml:"maxSize"`
	DefaultTTL time.Duration `json:"defaultTtl" yaml:"defaultTtl"`
	Backend    string        `json:"backend"    yaml:"backend"` // memory, redis, hybrid
}

// NewMemoryCache creates a new memory cache.
func NewMemoryCache(config any) Cache {
	cfg, ok := config.(CacheConfig)
	if !ok {
		cfg = CacheConfig{
			MaxSize:    10000,
			DefaultTTL: 5 * time.Minute,
		}
	}

	if cfg.MaxSize == 0 {
		cfg.MaxSize = 10000
	}

	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}

	return &MemoryCache{
		data:       make(map[string]*cacheEntry),
		maxSize:    cfg.MaxSize,
		defaultTTL: cfg.DefaultTTL,
	}
}

// Get retrieves a compiled policy from cache.
func (c *MemoryCache) Get(ctx context.Context, key string) (*engine.CompiledPolicy, error) {
	c.mu.RLock()
	entry, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		atomic.AddInt64(&c.misses, 1)

		return nil, nil
	}

	// Check expiration
	if time.Now().After(entry.expiration) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		atomic.AddInt64(&c.misses, 1)
		atomic.AddInt64(&c.evictions, 1)

		return nil, nil
	}

	// Update last access time for LRU
	c.mu.Lock()

	entry.lastAccess = time.Now()

	c.mu.Unlock()

	atomic.AddInt64(&c.hits, 1)

	return entry.policy, nil
}

// Set stores a compiled policy in cache.
func (c *MemoryCache) Set(ctx context.Context, key string, policy *engine.CompiledPolicy, ttl time.Duration) error {
	if policy == nil {
		return nil
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.data) >= c.maxSize {
		c.evictLRU()
	}

	c.data[key] = &cacheEntry{
		policy:     policy,
		expiration: time.Now().Add(ttl),
		lastAccess: time.Now(),
	}

	return nil
}

// Delete removes a policy from cache.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()

	return nil
}

// DeleteByApp removes all policies for an app.
func (c *MemoryCache) DeleteByApp(ctx context.Context, appID xid.ID) error {
	prefix := appID.String() + ":"

	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.data {
		if strings.HasPrefix(key, prefix) {
			delete(c.data, key)
			atomic.AddInt64(&c.evictions, 1)
		}
	}

	return nil
}

// DeleteByEnvironment removes all policies for an environment.
func (c *MemoryCache) DeleteByEnvironment(ctx context.Context, appID, envID xid.ID) error {
	prefix := appID.String() + ":" + envID.String() + ":"

	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.data {
		if strings.HasPrefix(key, prefix) {
			delete(c.data, key)
			atomic.AddInt64(&c.evictions, 1)
		}
	}

	return nil
}

// DeleteByOrganization removes all policies for an organization.
func (c *MemoryCache) DeleteByOrganization(ctx context.Context, appID, envID, userOrgID xid.ID) error {
	prefix := appID.String() + ":" + envID.String() + ":" + userOrgID.String() + ":"

	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.data {
		if strings.HasPrefix(key, prefix) {
			delete(c.data, key)
			atomic.AddInt64(&c.evictions, 1)
		}
	}

	return nil
}

// GetMulti retrieves multiple policies.
func (c *MemoryCache) GetMulti(ctx context.Context, keys []string) (map[string]*engine.CompiledPolicy, error) {
	result := make(map[string]*engine.CompiledPolicy, len(keys))

	for _, key := range keys {
		policy, err := c.Get(ctx, key)
		if err != nil {
			continue
		}

		if policy != nil {
			result[key] = policy
		}
	}

	return result, nil
}

// SetMulti stores multiple policies.
func (c *MemoryCache) SetMulti(ctx context.Context, policies map[string]*engine.CompiledPolicy, ttl time.Duration) error {
	for key, policy := range policies {
		if err := c.Set(ctx, key, policy, ttl); err != nil {
			return err
		}
	}

	return nil
}

// Stats returns cache statistics.
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	size := int64(len(c.data))
	c.mu.RUnlock()

	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)
	evictions := atomic.LoadInt64(&c.evictions)

	var hitRate float64

	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return CacheStats{
		Hits:        hits,
		Misses:      misses,
		Evictions:   evictions,
		Size:        size,
		HitRate:     hitRate,
		LastUpdated: time.Now(),
	}
}

// evictLRU removes the least recently used entry.
func (c *MemoryCache) evictLRU() {
	var (
		oldestKey  string
		oldestTime time.Time
	)

	for key, entry := range c.data {
		if oldestKey == "" || entry.lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastAccess
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
		atomic.AddInt64(&c.evictions, 1)
	}
}

// =============================================================================
// REDIS CACHE
// =============================================================================

// RedisCache is a Redis-backed cache implementation
// V2 Architecture: App → Environment → Organization.
type RedisCache struct {
	client     *redis.Client
	keyPrefix  string
	defaultTTL time.Duration

	// Stats (approximated)
	hits      int64
	misses    int64
	evictions int64
}

// NewRedisCache creates a new Redis cache.
func NewRedisCache(client *redis.Client, config any) Cache {
	cfg, ok := config.(CacheConfig)
	if !ok {
		cfg = CacheConfig{
			DefaultTTL: 5 * time.Minute,
		}
	}

	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}

	return &RedisCache{
		client:     client,
		keyPrefix:  "permissions:policy:",
		defaultTTL: cfg.DefaultTTL,
	}
}

// compiledPolicyJSON is a JSON-serializable version of CompiledPolicy.
type compiledPolicyJSON struct {
	PolicyID           string    `json:"policyId"`
	AppID              string    `json:"appId"`
	EnvironmentID      string    `json:"environmentId"`
	UserOrganizationID *string   `json:"userOrganizationId,omitempty"`
	NamespaceID        string    `json:"namespaceId"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	ResourceType       string    `json:"resourceType"`
	Actions            []string  `json:"actions"`
	Priority           int       `json:"priority"`
	Version            int       `json:"version"`
	CompiledAt         time.Time `json:"compiledAt"`
	// Note: Program and AST cannot be serialized to Redis
	// They need to be recompiled from the original expression
}

// Get retrieves a compiled policy from Redis
// Note: This returns nil because CEL programs cannot be serialized
// Use Redis cache for metadata caching only.
func (c *RedisCache) Get(ctx context.Context, key string) (*engine.CompiledPolicy, error) {
	if c.client == nil {
		return nil, nil
	}

	result, err := c.client.Get(ctx, c.keyPrefix+key).Result()
	if errors.Is(err, redis.Nil) {
		atomic.AddInt64(&c.misses, 1)

		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	// Note: We can deserialize metadata but not the CEL Program
	// This is useful for invalidation and stats tracking
	var data compiledPolicyJSON
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached policy: %w", err)
	}

	// Return a partial CompiledPolicy (without Program)
	// The caller needs to recompile if Program is needed
	policy := &engine.CompiledPolicy{
		Name:         data.Name,
		Description:  data.Description,
		ResourceType: data.ResourceType,
		Actions:      data.Actions,
		Priority:     data.Priority,
		Version:      data.Version,
		CompiledAt:   data.CompiledAt,
		// Program and AST are nil - need recompilation
	}

	// Parse XIDs
	if id, err := xid.FromString(data.PolicyID); err == nil {
		policy.PolicyID = id
	}

	if id, err := xid.FromString(data.AppID); err == nil {
		policy.AppID = id
	}

	if id, err := xid.FromString(data.EnvironmentID); err == nil {
		policy.EnvironmentID = id
	}

	if id, err := xid.FromString(data.NamespaceID); err == nil {
		policy.NamespaceID = id
	}

	if data.UserOrganizationID != nil {
		if id, err := xid.FromString(*data.UserOrganizationID); err == nil {
			policy.UserOrganizationID = &id
		}
	}

	atomic.AddInt64(&c.hits, 1)

	return policy, nil
}

// Set stores policy metadata in Redis.
func (c *RedisCache) Set(ctx context.Context, key string, policy *engine.CompiledPolicy, ttl time.Duration) error {
	if c.client == nil || policy == nil {
		return nil
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	// Serialize to JSON (without Program and AST)
	data := compiledPolicyJSON{
		PolicyID:      policy.PolicyID.String(),
		AppID:         policy.AppID.String(),
		EnvironmentID: policy.EnvironmentID.String(),
		NamespaceID:   policy.NamespaceID.String(),
		Name:          policy.Name,
		Description:   policy.Description,
		ResourceType:  policy.ResourceType,
		Actions:       policy.Actions,
		Priority:      policy.Priority,
		Version:       policy.Version,
		CompiledAt:    policy.CompiledAt,
	}

	if policy.UserOrganizationID != nil {
		orgID := policy.UserOrganizationID.String()
		data.UserOrganizationID = &orgID
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	return c.client.Set(ctx, c.keyPrefix+key, jsonData, ttl).Err()
}

// Delete removes a policy from Redis.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if c.client == nil {
		return nil
	}

	return c.client.Del(ctx, c.keyPrefix+key).Err()
}

// DeleteByApp removes all policies for an app using pattern matching.
func (c *RedisCache) DeleteByApp(ctx context.Context, appID xid.ID) error {
	if c.client == nil {
		return nil
	}

	pattern := c.keyPrefix + appID.String() + ":*"

	return c.deleteByPattern(ctx, pattern)
}

// DeleteByEnvironment removes all policies for an environment.
func (c *RedisCache) DeleteByEnvironment(ctx context.Context, appID, envID xid.ID) error {
	if c.client == nil {
		return nil
	}

	pattern := c.keyPrefix + appID.String() + ":" + envID.String() + ":*"

	return c.deleteByPattern(ctx, pattern)
}

// DeleteByOrganization removes all policies for an organization.
func (c *RedisCache) DeleteByOrganization(ctx context.Context, appID, envID, userOrgID xid.ID) error {
	if c.client == nil {
		return nil
	}

	pattern := c.keyPrefix + appID.String() + ":" + envID.String() + ":" + userOrgID.String() + ":*"

	return c.deleteByPattern(ctx, pattern)
}

// deleteByPattern deletes all keys matching a pattern.
func (c *RedisCache) deleteByPattern(ctx context.Context, pattern string) error {
	var (
		cursor uint64
		err    error
	)

	for {
		var keys []string

		keys, cursor, err = c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis scan failed: %w", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("redis delete failed: %w", err)
			}

			atomic.AddInt64(&c.evictions, int64(len(keys)))
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// GetMulti retrieves multiple policies from Redis.
func (c *RedisCache) GetMulti(ctx context.Context, keys []string) (map[string]*engine.CompiledPolicy, error) {
	result := make(map[string]*engine.CompiledPolicy, len(keys))

	for _, key := range keys {
		policy, err := c.Get(ctx, key)
		if err != nil {
			continue
		}

		if policy != nil {
			result[key] = policy
		}
	}

	return result, nil
}

// SetMulti stores multiple policies in Redis.
func (c *RedisCache) SetMulti(ctx context.Context, policies map[string]*engine.CompiledPolicy, ttl time.Duration) error {
	for key, policy := range policies {
		if err := c.Set(ctx, key, policy, ttl); err != nil {
			return err
		}
	}

	return nil
}

// Stats returns cache statistics.
func (c *RedisCache) Stats() CacheStats {
	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)
	evictions := atomic.LoadInt64(&c.evictions)

	var hitRate float64

	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	// Try to get size from Redis
	var size int64

	if c.client != nil {
		ctx := context.Background()

		keys, err := c.client.Keys(ctx, c.keyPrefix+"*").Result()
		if err == nil {
			size = int64(len(keys))
		}
	}

	return CacheStats{
		Hits:        hits,
		Misses:      misses,
		Evictions:   evictions,
		Size:        size,
		HitRate:     hitRate,
		LastUpdated: time.Now(),
	}
}

// =============================================================================
// HYBRID CACHE (Memory + Redis)
// =============================================================================

// HybridCache combines memory and Redis caching.
type HybridCache struct {
	memory *MemoryCache
	redis  *RedisCache
}

// NewHybridCache creates a new hybrid cache.
func NewHybridCache(redisClient *redis.Client, config any) Cache {
	return &HybridCache{
		memory: NewMemoryCache(config).(*MemoryCache),
		redis:  NewRedisCache(redisClient, config).(*RedisCache),
	}
}

// Get retrieves from memory first, then Redis.
func (c *HybridCache) Get(ctx context.Context, key string) (*engine.CompiledPolicy, error) {
	// Try memory first
	policy, err := c.memory.Get(ctx, key)
	if err == nil && policy != nil {
		return policy, nil
	}

	// Fall back to Redis
	policy, err = c.redis.Get(ctx, key)
	if err == nil && policy != nil {
		// Promote to memory cache
		_ = c.memory.Set(ctx, key, policy, 0)
	}

	return policy, err
}

// Set stores in both memory and Redis.
func (c *HybridCache) Set(ctx context.Context, key string, policy *engine.CompiledPolicy, ttl time.Duration) error {
	// Store in memory
	if err := c.memory.Set(ctx, key, policy, ttl); err != nil {
		return err
	}

	// Store in Redis (for distributed cache)
	return c.redis.Set(ctx, key, policy, ttl)
}

// Delete removes from both caches.
func (c *HybridCache) Delete(ctx context.Context, key string) error {
	_ = c.memory.Delete(ctx, key)

	return c.redis.Delete(ctx, key)
}

// DeleteByApp removes from both caches.
func (c *HybridCache) DeleteByApp(ctx context.Context, appID xid.ID) error {
	_ = c.memory.DeleteByApp(ctx, appID)

	return c.redis.DeleteByApp(ctx, appID)
}

// DeleteByEnvironment removes from both caches.
func (c *HybridCache) DeleteByEnvironment(ctx context.Context, appID, envID xid.ID) error {
	_ = c.memory.DeleteByEnvironment(ctx, appID, envID)

	return c.redis.DeleteByEnvironment(ctx, appID, envID)
}

// DeleteByOrganization removes from both caches.
func (c *HybridCache) DeleteByOrganization(ctx context.Context, appID, envID, userOrgID xid.ID) error {
	_ = c.memory.DeleteByOrganization(ctx, appID, envID, userOrgID)

	return c.redis.DeleteByOrganization(ctx, appID, envID, userOrgID)
}

// GetMulti retrieves from memory first, then Redis.
func (c *HybridCache) GetMulti(ctx context.Context, keys []string) (map[string]*engine.CompiledPolicy, error) {
	result := make(map[string]*engine.CompiledPolicy, len(keys))

	var missingKeys []string

	// Try memory first
	for _, key := range keys {
		policy, _ := c.memory.Get(ctx, key)
		if policy != nil {
			result[key] = policy
		} else {
			missingKeys = append(missingKeys, key)
		}
	}

	// Get missing from Redis
	if len(missingKeys) > 0 {
		redisResults, _ := c.redis.GetMulti(ctx, missingKeys)
		for key, policy := range redisResults {
			result[key] = policy
			// Promote to memory
			_ = c.memory.Set(ctx, key, policy, 0)
		}
	}

	return result, nil
}

// SetMulti stores in both caches.
func (c *HybridCache) SetMulti(ctx context.Context, policies map[string]*engine.CompiledPolicy, ttl time.Duration) error {
	for key, policy := range policies {
		if err := c.Set(ctx, key, policy, ttl); err != nil {
			return err
		}
	}

	return nil
}

// Stats returns combined cache statistics.
func (c *HybridCache) Stats() CacheStats {
	memStats := c.memory.Stats()
	redisStats := c.redis.Stats()

	return CacheStats{
		Hits:        memStats.Hits + redisStats.Hits,
		Misses:      memStats.Misses + redisStats.Misses,
		Evictions:   memStats.Evictions + redisStats.Evictions,
		Size:        memStats.Size + redisStats.Size,
		HitRate:     memStats.HitRate, // Use memory hit rate as primary
		LastUpdated: time.Now(),
	}
}
