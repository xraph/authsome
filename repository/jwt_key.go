package repository

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// JWTKeyRepository handles JWT key database operations
type JWTKeyRepository struct {
	db *bun.DB
}

// NewJWTKeyRepository creates a new JWT key repository
func NewJWTKeyRepository(db *bun.DB) *JWTKeyRepository {
	return &JWTKeyRepository{db: db}
}

// Create creates a new JWT key
func (r *JWTKeyRepository) Create(ctx context.Context, key *schema.JWTKey) error {
	_, err := r.db.NewInsert().Model(key).Exec(ctx)
	return err
}

// FindByID finds a JWT key by ID
func (r *JWTKeyRepository) FindByID(ctx context.Context, id string) (*schema.JWTKey, error) {
	key := &schema.JWTKey{}
	err := r.db.NewSelect().
		Model(key).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// FindByKeyID finds a JWT key by key ID (kid)
func (r *JWTKeyRepository) FindByKeyID(ctx context.Context, keyID string) (*schema.JWTKey, error) {
	key := &schema.JWTKey{}
	err := r.db.NewSelect().
		Model(key).
		Where("key_id = ?", keyID).
		Where("deleted_at IS NULL").
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// FindActiveByOrgID finds all active JWT keys for an organization
func (r *JWTKeyRepository) FindActiveByOrgID(ctx context.Context, orgID string) ([]*schema.JWTKey, error) {
	var keys []*schema.JWTKey
	err := r.db.NewSelect().
		Model(&keys).
		Where("org_id = ?", orgID).
		Where("deleted_at IS NULL").
		Where("active = ?", true).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Order("created_at DESC").
		Scan(ctx)
	return keys, err
}

// FindByOrgID finds all JWT keys for an organization
func (r *JWTKeyRepository) FindByOrgID(ctx context.Context, orgID string, limit, offset int) ([]*schema.JWTKey, error) {
	var keys []*schema.JWTKey
	query := r.db.NewSelect().
		Model(&keys).
		Where("org_id = ?", orgID).
		Where("deleted_at IS NULL").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return keys, err
}

// Update updates a JWT key
func (r *JWTKeyRepository) Update(ctx context.Context, key *schema.JWTKey) error {
	key.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(key).
		Where("id = ?", key.ID).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// UpdateUsage updates the usage statistics for a JWT key
func (r *JWTKeyRepository) UpdateUsage(ctx context.Context, keyID string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.JWTKey)(nil)).
		Set("usage_count = usage_count + 1").
		Set("last_used_at = ?", now).
		Set("updated_at = ?", now).
		Where("key_id = ?", keyID).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// Deactivate deactivates a JWT key
func (r *JWTKeyRepository) Deactivate(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.JWTKey)(nil)).
		Set("active = ?", false).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// Delete soft deletes a JWT key
func (r *JWTKeyRepository) Delete(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.JWTKey)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// CleanupExpired removes expired JWT keys
func (r *JWTKeyRepository) CleanupExpired(ctx context.Context) (int, error) {
	now := time.Now()
	result, err := r.db.NewUpdate().
		Model((*schema.JWTKey)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("expires_at IS NOT NULL").
		Where("expires_at < ?", now).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	
	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}

// CountByOrgID counts JWT keys for an organization
func (r *JWTKeyRepository) CountByOrgID(ctx context.Context, orgID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.JWTKey)(nil)).
		Where("org_id = ?", orgID).
		Where("deleted_at IS NULL").
		Count(ctx)
	return count, err
}