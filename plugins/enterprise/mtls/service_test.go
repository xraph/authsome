package mtls

import (
	"context"
	"encoding/pem"
	"testing"
	"time"
)

// Mock repository for testing
type mockRepository struct {
	certificates map[string]*Certificate
	trustAnchors map[string]*TrustAnchor
	policies     map[string]*CertificatePolicy
	authEvents   []*CertificateAuthEvent
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		certificates: make(map[string]*Certificate),
		trustAnchors: make(map[string]*TrustAnchor),
		policies:     make(map[string]*CertificatePolicy),
		authEvents:   make([]*CertificateAuthEvent, 0),
	}
}

func (m *mockRepository) CreateCertificate(ctx context.Context, cert *Certificate) error {
	m.certificates[cert.ID] = cert
	return nil
}

func (m *mockRepository) GetCertificate(ctx context.Context, id string) (*Certificate, error) {
	cert, exists := m.certificates[id]
	if !exists {
		return nil, ErrCertificateNotFound
	}
	return cert, nil
}

func (m *mockRepository) GetCertificateByFingerprint(ctx context.Context, fingerprint string) (*Certificate, error) {
	for _, cert := range m.certificates {
		if cert.Fingerprint == fingerprint {
			return cert, nil
		}
	}
	return nil, ErrCertificateNotFound
}

func (m *mockRepository) GetCertificateBySerialNumber(ctx context.Context, serialNumber string) (*Certificate, error) {
	for _, cert := range m.certificates {
		if cert.SerialNumber == serialNumber {
			return cert, nil
		}
	}
	return nil, ErrCertificateNotFound
}

func (m *mockRepository) ListCertificates(ctx context.Context, filters CertificateFilters) ([]*Certificate, error) {
	results := make([]*Certificate, 0)
	for _, cert := range m.certificates {
		if filters.OrganizationID != "" && cert.OrganizationID != filters.OrganizationID {
			continue
		}
		if filters.UserID != "" && cert.UserID != filters.UserID {
			continue
		}
		if filters.Status != "" && cert.Status != filters.Status {
			continue
		}
		results = append(results, cert)
	}
	return results, nil
}

func (m *mockRepository) UpdateCertificate(ctx context.Context, cert *Certificate) error {
	m.certificates[cert.ID] = cert
	return nil
}

func (m *mockRepository) RevokeCertificate(ctx context.Context, id string, reason string) error {
	cert, exists := m.certificates[id]
	if !exists {
		return ErrCertificateNotFound
	}
	now := time.Now()
	cert.Status = "revoked"
	cert.RevokedAt = &now
	cert.RevokedReason = reason
	return nil
}

func (m *mockRepository) DeleteCertificate(ctx context.Context, id string) error {
	delete(m.certificates, id)
	return nil
}

func (m *mockRepository) GetUserCertificates(ctx context.Context, userID string) ([]*Certificate, error) {
	results := make([]*Certificate, 0)
	for _, cert := range m.certificates {
		if cert.UserID == userID && cert.Status == "active" {
			results = append(results, cert)
		}
	}
	return results, nil
}

func (m *mockRepository) GetDeviceCertificates(ctx context.Context, deviceID string) ([]*Certificate, error) {
	results := make([]*Certificate, 0)
	for _, cert := range m.certificates {
		if cert.DeviceID == deviceID && cert.Status == "active" {
			results = append(results, cert)
		}
	}
	return results, nil
}

func (m *mockRepository) GetExpiringCertificates(ctx context.Context, orgID string, days int) ([]*Certificate, error) {
	expiryDate := time.Now().AddDate(0, 0, days)
	results := make([]*Certificate, 0)
	for _, cert := range m.certificates {
		if cert.OrganizationID == orgID &&
			cert.Status == "active" &&
			cert.NotAfter.Before(expiryDate) &&
			cert.NotAfter.After(time.Now()) {
			results = append(results, cert)
		}
	}
	return results, nil
}

func (m *mockRepository) CreateTrustAnchor(ctx context.Context, anchor *TrustAnchor) error {
	m.trustAnchors[anchor.ID] = anchor
	return nil
}

func (m *mockRepository) GetTrustAnchor(ctx context.Context, id string) (*TrustAnchor, error) {
	anchor, exists := m.trustAnchors[id]
	if !exists {
		return nil, ErrTrustAnchorNotFound
	}
	return anchor, nil
}

func (m *mockRepository) GetTrustAnchorByFingerprint(ctx context.Context, fingerprint string) (*TrustAnchor, error) {
	for _, anchor := range m.trustAnchors {
		if anchor.Fingerprint == fingerprint {
			return anchor, nil
		}
	}
	return nil, ErrTrustAnchorNotFound
}

func (m *mockRepository) ListTrustAnchors(ctx context.Context, orgID string) ([]*TrustAnchor, error) {
	results := make([]*TrustAnchor, 0)
	for _, anchor := range m.trustAnchors {
		if anchor.OrganizationID == orgID && anchor.Status == "active" {
			results = append(results, anchor)
		}
	}
	return results, nil
}

func (m *mockRepository) UpdateTrustAnchor(ctx context.Context, anchor *TrustAnchor) error {
	m.trustAnchors[anchor.ID] = anchor
	return nil
}

func (m *mockRepository) DeleteTrustAnchor(ctx context.Context, id string) error {
	delete(m.trustAnchors, id)
	return nil
}

func (m *mockRepository) CreateCRL(ctx context.Context, crl *CertificateRevocationList) error {
	return nil
}

func (m *mockRepository) GetCRL(ctx context.Context, id string) (*CertificateRevocationList, error) {
	return nil, nil
}

func (m *mockRepository) GetCRLByIssuer(ctx context.Context, issuer string) (*CertificateRevocationList, error) {
	return nil, nil
}

func (m *mockRepository) ListCRLs(ctx context.Context, trustAnchorID string) ([]*CertificateRevocationList, error) {
	return nil, nil
}

func (m *mockRepository) UpdateCRL(ctx context.Context, crl *CertificateRevocationList) error {
	return nil
}

func (m *mockRepository) DeleteCRL(ctx context.Context, id string) error {
	return nil
}

func (m *mockRepository) CreateOCSPResponse(ctx context.Context, resp *OCSPResponse) error {
	return nil
}

func (m *mockRepository) GetOCSPResponse(ctx context.Context, certificateID string) (*OCSPResponse, error) {
	return nil, nil
}

func (m *mockRepository) UpdateOCSPResponse(ctx context.Context, resp *OCSPResponse) error {
	return nil
}

func (m *mockRepository) DeleteExpiredOCSPResponses(ctx context.Context) error {
	return nil
}

func (m *mockRepository) CreateAuthEvent(ctx context.Context, event *CertificateAuthEvent) error {
	m.authEvents = append(m.authEvents, event)
	return nil
}

func (m *mockRepository) ListAuthEvents(ctx context.Context, filters AuthEventFilters) ([]*CertificateAuthEvent, error) {
	return m.authEvents, nil
}

func (m *mockRepository) GetAuthEventStats(ctx context.Context, orgID string, since time.Time) (*AuthEventStats, error) {
	stats := &AuthEventStats{}
	for _, event := range m.authEvents {
		if event.OrganizationID == orgID && event.CreatedAt.After(since) {
			stats.TotalAttempts++
			if event.Status == "success" {
				stats.SuccessfulAuths++
			} else if event.Status == "failed" {
				stats.FailedAuths++
			}
		}
	}
	return stats, nil
}

func (m *mockRepository) CreatePolicy(ctx context.Context, policy *CertificatePolicy) error {
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockRepository) GetPolicy(ctx context.Context, id string) (*CertificatePolicy, error) {
	policy, exists := m.policies[id]
	if !exists {
		return nil, ErrPolicyNotFound
	}
	return policy, nil
}

func (m *mockRepository) GetDefaultPolicy(ctx context.Context, orgID string) (*CertificatePolicy, error) {
	for _, policy := range m.policies {
		if policy.OrganizationID == orgID && policy.IsDefault {
			return policy, nil
		}
	}
	return nil, ErrPolicyNotFound
}

func (m *mockRepository) ListPolicies(ctx context.Context, orgID string) ([]*CertificatePolicy, error) {
	results := make([]*CertificatePolicy, 0)
	for _, policy := range m.policies {
		if policy.OrganizationID == orgID {
			results = append(results, policy)
		}
	}
	return results, nil
}

func (m *mockRepository) UpdatePolicy(ctx context.Context, policy *CertificatePolicy) error {
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockRepository) DeletePolicy(ctx context.Context, id string) error {
	delete(m.policies, id)
	return nil
}

// Tests

func TestService_RegisterCertificate(t *testing.T) {
	caCert, caKey := generateTestCA(t)
	_, _, certPEM := generateTestClientCert(t, caCert, caKey)

	repo := newMockRepository()
	config := DefaultConfig()
	config.Validation.ValidateChain = false // Disable chain validation for this test

	validator := NewCertificateValidator(config, repo, nil)
	service := NewService(config, repo, validator, nil, nil, nil)

	req := &RegisterCertificateRequest{
		OrganizationID:   "test_org",
		UserID:           "user_123",
		CertificatePEM:   string(certPEM),
		CertificateType:  "user",
		CertificateClass: "standard",
	}

	cert, err := service.RegisterCertificate(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to register certificate: %v", err)
	}

	if cert == nil {
		t.Fatal("expected certificate, got nil")
	}

	if cert.OrganizationID != "test_org" {
		t.Errorf("expected organization ID test_org, got %s", cert.OrganizationID)
	}

	if cert.UserID != "user_123" {
		t.Errorf("expected user ID user_123, got %s", cert.UserID)
	}

	if cert.Status != "active" {
		t.Errorf("expected status active, got %s", cert.Status)
	}

	// Verify certificate was stored in repository
	stored, err := repo.GetCertificate(context.Background(), cert.ID)
	if err != nil {
		t.Fatalf("failed to retrieve stored certificate: %v", err)
	}

	if stored.ID != cert.ID {
		t.Errorf("stored certificate ID mismatch")
	}
}

func TestService_AddTrustAnchor(t *testing.T) {
	caCert, _ := generateTestCA(t)
	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	})

	repo := newMockRepository()
	config := DefaultConfig()
	service := NewService(config, repo, nil, nil, nil, nil)

	req := &AddTrustAnchorRequest{
		OrganizationID: "test_org",
		Name:           "Test CA",
		CertificatePEM: string(caPEM),
		TrustLevel:     "root",
	}

	anchor, err := service.AddTrustAnchor(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to add trust anchor: %v", err)
	}

	if anchor == nil {
		t.Fatal("expected trust anchor, got nil")
	}

	if anchor.Name != "Test CA" {
		t.Errorf("expected name Test CA, got %s", anchor.Name)
	}

	if !anchor.IsRootCA {
		t.Error("expected IsRootCA to be true for self-signed CA")
	}
}

func TestService_CreatePolicy(t *testing.T) {
	repo := newMockRepository()
	config := DefaultConfig()
	service := NewService(config, repo, nil, nil, nil, nil)

	req := &CreatePolicyRequest{
		OrganizationID:       "test_org",
		Name:                 "High Security Policy",
		Description:          "Policy for sensitive operations",
		RequirePinning:       true,
		MinKeySize:           4096,
		AllowedKeyAlgorithms: StringArray{"RSA", "ECDSA"},
		RequireCRLCheck:      true,
		RequireOCSPCheck:     true,
		IsDefault:            true,
	}

	policy, err := service.CreatePolicy(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to create policy: %v", err)
	}

	if policy == nil {
		t.Fatal("expected policy, got nil")
	}

	if policy.Name != "High Security Policy" {
		t.Errorf("expected name High Security Policy, got %s", policy.Name)
	}

	if policy.MinKeySize != 4096 {
		t.Errorf("expected min key size 4096, got %d", policy.MinKeySize)
	}

	if !policy.IsDefault {
		t.Error("expected IsDefault to be true")
	}
}

func TestService_GetExpiringCertificates(t *testing.T) {
	repo := newMockRepository()
	config := DefaultConfig()
	service := NewService(config, repo, nil, nil, nil, nil)

	// Add certificate expiring in 15 days
	expiringSoon := &Certificate{
		ID:             "cert_1",
		OrganizationID: "test_org",
		Status:         "active",
		NotBefore:      time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:       time.Now().Add(15 * 24 * time.Hour),
	}
	repo.certificates["cert_1"] = expiringSoon

	// Add certificate expiring in 60 days
	expiringLater := &Certificate{
		ID:             "cert_2",
		OrganizationID: "test_org",
		Status:         "active",
		NotBefore:      time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:       time.Now().Add(60 * 24 * time.Hour),
	}
	repo.certificates["cert_2"] = expiringLater

	// Get certificates expiring in 30 days
	certs, err := service.GetExpiringCertificates(context.Background(), "test_org", 30)
	if err != nil {
		t.Fatalf("failed to get expiring certificates: %v", err)
	}

	if len(certs) != 1 {
		t.Errorf("expected 1 expiring certificate, got %d", len(certs))
	}

	if len(certs) > 0 && certs[0].ID != "cert_1" {
		t.Errorf("expected cert_1 to be expiring, got %s", certs[0].ID)
	}
}

func BenchmarkService_RegisterCertificate(b *testing.B) {
	caCert, caKey := generateTestCA(&testing.T{})
	_, _, certPEM := generateTestClientCert(&testing.T{}, caCert, caKey)

	repo := newMockRepository()
	config := DefaultConfig()
	config.Validation.ValidateChain = false

	validator := NewCertificateValidator(config, repo, nil)
	service := NewService(config, repo, validator, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &RegisterCertificateRequest{
			OrganizationID:   "test_org",
			UserID:           "user_123",
			CertificatePEM:   string(certPEM),
			CertificateType:  "user",
			CertificateClass: "standard",
		}
		_, _ = service.RegisterCertificate(context.Background(), req)
	}
}
