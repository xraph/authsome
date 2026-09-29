package store

import (
	"context"
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

// LegacyTokenHasher converts credential rows written before token hashing.
//
// Lookups already upgrade a legacy row the first time it is presented, but a
// row nobody presents again would keep its plaintext forever. The engine
// sweeps these at start so the window in which a database copy still holds
// live credentials closes on its own.
type LegacyTokenHasher interface {
	// HashLegacyTokens rewrites up to batch rows, across sessions,
	// verifications, password resets and invitations, that still hold a
	// plaintext credential with no hash. It reports how many rows it
	// converted; zero means the sweep is complete.
	HashLegacyTokens(ctx context.Context, batch int) (int64, error)
}
