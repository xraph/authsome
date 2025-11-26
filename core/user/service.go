package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/validator"
)

// =============================================================================
// SERVICE CONFIGURATION
// =============================================================================

// Config represents user service configuration
type Config struct {
	PasswordRequirements validator.PasswordRequirements

	ChangeEmail struct {
		Enabled             bool
		RequireVerification bool
	}

	DeleteAccount struct {
		RequirePassword     bool
		RequireVerification bool
	}
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

// Service provides user-related operations
type Service struct {
	repo         Repository
	config       Config
	webhookSvc   *webhook.Service
	hookExecutor HookExecutor
}

// NewService creates a new user service
func NewService(repo Repository, cfg Config, webhookSvc *webhook.Service, hookExecutor HookExecutor) *Service {
	return &Service{
		repo:         repo,
		config:       cfg,
		webhookSvc:   webhookSvc,
		hookExecutor: hookExecutor,
	}
}

// =============================================================================
// SERVICE METHODS
// =============================================================================

// Create creates a new user in the specified app
func (s *Service) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
	// Execute before user create hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeUserCreate(ctx, req); err != nil {
			return nil, err
		}
	}

	// Validate email
	if !validator.ValidateEmail(req.Email) {
		return nil, InvalidEmail(req.Email)
	}

	// Validate password
	ok, msg := validator.ValidatePassword(req.Password, s.config.PasswordRequirements)
	if !ok {
		return nil, WeakPassword(msg)
	}

	// Check if email already exists in this app
	existing, err := s.repo.FindByAppAndEmail(ctx, req.AppID, req.Email)
	if err == nil && existing != nil {
		return nil, EmailAlreadyExists(req.Email)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, UserCreationFailed(err)
	}

	// Generate ID and hash password
	id := xid.New()
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, UserCreationFailed(fmt.Errorf("failed to hash password: %w", err))
	}

	now := time.Now().UTC()

	// Create DTO
	user := &User{
		ID:              id,
		AppID:           req.AppID,
		Email:           req.Email,
		Name:            req.Name,
		PasswordHash:    hash,
		Username:        id.String(),
		DisplayUsername: "",
		EmailVerified:   false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Convert to schema and save
	if err := s.repo.Create(ctx, user.ToSchema()); err != nil {
		return nil, UserCreationFailed(err)
	}

	// Execute after user create hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterUserCreate(ctx, user); err != nil {
			// Log error but don't fail - user is already created
		}
	}

	// Emit webhook event
	if s.webhookSvc != nil {
		data := map[string]interface{}{
			"user_id": user.ID.String(),
			"app_id":  user.AppID.String(),
			"email":   user.Email,
			"name":    user.Name,
		}
		go s.webhookSvc.EmitEvent(ctx, req.AppID, xid.NilID(), "user.created", data)
	}

	return user, nil
}

// FindByID finds a user by ID
func (s *Service) FindByID(ctx context.Context, id xid.ID) (*User, error) {
	schemaUser, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, UserNotFound(id.String())
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return FromSchemaUser(schemaUser), nil
}

// FindByEmail finds a user by email (global search, not app-scoped)
func (s *Service) FindByEmail(ctx context.Context, email string) (*User, error) {
	schemaUser, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, UserNotFound(email)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return FromSchemaUser(schemaUser), nil
}

// FindByAppAndEmail finds a user by app ID and email (app-scoped search)
func (s *Service) FindByAppAndEmail(ctx context.Context, appID xid.ID, email string) (*User, error) {
	schemaUser, err := s.repo.FindByAppAndEmail(ctx, appID, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, UserNotFound(email)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return FromSchemaUser(schemaUser), nil
}

// FindByUsername finds a user by username
func (s *Service) FindByUsername(ctx context.Context, username string) (*User, error) {
	schemaUser, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, UserNotFound(username)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return FromSchemaUser(schemaUser), nil
}

// Update updates a user
func (s *Service) Update(ctx context.Context, u *User, req *UpdateUserRequest) (*User, error) {
	// Execute before user update hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeUserUpdate(ctx, u.ID, req); err != nil {
			return nil, err
		}
	}

	if req.Name != nil {
		u.Name = *req.Name
	}

	if req.Email != nil {
		// Check if email is changing and if new email is already taken in this app
		newEmail := *req.Email
		if newEmail != u.Email {
			if !validator.ValidateEmail(newEmail) {
				return nil, InvalidEmail(newEmail)
			}
			existing, err := s.repo.FindByAppAndEmail(ctx, u.AppID, newEmail)
			if err == nil && existing != nil && existing.ID != u.ID {
				return nil, EmailTaken(newEmail)
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, UserUpdateFailed(err)
			}
			u.Email = newEmail
		}
	}

	if req.EmailVerified != nil {
		u.EmailVerified = *req.EmailVerified
		if *req.EmailVerified {
			now := time.Now().UTC()
			u.EmailVerifiedAt = &now
		}
	}

	if req.Image != nil {
		u.Image = *req.Image
	}

	// Handle username update with validation and uniqueness check
	if req.Username != nil {
		raw := strings.TrimSpace(*req.Username)
		if raw == "" {
			return nil, InvalidUsername("", "username cannot be empty")
		}
		// Normalize: lowercase canonical username
		canonical := strings.ToLower(raw)
		// Allowed characters: a-z, 0-9, underscore, dot; length 3-32
		re := regexp.MustCompile(`^[a-z0-9_\.]{3,32}$`)
		if !re.MatchString(canonical) {
			return nil, InvalidUsername(canonical, "use 3-32 chars [a-z0-9_.]")
		}
		// Check uniqueness
		existing, err := s.repo.FindByUsername(ctx, canonical)
		if err == nil && existing != nil && existing.ID != u.ID {
			return nil, UsernameTaken(canonical)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, UserUpdateFailed(err)
		}
		u.Username = canonical
		// Set display username if provided; else preserve original casing
		if req.DisplayUsername != nil {
			disp := strings.TrimSpace(*req.DisplayUsername)
			if disp == "" {
				disp = raw
			}
			u.DisplayUsername = disp
		} else {
			u.DisplayUsername = raw
		}
	}

	u.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, u.ToSchema()); err != nil {
		return nil, UserUpdateFailed(err)
	}

	// Execute after user update hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterUserUpdate(ctx, u); err != nil {
			// Log error but don't fail - user is already updated
		}
	}

	// Emit webhook event
	if s.webhookSvc != nil {
		data := map[string]interface{}{
			"user_id": u.ID.String(),
			"app_id":  u.AppID.String(),
			"email":   u.Email,
			"name":    u.Name,
		}
		go s.webhookSvc.EmitEvent(ctx, u.AppID, xid.NilID(), "user.updated", data)
	}

	return u, nil
}

// Delete deletes a user by ID
func (s *Service) Delete(ctx context.Context, id xid.ID) error {
	// Execute before user delete hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteBeforeUserDelete(ctx, id); err != nil {
			return err
		}
	}

	// Get user before deletion for webhook event
	schemaUser, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserNotFound(id.String())
		}
		return UserDeletionFailed(err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return UserDeletionFailed(err)
	}

	// Execute after user delete hooks
	if s.hookExecutor != nil {
		if err := s.hookExecutor.ExecuteAfterUserDelete(ctx, id); err != nil {
			// Log error but don't fail - user is already deleted
		}
	}

	// Emit webhook event
	if s.webhookSvc != nil && schemaUser != nil {
		user := FromSchemaUser(schemaUser)
		data := map[string]interface{}{
			"user_id": user.ID.String(),
			"app_id":  user.AppID.String(),
			"email":   user.Email,
			"name":    user.Name,
		}
		go s.webhookSvc.EmitEvent(ctx, user.AppID, xid.NilID(), "user.deleted", data)
	}

	return nil
}

// ListUsers lists users with pagination and filtering
func (s *Service) ListUsers(ctx context.Context, filter *ListUsersFilter) (*pagination.PageResponse[*User], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, InvalidUserData("pagination", err.Error())
	}

	// Get paginated results from repository
	pageResp, err := s.repo.ListUsers(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert schema users to DTOs
	dtoUsers := FromSchemaUsers(pageResp.Data)

	// Return paginated response with DTOs
	return &pagination.PageResponse[*User]{
		Data:       dtoUsers,
		Pagination: pageResp.Pagination,
		Cursor:     pageResp.Cursor,
	}, nil
}

// CountUsers counts users with filtering
func (s *Service) CountUsers(ctx context.Context, filter *CountUsersFilter) (int, error) {
	count, err := s.repo.CountUsers(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}
