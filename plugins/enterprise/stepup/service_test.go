package stepup

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock repository for testing.
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) CreateVerification(ctx context.Context, verification *StepUpVerification) error {
	args := m.Called(ctx, verification)

	return args.Error(0)
}

func (m *mockRepository) GetVerification(ctx context.Context, id string) (*StepUpVerification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*StepUpVerification), args.Error(1)
}

func (m *mockRepository) GetLatestVerification(ctx context.Context, userID, orgID string, level SecurityLevel) (*StepUpVerification, error) {
	args := m.Called(ctx, userID, orgID, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*StepUpVerification), args.Error(1)
}

func (m *mockRepository) ListVerifications(ctx context.Context, userID, orgID string, limit, offset int) ([]*StepUpVerification, error) {
	args := m.Called(ctx, userID, orgID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*StepUpVerification), args.Error(1)
}

func (m *mockRepository) CreateRequirement(ctx context.Context, requirement *StepUpRequirement) error {
	args := m.Called(ctx, requirement)

	return args.Error(0)
}

func (m *mockRepository) GetRequirement(ctx context.Context, id string) (*StepUpRequirement, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*StepUpRequirement), args.Error(1)
}

func (m *mockRepository) GetRequirementByToken(ctx context.Context, token string) (*StepUpRequirement, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*StepUpRequirement), args.Error(1)
}

func (m *mockRepository) UpdateRequirement(ctx context.Context, requirement *StepUpRequirement) error {
	args := m.Called(ctx, requirement)

	return args.Error(0)
}

func (m *mockRepository) ListPendingRequirements(ctx context.Context, userID, orgID string) ([]*StepUpRequirement, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*StepUpRequirement), args.Error(1)
}

func (m *mockRepository) DeleteExpiredRequirements(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *mockRepository) CreateRememberedDevice(ctx context.Context, device *StepUpRememberedDevice) error {
	args := m.Called(ctx, device)

	return args.Error(0)
}

func (m *mockRepository) GetRememberedDevice(ctx context.Context, userID, orgID, deviceID string) (*StepUpRememberedDevice, error) {
	args := m.Called(ctx, userID, orgID, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*StepUpRememberedDevice), args.Error(1)
}

func (m *mockRepository) ListRememberedDevices(ctx context.Context, userID, orgID string) ([]*StepUpRememberedDevice, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*StepUpRememberedDevice), args.Error(1)
}

func (m *mockRepository) UpdateRememberedDevice(ctx context.Context, device *StepUpRememberedDevice) error {
	args := m.Called(ctx, device)

	return args.Error(0)
}

func (m *mockRepository) DeleteRememberedDevice(ctx context.Context, id string) error {
	args := m.Called(ctx, id)

	return args.Error(0)
}

func (m *mockRepository) DeleteExpiredRememberedDevices(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *mockRepository) CreateAttempt(ctx context.Context, attempt *StepUpAttempt) error {
	args := m.Called(ctx, attempt)

	return args.Error(0)
}

func (m *mockRepository) ListAttempts(ctx context.Context, requirementID string) ([]*StepUpAttempt, error) {
	args := m.Called(ctx, requirementID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*StepUpAttempt), args.Error(1)
}

func (m *mockRepository) CountFailedAttempts(ctx context.Context, userID, orgID string, since time.Time) (int, error) {
	args := m.Called(ctx, userID, orgID, since)

	return args.Int(0), args.Error(1)
}

func (m *mockRepository) CreatePolicy(ctx context.Context, policy *StepUpPolicy) error {
	args := m.Called(ctx, policy)

	return args.Error(0)
}

func (m *mockRepository) GetPolicy(ctx context.Context, id string) (*StepUpPolicy, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*StepUpPolicy), args.Error(1)
}

func (m *mockRepository) ListPolicies(ctx context.Context, orgID string) ([]*StepUpPolicy, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*StepUpPolicy), args.Error(1)
}

func (m *mockRepository) UpdatePolicy(ctx context.Context, policy *StepUpPolicy) error {
	args := m.Called(ctx, policy)

	return args.Error(0)
}

func (m *mockRepository) DeletePolicy(ctx context.Context, id string) error {
	args := m.Called(ctx, id)

	return args.Error(0)
}

func (m *mockRepository) CreateAuditLog(ctx context.Context, log *StepUpAuditLog) error {
	args := m.Called(ctx, log)

	return args.Error(0)
}

func (m *mockRepository) ListAuditLogs(ctx context.Context, userID, orgID string, limit, offset int) ([]*StepUpAuditLog, error) {
	args := m.Called(ctx, userID, orgID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*StepUpAuditLog), args.Error(1)
}

// Tests

func TestEvaluateRequirement_NoStepUpForSmallAmount(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	config.AmountRules = []AmountRule{} // No amount rules - small amounts don't trigger step-up
	service := NewService(repo, config, nil)

	// Mock no existing verifications
	repo.On("GetLatestVerification", mock.Anything, "user123", "org123", SecurityLevelCritical).
		Return(nil, assert.AnError)
	repo.On("GetLatestVerification", mock.Anything, "user123", "org123", SecurityLevelHigh).
		Return(nil, assert.AnError)
	repo.On("GetLatestVerification", mock.Anything, "user123", "org123", SecurityLevelMedium).
		Return(nil, assert.AnError)
	repo.On("GetRememberedDevice", mock.Anything, "user123", "org123", "").
		Return(nil, assert.AnError)
	repo.On("ListPolicies", mock.Anything, "org123").
		Return([]*StepUpPolicy{}, nil)

	// Execute
	evalCtx := &EvaluationContext{
		UserID:   "user123",
		OrgID:    "org123",
		Amount:   50,
		Currency: "USD",
	}

	result, err := service.EvaluateRequirement(context.Background(), evalCtx)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Required)
}

func TestEvaluateRequirement_MediumSecurityForMediumAmount(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	service := NewService(repo, config, nil)

	// Mock no existing verifications
	repo.On("GetLatestVerification", mock.Anything, "user123", "org123", mock.Anything).
		Return(nil, assert.AnError)
	repo.On("GetRememberedDevice", mock.Anything, "user123", "org123", "").
		Return(nil, assert.AnError)
	repo.On("ListPolicies", mock.Anything, "org123").
		Return([]*StepUpPolicy{}, nil)
	repo.On("CreateRequirement", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.Anything).
		Return(nil)

	// Execute
	evalCtx := &EvaluationContext{
		UserID:   "user123",
		OrgID:    "org123",
		Amount:   500,
		Currency: "USD",
	}

	result, err := service.EvaluateRequirement(context.Background(), evalCtx)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Required)
	assert.Equal(t, SecurityLevelMedium, result.SecurityLevel)
	assert.NotEmpty(t, result.RequirementID)
	assert.NotEmpty(t, result.ChallengeToken)
}

func TestEvaluateRequirement_CriticalSecurityForLargeAmount(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	service := NewService(repo, config, nil)

	// Mock no existing verifications
	repo.On("GetLatestVerification", mock.Anything, "user123", "org123", mock.Anything).
		Return(nil, assert.AnError)
	repo.On("GetRememberedDevice", mock.Anything, "user123", "org123", "").
		Return(nil, assert.AnError)
	repo.On("ListPolicies", mock.Anything, "org123").
		Return([]*StepUpPolicy{}, nil)
	repo.On("CreateRequirement", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.Anything).
		Return(nil)

	// Execute
	evalCtx := &EvaluationContext{
		UserID:   "user123",
		OrgID:    "org123",
		Amount:   15000,
		Currency: "USD",
	}

	result, err := service.EvaluateRequirement(context.Background(), evalCtx)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Required)
	assert.Equal(t, SecurityLevelCritical, result.SecurityLevel)
}

func TestEvaluateRequirement_RouteMatching(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	service := NewService(repo, config, nil)

	// Mock no existing verifications
	repo.On("GetLatestVerification", mock.Anything, "user123", "org123", mock.Anything).
		Return(nil, assert.AnError)
	repo.On("GetRememberedDevice", mock.Anything, "user123", "org123", "").
		Return(nil, assert.AnError)
	repo.On("ListPolicies", mock.Anything, "org123").
		Return([]*StepUpPolicy{}, nil)
	repo.On("CreateRequirement", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.Anything).
		Return(nil)

	// Execute
	evalCtx := &EvaluationContext{
		UserID: "user123",
		OrgID:  "org123",
		Route:  "/api/user/email",
		Method: "PUT",
	}

	result, err := service.EvaluateRequirement(context.Background(), evalCtx)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Required)
	assert.Equal(t, SecurityLevelMedium, result.SecurityLevel)
	assert.Contains(t, result.Reason, "email")
}

func TestEvaluateRequirement_RememberedDevice(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	service := NewService(repo, config, nil)

	// Mock remembered device
	rememberedDevice := &StepUpRememberedDevice{
		ID:            "device_123",
		UserID:        "user123",
		OrgID:         "org123",
		DeviceID:      "device_xyz",
		SecurityLevel: SecurityLevelHigh,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	repo.On("GetRememberedDevice", mock.Anything, "user123", "org123", "device_xyz").
		Return(rememberedDevice, nil)
	repo.On("UpdateRememberedDevice", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("GetLatestVerification", mock.Anything, "user123", "org123", mock.Anything).
		Return(nil, assert.AnError)
	repo.On("ListPolicies", mock.Anything, "org123").
		Return([]*StepUpPolicy{}, nil)

	// Execute
	evalCtx := &EvaluationContext{
		UserID:   "user123",
		OrgID:    "org123",
		Amount:   5000,
		Currency: "USD",
		DeviceID: "device_xyz",
	}

	result, err := service.EvaluateRequirement(context.Background(), evalCtx)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Required)
	assert.Equal(t, "Device is remembered", result.Reason)
}

func TestEvaluateRequirement_RiskBasedEvaluation(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	config.RiskBasedEnabled = true
	service := NewService(repo, config, nil)

	// Mock no existing verifications
	repo.On("GetLatestVerification", mock.Anything, "user123", "org123", mock.Anything).
		Return(nil, assert.AnError)
	repo.On("GetRememberedDevice", mock.Anything, "user123", "org123", "").
		Return(nil, assert.AnError)
	repo.On("ListPolicies", mock.Anything, "org123").
		Return([]*StepUpPolicy{}, nil)
	repo.On("CreateRequirement", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.Anything).
		Return(nil)

	// Execute - high risk score should trigger high security
	evalCtx := &EvaluationContext{
		UserID:    "user123",
		OrgID:     "org123",
		RiskScore: 0.75, // Between medium and high threshold
	}

	result, err := service.EvaluateRequirement(context.Background(), evalCtx)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Required)
	assert.Equal(t, SecurityLevelHigh, result.SecurityLevel)
	assert.Contains(t, result.Reason, "Risk-based")
}

func TestVerifyStepUp_Success(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	service := NewService(repo, config, nil)

	// Mock requirement
	requirement := &StepUpRequirement{
		ID:             "req_123",
		UserID:         "user123",
		OrgID:          "org123",
		RequiredLevel:  SecurityLevelMedium,
		CurrentLevel:   SecurityLevelLow,
		Status:         "pending",
		ChallengeToken: "token_xyz",
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}

	repo.On("GetRequirementByToken", mock.Anything, "token_xyz").
		Return(requirement, nil)
	repo.On("CreateVerification", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("UpdateRequirement", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateAttempt", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.Anything).
		Return(nil)

	// Execute
	verifyReq := &VerifyRequest{
		ChallengeToken: "token_xyz",
		Method:         MethodPassword,
		Credential:     "password123",
		RememberDevice: false,
	}

	response, err := service.VerifyStepUp(context.Background(), verifyReq)

	// Assert
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotEmpty(t, response.VerificationID)
	assert.Equal(t, SecurityLevelMedium, response.SecurityLevel)
}

func TestVerifyStepUp_RememberDevice(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	service := NewService(repo, config, nil)

	// Mock requirement
	requirement := &StepUpRequirement{
		ID:             "req_123",
		UserID:         "user123",
		OrgID:          "org123",
		RequiredLevel:  SecurityLevelMedium,
		Status:         "pending",
		ChallengeToken: "token_xyz",
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}

	repo.On("GetRequirementByToken", mock.Anything, "token_xyz").
		Return(requirement, nil)
	repo.On("CreateVerification", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("UpdateRequirement", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateAttempt", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateRememberedDevice", mock.Anything, mock.Anything).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.Anything).
		Return(nil)

	// Execute
	verifyReq := &VerifyRequest{
		ChallengeToken: "token_xyz",
		Method:         MethodPassword,
		Credential:     "password123",
		RememberDevice: true,
		DeviceID:       "device_xyz",
		DeviceName:     "Chrome on MacBook",
	}

	response, err := service.VerifyStepUp(context.Background(), verifyReq)

	// Assert
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.True(t, response.DeviceRemembered)
}

func TestVerifyStepUp_ExpiredRequirement(t *testing.T) {
	// Setup
	repo := new(mockRepository)
	config := DefaultConfig()
	service := NewService(repo, config, nil)

	// Mock expired requirement
	requirement := &StepUpRequirement{
		ID:             "req_123",
		UserID:         "user123",
		OrgID:          "org123",
		Status:         "pending",
		ChallengeToken: "token_xyz",
		ExpiresAt:      time.Now().Add(-1 * time.Minute), // Expired
	}

	repo.On("GetRequirementByToken", mock.Anything, "token_xyz").
		Return(requirement, nil)
	repo.On("UpdateRequirement", mock.Anything, mock.Anything).
		Return(nil)

	// Execute
	verifyReq := &VerifyRequest{
		ChallengeToken: "token_xyz",
		Method:         MethodPassword,
		Credential:     "password123",
	}

	response, err := service.VerifyStepUp(context.Background(), verifyReq)

	// Assert
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "expired")
}

func TestMatchesRoute(t *testing.T) {
	service := &Service{config: DefaultConfig()}

	tests := []struct {
		pattern string
		route   string
		matches bool
	}{
		{"/api/user/email", "/api/user/email", true},
		{"/api/user/email", "/api/user/password", false},
		{"/api/payment/*", "/api/payment/transfer", true},
		{"/api/payment/*", "/api/payment/history", true},
		{"/api/payment/*", "/api/user/transfer", false},
		{"/api/admin/*", "/api/admin/users/delete", true},
	}

	for _, tt := range tests {
		result := service.matchesRoute(tt.pattern, tt.route)
		assert.Equal(t, tt.matches, result, "Pattern: %s, Route: %s", tt.pattern, tt.route)
	}
}

func TestIsHigherLevel(t *testing.T) {
	service := &Service{}

	tests := []struct {
		level1 SecurityLevel
		level2 SecurityLevel
		higher bool
	}{
		{SecurityLevelMedium, SecurityLevelLow, true},
		{SecurityLevelHigh, SecurityLevelMedium, true},
		{SecurityLevelCritical, SecurityLevelHigh, true},
		{SecurityLevelLow, SecurityLevelMedium, false},
		{SecurityLevelMedium, SecurityLevelMedium, false},
		{SecurityLevelNone, SecurityLevelLow, false},
	}

	for _, tt := range tests {
		result := service.isHigherLevel(tt.level1, tt.level2)
		assert.Equal(t, tt.higher, result, "Level1: %s, Level2: %s", tt.level1, tt.level2)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test default config
	config := DefaultConfig()
	err := config.Validate()
	require.NoError(t, err)

	// Test config with zero values
	config2 := &Config{
		Enabled: true,
	}
	err = config2.Validate()
	require.NoError(t, err)
	assert.NotZero(t, config2.MediumAuthWindow)
	assert.NotZero(t, config2.HighAuthWindow)
	assert.NotZero(t, config2.RememberDuration)
}
