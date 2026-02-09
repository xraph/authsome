package repository

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// eventRepository implements EventRepository using Bun.
type eventRepository struct {
	db *bun.DB
}

// NewEventRepository creates a new event repository.
func NewEventRepository(db *bun.DB) EventRepository {
	return &eventRepository{db: db}
}

// Create creates a new event.
func (r *eventRepository) Create(ctx context.Context, event *schema.SubscriptionEvent) error {
	_, err := r.db.NewInsert().Model(event).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

// FindByID retrieves an event by ID.
func (r *eventRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionEvent, error) {
	event := new(schema.SubscriptionEvent)

	err := r.db.NewSelect().
		Model(event).
		Where("se.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find event: %w", err)
	}

	return event, nil
}

// List retrieves events with optional filters.
func (r *eventRepository) List(ctx context.Context, filter *EventFilter) ([]*schema.SubscriptionEvent, int, error) {
	var events []*schema.SubscriptionEvent

	query := r.db.NewSelect().
		Model(&events).
		Order("se.created_at DESC")

	if filter != nil {
		if filter.SubscriptionID != nil {
			query = query.Where("se.subscription_id = ?", *filter.SubscriptionID)
		}

		if filter.OrganizationID != nil {
			query = query.Where("se.organization_id = ?", *filter.OrganizationID)
		}

		if filter.EventType != "" {
			query = query.Where("se.event_type = ?", filter.EventType)
		}

		// Pagination
		pageSize := filter.PageSize
		if pageSize <= 0 {
			pageSize = 50
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
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	return events, count, nil
}
