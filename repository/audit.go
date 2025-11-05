package repository

import (
	"context"
	"github.com/uptrace/bun"
	core "github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/schema"
)

// AuditRepository implements core audit repository using Bun
type AuditRepository struct{ db *bun.DB }

func NewAuditRepository(db *bun.DB) *AuditRepository { return &AuditRepository{db: db} }

func (r *AuditRepository) toSchema(e *core.Event) *schema.AuditEvent {
	// Ensure auditable fields satisfy NOT NULL constraints
	createdBy := e.ID
	if e.UserID != nil {
		createdBy = *e.UserID
	}
	return &schema.AuditEvent{
		AuditableModel: schema.AuditableModel{
			CreatedBy: createdBy,
			UpdatedBy: createdBy,
		},
		ID:        e.ID,
		UserID:    e.UserID,
		Action:    e.Action,
		Resource:  e.Resource,
		IPAddress: e.IPAddress,
		UserAgent: e.UserAgent,
		Metadata:  e.Metadata,
	}
}

func (r *AuditRepository) fromSchema(ae *schema.AuditEvent) *core.Event {
	if ae == nil {
		return nil
	}
	return &core.Event{
		ID:        ae.ID,
		UserID:    ae.UserID,
		Action:    ae.Action,
		Resource:  ae.Resource,
		IPAddress: ae.IPAddress,
		UserAgent: ae.UserAgent,
		Metadata:  ae.Metadata,
		CreatedAt: ae.CreatedAt,
		UpdatedAt: ae.UpdatedAt.Time,
	}
}

func (r *AuditRepository) Create(ctx context.Context, e *core.Event) error {
	ae := r.toSchema(e)
	_, err := r.db.NewInsert().Model(ae).Exec(ctx)
	return err
}

// List returns recent audit events ordered by created_at desc
func (r *AuditRepository) List(ctx context.Context, limit, offset int) ([]*core.Event, error) {
	var rows []schema.AuditEvent
	q := r.db.NewSelect().Model(&rows).OrderExpr("created_at DESC").Limit(limit).Offset(offset)
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]*core.Event, 0, len(rows))
	for i := range rows {
		out = append(out, r.fromSchema(&rows[i]))
	}
	return out, nil
}

// Search returns events matching optional filters with pagination
// Filters: userId, action, since, until
func (r *AuditRepository) Search(ctx context.Context, params core.ListParams) ([]*core.Event, error) {
	var rows []schema.AuditEvent
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	q := r.db.NewSelect().Model(&rows).OrderExpr("created_at DESC").Limit(params.Limit).Offset(params.Offset)
	if params.UserID != nil {
		q = q.Where("user_id = ?", params.UserID.String())
	}
	if params.Action != "" {
		q = q.Where("action = ?", params.Action)
	}
	if params.Since != nil {
		q = q.Where("created_at >= ?", params.Since)
	}
	if params.Until != nil {
		q = q.Where("created_at <= ?", params.Until)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	out := make([]*core.Event, 0, len(rows))
	for i := range rows {
		out = append(out, r.fromSchema(&rows[i]))
	}
	return out, nil
}

// Count returns total number of audit events
func (r *AuditRepository) Count(ctx context.Context) (int, error) {
	q := r.db.NewSelect().Model((*schema.AuditEvent)(nil))
	return q.Count(ctx)
}

// SearchCount returns total number of events matching provided filters
func (r *AuditRepository) SearchCount(ctx context.Context, params core.ListParams) (int, error) {
	q := r.db.NewSelect().Model((*schema.AuditEvent)(nil))
	if params.UserID != nil {
		q = q.Where("user_id = ?", params.UserID.String())
	}
	if params.Action != "" {
		q = q.Where("action = ?", params.Action)
	}
	if params.Since != nil {
		q = q.Where("created_at >= ?", params.Since)
	}
	if params.Until != nil {
		q = q.Where("created_at <= ?", params.Until)
	}
	return q.Count(ctx)
}
