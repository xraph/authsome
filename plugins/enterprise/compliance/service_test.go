package compliance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_CreateProfile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	req := &CreateProfileRequest{
		OrganizationID: "org_123",
		Standard:       StandardSOC2,
		Requirements: []string{
			RequirementPasswordMinLength12,
			RequirementMFARequired,
		},
	}

	// Act
	profile, err := service.CreateProfile(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, profile)
	assert.NotEmpty(t, profile.ID)
	assert.Equal(t, "org_123", profile.OrganizationID)
	assert.Equal(t, StandardSOC2, profile.Standard)
	assert.Equal(t, StatusActive, profile.Status)
	assert.Contains(t, profile.Requirements, RequirementPasswordMinLength12)
	assert.Contains(t, profile.Requirements, RequirementMFARequired)
}

func TestService_CreateProfile_DuplicateOrganization(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	req := &CreateProfileRequest{
		OrganizationID: "org_123",
		Standard:       StandardSOC2,
		Requirements:   []string{RequirementPasswordMinLength12},
	}

	// Create first profile
	_, err := service.CreateProfile(ctx, req)
	require.NoError(t, err)

	// Act - try to create second profile for same org
	_, err = service.CreateProfile(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrProfileAlreadyExists, err)
}

func TestService_GetProfile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	// Create a profile
	created, err := service.CreateProfile(ctx, &CreateProfileRequest{
		OrganizationID: "org_123",
		Standard:       StandardSOC2,
		Requirements:   []string{RequirementPasswordMinLength12},
	})
	require.NoError(t, err)

	// Act
	retrieved, err := service.GetProfile(ctx, created.ID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.OrganizationID, retrieved.OrganizationID)
}

func TestService_GetProfile_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	// Act
	_, err := service.GetProfile(ctx, "non_existent_id")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrProfileNotFound, err)
}

func TestService_UpdateProfile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	// Create initial profile
	profile, err := service.CreateProfile(ctx, &CreateProfileRequest{
		OrganizationID: "org_123",
		Standard:       StandardSOC2,
		Requirements:   []string{RequirementPasswordMinLength12},
	})
	require.NoError(t, err)

	// Act - update requirements
	req := &UpdateProfileRequest{
		Requirements: []string{
			RequirementPasswordMinLength12,
			RequirementMFARequired,
			RequirementSessionTimeout30m,
		},
	}
	updated, err := service.UpdateProfile(ctx, profile.ID, req)

	// Assert
	require.NoError(t, err)
	assert.Len(t, updated.Requirements, 3)
	assert.Contains(t, updated.Requirements, RequirementMFARequired)
	assert.Contains(t, updated.Requirements, RequirementSessionTimeout30m)
}

func TestService_DeleteProfile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	profile, err := service.CreateProfile(ctx, &CreateProfileRequest{
		OrganizationID: "org_123",
		Standard:       StandardSOC2,
		Requirements:   []string{RequirementPasswordMinLength12},
	})
	require.NoError(t, err)

	// Act
	err = service.DeleteProfile(ctx, profile.ID)

	// Assert
	require.NoError(t, err)

	// Verify it's gone
	_, err = service.GetProfile(ctx, profile.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrProfileNotFound, err)
}

func TestService_RunChecks(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	profile := createTestProfile(t, service, "org_123", StandardSOC2)

	// Act
	checks, err := service.RunChecks(ctx, profile.ID)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, checks)

	// Verify all check types are covered
	checkTypes := make(map[CheckType]bool)
	for _, check := range checks {
		checkTypes[check.CheckType] = true
	}

	assert.True(t, checkTypes[CheckTypePasswordStrength])
	assert.True(t, checkTypes[CheckTypeMFACoverage])
	assert.True(t, checkTypes[CheckTypeSessionManagement])
	assert.True(t, checkTypes[CheckTypeAccessControl])
	assert.True(t, checkTypes[CheckTypeAuditLog])
	assert.True(t, checkTypes[CheckTypeDataRetention])
}

func TestService_GenerateReport(t *testing.T) {
	tests := []struct {
		name     string
		standard ComplianceStandard
		wantErr  bool
	}{
		{
			name:     "SOC 2 report",
			standard: StandardSOC2,
			wantErr:  false,
		},
		{
			name:     "HIPAA report",
			standard: StandardHIPAA,
			wantErr:  false,
		},
		{
			name:     "PCI-DSS report",
			standard: StandardPCIDSS,
			wantErr:  false,
		},
		{
			name:     "ISO 27001 report",
			standard: StandardISO27001,
			wantErr:  false,
		},
		{
			name:     "GDPR report",
			standard: StandardGDPR,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			service := setupTestService(t)
			profile := createTestProfile(t, service, "org_123", tt.standard)

			req := &GenerateReportRequest{
				ProfileID: profile.ID,
				Standard:  tt.standard,
				Format:    "pdf",
			}

			// Act
			report, err := service.GenerateReport(ctx, req)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, report)
				assert.NotEmpty(t, report.Content)
				assert.Equal(t, tt.standard, report.Standard)
				assert.Equal(t, "pdf", report.Format)
			}
		})
	}
}

func TestService_CreateViolation(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)
	profile := createTestProfile(t, service, "org_123", StandardSOC2)

	req := &CreateViolationRequest{
		ProfileID:      profile.ID,
		OrganizationID: "org_123",
		UserID:         "user_123",
		ViolationType:  "weak_password",
		Severity:       SeverityHigh,
		Description:    "Password does not meet minimum length requirement",
	}

	// Act
	violation, err := service.CreateViolation(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, violation.ID)
	assert.Equal(t, "weak_password", violation.ViolationType)
	assert.Equal(t, SeverityHigh, violation.Severity)
	assert.Equal(t, StatusOpen, violation.Status)
}

func TestService_ResolveViolation(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)
	profile := createTestProfile(t, service, "org_123", StandardSOC2)

	// Create a violation
	violation, err := service.CreateViolation(ctx, &CreateViolationRequest{
		ProfileID:      profile.ID,
		OrganizationID: "org_123",
		UserID:         "user_123",
		ViolationType:  "weak_password",
		Severity:       SeverityHigh,
		Description:    "Password does not meet minimum length requirement",
	})
	require.NoError(t, err)

	// Act
	err = service.ResolveViolation(ctx, violation.ID, "admin_user")

	// Assert
	require.NoError(t, err)

	// Verify status changed
	resolved, err := service.repo.GetViolation(ctx, violation.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusResolved, resolved.Status)
	assert.Equal(t, "admin_user", resolved.ResolvedBy)
}

func TestService_GetComplianceStatus(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)
	profile := createTestProfile(t, service, "org_123", StandardSOC2)

	// Run some checks first
	_, err := service.RunChecks(ctx, profile.ID)
	require.NoError(t, err)

	// Act
	status, err := service.GetComplianceStatus(ctx, profile.OrganizationID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, profile.OrganizationID, status.OrganizationID)
	assert.GreaterOrEqual(t, status.ComplianceScore, 0)
	assert.LessOrEqual(t, status.ComplianceScore, 100)
}

func TestService_CreateEvidence(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)
	profile := createTestProfile(t, service, "org_123", StandardSOC2)

	req := &CreateEvidenceRequest{
		ProfileID:      profile.ID,
		OrganizationID: "org_123",
		Title:          "Password Policy Document",
		Description:    "Corporate password policy document",
		Category:       "policy",
		EvidenceType:   "document",
		Content:        "Policy content here...",
	}

	// Act
	evidence, err := service.CreateEvidence(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, evidence.ID)
	assert.Equal(t, "Password Policy Document", evidence.Title)
	assert.Equal(t, "policy", evidence.Category)
}

func TestService_CreatePolicy(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)
	profile := createTestProfile(t, service, "org_123", StandardSOC2)

	req := &CreatePolicyRequest{
		OrganizationID: "org_123",
		ProfileID:      profile.ID,
		PolicyType:     "password",
		Name:           "Password Requirements",
		Description:    "Password must meet minimum complexity requirements",
		Rules:          map[string]interface{}{"min_length": 12, "require_special": true},
		Enabled:        true,
	}

	// Act
	policy, err := service.CreatePolicy(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, policy.ID)
	assert.Equal(t, "password", policy.PolicyType)
	assert.True(t, policy.Enabled)
}

func TestService_CreateTraining(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)
	profile := createTestProfile(t, service, "org_123", StandardSOC2)

	req := &CreateTrainingRequest{
		ProfileID:      profile.ID,
		OrganizationID: "org_123",
		UserID:         "user_123",
		TrainingType:   "security_awareness",
		Standard:       StandardSOC2,
		DueDate:        time.Now().Add(30 * 24 * time.Hour),
	}

	// Act
	training, err := service.CreateTraining(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, training.ID)
	assert.Equal(t, "security_awareness", training.TrainingType)
	assert.Equal(t, StatusRequired, training.Status)
}

func TestService_CompleteTraining(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)
	profile := createTestProfile(t, service, "org_123", StandardSOC2)

	// Create training
	training, err := service.CreateTraining(ctx, &CreateTrainingRequest{
		ProfileID:      profile.ID,
		OrganizationID: "org_123",
		UserID:         "user_123",
		TrainingType:   "security_awareness",
		Standard:       StandardSOC2,
		DueDate:        time.Now().Add(30 * 24 * time.Hour),
	})
	require.NoError(t, err)

	// Act
	err = service.CompleteTraining(ctx, training.ID, "user_123")

	// Assert
	require.NoError(t, err)

	// Verify status changed
	completed, err := service.repo.GetTraining(ctx, training.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, completed.Status)
}

func TestService_CreateProfileFromTemplate(t *testing.T) {
	tests := []struct {
		name     string
		standard ComplianceStandard
		wantErr  bool
	}{
		{
			name:     "SOC 2 template",
			standard: StandardSOC2,
			wantErr:  false,
		},
		{
			name:     "HIPAA template",
			standard: StandardHIPAA,
			wantErr:  false,
		},
		{
			name:     "PCI-DSS template",
			standard: StandardPCIDSS,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			service := setupTestService(t)

			// Act
			profile, err := service.CreateProfileFromTemplate(ctx, "org_123", tt.standard)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, profile)
				assert.Equal(t, tt.standard, profile.Standard)
				assert.NotEmpty(t, profile.Requirements)
			}
		})
	}
}

// Helper functions

func setupTestService(t *testing.T) *Service {
	t.Helper()

	mockRepo := NewMockRepository()
	mockAudit := NewMockAuditService()
	mockUser := NewMockUserService()
	mockOrg := NewMockOrganizationService()
	mockEmail := NewMockEmailService()

	config := DefaultConfig()

	return NewService(
		mockRepo,
		config,
		mockAudit,
		mockUser,
		mockOrg,
		mockEmail,
	)
}

func createTestProfile(t *testing.T, service *Service, orgID string, standard ComplianceStandard) *ComplianceProfile {
	t.Helper()

	profile, err := service.CreateProfile(context.Background(), &CreateProfileRequest{
		OrganizationID: orgID,
		Standard:       standard,
		Requirements: []string{
			RequirementPasswordMinLength12,
			RequirementMFARequired,
		},
	})

	require.NoError(t, err)
	return profile
}
