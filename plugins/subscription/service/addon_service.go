package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// AddOnService handles add-on business logic
type AddOnService struct {
	repo      repository.AddOnRepository
	subRepo   repository.SubscriptionRepository
	provider  providers.PaymentProvider
	eventRepo repository.EventRepository
}

// NewAddOnService creates a new add-on service
func NewAddOnService(
	repo repository.AddOnRepository,
	subRepo repository.SubscriptionRepository,
	provider providers.PaymentProvider,
	eventRepo repository.EventRepository,
) *AddOnService {
	return &AddOnService{
		repo:      repo,
		subRepo:   subRepo,
		provider:  provider,
		eventRepo: eventRepo,
	}
}

// Create creates a new add-on
func (s *AddOnService) Create(ctx context.Context, appID xid.ID, req *core.CreateAddOnRequest) (*core.AddOn, error) {
	// Validate billing pattern
	if !req.BillingPattern.IsValid() {
		return nil, suberrors.ErrInvalidBillingPattern
	}

	// Check for duplicate slug
	existing, _ := s.repo.FindBySlug(ctx, appID, req.Slug)
	if existing != nil {
		return nil, suberrors.ErrAddOnAlreadyExists
	}

	// Create schema model
	now := time.Now()
	addon := &schema.SubscriptionAddOn{
		ID:              xid.New(),
		AppID:           appID,
		Name:            req.Name,
		Slug:            req.Slug,
		Description:     req.Description,
		BillingPattern:  string(req.BillingPattern),
		BillingInterval: string(req.BillingInterval),
		Price:           req.Price,
		Currency:        req.Currency,
		TierMode:        string(req.TierMode),
		IsActive:        req.IsActive,
		IsPublic:        req.IsPublic,
		DisplayOrder:    req.DisplayOrder,
		MaxQuantity:     req.MaxQuantity,
	}
	addon.CreatedAt = now
	addon.UpdatedAt = now

	if req.Metadata != nil {
		addon.Metadata = req.Metadata
	} else {
		addon.Metadata = make(map[string]interface{})
	}

	if req.Currency == "" {
		addon.Currency = core.DefaultCurrency
	}

	// Convert plan IDs to strings
	if len(req.RequiresPlanIDs) > 0 {
		ids := make([]string, len(req.RequiresPlanIDs))
		for i, id := range req.RequiresPlanIDs {
			ids[i] = id.String()
		}
		addon.RequiresPlanIDs = ids
	}

	if len(req.ExcludesPlanIDs) > 0 {
		ids := make([]string, len(req.ExcludesPlanIDs))
		for i, id := range req.ExcludesPlanIDs {
			ids[i] = id.String()
		}
		addon.ExcludesPlanIDs = ids
	}

	// Create add-on
	if err := s.repo.Create(ctx, addon); err != nil {
		return nil, fmt.Errorf("failed to create add-on: %w", err)
	}

	// Create features
	for _, f := range req.Features {
		valueJSON, _ := json.Marshal(f.Value)
		feature := &schema.SubscriptionAddOnFeature{
			ID:          xid.New(),
			AddOnID:     addon.ID,
			Key:         f.Key,
			Name:        f.Name,
			Description: f.Description,
			Type:        string(f.Type),
			Value:       string(valueJSON),
			CreatedAt:   now,
		}
		if err := s.repo.CreateFeature(ctx, feature); err != nil {
			return nil, fmt.Errorf("failed to create feature: %w", err)
		}
	}

	// Create price tiers
	for i, t := range req.PriceTiers {
		tier := &schema.SubscriptionAddOnTier{
			ID:         xid.New(),
			AddOnID:    addon.ID,
			TierOrder:  i,
			UpTo:       t.UpTo,
			UnitAmount: t.UnitAmount,
			FlatAmount: t.FlatAmount,
			CreatedAt:  now,
		}
		if err := s.repo.CreateTier(ctx, tier); err != nil {
			return nil, fmt.Errorf("failed to create tier: %w", err)
		}
	}

	return s.schemaToCore(addon, req.Features, req.PriceTiers), nil
}

// Update updates an existing add-on
func (s *AddOnService) Update(ctx context.Context, id xid.ID, req *core.UpdateAddOnRequest) (*core.AddOn, error) {
	addon, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrAddOnNotFound
	}

	// Update fields
	if req.Name != nil {
		addon.Name = *req.Name
	}
	if req.Description != nil {
		addon.Description = *req.Description
	}
	if req.Price != nil {
		addon.Price = *req.Price
	}
	if req.TierMode != nil {
		addon.TierMode = string(*req.TierMode)
	}
	if req.IsActive != nil {
		addon.IsActive = *req.IsActive
	}
	if req.IsPublic != nil {
		addon.IsPublic = *req.IsPublic
	}
	if req.DisplayOrder != nil {
		addon.DisplayOrder = *req.DisplayOrder
	}
	if req.MaxQuantity != nil {
		addon.MaxQuantity = *req.MaxQuantity
	}
	if req.Metadata != nil {
		addon.Metadata = req.Metadata
	}

	addon.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, addon); err != nil {
		return nil, fmt.Errorf("failed to update add-on: %w", err)
	}

	// Update features if provided
	if req.Features != nil {
		if err := s.repo.DeleteFeatures(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to delete features: %w", err)
		}

		for _, f := range req.Features {
			valueJSON, _ := json.Marshal(f.Value)
			feature := &schema.SubscriptionAddOnFeature{
				ID:          xid.New(),
				AddOnID:     addon.ID,
				Key:         f.Key,
				Name:        f.Name,
				Description: f.Description,
				Type:        string(f.Type),
				Value:       string(valueJSON),
				CreatedAt:   time.Now(),
			}
			if err := s.repo.CreateFeature(ctx, feature); err != nil {
				return nil, fmt.Errorf("failed to create feature: %w", err)
			}
		}
	}

	// Update tiers if provided
	if req.PriceTiers != nil {
		if err := s.repo.DeleteTiers(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to delete tiers: %w", err)
		}

		for i, t := range req.PriceTiers {
			tier := &schema.SubscriptionAddOnTier{
				ID:         xid.New(),
				AddOnID:    addon.ID,
				TierOrder:  i,
				UpTo:       t.UpTo,
				UnitAmount: t.UnitAmount,
				FlatAmount: t.FlatAmount,
				CreatedAt:  time.Now(),
			}
			if err := s.repo.CreateTier(ctx, tier); err != nil {
				return nil, fmt.Errorf("failed to create tier: %w", err)
			}
		}
	}

	// Reload
	addon, _ = s.repo.FindByID(ctx, id)
	return s.schemaToCoreAddOn(addon), nil
}

// Delete deletes an add-on
func (s *AddOnService) Delete(ctx context.Context, id xid.ID) error {
	if err := s.repo.DeleteFeatures(ctx, id); err != nil {
		return fmt.Errorf("failed to delete features: %w", err)
	}
	
	if err := s.repo.DeleteTiers(ctx, id); err != nil {
		return fmt.Errorf("failed to delete tiers: %w", err)
	}

	return s.repo.Delete(ctx, id)
}

// GetByID retrieves an add-on by ID
func (s *AddOnService) GetByID(ctx context.Context, id xid.ID) (*core.AddOn, error) {
	addon, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrAddOnNotFound
	}
	return s.schemaToCoreAddOn(addon), nil
}

// GetBySlug retrieves an add-on by slug
func (s *AddOnService) GetBySlug(ctx context.Context, appID xid.ID, slug string) (*core.AddOn, error) {
	addon, err := s.repo.FindBySlug(ctx, appID, slug)
	if err != nil {
		return nil, suberrors.ErrAddOnNotFound
	}
	return s.schemaToCoreAddOn(addon), nil
}

// List retrieves add-ons with filtering
func (s *AddOnService) List(ctx context.Context, appID xid.ID, activeOnly, publicOnly bool, page, pageSize int) ([]*core.AddOn, int, error) {
	filter := &repository.AddOnFilter{
		AppID:    &appID,
		Page:     page,
		PageSize: pageSize,
	}

	if activeOnly {
		active := true
		filter.IsActive = &active
	}
	if publicOnly {
		public := true
		filter.IsPublic = &public
	}

	addons, count, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list add-ons: %w", err)
	}

	result := make([]*core.AddOn, len(addons))
	for i, a := range addons {
		result[i] = s.schemaToCoreAddOn(a)
	}

	return result, count, nil
}

// GetAvailableForPlan retrieves add-ons available for a specific plan
func (s *AddOnService) GetAvailableForPlan(ctx context.Context, planID xid.ID) ([]*core.AddOn, error) {
	// Get all active add-ons
	active := true
	filter := &repository.AddOnFilter{
		IsActive: &active,
	}

	addons, _, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list add-ons: %w", err)
	}

	// Filter by plan availability
	result := make([]*core.AddOn, 0)
	for _, addon := range addons {
		coreAddon := s.schemaToCoreAddOn(addon)
		if coreAddon.IsAvailableForPlan(planID) {
			result = append(result, coreAddon)
		}
	}

	return result, nil
}

// Helper methods

func (s *AddOnService) schemaToCore(addon *schema.SubscriptionAddOn, features []core.PlanFeature, tiers []core.PriceTier) *core.AddOn {
	requiresIDs := make([]xid.ID, len(addon.RequiresPlanIDs))
	for i, idStr := range addon.RequiresPlanIDs {
		id, _ := xid.FromString(idStr)
		requiresIDs[i] = id
	}

	excludesIDs := make([]xid.ID, len(addon.ExcludesPlanIDs))
	for i, idStr := range addon.ExcludesPlanIDs {
		id, _ := xid.FromString(idStr)
		excludesIDs[i] = id
	}

	return &core.AddOn{
		ID:              addon.ID,
		AppID:           addon.AppID,
		Name:            addon.Name,
		Slug:            addon.Slug,
		Description:     addon.Description,
		BillingPattern:  core.BillingPattern(addon.BillingPattern),
		BillingInterval: core.BillingInterval(addon.BillingInterval),
		Price:           addon.Price,
		Currency:        addon.Currency,
		Features:        features,
		PriceTiers:      tiers,
		TierMode:        core.TierMode(addon.TierMode),
		Metadata:        addon.Metadata,
		IsActive:        addon.IsActive,
		IsPublic:        addon.IsPublic,
		DisplayOrder:    addon.DisplayOrder,
		RequiresPlanIDs: requiresIDs,
		ExcludesPlanIDs: excludesIDs,
		MaxQuantity:     addon.MaxQuantity,
		ProviderPriceID: addon.ProviderPriceID,
		CreatedAt:       addon.CreatedAt,
		UpdatedAt:       addon.UpdatedAt,
	}
}

func (s *AddOnService) schemaToCoreAddOn(addon *schema.SubscriptionAddOn) *core.AddOn {
	features := make([]core.PlanFeature, len(addon.Features))
	for i, f := range addon.Features {
		var value interface{}
		json.Unmarshal([]byte(f.Value), &value)
		features[i] = core.PlanFeature{
			Key:         f.Key,
			Name:        f.Name,
			Description: f.Description,
			Type:        core.FeatureType(f.Type),
			Value:       value,
		}
	}

	tiers := make([]core.PriceTier, len(addon.Tiers))
	for i, t := range addon.Tiers {
		tiers[i] = core.PriceTier{
			UpTo:       t.UpTo,
			UnitAmount: t.UnitAmount,
			FlatAmount: t.FlatAmount,
		}
	}

	return s.schemaToCore(addon, features, tiers)
}

