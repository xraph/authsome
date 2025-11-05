package repository

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// APIKeyRepository handles API key database operations
type APIKeyRepository struct {
	db *bun.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *bun.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create creates a new API key
func (r *APIKeyRepository) Create(ctx context.Context, apiKey *schema.APIKey) error {
	_, err := r.db.NewInsert().Model(apiKey).Exec(ctx)
	return err
}

// FindByID finds an API key by ID
func (r *APIKeyRepository) FindByID(ctx context.Context, id string) (*schema.APIKey, error) {
	apiKey := &schema.APIKey{}
	err := r.db.NewSelect().
		Model(apiKey).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

// FindByPrefix finds an API key by prefix
func (r *APIKeyRepository) FindByPrefix(ctx context.Context, prefix string) (*schema.APIKey, error) {
	apiKey := &schema.APIKey{}
	err := r.db.NewSelect().
		Model(apiKey).
		Where("prefix = ?", prefix).
		Where("deleted_at IS NULL").
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

// FindByUserID finds all API keys for a user
func (r *APIKeyRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*schema.APIKey, error) {
	var apiKeys []*schema.APIKey
	query := r.db.NewSelect().
		Model(&apiKeys).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return apiKeys, err
}

// FindByOrgID finds all API keys for an organization
func (r *APIKeyRepository) FindByOrgID(ctx context.Context, orgID string, limit, offset int) ([]*schema.APIKey, error) {
	var apiKeys []*schema.APIKey
	query := r.db.NewSelect().
		Model(&apiKeys).
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
	return apiKeys, err
}

// Update updates an API key
func (r *APIKeyRepository) Update(ctx context.Context, apiKey *schema.APIKey) error {
	apiKey.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(apiKey).
		Where("id = ?", apiKey.ID).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// UpdateUsage updates the usage statistics for an API key
func (r *APIKeyRepository) UpdateUsage(ctx context.Context, id string, ip, userAgent string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.APIKey)(nil)).
		Set("usage_count = usage_count + 1").
		Set("last_used_at = ?", now).
		Set("last_used_ip = ?", ip).
		Set("last_used_ua = ?", userAgent).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// Delete soft deletes an API key
func (r *APIKeyRepository) Delete(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.APIKey)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// Deactivate deactivates an API key without deleting it
func (r *APIKeyRepository) Deactivate(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.APIKey)(nil)).
		Set("active = ?", false).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// CountByUserID counts API keys for a user
func (r *APIKeyRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.APIKey)(nil)).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Count(ctx)
	return count, err
}

// CountByOrgID counts API keys for an organization
func (r *APIKeyRepository) CountByOrgID(ctx context.Context, orgID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.APIKey)(nil)).
		Where("org_id = ?", orgID).
		Where("deleted_at IS NULL").
		Count(ctx)
	return count, err
}

// CleanupExpired removes expired API keys
func (r *APIKeyRepository) CleanupExpired(ctx context.Context) (int, error) {
	now := time.Now()
	result, err := r.db.NewUpdate().
		Model((*schema.APIKey)(nil)).
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
