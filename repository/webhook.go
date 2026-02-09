package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/schema"
)

// webhookRepository implements webhook.Repository.
type webhookRepository struct {
	db *bun.DB
}

// NewWebhookRepository creates a new webhook repository.
func NewWebhookRepository(db *bun.DB) webhook.Repository {
	return &webhookRepository{db: db}
}

// ===========================================================================
// WEBHOOK OPERATIONS
// ===========================================================================

// CreateWebhook creates a new webhook.
func (r *webhookRepository) CreateWebhook(ctx context.Context, wh *schema.Webhook) error {
	_, err := r.db.NewInsert().Model(wh).Exec(ctx)

	return err
}

// FindWebhookByID finds a webhook by ID.
func (r *webhookRepository) FindWebhookByID(ctx context.Context, id xid.ID) (*schema.Webhook, error) {
	var wh schema.Webhook

	err := r.db.NewSelect().
		Model(&wh).
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &wh, nil
}

// ListWebhooks lists webhooks with filtering and pagination.
func (r *webhookRepository) ListWebhooks(ctx context.Context, filter *webhook.ListWebhooksFilter) (*pagination.PageResponse[*schema.Webhook], error) {
	var webhooks []*schema.Webhook

	query := r.db.NewSelect().Model(&webhooks).Where("deleted_at IS NULL")

	// Apply filters
	query = query.Where("app_id = ?", filter.AppID)
	query = query.Where("environment_id = ?", filter.EnvironmentID)

	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}

	if filter.Event != nil {
		query = query.Where("? = ANY(events)", *filter.Event)
	}

	// Count total
	countQuery := r.db.NewSelect().Model((*schema.Webhook)(nil)).Where("deleted_at IS NULL")
	countQuery = countQuery.Where("app_id = ?", filter.AppID)

	countQuery = countQuery.Where("environment_id = ?", filter.EnvironmentID)
	if filter.Enabled != nil {
		countQuery = countQuery.Where("enabled = ?", *filter.Enabled)
	}

	if filter.Event != nil {
		countQuery = countQuery.Where("? = ANY(events)", *filter.Event)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination and ordering
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())
	query = query.Order("created_at DESC")

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(webhooks, int64(total), &filter.PaginationParams), nil
}

// FindWebhooksByAppAndEvent finds all enabled webhooks subscribed to an event.
func (r *webhookRepository) FindWebhooksByAppAndEvent(ctx context.Context, appID xid.ID, envID xid.ID, eventType string) ([]*schema.Webhook, error) {
	var webhooks []*schema.Webhook

	err := r.db.NewSelect().
		Model(&webhooks).
		Where("app_id = ? AND environment_id = ? AND enabled = true AND deleted_at IS NULL", appID, envID).
		Where("? = ANY(events)", eventType).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return webhooks, nil
}

// UpdateWebhook updates a webhook.
func (r *webhookRepository) UpdateWebhook(ctx context.Context, wh *schema.Webhook) error {
	wh.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(wh).
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)

	return err
}

// DeleteWebhook soft deletes a webhook.
func (r *webhookRepository) DeleteWebhook(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.Webhook)(nil)).
		Set("deleted_at = ?", time.Now()).
		Where("id = ? AND deleted_at IS NULL", id).
		Exec(ctx)

	return err
}

// UpdateFailureCount updates the failure count for a webhook.
func (r *webhookRepository) UpdateFailureCount(ctx context.Context, id xid.ID, count int) error {
	_, err := r.db.NewUpdate().
		Model((*schema.Webhook)(nil)).
		Set("failure_count = ?", count).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// UpdateLastDelivery updates the last delivery timestamp for a webhook.
func (r *webhookRepository) UpdateLastDelivery(ctx context.Context, id xid.ID, timestamp time.Time) error {
	_, err := r.db.NewUpdate().
		Model((*schema.Webhook)(nil)).
		Set("last_delivery = ?", timestamp).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// ===========================================================================
// EVENT OPERATIONS
// ===========================================================================

// CreateEvent creates a new webhook event.
func (r *webhookRepository) CreateEvent(ctx context.Context, event *schema.Event) error {
	_, err := r.db.NewInsert().Model(event).Exec(ctx)

	return err
}

// FindEventByID finds an event by ID.
func (r *webhookRepository) FindEventByID(ctx context.Context, id xid.ID) (*schema.Event, error) {
	var event schema.Event

	err := r.db.NewSelect().
		Model(&event).
		Where("id = ?", id).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &event, nil
}

// ListEvents lists events with filtering and pagination.
func (r *webhookRepository) ListEvents(ctx context.Context, filter *webhook.ListEventsFilter) (*pagination.PageResponse[*schema.Event], error) {
	var events []*schema.Event

	query := r.db.NewSelect().Model(&events)

	// Apply filters
	query = query.Where("app_id = ?", filter.AppID)
	query = query.Where("environment_id = ?", filter.EnvironmentID)

	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
	}

	// Count total
	countQuery := r.db.NewSelect().Model((*schema.Event)(nil))
	countQuery = countQuery.Where("app_id = ?", filter.AppID)

	countQuery = countQuery.Where("environment_id = ?", filter.EnvironmentID)
	if filter.Type != nil {
		countQuery = countQuery.Where("type = ?", *filter.Type)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination and ordering
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())
	query = query.Order("occurred_at DESC")

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(events, int64(total), &filter.PaginationParams), nil
}

// ===========================================================================
// DELIVERY OPERATIONS
// ===========================================================================

// CreateDelivery creates a new webhook delivery.
func (r *webhookRepository) CreateDelivery(ctx context.Context, delivery *schema.Delivery) error {
	_, err := r.db.NewInsert().Model(delivery).Exec(ctx)

	return err
}

// FindDeliveryByID finds a delivery by ID.
func (r *webhookRepository) FindDeliveryByID(ctx context.Context, id xid.ID) (*schema.Delivery, error) {
	var delivery schema.Delivery

	err := r.db.NewSelect().
		Model(&delivery).
		Where("id = ?", id).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &delivery, nil
}

// ListDeliveries lists deliveries with filtering and pagination.
func (r *webhookRepository) ListDeliveries(ctx context.Context, filter *webhook.ListDeliveriesFilter) (*pagination.PageResponse[*schema.Delivery], error) {
	var deliveries []*schema.Delivery

	query := r.db.NewSelect().Model(&deliveries)

	// Apply filters
	query = query.Where("webhook_id = ?", filter.WebhookID)

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Count total
	countQuery := r.db.NewSelect().Model((*schema.Delivery)(nil))

	countQuery = countQuery.Where("webhook_id = ?", filter.WebhookID)
	if filter.Status != nil {
		countQuery = countQuery.Where("status = ?", *filter.Status)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination and ordering
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())
	query = query.Order("created_at DESC")

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(deliveries, int64(total), &filter.PaginationParams), nil
}

// UpdateDelivery updates a delivery.
func (r *webhookRepository) UpdateDelivery(ctx context.Context, delivery *schema.Delivery) error {
	delivery.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(delivery).
		WherePK().
		Exec(ctx)

	return err
}

// FindPendingDeliveries finds deliveries that need retry.
func (r *webhookRepository) FindPendingDeliveries(ctx context.Context, limit int) ([]*schema.Delivery, error) {
	var deliveries []*schema.Delivery

	err := r.db.NewSelect().
		Model(&deliveries).
		Where("status IN (?, ?) AND (next_retry_at IS NULL OR next_retry_at <= ?)",
			webhook.DeliveryStatusPending, webhook.DeliveryStatusRetrying, time.Now()).
		Order("created_at ASC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return deliveries, nil
}
