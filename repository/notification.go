package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// notificationRepository implements notification.Repository.
type notificationRepository struct {
	db *bun.DB
}

// NewNotificationRepository creates a new notification repository.
func NewNotificationRepository(db *bun.DB) notification.Repository {
	return &notificationRepository{db: db}
}

// =============================================================================
// TEMPLATE OPERATIONS
// =============================================================================

// CreateTemplate creates a new notification template.
func (r *notificationRepository) CreateTemplate(ctx context.Context, template *schema.NotificationTemplate) error {
	// Set default language if not provided
	if template.Language == "" {
		template.Language = "en"
	}

	_, err := r.db.NewInsert().Model(template).Exec(ctx)

	return err
}

// FindTemplateByID finds a template by ID.
func (r *notificationRepository) FindTemplateByID(ctx context.Context, id xid.ID) (*schema.NotificationTemplate, error) {
	template := &schema.NotificationTemplate{}

	err := r.db.NewSelect().
		Model(template).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return template, nil
}

// FindTemplateByName finds a template by app ID and name.
func (r *notificationRepository) FindTemplateByName(ctx context.Context, appID xid.ID, name string) (*schema.NotificationTemplate, error) {
	template := &schema.NotificationTemplate{}

	err := r.db.NewSelect().
		Model(template).
		Where("app_id = ? AND name = ? AND active = true", appID, name).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return template, nil
}

// FindTemplateByKey finds a template by app, key, type, and language.
func (r *notificationRepository) FindTemplateByKey(ctx context.Context, appID xid.ID, templateKey, notifType, language string) (*schema.NotificationTemplate, error) {
	template := &schema.NotificationTemplate{}

	query := r.db.NewSelect().
		Model(template).
		Where("app_id = ? AND template_key = ? AND active = true", appID, templateKey).
		Where("deleted_at IS NULL")

	if notifType != "" {
		query = query.Where("type = ?", notifType)
	}

	// Try to find exact language match first
	if language != "" {
		query = query.Where("language = ?", language)
	}

	err := query.Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		// If exact language not found and language was specified, try default "en"
		if language != "" && language != "en" {
			query = r.db.NewSelect().
				Model(template).
				Where("app_id = ? AND template_key = ? AND language = ? AND active = true", appID, templateKey, "en").
				Where("deleted_at IS NULL")

			if notifType != "" {
				query = query.Where("type = ?", notifType)
			}

			err = query.Limit(1).Scan(ctx)
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
		} else {
			return nil, nil
		}
	}

	if err != nil {
		return nil, err
	}

	return template, nil
}

// ListTemplates lists templates with pagination.
func (r *notificationRepository) ListTemplates(ctx context.Context, filter *notification.ListTemplatesFilter) (*pagination.PageResponse[*schema.NotificationTemplate], error) {
	var templates []*schema.NotificationTemplate

	query := r.db.NewSelect().
		Model(&templates).
		Where("app_id = ?", filter.AppID).
		Where("deleted_at IS NULL")

	if filter.Type != nil {
		query = query.Where("type = ?", string(*filter.Type))
	}

	if filter.Language != nil {
		query = query.Where("language = ?", *filter.Language)
	}

	if filter.Active != nil {
		query = query.Where("active = ?", *filter.Active)
	}

	// Apply pagination
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	err = query.
		Order("created_at DESC").
		Limit(filter.GetLimit()).
		Offset(filter.GetOffset()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(templates, int64(total), &filter.PaginationParams), nil
}

// UpdateTemplate updates a template.
func (r *notificationRepository) UpdateTemplate(ctx context.Context, id xid.ID, req *notification.UpdateTemplateRequest) error {
	query := r.db.NewUpdate().
		Model((*schema.NotificationTemplate)(nil)).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Set("updated_at = ?", time.Now())

	if req.Name != nil {
		query = query.Set("name = ?", *req.Name)
	}

	if req.Subject != nil {
		query = query.Set("subject = ?", *req.Subject)
	}

	if req.Body != nil {
		query = query.Set("body = ?", *req.Body)
	}

	if req.Variables != nil {
		query = query.Set("variables = ?", req.Variables)
	}

	if req.Metadata != nil {
		query = query.Set("metadata = ?", req.Metadata)
	}

	if req.Active != nil {
		query = query.Set("active = ?", *req.Active)
	}

	_, err := query.Exec(ctx)

	return err
}

// UpdateTemplateMetadata updates template metadata fields (isDefault, isModified, defaultHash).
func (r *notificationRepository) UpdateTemplateMetadata(ctx context.Context, id xid.ID, isDefault, isModified bool, defaultHash string) error {
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationTemplate)(nil)).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Set("is_default = ?", isDefault).
		Set("is_modified = ?", isModified).
		Set("default_hash = ?", defaultHash).
		Set("updated_at = ?", time.Now()).
		Exec(ctx)

	return err
}

// DeleteTemplate deletes a template (soft delete).
func (r *notificationRepository) DeleteTemplate(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationTemplate)(nil)).
		Where("id = ?", id).
		Set("deleted_at = ?", time.Now()).
		Exec(ctx)

	return err
}

// =============================================================================
// NOTIFICATION OPERATIONS
// =============================================================================

// CreateNotification creates a new notification.
func (r *notificationRepository) CreateNotification(ctx context.Context, notif *schema.Notification) error {
	_, err := r.db.NewInsert().Model(notif).Exec(ctx)

	return err
}

// FindNotificationByID finds a notification by ID.
func (r *notificationRepository) FindNotificationByID(ctx context.Context, id xid.ID) (*schema.Notification, error) {
	notif := &schema.Notification{}

	err := r.db.NewSelect().
		Model(notif).
		Where("id = ?", id).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return notif, nil
}

// ListNotifications lists notifications with pagination.
func (r *notificationRepository) ListNotifications(ctx context.Context, filter *notification.ListNotificationsFilter) (*pagination.PageResponse[*schema.Notification], error) {
	var notifications []*schema.Notification

	query := r.db.NewSelect().
		Model(&notifications).
		Where("app_id = ?", filter.AppID)

	if filter.Type != nil {
		query = query.Where("type = ?", string(*filter.Type))
	}

	if filter.Status != nil {
		query = query.Where("status = ?", string(*filter.Status))
	}

	if filter.Recipient != nil {
		query = query.Where("recipient = ?", *filter.Recipient)
	}

	// Apply pagination
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	err = query.
		Order("created_at DESC").
		Limit(filter.GetLimit()).
		Offset(filter.GetOffset()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(notifications, int64(total), &filter.PaginationParams), nil
}

// UpdateNotificationStatus updates the status of a notification.
func (r *notificationRepository) UpdateNotificationStatus(ctx context.Context, id xid.ID, status notification.NotificationStatus, errorMsg string, providerID string) error {
	query := r.db.NewUpdate().
		Model((*schema.Notification)(nil)).
		Where("id = ?", id).
		Set("status = ?", string(status)).
		Set("updated_at = ?", time.Now())

	if errorMsg != "" {
		query = query.Set("error = ?", errorMsg)
	}

	if providerID != "" {
		query = query.Set("provider_id = ?", providerID)
	}

	_, err := query.Exec(ctx)

	return err
}

// UpdateNotificationDelivery updates the delivery timestamp of a notification.
func (r *notificationRepository) UpdateNotificationDelivery(ctx context.Context, id xid.ID, deliveredAt time.Time) error {
	_, err := r.db.NewUpdate().
		Model((*schema.Notification)(nil)).
		Where("id = ?", id).
		Set("status = ?", string(notification.NotificationStatusDelivered)).
		Set("delivered_at = ?", deliveredAt).
		Set("updated_at = ?", time.Now()).
		Exec(ctx)

	return err
}

// =============================================================================
// CLEANUP OPERATIONS
// =============================================================================

// CleanupOldNotifications removes old notifications.
func (r *notificationRepository) CleanupOldNotifications(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.NewDelete().
		Model((*schema.Notification)(nil)).
		Where("created_at < ?", olderThan).
		Exec(ctx)

	return err
}

// =============================================================================
// ORG-SCOPED TEMPLATE OPERATIONS
// =============================================================================

// FindTemplateByKeyOrgScoped finds a template with org-scoping support
// Priority: org-specific > app-level.
func (r *notificationRepository) FindTemplateByKeyOrgScoped(ctx context.Context, appID xid.ID, orgID *xid.ID, templateKey, notifType, language string) (*schema.NotificationTemplate, error) {
	template := &schema.NotificationTemplate{}

	// Try org-specific template first if orgID is provided
	if orgID != nil {
		query := r.db.NewSelect().
			Model(template).
			Where("app_id = ? AND organization_id = ? AND template_key = ? AND active = true", appID, orgID, templateKey).
			Where("deleted_at IS NULL")

		if notifType != "" {
			query = query.Where("type = ?", notifType)
		}

		if language != "" {
			query = query.Where("language = ?", language)
		}

		err := query.Limit(1).Scan(ctx)
		if err == nil {
			return template, nil
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// Fall back to app-level template
	query := r.db.NewSelect().
		Model(template).
		Where("app_id = ? AND organization_id IS NULL AND template_key = ? AND active = true", appID, templateKey).
		Where("deleted_at IS NULL")

	if notifType != "" {
		query = query.Where("type = ?", notifType)
	}

	if language != "" {
		query = query.Where("language = ?", language)
	}

	err := query.Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return template, nil
}

// UpdateTemplateAnalytics updates analytics counters for a template.
func (r *notificationRepository) UpdateTemplateAnalytics(ctx context.Context, id xid.ID, sendCount, openCount, clickCount, conversionCount int64) error {
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationTemplate)(nil)).
		Set("send_count = ?", sendCount).
		Set("open_count = ?", openCount).
		Set("click_count = ?", clickCount).
		Set("conversion_count = ?", conversionCount).
		Set("updated_at = ?", time.Now().UTC()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// =============================================================================
// TEMPLATE VERSIONING OPERATIONS
// =============================================================================

// CreateTemplateVersion creates a new template version.
func (r *notificationRepository) CreateTemplateVersion(ctx context.Context, version *schema.NotificationTemplateVersion) error {
	_, err := r.db.NewInsert().Model(version).Exec(ctx)

	return err
}

// FindTemplateVersionByID finds a version by ID.
func (r *notificationRepository) FindTemplateVersionByID(ctx context.Context, id xid.ID) (*schema.NotificationTemplateVersion, error) {
	version := &schema.NotificationTemplateVersion{}

	err := r.db.NewSelect().
		Model(version).
		Where("id = ?", id).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return version, err
}

// ListTemplateVersions lists all versions for a template.
func (r *notificationRepository) ListTemplateVersions(ctx context.Context, templateID xid.ID) ([]*schema.NotificationTemplateVersion, error) {
	var versions []*schema.NotificationTemplateVersion

	err := r.db.NewSelect().
		Model(&versions).
		Where("template_id = ?", templateID).
		Order("version DESC").
		Scan(ctx)

	return versions, err
}

// GetLatestTemplateVersion gets the most recent version.
func (r *notificationRepository) GetLatestTemplateVersion(ctx context.Context, templateID xid.ID) (*schema.NotificationTemplateVersion, error) {
	version := &schema.NotificationTemplateVersion{}

	err := r.db.NewSelect().
		Model(version).
		Where("template_id = ?", templateID).
		Order("version DESC").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return version, err
}

// =============================================================================
// PROVIDER OPERATIONS
// =============================================================================

// CreateProvider creates a new notification provider.
func (r *notificationRepository) CreateProvider(ctx context.Context, provider *schema.NotificationProvider) error {
	_, err := r.db.NewInsert().Model(provider).Exec(ctx)

	return err
}

// FindProviderByID finds a provider by ID.
func (r *notificationRepository) FindProviderByID(ctx context.Context, id xid.ID) (*schema.NotificationProvider, error) {
	provider := &schema.NotificationProvider{}

	err := r.db.NewSelect().
		Model(provider).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return provider, err
}

// FindProviderByTypeOrgScoped finds a default provider for a type with org-scoping.
func (r *notificationRepository) FindProviderByTypeOrgScoped(ctx context.Context, appID xid.ID, orgID *xid.ID, providerType string) (*schema.NotificationProvider, error) {
	provider := &schema.NotificationProvider{}

	// Try org-specific provider first if orgID is provided
	if orgID != nil {
		err := r.db.NewSelect().
			Model(provider).
			Where("app_id = ? AND organization_id = ? AND provider_type = ? AND is_default = true AND is_active = true", appID, orgID, providerType).
			Where("deleted_at IS NULL").
			Limit(1).
			Scan(ctx)
		if err == nil {
			return provider, nil
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// Fall back to app-level provider
	err := r.db.NewSelect().
		Model(provider).
		Where("app_id = ? AND organization_id IS NULL AND provider_type = ? AND is_default = true AND is_active = true", appID, providerType).
		Where("deleted_at IS NULL").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return provider, err
}

// ListProviders lists all providers for an app/org.
func (r *notificationRepository) ListProviders(ctx context.Context, appID xid.ID, orgID *xid.ID) ([]*schema.NotificationProvider, error) {
	var providers []*schema.NotificationProvider

	query := r.db.NewSelect().
		Model(&providers).
		Where("app_id = ?", appID).
		Where("deleted_at IS NULL")

	if orgID != nil {
		query = query.Where("(organization_id = ? OR organization_id IS NULL)", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	err := query.Scan(ctx)

	return providers, err
}

// UpdateProvider updates a provider.
func (r *notificationRepository) UpdateProvider(ctx context.Context, id xid.ID, config map[string]any, isActive, isDefault bool) error {
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationProvider)(nil)).
		Set("config = ?", config).
		Set("is_active = ?", isActive).
		Set("is_default = ?", isDefault).
		Set("updated_at = ?", time.Now().UTC()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// DeleteProvider soft-deletes a provider.
func (r *notificationRepository) DeleteProvider(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationProvider)(nil)).
		Set("deleted_at = ?", time.Now().UTC()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// =============================================================================
// ANALYTICS OPERATIONS
// =============================================================================

// CreateAnalyticsEvent creates a new analytics event.
func (r *notificationRepository) CreateAnalyticsEvent(ctx context.Context, event *schema.NotificationAnalytics) error {
	_, err := r.db.NewInsert().Model(event).Exec(ctx)

	return err
}

// FindAnalyticsByNotificationID finds all analytics events for a notification.
func (r *notificationRepository) FindAnalyticsByNotificationID(ctx context.Context, notificationID xid.ID) ([]*schema.NotificationAnalytics, error) {
	var events []*schema.NotificationAnalytics

	err := r.db.NewSelect().
		Model(&events).
		Where("notification_id = ?", notificationID).
		Order("created_at ASC").
		Scan(ctx)

	return events, err
}

// GetTemplateAnalytics aggregates analytics for a template.
func (r *notificationRepository) GetTemplateAnalytics(ctx context.Context, templateID xid.ID, startDate, endDate time.Time) (*notification.TemplateAnalyticsReport, error) {
	template := &schema.NotificationTemplate{}

	err := r.db.NewSelect().
		Model(template).
		Where("id = ?", templateID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	report := &notification.TemplateAnalyticsReport{
		TemplateID:   templateID,
		TemplateName: template.Name,
		StartDate:    startDate,
		EndDate:      endDate,
	}

	// Count events by type
	var results []struct {
		Event string `bun:"event"`
		Count int64  `bun:"count"`
	}

	err = r.db.NewSelect().
		Model((*schema.NotificationAnalytics)(nil)).
		Column("event").
		ColumnExpr("COUNT(*) as count").
		Where("template_id = ?", templateID).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("event").
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	// Map counts to report
	for _, result := range results {
		switch result.Event {
		case string(schema.NotificationEventSent):
			report.TotalSent = result.Count
		case string(schema.NotificationEventDelivered):
			report.TotalDelivered = result.Count
		case string(schema.NotificationEventOpened):
			report.TotalOpened = result.Count
		case string(schema.NotificationEventClicked):
			report.TotalClicked = result.Count
		case string(schema.NotificationEventConverted):
			report.TotalConverted = result.Count
		case string(schema.NotificationEventBounced):
			report.TotalBounced = result.Count
		case string(schema.NotificationEventComplained):
			report.TotalComplained = result.Count
		case string(schema.NotificationEventFailed):
			report.TotalFailed = result.Count
		}
	}

	return report, nil
}

// GetAppAnalytics aggregates analytics for an app.
func (r *notificationRepository) GetAppAnalytics(ctx context.Context, appID xid.ID, startDate, endDate time.Time) (*notification.AppAnalyticsReport, error) {
	report := &notification.AppAnalyticsReport{
		AppID:     appID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	var results []struct {
		Event string `bun:"event"`
		Count int64  `bun:"count"`
	}

	err := r.db.NewSelect().
		Model((*schema.NotificationAnalytics)(nil)).
		Column("event").
		ColumnExpr("COUNT(*) as count").
		Where("app_id = ?", appID).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("event").
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		switch result.Event {
		case string(schema.NotificationEventSent):
			report.TotalSent = result.Count
		case string(schema.NotificationEventDelivered):
			report.TotalDelivered = result.Count
		case string(schema.NotificationEventOpened):
			report.TotalOpened = result.Count
		case string(schema.NotificationEventClicked):
			report.TotalClicked = result.Count
		case string(schema.NotificationEventConverted):
			report.TotalConverted = result.Count
		case string(schema.NotificationEventBounced):
			report.TotalBounced = result.Count
		case string(schema.NotificationEventComplained):
			report.TotalComplained = result.Count
		case string(schema.NotificationEventFailed):
			report.TotalFailed = result.Count
		}
	}

	return report, nil
}

// GetOrgAnalytics aggregates analytics for an organization.
func (r *notificationRepository) GetOrgAnalytics(ctx context.Context, orgID xid.ID, startDate, endDate time.Time) (*notification.OrgAnalyticsReport, error) {
	report := &notification.OrgAnalyticsReport{
		OrganizationID: orgID,
		StartDate:      startDate,
		EndDate:        endDate,
	}

	var results []struct {
		Event string `bun:"event"`
		Count int64  `bun:"count"`
	}

	err := r.db.NewSelect().
		Model((*schema.NotificationAnalytics)(nil)).
		Column("event").
		ColumnExpr("COUNT(*) as count").
		Where("organization_id = ?", orgID).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("event").
		Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		switch result.Event {
		case string(schema.NotificationEventSent):
			report.TotalSent = result.Count
		case string(schema.NotificationEventDelivered):
			report.TotalDelivered = result.Count
		case string(schema.NotificationEventOpened):
			report.TotalOpened = result.Count
		case string(schema.NotificationEventClicked):
			report.TotalClicked = result.Count
		case string(schema.NotificationEventConverted):
			report.TotalConverted = result.Count
		case string(schema.NotificationEventBounced):
			report.TotalBounced = result.Count
		case string(schema.NotificationEventComplained):
			report.TotalComplained = result.Count
		case string(schema.NotificationEventFailed):
			report.TotalFailed = result.Count
		}
	}

	return report, nil
}

// =============================================================================
// TEST OPERATIONS
// =============================================================================

// CreateTest creates a new notification test.
func (r *notificationRepository) CreateTest(ctx context.Context, test *schema.NotificationTest) error {
	_, err := r.db.NewInsert().Model(test).Exec(ctx)

	return err
}

// FindTestByID finds a test by ID.
func (r *notificationRepository) FindTestByID(ctx context.Context, id xid.ID) (*schema.NotificationTest, error) {
	test := &schema.NotificationTest{}

	err := r.db.NewSelect().
		Model(test).
		Where("id = ?", id).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return test, err
}

// ListTests lists all tests for a template.
func (r *notificationRepository) ListTests(ctx context.Context, templateID xid.ID) ([]*schema.NotificationTest, error) {
	var tests []*schema.NotificationTest

	err := r.db.NewSelect().
		Model(&tests).
		Where("template_id = ?", templateID).
		Order("created_at DESC").
		Scan(ctx)

	return tests, err
}

// UpdateTestStatus updates the status of a test.
func (r *notificationRepository) UpdateTestStatus(ctx context.Context, id xid.ID, status string, results map[string]any, successCount, failureCount int) error {
	now := time.Now().UTC()
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationTest)(nil)).
		Set("status = ?", status).
		Set("results = ?", results).
		Set("success_count = ?", successCount).
		Set("failure_count = ?", failureCount).
		Set("completed_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// =============================================================================
// CLEANUP OPERATIONS
// =============================================================================

// CleanupOldAnalytics removes old analytics data.
func (r *notificationRepository) CleanupOldAnalytics(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.NewDelete().
		Model((*schema.NotificationAnalytics)(nil)).
		Where("created_at < ?", olderThan).
		Exec(ctx)

	return err
}

// CleanupOldTests removes old test records.
func (r *notificationRepository) CleanupOldTests(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.NewDelete().
		Model((*schema.NotificationTest)(nil)).
		Where("created_at < ?", olderThan).
		Exec(ctx)

	return err
}
