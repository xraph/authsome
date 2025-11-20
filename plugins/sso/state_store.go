package sso

import (
	"context"
	"sync"
	"time"
)

// OIDCState represents OIDC flow state data
type OIDCState struct {
	State        string
	Nonce        string
	CodeVerifier string
	ProviderID   string
	RedirectURI  string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// StateStore provides temporary storage for OIDC flow state
// In production, this should be backed by Redis or similar distributed cache
type StateStore struct {
	mu     sync.RWMutex
	states map[string]*OIDCState // keyed by state parameter
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	store := &StateStore{
		states: make(map[string]*OIDCState),
	}
	
	// Start background cleanup goroutine
	go store.cleanup()
	
	return store
}

// Store saves OIDC state data
func (s *StateStore) Store(ctx context.Context, state *OIDCState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Set expiration if not already set (default 10 minutes)
	if state.ExpiresAt.IsZero() {
		state.ExpiresAt = time.Now().Add(10 * time.Minute)
	}
	
	s.states[state.State] = state
	return nil
}

// Get retrieves OIDC state data by state parameter
func (s *StateStore) Get(ctx context.Context, state string) (*OIDCState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	oidcState, ok := s.states[state]
	if !ok {
		return nil, nil // Not found
	}
	
	// Check if expired
	if time.Now().After(oidcState.ExpiresAt) {
		return nil, nil // Expired
	}
	
	return oidcState, nil
}

// Delete removes OIDC state data (should be called after successful callback)
func (s *StateStore) Delete(ctx context.Context, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.states, state)
	return nil
}

// cleanup periodically removes expired state entries
func (s *StateStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, state := range s.states {
			if now.After(state.ExpiresAt) {
				delete(s.states, key)
			}
		}
		s.mu.Unlock()
	}
}

// =============================================================================
// REDIS-BACKED STATE STORE (for production)
// =============================================================================

// RedisStateStore is a production-ready state store backed by Redis
// This is a placeholder interface for future implementation
type RedisStateStore struct {
	// redis client would go here
}

// NewRedisStateStore creates a Redis-backed state store
// func NewRedisStateStore(redisClient *redis.Client) *RedisStateStore {
// 	return &RedisStateStore{}
// }

// Store saves OIDC state in Redis with TTL
// func (r *RedisStateStore) Store(ctx context.Context, state *OIDCState) error {
// 	// Marshal to JSON and store in Redis with TTL
// 	return nil
// }

// Get retrieves OIDC state from Redis
// func (r *RedisStateStore) Get(ctx context.Context, state string) (*OIDCState, error) {
// 	// Fetch from Redis and unmarshal
// 	return nil, nil
// }

// Delete removes OIDC state from Redis
// func (r *RedisStateStore) Delete(ctx context.Context, state string) error {
// 	// Delete from Redis
// 	return nil
// }

