package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// featureRepository implements FeatureRepository using Bun.
type featureRepository struct {
	db *bun.DB
}

// NewFeatureRepository creates a new feature repository.
func NewFeatureRepository(db *bun.DB) FeatureRepository {
	return &featureRepository{db: db}
}

// Create creates a new feature.
func (r *featureRepository) Create(ctx context.Context, feature *schema.Feature) error {
	feature.CreatedAt = time.Now()
	feature.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(feature).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create feature: %w", err)
	}

	return nil
}

// Update updates an existing feature.
func (r *featureRepository) Update(ctx context.Context, feature *schema.Feature) error {
	feature.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(feature).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update feature: %w", err)
	}

	return nil
}

// Delete deletes a feature.
func (r *featureRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Feature)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete feature: %w", err)
	}

	return nil
}

// FindByID retrieves a feature by ID.
func (r *featureRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Feature, error) {
	feature := new(schema.Feature)

	err := r.db.NewSelect().
		Model(feature).
		Relation("Tiers", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("tier_order ASC")
		}).
		Where("sf.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find feature: %w", err)
	}

	return feature, nil
}

// FindByKey retrieves a feature by key within an app.
func (r *featureRepository) FindByKey(ctx context.Context, appID xid.ID, key string) (*schema.Feature, error) {
	feature := new(schema.Feature)

	err := r.db.NewSelect().
		Model(feature).
		Relation("Tiers", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("tier_order ASC")
		}).
		Where("sf.app_id = ?", appID).
		Where("sf.key = ?", key).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find feature by key: %w", err)
	}

	return feature, nil
}

// List retrieves features with optional filters.
func (r *featureRepository) List(ctx context.Context, filter *FeatureFilter) ([]*schema.Feature, int, error) {
	var features []*schema.Feature

	query := r.db.NewSelect().
		Model(&features).
		Relation("Tiers", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("tier_order ASC")
		}).
		Order("display_order ASC", "created_at DESC")

	if filter != nil {
		if filter.AppID != nil {
			query = query.Where("sf.app_id = ?", *filter.AppID)
		}

		if filter.Type != "" {
			query = query.Where("sf.type = ?", filter.Type)
		}

		if filter.IsPublic != nil {
			query = query.Where("sf.is_public = ?", *filter.IsPublic)
		}

		// Pagination
		pageSize := filter.PageSize
		if pageSize <= 0 {
			pageSize = 20
		}

		page := filter.Page
		if page <= 0 {
			page = 1
		}

		offset := (page - 1) * pageSize
		query = query.Limit(pageSize).Offset(offset)
	}

	count, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list features: %w", err)
	}

	return features, count, nil
}

// CreateTier creates a feature tier.
func (r *featureRepository) CreateTier(ctx context.Context, tier *schema.FeatureTier) error {
	tier.CreatedAt = time.Now()

	_, err := r.db.NewInsert().Model(tier).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create feature tier: %w", err)
	}

	return nil
}

// DeleteTiers deletes all tiers for a feature.
func (r *featureRepository) DeleteTiers(ctx context.Context, featureID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.FeatureTier)(nil)).
		Where("feature_id = ?", featureID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete feature tiers: %w", err)
	}

	return nil
}

// GetTiers retrieves all tiers for a feature.
func (r *featureRepository) GetTiers(ctx context.Context, featureID xid.ID) ([]*schema.FeatureTier, error) {
	var tiers []*schema.FeatureTier

	err := r.db.NewSelect().
		Model(&tiers).
		Where("feature_id = ?", featureID).
		Order("tier_order ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature tiers: %w", err)
	}

	return tiers, nil
}

// LinkToPlan links a feature to a plan.
func (r *featureRepository) LinkToPlan(ctx context.Context, link *schema.PlanFeatureLink) error {
	link.CreatedAt = time.Now()
	link.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(link).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to link feature to plan: %w", err)
	}

	return nil
}

// UpdatePlanLink updates a feature-plan link.
func (r *featureRepository) UpdatePlanLink(ctx context.Context, link *schema.PlanFeatureLink) error {
	link.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(link).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update plan feature link: %w", err)
	}

	return nil
}

// UnlinkFromPlan removes a feature from a plan.
func (r *featureRepository) UnlinkFromPlan(ctx context.Context, planID, featureID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.PlanFeatureLink)(nil)).
		Where("plan_id = ?", planID).
		Where("feature_id = ?", featureID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unlink feature from plan: %w", err)
	}

	return nil
}

// GetPlanLinks retrieves all feature links for a plan.
func (r *featureRepository) GetPlanLinks(ctx context.Context, planID xid.ID) ([]*schema.PlanFeatureLink, error) {
	var links []*schema.PlanFeatureLink

	err := r.db.NewSelect().
		Model(&links).
		Relation("Feature").
		Relation("Feature.Tiers", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("tier_order ASC")
		}).
		Where("spfl.plan_id = ?", planID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan feature links: %w", err)
	}

	return links, nil
}

// GetPlanLink retrieves a specific feature link.
func (r *featureRepository) GetPlanLink(ctx context.Context, planID, featureID xid.ID) (*schema.PlanFeatureLink, error) {
	link := new(schema.PlanFeatureLink)

	err := r.db.NewSelect().
		Model(link).
		Relation("Feature").
		Relation("Feature.Tiers", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("tier_order ASC")
		}).
		Where("spfl.plan_id = ?", planID).
		Where("spfl.feature_id = ?", featureID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan feature link: %w", err)
	}

	return link, nil
}

// GetFeaturePlans retrieves all plans that have a feature.
func (r *featureRepository) GetFeaturePlans(ctx context.Context, featureID xid.ID) ([]*schema.PlanFeatureLink, error) {
	var links []*schema.PlanFeatureLink

	err := r.db.NewSelect().
		Model(&links).
		Relation("Plan").
		Where("spfl.feature_id = ?", featureID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature plans: %w", err)
	}

	return links, nil
}
