// Package secrets provides the secrets management plugin for AuthSome.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"golang.org/x/crypto/argon2"

	"github.com/xraph/authsome/plugins/secrets/core"
)

// Encryption constants
const (
	// MasterKeyLength is the required length for the master key (32 bytes for AES-256)
	MasterKeyLength = 32
	// NonceLength is the length of the nonce for AES-GCM (12 bytes)
	NonceLength = 12
	// SaltLength is the length of the salt for key derivation
	SaltLength = 32
)

// Argon2 parameters for key derivation
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64MB
	argon2Threads = 4
	argon2KeyLen  = 32
)

// EncryptionService handles encryption and decryption of secret values
// using AES-256-GCM with Argon2 key derivation for per-tenant isolation.
type EncryptionService struct {
	masterKey []byte
	keyCache  map[string][]byte // Cache derived keys: "appID:envID" -> derived key
	cacheMu   sync.RWMutex
}

// NewEncryptionService creates a new encryption service with the given master key.
// The master key must be a base64-encoded 32-byte key.
func NewEncryptionService(masterKeyBase64 string) (*EncryptionService, error) {
	if masterKeyBase64 == "" {
		return nil, core.ErrMasterKeyRequired()
	}

	masterKey, err := base64.StdEncoding.DecodeString(masterKeyBase64)
	if err != nil {
		return nil, core.ErrMasterKeyInvalid("invalid base64 encoding: " + err.Error())
	}

	if len(masterKey) != MasterKeyLength {
		return nil, core.ErrMasterKeyInvalid(
			fmt.Sprintf("key must be %d bytes, got %d bytes", MasterKeyLength, len(masterKey)),
		)
	}

	return &EncryptionService{
		masterKey: masterKey,
		keyCache:  make(map[string][]byte),
	}, nil
}

// GenerateMasterKey generates a new random master key and returns it as base64-encoded string.
// This is a utility function for initial setup.
func GenerateMasterKey() (string, error) {
	key := make([]byte, MasterKeyLength)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate master key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// DeriveKey derives an encryption key for a specific app and environment using Argon2.
// This provides cryptographic isolation between different tenants.
func (e *EncryptionService) DeriveKey(appID, envID string) []byte {
	cacheKey := appID + ":" + envID

	// Check cache first
	e.cacheMu.RLock()
	if derivedKey, ok := e.keyCache[cacheKey]; ok {
		e.cacheMu.RUnlock()
		return derivedKey
	}
	e.cacheMu.RUnlock()

	// Create a deterministic salt from app and environment IDs
	saltInput := []byte(appID + ":" + envID)
	saltHash := sha256.Sum256(saltInput)
	salt := saltHash[:]

	// Derive key using Argon2id
	derivedKey := argon2.IDKey(
		e.masterKey,
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)

	// Cache the derived key
	e.cacheMu.Lock()
	e.keyCache[cacheKey] = derivedKey
	e.cacheMu.Unlock()

	return derivedKey
}

// Encrypt encrypts plaintext using AES-256-GCM with a key derived for the specific app/environment.
// Returns the ciphertext and nonce, which must both be stored.
func (e *EncryptionService) Encrypt(plaintext []byte, appID, envID string) (ciphertext, nonce []byte, err error) {
	// Derive the key for this app/environment
	key := e.DeriveKey(appID, envID)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, core.ErrEncryptionFailed(fmt.Errorf("failed to create cipher: %w", err))
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, core.ErrEncryptionFailed(fmt.Errorf("failed to create GCM: %w", err))
	}

	// Generate a random nonce
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, core.ErrEncryptionFailed(fmt.Errorf("failed to generate nonce: %w", err))
	}

	// Encrypt the plaintext
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)

	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with a key derived for the specific app/environment.
func (e *EncryptionService) Decrypt(ciphertext, nonce []byte, appID, envID string) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, core.ErrDecryptionFailed(fmt.Errorf("ciphertext is empty"))
	}

	if len(nonce) == 0 {
		return nil, core.ErrDecryptionFailed(fmt.Errorf("nonce is empty"))
	}

	// Derive the key for this app/environment
	key := e.DeriveKey(appID, envID)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, core.ErrDecryptionFailed(fmt.Errorf("failed to create cipher: %w", err))
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, core.ErrDecryptionFailed(fmt.Errorf("failed to create GCM: %w", err))
	}

	// Validate nonce size
	if len(nonce) != gcm.NonceSize() {
		return nil, core.ErrDecryptionFailed(
			fmt.Errorf("invalid nonce size: expected %d, got %d", gcm.NonceSize(), len(nonce)),
		)
	}

	// Decrypt the ciphertext
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, core.ErrDecryptionFailed(fmt.Errorf("decryption failed: %w", err))
	}

	return plaintext, nil
}

// ReEncrypt re-encrypts a value with a new key derivation.
// This is useful when rotating encryption keys or migrating secrets between environments.
func (e *EncryptionService) ReEncrypt(
	ciphertext, nonce []byte,
	oldAppID, oldEnvID string,
	newAppID, newEnvID string,
) (newCiphertext, newNonce []byte, err error) {
	// Decrypt with old key
	plaintext, err := e.Decrypt(ciphertext, nonce, oldAppID, oldEnvID)
	if err != nil {
		return nil, nil, err
	}

	// Encrypt with new key
	return e.Encrypt(plaintext, newAppID, newEnvID)
}

// ClearKeyCache clears the derived key cache.
// This should be called when rotating the master key.
func (e *EncryptionService) ClearKeyCache() {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	e.keyCache = make(map[string][]byte)
}

// ClearKeyForTenant clears the cached key for a specific tenant.
func (e *EncryptionService) ClearKeyForTenant(appID, envID string) {
	cacheKey := appID + ":" + envID
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	delete(e.keyCache, cacheKey)
}

// ValidateMasterKey validates that the master key is properly configured.
func (e *EncryptionService) ValidateMasterKey() error {
	if e.masterKey == nil || len(e.masterKey) != MasterKeyLength {
		return core.ErrMasterKeyInvalid("master key is not properly initialized")
	}
	return nil
}

// TestEncryption performs a test encryption/decryption cycle to verify the service is working.
func (e *EncryptionService) TestEncryption() error {
	testPlaintext := []byte("test-encryption-verification")
	testAppID := "test-app"
	testEnvID := "test-env"

	ciphertext, nonce, err := e.Encrypt(testPlaintext, testAppID, testEnvID)
	if err != nil {
		return fmt.Errorf("test encryption failed: %w", err)
	}

	decrypted, err := e.Decrypt(ciphertext, nonce, testAppID, testEnvID)
	if err != nil {
		return fmt.Errorf("test decryption failed: %w", err)
	}

	if string(decrypted) != string(testPlaintext) {
		return fmt.Errorf("test encryption/decryption round-trip failed: data mismatch")
	}

	// Clean up test key from cache
	e.ClearKeyForTenant(testAppID, testEnvID)

	return nil
}

