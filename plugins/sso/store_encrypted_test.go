package sso

import (
	"context"
	"strings"
	"testing"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
)

func testEncryptor(t *testing.T) bridge.Encryptor {
	t.Helper()
	enc, err := bridge.NewAESGCMEncryptor([]byte("0123456789abcdef0123456789abcdef")) // 32 bytes
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	return enc
}

// TestEncryptedStore_SecretsAtRest verifies the OIDC client secret and SAML SP
// private key are stored as ciphertext but read back transparently decrypted.
func TestEncryptedStore_SecretsAtRest(t *testing.T) {
	ctx := context.Background()
	inner := NewMemoryStore()
	store := NewEncryptedStore(inner, testEncryptor(t))

	const secret = "super-secret-oidc"
	const spKey = "-----BEGIN PRIVATE KEY-----xyz"

	conn := &Connection{
		ID:           id.NewSSOConnectionID(),
		AppID:        id.NewAppID(),
		EnvID:        id.NewEnvironmentID().String(),
		OrgID:        id.NewOrgID(),
		Provider:     "acme.com",
		Protocol:     "oidc",
		Domain:       "acme.com",
		ClientSecret: secret,
		SPPrivateKey: spKey,
		Active:       true,
	}
	if err := store.CreateConnection(ctx, conn); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Read the raw at-rest row (inner store) — secrets must be encrypted.
	raw, err := inner.GetConnection(ctx, conn.ID)
	if err != nil {
		t.Fatalf("inner get: %v", err)
	}
	if raw.ClientSecret == secret || !strings.HasPrefix(raw.ClientSecret, "v1:") {
		t.Fatalf("ClientSecret not encrypted at rest: %q", raw.ClientSecret)
	}
	if raw.SPPrivateKey == spKey || !strings.HasPrefix(raw.SPPrivateKey, "v1:") {
		t.Fatalf("SPPrivateKey not encrypted at rest: %q", raw.SPPrivateKey)
	}

	// Through the encrypted store, reads are transparently decrypted.
	got, err := store.GetConnection(ctx, conn.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ClientSecret != secret {
		t.Fatalf("ClientSecret not decrypted: %q", got.ClientSecret)
	}
	if got.SPPrivateKey != spKey {
		t.Fatalf("SPPrivateKey not decrypted: %q", got.SPPrivateKey)
	}
}

// TestEncryptedStore_LegacyPlaintextPassthrough ensures rows written before
// encryption was enabled remain readable (Decrypt passes non-prefixed values
// through), so enabling encryption doesn't break existing connections.
func TestEncryptedStore_LegacyPlaintextPassthrough(t *testing.T) {
	ctx := context.Background()
	inner := NewMemoryStore()

	// Simulate a legacy plaintext row by writing straight to the inner store.
	conn := &Connection{
		ID:           id.NewSSOConnectionID(),
		AppID:        id.NewAppID(),
		EnvID:        id.NewEnvironmentID().String(),
		Provider:     "acme.com",
		Protocol:     "oidc",
		Domain:       "acme.com",
		ClientSecret: "legacy-plaintext",
	}
	if err := inner.CreateConnection(ctx, conn); err != nil {
		t.Fatalf("inner create: %v", err)
	}

	store := NewEncryptedStore(inner, testEncryptor(t))
	got, err := store.GetConnection(ctx, conn.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ClientSecret != "legacy-plaintext" {
		t.Fatalf("legacy plaintext not returned as-is: %q", got.ClientSecret)
	}
}
