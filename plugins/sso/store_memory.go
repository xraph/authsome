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
	conns map[id.SSOConnectionID]*Connection
}

// NewMemoryStore creates a new in-memory SSO connection store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conns: make(map[id.SSOConnectionID]*Connection),
	}
}

var _ Store = (*MemoryStore)(nil)

func (s *MemoryStore) CreateConnection(_ context.Context, c *Connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[c.ID] = cloneConnection(c)
	return nil
}

func (s *MemoryStore) GetConnection(_ context.Context, connID id.SSOConnectionID) (*Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.conns[connID]
	if !ok {
		return nil, ErrConnectionNotFound
	}
	return cloneConnection(c), nil
}

func (s *MemoryStore) GetConnectionByDomain(_ context.Context, appID id.AppID, domain string) (*Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.conns {
		if c.AppID == appID && c.Domain == domain && c.Active {
			return cloneConnection(c), nil
		}
	}
	return nil, ErrConnectionNotFound
}

func (s *MemoryStore) GetConnectionByDomainAndOrg(_ context.Context, appID id.AppID, orgID id.OrgID, domain string) (*Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.conns {
		if c.AppID == appID && c.OrgID == orgID && c.Domain == domain && c.Active {
			return cloneConnection(c), nil
		}
	}
	return nil, ErrConnectionNotFound
}

func (s *MemoryStore) GetConnectionByProvider(_ context.Context, appID id.AppID, provider string) (*Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.conns {
		if c.AppID == appID && c.Provider == provider && c.Active {
			return cloneConnection(c), nil
		}
	}
	return nil, ErrConnectionNotFound
}

func (s *MemoryStore) ListConnections(_ context.Context, appID id.AppID) ([]*Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Connection
	for _, c := range s.conns {
		if c.AppID == appID {
			result = append(result, cloneConnection(c))
		}
	}
	return result, nil
}

func (s *MemoryStore) UpdateConnection(_ context.Context, c *Connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conns[c.ID]; !ok {
		return ErrConnectionNotFound
	}
	s.conns[c.ID] = cloneConnection(c)
	return nil
}

func (s *MemoryStore) DeleteConnection(_ context.Context, connID id.SSOConnectionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conns[connID]; !ok {
		return ErrConnectionNotFound
	}
	delete(s.conns, connID)
	return nil
}

// cloneConnection copies c so the store never hands out, or retains, a
// pointer into its own map.
//
// Without this a caller could write through what GetConnection returned and
// silently rewrite the stored connection: its Domain, its Active flag, its
// IDP certificate. That write is also unsynchronized against every concurrent
// read, since it happens after the read lock is released.
//
// AttributeMappings is a map, so it is copied explicitly. A shallow
// `cp := *c` shares it, which would leave an SSO attribute mapping (the thing
// that decides which IdP claim becomes which local identity field) writable
// by any caller that ever received the connection.
func cloneConnection(c *Connection) *Connection {
	if c == nil {
		return nil
	}
	cp := *c
	if c.AttributeMappings != nil {
		cp.AttributeMappings = make(map[string]string, len(c.AttributeMappings))
		for k, v := range c.AttributeMappings {
			cp.AttributeMappings[k] = v
		}
	}
	return &cp
}
