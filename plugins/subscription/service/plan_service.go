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

// PlanService handles plan business logic
type PlanService struct {
	repo      repository.PlanRepository
	provider  providers.PaymentProvider
	eventRepo repository.EventRepository
}

// NewPlanService creates a new plan service
func NewPlanService(
	repo repository.PlanRepository,
	provider providers.PaymentProvider,
	eventRepo repository.EventRepository,
) *PlanService {
	return &PlanService{
		repo:      repo,
		provider:  provider,
		eventRepo: eventRepo,
	}
}

// Create creates a new plan
func (s *PlanService) Create(ctx context.Context, appID xid.ID, req *core.CreatePlanRequest) (*core.Plan, error) {
	// Validate billing pattern
	if !req.BillingPattern.IsValid() {
		return nil, suberrors.ErrInvalidBillingPattern
	}

	// Validate billing interval
	if !req.BillingInterval.IsValid() {
		return nil, suberrors.ErrInvalidBillingInterval
	}

	// Check for duplicate slug
	existing, _ := s.repo.FindBySlug(ctx, appID, req.Slug)
	if existing != nil {
		return nil, suberrors.ErrPlanAlreadyExists
	}

	// Create schema model
	now := time.Now()
	plan := &schema.SubscriptionPlan{
		ID:              xid.New(),
		AppID:           appID,
		Name:            req.Name,
		Slug:            req.Slug,
		Description:     req.Description,
		BillingPattern:  string(req.BillingPattern),
		BillingInterval: string(req.BillingInterval),
		BasePrice:       req.BasePrice,
		Currency:        req.Currency,
		TrialDays:       req.TrialDays,
		TierMode:        string(req.TierMode),
		IsActive:        req.IsActive,
		IsPublic:        req.IsPublic,
		DisplayOrder:    req.DisplayOrder,
	}
	plan.CreatedAt = now
	plan.UpdatedAt = now

	if req.Metadata != nil {
		plan.Metadata = req.Metadata
	} else {
		plan.Metadata = make(map[string]interface{})
	}

	if req.Currency == "" {
		plan.Currency = core.DefaultCurrency
	}

	if req.TierMode == "" {
		plan.TierMode = string(core.TierModeGraduated)
	}

	// Create plan
	if err := s.repo.Create(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Create features
	for _, f := range req.Features {
		valueJSON, _ := json.Marshal(f.Value)
		feature := &schema.SubscriptionPlanFeature{
			ID:          xid.New(),
			PlanID:      plan.ID,
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
		tier := &schema.SubscriptionPlanTier{
			ID:         xid.New(),
			PlanID:     plan.ID,
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

	return s.schemaToDomain(plan, req.Features, req.PriceTiers), nil
}

// Update updates an existing plan
func (s *PlanService) Update(ctx context.Context, id xid.ID, req *core.UpdatePlanRequest) (*core.Plan, error) {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrPlanNotFound
	}

	// Update fields
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Description != nil {
		plan.Description = *req.Description
	}
	if req.BasePrice != nil {
		plan.BasePrice = *req.BasePrice
	}
	if req.TrialDays != nil {
		plan.TrialDays = *req.TrialDays
	}
	if req.TierMode != nil {
		plan.TierMode = string(*req.TierMode)
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}
	if req.IsPublic != nil {
		plan.IsPublic = *req.IsPublic
	}
	if req.DisplayOrder != nil {
		plan.DisplayOrder = *req.DisplayOrder
	}
	if req.Metadata != nil {
		plan.Metadata = req.Metadata
	}

	plan.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	// Update features if provided
	if req.Features != nil {
		// Delete existing features
		if err := s.repo.DeleteFeatures(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to delete features: %w", err)
		}

		// Create new features
		for _, f := range req.Features {
			valueJSON, _ := json.Marshal(f.Value)
			feature := &schema.SubscriptionPlanFeature{
				ID:          xid.New(),
				PlanID:      plan.ID,
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
		// Delete existing tiers
		if err := s.repo.DeleteTiers(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to delete tiers: %w", err)
		}

		// Create new tiers
		for i, t := range req.PriceTiers {
			tier := &schema.SubscriptionPlanTier{
				ID:         xid.New(),
				PlanID:     plan.ID,
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

	// Reload plan with relations
	plan, err = s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to reload plan: %w", err)
	}

	return s.schemaToCorePlan(plan), nil
}

// Delete deletes a plan
func (s *PlanService) Delete(ctx context.Context, id xid.ID) error {
	// Check if plan has active subscriptions
	// This would require subscription repository check
	// For now, just delete
	
	if err := s.repo.DeleteFeatures(ctx, id); err != nil {
		return fmt.Errorf("failed to delete features: %w", err)
	}
	
	if err := s.repo.DeleteTiers(ctx, id); err != nil {
		return fmt.Errorf("failed to delete tiers: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	return nil
}

// GetByID retrieves a plan by ID
func (s *PlanService) GetByID(ctx context.Context, id xid.ID) (*core.Plan, error) {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrPlanNotFound
	}
	return s.schemaToCorePlan(plan), nil
}

// GetBySlug retrieves a plan by slug
func (s *PlanService) GetBySlug(ctx context.Context, appID xid.ID, slug string) (*core.Plan, error) {
	plan, err := s.repo.FindBySlug(ctx, appID, slug)
	if err != nil {
		return nil, suberrors.ErrPlanNotFound
	}
	return s.schemaToCorePlan(plan), nil
}

// List retrieves plans with filtering
func (s *PlanService) List(ctx context.Context, appID xid.ID, activeOnly, publicOnly bool, page, pageSize int) ([]*core.Plan, int, error) {
	filter := &repository.PlanFilter{
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

	plans, count, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list plans: %w", err)
	}

	result := make([]*core.Plan, len(plans))
	for i, p := range plans {
		result[i] = s.schemaToCorePlan(p)
	}

	return result, count, nil
}

// SetActive sets the active status of a plan
func (s *PlanService) SetActive(ctx context.Context, id xid.ID, active bool) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrPlanNotFound
	}

	plan.IsActive = active
	plan.UpdatedAt = time.Now()

	return s.repo.Update(ctx, plan)
}

// SetPublic sets the public visibility of a plan
func (s *PlanService) SetPublic(ctx context.Context, id xid.ID, public bool) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrPlanNotFound
	}

	plan.IsPublic = public
	plan.UpdatedAt = time.Now()

	return s.repo.Update(ctx, plan)
}

// SyncToProvider syncs the plan to the payment provider
func (s *PlanService) SyncToProvider(ctx context.Context, id xid.ID) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return suberrors.ErrPlanNotFound
	}

	corePlan := s.schemaToCorePlan(plan)

	// Sync to provider
	if err := s.provider.SyncPlan(ctx, corePlan); err != nil {
		return fmt.Errorf("failed to sync plan to provider: %w", err)
	}

	// Update provider IDs
	plan.ProviderPlanID = corePlan.ProviderPlanID
	plan.ProviderPriceID = corePlan.ProviderPriceID
	plan.UpdatedAt = time.Now()

	return s.repo.Update(ctx, plan)
}

// Helper methods

func (s *PlanService) schemaToDomain(plan *schema.SubscriptionPlan, features []core.PlanFeature, tiers []core.PriceTier) *core.Plan {
	return &core.Plan{
		ID:              plan.ID,
		AppID:           plan.AppID,
		Name:            plan.Name,
		Slug:            plan.Slug,
		Description:     plan.Description,
		BillingPattern:  core.BillingPattern(plan.BillingPattern),
		BillingInterval: core.BillingInterval(plan.BillingInterval),
		BasePrice:       plan.BasePrice,
		Currency:        plan.Currency,
		TrialDays:       plan.TrialDays,
		Features:        features,
		PriceTiers:      tiers,
		TierMode:        core.TierMode(plan.TierMode),
		Metadata:        plan.Metadata,
		IsActive:        plan.IsActive,
		IsPublic:        plan.IsPublic,
		DisplayOrder:    plan.DisplayOrder,
		ProviderPlanID:  plan.ProviderPlanID,
		ProviderPriceID: plan.ProviderPriceID,
		CreatedAt:       plan.CreatedAt,
		UpdatedAt:       plan.UpdatedAt,
	}
}

func (s *PlanService) schemaToCorePlan(plan *schema.SubscriptionPlan) *core.Plan {
	features := make([]core.PlanFeature, len(plan.Features))
	for i, f := range plan.Features {
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

	tiers := make([]core.PriceTier, len(plan.Tiers))
	for i, t := range plan.Tiers {
		tiers[i] = core.PriceTier{
			UpTo:       t.UpTo,
			UnitAmount: t.UnitAmount,
			FlatAmount: t.FlatAmount,
		}
	}

	return &core.Plan{
		ID:              plan.ID,
		AppID:           plan.AppID,
		Name:            plan.Name,
		Slug:            plan.Slug,
		Description:     plan.Description,
		BillingPattern:  core.BillingPattern(plan.BillingPattern),
		BillingInterval: core.BillingInterval(plan.BillingInterval),
		BasePrice:       plan.BasePrice,
		Currency:        plan.Currency,
		TrialDays:       plan.TrialDays,
		Features:        features,
		PriceTiers:      tiers,
		TierMode:        core.TierMode(plan.TierMode),
		Metadata:        plan.Metadata,
		IsActive:        plan.IsActive,
		IsPublic:        plan.IsPublic,
		DisplayOrder:    plan.DisplayOrder,
		ProviderPlanID:  plan.ProviderPlanID,
		ProviderPriceID: plan.ProviderPriceID,
		CreatedAt:       plan.CreatedAt,
		UpdatedAt:       plan.UpdatedAt,
	}
}

