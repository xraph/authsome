package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// FeatureService handles feature business logic
type FeatureService struct {
	featureRepo repository.FeatureRepository
	planRepo    repository.PlanRepository
	eventRepo   repository.EventRepository
}

// NewFeatureService creates a new feature service
func NewFeatureService(
	featureRepo repository.FeatureRepository,
	planRepo repository.PlanRepository,
	eventRepo repository.EventRepository,
) *FeatureService {
	return &FeatureService{
		featureRepo: featureRepo,
		planRepo:    planRepo,
		eventRepo:   eventRepo,
	}
}

// Create creates a new feature
func (s *FeatureService) Create(ctx context.Context, appID xid.ID, req *core.CreateFeatureRequest) (*core.Feature, error) {
	// Validate feature type
	if !req.Type.IsValid() {
		return nil, suberrors.ErrInvalidFeatureType
	}

	// Validate reset period if provided
	if req.ResetPeriod != "" && !req.ResetPeriod.IsValid() {
		return nil, suberrors.ErrInvalidResetPeriod
	}

	// Check for duplicate key
	existing, _ := s.featureRepo.FindByKey(ctx, appID, req.Key)
	if existing != nil {
		return nil, suberrors.ErrFeatureAlreadyExists
	}

	// Set defaults
	resetPeriod := req.ResetPeriod
	if resetPeriod == "" {
		if req.Type.IsConsumable() {
			resetPeriod = core.ResetPeriodBillingCycle
		} else {
			resetPeriod = core.ResetPeriodNone
		}
	}

	// Create schema model
	now := time.Now()
	feature := &schema.Feature{
		ID:           xid.New(),
		AppID:        appID,
		Key:          req.Key,
		Name:         req.Name,
		Description:  req.Description,
		Type:         string(req.Type),
		Unit:         req.Unit,
		ResetPeriod:  string(resetPeriod),
		IsPublic:     req.IsPublic,
		DisplayOrder: req.DisplayOrder,
		Icon:         req.Icon,
	}
	feature.CreatedAt = now
	feature.UpdatedAt = now

	if req.Metadata != nil {
		feature.Metadata = req.Metadata
	} else {
		feature.Metadata = make(map[string]interface{})
	}

	// Create feature
	if err := s.featureRepo.Create(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to create feature: %w", err)
	}

	// Create tiers if provided
	for i, t := range req.Tiers {
		tier := &schema.FeatureTier{
			ID:        xid.New(),
			FeatureID: feature.ID,
			TierOrder: i,
			UpTo:      t.UpTo,
			Value:     t.Value,
			Label:     t.Label,
			CreatedAt: now,
		}
		if err := s.featureRepo.CreateTier(ctx, tier); err != nil {
			return nil, fmt.Errorf("failed to create feature tier: %w", err)
		}
	}

	return s.schemaToCore(feature, req.Tiers), nil
}

// Update updates an existing feature
func (s *FeatureService) Update(ctx context.Context, id xid.ID, req *core.UpdateFeatureRequest) (*core.Feature, error) {
	feature, err := s.featureRepo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrFeatureNotFound
	}

	// Update fields
	if req.Name != nil {
		feature.Name = *req.Name
	}
	if req.Description != nil {
		feature.Description = *req.Description
	}
	if req.Unit != nil {
		feature.Unit = *req.Unit
	}
	if req.ResetPeriod != nil {
		if !req.ResetPeriod.IsValid() {
			return nil, suberrors.ErrInvalidResetPeriod
		}
		feature.ResetPeriod = string(*req.ResetPeriod)
	}
	if req.IsPublic != nil {
		feature.IsPublic = *req.IsPublic
	}
	if req.DisplayOrder != nil {
		feature.DisplayOrder = *req.DisplayOrder
	}
	if req.Icon != nil {
		feature.Icon = *req.Icon
	}
	if req.Metadata != nil {
		feature.Metadata = req.Metadata
	}

	feature.UpdatedAt = time.Now()

	if err := s.featureRepo.Update(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to update feature: %w", err)
	}

	// Update tiers if provided
	if req.Tiers != nil {
		// Delete existing tiers
		if err := s.featureRepo.DeleteTiers(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to delete tiers: %w", err)
		}

		// Create new tiers
		for i, t := range req.Tiers {
			tier := &schema.FeatureTier{
				ID:        xid.New(),
				FeatureID: id,
				TierOrder: i,
				UpTo:      t.UpTo,
				Value:     t.Value,
				Label:     t.Label,
				CreatedAt: time.Now(),
			}
			if err := s.featureRepo.CreateTier(ctx, tier); err != nil {
				return nil, fmt.Errorf("failed to create tier: %w", err)
			}
		}
	}

	// Reload feature with relations
	feature, err = s.featureRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to reload feature: %w", err)
	}

	return s.schemaToCore(feature, nil), nil
}

// Delete deletes a feature
func (s *FeatureService) Delete(ctx context.Context, id xid.ID) error {
	// Check if feature is linked to any plans
	links, err := s.featureRepo.GetFeaturePlans(ctx, id)
	if err == nil && len(links) > 0 {
		return suberrors.ErrFeatureInUse
	}

	// Delete tiers first
	if err := s.featureRepo.DeleteTiers(ctx, id); err != nil {
		return fmt.Errorf("failed to delete tiers: %w", err)
	}

	if err := s.featureRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete feature: %w", err)
	}

	return nil
}

// GetByID retrieves a feature by ID
func (s *FeatureService) GetByID(ctx context.Context, id xid.ID) (*core.Feature, error) {
	feature, err := s.featureRepo.FindByID(ctx, id)
	if err != nil {
		return nil, suberrors.ErrFeatureNotFound
	}
	return s.schemaToCore(feature, nil), nil
}

// GetByKey retrieves a feature by key
func (s *FeatureService) GetByKey(ctx context.Context, appID xid.ID, key string) (*core.Feature, error) {
	feature, err := s.featureRepo.FindByKey(ctx, appID, key)
	if err != nil {
		return nil, suberrors.ErrFeatureNotFound
	}
	return s.schemaToCore(feature, nil), nil
}

// List retrieves features with filtering
func (s *FeatureService) List(ctx context.Context, appID xid.ID, featureType string, publicOnly bool, page, pageSize int) ([]*core.Feature, int, error) {
	filter := &repository.FeatureFilter{
		AppID:    &appID,
		Type:     featureType,
		Page:     page,
		PageSize: pageSize,
	}

	if publicOnly {
		public := true
		filter.IsPublic = &public
	}

	features, count, err := s.featureRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list features: %w", err)
	}

	result := make([]*core.Feature, len(features))
	for i, f := range features {
		result[i] = s.schemaToCore(f, nil)
	}

	return result, count, nil
}

// LinkToPlan links a feature to a plan
func (s *FeatureService) LinkToPlan(ctx context.Context, planID xid.ID, req *core.LinkFeatureRequest) (*core.PlanFeatureLink, error) {
	// Verify plan exists
	_, err := s.planRepo.FindByID(ctx, planID)
	if err != nil {
		return nil, suberrors.ErrPlanNotFound
	}

	// Verify feature exists
	feature, err := s.featureRepo.FindByID(ctx, req.FeatureID)
	if err != nil {
		return nil, suberrors.ErrFeatureNotFound
	}

	// Check if already linked
	existing, _ := s.featureRepo.GetPlanLink(ctx, planID, req.FeatureID)
	if existing != nil {
		return nil, suberrors.ErrFeatureAlreadyLinked
	}

	// Create link
	link := &schema.PlanFeatureLink{
		ID:               xid.New(),
		PlanID:           planID,
		FeatureID:        req.FeatureID,
		Value:            req.Value,
		IsBlocked:        req.IsBlocked,
		IsHighlighted:    req.IsHighlighted,
		OverrideSettings: req.OverrideSettings,
	}

	if link.OverrideSettings == nil {
		link.OverrideSettings = make(map[string]interface{})
	}

	if err := s.featureRepo.LinkToPlan(ctx, link); err != nil {
		return nil, fmt.Errorf("failed to link feature to plan: %w", err)
	}

	return &core.PlanFeatureLink{
		ID:               link.ID,
		PlanID:           link.PlanID,
		FeatureID:        link.FeatureID,
		Value:            link.Value,
		IsBlocked:        link.IsBlocked,
		IsHighlighted:    link.IsHighlighted,
		OverrideSettings: link.OverrideSettings,
		Feature:          s.schemaToCore(feature, nil),
	}, nil
}

// UpdatePlanLink updates a feature-plan link
func (s *FeatureService) UpdatePlanLink(ctx context.Context, planID, featureID xid.ID, req *core.UpdateLinkRequest) (*core.PlanFeatureLink, error) {
	link, err := s.featureRepo.GetPlanLink(ctx, planID, featureID)
	if err != nil {
		return nil, suberrors.ErrFeatureLinkNotFound
	}

	if req.Value != nil {
		link.Value = *req.Value
	}
	if req.IsBlocked != nil {
		link.IsBlocked = *req.IsBlocked
	}
	if req.IsHighlighted != nil {
		link.IsHighlighted = *req.IsHighlighted
	}
	if req.OverrideSettings != nil {
		link.OverrideSettings = req.OverrideSettings
	}

	if err := s.featureRepo.UpdatePlanLink(ctx, link); err != nil {
		return nil, fmt.Errorf("failed to update plan link: %w", err)
	}

	// Reload with feature
	link, _ = s.featureRepo.GetPlanLink(ctx, planID, featureID)

	return &core.PlanFeatureLink{
		ID:               link.ID,
		PlanID:           link.PlanID,
		FeatureID:        link.FeatureID,
		Value:            link.Value,
		IsBlocked:        link.IsBlocked,
		IsHighlighted:    link.IsHighlighted,
		OverrideSettings: link.OverrideSettings,
		Feature:          s.schemaToCore(link.Feature, nil),
	}, nil
}

// UnlinkFromPlan removes a feature from a plan
func (s *FeatureService) UnlinkFromPlan(ctx context.Context, planID, featureID xid.ID) error {
	return s.featureRepo.UnlinkFromPlan(ctx, planID, featureID)
}

// GetPlanFeatures retrieves all features linked to a plan
func (s *FeatureService) GetPlanFeatures(ctx context.Context, planID xid.ID) ([]*core.PlanFeatureLink, error) {
	links, err := s.featureRepo.GetPlanLinks(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan features: %w", err)
	}

	result := make([]*core.PlanFeatureLink, len(links))
	for i, l := range links {
		result[i] = &core.PlanFeatureLink{
			ID:               l.ID,
			PlanID:           l.PlanID,
			FeatureID:        l.FeatureID,
			Value:            l.Value,
			IsBlocked:        l.IsBlocked,
			IsHighlighted:    l.IsHighlighted,
			OverrideSettings: l.OverrideSettings,
			Feature:          s.schemaToCore(l.Feature, nil),
		}
	}

	return result, nil
}

// GetPublicFeatures retrieves public features for pricing pages
func (s *FeatureService) GetPublicFeatures(ctx context.Context, appID xid.ID) ([]*core.PublicFeature, error) {
	public := true
	filter := &repository.FeatureFilter{
		AppID:    &appID,
		IsPublic: &public,
	}

	features, _, err := s.featureRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list public features: %w", err)
	}

	result := make([]*core.PublicFeature, len(features))
	for i, f := range features {
		result[i] = &core.PublicFeature{
			Key:          f.Key,
			Name:         f.Name,
			Description:  f.Description,
			Type:         f.Type,
			Unit:         f.Unit,
			Icon:         f.Icon,
			DisplayOrder: f.DisplayOrder,
		}
	}

	return result, nil
}

// GetPublicPlanFeatures retrieves features for a plan formatted for public API
func (s *FeatureService) GetPublicPlanFeatures(ctx context.Context, planID xid.ID) ([]*core.PublicPlanFeature, error) {
	links, err := s.featureRepo.GetPlanLinks(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan features: %w", err)
	}

	result := make([]*core.PublicPlanFeature, 0, len(links))
	for _, l := range links {
		if l.Feature == nil || !l.Feature.IsPublic {
			continue
		}

		// Parse value based on feature type
		var value any
		switch l.Feature.Type {
		case "boolean":
			json.Unmarshal([]byte(l.Value), &value)
			if value == nil {
				value = !l.IsBlocked
			}
		case "limit", "metered":
			json.Unmarshal([]byte(l.Value), &value)
		case "unlimited":
			value = -1
		case "tiered":
			json.Unmarshal([]byte(l.Value), &value)
		}

		result = append(result, &core.PublicPlanFeature{
			Key:           l.Feature.Key,
			Name:          l.Feature.Name,
			Description:   l.Feature.Description,
			Type:          l.Feature.Type,
			Unit:          l.Feature.Unit,
			Value:         value,
			IsHighlighted: l.IsHighlighted,
			IsBlocked:     l.IsBlocked,
			DisplayOrder:  l.Feature.DisplayOrder,
		})
	}

	return result, nil
}

// Helper methods

func (s *FeatureService) schemaToCore(f *schema.Feature, inputTiers []core.FeatureTier) *core.Feature {
	if f == nil {
		return nil
	}

	var tiers []core.FeatureTier
	if inputTiers != nil {
		tiers = inputTiers
	} else if len(f.Tiers) > 0 {
		tiers = make([]core.FeatureTier, len(f.Tiers))
		for i, t := range f.Tiers {
			tiers[i] = core.FeatureTier{
				ID:        t.ID,
				FeatureID: t.FeatureID,
				TierOrder: t.TierOrder,
				UpTo:      t.UpTo,
				Value:     t.Value,
				Label:     t.Label,
			}
		}
	}

	return &core.Feature{
		ID:           f.ID,
		AppID:        f.AppID,
		Key:          f.Key,
		Name:         f.Name,
		Description:  f.Description,
		Type:         core.FeatureType(f.Type),
		Unit:         f.Unit,
		ResetPeriod:  core.ResetPeriod(f.ResetPeriod),
		IsPublic:     f.IsPublic,
		DisplayOrder: f.DisplayOrder,
		Icon:         f.Icon,
		Metadata:     f.Metadata,
		Tiers:        tiers,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}
