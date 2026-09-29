package oauth2provider

import (
	"context"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// MemoryStore is an in-memory implementation of the OAuth2 Store for testing.
type MemoryStore struct {
	mu          sync.RWMutex
	clients     map[string]*OAuth2Client      // keyed by ClientID (the OAuth2 client_id string)
	codes       map[string]*AuthorizationCode // keyed by store.HashToken(Code)
	deviceCodes map[string]*DeviceCode        // keyed by ID; codes are held hashed
	grants      map[string]*Grant             // keyed by grantKey
}

func grantKey(appID id.AppID, userID id.UserID, clientID string) string {
	return appID.String() + "|" + userID.String() + "|" + clientID
}

// NewMemoryStore creates a new in-memory OAuth2 store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		clients:     make(map[string]*OAuth2Client),
		codes:       make(map[string]*AuthorizationCode),
		deviceCodes: make(map[string]*DeviceCode),
		grants:      make(map[string]*Grant),
	}
}

func (s *MemoryStore) CreateClient(_ context.Context, c *OAuth2Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// The persistent backends normalise nil slices to empty ones on the way
	// through, so a client read back from them never carries a nil list. Do
	// the same here or the in-memory store serialises null where the others
	// serialise [].
	if c.RedirectURIs == nil {
		c.RedirectURIs = []string{}
	}
	if c.Scopes == nil {
		c.Scopes = []string{}
	}
	if c.GrantTypes == nil {
		c.GrantTypes = []string{}
	}
	if c.Resources == nil {
		c.Resources = []string{}
	}
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

func (s *MemoryStore) UpdateClient(_ context.Context, c *OAuth2Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.clients[c.ClientID]
	if !ok || existing.ID != c.ID {
		return ErrClientNotFound
	}
	c.UpdatedAt = time.Now()
	s.clients[c.ClientID] = c
	return nil
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
	stored := *code
	stored.Code = store.HashToken(code.Code)
	s.codes[stored.Code] = &stored
	return nil
}

func (s *MemoryStore) GetAuthCode(_ context.Context, code string) (*AuthorizationCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.codes[store.HashToken(code)]
	if !ok {
		return nil, ErrCodeNotFound
	}
	copied := *c
	copied.Code = code
	return &copied, nil
}

func (s *MemoryStore) ConsumeAuthCode(_ context.Context, code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[store.HashToken(code)]
	if !ok {
		return false, ErrCodeNotFound
	}
	if c.Consumed {
		return false, nil
	}
	c.Consumed = true
	return true, nil
}

// ──────────────────────────────────────────────────
// Device code methods (RFC 8628)
// ──────────────────────────────────────────────────

func (s *MemoryStore) CreateDeviceCode(_ context.Context, dc *DeviceCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := *dc
	stored.DeviceCode = store.HashToken(dc.DeviceCode)
	stored.UserCode = store.HashToken(dc.UserCode)
	s.deviceCodes[dc.ID.String()] = &stored
	return nil
}

func (s *MemoryStore) GetDeviceCodeByDeviceCode(_ context.Context, deviceCode string) (*DeviceCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := store.HashToken(deviceCode)
	for _, dc := range s.deviceCodes {
		if dc.DeviceCode == h {
			copied := *dc
			copied.DeviceCode, copied.UserCode = deviceCode, ""
			return &copied, nil
		}
	}
	return nil, ErrDeviceCodeNotFound
}

func (s *MemoryStore) GetDeviceCodeByUserCode(_ context.Context, userCode string) (*DeviceCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := store.HashToken(userCode)
	for _, dc := range s.deviceCodes {
		if dc.UserCode == h {
			copied := *dc
			copied.UserCode, copied.DeviceCode = userCode, ""
			return &copied, nil
		}
	}
	return nil, ErrDeviceCodeNotFound
}

func (s *MemoryStore) UpdateDeviceCode(_ context.Context, dc *DeviceCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.deviceCodes[dc.ID.String()]
	if !ok {
		return ErrDeviceCodeNotFound
	}
	// Only the fields the flow mutates; dc carries whichever plaintext code
	// the lookup that found it presented, never both hashes.
	stored.Status = dc.Status
	stored.UserID = dc.UserID
	stored.LastPolledAt = dc.LastPolledAt
	return nil
}

func (s *MemoryStore) DeleteExpiredDeviceCodes(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, dc := range s.deviceCodes {
		if now.After(dc.ExpiresAt) {
			delete(s.deviceCodes, key)
		}
	}
	return nil
}

// ──────────────────────────────────────────────────
// Grants
// ──────────────────────────────────────────────────

func (s *MemoryStore) UpsertGrant(_ context.Context, g *Grant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := grantKey(g.AppID, g.UserID, g.ClientID)
	now := time.Now()
	if existing, ok := s.grants[key]; ok {
		g.ID = existing.ID
		g.CreatedAt = existing.CreatedAt
	} else {
		if g.ID.IsNil() {
			g.ID = id.NewOAuth2GrantID()
		}
		if g.CreatedAt.IsZero() {
			g.CreatedAt = now
		}
	}
	g.UpdatedAt = now
	cp := *g
	cp.Scopes = append([]string(nil), g.Scopes...)
	s.grants[key] = &cp
	return nil
}

func (s *MemoryStore) GetGrant(_ context.Context, appID id.AppID, userID id.UserID, clientID string) (*Grant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.grants[grantKey(appID, userID, clientID)]
	if !ok {
		return nil, ErrGrantNotFound
	}
	cp := *g
	cp.Scopes = append([]string(nil), g.Scopes...)
	return &cp, nil
}

func (s *MemoryStore) ListGrantsByUser(_ context.Context, appID id.AppID, userID id.UserID) ([]*Grant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Grant, 0)
	for _, g := range s.grants {
		if g.AppID == appID && g.UserID == userID {
			cp := *g
			cp.Scopes = append([]string(nil), g.Scopes...)
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (s *MemoryStore) DeleteGrant(_ context.Context, appID id.AppID, userID id.UserID, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := grantKey(appID, userID, clientID)
	if _, ok := s.grants[key]; !ok {
		return ErrGrantNotFound
	}
	delete(s.grants, key)
	return nil
}

// Compile-time interface check.
var _ Store = (*MemoryStore)(nil)
