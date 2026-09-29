package authsome

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
)

func hexKey(b byte) string { return hex.EncodeToString(bytes.Repeat([]byte{b}, 32)) }

func TestResolveTokenEncryptor_MissingKeyFailsClosed(t *testing.T) {
	t.Setenv(envTokenEncryptionKeys, "")
	t.Setenv(envTokenEncryptionKey, "")
	_, err := resolveTokenEncryptor(TokenEncryptionConfig{}, nil, false)
	assert.ErrorIs(t, err, ErrTokenEncryptionKeyRequired)

	enc, err := resolveTokenEncryptor(TokenEncryptionConfig{}, nil, true)
	require.NoError(t, err)
	assert.IsType(t, bridge.NoopEncryptor{}, enc, "only a test process may run without a key")
}

func TestResolveTokenEncryptor_InvalidKeyFailsClosed(t *testing.T) {
	t.Setenv(envTokenEncryptionKeys, "")
	t.Setenv(envTokenEncryptionKey, "not-hex")
	_, err := resolveTokenEncryptor(TokenEncryptionConfig{}, nil, true)
	assert.Error(t, err, "an invalid key never degrades to plaintext")

	t.Setenv(envTokenEncryptionKey, hexKey(1)[:32])
	_, err = resolveTokenEncryptor(TokenEncryptionConfig{}, nil, true)
	assert.ErrorIs(t, err, bridge.ErrInvalidKeyLength)

	t.Setenv(envTokenEncryptionKey, "")
	t.Setenv(envTokenEncryptionKeys, "k1")
	_, err = resolveTokenEncryptor(TokenEncryptionConfig{}, nil, true)
	assert.Error(t, err, "a ring entry without a key id is refused")
}

func TestResolveTokenEncryptor_RingWritesUnderFirstKeyAndReadsAll(t *testing.T) {
	t.Setenv(envTokenEncryptionKeys, "")
	t.Setenv(envTokenEncryptionKey, hexKey(1))
	single, err := resolveTokenEncryptor(TokenEncryptionConfig{}, nil, false)
	require.NoError(t, err)
	v1, err := single.Encrypt([]byte("before rotation"))
	require.NoError(t, err)

	t.Setenv(envTokenEncryptionKeys, "k2:"+hexKey(2)+", k1:"+hexKey(1))
	ring, err := resolveTokenEncryptor(TokenEncryptionConfig{}, nil, false)
	require.NoError(t, err)
	pt, err := ring.Decrypt(v1)
	require.NoError(t, err, "rows sealed by the single key still read after moving to a ring")
	assert.Equal(t, "before rotation", string(pt))
	v2, err := ring.Encrypt([]byte("after rotation"))
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(v2, []byte("v2:k2:")), "new values are sealed by the first listed key")

	// Config wins over the environment.
	fromCfg, err := resolveTokenEncryptor(TokenEncryptionConfig{Keys: "only:" + hexKey(3)}, nil, false)
	require.NoError(t, err)
	_, err = fromCfg.Decrypt(v2)
	assert.ErrorIs(t, err, bridge.ErrUnknownKeyID)
}

func TestResolveTokenEncryptor_StrictRefusesPlaintext(t *testing.T) {
	t.Setenv(envTokenEncryptionKeys, "")
	t.Setenv(envTokenEncryptionKey, hexKey(1))
	lenient, err := resolveTokenEncryptor(TokenEncryptionConfig{}, nil, false)
	require.NoError(t, err)
	pt, err := lenient.Decrypt([]byte("legacy"))
	require.NoError(t, err)
	assert.Equal(t, "legacy", string(pt))

	strict, err := resolveTokenEncryptor(TokenEncryptionConfig{Strict: true}, nil, false)
	require.NoError(t, err)
	_, err = strict.Decrypt([]byte("legacy"))
	assert.ErrorIs(t, err, bridge.ErrPlaintextRefused)

	t.Setenv(envTokenEncryptionStrict, "true")
	strictEnv, err := resolveTokenEncryptor(TokenEncryptionConfig{}, nil, false)
	require.NoError(t, err)
	_, err = strictEnv.Decrypt([]byte("legacy"))
	assert.ErrorIs(t, err, bridge.ErrPlaintextRefused)
}

func TestResolveAPIKeyPepper_AcceptsHexOrRaw(t *testing.T) {
	t.Setenv(envAPIKeyPepper, "")
	assert.Nil(t, resolveAPIKeyPepper(""), "no pepper is a valid, if weaker, configuration")
	assert.Equal(t, bytes.Repeat([]byte{4}, 32), resolveAPIKeyPepper(hexKey(4)), "hex is decoded")
	assert.Equal(t, []byte("plain-secret-value"), resolveAPIKeyPepper("plain-secret-value"), "anything else is used as given")
	t.Setenv(envAPIKeyPepper, hexKey(5))
	assert.Equal(t, bytes.Repeat([]byte{5}, 32), resolveAPIKeyPepper(""), "the environment fills in when configuration is empty")
	assert.Equal(t, []byte("from-config"), resolveAPIKeyPepper("from-config"), "configuration wins over the environment")
}
