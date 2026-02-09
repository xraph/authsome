package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// ImpersonationRepository implements the impersonation repository using Bun
// Updated for V2 architecture: App → Environment → Organization.
type ImpersonationRepository struct {
	db *bun.DB
}

// NewImpersonationRepository creates a new impersonation repository.
func NewImpersonationRepository(db *bun.DB) *ImpersonationRepository {
	return &ImpersonationRepository{db: db}
}

// Create creates a new impersonation session.
func (r *ImpersonationRepository) Create(ctx context.Context, session *schema.ImpersonationSession) error {
	_, err := r.db.NewInsert().
		Model(session).
		Exec(ctx)

	return err
}

// Get retrieves an impersonation session by ID and app (column organization_id contains appID).
func (r *ImpersonationRepository) Get(ctx context.Context, id xid.ID, appID xid.ID) (*schema.ImpersonationSession, error) {
	session := new(schema.ImpersonationSession)

	err := r.db.NewSelect().
		Model(session).
		Where("id = ? AND organization_id = ?", id, appID).
		Relation("Impersonator").
		Relation("TargetUser").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetBySessionID retrieves an impersonation session by the session ID.
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

// Update updates an impersonation session.
func (r *ImpersonationRepository) Update(ctx context.Context, session *schema.ImpersonationSession) error {
	session.UpdatedAt = time.Now().UTC()
	_, err := r.db.NewUpdate().
		Model(session).
		WherePK().
		Exec(ctx)

	return err
}

// ListSessions retrieves impersonation sessions with pagination and filtering
// Note: filter.AppID maps to column organization_id (V2 architecture).
func (r *ImpersonationRepository) ListSessions(ctx context.Context, filter *impersonation.ListSessionsFilter) (*pagination.PageResponse[*schema.ImpersonationSession], error) {
	var sessions []*schema.ImpersonationSession

	// Build base query
	query := r.db.NewSelect().
		Model(&sessions).
		Where("organization_id = ?", filter.AppID)

	// Apply filters
	if filter.EnvironmentID != nil && !filter.EnvironmentID.IsNil() {
		query = query.Where("environment_id = ?", *filter.EnvironmentID)
	}

	if filter.OrganizationID != nil && !filter.OrganizationID.IsNil() {
		query = query.Where("user_organization_id = ?", *filter.OrganizationID)
	}

	if filter.ImpersonatorID != nil {
		query = query.Where("impersonator_id = ?", *filter.ImpersonatorID)
	}

	if filter.TargetUserID != nil {
		query = query.Where("target_user_id = ?", *filter.TargetUserID)
	}

	if filter.ActiveOnly != nil && *filter.ActiveOnly {
		query = query.Where("active = ?", true).
			Where("expires_at > ?", time.Now().UTC()).
			Where("ended_at IS NULL")
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().
		Model((*schema.ImpersonationSession)(nil)).
		Where("organization_id = ?", filter.AppID)

	if filter.EnvironmentID != nil && !filter.EnvironmentID.IsNil() {
		countQuery = countQuery.Where("environment_id = ?", *filter.EnvironmentID)
	}

	if filter.OrganizationID != nil && !filter.OrganizationID.IsNil() {
		countQuery = countQuery.Where("user_organization_id = ?", *filter.OrganizationID)
	}

	if filter.ImpersonatorID != nil {
		countQuery = countQuery.Where("impersonator_id = ?", *filter.ImpersonatorID)
	}

	if filter.TargetUserID != nil {
		countQuery = countQuery.Where("target_user_id = ?", *filter.TargetUserID)
	}

	if filter.ActiveOnly != nil && *filter.ActiveOnly {
		countQuery = countQuery.Where("active = ?", true).
			Where("expires_at > ?", time.Now().UTC()).
			Where("ended_at IS NULL")
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	offset := filter.GetOffset()
	limit := filter.GetLimit()
	query = query.Limit(limit).Offset(offset)

	// Apply ordering
	query = query.Order("created_at DESC")

	// Load relations
	query = query.Relation("Impersonator").Relation("TargetUser")

	// Execute query
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(sessions, int64(total), &filter.PaginationParams), nil
}

// GetActive retrieves the active impersonation session for an impersonator
// Note: appID maps to column organization_id (V2 architecture).
func (r *ImpersonationRepository) GetActive(ctx context.Context, impersonatorID xid.ID, appID xid.ID) (*schema.ImpersonationSession, error) {
	session := new(schema.ImpersonationSession)

	err := r.db.NewSelect().
		Model(session).
		Where("impersonator_id = ? AND organization_id = ?", impersonatorID, appID).
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

// ExpireOldSessions expires sessions that have passed their expiry time.
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

// CreateAuditEvent creates an audit event.
func (r *ImpersonationRepository) CreateAuditEvent(ctx context.Context, event *schema.ImpersonationAuditEvent) error {
	_, err := r.db.NewInsert().
		Model(event).
		Exec(ctx)

	return err
}

// ListAuditEvents retrieves audit events with pagination and filtering
// Note: filter.AppID maps to column organization_id (V2 architecture).
func (r *ImpersonationRepository) ListAuditEvents(ctx context.Context, filter *impersonation.ListAuditEventsFilter) (*pagination.PageResponse[*schema.ImpersonationAuditEvent], error) {
	var events []*schema.ImpersonationAuditEvent

	// Build base query
	query := r.db.NewSelect().
		Model(&events).
		Where("organization_id = ?", filter.AppID)

	// Apply filters
	if filter.EnvironmentID != nil && !filter.EnvironmentID.IsNil() {
		query = query.Where("environment_id = ?", *filter.EnvironmentID)
	}

	if filter.OrganizationID != nil && !filter.OrganizationID.IsNil() {
		query = query.Where("user_organization_id = ?", *filter.OrganizationID)
	}

	if filter.ImpersonationID != nil {
		query = query.Where("impersonation_id = ?", *filter.ImpersonationID)
	}

	if filter.ImpersonatorID != nil {
		query = query.Where("impersonator_id = ?", *filter.ImpersonatorID)
	}

	if filter.TargetUserID != nil {
		query = query.Where("target_user_id = ?", *filter.TargetUserID)
	}

	if filter.EventType != nil && *filter.EventType != "" {
		query = query.Where("event_type = ?", *filter.EventType)
	}

	if filter.Since != nil {
		query = query.Where("created_at >= ?", *filter.Since)
	}

	if filter.Until != nil {
		query = query.Where("created_at <= ?", *filter.Until)
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().
		Model((*schema.ImpersonationAuditEvent)(nil)).
		Where("organization_id = ?", filter.AppID)

	if filter.EnvironmentID != nil && !filter.EnvironmentID.IsNil() {
		countQuery = countQuery.Where("environment_id = ?", *filter.EnvironmentID)
	}

	if filter.OrganizationID != nil && !filter.OrganizationID.IsNil() {
		countQuery = countQuery.Where("user_organization_id = ?", *filter.OrganizationID)
	}

	if filter.ImpersonationID != nil {
		countQuery = countQuery.Where("impersonation_id = ?", *filter.ImpersonationID)
	}

	if filter.ImpersonatorID != nil {
		countQuery = countQuery.Where("impersonator_id = ?", *filter.ImpersonatorID)
	}

	if filter.TargetUserID != nil {
		countQuery = countQuery.Where("target_user_id = ?", *filter.TargetUserID)
	}

	if filter.EventType != nil && *filter.EventType != "" {
		countQuery = countQuery.Where("event_type = ?", *filter.EventType)
	}

	if filter.Since != nil {
		countQuery = countQuery.Where("created_at >= ?", *filter.Since)
	}

	if filter.Until != nil {
		countQuery = countQuery.Where("created_at <= ?", *filter.Until)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	offset := filter.GetOffset()
	limit := filter.GetLimit()
	query = query.Limit(limit).Offset(offset)

	// Apply ordering
	query = query.Order("created_at DESC")

	// Load relations
	query = query.Relation("ImpersonationSession")

	// Execute query
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(events, int64(total), &filter.PaginationParams), nil
}
