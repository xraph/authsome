package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// ComponentSchemaRepository defines the interface for component schema storage operations.
type ComponentSchemaRepository interface {
	// CRUD operations
	Create(ctx context.Context, component *schema.ComponentSchema) error
	FindByID(ctx context.Context, id xid.ID) (*schema.ComponentSchema, error)
	FindByName(ctx context.Context, appID, envID xid.ID, name string) (*schema.ComponentSchema, error)
	List(ctx context.Context, appID, envID xid.ID, query *core.ListComponentSchemasQuery) ([]*schema.ComponentSchema, int, error)
	Update(ctx context.Context, component *schema.ComponentSchema) error
	Delete(ctx context.Context, id xid.ID) error
	HardDelete(ctx context.Context, id xid.ID) error

	// Stats operations
	Count(ctx context.Context, appID, envID xid.ID) (int, error)
	ExistsWithName(ctx context.Context, appID, envID xid.ID, name string) (bool, error)

	// Usage queries
	FindUsages(ctx context.Context, appID, envID xid.ID, componentSlug string) ([]*schema.ContentField, error)
	CountUsages(ctx context.Context, appID, envID xid.ID, componentSlug string) (int, error)
}

// componentSchemaRepository implements ComponentSchemaRepository using Bun ORM.
type componentSchemaRepository struct {
	db *bun.DB
}

// NewComponentSchemaRepository creates a new component schema repository instance.
func NewComponentSchemaRepository(db *bun.DB) ComponentSchemaRepository {
	return &componentSchemaRepository{db: db}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new component schema.
func (r *componentSchemaRepository) Create(ctx context.Context, component *schema.ComponentSchema) error {
	if component.ID.IsNil() {
		component.ID = xid.New()
	}

	now := time.Now()
	component.CreatedAt = now
	component.UpdatedAt = now

	if component.Fields == nil {
		component.Fields = schema.NestedFieldDefs{}
	}

	_, err := r.db.NewInsert().
		Model(component).
		Exec(ctx)

	return err
}

// FindByID finds a component schema by ID.
func (r *componentSchemaRepository) FindByID(ctx context.Context, id xid.ID) (*schema.ComponentSchema, error) {
	component := new(schema.ComponentSchema)

	err := r.db.NewSelect().
		Model(component).
		Where("cs.id = ?", id).
		Where("cs.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrComponentSchemaNotFound(id.String())
		}

		return nil, err
	}

	return component, nil
}

// FindByName finds a component schema by slug within an app/environment.
func (r *componentSchemaRepository) FindByName(ctx context.Context, appID, envID xid.ID, name string) (*schema.ComponentSchema, error) {
	component := new(schema.ComponentSchema)

	err := r.db.NewSelect().
		Model(component).
		Where("cs.app_id = ?", appID).
		Where("cs.environment_id = ?", envID).
		Where("LOWER(cs.name) = LOWER(?)", name).
		Where("cs.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrComponentSchemaNotFound(name)
		}

		return nil, err
	}

	return component, nil
}

// List lists component schemas with filtering and pagination.
func (r *componentSchemaRepository) List(ctx context.Context, appID, envID xid.ID, query *core.ListComponentSchemasQuery) ([]*schema.ComponentSchema, int, error) {
	if query == nil {
		query = &core.ListComponentSchemasQuery{}
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
		Model((*schema.ComponentSchema)(nil)).
		Where("cs.app_id = ?", appID).
		Where("cs.environment_id = ?", envID).
		Where("cs.deleted_at IS NULL")

	// Apply search filter
	if query.Search != "" {
		search := "%" + strings.ToLower(query.Search) + "%"
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("LOWER(cs.name) LIKE ?", search).
				WhereOr("LOWER(cs.slug) LIKE ?", search).
				WhereOr("LOWER(cs.description) LIKE ?", search)
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
			q = q.Order("cs.name DESC")
		} else {
			q = q.Order("cs.name ASC")
		}
	case "slug":
		if query.SortOrder == "desc" {
			q = q.Order("cs.slug DESC")
		} else {
			q = q.Order("cs.slug ASC")
		}
	case "created_at":
		if query.SortOrder == "desc" {
			q = q.Order("cs.created_at DESC")
		} else {
			q = q.Order("cs.created_at ASC")
		}
	case "updated_at":
		if query.SortOrder == "desc" {
			q = q.Order("cs.updated_at DESC")
		} else {
			q = q.Order("cs.updated_at ASC")
		}
	default:
		q = q.Order("cs.name ASC")
	}

	// Apply pagination
	offset := (query.Page - 1) * query.PageSize
	q = q.Limit(query.PageSize).Offset(offset)

	// components query
	var components []*schema.ComponentSchema

	err = q.Scan(ctx, &components)
	if err != nil {
		return nil, 0, err
	}

	return components, total, nil
}

// Update updates a component schema.
func (r *componentSchemaRepository) Update(ctx context.Context, component *schema.ComponentSchema) error {
	component.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(component).
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)

	return err
}

// Delete soft-deletes a component schema.
func (r *componentSchemaRepository) Delete(ctx context.Context, id xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.ComponentSchema)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)

	return err
}

// HardDelete permanently deletes a component schema.
func (r *componentSchemaRepository) HardDelete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ComponentSchema)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// =============================================================================
// Stats Operations
// =============================================================================

// Count counts total component schemas for an app/environment.
func (r *componentSchemaRepository) Count(ctx context.Context, appID, envID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.ComponentSchema)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Count(ctx)
}

// ExistsWithName checks if a component schema with the given name exists.
func (r *componentSchemaRepository) ExistsWithName(ctx context.Context, appID, envID xid.ID, name string) (bool, error) {
	count, err := r.db.NewSelect().
		Model((*schema.ComponentSchema)(nil)).
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

// =============================================================================
// Usage Queries
// =============================================================================

// FindUsages finds all content fields that reference this component schema.
func (r *componentSchemaRepository) FindUsages(ctx context.Context, appID, envID xid.ID, componentSlug string) ([]*schema.ContentField, error) {
	var fields []*schema.ContentField

	err := r.db.NewSelect().
		Model(&fields).
		Join("JOIN cms_content_types ct ON ct.id = cf.content_type_id").
		Where("ct.app_id = ?", appID).
		Where("ct.environment_id = ?", envID).
		Where("ct.deleted_at IS NULL").
		Where("cf.options->>'componentRef' = ?", componentSlug).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

// CountUsages counts how many content fields reference this component schema.
func (r *componentSchemaRepository) CountUsages(ctx context.Context, appID, envID xid.ID, componentSlug string) (int, error) {
	return r.db.NewSelect().
		Model((*schema.ContentField)(nil)).
		Join("JOIN cms_content_types ct ON ct.id = cf.content_type_id").
		Where("ct.app_id = ?", appID).
		Where("ct.environment_id = ?", envID).
		Where("ct.deleted_at IS NULL").
		Where("cf.options->>'componentRef' = ?", componentSlug).
		Count(ctx)
}
