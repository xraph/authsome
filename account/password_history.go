package account

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
)

// PasswordHistoryEntry records a previously used password hash.
type PasswordHistoryEntry struct {
	ID        string    `json:"id"`
	UserID    id.UserID `json:"user_id"`
	Hash      string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// PasswordHistoryStore persists previous password hashes so they cannot
// be reused. Implementations should store only the hash, never plaintext.
type PasswordHistoryStore interface {
	// SavePasswordHash records a new password hash for the user.
	SavePasswordHash(ctx context.Context, userID id.UserID, hash string) error

	// GetPasswordHistory returns the most recent N password hashes.
	GetPasswordHistory(ctx context.Context, userID id.UserID, limit int) ([]PasswordHistoryEntry, error)
}
