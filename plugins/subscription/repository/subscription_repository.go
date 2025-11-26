package repository

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// subscriptionRepository implements SubscriptionRepository using Bun
type subscriptionRepository struct {
	db *bun.DB
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *bun.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

// Create creates a new subscription
func (r *subscriptionRepository) Create(ctx context.Context, sub *schema.Subscription) error {
	_, err := r.db.NewInsert().Model(sub).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

// Update updates an existing subscription
func (r *subscriptionRepository) Update(ctx context.Context, sub *schema.Subscription) error {
	_, err := r.db.NewUpdate().
		Model(sub).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}

// Delete soft-deletes a subscription
func (r *subscriptionRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Subscription)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}
	return nil
}

// FindByID retrieves a subscription by ID
func (r *subscriptionRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Subscription, error) {
	sub := new(schema.Subscription)
	err := r.db.NewSelect().
		Model(sub).
		Relation("Plan").
		Relation("Plan.Features").
		Relation("AddOns").
		Relation("AddOns.AddOn").
		Where("sub.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription: %w", err)
	}
	return sub, nil
}

// FindByOrganizationID retrieves the active subscription for an organization
func (r *subscriptionRepository) FindByOrganizationID(ctx context.Context, orgID xid.ID) (*schema.Subscription, error) {
	sub := new(schema.Subscription)
	err := r.db.NewSelect().
		Model(sub).
		Relation("Plan").
		Relation("Plan.Features").
		Relation("AddOns").
		Relation("AddOns.AddOn").
		Where("sub.organization_id = ?", orgID).
		Where("sub.status IN ('trialing', 'active', 'past_due')").
		Order("sub.created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription by organization: %w", err)
	}
	return sub, nil
}

// FindByProviderID retrieves a subscription by provider subscription ID
func (r *subscriptionRepository) FindByProviderID(ctx context.Context, providerSubID string) (*schema.Subscription, error) {
	sub := new(schema.Subscription)
	err := r.db.NewSelect().
		Model(sub).
		Relation("Plan").
		Relation("Plan.Features").
		Relation("AddOns").
		Relation("AddOns.AddOn").
		Where("sub.provider_sub_id = ?", providerSubID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription by provider ID: %w", err)
	}
	return sub, nil
}

// List retrieves subscriptions with optional filters
func (r *subscriptionRepository) List(ctx context.Context, filter *SubscriptionFilter) ([]*schema.Subscription, int, error) {
	var subs []*schema.Subscription
	
	query := r.db.NewSelect().
		Model(&subs).
		Relation("Plan").
		Relation("Organization").
		Order("sub.created_at DESC")
	
	if filter != nil {
		if filter.AppID != nil {
			query = query.
				Join("JOIN subscription_plans AS plan ON plan.id = sub.plan_id").
				Where("plan.app_id = ?", *filter.AppID)
		}
		if filter.OrganizationID != nil {
			query = query.Where("sub.organization_id = ?", *filter.OrganizationID)
		}
		if filter.PlanID != nil {
			query = query.Where("sub.plan_id = ?", *filter.PlanID)
		}
		if filter.Status != "" {
			query = query.Where("sub.status = ?", filter.Status)
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
		return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	
	return subs, count, nil
}

// CreateAddOnItem attaches an add-on to a subscription
func (r *subscriptionRepository) CreateAddOnItem(ctx context.Context, item *schema.SubscriptionAddOnItem) error {
	_, err := r.db.NewInsert().Model(item).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create add-on item: %w", err)
	}
	return nil
}

// DeleteAddOnItem removes an add-on from a subscription
func (r *subscriptionRepository) DeleteAddOnItem(ctx context.Context, subscriptionID, addOnID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionAddOnItem)(nil)).
		Where("subscription_id = ?", subscriptionID).
		Where("addon_id = ?", addOnID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete add-on item: %w", err)
	}
	return nil
}

// GetAddOnItems retrieves all add-ons for a subscription
func (r *subscriptionRepository) GetAddOnItems(ctx context.Context, subscriptionID xid.ID) ([]*schema.SubscriptionAddOnItem, error) {
	var items []*schema.SubscriptionAddOnItem
	err := r.db.NewSelect().
		Model(&items).
		Relation("AddOn").
		Where("sai.subscription_id = ?", subscriptionID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get add-on items: %w", err)
	}
	return items, nil
}

