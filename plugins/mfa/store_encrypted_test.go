package mfa_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	mfa "github.com/xraph/authsome/plugins/mfa"
)

func aesEnc(t *testing.T) bridge.Encryptor {
	t.Helper()
	enc, err := bridge.NewAESGCMEncryptor(bytes.Repeat([]byte{0x2a}, 32))
	require.NoError(t, err)
	return enc
}

// TestEncryptedStore_EncryptsSecretAtRest pins that the TOTP secret is stored
// encrypted in the underlying store and decrypted transparently on read.
func TestEncryptedStore_EncryptsSecretAtRest(t *testing.T) {
	ctx := context.Background()
	inner := mfa.NewMemoryStore()
	store := mfa.NewEncryptedStore(inner, aesEnc(t))

	userID := id.NewUserID()
	require.NoError(t, store.CreateEnrollment(ctx, &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "SUPERSECRETVALUE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))

	// The underlying store must not hold the plaintext secret.
	raw, err := inner.GetEnrollment(ctx, userID, "totp")
	require.NoError(t, err)
	assert.NotEqual(t, "SUPERSECRETVALUE", raw.Secret, "secret must be encrypted at rest")
	assert.NotEmpty(t, raw.Secret)

	// Reading through the wrapper returns the plaintext secret.
	got, err := store.GetEnrollment(ctx, userID, "totp")
	require.NoError(t, err)
	assert.Equal(t, "SUPERSECRETVALUE", got.Secret)
}

// TestEncryptedStore_ReadsLegacyPlaintext confirms the wrapper is backward
// compatible: a pre-existing plaintext secret decrypts to itself.
func TestEncryptedStore_ReadsLegacyPlaintext(t *testing.T) {
	ctx := context.Background()
	inner := mfa.NewMemoryStore()
	userID := id.NewUserID()
	require.NoError(t, inner.CreateEnrollment(ctx, &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "LEGACYPLAINTEXT",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))

	store := mfa.NewEncryptedStore(inner, aesEnc(t))
	got, err := store.GetEnrollment(ctx, userID, "totp")
	require.NoError(t, err)
	assert.Equal(t, "LEGACYPLAINTEXT", got.Secret, "legacy plaintext secret must read back unchanged")
}
