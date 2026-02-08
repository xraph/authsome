package oidcprovider

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// DatabaseKeyStore manages keys persisted in the database
type DatabaseKeyStore struct {
	db     *bun.DB
	appID  xid.ID
	cache  *KeyStore // In-memory cache
	logger interface{ Printf(string, ...interface{}) }
}

// NewDatabaseKeyStore creates a key store backed by the database
func NewDatabaseKeyStore(db *bun.DB, appID xid.ID, logger interface{ Printf(string, ...interface{}) }) (*DatabaseKeyStore, error) {
	dks := &DatabaseKeyStore{
		db:     db,
		appID:  appID,
		logger: logger,
	}

	// Initialize in-memory cache
	cache, err := NewKeyStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create key cache: %w", err)
	}
	dks.cache = cache

	// Load keys from database
	if err := dks.loadKeysFromDatabase(); err != nil {
		dks.logger.Printf("No existing keys in database, generating new ones: %v", err)
		// Generate and persist initial key if none exist
		if err := dks.generateAndPersistKey(); err != nil {
			return nil, fmt.Errorf("failed to generate initial key: %w", err)
		}
	}

	return dks, nil
}

// loadKeysFromDatabase loads all active keys from the database into the cache
func (dks *DatabaseKeyStore) loadKeysFromDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var jwtKeys []schema.JWTKey
	err := dks.db.NewSelect().
		Model(&jwtKeys).
		Where("app_id = ? OR is_platform_key = true", dks.appID).
		Where("active = true").
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to load keys from database: %w", err)
	}

	if len(jwtKeys) == 0 {
		return fmt.Errorf("no keys found in database")
	}

	dks.logger.Printf("Loaded %d JWT keys from database", len(jwtKeys))

	// Convert database keys to in-memory key pairs
	for i, jwtKey := range jwtKeys {
		keyPair, err := dks.parseKeyPair(&jwtKey)
		if err != nil {
			dks.logger.Printf("Failed to parse key %s: %v", jwtKey.KeyID, err)
			continue
		}

		dks.cache.mu.Lock()
		dks.cache.keys[keyPair.ID] = keyPair
		// First (newest) key becomes active
		if i == 0 {
			dks.cache.activeKey = keyPair.ID
		}
		dks.cache.mu.Unlock()

		dks.logger.Printf("Loaded JWT key %s (active: %v)", keyPair.ID, i == 0)
	}

	return nil
}

// parseKeyPair converts a database JWTKey to an in-memory KeyPair
func (dks *DatabaseKeyStore) parseKeyPair(jwtKey *schema.JWTKey) (*KeyPair, error) {
	// Parse private key
	block, _ := pem.Decode(jwtKey.PrivateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	keyPair := &KeyPair{
		ID:         jwtKey.KeyID,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		CreatedAt:  jwtKey.CreatedAt,
		Active:     jwtKey.Active,
	}

	if jwtKey.ExpiresAt != nil {
		keyPair.ExpiresAt = *jwtKey.ExpiresAt
	}

	return keyPair, nil
}

// generateAndPersistKey generates a new key pair and persists it to the database
func (dks *DatabaseKeyStore) generateAndPersistKey() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dks.logger.Printf("Generating new JWT signing key")

	// Generate key in cache
	if err := dks.cache.generateNewKey(); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Get the newly generated key
	activeKey := dks.cache.GetActiveKey()
	if activeKey == nil {
		return fmt.Errorf("failed to get generated key")
	}

	// Encode private key to PEM
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(activeKey.PrivateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Encode public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(activeKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Persist to database
	jwtKey := &schema.JWTKey{
		AuditableModel: schema.AuditableModel{
			ID:        xid.New(),
			CreatedAt: activeKey.CreatedAt,
			UpdatedAt: activeKey.CreatedAt,
			CreatedBy: xid.NilID(), // System-generated
			UpdatedBy: xid.NilID(),
		},
		AppID:         dks.appID,
		IsPlatformKey: dks.appID.IsNil(), // Platform key if no app ID
		KeyID:         activeKey.ID,
		Algorithm:     "RS256",
		KeyType:       "RSA",
		PrivateKey:    privateKeyPEM,
		PublicKey:     publicKeyPEM,
		Active:        true,
		ExpiresAt:     &activeKey.ExpiresAt,
	}

	if _, err := dks.db.NewInsert().Model(jwtKey).Exec(ctx); err != nil {
		return fmt.Errorf("failed to persist key to database: %w", err)
	}

	dks.logger.Printf("JWT key generated and persisted: %s", activeKey.ID)

	return nil
}

// GetActiveKey returns the current active signing key
func (dks *DatabaseKeyStore) GetActiveKey() *KeyPair {
	return dks.cache.GetActiveKey()
}

// GetKey retrieves a specific key by ID (used for verification)
func (dks *DatabaseKeyStore) GetKey(kid string) *KeyPair {
	return dks.cache.GetKey(kid)
}

// GetJWKS returns the public JWKS for all active keys
func (dks *DatabaseKeyStore) GetJWKS() (*JWKS, error) {
	keys := dks.cache.GetAllValidKeys()
	jwks := &JWKS{
		Keys: make([]JWK, 0, len(keys)),
	}

	for _, keyPair := range keys {
		// Convert RSA public key to JWK
		nBytes := keyPair.PublicKey.N.Bytes()
		eBytes := big.NewInt(int64(keyPair.PublicKey.E)).Bytes()

		jwk := JWK{
			KeyType:   "RSA",
			Use:       "sig",
			KeyID:     keyPair.ID,
			Algorithm: "RS256",
			N:         base64.RawURLEncoding.EncodeToString(nBytes),
			E:         base64.RawURLEncoding.EncodeToString(eBytes),
		}
		jwks.Keys = append(jwks.Keys, jwk)
	}

	return jwks, nil
}

// GetAllValidKeys returns all valid (non-expired) keys
func (dks *DatabaseKeyStore) GetAllValidKeys() []*KeyPair {
	return dks.cache.GetAllValidKeys()
}

// GetLastRotation returns when keys were last rotated
func (dks *DatabaseKeyStore) GetLastRotation() time.Time {
	return dks.cache.GetLastRotation()
}

// RotateKeys generates a new key and deactivates old ones
func (dks *DatabaseKeyStore) RotateKeys() error {
	return dks.generateAndPersistKey()
}

// ShouldRotate checks if keys should be rotated
func (dks *DatabaseKeyStore) ShouldRotate() bool {
	return dks.cache.ShouldRotate()
}

// NewDatabaseJWKSService creates a JWKS service backed by the database
func NewDatabaseJWKSService(db *bun.DB, appID xid.ID, logger interface{ Printf(string, ...interface{}) }) (*JWKSService, error) {
	keyStore, err := NewDatabaseKeyStore(db, appID, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database key store: %w", err)
	}

	return &JWKSService{
		keyStore: keyStore,
	}, nil
}
