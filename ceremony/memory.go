package ceremony

import (
	"context"
	"sync"
	"time"
)

// entry holds a single ceremony value with its expiration time.
type entry struct {
	data      []byte
	counter   int64
	expiresAt time.Time
}

// DefaultMaxEntries bounds the memory store. Ceremony state is short-lived,
// so the bound is only reached under a flood; when it is, expired entries go
// first and then arbitrary ones, which costs a caller a retry rather than
// costing the process its memory.
const DefaultMaxEntries = 100_000

// MemoryStore is an in-memory ceremony store with TTL-based expiration.
// It is safe for concurrent use and suitable for single-instance
// deployments or testing.
type MemoryStore struct {
	mu         sync.RWMutex
	entries    map[string]*entry
	maxEntries int
}

var _ Store = (*MemoryStore)(nil)

// NewMemory creates a new in-memory ceremony store.
func NewMemory() *MemoryStore {
	return &MemoryStore{
		entries:    make(map[string]*entry),
		maxEntries: DefaultMaxEntries,
	}
}

// NewMemoryWithLimit creates a memory store bounded to maxEntries.
func NewMemoryWithLimit(maxEntries int) *MemoryStore {
	s := NewMemory()
	if maxEntries > 0 {
		s.maxEntries = maxEntries
	}
	return s
}

// Set stores data under key with a TTL.
func (s *MemoryStore) Set(_ context.Context, key string, data []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.makeRoomLocked(key)
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

// SetNX stores data only when key is absent or expired.
func (s *MemoryStore) SetNX(_ context.Context, key string, data []byte, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if e, ok := s.entries[key]; ok && now.Before(e.expiresAt) {
		return false, nil
	}
	s.makeRoomLocked(key)
	s.entries[key] = &entry{data: data, expiresAt: now.Add(ttl)}
	return true, nil
}

// Increment adds one to the counter under key.
func (s *MemoryStore) Increment(_ context.Context, key string, ttl time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	e, ok := s.entries[key]
	if !ok || !now.Before(e.expiresAt) {
		s.makeRoomLocked(key)
		e = &entry{expiresAt: now.Add(ttl)}
		s.entries[key] = e
	}
	e.counter++
	return e.counter, nil
}

// Len reports how many entries the store holds, expired ones included.
func (s *MemoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// makeRoomLocked keeps the store under its bound before key is added:
// expired entries are dropped first, then arbitrary ones. Called with the
// write lock held.
func (s *MemoryStore) makeRoomLocked(key string) {
	if _, exists := s.entries[key]; exists || len(s.entries) < s.maxEntries {
		return
	}
	now := time.Now()
	for k, e := range s.entries {
		if now.After(e.expiresAt) {
			delete(s.entries, k)
		}
	}
	for k := range s.entries {
		if len(s.entries) < s.maxEntries {
			break
		}
		delete(s.entries, k)
	}
}
