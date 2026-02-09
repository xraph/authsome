package auth

import (
	"context"

	"github.com/xraph/authsome/core/responses"
)

// HookExecutor defines the interface for executing auth-related hooks
// This interface allows the auth service to execute hooks without importing the hooks package,
// avoiding circular dependencies (hooks package imports auth for request types).
type HookExecutor interface {
	ExecuteBeforeSignUp(ctx context.Context, req *SignUpRequest) error
	ExecuteAfterSignUp(ctx context.Context, response *responses.AuthResponse) error
	ExecuteBeforeSignIn(ctx context.Context, req *SignInRequest) error
	ExecuteAfterSignIn(ctx context.Context, response *responses.AuthResponse) error
	ExecuteBeforeSignOut(ctx context.Context, token string) error
	ExecuteAfterSignOut(ctx context.Context, token string) error
}
