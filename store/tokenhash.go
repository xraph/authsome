package store

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken returns the hex-encoded SHA-256 of a presented credential.
//
// Every token column in every backend stores this digest instead of the
// plaintext: session and refresh tokens, verification codes, password reset
// tokens, invitation tokens and OAuth2 codes. The tokens themselves carry at
// least 128 bits of entropy, so an unsalted digest is enough to make a copy
// of the database useless while keeping lookups a single indexed equality.
//
// The refresh-token revocation set already keys on this exact digest, which
// is why the two must never diverge: a session's RefreshTokenHash is the key
// the replay detector consults.
func HashToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}
