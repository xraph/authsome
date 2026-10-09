package apikey

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/xraph/authsome/environment"
)

const (
	// rawKeyLen is the length in bytes of the random part of an API key.
	rawKeyLen = 32

	// prefixLen is how many characters of the hex-encoded key to keep as the visible prefix.
	prefixLen = 8

	// keyMarker is the legacy prefix for all raw API keys (treated as production).
	keyMarker = "ask_"

	// Environment-specific secret key markers.
	keyMarkerTest = "sk_test_" // development
	keyMarkerStg  = "sk_stg_"  // staging
	keyMarkerLive = "sk_live_" // production

	// Environment-specific public key markers.
	pkMarkerTest = "pk_test_" // development
	pkMarkerStg  = "pk_stg_"  // staging
	pkMarkerLive = "pk_live_" // production
)

// GenerateKey creates a new API key, returning the raw key (shown to user once)
// and the hashed value (stored in the database).
func GenerateKey() (raw, hash, prefix string, err error) {
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

// pepper is the process-wide secret mixed into key digests. It is set once
// at engine construction; a nil pepper means plain SHA-256, the digest every
// release before the pepper produced.
var pepper atomic.Pointer[[]byte]

// SetPepper installs the server-side secret mixed into every API key digest
// from now on. With a pepper set, HashKey is HMAC-SHA256 over the raw key,
// so a copy of the key table cannot be checked against a guessed or leaked
// key without the pepper too. Passing nil returns to plain SHA-256.
func SetPepper(p []byte) {
	if len(p) == 0 {
		pepper.Store(nil)
		return
	}
	cp := append([]byte(nil), p...)
	pepper.Store(&cp)
}

// HashKey returns the digest stored for a raw API key: HMAC-SHA256 under the
// pepper when one is set, plain SHA-256 otherwise.
func HashKey(raw string) string {
	if p := pepper.Load(); p != nil {
		mac := hmac.New(sha256.New, *p)
		mac.Write([]byte(raw))
		return hex.EncodeToString(mac.Sum(nil))
	}
	return legacyHashKey(raw)
}

// legacyHashKey is the plain SHA-256 digest keys carried before a pepper.
func legacyHashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// VerifyKey checks whether the raw key matches the stored hash, accepting
// the peppered digest and, when a pepper is set, the digest a key was given
// before the pepper existed. Both comparisons are constant time and both
// run, so a legacy key costs the same as a current one.
func VerifyKey(raw, hash string) bool {
	current := subtle.ConstantTimeCompare([]byte(HashKey(raw)), []byte(hash)) == 1
	if pepper.Load() == nil {
		return current
	}
	legacy := subtle.ConstantTimeCompare([]byte(legacyHashKey(raw)), []byte(hash)) == 1
	return current || legacy
}

// NeedsRehash reports whether a stored hash that just verified raw was made
// without the pepper now in force, so the caller can store HashKey(raw) in
// its place.
func NeedsRehash(raw, hash string) bool {
	if pepper.Load() == nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(legacyHashKey(raw)), []byte(hash)) == 1
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
func GenerateKeyForEnvironment(envType environment.Type) (raw, hash, prefix string, err error) {
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
	case strings.HasPrefix(rawKey, keyMarkerTest), strings.HasPrefix(rawKey, pkMarkerTest):
		return environment.TypeDevelopment, true
	case strings.HasPrefix(rawKey, keyMarkerStg), strings.HasPrefix(rawKey, pkMarkerStg):
		return environment.TypeStaging, true
	case strings.HasPrefix(rawKey, keyMarkerLive), strings.HasPrefix(rawKey, pkMarkerLive):
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
	for _, marker := range []string{
		keyMarkerTest, keyMarkerStg, keyMarkerLive,
		pkMarkerTest, pkMarkerStg, pkMarkerLive,
		keyMarker,
	} {
		if strings.HasPrefix(rawKey, marker) {
			return rawKey[len(marker):]
		}
	}
	return rawKey
}

// PublicKeyMarker returns the public key prefix marker for a given environment type.
func PublicKeyMarker(envType environment.Type) string {
	switch envType {
	case environment.TypeDevelopment:
		return pkMarkerTest
	case environment.TypeStaging:
		return pkMarkerStg
	case environment.TypeProduction:
		return pkMarkerLive
	default:
		return pkMarkerLive
	}
}

// GenerateKeyPair creates a Clerk-style key pair: a publishable key (pk_*)
// for frontend use and a secret key (sk_*) for server-side auth.
// The secret key is hashed; the public key is stored in plaintext.
//
//nolint:gocritic // tooManyResultsChecker: key pair requires multiple components
func GenerateKeyPair() (publicKey, secretKey, secretHash, publicPrefix, secretPrefix string, err error) {
	return GenerateKeyPairForEnvironment(environment.TypeProduction)
}

// GenerateKeyPairForEnvironment creates an environment-aware key pair.
//
//nolint:gocritic // tooManyResultsChecker: key pair requires multiple components
func GenerateKeyPairForEnvironment(envType environment.Type) (publicKey, secretKey, secretHash, publicPrefix, secretPrefix string, err error) {
	// Generate secret key.
	skBytes := make([]byte, rawKeyLen)
	if _, err := rand.Read(skBytes); err != nil {
		return "", "", "", "", "", fmt.Errorf("apikey: generate secret key: %w", err)
	}
	skMarker := EnvironmentKeyMarker(envType)
	skHex := hex.EncodeToString(skBytes)
	secretKey = skMarker + skHex
	secretPrefix = secretKey[:prefixLen+len(skMarker)]
	secretHash = HashKey(secretKey)

	// Generate public key.
	pkBytes := make([]byte, rawKeyLen)
	if _, err := rand.Read(pkBytes); err != nil {
		return "", "", "", "", "", fmt.Errorf("apikey: generate public key: %w", err)
	}
	pkMarker := PublicKeyMarker(envType)
	pkHex := hex.EncodeToString(pkBytes)
	publicKey = pkMarker + pkHex
	publicPrefix = publicKey[:prefixLen+len(pkMarker)]

	return publicKey, secretKey, secretHash, publicPrefix, secretPrefix, nil
}

// IsPublicKey returns true if the key has a publishable key prefix (pk_*).
func IsPublicKey(key string) bool {
	return strings.HasPrefix(key, "pk_")
}

// IsSecretKey returns true if the key has a secret key prefix (sk_* or ask_).
func IsSecretKey(key string) bool {
	return strings.HasPrefix(key, "sk_") || strings.HasPrefix(key, keyMarker)
}

// IsAPIKey returns true if the string looks like an API key (pk_*, sk_*, or ask_).
func IsAPIKey(key string) bool {
	return IsPublicKey(key) || IsSecretKey(key)
}
