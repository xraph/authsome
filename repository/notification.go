package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// notificationRepository implements notification.Repository
type notificationRepository struct {
	db *bun.DB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *bun.DB) notification.Repository {
	return &notificationRepository{db: db}
}

// =============================================================================
// TEMPLATE OPERATIONS
// =============================================================================

// CreateTemplate creates a new notification template
func (r *notificationRepository) CreateTemplate(ctx context.Context, template *schema.NotificationTemplate) error {
	// Set default language if not provided
	if template.Language == "" {
		template.Language = "en"
	}

	_, err := r.db.NewInsert().Model(template).Exec(ctx)
	return err
}

// FindTemplateByID finds a template by ID
func (r *notificationRepository) FindTemplateByID(ctx context.Context, id xid.ID) (*schema.NotificationTemplate, error) {
	template := &schema.NotificationTemplate{}
	err := r.db.NewSelect().
		Model(template).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return template, nil
}

// FindTemplateByName finds a template by app ID and name
func (r *notificationRepository) FindTemplateByName(ctx context.Context, appID xid.ID, name string) (*schema.NotificationTemplate, error) {
	template := &schema.NotificationTemplate{}
	err := r.db.NewSelect().
		Model(template).
		Where("app_id = ? AND name = ? AND active = true", appID, name).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return template, nil
}

// FindTemplateByKey finds a template by app, key, type, and language
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
	if err == sql.ErrNoRows {
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
			if err == sql.ErrNoRows {
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

// ListTemplates lists templates with pagination
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

// UpdateTemplate updates a template
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

// UpdateTemplateMetadata updates template metadata fields (isDefault, isModified, defaultHash)
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

// DeleteTemplate deletes a template (soft delete)
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

// CreateNotification creates a new notification
func (r *notificationRepository) CreateNotification(ctx context.Context, notif *schema.Notification) error {
	_, err := r.db.NewInsert().Model(notif).Exec(ctx)
	return err
}

// FindNotificationByID finds a notification by ID
func (r *notificationRepository) FindNotificationByID(ctx context.Context, id xid.ID) (*schema.Notification, error) {
	notif := &schema.Notification{}
	err := r.db.NewSelect().
		Model(notif).
		Where("id = ?", id).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return notif, nil
}

// ListNotifications lists notifications with pagination
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

// UpdateNotificationStatus updates the status of a notification
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

// UpdateNotificationDelivery updates the delivery timestamp of a notification
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

// CleanupOldNotifications removes old notifications
func (r *notificationRepository) CleanupOldNotifications(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.NewDelete().
		Model((*schema.Notification)(nil)).
		Where("created_at < ?", olderThan).
		Exec(ctx)
	return err
}
