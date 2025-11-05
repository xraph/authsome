package session

import (
	"context"
	"github.com/rs/xid"
)

// Repository defines session persistence operations
type Repository interface {
	Create(ctx context.Context, s *Session) error
	FindByToken(ctx context.Context, token string) (*Session, error)
	Revoke(ctx context.Context, token string) error
	// Multi-session operations
	FindByID(ctx context.Context, id xid.ID) (*Session, error)
	ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*Session, error)
	ListAll(ctx context.Context, limit, offset int) ([]*Session, error)
	RevokeByID(ctx context.Context, id xid.ID) error
}
