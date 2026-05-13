package bridge

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// envelopeV1Prefix is the version tag for AES-256-GCM ciphertexts produced
// by AESGCMEncryptor. The on-disk format is:
//
//	"v1:" + base64(nonce[12] || ciphertext || tag[16])
//
// The version prefix lets future algorithms (v2:, ...) coexist with v1
// rows in the same column.
const envelopeV1Prefix = "v1:"

// ErrInvalidKeyLength is returned when a key passed to NewAESGCMEncryptor
// is not exactly 32 bytes (AES-256).
var ErrInvalidKeyLength = errors.New("bridge: AES-256 key must be exactly 32 bytes")

// AESGCMEncryptor implements Encryptor using AES-256-GCM with a 12-byte
// random nonce per encryption. Decrypt is tolerant of legacy plaintext
// (no v1: prefix) and returns it unchanged so existing rows remain
// readable during the at-rest encryption rollout.
type AESGCMEncryptor struct {
	gcm cipher.AEAD
}

// NewAESGCMEncryptor constructs an AES-256-GCM Encryptor. The key must be
// exactly 32 bytes. Sourcing: typically AUTHSOME_TOKEN_ENCRYPTION_KEY decoded
// from 64 hex chars.
func NewAESGCMEncryptor(key []byte) (Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("bridge: aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("bridge: cipher.NewGCM: %w", err)
	}
	return &AESGCMEncryptor{gcm: gcm}, nil
}

// Encrypt produces "v1:" + base64(nonce || ciphertext || tag).
func (e *AESGCMEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("bridge: nonce read: %w", err)
	}
	sealed := e.gcm.Seal(nil, nonce, plaintext, nil)

	buf := make([]byte, 0, len(nonce)+len(sealed))
	buf = append(buf, nonce...)
	buf = append(buf, sealed...)

	encoded := base64.StdEncoding.EncodeToString(buf)
	out := make([]byte, 0, len(envelopeV1Prefix)+len(encoded))
	out = append(out, envelopeV1Prefix...)
	out = append(out, encoded...)
	return out, nil
}

// Decrypt accepts either a "v1:"-prefixed envelope (decodes & authenticates)
// or legacy plaintext (no recognized prefix), which it returns unchanged.
// This passthrough is intentional: it is the migration path for rows
// written before at-rest encryption was deployed.
func (e *AESGCMEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if !hasPrefix(ciphertext, envelopeV1Prefix) {
		// Legacy plaintext row — return as-is.
		return ciphertext, nil
	}
	body := ciphertext[len(envelopeV1Prefix):]
	raw, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return nil, fmt.Errorf("bridge: aesgcm: invalid base64: %w", err)
	}
	ns := e.gcm.NonceSize()
	if len(raw) < ns+e.gcm.Overhead() {
		return nil, errors.New("bridge: aesgcm: ciphertext too short")
	}
	nonce, sealed := raw[:ns], raw[ns:]
	pt, err := e.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("bridge: aesgcm: open: %w", err)
	}
	return pt, nil
}

func hasPrefix(b []byte, p string) bool {
	if len(b) < len(p) {
		return false
	}
	for i := 0; i < len(p); i++ {
		if b[i] != p[i] {
			return false
		}
	}
	return true
}
