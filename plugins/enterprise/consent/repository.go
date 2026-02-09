package consent

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Repository defines the interface for consent data access.
type Repository interface {
	// Consent Records
	CreateConsent(ctx context.Context, consent *ConsentRecord) error
	GetConsent(ctx context.Context, id string) (*ConsentRecord, error)
	GetConsentByUserAndType(ctx context.Context, userID, orgID, consentType, purpose string) (*ConsentRecord, error)
	ListConsentsByUser(ctx context.Context, userID, orgID string) ([]*ConsentRecord, error)
	UpdateConsent(ctx context.Context, consent *ConsentRecord) error
	DeleteConsent(ctx context.Context, id string) error
	ExpireConsents(ctx context.Context, beforeDate time.Time) (int, error)

	// Consent Policies
	CreatePolicy(ctx context.Context, policy *ConsentPolicy) error
	GetPolicy(ctx context.Context, id string) (*ConsentPolicy, error)
	GetPolicyByTypeAndVersion(ctx context.Context, orgID, consentType, version string) (*ConsentPolicy, error)
	GetLatestPolicy(ctx context.Context, orgID, consentType string) (*ConsentPolicy, error)
	ListPolicies(ctx context.Context, orgID string, active *bool) ([]*ConsentPolicy, error)
	UpdatePolicy(ctx context.Context, policy *ConsentPolicy) error
	DeletePolicy(ctx context.Context, id string) error

	// Data Processing Agreements
	CreateDPA(ctx context.Context, dpa *DataProcessingAgreement) error
	GetDPA(ctx context.Context, id string) (*DataProcessingAgreement, error)
	GetActiveDPA(ctx context.Context, orgID, agreementType string) (*DataProcessingAgreement, error)
	ListDPAs(ctx context.Context, orgID string, status *string) ([]*DataProcessingAgreement, error)
	UpdateDPA(ctx context.Context, dpa *DataProcessingAgreement) error

	// Consent Audit Logs
	CreateAuditLog(ctx context.Context, log *ConsentAuditLog) error
	ListAuditLogs(ctx context.Context, userID, orgID string, limit int) ([]*ConsentAuditLog, error)
	GetAuditLogsByConsent(ctx context.Context, consentID string) ([]*ConsentAuditLog, error)

	// Cookie Consents
	CreateCookieConsent(ctx context.Context, consent *CookieConsent) error
	GetCookieConsent(ctx context.Context, userID, orgID string) (*CookieConsent, error)
	GetCookieConsentBySession(ctx context.Context, sessionID, orgID string) (*CookieConsent, error)
	UpdateCookieConsent(ctx context.Context, consent *CookieConsent) error

	// Data Export Requests
	CreateExportRequest(ctx context.Context, request *DataExportRequest) error
	GetExportRequest(ctx context.Context, id string) (*DataExportRequest, error)
	ListExportRequests(ctx context.Context, userID, orgID string, status *string) ([]*DataExportRequest, error)
	UpdateExportRequest(ctx context.Context, request *DataExportRequest) error
	DeleteExpiredExports(ctx context.Context, beforeDate time.Time) (int, error)

	// Data Deletion Requests
	CreateDeletionRequest(ctx context.Context, request *DataDeletionRequest) error
	GetDeletionRequest(ctx context.Context, id string) (*DataDeletionRequest, error)
	ListDeletionRequests(ctx context.Context, userID, orgID string, status *string) ([]*DataDeletionRequest, error)
	UpdateDeletionRequest(ctx context.Context, request *DataDeletionRequest) error
	GetPendingDeletionRequest(ctx context.Context, userID, orgID string) (*DataDeletionRequest, error)

	// Privacy Settings
	CreatePrivacySettings(ctx context.Context, settings *PrivacySettings) error
	GetPrivacySettings(ctx context.Context, orgID string) (*PrivacySettings, error)
	UpdatePrivacySettings(ctx context.Context, settings *PrivacySettings) error

	// Analytics
	GetConsentStats(ctx context.Context, orgID string, startDate, endDate time.Time) (map[string]any, error)
}

// BunRepository implements Repository using Bun ORM.
type BunRepository struct {
	db *bun.DB
}

// NewBunRepository creates a new Bun-based repository.
func NewBunRepository(db *bun.DB) Repository {
	return &BunRepository{db: db}
}

// Consent Records

func (r *BunRepository) CreateConsent(ctx context.Context, consent *ConsentRecord) error {
	consent.ID = xid.New()
	consent.CreatedAt = time.Now()
	consent.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(consent).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create consent record: %w", err)
	}

	return nil
}

func (r *BunRepository) GetConsent(ctx context.Context, id string) (*ConsentRecord, error) {
	consent := new(ConsentRecord)

	err := r.db.NewSelect().Model(consent).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent record: %w", err)
	}

	return consent, nil
}

func (r *BunRepository) GetConsentByUserAndType(ctx context.Context, userID, orgID, consentType, purpose string) (*ConsentRecord, error) {
	consent := new(ConsentRecord)

	err := r.db.NewSelect().
		Model(consent).
		Where("user_id = ?", userID).
		Where("organization_id = ?", orgID).
		Where("consent_type = ?", consentType).
		Where("purpose = ?", purpose).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent by user and type: %w", err)
	}

	return consent, nil
}

func (r *BunRepository) ListConsentsByUser(ctx context.Context, userID, orgID string) ([]*ConsentRecord, error) {
	var consents []*ConsentRecord

	err := r.db.NewSelect().
		Model(&consents).
		Where("user_id = ?", userID).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list consents by user: %w", err)
	}

	return consents, nil
}

func (r *BunRepository) UpdateConsent(ctx context.Context, consent *ConsentRecord) error {
	consent.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().Model(consent).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update consent record: %w", err)
	}

	return nil
}

func (r *BunRepository) DeleteConsent(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*ConsentRecord)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete consent record: %w", err)
	}

	return nil
}

func (r *BunRepository) ExpireConsents(ctx context.Context, beforeDate time.Time) (int, error) {
	result, err := r.db.NewUpdate().
		Model((*ConsentRecord)(nil)).
		Set("granted = ?", false).
		Set("revoked_at = ?", time.Now()).
		Set("updated_at = ?", time.Now()).
		Where("expires_at IS NOT NULL").
		Where("expires_at < ?", beforeDate).
		Where("granted = ?", true).
		Where("revoked_at IS NULL").
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to expire consents: %w", err)
	}

	rows, _ := result.RowsAffected()

	return int(rows), nil
}

// Consent Policies

func (r *BunRepository) CreatePolicy(ctx context.Context, policy *ConsentPolicy) error {
	policy.ID = xid.New()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(policy).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create consent policy: %w", err)
	}

	return nil
}

func (r *BunRepository) GetPolicy(ctx context.Context, id string) (*ConsentPolicy, error) {
	policy := new(ConsentPolicy)

	err := r.db.NewSelect().Model(policy).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent policy: %w", err)
	}

	return policy, nil
}

func (r *BunRepository) GetPolicyByTypeAndVersion(ctx context.Context, orgID, consentType, version string) (*ConsentPolicy, error) {
	policy := new(ConsentPolicy)

	err := r.db.NewSelect().
		Model(policy).
		Where("organization_id = ?", orgID).
		Where("consent_type = ?", consentType).
		Where("version = ?", version).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy by type and version: %w", err)
	}

	return policy, nil
}

func (r *BunRepository) GetLatestPolicy(ctx context.Context, orgID, consentType string) (*ConsentPolicy, error) {
	policy := new(ConsentPolicy)

	err := r.db.NewSelect().
		Model(policy).
		Where("organization_id = ?", orgID).
		Where("consent_type = ?", consentType).
		Where("active = ?", true).
		Order("published_at DESC", "created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest policy: %w", err)
	}

	return policy, nil
}

func (r *BunRepository) ListPolicies(ctx context.Context, orgID string, active *bool) ([]*ConsentPolicy, error) {
	var policies []*ConsentPolicy

	query := r.db.NewSelect().Model(&policies).Where("organization_id = ?", orgID)
	if active != nil {
		query = query.Where("active = ?", *active)
	}

	err := query.Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	return policies, nil
}

func (r *BunRepository) UpdatePolicy(ctx context.Context, policy *ConsentPolicy) error {
	policy.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().Model(policy).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update consent policy: %w", err)
	}

	return nil
}

func (r *BunRepository) DeletePolicy(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*ConsentPolicy)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete consent policy: %w", err)
	}

	return nil
}

// Data Processing Agreements

func (r *BunRepository) CreateDPA(ctx context.Context, dpa *DataProcessingAgreement) error {
	dpa.ID = xid.New()
	dpa.CreatedAt = time.Now()
	dpa.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(dpa).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create DPA: %w", err)
	}

	return nil
}

func (r *BunRepository) GetDPA(ctx context.Context, id string) (*DataProcessingAgreement, error) {
	dpa := new(DataProcessingAgreement)

	err := r.db.NewSelect().Model(dpa).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DPA: %w", err)
	}

	return dpa, nil
}

func (r *BunRepository) GetActiveDPA(ctx context.Context, orgID, agreementType string) (*DataProcessingAgreement, error) {
	dpa := new(DataProcessingAgreement)

	err := r.db.NewSelect().
		Model(dpa).
		Where("organization_id = ?", orgID).
		Where("agreement_type = ?", agreementType).
		Where("status = ?", "active").
		Order("effective_date DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active DPA: %w", err)
	}

	return dpa, nil
}

func (r *BunRepository) ListDPAs(ctx context.Context, orgID string, status *string) ([]*DataProcessingAgreement, error) {
	var dpas []*DataProcessingAgreement

	query := r.db.NewSelect().Model(&dpas).Where("organization_id = ?", orgID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Order("effective_date DESC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list DPAs: %w", err)
	}

	return dpas, nil
}

func (r *BunRepository) UpdateDPA(ctx context.Context, dpa *DataProcessingAgreement) error {
	dpa.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().Model(dpa).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update DPA: %w", err)
	}

	return nil
}

// Consent Audit Logs

func (r *BunRepository) CreateAuditLog(ctx context.Context, log *ConsentAuditLog) error {
	log.ID = xid.New()
	log.CreatedAt = time.Now()

	_, err := r.db.NewInsert().Model(log).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (r *BunRepository) ListAuditLogs(ctx context.Context, userID, orgID string, limit int) ([]*ConsentAuditLog, error) {
	var logs []*ConsentAuditLog

	query := r.db.NewSelect().Model(&logs)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return logs, nil
}

func (r *BunRepository) GetAuditLogsByConsent(ctx context.Context, consentID string) ([]*ConsentAuditLog, error) {
	var logs []*ConsentAuditLog

	err := r.db.NewSelect().
		Model(&logs).
		Where("consent_id = ?", consentID).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by consent: %w", err)
	}

	return logs, nil
}

// Cookie Consents

func (r *BunRepository) CreateCookieConsent(ctx context.Context, consent *CookieConsent) error {
	consent.ID = xid.New()
	consent.CreatedAt = time.Now()
	consent.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(consent).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create cookie consent: %w", err)
	}

	return nil
}

func (r *BunRepository) GetCookieConsent(ctx context.Context, userID, orgID string) (*CookieConsent, error) {
	consent := new(CookieConsent)

	err := r.db.NewSelect().
		Model(consent).
		Where("user_id = ?", userID).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookie consent: %w", err)
	}

	return consent, nil
}

func (r *BunRepository) GetCookieConsentBySession(ctx context.Context, sessionID, orgID string) (*CookieConsent, error) {
	consent := new(CookieConsent)

	err := r.db.NewSelect().
		Model(consent).
		Where("session_id = ?", sessionID).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookie consent by session: %w", err)
	}

	return consent, nil
}

func (r *BunRepository) UpdateCookieConsent(ctx context.Context, consent *CookieConsent) error {
	consent.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().Model(consent).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update cookie consent: %w", err)
	}

	return nil
}

// Data Export Requests

func (r *BunRepository) CreateExportRequest(ctx context.Context, request *DataExportRequest) error {
	request.ID = xid.New()
	request.CreatedAt = time.Now()
	request.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(request).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create export request: %w", err)
	}

	return nil
}

func (r *BunRepository) GetExportRequest(ctx context.Context, id string) (*DataExportRequest, error) {
	request := new(DataExportRequest)

	err := r.db.NewSelect().Model(request).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get export request: %w", err)
	}

	return request, nil
}

func (r *BunRepository) ListExportRequests(ctx context.Context, userID, orgID string, status *string) ([]*DataExportRequest, error) {
	var requests []*DataExportRequest

	query := r.db.NewSelect().Model(&requests)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	}

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list export requests: %w", err)
	}

	return requests, nil
}

func (r *BunRepository) UpdateExportRequest(ctx context.Context, request *DataExportRequest) error {
	request.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().Model(request).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update export request: %w", err)
	}

	return nil
}

func (r *BunRepository) DeleteExpiredExports(ctx context.Context, beforeDate time.Time) (int, error) {
	result, err := r.db.NewDelete().
		Model((*DataExportRequest)(nil)).
		Where("expires_at IS NOT NULL").
		Where("expires_at < ?", beforeDate).
		Where("status = ?", "completed").
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired exports: %w", err)
	}

	rows, _ := result.RowsAffected()

	return int(rows), nil
}

// Data Deletion Requests

func (r *BunRepository) CreateDeletionRequest(ctx context.Context, request *DataDeletionRequest) error {
	request.ID = xid.New()
	request.CreatedAt = time.Now()
	request.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(request).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create deletion request: %w", err)
	}

	return nil
}

func (r *BunRepository) GetDeletionRequest(ctx context.Context, id string) (*DataDeletionRequest, error) {
	request := new(DataDeletionRequest)

	err := r.db.NewSelect().Model(request).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get deletion request: %w", err)
	}

	return request, nil
}

func (r *BunRepository) ListDeletionRequests(ctx context.Context, userID, orgID string, status *string) ([]*DataDeletionRequest, error) {
	var requests []*DataDeletionRequest

	query := r.db.NewSelect().Model(&requests)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	}

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list deletion requests: %w", err)
	}

	return requests, nil
}

func (r *BunRepository) UpdateDeletionRequest(ctx context.Context, request *DataDeletionRequest) error {
	request.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().Model(request).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update deletion request: %w", err)
	}

	return nil
}

func (r *BunRepository) GetPendingDeletionRequest(ctx context.Context, userID, orgID string) (*DataDeletionRequest, error) {
	request := new(DataDeletionRequest)

	err := r.db.NewSelect().
		Model(request).
		Where("user_id = ?", userID).
		Where("organization_id = ?", orgID).
		Where("status IN (?)", bun.In([]string{"pending", "approved", "processing"})).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending deletion request: %w", err)
	}

	return request, nil
}

// Privacy Settings

func (r *BunRepository) CreatePrivacySettings(ctx context.Context, settings *PrivacySettings) error {
	settings.ID = xid.New()
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(settings).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create privacy settings: %w", err)
	}

	return nil
}

func (r *BunRepository) GetPrivacySettings(ctx context.Context, orgID string) (*PrivacySettings, error) {
	settings := new(PrivacySettings)

	err := r.db.NewSelect().Model(settings).Where("organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get privacy settings: %w", err)
	}

	return settings, nil
}

func (r *BunRepository) UpdatePrivacySettings(ctx context.Context, settings *PrivacySettings) error {
	settings.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().Model(settings).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update privacy settings: %w", err)
	}

	return nil
}

// Analytics

func (r *BunRepository) GetConsentStats(ctx context.Context, orgID string, startDate, endDate time.Time) (map[string]any, error) {
	stats := make(map[string]any)

	// Total consents
	totalConsents, err := r.db.NewSelect().
		Model((*ConsentRecord)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", startDate).
		Where("created_at <= ?", endDate).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total consents: %w", err)
	}

	stats["totalConsents"] = totalConsents

	// Granted consents
	grantedConsents, err := r.db.NewSelect().
		Model((*ConsentRecord)(nil)).
		Where("organization_id = ?", orgID).
		Where("granted = ?", true).
		Where("created_at >= ?", startDate).
		Where("created_at <= ?", endDate).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count granted consents: %w", err)
	}

	stats["grantedConsents"] = grantedConsents

	// Revoked consents
	revokedConsents, err := r.db.NewSelect().
		Model((*ConsentRecord)(nil)).
		Where("organization_id = ?", orgID).
		Where("granted = ?", false).
		Where("revoked_at IS NOT NULL").
		Where("revoked_at >= ?", startDate).
		Where("revoked_at <= ?", endDate).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count revoked consents: %w", err)
	}

	stats["revokedConsents"] = revokedConsents

	// Pending deletions
	pendingDeletions, err := r.db.NewSelect().
		Model((*DataDeletionRequest)(nil)).
		Where("organization_id = ?", orgID).
		Where("status IN (?)", bun.In([]string{"pending", "approved", "processing"})).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count pending deletions: %w", err)
	}

	stats["pendingDeletions"] = pendingDeletions

	// Data exports
	dataExports, err := r.db.NewSelect().
		Model((*DataExportRequest)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", startDate).
		Where("created_at <= ?", endDate).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count data exports: %w", err)
	}

	stats["dataExports"] = dataExports

	return stats, nil
}
