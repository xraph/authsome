package bridge

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, 32)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return k
}

func TestAESGCM_RoundTrip(t *testing.T) {
	enc, err := NewAESGCMEncryptor(newTestKey(t))
	require.NoError(t, err)

	plaintext := []byte("super-secret-oauth-access-token")
	ct, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(ct), "v1:"), "envelope must be v1-prefixed")
	require.False(t, bytes.Equal(plaintext, ct), "ciphertext must not equal plaintext")

	pt, err := enc.Decrypt(ct)
	require.NoError(t, err)
	require.Equal(t, plaintext, pt)
}

func TestAESGCM_DifferentNoncesEachEncrypt(t *testing.T) {
	enc, err := NewAESGCMEncryptor(newTestKey(t))
	require.NoError(t, err)

	pt := []byte("repeat-me")
	a, err := enc.Encrypt(pt)
	require.NoError(t, err)
	b, err := enc.Encrypt(pt)
	require.NoError(t, err)
	require.False(t, bytes.Equal(a, b), "nonce should randomize each output")
}

func TestAESGCM_TamperDetection(t *testing.T) {
	enc, err := NewAESGCMEncryptor(newTestKey(t))
	require.NoError(t, err)

	ct, err := enc.Encrypt([]byte("hello world"))
	require.NoError(t, err)

	// Flip a byte in the base64 portion (after the v1: prefix).
	tampered := make([]byte, len(ct))
	copy(tampered, ct)
	// pick a byte well past v1:
	idx := len(tampered) - 4
	if tampered[idx] == 'A' {
		tampered[idx] = 'B'
	} else {
		tampered[idx] = 'A'
	}

	_, err = enc.Decrypt(tampered)
	require.Error(t, err)
}

func TestAESGCM_WrongKeyFails(t *testing.T) {
	encA, err := NewAESGCMEncryptor(newTestKey(t))
	require.NoError(t, err)
	encB, err := NewAESGCMEncryptor(newTestKey(t))
	require.NoError(t, err)

	ct, err := encA.Encrypt([]byte("mine"))
	require.NoError(t, err)

	_, err = encB.Decrypt(ct)
	require.Error(t, err)
}

func TestAESGCM_LegacyPlaintextPassthrough(t *testing.T) {
	enc, err := NewAESGCMEncryptor(newTestKey(t))
	require.NoError(t, err)

	legacy := []byte("legacy-plaintext-row")
	got, err := enc.Decrypt(legacy)
	require.NoError(t, err)
	require.Equal(t, legacy, got)
}

func TestAESGCM_KeyLengthValidation(t *testing.T) {
	_, err := NewAESGCMEncryptor(make([]byte, 16))
	require.Error(t, err)
	_, err = NewAESGCMEncryptor(make([]byte, 31))
	require.Error(t, err)
	_, err = NewAESGCMEncryptor(nil)
	require.Error(t, err)
	_, err = NewAESGCMEncryptor(make([]byte, 32))
	require.NoError(t, err)
}

func TestNoopEncryptor_PassthroughBoth(t *testing.T) {
	var enc Encryptor = NoopEncryptor{}

	pt := []byte("noop")
	ct, err := enc.Encrypt(pt)
	require.NoError(t, err)
	require.Equal(t, pt, ct)

	got, err := enc.Decrypt(ct)
	require.NoError(t, err)
	require.Equal(t, pt, got)
}
