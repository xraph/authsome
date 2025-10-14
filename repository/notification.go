package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/schema"
)

// notificationRepository implements notification.Repository
type notificationRepository struct {
	db *bun.DB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *bun.DB) *notificationRepository {
	return &notificationRepository{db: db}
}

// CreateTemplate creates a new notification template
func (r *notificationRepository) CreateTemplate(ctx context.Context, template *notification.Template) error {
	schemaTemplate := &schema.NotificationTemplate{
		ID:             template.ID,
		OrganizationID: template.OrganizationID,
		Name:           template.Name,
		Type:           string(template.Type),
		Subject:        template.Subject,
		Body:           template.Body,
		Variables:      template.Variables,
		Metadata:       template.Metadata,
		Active:         template.Active,
		CreatedAt:      template.CreatedAt,
		UpdatedAt:      template.UpdatedAt,
	}
	_, err := r.db.NewInsert().Model(schemaTemplate).Exec(ctx)
	return err
}

// FindTemplateByID finds a template by ID
func (r *notificationRepository) FindTemplateByID(ctx context.Context, id xid.ID) (*notification.Template, error) {
	schemaTemplate := &schema.NotificationTemplate{}
	err := r.db.NewSelect().
		Model(schemaTemplate).
		Where("id = ?", id).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &notification.Template{
		ID:             schemaTemplate.ID,
		OrganizationID: schemaTemplate.OrganizationID,
		Name:           schemaTemplate.Name,
		Type:           notification.NotificationType(schemaTemplate.Type),
		Subject:        schemaTemplate.Subject,
		Body:           schemaTemplate.Body,
		Variables:      schemaTemplate.Variables,
		Metadata:       schemaTemplate.Metadata,
		Active:         schemaTemplate.Active,
		CreatedAt:      schemaTemplate.CreatedAt,
		UpdatedAt:      schemaTemplate.UpdatedAt,
	}, nil
}

// FindTemplateByName finds a template by organization ID and name
func (r *notificationRepository) FindTemplateByName(ctx context.Context, orgID, name string) (*notification.Template, error) {
	schemaTemplate := &schema.NotificationTemplate{}
	err := r.db.NewSelect().
		Model(schemaTemplate).
		Where("organization_id = ? AND name = ? AND active = true", orgID, name).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &notification.Template{
		ID:             schemaTemplate.ID,
		OrganizationID: schemaTemplate.OrganizationID,
		Name:           schemaTemplate.Name,
		Type:           notification.NotificationType(schemaTemplate.Type),
		Subject:        schemaTemplate.Subject,
		Body:           schemaTemplate.Body,
		Variables:      schemaTemplate.Variables,
		Metadata:       schemaTemplate.Metadata,
		Active:         schemaTemplate.Active,
		CreatedAt:      schemaTemplate.CreatedAt,
		UpdatedAt:      schemaTemplate.UpdatedAt,
	}, nil
}

// ListTemplates lists templates with pagination
func (r *notificationRepository) ListTemplates(ctx context.Context, req *notification.ListTemplatesRequest) ([]*notification.Template, int64, error) {
	var schemaTemplates []*schema.NotificationTemplate
	query := r.db.NewSelect().
		Model(&schemaTemplates).
		Where("organization_id = ?", req.OrganizationID)

	if req.Type != "" {
		query = query.Where("type = ?", string(req.Type))
	}
	if req.Active != nil {
		query = query.Where("active = ?", *req.Active)
	}

	err := query.Order("created_at DESC").
		Limit(req.Limit).
		Offset(req.Offset).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Count total
	countQuery := r.db.NewSelect().
		Model((*schema.NotificationTemplate)(nil)).
		Where("organization_id = ?", req.OrganizationID)
	if req.Type != "" {
		countQuery = countQuery.Where("type = ?", string(req.Type))
	}
	if req.Active != nil {
		countQuery = countQuery.Where("active = ?", *req.Active)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to core types
	templates := make([]*notification.Template, len(schemaTemplates))
	for i, st := range schemaTemplates {
		templates[i] = &notification.Template{
			ID:             st.ID,
			OrganizationID: st.OrganizationID,
			Name:           st.Name,
			Type:           notification.NotificationType(st.Type),
			Subject:        st.Subject,
			Body:           st.Body,
			Variables:      st.Variables,
			Metadata:       st.Metadata,
			Active:         st.Active,
			CreatedAt:      st.CreatedAt,
			UpdatedAt:      st.UpdatedAt,
		}
	}

	return templates, int64(total), nil
}

// UpdateTemplate updates a template
func (r *notificationRepository) UpdateTemplate(ctx context.Context, id xid.ID, req *notification.UpdateTemplateRequest) error {
	query := r.db.NewUpdate().
		Model((*schema.NotificationTemplate)(nil)).
		Where("id = ?", id).
		Set("updated_at = ?", time.Now())

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

// DeleteTemplate deletes a template (soft delete)
func (r *notificationRepository) DeleteTemplate(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationTemplate)(nil)).
		Where("id = ?", id).
		Set("deleted_at = ?", time.Now()).
		Exec(ctx)
	return err
}

// CreateNotification creates a new notification
func (r *notificationRepository) CreateNotification(ctx context.Context, notification *notification.Notification) error {
	schemaNotification := &schema.Notification{
		ID:             notification.ID,
		OrganizationID: notification.OrganizationID,
		TemplateID:     notification.TemplateID,
		Type:           string(notification.Type),
		Recipient:      notification.Recipient,
		Subject:        notification.Subject,
		Body:           notification.Body,
		Status:         string(notification.Status),
		Error:          notification.Error,
		ProviderID:     notification.ProviderID,
		Metadata:       notification.Metadata,
		SentAt:         notification.SentAt,
		DeliveredAt:    notification.DeliveredAt,
		CreatedAt:      notification.CreatedAt,
		UpdatedAt:      notification.UpdatedAt,
	}
	_, err := r.db.NewInsert().Model(schemaNotification).Exec(ctx)
	return err
}

// FindNotificationByID finds a notification by ID
func (r *notificationRepository) FindNotificationByID(ctx context.Context, id xid.ID) (*notification.Notification, error) {
	schemaNotification := &schema.Notification{}
	err := r.db.NewSelect().
		Model(schemaNotification).
		Where("id = ?", id).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &notification.Notification{
		ID:             schemaNotification.ID,
		OrganizationID: schemaNotification.OrganizationID,
		TemplateID:     schemaNotification.TemplateID,
		Type:           notification.NotificationType(schemaNotification.Type),
		Recipient:      schemaNotification.Recipient,
		Subject:        schemaNotification.Subject,
		Body:           schemaNotification.Body,
		Status:         notification.NotificationStatus(schemaNotification.Status),
		Error:          schemaNotification.Error,
		ProviderID:     schemaNotification.ProviderID,
		Metadata:       schemaNotification.Metadata,
		SentAt:         schemaNotification.SentAt,
		DeliveredAt:    schemaNotification.DeliveredAt,
		CreatedAt:      schemaNotification.CreatedAt,
		UpdatedAt:      schemaNotification.UpdatedAt,
	}, nil
}

// ListNotifications lists notifications with pagination
func (r *notificationRepository) ListNotifications(ctx context.Context, req *notification.ListNotificationsRequest) ([]*notification.Notification, int64, error) {
	var schemaNotifications []*schema.Notification
	query := r.db.NewSelect().
		Model(&schemaNotifications).
		Where("organization_id = ?", req.OrganizationID)

	if req.Type != "" {
		query = query.Where("type = ?", string(req.Type))
	}
	if req.Status != "" {
		query = query.Where("status = ?", string(req.Status))
	}
	if req.Recipient != "" {
		query = query.Where("recipient = ?", req.Recipient)
	}

	err := query.Order("created_at DESC").
		Limit(req.Limit).
		Offset(req.Offset).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Count total
	countQuery := r.db.NewSelect().
		Model((*schema.Notification)(nil)).
		Where("organization_id = ?", req.OrganizationID)
	if req.Type != "" {
		countQuery = countQuery.Where("type = ?", string(req.Type))
	}
	if req.Status != "" {
		countQuery = countQuery.Where("status = ?", string(req.Status))
	}
	if req.Recipient != "" {
		countQuery = countQuery.Where("recipient = ?", req.Recipient)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to core types
	notifications := make([]*notification.Notification, len(schemaNotifications))
	for i, sn := range schemaNotifications {
		notifications[i] = &notification.Notification{
			ID:             sn.ID,
			OrganizationID: sn.OrganizationID,
			TemplateID:     sn.TemplateID,
			Type:           notification.NotificationType(sn.Type),
			Recipient:      sn.Recipient,
			Subject:        sn.Subject,
			Body:           sn.Body,
			Status:         notification.NotificationStatus(sn.Status),
			Error:          sn.Error,
			ProviderID:     sn.ProviderID,
			Metadata:       sn.Metadata,
			SentAt:         sn.SentAt,
			DeliveredAt:    sn.DeliveredAt,
			CreatedAt:      sn.CreatedAt,
			UpdatedAt:      sn.UpdatedAt,
		}
	}

	return notifications, int64(total), nil
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

// CleanupOldNotifications removes old notifications
func (r *notificationRepository) CleanupOldNotifications(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.NewDelete().
		Model((*schema.Notification)(nil)).
		Where("created_at < ?", olderThan).
		Exec(ctx)
	return err
}