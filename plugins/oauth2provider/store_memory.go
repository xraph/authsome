package oauth2provider

import (
	"context"
	"sync"

	"github.com/xraph/authsome/id"
)

// MemoryStore is an in-memory implementation of the OAuth2 Store for testing.
type MemoryStore struct {
	mu      sync.RWMutex
	clients map[string]*OAuth2Client  // keyed by ClientID (the OAuth2 client_id string)
	codes   map[string]*AuthorizationCode // keyed by Code
}

// NewMemoryStore creates a new in-memory OAuth2 store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		clients: make(map[string]*OAuth2Client),
		codes:   make(map[string]*AuthorizationCode),
	}
}

func (s *MemoryStore) CreateClient(_ context.Context, c *OAuth2Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ClientID] = c
	return nil
}

func (s *MemoryStore) GetClient(_ context.Context, clientID string) (*OAuth2Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clients[clientID]
	if !ok {
		return nil, ErrClientNotFound
	}
	return c, nil
}

func (s *MemoryStore) GetClientByID(_ context.Context, clientID id.OAuth2ClientID) (*OAuth2Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		if c.ID == clientID {
			return c, nil
		}
	}
	return nil, ErrClientNotFound
}

func (s *MemoryStore) ListClients(_ context.Context, appID id.AppID) ([]*OAuth2Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*OAuth2Client
	for _, c := range s.clients {
		if c.AppID == appID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *MemoryStore) DeleteClient(_ context.Context, clientID id.OAuth2ClientID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, c := range s.clients {
		if c.ID == clientID {
			delete(s.clients, key)
			return nil
		}
	}
	return ErrClientNotFound
}

func (s *MemoryStore) CreateAuthCode(_ context.Context, code *AuthorizationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code.Code] = code
	return nil
}

func (s *MemoryStore) GetAuthCode(_ context.Context, code string) (*AuthorizationCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.codes[code]
	if !ok {
		return nil, ErrCodeNotFound
	}
	return c, nil
}

func (s *MemoryStore) ConsumeAuthCode(_ context.Context, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return ErrCodeNotFound
	}
	c.Consumed = true
	return nil
}
