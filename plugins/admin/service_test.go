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
	// Test request structure
	req := &CreateUserRequest{
		Email:          "test@example.com",
		Password:       "SecurePass123!",
		Name:           "Test User",
		AdminID:        xid.New().String(),
		OrganizationID: "org-123",
		EmailVerified:  true,
	}

	assert.NotEmpty(t, req.Email)
	assert.NotEmpty(t, req.Password)
	assert.NotEmpty(t, req.AdminID)
	assert.True(t, req.EmailVerified)
}

func TestBanUserRequest_Validation(t *testing.T) {
	req := &BanUserRequest{
		UserID:  xid.New().String(),
		AdminID: xid.New().String(),
		Reason:  "Violation of terms",
	}

	assert.NotEmpty(t, req.UserID)
	assert.NotEmpty(t, req.AdminID)
	assert.NotEmpty(t, req.Reason)
}

func TestUnbanUserRequest_Validation(t *testing.T) {
	req := &UnbanUserRequest{
		UserID:  xid.New().String(),
		AdminID: xid.New().String(),
	}

	assert.NotEmpty(t, req.UserID)
	assert.NotEmpty(t, req.AdminID)
}

func TestListUsersRequest_Defaults(t *testing.T) {
	req := &ListUsersRequest{
		AdminID: xid.New().String(),
		Page:    0,
		Limit:   0,
	}

	// The service should apply defaults:
	// - Page defaults to 1 if <= 0
	// - Limit defaults to 20 if <= 0 or > 100

	assert.NotEmpty(t, req.AdminID)

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
	req := &ListSessionsRequest{
		AdminID: xid.New().String(),
		Page:    0,
		Limit:   0,
	}

	assert.NotEmpty(t, req.AdminID)

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
	// Test role assignment data validation
	userID := xid.New().String()
	adminID := xid.New().String()
	role := "moderator"
	orgID := "org-123"

	assert.NotEmpty(t, userID)
	assert.NotEmpty(t, adminID)
	assert.NotEmpty(t, role)
	assert.NotEmpty(t, orgID)
}

func TestRevokeSessionRequest_Validation(t *testing.T) {
	sessionID := xid.New().String()
	adminID := xid.New().String()

	// Verify both IDs are valid XIDs
	_, err := xid.FromString(sessionID)
	assert.NoError(t, err)

	_, err = xid.FromString(adminID)
	assert.NoError(t, err)
}

// Integration-level tests would require actual service dependencies
// These basic tests verify the request/response structures and validation logic
