package ceremony

import (
	"context"
	"sync"
	"time"
)

// entry holds a single ceremony value with its expiration time.
type entry struct {
	data      []byte
	expiresAt time.Time
}

// MemoryStore is an in-memory ceremony store with TTL-based expiration.
// It is safe for concurrent use and suitable for single-instance
// deployments or testing.
type MemoryStore struct {
	mu      sync.RWMutex
	entries map[string]*entry
}

var _ Store = (*MemoryStore)(nil)

// NewMemory creates a new in-memory ceremony store.
func NewMemory() *MemoryStore {
	return &MemoryStore{
		entries: make(map[string]*entry),
	}
}

// Set stores data under key with a TTL.
func (s *MemoryStore) Set(_ context.Context, key string, data []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[key] = &entry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Get retrieves data by key. Returns ErrNotFound if absent or expired.
func (s *MemoryStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}
	if time.Now().After(e.expiresAt) {
		// Lazy cleanup of expired entry.
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return nil, ErrNotFound
	}
	return e.data, nil
}

// Delete removes data by key (idempotent).
func (s *MemoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}
