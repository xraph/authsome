package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// PlanService handles plan business logic
type PlanService struct {
	repo          repository.PlanRepository
	provider      providers.PaymentProvider
	eventRepo     repository.EventRepository
	autoSyncPlans bool
}

// NewPlanService creates a new plan service
func NewPlanService(
	repo repository.PlanRepository,
	provider providers.PaymentProvider,
	eventRepo repository.EventRepository,
) *PlanService {
	return &PlanService{
		repo:          repo,
		provider:      provider,
		eventRepo:     eventRepo,
		autoSyncPlans: false,
	}
}

// SetAutoSyncPlans enables or disables automatic plan sync to provider
func (s *PlanService) SetAutoSyncPlans(enabled bool) {
	s.autoSyncPlans = enabled
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

	result := s.schemaToDomain(plan, req.Features, req.PriceTiers)

	// Auto-sync to provider if enabled
	if s.autoSyncPlans {
		if err := s.provider.SyncPlan(ctx, result); err != nil {
			// Log the error but don't fail the creation
			// The plan can be synced manually later
		} else {
			// Update provider IDs in database
			plan.ProviderPlanID = result.ProviderPlanID
			plan.ProviderPriceID = result.ProviderPriceID
			plan.UpdatedAt = time.Now()
			_ = s.repo.Update(ctx, plan)
		}
	}

	return result, nil
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

// SyncFromProvider syncs a single plan from the payment provider using the provider plan ID.
// If the plan exists locally, it updates the local record with data from the provider.
// If the plan doesn't exist locally but has AuthSome metadata, it creates a new local record.
func (s *PlanService) SyncFromProvider(ctx context.Context, providerPlanID string) (*core.Plan, error) {
	// Get product from provider
	product, err := s.provider.GetProduct(ctx, providerPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product from provider: %w", err)
	}

	// Get prices for this product
	prices, err := s.provider.ListPrices(ctx, providerPlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to list prices from provider: %w", err)
	}

	// Try to find existing plan by provider ID
	existingPlan, _ := s.repo.FindByProviderID(ctx, providerPlanID)

	if existingPlan != nil {
		// Update existing plan with data from provider
		return s.updatePlanFromProvider(ctx, existingPlan, product, prices)
	}

	// Check if this is an AuthSome-created product
	if product.Metadata["authsome"] != "true" {
		return nil, fmt.Errorf("product %s is not an AuthSome-created product (missing authsome metadata)", providerPlanID)
	}

	// Try to get app_id from metadata
	appIDStr := product.Metadata["app_id"]
	if appIDStr == "" {
		return nil, fmt.Errorf("product %s is missing app_id metadata", providerPlanID)
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app_id in product metadata: %w", err)
	}

	// Create new plan from provider data
	return s.createPlanFromProvider(ctx, appID, product, prices)
}

// SyncAllFromProvider syncs all plans from the payment provider for a given app.
// It fetches all products from the provider that have AuthSome metadata,
// creates new local records for products that don't exist locally,
// and updates existing local records with data from the provider.
func (s *PlanService) SyncAllFromProvider(ctx context.Context, appID xid.ID) ([]*core.Plan, error) {
	// List all products from provider
	products, err := s.provider.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list products from provider: %w", err)
	}

	var syncedPlans []*core.Plan

	for _, product := range products {
		// Skip non-AuthSome products
		if product.Metadata["authsome"] != "true" {
			continue
		}

		// Get prices for this product
		prices, err := s.provider.ListPrices(ctx, product.ID)
		if err != nil {
			// Log error but continue with other products
			continue
		}

		// Try to find existing plan by provider ID
		existingPlan, _ := s.repo.FindByProviderID(ctx, product.ID)

		var plan *core.Plan

		if existingPlan != nil {
			// Update existing plan
			plan, err = s.updatePlanFromProvider(ctx, existingPlan, product, prices)
			if err != nil {
				continue
			}
		} else {
			// Determine app ID: use metadata if available, otherwise use provided appID
			productAppIDStr := product.Metadata["app_id"]
			productAppID := appID
			if productAppIDStr != "" {
				if parsed, err := xid.FromString(productAppIDStr); err == nil {
					productAppID = parsed
				}
			}

			// Only sync if app ID matches or we're doing a full sync (appID provided)
			if productAppID != appID && !appID.IsNil() {
				// Skip products from different apps unless appID is nil (sync all)
				continue
			}

			// Create new plan
			plan, err = s.createPlanFromProvider(ctx, productAppID, product, prices)
			if err != nil {
				continue
			}
		}

		syncedPlans = append(syncedPlans, plan)
	}

	return syncedPlans, nil
}

// updatePlanFromProvider updates an existing plan with data from the provider
func (s *PlanService) updatePlanFromProvider(ctx context.Context, plan *schema.SubscriptionPlan, product *providers.ProviderProduct, prices []*providers.ProviderPrice) (*core.Plan, error) {
	// Update plan fields from product
	plan.Name = product.Name
	plan.Description = product.Description
	plan.IsActive = product.Active
	plan.UpdatedAt = time.Now()

	// Update from first active price
	for _, price := range prices {
		if price.Active {
			plan.BasePrice = price.UnitAmount
			plan.Currency = price.Currency
			plan.ProviderPriceID = price.ID

			// Update billing interval from price recurring info
			if price.Recurring != nil {
				switch price.Recurring.Interval {
				case "month":
					plan.BillingInterval = string(core.BillingIntervalMonthly)
				case "year":
					plan.BillingInterval = string(core.BillingIntervalYearly)
				case "week":
					plan.BillingInterval = "weekly"
				case "day":
					plan.BillingInterval = "daily"
				}
			} else {
				plan.BillingInterval = string(core.BillingIntervalOneTime)
			}
			break
		}
	}

	// Update billing pattern from metadata if available
	if billingPattern := product.Metadata["billing_pattern"]; billingPattern != "" {
		plan.BillingPattern = billingPattern
	}

	if err := s.repo.Update(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	return s.schemaToCorePlan(plan), nil
}

// createPlanFromProvider creates a new plan from provider product data
func (s *PlanService) createPlanFromProvider(ctx context.Context, appID xid.ID, product *providers.ProviderProduct, prices []*providers.ProviderPrice) (*core.Plan, error) {
	now := time.Now()

	// Determine slug from metadata or generate from name
	slug := product.Metadata["slug"]
	if slug == "" {
		slug = generateSlug(product.Name)
	}

	// Check if slug already exists, append provider ID to make unique
	existing, _ := s.repo.FindBySlug(ctx, appID, slug)
	if existing != nil {
		slug = slug + "-" + product.ID[len(product.ID)-8:]
	}

	// Determine plan ID from metadata or generate new
	var planID xid.ID
	if planIDStr := product.Metadata["plan_id"]; planIDStr != "" {
		if parsed, err := xid.FromString(planIDStr); err == nil {
			planID = parsed
		} else {
			planID = xid.New()
		}
	} else {
		planID = xid.New()
	}

	// Determine billing pattern from metadata
	billingPattern := product.Metadata["billing_pattern"]
	if billingPattern == "" {
		billingPattern = string(core.BillingPatternFlat)
	}

	// Default values
	billingInterval := string(core.BillingIntervalMonthly)
	var basePrice int64
	currency := core.DefaultCurrency
	var priceID string

	// Get price info from first active price
	for _, price := range prices {
		if price.Active {
			basePrice = price.UnitAmount
			currency = price.Currency
			priceID = price.ID

			if price.Recurring != nil {
				switch price.Recurring.Interval {
				case "month":
					billingInterval = string(core.BillingIntervalMonthly)
				case "year":
					billingInterval = string(core.BillingIntervalYearly)
				case "week":
					billingInterval = "weekly"
				case "day":
					billingInterval = "daily"
				}
			} else {
				billingInterval = string(core.BillingIntervalOneTime)
			}
			break
		}
	}

	// Override billing interval from metadata if available
	if metaBillingInterval := product.Metadata["billing_interval"]; metaBillingInterval != "" {
		billingInterval = metaBillingInterval
	}

	plan := &schema.SubscriptionPlan{
		ID:              planID,
		AppID:           appID,
		Name:            product.Name,
		Slug:            slug,
		Description:     product.Description,
		BillingPattern:  billingPattern,
		BillingInterval: billingInterval,
		BasePrice:       basePrice,
		Currency:        currency,
		TierMode:        string(core.TierModeGraduated),
		IsActive:        product.Active,
		IsPublic:        true,
		ProviderPlanID:  product.ID,
		ProviderPriceID: priceID,
		Metadata:        make(map[string]interface{}),
	}
	// Set timestamps via embedded AuditableModel
	plan.AuditableModel.CreatedAt = now
	plan.AuditableModel.UpdatedAt = now

	if err := s.repo.Create(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	return s.schemaToCorePlan(plan), nil
}

// generateSlug creates a URL-safe slug from a name
func generateSlug(name string) string {
	slug := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			slug += string(r)
		} else if r >= 'A' && r <= 'Z' {
			slug += string(r + 32) // lowercase
		} else if r == ' ' || r == '-' || r == '_' {
			if len(slug) > 0 && slug[len(slug)-1] != '-' {
				slug += "-"
			}
		}
	}
	// Trim trailing dash
	for len(slug) > 0 && slug[len(slug)-1] == '-' {
		slug = slug[:len(slug)-1]
	}
	if slug == "" {
		slug = "plan"
	}
	return slug
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
	// Convert legacy features first
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

	// Convert new feature links (if present)
	// These take precedence over legacy features with same key
	if len(plan.FeatureLinks) > 0 {
		linkFeatureMap := make(map[string]core.PlanFeature)
		for _, link := range plan.FeatureLinks {
			if link.Feature != nil && !link.IsBlocked {
				var value interface{}
				json.Unmarshal([]byte(link.Value), &value)
				linkFeatureMap[link.Feature.Key] = core.PlanFeature{
					Key:           link.Feature.Key,
					Name:          link.Feature.Name,
					Description:   link.Feature.Description,
					Type:          core.FeatureType(link.Feature.Type),
					Value:         value,
					Unit:          link.Feature.Unit,
					IsHighlighted: link.IsHighlighted,
				}
			}
		}

		// Merge: new features override legacy features
		existingKeys := make(map[string]bool)
		for i, f := range features {
			if newF, ok := linkFeatureMap[f.Key]; ok {
				features[i] = newF
				existingKeys[f.Key] = true
			}
		}

		// Add any new features not in legacy
		for key, f := range linkFeatureMap {
			if !existingKeys[key] {
				features = append(features, f)
			}
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
