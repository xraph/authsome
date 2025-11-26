package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/repository"
)

// AlertService handles usage alerts and notifications
type AlertService struct {
	repo      repository.AlertRepository
	usageRepo repository.UsageRepository
	subRepo   repository.SubscriptionRepository
	// notificationService would be injected here for actual sending
}

// NewAlertService creates a new alert service
func NewAlertService(repo repository.AlertRepository, usageRepo repository.UsageRepository, subRepo repository.SubscriptionRepository) *AlertService {
	return &AlertService{
		repo:      repo,
		usageRepo: usageRepo,
		subRepo:   subRepo,
	}
}

// CreateAlertConfig creates a new alert configuration
func (s *AlertService) CreateAlertConfig(ctx context.Context, appID xid.ID, req *core.CreateAlertConfigRequest) (*core.AlertConfig, error) {
	config := &core.AlertConfig{
		ID:               xid.New(),
		AppID:            appID,
		OrganizationID:   req.OrganizationID,
		AlertType:        req.AlertType,
		IsEnabled:        true,
		ThresholdPercent: req.ThresholdPercent,
		MetricKey:        req.MetricKey,
		DaysBeforeEnd:    req.DaysBeforeEnd,
		Channels:         req.Channels,
		Recipients:       req.Recipients,
		WebhookURL:       req.WebhookURL,
		SlackChannel:     req.SlackChannel,
		MinInterval:      req.MinInterval,
		MaxAlertsPerDay:  req.MaxAlertsPerDay,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if config.MinInterval == 0 {
		config.MinInterval = 60 // Default 1 hour
	}
	if config.MaxAlertsPerDay == 0 {
		config.MaxAlertsPerDay = 5
	}

	if err := s.repo.CreateAlertConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create alert config: %w", err)
	}

	return config, nil
}

// GetAlertConfig returns an alert config by ID
func (s *AlertService) GetAlertConfig(ctx context.Context, id xid.ID) (*core.AlertConfig, error) {
	return s.repo.GetAlertConfig(ctx, id)
}

// ListAlertConfigs returns all alert configs for an organization
func (s *AlertService) ListAlertConfigs(ctx context.Context, orgID xid.ID) ([]*core.AlertConfig, error) {
	return s.repo.ListAlertConfigs(ctx, orgID)
}

// UpdateAlertConfig updates an alert config
func (s *AlertService) UpdateAlertConfig(ctx context.Context, id xid.ID, req *core.UpdateAlertConfigRequest) (*core.AlertConfig, error) {
	config, err := s.repo.GetAlertConfig(ctx, id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("alert config not found")
	}

	if req.IsEnabled != nil {
		config.IsEnabled = *req.IsEnabled
	}
	if req.ThresholdPercent != nil {
		config.ThresholdPercent = *req.ThresholdPercent
	}
	if req.DaysBeforeEnd != nil {
		config.DaysBeforeEnd = *req.DaysBeforeEnd
	}
	if len(req.Channels) > 0 {
		config.Channels = req.Channels
	}
	if len(req.Recipients) > 0 {
		config.Recipients = req.Recipients
	}
	if req.WebhookURL != nil {
		config.WebhookURL = *req.WebhookURL
	}
	if req.SlackChannel != nil {
		config.SlackChannel = *req.SlackChannel
	}
	if req.MinInterval != nil {
		config.MinInterval = *req.MinInterval
	}
	if req.MaxAlertsPerDay != nil {
		config.MaxAlertsPerDay = *req.MaxAlertsPerDay
	}

	config.UpdatedAt = time.Now()

	if err := s.repo.UpdateAlertConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update alert config: %w", err)
	}

	return config, nil
}

// DeleteAlertConfig deletes an alert config
func (s *AlertService) DeleteAlertConfig(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteAlertConfig(ctx, id)
}

// SnoozeAlertConfig snoozes alerts for a config
func (s *AlertService) SnoozeAlertConfig(ctx context.Context, id xid.ID, until time.Time) error {
	config, err := s.repo.GetAlertConfig(ctx, id)
	if err != nil {
		return err
	}
	if config == nil {
		return fmt.Errorf("alert config not found")
	}

	config.SnoozedUntil = &until
	config.UpdatedAt = time.Now()

	return s.repo.UpdateAlertConfig(ctx, config)
}

// CreateAlert creates a new alert
func (s *AlertService) CreateAlert(ctx context.Context, alert *core.Alert) error {
	if alert.ID.IsNil() {
		alert.ID = xid.New()
	}
	alert.Status = core.AlertStatusPending
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()

	return s.repo.CreateAlert(ctx, alert)
}

// GetAlert returns an alert by ID
func (s *AlertService) GetAlert(ctx context.Context, id xid.ID) (*core.Alert, error) {
	return s.repo.GetAlert(ctx, id)
}

// ListAlerts returns alerts for an organization
func (s *AlertService) ListAlerts(ctx context.Context, orgID xid.ID, status *core.AlertStatus, page, pageSize int) ([]*core.Alert, int, error) {
	return s.repo.ListAlerts(ctx, orgID, status, page, pageSize)
}

// AcknowledgeAlert marks an alert as acknowledged
func (s *AlertService) AcknowledgeAlert(ctx context.Context, req *core.AcknowledgeAlertRequest) error {
	alert, err := s.repo.GetAlert(ctx, req.AlertID)
	if err != nil {
		return err
	}
	if alert == nil {
		return fmt.Errorf("alert not found")
	}

	now := time.Now()
	alert.Status = core.AlertStatusAcknowledged
	alert.AcknowledgedAt = &now
	alert.AcknowledgedBy = req.AcknowledgedBy
	alert.UpdatedAt = now

	return s.repo.UpdateAlert(ctx, alert)
}

// ResolveAlert marks an alert as resolved
func (s *AlertService) ResolveAlert(ctx context.Context, req *core.ResolveAlertRequest) error {
	alert, err := s.repo.GetAlert(ctx, req.AlertID)
	if err != nil {
		return err
	}
	if alert == nil {
		return fmt.Errorf("alert not found")
	}

	now := time.Now()
	alert.Status = core.AlertStatusResolved
	alert.ResolvedAt = &now
	alert.Resolution = req.Resolution
	alert.UpdatedAt = now

	return s.repo.UpdateAlert(ctx, alert)
}

// TriggerAlert manually triggers an alert
func (s *AlertService) TriggerAlert(ctx context.Context, appID xid.ID, req *core.TriggerAlertRequest) (*core.Alert, error) {
	alert := &core.Alert{
		ID:             xid.New(),
		AppID:          appID,
		OrganizationID: req.OrganizationID,
		Type:           req.Type,
		Severity:       req.Severity,
		Status:         core.AlertStatusPending,
		Title:          req.Title,
		Message:        req.Message,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if alert.Severity == "" {
		alert.Severity = core.AlertSeverityInfo
	}

	if err := s.repo.CreateAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	// In a real implementation, this would dispatch to notification service
	// s.notificationService.SendAlert(ctx, alert)

	return alert, nil
}

// CheckUsageThresholds checks all organizations for usage threshold alerts
// This would typically be called by a scheduled job
func (s *AlertService) CheckUsageThresholds(ctx context.Context, appID xid.ID) error {
	// This is a stub - in production would:
	// 1. List all active subscriptions
	// 2. For each subscription, check usage against plan limits
	// 3. If threshold is exceeded, check if we should send alert (interval, daily limit)
	// 4. Create and send alert if needed
	return nil
}

// CheckTrialEndings checks for subscriptions with trials ending soon
// This would typically be called by a scheduled job
func (s *AlertService) CheckTrialEndings(ctx context.Context, appID xid.ID) error {
	// This is a stub - in production would:
	// 1. List subscriptions with trials ending in next X days
	// 2. For each, check if alert config exists and is enabled
	// 3. Check if we should send alert (haven't sent one recently)
	// 4. Create and send alert if needed
	return nil
}

// ProcessPendingAlerts processes and sends pending alerts
// This would typically be called by a scheduled job
func (s *AlertService) ProcessPendingAlerts(ctx context.Context) error {
	alerts, err := s.repo.ListPendingAlerts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list pending alerts: %w", err)
	}

	for _, alert := range alerts {
		// In production, this would send via configured channels
		// For now, just mark as sent
		now := time.Now()
		alert.Status = core.AlertStatusSent
		alert.SentAt = &now
		alert.UpdatedAt = now

		if err := s.repo.UpdateAlert(ctx, alert); err != nil {
			// Log error but continue processing
			continue
		}
	}

	return nil
}

// GetAlertSummary returns an alert summary for an organization
func (s *AlertService) GetAlertSummary(ctx context.Context, orgID xid.ID) (*core.AlertSummary, error) {
	// Get all alerts for the org
	pending, _ := core.AlertStatusPending, error(nil)
	pendingAlerts, pendingCount, _ := s.repo.ListAlerts(ctx, orgID, &pending, 1, 1000)

	sent := core.AlertStatusSent
	sentAlerts, sentCount, _ := s.repo.ListAlerts(ctx, orgID, &sent, 1, 1000)

	acked := core.AlertStatusAcknowledged
	_, ackedCount, _ := s.repo.ListAlerts(ctx, orgID, &acked, 1, 1000)

	// Build summary
	summary := &core.AlertSummary{
		TotalAlerts:        pendingCount + sentCount + ackedCount,
		PendingAlerts:      pendingCount,
		SentAlerts:         sentCount,
		AcknowledgedAlerts: ackedCount,
		BySeverity:         make(map[core.AlertSeverity]int),
		ByType:             make(map[core.AlertType]int),
		RecentAlerts:       make([]core.Alert, 0),
	}

	// Count by severity and type
	allAlerts := append(pendingAlerts, sentAlerts...)
	for _, alert := range allAlerts {
		summary.BySeverity[alert.Severity]++
		summary.ByType[alert.Type]++
	}

	// Get recent alerts
	if len(allAlerts) > 10 {
		allAlerts = allAlerts[:10]
	}
	for _, alert := range allAlerts {
		summary.RecentAlerts = append(summary.RecentAlerts, *alert)
	}

	return summary, nil
}

// SetupDefaultAlerts creates default alert configs for a new organization
func (s *AlertService) SetupDefaultAlerts(ctx context.Context, appID, orgID xid.ID) error {
	defaults := core.DefaultAlertConfigs(appID, orgID)
	for _, config := range defaults {
		config.ID = xid.New()
		config.CreatedAt = time.Now()
		config.UpdatedAt = time.Now()
		if err := s.repo.CreateAlertConfig(ctx, &config); err != nil {
			// Log but continue - non-critical
			continue
		}
	}
	return nil
}
