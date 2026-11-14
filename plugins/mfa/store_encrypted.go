package mfa

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
)

// EncryptedStore wraps a Store, transparently encrypting the TOTP/SMS secret
// on an Enrollment before it reaches the underlying storage and decrypting it
// on the way back out.
//
// It is safe over a store that already holds legacy plaintext secrets:
// bridge.AESGCMEncryptor's Decrypt returns input unchanged when no envelope
// prefix is present. Recovery codes are already hashed by the caller, so those
// methods delegate without modification.
type EncryptedStore struct {
	inner Store
	enc   bridge.Encryptor
}

// NewEncryptedStore wraps inner with at-rest secret encryption. A nil encryptor
// falls back to bridge.NoopEncryptor (no encryption).
func NewEncryptedStore(inner Store, enc bridge.Encryptor) *EncryptedStore {
	if enc == nil {
		enc = bridge.NoopEncryptor{}
	}
	return &EncryptedStore{inner: inner, enc: enc}
}

// Compile-time interface check.
var _ Store = (*EncryptedStore)(nil)

func (s *EncryptedStore) CreateEnrollment(ctx context.Context, e *Enrollment) error {
	enc, err := s.encryptCopy(e)
	if err != nil {
		return fmt.Errorf("mfa: encrypt enrollment secret: %w", err)
	}
	if err := s.inner.CreateEnrollment(ctx, enc); err != nil {
		return err
	}
	e.ID = enc.ID
	e.CreatedAt = enc.CreatedAt
	e.UpdatedAt = enc.UpdatedAt
	return nil
}

func (s *EncryptedStore) UpdateEnrollment(ctx context.Context, e *Enrollment) error {
	enc, err := s.encryptCopy(e)
	if err != nil {
		return fmt.Errorf("mfa: encrypt enrollment secret: %w", err)
	}
	if err := s.inner.UpdateEnrollment(ctx, enc); err != nil {
		return err
	}
	e.UpdatedAt = enc.UpdatedAt
	return nil
}

func (s *EncryptedStore) GetEnrollment(ctx context.Context, userID id.UserID, method string) (*Enrollment, error) {
	e, err := s.inner.GetEnrollment(ctx, userID, method)
	if err != nil {
		return nil, err
	}
	return s.decryptInPlace(e)
}

func (s *EncryptedStore) GetEnrollmentByID(ctx context.Context, mfaID id.MFAID) (*Enrollment, error) {
	e, err := s.inner.GetEnrollmentByID(ctx, mfaID)
	if err != nil {
		return nil, err
	}
	return s.decryptInPlace(e)
}

func (s *EncryptedStore) ListEnrollments(ctx context.Context, userID id.UserID) ([]*Enrollment, error) {
	list, err := s.inner.ListEnrollments(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i, e := range list {
		dec, derr := s.decryptInPlace(e)
		if derr != nil {
			return nil, derr
		}
		list[i] = dec
	}
	return list, nil
}

func (s *EncryptedStore) DeleteEnrollment(ctx context.Context, mfaID id.MFAID) error {
	return s.inner.DeleteEnrollment(ctx, mfaID)
}

// Recovery codes are hashed by the caller — delegate unchanged.
func (s *EncryptedStore) CreateRecoveryCodes(ctx context.Context, codes []*RecoveryCode) error {
	return s.inner.CreateRecoveryCodes(ctx, codes)
}

func (s *EncryptedStore) GetRecoveryCodes(ctx context.Context, userID id.UserID) ([]*RecoveryCode, error) {
	return s.inner.GetRecoveryCodes(ctx, userID)
}

func (s *EncryptedStore) ConsumeRecoveryCode(ctx context.Context, codeID id.RecoveryCodeID) error {
	return s.inner.ConsumeRecoveryCode(ctx, codeID)
}

func (s *EncryptedStore) DeleteRecoveryCodes(ctx context.Context, userID id.UserID) error {
	return s.inner.DeleteRecoveryCodes(ctx, userID)
}

// Trusted devices hold only a token hash, never a secret — delegate unchanged.
func (s *EncryptedStore) CreateTrustedDevice(ctx context.Context, d *TrustedDevice) error {
	return s.inner.CreateTrustedDevice(ctx, d)
}

func (s *EncryptedStore) GetTrustedDeviceByTokenHash(ctx context.Context, tokenHash string) (*TrustedDevice, error) {
	return s.inner.GetTrustedDeviceByTokenHash(ctx, tokenHash)
}

func (s *EncryptedStore) ListTrustedDevices(ctx context.Context, userID id.UserID) ([]*TrustedDevice, error) {
	return s.inner.ListTrustedDevices(ctx, userID)
}

func (s *EncryptedStore) TouchTrustedDevice(ctx context.Context, deviceID id.TrustedDeviceID, at time.Time) error {
	return s.inner.TouchTrustedDevice(ctx, deviceID, at)
}

func (s *EncryptedStore) DeleteTrustedDevice(ctx context.Context, userID id.UserID, deviceID id.TrustedDeviceID) error {
	return s.inner.DeleteTrustedDevice(ctx, userID, deviceID)
}

func (s *EncryptedStore) DeleteTrustedDevices(ctx context.Context, userID id.UserID) error {
	return s.inner.DeleteTrustedDevices(ctx, userID)
}

func (s *EncryptedStore) encryptCopy(e *Enrollment) (*Enrollment, error) {
	cp := *e
	if cp.Secret != "" {
		ct, err := s.enc.Encrypt([]byte(cp.Secret))
		if err != nil {
			return nil, err
		}
		cp.Secret = string(ct)
	}
	return &cp, nil
}

func (s *EncryptedStore) decryptInPlace(e *Enrollment) (*Enrollment, error) {
	if e == nil {
		return nil, nil
	}
	if e.Secret != "" {
		pt, err := s.enc.Decrypt([]byte(e.Secret))
		if err != nil {
			return nil, fmt.Errorf("mfa: decrypt enrollment secret: %w", err)
		}
		e.Secret = string(pt)
	}
	return e, nil
}
