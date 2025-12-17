package auth

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/user"
)

// ServiceInterface defines the contract for auth service operations
// This allows plugins to decorate the service with additional behavior
type ServiceInterface interface {
	SignUp(ctx context.Context, req *SignUpRequest) (*responses.AuthResponse, error)
	SignIn(ctx context.Context, req *SignInRequest) (*responses.AuthResponse, error)
	SignOut(ctx context.Context, req *SignOutRequest) error
	CheckCredentials(ctx context.Context, email, password string) (*user.User, error)
	CreateSessionForUser(ctx context.Context, u *user.User, remember bool, ipAddress, userAgent string) (*responses.AuthResponse, error)
	GetSession(ctx context.Context, token string) (*responses.AuthResponse, error)
	UpdateUser(ctx context.Context, id xid.ID, req *user.UpdateUserRequest) (*user.User, error)
	RefreshSession(ctx context.Context, refreshToken string) (*responses.RefreshSessionResponse, error)

	// Password management
	RequestPasswordReset(ctx context.Context, email string) (string, error)
	ResetPassword(ctx context.Context, token, newPassword string) error
	ValidateResetToken(ctx context.Context, token string) (bool, error)
	ChangePassword(ctx context.Context, userID xid.ID, oldPassword, newPassword string) error

	// Email change
	RequestEmailChange(ctx context.Context, userID xid.ID, newEmail string) (string, error)
	ConfirmEmailChange(ctx context.Context, token string) error
	ValidateEmailChangeToken(ctx context.Context, token string) (bool, error)
}

// Ensure Service implements ServiceInterface
var _ ServiceInterface = (*Service)(nil)
