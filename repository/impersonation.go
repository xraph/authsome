package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/schema"
)

// ImpersonationRepository implements the impersonation repository using Bun
type ImpersonationRepository struct {
	db *bun.DB
}

// NewImpersonationRepository creates a new impersonation repository
func NewImpersonationRepository(db *bun.DB) *ImpersonationRepository {
	return &ImpersonationRepository{db: db}
}

// Create creates a new impersonation session
func (r *ImpersonationRepository) Create(ctx context.Context, session *schema.ImpersonationSession) error {
	_, err := r.db.NewInsert().
		Model(session).
		Exec(ctx)
	return err
}

// Get retrieves an impersonation session by ID and organization
func (r *ImpersonationRepository) Get(ctx context.Context, id xid.ID, orgID xid.ID) (*schema.ImpersonationSession, error) {
	session := new(schema.ImpersonationSession)
	err := r.db.NewSelect().
		Model(session).
		Where("id = ? AND organization_id = ?", id, orgID).
		Relation("Impersonator").
		Relation("TargetUser").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// GetBySessionID retrieves an impersonation session by the session ID
func (r *ImpersonationRepository) GetBySessionID(ctx context.Context, sessionID xid.ID) (*schema.ImpersonationSession, error) {
	session := new(schema.ImpersonationSession)
	err := r.db.NewSelect().
		Model(session).
		Where("new_session_id = ?", sessionID).
		Where("active = ?", true).
		Relation("Impersonator").
		Relation("TargetUser").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// Update updates an impersonation session
func (r *ImpersonationRepository) Update(ctx context.Context, session *schema.ImpersonationSession) error {
	session.UpdatedAt = time.Now().UTC()
	_, err := r.db.NewUpdate().
		Model(session).
		WherePK().
		Exec(ctx)
	return err
}

// List retrieves impersonation sessions with filters
func (r *ImpersonationRepository) List(ctx context.Context, req *impersonation.ListRequest) ([]*schema.ImpersonationSession, error) {
	var sessions []*schema.ImpersonationSession

	query := r.db.NewSelect().
		Model(&sessions).
		Where("organization_id = ?", req.OrganizationID)

	if req.ImpersonatorID != nil {
		query = query.Where("impersonator_id = ?", *req.ImpersonatorID)
	}

	if req.TargetUserID != nil {
		query = query.Where("target_user_id = ?", *req.TargetUserID)
	}

	if req.ActiveOnly {
		query = query.Where("active = ?", true).
			Where("expires_at > ?", time.Now().UTC()).
			Where("ended_at IS NULL")
	}

	// Pagination
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Offset > 0 {
		query = query.Offset(req.Offset)
	}

	// Order by most recent first
	query = query.Order("created_at DESC")

	// Load relations
	query = query.Relation("Impersonator").Relation("TargetUser")

	err := query.Scan(ctx)
	return sessions, err
}

// Count counts impersonation sessions with filters
func (r *ImpersonationRepository) Count(ctx context.Context, req *impersonation.ListRequest) (int, error) {
	query := r.db.NewSelect().
		Model((*schema.ImpersonationSession)(nil)).
		Where("organization_id = ?", req.OrganizationID)

	if req.ImpersonatorID != nil {
		query = query.Where("impersonator_id = ?", *req.ImpersonatorID)
	}

	if req.TargetUserID != nil {
		query = query.Where("target_user_id = ?", *req.TargetUserID)
	}

	if req.ActiveOnly {
		query = query.Where("active = ?", true).
			Where("expires_at > ?", time.Now().UTC()).
			Where("ended_at IS NULL")
	}

	return query.Count(ctx)
}

// GetActive retrieves the active impersonation session for an impersonator
func (r *ImpersonationRepository) GetActive(ctx context.Context, impersonatorID xid.ID, orgID xid.ID) (*schema.ImpersonationSession, error) {
	session := new(schema.ImpersonationSession)
	err := r.db.NewSelect().
		Model(session).
		Where("impersonator_id = ? AND organization_id = ?", impersonatorID, orgID).
		Where("active = ?", true).
		Where("expires_at > ?", time.Now().UTC()).
		Where("ended_at IS NULL").
		Relation("Impersonator").
		Relation("TargetUser").
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// ExpireOldSessions expires sessions that have passed their expiry time
func (r *ImpersonationRepository) ExpireOldSessions(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	result, err := r.db.NewUpdate().
		Model((*schema.ImpersonationSession)(nil)).
		Set("active = ?", false).
		Set("ended_at = ?", now).
		Set("end_reason = ?", "timeout").
		Set("updated_at = ?", now).
		Where("active = ?", true).
		Where("expires_at <= ?", now).
		Where("ended_at IS NULL").
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// CreateAuditEvent creates an audit event
func (r *ImpersonationRepository) CreateAuditEvent(ctx context.Context, event *schema.ImpersonationAuditEvent) error {
	_, err := r.db.NewInsert().
		Model(event).
		Exec(ctx)
	return err
}

// ListAuditEvents retrieves audit events with filters
func (r *ImpersonationRepository) ListAuditEvents(ctx context.Context, req *impersonation.AuditListRequest) ([]*schema.ImpersonationAuditEvent, error) {
	var events []*schema.ImpersonationAuditEvent

	query := r.db.NewSelect().
		Model(&events).
		Where("organization_id = ?", req.OrganizationID)

	if req.ImpersonationID != nil {
		query = query.Where("impersonation_id = ?", *req.ImpersonationID)
	}

	if req.EventType != "" {
		query = query.Where("event_type = ?", req.EventType)
	}

	if req.Since != nil {
		query = query.Where("created_at >= ?", *req.Since)
	}

	if req.Until != nil {
		query = query.Where("created_at <= ?", *req.Until)
	}

	// Pagination
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Offset > 0 {
		query = query.Offset(req.Offset)
	}

	// Order by most recent first
	query = query.Order("created_at DESC")

	// Load relations
	query = query.Relation("ImpersonationSession")

	err := query.Scan(ctx)
	return events, err
}

// CountAuditEvents counts audit events with filters
func (r *ImpersonationRepository) CountAuditEvents(ctx context.Context, req *impersonation.AuditListRequest) (int, error) {
	query := r.db.NewSelect().
		Model((*schema.ImpersonationAuditEvent)(nil)).
		Where("organization_id = ?", req.OrganizationID)

	if req.ImpersonationID != nil {
		query = query.Where("impersonation_id = ?", *req.ImpersonationID)
	}

	if req.EventType != "" {
		query = query.Where("event_type = ?", req.EventType)
	}

	if req.Since != nil {
		query = query.Where("created_at >= ?", *req.Since)
	}

	if req.Until != nil {
		query = query.Where("created_at <= ?", *req.Until)
	}

	return query.Count(ctx)
}
