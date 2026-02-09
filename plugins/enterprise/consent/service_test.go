package consent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/xid"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	mu              sync.RWMutex
	consents        map[string]*ConsentRecord
	policies        map[string]*ConsentPolicy
	cookieConsents  map[string]*CookieConsent
	exportRequests  map[string]*DataExportRequest
	deleteRequests  map[string]*DataDeletionRequest
	privacySettings map[string]*PrivacySettings
	auditLogs       []*ConsentAuditLog
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		consents:        make(map[string]*ConsentRecord),
		policies:        make(map[string]*ConsentPolicy),
		cookieConsents:  make(map[string]*CookieConsent),
		exportRequests:  make(map[string]*DataExportRequest),
		deleteRequests:  make(map[string]*DataDeletionRequest),
		privacySettings: make(map[string]*PrivacySettings),
		auditLogs:       make([]*ConsentAuditLog, 0),
	}
}

// Consent Records
func (m *MockRepository) CreateConsent(ctx context.Context, consent *ConsentRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	consent.ID = xid.New()
	m.consents[consent.ID.String()] = consent
	return nil
}

func (m *MockRepository) GetConsent(ctx context.Context, id string) (*ConsentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	consent, ok := m.consents[id]
	if !ok {
		return nil, ErrConsentNotFound
	}
	consentCopy := *consent
	return &consentCopy, nil
}

func (m *MockRepository) GetConsentByUserAndType(ctx context.Context, userID, orgID, consentType, purpose string) (*ConsentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, consent := range m.consents {
		if consent.UserID == userID && consent.OrganizationID == orgID &&
			consent.ConsentType == consentType && consent.Purpose == purpose {
			consentCopy := *consent
			return &consentCopy, nil
		}
	}
	return nil, ErrConsentNotFound
}

func (m *MockRepository) ListConsentsByUser(ctx context.Context, userID, orgID string) ([]*ConsentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ConsentRecord
	for _, consent := range m.consents {
		if consent.UserID == userID && consent.OrganizationID == orgID {
			consentCopy := *consent
			result = append(result, &consentCopy)
		}
	}
	return result, nil
}

func (m *MockRepository) UpdateConsent(ctx context.Context, consent *ConsentRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consents[consent.ID.String()] = consent
	return nil
}

func (m *MockRepository) DeleteConsent(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.consents, id)
	return nil
}

func (m *MockRepository) ExpireConsents(ctx context.Context, beforeDate time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, consent := range m.consents {
		if consent.ExpiresAt != nil && consent.ExpiresAt.Before(beforeDate) && consent.Granted {
			consent.Granted = false
			now := time.Now()
			consent.RevokedAt = &now
			count++
		}
	}
	return count, nil
}

// Consent Policies
func (m *MockRepository) CreatePolicy(ctx context.Context, policy *ConsentPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	policy.ID = xid.New()
	m.policies[policy.ID.String()] = policy
	return nil
}

func (m *MockRepository) GetPolicy(ctx context.Context, id string) (*ConsentPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.policies[id]
	if !ok {
		return nil, ErrPolicyNotFound
	}
	policyCopy := *policy
	return &policyCopy, nil
}

func (m *MockRepository) GetPolicyByTypeAndVersion(ctx context.Context, orgID, consentType, version string) (*ConsentPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, policy := range m.policies {
		if policy.OrganizationID == orgID && policy.ConsentType == consentType && policy.Version == version {
			policyCopy := *policy
			return &policyCopy, nil
		}
	}
	return nil, ErrPolicyNotFound
}

func (m *MockRepository) GetLatestPolicy(ctx context.Context, orgID, consentType string) (*ConsentPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, policy := range m.policies {
		if policy.OrganizationID == orgID && policy.ConsentType == consentType && policy.Active {
			policyCopy := *policy
			return &policyCopy, nil
		}
	}
	return nil, ErrPolicyNotFound
}

func (m *MockRepository) ListPolicies(ctx context.Context, orgID string, active *bool) ([]*ConsentPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ConsentPolicy
	for _, policy := range m.policies {
		if policy.OrganizationID == orgID {
			if active == nil || policy.Active == *active {
				policyCopy := *policy
				result = append(result, &policyCopy)
			}
		}
	}
	return result, nil
}

func (m *MockRepository) UpdatePolicy(ctx context.Context, policy *ConsentPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.policies[policy.ID.String()] = policy
	return nil
}

func (m *MockRepository) DeletePolicy(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.policies, id)
	return nil
}

// Data Processing Agreements
func (m *MockRepository) CreateDPA(ctx context.Context, dpa *DataProcessingAgreement) error {
	return nil
}

func (m *MockRepository) GetDPA(ctx context.Context, id string) (*DataProcessingAgreement, error) {
	return nil, nil
}

func (m *MockRepository) GetActiveDPA(ctx context.Context, orgID, agreementType string) (*DataProcessingAgreement, error) {
	return nil, nil
}

func (m *MockRepository) ListDPAs(ctx context.Context, orgID string, status *string) ([]*DataProcessingAgreement, error) {
	return nil, nil
}

func (m *MockRepository) UpdateDPA(ctx context.Context, dpa *DataProcessingAgreement) error {
	return nil
}

// Consent Audit Logs
func (m *MockRepository) CreateAuditLog(ctx context.Context, log *ConsentAuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.auditLogs = append(m.auditLogs, log)
	return nil
}

func (m *MockRepository) ListAuditLogs(ctx context.Context, userID, orgID string, limit int) ([]*ConsentAuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ConsentAuditLog, len(m.auditLogs))
	copy(result, m.auditLogs)
	return result, nil
}

func (m *MockRepository) GetAuditLogsByConsent(ctx context.Context, consentID string) ([]*ConsentAuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ConsentAuditLog
	for _, log := range m.auditLogs {
		if log.ConsentID == consentID {
			result = append(result, log)
		}
	}
	return result, nil
}

// Cookie Consents
func (m *MockRepository) CreateCookieConsent(ctx context.Context, consent *CookieConsent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if a consent already exists for this user+org (should be unique)
	for id, existing := range m.cookieConsents {
		if existing.UserID == consent.UserID && existing.OrganizationID == consent.OrganizationID {
			// Update existing record
			consent.ID = existing.ID
			m.cookieConsents[id] = consent
			return nil
		}
	}

	// Create new record
	consent.ID = xid.New()
	m.cookieConsents[consent.ID.String()] = consent
	return nil
}

func (m *MockRepository) GetCookieConsent(ctx context.Context, userID, orgID string) (*CookieConsent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, consent := range m.cookieConsents {
		if consent.UserID == userID && consent.OrganizationID == orgID {
			consentCopy := *consent
			return &consentCopy, nil
		}
	}
	return nil, ErrCookieConsentNotFound
}

func (m *MockRepository) GetCookieConsentBySession(ctx context.Context, sessionID, orgID string) (*CookieConsent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, consent := range m.cookieConsents {
		if consent.SessionID == sessionID && consent.OrganizationID == orgID {
			consentCopy := *consent
			return &consentCopy, nil
		}
	}
	return nil, ErrCookieConsentNotFound
}

func (m *MockRepository) UpdateCookieConsent(ctx context.Context, consent *CookieConsent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cookieConsents[consent.ID.String()] = consent
	return nil
}

// Data Export Requests
func (m *MockRepository) CreateExportRequest(ctx context.Context, request *DataExportRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	request.ID = xid.New()
	m.exportRequests[request.ID.String()] = request
	return nil
}

func (m *MockRepository) GetExportRequest(ctx context.Context, id string) (*DataExportRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	request, ok := m.exportRequests[id]
	if !ok {
		return nil, ErrExportNotFound
	}

	// Return a copy to avoid race conditions
	reqCopy := *request
	return &reqCopy, nil
}

func (m *MockRepository) ListExportRequests(ctx context.Context, userID, orgID string, status *string) ([]*DataExportRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*DataExportRequest
	for _, request := range m.exportRequests {
		if request.UserID == userID && request.OrganizationID == orgID {
			if status == nil || request.Status == *status {
				reqCopy := *request
				result = append(result, &reqCopy)
			}
		}
	}
	return result, nil
}

func (m *MockRepository) UpdateExportRequest(ctx context.Context, request *DataExportRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.exportRequests[request.ID.String()] = request
	return nil
}

func (m *MockRepository) DeleteExpiredExports(ctx context.Context, beforeDate time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, request := range m.exportRequests {
		if request.ExpiresAt != nil && request.ExpiresAt.Before(beforeDate) {
			delete(m.exportRequests, id)
			count++
		}
	}
	return count, nil
}

// Data Deletion Requests
func (m *MockRepository) CreateDeletionRequest(ctx context.Context, request *DataDeletionRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	request.ID = xid.New()
	m.deleteRequests[request.ID.String()] = request
	return nil
}

func (m *MockRepository) GetDeletionRequest(ctx context.Context, id string) (*DataDeletionRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	request, ok := m.deleteRequests[id]
	if !ok {
		return nil, ErrDeletionNotFound
	}

	// Return a copy to avoid race conditions
	reqCopy := *request
	return &reqCopy, nil
}

func (m *MockRepository) ListDeletionRequests(ctx context.Context, userID, orgID string, status *string) ([]*DataDeletionRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*DataDeletionRequest
	for _, request := range m.deleteRequests {
		if request.UserID == userID && request.OrganizationID == orgID {
			if status == nil || request.Status == *status {
				reqCopy := *request
				result = append(result, &reqCopy)
			}
		}
	}
	return result, nil
}

func (m *MockRepository) UpdateDeletionRequest(ctx context.Context, request *DataDeletionRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteRequests[request.ID.String()] = request
	return nil
}

func (m *MockRepository) GetPendingDeletionRequest(ctx context.Context, userID, orgID string) (*DataDeletionRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, request := range m.deleteRequests {
		if request.UserID == userID && request.OrganizationID == orgID &&
			(request.Status == string(StatusPending) || request.Status == string(StatusApproved) || request.Status == string(StatusProcessing)) {
			reqCopy := *request
			return &reqCopy, nil
		}
	}
	return nil, ErrDeletionNotFound
}

// Privacy Settings
func (m *MockRepository) CreatePrivacySettings(ctx context.Context, settings *PrivacySettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.privacySettings[settings.OrganizationID] = settings
	return nil
}

func (m *MockRepository) GetPrivacySettings(ctx context.Context, orgID string) (*PrivacySettings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	settings, ok := m.privacySettings[orgID]
	if !ok {
		return nil, ErrPrivacySettingsNotFound
	}
	settingsCopy := *settings
	return &settingsCopy, nil
}

func (m *MockRepository) UpdatePrivacySettings(ctx context.Context, settings *PrivacySettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.privacySettings[settings.OrganizationID] = settings
	return nil
}

// Analytics
func (m *MockRepository) GetConsentStats(ctx context.Context, orgID string, startDate, endDate time.Time) (map[string]interface{}, error) {
	return map[string]interface{}{
		"totalConsents":    len(m.consents),
		"grantedConsents":  0,
		"revokedConsents":  0,
		"pendingDeletions": len(m.deleteRequests),
		"dataExports":      len(m.exportRequests),
	}, nil
}

// Tests

func TestCreateConsent(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, DefaultConfig(), nil)

	ctx := context.Background()
	userID := "user_123"
	orgID := "org_456"

	req := &CreateConsentRequest{
		ConsentType: "marketing",
		Purpose:     "email_campaigns",
		Granted:     true,
		Version:     "1.0",
	}

	consent, err := service.CreateConsent(ctx, orgID, userID, req)
	if err != nil {
		t.Fatalf("Failed to create consent: %v", err)
	}

	if consent.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, consent.UserID)
	}

	if consent.ConsentType != "marketing" {
		t.Errorf("Expected consentType marketing, got %s", consent.ConsentType)
	}

	if !consent.Granted {
		t.Error("Expected consent to be granted")
	}
}

func TestRevokeConsent(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, DefaultConfig(), nil)

	ctx := context.Background()
	userID := "user_123"
	orgID := "org_456"

	// Create consent first
	req := &CreateConsentRequest{
		ConsentType: "marketing",
		Purpose:     "email_campaigns",
		Granted:     true,
		Version:     "1.0",
	}

	consent, _ := service.CreateConsent(ctx, orgID, userID, req)

	// Revoke it
	err := service.RevokeConsent(ctx, userID, orgID, "marketing", "email_campaigns")
	if err != nil {
		t.Fatalf("Failed to revoke consent: %v", err)
	}

	// Verify revocation
	updated, _ := service.GetConsent(ctx, consent.ID.String())
	if updated.Granted {
		t.Error("Expected consent to be revoked")
	}

	if updated.RevokedAt == nil {
		t.Error("Expected RevokedAt to be set")
	}
}

func TestConsentExpiry(t *testing.T) {
	repo := NewMockRepository()
	config := DefaultConfig()
	config.Expiry.Enabled = true
	config.Expiry.AutoExpireCheck = true
	service := NewService(repo, config, nil)

	ctx := context.Background()
	userID := "user_123"
	orgID := "org_456"

	// Create consent with expiry in the past
	pastDate := time.Now().Add(-24 * time.Hour)
	consent := &ConsentRecord{
		UserID:         userID,
		OrganizationID: orgID,
		ConsentType:    "marketing",
		Purpose:        "email_campaigns",
		Granted:        true,
		Version:        "1.0",
		ExpiresAt:      &pastDate,
		GrantedAt:      time.Now().Add(-48 * time.Hour),
	}
	repo.CreateConsent(ctx, consent)

	// Run expiry check
	count, err := service.ExpireConsents(ctx)
	if err != nil {
		t.Fatalf("Failed to expire consents: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 expired consent, got %d", count)
	}
}

func TestCookieConsent(t *testing.T) {
	repo := NewMockRepository()
	config := DefaultConfig()
	config.CookieConsent.Enabled = true
	service := NewService(repo, config, nil)

	ctx := context.Background()
	userID := "user_123"
	orgID := "org_456"

	req := &CookieConsentRequest{
		Functional:      true,
		Analytics:       false,
		Marketing:       false,
		Personalization: true,
		ThirdParty:      false,
		BannerVersion:   "1.0",
	}

	consent, err := service.RecordCookieConsent(ctx, orgID, userID, req)
	if err != nil {
		t.Fatalf("Failed to record cookie consent: %v", err)
	}

	if !consent.Essential {
		t.Error("Expected essential cookies to always be true")
	}

	if !consent.Functional {
		t.Error("Expected functional cookies to be enabled")
	}

	if consent.Analytics {
		t.Error("Expected analytics cookies to be disabled")
	}
}

func TestDataExportRequest(t *testing.T) {
	repo := NewMockRepository()
	config := DefaultConfig()
	config.DataExport.Enabled = true
	config.DataExport.MaxRequests = 5
	config.DataExport.RequestPeriod = 30 * 24 * time.Hour
	service := NewService(repo, config, nil)

	ctx := context.Background()
	userID := "user_123"
	orgID := "org_456"

	req := &DataExportRequestInput{
		Format:          "json",
		IncludeSections: []string{"profile", "consents"},
	}

	exportReq, err := service.RequestDataExport(ctx, userID, orgID, req)
	if err != nil {
		t.Fatalf("Failed to request data export: %v", err)
	}

	if exportReq.Status != string(StatusPending) {
		t.Errorf("Expected status pending, got %s", exportReq.Status)
	}

	if exportReq.Format != "json" {
		t.Errorf("Expected format json, got %s", exportReq.Format)
	}

	// Test duplicate request
	_, err = service.RequestDataExport(ctx, userID, orgID, req)
	if err != ErrExportAlreadyPending {
		t.Error("Expected error for duplicate pending export request")
	}
}

func TestDataDeletionRequest(t *testing.T) {
	repo := NewMockRepository()
	config := DefaultConfig()
	config.DataDeletion.Enabled = true
	config.DataDeletion.RequireAdminApproval = true
	config.DataDeletion.GracePeriodDays = 30
	service := NewService(repo, config, nil)

	ctx := context.Background()
	userID := "user_123"
	orgID := "org_456"

	req := &DataDeletionRequestInput{
		Reason:         "GDPR Article 17 request",
		DeleteSections: []string{"all"},
	}

	deletionReq, err := service.RequestDataDeletion(ctx, userID, orgID, req)
	if err != nil {
		t.Fatalf("Failed to request data deletion: %v", err)
	}

	if deletionReq.Status != string(StatusPending) {
		t.Errorf("Expected status pending, got %s", deletionReq.Status)
	}

	// Test duplicate request
	_, err = service.RequestDataDeletion(ctx, userID, orgID, req)
	if err != ErrDeletionAlreadyPending {
		t.Error("Expected error for duplicate pending deletion request")
	}

	// Approve deletion
	err = service.ApproveDeletionRequest(ctx, deletionReq.ID.String(), "admin_123", orgID)
	if err != nil {
		t.Fatalf("Failed to approve deletion: %v", err)
	}

	updated, _ := service.GetDeletionRequest(ctx, deletionReq.ID.String())
	if updated.Status != string(StatusApproved) {
		t.Errorf("Expected status approved, got %s", updated.Status)
	}
}

func TestConsentSummary(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, DefaultConfig(), nil)

	ctx := context.Background()
	userID := "user_123"
	orgID := "org_456"

	// Create multiple consents with different types
	consentsToCreate := []struct {
		consentType string
		purpose     string
		granted     bool
	}{
		{"marketing", "email_campaigns", true},
		{"analytics", "usage_tracking", true},
		{"cookies", "preferences", false},
	}

	// Create each consent and verify no errors
	for _, c := range consentsToCreate {
		_, err := service.CreateConsent(ctx, orgID, userID, &CreateConsentRequest{
			ConsentType: c.consentType,
			Purpose:     c.purpose,
			Granted:     c.granted,
			Version:     "1.0",
		})
		if err != nil {
			t.Fatalf("Failed to create consent %s/%s: %v", c.consentType, c.purpose, err)
		}
	}

	// Get summary
	summary, err := service.GetConsentSummary(ctx, userID, orgID)
	if err != nil {
		t.Fatalf("Failed to get consent summary: %v", err)
	}

	if summary.TotalConsents != 3 {
		t.Errorf("Expected 3 total consents, got %d", summary.TotalConsents)
	}

	if summary.GrantedConsents != 2 {
		t.Errorf("Expected 2 granted consents, got %d", summary.GrantedConsents)
	}

	if len(summary.ConsentsByType) != 3 {
		t.Errorf("Expected 3 consent types, got %d", len(summary.ConsentsByType))
	}

	// Verify the consent types are correct
	expectedTypes := map[string]bool{
		"marketing": true,
		"analytics": true,
		"cookies":   true,
	}
	for consentType := range summary.ConsentsByType {
		if !expectedTypes[consentType] {
			t.Errorf("Unexpected consent type in summary: %s", consentType)
		}
	}
}

func TestPrivacySettings(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, DefaultConfig(), nil)

	ctx := context.Background()
	orgID := "org_456"

	// Get settings (should create defaults)
	settings, err := service.GetPrivacySettings(ctx, orgID)
	if err != nil {
		t.Fatalf("Failed to get privacy settings: %v", err)
	}

	if !settings.ConsentRequired {
		t.Error("Expected consent required to be true by default")
	}

	// Update settings
	req := &PrivacySettingsRequest{
		GDPRMode:          ptrBool(true),
		CCPAMode:          ptrBool(false),
		DataRetentionDays: ptrInt(2555),
	}

	updated, err := service.UpdatePrivacySettings(ctx, orgID, "admin_123", req)
	if err != nil {
		t.Fatalf("Failed to update privacy settings: %v", err)
	}

	if !updated.GDPRMode {
		t.Error("Expected GDPR mode to be enabled")
	}

	if updated.CCPAMode {
		t.Error("Expected CCPA mode to be disabled")
	}
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrInt(i int) *int {
	return &i
}
