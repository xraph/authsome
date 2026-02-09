package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// environmentRepository is the Bun implementation of environment.Repository.
type environmentRepository struct {
	db *bun.DB
}

// NewEnvironmentRepository creates a new environment repository.
func NewEnvironmentRepository(db *bun.DB) *environmentRepository {
	return &environmentRepository{db: db}
}

// Create creates a new environment.
func (r *environmentRepository) Create(ctx context.Context, env *schema.Environment) error {
	_, err := r.db.NewInsert().
		Model(env).
		Exec(ctx)

	return err
}

// FindByID retrieves an environment by ID.
func (r *environmentRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Environment, error) {
	env := new(schema.Environment)
	err := r.db.NewSelect().
		Model(env).
		Where("id = ?", id).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.NotFound("environment not found")
	}

	return env, err
}

// FindByAppAndSlug retrieves an environment by app ID and slug.
func (r *environmentRepository) FindByAppAndSlug(ctx context.Context, appID xid.ID, slug string) (*schema.Environment, error) {
	env := new(schema.Environment)
	err := r.db.NewSelect().
		Model(env).
		Where("app_id = ? AND slug = ?", appID, slug).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.NotFound("environment not found")
	}

	return env, err
}

// FindDefaultByApp retrieves the default environment for an app.
func (r *environmentRepository) FindDefaultByApp(ctx context.Context, appID xid.ID) (*schema.Environment, error) {
	env := new(schema.Environment)
	err := r.db.NewSelect().
		Model(env).
		Where("app_id = ? AND is_default = ?", appID, true).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.NotFound("default environment not found")
	}

	return env, err
}

// ListEnvironments lists environments with pagination and filtering.
func (r *environmentRepository) ListEnvironments(ctx context.Context, filter *environment.ListEnvironmentsFilter) (*pagination.PageResponse[*schema.Environment], error) {
	var envs []*schema.Environment

	// Build base query
	query := r.db.NewSelect().
		Model(&envs).
		Where("app_id = ?", filter.AppID)

	// Apply filters
	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
	}

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().
		Model((*schema.Environment)(nil)).
		Where("app_id = ?", filter.AppID)
	if filter.Type != nil {
		countQuery = countQuery.Where("type = ?", *filter.Type)
	}

	if filter.Status != nil {
		countQuery = countQuery.Where("status = ?", *filter.Status)
	}

	if filter.IsDefault != nil {
		countQuery = countQuery.Where("is_default = ?", *filter.IsDefault)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	offset := filter.GetOffset()
	limit := filter.GetLimit()
	query = query.Limit(limit).Offset(offset)

	// Apply ordering
	query = query.Order("is_default DESC", "created_at ASC")

	// Execute query
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(envs, int64(total), &filter.PaginationParams), nil
}

// CountByApp counts environments for an app.
func (r *environmentRepository) CountByApp(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Environment)(nil)).
		Where("app_id = ?", appID).
		Count(ctx)

	return count, err
}

// Update updates an environment.
func (r *environmentRepository) Update(ctx context.Context, env *schema.Environment) error {
	_, err := r.db.NewUpdate().
		Model(env).
		WherePK().
		Exec(ctx)

	return err
}

// Delete deletes an environment.
func (r *environmentRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Environment)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// CreatePromotion creates a new environment promotion.
func (r *environmentRepository) CreatePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion) error {
	_, err := r.db.NewInsert().
		Model(promotion).
		Exec(ctx)

	return err
}

// FindPromotionByID retrieves a promotion by ID.
func (r *environmentRepository) FindPromotionByID(ctx context.Context, id xid.ID) (*schema.EnvironmentPromotion, error) {
	promotion := new(schema.EnvironmentPromotion)
	err := r.db.NewSelect().
		Model(promotion).
		Relation("SourceEnv").
		Relation("TargetEnv").
		Where("ep.id = ?", id).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.NotFound("promotion not found")
	}

	return promotion, err
}

// ListPromotions lists promotions with pagination and filtering.
func (r *environmentRepository) ListPromotions(ctx context.Context, filter *environment.ListPromotionsFilter) (*pagination.PageResponse[*schema.EnvironmentPromotion], error) {
	var promotions []*schema.EnvironmentPromotion

	// Build base query
	query := r.db.NewSelect().
		Model(&promotions).
		Relation("SourceEnv").
		Relation("TargetEnv").
		Where("ep.app_id = ?", filter.AppID)

	// Apply filters
	if filter.SourceEnvID != nil {
		query = query.Where("ep.source_env_id = ?", *filter.SourceEnvID)
	}

	if filter.TargetEnvID != nil {
		query = query.Where("ep.target_env_id = ?", *filter.TargetEnvID)
	}

	if filter.Status != nil {
		query = query.Where("ep.status = ?", *filter.Status)
	}

	if filter.PromotedBy != nil {
		query = query.Where("ep.promoted_by = ?", *filter.PromotedBy)
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().
		Model((*schema.EnvironmentPromotion)(nil)).
		Where("app_id = ?", filter.AppID)
	if filter.SourceEnvID != nil {
		countQuery = countQuery.Where("source_env_id = ?", *filter.SourceEnvID)
	}

	if filter.TargetEnvID != nil {
		countQuery = countQuery.Where("target_env_id = ?", *filter.TargetEnvID)
	}

	if filter.Status != nil {
		countQuery = countQuery.Where("status = ?", *filter.Status)
	}

	if filter.PromotedBy != nil {
		countQuery = countQuery.Where("promoted_by = ?", *filter.PromotedBy)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	offset := filter.GetOffset()
	limit := filter.GetLimit()
	query = query.Limit(limit).Offset(offset)

	// Apply ordering
	query = query.Order("ep.created_at DESC")

	// Execute query
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(promotions, int64(total), &filter.PaginationParams), nil
}

// UpdatePromotion updates a promotion.
func (r *environmentRepository) UpdatePromotion(ctx context.Context, promotion *schema.EnvironmentPromotion) error {
	_, err := r.db.NewUpdate().
		Model(promotion).
		WherePK().
		Exec(ctx)

	return err
}
