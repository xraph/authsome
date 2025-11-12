package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/multitenancy/app"
)

// AppRepository implements app.AppRepository
type AppRepository struct {
	db *bun.DB
}

// NewAppRepository creates a new app repository
func NewAppRepository(db *bun.DB) *AppRepository {
	return &AppRepository{db: db}
}

// Create creates a new app
func (r *AppRepository) Create(ctx context.Context, a *app.App) error {
	_, err := r.db.NewInsert().Model(a).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}
	return nil
}

// FindByID finds an app by ID
func (r *AppRepository) FindByID(ctx context.Context, id xid.ID) (*app.App, error) {
	a := &app.App{}
	err := r.db.NewSelect().Model(a).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrAppNotFound
		}
		return nil, fmt.Errorf("failed to find app: %w", err)
	}
	return a, nil
}

// FindBySlug finds an app by slug
func (r *AppRepository) FindBySlug(ctx context.Context, slug string) (*app.App, error) {
	a := &app.App{}
	err := r.db.NewSelect().Model(a).Where("slug = ?", slug).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrAppNotFound
		}
		return nil, fmt.Errorf("failed to find app by slug: %w", err)
	}
	return a, nil
}

// Update updates an app
func (r *AppRepository) Update(ctx context.Context, a *app.App) error {
	_, err := r.db.NewUpdate().Model(a).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update app: %w", err)
	}
	return nil
}

// Delete deletes an app
func (r *AppRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*app.App)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}
	return nil
}

// List lists apps with pagination
func (r *AppRepository) List(ctx context.Context, limit, offset int) ([]*app.App, error) {
	var apps []*app.App

	// Get paginated results
	err := r.db.NewSelect().
		Model(&apps).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	return apps, nil
}

// Count returns the total number of apps
func (r *AppRepository) Count(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*app.App)(nil)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count apps: %w", err)
	}
	return count, nil
}
