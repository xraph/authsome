package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// ContentEntryRepository defines the interface for content entry storage operations.
type ContentEntryRepository interface {
	// CRUD operations
	Create(ctx context.Context, entry *schema.ContentEntry) error
	FindByID(ctx context.Context, id xid.ID) (*schema.ContentEntry, error)
	FindByIDWithType(ctx context.Context, id xid.ID) (*schema.ContentEntry, error)
	List(ctx context.Context, contentTypeID xid.ID, query *EntryListQuery) ([]*schema.ContentEntry, int, error)
	Update(ctx context.Context, entry *schema.ContentEntry) error
	Delete(ctx context.Context, id xid.ID) error
	HardDelete(ctx context.Context, id xid.ID) error

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, ids []xid.ID, status string) error
	BulkDelete(ctx context.Context, ids []xid.ID) error

	// Query operations
	FindByFieldValue(ctx context.Context, contentTypeID xid.ID, field string, value any) ([]*schema.ContentEntry, error)
	ExistsWithFieldValue(ctx context.Context, contentTypeID xid.ID, field string, value any, excludeID *xid.ID) (bool, error)

	// Scheduled entries
	FindScheduledForPublish(ctx context.Context, before time.Time) ([]*schema.ContentEntry, error)

	// Stats operations
	Count(ctx context.Context, contentTypeID xid.ID) (int, error)
	CountByStatus(ctx context.Context, contentTypeID xid.ID) (map[string]int, error)
	CountByAppEnv(ctx context.Context, appID, envID xid.ID) (int, error)
	CountByAppEnvAndStatus(ctx context.Context, appID, envID xid.ID) (map[string]int, error)
}

// EntryListQuery defines query parameters for listing entries.
type EntryListQuery struct {
	Status      string
	Search      string
	Filters     map[string]FilterCondition
	SortBy      string
	SortOrder   string
	Page        int
	PageSize    int
	Select      []string
	IncludeType bool
}

// FilterCondition represents a filter condition.
type FilterCondition struct {
	Operator string
	Value    any
}

// contentEntryRepository implements ContentEntryRepository using Bun ORM.
type contentEntryRepository struct {
	db *bun.DB
}

// NewContentEntryRepository creates a new content entry repository instance.
func NewContentEntryRepository(db *bun.DB) ContentEntryRepository {
	return &contentEntryRepository{db: db}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new content entry.
func (r *contentEntryRepository) Create(ctx context.Context, entry *schema.ContentEntry) error {
	if entry.ID.IsNil() {
		entry.ID = xid.New()
	}

	now := time.Now()
	entry.CreatedAt = now

	entry.UpdatedAt = now
	if entry.Status == "" {
		entry.Status = "draft"
	}

	if entry.Version == 0 {
		entry.Version = 1
	}

	if entry.Data == nil {
		entry.Data = make(schema.EntryData)
	}

	_, err := r.db.NewInsert().
		Model(entry).
		Exec(ctx)

	return err
}

// FindByID finds a content entry by ID.
func (r *contentEntryRepository) FindByID(ctx context.Context, id xid.ID) (*schema.ContentEntry, error) {
	entry := new(schema.ContentEntry)

	err := r.db.NewSelect().
		Model(entry).
		Where("ce.id = ?", id).
		Where("ce.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrEntryNotFound(id.String())
		}

		return nil, err
	}

	return entry, nil
}

// FindByIDWithType finds a content entry by ID with its content type loaded.
func (r *contentEntryRepository) FindByIDWithType(ctx context.Context, id xid.ID) (*schema.ContentEntry, error) {
	entry := new(schema.ContentEntry)

	err := r.db.NewSelect().
		Model(entry).
		Relation("ContentType").
		Relation("ContentType.Fields", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("\"order\" ASC")
		}).
		Where("ce.id = ?", id).
		Where("ce.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrEntryNotFound(id.String())
		}

		return nil, err
	}

	return entry, nil
}

// List lists content entries with filtering and pagination.
func (r *contentEntryRepository) List(ctx context.Context, contentTypeID xid.ID, query *EntryListQuery) ([]*schema.ContentEntry, int, error) {
	if query == nil {
		query = &EntryListQuery{}
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
		Model((*schema.ContentEntry)(nil)).
		Where("ce.content_type_id = ?", contentTypeID).
		Where("ce.deleted_at IS NULL")

	// Filter by status
	if query.Status != "" {
		q = q.Where("ce.status = ?", query.Status)
	}

	// Apply JSONB filters
	for field, cond := range query.Filters {
		q = r.applyJSONBFilter(q, field, cond)
	}

	// Include content type if requested
	if query.IncludeType {
		q = q.Relation("ContentType")
	}

	// Count total
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply sorting
	switch query.SortBy {
	case "created_at":
		if query.SortOrder == "desc" {
			q = q.Order("ce.created_at DESC")
		} else {
			q = q.Order("ce.created_at ASC")
		}
	case "updated_at":
		if query.SortOrder == "desc" {
			q = q.Order("ce.updated_at DESC")
		} else {
			q = q.Order("ce.updated_at ASC")
		}
	case "published_at":
		if query.SortOrder == "desc" {
			q = q.Order("ce.published_at DESC NULLS LAST")
		} else {
			q = q.Order("ce.published_at ASC NULLS LAST")
		}
	default:
		// Sort by JSONB field
		if query.SortBy != "" {
			if query.SortOrder == "desc" {
				q = q.OrderExpr("ce.data->? DESC NULLS LAST", query.SortBy)
			} else {
				q = q.OrderExpr("ce.data->? ASC NULLS LAST", query.SortBy)
			}
		} else {
			q = q.Order("ce.created_at DESC")
		}
	}

	// Apply pagination
	offset := (query.Page - 1) * query.PageSize
	q = q.Limit(query.PageSize).Offset(offset)

	// Execute query
	var entries []*schema.ContentEntry

	err = q.Scan(ctx, &entries)
	if err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// applyJSONBFilter applies a JSONB filter condition.
func (r *contentEntryRepository) applyJSONBFilter(q *bun.SelectQuery, field string, cond FilterCondition) *bun.SelectQuery {
	// Convert value to string for text comparisons
	valueStr := fmt.Sprintf("%v", cond.Value)

	switch cond.Operator {
	case "eq":
		return q.Where("ce.data->>? = ?", field, valueStr)
	case "ne":
		return q.Where("ce.data->>? != ?", field, valueStr)
	case "gt":
		return q.Where("(ce.data->>?)::numeric > ?", field, cond.Value)
	case "gte":
		return q.Where("(ce.data->>?)::numeric >= ?", field, cond.Value)
	case "lt":
		return q.Where("(ce.data->>?)::numeric < ?", field, cond.Value)
	case "lte":
		return q.Where("(ce.data->>?)::numeric <= ?", field, cond.Value)
	case "like":
		return q.Where("ce.data->>? LIKE ?", field, cond.Value)
	case "ilike":
		return q.Where("ce.data->>? ILIKE ?", field, cond.Value)
	case "contains":
		// Text contains (case-insensitive)
		return q.Where("ce.data->>? ILIKE ?", field, "%"+valueStr+"%")
	case "startsWith":
		return q.Where("ce.data->>? LIKE ?", field, valueStr+"%")
	case "endsWith":
		return q.Where("ce.data->>? LIKE ?", field, "%"+valueStr)
	case "in":
		return q.Where("ce.data->>? IN (?)", field, bun.In(cond.Value))
	case "nin", "notIn":
		return q.Where("ce.data->>? NOT IN (?)", field, bun.In(cond.Value))
	case "null", "isNull":
		if cond.Value == true || cond.Value == "true" {
			return q.Where("ce.data->>? IS NULL OR ce.data->>? = 'null'", field, field)
		}

		return q.Where("ce.data->>? IS NOT NULL AND ce.data->>? != 'null'", field, field)
	case "exists":
		if cond.Value == true || cond.Value == "true" {
			return q.Where("ce.data ? ?", field)
		}

		return q.Where("NOT (ce.data ? ?)", field)
	case "jsonContains":
		return q.Where("ce.data->? @> ?", field, cond.Value)
	case "jsonHasKey":
		return q.Where("ce.data->? ? ?", field, cond.Value)
	default:
		return q.Where("ce.data->>? = ?", field, valueStr)
	}
}

// Update updates a content entry.
func (r *contentEntryRepository) Update(ctx context.Context, entry *schema.ContentEntry) error {
	entry.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(entry).
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)

	return err
}

// Delete soft-deletes a content entry.
func (r *contentEntryRepository) Delete(ctx context.Context, id xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.ContentEntry)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)

	return err
}

// HardDelete permanently deletes a content entry.
func (r *contentEntryRepository) HardDelete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentEntry)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// =============================================================================
// Bulk Operations
// =============================================================================

// BulkUpdateStatus updates the status of multiple entries.
func (r *contentEntryRepository) BulkUpdateStatus(ctx context.Context, ids []xid.ID, status string) error {
	now := time.Now()
	update := r.db.NewUpdate().
		Model((*schema.ContentEntry)(nil)).
		Set("status = ?", status).
		Set("updated_at = ?", now).
		Where("id IN (?)", bun.In(ids)).
		Where("deleted_at IS NULL")

	if status == "published" {
		update = update.Set("published_at = ?", now)
	}

	_, err := update.Exec(ctx)

	return err
}

// BulkDelete soft-deletes multiple entries.
func (r *contentEntryRepository) BulkDelete(ctx context.Context, ids []xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.ContentEntry)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id IN (?)", bun.In(ids)).
		Where("deleted_at IS NULL").
		Exec(ctx)

	return err
}

// =============================================================================
// Query Operations
// =============================================================================

// FindByFieldValue finds entries with a specific field value.
func (r *contentEntryRepository) FindByFieldValue(ctx context.Context, contentTypeID xid.ID, field string, value any) ([]*schema.ContentEntry, error) {
	var entries []*schema.ContentEntry

	err := r.db.NewSelect().
		Model(&entries).
		Where("content_type_id = ?", contentTypeID).
		Where("data->>? = ?", field, fmt.Sprintf("%v", value)).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// ExistsWithFieldValue checks if an entry exists with a specific field value.
func (r *contentEntryRepository) ExistsWithFieldValue(ctx context.Context, contentTypeID xid.ID, field string, value any, excludeID *xid.ID) (bool, error) {
	q := r.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("content_type_id = ?", contentTypeID).
		Where("data->>? = ?", field, fmt.Sprintf("%v", value)).
		Where("deleted_at IS NULL")

	if excludeID != nil {
		q = q.Where("id != ?", *excludeID)
	}

	count, err := q.Count(ctx)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// FindScheduledForPublish finds entries scheduled to be published before the given time.
func (r *contentEntryRepository) FindScheduledForPublish(ctx context.Context, before time.Time) ([]*schema.ContentEntry, error) {
	var entries []*schema.ContentEntry

	err := r.db.NewSelect().
		Model(&entries).
		Where("status = ?", "scheduled").
		Where("scheduled_at <= ?", before).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// =============================================================================
// Stats Operations
// =============================================================================

// Count counts total entries for a content type.
func (r *contentEntryRepository) Count(ctx context.Context, contentTypeID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL").
		Count(ctx)
}

// CountByStatus counts entries grouped by status for a content type.
func (r *contentEntryRepository) CountByStatus(ctx context.Context, contentTypeID xid.ID) (map[string]int, error) {
	var results []struct {
		Status string `bun:"status"`
		Count  int    `bun:"count"`
	}

	err := r.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		ColumnExpr("status, COUNT(*) as count").
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL").
		Group("status").
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range results {
		counts[r.Status] = r.Count
	}

	return counts, nil
}

// CountByAppEnv counts total entries for an app/environment.
func (r *contentEntryRepository) CountByAppEnv(ctx context.Context, appID, envID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Count(ctx)
}

// CountByAppEnvAndStatus counts entries grouped by status for an app/environment.
func (r *contentEntryRepository) CountByAppEnvAndStatus(ctx context.Context, appID, envID xid.ID) (map[string]int, error) {
	var results []struct {
		Status string `bun:"status"`
		Count  int    `bun:"count"`
	}

	err := r.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		ColumnExpr("status, COUNT(*) as count").
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Group("status").
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range results {
		counts[r.Status] = r.Count
	}

	return counts, nil
}
