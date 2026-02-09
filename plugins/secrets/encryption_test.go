package secrets

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestEncryptionService_GenerateMasterKey(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey() error = %v", err)
	}

	// Decode and check length
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("Generated key is not valid base64: %v", err)
	}

	if len(decoded) != MasterKeyLength {
		t.Errorf("Generated key length = %d, want %d", len(decoded), MasterKeyLength)
	}
}

func TestEncryptionService_NewEncryptionService(t *testing.T) {
	tests := []struct {
		name      string
		masterKey string
		wantErr   bool
	}{
		{
			name:      "valid key",
			masterKey: base64.StdEncoding.EncodeToString(make([]byte, MasterKeyLength)),
			wantErr:   false,
		},
		{
			name:      "empty key",
			masterKey: "",
			wantErr:   true,
		},
		{
			name:      "invalid base64",
			masterKey: "not-valid-base64!@#$",
			wantErr:   true,
		},
		{
			name:      "wrong length",
			masterKey: base64.StdEncoding.EncodeToString(make([]byte, 16)),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptionService(tt.masterKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptionService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptionService_EncryptDecrypt(t *testing.T) {
	key, _ := GenerateMasterKey()

	svc, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
		appID     string
		envID     string
	}{
		{
			name:      "simple string",
			plaintext: []byte("hello world"),
			appID:     "app1",
			envID:     "env1",
		},
		{
			name:      "empty string",
			plaintext: []byte(""),
			appID:     "app2",
			envID:     "env2",
		},
		{
			name:      "json data",
			plaintext: []byte(`{"username": "admin", "password": "secret123"}`),
			appID:     "app3",
			envID:     "env3",
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd},
			appID:     "app4",
			envID:     "env4",
		},
		{
			name:      "large data",
			plaintext: bytes.Repeat([]byte("a"), 10000),
			appID:     "app5",
			envID:     "env5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, nonce, err := svc.Encrypt(tt.plaintext, tt.appID, tt.envID)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Verify ciphertext is different from plaintext
			if len(tt.plaintext) > 0 && bytes.Equal(ciphertext, tt.plaintext) {
				t.Error("Ciphertext should not equal plaintext")
			}

			// Verify nonce length
			if len(nonce) != NonceLength {
				t.Errorf("Nonce length = %d, want %d", len(nonce), NonceLength)
			}

			// Decrypt
			decrypted, err := svc.Decrypt(ciphertext, nonce, tt.appID, tt.envID)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify decrypted matches original
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptionService_TenantIsolation(t *testing.T) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)

	plaintext := []byte("secret data")

	// Encrypt with app1/env1
	ciphertext1, nonce1, _ := svc.Encrypt(plaintext, "app1", "env1")

	// Encrypt with app2/env2
	ciphertext2, nonce2, _ := svc.Encrypt(plaintext, "app2", "env2")

	// Ciphertexts should be different (different derived keys)
	if bytes.Equal(ciphertext1, ciphertext2) && bytes.Equal(nonce1, nonce2) {
		t.Error("Same plaintext encrypted for different tenants should produce different ciphertexts")
	}

	// Decrypting with wrong tenant should fail
	_, err := svc.Decrypt(ciphertext1, nonce1, "app2", "env2")
	if err == nil {
		t.Error("Decrypting with wrong tenant should fail")
	}
}

func TestEncryptionService_DeriveKey(t *testing.T) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)

	// Same inputs should produce same key
	key1 := svc.DeriveKey("app1", "env1")
	key2 := svc.DeriveKey("app1", "env1")

	if !bytes.Equal(key1, key2) {
		t.Error("DeriveKey should be deterministic for same inputs")
	}

	// Different inputs should produce different keys
	key3 := svc.DeriveKey("app2", "env1")
	if bytes.Equal(key1, key3) {
		t.Error("DeriveKey should produce different keys for different inputs")
	}

	// Key should be cached
	svc.cacheMu.RLock()
	_, cached := svc.keyCache["app1:env1"]
	svc.cacheMu.RUnlock()

	if !cached {
		t.Error("Derived key should be cached")
	}
}

func TestEncryptionService_ClearKeyCache(t *testing.T) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)

	// Generate some cached keys
	svc.DeriveKey("app1", "env1")
	svc.DeriveKey("app2", "env2")

	// Clear cache
	svc.ClearKeyCache()

	svc.cacheMu.RLock()
	cacheLen := len(svc.keyCache)
	svc.cacheMu.RUnlock()

	if cacheLen != 0 {
		t.Errorf("Cache should be empty after clear, got %d items", cacheLen)
	}
}

func TestEncryptionService_ClearKeyForTenant(t *testing.T) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)

	// Generate cached keys
	svc.DeriveKey("app1", "env1")
	svc.DeriveKey("app2", "env2")

	// Clear specific tenant
	svc.ClearKeyForTenant("app1", "env1")

	svc.cacheMu.RLock()
	_, exists1 := svc.keyCache["app1:env1"]
	_, exists2 := svc.keyCache["app2:env2"]
	svc.cacheMu.RUnlock()

	if exists1 {
		t.Error("app1:env1 should be cleared from cache")
	}

	if !exists2 {
		t.Error("app2:env2 should still be in cache")
	}
}

func TestEncryptionService_TestEncryption(t *testing.T) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)

	err := svc.TestEncryption()
	if err != nil {
		t.Errorf("TestEncryption() error = %v", err)
	}
}

func TestEncryptionService_ReEncrypt(t *testing.T) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)

	plaintext := []byte("secret data")

	// Encrypt with old tenant
	ciphertext, nonce, _ := svc.Encrypt(plaintext, "oldApp", "oldEnv")

	// Re-encrypt for new tenant
	newCiphertext, newNonce, err := svc.ReEncrypt(ciphertext, nonce, "oldApp", "oldEnv", "newApp", "newEnv")
	if err != nil {
		t.Fatalf("ReEncrypt() error = %v", err)
	}

	// Decrypt with new tenant
	decrypted, err := svc.Decrypt(newCiphertext, newNonce, "newApp", "newEnv")
	if err != nil {
		t.Fatalf("Decrypt after ReEncrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted = %v, want %v", decrypted, plaintext)
	}

	// Old credentials should not work on new ciphertext
	_, err = svc.Decrypt(newCiphertext, newNonce, "oldApp", "oldEnv")
	if err == nil {
		t.Error("Old tenant credentials should not decrypt re-encrypted data")
	}
}

// Benchmark tests.
func BenchmarkEncrypt(b *testing.B) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)
	plaintext := []byte("secret data for benchmarking")

	for b.Loop() {
		svc.Encrypt(plaintext, "app", "env")
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)
	plaintext := []byte("secret data for benchmarking")
	ciphertext, nonce, _ := svc.Encrypt(plaintext, "app", "env")

	for b.Loop() {
		svc.Decrypt(ciphertext, nonce, "app", "env")
	}
}

func BenchmarkDeriveKey(b *testing.B) {
	key, _ := GenerateMasterKey()
	svc, _ := NewEncryptionService(key)

	for b.Loop() {
		// Clear cache to benchmark actual derivation
		svc.ClearKeyCache()
		svc.DeriveKey("app", "env")
	}
}
