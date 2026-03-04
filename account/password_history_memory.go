package account

import (
	"context"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryPasswordHistoryStore is an in-memory password history store
// for development and testing.
type MemoryPasswordHistoryStore struct {
	mu      sync.RWMutex
	entries map[string][]PasswordHistoryEntry // keyed by user ID
}

// NewMemoryPasswordHistoryStore creates an in-memory password history store.
func NewMemoryPasswordHistoryStore() *MemoryPasswordHistoryStore {
	return &MemoryPasswordHistoryStore{
		entries: make(map[string][]PasswordHistoryEntry),
	}
}

func (s *MemoryPasswordHistoryStore) SavePasswordHash(_ context.Context, userID id.UserID, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userID.String()
	s.entries[key] = append(s.entries[key], PasswordHistoryEntry{
		ID:        id.NewUserID().String(), // reuse for unique ID
		UserID:    userID,
		Hash:      hash,
		CreatedAt: time.Now(),
	})
	return nil
}

func (s *MemoryPasswordHistoryStore) GetPasswordHistory(_ context.Context, userID id.UserID, limit int) ([]PasswordHistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.entries[userID.String()]
	if len(entries) == 0 {
		return nil, nil
	}

	// Return the most recent `limit` entries (entries are appended, so last N).
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}
	start := len(entries) - limit
	result := make([]PasswordHistoryEntry, limit)
	copy(result, entries[start:])
	return result, nil
}
