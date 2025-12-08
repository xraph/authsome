package session

import (
	"context"

	"github.com/rs/xid"
)

// HookExecutor defines the interface for executing session-related hooks
// This interface allows the session service to execute hooks without importing the hooks package,
// avoiding circular dependencies (hooks package imports session for types)
type HookExecutor interface {
	ExecuteBeforeSessionCreate(ctx context.Context, req *CreateSessionRequest) error
	ExecuteAfterSessionCreate(ctx context.Context, session *Session) error
	ExecuteBeforeSessionRevoke(ctx context.Context, token string) error
	ExecuteAfterSessionRevoke(ctx context.Context, sessionID xid.ID) error
}
