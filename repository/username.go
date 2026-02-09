package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/schema"
)

// UsernameRepository handles username plugin-specific database operations.
type UsernameRepository struct {
	db *bun.DB
}

// NewUsernameRepository creates a new username repository.
func NewUsernameRepository(db *bun.DB) *UsernameRepository {
	return &UsernameRepository{db: db}
}

// =============================================================================
// FAILED LOGIN ATTEMPT METHODS
// =============================================================================

// RecordFailedAttempt records a failed login attempt.
func (r *UsernameRepository) RecordFailedAttempt(ctx context.Context, username string, appID xid.ID, ip, ua string) error {
	attempt := &schema.FailedLoginAttempt{
		ID:        xid.New(),
		Username:  username,
		AppID:     appID,
		IP:        ip,
		UserAgent: ua,
		AttemptAt: time.Now(),
	}
	_, err := r.db.NewInsert().Model(attempt).Exec(ctx)

	return err
}

// GetFailedAttempts returns the number of failed attempts for a username within a time window.
func (r *UsernameRepository) GetFailedAttempts(ctx context.Context, username string, appID xid.ID, since time.Time) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.FailedLoginAttempt)(nil)).
		Where("username = ?", username).
		Where("app_id = ?", appID).
		Where("attempt_at >= ?", since).
		Count(ctx)

	return count, err
}

// ClearFailedAttempts removes all failed attempts for a username.
func (r *UsernameRepository) ClearFailedAttempts(ctx context.Context, username string, appID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.FailedLoginAttempt)(nil)).
		Where("username = ?", username).
		Where("app_id = ?", appID).
		Exec(ctx)

	return err
}

// CleanupOldFailedAttempts removes failed attempts older than the specified duration.
func (r *UsernameRepository) CleanupOldFailedAttempts(ctx context.Context, before time.Time) error {
	_, err := r.db.NewDelete().
		Model((*schema.FailedLoginAttempt)(nil)).
		Where("attempt_at < ?", before).
		Exec(ctx)

	return err
}

// =============================================================================
// ACCOUNT LOCKOUT METHODS
// =============================================================================

// LockAccount locks a user account for a specified duration.
func (r *UsernameRepository) LockAccount(ctx context.Context, userID xid.ID, duration time.Duration, reason string) error {
	lockout := &schema.AccountLockout{
		ID:          xid.New(),
		UserID:      userID,
		LockedUntil: time.Now().Add(duration),
		Reason:      reason,
		CreatedAt:   time.Now(),
	}
	_, err := r.db.NewInsert().Model(lockout).Exec(ctx)

	return err
}

// IsAccountLocked checks if a user account is currently locked
// Returns true if locked, the locked_until time, and any error.
func (r *UsernameRepository) IsAccountLocked(ctx context.Context, userID xid.ID) (bool, *time.Time, error) {
	var lockout schema.AccountLockout

	err := r.db.NewSelect().
		Model(&lockout).
		Where("user_id = ?", userID).
		Where("locked_until > ?", time.Now()).
		Order("locked_until DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false, nil, nil
		}

		return false, nil, err
	}

	return true, &lockout.LockedUntil, nil
}

// UnlockAccount removes all active lockouts for a user.
func (r *UsernameRepository) UnlockAccount(ctx context.Context, userID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.AccountLockout)(nil)).
		Where("user_id = ?", userID).
		Where("locked_until > ?", time.Now()).
		Exec(ctx)

	return err
}

// CleanupExpiredLockouts removes expired lockouts.
func (r *UsernameRepository) CleanupExpiredLockouts(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*schema.AccountLockout)(nil)).
		Where("locked_until <= ?", time.Now()).
		Exec(ctx)

	return err
}

// =============================================================================
// PASSWORD HISTORY METHODS
// =============================================================================

// SavePasswordHistory saves a password hash to history.
func (r *UsernameRepository) SavePasswordHistory(ctx context.Context, userID xid.ID, passwordHash string) error {
	history := &schema.PasswordHistory{
		ID:           xid.New(),
		UserID:       userID,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	_, err := r.db.NewInsert().Model(history).Exec(ctx)

	return err
}

// GetPasswordHistory retrieves the most recent password hashes for a user.
func (r *UsernameRepository) GetPasswordHistory(ctx context.Context, userID xid.ID, limit int) ([]string, error) {
	var histories []schema.PasswordHistory

	err := r.db.NewSelect().
		Model(&histories).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	hashes := make([]string, len(histories))
	for i, h := range histories {
		hashes[i] = h.PasswordHash
	}

	return hashes, nil
}

// CheckPasswordInHistory checks if a password matches any in the user's history.
func (r *UsernameRepository) CheckPasswordInHistory(ctx context.Context, userID xid.ID, password string, limit int) (bool, error) {
	hashes, err := r.GetPasswordHistory(ctx, userID, limit)
	if err != nil {
		return false, err
	}

	for _, hash := range hashes {
		if crypto.CheckPassword(password, hash) {
			return true, nil
		}
	}

	return false, nil
}

// CleanupOldPasswordHistory removes old password history entries beyond the limit.
func (r *UsernameRepository) CleanupOldPasswordHistory(ctx context.Context, userID xid.ID, keepCount int) error {
	// Get the creation time of the Nth most recent password
	var cutoffTime time.Time

	err := r.db.NewSelect().
		Model((*schema.PasswordHistory)(nil)).
		Column("created_at").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(keepCount).
		Limit(1).
		Scan(ctx, &cutoffTime)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			// Less than keepCount entries, nothing to delete
			return nil
		}

		return err
	}

	// Delete all entries older than the cutoff
	_, err = r.db.NewDelete().
		Model((*schema.PasswordHistory)(nil)).
		Where("user_id = ?", userID).
		Where("created_at < ?", cutoffTime).
		Exec(ctx)

	return err
}
