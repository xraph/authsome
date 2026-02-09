package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// RelationRepository defines the interface for content relation storage operations.
type RelationRepository interface {
	// Content Relations (entry-to-entry)
	CreateRelation(ctx context.Context, relation *schema.ContentRelation) error
	DeleteRelation(ctx context.Context, id xid.ID) error
	DeleteRelationByEntries(ctx context.Context, sourceID, targetID xid.ID, fieldSlug string) error
	DeleteAllForEntry(ctx context.Context, entryID xid.ID) error
	DeleteAllForField(ctx context.Context, entryID xid.ID, fieldSlug string) error
	FindRelations(ctx context.Context, sourceID xid.ID, fieldSlug string) ([]*schema.ContentRelation, error)
	FindReverseRelations(ctx context.Context, targetID xid.ID, fieldSlug string) ([]*schema.ContentRelation, error)
	FindAllRelations(ctx context.Context, entryID xid.ID) ([]*schema.ContentRelation, error)
	UpdateRelationOrder(ctx context.Context, id xid.ID, order int) error
	BulkCreateRelations(ctx context.Context, relations []*schema.ContentRelation) error
	BulkUpdateOrder(ctx context.Context, sourceID xid.ID, fieldSlug string, orderedTargetIDs []xid.ID) error

	// Content Type Relations (type-to-type definitions)
	CreateTypeRelation(ctx context.Context, relation *schema.ContentTypeRelation) error
	UpdateTypeRelation(ctx context.Context, relation *schema.ContentTypeRelation) error
	DeleteTypeRelation(ctx context.Context, id xid.ID) error
	FindTypeRelationByID(ctx context.Context, id xid.ID) (*schema.ContentTypeRelation, error)
	FindTypeRelationByField(ctx context.Context, contentTypeID xid.ID, fieldSlug string) (*schema.ContentTypeRelation, error)
	FindTypeRelationsForType(ctx context.Context, contentTypeID xid.ID) ([]*schema.ContentTypeRelation, error)
	FindInverseRelation(ctx context.Context, targetTypeID xid.ID, targetField string) (*schema.ContentTypeRelation, error)
}

// bunRelationRepository implements RelationRepository using Bun ORM.
type bunRelationRepository struct {
	db *bun.DB
}

// NewRelationRepository creates a new relation repository.
func NewRelationRepository(db *bun.DB) RelationRepository {
	return &bunRelationRepository{db: db}
}

// =============================================================================
// Content Relations (entry-to-entry)
// =============================================================================

// CreateRelation creates a new content relation.
func (r *bunRelationRepository) CreateRelation(ctx context.Context, relation *schema.ContentRelation) error {
	relation.BeforeInsert()

	_, err := r.db.NewInsert().Model(relation).Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to create relation", err)
	}

	return nil
}

// DeleteRelation deletes a relation by ID.
func (r *bunRelationRepository) DeleteRelation(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentRelation)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to delete relation", err)
	}

	return nil
}

// DeleteRelationByEntries deletes a specific relation between two entries.
func (r *bunRelationRepository) DeleteRelationByEntries(ctx context.Context, sourceID, targetID xid.ID, fieldSlug string) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentRelation)(nil)).
		Where("source_entry_id = ?", sourceID).
		Where("target_entry_id = ?", targetID).
		Where("field_name = ?", fieldSlug).
		Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to delete relation", err)
	}

	return nil
}

// DeleteAllForEntry deletes all relations involving an entry (as source or target).
func (r *bunRelationRepository) DeleteAllForEntry(ctx context.Context, entryID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentRelation)(nil)).
		Where("source_entry_id = ? OR target_entry_id = ?", entryID, entryID).
		Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to delete relations for entry", err)
	}

	return nil
}

// DeleteAllForField deletes all relations for a specific field on an entry.
func (r *bunRelationRepository) DeleteAllForField(ctx context.Context, entryID xid.ID, fieldSlug string) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentRelation)(nil)).
		Where("source_entry_id = ?", entryID).
		Where("field_name = ?", fieldSlug).
		Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to delete relations for field", err)
	}

	return nil
}

// FindRelations finds all relations from a source entry for a specific field.
func (r *bunRelationRepository) FindRelations(ctx context.Context, sourceID xid.ID, fieldSlug string) ([]*schema.ContentRelation, error) {
	var relations []*schema.ContentRelation

	err := r.db.NewSelect().
		Model(&relations).
		Where("source_entry_id = ?", sourceID).
		Where("field_name = ?", fieldSlug).
		OrderExpr("\"order\" ASC").
		Relation("TargetEntry").
		Scan(ctx)
	if err != nil {
		return nil, core.ErrDatabaseError("failed to find relations", err)
	}

	return relations, nil
}

// FindReverseRelations finds all relations pointing to a target entry.
func (r *bunRelationRepository) FindReverseRelations(ctx context.Context, targetID xid.ID, fieldSlug string) ([]*schema.ContentRelation, error) {
	var relations []*schema.ContentRelation

	err := r.db.NewSelect().
		Model(&relations).
		Where("target_entry_id = ?", targetID).
		Where("field_name = ?", fieldSlug).
		OrderExpr("\"order\" ASC").
		Relation("SourceEntry").
		Scan(ctx)
	if err != nil {
		return nil, core.ErrDatabaseError("failed to find reverse relations", err)
	}

	return relations, nil
}

// FindAllRelations finds all relations for an entry (as source).
func (r *bunRelationRepository) FindAllRelations(ctx context.Context, entryID xid.ID) ([]*schema.ContentRelation, error) {
	var relations []*schema.ContentRelation

	err := r.db.NewSelect().
		Model(&relations).
		Where("source_entry_id = ?", entryID).
		OrderExpr("field_name ASC, \"order\" ASC").
		Relation("TargetEntry").
		Scan(ctx)
	if err != nil {
		return nil, core.ErrDatabaseError("failed to find relations", err)
	}

	return relations, nil
}

// UpdateRelationOrder updates the order of a relation.
func (r *bunRelationRepository) UpdateRelationOrder(ctx context.Context, id xid.ID, order int) error {
	_, err := r.db.NewUpdate().
		Model((*schema.ContentRelation)(nil)).
		Set("\"order\" = ?", order).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to update relation order", err)
	}

	return nil
}

// BulkCreateRelations creates multiple relations in a single transaction.
func (r *bunRelationRepository) BulkCreateRelations(ctx context.Context, relations []*schema.ContentRelation) error {
	if len(relations) == 0 {
		return nil
	}

	for _, rel := range relations {
		rel.BeforeInsert()
	}

	_, err := r.db.NewInsert().Model(&relations).Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to bulk create relations", err)
	}

	return nil
}

// BulkUpdateOrder updates the order of all relations for a field.
func (r *bunRelationRepository) BulkUpdateOrder(ctx context.Context, sourceID xid.ID, fieldSlug string, orderedTargetIDs []xid.ID) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for i, targetID := range orderedTargetIDs {
			_, err := tx.NewUpdate().
				Model((*schema.ContentRelation)(nil)).
				Set("\"order\" = ?", i).
				Where("source_entry_id = ?", sourceID).
				Where("target_entry_id = ?", targetID).
				Where("field_name = ?", fieldSlug).
				Exec(ctx)
			if err != nil {
				return core.ErrDatabaseError("failed to update relation order", err)
			}
		}

		return nil
	})
}

// =============================================================================
// Content Type Relations (type-to-type definitions)
// =============================================================================

// CreateTypeRelation creates a new content type relation definition.
func (r *bunRelationRepository) CreateTypeRelation(ctx context.Context, relation *schema.ContentTypeRelation) error {
	relation.BeforeInsert()

	_, err := r.db.NewInsert().Model(relation).Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to create type relation", err)
	}

	return nil
}

// UpdateTypeRelation updates a content type relation definition.
func (r *bunRelationRepository) UpdateTypeRelation(ctx context.Context, relation *schema.ContentTypeRelation) error {
	_, err := r.db.NewUpdate().
		Model(relation).
		WherePK().
		Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to update type relation", err)
	}

	return nil
}

// DeleteTypeRelation deletes a content type relation definition.
func (r *bunRelationRepository) DeleteTypeRelation(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentTypeRelation)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return core.ErrDatabaseError("failed to delete type relation", err)
	}

	return nil
}

// FindTypeRelationByID finds a type relation by ID.
func (r *bunRelationRepository) FindTypeRelationByID(ctx context.Context, id xid.ID) (*schema.ContentTypeRelation, error) {
	relation := new(schema.ContentTypeRelation)

	err := r.db.NewSelect().
		Model(relation).
		Where("id = ?", id).
		Relation("SourceContentType").
		Relation("TargetContentType").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, core.ErrTypeRelationNotFound(id.String())
		}

		return nil, core.ErrDatabaseError("failed to find type relation", err)
	}

	return relation, nil
}

// FindTypeRelationByField finds a type relation by content type and field name.
func (r *bunRelationRepository) FindTypeRelationByField(ctx context.Context, contentTypeID xid.ID, fieldSlug string) (*schema.ContentTypeRelation, error) {
	relation := new(schema.ContentTypeRelation)

	err := r.db.NewSelect().
		Model(relation).
		Where("source_content_type_id = ?", contentTypeID).
		Where("source_field_name = ?", fieldSlug).
		Relation("SourceContentType").
		Relation("TargetContentType").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No relation defined for this field
		}

		return nil, core.ErrDatabaseError("failed to find type relation", err)
	}

	return relation, nil
}

// FindTypeRelationsForType finds all type relations for a content type.
func (r *bunRelationRepository) FindTypeRelationsForType(ctx context.Context, contentTypeID xid.ID) ([]*schema.ContentTypeRelation, error) {
	var relations []*schema.ContentTypeRelation

	err := r.db.NewSelect().
		Model(&relations).
		Where("source_content_type_id = ? OR target_content_type_id = ?", contentTypeID, contentTypeID).
		Relation("SourceContentType").
		Relation("TargetContentType").
		Scan(ctx)
	if err != nil {
		return nil, core.ErrDatabaseError("failed to find type relations", err)
	}

	return relations, nil
}

// FindInverseRelation finds the inverse relation for a bidirectional relation.
func (r *bunRelationRepository) FindInverseRelation(ctx context.Context, targetTypeID xid.ID, targetField string) (*schema.ContentTypeRelation, error) {
	relation := new(schema.ContentTypeRelation)

	err := r.db.NewSelect().
		Model(relation).
		Where("target_content_type_id = ?", targetTypeID).
		Where("target_field_name = ?", targetField).
		Relation("SourceContentType").
		Relation("TargetContentType").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, core.ErrDatabaseError("failed to find inverse relation", err)
	}

	return relation, nil
}
