package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

// GetVerificationByID retrieves a verification by ID
func (r *IdentityVerificationRepository) GetVerificationByID(ctx context.Context, id string) (*schema.IdentityVerification, error) {
	verification := new(schema.IdentityVerification)
	
	err := r.db.NewSelect().
		Model(verification).
		Where("id = ?", id).
		Scan(ctx)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}
	
	return verification, nil
}

// GetVerificationsByUserID retrieves all verifications for a user
func (r *IdentityVerificationRepository) GetVerificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification
	
	err := r.db.NewSelect().
		Model(&verifications).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get verifications: %w", err)
	}
	
	return verifications, nil
}

// GetVerificationsByOrgID retrieves all verifications for an organization
func (r *IdentityVerificationRepository) GetVerificationsByOrgID(ctx context.Context, orgID string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification
	
	err := r.db.NewSelect().
		Model(&verifications).
		Where("organization_id = ?", orgID).
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

// DeleteVerification deletes a verification record
func (r *IdentityVerificationRepository) DeleteVerification(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*schema.IdentityVerification)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to delete verification: %w", err)
	}
	
	return nil
}

// GetLatestVerificationByUser retrieves the most recent verification for a user
func (r *IdentityVerificationRepository) GetLatestVerificationByUser(ctx context.Context, userID string) (*schema.IdentityVerification, error) {
	verification := new(schema.IdentityVerification)
	
	err := r.db.NewSelect().
		Model(verification).
		Where("user_id = ?", userID).
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

// GetVerificationByProviderCheckID retrieves a verification by provider check ID
func (r *IdentityVerificationRepository) GetVerificationByProviderCheckID(ctx context.Context, providerCheckID string) (*schema.IdentityVerification, error) {
	verification := new(schema.IdentityVerification)
	
	err := r.db.NewSelect().
		Model(verification).
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

// GetVerificationsByStatus retrieves verifications by status
func (r *IdentityVerificationRepository) GetVerificationsByStatus(ctx context.Context, status string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification
	
	err := r.db.NewSelect().
		Model(&verifications).
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

// GetVerificationsByType retrieves verifications by type
func (r *IdentityVerificationRepository) GetVerificationsByType(ctx context.Context, verificationType string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification
	
	err := r.db.NewSelect().
		Model(&verifications).
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

// CountVerificationsByUser counts verifications for a user since a given time
func (r *IdentityVerificationRepository) CountVerificationsByUser(ctx context.Context, userID string, since time.Time) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("user_id = ?", userID).
		Where("created_at >= ?", since).
		Count(ctx)
	
	if err != nil {
		return 0, fmt.Errorf("failed to count verifications: %w", err)
	}
	
	return count, nil
}

// GetExpiredVerifications retrieves expired verifications
func (r *IdentityVerificationRepository) GetExpiredVerifications(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerification, error) {
	var verifications []*schema.IdentityVerification
	
	err := r.db.NewSelect().
		Model(&verifications).
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

// GetDocumentByID retrieves a document by ID
func (r *IdentityVerificationRepository) GetDocumentByID(ctx context.Context, id string) (*schema.IdentityVerificationDocument, error) {
	document := new(schema.IdentityVerificationDocument)
	
	err := r.db.NewSelect().
		Model(document).
		Where("id = ?", id).
		Scan(ctx)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	
	return document, nil
}

// GetDocumentsByVerificationID retrieves all documents for a verification
func (r *IdentityVerificationRepository) GetDocumentsByVerificationID(ctx context.Context, verificationID string) ([]*schema.IdentityVerificationDocument, error) {
	var documents []*schema.IdentityVerificationDocument
	
	err := r.db.NewSelect().
		Model(&documents).
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

// DeleteDocument soft deletes a document record
func (r *IdentityVerificationRepository) DeleteDocument(ctx context.Context, id string) error {
	now := time.Now()
	
	_, err := r.db.NewUpdate().
		Model((*schema.IdentityVerificationDocument)(nil)).
		Set("deleted_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	
	return nil
}

// GetDocumentsForDeletion retrieves documents that should be deleted
func (r *IdentityVerificationRepository) GetDocumentsForDeletion(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerificationDocument, error) {
	var documents []*schema.IdentityVerificationDocument
	
	err := r.db.NewSelect().
		Model(&documents).
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

// GetSessionByID retrieves a session by ID
func (r *IdentityVerificationRepository) GetSessionByID(ctx context.Context, id string) (*schema.IdentityVerificationSession, error) {
	session := new(schema.IdentityVerificationSession)
	
	err := r.db.NewSelect().
		Model(session).
		Where("id = ?", id).
		Scan(ctx)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	return session, nil
}

// GetSessionsByUserID retrieves all sessions for a user
func (r *IdentityVerificationRepository) GetSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]*schema.IdentityVerificationSession, error) {
	var sessions []*schema.IdentityVerificationSession
	
	err := r.db.NewSelect().
		Model(&sessions).
		Where("user_id = ?", userID).
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

// DeleteSession deletes a session record
func (r *IdentityVerificationRepository) DeleteSession(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*schema.IdentityVerificationSession)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	return nil
}

// GetExpiredSessions retrieves expired sessions
func (r *IdentityVerificationRepository) GetExpiredSessions(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerificationSession, error) {
	var sessions []*schema.IdentityVerificationSession
	
	err := r.db.NewSelect().
		Model(&sessions).
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

// GetUserVerificationStatus retrieves the verification status for a user
func (r *IdentityVerificationRepository) GetUserVerificationStatus(ctx context.Context, userID string) (*schema.UserVerificationStatus, error) {
	status := new(schema.UserVerificationStatus)
	
	err := r.db.NewSelect().
		Model(status).
		Where("user_id = ?", userID).
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

// DeleteUserVerificationStatus deletes a user verification status
func (r *IdentityVerificationRepository) DeleteUserVerificationStatus(ctx context.Context, userID string) error {
	_, err := r.db.NewDelete().
		Model((*schema.UserVerificationStatus)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to delete user verification status: %w", err)
	}
	
	return nil
}

// GetUsersRequiringReverification retrieves users requiring re-verification
func (r *IdentityVerificationRepository) GetUsersRequiringReverification(ctx context.Context, limit int) ([]*schema.UserVerificationStatus, error) {
	var statuses []*schema.UserVerificationStatus
	
	err := r.db.NewSelect().
		Model(&statuses).
		Where("requires_reverification = ?", true).
		Limit(limit).
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get users requiring reverification: %w", err)
	}
	
	return statuses, nil
}

// GetUsersByVerificationLevel retrieves users by verification level
func (r *IdentityVerificationRepository) GetUsersByVerificationLevel(ctx context.Context, level string, limit, offset int) ([]*schema.UserVerificationStatus, error) {
	var statuses []*schema.UserVerificationStatus
	
	err := r.db.NewSelect().
		Model(&statuses).
		Where("verification_level = ?", level).
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get users by verification level: %w", err)
	}
	
	return statuses, nil
}

// GetBlockedUsers retrieves blocked users
func (r *IdentityVerificationRepository) GetBlockedUsers(ctx context.Context, limit, offset int) ([]*schema.UserVerificationStatus, error) {
	var statuses []*schema.UserVerificationStatus
	
	err := r.db.NewSelect().
		Model(&statuses).
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

// GetVerificationStats retrieves verification statistics
func (r *IdentityVerificationRepository) GetVerificationStats(ctx context.Context, orgID string, from, to time.Time) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"by_provider":  make(map[string]int64),
		"by_type":      make(map[string]int64),
		"by_risk_level": make(map[string]int64),
	}
	
	// Total verifications
	total, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total_verifications"] = int64(total)
	
	// Successful verifications
	successful, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where("is_verified = ?", true).
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get successful count: %w", err)
	}
	stats["successful_verifications"] = int64(successful)
	
	// Failed verifications
	failed, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where("status = ?", "failed").
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get failed count: %w", err)
	}
	stats["failed_verifications"] = int64(failed)
	
	// Pending verifications
	pending, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where("status IN (?)", bun.In([]string{"pending", "in_progress"})).
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}
	stats["pending_verifications"] = int64(pending)
	
	// High risk count
	highRisk, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where("risk_level = ?", "high").
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get high risk count: %w", err)
	}
	stats["high_risk_count"] = int64(highRisk)
	
	// Sanctions matches
	sanctions, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where("is_on_sanctions_list = ?", true).
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get sanctions count: %w", err)
	}
	stats["sanctions_matches"] = int64(sanctions)
	
	// PEP matches
	pep, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where("is_pep = ?", true).
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get PEP count: %w", err)
	}
	stats["pep_matches"] = int64(pep)
	
	return stats, nil
}

// GetProviderStats retrieves provider-specific statistics
func (r *IdentityVerificationRepository) GetProviderStats(ctx context.Context, provider string, from, to time.Time) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"provider": provider,
	}
	
	// Total checks
	total, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("provider = ?", provider).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get total checks: %w", err)
	}
	stats["total_checks"] = int64(total)
	
	// Successful checks
	successful, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("provider = ?", provider).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where("is_verified = ?", true).
		Count(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get successful checks: %w", err)
	}
	stats["successful_checks"] = int64(successful)
	
	// Failed checks
	failed, err := r.db.NewSelect().
		Model((*schema.IdentityVerification)(nil)).
		Where("provider = ?", provider).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where("status = ?", "failed").
		Count(ctx)
	
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

