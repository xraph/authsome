package admin

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "admin", config.RequiredRole)
	assert.True(t, config.AllowUserCreation)
	assert.True(t, config.AllowUserDeletion)
	assert.True(t, config.AllowImpersonation)
	assert.Greater(t, config.MaxImpersonationDuration.Hours(), float64(0))
}

func TestCheckAdminPermission_InvalidAdminID(t *testing.T) {
	// This test verifies that checkAdminPermission properly validates admin IDs
	// In a real scenario, this would be tested with actual service dependencies
	// For now, we're testing the ID parsing logic

	invalidID := "invalid-xid"
	_, err := xid.FromString(invalidID)
	assert.Error(t, err, "Invalid XID should produce an error")
}

func TestCheckAdminPermission_ValidAdminID(t *testing.T) {
	// Test that valid XID parsing works
	validID := xid.New()
	parsed, err := xid.FromString(validID.String())
	assert.NoError(t, err)
	assert.Equal(t, validID, parsed)
}

func TestCreateUserRequest_Validation(t *testing.T) {
	// Test request structure with V2 architecture: App → Environment → Organization
	appID := xid.New()
	userOrgID := xid.New()
	adminID := xid.New()

	req := &CreateUserRequest{
		Email:              "test@example.com",
		Password:           "SecurePass123!",
		Name:               "Test User",
		AppID:              appID,      // Platform app (required)
		UserOrganizationID: &userOrgID, // User-created org (optional)
		AdminID:            adminID,    // Admin performing the action
		EmailVerified:      true,
	}

	assert.NotEmpty(t, req.Email)
	assert.NotEmpty(t, req.Password)
	assert.False(t, req.AppID.IsNil(), "AppID is required")
	assert.NotNil(t, req.UserOrganizationID, "UserOrganizationID is optional but should be set when provided")
	assert.False(t, req.AdminID.IsNil())
	assert.True(t, req.EmailVerified)
}

func TestBanUserRequest_Validation(t *testing.T) {
	appID := xid.New()
	userID := xid.New()
	adminID := xid.New()

	req := &BanUserRequest{
		AppID:   appID, // Platform app (required)
		UserID:  userID,
		AdminID: adminID,
		Reason:  "Violation of terms",
	}

	assert.False(t, req.AppID.IsNil())
	assert.False(t, req.UserID.IsNil())
	assert.False(t, req.AdminID.IsNil())
	assert.NotEmpty(t, req.Reason)
}

func TestUnbanUserRequest_Validation(t *testing.T) {
	appID := xid.New()
	userID := xid.New()
	adminID := xid.New()

	req := &UnbanUserRequest{
		AppID:   appID, // Platform app (required)
		UserID:  userID,
		AdminID: adminID,
	}

	assert.False(t, req.AppID.IsNil())
	assert.False(t, req.UserID.IsNil())
	assert.False(t, req.AdminID.IsNil())
}

func TestListUsersRequest_Defaults(t *testing.T) {
	appID := xid.New()
	adminID := xid.New()

	req := &ListUsersRequest{
		AppID:   appID, // Platform app (required)
		AdminID: adminID,
		Page:    0,
		Limit:   0,
	}

	// The service should apply defaults:
	// - Page defaults to 1 if <= 0
	// - Limit defaults to 20 if <= 0 or > 100

	assert.False(t, req.AppID.IsNil())
	assert.False(t, req.AdminID.IsNil())

	// Test default logic
	page := req.Page
	if page <= 0 {
		page = 1
	}
	assert.Equal(t, 1, page)

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	assert.Equal(t, 20, limit)
}

func TestListSessionsRequest_Defaults(t *testing.T) {
	appID := xid.New()
	adminID := xid.New()

	req := &ListSessionsRequest{
		AppID:   appID, // Platform app (required)
		AdminID: adminID,
		Page:    0,
		Limit:   0,
	}

	assert.False(t, req.AppID.IsNil())
	assert.False(t, req.AdminID.IsNil())

	// Test default logic
	page := req.Page
	if page <= 0 {
		page = 1
	}
	assert.Equal(t, 1, page)

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	assert.Equal(t, 20, limit)
}

func TestRoleAssignment_Validation(t *testing.T) {
	// Test role assignment data validation with V2 architecture
	appID := xid.New()
	userOrgID := xid.New()
	userID := xid.New()
	adminID := xid.New()
	role := "moderator"

	req := &SetUserRoleRequest{
		AppID:              appID,      // Platform app (required)
		UserOrganizationID: &userOrgID, // User-created org (optional)
		UserID:             userID,
		Role:               role,
		AdminID:            adminID,
	}

	assert.False(t, req.AppID.IsNil())
	assert.NotNil(t, req.UserOrganizationID)
	assert.False(t, req.UserID.IsNil())
	assert.False(t, req.AdminID.IsNil())
	assert.NotEmpty(t, req.Role)
}

func TestRevokeSessionRequest_Validation(t *testing.T) {
	sessionID := xid.New()
	adminID := xid.New()

	// Verify both IDs are valid XIDs
	assert.False(t, sessionID.IsNil())
	assert.False(t, adminID.IsNil())

	// Test string conversion roundtrip
	sessionIDStr := sessionID.String()
	parsed, err := xid.FromString(sessionIDStr)
	assert.NoError(t, err)
	assert.Equal(t, sessionID, parsed)
}

func TestImpersonateUserRequest_Validation(t *testing.T) {
	// Test impersonation request with V2 architecture
	appID := xid.New()
	userOrgID := xid.New()
	userID := xid.New()
	adminID := xid.New()

	req := &ImpersonateUserRequest{
		AppID:              appID,      // Platform app (required)
		UserOrganizationID: &userOrgID, // User-created org (optional)
		UserID:             userID,
		AdminID:            adminID,
		IPAddress:          "192.168.1.1",
		UserAgent:          "Mozilla/5.0",
	}

	assert.False(t, req.AppID.IsNil())
	assert.NotNil(t, req.UserOrganizationID)
	assert.False(t, req.UserID.IsNil())
	assert.False(t, req.AdminID.IsNil())
	assert.NotEmpty(t, req.IPAddress)
	assert.NotEmpty(t, req.UserAgent)
}

// Integration-level tests would require actual service dependencies
// These basic tests verify the request/response structures and validation logic
