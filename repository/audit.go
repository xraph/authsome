package repository

import (
	"context"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// AuditRepository implements core audit repository using Bun
type AuditRepository struct {
	db *bun.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *bun.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create creates a new audit event
func (r *AuditRepository) Create(ctx context.Context, e *schema.AuditEvent) error {
	_, err := r.db.NewInsert().Model(e).Exec(ctx)
	return err
}

// Get retrieves an audit event by ID
func (r *AuditRepository) Get(ctx context.Context, id xid.ID) (*schema.AuditEvent, error) {
	var event schema.AuditEvent
	err := r.db.NewSelect().
		Model(&event).
		Where("id = ?", id.String()).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return &event, nil
}

// List returns paginated audit events with optional filters
func (r *AuditRepository) List(ctx context.Context, filter *audit.ListEventsFilter) (*pagination.PageResponse[*schema.AuditEvent], error) {
	// Build base query
	baseQuery := r.db.NewSelect().Model((*schema.AuditEvent)(nil))

	// Apply filters
	baseQuery = r.applyFilters(baseQuery, filter)

	// Count total matching records
	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply sorting
	sortBy := "created_at"
	sortOrder := "DESC"
	if filter.SortBy != nil {
		sortBy = *filter.SortBy
	}
	if filter.SortOrder != nil {
		sortOrder = *filter.SortOrder
	}
	baseQuery = baseQuery.OrderExpr("? ?", bun.Ident(sortBy), bun.Safe(sortOrder))

	// Apply pagination
	baseQuery = baseQuery.Limit(filter.Limit).Offset(filter.Offset)

	// Execute query
	var events []*schema.AuditEvent
	if err := baseQuery.Scan(ctx); err != nil {
		return nil, err
	}

	// Create pagination params for NewPageResponse
	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	// Return paginated response
	return pagination.NewPageResponse(events, int64(total), params), nil
}

// applyFilters applies filter conditions to the query
func (r *AuditRepository) applyFilters(q *bun.SelectQuery, filter *audit.ListEventsFilter) *bun.SelectQuery {
	if filter.UserID != nil {
		q = q.Where("user_id = ?", filter.UserID.String())
	}

	if filter.Action != nil {
		q = q.Where("action = ?", *filter.Action)
	}

	if filter.Resource != nil {
		q = q.Where("resource = ?", *filter.Resource)
	}

	if filter.IPAddress != nil {
		q = q.Where("ip_address = ?", *filter.IPAddress)
	}

	if filter.Since != nil {
		q = q.Where("created_at >= ?", *filter.Since)
	}

	if filter.Until != nil {
		q = q.Where("created_at <= ?", *filter.Until)
	}

	return q
}
