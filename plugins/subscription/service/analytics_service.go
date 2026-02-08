package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// AnalyticsService handles analytics and metrics calculations
type AnalyticsService struct {
	analyticsRepo repository.AnalyticsRepository
	subRepo       repository.SubscriptionRepository
	planRepo      repository.PlanRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(
	analyticsRepo repository.AnalyticsRepository,
	subRepo repository.SubscriptionRepository,
	planRepo repository.PlanRepository,
) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		subRepo:       subRepo,
		planRepo:      planRepo,
	}
}

// GetDashboardMetrics calculates comprehensive dashboard metrics
func (s *AnalyticsService) GetDashboardMetrics(ctx context.Context, appID xid.ID, startDate, endDate time.Time, currency string) (*core.DashboardMetrics, error) {
	// Get all subscriptions for the app
	subs, _, err := s.subRepo.List(ctx, &repository.SubscriptionFilter{
		AppID: &appID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	var totalMRR int64
	var activeCount int
	var trialingCount int
	var newMRR int64
	var expansionMRR int64
	var churnedMRR int64

	// Track subscriptions at period start for churn calculation
	var activeAtStart int
	var churnedCount int

	for _, sub := range subs {
		// Skip non-matching currency
		if sub.Plan != nil && sub.Plan.Currency != currency {
			continue
		}

		// Calculate MRR for active subscriptions
		if sub.Status == "active" {
			activeCount++
			mrr := s.calculateSubMRR(sub)
			totalMRR += mrr

			// Check if created in period (new MRR)
			if sub.CreatedAt.After(startDate) && sub.CreatedAt.Before(endDate) {
				newMRR += mrr
			}
		}

		// Count trialing subscriptions
		if sub.Status == "trialing" {
			trialingCount++
		}

		// Track churn
		if sub.Status == "canceled" || sub.Status == "cancelled" {
			// If was active at start and canceled in period
			if sub.CreatedAt.Before(startDate) {
				activeAtStart++
			}
			if sub.CanceledAt != nil && sub.CanceledAt.After(startDate) && sub.CanceledAt.Before(endDate) {
				churnedCount++
				churnedMRR += s.calculateSubMRR(sub)
			}
		}
	}

	// Calculate ARR
	totalARR := totalMRR * 12

	// Calculate churn rate
	var churnRate float64
	if activeAtStart > 0 {
		churnRate = float64(churnedCount) / float64(activeAtStart) * 100
	}

	// Calculate MRR growth (comparing to previous period)
	prevStartDate := startDate.AddDate(0, 0, -int(endDate.Sub(startDate).Hours()/24))
	prevMetrics, _ := s.GetDashboardMetrics(ctx, appID, prevStartDate, startDate, currency)
	var mrrGrowth float64
	if prevMetrics != nil && prevMetrics.TotalMRR > 0 {
		mrrGrowth = float64(totalMRR-prevMetrics.TotalMRR) / float64(prevMetrics.TotalMRR) * 100
	}

	// Calculate subscription growth
	var subGrowth float64
	if prevMetrics != nil && prevMetrics.ActiveSubscriptions > 0 {
		subGrowth = float64(activeCount-prevMetrics.ActiveSubscriptions) / float64(prevMetrics.ActiveSubscriptions) * 100
	}

	return &core.DashboardMetrics{
		TotalMRR:              totalMRR,
		TotalARR:              totalARR,
		ActiveSubscriptions:   activeCount,
		TrialingSubscriptions: trialingCount,
		MRRGrowth:             mrrGrowth,
		SubscriptionGrowth:    subGrowth,
		ChurnRate:             churnRate,
		NewMRR:                newMRR,
		ExpansionMRR:          expansionMRR,
		ChurnedMRR:            churnedMRR,
		Currency:              currency,
		AsOf:                  time.Now(),
	}, nil
}

// GetMRRHistory returns MRR breakdown over time
func (s *AnalyticsService) GetMRRHistory(ctx context.Context, appID xid.ID, startDate, endDate time.Time, currency string) ([]*core.MRRBreakdown, error) {
	// Get stored MRR history from analytics repository
	history, err := s.analyticsRepo.GetMRRHistory(ctx, appID, startDate, endDate, currency)
	if err == nil && len(history) > 0 {
		return history, nil
	}

	// If no stored history, calculate it
	var breakdown []*core.MRRBreakdown

	// Calculate for each day in the range
	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		nextDay := d.AddDate(0, 0, 1)

		// Get subscriptions active on this day
		subs, _, err := s.subRepo.List(ctx, &repository.SubscriptionFilter{
			AppID:  &appID,
			Status: "active",
		})
		if err != nil {
			continue
		}

		var totalMRR int64
		var newMRR int64
		var churnedMRR int64

		for _, sub := range subs {
			if sub.Plan != nil && sub.Plan.Currency != currency {
				continue
			}

			mrr := s.calculateSubMRR(sub)

			// Total MRR for active subs on this day
			if sub.Status == "active" && sub.CreatedAt.Before(nextDay) {
				if sub.CanceledAt == nil || sub.CanceledAt.After(d) {
					totalMRR += mrr
				}
			}

			// New MRR created on this day
			if sub.CreatedAt.Year() == d.Year() && sub.CreatedAt.Month() == d.Month() && sub.CreatedAt.Day() == d.Day() {
				newMRR += mrr
			}

			// Churned MRR on this day
			if sub.CanceledAt != nil {
				if sub.CanceledAt.Year() == d.Year() && sub.CanceledAt.Month() == d.Month() && sub.CanceledAt.Day() == d.Day() {
					churnedMRR += mrr
				}
			}
		}

		breakdown = append(breakdown, &core.MRRBreakdown{
			Date:       d,
			Currency:   currency,
			TotalMRR:   totalMRR,
			NewMRR:     newMRR,
			ChurnedMRR: churnedMRR,
			NetNewMRR:  newMRR - churnedMRR,
		})
	}

	return breakdown, nil
}

// GetRevenueByOrg returns revenue breakdown by organization
func (s *AnalyticsService) GetRevenueByOrg(ctx context.Context, appID xid.ID, startDate, endDate time.Time) ([]*core.OrgRevenue, error) {
	// Get all active subscriptions for the app
	subs, _, err := s.subRepo.List(ctx, &repository.SubscriptionFilter{
		AppID: &appID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	orgRevenueMap := make(map[xid.ID]*core.OrgRevenue)

	for _, sub := range subs {
		// Only include active and trialing subscriptions
		if sub.Status != "active" && sub.Status != "trialing" {
			continue
		}

		mrr := s.calculateSubMRR(sub)
		arr := mrr * 12

		if existing, ok := orgRevenueMap[sub.OrganizationID]; ok {
			existing.MRR += mrr
			existing.ARR += arr
		} else {
			orgName := "Unknown"
			if sub.Organization != nil {
				orgName = sub.Organization.Name
			}

			planName := "Unknown"
			planID := xid.NilID()
			if sub.Plan != nil {
				planName = sub.Plan.Name
				planID = sub.Plan.ID
			}

			orgRevenueMap[sub.OrganizationID] = &core.OrgRevenue{
				OrgID:    sub.OrganizationID,
				OrgName:  orgName,
				MRR:      mrr,
				ARR:      arr,
				PlanID:   planID,
				PlanName: planName,
				Status:   sub.Status,
			}
		}
	}

	// Convert map to slice
	result := make([]*core.OrgRevenue, 0, len(orgRevenueMap))
	for _, rev := range orgRevenueMap {
		result = append(result, rev)
	}

	return result, nil
}

// GetChurnRate calculates churn rate for the period
func (s *AnalyticsService) GetChurnRate(ctx context.Context, appID xid.ID, startDate, endDate time.Time) (float64, error) {
	metrics, err := s.GetDashboardMetrics(ctx, appID, startDate, endDate, "USD")
	if err != nil {
		return 0, err
	}
	return metrics.ChurnRate, nil
}

// GetSubscriptionGrowth returns subscription growth data points
func (s *AnalyticsService) GetSubscriptionGrowth(ctx context.Context, appID xid.ID, startDate, endDate time.Time) ([]*core.GrowthPoint, error) {
	var growthPoints []*core.GrowthPoint

	// Calculate growth for each day
	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		nextDay := d.AddDate(0, 0, 1)

		// Get subscriptions created on this day
		newSubs, _, _ := s.subRepo.List(ctx, &repository.SubscriptionFilter{
			AppID: &appID,
		})

		var newCount int
		var churnedCount int
		var activeCount int

		for _, sub := range newSubs {
			// New subscriptions
			if sub.CreatedAt.Year() == d.Year() && sub.CreatedAt.Month() == d.Month() && sub.CreatedAt.Day() == d.Day() {
				newCount++
			}

			// Churned subscriptions
			if sub.CanceledAt != nil {
				if sub.CanceledAt.Year() == d.Year() && sub.CanceledAt.Month() == d.Month() && sub.CanceledAt.Day() == d.Day() {
					churnedCount++
				}
			}

			// Active on this day
			if sub.Status == "active" && sub.CreatedAt.Before(nextDay) {
				if sub.CanceledAt == nil || sub.CanceledAt.After(d) {
					activeCount++
				}
			}
		}

		growthPoints = append(growthPoints, &core.GrowthPoint{
			Date:        d,
			NewSubs:     newCount,
			ChurnedSubs: churnedCount,
			ActiveSubs:  activeCount,
			NetGrowth:   newCount - churnedCount,
		})
	}

	return growthPoints, nil
}

// calculateSubMRR calculates MRR for a single subscription
func (s *AnalyticsService) calculateSubMRR(sub *schema.Subscription) int64 {
	if sub.Plan == nil {
		return 0
	}

	baseAmount := sub.Plan.BasePrice
	quantity := sub.Quantity
	if quantity == 0 {
		quantity = 1
	}

	// Normalize to monthly
	switch sub.Plan.BillingInterval {
	case "yearly":
		return (baseAmount * int64(quantity)) / 12
	case "one_time":
		return 0 // One-time charges don't contribute to MRR
	default: // Monthly
		return baseAmount * int64(quantity)
	}
}
