package apikey

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withPepper(t *testing.T, p []byte) {
	t.Helper()
	SetPepper(p)
	t.Cleanup(func() { SetPepper(nil) })
}

func TestHashKey_PepperChangesTheDigest(t *testing.T) {
	SetPepper(nil)
	raw, plain, _, err := GenerateKey()
	require.NoError(t, err)
	sum := sha256.Sum256([]byte(raw))
	assert.Equal(t, hex.EncodeToString(sum[:]), plain, "without a pepper the digest is plain SHA-256")

	withPepper(t, []byte("server-side-secret"))
	peppered := HashKey(raw)
	assert.NotEqual(t, plain, peppered, "the pepper changes the stored digest")
	assert.Len(t, peppered, 64)
	withPepper(t, []byte("a-different-secret"))
	assert.NotEqual(t, peppered, HashKey(raw), "the digest depends on the pepper")
}

func TestVerifyKey_AcceptsLegacyDigestsUnderAPepper(t *testing.T) {
	SetPepper(nil)
	raw, legacy, _, err := GenerateKey()
	require.NoError(t, err)

	withPepper(t, []byte("server-side-secret"))
	assert.True(t, VerifyKey(raw, legacy), "a key hashed before the pepper keeps working")
	assert.True(t, NeedsRehash(raw, legacy), "and is flagged for rewriting")
	current := HashKey(raw)
	assert.True(t, VerifyKey(raw, current))
	assert.False(t, NeedsRehash(raw, current))
	assert.False(t, VerifyKey(raw+"x", legacy))
	assert.False(t, VerifyKey(raw+"x", current))

	SetPepper(nil)
	assert.False(t, VerifyKey(raw, current), "without the pepper a peppered digest cannot be checked")
	assert.False(t, NeedsRehash(raw, legacy), "nothing to rewrite without a pepper")
}
