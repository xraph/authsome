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

// RevisionRepository defines the interface for content revision storage operations
type RevisionRepository interface {
	// CRUD operations
	Create(ctx context.Context, revision *schema.ContentRevision) error
	FindByID(ctx context.Context, id xid.ID) (*schema.ContentRevision, error)
	FindByVersion(ctx context.Context, entryID xid.ID, version int) (*schema.ContentRevision, error)
	List(ctx context.Context, entryID xid.ID, page, pageSize int) ([]*schema.ContentRevision, int, error)
	Delete(ctx context.Context, id xid.ID) error
	DeleteAllForEntry(ctx context.Context, entryID xid.ID) error
	DeleteOldRevisions(ctx context.Context, entryID xid.ID, keepCount int) error

	// Stats operations
	Count(ctx context.Context, entryID xid.ID) (int, error)
	GetLatestVersion(ctx context.Context, entryID xid.ID) (int, error)
}

// revisionRepository implements RevisionRepository using Bun ORM
type revisionRepository struct {
	db *bun.DB
}

// NewRevisionRepository creates a new revision repository instance
func NewRevisionRepository(db *bun.DB) RevisionRepository {
	return &revisionRepository{db: db}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create creates a new content revision
func (r *revisionRepository) Create(ctx context.Context, revision *schema.ContentRevision) error {
	if revision.ID.IsNil() {
		revision.ID = xid.New()
	}
	if revision.CreatedAt.IsZero() {
		revision.CreatedAt = time.Now()
	}
	if revision.Data == nil {
		revision.Data = make(schema.EntryData)
	}

	_, err := r.db.NewInsert().
		Model(revision).
		Exec(ctx)
	return err
}

// FindByID finds a revision by ID
func (r *revisionRepository) FindByID(ctx context.Context, id xid.ID) (*schema.ContentRevision, error) {
	revision := new(schema.ContentRevision)
	err := r.db.NewSelect().
		Model(revision).
		Where("cr.id = ?", id).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrRevisionNotFound("", 0)
		}
		return nil, err
	}
	return revision, nil
}

// FindByVersion finds a specific version of an entry
func (r *revisionRepository) FindByVersion(ctx context.Context, entryID xid.ID, version int) (*schema.ContentRevision, error) {
	revision := new(schema.ContentRevision)
	err := r.db.NewSelect().
		Model(revision).
		Where("cr.entry_id = ?", entryID).
		Where("cr.version = ?", version).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrRevisionNotFound(entryID.String(), version)
		}
		return nil, err
	}
	return revision, nil
}

// List lists revisions for an entry with pagination
func (r *revisionRepository) List(ctx context.Context, entryID xid.ID, page, pageSize int) ([]*schema.ContentRevision, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	// Count total
	total, err := r.db.NewSelect().
		Model((*schema.ContentRevision)(nil)).
		Where("entry_id = ?", entryID).
		Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Get revisions
	offset := (page - 1) * pageSize
	var revisions []*schema.ContentRevision
	err = r.db.NewSelect().
		Model(&revisions).
		Where("entry_id = ?", entryID).
		Order("version DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	return revisions, total, nil
}

// Delete deletes a revision
func (r *revisionRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentRevision)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// DeleteAllForEntry deletes all revisions for an entry
func (r *revisionRepository) DeleteAllForEntry(ctx context.Context, entryID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.ContentRevision)(nil)).
		Where("entry_id = ?", entryID).
		Exec(ctx)
	return err
}

// DeleteOldRevisions deletes old revisions, keeping only the most recent N versions
func (r *revisionRepository) DeleteOldRevisions(ctx context.Context, entryID xid.ID, keepCount int) error {
	if keepCount <= 0 {
		return nil
	}

	// Get versions to keep
	var keepVersions []int
	err := r.db.NewSelect().
		Model((*schema.ContentRevision)(nil)).
		Column("version").
		Where("entry_id = ?", entryID).
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
		Model((*schema.ContentRevision)(nil)).
		Where("entry_id = ?", entryID).
		Where("version NOT IN (?)", bun.In(keepVersions)).
		Exec(ctx)
	return err
}

// =============================================================================
// Stats Operations
// =============================================================================

// Count counts total revisions for an entry
func (r *revisionRepository) Count(ctx context.Context, entryID xid.ID) (int, error) {
	return r.db.NewSelect().
		Model((*schema.ContentRevision)(nil)).
		Where("entry_id = ?", entryID).
		Count(ctx)
}

// GetLatestVersion returns the latest version number for an entry
func (r *revisionRepository) GetLatestVersion(ctx context.Context, entryID xid.ID) (int, error) {
	var version int
	err := r.db.NewSelect().
		Model((*schema.ContentRevision)(nil)).
		ColumnExpr("COALESCE(MAX(version), 0)").
		Where("entry_id = ?", entryID).
		Scan(ctx, &version)
	if err != nil {
		return 0, err
	}
	return version, nil
}
