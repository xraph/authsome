package auth

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/types"
)

// Service provides authentication operations
type Service struct {
	users        user.ServiceInterface
	session      session.ServiceInterface
	config       Config
	hookExecutor HookExecutor
}

// NewService creates a new auth service
func NewService(users user.ServiceInterface, session session.ServiceInterface, cfg Config, hookExecutor HookExecutor) *Service {
	return &Service{users: users, session: session, config: cfg, hookExecutor: hookExecutor}
}

// SignUp registers a new user and returns a session
func (s *Service) SignUp(ctx context.Context, req *SignUpRequest) (*responses.AuthResponse, error) {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return nil, contexts.ErrAppContextRequired
	}

	// Execute before sign up hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeSignUp(ctx, req); err != nil {
			return nil, err
		}
	}

	// ensure user does not exist
	existing, err := s.users.FindByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, types.ErrEmailAlreadyExists
	}
	userReq := &user.CreateUserRequest{
		AppID:    appID,
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	}
	u, err := s.users.Create(ctx, userReq)
	if err != nil {
		return nil, err
	}

	// if verification is required, do not create session
	if s.config.RequireEmailVerification {
		response := &responses.AuthResponse{User: u}
		// Execute after sign up hooks
		if s.hookExecutor != nil {
			if err := s.hookExecutor.ExecuteAfterSignUp(ctx, response); err != nil {
				// Log error but don't fail the operation - user is already created
				// TODO: Add proper logging
			}
		}
		return response, nil
	}

	sess, err := s.session.Create(ctx, &session.CreateSessionRequest{
		AppID:     appID,
		UserID:    u.ID,
		Remember:  req.RememberMe,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	})
	if err != nil {
		return nil, err
	}

	response := &responses.AuthResponse{User: u, Session: sess, Token: sess.Token}

	// Execute after sign up hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterSignUp(ctx, response); err != nil {
			// Log error but don't fail the operation - user is already signed up
			// TODO: Add proper logging
		}
	}

	return response, nil
}

// SignIn authenticates a user and returns a session
func (s *Service) SignIn(ctx context.Context, req *SignInRequest) (*responses.AuthResponse, error) {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return nil, contexts.ErrAppContextRequired
	}

	// Execute before sign in hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeSignIn(ctx, req); err != nil {
			return nil, err
		}
	}

	u, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil || u == nil {
		return nil, types.ErrInvalidCredentials
	}
	if ok := crypto.CheckPassword(req.Password, u.PasswordHash); !ok {
		return nil, types.ErrInvalidCredentials
	}

	// Check email verification if required
	if s.config.RequireEmailVerification && !u.EmailVerified {
		return nil, types.ErrEmailNotVerified
	}

	sess, err := s.session.Create(ctx, &session.CreateSessionRequest{
		AppID:     appID,
		UserID:    u.ID,
		Remember:  req.RememberMe,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	})
	if err != nil {
		return nil, err
	}

	response := &responses.AuthResponse{User: u, Session: sess, Token: sess.Token}

	// Execute after sign in hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterSignIn(ctx, response); err != nil {
			// Log error but don't fail the operation - user is already signed in
			// TODO: Add proper logging
		}
	}

	return response, nil
}

// CheckCredentials validates a user's credentials and returns the user without creating a session
func (s *Service) CheckCredentials(ctx context.Context, email, password string) (*user.User, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil || u == nil {
		return nil, types.ErrInvalidCredentials
	}
	if ok := crypto.CheckPassword(password, u.PasswordHash); !ok {
		return nil, types.ErrInvalidCredentials
	}
	return u, nil
}

// CreateSessionForUser creates a session for a given user and returns auth response
// This is typically used after credentials are already validated (e.g., after 2FA verification)
func (s *Service) CreateSessionForUser(ctx context.Context, u *user.User, remember bool, ip, ua string) (*responses.AuthResponse, error) {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return nil, contexts.ErrAppContextRequired
	}

	sess, err := s.session.Create(ctx, &session.CreateSessionRequest{
		AppID:     appID,
		UserID:    u.ID,
		Remember:  remember,
		IPAddress: ip,
		UserAgent: ua,
	})
	if err != nil {
		return nil, err
	}

	response := &responses.AuthResponse{User: u, Session: sess, Token: sess.Token}

	// Execute after sign in hooks
	// Note: BeforeSignIn hooks are not executed here because credentials were already validated
	// This method is used in flows where validation happens separately (e.g., 2FA, magic link)
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterSignIn(ctx, response); err != nil {
			// Log error but don't fail the operation - user is already signed in
			// TODO: Add proper logging
		}
	}

	return response, nil
}

// SignOut revokes a session
func (s *Service) SignOut(ctx context.Context, req *SignOutRequest) error {
	// Execute before sign out hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeSignOut(ctx, req.Token); err != nil {
			return err
		}
	}

	err := s.session.Revoke(ctx, req.Token)
	if err != nil {
		return err
	}

	// Execute after sign out hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterSignOut(ctx, req.Token); err != nil {
			// Log error but don't fail the operation - session is already revoked
			// TODO: Add proper logging
		}
	}

	return nil
}

// GetSession validates and returns session details
func (s *Service) GetSession(ctx context.Context, token string) (*responses.AuthResponse, error) {
	sess, err := s.session.FindByToken(ctx, token)
	if err != nil || sess == nil {
		return nil, types.ErrSessionNotFound
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, types.ErrSessionExpired
	}
	u, err := s.users.FindByID(ctx, sess.UserID)
	if err != nil || u == nil {
		return nil, types.ErrUserNotFound
	}
	return &responses.AuthResponse{User: u, Session: sess, Token: sess.Token}, nil
}

// UpdateUser updates the current user's fields via user service
func (s *Service) UpdateUser(ctx context.Context, userID xid.ID, req *user.UpdateUserRequest) (*user.User, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil || u == nil {
		return nil, types.ErrUserNotFound
	}
	return s.users.Update(ctx, u, req)
}

// RefreshSession refreshes an access token using a refresh token
func (s *Service) RefreshSession(ctx context.Context, refreshToken string) (*responses.RefreshSessionResponse, error) {
	// Delegate to session service
	refreshResp, err := s.session.RefreshSession(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Load user
	u, err := s.users.FindByID(ctx, refreshResp.Session.UserID)
	if err != nil || u == nil {
		return nil, types.ErrUserNotFound
	}

	// Execute after sign in hooks (session was refreshed, similar to signing in)
	if s.hookExecutor != nil {
		authResp := &responses.AuthResponse{
			User:    u,
			Session: refreshResp.Session,
			Token:   refreshResp.AccessToken,
		}
		_ = s.hookExecutor.ExecuteAfterSignIn(ctx, authResp)
	}

	return &responses.RefreshSessionResponse{
		User:             u,
		Session:          refreshResp.Session,
		AccessToken:      refreshResp.AccessToken,
		RefreshToken:     refreshResp.RefreshToken,
		ExpiresAt:        refreshResp.ExpiresAt.Format(time.RFC3339),
		RefreshExpiresAt: refreshResp.RefreshExpiresAt.Format(time.RFC3339),
	}, nil
}
