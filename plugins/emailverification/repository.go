package emailverification

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// VerificationRepository handles database operations for email verification tokens.
type VerificationRepository struct {
	db *bun.DB
}

// NewVerificationRepository creates a new verification repository.
func NewVerificationRepository(db *bun.DB) *VerificationRepository {
	return &VerificationRepository{db: db}
}

// Create creates a new email verification record.
func (r *VerificationRepository) Create(ctx context.Context, appID, userID xid.ID, token string, expiresAt time.Time) error {
	verification := &schema.Verification{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    userID,
		Token:     token,
		Type:      "email",
		ExpiresAt: expiresAt,
		Used:      false,
	}

	_, err := r.db.NewInsert().
		Model(verification).
		Exec(ctx)

	return err
}

// FindByToken finds a verification record by token.
func (r *VerificationRepository) FindByToken(ctx context.Context, token string) (*schema.Verification, error) {
	verification := new(schema.Verification)

	err := r.db.NewSelect().
		Model(verification).
		Where("token = ?", token).
		Where("type = ?", "email").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return verification, nil
}

// FindByUserID finds the latest email verification for a user.
func (r *VerificationRepository) FindByUserID(ctx context.Context, userID xid.ID) (*schema.Verification, error) {
	verification := new(schema.Verification)

	err := r.db.NewSelect().
		Model(verification).
		Where("user_id = ?", userID).
		Where("type = ?", "email").
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return verification, nil
}

// MarkAsUsed marks a verification token as used.
func (r *VerificationRepository) MarkAsUsed(ctx context.Context, verificationID xid.ID) error {
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*schema.Verification)(nil)).
		Set("used = ?", true).
		Set("used_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", verificationID).
		Exec(ctx)

	return err
}

// DeleteExpired deletes expired verification tokens.
func (r *VerificationRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*schema.Verification)(nil)).
		Where("type = ?", "email").
		Where("expires_at < ?", before).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

// CountRecentByUser counts recent verification requests by a user (for rate limiting).
func (r *VerificationRepository) CountRecentByUser(ctx context.Context, userID xid.ID, since time.Time) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Verification)(nil)).
		Where("user_id = ?", userID).
		Where("type = ?", "email").
		Where("created_at >= ?", since).
		Count(ctx)

	return count, err
}

// InvalidateOldTokens marks all unused tokens for a user as used (when sending new verification).
func (r *VerificationRepository) InvalidateOldTokens(ctx context.Context, userID xid.ID) error {
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*schema.Verification)(nil)).
		Set("used = ?", true).
		Set("used_at = ?", now).
		Set("updated_at = ?", now).
		Where("user_id = ?", userID).
		Where("type = ?", "email").
		Where("used = ?", false).
		Exec(ctx)

	return err
}
