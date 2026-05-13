package session

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
)

// RevokedRefreshToken records a previously-rotated or replay-flagged
// refresh token. The raw token is NEVER stored — only its SHA-256 hash.
type RevokedRefreshToken struct {
	TokenHash string             // hex-encoded SHA-256 of the original refresh token
	FamilyID  id.SessionFamilyID // family the original token belonged to
	RevokedAt time.Time
	Reason    string // "rotated" | "replay_detected" | "logout"
}

// Revocation reason constants for RevokedRefreshToken.Reason.
const (
	RevokeReasonRotated          = "rotated"
	RevokeReasonReplayDetected   = "replay_detected"
	RevokeReasonLogout           = "logout"
	RevokeReasonFamilyRevocation = "family_revocation"
)

// Store defines the persistence interface for session operations.
type Store interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, sessionID id.SessionID) (*Session, error)
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)
	UpdateSession(ctx context.Context, s *Session) error
	// TouchSession performs a lightweight update of last_activity_at, expires_at,
	// and updated_at without rewriting the entire session row.
	TouchSession(ctx context.Context, sessionID id.SessionID, lastActivityAt, expiresAt time.Time) error
	DeleteSession(ctx context.Context, sessionID id.SessionID) error
	DeleteUserSessions(ctx context.Context, userID id.UserID) error
	ListUserSessions(ctx context.Context, userID id.UserID) ([]*Session, error)
	// ListSessions returns the most recent sessions across all users, up to limit.
	ListSessions(ctx context.Context, limit int) ([]*Session, error)

	// IsRefreshTokenRevoked reports whether the given hash is in the revoked
	// set. Used to detect replay on every refresh exchange.
	IsRefreshTokenRevoked(ctx context.Context, tokenHash string) (bool, error)

	// MarkRefreshTokenRevoked persists a hash as revoked. Called on every
	// successful rotation (the OLD token's hash is recorded with
	// reason="rotated") and on replay detection (reason="replay_detected").
	// Idempotent — duplicate inserts succeed silently.
	MarkRefreshTokenRevoked(ctx context.Context, tokenHash string, familyID id.SessionFamilyID, reason string) error

	// GetRevokedRefreshTokenFamily looks up the family bound to a
	// previously-rotated token. Used to identify which sessions to
	// cascade-revoke when a replay is detected.
	GetRevokedRefreshTokenFamily(ctx context.Context, tokenHash string) (id.SessionFamilyID, error)

	// RevokeRefreshTokenFamily revokes every session sharing the given
	// family. Called on replay detection so a leaked token doesn't keep
	// working through siblings.
	RevokeRefreshTokenFamily(ctx context.Context, familyID id.SessionFamilyID, reason string) error
}
