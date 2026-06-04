package user

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for user operations.
type Store interface {
	CreateUser(ctx context.Context, u *User) error
	GetUser(ctx context.Context, userID id.UserID) (*User, error)
	GetUserByEmail(ctx context.Context, appID id.AppID, email string) (*User, error)
	GetUserByPhone(ctx context.Context, appID id.AppID, phone string) (*User, error)
	GetUserByUsername(ctx context.Context, appID id.AppID, username string) (*User, error)
	UpdateUser(ctx context.Context, u *User) error
	DeleteUser(ctx context.Context, userID id.UserID) error
	ListUsers(ctx context.Context, q *Query) (*List, error)

	// ── Multiple emails per account ──────────────────────────────
	// A user owns one or more emails (UserEmail). Exactly one is primary
	// and mirrors User.Email/User.EmailVerified. Addresses are unique per
	// (AppID, EnvID, Email) across all accounts.

	// CreateUserWithPrimaryEmail creates a user and its primary email row.
	// The address must be free within (AppID, EnvID), else ErrEmailTaken.
	CreateUserWithPrimaryEmail(ctx context.Context, u *User, primary *UserEmail) error

	// AddUserEmail attaches an additional email to a user. Returns
	// account.ErrEmailTaken if the address is already owned within the env.
	AddUserEmail(ctx context.Context, e *UserEmail) error

	// GetUserByAnyEmail resolves the user owning any (primary or secondary)
	// verified-or-not email matching (appID, envID, email). ErrNotFound when none.
	GetUserByAnyEmail(ctx context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*User, error)

	// GetUserEmailRecord returns the email row matching (appID, envID, email)
	// so callers can inspect Verified before linking. ErrNotFound when none.
	GetUserEmailRecord(ctx context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*UserEmail, error)

	// GetUserEmails returns all live emails for a user, primary first.
	GetUserEmails(ctx context.Context, userID id.UserID) ([]*UserEmail, error)

	// MarkUserEmailVerified marks an email verified, mirroring onto the user
	// record when it is the primary email.
	MarkUserEmailVerified(ctx context.Context, userID id.UserID, email string) error

	// SetPrimaryEmail makes a verified, user-owned email primary, updating the
	// user mirror. Returns account.ErrEmailNotVerified if the target is unverified.
	SetPrimaryEmail(ctx context.Context, userID id.UserID, email string) error

	// DeleteUserEmail soft-deletes a non-primary email. Refuses the primary
	// with store.ErrConflict.
	DeleteUserEmail(ctx context.Context, userID id.UserID, email string) error
}
