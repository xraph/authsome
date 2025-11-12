package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// environmentRepository is the Bun implementation of environment.Repository
type environmentRepository struct {
	db *bun.DB
}

// NewEnvironmentRepository creates a new environment repository
func NewEnvironmentRepository(db *bun.DB) *environmentRepository {
	return &environmentRepository{db: db}
}

// Create creates a new environment
func (r *environmentRepository) Create(ctx context.Context, env *schema.Environment) error {
	_, err := r.db.NewInsert().
		Model(env).
		Exec(ctx)
	return err
}

// FindByID retrieves an environment by ID
func (r *environmentRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Environment, error) {
	env := new(schema.Environment)
	err := r.db.NewSelect().
		Model(env).
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found")
	}
	return env, err
}

// FindByAppAndSlug retrieves an environment by app ID and slug
func (r *environmentRepository) FindByAppAndSlug(ctx context.Context, appID xid.ID, slug string) (*schema.Environment, error) {
	env := new(schema.Environment)
	err := r.db.NewSelect().
		Model(env).
		Where("app_id = ? AND slug = ?", appID, slug).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found")
	}
	return env, err
}

// FindDefaultByApp retrieves the default environment for an app
func (r *environmentRepository) FindDefaultByApp(ctx context.Context, appID xid.ID) (*schema.Environment, error) {
	env := new(schema.Environment)
	err := r.db.NewSelect().
		Model(env).
		Where("app_id = ? AND is_default = ?", appID, true).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("default environment not found")
	}
	return env, err
}

// ListByApp lists environments for an app
func (r *environmentRepository) ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.Environment, error) {
	var envs []*schema.Environment
	query := r.db.NewSelect().
		Model(&envs).
		Where("app_id = ?", appID).
		Order("is_default DESC", "created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return envs, err
}

// CountByApp counts environments for an app
func (r *environmentRepository) CountByApp(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Environment)(nil)).
		Where("app_id = ?", appID).
		Count(ctx)
	return count, err
}

// Update updates an environment
func (r *environmentRepository) Update(ctx context.Context, env *schema.Environment) error {
	_, err := r.db.NewUpdate().
		Model(env).
		WherePK().
		Exec(ctx)
	return err
}

// Delete deletes an environment
func (r *environmentRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Environment)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// CreatePromotion creates a new environment promotion
func (r *environmentRepository) CreatePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion) error {
	_, err := r.db.NewInsert().
		Model(promotion).
		Exec(ctx)
	return err
}

// FindPromotionByID retrieves a promotion by ID
func (r *environmentRepository) FindPromotionByID(ctx context.Context, id xid.ID) (*schema.EnvironmentPromotion, error) {
	promotion := new(schema.EnvironmentPromotion)
	err := r.db.NewSelect().
		Model(promotion).
		Relation("SourceEnv").
		Relation("TargetEnv").
		Where("ep.id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("promotion not found")
	}
	return promotion, err
}

// ListPromotionsByApp lists promotions for an app
func (r *environmentRepository) ListPromotionsByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.EnvironmentPromotion, error) {
	var promotions []*schema.EnvironmentPromotion
	query := r.db.NewSelect().
		Model(&promotions).
		Relation("SourceEnv").
		Relation("TargetEnv").
		Where("ep.app_id = ?", appID).
		Order("ep.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return promotions, err
}

// UpdatePromotion updates a promotion
func (r *environmentRepository) UpdatePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion) error {
	_, err := r.db.NewUpdate().
		Model(promotion).
		WherePK().
		Exec(ctx)
	return err
}
