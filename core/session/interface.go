package session

import (
	"context"
)

// ServiceInterface defines the contract for session service operations
// This allows plugins to decorate the service with additional behavior
type ServiceInterface interface {
	Create(ctx context.Context, req *CreateSessionRequest) (*Session, error)
	FindByToken(ctx context.Context, token string) (*Session, error)
	Revoke(ctx context.Context, token string) error
}

// Ensure Service implements ServiceInterface
var _ ServiceInterface = (*Service)(nil)

