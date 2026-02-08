package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// featureUsageRepository implements FeatureUsageRepository using Bun
type featureUsageRepository struct {
	db *bun.DB
}

// NewFeatureUsageRepository creates a new feature usage repository
func NewFeatureUsageRepository(db *bun.DB) FeatureUsageRepository {
	return &featureUsageRepository{db: db}
}

// CreateUsage creates feature usage for an organization
func (r *featureUsageRepository) CreateUsage(ctx context.Context, usage *schema.OrganizationFeatureUsage) error {
	usage.CreatedAt = time.Now()
	usage.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().Model(usage).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create feature usage: %w", err)
	}
	return nil
}

// UpdateUsage updates feature usage
func (r *featureUsageRepository) UpdateUsage(ctx context.Context, usage *schema.OrganizationFeatureUsage) error {
	usage.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(usage).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update feature usage: %w", err)
	}
	return nil
}

// FindUsage retrieves feature usage for an organization and feature
func (r *featureUsageRepository) FindUsage(ctx context.Context, orgID, featureID xid.ID) (*schema.OrganizationFeatureUsage, error) {
	usage := new(schema.OrganizationFeatureUsage)
	err := r.db.NewSelect().
		Model(usage).
		Where("organization_id = ?", orgID).
		Where("feature_id = ?", featureID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find feature usage: %w", err)
	}
	return usage, nil
}

// FindUsageByKey retrieves feature usage by feature key
func (r *featureUsageRepository) FindUsageByKey(ctx context.Context, orgID, appID xid.ID, featureKey string) (*schema.OrganizationFeatureUsage, error) {
	usage := new(schema.OrganizationFeatureUsage)
	err := r.db.NewSelect().
		Model(usage).
		Join("JOIN subscription_features sf ON sf.id = sofu.feature_id").
		Where("sofu.organization_id = ?", orgID).
		Where("sf.app_id = ?", appID).
		Where("sf.key = ?", featureKey).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find feature usage by key: %w", err)
	}
	return usage, nil
}

// ListUsage retrieves all feature usage for an organization
func (r *featureUsageRepository) ListUsage(ctx context.Context, orgID xid.ID) ([]*schema.OrganizationFeatureUsage, error) {
	var usages []*schema.OrganizationFeatureUsage
	err := r.db.NewSelect().
		Model(&usages).
		Relation("Feature").
		Where("sofu.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list feature usage: %w", err)
	}
	return usages, nil
}

// IncrementUsage atomically increments usage by a quantity
func (r *featureUsageRepository) IncrementUsage(ctx context.Context, orgID, featureID xid.ID, quantity int64) (*schema.OrganizationFeatureUsage, error) {
	usage := new(schema.OrganizationFeatureUsage)
	_, err := r.db.NewUpdate().
		Model(usage).
		Set("current_usage = current_usage + ?", quantity).
		Set("updated_at = ?", time.Now()).
		Where("organization_id = ?", orgID).
		Where("feature_id = ?", featureID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to increment usage: %w", err)
	}
	return usage, nil
}

// DecrementUsage atomically decrements usage by a quantity
func (r *featureUsageRepository) DecrementUsage(ctx context.Context, orgID, featureID xid.ID, quantity int64) (*schema.OrganizationFeatureUsage, error) {
	usage := new(schema.OrganizationFeatureUsage)
	_, err := r.db.NewUpdate().
		Model(usage).
		Set("current_usage = CASE WHEN current_usage >= ? THEN current_usage - ? ELSE 0 END", quantity, quantity).
		Set("updated_at = ?", time.Now()).
		Where("organization_id = ?", orgID).
		Where("feature_id = ?", featureID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to decrement usage: %w", err)
	}
	return usage, nil
}

// ResetUsage resets usage to zero
func (r *featureUsageRepository) ResetUsage(ctx context.Context, orgID, featureID xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.OrganizationFeatureUsage)(nil)).
		Set("current_usage = 0").
		Set("last_reset = ?", now).
		Set("updated_at = ?", now).
		Where("organization_id = ?", orgID).
		Where("feature_id = ?", featureID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset usage: %w", err)
	}
	return nil
}

// CreateLog creates a usage log entry
func (r *featureUsageRepository) CreateLog(ctx context.Context, log *schema.FeatureUsageLog) error {
	log.CreatedAt = time.Now()
	_, err := r.db.NewInsert().Model(log).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create usage log: %w", err)
	}
	return nil
}

// ListLogs retrieves usage logs with filters
func (r *featureUsageRepository) ListLogs(ctx context.Context, filter *FeatureUsageLogFilter) ([]*schema.FeatureUsageLog, int, error) {
	var logs []*schema.FeatureUsageLog

	query := r.db.NewSelect().
		Model(&logs).
		Order("created_at DESC")

	if filter != nil {
		if filter.OrganizationID != nil {
			query = query.Where("sful.organization_id = ?", *filter.OrganizationID)
		}
		if filter.FeatureID != nil {
			query = query.Where("sful.feature_id = ?", *filter.FeatureID)
		}
		if filter.Action != "" {
			query = query.Where("sful.action = ?", filter.Action)
		}

		// Pagination
		pageSize := filter.PageSize
		if pageSize <= 0 {
			pageSize = 20
		}
		page := filter.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * pageSize
		query = query.Limit(pageSize).Offset(offset)
	}

	count, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list usage logs: %w", err)
	}

	return logs, count, nil
}

// FindLogByIdempotencyKey finds a log by idempotency key
func (r *featureUsageRepository) FindLogByIdempotencyKey(ctx context.Context, key string) (*schema.FeatureUsageLog, error) {
	log := new(schema.FeatureUsageLog)
	err := r.db.NewSelect().
		Model(log).
		Where("idempotency_key = ?", key).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find log by idempotency key: %w", err)
	}
	return log, nil
}

// CreateGrant creates a feature grant
func (r *featureUsageRepository) CreateGrant(ctx context.Context, grant *schema.FeatureGrant) error {
	grant.CreatedAt = time.Now()
	grant.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().Model(grant).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create feature grant: %w", err)
	}
	return nil
}

// UpdateGrant updates a feature grant
func (r *featureUsageRepository) UpdateGrant(ctx context.Context, grant *schema.FeatureGrant) error {
	grant.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(grant).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update feature grant: %w", err)
	}
	return nil
}

// DeleteGrant deletes a feature grant
func (r *featureUsageRepository) DeleteGrant(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.FeatureGrant)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete feature grant: %w", err)
	}
	return nil
}

// FindGrantByID retrieves a grant by ID
func (r *featureUsageRepository) FindGrantByID(ctx context.Context, id xid.ID) (*schema.FeatureGrant, error) {
	grant := new(schema.FeatureGrant)
	err := r.db.NewSelect().
		Model(grant).
		Where("sfg.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find feature grant: %w", err)
	}
	return grant, nil
}

// ListGrants retrieves all active grants for an organization and feature
func (r *featureUsageRepository) ListGrants(ctx context.Context, orgID, featureID xid.ID) ([]*schema.FeatureGrant, error) {
	var grants []*schema.FeatureGrant
	err := r.db.NewSelect().
		Model(&grants).
		Where("organization_id = ?", orgID).
		Where("feature_id = ?", featureID).
		Where("is_active = ?", true).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list feature grants: %w", err)
	}
	return grants, nil
}

// ListAllOrgGrants retrieves all active grants for an organization
func (r *featureUsageRepository) ListAllOrgGrants(ctx context.Context, orgID xid.ID) ([]*schema.FeatureGrant, error) {
	var grants []*schema.FeatureGrant
	err := r.db.NewSelect().
		Model(&grants).
		Relation("Feature").
		Where("sfg.organization_id = ?", orgID).
		Where("sfg.is_active = ?", true).
		Where("(sfg.expires_at IS NULL OR sfg.expires_at > ?)", time.Now()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization grants: %w", err)
	}
	return grants, nil
}

// GetTotalGrantedValue calculates total granted quota for an organization and feature
func (r *featureUsageRepository) GetTotalGrantedValue(ctx context.Context, orgID, featureID xid.ID) (int64, error) {
	var total int64
	err := r.db.NewSelect().
		Model((*schema.FeatureGrant)(nil)).
		ColumnExpr("COALESCE(SUM(value), 0)").
		Where("organization_id = ?", orgID).
		Where("feature_id = ?", featureID).
		Where("is_active = ?", true).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Scan(ctx, &total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total granted value: %w", err)
	}
	return total, nil
}

// ExpireGrants marks expired grants as inactive
func (r *featureUsageRepository) ExpireGrants(ctx context.Context) error {
	_, err := r.db.NewUpdate().
		Model((*schema.FeatureGrant)(nil)).
		Set("is_active = ?", false).
		Set("updated_at = ?", time.Now()).
		Where("is_active = ?", true).
		Where("expires_at IS NOT NULL").
		Where("expires_at <= ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to expire grants: %w", err)
	}
	return nil
}

// GetUsageNeedingReset retrieves usage records that need to be reset
func (r *featureUsageRepository) GetUsageNeedingReset(ctx context.Context, resetPeriod string) ([]*schema.OrganizationFeatureUsage, error) {
	var usages []*schema.OrganizationFeatureUsage
	err := r.db.NewSelect().
		Model(&usages).
		Join("JOIN subscription_features sf ON sf.id = sofu.feature_id").
		Where("sf.reset_period = ?", resetPeriod).
		Where("sofu.period_end <= ?", time.Now()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage needing reset: %w", err)
	}
	return usages, nil
}

// GetCurrentUsageSnapshot retrieves all current usage across all organizations for an app
func (r *featureUsageRepository) GetCurrentUsageSnapshot(ctx context.Context, appID xid.ID) ([]*core.CurrentUsage, error) {
	type result struct {
		OrganizationID   xid.ID `bun:"organization_id"`
		OrganizationName string `bun:"org_name"`
		FeatureID        xid.ID `bun:"feature_id"`
		FeatureName      string `bun:"feature_name"`
		FeatureType      string `bun:"feature_type"`
		Unit             string `bun:"unit"`
		CurrentUsage     int64  `bun:"current_usage"`
		PlanLimit        int64  `bun:"plan_limit"`
	}

	var results []result
	err := r.db.NewSelect().
		Model((*schema.OrganizationFeatureUsage)(nil)).
		ColumnExpr("sofu.organization_id").
		ColumnExpr("o.name AS org_name").
		ColumnExpr("sofu.feature_id").
		ColumnExpr("sf.name AS feature_name").
		ColumnExpr("sf.type AS feature_type").
		ColumnExpr("sf.unit").
		ColumnExpr("sofu.current_usage").
		ColumnExpr("COALESCE(spfl.limit_value, -1) AS plan_limit").
		Join("JOIN subscription_features sf ON sf.id = sofu.feature_id").
		Join("JOIN organizations o ON o.id = sofu.organization_id").
		Join("LEFT JOIN subscriptions sub ON sub.organization_id = sofu.organization_id").
		Join("LEFT JOIN subscription_plan_feature_links spfl ON spfl.plan_id = sub.plan_id AND spfl.feature_id = sofu.feature_id").
		Where("sf.app_id = ?", appID).
		Where("sofu.current_usage > 0 OR spfl.limit_value IS NOT NULL").
		Scan(ctx, &results)

	if err != nil {
		return nil, fmt.Errorf("failed to get current usage snapshot: %w", err)
	}

	// Convert to core.CurrentUsage
	usage := make([]*core.CurrentUsage, len(results))
	for i, r := range results {
		percentUsed := 0.0
		if r.PlanLimit > 0 {
			percentUsed = float64(r.CurrentUsage) / float64(r.PlanLimit) * 100
		}

		usage[i] = &core.CurrentUsage{
			OrganizationID:   r.OrganizationID,
			OrganizationName: r.OrganizationName,
			FeatureID:        r.FeatureID,
			FeatureName:      r.FeatureName,
			FeatureType:      r.FeatureType,
			Unit:             r.Unit,
			CurrentUsage:     r.CurrentUsage,
			Limit:            r.PlanLimit,
			PercentUsed:      percentUsed,
		}
	}

	return usage, nil
}

// GetUsageByOrg retrieves usage statistics by organization
func (r *featureUsageRepository) GetUsageByOrg(ctx context.Context, appID xid.ID, startDate, endDate time.Time) ([]*core.OrgUsageStats, error) {
	type result struct {
		OrgID       xid.ID `bun:"organization_id"`
		OrgName     string `bun:"org_name"`
		FeatureID   xid.ID `bun:"feature_id"`
		FeatureName string `bun:"feature_name"`
		FeatureType string `bun:"feature_type"`
		Unit        string `bun:"unit"`
		TotalUsage  int64  `bun:"total_usage"`
		PlanLimit   int64  `bun:"plan_limit"`
	}

	var results []result
	err := r.db.NewSelect().
		Model((*schema.FeatureUsageLog)(nil)).
		ColumnExpr("sful.organization_id").
		ColumnExpr("o.name AS org_name").
		ColumnExpr("sful.feature_id").
		ColumnExpr("sf.name AS feature_name").
		ColumnExpr("sf.type AS feature_type").
		ColumnExpr("sf.unit").
		ColumnExpr("SUM(CASE WHEN sful.action = 'consume' THEN sful.quantity ELSE 0 END) AS total_usage").
		ColumnExpr("COALESCE(spfl.limit_value, -1) AS plan_limit").
		Join("JOIN subscription_features sf ON sf.id = sful.feature_id").
		Join("JOIN organizations o ON o.id = sful.organization_id").
		Join("LEFT JOIN subscriptions sub ON sub.organization_id = sful.organization_id").
		Join("LEFT JOIN subscription_plan_feature_links spfl ON spfl.plan_id = sub.plan_id AND spfl.feature_id = sful.feature_id").
		Where("sf.app_id = ?", appID).
		Where("sful.created_at >= ?", startDate).
		Where("sful.created_at <= ?", endDate).
		Group("sful.organization_id, o.name, sful.feature_id, sf.name, sf.type, sf.unit, spfl.limit_value").
		Order("total_usage DESC").
		Scan(ctx, &results)

	if err != nil {
		return nil, fmt.Errorf("failed to get usage by org: %w", err)
	}

	// Convert to core.OrgUsageStats
	stats := make([]*core.OrgUsageStats, len(results))
	for i, r := range results {
		percentUsed := 0.0
		if r.PlanLimit > 0 {
			percentUsed = float64(r.TotalUsage) / float64(r.PlanLimit) * 100
		}

		stats[i] = &core.OrgUsageStats{
			OrgID:       r.OrgID,
			OrgName:     r.OrgName,
			FeatureID:   r.FeatureID,
			FeatureName: r.FeatureName,
			FeatureType: core.FeatureType(r.FeatureType),
			Unit:        r.Unit,
			Usage:       r.TotalUsage,
			Limit:       r.PlanLimit,
			PercentUsed: percentUsed,
		}
	}

	return stats, nil
}

// GetUsageTrends retrieves usage trends over time for a feature
func (r *featureUsageRepository) GetUsageTrends(ctx context.Context, appID xid.ID, featureID *xid.ID, startDate, endDate time.Time) ([]*core.UsageTrend, error) {
	type result struct {
		Date   time.Time `bun:"date"`
		Usage  int64     `bun:"usage"`
		Action string    `bun:"action"`
	}

	var results []result
	query := r.db.NewSelect().
		Model((*schema.FeatureUsageLog)(nil)).
		ColumnExpr("DATE(sful.created_at) AS date").
		ColumnExpr("SUM(sful.quantity) AS usage").
		ColumnExpr("sful.action").
		Join("JOIN subscription_features sf ON sf.id = sful.feature_id").
		Where("sf.app_id = ?", appID).
		Where("sful.created_at >= ?", startDate).
		Where("sful.created_at <= ?", endDate).
		Group("DATE(sful.created_at), sful.action").
		Order("date ASC")

	if featureID != nil {
		query = query.Where("sful.feature_id = ?", *featureID)
	}

	err := query.Scan(ctx, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage trends: %w", err)
	}

	// Convert to core.UsageTrend
	trends := make([]*core.UsageTrend, len(results))
	for i, r := range results {
		trends[i] = &core.UsageTrend{
			Date:   r.Date,
			Usage:  r.Usage,
			Action: r.Action,
		}
	}

	return trends, nil
}

// GetTopConsumers retrieves top consuming organizations
func (r *featureUsageRepository) GetTopConsumers(ctx context.Context, appID xid.ID, featureID *xid.ID, startDate, endDate time.Time, limit int) ([]*core.OrgUsageStats, error) {
	stats, err := r.GetUsageByOrg(ctx, appID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Filter by feature if specified
	if featureID != nil {
		filtered := make([]*core.OrgUsageStats, 0)
		for _, s := range stats {
			if s.FeatureID == *featureID {
				filtered = append(filtered, s)
			}
		}
		stats = filtered
	}

	// Limit results
	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}

	return stats, nil
}

// GetUsageByFeatureType retrieves usage aggregated by feature type
func (r *featureUsageRepository) GetUsageByFeatureType(ctx context.Context, appID xid.ID, startDate, endDate time.Time) (map[core.FeatureType]*core.UsageStats, error) {
	type result struct {
		FeatureType string `bun:"feature_type"`
		TotalUsage  int64  `bun:"total_usage"`
		TotalOrgs   int    `bun:"total_orgs"`
	}

	var results []result
	err := r.db.NewSelect().
		Model((*schema.FeatureUsageLog)(nil)).
		ColumnExpr("sf.type AS feature_type").
		ColumnExpr("SUM(CASE WHEN sful.action = 'consume' THEN sful.quantity ELSE 0 END) AS total_usage").
		ColumnExpr("COUNT(DISTINCT sful.organization_id) AS total_orgs").
		Join("JOIN subscription_features sf ON sf.id = sful.feature_id").
		Where("sf.app_id = ?", appID).
		Where("sful.created_at >= ?", startDate).
		Where("sful.created_at <= ?", endDate).
		Group("sf.type").
		Scan(ctx, &results)

	if err != nil {
		return nil, fmt.Errorf("failed to get usage by feature type: %w", err)
	}

	// Convert to map
	statsMap := make(map[core.FeatureType]*core.UsageStats)
	for _, r := range results {
		avgUsage := 0.0
		if r.TotalOrgs > 0 {
			avgUsage = float64(r.TotalUsage) / float64(r.TotalOrgs)
		}

		statsMap[core.FeatureType(r.FeatureType)] = &core.UsageStats{
			TotalUsage:   r.TotalUsage,
			TotalOrgs:    r.TotalOrgs,
			AverageUsage: avgUsage,
		}
	}

	return statsMap, nil
}
