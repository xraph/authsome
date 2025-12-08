package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// ExportImportService handles export/import of features and plans
type ExportImportService struct {
	featureRepo repository.FeatureRepository
	planRepo    repository.PlanRepository
	eventRepo   repository.EventRepository
}

// NewExportImportService creates a new export/import service
func NewExportImportService(
	featureRepo repository.FeatureRepository,
	planRepo repository.PlanRepository,
	eventRepo repository.EventRepository,
) *ExportImportService {
	return &ExportImportService{
		featureRepo: featureRepo,
		planRepo:    planRepo,
		eventRepo:   eventRepo,
	}
}

// ExportData represents the exported data structure
type ExportData struct {
	Version   string                 `json:"version"`
	ExportedAt time.Time             `json:"exportedAt"`
	AppID     string                 `json:"appId"`
	Features  []ExportFeature        `json:"features"`
	Plans     []ExportPlan           `json:"plans"`
}

// ExportFeature represents a feature in export format
type ExportFeature struct {
	Key          string                 `json:"key"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type"`
	Unit         string                 `json:"unit"`
	ResetPeriod  string                 `json:"resetPeriod"`
	IsPublic     bool                   `json:"isPublic"`
	DisplayOrder int                    `json:"displayOrder"`
	Icon         string                 `json:"icon,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Tiers        []ExportFeatureTier    `json:"tiers,omitempty"`
}

// ExportFeatureTier represents a feature tier in export format
type ExportFeatureTier struct {
	UpTo  int64  `json:"upTo"`
	Value string `json:"value"`
	Label string `json:"label"`
}

// ExportPlan represents a plan in export format
type ExportPlan struct {
	Name            string                 `json:"name"`
	Slug            string                 `json:"slug"`
	Description     string                 `json:"description"`
	BillingPattern  string                 `json:"billingPattern"`
	BillingInterval string                 `json:"billingInterval"`
	BasePrice       int64                  `json:"basePrice"`
	Currency        string                 `json:"currency"`
	TrialDays       int                    `json:"trialDays"`
	TierMode        string                 `json:"tierMode"`
	IsActive        bool                   `json:"isActive"`
	IsPublic        bool                   `json:"isPublic"`
	DisplayOrder    int                    `json:"displayOrder"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Features        []ExportPlanFeature    `json:"features,omitempty"`
	PriceTiers      []ExportPriceTier      `json:"priceTiers,omitempty"`
}

// ExportPlanFeature represents a plan feature link in export format
type ExportPlanFeature struct {
	FeatureKey    string      `json:"featureKey"`
	Value         interface{} `json:"value"`
	IsHighlighted bool        `json:"isHighlighted,omitempty"`
	IsBlocked     bool        `json:"isBlocked,omitempty"`
}

// ExportPriceTier represents a price tier in export format
type ExportPriceTier struct {
	UpTo       int64 `json:"upTo"`
	UnitAmount int64 `json:"unitAmount"`
	FlatAmount int64 `json:"flatAmount"`
}

// ExportFeaturesAndPlans exports all features and plans for an app
func (s *ExportImportService) ExportFeaturesAndPlans(ctx context.Context, appID xid.ID) (*ExportData, error) {
	export := &ExportData{
		Version:    "1.0",
		ExportedAt: time.Now(),
		AppID:      appID.String(),
		Features:   []ExportFeature{},
		Plans:      []ExportPlan{},
	}

	// Export features
	features, _, err := s.featureRepo.List(ctx, &repository.FeatureFilter{
		AppID:    &appID,
		Page:     1,
		PageSize: 1000, // Export all
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	for _, feature := range features {
		exportFeature := ExportFeature{
			Key:          feature.Key,
			Name:         feature.Name,
			Description:  feature.Description,
			Type:         feature.Type,
			Unit:         feature.Unit,
			ResetPeriod:  feature.ResetPeriod,
			IsPublic:     feature.IsPublic,
			DisplayOrder: feature.DisplayOrder,
			Icon:         feature.Icon,
			Metadata:     feature.Metadata,
			Tiers:        []ExportFeatureTier{},
		}

		// Export feature tiers
		for _, tier := range feature.Tiers {
			exportFeature.Tiers = append(exportFeature.Tiers, ExportFeatureTier{
				UpTo:  tier.UpTo,
				Value: tier.Value,
				Label: tier.Label,
			})
		}

		export.Features = append(export.Features, exportFeature)
	}

	// Export plans
	plans, _, err := s.planRepo.List(ctx, &repository.PlanFilter{
		AppID:    &appID,
		Page:     1,
		PageSize: 1000, // Export all
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	for _, plan := range plans {
		exportPlan := ExportPlan{
			Name:            plan.Name,
			Slug:            plan.Slug,
			Description:     plan.Description,
			BillingPattern:  plan.BillingPattern,
			BillingInterval: plan.BillingInterval,
			BasePrice:       plan.BasePrice,
			Currency:        plan.Currency,
			TrialDays:       plan.TrialDays,
			TierMode:        plan.TierMode,
			IsActive:        plan.IsActive,
			IsPublic:        plan.IsPublic,
			DisplayOrder:    plan.DisplayOrder,
			Metadata:        plan.Metadata,
			Features:        []ExportPlanFeature{},
			PriceTiers:      []ExportPriceTier{},
		}

		// Export plan features (legacy format)
		for _, feature := range plan.Features {
			var value interface{}
			json.Unmarshal([]byte(feature.Value), &value)
			exportPlan.Features = append(exportPlan.Features, ExportPlanFeature{
				FeatureKey: feature.Key,
				Value:      value,
			})
		}

		// Export new feature links (if present)
		for _, link := range plan.FeatureLinks {
			if link.Feature != nil {
				var value interface{}
				json.Unmarshal([]byte(link.Value), &value)
				
				// Check if already exported (prefer feature links)
				found := false
				for i, f := range exportPlan.Features {
					if f.FeatureKey == link.Feature.Key {
						exportPlan.Features[i] = ExportPlanFeature{
							FeatureKey:    link.Feature.Key,
							Value:         value,
							IsHighlighted: link.IsHighlighted,
							IsBlocked:     link.IsBlocked,
						}
						found = true
						break
					}
				}
				
				if !found {
					exportPlan.Features = append(exportPlan.Features, ExportPlanFeature{
						FeatureKey:    link.Feature.Key,
						Value:         value,
						IsHighlighted: link.IsHighlighted,
						IsBlocked:     link.IsBlocked,
					})
				}
			}
		}

		// Export price tiers
		for _, tier := range plan.Tiers {
			exportPlan.PriceTiers = append(exportPlan.PriceTiers, ExportPriceTier{
				UpTo:       tier.UpTo,
				UnitAmount: tier.UnitAmount,
				FlatAmount: tier.FlatAmount,
			})
		}

		export.Plans = append(export.Plans, exportPlan)
	}

	return export, nil
}

// ImportFeaturesAndPlans imports features and plans from export data
func (s *ExportImportService) ImportFeaturesAndPlans(ctx context.Context, appID xid.ID, data *ExportData, overwriteExisting bool) (*ImportResult, error) {
	result := &ImportResult{
		FeaturesCreated: 0,
		FeaturesSkipped: 0,
		PlansCreated:    0,
		PlansSkipped:    0,
		Errors:          []string{},
	}

	// Import features first
	featureKeyToID := make(map[string]xid.ID)
	for _, exportFeature := range data.Features {
		// Check if feature already exists
		existing, _ := s.featureRepo.FindByKey(ctx, appID, exportFeature.Key)
		
		if existing != nil {
			if !overwriteExisting {
				result.FeaturesSkipped++
				featureKeyToID[exportFeature.Key] = existing.ID
				continue
			}
			// Update existing feature
			existing.Name = exportFeature.Name
			existing.Description = exportFeature.Description
			existing.Unit = exportFeature.Unit
			existing.ResetPeriod = exportFeature.ResetPeriod
			existing.IsPublic = exportFeature.IsPublic
			existing.DisplayOrder = exportFeature.DisplayOrder
			existing.Icon = exportFeature.Icon
			existing.Metadata = exportFeature.Metadata
			existing.UpdatedAt = time.Now()
			
			if err := s.featureRepo.Update(ctx, existing); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to update feature %s: %v", exportFeature.Key, err))
				continue
			}
			
			// Update tiers
			if err := s.featureRepo.DeleteTiers(ctx, existing.ID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to delete tiers for feature %s: %v", exportFeature.Key, err))
			}
			
			for i, tier := range exportFeature.Tiers {
				newTier := &schema.FeatureTier{
					ID:        xid.New(),
					FeatureID: existing.ID,
					TierOrder: i,
					UpTo:      tier.UpTo,
					Value:     tier.Value,
					Label:     tier.Label,
					CreatedAt: time.Now(),
				}
				if err := s.featureRepo.CreateTier(ctx, newTier); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to create tier for feature %s: %v", exportFeature.Key, err))
				}
			}
			
			featureKeyToID[exportFeature.Key] = existing.ID
			result.FeaturesCreated++
		} else {
			// Create new feature
			now := time.Now()
			feature := &schema.Feature{
				ID:           xid.New(),
				AppID:        appID,
				Key:          exportFeature.Key,
				Name:         exportFeature.Name,
				Description:  exportFeature.Description,
				Type:         exportFeature.Type,
				Unit:         exportFeature.Unit,
				ResetPeriod:  exportFeature.ResetPeriod,
				IsPublic:     exportFeature.IsPublic,
				DisplayOrder: exportFeature.DisplayOrder,
				Icon:         exportFeature.Icon,
				Metadata:     exportFeature.Metadata,
			}
			feature.CreatedAt = now
			feature.UpdatedAt = now

			if feature.Metadata == nil {
				feature.Metadata = make(map[string]interface{})
			}

			if err := s.featureRepo.Create(ctx, feature); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to create feature %s: %v", exportFeature.Key, err))
				continue
			}

			// Create tiers
			for i, tier := range exportFeature.Tiers {
				newTier := &schema.FeatureTier{
					ID:        xid.New(),
					FeatureID: feature.ID,
					TierOrder: i,
					UpTo:      tier.UpTo,
					Value:     tier.Value,
					Label:     tier.Label,
					CreatedAt: now,
				}
				if err := s.featureRepo.CreateTier(ctx, newTier); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to create tier for feature %s: %v", exportFeature.Key, err))
				}
			}

			featureKeyToID[exportFeature.Key] = feature.ID
			result.FeaturesCreated++
		}
	}

	// Import plans
	for _, exportPlan := range data.Plans {
		// Check if plan already exists by slug
		existing, _ := s.planRepo.FindBySlug(ctx, appID, exportPlan.Slug)
		
		if existing != nil {
			if !overwriteExisting {
				result.PlansSkipped++
				continue
			}
			// Skip updating existing plans to avoid conflicts
			result.PlansSkipped++
			result.Errors = append(result.Errors, fmt.Sprintf("Plan with slug %s already exists, skipped", exportPlan.Slug))
			continue
		}

		// Create new plan
		now := time.Now()
		plan := &schema.SubscriptionPlan{
			ID:              xid.New(),
			AppID:           appID,
			Name:            exportPlan.Name,
			Slug:            exportPlan.Slug,
			Description:     exportPlan.Description,
			BillingPattern:  exportPlan.BillingPattern,
			BillingInterval: exportPlan.BillingInterval,
			BasePrice:       exportPlan.BasePrice,
			Currency:        exportPlan.Currency,
			TrialDays:       exportPlan.TrialDays,
			TierMode:        exportPlan.TierMode,
			IsActive:        exportPlan.IsActive,
			IsPublic:        exportPlan.IsPublic,
			DisplayOrder:    exportPlan.DisplayOrder,
			Metadata:        exportPlan.Metadata,
		}
		plan.CreatedAt = now
		plan.UpdatedAt = now

		if plan.Metadata == nil {
			plan.Metadata = make(map[string]interface{})
		}

		if err := s.planRepo.Create(ctx, plan); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create plan %s: %v", exportPlan.Slug, err))
			continue
		}

		// Create plan features (using new feature link system if features exist)
		for _, planFeature := range exportPlan.Features {
			featureID, exists := featureKeyToID[planFeature.FeatureKey]
			if !exists {
				result.Errors = append(result.Errors, fmt.Sprintf("Feature %s not found for plan %s", planFeature.FeatureKey, exportPlan.Slug))
				continue
			}

			valueJSON, _ := json.Marshal(planFeature.Value)
			link := &schema.PlanFeatureLink{
				ID:            xid.New(),
				PlanID:        plan.ID,
				FeatureID:     featureID,
				Value:         string(valueJSON),
				IsHighlighted: planFeature.IsHighlighted,
				IsBlocked:     planFeature.IsBlocked,
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			if err := s.featureRepo.LinkToPlan(ctx, link); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to link feature %s to plan %s: %v", planFeature.FeatureKey, exportPlan.Slug, err))
			}
		}

		// Create price tiers
		for i, tier := range exportPlan.PriceTiers {
			newTier := &schema.SubscriptionPlanTier{
				ID:         xid.New(),
				PlanID:     plan.ID,
				TierOrder:  i,
				UpTo:       tier.UpTo,
				UnitAmount: tier.UnitAmount,
				FlatAmount: tier.FlatAmount,
				CreatedAt:  now,
			}
			if err := s.planRepo.CreateTier(ctx, newTier); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to create tier for plan %s: %v", exportPlan.Slug, err))
			}
		}

		result.PlansCreated++
	}

	return result, nil
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	FeaturesCreated int      `json:"featuresCreated"`
	FeaturesSkipped int      `json:"featuresSkipped"`
	PlansCreated    int      `json:"plansCreated"`
	PlansSkipped    int      `json:"plansSkipped"`
	Errors          []string `json:"errors,omitempty"`
}

