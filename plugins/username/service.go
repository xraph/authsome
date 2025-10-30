package username

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
)

// Service provides username-based auth operations backed by core services
type Service struct {
	users *user.Service
	auth  *auth.Service
}

func NewService(users *user.Service, authSvc *auth.Service) *Service {
	return &Service{users: users, auth: authSvc}
}

// SignUpWithUsername is not supported due to email non-null constraint
func (s *Service) SignUpWithUsername(ctx context.Context, username, password string) error {
	// Normalize and validate inputs
	disp := strings.TrimSpace(username)
	if disp == "" || strings.TrimSpace(password) == "" {
		return fmt.Errorf("missing fields")
	}
	canonical := strings.ToLower(disp)
	// Generate a temporary, unique email to satisfy non-null/unique constraints
	id := xid.New()
	tempEmail := fmt.Sprintf("u-%s@temp.local", id.String())
	// Create user via core user service to reuse password hashing and validations
	u, err := s.users.Create(ctx, &user.CreateUserRequest{Email: tempEmail, Password: password, Name: disp})
	if err != nil {
		return err
	}
	// Update canonical and display username
	req := &user.UpdateUserRequest{Username: &canonical, DisplayUsername: &disp}
	if _, err := s.users.Update(ctx, u, req); err != nil {
		return err
	}
	return nil
}

// SignInWithUsername authenticates by username and password
func (s *Service) SignInWithUsername(ctx context.Context, username, password string) (*auth.AuthResponse, error) {
	un := strings.ToLower(strings.TrimSpace(username))
	if un == "" || password == "" {
		return nil, fmt.Errorf("missing fields")
	}
	u, err := s.users.FindByUsername(ctx, un)
	if err != nil || u == nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if ok := crypto.CheckPassword(password, u.PasswordHash); !ok {
		return nil, fmt.Errorf("invalid credentials")
	}
	return s.auth.CreateSessionForUser(ctx, u, false, "", "username-plugin")
}
