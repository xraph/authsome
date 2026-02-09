package anonymous

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/session"
	coreuser "github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
)

// Service provides anonymous sign-in by creating a guest user and session.
type Service struct {
	users  *repo.UserRepository
	sess   *session.Service
	config Config
}

func NewService(users *repo.UserRepository, sess *session.Service, config Config) *Service {
	return &Service{users: users, sess: sess, config: config}
}

// SignInGuest creates a guest user and returns a session token.
func (s *Service) SignInGuest(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, ip, ua string) (*session.Session, error) {
	// Validate app context
	if appID.IsNil() {
		return nil, errs.New("APP_CONTEXT_REQUIRED", "App context is required", 400)
	}

	id := xid.New()
	email := fmt.Sprintf("guest-%s@guest.local", id.String())
	now := time.Now().UTC()

	// Create guest user (need to populate appID if required by schema)
	// Note: Anonymous users may not have a proper AppID initially
	// but we'll use the provided appID for proper multi-tenancy support
	u := &coreuser.User{
		ID:        id,
		AppID:     appID,
		Email:     email,
		Name:      "Guest",
		Username:  "guest_" + id.String(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.users.Create(ctx, u.ToSchema()); err != nil {
		return nil, errs.Wrap(err, "USER_CREATION_FAILED", "Failed to create guest user", 500)
	}

	// Create session with app/environment context
	sess, err := s.sess.Create(ctx, &session.CreateSessionRequest{
		AppID:          appID,
		EnvironmentID:  &envID,
		OrganizationID: orgID,
		UserID:         id,
		Remember:       false,
		IPAddress:      ip,
		UserAgent:      ua,
	})
	if err != nil {
		return nil, errs.Wrap(err, "SESSION_CREATION_FAILED", "Failed to create session", 500)
	}

	return sess, nil
}

// LinkGuest upgrades an anonymous guest account to a real account.
func (s *Service) LinkGuest(ctx context.Context, token, email, password, name string) (*coreuser.User, error) {
	// Validate input
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)

	if email == "" {
		return nil, errs.New("EMAIL_REQUIRED", "Email is required", 400)
	}

	if password == "" {
		return nil, errs.New("PASSWORD_REQUIRED", "Password is required", 400)
	}

	// Resolve session and current user
	sess, err := s.sess.FindByToken(ctx, token)
	if err != nil {
		return nil, errs.SessionInvalid()
	}

	if sess == nil {
		return nil, errs.SessionNotFound()
	}

	schemaUser, err := s.users.FindByID(ctx, sess.UserID)
	if err != nil || schemaUser == nil {
		return nil, errs.UserNotFound()
	}

	// Convert to DTO for easier manipulation
	u := coreuser.FromSchemaUser(schemaUser)

	// Verify it's a guest user
	if !strings.HasSuffix(strings.ToLower(u.Email), "@guest.local") {
		return nil, errs.New("NOT_ANONYMOUS_ACCOUNT", "Account is not anonymous", 400)
	}

	// Check email uniqueness
	if existing, err := s.users.FindByEmail(ctx, email); err == nil && existing != nil {
		return nil, errs.EmailAlreadyExists(email)
	}

	// Hash password and update fields
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, errs.Wrap(err, "PASSWORD_HASH_FAILED", "Failed to hash password", 500)
	}

	u.Email = email

	u.PasswordHash = hash
	if strings.TrimSpace(name) != "" {
		u.Name = name
	}

	u.UpdatedAt = time.Now().UTC()

	// Convert back to schema and update
	if err := s.users.Update(ctx, u.ToSchema()); err != nil {
		return nil, errs.Wrap(err, "USER_UPDATE_FAILED", "Failed to update user", 500)
	}

	return u, nil
}

// GetUserByID is a helper to get a user by ID (returns DTO).
func (s *Service) GetUserByID(ctx context.Context, id xid.ID) (*coreuser.User, error) {
	schemaUser, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return coreuser.FromSchemaUser(schemaUser), nil
}
