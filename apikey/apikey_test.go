package apikey

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// APIKey domain tests
// ──────────────────────────────────────────────────

func TestAPIKey_IsExpired_NoExpiry(t *testing.T) {
	k := &APIKey{ID: id.NewAPIKeyID()}
	assert.False(t, k.IsExpired(), "key without ExpiresAt should not be expired")
}

func TestAPIKey_IsExpired_FutureExpiry(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	k := &APIKey{ID: id.NewAPIKeyID(), ExpiresAt: &future}
	assert.False(t, k.IsExpired())
}

func TestAPIKey_IsExpired_PastExpiry(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	k := &APIKey{ID: id.NewAPIKeyID(), ExpiresAt: &past}
	assert.True(t, k.IsExpired())
}

func TestAPIKey_IsValid(t *testing.T) {
	k := &APIKey{ID: id.NewAPIKeyID()}
	assert.True(t, k.IsValid(), "non-revoked, non-expired key should be valid")
}

func TestAPIKey_IsValid_Revoked(t *testing.T) {
	k := &APIKey{ID: id.NewAPIKeyID(), Revoked: true}
	assert.False(t, k.IsValid(), "revoked key should not be valid")
}

func TestAPIKey_IsValid_Expired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	k := &APIKey{ID: id.NewAPIKeyID(), ExpiresAt: &past}
	assert.False(t, k.IsValid(), "expired key should not be valid")
}

func TestAPIKey_IsValid_RevokedAndExpired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	k := &APIKey{ID: id.NewAPIKeyID(), Revoked: true, ExpiresAt: &past}
	assert.False(t, k.IsValid())
}

// ──────────────────────────────────────────────────
// Key generation and hashing tests
// ──────────────────────────────────────────────────

func TestGenerateKey(t *testing.T) {
	raw, hash, prefix, err := GenerateKey()
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, prefix)

	// Raw key starts with marker
	assert.True(t, strings.HasPrefix(raw, keyMarker))

	// Prefix starts with marker
	assert.True(t, strings.HasPrefix(prefix, keyMarker))

	// Prefix is a prefix of the raw key
	assert.True(t, strings.HasPrefix(raw, prefix))
}

func TestGenerateKey_Unique(t *testing.T) {
	raw1, hash1, _, err1 := GenerateKey()
	require.NoError(t, err1)
	raw2, hash2, _, err2 := GenerateKey()
	require.NoError(t, err2)

	assert.NotEqual(t, raw1, raw2)
	assert.NotEqual(t, hash1, hash2)
}

func TestHashKey_Deterministic(t *testing.T) {
	raw := "ask_abcdef1234567890"
	h1 := HashKey(raw)
	h2 := HashKey(raw)
	assert.Equal(t, h1, h2)
}

func TestVerifyKey_Valid(t *testing.T) {
	raw, hash, _, err := GenerateKey()
	require.NoError(t, err)
	assert.True(t, VerifyKey(raw, hash))
}

func TestVerifyKey_Invalid(t *testing.T) {
	raw, _, _, err := GenerateKey()
	require.NoError(t, err)
	assert.False(t, VerifyKey(raw, "wrong-hash"))
}

func TestVerifyKey_WrongRaw(t *testing.T) {
	_, hash, _, err := GenerateKey()
	require.NoError(t, err)
	assert.False(t, VerifyKey("wrong-raw-key", hash))
}

func TestVerifyKey_ConstantTimeComparison(t *testing.T) {
	// Verify that VerifyKey correctly validates keys after the
	// subtle.ConstantTimeCompare change (H4 security fix).
	raw, hash, _, err := GenerateKey()
	require.NoError(t, err)

	// Correct key should verify.
	assert.True(t, VerifyKey(raw, hash))

	// Slightly modified hash should not verify.
	if len(hash) > 0 {
		modified := hash[:len(hash)-1] + "0"
		if modified == hash {
			modified = hash[:len(hash)-1] + "1"
		}
		assert.False(t, VerifyKey(raw, modified), "modified hash should not verify")
	}

	// Empty inputs should not verify.
	assert.False(t, VerifyKey("", hash))
	assert.False(t, VerifyKey(raw, ""))
	assert.False(t, VerifyKey("", ""))
}

// ──────────────────────────────────────────────────
// Environment key generation tests
// ──────────────────────────────────────────────────

func TestGenerateKeyForEnvironment_Development(t *testing.T) {
	raw, hash, prefix, err := GenerateKeyForEnvironment("development")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw, keyMarkerTest), "dev key should start with sk_test_")
	assert.True(t, strings.HasPrefix(prefix, keyMarkerTest))
	assert.NotEmpty(t, hash)
	assert.True(t, VerifyKey(raw, hash))
}

func TestGenerateKeyForEnvironment_Staging(t *testing.T) {
	raw, hash, prefix, err := GenerateKeyForEnvironment("staging")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw, keyMarkerStg), "staging key should start with sk_stg_")
	assert.True(t, strings.HasPrefix(prefix, keyMarkerStg))
	assert.NotEmpty(t, hash)
	assert.True(t, VerifyKey(raw, hash))
}

func TestGenerateKeyForEnvironment_Production(t *testing.T) {
	raw, hash, prefix, err := GenerateKeyForEnvironment("production")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw, keyMarkerLive), "production key should start with sk_live_")
	assert.True(t, strings.HasPrefix(prefix, keyMarkerLive))
	assert.NotEmpty(t, hash)
	assert.True(t, VerifyKey(raw, hash))
}

// ──────────────────────────────────────────────────
// DetectEnvironmentType tests
// ──────────────────────────────────────────────────

func TestDetectEnvironmentType_Test(t *testing.T) {
	envType, ok := DetectEnvironmentType("sk_test_abcdef1234567890")
	assert.True(t, ok)
	assert.Equal(t, environment.TypeDevelopment, envType)
}

func TestDetectEnvironmentType_Staging(t *testing.T) {
	envType, ok := DetectEnvironmentType("sk_stg_abcdef1234567890")
	assert.True(t, ok)
	assert.Equal(t, environment.TypeStaging, envType)
}

func TestDetectEnvironmentType_Live(t *testing.T) {
	envType, ok := DetectEnvironmentType("sk_live_abcdef1234567890")
	assert.True(t, ok)
	assert.Equal(t, environment.TypeProduction, envType)
}

func TestDetectEnvironmentType_Legacy(t *testing.T) {
	envType, ok := DetectEnvironmentType("ask_abcdef1234567890")
	assert.True(t, ok)
	assert.Equal(t, environment.TypeProduction, envType, "legacy keys treated as production")
}

func TestDetectEnvironmentType_Unknown(t *testing.T) {
	_, ok := DetectEnvironmentType("unknown_prefix_key")
	assert.False(t, ok)
}

// ──────────────────────────────────────────────────
// EnvironmentKeyMarker tests
// ──────────────────────────────────────────────────

func TestEnvironmentKeyMarker_Development(t *testing.T) {
	assert.Equal(t, keyMarkerTest, EnvironmentKeyMarker(environment.TypeDevelopment))
}

func TestEnvironmentKeyMarker_Staging(t *testing.T) {
	assert.Equal(t, keyMarkerStg, EnvironmentKeyMarker(environment.TypeStaging))
}

func TestEnvironmentKeyMarker_Production(t *testing.T) {
	assert.Equal(t, keyMarkerLive, EnvironmentKeyMarker(environment.TypeProduction))
}

func TestEnvironmentKeyMarker_Unknown(t *testing.T) {
	assert.Equal(t, keyMarker, EnvironmentKeyMarker(environment.Type("custom")))
}

// ──────────────────────────────────────────────────
// StripKeyMarker tests
// ──────────────────────────────────────────────────

func TestStripKeyMarker_TestKey(t *testing.T) {
	assert.Equal(t, "abcdef", StripKeyMarker("sk_test_abcdef"))
}

func TestStripKeyMarker_LiveKey(t *testing.T) {
	assert.Equal(t, "abcdef", StripKeyMarker("sk_live_abcdef"))
}

func TestStripKeyMarker_LegacyKey(t *testing.T) {
	assert.Equal(t, "abcdef", StripKeyMarker("ask_abcdef"))
}

func TestStripKeyMarker_NoMarker(t *testing.T) {
	assert.Equal(t, "plain_key", StripKeyMarker("plain_key"))
}
