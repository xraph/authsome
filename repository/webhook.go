package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/schema"
)

// webhookRepository implements webhook.Repository
type webhookRepository struct {
	db *bun.DB
}

// NewWebhookRepository creates a new webhook repository
func NewWebhookRepository(db *bun.DB) webhook.Repository {
	return &webhookRepository{db: db}
}

// Create creates a new webhook
func (r *webhookRepository) Create(ctx context.Context, wh *webhook.Webhook) error {
	schemaWebhook := &schema.Webhook{
		ID:             wh.ID,
		OrganizationID: wh.OrganizationID,
		URL:            wh.URL,
		Events:         wh.Events,
		Secret:         wh.Secret,
		Active:         wh.Enabled,
		Headers:        wh.Headers,
		CreatedAt:      wh.CreatedAt,
		UpdatedAt:      wh.UpdatedAt,
	}
	_, err := r.db.NewInsert().Model(schemaWebhook).Exec(ctx)
	return err
}

// FindByID finds a webhook by ID
func (r *webhookRepository) FindByID(ctx context.Context, id xid.ID) (*webhook.Webhook, error) {
	schemaWebhook := &schema.Webhook{}
	err := r.db.NewSelect().
		Model(schemaWebhook).
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &webhook.Webhook{
		ID:             schemaWebhook.ID,
		OrganizationID: schemaWebhook.OrganizationID,
		URL:            schemaWebhook.URL,
		Events:         schemaWebhook.Events,
		Secret:         schemaWebhook.Secret,
		Enabled:        schemaWebhook.Active,
		Headers:        schemaWebhook.Headers,
		CreatedAt:      schemaWebhook.CreatedAt,
		UpdatedAt:      schemaWebhook.UpdatedAt,
	}, nil
}

// FindByOrgID finds webhooks by organization ID
func (r *webhookRepository) FindByOrgID(ctx context.Context, orgID string, enabled *bool, offset, limit int) ([]*webhook.Webhook, int64, error) {
	var schemaWebhooks []*schema.Webhook
	query := r.db.NewSelect().
		Model(&schemaWebhooks).
		Where("organization_id = ? AND deleted_at IS NULL", orgID)
	
	if enabled != nil {
		query = query.Where("active = ?", *enabled)
	}
	
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}
	
	// Count total
	countQuery := r.db.NewSelect().
		Model((*schema.Webhook)(nil)).
		Where("organization_id = ? AND deleted_at IS NULL", orgID)
	if enabled != nil {
		countQuery = countQuery.Where("active = ?", *enabled)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	
	// Convert to core types
	webhooks := make([]*webhook.Webhook, len(schemaWebhooks))
	for i, sw := range schemaWebhooks {
		webhooks[i] = &webhook.Webhook{
			ID:             sw.ID,
			OrganizationID: sw.OrganizationID,
			URL:            sw.URL,
			Events:         sw.Events,
			Secret:         sw.Secret,
			Enabled:        sw.Active,
			Headers:        sw.Headers,
			CreatedAt:      sw.CreatedAt,
			UpdatedAt:      sw.UpdatedAt,
		}
	}
	
	return webhooks, int64(total), nil
}

// FindByOrgAndEvent finds webhooks by organization and event type
func (r *webhookRepository) FindByOrgAndEvent(ctx context.Context, orgID, eventType string) ([]*webhook.Webhook, error) {
	var schemaWebhooks []*schema.Webhook
	err := r.db.NewSelect().
		Model(&schemaWebhooks).
		Where("organization_id = ? AND active = true AND deleted_at IS NULL", orgID).
		Where("? = ANY(events)", eventType).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert to core types
	webhooks := make([]*webhook.Webhook, len(schemaWebhooks))
	for i, sw := range schemaWebhooks {
		webhooks[i] = &webhook.Webhook{
			ID:             sw.ID,
			OrganizationID: sw.OrganizationID,
			URL:            sw.URL,
			Events:         sw.Events,
			Secret:         sw.Secret,
			Enabled:        sw.Active,
			Headers:        sw.Headers,
			CreatedAt:      sw.CreatedAt,
			UpdatedAt:      sw.UpdatedAt,
		}
	}
	
	return webhooks, nil
}

// Update updates a webhook
func (r *webhookRepository) Update(ctx context.Context, wh *webhook.Webhook) error {
	schemaWebhook := &schema.Webhook{
		ID:             wh.ID,
		OrganizationID: wh.OrganizationID,
		URL:            wh.URL,
		Events:         wh.Events,
		Secret:         wh.Secret,
		Active:         wh.Enabled,
		Headers:        wh.Headers,
		UpdatedAt:      time.Now(),
	}
	_, err := r.db.NewUpdate().
		Model(schemaWebhook).
		Where("id = ? AND deleted_at IS NULL", wh.ID).
		Exec(ctx)
	return err
}

// Delete soft deletes a webhook
func (r *webhookRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.Webhook)(nil)).
		Set("deleted_at = ?", time.Now()).
		Where("id = ? AND deleted_at IS NULL", id).
		Exec(ctx)
	return err
}

// UpdateFailureCount updates the failure count for a webhook
func (r *webhookRepository) UpdateFailureCount(ctx context.Context, id xid.ID, count int) error {
	_, err := r.db.NewUpdate().
		Model((*schema.Webhook)(nil)).
		Set("failure_count = ?", count).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// UpdateLastDelivery updates the last delivery timestamp for a webhook
func (r *webhookRepository) UpdateLastDelivery(ctx context.Context, id xid.ID, timestamp time.Time) error {
	_, err := r.db.NewUpdate().
		Model((*schema.Webhook)(nil)).
		Set("last_delivery = ?", timestamp).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// CreateEvent creates a new webhook event
func (r *webhookRepository) CreateEvent(ctx context.Context, event *webhook.Event) error {
	schemaEvent := &schema.Event{
		ID:             event.ID,
		OrganizationID: event.OrganizationID,
		Type:           event.Type,
		Data:           event.Data,
		CreatedAt:      event.CreatedAt,
	}
	_, err := r.db.NewInsert().Model(schemaEvent).Exec(ctx)
	return err
}

// FindEventByID finds an event by ID
func (r *webhookRepository) FindEventByID(ctx context.Context, id xid.ID) (*webhook.Event, error) {
	schemaEvent := &schema.Event{}
	err := r.db.NewSelect().
		Model(schemaEvent).
		Where("id = ?", id).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &webhook.Event{
		ID:             schemaEvent.ID,
		Type:           schemaEvent.Type,
		OrganizationID: schemaEvent.OrganizationID,
		Data:           schemaEvent.Data,
		OccurredAt:     schemaEvent.CreatedAt,
		CreatedAt:      schemaEvent.CreatedAt,
	}, nil
}

// ListEvents lists events for an organization
func (r *webhookRepository) ListEvents(ctx context.Context, orgID string, offset, limit int) ([]*webhook.Event, int64, error) {
	var schemaEvents []*schema.Event
	err := r.db.NewSelect().
		Model(&schemaEvents).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}
	
	// Count total
	total, err := r.db.NewSelect().
		Model((*schema.Event)(nil)).
		Where("organization_id = ?", orgID).
		Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	
	// Convert to core types
	events := make([]*webhook.Event, len(schemaEvents))
	for i, se := range schemaEvents {
		events[i] = &webhook.Event{
			ID:             se.ID,
			Type:           se.Type,
			OrganizationID: se.OrganizationID,
			Data:           se.Data,
			OccurredAt:     se.CreatedAt,
			CreatedAt:      se.CreatedAt,
		}
	}
	
	return events, int64(total), nil
}

// CreateDelivery creates a new webhook delivery
func (r *webhookRepository) CreateDelivery(ctx context.Context, delivery *webhook.Delivery) error {
	schemaDelivery := &schema.Delivery{
		ID:           delivery.ID,
		WebhookID:    delivery.WebhookID,
		EventID:      delivery.EventID,
		Status:       delivery.Status,
		StatusCode:   &delivery.StatusCode,
		ResponseBody: []byte(delivery.Response),
		Error:        &delivery.Error,
		DeliveredAt:  delivery.DeliveredAt,
		CreatedAt:    delivery.CreatedAt,
		UpdatedAt:    delivery.UpdatedAt,
	}
	_, err := r.db.NewInsert().Model(schemaDelivery).Exec(ctx)
	return err
}

// FindDeliveryByID finds a delivery by ID
func (r *webhookRepository) FindDeliveryByID(ctx context.Context, id xid.ID) (*webhook.Delivery, error) {
	schemaDelivery := &schema.Delivery{}
	err := r.db.NewSelect().
		Model(schemaDelivery).
		Where("id = ?", id).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	delivery := &webhook.Delivery{
		ID:          schemaDelivery.ID,
		WebhookID:   schemaDelivery.WebhookID,
		EventID:     schemaDelivery.EventID,
		Attempt:     schemaDelivery.AttemptNumber,
		Status:      schemaDelivery.Status,
		DeliveredAt: schemaDelivery.DeliveredAt,
		CreatedAt:   schemaDelivery.CreatedAt,
		UpdatedAt:   schemaDelivery.UpdatedAt,
	}
	
	if schemaDelivery.StatusCode != nil {
		delivery.StatusCode = *schemaDelivery.StatusCode
	}
	if schemaDelivery.ResponseBody != nil {
		delivery.Response = string(schemaDelivery.ResponseBody)
	}
	if schemaDelivery.Error != nil {
		delivery.Error = *schemaDelivery.Error
	}
	
	return delivery, nil
}

// FindDeliveriesByWebhook finds deliveries by webhook ID
func (r *webhookRepository) FindDeliveriesByWebhook(ctx context.Context, webhookID xid.ID, status string, offset, limit int) ([]*webhook.Delivery, int64, error) {
	var schemaDeliveries []*schema.Delivery
	query := r.db.NewSelect().
		Model(&schemaDeliveries).
		Where("webhook_id = ?", webhookID)
	
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}
	
	// Count total
	countQuery := r.db.NewSelect().
		Model((*schema.Delivery)(nil)).
		Where("webhook_id = ?", webhookID)
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	
	// Convert to core types
	deliveries := make([]*webhook.Delivery, len(schemaDeliveries))
	for i, sd := range schemaDeliveries {
		delivery := &webhook.Delivery{
			ID:          sd.ID,
			WebhookID:   sd.WebhookID,
			EventID:     sd.EventID,
			Attempt:     sd.AttemptNumber,
			Status:      sd.Status,
			DeliveredAt: sd.DeliveredAt,
			CreatedAt:   sd.CreatedAt,
			UpdatedAt:   sd.UpdatedAt,
		}
		
		if sd.StatusCode != nil {
			delivery.StatusCode = *sd.StatusCode
		}
		if sd.ResponseBody != nil {
			delivery.Response = string(sd.ResponseBody)
		}
		if sd.Error != nil {
			delivery.Error = *sd.Error
		}
		
		deliveries[i] = delivery
	}
	
	return deliveries, int64(total), nil
}

// UpdateDelivery updates a delivery
func (r *webhookRepository) UpdateDelivery(ctx context.Context, delivery *webhook.Delivery) error {
	schemaDelivery := &schema.Delivery{
		ID:           delivery.ID,
		WebhookID:    delivery.WebhookID,
		EventID:      delivery.EventID,
		Status:       delivery.Status,
		StatusCode:   &delivery.StatusCode,
		ResponseBody: []byte(delivery.Response),
		Error:        &delivery.Error,
		DeliveredAt:  delivery.DeliveredAt,
		UpdatedAt:    time.Now(),
	}
	_, err := r.db.NewUpdate().
		Model(schemaDelivery).
		Where("id = ?", delivery.ID).
		Exec(ctx)
	return err
}

// FindPendingDeliveries finds deliveries that need retry
func (r *webhookRepository) FindPendingDeliveries(ctx context.Context, limit int) ([]*webhook.Delivery, error) {
	var schemaDeliveries []*schema.Delivery
	err := r.db.NewSelect().
		Model(&schemaDeliveries).
		Where("status IN (?, ?) AND (next_retry_at IS NULL OR next_retry_at <= ?)", 
			webhook.DeliveryStatusPending, webhook.DeliveryStatusRetrying, time.Now()).
		Order("created_at ASC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert to core types
	deliveries := make([]*webhook.Delivery, len(schemaDeliveries))
	for i, sd := range schemaDeliveries {
		delivery := &webhook.Delivery{
			ID:          sd.ID,
			WebhookID:   sd.WebhookID,
			EventID:     sd.EventID,
			Attempt:     sd.AttemptNumber,
			Status:      sd.Status,
			DeliveredAt: sd.DeliveredAt,
			CreatedAt:   sd.CreatedAt,
			UpdatedAt:   sd.UpdatedAt,
		}
		
		if sd.StatusCode != nil {
			delivery.StatusCode = *sd.StatusCode
		}
		if sd.ResponseBody != nil {
			delivery.Response = string(sd.ResponseBody)
		}
		if sd.Error != nil {
			delivery.Error = *sd.Error
		}
		
		deliveries[i] = delivery
	}
	
	return deliveries, nil
}