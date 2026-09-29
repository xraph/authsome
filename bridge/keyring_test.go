package bridge

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ringKey(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func TestKeyring_RoundTripNamesTheWriter(t *testing.T) {
	ring, err := NewKeyring("k2", map[string][]byte{"k1": ringKey(1), "k2": ringKey(2)}, []string{"k2", "k1"})
	require.NoError(t, err)
	ct, err := ring.Encrypt([]byte("secret"))
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(ct, []byte("v2:k2:")), "the envelope names the writer: %s", ct)
	pt, err := ring.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "secret", string(pt))
}

func TestKeyring_RotationKeepsOldRowsReadable(t *testing.T) {
	old, err := NewKeyring("k1", map[string][]byte{"k1": ringKey(1)}, []string{"k1"})
	require.NoError(t, err)
	ct, err := old.Encrypt([]byte("written under k1"))
	require.NoError(t, err)

	rotated, err := NewKeyring("k2", map[string][]byte{"k1": ringKey(1), "k2": ringKey(2)}, []string{"k2", "k1"})
	require.NoError(t, err)
	pt, err := rotated.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "written under k1", string(pt))

	retired, err := NewKeyring("k2", map[string][]byte{"k2": ringKey(2)}, []string{"k2"})
	require.NoError(t, err)
	_, err = retired.Decrypt(ct)
	assert.ErrorIs(t, err, ErrUnknownKeyID, "dropping a key that still has rows is reported, not silently unreadable")
}

func TestKeyring_OpensSingleKeyEnvelopes(t *testing.T) {
	single, err := NewAESGCMEncryptor(ringKey(7))
	require.NoError(t, err)
	ct, err := single.Encrypt([]byte("v1 row"))
	require.NoError(t, err)

	ring, err := NewKeyring("new", map[string][]byte{"new": ringKey(9), "old": ringKey(7)}, []string{"new", "old"})
	require.NoError(t, err)
	pt, err := ring.Decrypt(ct)
	require.NoError(t, err, "a v1 envelope is tried against every key")
	assert.Equal(t, "v1 row", string(pt))

	wrong, err := NewKeyring("x", map[string][]byte{"x": ringKey(3)}, []string{"x"})
	require.NoError(t, err)
	_, err = wrong.Decrypt(ct)
	assert.Error(t, err, "a ring without the v1 key cannot open it")
}

func TestKeyring_TamperAndUnknownKeyAreRefused(t *testing.T) {
	ring, err := NewKeyring("k1", map[string][]byte{"k1": ringKey(1)}, []string{"k1"})
	require.NoError(t, err)
	ct, err := ring.Encrypt([]byte("intact"))
	require.NoError(t, err)
	ct[len(ct)-1] ^= 0x01
	_, err = ring.Decrypt(ct)
	assert.Error(t, err)
	_, err = ring.Decrypt([]byte("v2:nope:AAAA"))
	assert.ErrorIs(t, err, ErrUnknownKeyID)
}

func TestKeyring_RejectsBadConfiguration(t *testing.T) {
	_, err := NewKeyring("k1", map[string][]byte{"k1": ringKey(1)[:16]}, nil)
	assert.ErrorIs(t, err, ErrInvalidKeyLength)
	_, err = NewKeyring("k2", map[string][]byte{"k1": ringKey(1)}, nil)
	assert.Error(t, err, "the writer must be in the ring")
	_, err = NewKeyring("a:b", map[string][]byte{"a:b": ringKey(1)}, nil)
	assert.ErrorIs(t, err, ErrBadKeyID)
}

func TestStrict_RefusesPlaintextAndPassesEnvelopes(t *testing.T) {
	ring, err := NewKeyring("k1", map[string][]byte{"k1": ringKey(1)}, []string{"k1"})
	require.NoError(t, err)
	pt, err := ring.Decrypt([]byte("legacy plaintext"))
	require.NoError(t, err)
	assert.Equal(t, "legacy plaintext", string(pt), "a lenient ring passes legacy rows through")

	strict := NewStrict(ring)
	_, err = strict.Decrypt([]byte("legacy plaintext"))
	assert.ErrorIs(t, err, ErrPlaintextRefused)
	ct, err := strict.Encrypt([]byte("sealed"))
	require.NoError(t, err)
	pt, err = strict.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "sealed", string(pt))
	empty, err := strict.Decrypt(nil)
	require.NoError(t, err, "an unset column is not a plaintext secret")
	assert.Empty(t, empty)
}
