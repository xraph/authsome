package sso

import (
	"context"
	"errors"
	"sync"

	"github.com/xraph/authsome/id"
)

// ErrConnectionNotFound is returned when an SSO connection is not found.
var ErrConnectionNotFound = errors.New("sso: connection not found")

// MemoryStore is an in-memory Store for testing.
type MemoryStore struct {
	mu    sync.RWMutex
	conns map[id.SSOConnectionID]*SSOConnection
}

// NewMemoryStore creates a new in-memory SSO connection store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conns: make(map[id.SSOConnectionID]*SSOConnection),
	}
}

var _ Store = (*MemoryStore)(nil)

func (s *MemoryStore) CreateSSOConnection(_ context.Context, c *SSOConnection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[c.ID] = c
	return nil
}

func (s *MemoryStore) GetSSOConnection(_ context.Context, connID id.SSOConnectionID) (*SSOConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.conns[connID]
	if !ok {
		return nil, ErrConnectionNotFound
	}
	return c, nil
}

func (s *MemoryStore) GetSSOConnectionByDomain(_ context.Context, appID id.AppID, domain string) (*SSOConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.conns {
		if c.AppID == appID && c.Domain == domain && c.Active {
			return c, nil
		}
	}
	return nil, ErrConnectionNotFound
}

func (s *MemoryStore) GetSSOConnectionByProvider(_ context.Context, appID id.AppID, provider string) (*SSOConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.conns {
		if c.AppID == appID && c.Provider == provider && c.Active {
			return c, nil
		}
	}
	return nil, ErrConnectionNotFound
}

func (s *MemoryStore) ListSSOConnections(_ context.Context, appID id.AppID) ([]*SSOConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*SSOConnection
	for _, c := range s.conns {
		if c.AppID == appID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *MemoryStore) UpdateSSOConnection(_ context.Context, c *SSOConnection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conns[c.ID]; !ok {
		return ErrConnectionNotFound
	}
	s.conns[c.ID] = c
	return nil
}

func (s *MemoryStore) DeleteSSOConnection(_ context.Context, connID id.SSOConnectionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conns[connID]; !ok {
		return ErrConnectionNotFound
	}
	delete(s.conns, connID)
	return nil
}
