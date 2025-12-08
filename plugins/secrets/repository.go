package secrets

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/secrets/core"
	"github.com/xraph/authsome/plugins/secrets/schema"
)

// Repository defines the interface for secret storage operations
type Repository interface {
	// Secret CRUD operations
	Create(ctx context.Context, secret *schema.Secret) error
	FindByID(ctx context.Context, id xid.ID) (*schema.Secret, error)
	FindByPath(ctx context.Context, appID, envID xid.ID, path string) (*schema.Secret, error)
	List(ctx context.Context, appID, envID xid.ID, query *core.ListSecretsQuery) ([]*schema.Secret, int, error)
	Update(ctx context.Context, secret *schema.Secret) error
	Delete(ctx context.Context, id xid.ID) error
	HardDelete(ctx context.Context, id xid.ID) error

	// Version operations
	CreateVersion(ctx context.Context, version *schema.SecretVersion) error
	FindVersion(ctx context.Context, secretID xid.ID, version int) (*schema.SecretVersion, error)
	ListVersions(ctx context.Context, secretID xid.ID, page, pageSize int) ([]*schema.SecretVersion, int, error)
	DeleteOldVersions(ctx context.Context, secretID xid.ID, keepCount int) error

	// Access log operations
	LogAccess(ctx context.Context, log *schema.SecretAccessLog) error
	ListAccessLogs(ctx context.Context, secretID xid.ID, query *core.GetAccessLogsQuery) ([]*schema.SecretAccessLog, int, error)
	DeleteOldAccessLogs(ctx context.Context, olderThan time.Time) (int64, error)

	// Stats operations
	CountSecrets(ctx context.Context, appID, envID xid.ID) (int, error)
	CountVersions(ctx context.Context, appID, envID xid.ID) (int, error)
	GetSecretsByType(ctx context.Context, appID, envID xid.ID) (map[string]int, error)
	CountExpiringSecrets(ctx context.Context, appID, envID xid.ID, withinDays int) (int, error)
}

// bunRepository implements Repository using Bun ORM
type bunRepository struct {
	db *bun.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *bun.DB) Repository {
	return &bunRepository{db: db}
}

// =============================================================================
// Secret CRUD Operations
// =============================================================================

// Create creates a new secret
func (r *bunRepository) Create(ctx context.Context, secret *schema.Secret) error {
	_, err := r.db.NewInsert().
		Model(secret).
		Exec(ctx)
	return err
}

// FindByID finds a secret by ID
func (r *bunRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Secret, error) {
	secret := new(schema.Secret)
	err := r.db.NewSelect().
		Model(secret).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrSecretNotFound(id.String())
		}
		return nil, err
	}
	return secret, nil
}

// FindByPath finds a secret by app, environment, and path
func (r *bunRepository) FindByPath(ctx context.Context, appID, envID xid.ID, path string) (*schema.Secret, error) {
	secret := new(schema.Secret)
	err := r.db.NewSelect().
		Model(secret).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("path = ?", path).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrSecretNotFoundByPath(path)
		}
		return nil, err
	}
	return secret, nil
}

// List lists secrets with filtering and pagination
func (r *bunRepository) List(ctx context.Context, appID, envID xid.ID, query *core.ListSecretsQuery) ([]*schema.Secret, int, error) {
	if query == nil {
		query = &core.ListSecretsQuery{}
	}

	// Set defaults
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	// Build query
	q := r.db.NewSelect().
		Model((*schema.Secret)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL")

	// Apply filters
	if query.Prefix != "" {
		prefix := core.NormalizePath(query.Prefix)
		q = q.Where("path LIKE ?", prefix+"%")
	}

	if query.ValueType != "" {
		q = q.Where("value_type = ?", query.ValueType)
	}

	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			q = q.Where("? = ANY(tags)", tag)
		}
	}

	if query.Search != "" {
		search := "%" + strings.ToLower(query.Search) + "%"
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("LOWER(path) LIKE ?", search).
				WhereOr("LOWER(description) LIKE ?", search)
		})
	}

	// Count total
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply sorting
	switch query.SortBy {
	case "path":
		if query.SortOrder == "desc" {
			q = q.Order("path DESC")
		} else {
			q = q.Order("path ASC")
		}
	case "created_at":
		if query.SortOrder == "desc" {
			q = q.Order("created_at DESC")
		} else {
			q = q.Order("created_at ASC")
		}
	case "updated_at":
		if query.SortOrder == "desc" {
			q = q.Order("updated_at DESC")
		} else {
			q = q.Order("updated_at ASC")
		}
	default:
		q = q.Order("path ASC")
	}

	// Apply pagination
	offset := (query.Page - 1) * query.PageSize
	q = q.Limit(query.PageSize).Offset(offset)

	// Execute query
	var secrets []*schema.Secret
	err = q.Scan(ctx, &secrets)
	if err != nil {
		return nil, 0, err
	}

	return secrets, total, nil
}

// Update updates a secret
func (r *bunRepository) Update(ctx context.Context, secret *schema.Secret) error {
	secret.UpdatedAt = time.Now().UTC()
	_, err := r.db.NewUpdate().
		Model(secret).
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// Delete soft-deletes a secret
func (r *bunRepository) Delete(ctx context.Context, id xid.ID) error {
	now := time.Now().UTC()
	_, err := r.db.NewUpdate().
		Model((*schema.Secret)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// HardDelete permanently deletes a secret and its versions
func (r *bunRepository) HardDelete(ctx context.Context, id xid.ID) error {
	// Delete versions first
	_, err := r.db.NewDelete().
		Model((*schema.SecretVersion)(nil)).
		Where("secret_id = ?", id).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Delete the secret
	_, err = r.db.NewDelete().
		Model((*schema.Secret)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// =============================================================================
// Version Operations
// =============================================================================

// CreateVersion creates a new secret version
func (r *bunRepository) CreateVersion(ctx context.Context, version *schema.SecretVersion) error {
	_, err := r.db.NewInsert().
		Model(version).
		Exec(ctx)
	return err
}

// FindVersion finds a specific version of a secret
func (r *bunRepository) FindVersion(ctx context.Context, secretID xid.ID, version int) (*schema.SecretVersion, error) {
	secretVersion := new(schema.SecretVersion)
	err := r.db.NewSelect().
		Model(secretVersion).
		Where("secret_id = ?", secretID).
		Where("version = ?", version).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrVersionNotFound(secretID.String(), version)
		}
		return nil, err
	}
	return secretVersion, nil
}

// ListVersions lists versions for a secret with pagination
func (r *bunRepository) ListVersions(ctx context.Context, secretID xid.ID, page, pageSize int) ([]*schema.SecretVersion, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	// Count total
	total, err := r.db.NewSelect().
		Model((*schema.SecretVersion)(nil)).
		Where("secret_id = ?", secretID).
		Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Get versions
	offset := (page - 1) * pageSize
	var versions []*schema.SecretVersion
	err = r.db.NewSelect().
		Model(&versions).
		Where("secret_id = ?", secretID).
		Order("version DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

// DeleteOldVersions deletes old versions, keeping only the most recent N versions
func (r *bunRepository) DeleteOldVersions(ctx context.Context, secretID xid.ID, keepCount int) error {
	// Get versions to keep
	var keepVersions []int
	err := r.db.NewSelect().
		Model((*schema.SecretVersion)(nil)).
		Column("version").
		Where("secret_id = ?", secretID).
		Order("version DESC").
		Limit(keepCount).
		Scan(ctx, &keepVersions)
	if err != nil {
		return err
	}

	if len(keepVersions) == 0 {
		return nil
	}

	// Delete older versions
	_, err = r.db.NewDelete().
		Model((*schema.SecretVersion)(nil)).
		Where("secret_id = ?", secretID).
		Where("version NOT IN (?)", bun.In(keepVersions)).
		Exec(ctx)
	return err
}

// =============================================================================
// Access Log Operations
// =============================================================================

// LogAccess logs an access event
func (r *bunRepository) LogAccess(ctx context.Context, log *schema.SecretAccessLog) error {
	_, err := r.db.NewInsert().
		Model(log).
		Exec(ctx)
	return err
}

// ListAccessLogs lists access logs for a secret
func (r *bunRepository) ListAccessLogs(ctx context.Context, secretID xid.ID, query *core.GetAccessLogsQuery) ([]*schema.SecretAccessLog, int, error) {
	if query == nil {
		query = &core.GetAccessLogsQuery{}
	}
	if query.PageSize <= 0 {
		query.PageSize = 50
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	q := r.db.NewSelect().
		Model((*schema.SecretAccessLog)(nil)).
		Where("secret_id = ?", secretID)

	if query.Action != "" {
		q = q.Where("action = ?", query.Action)
	}
	if query.FromDate != nil {
		q = q.Where("created_at >= ?", *query.FromDate)
	}
	if query.ToDate != nil {
		q = q.Where("created_at <= ?", *query.ToDate)
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PageSize
	var logs []*schema.SecretAccessLog
	err = q.
		Order("created_at DESC").
		Limit(query.PageSize).
		Offset(offset).
		Scan(ctx, &logs)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// DeleteOldAccessLogs deletes access logs older than the specified time
func (r *bunRepository) DeleteOldAccessLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	res, err := r.db.NewDelete().
		Model((*schema.SecretAccessLog)(nil)).
		Where("created_at < ?", olderThan).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// =============================================================================
// Stats Operations
// =============================================================================

// CountSecrets counts total secrets for an app/environment
func (r *bunRepository) CountSecrets(ctx context.Context, appID, envID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.Secret)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Count(ctx)
}

// CountVersions counts total versions for an app/environment
func (r *bunRepository) CountVersions(ctx context.Context, appID, envID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.SecretVersion)(nil)).
		Join("JOIN secrets s ON s.id = sv.secret_id").
		Where("s.app_id = ?", appID).
		Where("s.environment_id = ?", envID).
		Where("s.deleted_at IS NULL").
		Count(ctx)
}

// GetSecretsByType returns count of secrets grouped by value type
func (r *bunRepository) GetSecretsByType(ctx context.Context, appID, envID xid.ID) (map[string]int, error) {
	var results []struct {
		ValueType string `bun:"value_type"`
		Count     int    `bun:"count"`
	}

	err := r.db.NewSelect().
		Model((*schema.Secret)(nil)).
		ColumnExpr("value_type, COUNT(*) as count").
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Group("value_type").
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range results {
		counts[r.ValueType] = r.Count
	}
	return counts, nil
}

// CountExpiringSecrets counts secrets expiring within the specified days
func (r *bunRepository) CountExpiringSecrets(ctx context.Context, appID, envID xid.ID, withinDays int) (int, error) {
	threshold := time.Now().AddDate(0, 0, withinDays)
	return r.db.NewSelect().
		Model((*schema.Secret)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Where("expires_at IS NOT NULL").
		Where("expires_at <= ?", threshold).
		Where("expires_at > ?", time.Now()).
		Count(ctx)
}
