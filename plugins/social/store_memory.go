package social

import (
	"context"
	"errors"
	"sync"

	"github.com/xraph/authsome/id"
)

// ErrConnectionNotFound is returned when an OAuth connection is not found.
var ErrConnectionNotFound = errors.New("social: oauth connection not found")

// MemoryStore is an in-memory Store for testing.
type MemoryStore struct {
	mu    sync.RWMutex
	conns map[id.OAuthConnectionID]*OAuthConnection
}

// NewMemoryStore creates a new in-memory OAuth connection store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conns: make(map[id.OAuthConnectionID]*OAuthConnection),
	}
}

var _ Store = (*MemoryStore)(nil)

// CreateOAuthConnection stores a new OAuth connection.
func (s *MemoryStore) CreateOAuthConnection(_ context.Context, c *OAuthConnection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[c.ID] = c
	return nil
}

// GetOAuthConnection finds a connection by provider and provider user ID.
func (s *MemoryStore) GetOAuthConnection(_ context.Context, provider, providerUserID string) (*OAuthConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.conns {
		if c.Provider == provider && c.ProviderUserID == providerUserID {
			return c, nil
		}
	}
	return nil, ErrConnectionNotFound
}

// GetOAuthConnectionsByUserID returns all connections for a user.
func (s *MemoryStore) GetOAuthConnectionsByUserID(_ context.Context, userID id.UserID) ([]*OAuthConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*OAuthConnection
	for _, c := range s.conns {
		if c.UserID == userID {
			result = append(result, c)
		}
	}
	return result, nil
}

// DeleteOAuthConnection removes a connection by ID.
func (s *MemoryStore) DeleteOAuthConnection(_ context.Context, connID id.OAuthConnectionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conns[connID]; !ok {
		return ErrConnectionNotFound
	}
	delete(s.conns, connID)
	return nil
}
