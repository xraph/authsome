package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/pagination"
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

// CreateJWTKey creates a new JWT key
func (r *JWTKeyRepository) CreateJWTKey(ctx context.Context, key *schema.JWTKey) error {
	_, err := r.db.NewInsert().Model(key).Exec(ctx)
	return err
}

// FindJWTKeyByID finds a JWT key by ID
func (r *JWTKeyRepository) FindJWTKeyByID(ctx context.Context, id xid.ID) (*schema.JWTKey, error) {
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

// FindJWTKeyByKeyID finds a JWT key by key ID and app ID
func (r *JWTKeyRepository) FindJWTKeyByKeyID(ctx context.Context, keyID string, appID xid.ID) (*schema.JWTKey, error) {
	key := &schema.JWTKey{}
	err := r.db.NewSelect().
		Model(key).
		Where("key_id = ?", keyID).
		Where("app_id = ?", appID).
		Where("deleted_at IS NULL").
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// FindPlatformJWTKeyByKeyID finds a platform JWT key by key ID
func (r *JWTKeyRepository) FindPlatformJWTKeyByKeyID(ctx context.Context, keyID string) (*schema.JWTKey, error) {
	key := &schema.JWTKey{}
	err := r.db.NewSelect().
		Model(key).
		Where("key_id = ?", keyID).
		Where("is_platform_key = ?", true).
		Where("deleted_at IS NULL").
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// ListJWTKeys lists JWT keys with pagination and filtering
func (r *JWTKeyRepository) ListJWTKeys(ctx context.Context, filter *jwt.ListJWTKeysFilter) (*pagination.PageResponse[*schema.JWTKey], error) {
	var keys []*schema.JWTKey

	// Build base query with filters
	query := r.db.NewSelect().Model(&keys).Where("deleted_at IS NULL")

	// Apply app ID filter
	if !filter.AppID.IsNil() {
		query = query.Where("app_id = ?", filter.AppID)
	}

	// Apply platform key filter
	if filter.IsPlatformKey != nil {
		query = query.Where("is_platform_key = ?", *filter.IsPlatformKey)
	}

	// Apply active filter
	if filter.Active != nil {
		query = query.Where("active = ?", *filter.Active)
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().Model((*schema.JWTKey)(nil)).Where("deleted_at IS NULL")
	if !filter.AppID.IsNil() {
		countQuery = countQuery.Where("app_id = ?", filter.AppID)
	}
	if filter.IsPlatformKey != nil {
		countQuery = countQuery.Where("is_platform_key = ?", *filter.IsPlatformKey)
	}
	if filter.Active != nil {
		countQuery = countQuery.Where("active = ?", *filter.Active)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	offset := filter.GetOffset()
	limit := filter.GetLimit()
	query = query.Limit(limit).Offset(offset)

	// Apply ordering
	query = query.Order("created_at DESC")

	// Execute query
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(keys, int64(total), &filter.PaginationParams), nil
}

// ListPlatformJWTKeys lists platform JWT keys with pagination
func (r *JWTKeyRepository) ListPlatformJWTKeys(ctx context.Context, filter *jwt.ListJWTKeysFilter) (*pagination.PageResponse[*schema.JWTKey], error) {
	var keys []*schema.JWTKey

	// Build base query
	query := r.db.NewSelect().
		Model(&keys).
		Where("deleted_at IS NULL").
		Where("is_platform_key = ?", true)

	// Apply active filter
	if filter.Active != nil {
		query = query.Where("active = ?", *filter.Active)
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().
		Model((*schema.JWTKey)(nil)).
		Where("deleted_at IS NULL").
		Where("is_platform_key = ?", true)
	if filter.Active != nil {
		countQuery = countQuery.Where("active = ?", *filter.Active)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	offset := filter.GetOffset()
	limit := filter.GetLimit()
	query = query.Limit(limit).Offset(offset)

	// Apply ordering
	query = query.Order("created_at DESC")

	// Execute query
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(keys, int64(total), &filter.PaginationParams), nil
}

// UpdateJWTKey updates a JWT key
func (r *JWTKeyRepository) UpdateJWTKey(ctx context.Context, key *schema.JWTKey) error {
	key.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(key).
		Where("id = ?", key.ID).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// UpdateJWTKeyUsage updates the usage statistics for a JWT key
func (r *JWTKeyRepository) UpdateJWTKeyUsage(ctx context.Context, keyID string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.JWTKey)(nil)).
		Set("usage_count = usage_count + 1").
		Set("last_used_at = ?", now).
		Where("key_id = ?", keyID).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// DeactivateJWTKey deactivates a JWT key
func (r *JWTKeyRepository) DeactivateJWTKey(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.JWTKey)(nil)).
		Set("active = ?", false).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// DeleteJWTKey soft deletes a JWT key
func (r *JWTKeyRepository) DeleteJWTKey(ctx context.Context, id xid.ID) error {
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

// CleanupExpiredJWTKeys removes expired JWT keys
func (r *JWTKeyRepository) CleanupExpiredJWTKeys(ctx context.Context) (int64, error) {
	now := time.Now()
	result, err := r.db.NewDelete().
		Model((*schema.JWTKey)(nil)).
		Where("expires_at IS NOT NULL").
		Where("expires_at < ?", now).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return count, err
}

// CountJWTKeys counts JWT keys for an app
func (r *JWTKeyRepository) CountJWTKeys(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.JWTKey)(nil)).
		Where("app_id = ?", appID).
		Where("deleted_at IS NULL").
		Count(ctx)
	return count, err
}
