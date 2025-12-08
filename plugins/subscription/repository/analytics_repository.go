package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// AnalyticsRepository defines the interface for analytics operations
type AnalyticsRepository interface {
	// Billing metric operations
	CreateMetric(ctx context.Context, metric *core.BillingMetric) error
	GetMetric(ctx context.Context, appID xid.ID, metricType core.MetricType, date time.Time, period core.MetricPeriod) (*core.BillingMetric, error)
	ListMetrics(ctx context.Context, appID xid.ID, metricTypes []core.MetricType, period core.MetricPeriod, startDate, endDate time.Time) ([]*core.BillingMetric, error)
	UpsertMetric(ctx context.Context, metric *core.BillingMetric) error

	// Subscription movement operations
	CreateMovement(ctx context.Context, movement *core.SubscriptionMovement) error
	ListMovements(ctx context.Context, appID xid.ID, startDate, endDate time.Time, movementType string, page, pageSize int) ([]*core.SubscriptionMovement, int, error)
	GetMovementSummary(ctx context.Context, appID xid.ID, startDate, endDate time.Time, currency string) (*MovementSummary, error)

	// Cohort operations
	CreateCohort(ctx context.Context, cohort *core.CohortAnalysis) error
	GetCohort(ctx context.Context, appID xid.ID, cohortMonth time.Time) (*core.CohortAnalysis, error)
	ListCohorts(ctx context.Context, appID xid.ID, startMonth time.Time, numMonths int) ([]*core.CohortAnalysis, error)
	UpsertCohort(ctx context.Context, appID xid.ID, cohortMonth time.Time, monthNumber int, activeCustomers int, revenue int64, currency string) error

	// Dashboard metrics
	GetDashboardMetrics(ctx context.Context, appID xid.ID, currency string) (*core.DashboardMetrics, error)
	GetMRRHistory(ctx context.Context, appID xid.ID, startDate, endDate time.Time, currency string) ([]*core.MRRBreakdown, error)
}

// MovementSummary represents aggregated movement data
type MovementSummary struct {
	NewMRR          int64 `json:"newMrr"`
	ExpansionMRR    int64 `json:"expansionMrr"`
	ContractionMRR  int64 `json:"contractionMrr"`
	ChurnedMRR      int64 `json:"churnedMrr"`
	ReactivationMRR int64 `json:"reactivationMrr"`
	NewCount        int   `json:"newCount"`
	UpgradeCount    int   `json:"upgradeCount"`
	DowngradeCount  int   `json:"downgradeCount"`
	ChurnCount      int   `json:"churnCount"`
	ReactivateCount int   `json:"reactivateCount"`
}

// analyticsRepository implements AnalyticsRepository using Bun
type analyticsRepository struct {
	db *bun.DB
}

// NewAnalyticsRepository creates a new analytics repository
func NewAnalyticsRepository(db *bun.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

// CreateMetric creates a new billing metric
func (r *analyticsRepository) CreateMetric(ctx context.Context, metric *core.BillingMetric) error {
	model := billingMetricToSchema(metric)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// GetMetric returns a specific metric
func (r *analyticsRepository) GetMetric(ctx context.Context, appID xid.ID, metricType core.MetricType, date time.Time, period core.MetricPeriod) (*core.BillingMetric, error) {
	var metric schema.SubscriptionBillingMetric
	err := r.db.NewSelect().
		Model(&metric).
		Where("app_id = ?", appID).
		Where("type = ?", string(metricType)).
		Where("period = ?", string(period)).
		Where("date = ?", date.Truncate(24*time.Hour)).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToBillingMetric(&metric), nil
}

// ListMetrics returns metrics for a date range
func (r *analyticsRepository) ListMetrics(ctx context.Context, appID xid.ID, metricTypes []core.MetricType, period core.MetricPeriod, startDate, endDate time.Time) ([]*core.BillingMetric, error) {
	var metrics []schema.SubscriptionBillingMetric
	query := r.db.NewSelect().
		Model(&metrics).
		Where("app_id = ?", appID).
		Where("period = ?", string(period)).
		Where("date >= ?", startDate).
		Where("date <= ?", endDate)

	if len(metricTypes) > 0 {
		types := make([]string, len(metricTypes))
		for i, t := range metricTypes {
			types[i] = string(t)
		}
		query = query.Where("type IN (?)", bun.In(types))
	}

	err := query.Order("date ASC", "type ASC").Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.BillingMetric, len(metrics))
	for i, m := range metrics {
		result[i] = schemaToBillingMetric(&m)
	}
	return result, nil
}

// UpsertMetric creates or updates a billing metric
func (r *analyticsRepository) UpsertMetric(ctx context.Context, metric *core.BillingMetric) error {
	model := billingMetricToSchema(metric)
	_, err := r.db.NewInsert().
		Model(model).
		On("CONFLICT (app_id, type, period, date) DO UPDATE").
		Set("value = EXCLUDED.value").
		Exec(ctx)
	return err
}

// CreateMovement creates a subscription movement record
func (r *analyticsRepository) CreateMovement(ctx context.Context, movement *core.SubscriptionMovement) error {
	model := movementToSchema(movement)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// ListMovements returns subscription movements
func (r *analyticsRepository) ListMovements(ctx context.Context, appID xid.ID, startDate, endDate time.Time, movementType string, page, pageSize int) ([]*core.SubscriptionMovement, int, error) {
	var movements []schema.SubscriptionMovement
	query := r.db.NewSelect().
		Model(&movements).
		Where("app_id = ?", appID).
		Where("occurred_at >= ?", startDate).
		Where("occurred_at <= ?", endDate)

	if movementType != "" {
		query = query.Where("movement_type = ?", movementType)
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	err = query.
		Order("occurred_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*core.SubscriptionMovement, len(movements))
	for i, m := range movements {
		result[i] = schemaToMovement(&m)
	}
	return result, count, nil
}

// GetMovementSummary returns aggregated movement data
func (r *analyticsRepository) GetMovementSummary(ctx context.Context, appID xid.ID, startDate, endDate time.Time, currency string) (*MovementSummary, error) {
	summary := &MovementSummary{}

	// New subscriptions
	var newResult struct {
		Count    int   `bun:"count"`
		TotalMRR int64 `bun:"total_mrr"`
	}
	err := r.db.NewSelect().
		Model((*schema.SubscriptionMovement)(nil)).
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("COALESCE(SUM(mrr_change), 0) AS total_mrr").
		Where("app_id = ?", appID).
		Where("movement_type = ?", "new").
		Where("currency = ?", currency).
		Where("occurred_at >= ?", startDate).
		Where("occurred_at <= ?", endDate).
		Scan(ctx, &newResult)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	summary.NewCount = newResult.Count
	summary.NewMRR = newResult.TotalMRR

	// Upgrades
	var upgradeResult struct {
		Count    int   `bun:"count"`
		TotalMRR int64 `bun:"total_mrr"`
	}
	err = r.db.NewSelect().
		Model((*schema.SubscriptionMovement)(nil)).
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("COALESCE(SUM(mrr_change), 0) AS total_mrr").
		Where("app_id = ?", appID).
		Where("movement_type = ?", "upgrade").
		Where("currency = ?", currency).
		Where("occurred_at >= ?", startDate).
		Where("occurred_at <= ?", endDate).
		Scan(ctx, &upgradeResult)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	summary.UpgradeCount = upgradeResult.Count
	summary.ExpansionMRR = upgradeResult.TotalMRR

	// Downgrades
	var downgradeResult struct {
		Count    int   `bun:"count"`
		TotalMRR int64 `bun:"total_mrr"`
	}
	err = r.db.NewSelect().
		Model((*schema.SubscriptionMovement)(nil)).
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("COALESCE(ABS(SUM(mrr_change)), 0) AS total_mrr").
		Where("app_id = ?", appID).
		Where("movement_type = ?", "downgrade").
		Where("currency = ?", currency).
		Where("occurred_at >= ?", startDate).
		Where("occurred_at <= ?", endDate).
		Scan(ctx, &downgradeResult)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	summary.DowngradeCount = downgradeResult.Count
	summary.ContractionMRR = downgradeResult.TotalMRR

	// Churn
	var churnResult struct {
		Count    int   `bun:"count"`
		TotalMRR int64 `bun:"total_mrr"`
	}
	err = r.db.NewSelect().
		Model((*schema.SubscriptionMovement)(nil)).
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("COALESCE(ABS(SUM(mrr_change)), 0) AS total_mrr").
		Where("app_id = ?", appID).
		Where("movement_type = ?", "churn").
		Where("currency = ?", currency).
		Where("occurred_at >= ?", startDate).
		Where("occurred_at <= ?", endDate).
		Scan(ctx, &churnResult)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	summary.ChurnCount = churnResult.Count
	summary.ChurnedMRR = churnResult.TotalMRR

	// Reactivations
	var reactivateResult struct {
		Count    int   `bun:"count"`
		TotalMRR int64 `bun:"total_mrr"`
	}
	err = r.db.NewSelect().
		Model((*schema.SubscriptionMovement)(nil)).
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("COALESCE(SUM(mrr_change), 0) AS total_mrr").
		Where("app_id = ?", appID).
		Where("movement_type = ?", "reactivation").
		Where("currency = ?", currency).
		Where("occurred_at >= ?", startDate).
		Where("occurred_at <= ?", endDate).
		Scan(ctx, &reactivateResult)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	summary.ReactivateCount = reactivateResult.Count
	summary.ReactivationMRR = reactivateResult.TotalMRR

	return summary, nil
}

// CreateCohort creates a cohort record
func (r *analyticsRepository) CreateCohort(ctx context.Context, cohort *core.CohortAnalysis) error {
	// Store as multiple rows, one per month
	for monthNum, retention := range cohort.Retention {
		var revenue int64
		if monthNum < len(cohort.Revenue) {
			revenue = cohort.Revenue[monthNum]
		}

		model := &schema.SubscriptionCohort{
			ID:              xid.New(),
			AppID:           xid.ID{}, // Need to pass app ID
			CohortMonth:     cohort.CohortMonth,
			TotalCustomers:  cohort.TotalCustomers,
			MonthNumber:     monthNum,
			ActiveCustomers: int(float64(cohort.TotalCustomers) * retention / 100),
			RetentionRate:   retention,
			Revenue:         revenue,
			Currency:        cohort.Currency,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		_, err := r.db.NewInsert().Model(model).Exec(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetCohort returns a cohort by month
func (r *analyticsRepository) GetCohort(ctx context.Context, appID xid.ID, cohortMonth time.Time) (*core.CohortAnalysis, error) {
	var cohorts []schema.SubscriptionCohort
	err := r.db.NewSelect().
		Model(&cohorts).
		Where("app_id = ?", appID).
		Where("cohort_month = ?", cohortMonth.Truncate(24*time.Hour)).
		Order("month_number ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	if len(cohorts) == 0 {
		return nil, nil
	}

	result := &core.CohortAnalysis{
		CohortMonth:    cohorts[0].CohortMonth,
		TotalCustomers: cohorts[0].TotalCustomers,
		Retention:      make([]float64, len(cohorts)),
		Revenue:        make([]int64, len(cohorts)),
		Currency:       cohorts[0].Currency,
	}

	for i, c := range cohorts {
		result.Retention[i] = c.RetentionRate
		result.Revenue[i] = c.Revenue
	}

	return result, nil
}

// ListCohorts returns cohorts for analysis
func (r *analyticsRepository) ListCohorts(ctx context.Context, appID xid.ID, startMonth time.Time, numMonths int) ([]*core.CohortAnalysis, error) {
	var cohorts []schema.SubscriptionCohort
	endMonth := startMonth.AddDate(0, numMonths, 0)

	err := r.db.NewSelect().
		Model(&cohorts).
		Where("app_id = ?", appID).
		Where("cohort_month >= ?", startMonth).
		Where("cohort_month < ?", endMonth).
		Order("cohort_month ASC", "month_number ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	// Group by cohort month
	cohortMap := make(map[time.Time]*core.CohortAnalysis)
	for _, c := range cohorts {
		key := c.CohortMonth.Truncate(24 * time.Hour)
		if _, ok := cohortMap[key]; !ok {
			cohortMap[key] = &core.CohortAnalysis{
				CohortMonth:    c.CohortMonth,
				TotalCustomers: c.TotalCustomers,
				Retention:      []float64{},
				Revenue:        []int64{},
				Currency:       c.Currency,
			}
		}
		cohortMap[key].Retention = append(cohortMap[key].Retention, c.RetentionRate)
		cohortMap[key].Revenue = append(cohortMap[key].Revenue, c.Revenue)
	}

	result := make([]*core.CohortAnalysis, 0, len(cohortMap))
	for _, c := range cohortMap {
		result = append(result, c)
	}
	return result, nil
}

// UpsertCohort creates or updates a cohort data point
func (r *analyticsRepository) UpsertCohort(ctx context.Context, appID xid.ID, cohortMonth time.Time, monthNumber int, activeCustomers int, revenue int64, currency string) error {
	model := &schema.SubscriptionCohort{
		ID:              xid.New(),
		AppID:           appID,
		CohortMonth:     cohortMonth.Truncate(24 * time.Hour),
		MonthNumber:     monthNumber,
		ActiveCustomers: activeCustomers,
		Revenue:         revenue,
		Currency:        currency,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err := r.db.NewInsert().
		Model(model).
		On("CONFLICT (app_id, cohort_month, month_number) DO UPDATE").
		Set("active_customers = EXCLUDED.active_customers").
		Set("retention_rate = EXCLUDED.retention_rate").
		Set("revenue = EXCLUDED.revenue").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

// GetDashboardMetrics returns dashboard metrics (stub - needs actual calculation)
func (r *analyticsRepository) GetDashboardMetrics(ctx context.Context, appID xid.ID, currency string) (*core.DashboardMetrics, error) {
	// This would typically aggregate from multiple sources
	// For now, return a stub implementation
	return &core.DashboardMetrics{
		TotalMRR:              0,
		TotalARR:              0,
		ActiveSubscriptions:   0,
		TrialingSubscriptions: 0,
		MRRGrowth:             0,
		SubscriptionGrowth:    0,
		ChurnRate:             0,
		TrialConversionRate:   0,
		NetRevenueRetention:   100,
		NewMRR:                0,
		ExpansionMRR:          0,
		ChurnedMRR:            0,
		Currency:              currency,
		AsOf:                  time.Now(),
	}, nil
}

// GetMRRHistory returns MRR breakdown history (stub)
func (r *analyticsRepository) GetMRRHistory(ctx context.Context, appID xid.ID, startDate, endDate time.Time, currency string) ([]*core.MRRBreakdown, error) {
	// Stub implementation - would aggregate movements by date
	return []*core.MRRBreakdown{}, nil
}

// Helper functions

func schemaToBillingMetric(s *schema.SubscriptionBillingMetric) *core.BillingMetric {
	return &core.BillingMetric{
		ID:        s.ID,
		AppID:     s.AppID,
		Type:      core.MetricType(s.Type),
		Period:    core.MetricPeriod(s.Period),
		Value:     s.Value,
		Currency:  s.Currency,
		Date:      s.Date,
		CreatedAt: s.CreatedAt,
	}
}

func billingMetricToSchema(m *core.BillingMetric) *schema.SubscriptionBillingMetric {
	return &schema.SubscriptionBillingMetric{
		ID:        m.ID,
		AppID:     m.AppID,
		Type:      string(m.Type),
		Period:    string(m.Period),
		Value:     m.Value,
		Currency:  m.Currency,
		Date:      m.Date,
		CreatedAt: m.CreatedAt,
	}
}

func schemaToMovement(s *schema.SubscriptionMovement) *core.SubscriptionMovement {
	return &core.SubscriptionMovement{
		ID:             s.ID,
		AppID:          s.AppID,
		SubscriptionID: s.SubscriptionID,
		OrganizationID: s.OrganizationID,
		MovementType:   s.MovementType,
		PreviousMRR:    s.PreviousMRR,
		NewMRR:         s.NewMRR,
		MRRChange:      s.MRRChange,
		Currency:       s.Currency,
		PreviousPlanID: s.PreviousPlanID,
		NewPlanID:      s.NewPlanID,
		Reason:         s.Reason,
		Notes:          s.Notes,
		OccurredAt:     s.OccurredAt,
		CreatedAt:      s.CreatedAt,
	}
}

func movementToSchema(m *core.SubscriptionMovement) *schema.SubscriptionMovement {
	return &schema.SubscriptionMovement{
		ID:             m.ID,
		AppID:          m.AppID,
		SubscriptionID: m.SubscriptionID,
		OrganizationID: m.OrganizationID,
		MovementType:   m.MovementType,
		PreviousMRR:    m.PreviousMRR,
		NewMRR:         m.NewMRR,
		MRRChange:      m.MRRChange,
		Currency:       m.Currency,
		PreviousPlanID: m.PreviousPlanID,
		NewPlanID:      m.NewPlanID,
		Reason:         m.Reason,
		Notes:          m.Notes,
		OccurredAt:     m.OccurredAt,
		CreatedAt:      m.CreatedAt,
	}
}
