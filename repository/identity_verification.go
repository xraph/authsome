package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// IdentityVerificationRepository implements the identity verification repository
type IdentityVerificationRepository struct {
	db *bun.DB
}

// NewIdentityVerificationRepository creates a new identity verification repository
func NewIdentityVerificationRepository(db *bun.DB) *IdentityVerificationRepository {
	return &IdentityVerificationRepository{db: db}
}

// CreateVerification creates a new verification record
func (r *IdentityVerificationRepository) CreateVerification(ctx context.Context, verification *schema.IdentityVerification) error {
	_, err := r.db.NewInsert().
		Model(verification).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create verification: %w", err)
	}

	return nil
}

// GetVerificationByID retrieves a verification by ID with V2 context filtering
func (r *IdentityVerificationRepository) GetVerificationByID(ctx context.Context, appID xid.ID, id string) (*schema.IdentityVerification, error) {
	verification := new(schema.IdentityVerification)

	err := r.db.NewSelect().
		Model(verification).
		Where("id = ?", id).
		Where("app_id = ?", appID.String()).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	return verification, nil
}

// GetVerificationsByUserID retrieves all verifications for a user with V2 context filtering
func (r *IdentityVerificationRepository) GetVerificationsByUserID(ctx context.Context, appID xid.ID, userID xid.ID, limit, offset int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification

	err := r.db.NewSelect().
		Model(&verifications).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get verifications: %w", err)
	}

	return verifications, nil
}

// GetVerificationsByOrgID retrieves all verifications for an organization with V2 context filtering
func (r *IdentityVerificationRepository) GetVerificationsByOrgID(ctx context.Context, appID xid.ID, orgID xid.ID, limit, offset int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification

	err := r.db.NewSelect().
		Model(&verifications).
		Where("app_id = ?", appID.String()).
		Where("organization_id = ?", orgID.String()).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get verifications: %w", err)
	}

	return verifications, nil
}

// UpdateVerification updates a verification record
func (r *IdentityVerificationRepository) UpdateVerification(ctx context.Context, verification *schema.IdentityVerification) error {
	verification.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(verification).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update verification: %w", err)
	}

	return nil
}

// DeleteVerification deletes a verification record with V2 context filtering
func (r *IdentityVerificationRepository) DeleteVerification(ctx context.Context, appID xid.ID, id string) error {
	_, err := r.db.NewDelete().
		Model((*schema.IdentityVerification)(nil)).
		Where("id = ?", id).
		Where("app_id = ?", appID.String()).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete verification: %w", err)
	}

	return nil
}

// GetLatestVerificationByUser retrieves the most recent verification for a user with V2 context filtering
func (r *IdentityVerificationRepository) GetLatestVerificationByUser(ctx context.Context, appID xid.ID, userID xid.ID) (*schema.IdentityVerification, error) {
	verification := new(schema.IdentityVerification)

	err := r.db.NewSelect().
		Model(verification).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get latest verification: %w", err)
	}

	return verification, nil
}

// GetVerificationByProviderCheckID retrieves a verification by provider check ID with V2 context filtering
func (r *IdentityVerificationRepository) GetVerificationByProviderCheckID(ctx context.Context, appID xid.ID, providerCheckID string) (*schema.IdentityVerification, error) {
	verification := new(schema.IdentityVerification)

	err := r.db.NewSelect().
		Model(verification).
		Where("app_id = ?", appID.String()).
		Where("provider_check_id = ?", providerCheckID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	return verification, nil
}

// GetVerificationsByStatus retrieves verifications by status with V2 context filtering
func (r *IdentityVerificationRepository) GetVerificationsByStatus(ctx context.Context, appID xid.ID, status string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification

	err := r.db.NewSelect().
		Model(&verifications).
		Where("app_id = ?", appID.String()).
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get verifications: %w", err)
	}

	return verifications, nil
}

// GetVerificationsByType retrieves verifications by type with V2 context filtering
func (r *IdentityVerificationRepository) GetVerificationsByType(ctx context.Context, appID xid.ID, verificationType string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification

	err := r.db.NewSelect().
		Model(&verifications).
		Where("app_id = ?", appID.String()).
		Where("verification_type = ?", verificationType).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get verifications: %w", err)
	}

	return verifications, nil
}

// CountVerificationsByUser counts verifications for a user since a given time with V2 context filtering
func (r *IdentityVerificationRepository) CountVerificationsByUser(ctx context.Context, appID xid.ID, userID xid.ID, since time.Time) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		Where("created_at >= ?", since).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to count verifications: %w", err)
	}

	return count, nil
}

// GetExpiredVerifications retrieves expired verifications with V2 context filtering
func (r *IdentityVerificationRepository) GetExpiredVerifications(ctx context.Context, appID xid.ID, before time.Time, limit int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification

	err := r.db.NewSelect().
		Model(&verifications).
		Where("app_id = ?", appID.String()).
		Where("expires_at IS NOT NULL").
		Where("expires_at < ?", before).
		Where("status != ?", "expired").
		Limit(limit).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get expired verifications: %w", err)
	}

	return verifications, nil
}

// Document operations

// CreateDocument creates a new document record
func (r *IdentityVerificationRepository) CreateDocument(ctx context.Context, document *schema.IdentityVerificationDocument) error {
	_, err := r.db.NewInsert().
		Model(document).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	return nil
}

// GetDocumentByID retrieves a document by ID with V2 context filtering
func (r *IdentityVerificationRepository) GetDocumentByID(ctx context.Context, appID xid.ID, id string) (*schema.IdentityVerificationDocument, error) {
	document := new(schema.IdentityVerificationDocument)

	err := r.db.NewSelect().
		Model(document).
		Where("id = ?", id).
		Where("app_id = ?", appID.String()).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return document, nil
}

// GetDocumentsByVerificationID retrieves all documents for a verification with V2 context filtering
func (r *IdentityVerificationRepository) GetDocumentsByVerificationID(ctx context.Context, appID xid.ID, verificationID string) ([]*schema.IdentityVerificationDocument, error) {
	var documents []*schema.IdentityVerificationDocument

	err := r.db.NewSelect().
		Model(&documents).
		Where("app_id = ?", appID.String()).
		Where("verification_id = ?", verificationID).
		Where("deleted_at IS NULL").
		Order("created_at ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	return documents, nil
}

// UpdateDocument updates a document record
func (r *IdentityVerificationRepository) UpdateDocument(ctx context.Context, document *schema.IdentityVerificationDocument) error {
	document.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(document).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

// DeleteDocument soft deletes a document record with V2 context filtering
func (r *IdentityVerificationRepository) DeleteDocument(ctx context.Context, appID xid.ID, id string) error {
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*schema.IdentityVerificationDocument)(nil)).
		Set("deleted_at = ?", now).
		Where("id = ?", id).
		Where("app_id = ?", appID.String()).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// GetDocumentsForDeletion retrieves documents that should be deleted with V2 context filtering
func (r *IdentityVerificationRepository) GetDocumentsForDeletion(ctx context.Context, appID xid.ID, before time.Time, limit int) ([]*schema.IdentityVerificationDocument, error) {
	var documents []*schema.IdentityVerificationDocument

	err := r.db.NewSelect().
		Model(&documents).
		Where("app_id = ?", appID.String()).
		Where("retain_until IS NOT NULL").
		Where("retain_until < ?", before).
		Where("deleted_at IS NULL").
		Limit(limit).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get documents for deletion: %w", err)
	}

	return documents, nil
}

// Session operations

// CreateSession creates a new session record
func (r *IdentityVerificationRepository) CreateSession(ctx context.Context, session *schema.IdentityVerificationSession) error {
	_, err := r.db.NewInsert().
		Model(session).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSessionByID retrieves a session by ID with V2 context filtering
func (r *IdentityVerificationRepository) GetSessionByID(ctx context.Context, appID xid.ID, id string) (*schema.IdentityVerificationSession, error) {
	session := new(schema.IdentityVerificationSession)

	err := r.db.NewSelect().
		Model(session).
		Where("id = ?", id).
		Where("app_id = ?", appID.String()).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetSessionsByUserID retrieves all sessions for a user with V2 context filtering
func (r *IdentityVerificationRepository) GetSessionsByUserID(ctx context.Context, appID xid.ID, userID xid.ID, limit, offset int) ([]*schema.IdentityVerificationSession, error) {
	var sessions []*schema.IdentityVerificationSession

	err := r.db.NewSelect().
		Model(&sessions).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	return sessions, nil
}

// UpdateSession updates a session record
func (r *IdentityVerificationRepository) UpdateSession(ctx context.Context, session *schema.IdentityVerificationSession) error {
	session.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(session).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// DeleteSession deletes a session record with V2 context filtering
func (r *IdentityVerificationRepository) DeleteSession(ctx context.Context, appID xid.ID, id string) error {
	_, err := r.db.NewDelete().
		Model((*schema.IdentityVerificationSession)(nil)).
		Where("id = ?", id).
		Where("app_id = ?", appID.String()).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// GetExpiredSessions retrieves expired sessions with V2 context filtering
func (r *IdentityVerificationRepository) GetExpiredSessions(ctx context.Context, appID xid.ID, before time.Time, limit int) ([]*schema.IdentityVerificationSession, error) {
	var sessions []*schema.IdentityVerificationSession

	err := r.db.NewSelect().
		Model(&sessions).
		Where("app_id = ?", appID.String()).
		Where("expires_at < ?", before).
		Where("status != ?", "expired").
		Limit(limit).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get expired sessions: %w", err)
	}

	return sessions, nil
}

// User verification status operations

// CreateUserVerificationStatus creates a new user verification status
func (r *IdentityVerificationRepository) CreateUserVerificationStatus(ctx context.Context, status *schema.UserVerificationStatus) error {
	_, err := r.db.NewInsert().
		Model(status).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create user verification status: %w", err)
	}

	return nil
}

// GetUserVerificationStatus retrieves the verification status for a user with V2 context filtering
func (r *IdentityVerificationRepository) GetUserVerificationStatus(ctx context.Context, appID xid.ID, orgID xid.ID, userID xid.ID) (*schema.UserVerificationStatus, error) {
	status := new(schema.UserVerificationStatus)

	err := r.db.NewSelect().
		Model(status).
		Where("app_id = ?", appID.String()).
		Where("organization_id = ?", orgID.String()).
		Where("user_id = ?", userID.String()).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user verification status: %w", err)
	}

	return status, nil
}

// UpdateUserVerificationStatus updates a user verification status
func (r *IdentityVerificationRepository) UpdateUserVerificationStatus(ctx context.Context, status *schema.UserVerificationStatus) error {
	status.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(status).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update user verification status: %w", err)
	}

	return nil
}

// DeleteUserVerificationStatus deletes a user verification status with V2 context filtering
func (r *IdentityVerificationRepository) DeleteUserVerificationStatus(ctx context.Context, appID xid.ID, orgID xid.ID, userID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.UserVerificationStatus)(nil)).
		Where("app_id = ?", appID.String()).
		Where("organization_id = ?", orgID.String()).
		Where("user_id = ?", userID.String()).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete user verification status: %w", err)
	}

	return nil
}

// GetUsersRequiringReverification retrieves users requiring re-verification with V2 context filtering
func (r *IdentityVerificationRepository) GetUsersRequiringReverification(ctx context.Context, appID xid.ID, limit int) ([]*schema.UserVerificationStatus, error) {
	var statuses []*schema.UserVerificationStatus

	err := r.db.NewSelect().
		Model(&statuses).
		Where("app_id = ?", appID.String()).
		Where("requires_reverification = ?", true).
		Limit(limit).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get users requiring reverification: %w", err)
	}

	return statuses, nil
}

// GetUsersByVerificationLevel retrieves users by verification level with V2 context filtering
func (r *IdentityVerificationRepository) GetUsersByVerificationLevel(ctx context.Context, appID xid.ID, level string, limit, offset int) ([]*schema.UserVerificationStatus, error) {
	var statuses []*schema.UserVerificationStatus

	err := r.db.NewSelect().
		Model(&statuses).
		Where("app_id = ?", appID.String()).
		Where("verification_level = ?", level).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get users by verification level: %w", err)
	}

	return statuses, nil
}

// GetBlockedUsers retrieves blocked users with V2 context filtering
func (r *IdentityVerificationRepository) GetBlockedUsers(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.UserVerificationStatus, error) {
	var statuses []*schema.UserVerificationStatus

	err := r.db.NewSelect().
		Model(&statuses).
		Where("app_id = ?", appID.String()).
		Where("is_blocked = ?", true).
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get blocked users: %w", err)
	}

	return statuses, nil
}

// Analytics and reporting

// GetVerificationStats retrieves verification statistics with V2 context filtering
func (r *IdentityVerificationRepository) GetVerificationStats(ctx context.Context, appID xid.ID, orgID xid.ID, from, to time.Time) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"by_provider":   make(map[string]int64),
		"by_type":       make(map[string]int64),
		"by_risk_level": make(map[string]int64),
	}

	baseQuery := func() *bun.SelectQuery {
		return r.db.NewSelect().
			Model((*schema.IdentityVerification)(nil)).
			Where("app_id = ?", appID.String()).
			Where("organization_id = ?", orgID.String()).
			Where("created_at >= ?", from).
			Where("created_at <= ?", to)
	}

	// Total verifications
	total, err := baseQuery().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total_verifications"] = int64(total)

	// Successful verifications
	successful, err := baseQuery().Where("is_verified = ?", true).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get successful count: %w", err)
	}
	stats["successful_verifications"] = int64(successful)

	// Failed verifications
	failed, err := baseQuery().Where("status = ?", "failed").Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed count: %w", err)
	}
	stats["failed_verifications"] = int64(failed)

	// Pending verifications
	pending, err := baseQuery().Where("status IN (?)", bun.In([]string{"pending", "in_progress"})).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}
	stats["pending_verifications"] = int64(pending)

	// High risk count
	highRisk, err := baseQuery().Where("risk_level = ?", "high").Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get high risk count: %w", err)
	}
	stats["high_risk_count"] = int64(highRisk)

	// Sanctions matches
	sanctions, err := baseQuery().Where("is_on_sanctions_list = ?", true).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sanctions count: %w", err)
	}
	stats["sanctions_matches"] = int64(sanctions)

	// PEP matches
	pep, err := baseQuery().Where("is_pep = ?", true).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get PEP count: %w", err)
	}
	stats["pep_matches"] = int64(pep)

	return stats, nil
}

// GetProviderStats retrieves provider-specific statistics with V2 context filtering
func (r *IdentityVerificationRepository) GetProviderStats(ctx context.Context, appID xid.ID, provider string, from, to time.Time) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"provider": provider,
	}

	baseQuery := func() *bun.SelectQuery {
		return r.db.NewSelect().
			Model((*schema.IdentityVerification)(nil)).
			Where("app_id = ?", appID.String()).
			Where("provider = ?", provider).
			Where("created_at >= ?", from).
			Where("created_at <= ?", to)
	}

	// Total checks
	total, err := baseQuery().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total checks: %w", err)
	}
	stats["total_checks"] = int64(total)

	// Successful checks
	successful, err := baseQuery().Where("is_verified = ?", true).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get successful checks: %w", err)
	}
	stats["successful_checks"] = int64(successful)

	// Failed checks
	failed, err := baseQuery().Where("status = ?", "failed").Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed checks: %w", err)
	}
	stats["failed_checks"] = int64(failed)

	// Calculate error rate
	if total > 0 {
		stats["error_rate"] = float64(failed) / float64(total) * 100
	}

	return stats, nil
}
