package idverification

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/xraph/authsome/schema"
)

// Mock repository for testing
type mockRepository struct {
	verifications map[string]*schema.IdentityVerification
	sessions      map[string]*schema.IdentityVerificationSession
	statuses      map[string]*schema.UserVerificationStatus
	documents     map[string]*schema.IdentityVerificationDocument

	createVerificationError error
	getVerificationError    error
	updateVerificationError error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		verifications: make(map[string]*schema.IdentityVerification),
		sessions:      make(map[string]*schema.IdentityVerificationSession),
		statuses:      make(map[string]*schema.UserVerificationStatus),
		documents:     make(map[string]*schema.IdentityVerificationDocument),
	}
}

func (m *mockRepository) CreateVerification(ctx context.Context, v *schema.IdentityVerification) error {
	if m.createVerificationError != nil {
		return m.createVerificationError
	}
	m.verifications[v.ID] = v
	return nil
}

func (m *mockRepository) GetVerificationByID(ctx context.Context, id string) (*schema.IdentityVerification, error) {
	if m.getVerificationError != nil {
		return nil, m.getVerificationError
	}
	return m.verifications[id], nil
}

func (m *mockRepository) GetVerificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var result []*schema.IdentityVerification
	for _, v := range m.verifications {
		if v.UserID == userID {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockRepository) GetVerificationsByOrgID(ctx context.Context, orgID string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var result []*schema.IdentityVerification
	for _, v := range m.verifications {
		if v.OrganizationID == orgID {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateVerification(ctx context.Context, v *schema.IdentityVerification) error {
	if m.updateVerificationError != nil {
		return m.updateVerificationError
	}
	m.verifications[v.ID] = v
	return nil
}

func (m *mockRepository) DeleteVerification(ctx context.Context, id string) error {
	delete(m.verifications, id)
	return nil
}

func (m *mockRepository) GetLatestVerificationByUser(ctx context.Context, userID string) (*schema.IdentityVerification, error) {
	var latest *schema.IdentityVerification
	for _, v := range m.verifications {
		if v.UserID == userID {
			if latest == nil || v.CreatedAt.After(latest.CreatedAt) {
				latest = v
			}
		}
	}
	return latest, nil
}

func (m *mockRepository) GetVerificationByProviderCheckID(ctx context.Context, providerCheckID string) (*schema.IdentityVerification, error) {
	for _, v := range m.verifications {
		if v.ProviderCheckID == providerCheckID {
			return v, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) GetVerificationsByStatus(ctx context.Context, status string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var result []*schema.IdentityVerification
	for _, v := range m.verifications {
		if v.Status == status {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockRepository) GetVerificationsByType(ctx context.Context, verificationType string, limit, offset int) ([]*schema.IdentityVerification, error) {
	var result []*schema.IdentityVerification
	for _, v := range m.verifications {
		if v.VerificationType == verificationType {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockRepository) CountVerificationsByUser(ctx context.Context, userID string, since time.Time) (int, error) {
	count := 0
	for _, v := range m.verifications {
		if v.UserID == userID && v.CreatedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (m *mockRepository) GetExpiredVerifications(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerification, error) {
	var result []*schema.IdentityVerification
	for _, v := range m.verifications {
		if v.ExpiresAt != nil && v.ExpiresAt.Before(before) && v.Status != "expired" {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockRepository) CreateDocument(ctx context.Context, d *schema.IdentityVerificationDocument) error {
	m.documents[d.ID] = d
	return nil
}

func (m *mockRepository) GetDocumentByID(ctx context.Context, id string) (*schema.IdentityVerificationDocument, error) {
	return m.documents[id], nil
}

func (m *mockRepository) GetDocumentsByVerificationID(ctx context.Context, verificationID string) ([]*schema.IdentityVerificationDocument, error) {
	var result []*schema.IdentityVerificationDocument
	for _, d := range m.documents {
		if d.VerificationID == verificationID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateDocument(ctx context.Context, d *schema.IdentityVerificationDocument) error {
	m.documents[d.ID] = d
	return nil
}

func (m *mockRepository) DeleteDocument(ctx context.Context, id string) error {
	delete(m.documents, id)
	return nil
}

func (m *mockRepository) GetDocumentsForDeletion(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerificationDocument, error) {
	var result []*schema.IdentityVerificationDocument
	for _, d := range m.documents {
		if d.RetainUntil != nil && d.RetainUntil.Before(before) && d.DeletedAt == nil {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockRepository) CreateSession(ctx context.Context, s *schema.IdentityVerificationSession) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepository) GetSessionByID(ctx context.Context, id string) (*schema.IdentityVerificationSession, error) {
	return m.sessions[id], nil
}

func (m *mockRepository) GetSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]*schema.IdentityVerificationSession, error) {
	var result []*schema.IdentityVerificationSession
	for _, s := range m.sessions {
		if s.UserID == userID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateSession(ctx context.Context, s *schema.IdentityVerificationSession) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepository) DeleteSession(ctx context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *mockRepository) GetExpiredSessions(ctx context.Context, before time.Time, limit int) ([]*schema.IdentityVerificationSession, error) {
	var result []*schema.IdentityVerificationSession
	for _, s := range m.sessions {
		if s.ExpiresAt.Before(before) && s.Status != "expired" {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockRepository) CreateUserVerificationStatus(ctx context.Context, s *schema.UserVerificationStatus) error {
	m.statuses[s.UserID] = s
	return nil
}

func (m *mockRepository) GetUserVerificationStatus(ctx context.Context, userID string) (*schema.UserVerificationStatus, error) {
	return m.statuses[userID], nil
}

func (m *mockRepository) UpdateUserVerificationStatus(ctx context.Context, s *schema.UserVerificationStatus) error {
	m.statuses[s.UserID] = s
	return nil
}

func (m *mockRepository) DeleteUserVerificationStatus(ctx context.Context, userID string) error {
	delete(m.statuses, userID)
	return nil
}

func (m *mockRepository) GetUsersRequiringReverification(ctx context.Context, limit int) ([]*schema.UserVerificationStatus, error) {
	var result []*schema.UserVerificationStatus
	for _, s := range m.statuses {
		if s.RequiresReverification {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockRepository) GetUsersByVerificationLevel(ctx context.Context, level string, limit, offset int) ([]*schema.UserVerificationStatus, error) {
	var result []*schema.UserVerificationStatus
	for _, s := range m.statuses {
		if s.VerificationLevel == level {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockRepository) GetBlockedUsers(ctx context.Context, limit, offset int) ([]*schema.UserVerificationStatus, error) {
	var result []*schema.UserVerificationStatus
	for _, s := range m.statuses {
		if s.IsBlocked {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockRepository) GetVerificationStats(ctx context.Context, orgID string, from, to time.Time) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_verifications":      len(m.verifications),
		"successful_verifications": 0,
		"failed_verifications":     0,
		"pending_verifications":    0,
	}, nil
}

func (m *mockRepository) GetProviderStats(ctx context.Context, provider string, from, to time.Time) (map[string]interface{}, error) {
	return map[string]interface{}{
		"provider":          provider,
		"total_checks":      0,
		"successful_checks": 0,
		"failed_checks":     0,
		"error_rate":        0.0,
	}, nil
}

// Mock provider for testing
type mockProvider struct {
	sessionResponse *ProviderSession
	sessionError    error
	checkResult     *ProviderCheckResult
	checkError      error
}

func (m *mockProvider) CreateSession(ctx context.Context, req *ProviderSessionRequest) (*ProviderSession, error) {
	if m.sessionError != nil {
		return nil, m.sessionError
	}
	return m.sessionResponse, nil
}

func (m *mockProvider) GetSession(ctx context.Context, sessionID string) (*ProviderSession, error) {
	return m.sessionResponse, nil
}

func (m *mockProvider) GetCheck(ctx context.Context, checkID string) (*ProviderCheckResult, error) {
	if m.checkError != nil {
		return nil, m.checkError
	}
	return m.checkResult, nil
}

func (m *mockProvider) VerifyWebhook(signature, payload string) (bool, error) {
	return true, nil
}

func (m *mockProvider) ParseWebhook(payload []byte) (*WebhookPayload, error) {
	return &WebhookPayload{
		EventType: "verification.completed",
	}, nil
}

func (m *mockProvider) GetProviderName() string {
	return "mock"
}

// Tests

func TestService_CreateVerification(t *testing.T) {
	repo := newMockRepository()
	config := DefaultConfig()
	config.MaxVerificationAttempts = 3
	config.Onfido.Enabled = true
	config.Onfido.APIToken = "test_token"

	service, err := NewService(repo, config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		req := &CreateVerificationRequest{
			UserID:           "user_123",
			OrganizationID:   "org_456",
			Provider:         "onfido",
			ProviderCheckID:  "check_789",
			VerificationType: "document",
			DocumentType:     "passport",
		}

		verification, err := service.CreateVerification(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if verification == nil {
			t.Error("Expected verification, got nil")
		}

		if verification.UserID != req.UserID {
			t.Errorf("Expected UserID %s, got %s", req.UserID, verification.UserID)
		}

		if verification.Status != "pending" {
			t.Errorf("Expected status 'pending', got %s", verification.Status)
		}
	})

	t.Run("blocked user", func(t *testing.T) {
		// Block the user
		repo.statuses["user_blocked"] = &schema.UserVerificationStatus{
			UserID:    "user_blocked",
			IsBlocked: true,
		}

		req := &CreateVerificationRequest{
			UserID:           "user_blocked",
			OrganizationID:   "org_456",
			Provider:         "onfido",
			VerificationType: "document",
		}

		_, err := service.CreateVerification(ctx, req)
		if err != ErrVerificationBlocked {
			t.Errorf("Expected ErrVerificationBlocked, got %v", err)
		}
	})

	t.Run("max attempts reached", func(t *testing.T) {
		userID := "user_maxed"

		// Create 3 verifications in last 24 hours
		for i := 0; i < 3; i++ {
			v := &schema.IdentityVerification{
				ID:        fmt.Sprintf("ver_%d", i),
				UserID:    userID,
				CreatedAt: time.Now(),
			}
			repo.verifications[v.ID] = v
		}

		req := &CreateVerificationRequest{
			UserID:           userID,
			OrganizationID:   "org_456",
			Provider:         "onfido",
			VerificationType: "document",
		}

		_, err := service.CreateVerification(ctx, req)
		if err != ErrMaxAttemptsReached {
			t.Errorf("Expected ErrMaxAttemptsReached, got %v", err)
		}
	})
}

func TestService_ProcessVerificationResult(t *testing.T) {
	repo := newMockRepository()
	config := DefaultConfig()
	config.MaxAllowedRiskScore = 70
	config.MinConfidenceScore = 80
	config.Onfido.Enabled = true
	config.Onfido.APIToken = "test_token"

	service, err := NewService(repo, config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()

	t.Run("successful verification", func(t *testing.T) {
		// Create a verification
		verification := &schema.IdentityVerification{
			ID:               "ver_123",
			UserID:           "user_123",
			OrganizationID:   "org_456",
			VerificationType: "document",
			Status:           "pending",
			CreatedAt:        time.Now(),
		}
		repo.verifications[verification.ID] = verification

		// Process result
		result := &VerificationResult{
			Status:          "completed",
			IsVerified:      true,
			RiskScore:       30,
			RiskLevel:       "low",
			ConfidenceScore: 95,
		}

		err := service.ProcessVerificationResult(ctx, verification.ID, result)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check updated verification
		updated := repo.verifications[verification.ID]
		if !updated.IsVerified {
			t.Error("Expected IsVerified to be true")
		}

		if updated.RiskScore != 30 {
			t.Errorf("Expected RiskScore 30, got %d", updated.RiskScore)
		}
	})

	t.Run("high risk rejection", func(t *testing.T) {
		verification := &schema.IdentityVerification{
			ID:               "ver_456",
			UserID:           "user_456",
			OrganizationID:   "org_456",
			VerificationType: "document",
			Status:           "pending",
			CreatedAt:        time.Now(),
		}
		repo.verifications[verification.ID] = verification

		result := &VerificationResult{
			Status:          "completed",
			IsVerified:      true,
			RiskScore:       85, // Above threshold
			ConfidenceScore: 90,
		}

		err := service.ProcessVerificationResult(ctx, verification.ID, result)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		updated := repo.verifications[verification.ID]
		if updated.Status != "failed" {
			t.Errorf("Expected status 'failed', got %s", updated.Status)
		}
	})

	t.Run("low confidence rejection", func(t *testing.T) {
		verification := &schema.IdentityVerification{
			ID:               "ver_789",
			UserID:           "user_789",
			OrganizationID:   "org_456",
			VerificationType: "document",
			Status:           "pending",
			CreatedAt:        time.Now(),
		}
		repo.verifications[verification.ID] = verification

		result := &VerificationResult{
			Status:          "completed",
			IsVerified:      true,
			RiskScore:       50,
			ConfidenceScore: 70, // Below threshold
		}

		err := service.ProcessVerificationResult(ctx, verification.ID, result)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		updated := repo.verifications[verification.ID]
		if updated.Status != "failed" {
			t.Errorf("Expected status 'failed', got %s", updated.Status)
		}
	})
}

func TestService_GetUserVerificationStatus(t *testing.T) {
	repo := newMockRepository()
	config := DefaultConfig()
	config.Onfido.Enabled = true
	config.Onfido.APIToken = "test_token"

	service, err := NewService(repo, config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()

	t.Run("existing status", func(t *testing.T) {
		repo.statuses["user_123"] = &schema.UserVerificationStatus{
			ID:                "status_123",
			UserID:            "user_123",
			IsVerified:        true,
			VerificationLevel: "full",
		}

		status, err := service.GetUserVerificationStatus(ctx, "user_123")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if status.VerificationLevel != "full" {
			t.Errorf("Expected level 'full', got %s", status.VerificationLevel)
		}
	})

	t.Run("non-existing status", func(t *testing.T) {
		status, err := service.GetUserVerificationStatus(ctx, "user_new")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if status.VerificationLevel != "none" {
			t.Errorf("Expected level 'none', got %s", status.VerificationLevel)
		}
	})
}

func TestService_BlockUser(t *testing.T) {
	repo := newMockRepository()
	config := DefaultConfig()
	config.Onfido.Enabled = true
	config.Onfido.APIToken = "test_token"

	service, err := NewService(repo, config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()

	t.Run("block user", func(t *testing.T) {
		err := service.BlockUser(ctx, "user_123", "org_456", "Suspicious activity")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		status := repo.statuses["user_123"]
		if status == nil {
			t.Fatal("Expected status to be created")
		}

		if !status.IsBlocked {
			t.Error("Expected IsBlocked to be true")
		}

		if status.BlockReason != "Suspicious activity" {
			t.Errorf("Expected BlockReason 'Suspicious activity', got %s", status.BlockReason)
		}
	})
}

func TestService_UnblockUser(t *testing.T) {
	repo := newMockRepository()
	config := DefaultConfig()
	config.Onfido.Enabled = true
	config.Onfido.APIToken = "test_token"

	service, err := NewService(repo, config, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()

	t.Run("unblock user", func(t *testing.T) {
		// Create blocked status
		repo.statuses["user_123"] = &schema.UserVerificationStatus{
			ID:          "status_123",
			UserID:      "user_123",
			IsBlocked:   true,
			BlockReason: "Test",
		}

		err := service.UnblockUser(ctx, "user_123", "org_456")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		status := repo.statuses["user_123"]
		if status.IsBlocked {
			t.Error("Expected IsBlocked to be false")
		}

		if status.BlockReason != "" {
			t.Error("Expected BlockReason to be empty")
		}
	})
}

func TestCalculateAge(t *testing.T) {
	tests := []struct {
		name     string
		dob      time.Time
		expected int
	}{
		{
			name:     "18 years old",
			dob:      time.Date(time.Now().Year()-18, time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC),
			expected: 18,
		},
		{
			name:     "21 years old",
			dob:      time.Date(time.Now().Year()-21, time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC),
			expected: 21,
		},
		{
			name:     "not yet birthday",
			dob:      time.Date(time.Now().Year()-18, time.Now().Month(), time.Now().Day()+1, 0, 0, 0, 0, time.UTC),
			expected: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age := calculateAge(tt.dob)
			if age != tt.expected {
				t.Errorf("Expected age %d, got %d", tt.expected, age)
			}
		})
	}
}
