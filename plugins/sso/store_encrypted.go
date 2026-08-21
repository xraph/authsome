package sso

import (
	"context"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
)

// EncryptedStore wraps a Store, transparently encrypting the sensitive SSO
// connection secrets at rest — the OIDC ClientSecret and the SAML SP private key.
// Reads decrypt; bridge.AESGCMEncryptor tags its output with a version prefix and
// returns un-prefixed values unchanged on Decrypt, so pre-existing plaintext rows
// keep working and re-saving a connection upgrades it in place.
type EncryptedStore struct {
	inner Store
	enc   bridge.Encryptor
}

// NewEncryptedStore wraps inner with secret encryption. A nil encryptor falls
// back to bridge.NoopEncryptor (no-op passthrough, e.g. dev without a key). Pass
// an AES-GCM encryptor (engine.TokenEncryptor()) in production.
func NewEncryptedStore(inner Store, enc bridge.Encryptor) *EncryptedStore {
	if enc == nil {
		enc = bridge.NoopEncryptor{}
	}
	return &EncryptedStore{inner: inner, enc: enc}
}

var _ Store = (*EncryptedStore)(nil)

func (s *EncryptedStore) CreateConnection(ctx context.Context, c *Connection) error {
	enc, err := s.encryptCopy(c)
	if err != nil {
		return err
	}
	return s.inner.CreateConnection(ctx, enc)
}

func (s *EncryptedStore) UpdateConnection(ctx context.Context, c *Connection) error {
	enc, err := s.encryptCopy(c)
	if err != nil {
		return err
	}
	return s.inner.UpdateConnection(ctx, enc)
}

func (s *EncryptedStore) GetConnection(ctx context.Context, connID id.SSOConnectionID) (*Connection, error) {
	c, err := s.inner.GetConnection(ctx, connID)
	if err != nil {
		return nil, err
	}
	return s.decryptInPlace(c)
}

func (s *EncryptedStore) GetConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*Connection, error) {
	c, err := s.inner.GetConnectionByDomain(ctx, appID, domain)
	if err != nil {
		return nil, err
	}
	return s.decryptInPlace(c)
}

func (s *EncryptedStore) GetConnectionByDomainAndOrg(ctx context.Context, appID id.AppID, orgID id.OrgID, domain string) (*Connection, error) {
	c, err := s.inner.GetConnectionByDomainAndOrg(ctx, appID, orgID, domain)
	if err != nil {
		return nil, err
	}
	return s.decryptInPlace(c)
}

func (s *EncryptedStore) GetConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*Connection, error) {
	c, err := s.inner.GetConnectionByProvider(ctx, appID, provider)
	if err != nil {
		return nil, err
	}
	return s.decryptInPlace(c)
}

func (s *EncryptedStore) ListConnections(ctx context.Context, appID id.AppID) ([]*Connection, error) {
	list, err := s.inner.ListConnections(ctx, appID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if _, derr := s.decryptInPlace(list[i]); derr != nil {
			return nil, derr
		}
	}
	return list, nil
}

func (s *EncryptedStore) DeleteConnection(ctx context.Context, connID id.SSOConnectionID) error {
	return s.inner.DeleteConnection(ctx, connID)
}

// encryptCopy returns a copy of c with its secret fields encrypted, leaving the
// caller's struct untouched.
func (s *EncryptedStore) encryptCopy(c *Connection) (*Connection, error) {
	cp := *c
	if cp.ClientSecret != "" {
		ct, err := s.enc.Encrypt([]byte(cp.ClientSecret))
		if err != nil {
			return nil, err
		}
		cp.ClientSecret = string(ct)
	}
	if cp.SPPrivateKey != "" {
		ct, err := s.enc.Encrypt([]byte(cp.SPPrivateKey))
		if err != nil {
			return nil, err
		}
		cp.SPPrivateKey = string(ct)
	}
	return &cp, nil
}

// decryptInPlace decrypts c's secret fields in place (nil-safe).
func (s *EncryptedStore) decryptInPlace(c *Connection) (*Connection, error) {
	if c == nil {
		return nil, nil
	}
	if c.ClientSecret != "" {
		pt, err := s.enc.Decrypt([]byte(c.ClientSecret))
		if err != nil {
			return nil, err
		}
		c.ClientSecret = string(pt)
	}
	if c.SPPrivateKey != "" {
		pt, err := s.enc.Decrypt([]byte(c.SPPrivateKey))
		if err != nil {
			return nil, err
		}
		c.SPPrivateKey = string(pt)
	}
	return c, nil
}
