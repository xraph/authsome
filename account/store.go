package account

import "context"

// Store defines the persistence interface for account lifecycle operations.
type Store interface {
	CreateVerification(ctx context.Context, v *Verification) error
	GetVerification(ctx context.Context, token string) (*Verification, error)
	ConsumeVerification(ctx context.Context, token string) error

	CreatePasswordReset(ctx context.Context, pr *PasswordReset) error
	GetPasswordReset(ctx context.Context, token string) (*PasswordReset, error)
	ConsumePasswordReset(ctx context.Context, token string) error
}
