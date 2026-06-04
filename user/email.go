package user

import (
	"strings"
	"time"

	"github.com/xraph/authsome/id"
)

// UserEmail is a single email address belonging to a user. A user may own
// multiple emails; exactly one is marked primary and mirrors User.Email /
// User.EmailVerified. Email uniqueness is enforced per (AppID, EnvID, Email)
// so an address can belong to at most one account within an environment.
type UserEmail struct { //nolint:revive // exported name stutter is intentional: User.Email is the field, UserEmail is the row type
	ID        id.UserEmailID   `json:"id"`
	UserID    id.UserID        `json:"user_id"`
	AppID     id.AppID         `json:"app_id"`
	EnvID     id.EnvironmentID `json:"env_id"`
	Email     string           `json:"email"` // always stored lowercased + trimmed
	Verified  bool             `json:"verified"`
	IsPrimary bool             `json:"is_primary"`
	Source    string           `json:"source"` // e.g. "password", "social:github", "sso", "scim", "admin", "backfill"
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt *time.Time       `json:"deleted_at,omitempty"`
}

// NormalizeEmail lowercases and trims an email address so lookups and
// uniqueness checks are case-insensitive and whitespace-insensitive.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// NewPrimaryEmail builds the primary UserEmail row mirroring a user's current
// Email/EmailVerified, for use with Store.CreateUserWithPrimaryEmail. source
// records where the address came from (e.g. "password", "sso", "scim").
func NewPrimaryEmail(u *User, source string) *UserEmail {
	return &UserEmail{
		ID:        id.NewUserEmailID(),
		UserID:    u.ID,
		AppID:     u.AppID,
		EnvID:     u.EnvID,
		Email:     NormalizeEmail(u.Email),
		Verified:  u.EmailVerified,
		IsPrimary: true,
		Source:    source,
	}
}
