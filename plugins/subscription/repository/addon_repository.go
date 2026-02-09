package repository

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// addOnRepository implements AddOnRepository using Bun.
type addOnRepository struct {
	db *bun.DB
}

// NewAddOnRepository creates a new add-on repository.
func NewAddOnRepository(db *bun.DB) AddOnRepository {
	return &addOnRepository{db: db}
}

// Create creates a new add-on.
func (r *addOnRepository) Create(ctx context.Context, addon *schema.SubscriptionAddOn) error {
	_, err := r.db.NewInsert().Model(addon).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create add-on: %w", err)
	}

	return nil
}

// Update updates an existing add-on.
func (r *addOnRepository) Update(ctx context.Context, addon *schema.SubscriptionAddOn) error {
	_, err := r.db.NewUpdate().
		Model(addon).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update add-on: %w", err)
	}

	return nil
}

// Delete soft-deletes an add-on.
func (r *addOnRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionAddOn)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete add-on: %w", err)
	}

	return nil
}

// FindByID retrieves an add-on by ID.
func (r *addOnRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionAddOn, error) {
	addon := new(schema.SubscriptionAddOn)

	err := r.db.NewSelect().
		Model(addon).
		Relation("Features").
		Relation("Tiers").
		Where("sao.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find add-on: %w", err)
	}

	return addon, nil
}

// FindBySlug retrieves an add-on by slug within an app.
func (r *addOnRepository) FindBySlug(ctx context.Context, appID xid.ID, slug string) (*schema.SubscriptionAddOn, error) {
	addon := new(schema.SubscriptionAddOn)

	err := r.db.NewSelect().
		Model(addon).
		Relation("Features").
		Relation("Tiers").
		Where("sao.app_id = ?", appID).
		Where("sao.slug = ?", slug).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find add-on by slug: %w", err)
	}

	return addon, nil
}

// List retrieves add-ons with optional filters.
func (r *addOnRepository) List(ctx context.Context, filter *AddOnFilter) ([]*schema.SubscriptionAddOn, int, error) {
	var addons []*schema.SubscriptionAddOn

	query := r.db.NewSelect().
		Model(&addons).
		Relation("Features").
		Relation("Tiers").
		Order("display_order ASC", "created_at DESC")

	if filter != nil {
		if filter.AppID != nil {
			query = query.Where("sao.app_id = ?", *filter.AppID)
		}

		if filter.IsActive != nil {
			query = query.Where("sao.is_active = ?", *filter.IsActive)
		}

		if filter.IsPublic != nil {
			query = query.Where("sao.is_public = ?", *filter.IsPublic)
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
		return nil, 0, fmt.Errorf("failed to list add-ons: %w", err)
	}

	return addons, count, nil
}

// CreateFeature creates an add-on feature.
func (r *addOnRepository) CreateFeature(ctx context.Context, feature *schema.SubscriptionAddOnFeature) error {
	_, err := r.db.NewInsert().Model(feature).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create add-on feature: %w", err)
	}

	return nil
}

// DeleteFeatures deletes all features for an add-on.
func (r *addOnRepository) DeleteFeatures(ctx context.Context, addOnID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionAddOnFeature)(nil)).
		Where("addon_id = ?", addOnID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete add-on features: %w", err)
	}

	return nil
}

// CreateTier creates a pricing tier.
func (r *addOnRepository) CreateTier(ctx context.Context, tier *schema.SubscriptionAddOnTier) error {
	_, err := r.db.NewInsert().Model(tier).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create add-on tier: %w", err)
	}

	return nil
}

// DeleteTiers deletes all tiers for an add-on.
func (r *addOnRepository) DeleteTiers(ctx context.Context, addOnID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionAddOnTier)(nil)).
		Where("addon_id = ?", addOnID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete add-on tiers: %w", err)
	}

	return nil
}
