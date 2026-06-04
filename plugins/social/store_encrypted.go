package social

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
)

// EncryptedStore wraps a Store, transparently encrypting AccessToken /
// RefreshToken before they reach the underlying storage and decrypting
// them on the way back out.
//
// The wrapper is safe to use over a store that already contains legacy
// plaintext rows — bridge.AESGCMEncryptor's Decrypt returns plaintext
// unchanged when no envelope prefix is present.
type EncryptedStore struct {
	inner Store
	enc   bridge.Encryptor
}

// NewEncryptedStore wraps inner with token encryption. If enc is nil,
// a bridge.NoopEncryptor is used (no encryption). Pass an AES-GCM
// encryptor in production.
func NewEncryptedStore(inner Store, enc bridge.Encryptor) *EncryptedStore {
	if enc == nil {
		enc = bridge.NoopEncryptor{}
	}
	return &EncryptedStore{inner: inner, enc: enc}
}

// Compile-time interface check.
var _ Store = (*EncryptedStore)(nil)

// CreateOAuthConnection encrypts tokens then delegates to the inner store.
// The caller's *OAuthConnection is left untouched.
func (s *EncryptedStore) CreateOAuthConnection(ctx context.Context, c *OAuthConnection) error {
	encConn, err := s.encryptCopy(c)
	if err != nil {
		return fmt.Errorf("social: encrypt oauth tokens: %w", err)
	}
	if err := s.inner.CreateOAuthConnection(ctx, encConn); err != nil {
		return err
	}
	// Propagate any backend-set fields (id, timestamps) back to the caller.
	c.ID = encConn.ID
	c.CreatedAt = encConn.CreatedAt
	c.UpdatedAt = encConn.UpdatedAt
	return nil
}

// GetOAuthConnection decrypts tokens before returning the connection.
func (s *EncryptedStore) GetOAuthConnection(ctx context.Context, provider, providerUserID string) (*OAuthConnection, error) {
	conn, err := s.inner.GetOAuthConnection(ctx, provider, providerUserID)
	if err != nil {
		return nil, err
	}
	return s.decryptInPlace(conn)
}

// GetOAuthConnectionsByUserID decrypts tokens for each connection.
func (s *EncryptedStore) GetOAuthConnectionsByUserID(ctx context.Context, userID id.UserID) ([]*OAuthConnection, error) {
	conns, err := s.inner.GetOAuthConnectionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i, c := range conns {
		dec, derr := s.decryptInPlace(c)
		if derr != nil {
			return nil, derr
		}
		conns[i] = dec
	}
	return conns, nil
}

// UpdateOAuthConnection encrypts tokens then delegates to the inner store.
// The caller's *OAuthConnection is left untouched.
func (s *EncryptedStore) UpdateOAuthConnection(ctx context.Context, c *OAuthConnection) error {
	encConn, err := s.encryptCopy(c)
	if err != nil {
		return fmt.Errorf("social: encrypt oauth tokens: %w", err)
	}
	if err := s.inner.UpdateOAuthConnection(ctx, encConn); err != nil {
		return err
	}
	c.UpdatedAt = encConn.UpdatedAt
	return nil
}

// DeleteOAuthConnection delegates without modification.
func (s *EncryptedStore) DeleteOAuthConnection(ctx context.Context, connID id.OAuthConnectionID) error {
	return s.inner.DeleteOAuthConnection(ctx, connID)
}

func (s *EncryptedStore) encryptCopy(c *OAuthConnection) (*OAuthConnection, error) {
	cp := *c
	if cp.AccessToken != "" {
		ct, err := s.enc.Encrypt([]byte(cp.AccessToken))
		if err != nil {
			return nil, err
		}
		cp.AccessToken = string(ct)
	}
	if cp.RefreshToken != "" {
		ct, err := s.enc.Encrypt([]byte(cp.RefreshToken))
		if err != nil {
			return nil, err
		}
		cp.RefreshToken = string(ct)
	}
	return &cp, nil
}

func (s *EncryptedStore) decryptInPlace(c *OAuthConnection) (*OAuthConnection, error) {
	if c == nil {
		return nil, nil
	}
	if c.AccessToken != "" {
		pt, err := s.enc.Decrypt([]byte(c.AccessToken))
		if err != nil {
			return nil, fmt.Errorf("social: decrypt access_token: %w", err)
		}
		c.AccessToken = string(pt)
	}
	if c.RefreshToken != "" {
		pt, err := s.enc.Decrypt([]byte(c.RefreshToken))
		if err != nil {
			return nil, fmt.Errorf("social: decrypt refresh_token: %w", err)
		}
		c.RefreshToken = string(pt)
	}
	return c, nil
}
