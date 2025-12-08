package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// usageRepository implements UsageRepository using Bun
type usageRepository struct {
	db *bun.DB
}

// NewUsageRepository creates a new usage repository
func NewUsageRepository(db *bun.DB) UsageRepository {
	return &usageRepository{db: db}
}

// Create creates a new usage record
func (r *usageRepository) Create(ctx context.Context, record *schema.SubscriptionUsageRecord) error {
	_, err := r.db.NewInsert().Model(record).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create usage record: %w", err)
	}
	return nil
}

// FindByID retrieves a usage record by ID
func (r *usageRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionUsageRecord, error) {
	record := new(schema.SubscriptionUsageRecord)
	err := r.db.NewSelect().
		Model(record).
		Where("sur.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find usage record: %w", err)
	}
	return record, nil
}

// FindByIdempotencyKey retrieves a usage record by idempotency key
func (r *usageRepository) FindByIdempotencyKey(ctx context.Context, key string) (*schema.SubscriptionUsageRecord, error) {
	if key == "" {
		return nil, nil
	}
	record := new(schema.SubscriptionUsageRecord)
	err := r.db.NewSelect().
		Model(record).
		Where("sur.idempotency_key = ?", key).
		Scan(ctx)
	if err != nil {
		return nil, nil // Not found is not an error for idempotency
	}
	return record, nil
}

// List retrieves usage records with optional filters
func (r *usageRepository) List(ctx context.Context, filter *UsageFilter) ([]*schema.SubscriptionUsageRecord, int, error) {
	var records []*schema.SubscriptionUsageRecord

	query := r.db.NewSelect().
		Model(&records).
		Order("sur.timestamp DESC")

	if filter != nil {
		if filter.SubscriptionID != nil {
			query = query.Where("sur.subscription_id = ?", *filter.SubscriptionID)
		}
		if filter.OrganizationID != nil {
			query = query.Where("sur.organization_id = ?", *filter.OrganizationID)
		}
		if filter.MetricKey != "" {
			query = query.Where("sur.metric_key = ?", filter.MetricKey)
		}
		if filter.Reported != nil {
			query = query.Where("sur.reported = ?", *filter.Reported)
		}

		// Pagination
		pageSize := filter.PageSize
		if pageSize <= 0 {
			pageSize = 100
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
		return nil, 0, fmt.Errorf("failed to list usage records: %w", err)
	}

	return records, count, nil
}

// GetSummary calculates usage summary for a subscription and metric
func (r *usageRepository) GetSummary(ctx context.Context, subscriptionID xid.ID, metricKey string, periodStart, periodEnd interface{}) (*UsageSummary, error) {
	var summary UsageSummary
	summary.MetricKey = metricKey

	startTime, ok := periodStart.(time.Time)
	if !ok {
		return nil, fmt.Errorf("invalid period start type")
	}
	endTime, ok := periodEnd.(time.Time)
	if !ok {
		return nil, fmt.Errorf("invalid period end type")
	}

	// Sum up usage records
	err := r.db.NewSelect().
		Model((*schema.SubscriptionUsageRecord)(nil)).
		ColumnExpr("COALESCE(SUM(CASE WHEN action = 'decrement' THEN -quantity ELSE quantity END), 0) AS total_quantity").
		ColumnExpr("COUNT(*) AS record_count").
		Where("subscription_id = ?", subscriptionID).
		Where("metric_key = ?", metricKey).
		Where("timestamp >= ?", startTime).
		Where("timestamp <= ?", endTime).
		Scan(ctx, &summary.TotalQuantity, &summary.RecordCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage summary: %w", err)
	}

	return &summary, nil
}

// GetUnreported retrieves usage records not yet reported to provider
func (r *usageRepository) GetUnreported(ctx context.Context, limit int) ([]*schema.SubscriptionUsageRecord, error) {
	var records []*schema.SubscriptionUsageRecord

	query := r.db.NewSelect().
		Model(&records).
		Where("sur.reported = ?", false).
		Order("sur.timestamp ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unreported usage records: %w", err)
	}

	return records, nil
}

// MarkReported marks a usage record as reported
func (r *usageRepository) MarkReported(ctx context.Context, id xid.ID, providerRecordID string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.SubscriptionUsageRecord)(nil)).
		Set("reported = ?", true).
		Set("reported_at = ?", now).
		Set("provider_record_id = ?", providerRecordID).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to mark usage record as reported: %w", err)
	}
	return nil
}
