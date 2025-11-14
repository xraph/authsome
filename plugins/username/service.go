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
	users  *user.Service
	auth   *auth.Service
	config Config
}

func NewService(users *user.Service, authSvc *auth.Service, config Config) *Service {
	return &Service{users: users, auth: authSvc, config: config}
}

// ValidatePassword validates password against configured requirements
func (s *Service) ValidatePassword(password string) error {
	if len(password) < s.config.MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", s.config.MinPasswordLength)
	}
	if len(password) > s.config.MaxPasswordLength {
		return fmt.Errorf("password must be at most %d characters", s.config.MaxPasswordLength)
	}

	if s.config.RequireUppercase {
		hasUpper := false
		for _, c := range password {
			if c >= 'A' && c <= 'Z' {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			return fmt.Errorf("password must contain at least one uppercase letter")
		}
	}

	if s.config.RequireLowercase {
		hasLower := false
		for _, c := range password {
			if c >= 'a' && c <= 'z' {
				hasLower = true
				break
			}
		}
		if !hasLower {
			return fmt.Errorf("password must contain at least one lowercase letter")
		}
	}

	if s.config.RequireNumber {
		hasNumber := false
		for _, c := range password {
			if c >= '0' && c <= '9' {
				hasNumber = true
				break
			}
		}
		if !hasNumber {
			return fmt.Errorf("password must contain at least one number")
		}
	}

	if s.config.RequireSpecialChar {
		hasSpecial := false
		specialChars := "!@#$%^&*()_+-=[]{}|;:',.<>?/~`"
		for _, c := range password {
			if strings.ContainsRune(specialChars, c) {
				hasSpecial = true
				break
			}
		}
		if !hasSpecial {
			return fmt.Errorf("password must contain at least one special character")
		}
	}

	return nil
}

// SignUpWithUsername is not supported due to email non-null constraint
func (s *Service) SignUpWithUsername(ctx context.Context, username, password string) error {
	// Normalize and validate inputs
	disp := strings.TrimSpace(username)
	if disp == "" || strings.TrimSpace(password) == "" {
		return fmt.Errorf("missing fields")
	}

	// Validate password against configured requirements
	if err := s.ValidatePassword(password); err != nil {
		return err
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
