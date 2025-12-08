package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// ContentFieldRepository defines the interface for content field storage operations
type ContentFieldRepository interface {
	// CRUD operations
	Create(ctx context.Context, field *schema.ContentField) error
	FindByID(ctx context.Context, id xid.ID) (*schema.ContentField, error)
	FindByName(ctx context.Context, contentTypeID xid.ID, name string) (*schema.ContentField, error)
	ListByContentType(ctx context.Context, contentTypeID xid.ID) ([]*schema.ContentField, error)
	Update(ctx context.Context, field *schema.ContentField) error
	Delete(ctx context.Context, id xid.ID) error
	DeleteAllForContentType(ctx context.Context, contentTypeID xid.ID) error

	// Ordering operations
	UpdateOrder(ctx context.Context, id xid.ID, order int) error
	ReorderFields(ctx context.Context, contentTypeID xid.ID, orders []FieldOrder) error
	GetMaxOrder(ctx context.Context, contentTypeID xid.ID) (int, error)

	// Stats operations
	Count(ctx context.Context, contentTypeID xid.ID) (int, error)
	ExistsWithName(ctx context.Context, contentTypeID xid.ID, name string) (bool, error)
}

// FieldOrder represents a field ID and its order
type FieldOrder struct {
	FieldID xid.ID
	Order   int
}

// contentFieldRepository implements ContentFieldRepository using Bun ORM
type contentFieldRepository struct {
	db *bun.DB
}

// NewContentFieldRepository creates a new content field repository instance
func NewContentFieldRepository(db *bun.DB) ContentFieldRepository {
	return &contentFieldRepository{db: db}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new content field
func (r *contentFieldRepository) Create(ctx context.Context, field *schema.ContentField) error {
	if field.ID.IsNil() {
		field.ID = xid.New()
	}
	now := time.Now()
	field.CreatedAt = now
	field.UpdatedAt = now

	// Get next order if not set
	if field.Order == 0 {
		maxOrder, _ := r.GetMaxOrder(ctx, field.ContentTypeID)
		field.Order = maxOrder + 1
	}

	_, err := r.db.NewInsert().
		Model(field).
		Exec(ctx)
	return err
}

// FindByID finds a content field by ID
func (r *contentFieldRepository) FindByID(ctx context.Context, id xid.ID) (*schema.ContentField, error) {
	field := new(schema.ContentField)
	err := r.db.NewSelect().
		Model(field).
		Where("cf.id = ?", id).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrFieldNotFound(id.String())
		}
		return nil, err
	}
	return field, nil
}

// FindBySlug finds a content field by slug within a content type
func (r *contentFieldRepository) FindByName(ctx context.Context, contentTypeID xid.ID, name string) (*schema.ContentField, error) {
	field := new(schema.ContentField)
	err := r.db.NewSelect().
		Model(field).
		Where("cf.content_type_id = ?", contentTypeID).
		Where("LOWER(cf.name) = LOWER(?)", name).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrFieldNotFound(name)
		}
		return nil, err
	}
	return field, nil
}

// ListByContentType lists all fields for a content type ordered by Order
func (r *contentFieldRepository) ListByContentType(ctx context.Context, contentTypeID xid.ID) ([]*schema.ContentField, error) {
	var fields []*schema.ContentField
	err := r.db.NewSelect().
		Model(&fields).
		Where("content_type_id = ?", contentTypeID).
		OrderExpr("\"order\" ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

// Update updates a content field
func (r *contentFieldRepository) Update(ctx context.Context, field *schema.ContentField) error {
	field.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(field).
		WherePK().
		Exec(ctx)
	return err
}

// Delete deletes a content field
func (r *contentFieldRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentField)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// DeleteAllForContentType deletes all fields for a content type
func (r *contentFieldRepository) DeleteAllForContentType(ctx context.Context, contentTypeID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentField)(nil)).
		Where("content_type_id = ?", contentTypeID).
		Exec(ctx)
	return err
}

// =============================================================================
// Ordering Operations
// =============================================================================

// UpdateOrder updates the order of a single field
func (r *contentFieldRepository) UpdateOrder(ctx context.Context, id xid.ID, order int) error {
	_, err := r.db.NewUpdate().
		Model((*schema.ContentField)(nil)).
		Set("\"order\" = ?", order).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// ReorderFields reorders multiple fields in a content type
func (r *contentFieldRepository) ReorderFields(ctx context.Context, contentTypeID xid.ID, orders []FieldOrder) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		now := time.Now()
		for _, fo := range orders {
			_, err := tx.NewUpdate().
				Model((*schema.ContentField)(nil)).
				Set("\"order\" = ?", fo.Order).
				Set("updated_at = ?", now).
				Where("id = ?", fo.FieldID).
				Where("content_type_id = ?", contentTypeID).
				Exec(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetMaxOrder returns the maximum order value for fields in a content type
func (r *contentFieldRepository) GetMaxOrder(ctx context.Context, contentTypeID xid.ID) (int, error) {
	var maxOrder int
	err := r.db.NewSelect().
		Model((*schema.ContentField)(nil)).
		ColumnExpr("COALESCE(MAX(\"order\"), 0)").
		Where("content_type_id = ?", contentTypeID).
		Scan(ctx, &maxOrder)
	if err != nil {
		return 0, err
	}
	return maxOrder, nil
}

// =============================================================================
// Stats Operations
// =============================================================================

// Count counts total fields for a content type
func (r *contentFieldRepository) Count(ctx context.Context, contentTypeID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.ContentField)(nil)).
		Where("content_type_id = ?", contentTypeID).
		Count(ctx)
}

// ExistsWithSlug checks if a field with the given name exists in a content type
func (r *contentFieldRepository) ExistsWithName(ctx context.Context, contentTypeID xid.ID, name string) (bool, error) {
	count, err := r.db.NewSelect().
		Model((*schema.ContentField)(nil)).
		Where("content_type_id = ?", contentTypeID).
		Where("LOWER(name) = LOWER(?)", name).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
