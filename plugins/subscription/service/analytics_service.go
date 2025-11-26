package service

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/repository"
)

// AnalyticsService handles billing analytics and metrics
type AnalyticsService struct {
	repo     repository.AnalyticsRepository
	subRepo  repository.SubscriptionRepository
	planRepo repository.PlanRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(repo repository.AnalyticsRepository, subRepo repository.SubscriptionRepository, planRepo repository.PlanRepository) *AnalyticsService {
	return &AnalyticsService{
		repo:     repo,
		subRepo:  subRepo,
		planRepo: planRepo,
	}
}

// GetDashboardMetrics returns dashboard metrics for an app
func (s *AnalyticsService) GetDashboardMetrics(ctx context.Context, appID xid.ID, currency string) (*core.DashboardMetrics, error) {
	if currency == "" {
		currency = core.CurrencyUSD
	}

	// Stub values - would need proper calculation
	return &core.DashboardMetrics{
		TotalMRR:              0,
		TotalARR:              0,
		ActiveSubscriptions:   0,
		TrialingSubscriptions: 0,
		MRRGrowth:             0,
		Currency:              currency,
		AsOf:                  time.Now(),
	}, nil
}

// GetMRRBreakdown returns MRR breakdown for a date range
func (s *AnalyticsService) GetMRRBreakdown(ctx context.Context, appID xid.ID, startDate, endDate time.Time, currency string) ([]*core.MRRBreakdown, error) {
	return s.repo.GetMRRHistory(ctx, appID, startDate, endDate, currency)
}

// GetChurnAnalysis returns churn analysis for a period
func (s *AnalyticsService) GetChurnAnalysis(ctx context.Context, appID xid.ID, period core.MetricPeriod, startDate, endDate time.Time, currency string) (*core.ChurnAnalysis, error) {
	summary, err := s.repo.GetMovementSummary(ctx, appID, startDate, endDate, currency)
	if err != nil {
		return nil, err
	}

	analysis := &core.ChurnAnalysis{
		Period:            period,
		StartDate:         startDate,
		EndDate:           endDate,
		CustomersAtStart:  0,
		CustomersAtEnd:    0,
		CustomersChurned:  summary.ChurnCount,
		CustomerChurnRate: 0,
		MRRAtStart:        0,
		MRRAtEnd:          0,
		MRRChurned:        summary.ChurnedMRR,
		Currency:          currency,
		ChurnByReason:     make(map[string]int),
		ChurnByPlan:       make(map[string]int),
	}

	return analysis, nil
}

// GetCohortAnalysis returns cohort retention analysis
func (s *AnalyticsService) GetCohortAnalysis(ctx context.Context, appID xid.ID, req *core.GetCohortAnalysisRequest) (*core.CohortAnalysisResponse, error) {
	cohorts, err := s.repo.ListCohorts(ctx, appID, req.StartMonth, req.NumMonths)
	if err != nil {
		return nil, err
	}

	// Convert from slice of pointers to slice of values
	cohortValues := make([]core.CohortAnalysis, len(cohorts))
	for i, c := range cohorts {
		cohortValues[i] = *c
	}

	return &core.CohortAnalysisResponse{
		Cohorts:  cohortValues,
		Currency: req.Currency,
	}, nil
}

// GetRevenueByPlan returns revenue breakdown by plan
func (s *AnalyticsService) GetRevenueByPlan(ctx context.Context, appID xid.ID, currency string) ([]*core.RevenueByPlan, error) {
	filter := &repository.PlanFilter{
		AppID: &appID,
	}
	plans, _, err := s.planRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]*core.RevenueByPlan, 0)

	for _, plan := range plans {
		result = append(result, &core.RevenueByPlan{
			PlanID:            plan.ID,
			PlanName:          plan.Name,
			PlanSlug:          plan.Slug,
			ActiveSubscribers: 0,
			MRR:               0,
			ARR:               0,
			PercentOfTotal:    0,
			Currency:          currency,
		})
	}

	return result, nil
}

// GetTrialMetrics returns trial conversion metrics
func (s *AnalyticsService) GetTrialMetrics(ctx context.Context, appID xid.ID, period core.MetricPeriod, startDate, endDate time.Time) (*core.TrialMetrics, error) {
	return &core.TrialMetrics{
		Period:          period,
		TrialsStarted:   0,
		TrialsConverted: 0,
		TrialsExpired:   0,
		TrialsActive:    0,
		ConversionRate:  0,
	}, nil
}

// RecordMovement records a subscription movement event
func (s *AnalyticsService) RecordMovement(ctx context.Context, movement *core.SubscriptionMovement) error {
	if movement.ID.IsNil() {
		movement.ID = xid.New()
	}
	movement.CreatedAt = time.Now()
	if movement.OccurredAt.IsZero() {
		movement.OccurredAt = time.Now()
	}

	return s.repo.CreateMovement(ctx, movement)
}

// GetMetrics returns billing metrics for a date range
func (s *AnalyticsService) GetMetrics(ctx context.Context, appID xid.ID, req *core.GetMetricsRequest) (*core.MetricsResponse, error) {
	metrics, err := s.repo.ListMetrics(ctx, appID, req.MetricTypes, req.Period, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	// Convert from slice of pointers to slice of values
	metricValues := make([]core.BillingMetric, len(metrics))
	for i, m := range metrics {
		metricValues[i] = *m
	}

	return &core.MetricsResponse{
		Metrics:   metricValues,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Currency:  req.Currency,
	}, nil
}

// CalculateAndStoreDailyMetrics calculates and stores daily metrics
// This would typically be called by a scheduled job
func (s *AnalyticsService) CalculateAndStoreDailyMetrics(ctx context.Context, appID xid.ID, date time.Time, currency string) error {
	// Stub - would calculate actual metrics
	mrrMetric := &core.BillingMetric{
		ID:        xid.New(),
		AppID:     appID,
		Type:      core.MetricTypeMRR,
		Period:    core.MetricPeriodDaily,
		Value:     0,
		Currency:  currency,
		Date:      date.Truncate(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	return s.repo.UpsertMetric(ctx, mrrMetric)
}

// UpdateCohortData updates cohort retention data
// This would typically be called by a scheduled job
func (s *AnalyticsService) UpdateCohortData(ctx context.Context, appID xid.ID, currency string) error {
	// Stub implementation
	return nil
}
