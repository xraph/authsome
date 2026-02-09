package repository

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// planRepository implements PlanRepository using Bun.
type planRepository struct {
	db *bun.DB
}

// NewPlanRepository creates a new plan repository.
func NewPlanRepository(db *bun.DB) PlanRepository {
	return &planRepository{db: db}
}

// Create creates a new plan.
func (r *planRepository) Create(ctx context.Context, plan *schema.SubscriptionPlan) error {
	_, err := r.db.NewInsert().Model(plan).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}

	return nil
}

// Update updates an existing plan.
func (r *planRepository) Update(ctx context.Context, plan *schema.SubscriptionPlan) error {
	_, err := r.db.NewUpdate().
		Model(plan).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update plan: %w", err)
	}

	return nil
}

// Delete soft-deletes a plan.
func (r *planRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionPlan)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	return nil
}

// FindByID retrieves a plan by ID.
func (r *planRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionPlan, error) {
	plan := new(schema.SubscriptionPlan)

	err := r.db.NewSelect().
		Model(plan).
		Relation("Features").
		Relation("Tiers").
		Relation("FeatureLinks").
		Relation("FeatureLinks.Feature").
		Where("sp.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find plan: %w", err)
	}

	return plan, nil
}

// FindBySlug retrieves a plan by slug within an app.
func (r *planRepository) FindBySlug(ctx context.Context, appID xid.ID, slug string) (*schema.SubscriptionPlan, error) {
	plan := new(schema.SubscriptionPlan)

	err := r.db.NewSelect().
		Model(plan).
		Relation("Features").
		Relation("Tiers").
		Relation("FeatureLinks").
		Relation("FeatureLinks.Feature").
		Where("sp.app_id = ?", appID).
		Where("sp.slug = ?", slug).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find plan by slug: %w", err)
	}

	return plan, nil
}

// FindByProviderID retrieves a plan by provider plan ID.
func (r *planRepository) FindByProviderID(ctx context.Context, providerPlanID string) (*schema.SubscriptionPlan, error) {
	plan := new(schema.SubscriptionPlan)

	err := r.db.NewSelect().
		Model(plan).
		Relation("Features").
		Relation("Tiers").
		Relation("FeatureLinks").
		Relation("FeatureLinks.Feature").
		Where("sp.provider_plan_id = ?", providerPlanID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find plan by provider ID: %w", err)
	}

	return plan, nil
}

// List retrieves plans with optional filters.
func (r *planRepository) List(ctx context.Context, filter *PlanFilter) ([]*schema.SubscriptionPlan, int, error) {
	var plans []*schema.SubscriptionPlan

	query := r.db.NewSelect().
		Model(&plans).
		Relation("Features").
		Relation("Tiers").
		Relation("FeatureLinks").
		Relation("FeatureLinks.Feature").
		Order("display_order ASC", "created_at DESC")

	if filter != nil {
		if filter.AppID != nil {
			query = query.Where("sp.app_id = ?", *filter.AppID)
		}

		if filter.IsActive != nil {
			query = query.Where("sp.is_active = ?", *filter.IsActive)
		}

		if filter.IsPublic != nil {
			query = query.Where("sp.is_public = ?", *filter.IsPublic)
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
		return nil, 0, fmt.Errorf("failed to list plans: %w", err)
	}

	return plans, count, nil
}

// CreateFeature creates a plan feature.
func (r *planRepository) CreateFeature(ctx context.Context, feature *schema.SubscriptionPlanFeature) error {
	_, err := r.db.NewInsert().Model(feature).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plan feature: %w", err)
	}

	return nil
}

// DeleteFeatures deletes all features for a plan.
func (r *planRepository) DeleteFeatures(ctx context.Context, planID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionPlanFeature)(nil)).
		Where("plan_id = ?", planID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete plan features: %w", err)
	}

	return nil
}

// CreateTier creates a pricing tier.
func (r *planRepository) CreateTier(ctx context.Context, tier *schema.SubscriptionPlanTier) error {
	_, err := r.db.NewInsert().Model(tier).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create plan tier: %w", err)
	}

	return nil
}

// DeleteTiers deletes all tiers for a plan.
func (r *planRepository) DeleteTiers(ctx context.Context, planID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionPlanTier)(nil)).
		Where("plan_id = ?", planID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete plan tiers: %w", err)
	}

	return nil
}
