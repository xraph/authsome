package passkey

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryStore is an in-memory Store for testing.
type MemoryStore struct {
	mu          sync.RWMutex
	credentials map[string]*Credential // keyed by hex(credentialID)
}

// NewMemoryStore creates a new in-memory passkey credential store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		credentials: make(map[string]*Credential),
	}
}

var _ Store = (*MemoryStore)(nil)

func credKey(credentialID []byte) string {
	return hex.EncodeToString(credentialID)
}

// CreateCredential stores a new passkey credential.
func (s *MemoryStore) CreateCredential(_ context.Context, c *Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	c.UpdatedAt = c.CreatedAt
	s.credentials[credKey(c.CredentialID)] = c
	return nil
}

// GetCredential finds a credential by its WebAuthn credential ID.
func (s *MemoryStore) GetCredential(_ context.Context, credentialID []byte) (*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.credentials[credKey(credentialID)]
	if !ok {
		return nil, ErrCredentialNotFound
	}
	return c, nil
}

// ListUserCredentials returns all credentials for a user.
func (s *MemoryStore) ListUserCredentials(_ context.Context, userID id.UserID) ([]*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Credential
	for _, c := range s.credentials {
		if c.UserID == userID {
			result = append(result, c)
		}
	}
	return result, nil
}

// DeleteCredential removes a credential by its WebAuthn credential ID.
func (s *MemoryStore) DeleteCredential(_ context.Context, credentialID []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := credKey(credentialID)
	if _, ok := s.credentials[key]; !ok {
		return ErrCredentialNotFound
	}
	delete(s.credentials, key)
	return nil
}

// UpdateSignCount updates the sign counter for a credential.
func (s *MemoryStore) UpdateSignCount(_ context.Context, credentialID []byte, count uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := credKey(credentialID)
	c, ok := s.credentials[key]
	if !ok {
		return ErrCredentialNotFound
	}
	c.SignCount = count
	c.UpdatedAt = time.Now()
	return nil
}
