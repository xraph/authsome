package oidcprovider

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"math/big"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
)

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	KeyType   string `json:"kty"`
	Use       string `json:"use"`
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	N         string `json:"n"` // RSA modulus
	E         string `json:"e"` // RSA exponent
}

// KeyPair represents an RSA key pair with metadata.
type KeyPair struct {
	ID         string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	CreatedAt  time.Time
	ExpiresAt  time.Time
	Active     bool // Whether this key is used for signing new tokens
}

// KeyStoreInterface defines the interface for key storage backends.
type KeyStoreInterface interface {
	GetActiveKey() *KeyPair
	GetKey(kid string) *KeyPair
	GetAllValidKeys() []*KeyPair
	RotateKeys() error
	ShouldRotate() bool
	GetLastRotation() time.Time
}

// KeyStore manages multiple key pairs for rotation (in-memory implementation).
type KeyStore struct {
	keys             map[string]*KeyPair
	activeKey        string
	mu               sync.RWMutex
	rotationInterval time.Duration
	keyLifetime      time.Duration
	lastRotation     time.Time
}

// Ensure KeyStore implements KeyStoreInterface.
var _ KeyStoreInterface = (*KeyStore)(nil)

// GetLastRotation returns the last rotation time.
func (ks *KeyStore) GetLastRotation() time.Time {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return ks.lastRotation
}

// JWKSService manages JSON Web Key Sets for the OIDC Provider.
type JWKSService struct {
	keyStore KeyStoreInterface // Can be in-memory or database-backed
}

// NewKeyStore creates a new key store with initial key pair.
func NewKeyStore() (*KeyStore, error) {
	ks := &KeyStore{
		keys:             make(map[string]*KeyPair),
		rotationInterval: 24 * time.Hour,     // Rotate keys daily
		keyLifetime:      7 * 24 * time.Hour, // Keep keys for 7 days
	}

	// Generate initial key pair
	if err := ks.generateNewKey(); err != nil {
		return nil, fmt.Errorf("failed to generate initial key: %w", err)
	}

	return ks, nil
}

// NewKeyStoreFromFiles creates a new key store with keys loaded from files.
func NewKeyStoreFromFiles(privateKeyPath, publicKeyPath, rotationInterval, keyLifetime string) (*KeyStore, error) {
	// Parse duration strings
	rotationDur, err := time.ParseDuration(rotationInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid rotation interval: %w", err)
	}

	lifetimeDur, err := time.ParseDuration(keyLifetime)
	if err != nil {
		return nil, fmt.Errorf("invalid key lifetime: %w", err)
	}

	ks := &KeyStore{
		keys:             make(map[string]*KeyPair),
		rotationInterval: rotationDur,
		keyLifetime:      lifetimeDur,
	}

	// Load private key from file
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return nil, errs.InternalServerErrorWithMessage("failed to decode PEM block from private key")
	}

	var privateKey *rsa.PrivateKey

	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}

		var ok bool

		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errs.InternalServerErrorWithMessage("private key is not RSA")
		}
	default:
		return nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create key pair
	keyID := xid.New().String()
	now := time.Now()
	keyPair := &KeyPair{
		ID:         keyID,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		CreatedAt:  now,
		ExpiresAt:  now.Add(lifetimeDur),
		Active:     true,
	}

	ks.keys[keyID] = keyPair
	ks.activeKey = keyID

	return ks, nil
}

// generateNewKey creates a new RSA key pair and adds it to the store.
func (ks *KeyStore) generateNewKey() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	keyID := xid.New().String()
	keyPair := &KeyPair{
		ID:         keyID,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ks.keyLifetime),
		Active:     true,
	}

	// Deactivate previous active key
	if ks.activeKey != "" {
		if oldKey, exists := ks.keys[ks.activeKey]; exists {
			oldKey.Active = false
		}
	}

	// Add new key and set as active
	ks.keys[keyID] = keyPair
	ks.activeKey = keyID
	ks.lastRotation = time.Now()

	return nil
}

// GetActiveKey returns the current active key pair for signing.
func (ks *KeyStore) GetActiveKey() *KeyPair {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.activeKey == "" {
		return nil
	}

	return ks.keys[ks.activeKey]
}

// GetKeyByID returns a key pair by its ID.
func (ks *KeyStore) GetKeyByID(keyID string) *KeyPair {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return ks.keys[keyID]
}

// GetKey is an alias for GetKeyByID (implements KeyStoreInterface).
func (ks *KeyStore) GetKey(kid string) *KeyPair {
	return ks.GetKeyByID(kid)
}

// GetAllValidKeys returns all keys that haven't expired.
func (ks *KeyStore) GetAllValidKeys() []*KeyPair {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	var validKeys []*KeyPair

	now := time.Now()

	for _, key := range ks.keys {
		if now.Before(key.ExpiresAt) {
			validKeys = append(validKeys, key)
		}
	}

	return validKeys
}

// RotateKeys generates a new key and cleans up expired keys.
func (ks *KeyStore) RotateKeys() error {
	// Generate new key
	if err := ks.generateNewKey(); err != nil {
		return err
	}

	// Clean up expired keys
	ks.cleanupExpiredKeys()

	return nil
}

// cleanupExpiredKeys removes expired keys from the store.
func (ks *KeyStore) cleanupExpiredKeys() {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	now := time.Now()
	for keyID, key := range ks.keys {
		if now.After(key.ExpiresAt) {
			delete(ks.keys, keyID)
		}
	}
}

// ShouldRotate checks if keys should be rotated based on the rotation interval.
func (ks *KeyStore) ShouldRotate() bool {
	activeKey := ks.GetActiveKey()
	if activeKey == nil {
		return true
	}

	return time.Since(activeKey.CreatedAt) >= ks.rotationInterval
}

// NewJWKSService creates a new JWKS service.
func NewJWKSService() (*JWKSService, error) {
	keyStore, err := NewKeyStore()
	if err != nil {
		return nil, err
	}

	return &JWKSService{
		keyStore: keyStore,
	}, nil
}

// NewJWKSServiceFromFiles creates a JWKS service with keys loaded from files.
func NewJWKSServiceFromFiles(privateKeyPath, publicKeyPath, rotationInterval, keyLifetime string) (*JWKSService, error) {
	keyStore, err := NewKeyStoreFromFiles(privateKeyPath, publicKeyPath, rotationInterval, keyLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to create key store from files: %w", err)
	}

	return &JWKSService{
		keyStore: keyStore,
	}, nil
}

// GetJWKS returns the current JSON Web Key Set.
func (j *JWKSService) GetJWKS() (*JWKS, error) {
	validKeys := j.keyStore.GetAllValidKeys()

	var jwks []JWK

	for _, keyPair := range validKeys {
		jwk, err := j.rsaPublicKeyToJWK(keyPair.PublicKey, keyPair.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert key %s to JWK: %w", keyPair.ID, err)
		}

		jwks = append(jwks, *jwk)
	}

	return &JWKS{
		Keys: jwks,
	}, nil
}

// GetActiveKeyPair returns the current active key pair for signing.
func (j *JWKSService) GetActiveKeyPair() *KeyPair {
	return j.keyStore.GetActiveKey()
}

// RotateKeys triggers key rotation.
func (j *JWKSService) RotateKeys() error {
	return j.keyStore.RotateKeys()
}

// ShouldRotate checks if keys should be rotated.
func (j *JWKSService) ShouldRotate() bool {
	return j.keyStore.ShouldRotate()
}

// GetCurrentKeyID returns the ID of the current active key.
func (j *JWKSService) GetCurrentKeyID() string {
	activeKey := j.keyStore.GetActiveKey()
	if activeKey == nil {
		return ""
	}

	return activeKey.ID
}

// GetLastRotation returns the last key rotation time.
func (j *JWKSService) GetLastRotation() time.Time {
	return j.keyStore.GetLastRotation()
}

// GetCurrentPrivateKey returns the private key of the current active key.
func (j *JWKSService) GetCurrentPrivateKey() *rsa.PrivateKey {
	activeKey := j.keyStore.GetActiveKey()
	if activeKey == nil {
		return nil
	}

	return activeKey.PrivateKey
}

// GetPublicKey returns the public key for a given key ID.
func (j *JWKSService) GetPublicKey(keyID string) (*rsa.PublicKey, error) {
	keyPair := j.keyStore.GetKey(keyID)
	if keyPair == nil {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	return keyPair.PublicKey, nil
}

// rsaPublicKeyToJWK converts an RSA public key to JWK format.
func (j *JWKSService) rsaPublicKeyToJWK(publicKey *rsa.PublicKey, keyID string) (*JWK, error) {
	// Convert RSA modulus (N) to base64url
	nBytes := publicKey.N.Bytes()
	n := base64.RawURLEncoding.EncodeToString(nBytes)

	// Convert RSA exponent (E) to base64url
	eBytes := big.NewInt(int64(publicKey.E)).Bytes()
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	return &JWK{
		KeyType:   "RSA",
		Use:       "sig", // For signature verification
		KeyID:     keyID,
		Algorithm: "RS256",
		N:         n,
		E:         e,
	}, nil
}

// GetKeyByID returns a specific key by its ID.
func (j *JWKSService) GetKeyByID(keyID string) (*JWK, error) {
	jwks, err := j.GetJWKS()
	if err != nil {
		return nil, err
	}

	for _, key := range jwks.Keys {
		if key.KeyID == keyID {
			return &key, nil
		}
	}

	return nil, nil // Key not found
}
