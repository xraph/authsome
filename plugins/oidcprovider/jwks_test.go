package oidcprovider

import (
	"testing"
)

func TestNewJWKSServiceFromFiles(t *testing.T) {
	// Test loading keys from the generated files
	privateKeyPath := "../../keys/oidc-private.pem"
	publicKeyPath := "../../keys/oidc-public.pem"

	// Test with valid key files
	service, err := NewJWKSServiceFromFiles(privateKeyPath, publicKeyPath, "24h", "168h")
	if err != nil {
		t.Fatalf("Failed to create JWKS service from files: %v", err)
	}

	if service == nil {
		t.Fatal("JWKS service should not be nil")
	}

	// Test getting active key
	activeKey := service.keyStore.GetActiveKey()
	if activeKey == nil {
		t.Fatal("Active key should not be nil")
	}

	// Test that the key has a valid ID
	if activeKey.ID == "" {
		t.Fatal("Key ID should not be empty")
	}

	// Test JWKS generation
	jwks, err := service.GetJWKS()
	if err != nil {
		t.Fatalf("Failed to get JWKS: %v", err)
	}

	if len(jwks.Keys) == 0 {
		t.Fatal("JWKS should contain at least one key")
	}

	// Verify the key in JWKS has the same ID as the active key
	if jwks.Keys[0].KeyID != activeKey.ID {
		t.Fatalf("JWKS key ID (%s) should match active key ID (%s)", jwks.Keys[0].KeyID, activeKey.ID)
	}
}

func TestNewJWKSServiceFromFiles_InvalidPath(t *testing.T) {
	// Test with invalid key files
	_, err := NewJWKSServiceFromFiles("nonexistent.pem", "nonexistent.pem", "24h", "168h")
	if err == nil {
		t.Fatal("Should fail with nonexistent key files")
	}
}
