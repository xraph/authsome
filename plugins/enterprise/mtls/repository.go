package mtls

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Repository defines the data access interface for mTLS
type Repository interface {
	// Certificates
	CreateCertificate(ctx context.Context, cert *Certificate) error
	GetCertificate(ctx context.Context, id string) (*Certificate, error)
	GetCertificateByFingerprint(ctx context.Context, fingerprint string) (*Certificate, error)
	GetCertificateBySerialNumber(ctx context.Context, serialNumber string) (*Certificate, error)
	ListCertificates(ctx context.Context, filters CertificateFilters) ([]*Certificate, error)
	UpdateCertificate(ctx context.Context, cert *Certificate) error
	RevokeCertificate(ctx context.Context, id string, reason string) error
	DeleteCertificate(ctx context.Context, id string) error
	
	// Certificate queries
	GetUserCertificates(ctx context.Context, userID string) ([]*Certificate, error)
	GetDeviceCertificates(ctx context.Context, deviceID string) ([]*Certificate, error)
	GetExpiringCertificates(ctx context.Context, orgID string, days int) ([]*Certificate, error)
	
	// Trust Anchors
	CreateTrustAnchor(ctx context.Context, anchor *TrustAnchor) error
	GetTrustAnchor(ctx context.Context, id string) (*TrustAnchor, error)
	GetTrustAnchorByFingerprint(ctx context.Context, fingerprint string) (*TrustAnchor, error)
	ListTrustAnchors(ctx context.Context, orgID string) ([]*TrustAnchor, error)
	UpdateTrustAnchor(ctx context.Context, anchor *TrustAnchor) error
	DeleteTrustAnchor(ctx context.Context, id string) error
	
	// CRLs
	CreateCRL(ctx context.Context, crl *CertificateRevocationList) error
	GetCRL(ctx context.Context, id string) (*CertificateRevocationList, error)
	GetCRLByIssuer(ctx context.Context, issuer string) (*CertificateRevocationList, error)
	ListCRLs(ctx context.Context, trustAnchorID string) ([]*CertificateRevocationList, error)
	UpdateCRL(ctx context.Context, crl *CertificateRevocationList) error
	DeleteCRL(ctx context.Context, id string) error
	
	// OCSP Responses
	CreateOCSPResponse(ctx context.Context, resp *OCSPResponse) error
	GetOCSPResponse(ctx context.Context, certificateID string) (*OCSPResponse, error)
	UpdateOCSPResponse(ctx context.Context, resp *OCSPResponse) error
	DeleteExpiredOCSPResponses(ctx context.Context) error
	
	// Auth Events
	CreateAuthEvent(ctx context.Context, event *CertificateAuthEvent) error
	ListAuthEvents(ctx context.Context, filters AuthEventFilters) ([]*CertificateAuthEvent, error)
	GetAuthEventStats(ctx context.Context, orgID string, since time.Time) (*AuthEventStats, error)
	
	// Policies
	CreatePolicy(ctx context.Context, policy *CertificatePolicy) error
	GetPolicy(ctx context.Context, id string) (*CertificatePolicy, error)
	GetDefaultPolicy(ctx context.Context, orgID string) (*CertificatePolicy, error)
	ListPolicies(ctx context.Context, orgID string) ([]*CertificatePolicy, error)
	UpdatePolicy(ctx context.Context, policy *CertificatePolicy) error
	DeletePolicy(ctx context.Context, id string) error
}

// CertificateFilters for filtering certificate queries
type CertificateFilters struct {
	OrganizationID string
	UserID         string
	DeviceID       string
	Status         string
	CertType       string
	Limit          int
	Offset         int
}

// AuthEventFilters for filtering auth event queries
type AuthEventFilters struct {
	OrganizationID string
	CertificateID  string
	UserID         string
	EventType      string
	Status         string
	Since          time.Time
	Until          time.Time
	Limit          int
	Offset         int
}

// AuthEventStats contains authentication event statistics
type AuthEventStats struct {
	TotalAttempts    int
	SuccessfulAuths  int
	FailedAuths      int
	ValidationErrors int
	UniqueUsers      int
	UniqueCerts      int
}

// BunRepository implements Repository using Bun ORM
type BunRepository struct {
	db *bun.DB
}

// NewBunRepository creates a new Bun repository
func NewBunRepository(db *bun.DB) *BunRepository {
	return &BunRepository{db: db}
}

// Certificate methods

func (r *BunRepository) CreateCertificate(ctx context.Context, cert *Certificate) error {
	_, err := r.db.NewInsert().Model(cert).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}
	return nil
}

func (r *BunRepository) GetCertificate(ctx context.Context, id string) (*Certificate, error) {
	cert := new(Certificate)
	err := r.db.NewSelect().
		Model(cert).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCertificateNotFound
		}
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	return cert, nil
}

func (r *BunRepository) GetCertificateByFingerprint(ctx context.Context, fingerprint string) (*Certificate, error) {
	cert := new(Certificate)
	err := r.db.NewSelect().
		Model(cert).
		Where("fingerprint = ?", fingerprint).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCertificateNotFound
		}
		return nil, fmt.Errorf("failed to get certificate by fingerprint: %w", err)
	}
	return cert, nil
}

func (r *BunRepository) GetCertificateBySerialNumber(ctx context.Context, serialNumber string) (*Certificate, error) {
	cert := new(Certificate)
	err := r.db.NewSelect().
		Model(cert).
		Where("serial_number = ?", serialNumber).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCertificateNotFound
		}
		return nil, fmt.Errorf("failed to get certificate by serial number: %w", err)
	}
	return cert, nil
}

func (r *BunRepository) ListCertificates(ctx context.Context, filters CertificateFilters) ([]*Certificate, error) {
	query := r.db.NewSelect().Model((*Certificate)(nil))
	
	if filters.OrganizationID != "" {
		query = query.Where("organization_id = ?", filters.OrganizationID)
	}
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.DeviceID != "" {
		query = query.Where("device_id = ?", filters.DeviceID)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.CertType != "" {
		query = query.Where("certificate_type = ?", filters.CertType)
	}
	
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}
	
	query = query.Order("created_at DESC")
	
	var certs []*Certificate
	err := query.Scan(ctx, &certs)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}
	
	return certs, nil
}

func (r *BunRepository) UpdateCertificate(ctx context.Context, cert *Certificate) error {
	cert.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(cert).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}
	return nil
}

func (r *BunRepository) RevokeCertificate(ctx context.Context, id string, reason string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*Certificate)(nil)).
		Set("status = ?", "revoked").
		Set("revoked_at = ?", now).
		Set("revoked_reason = ?", reason).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke certificate: %w", err)
	}
	return nil
}

func (r *BunRepository) DeleteCertificate(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*Certificate)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}
	return nil
}

func (r *BunRepository) GetUserCertificates(ctx context.Context, userID string) ([]*Certificate, error) {
	var certs []*Certificate
	err := r.db.NewSelect().
		Model(&certs).
		Where("user_id = ?", userID).
		Where("status = ?", "active").
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user certificates: %w", err)
	}
	return certs, nil
}

func (r *BunRepository) GetDeviceCertificates(ctx context.Context, deviceID string) ([]*Certificate, error) {
	var certs []*Certificate
	err := r.db.NewSelect().
		Model(&certs).
		Where("device_id = ?", deviceID).
		Where("status = ?", "active").
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device certificates: %w", err)
	}
	return certs, nil
}

func (r *BunRepository) GetExpiringCertificates(ctx context.Context, orgID string, days int) ([]*Certificate, error) {
	expiryDate := time.Now().AddDate(0, 0, days)
	var certs []*Certificate
	err := r.db.NewSelect().
		Model(&certs).
		Where("organization_id = ?", orgID).
		Where("status = ?", "active").
		Where("not_after <= ?", expiryDate).
		Where("not_after > ?", time.Now()).
		Order("not_after ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring certificates: %w", err)
	}
	return certs, nil
}

// Trust Anchor methods

func (r *BunRepository) CreateTrustAnchor(ctx context.Context, anchor *TrustAnchor) error {
	_, err := r.db.NewInsert().Model(anchor).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create trust anchor: %w", err)
	}
	return nil
}

func (r *BunRepository) GetTrustAnchor(ctx context.Context, id string) (*TrustAnchor, error) {
	anchor := new(TrustAnchor)
	err := r.db.NewSelect().
		Model(anchor).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTrustAnchorNotFound
		}
		return nil, fmt.Errorf("failed to get trust anchor: %w", err)
	}
	return anchor, nil
}

func (r *BunRepository) GetTrustAnchorByFingerprint(ctx context.Context, fingerprint string) (*TrustAnchor, error) {
	anchor := new(TrustAnchor)
	err := r.db.NewSelect().
		Model(anchor).
		Where("fingerprint = ?", fingerprint).
		Where("status = ?", "active").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTrustAnchorNotFound
		}
		return nil, fmt.Errorf("failed to get trust anchor by fingerprint: %w", err)
	}
	return anchor, nil
}

func (r *BunRepository) ListTrustAnchors(ctx context.Context, orgID string) ([]*TrustAnchor, error) {
	var anchors []*TrustAnchor
	err := r.db.NewSelect().
		Model(&anchors).
		Where("organization_id = ?", orgID).
		Where("status = ?", "active").
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list trust anchors: %w", err)
	}
	return anchors, nil
}

func (r *BunRepository) UpdateTrustAnchor(ctx context.Context, anchor *TrustAnchor) error {
	anchor.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(anchor).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update trust anchor: %w", err)
	}
	return nil
}

func (r *BunRepository) DeleteTrustAnchor(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*TrustAnchor)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete trust anchor: %w", err)
	}
	return nil
}

// CRL methods

func (r *BunRepository) CreateCRL(ctx context.Context, crl *CertificateRevocationList) error {
	_, err := r.db.NewInsert().Model(crl).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create CRL: %w", err)
	}
	return nil
}

func (r *BunRepository) GetCRL(ctx context.Context, id string) (*CertificateRevocationList, error) {
	crl := new(CertificateRevocationList)
	err := r.db.NewSelect().
		Model(crl).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("CRL not found")
		}
		return nil, fmt.Errorf("failed to get CRL: %w", err)
	}
	return crl, nil
}

func (r *BunRepository) GetCRLByIssuer(ctx context.Context, issuer string) (*CertificateRevocationList, error) {
	crl := new(CertificateRevocationList)
	err := r.db.NewSelect().
		Model(crl).
		Where("issuer = ?", issuer).
		Where("status = ?", "valid").
		Order("this_update DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("CRL not found for issuer")
		}
		return nil, fmt.Errorf("failed to get CRL by issuer: %w", err)
	}
	return crl, nil
}

func (r *BunRepository) ListCRLs(ctx context.Context, trustAnchorID string) ([]*CertificateRevocationList, error) {
	var crls []*CertificateRevocationList
	err := r.db.NewSelect().
		Model(&crls).
		Where("trust_anchor_id = ?", trustAnchorID).
		Order("this_update DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list CRLs: %w", err)
	}
	return crls, nil
}

func (r *BunRepository) UpdateCRL(ctx context.Context, crl *CertificateRevocationList) error {
	crl.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(crl).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update CRL: %w", err)
	}
	return nil
}

func (r *BunRepository) DeleteCRL(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*CertificateRevocationList)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete CRL: %w", err)
	}
	return nil
}

// OCSP methods

func (r *BunRepository) CreateOCSPResponse(ctx context.Context, resp *OCSPResponse) error {
	_, err := r.db.NewInsert().Model(resp).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create OCSP response: %w", err)
	}
	return nil
}

func (r *BunRepository) GetOCSPResponse(ctx context.Context, certificateID string) (*OCSPResponse, error) {
	resp := new(OCSPResponse)
	err := r.db.NewSelect().
		Model(resp).
		Where("certificate_id = ?", certificateID).
		Where("expires_at > ?", time.Now()).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cached response
		}
		return nil, fmt.Errorf("failed to get OCSP response: %w", err)
	}
	return resp, nil
}

func (r *BunRepository) UpdateOCSPResponse(ctx context.Context, resp *OCSPResponse) error {
	resp.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(resp).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update OCSP response: %w", err)
	}
	return nil
}

func (r *BunRepository) DeleteExpiredOCSPResponses(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*OCSPResponse)(nil)).
		Where("expires_at <= ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete expired OCSP responses: %w", err)
	}
	return nil
}

// Auth Event methods

func (r *BunRepository) CreateAuthEvent(ctx context.Context, event *CertificateAuthEvent) error {
	_, err := r.db.NewInsert().Model(event).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auth event: %w", err)
	}
	return nil
}

func (r *BunRepository) ListAuthEvents(ctx context.Context, filters AuthEventFilters) ([]*CertificateAuthEvent, error) {
	query := r.db.NewSelect().Model((*CertificateAuthEvent)(nil))
	
	if filters.OrganizationID != "" {
		query = query.Where("organization_id = ?", filters.OrganizationID)
	}
	if filters.CertificateID != "" {
		query = query.Where("certificate_id = ?", filters.CertificateID)
	}
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.EventType != "" {
		query = query.Where("event_type = ?", filters.EventType)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if !filters.Since.IsZero() {
		query = query.Where("created_at >= ?", filters.Since)
	}
	if !filters.Until.IsZero() {
		query = query.Where("created_at <= ?", filters.Until)
	}
	
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}
	
	query = query.Order("created_at DESC")
	
	var events []*CertificateAuthEvent
	err := query.Scan(ctx, &events)
	if err != nil {
		return nil, fmt.Errorf("failed to list auth events: %w", err)
	}
	
	return events, nil
}

func (r *BunRepository) GetAuthEventStats(ctx context.Context, orgID string, since time.Time) (*AuthEventStats, error) {
	stats := &AuthEventStats{}
	
	// Total attempts
	count, err := r.db.NewSelect().
		Model((*CertificateAuthEvent)(nil)).
		Where("organization_id = ?", orgID).
		Where("created_at >= ?", since).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total attempts: %w", err)
	}
	stats.TotalAttempts = count
	
	// Successful auths
	successCount, err := r.db.NewSelect().
		Model((*CertificateAuthEvent)(nil)).
		Where("organization_id = ?", orgID).
		Where("status = ?", "success").
		Where("created_at >= ?", since).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count successful auths: %w", err)
	}
	stats.SuccessfulAuths = successCount
	
	// Failed auths
	failedCount, err := r.db.NewSelect().
		Model((*CertificateAuthEvent)(nil)).
		Where("organization_id = ?", orgID).
		Where("status = ?", "failed").
		Where("created_at >= ?", since).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count failed auths: %w", err)
	}
	stats.FailedAuths = failedCount
	
	// Validation errors
	errorCount, err := r.db.NewSelect().
		Model((*CertificateAuthEvent)(nil)).
		Where("organization_id = ?", orgID).
		Where("status = ?", "error").
		Where("created_at >= ?", since).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count errors: %w", err)
	}
	stats.ValidationErrors = errorCount
	
	return stats, nil
}

// Policy methods

func (r *BunRepository) CreatePolicy(ctx context.Context, policy *CertificatePolicy) error {
	_, err := r.db.NewInsert().Model(policy).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}
	return nil
}

func (r *BunRepository) GetPolicy(ctx context.Context, id string) (*CertificatePolicy, error) {
	policy := new(CertificatePolicy)
	err := r.db.NewSelect().
		Model(policy).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPolicyNotFound
		}
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	return policy, nil
}

func (r *BunRepository) GetDefaultPolicy(ctx context.Context, orgID string) (*CertificatePolicy, error) {
	policy := new(CertificatePolicy)
	err := r.db.NewSelect().
		Model(policy).
		Where("organization_id = ?", orgID).
		Where("is_default = ?", true).
		Where("status = ?", "active").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPolicyNotFound
		}
		return nil, fmt.Errorf("failed to get default policy: %w", err)
	}
	return policy, nil
}

func (r *BunRepository) ListPolicies(ctx context.Context, orgID string) ([]*CertificatePolicy, error) {
	var policies []*CertificatePolicy
	err := r.db.NewSelect().
		Model(&policies).
		Where("organization_id = ?", orgID).
		Order("is_default DESC, created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	return policies, nil
}

func (r *BunRepository) UpdatePolicy(ctx context.Context, policy *CertificatePolicy) error {
	policy.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(policy).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}
	return nil
}

func (r *BunRepository) DeletePolicy(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*CertificatePolicy)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	return nil
}

