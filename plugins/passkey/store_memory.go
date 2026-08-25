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
	s.credentials[credKey(c.CredentialID)] = cloneCredential(c)
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
	return cloneCredential(c), nil
}

// ListUserCredentials returns all credentials for a user.
func (s *MemoryStore) ListUserCredentials(_ context.Context, userID id.UserID) ([]*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Credential
	for _, c := range s.credentials {
		if c.UserID == userID {
			result = append(result, cloneCredential(c))
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

// cloneCredential deep-copies c, every byte slice included.
//
// The store mutates credentials in place: UpdateSignCount writes SignCount
// and UpdatedAt on the stored record under the write lock. Handing that same
// pointer back from GetCredential and ListUserCredentials meant a caller was
// reading fields another goroutine could be writing, with nothing ordering
// the two. For SignCount that is worse than a generic race, because the sign
// counter is WebAuthn's clone-detection signal and a verifier comparing a
// torn read against the authenticator's value is checking nothing.
//
// CredentialID, PublicKey, AAGUID and Transport are all reference types, so a
// shallow `cp := *c` would leave the authentication material itself shared
// between the store and every caller. They are copied explicitly here for the
// same reason agentauth's cloneGrant copies Scopes.
func cloneCredential(c *Credential) *Credential {
	if c == nil {
		return nil
	}
	cp := *c
	cp.CredentialID = append([]byte(nil), c.CredentialID...)
	cp.PublicKey = append([]byte(nil), c.PublicKey...)
	cp.AAGUID = append([]byte(nil), c.AAGUID...)
	cp.Transport = append([]string(nil), c.Transport...)
	return &cp
}
