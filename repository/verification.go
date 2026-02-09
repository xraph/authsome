package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// verificationRepository implements verification token storage.
type verificationRepository struct {
	db *bun.DB
}

// NewVerificationRepository creates a new verification repository.
func NewVerificationRepository(db *bun.DB) *verificationRepository {
	return &verificationRepository{db: db}
}

// CreateVerification creates a new verification token.
func (r *verificationRepository) CreateVerification(ctx context.Context, verification *schema.Verification) error {
	_, err := r.db.NewInsert().
		Model(verification).
		Exec(ctx)

	return err
}

// FindVerificationByToken finds a verification by token.
func (r *verificationRepository) FindVerificationByToken(ctx context.Context, token string) (*schema.Verification, error) {
	var verification schema.Verification

	err := r.db.NewSelect().
		Model(&verification).
		Where("token = ?", token).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}

	return &verification, err
}

// MarkVerificationAsUsed marks a verification token as used.
func (r *verificationRepository) MarkVerificationAsUsed(ctx context.Context, id xid.ID) error {
	now := time.Now().UTC()
	_, err := r.db.NewUpdate().
		Model((*schema.Verification)(nil)).
		Set("used = ?", true).
		Set("used_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// DeleteExpiredVerifications deletes expired verification tokens.
func (r *verificationRepository) DeleteExpiredVerifications(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*schema.Verification)(nil)).
		Where("expires_at < ?", time.Now().UTC()).
		Exec(ctx)

	return err
}

// FindVerificationByUserAndType finds a verification by user and type.
func (r *verificationRepository) FindVerificationByUserAndType(ctx context.Context, userID xid.ID, verificationType string) (*schema.Verification, error) {
	var verification schema.Verification

	err := r.db.NewSelect().
		Model(&verification).
		Where("user_id = ?", userID).
		Where("type = ?", verificationType).
		Where("used = ?", false).
		Where("expires_at > ?", time.Now().UTC()).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}

	return &verification, err
}

// FindVerificationByCode finds a verification by 6-digit code and type.
func (r *verificationRepository) FindVerificationByCode(ctx context.Context, code string, verificationType string) (*schema.Verification, error) {
	var verification schema.Verification

	err := r.db.NewSelect().
		Model(&verification).
		Where("code = ?", code).
		Where("type = ?", verificationType).
		Where("used = ?", false).
		Where("expires_at > ?", time.Now().UTC()).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}

	return &verification, err
}
