package session

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for session operations.
type Store interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, sessionID id.SessionID) (*Session, error)
	GetSessionByToken(ctx context.Context, token string) (*Session, error)
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)
	UpdateSession(ctx context.Context, s *Session) error
	DeleteSession(ctx context.Context, sessionID id.SessionID) error
	DeleteUserSessions(ctx context.Context, userID id.UserID) error
	ListUserSessions(ctx context.Context, userID id.UserID) ([]*Session, error)
}
