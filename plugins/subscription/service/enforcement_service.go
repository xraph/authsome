package service

import (
	"context"
	"encoding/json"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/internal/errs"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"

	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// EnforcementService handles subscription limit enforcement.
type EnforcementService struct {
	subRepo          repository.SubscriptionRepository
	planRepo         repository.PlanRepository
	usageRepo        repository.UsageRepository
	featureRepo      repository.FeatureRepository
	featureUsageRepo repository.FeatureUsageRepository
	orgService       *organization.Service
	config           core.Config
}

// NewEnforcementService creates a new enforcement service.
func NewEnforcementService(
	subRepo repository.SubscriptionRepository,
	planRepo repository.PlanRepository,
	usageRepo repository.UsageRepository,
	orgService *organization.Service,
	config core.Config,
) *EnforcementService {
	return &EnforcementService{
		subRepo:    subRepo,
		planRepo:   planRepo,
		usageRepo:  usageRepo,
		orgService: orgService,
		config:     config,
	}
}

// SetFeatureRepositories sets the feature repositories for enhanced feature checking.
func (s *EnforcementService) SetFeatureRepositories(featureRepo repository.FeatureRepository, featureUsageRepo repository.FeatureUsageRepository) {
	s.featureRepo = featureRepo
	s.featureUsageRepo = featureUsageRepo
}

// CheckFeatureAccess checks if an organization has access to a feature.
func (s *EnforcementService) CheckFeatureAccess(ctx context.Context, orgID xid.ID, feature string) (bool, error) {
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return false, nil // No subscription = no access
	}

	if sub.Status != string(core.StatusActive) && sub.Status != string(core.StatusTrialing) {
		return false, nil
	}

	if sub.Plan == nil {
		return false, nil
	}

	// Check plan features
	for _, f := range sub.Plan.Features {
		if f.Key != feature {
			continue
		}

		switch f.Type {
		case "boolean":
			var val bool
			if err := json.Unmarshal([]byte(f.Value), &val); err != nil {
				return false, err
			}

			return val, nil
		case "unlimited":
			return true, nil
		case "limit":
			var val float64
			if err := json.Unmarshal([]byte(f.Value), &val); err != nil {
				return false, err
			}

			return val > 0, nil
		}
	}

	return false, nil
}

// GetRemainingSeats returns the number of available seats.
func (s *EnforcementService) GetRemainingSeats(ctx context.Context, orgID xid.ID) (int, error) {
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return 0, suberrors.ErrSubscriptionNotFound
	}

	maxSeats := s.getFeatureLimitInt(sub.Plan, core.FeatureKeyMaxMembers)
	if maxSeats == -1 {
		return -1, nil // Unlimited
	}

	// Get current member count
	currentMembers := 0

	if s.orgService != nil {
		filter := &organization.ListMembersFilter{
			OrganizationID: orgID,
		}
		filter.Limit = 1 // We only care about the total count

		response, err := s.orgService.ListMembers(ctx, filter)
		if err == nil && response != nil && response.Pagination != nil {
			currentMembers = int(response.Pagination.Total)
		}
	}

	remaining := max(int(maxSeats)-currentMembers, 0)

	return remaining, nil
}

// GetFeatureLimit returns the limit for a feature.
func (s *EnforcementService) GetFeatureLimit(ctx context.Context, orgID xid.ID, feature string) (int64, error) {
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return 0, suberrors.ErrSubscriptionNotFound
	}

	return s.getFeatureLimitInt(sub.Plan, feature), nil
}

// GetAllLimits returns all limits and current usage for an organization.
func (s *EnforcementService) GetAllLimits(ctx context.Context, orgID xid.ID) (map[string]*core.UsageLimit, error) {
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	if sub.Plan == nil {
		return nil, errs.BadRequest("plan not loaded")
	}

	result := make(map[string]*core.UsageLimit)

	for _, f := range sub.Plan.Features {
		if f.Type != "limit" && f.Type != "unlimited" {
			continue
		}

		limit := s.getFeatureLimitInt(sub.Plan, f.Key)

		// Get current usage for metered features
		var currentUsage int64 = 0

		summary, err := s.usageRepo.GetSummary(ctx, sub.ID, f.Key, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
		if err == nil && summary != nil {
			currentUsage = summary.TotalQuantity
		}

		result[f.Key] = core.NewUsageLimit(f.Key, currentUsage, limit)
	}

	// Add seat limit
	maxSeats := s.getFeatureLimitInt(sub.Plan, core.FeatureKeyMaxMembers)
	currentMembers := int64(0)

	if s.orgService != nil {
		filter := &organization.ListMembersFilter{
			OrganizationID: orgID,
		}
		filter.Limit = 1

		response, err := s.orgService.ListMembers(ctx, filter)
		if err == nil && response != nil && response.Pagination != nil {
			currentMembers = int64(response.Pagination.Total)
		}
	}

	result[core.FeatureKeyMaxMembers] = core.NewUsageLimit(core.FeatureKeyMaxMembers, currentMembers, maxSeats)

	return result, nil
}

// EnforceSubscriptionRequired is a hook to enforce subscription requirement.
func (s *EnforcementService) EnforceSubscriptionRequired(ctx context.Context, req any) error {
	if !s.config.RequireSubscription {
		return nil
	}

	// This is called before org creation - we need to check if the user has a subscription
	// In most cases, org creation is the first step before subscribing, so we may need to
	// allow it but require subscription before other operations

	// For now, allow org creation
	return nil
}

// EnforceSeatLimit is a hook to enforce seat limits when adding members.
func (s *EnforcementService) EnforceSeatLimit(ctx context.Context, orgIDStr string, userID xid.ID) error {
	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return nil // Invalid org ID, let other validation handle it
	}

	remaining, err := s.GetRemainingSeats(ctx, orgID)
	if err != nil {
		// No subscription - check if subscription is required
		if s.config.RequireSubscription {
			return suberrors.ErrSubscriptionRequired
		}

		return nil
	}

	if remaining == -1 {
		return nil // Unlimited
	}

	if remaining <= 0 {
		return suberrors.ErrSeatLimitExceeded
	}

	return nil
}

// EnforceTeamLimit enforces team creation limits.
func (s *EnforcementService) EnforceTeamLimit(ctx context.Context, orgID xid.ID) error {
	limit, err := s.GetFeatureLimit(ctx, orgID, core.FeatureKeyMaxTeams)
	if err != nil {
		if s.config.RequireSubscription {
			return suberrors.ErrSubscriptionRequired
		}

		return nil
	}

	if limit == -1 {
		return nil // Unlimited
	}

	// Get current team count
	if s.orgService != nil {
		filter := &organization.ListTeamsFilter{
			OrganizationID: orgID,
		}
		filter.Limit = 1 // We only need the total count

		response, err := s.orgService.ListTeams(ctx, filter)
		if err == nil && response != nil && response.Pagination != nil && int64(response.Pagination.Total) >= limit {
			return suberrors.ErrFeatureLimitExceeded
		}
	}

	return nil
}

// CheckFeatureAccessEnhanced checks feature access using the new feature system
// Falls back to legacy system if new system is not configured.
func (s *EnforcementService) CheckFeatureAccessEnhanced(ctx context.Context, orgID xid.ID, featureKey string) (*core.FeatureAccess, error) {
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

	// Try new feature system first if configured
	if s.featureRepo != nil && sub.Plan != nil {
		feature, err := s.featureRepo.FindByKey(ctx, sub.Plan.AppID, featureKey)
		if err == nil && feature != nil {
			link, err := s.featureRepo.GetPlanLink(ctx, sub.PlanID, feature.ID)
			if err == nil && link != nil {
				// Feature found in new system
				if link.IsBlocked {
					return &core.FeatureAccess{
						HasAccess: false,
						IsBlocked: true,
					}, nil
				}

				// Determine access based on feature type
				var (
					hasAccess bool
					limit     int64 = 0
				)

				switch feature.Type {
				case "boolean":
					var val bool
					if err := json.Unmarshal([]byte(link.Value), &val); err != nil {
						return nil, err
					}
					hasAccess = val
				case "unlimited":
					hasAccess = true
					limit = -1
				case "limit", "metered":
					var val float64
					if err := json.Unmarshal([]byte(link.Value), &val); err != nil {
						return nil, err
					}
					limit = int64(val)
					hasAccess = limit > 0
				case "tiered":
					hasAccess = true
				}

				// Get current usage
				var currentUsage int64 = 0

				if s.featureUsageRepo != nil {
					usage, err := s.featureUsageRepo.FindUsage(ctx, orgID, feature.ID)
					if err == nil && usage != nil {
						currentUsage = usage.CurrentUsage
					}
				}

				// Get granted extra
				var grantedExtra int64 = 0
				if s.featureUsageRepo != nil {
					grantedExtra, _ = s.featureUsageRepo.GetTotalGrantedValue(ctx, orgID, feature.ID)
				}

				effectiveLimit := limit
				if limit > 0 {
					effectiveLimit = limit + grantedExtra
				}

				var remaining int64 = -1
				if effectiveLimit > 0 {
					remaining = max(effectiveLimit-currentUsage, 0)
				}

				return &core.FeatureAccess{
					HasAccess:    hasAccess,
					IsBlocked:    false,
					Limit:        effectiveLimit,
					CurrentUsage: currentUsage,
					Remaining:    remaining,
					GrantedExtra: grantedExtra,
					PlanValue:    link.Value,
				}, nil
			}
		}
	}

	// Fall back to legacy system
	hasAccess, err := s.CheckFeatureAccess(ctx, orgID, featureKey)
	if err != nil {
		return &core.FeatureAccess{HasAccess: false}, err
	}

	limit := s.getFeatureLimitInt(sub.Plan, featureKey)

	return &core.FeatureAccess{
		HasAccess: hasAccess,
		Limit:     limit,
		Remaining: limit, // Legacy doesn't track usage
	}, nil
}

// GetEffectiveLimitEnhanced returns the effective limit for a feature including grants.
func (s *EnforcementService) GetEffectiveLimitEnhanced(ctx context.Context, orgID xid.ID, featureKey string) (int64, error) {
	// Get subscription
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return 0, suberrors.ErrSubscriptionNotFound
	}

	// Try new feature system first
	if s.featureRepo != nil && sub.Plan != nil {
		feature, err := s.featureRepo.FindByKey(ctx, sub.Plan.AppID, featureKey)
		if err == nil && feature != nil {
			link, err := s.featureRepo.GetPlanLink(ctx, sub.PlanID, feature.ID)
			if err == nil && link != nil {
				if link.IsBlocked {
					return 0, nil
				}

				var baseLimit int64 = 0

				switch feature.Type {
				case "unlimited":
					return -1, nil
				case "limit", "metered":
					var val float64
					if err := json.Unmarshal([]byte(link.Value), &val); err != nil {
						return 0, err
					}
					baseLimit = int64(val)
				case "boolean":
					var val bool
					if err := json.Unmarshal([]byte(link.Value), &val); err != nil {
						return 0, err
					}

					if val {
						return -1, nil
					}

					return 0, nil
				case "tiered":
					return -1, nil
				}

				// Add grants
				if s.featureUsageRepo != nil {
					grantedExtra, _ := s.featureUsageRepo.GetTotalGrantedValue(ctx, orgID, feature.ID)

					return baseLimit + grantedExtra, nil
				}

				return baseLimit, nil
			}
		}
	}

	// Fall back to legacy
	return s.getFeatureLimitInt(sub.Plan, featureKey), nil
}

// Helper methods

func (s *EnforcementService) getFeatureLimitInt(plan *schema.SubscriptionPlan, key string) int64 {
	if plan == nil {
		return 0
	}

	// First check new feature links if available
	if s.featureRepo != nil && len(plan.FeatureLinks) > 0 {
		for _, link := range plan.FeatureLinks {
			if link.Feature != nil && link.Feature.Key == key {
				if link.IsBlocked {
					return 0
				}

				switch link.Feature.Type {
				case "unlimited":
					return -1
				case "limit", "metered":
					var val float64
					if err := json.Unmarshal([]byte(link.Value), &val); err == nil {
						return int64(val)
					}
				}
			}
		}
	}

	// Fall back to legacy features
	for _, f := range plan.Features {
		if f.Key != key {
			continue
		}

		switch f.Type {
		case "unlimited":
			return -1
		case "limit":
			var val float64
			if err := json.Unmarshal([]byte(f.Value), &val); err == nil {
				return int64(val)
			}
		}
	}

	return 0
}
