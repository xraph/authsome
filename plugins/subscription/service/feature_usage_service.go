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

// FeatureUsageService handles feature usage tracking and enforcement
type FeatureUsageService struct {
	usageRepo   repository.FeatureUsageRepository
	featureRepo repository.FeatureRepository
	subRepo     repository.SubscriptionRepository
	planRepo    repository.PlanRepository
	eventRepo   repository.EventRepository
}

// NewFeatureUsageService creates a new feature usage service
func NewFeatureUsageService(
	usageRepo repository.FeatureUsageRepository,
	featureRepo repository.FeatureRepository,
	subRepo repository.SubscriptionRepository,
	planRepo repository.PlanRepository,
	eventRepo repository.EventRepository,
) *FeatureUsageService {
	return &FeatureUsageService{
		usageRepo:   usageRepo,
		featureRepo: featureRepo,
		subRepo:     subRepo,
		planRepo:    planRepo,
		eventRepo:   eventRepo,
	}
}

// ConsumeFeature consumes feature quota for an organization
func (s *FeatureUsageService) ConsumeFeature(ctx context.Context, req *core.ConsumeFeatureRequest) (*core.FeatureUsageResponse, error) {
	// Check idempotency
	if req.IdempotencyKey != "" {
		existing, _ := s.usageRepo.FindLogByIdempotencyKey(ctx, req.IdempotencyKey)
		if existing != nil {
			// Return current usage state
			return s.GetUsage(ctx, req.OrganizationID, req.FeatureKey)
		}
	}

	// Get subscription to find app and plan
	sub, err := s.subRepo.FindByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	// Get feature
	feature, err := s.featureRepo.FindByKey(ctx, sub.Plan.AppID, req.FeatureKey)
	if err != nil {
		return nil, suberrors.ErrFeatureNotFound
	}

	// Check if feature is consumable
	featureType := core.FeatureType(feature.Type)
	if !featureType.IsConsumable() {
		return nil, suberrors.ErrInvalidFeatureType
	}

	// Get effective limit
	limit, err := s.GetEffectiveLimit(ctx, req.OrganizationID, req.FeatureKey)
	if err != nil {
		return nil, err
	}

	// Check for unlimited
	if limit == -1 {
		// Unlimited - just log the consumption
		s.logUsage(ctx, req.OrganizationID, feature.ID, req.FeatureKey, string(core.FeatureUsageActionConsume), req.Quantity, 0, 0, nil, req.Reason, req.IdempotencyKey, req.Metadata)
		return &core.FeatureUsageResponse{
			FeatureKey:   req.FeatureKey,
			FeatureName:  feature.Name,
			FeatureType:  feature.Type,
			CurrentUsage: 0, // Not tracked for unlimited
			Limit:        -1,
			Remaining:    -1,
		}, nil
	}

	// Get or create usage record
	usage, err := s.getOrCreateUsage(ctx, req.OrganizationID, feature.ID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if err != nil {
		return nil, err
	}

	// Check if consumption would exceed limit
	if usage.CurrentUsage+req.Quantity > limit {
		return nil, suberrors.ErrInsufficientQuota
	}

	previousUsage := usage.CurrentUsage

	// Increment usage
	usage, err = s.usageRepo.IncrementUsage(ctx, req.OrganizationID, feature.ID, req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to increment usage: %w", err)
	}

	// Log the consumption
	s.logUsage(ctx, req.OrganizationID, feature.ID, req.FeatureKey, string(core.FeatureUsageActionConsume), req.Quantity, previousUsage, usage.CurrentUsage, nil, req.Reason, req.IdempotencyKey, req.Metadata)

	// Get granted extra
	grantedExtra, _ := s.usageRepo.GetTotalGrantedValue(ctx, req.OrganizationID, feature.ID)

	return &core.FeatureUsageResponse{
		FeatureKey:   req.FeatureKey,
		FeatureName:  feature.Name,
		FeatureType:  feature.Type,
		CurrentUsage: usage.CurrentUsage,
		Limit:        limit,
		Remaining:    limit - usage.CurrentUsage,
		PeriodStart:  usage.PeriodStart,
		PeriodEnd:    usage.PeriodEnd,
		GrantedExtra: grantedExtra,
	}, nil
}

// GrantFeature grants additional feature quota to an organization
func (s *FeatureUsageService) GrantFeature(ctx context.Context, req *core.GrantFeatureRequest) (*core.FeatureGrant, error) {
	// Validate grant type
	if !req.GrantType.IsValid() {
		return nil, suberrors.ErrInvalidFeatureType
	}

	// Get subscription to find app
	sub, err := s.subRepo.FindByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	// Get feature
	feature, err := s.featureRepo.FindByKey(ctx, sub.Plan.AppID, req.FeatureKey)
	if err != nil {
		return nil, suberrors.ErrFeatureNotFound
	}

	// Create grant
	now := time.Now()
	grant := &schema.FeatureGrant{
		ID:             xid.New(),
		OrganizationID: req.OrganizationID,
		FeatureID:      feature.ID,
		GrantType:      string(req.GrantType),
		Value:          req.Value,
		ExpiresAt:      req.ExpiresAt,
		SourceType:     req.SourceType,
		SourceID:       req.SourceID,
		Reason:         req.Reason,
		IsActive:       true,
	}
	grant.CreatedAt = now
	grant.UpdatedAt = now

	if req.Metadata != nil {
		grant.Metadata = req.Metadata
	} else {
		grant.Metadata = make(map[string]interface{})
	}

	if err := s.usageRepo.CreateGrant(ctx, grant); err != nil {
		return nil, fmt.Errorf("failed to create grant: %w", err)
	}

	// Log the grant
	s.logUsage(ctx, req.OrganizationID, feature.ID, req.FeatureKey, string(core.FeatureUsageActionGrant), req.Value, 0, 0, nil, req.Reason, "", req.Metadata)

	return &core.FeatureGrant{
		ID:             grant.ID,
		OrganizationID: grant.OrganizationID,
		FeatureID:      grant.FeatureID,
		FeatureKey:     req.FeatureKey,
		GrantType:      req.GrantType,
		Value:          grant.Value,
		ExpiresAt:      grant.ExpiresAt,
		SourceType:     grant.SourceType,
		SourceID:       grant.SourceID,
		Reason:         grant.Reason,
		IsActive:       grant.IsActive,
		Metadata:       grant.Metadata,
		CreatedAt:      grant.CreatedAt,
		UpdatedAt:      grant.UpdatedAt,
	}, nil
}

// RevokeGrant revokes a feature grant
func (s *FeatureUsageService) RevokeGrant(ctx context.Context, grantID xid.ID) error {
	grant, err := s.usageRepo.FindGrantByID(ctx, grantID)
	if err != nil {
		return suberrors.ErrFeatureGrantNotFound
	}

	grant.IsActive = false
	grant.UpdatedAt = time.Now()

	return s.usageRepo.UpdateGrant(ctx, grant)
}

// GetUsage retrieves current usage for a feature
func (s *FeatureUsageService) GetUsage(ctx context.Context, orgID xid.ID, featureKey string) (*core.FeatureUsageResponse, error) {
	// Get subscription
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	// Get feature
	feature, err := s.featureRepo.FindByKey(ctx, sub.Plan.AppID, featureKey)
	if err != nil {
		return nil, suberrors.ErrFeatureNotFound
	}

	// Get usage
	usage, _ := s.usageRepo.FindUsage(ctx, orgID, feature.ID)

	var currentUsage int64 = 0
	var periodStart, periodEnd time.Time
	if usage != nil {
		currentUsage = usage.CurrentUsage
		periodStart = usage.PeriodStart
		periodEnd = usage.PeriodEnd
	} else {
		periodStart = sub.CurrentPeriodStart
		periodEnd = sub.CurrentPeriodEnd
	}

	// Get effective limit
	limit, _ := s.GetEffectiveLimit(ctx, orgID, featureKey)

	// Get granted extra
	grantedExtra, _ := s.usageRepo.GetTotalGrantedValue(ctx, orgID, feature.ID)

	var remaining int64 = -1
	if limit != -1 {
		remaining = limit - currentUsage
		if remaining < 0 {
			remaining = 0
		}
	}

	return &core.FeatureUsageResponse{
		FeatureKey:   featureKey,
		FeatureName:  feature.Name,
		FeatureType:  feature.Type,
		CurrentUsage: currentUsage,
		Limit:        limit,
		Remaining:    remaining,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		GrantedExtra: grantedExtra,
	}, nil
}

// CheckAccess checks if an organization has access to a feature
func (s *FeatureUsageService) CheckAccess(ctx context.Context, orgID xid.ID, featureKey string) (*core.FeatureAccess, error) {
	// Get subscription
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return &core.FeatureAccess{HasAccess: false}, nil
	}

	// Check subscription status
	status := core.SubscriptionStatus(sub.Status)
	if !status.IsActiveOrTrialing() {
		return &core.FeatureAccess{HasAccess: false}, nil
	}

	// Get feature
	feature, err := s.featureRepo.FindByKey(ctx, sub.Plan.AppID, featureKey)
	if err != nil {
		return &core.FeatureAccess{HasAccess: false}, nil
	}

	// Get plan feature link
	link, err := s.featureRepo.GetPlanLink(ctx, sub.PlanID, feature.ID)
	if err != nil {
		// Feature not linked to plan - no access
		return &core.FeatureAccess{HasAccess: false}, nil
	}

	// Check if explicitly blocked
	if link.IsBlocked {
		return &core.FeatureAccess{
			Feature:   s.schemaFeatureToCore(feature),
			HasAccess: false,
			IsBlocked: true,
		}, nil
	}

	// Determine access based on feature type
	featureType := core.FeatureType(feature.Type)
	var hasAccess bool
	var limit int64 = 0

	switch featureType {
	case core.FeatureTypeBoolean:
		var val bool
		json.Unmarshal([]byte(link.Value), &val)
		hasAccess = val
	case core.FeatureTypeUnlimited:
		hasAccess = true
		limit = -1
	case core.FeatureTypeLimit, core.FeatureTypeMetered:
		var val float64
		json.Unmarshal([]byte(link.Value), &val)
		limit = int64(val)
		hasAccess = limit > 0
	case core.FeatureTypeTiered:
		hasAccess = true // Tiered features are always accessible, value determines tier
	}

	// Get current usage
	var currentUsage int64 = 0
	usage, _ := s.usageRepo.FindUsage(ctx, orgID, feature.ID)
	if usage != nil {
		currentUsage = usage.CurrentUsage
	}

	// Get granted extra
	grantedExtra, _ := s.usageRepo.GetTotalGrantedValue(ctx, orgID, feature.ID)

	// Adjust limit with grants
	effectiveLimit := limit
	if limit > 0 {
		effectiveLimit = limit + grantedExtra
	}

	var remaining int64 = -1
	if effectiveLimit > 0 {
		remaining = effectiveLimit - currentUsage
		if remaining < 0 {
			remaining = 0
		}
	}

	return &core.FeatureAccess{
		Feature:      s.schemaFeatureToCore(feature),
		HasAccess:    hasAccess,
		IsBlocked:    false,
		Limit:        effectiveLimit,
		CurrentUsage: currentUsage,
		Remaining:    remaining,
		GrantedExtra: grantedExtra,
		PlanValue:    link.Value,
	}, nil
}

// GetEffectiveLimit returns the total limit for a feature (plan limit + grants)
func (s *FeatureUsageService) GetEffectiveLimit(ctx context.Context, orgID xid.ID, featureKey string) (int64, error) {
	// Get subscription
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return 0, suberrors.ErrSubscriptionNotFound
	}

	// Get feature
	feature, err := s.featureRepo.FindByKey(ctx, sub.Plan.AppID, featureKey)
	if err != nil {
		return 0, suberrors.ErrFeatureNotFound
	}

	// Get plan feature link
	link, err := s.featureRepo.GetPlanLink(ctx, sub.PlanID, feature.ID)
	if err != nil {
		return 0, suberrors.ErrFeatureNotAvailable
	}

	// Check if blocked
	if link.IsBlocked {
		return 0, nil
	}

	// Determine base limit
	var baseLimit int64 = 0
	featureType := core.FeatureType(feature.Type)

	switch featureType {
	case core.FeatureTypeUnlimited:
		return -1, nil // Unlimited
	case core.FeatureTypeLimit, core.FeatureTypeMetered:
		var val float64
		json.Unmarshal([]byte(link.Value), &val)
		baseLimit = int64(val)
	case core.FeatureTypeBoolean:
		var val bool
		json.Unmarshal([]byte(link.Value), &val)
		if val {
			return -1, nil // Boolean true = unlimited access
		}
		return 0, nil // Boolean false = no access
	case core.FeatureTypeTiered:
		// For tiered, the limit might be in the value or infinite
		return -1, nil
	}

	// Add grants
	grantedExtra, _ := s.usageRepo.GetTotalGrantedValue(ctx, orgID, feature.ID)

	return baseLimit + grantedExtra, nil
}

// ResetUsage resets usage for a feature
func (s *FeatureUsageService) ResetUsage(ctx context.Context, orgID xid.ID, featureKey string, actorID *xid.ID, reason string) error {
	// Get subscription
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return suberrors.ErrSubscriptionNotFound
	}

	// Get feature
	feature, err := s.featureRepo.FindByKey(ctx, sub.Plan.AppID, featureKey)
	if err != nil {
		return suberrors.ErrFeatureNotFound
	}

	// Get current usage for logging
	usage, err := s.usageRepo.FindUsage(ctx, orgID, feature.ID)
	if err != nil {
		return nil // No usage to reset
	}

	previousUsage := usage.CurrentUsage

	// Reset usage
	if err := s.usageRepo.ResetUsage(ctx, orgID, feature.ID); err != nil {
		return fmt.Errorf("failed to reset usage: %w", err)
	}

	// Log the reset
	s.logUsage(ctx, orgID, feature.ID, featureKey, string(core.FeatureUsageActionReset), previousUsage, previousUsage, 0, actorID, reason, "", nil)

	return nil
}

// GetAllUsage retrieves all feature usage for an organization
func (s *FeatureUsageService) GetAllUsage(ctx context.Context, orgID xid.ID) ([]*core.FeatureUsageResponse, error) {
	usages, err := s.usageRepo.ListUsage(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list usage: %w", err)
	}

	result := make([]*core.FeatureUsageResponse, 0, len(usages))
	for _, u := range usages {
		if u.Feature == nil {
			continue
		}

		limit, _ := s.GetEffectiveLimit(ctx, orgID, u.Feature.Key)
		grantedExtra, _ := s.usageRepo.GetTotalGrantedValue(ctx, orgID, u.FeatureID)

		var remaining int64 = -1
		if limit != -1 {
			remaining = limit - u.CurrentUsage
			if remaining < 0 {
				remaining = 0
			}
		}

		result = append(result, &core.FeatureUsageResponse{
			FeatureKey:   u.Feature.Key,
			FeatureName:  u.Feature.Name,
			FeatureType:  u.Feature.Type,
			CurrentUsage: u.CurrentUsage,
			Limit:        limit,
			Remaining:    remaining,
			PeriodStart:  u.PeriodStart,
			PeriodEnd:    u.PeriodEnd,
			GrantedExtra: grantedExtra,
		})
	}

	return result, nil
}

// ListGrants lists all active grants for an organization
func (s *FeatureUsageService) ListGrants(ctx context.Context, orgID xid.ID) ([]*core.FeatureGrant, error) {
	grants, err := s.usageRepo.ListAllOrgGrants(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list grants: %w", err)
	}

	result := make([]*core.FeatureGrant, len(grants))
	for i, g := range grants {
		featureKey := ""
		if g.Feature != nil {
			featureKey = g.Feature.Key
		}
		result[i] = &core.FeatureGrant{
			ID:             g.ID,
			OrganizationID: g.OrganizationID,
			FeatureID:      g.FeatureID,
			FeatureKey:     featureKey,
			GrantType:      core.FeatureGrantType(g.GrantType),
			Value:          g.Value,
			ExpiresAt:      g.ExpiresAt,
			SourceType:     g.SourceType,
			SourceID:       g.SourceID,
			Reason:         g.Reason,
			IsActive:       g.IsActive,
			Metadata:       g.Metadata,
			CreatedAt:      g.CreatedAt,
			UpdatedAt:      g.UpdatedAt,
		}
	}

	return result, nil
}

// ProcessResets processes usage resets for features that need it
func (s *FeatureUsageService) ProcessResets(ctx context.Context) error {
	// Process each reset period
	periods := []string{
		string(core.ResetPeriodDaily),
		string(core.ResetPeriodWeekly),
		string(core.ResetPeriodMonthly),
		string(core.ResetPeriodYearly),
	}

	for _, period := range periods {
		usages, err := s.usageRepo.GetUsageNeedingReset(ctx, period)
		if err != nil {
			continue
		}

		for _, usage := range usages {
			// Reset and update period
			usage.CurrentUsage = 0
			usage.LastReset = time.Now()
			usage.PeriodStart = time.Now()
			usage.PeriodEnd = s.calculateNextPeriodEnd(core.ResetPeriod(period))
			s.usageRepo.UpdateUsage(ctx, usage)
		}
	}

	// Expire grants
	s.usageRepo.ExpireGrants(ctx)

	return nil
}

// Helper methods

func (s *FeatureUsageService) getOrCreateUsage(ctx context.Context, orgID, featureID xid.ID, periodStart, periodEnd time.Time) (*schema.OrganizationFeatureUsage, error) {
	usage, err := s.usageRepo.FindUsage(ctx, orgID, featureID)
	if err == nil {
		return usage, nil
	}

	// Create new usage record
	now := time.Now()
	usage = &schema.OrganizationFeatureUsage{
		ID:             xid.New(),
		OrganizationID: orgID,
		FeatureID:      featureID,
		CurrentUsage:   0,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		LastReset:      now,
		Metadata:       make(map[string]interface{}),
	}
	usage.CreatedAt = now
	usage.UpdatedAt = now

	if err := s.usageRepo.CreateUsage(ctx, usage); err != nil {
		return nil, fmt.Errorf("failed to create usage: %w", err)
	}

	return usage, nil
}

func (s *FeatureUsageService) logUsage(ctx context.Context, orgID, featureID xid.ID, featureKey, action string, quantity, previousUsage, newUsage int64, actorID *xid.ID, reason, idempotencyKey string, metadata map[string]any) {
	log := &schema.FeatureUsageLog{
		ID:             xid.New(),
		OrganizationID: orgID,
		FeatureID:      featureID,
		Action:         action,
		Quantity:       quantity,
		PreviousUsage:  previousUsage,
		NewUsage:       newUsage,
		ActorID:        actorID,
		Reason:         reason,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
	}

	if metadata != nil {
		log.Metadata = metadata
	} else {
		log.Metadata = make(map[string]interface{})
	}

	s.usageRepo.CreateLog(ctx, log)
}

func (s *FeatureUsageService) calculateNextPeriodEnd(period core.ResetPeriod) time.Time {
	now := time.Now()
	switch period {
	case core.ResetPeriodDaily:
		return now.AddDate(0, 0, 1)
	case core.ResetPeriodWeekly:
		return now.AddDate(0, 0, 7)
	case core.ResetPeriodMonthly:
		return now.AddDate(0, 1, 0)
	case core.ResetPeriodYearly:
		return now.AddDate(1, 0, 0)
	default:
		return now.AddDate(0, 1, 0) // Default to monthly
	}
}

func (s *FeatureUsageService) schemaFeatureToCore(f *schema.Feature) *core.Feature {
	if f == nil {
		return nil
	}

	var tiers []core.FeatureTier
	if len(f.Tiers) > 0 {
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
