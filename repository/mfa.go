package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// MFARepository provides persistence for MFA entities.
type MFARepository struct {
	db *bun.DB
}

// NewMFARepository creates a new MFA repository.
func NewMFARepository(db *bun.DB) *MFARepository {
	return &MFARepository{db: db}
}

// DB returns the underlying database connection.
func (r *MFARepository) DB() *bun.DB {
	return r.db
}

// ==================== Factor Operations ====================

// CreateFactor creates a new MFA factor.
func (r *MFARepository) CreateFactor(ctx context.Context, factor *schema.MFAFactor) error {
	_, err := r.db.NewInsert().Model(factor).Exec(ctx)

	return err
}

// GetFactor retrieves a factor by ID.
func (r *MFARepository) GetFactor(ctx context.Context, factorID xid.ID) (*schema.MFAFactor, error) {
	factor := new(schema.MFAFactor)

	err := r.db.NewSelect().Model(factor).Where("id = ?", factorID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return factor, err
}

// ListUserFactors retrieves all factors for a user.
func (r *MFARepository) ListUserFactors(ctx context.Context, userID xid.ID, statusFilter ...string) ([]*schema.MFAFactor, error) {
	var factors []*schema.MFAFactor

	query := r.db.NewSelect().Model(&factors).Where("user_id = ?", userID)

	if len(statusFilter) > 0 {
		query = query.Where("status IN (?)", bun.In(statusFilter))
	}

	err := query.Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return factors, nil
}

// UpdateFactor updates a factor.
func (r *MFARepository) UpdateFactor(ctx context.Context, factor *schema.MFAFactor) error {
	_, err := r.db.NewUpdate().Model(factor).WherePK().Exec(ctx)

	return err
}

// DeleteFactor deletes a factor.
func (r *MFARepository) DeleteFactor(ctx context.Context, factorID xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.MFAFactor)(nil)).Where("id = ?", factorID).Exec(ctx)

	return err
}

// UpdateFactorLastUsed updates the last used timestamp.
func (r *MFARepository) UpdateFactorLastUsed(ctx context.Context, factorID xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.MFAFactor)(nil)).
		Set("last_used_at = ?", now).
		Where("id = ?", factorID).
		Exec(ctx)

	return err
}

// ==================== Challenge Operations ====================

// CreateChallenge creates a new MFA challenge.
func (r *MFARepository) CreateChallenge(ctx context.Context, challenge *schema.MFAChallenge) error {
	_, err := r.db.NewInsert().Model(challenge).Exec(ctx)

	return err
}

// GetChallenge retrieves a challenge by ID.
func (r *MFARepository) GetChallenge(ctx context.Context, challengeID xid.ID) (*schema.MFAChallenge, error) {
	challenge := new(schema.MFAChallenge)

	err := r.db.NewSelect().Model(challenge).Where("id = ?", challengeID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return challenge, err
}

// UpdateChallenge updates a challenge.
func (r *MFARepository) UpdateChallenge(ctx context.Context, challenge *schema.MFAChallenge) error {
	_, err := r.db.NewUpdate().Model(challenge).WherePK().Exec(ctx)

	return err
}

// IncrementChallengeAttempts increments the attempt counter.
func (r *MFARepository) IncrementChallengeAttempts(ctx context.Context, challengeID xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.MFAChallenge)(nil)).
		Set("attempts = attempts + 1").
		Where("id = ?", challengeID).
		Exec(ctx)

	return err
}

// CleanupExpiredChallenges removes expired challenges.
func (r *MFARepository) CleanupExpiredChallenges(ctx context.Context) (int, error) {
	result, err := r.db.NewDelete().
		Model((*schema.MFAChallenge)(nil)).
		Where("expires_at < ?", time.Now()).
		Where("status = ?", "pending").
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()

	return int(rows), nil
}

// ==================== Session Operations ====================

// CreateSession creates a new MFA session.
func (r *MFARepository) CreateSession(ctx context.Context, session *schema.MFASession) error {
	_, err := r.db.NewInsert().Model(session).Exec(ctx)

	return err
}

// GetSession retrieves a session by ID.
func (r *MFARepository) GetSession(ctx context.Context, sessionID xid.ID) (*schema.MFASession, error) {
	session := new(schema.MFASession)

	err := r.db.NewSelect().Model(session).Where("id = ?", sessionID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return session, err
}

// GetSessionByToken retrieves a session by token.
func (r *MFARepository) GetSessionByToken(ctx context.Context, token string) (*schema.MFASession, error) {
	session := new(schema.MFASession)

	err := r.db.NewSelect().Model(session).
		Where("session_token = ?", token).
		Where("expires_at > ?", time.Now()).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return session, err
}

// UpdateSession updates a session.
func (r *MFARepository) UpdateSession(ctx context.Context, session *schema.MFASession) error {
	_, err := r.db.NewUpdate().Model(session).WherePK().Exec(ctx)

	return err
}

// CompleteSession marks a session as completed.
func (r *MFARepository) CompleteSession(ctx context.Context, sessionID xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.MFASession)(nil)).
		Set("completed_at = ?", now).
		Set("factors_verified = factors_required").
		Where("id = ?", sessionID).
		Exec(ctx)

	return err
}

// CleanupExpiredSessions removes expired sessions.
func (r *MFARepository) CleanupExpiredSessions(ctx context.Context) (int, error) {
	result, err := r.db.NewDelete().
		Model((*schema.MFASession)(nil)).
		Where("expires_at < ?", time.Now()).
		Where("completed_at IS NULL").
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()

	return int(rows), nil
}

// ==================== Trusted Device Operations ====================

// CreateTrustedDevice creates a new trusted device.
func (r *MFARepository) CreateTrustedDevice(ctx context.Context, device *schema.MFATrustedDevice) error {
	_, err := r.db.NewInsert().Model(device).Exec(ctx)

	return err
}

// GetTrustedDevice retrieves a trusted device.
func (r *MFARepository) GetTrustedDevice(ctx context.Context, userID xid.ID, deviceID string) (*schema.MFATrustedDevice, error) {
	device := new(schema.MFATrustedDevice)

	err := r.db.NewSelect().Model(device).
		Where("user_id = ?", userID).
		Where("device_id = ?", deviceID).
		Where("expires_at > ?", time.Now()).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return device, err
}

// ListTrustedDevices retrieves all trusted devices for a user.
func (r *MFARepository) ListTrustedDevices(ctx context.Context, userID xid.ID) ([]*schema.MFATrustedDevice, error) {
	var devices []*schema.MFATrustedDevice

	err := r.db.NewSelect().Model(&devices).
		Where("user_id = ?", userID).
		Where("expires_at > ?", time.Now()).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return devices, nil
}

// UpdateTrustedDevice updates a trusted device.
func (r *MFARepository) UpdateTrustedDevice(ctx context.Context, device *schema.MFATrustedDevice) error {
	_, err := r.db.NewUpdate().Model(device).WherePK().Exec(ctx)

	return err
}

// DeleteTrustedDevice deletes a trusted device.
func (r *MFARepository) DeleteTrustedDevice(ctx context.Context, deviceID xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.MFATrustedDevice)(nil)).Where("id = ?", deviceID).Exec(ctx)

	return err
}

// UpdateDeviceLastUsed updates the last used timestamp.
func (r *MFARepository) UpdateDeviceLastUsed(ctx context.Context, deviceID xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.MFATrustedDevice)(nil)).
		Set("last_used_at = ?", now).
		Where("id = ?", deviceID).
		Exec(ctx)

	return err
}

// CleanupExpiredDevices removes expired trusted devices.
func (r *MFARepository) CleanupExpiredDevices(ctx context.Context) (int, error) {
	result, err := r.db.NewDelete().
		Model((*schema.MFATrustedDevice)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()

	return int(rows), nil
}

// ==================== Policy Operations ====================

// GetPolicy retrieves the MFA policy for an app/organization.
func (r *MFARepository) GetPolicy(ctx context.Context, appID xid.ID, orgID *xid.ID) (*schema.MFAPolicy, error) {
	policy := new(schema.MFAPolicy)
	query := r.db.NewSelect().Model(policy).Where("app_id = ?", appID)

	if orgID != nil && !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return policy, err
}

// UpsertPolicy creates or updates an MFA policy.
func (r *MFARepository) UpsertPolicy(ctx context.Context, policy *schema.MFAPolicy) error {
	// Try to find existing policy
	existing := new(schema.MFAPolicy)
	query := r.db.NewSelect().Model(existing).Where("app_id = ?", policy.AppID)

	if policy.OrganizationID != nil && !policy.OrganizationID.IsNil() {
		query = query.Where("organization_id = ?", policy.OrganizationID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	err := query.Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		// Create new
		_, err = r.db.NewInsert().Model(policy).Exec(ctx)

		return err
	}

	if err != nil {
		return err
	}

	// Update existing
	policy.ID = existing.ID
	_, err = r.db.NewUpdate().Model(policy).WherePK().Exec(ctx)

	return err
}

// ==================== Attempt Tracking ====================

// CreateAttempt creates a new MFA attempt record.
func (r *MFARepository) CreateAttempt(ctx context.Context, attempt *schema.MFAAttempt) error {
	_, err := r.db.NewInsert().Model(attempt).Exec(ctx)

	return err
}

// GetRecentAttempts retrieves recent attempts for rate limiting.
func (r *MFARepository) GetRecentAttempts(ctx context.Context, userID xid.ID, since time.Time) ([]*schema.MFAAttempt, error) {
	var attempts []*schema.MFAAttempt

	err := r.db.NewSelect().Model(&attempts).
		Where("user_id = ?", userID).
		Where("created_at > ?", since).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return attempts, nil
}

// CountFailedAttempts counts failed attempts within a time window.
func (r *MFARepository) CountFailedAttempts(ctx context.Context, userID xid.ID, since time.Time) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.MFAAttempt)(nil)).
		Where("user_id = ?", userID).
		Where("success = ?", false).
		Where("created_at > ?", since).
		Count(ctx)

	return count, err
}

// CleanupOldAttempts removes old attempt records.
func (r *MFARepository) CleanupOldAttempts(ctx context.Context, olderThan time.Time) (int, error) {
	result, err := r.db.NewDelete().
		Model((*schema.MFAAttempt)(nil)).
		Where("created_at < ?", olderThan).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()

	return int(rows), nil
}

// ==================== Risk Assessment ====================

// CreateRiskAssessment creates a new risk assessment.
func (r *MFARepository) CreateRiskAssessment(ctx context.Context, assessment *schema.MFARiskAssessment) error {
	_, err := r.db.NewInsert().Model(assessment).Exec(ctx)

	return err
}

// GetLatestRiskAssessment retrieves the most recent risk assessment for a user.
func (r *MFARepository) GetLatestRiskAssessment(ctx context.Context, userID xid.ID) (*schema.MFARiskAssessment, error) {
	assessment := new(schema.MFARiskAssessment)

	err := r.db.NewSelect().Model(assessment).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return assessment, err
}

// GetRiskAssessmentBySession retrieves risk assessment for a session.
func (r *MFARepository) GetRiskAssessmentBySession(ctx context.Context, sessionID xid.ID) (*schema.MFARiskAssessment, error) {
	assessment := new(schema.MFARiskAssessment)

	err := r.db.NewSelect().Model(assessment).
		Where("session_id = ?", sessionID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return assessment, err
}

// ==================== MFA Bypass ====================

// CreateBypass creates a new MFA bypass.
func (r *MFARepository) CreateBypass(ctx context.Context, bypass *schema.MFABypass) error {
	_, err := r.db.NewInsert().Model(bypass).Exec(ctx)

	return err
}

// GetActiveBypass retrieves an active bypass for a user.
func (r *MFARepository) GetActiveBypass(ctx context.Context, appID, userID xid.ID) (*schema.MFABypass, error) {
	bypass := new(schema.MFABypass)

	err := r.db.NewSelect().Model(bypass).
		Where("app_id = ?", appID).
		Where("user_id = ?", userID).
		Where("expires_at > ?", time.Now()).
		Where("revoked_at IS NULL").
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return bypass, err
}

// RevokeBypass revokes an MFA bypass.
func (r *MFARepository) RevokeBypass(ctx context.Context, bypassID, revokedBy xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().Model((*schema.MFABypass)(nil)).
		Set("revoked_at = ?", now).
		Set("revoked_by = ?", revokedBy).
		Where("id = ?", bypassID).
		Exec(ctx)

	return err
}
