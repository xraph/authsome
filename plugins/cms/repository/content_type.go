// Package repository implements the data access layer for the CMS plugin.
package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// ContentTypeRepository defines the interface for content type storage operations
type ContentTypeRepository interface {
	// CRUD operations
	Create(ctx context.Context, contentType *schema.ContentType) error
	FindByID(ctx context.Context, id xid.ID) (*schema.ContentType, error)
	FindByName(ctx context.Context, appID, envID xid.ID, name string) (*schema.ContentType, error)
	List(ctx context.Context, appID, envID xid.ID, query *core.ListContentTypesQuery) ([]*schema.ContentType, int, error)
	Update(ctx context.Context, contentType *schema.ContentType) error
	Delete(ctx context.Context, id xid.ID) error
	HardDelete(ctx context.Context, id xid.ID) error

	// Relation queries
	FindWithFields(ctx context.Context, id xid.ID) (*schema.ContentType, error)
	FindByNameWithFields(ctx context.Context, appID, envID xid.ID, name string) (*schema.ContentType, error)

	// Stats operations
	Count(ctx context.Context, appID, envID xid.ID) (int, error)
	CountEntries(ctx context.Context, contentTypeID xid.ID) (int, error)
	ExistsWithName(ctx context.Context, appID, envID xid.ID, name string) (bool, error)
}

// contentTypeRepository implements ContentTypeRepository using Bun ORM
type contentTypeRepository struct {
	db *bun.DB
}

// NewContentTypeRepository creates a new content type repository instance
func NewContentTypeRepository(db *bun.DB) ContentTypeRepository {
	return &contentTypeRepository{db: db}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new content type
func (r *contentTypeRepository) Create(ctx context.Context, contentType *schema.ContentType) error {
	if contentType.ID.IsNil() {
		contentType.ID = xid.New()
	}
	now := time.Now()
	contentType.CreatedAt = now
	contentType.UpdatedAt = now

	_, err := r.db.NewInsert().
		Model(contentType).
		Exec(ctx)
	return err
}

// FindByID finds a content type by ID
func (r *contentTypeRepository) FindByID(ctx context.Context, id xid.ID) (*schema.ContentType, error) {
	contentType := new(schema.ContentType)
	err := r.db.NewSelect().
		Model(contentType).
		Where("ct.id = ?", id).
		Where("ct.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrContentTypeNotFound(id.String())
		}
		return nil, err
	}
	return contentType, nil
}

// FindBySlug finds a content type by slug within an app/environment
func (r *contentTypeRepository) FindByName(ctx context.Context, appID, envID xid.ID, name string) (*schema.ContentType, error) {
	contentType := new(schema.ContentType)
	err := r.db.NewSelect().
		Model(contentType).
		Where("ct.app_id = ?", appID).
		Where("ct.environment_id = ?", envID).
		Where("LOWER(ct.name) = LOWER(?)", name).
		Where("ct.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrContentTypeNotFound(name)
		}
		return nil, err
	}
	return contentType, nil
}

// List lists content types with filtering and pagination
func (r *contentTypeRepository) List(ctx context.Context, appID, envID xid.ID, query *core.ListContentTypesQuery) ([]*schema.ContentType, int, error) {
	if query == nil {
		query = &core.ListContentTypesQuery{}
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
		Model((*schema.ContentType)(nil)).
		Where("ct.app_id = ?", appID).
		Where("ct.environment_id = ?", envID).
		Where("ct.deleted_at IS NULL")

	// Apply search filter
	if query.Search != "" {
		search := "%" + strings.ToLower(query.Search) + "%"
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("LOWER(ct.name) LIKE ?", search).
				WhereOr("LOWER(ct.slug) LIKE ?", search).
				WhereOr("LOWER(ct.description) LIKE ?", search)
		})
	}

	// Count total
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply sorting
	switch query.SortBy {
	case "name":
		if query.SortOrder == "desc" {
			q = q.Order("ct.name DESC")
		} else {
			q = q.Order("ct.name ASC")
		}
	case "slug":
		if query.SortOrder == "desc" {
			q = q.Order("ct.slug DESC")
		} else {
			q = q.Order("ct.slug ASC")
		}
	case "created_at":
		if query.SortOrder == "desc" {
			q = q.Order("ct.created_at DESC")
		} else {
			q = q.Order("ct.created_at ASC")
		}
	case "updated_at":
		if query.SortOrder == "desc" {
			q = q.Order("ct.updated_at DESC")
		} else {
			q = q.Order("ct.updated_at ASC")
		}
	default:
		q = q.Order("ct.name ASC")
	}

	// Apply pagination
	offset := (query.Page - 1) * query.PageSize
	q = q.Limit(query.PageSize).Offset(offset)

	// Execute query
	var contentTypes []*schema.ContentType
	err = q.Scan(ctx, &contentTypes)
	if err != nil {
		return nil, 0, err
	}

	return contentTypes, total, nil
}

// Update updates a content type
func (r *contentTypeRepository) Update(ctx context.Context, contentType *schema.ContentType) error {
	contentType.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(contentType).
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// Delete soft-deletes a content type
func (r *contentTypeRepository) Delete(ctx context.Context, id xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.ContentType)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// HardDelete permanently deletes a content type
func (r *contentTypeRepository) HardDelete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentType)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// =============================================================================
// Relation Queries
// =============================================================================

// FindWithFields finds a content type with its fields loaded
func (r *contentTypeRepository) FindWithFields(ctx context.Context, id xid.ID) (*schema.ContentType, error) {
	contentType := new(schema.ContentType)
	err := r.db.NewSelect().
		Model(contentType).
		Relation("Fields", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("cf.\"order\" ASC")
		}).
		Where("ct.id = ?", id).
		Where("ct.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrContentTypeNotFound(id.String())
		}
		return nil, err
	}
	return contentType, nil
}

// FindBySlugWithFields finds a content type by slug with its fields loaded
func (r *contentTypeRepository) FindByNameWithFields(ctx context.Context, appID, envID xid.ID, name string) (*schema.ContentType, error) {
	contentType := new(schema.ContentType)
	err := r.db.NewSelect().
		Model(contentType).
		Relation("Fields", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("cf.\"order\" ASC")
		}).
		Where("ct.app_id = ?", appID).
		Where("ct.environment_id = ?", envID).
		Where("LOWER(ct.name) = LOWER(?)", name).
		Where("ct.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrContentTypeNotFound(name)
		}
		return nil, err
	}
	return contentType, nil
}

// =============================================================================
// Stats Operations
// =============================================================================

// Count counts total content types for an app/environment
func (r *contentTypeRepository) Count(ctx context.Context, appID, envID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.ContentType)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Count(ctx)
}

// CountEntries counts total entries for a content type
func (r *contentTypeRepository) CountEntries(ctx context.Context, contentTypeID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL").
		Count(ctx)
}

// ExistsWithSlug checks if a content type with the given name exists
func (r *contentTypeRepository) ExistsWithName(ctx context.Context, appID, envID xid.ID, name string) (bool, error) {
	count, err := r.db.NewSelect().
		Model((*schema.ContentType)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("LOWER(name) = LOWER(?)", name).
		Where("deleted_at IS NULL").
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
