package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// SocialAccountRepository handles social account persistence.
type SocialAccountRepository interface {
	Create(ctx context.Context, account *schema.SocialAccount) error
	FindByID(ctx context.Context, id xid.ID) (*schema.SocialAccount, error)
	FindByUserAndProvider(ctx context.Context, userID xid.ID, provider string) (*schema.SocialAccount, error)
	FindByProviderAndProviderID(ctx context.Context, provider, providerID string, appID xid.ID, userOrganizationID *xid.ID) (*schema.SocialAccount, error)
	FindByUser(ctx context.Context, userID xid.ID) ([]*schema.SocialAccount, error)
	Update(ctx context.Context, account *schema.SocialAccount) error
	Delete(ctx context.Context, id xid.ID) error
	Unlink(ctx context.Context, userID xid.ID, provider string) error
}

type socialAccountRepository struct {
	db *bun.DB
}

func NewSocialAccountRepository(db *bun.DB) SocialAccountRepository {
	return &socialAccountRepository{db: db}
}

func (r *socialAccountRepository) Create(ctx context.Context, account *schema.SocialAccount) error {
	if account.ID.IsNil() {
		account.ID = xid.New()
	}

	_, err := r.db.NewInsert().Model(account).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create social account: %w", err)
	}

	return nil
}

func (r *socialAccountRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SocialAccount, error) {
	account := &schema.SocialAccount{}

	err := r.db.NewSelect().
		Model(account).
		Where("id = ?", id).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.NotFound("social account not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find social account: %w", err)
	}

	return account, nil
}

func (r *socialAccountRepository) FindByUserAndProvider(ctx context.Context, userID xid.ID, provider string) (*schema.SocialAccount, error) {
	account := &schema.SocialAccount{}

	err := r.db.NewSelect().
		Model(account).
		Where("user_id = ? AND provider = ? AND revoked = false", userID, provider).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // Not found is not an error
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find social account: %w", err)
	}

	return account, nil
}

func (r *socialAccountRepository) FindByProviderAndProviderID(ctx context.Context, provider, providerID string, appID xid.ID, userOrganizationID *xid.ID) (*schema.SocialAccount, error) {
	account := &schema.SocialAccount{}
	q := r.db.NewSelect().
		Model(account).
		Where("provider = ? AND provider_id = ? AND app_id = ? AND revoked = false", provider, providerID, appID)

	// Scope to org if provided
	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // Not found is not an error
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find social account: %w", err)
	}

	return account, nil
}

func (r *socialAccountRepository) FindByUser(ctx context.Context, userID xid.ID) ([]*schema.SocialAccount, error) {
	var accounts []*schema.SocialAccount

	err := r.db.NewSelect().
		Model(&accounts).
		Where("user_id = ? AND revoked = false", userID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find social accounts: %w", err)
	}

	return accounts, nil
}

func (r *socialAccountRepository) Update(ctx context.Context, account *schema.SocialAccount) error {
	_, err := r.db.NewUpdate().
		Model(account).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update social account: %w", err)
	}

	return nil
}

func (r *socialAccountRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SocialAccount)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete social account: %w", err)
	}

	return nil
}

func (r *socialAccountRepository) Unlink(ctx context.Context, userID xid.ID, provider string) error {
	_, err := r.db.NewUpdate().
		Model((*schema.SocialAccount)(nil)).
		Set("revoked = true, revoked_at = now()").
		Where("user_id = ? AND provider = ?", userID, provider).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unlink social account: %w", err)
	}

	return nil
}
