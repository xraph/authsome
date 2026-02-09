package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/core"
	suberrors "github.com/xraph/authsome/plugins/subscription/errors"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// UsageService handles usage tracking and metering.
type UsageService struct {
	repo      repository.UsageRepository
	subRepo   repository.SubscriptionRepository
	planRepo  repository.PlanRepository
	provider  providers.PaymentProvider
	eventRepo repository.EventRepository
}

// NewUsageService creates a new usage service.
func NewUsageService(
	repo repository.UsageRepository,
	subRepo repository.SubscriptionRepository,
	provider providers.PaymentProvider,
	eventRepo repository.EventRepository,
) *UsageService {
	return &UsageService{
		repo:      repo,
		subRepo:   subRepo,
		provider:  provider,
		eventRepo: eventRepo,
	}
}

// RecordUsage records a usage event.
func (s *UsageService) RecordUsage(ctx context.Context, req *core.RecordUsageRequest) (*core.UsageRecord, error) {
	// Validate metric key
	if req.MetricKey == "" {
		return nil, suberrors.ErrInvalidUsageMetric
	}

	// Validate action
	if !req.Action.IsValid() {
		return nil, suberrors.ErrInvalidUsageAction
	}

	// Check idempotency
	if req.IdempotencyKey != "" {
		existing, _ := s.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
		if existing != nil {
			return s.schemaToCoreRecord(existing), nil
		}
	}

	// Get subscription to find organization
	sub, err := s.subRepo.FindByID(ctx, req.SubscriptionID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	// Create record
	now := time.Now()

	timestamp := now
	if req.Timestamp != nil {
		timestamp = *req.Timestamp
	}

	record := &schema.SubscriptionUsageRecord{
		ID:             xid.New(),
		SubscriptionID: req.SubscriptionID,
		OrganizationID: sub.OrganizationID,
		MetricKey:      req.MetricKey,
		Quantity:       req.Quantity,
		Action:         string(req.Action),
		Timestamp:      timestamp,
		IdempotencyKey: req.IdempotencyKey,
		Reported:       false,
		CreatedAt:      now,
	}

	if req.Metadata != nil {
		record.Metadata = req.Metadata
	} else {
		record.Metadata = make(map[string]any)
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to record usage: %w", err)
	}

	// Record event
	s.recordEvent(ctx, sub.ID, sub.OrganizationID, string(core.EventUsageRecorded), map[string]any{
		"metricKey": req.MetricKey,
		"quantity":  req.Quantity,
		"action":    string(req.Action),
	})

	return s.schemaToCoreRecord(record), nil
}

// GetSummary retrieves usage summary for a subscription and metric.
func (s *UsageService) GetSummary(ctx context.Context, req *core.GetUsageSummaryRequest) (*core.UsageSummary, error) {
	summary, err := s.repo.GetSummary(ctx, req.SubscriptionID, req.MetricKey, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage summary: %w", err)
	}

	return &core.UsageSummary{
		MetricKey:     summary.MetricKey,
		TotalQuantity: summary.TotalQuantity,
		PeriodStart:   req.PeriodStart,
		PeriodEnd:     req.PeriodEnd,
		RecordCount:   summary.RecordCount,
	}, nil
}

// GetUsageLimit retrieves current usage against limit for a metric.
func (s *UsageService) GetUsageLimit(ctx context.Context, orgID xid.ID, metricKey string) (*core.UsageLimit, error) {
	// Get subscription
	sub, err := s.subRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	if sub.Plan == nil {
		return nil, errs.New(errs.CodeNotFound, "plan not loaded", http.StatusNotFound)
	}

	// Get feature limit from plan
	var limit int64 = 0

	for _, f := range sub.Plan.Features {
		if f.Key == metricKey {
			switch f.Type {
			case "unlimited":
				limit = -1
			case "limit":
				// Parse value as int (stored as string)
				var val float64
				if err := json.Unmarshal([]byte(f.Value), &val); err == nil {
					limit = int64(val)
				}
			}

			break
		}
	}

	// Get current usage
	summary, err := s.repo.GetSummary(ctx, sub.ID, metricKey, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	return core.NewUsageLimit(metricKey, summary.TotalQuantity, limit), nil
}

// List retrieves usage records with filtering.
func (s *UsageService) List(ctx context.Context, subID, orgID *xid.ID, metricKey string, reported *bool, page, pageSize int) ([]*core.UsageRecord, int, error) {
	filter := &repository.UsageFilter{
		SubscriptionID: subID,
		OrganizationID: orgID,
		MetricKey:      metricKey,
		Reported:       reported,
		Page:           page,
		PageSize:       pageSize,
	}

	records, count, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list usage records: %w", err)
	}

	result := make([]*core.UsageRecord, len(records))
	for i, r := range records {
		result[i] = s.schemaToCoreRecord(r)
	}

	return result, count, nil
}

// ReportToProvider reports unreported usage to the payment provider.
func (s *UsageService) ReportToProvider(ctx context.Context, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 100
	}

	records, err := s.repo.GetUnreported(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get unreported records: %w", err)
	}

	for _, record := range records {
		// Get subscription for provider info
		sub, err := s.subRepo.FindByID(ctx, record.SubscriptionID)
		if err != nil {
			continue
		}

		if sub.ProviderSubID == "" {
			continue
		}

		// Convert to core record
		coreRecord := s.schemaToCoreRecord(record)

		// Report to provider
		providerID, err := s.provider.ReportUsage(ctx, sub.ProviderSubID, []*core.UsageRecord{coreRecord})
		if err != nil {
			// Log error but continue
			continue
		}

		// Mark as reported
		s.repo.MarkReported(ctx, record.ID, providerID)
	}

	return nil
}

// GetCurrentPeriodUsage gets all usage for the current billing period.
func (s *UsageService) GetCurrentPeriodUsage(ctx context.Context, subID xid.ID) (map[string]int64, error) {
	sub, err := s.subRepo.FindByID(ctx, subID)
	if err != nil {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	// Get all unique metric keys
	filter := &repository.UsageFilter{
		SubscriptionID: &subID,
		Page:           1,
		PageSize:       1000,
	}

	records, _, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list usage: %w", err)
	}

	// Group by metric key
	metrics := make(map[string]bool)
	for _, r := range records {
		metrics[r.MetricKey] = true
	}

	// Get summary for each metric
	result := make(map[string]int64)

	for metric := range metrics {
		summary, err := s.repo.GetSummary(ctx, subID, metric, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
		if err != nil {
			continue
		}

		result[metric] = summary.TotalQuantity
	}

	return result, nil
}

// Helper methods

func (s *UsageService) recordEvent(ctx context.Context, subID, orgID xid.ID, eventType string, data map[string]any) {
	event := &schema.SubscriptionEvent{
		ID:             xid.New(),
		SubscriptionID: &subID,
		OrganizationID: orgID,
		EventType:      eventType,
		EventData:      data,
		CreatedAt:      time.Now(),
	}
	s.eventRepo.Create(ctx, event)
}

func (s *UsageService) schemaToCoreRecord(record *schema.SubscriptionUsageRecord) *core.UsageRecord {
	return &core.UsageRecord{
		ID:               record.ID,
		SubscriptionID:   record.SubscriptionID,
		OrganizationID:   record.OrganizationID,
		MetricKey:        record.MetricKey,
		Quantity:         record.Quantity,
		Action:           core.UsageAction(record.Action),
		Timestamp:        record.Timestamp,
		IdempotencyKey:   record.IdempotencyKey,
		Metadata:         record.Metadata,
		ProviderRecordID: record.ProviderRecordID,
		Reported:         record.Reported,
		ReportedAt:       record.ReportedAt,
		CreatedAt:        record.CreatedAt,
	}
}
