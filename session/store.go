package session

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
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
}
