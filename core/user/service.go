package user

import (
    "context"
    "fmt"
    "time"
    "regexp"
    "strings"

    "github.com/rs/xid"
    "github.com/xraph/authsome/core/webhook"
    "github.com/xraph/authsome/internal/crypto"
    "github.com/xraph/authsome/internal/validator"
    "github.com/xraph/authsome/types"
)

// Service provides user-related operations
type Service struct {
    repo       Repository
    config     Config
    webhookSvc *webhook.Service
}

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

// NewService creates a new user service
func NewService(repo Repository, cfg Config, webhookSvc *webhook.Service) *Service {
    return &Service{
        repo:       repo,
        config:     cfg,
        webhookSvc: webhookSvc,
    }
}

// Create creates a new user
func (s *Service) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
    if !validator.ValidateEmail(req.Email) {
        return nil, types.NewValidationError("email", "invalid email")
    }
    ok, msg := validator.ValidatePassword(req.Password, s.config.PasswordRequirements)
    if !ok {
        return nil, types.NewValidationError("password", msg)
    }

    existing, err := s.repo.FindByEmail(ctx, req.Email)
    if err == nil && existing != nil {
        return nil, types.ErrEmailAlreadyExists
    }

    id := xid.New()

    hash, err := crypto.HashPassword(req.Password)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }

    now := time.Now().UTC()
    u := &User{
        ID:           id,
        Email:        req.Email,
        Name:         req.Name,
        PasswordHash: hash,
        Username:     id.String(),
        DisplayUsername: "",
        CreatedAt:    now,
        UpdatedAt:    now,
    }
    if err := s.repo.Create(ctx, u); err != nil {
        return nil, err
    }

    // Emit webhook event
    if s.webhookSvc != nil {
        data := map[string]interface{}{
            "user_id": u.ID.String(),
            "email":   u.Email,
            "name":    u.Name,
        }
        go s.webhookSvc.EmitEvent(ctx, "user.created", "default", data) // TODO: Get orgID from context
    }

    return u, nil
}

// FindByID finds a user by ID
func (s *Service) FindByID(ctx context.Context, id xid.ID) (*User, error) {
    return s.repo.FindByID(ctx, id)
}

// FindByEmail finds a user by email
func (s *Service) FindByEmail(ctx context.Context, email string) (*User, error) {
    return s.repo.FindByEmail(ctx, email)
}

// FindByUsername finds a user by username
func (s *Service) FindByUsername(ctx context.Context, username string) (*User, error) {
    return s.repo.FindByUsername(ctx, username)
}

// Update updates a user
func (s *Service) Update(ctx context.Context, u *User, req *UpdateUserRequest) (*User, error) {
    if req.Name != nil {
        u.Name = *req.Name
    }
    if req.Email != nil {
        // Check if email is changing and if new email is already taken
        newEmail := *req.Email
        if newEmail != u.Email {
            if existing, err := s.repo.FindByEmail(ctx, newEmail); err == nil && existing != nil && existing.ID != u.ID {
                return nil, fmt.Errorf("email already taken")
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
            return nil, fmt.Errorf("username cannot be empty")
        }
        // Normalize: lowercase canonical username
        canonical := strings.ToLower(raw)
        // Allowed characters: a-z, 0-9, underscore, dot; length 3-32
        re := regexp.MustCompile(`^[a-z0-9_\.]{3,32}$`)
        if !re.MatchString(canonical) {
            return nil, fmt.Errorf("invalid username: use 3-32 chars [a-z0-9_.]")
        }
        // Check uniqueness
        if existing, err := s.repo.FindByUsername(ctx, canonical); err == nil && existing != nil && existing.ID != u.ID {
            return nil, fmt.Errorf("username already taken")
        }
        u.Username = canonical
        // Set display username if provided; else preserve original casing
        if req.DisplayUsername != nil {
            disp := strings.TrimSpace(*req.DisplayUsername)
            if disp == "" { disp = raw }
            u.DisplayUsername = disp
        } else {
            u.DisplayUsername = raw
        }
    }
    u.UpdatedAt = time.Now().UTC()
    if err := s.repo.Update(ctx, u); err != nil {
        return nil, err
    }

    // Emit webhook event
    if s.webhookSvc != nil {
        data := map[string]interface{}{
            "user_id": u.ID.String(),
            "email":   u.Email,
            "name":    u.Name,
        }
        go s.webhookSvc.EmitEvent(ctx, "user.updated", "default", data) // TODO: Get orgID from context
    }

    return u, nil
}

// Delete deletes a user by ID
func (s *Service) Delete(ctx context.Context, id xid.ID) error {
    // Get user before deletion for webhook event
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return err
    }

    if err := s.repo.Delete(ctx, id); err != nil {
        return err
    }

    // Emit webhook event
    if s.webhookSvc != nil && user != nil {
        data := map[string]interface{}{
            "user_id": user.ID.String(),
            "email":   user.Email,
            "name":    user.Name,
        }
        go s.webhookSvc.EmitEvent(ctx, "user.deleted", "default", data) // TODO: Get orgID from context
    }

    return nil
}

// List lists users with pagination
func (s *Service) List(ctx context.Context, opts types.PaginationOptions) ([]*User, int, error) {
    if opts.Page <= 0 {
        opts.Page = 1
    }
    if opts.PageSize <= 0 {
        opts.PageSize = 20
    }
    offset := (opts.Page - 1) * opts.PageSize
    list, err := s.repo.List(ctx, opts.PageSize, offset)
    if err != nil {
        return nil, 0, err
    }
    total, err := s.repo.Count(ctx)
    if err != nil {
        return nil, 0, err
    }
    return list, total, nil
}

// Search searches users by name or email with pagination
func (s *Service) Search(ctx context.Context, query string, opts types.PaginationOptions) ([]*User, int, error) {
    if opts.Page <= 0 {
        opts.Page = 1
    }
    if opts.PageSize <= 0 {
        opts.PageSize = 20
    }
    offset := (opts.Page - 1) * opts.PageSize
    list, err := s.repo.Search(ctx, query, opts.PageSize, offset)
    if err != nil {
        return nil, 0, err
    }
    total, err := s.repo.CountSearch(ctx, query)
    if err != nil {
        return nil, 0, err
    }
    return list, total, nil
}

// CountCreatedToday returns the count of users created today
func (s *Service) CountCreatedToday(ctx context.Context) (int, error) {
    // Get start of today in UTC
    now := time.Now().UTC()
    startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
    return s.repo.CountCreatedSince(ctx, startOfDay)
}