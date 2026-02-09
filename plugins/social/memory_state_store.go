package social

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
)

// StateStore defines the interface for OAuth state storage.
type StateStore interface {
	Set(ctx context.Context, key string, state *OAuthState, ttl time.Duration) error
	Get(ctx context.Context, key string) (*OAuthState, error)
	Delete(ctx context.Context, key string) error
}

// OAuthState stores temporary OAuth state data.
type OAuthState struct {
	Provider           string    `json:"provider"`
	AppID              xid.ID    `json:"app_id"`
	UserOrganizationID *xid.ID   `json:"user_organization_id,omitempty"`
	RedirectURL        string    `json:"redirect_url,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	ExtraScopes        []string  `json:"extra_scopes,omitempty"`
	LinkUserID         *xid.ID   `json:"link_user_id,omitempty"`
}

// =============================================================================
// MEMORY STATE STORE
// =============================================================================

// MemoryStateStore is an in-memory implementation of StateStore.
type MemoryStateStore struct {
	mu     sync.RWMutex
	states map[string]*stateEntry
}

type stateEntry struct {
	state     *OAuthState
	expiresAt time.Time
}

// NewMemoryStateStore creates a new in-memory state store.
func NewMemoryStateStore() *MemoryStateStore {
	s := &MemoryStateStore{
		states: make(map[string]*stateEntry),
	}

	// Start cleanup goroutine
	go s.cleanup()

	return s
}

// Set stores a state with TTL.
func (s *MemoryStateStore) Set(ctx context.Context, key string, state *OAuthState, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[key] = &stateEntry{
		state:     state,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Get retrieves a state.
func (s *MemoryStateStore) Get(ctx context.Context, key string) (*OAuthState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.states[key]
	if !ok {
		return nil, errs.NotFound("state not found")
	}

	if time.Now().After(entry.expiresAt) {
		return nil, errs.BadRequest("state expired")
	}

	return entry.state, nil
}

// Delete removes a state.
func (s *MemoryStateStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, key)

	return nil
}

// cleanup periodically removes expired states.
func (s *MemoryStateStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()

		now := time.Now()
		for key, entry := range s.states {
			if now.After(entry.expiresAt) {
				delete(s.states, key)
			}
		}

		s.mu.Unlock()
	}
}

// =============================================================================
// REDIS STATE STORE
// =============================================================================

// RedisStateStore is a Redis-backed implementation of StateStore.
type RedisStateStore struct {
	client *redis.Client
}

// NewRedisStateStore creates a new Redis state store.
func NewRedisStateStore(client *redis.Client) *RedisStateStore {
	return &RedisStateStore{
		client: client,
	}
}

// Set stores a state with TTL in Redis.
func (s *RedisStateStore) Set(ctx context.Context, key string, state *OAuthState, ttl time.Duration) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Prefix key to avoid collisions
	redisKey := "oauth_state:" + key

	if err := s.client.Set(ctx, redisKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store state in Redis: %w", err)
	}

	return nil
}

// Get retrieves a state from Redis.
func (s *RedisStateStore) Get(ctx context.Context, key string) (*OAuthState, error) {
	redisKey := "oauth_state:" + key

	data, err := s.client.Get(ctx, redisKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, errs.NotFound("state not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve state from Redis: %w", err)
	}

	var state OAuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Delete removes a state from Redis.
func (s *RedisStateStore) Delete(ctx context.Context, key string) error {
	redisKey := "oauth_state:" + key

	if err := s.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to delete state from Redis: %w", err)
	}

	return nil
}

// Ensure implementations satisfy the interface.
var (
	_ StateStore = (*MemoryStateStore)(nil)
	_ StateStore = (*RedisStateStore)(nil)
)
