package session

import (
	"context"

	"github.com/rs/xid"
)

// ServiceInterface defines the contract for session service operations
// This allows plugins to decorate the service with additional behavior
type ServiceInterface interface {
	Create(ctx context.Context, req *CreateSessionRequest) (*Session, error)
	FindByToken(ctx context.Context, token string) (*Session, error)
	Revoke(ctx context.Context, token string) error
	ListAll(ctx context.Context, limit, offset int) ([]*Session, error)
	ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*Session, error)
	RevokeByID(ctx context.Context, id xid.ID) error
}

// Ensure Service implements ServiceInterface
var _ ServiceInterface = (*Service)(nil)
