package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/xraph/authsome/environment"
)

const (
	// rawKeyLen is the length in bytes of the random part of an API key.
	rawKeyLen = 32

	// prefixLen is how many characters of the hex-encoded key to keep as the visible prefix.
	prefixLen = 8

	// keyMarker is the legacy prefix for all raw API keys (treated as production).
	keyMarker = "ask_"

	// Environment-specific key markers.
	keyMarkerTest = "sk_test_" // development
	keyMarkerStg  = "sk_stg_"  // staging
	keyMarkerLive = "sk_live_" // production
)

// GenerateKey creates a new API key, returning the raw key (shown to user once)
// and the hashed value (stored in the database).
func GenerateKey() (raw string, hash string, prefix string, err error) {
	b := make([]byte, rawKeyLen)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("apikey: generate random bytes: %w", err)
	}

	hexKey := hex.EncodeToString(b)
	raw = keyMarker + hexKey

	prefix = raw[:prefixLen+len(keyMarker)]

	hash = HashKey(raw)

	return raw, hash, prefix, nil
}

// HashKey returns the SHA-256 hash of a raw API key.
func HashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// VerifyKey checks whether the raw key matches the stored hash.
func VerifyKey(raw, hash string) bool {
	return HashKey(raw) == hash
}

// EnvironmentKeyMarker returns the key prefix marker for a given environment type.
func EnvironmentKeyMarker(envType environment.Type) string {
	switch envType {
	case environment.TypeDevelopment:
		return keyMarkerTest
	case environment.TypeStaging:
		return keyMarkerStg
	case environment.TypeProduction:
		return keyMarkerLive
	default:
		return keyMarker
	}
}

// GenerateKeyForEnvironment creates a new API key with an environment-specific prefix.
func GenerateKeyForEnvironment(envType environment.Type) (raw string, hash string, prefix string, err error) {
	b := make([]byte, rawKeyLen)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("apikey: generate random bytes: %w", err)
	}

	marker := EnvironmentKeyMarker(envType)
	hexKey := hex.EncodeToString(b)
	raw = marker + hexKey

	prefix = raw[:prefixLen+len(marker)]

	hash = HashKey(raw)

	return raw, hash, prefix, nil
}

// DetectEnvironmentType determines the environment type from a raw API key
// by examining its prefix marker. Returns the detected type and true if
// recognized, or empty and false for unknown keys.
func DetectEnvironmentType(rawKey string) (environment.Type, bool) {
	switch {
	case strings.HasPrefix(rawKey, keyMarkerTest):
		return environment.TypeDevelopment, true
	case strings.HasPrefix(rawKey, keyMarkerStg):
		return environment.TypeStaging, true
	case strings.HasPrefix(rawKey, keyMarkerLive):
		return environment.TypeProduction, true
	case strings.HasPrefix(rawKey, keyMarker):
		// Legacy keys are treated as production.
		return environment.TypeProduction, true
	default:
		return "", false
	}
}

// StripKeyMarker removes the environment/legacy marker prefix from a raw key,
// returning just the hex-encoded random part. Useful for prefix extraction.
func StripKeyMarker(rawKey string) string {
	for _, marker := range []string{keyMarkerTest, keyMarkerStg, keyMarkerLive, keyMarker} {
		if strings.HasPrefix(rawKey, marker) {
			return rawKey[len(marker):]
		}
	}
	return rawKey
}
