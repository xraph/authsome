package auth

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/types"
)

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

// ChangePassword changes a user's password after verifying the old password
func (s *Service) ChangePassword(ctx context.Context, userID xid.ID, oldPassword, newPassword string) error {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return contexts.ErrAppContextRequired
	}

	// Get user
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify old password
	if !crypto.CheckPassword(oldPassword, user.PasswordHash) {
		return types.ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.users.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Execute password changed hook
	if hookRegistry := s.getHookRegistry(); hookRegistry != nil {
		if registry, ok := hookRegistry.(interface {
			ExecuteOnPasswordChanged(context.Context, xid.ID) error
		}); ok {
			_ = registry.ExecuteOnPasswordChanged(ctx, userID)
		}
	}

	return nil
}

// getHookRegistry retrieves the hook registry if available
func (s *Service) getHookRegistry() interface{} {
	// Try to get from users service if it implements the interface
	if userSvc, ok := s.users.(interface {
		GetHookRegistry() interface{}
	}); ok {
		return userSvc.GetHookRegistry()
	}
	return nil
}
