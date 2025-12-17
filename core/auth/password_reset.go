package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/schema"
)

// PasswordResetRepository defines verification token operations
type PasswordResetRepository interface {
	CreateVerification(ctx context.Context, verification *schema.Verification) error
	FindVerificationByToken(ctx context.Context, token string) (*schema.Verification, error)
	MarkVerificationAsUsed(ctx context.Context, id xid.ID) error
	DeleteExpiredVerifications(ctx context.Context) error
}

// RequestPasswordResetRequest represents a password reset request
type RequestPasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents a password reset confirmation
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

// RequestPasswordReset initiates a password reset flow
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return "", contexts.ErrAppContextRequired
	}

	// Find user by email
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists - security best practice
		// Still return success but don't send email
		return "", nil
	}

	// Generate reset token
	token, err := crypto.GenerateToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Create verification record
	verification := &schema.Verification{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    user.ID,
		Token:     token,
		Type:      "password_reset",
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour), // 1 hour expiry
		Used:      false,
		AuditableModel: schema.AuditableModel{
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}

	// Store verification token
	if repo, ok := s.getPasswordResetRepo(); ok {
		if err := repo.CreateVerification(ctx, verification); err != nil {
			return "", fmt.Errorf("failed to create verification: %w", err)
		}
	}

	return token, nil
}

// ResetPassword completes the password reset flow
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return contexts.ErrAppContextRequired
	}

	// Find verification token
	repo, ok := s.getPasswordResetRepo()
	if !ok {
		return fmt.Errorf("password reset repository not available")
	}

	verification, err := repo.FindVerificationByToken(ctx, token)
	if err != nil {
		return ErrInvalidResetToken
	}

	// Validate token
	if verification.Used {
		return ErrResetTokenAlreadyUsed
	}

	if verification.Type != "password_reset" {
		return ErrInvalidResetToken
	}

	if verification.ExpiresAt.Before(time.Now().UTC()) {
		return ErrResetTokenExpired
	}

	if verification.AppID != appID {
		return ErrInvalidResetToken
	}

	// Get user
	user, err := s.users.FindByID(ctx, verification.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Hash new password
	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user password
	if err := s.users.UpdatePassword(ctx, user.ID, hashedPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	if err := repo.MarkVerificationAsUsed(ctx, verification.ID); err != nil {
		// Log error but don't fail - password is already updated
		fmt.Printf("failed to mark verification as used: %v\n", err)
	}

	// TODO: Optionally revoke all existing sessions for security
	// s.session.RevokeAllForUser(ctx, user.ID)

	return nil
}

// ValidateResetToken checks if a reset token is valid
func (s *Service) ValidateResetToken(ctx context.Context, token string) (bool, error) {
	repo, ok := s.getPasswordResetRepo()
	if !ok {
		return false, fmt.Errorf("password reset repository not available")
	}

	verification, err := repo.FindVerificationByToken(ctx, token)
	if err != nil {
		return false, nil
	}

	// Check if valid
	if verification.Used || verification.Type != "password_reset" || verification.ExpiresAt.Before(time.Now().UTC()) {
		return false, nil
	}

	return true, nil
}

// getPasswordResetRepo attempts to get the password reset repository
func (s *Service) getPasswordResetRepo() (PasswordResetRepository, bool) {
	if s.users == nil {
		return nil, false
	}

	// Try to cast users service to get verification repo
	if repoProvider, ok := s.users.(interface {
		GetVerificationRepo() PasswordResetRepository
	}); ok {
		return repoProvider.GetVerificationRepo(), true
	}

	return nil, false
}

// Password reset specific errors
var (
	ErrInvalidResetToken     = fmt.Errorf("invalid reset token")
	ErrResetTokenExpired     = fmt.Errorf("reset token has expired")
	ErrResetTokenAlreadyUsed = fmt.Errorf("reset token has already been used")
)
