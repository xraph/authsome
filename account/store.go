package account

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for account lifecycle operations.
type Store interface {
	CreateVerification(ctx context.Context, v *Verification) error
	GetVerification(ctx context.Context, token string) (*Verification, error)
	ConsumeVerification(ctx context.Context, token string) error

	// GetActiveEmailVerification returns the most recent unconsumed, unexpired
	// email verification for the user, or ErrNotFound. The OTP flow looks up by
	// user (codes are short and not globally unique) rather than by token.
	GetActiveEmailVerification(ctx context.Context, userID id.UserID) (*Verification, error)

	// UpdateVerification persists mutable fields of an existing verification
	// (Attempts, Consumed).
	UpdateVerification(ctx context.Context, v *Verification) error

	CreatePasswordReset(ctx context.Context, pr *PasswordReset) error
	GetPasswordReset(ctx context.Context, token string) (*PasswordReset, error)
	ConsumePasswordReset(ctx context.Context, token string) error
}
