package compliance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_CreateProfile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	req := &CreateProfileRequest{
		AppID:             "org_123",
		Name:              "Test Profile",
		Standards:         []ComplianceStandard{StandardSOC2},
		MFARequired:       true,
		PasswordMinLength: 12,
		RetentionDays:     90,
	}

	// Act
	profile, err := service.CreateProfile(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, profile)
	assert.NotEmpty(t, profile.ID)
	assert.Equal(t, "org_123", profile.AppID)
	assert.Contains(t, profile.Standards, StandardSOC2)
	assert.Equal(t, "active", profile.Status)
	assert.True(t, profile.MFARequired)
	assert.Equal(t, 12, profile.PasswordMinLength)
}

func TestService_CreateProfile_DuplicateOrganization(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	req := &CreateProfileRequest{
		AppID:             "org_123",
		Name:              "Test Profile",
		Standards:         []ComplianceStandard{StandardSOC2},
		MFARequired:       true,
		PasswordMinLength: 12,
	}

	// Create first profile
	_, err := service.CreateProfile(ctx, req)
	require.NoError(t, err)

	// Act - try to create second profile for same org
	_, err = service.CreateProfile(ctx, req)

	// Assert
	assert.Error(t, err)
	// ProfileExists error is defined in errors.go
}

func TestService_GetProfile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	// Create a profile
	created, err := service.CreateProfile(ctx, &CreateProfileRequest{
		AppID:             "org_123",
		Name:              "Test Profile",
		Standards:         []ComplianceStandard{StandardSOC2},
		PasswordMinLength: 12,
	})
	require.NoError(t, err)

	// Act
	retrieved, err := service.GetProfile(ctx, created.ID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.AppID, retrieved.AppID)
}

func TestService_GetProfile_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	// Act
	_, err := service.GetProfile(ctx, "non_existent_id")

	// Assert
	assert.Error(t, err)
	// ProfileNotFound error is defined in errors.go
}

func TestService_UpdateProfile(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	// Create initial profile
	profile, err := service.CreateProfile(ctx, &CreateProfileRequest{
		AppID:             "org_123",
		Name:              "Test Profile",
		Standards:         []ComplianceStandard{StandardSOC2},
		PasswordMinLength: 12,
		MFARequired:       false,
	})
	require.NoError(t, err)

	// Act - update profile
	mfaRequired := true
	retentionDays := 365
	req := &UpdateProfileRequest{
		MFARequired:   &mfaRequired,
		RetentionDays: &retentionDays,
	}
	updated, err := service.UpdateProfile(ctx, profile.ID, req)

	// Assert
	require.NoError(t, err)
	assert.True(t, updated.MFARequired)
	assert.Equal(t, 365, updated.RetentionDays)
}

func TestService_CreateProfileFromTemplate(t *testing.T) {
	// Arrange
	ctx := context.Background()
	service := setupTestService(t)

	// Act
	profile, err := service.CreateProfileFromTemplate(ctx, "org_456", StandardHIPAA)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, "org_456", profile.AppID)
	assert.Contains(t, profile.Standards, StandardHIPAA)
	// HIPAA requires MFA
	assert.True(t, profile.MFARequired)
}

// Helper functions

func setupTestService(t *testing.T) *Service {
	t.Helper()

	mockRepo := NewMockRepository()
	mockAudit := &MockAuditService{}
	mockUser := &MockUserService{}
	mockApp := &MockAppService{}
	mockEmail := &MockEmailService{}

	config := DefaultConfig()

	service := NewService(
		mockRepo,
		config,
		mockAudit,
		mockUser,
		mockApp,
		mockEmail,
	)

	return service
}

func createTestProfile(t *testing.T, service *Service, appID string, standard ComplianceStandard) *ComplianceProfile {
	t.Helper()

	profile, err := service.CreateProfileFromTemplate(context.Background(), appID, standard)
	require.NoError(t, err)

	return profile
}

// TODO: Additional tests should be added for:
// - Checks (RunCheck, ListChecks)
// - Violations (ListViolations)
// - Reports (ListReports)
// - Evidence (ListEvidence)
// - Policies (ListPolicies)
// - Training (ListTraining)
