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
	FindByID(ctx context.Context, id xid.ID) (*Session, error)
	ListSessions(ctx context.Context, filter *ListSessionsFilter) (*ListSessionsResponse, error)
	Revoke(ctx context.Context, token string) error
	RevokeByID(ctx context.Context, id xid.ID) error
	
	// Sliding session renewal (Option 1)
	TouchSession(ctx context.Context, sess *Session) (*Session, bool, error)
	
	// Refresh token pattern (Option 3)
	RefreshSession(ctx context.Context, refreshToken string) (*RefreshResponse, error)
}

// Ensure Service implements ServiceInterface
var _ ServiceInterface = (*Service)(nil)
