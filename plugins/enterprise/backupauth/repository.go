package backupauth

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Repository provides persistence for backup authentication entities
type Repository interface {
	// Security Questions
	CreateSecurityQuestion(ctx context.Context, q *SecurityQuestion) error
	GetSecurityQuestion(ctx context.Context, id xid.ID) (*SecurityQuestion, error)
	GetSecurityQuestionsByUser(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) ([]*SecurityQuestion, error)
	UpdateSecurityQuestion(ctx context.Context, q *SecurityQuestion) error
	DeleteSecurityQuestion(ctx context.Context, id xid.ID) error
	IncrementQuestionFailedAttempts(ctx context.Context, id xid.ID) error

	// Trusted Contacts
	CreateTrustedContact(ctx context.Context, tc *TrustedContact) error
	GetTrustedContact(ctx context.Context, id xid.ID) (*TrustedContact, error)
	GetTrustedContactsByUser(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) ([]*TrustedContact, error)
	GetTrustedContactByToken(ctx context.Context, token string) (*TrustedContact, error)
	UpdateTrustedContact(ctx context.Context, tc *TrustedContact) error
	DeleteTrustedContact(ctx context.Context, id xid.ID) error
	CountActiveTrustedContacts(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) (int, error)

	// Recovery Sessions
	CreateRecoverySession(ctx context.Context, rs *RecoverySession) error
	GetRecoverySession(ctx context.Context, id xid.ID) (*RecoverySession, error)
	UpdateRecoverySession(ctx context.Context, rs *RecoverySession) error
	DeleteRecoverySession(ctx context.Context, id xid.ID) error
	GetActiveRecoverySession(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) (*RecoverySession, error)
	ListRecoverySessions(ctx context.Context, appID xid.ID, userOrganizationID *xid.ID, status RecoveryStatus, requiresReview bool, limit, offset int) ([]*RecoverySession, int, error)
	ExpireRecoverySessions(ctx context.Context, before time.Time) (int, error)
	IncrementSessionAttempts(ctx context.Context, id xid.ID) error

	// Video Verification
	CreateVideoSession(ctx context.Context, vs *VideoVerificationSession) error
	GetVideoSession(ctx context.Context, id xid.ID) (*VideoVerificationSession, error)
	GetVideoSessionByRecovery(ctx context.Context, recoveryID xid.ID) (*VideoVerificationSession, error)
	UpdateVideoSession(ctx context.Context, vs *VideoVerificationSession) error
	DeleteVideoSession(ctx context.Context, id xid.ID) error

	// Document Verification
	CreateDocumentVerification(ctx context.Context, dv *DocumentVerification) error
	GetDocumentVerification(ctx context.Context, id xid.ID) (*DocumentVerification, error)
	GetDocumentVerificationByRecovery(ctx context.Context, recoveryID xid.ID) (*DocumentVerification, error)
	UpdateDocumentVerification(ctx context.Context, dv *DocumentVerification) error
	DeleteDocumentVerification(ctx context.Context, id xid.ID) error

	// Recovery Attempt Logs
	CreateRecoveryLog(ctx context.Context, log *RecoveryAttemptLog) error
	GetRecoveryLogs(ctx context.Context, recoveryID xid.ID) ([]*RecoveryAttemptLog, error)
	GetRecoveryLogsByUser(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, limit int) ([]*RecoveryAttemptLog, error)

	// Recovery Configuration
	CreateRecoveryConfig(ctx context.Context, rc *RecoveryConfiguration) error
	GetRecoveryConfig(ctx context.Context, appID xid.ID, userOrganizationID *xid.ID) (*RecoveryConfiguration, error)
	UpdateRecoveryConfig(ctx context.Context, rc *RecoveryConfiguration) error

	// Recovery Code Usage
	CreateRecoveryCodeUsage(ctx context.Context, rcu *RecoveryCodeUsage) error
	GetRecoveryCodeUsage(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, codeHash string) (*RecoveryCodeUsage, error)
	GetRecentRecoveryAttempts(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, since time.Time) (int, error)

	// Analytics
	GetRecoveryStats(ctx context.Context, appID xid.ID, userOrganizationID *xid.ID, startDate, endDate time.Time) (map[string]interface{}, error)
}

// BunRepository implements Repository using Bun ORM
type BunRepository struct {
	db *bun.DB
}

// NewBunRepository creates a new Bun repository
func NewBunRepository(db *bun.DB) Repository {
	return &BunRepository{db: db}
}

// ===== Security Questions =====

func (r *BunRepository) CreateSecurityQuestion(ctx context.Context, q *SecurityQuestion) error {
	q.ID = xid.New()
	q.IsActive = true
	_, err := r.db.NewInsert().Model(q).Exec(ctx)
	return err
}

func (r *BunRepository) GetSecurityQuestion(ctx context.Context, id xid.ID) (*SecurityQuestion, error) {
	q := new(SecurityQuestion)
	err := r.db.NewSelect().Model(q).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrSecurityQuestionNotFound
	}
	return q, err
}

func (r *BunRepository) GetSecurityQuestionsByUser(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) ([]*SecurityQuestion, error) {
	var questions []*SecurityQuestion
	q := r.db.NewSelect().Model(&questions).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("is_active = ?", true)

	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Order("created_at ASC").Scan(ctx)
	if err == sql.ErrNoRows {
		return []*SecurityQuestion{}, nil
	}
	return questions, err
}

func (r *BunRepository) UpdateSecurityQuestion(ctx context.Context, q *SecurityQuestion) error {
	_, err := r.db.NewUpdate().Model(q).WherePK().Exec(ctx)
	return err
}

func (r *BunRepository) DeleteSecurityQuestion(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*SecurityQuestion)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *BunRepository) IncrementQuestionFailedAttempts(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().Model((*SecurityQuestion)(nil)).
		Set("failed_attempts = failed_attempts + 1").
		Set("last_used_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// ===== Trusted Contacts =====

func (r *BunRepository) CreateTrustedContact(ctx context.Context, tc *TrustedContact) error {
	tc.ID = xid.New()
	tc.IsActive = true
	_, err := r.db.NewInsert().Model(tc).Exec(ctx)
	return err
}

func (r *BunRepository) GetTrustedContact(ctx context.Context, id xid.ID) (*TrustedContact, error) {
	tc := new(TrustedContact)
	err := r.db.NewSelect().Model(tc).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrTrustedContactNotFound
	}
	return tc, err
}

func (r *BunRepository) GetTrustedContactsByUser(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) ([]*TrustedContact, error) {
	var contacts []*TrustedContact
	q := r.db.NewSelect().Model(&contacts).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("is_active = ?", true)

	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Order("created_at DESC").Scan(ctx)
	if err == sql.ErrNoRows {
		return []*TrustedContact{}, nil
	}
	return contacts, err
}

func (r *BunRepository) GetTrustedContactByToken(ctx context.Context, token string) (*TrustedContact, error) {
	tc := new(TrustedContact)
	err := r.db.NewSelect().Model(tc).Where("verification_token = ?", token).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrTrustedContactNotFound
	}
	return tc, err
}

func (r *BunRepository) UpdateTrustedContact(ctx context.Context, tc *TrustedContact) error {
	_, err := r.db.NewUpdate().Model(tc).WherePK().Exec(ctx)
	return err
}

func (r *BunRepository) DeleteTrustedContact(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().Model((*TrustedContact)(nil)).
		Set("is_active = ?", false).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *BunRepository) CountActiveTrustedContacts(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) (int, error) {
	q := r.db.NewSelect().Model((*TrustedContact)(nil)).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("is_active = ?", true)

	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	count, err := q.Count(ctx)
	return count, err
}

// ===== Recovery Sessions =====

func (r *BunRepository) CreateRecoverySession(ctx context.Context, rs *RecoverySession) error {
	rs.ID = xid.New()
	rs.Status = RecoveryStatusPending
	_, err := r.db.NewInsert().Model(rs).Exec(ctx)
	return err
}

func (r *BunRepository) GetRecoverySession(ctx context.Context, id xid.ID) (*RecoverySession, error) {
	rs := new(RecoverySession)
	err := r.db.NewSelect().Model(rs).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrRecoverySessionNotFound
	}
	return rs, err
}

func (r *BunRepository) UpdateRecoverySession(ctx context.Context, rs *RecoverySession) error {
	_, err := r.db.NewUpdate().Model(rs).WherePK().Exec(ctx)
	return err
}

func (r *BunRepository) DeleteRecoverySession(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*RecoverySession)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *BunRepository) GetActiveRecoverySession(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) (*RecoverySession, error) {
	rs := new(RecoverySession)
	q := r.db.NewSelect().Model(rs).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("status IN (?)", bun.In([]RecoveryStatus{RecoveryStatusPending, RecoveryStatusInProgress})).
		Where("expires_at > ?", time.Now())

	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Order("created_at DESC").Limit(1).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rs, err
}

func (r *BunRepository) ListRecoverySessions(ctx context.Context, appID xid.ID, userOrganizationID *xid.ID, status RecoveryStatus, requiresReview bool, limit, offset int) ([]*RecoverySession, int, error) {
	query := r.db.NewSelect().Model((*RecoverySession)(nil)).
		Where("app_id = ?", appID)

	if userOrganizationID != nil {
		query = query.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		query = query.Where("user_organization_id IS NULL")
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if requiresReview {
		query = query.Where("requires_review = ?", true)
	}

	// Get total count
	totalCount, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	var sessions []*RecoverySession
	err = query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx, &sessions)

	if err == sql.ErrNoRows {
		return []*RecoverySession{}, 0, nil
	}
	return sessions, totalCount, err
}

func (r *BunRepository) ExpireRecoverySessions(ctx context.Context, before time.Time) (int, error) {
	result, err := r.db.NewUpdate().Model((*RecoverySession)(nil)).
		Set("status = ?", RecoveryStatusExpired).
		Where("status IN (?)", bun.In([]RecoveryStatus{RecoveryStatusPending, RecoveryStatusInProgress})).
		Where("expires_at < ?", before).
		Exec(ctx)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}

func (r *BunRepository) IncrementSessionAttempts(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().Model((*RecoverySession)(nil)).
		Set("attempts = attempts + 1").
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// ===== Video Verification =====

func (r *BunRepository) CreateVideoSession(ctx context.Context, vs *VideoVerificationSession) error {
	vs.ID = xid.New()
	_, err := r.db.NewInsert().Model(vs).Exec(ctx)
	return err
}

func (r *BunRepository) GetVideoSession(ctx context.Context, id xid.ID) (*VideoVerificationSession, error) {
	vs := new(VideoVerificationSession)
	err := r.db.NewSelect().Model(vs).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrVideoSessionNotFound
	}
	return vs, err
}

func (r *BunRepository) GetVideoSessionByRecovery(ctx context.Context, recoveryID xid.ID) (*VideoVerificationSession, error) {
	vs := new(VideoVerificationSession)
	err := r.db.NewSelect().Model(vs).Where("recovery_id = ?", recoveryID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrVideoSessionNotFound
	}
	return vs, err
}

func (r *BunRepository) UpdateVideoSession(ctx context.Context, vs *VideoVerificationSession) error {
	_, err := r.db.NewUpdate().Model(vs).WherePK().Exec(ctx)
	return err
}

func (r *BunRepository) DeleteVideoSession(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*VideoVerificationSession)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// ===== Document Verification =====

func (r *BunRepository) CreateDocumentVerification(ctx context.Context, dv *DocumentVerification) error {
	dv.ID = xid.New()
	_, err := r.db.NewInsert().Model(dv).Exec(ctx)
	return err
}

func (r *BunRepository) GetDocumentVerification(ctx context.Context, id xid.ID) (*DocumentVerification, error) {
	dv := new(DocumentVerification)
	err := r.db.NewSelect().Model(dv).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrDocumentVerificationNotFound
	}
	return dv, err
}

func (r *BunRepository) GetDocumentVerificationByRecovery(ctx context.Context, recoveryID xid.ID) (*DocumentVerification, error) {
	dv := new(DocumentVerification)
	err := r.db.NewSelect().Model(dv).Where("recovery_id = ?", recoveryID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrDocumentVerificationNotFound
	}
	return dv, err
}

func (r *BunRepository) UpdateDocumentVerification(ctx context.Context, dv *DocumentVerification) error {
	_, err := r.db.NewUpdate().Model(dv).WherePK().Exec(ctx)
	return err
}

func (r *BunRepository) DeleteDocumentVerification(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*DocumentVerification)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// ===== Recovery Attempt Logs =====

func (r *BunRepository) CreateRecoveryLog(ctx context.Context, log *RecoveryAttemptLog) error {
	log.ID = xid.New()
	_, err := r.db.NewInsert().Model(log).Exec(ctx)
	return err
}

func (r *BunRepository) GetRecoveryLogs(ctx context.Context, recoveryID xid.ID) ([]*RecoveryAttemptLog, error) {
	var logs []*RecoveryAttemptLog
	err := r.db.NewSelect().Model(&logs).
		Where("recovery_id = ?", recoveryID).
		Order("created_at ASC").
		Scan(ctx)
	if err == sql.ErrNoRows {
		return []*RecoveryAttemptLog{}, nil
	}
	return logs, err
}

func (r *BunRepository) GetRecoveryLogsByUser(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, limit int) ([]*RecoveryAttemptLog, error) {
	var logs []*RecoveryAttemptLog
	q := r.db.NewSelect().Model(&logs).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID)

	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Order("created_at DESC").Limit(limit).Scan(ctx)
	if err == sql.ErrNoRows {
		return []*RecoveryAttemptLog{}, nil
	}
	return logs, err
}

// ===== Recovery Configuration =====

func (r *BunRepository) CreateRecoveryConfig(ctx context.Context, rc *RecoveryConfiguration) error {
	rc.ID = xid.New()
	_, err := r.db.NewInsert().Model(rc).Exec(ctx)
	return err
}

func (r *BunRepository) GetRecoveryConfig(ctx context.Context, appID xid.ID, userOrganizationID *xid.ID) (*RecoveryConfiguration, error) {
	rc := new(RecoveryConfiguration)
	q := r.db.NewSelect().Model(rc).Where("app_id = ?", appID)

	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, ErrRecoveryNotConfigured
	}
	return rc, err
}

func (r *BunRepository) UpdateRecoveryConfig(ctx context.Context, rc *RecoveryConfiguration) error {
	_, err := r.db.NewUpdate().Model(rc).WherePK().Exec(ctx)
	return err
}

// ===== Recovery Code Usage =====

func (r *BunRepository) CreateRecoveryCodeUsage(ctx context.Context, rcu *RecoveryCodeUsage) error {
	rcu.ID = xid.New()
	_, err := r.db.NewInsert().Model(rcu).Exec(ctx)
	return err
}

func (r *BunRepository) GetRecoveryCodeUsage(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, codeHash string) (*RecoveryCodeUsage, error) {
	rcu := new(RecoveryCodeUsage)
	q := r.db.NewSelect().Model(rcu).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("code_hash = ?", codeHash)

	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	err := q.Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rcu, err
}

func (r *BunRepository) GetRecentRecoveryAttempts(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, since time.Time) (int, error) {
	q := r.db.NewSelect().Model((*RecoveryAttemptLog)(nil)).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("created_at > ?", since)

	if userOrganizationID != nil {
		q = q.Where("user_organization_id = ?", *userOrganizationID)
	} else {
		q = q.Where("user_organization_id IS NULL")
	}

	count, err := q.Count(ctx)
	return count, err
}

// ===== Analytics =====

func (r *BunRepository) GetRecoveryStats(ctx context.Context, appID xid.ID, userOrganizationID *xid.ID, startDate, endDate time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Helper to add app/org filters
	addOrgFilter := func(q *bun.SelectQuery) *bun.SelectQuery {
		q = q.Where("app_id = ?", appID)
		if userOrganizationID != nil {
			q = q.Where("user_organization_id = ?", *userOrganizationID)
		} else {
			q = q.Where("user_organization_id IS NULL")
		}
		return q
	}

	// Total attempts
	q := addOrgFilter(r.db.NewSelect().Model((*RecoverySession)(nil)))
	totalAttempts, err := q.Where("created_at BETWEEN ? AND ?", startDate, endDate).Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["totalAttempts"] = totalAttempts

	// Successful recoveries
	q = addOrgFilter(r.db.NewSelect().Model((*RecoverySession)(nil)))
	successfulRecoveries, err := q.
		Where("status = ?", RecoveryStatusCompleted).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["successfulRecoveries"] = successfulRecoveries

	// Failed recoveries
	q = addOrgFilter(r.db.NewSelect().Model((*RecoverySession)(nil)))
	failedRecoveries, err := q.
		Where("status = ?", RecoveryStatusFailed).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["failedRecoveries"] = failedRecoveries

	// Pending recoveries
	q = addOrgFilter(r.db.NewSelect().Model((*RecoverySession)(nil)))
	pendingRecoveries, err := q.
		Where("status IN (?)", bun.In([]RecoveryStatus{RecoveryStatusPending, RecoveryStatusInProgress})).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["pendingRecoveries"] = pendingRecoveries

	// High risk attempts
	q = addOrgFilter(r.db.NewSelect().Model((*RecoverySession)(nil)))
	highRiskAttempts, err := q.
		Where("risk_score > ?", 70.0).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["highRiskAttempts"] = highRiskAttempts

	// Admin reviews required
	q = addOrgFilter(r.db.NewSelect().Model((*RecoverySession)(nil)))
	adminReviewsRequired, err := q.
		Where("requires_review = ?", true).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["adminReviewsRequired"] = adminReviewsRequired

	// Method stats would require more complex query
	// Simplified version here
	stats["methodStats"] = map[RecoveryMethod]int{}

	return stats, nil
}
