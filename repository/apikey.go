package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// APIKeyRepository handles API key database operations
// Updated for V2 architecture: App → Environment → Organization.
type APIKeyRepository struct {
	db *bun.DB
}

// NewAPIKeyRepository creates a new API key repository.
func NewAPIKeyRepository(db *bun.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// CreateAPIKey creates a new API key.
func (r *APIKeyRepository) CreateAPIKey(ctx context.Context, apiKey *schema.APIKey) error {
	_, err := r.db.NewInsert().Model(apiKey).Exec(ctx)

	return err
}

// FindAPIKeyByID finds an API key by ID.
func (r *APIKeyRepository) FindAPIKeyByID(ctx context.Context, id xid.ID) (*schema.APIKey, error) {
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

// FindAPIKeyByPrefix finds an API key by prefix.
func (r *APIKeyRepository) FindAPIKeyByPrefix(ctx context.Context, prefix string) (*schema.APIKey, error) {
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

// ListAPIKeys lists API keys with filtering and pagination.
func (r *APIKeyRepository) ListAPIKeys(ctx context.Context, filter *apikey.ListAPIKeysFilter) (*pagination.PageResponse[*schema.APIKey], error) {
	var keys []*schema.APIKey

	// Build query with filters
	query := r.db.NewSelect().Model(&keys).Where("deleted_at IS NULL")
	query = query.Where("app_id = ?", filter.AppID)

	if filter.EnvironmentID != nil {
		query = query.Where("environment_id = ?", *filter.EnvironmentID)
	}

	if filter.OrganizationID != nil {
		query = query.Where("organization_id = ?", *filter.OrganizationID)
	}

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	if filter.Active != nil {
		query = query.Where("active = ?", *filter.Active)
	}

	// Count query with same filters
	countQuery := r.db.NewSelect().Model((*schema.APIKey)(nil)).Where("deleted_at IS NULL")
	countQuery = countQuery.Where("app_id = ?", filter.AppID)

	if filter.EnvironmentID != nil {
		countQuery = countQuery.Where("environment_id = ?", *filter.EnvironmentID)
	}

	if filter.OrganizationID != nil {
		countQuery = countQuery.Where("organization_id = ?", *filter.OrganizationID)
	}

	if filter.UserID != nil {
		countQuery = countQuery.Where("user_id = ?", *filter.UserID)
	}

	if filter.Active != nil {
		countQuery = countQuery.Where("active = ?", *filter.Active)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination and ordering
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())
	query = query.Order("created_at DESC")

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(keys, int64(total), &filter.PaginationParams), nil
}

// UpdateAPIKey updates an API key.
func (r *APIKeyRepository) UpdateAPIKey(ctx context.Context, apiKey *schema.APIKey) error {
	apiKey.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(apiKey).
		Where("id = ?", apiKey.ID).
		Where("deleted_at IS NULL").
		Exec(ctx)

	return err
}

// UpdateAPIKeyUsage updates the usage statistics for an API key.
func (r *APIKeyRepository) UpdateAPIKeyUsage(ctx context.Context, id xid.ID, ip, userAgent string) error {
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

// DeleteAPIKey soft deletes an API key.
func (r *APIKeyRepository) DeleteAPIKey(ctx context.Context, id xid.ID) error {
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

// DeactivateAPIKey deactivates an API key without deleting it.
func (r *APIKeyRepository) DeactivateAPIKey(ctx context.Context, id xid.ID) error {
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

// CountAPIKeys counts API keys with flexible filtering.
func (r *APIKeyRepository) CountAPIKeys(ctx context.Context, appID xid.ID, envID *xid.ID, orgID *xid.ID, userID *xid.ID) (int, error) {
	query := r.db.NewSelect().
		Model((*schema.APIKey)(nil)).
		Where("app_id = ?", appID).
		Where("deleted_at IS NULL")

	if envID != nil {
		query = query.Where("environment_id = ?", *envID)
	}

	if orgID != nil {
		query = query.Where("organization_id = ?", *orgID)
	}

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	count, err := query.Count(ctx)

	return count, err
}

// CleanupExpiredAPIKeys removes expired API keys.
func (r *APIKeyRepository) CleanupExpiredAPIKeys(ctx context.Context) (int, error) {
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
