package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// AnalyticsService handles notification analytics operations.
type AnalyticsService struct {
	repo Repository
}

// NewAnalyticsService creates a new analytics service.
func NewAnalyticsService(repo Repository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// TrackEvent tracks a notification event.
func (s *AnalyticsService) TrackEvent(ctx context.Context, notificationID, templateID, appID xid.ID, orgID *xid.ID, eventType string, eventData map[string]any) error {
	event := &schema.NotificationAnalytics{
		ID:             xid.New(),
		NotificationID: notificationID,
		TemplateID:     &templateID,
		AppID:          appID,
		OrganizationID: orgID,
		Event:          eventType,
		EventData:      eventData,
	}

	if err := s.repo.CreateAnalyticsEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to create analytics event: %w", err)
	}

	// Update template analytics counters
	if err := s.incrementTemplateCounter(ctx, templateID, eventType); err != nil {
		// Log but don't fail - analytics are non-critical
		_ = err
	}

	return nil
}

// incrementTemplateCounter increments the appropriate counter on the template.
func (s *AnalyticsService) incrementTemplateCounter(ctx context.Context, templateID xid.ID, eventType string) error {
	template, err := s.repo.FindTemplateByID(ctx, templateID)
	if err != nil || template == nil {
		return err
	}

	// Increment appropriate counter
	switch eventType {
	case string(schema.NotificationEventSent):
		template.SendCount++
	case string(schema.NotificationEventOpened):
		template.OpenCount++
	case string(schema.NotificationEventClicked):
		template.ClickCount++
	case string(schema.NotificationEventConverted):
		template.ConversionCount++
	}

	return s.repo.UpdateTemplateAnalytics(ctx, templateID, template.SendCount, template.OpenCount, template.ClickCount, template.ConversionCount)
}

// GetTemplateAnalytics retrieves analytics for a specific template.
func (s *AnalyticsService) GetTemplateAnalytics(ctx context.Context, templateID xid.ID, startDate, endDate time.Time) (*TemplateAnalyticsReport, error) {
	report, err := s.repo.GetTemplateAnalytics(ctx, templateID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get template analytics: %w", err)
	}

	// Calculate rates
	if report.TotalSent > 0 {
		report.DeliveryRate = float64(report.TotalDelivered) / float64(report.TotalSent) * 100
		report.BounceRate = float64(report.TotalBounced) / float64(report.TotalSent) * 100
	}

	if report.TotalDelivered > 0 {
		report.OpenRate = float64(report.TotalOpened) / float64(report.TotalDelivered) * 100
		report.ComplaintRate = float64(report.TotalComplained) / float64(report.TotalDelivered) * 100
	}

	if report.TotalOpened > 0 {
		report.ClickRate = float64(report.TotalClicked) / float64(report.TotalOpened) * 100
	}

	if report.TotalClicked > 0 {
		report.ConversionRate = float64(report.TotalConverted) / float64(report.TotalClicked) * 100
	}

	return report, nil
}

// GetAppAnalytics retrieves aggregate analytics for an app.
func (s *AnalyticsService) GetAppAnalytics(ctx context.Context, appID xid.ID, startDate, endDate time.Time) (*AppAnalyticsReport, error) {
	report, err := s.repo.GetAppAnalytics(ctx, appID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get app analytics: %w", err)
	}

	// Calculate rates
	if report.TotalSent > 0 {
		report.DeliveryRate = float64(report.TotalDelivered) / float64(report.TotalSent) * 100
		report.BounceRate = float64(report.TotalBounced) / float64(report.TotalSent) * 100
	}

	if report.TotalDelivered > 0 {
		report.OpenRate = float64(report.TotalOpened) / float64(report.TotalDelivered) * 100
		report.ComplaintRate = float64(report.TotalComplained) / float64(report.TotalDelivered) * 100
	}

	if report.TotalOpened > 0 {
		report.ClickRate = float64(report.TotalClicked) / float64(report.TotalOpened) * 100
	}

	if report.TotalClicked > 0 {
		report.ConversionRate = float64(report.TotalConverted) / float64(report.TotalClicked) * 100
	}

	return report, nil
}

// GetOrgAnalytics retrieves aggregate analytics for an organization.
func (s *AnalyticsService) GetOrgAnalytics(ctx context.Context, orgID xid.ID, startDate, endDate time.Time) (*OrgAnalyticsReport, error) {
	report, err := s.repo.GetOrgAnalytics(ctx, orgID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get org analytics: %w", err)
	}

	// Calculate rates
	if report.TotalSent > 0 {
		report.DeliveryRate = float64(report.TotalDelivered) / float64(report.TotalSent) * 100
		report.BounceRate = float64(report.TotalBounced) / float64(report.TotalSent) * 100
	}

	if report.TotalDelivered > 0 {
		report.OpenRate = float64(report.TotalOpened) / float64(report.TotalDelivered) * 100
		report.ComplaintRate = float64(report.TotalComplained) / float64(report.TotalDelivered) * 100
	}

	if report.TotalOpened > 0 {
		report.ClickRate = float64(report.TotalClicked) / float64(report.TotalOpened) * 100
	}

	if report.TotalClicked > 0 {
		report.ConversionRate = float64(report.TotalConverted) / float64(report.TotalClicked) * 100
	}

	return report, nil
}

// GetEventsByNotification retrieves all analytics events for a notification.
func (s *AnalyticsService) GetEventsByNotification(ctx context.Context, notificationID xid.ID) ([]*schema.NotificationAnalytics, error) {
	events, err := s.repo.FindAnalyticsByNotificationID(ctx, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find analytics events: %w", err)
	}

	return events, nil
}

// TrackOpen tracks when a recipient opens an email (via tracking pixel).
func (s *AnalyticsService) TrackOpen(ctx context.Context, notificationID xid.ID, userAgent, ipAddress string) error {
	// Get notification to find template and app info
	notification, err := s.repo.FindNotificationByID(ctx, notificationID)
	if err != nil || notification == nil {
		return fmt.Errorf("notification not found: %w", err)
	}

	eventData := map[string]any{
		"user_agent": userAgent,
		"ip_address": ipAddress,
	}

	templateID := xid.NilID()
	if notification.TemplateID != nil {
		templateID = *notification.TemplateID
	}

	return s.TrackEvent(ctx, notificationID, templateID, notification.AppID, nil, string(schema.NotificationEventOpened), eventData)
}

// TrackClick tracks when a recipient clicks a link.
func (s *AnalyticsService) TrackClick(ctx context.Context, notificationID xid.ID, linkURL, userAgent, ipAddress string) error {
	// Get notification to find template and app info
	notification, err := s.repo.FindNotificationByID(ctx, notificationID)
	if err != nil || notification == nil {
		return fmt.Errorf("notification not found: %w", err)
	}

	eventData := map[string]any{
		"link_url":   linkURL,
		"user_agent": userAgent,
		"ip_address": ipAddress,
	}

	templateID := xid.NilID()
	if notification.TemplateID != nil {
		templateID = *notification.TemplateID
	}

	return s.TrackEvent(ctx, notificationID, templateID, notification.AppID, nil, string(schema.NotificationEventClicked), eventData)
}

// TrackConversion tracks when a recipient completes a desired action.
func (s *AnalyticsService) TrackConversion(ctx context.Context, notificationID xid.ID, conversionType string, conversionValue float64) error {
	// Get notification to find template and app info
	notification, err := s.repo.FindNotificationByID(ctx, notificationID)
	if err != nil || notification == nil {
		return fmt.Errorf("notification not found: %w", err)
	}

	eventData := map[string]any{
		"conversion_type":  conversionType,
		"conversion_value": conversionValue,
	}

	templateID := xid.NilID()
	if notification.TemplateID != nil {
		templateID = *notification.TemplateID
	}

	return s.TrackEvent(ctx, notificationID, templateID, notification.AppID, nil, string(schema.NotificationEventConverted), eventData)
}
