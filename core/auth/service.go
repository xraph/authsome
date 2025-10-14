package auth

import (
    "context"
    "time"

    "github.com/rs/xid"
    "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/internal/crypto"
    "github.com/xraph/authsome/types"
)

// Service provides authentication operations
type Service struct {
    users   *user.Service
    session *session.Service
    config  Config
}

// NewService creates a new auth service
func NewService(users *user.Service, session *session.Service, cfg Config) *Service {
	return &Service{users: users, session: session, config: cfg}
}

// SignUp registers a new user and returns a session
func (s *Service) SignUp(ctx context.Context, req *SignUpRequest) (*AuthResponse, error) {
	// ensure user does not exist
	existing, err := s.users.FindByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, types.ErrEmailAlreadyExists
	}
	userReq := &user.CreateUserRequest{
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
		return &AuthResponse{User: u}, nil
	}

	sess, err := s.session.Create(ctx, &session.CreateSessionRequest{
		UserID:    u.ID,
		Remember:  req.Remember,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	})
	if err != nil {
		return nil, err
	}
	return &AuthResponse{User: u, Session: sess, Token: sess.Token}, nil
}

// SignIn authenticates a user and returns a session
func (s *Service) SignIn(ctx context.Context, req *SignInRequest) (*AuthResponse, error) {
    u, err := s.users.FindByEmail(ctx, req.Email)
    if err != nil || u == nil {
        return nil, types.ErrInvalidCredentials
    }
    if ok := crypto.CheckPassword(req.Password, u.PasswordHash); !ok {
        return nil, types.ErrInvalidCredentials
    }
    sess, err := s.session.Create(ctx, &session.CreateSessionRequest{
        UserID:    u.ID,
        Remember:  req.Remember || req.RememberMe,
        IPAddress: req.IPAddress,
        UserAgent: req.UserAgent,
    })
    if err != nil {
        return nil, err
    }
    return &AuthResponse{User: u, Session: sess, Token: sess.Token}, nil
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
func (s *Service) CreateSessionForUser(ctx context.Context, u *user.User, remember bool, ip, ua string) (*AuthResponse, error) {
    sess, err := s.session.Create(ctx, &session.CreateSessionRequest{
        UserID:    u.ID,
        Remember:  remember,
        IPAddress: ip,
        UserAgent: ua,
    })
    if err != nil {
        return nil, err
    }
    return &AuthResponse{User: u, Session: sess, Token: sess.Token}, nil
}

// SignOut revokes a session
func (s *Service) SignOut(ctx context.Context, req *SignOutRequest) error {
	return s.session.Revoke(ctx, req.Token)
}

// GetSession validates and returns session details
func (s *Service) GetSession(ctx context.Context, token string) (*AuthResponse, error) {
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
	return &AuthResponse{User: u, Session: sess, Token: sess.Token}, nil
}

// UpdateUser updates the current user's fields via user service
func (s *Service) UpdateUser(ctx context.Context, userID xid.ID, req *user.UpdateUserRequest) (*user.User, error) {
    u, err := s.users.FindByID(ctx, userID)
    if err != nil || u == nil {
        return nil, types.ErrUserNotFound
    }
    return s.users.Update(ctx, u, req)
}
