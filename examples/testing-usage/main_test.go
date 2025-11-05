package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/schema"
	authsometesting "github.com/xraph/authsome/testing"
)

// Example application code that depends on AuthSome
// These would be your actual application handlers/services

// UserProfileService is an example service in your application
type UserProfileService struct {
	// In a real app, you might have dependencies here
}

// GetProfile retrieves the user's profile information
func (s *UserProfileService) GetProfile(ctx context.Context) (map[string]interface{}, error) {
	user, ok := authsometesting.GetLoggedInUser(ctx)
	if !ok {
		return nil, authsometesting.ErrNotAuthenticated
	}

	org, ok := authsometesting.GetCurrentOrg(ctx)
	if !ok {
		return nil, authsometesting.ErrOrgNotFound
	}

	return map[string]interface{}{
		"user_id":        user.ID.String(),
		"user_name":      user.Name,
		"user_email":     user.Email,
		"email_verified": user.EmailVerified,
		"org_id":         org.ID.String(),
		"org_name":       org.Name,
		"org_slug":       org.Slug,
	}, nil
}

// UpdateProfile updates user profile information
func (s *UserProfileService) UpdateProfile(ctx context.Context, name string) error {
	user, ok := authsometesting.GetLoggedInUser(ctx)
	if !ok {
		return authsometesting.ErrNotAuthenticated
	}

	if !user.EmailVerified {
		return fmt.Errorf("email must be verified to update profile")
	}

	// In real app, you would update the database here
	user.Name = name
	return nil
}

// AdminService is an example service with admin-only operations
type AdminService struct{}

// DeleteUser is an admin-only operation
func (a *AdminService) DeleteUser(ctx context.Context, mock *authsometesting.Mock, targetUserID string) error {
	// Check authentication
	_, err := mock.RequireAuth(ctx)
	if err != nil {
		return err
	}

	// Check admin role
	orgID, ok := authsometesting.GetCurrentOrgID(ctx)
	if !ok {
		return authsometesting.ErrOrgNotFound
	}

	_, err = mock.RequireOrgRole(ctx, orgID, "admin")
	if err != nil {
		return err
	}

	// Perform deletion (mock for testing)
	return nil
}

// OrganizationService manages multi-tenant operations
type OrganizationService struct{}

// ListUserOrganizations returns all organizations the user belongs to
func (o *OrganizationService) ListUserOrganizations(ctx context.Context, mock *authsometesting.Mock) ([]*schema.Organization, error) {
	user, ok := authsometesting.GetLoggedInUser(ctx)
	if !ok {
		return nil, authsometesting.ErrNotAuthenticated
	}

	return mock.GetUserOrgs(user.ID.String())
}

// SwitchOrganization changes the active organization context
func (o *OrganizationService) SwitchOrganization(ctx context.Context, mock *authsometesting.Mock, orgID string) (context.Context, error) {
	user, ok := authsometesting.GetLoggedInUser(ctx)
	if !ok {
		return ctx, authsometesting.ErrNotAuthenticated
	}

	// Verify user is a member
	_, err := mock.RequireOrgMember(ctx, orgID)
	if err != nil {
		return ctx, err
	}

	// Create new session for the organization
	session := mock.CreateSession(user.ID.String(), orgID)

	// Update context
	ctx = mock.WithSession(ctx, session.ID.String())
	ctx = mock.WithOrg(ctx, orgID)

	return ctx, nil
}

// Test Suite: User Profile Service

func TestUserProfileService_GetProfile(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	service := &UserProfileService{}

	t.Run("authenticated user can get profile", func(t *testing.T) {
		// Use common scenario
		scenarios := mock.NewCommonScenarios()
		scenario := scenarios.AuthenticatedUser()

		profile, err := service.GetProfile(scenario.Context)
		require.NoError(t, err)
		assert.Equal(t, scenario.User.ID.String(), profile["user_id"])
		assert.Equal(t, scenario.User.Name, profile["user_name"])
		assert.Equal(t, scenario.Org.ID.String(), profile["org_id"])
	})

	t.Run("unauthenticated user cannot get profile", func(t *testing.T) {
		ctx := context.Background()
		_, err := service.GetProfile(ctx)
		assert.Equal(t, authsometesting.ErrNotAuthenticated, err)
	})

	t.Run("profile includes organization info", func(t *testing.T) {
		user := mock.CreateUser("test@example.com", "Test User")
		org := mock.CreateOrganization("Custom Org", "custom-org")
		mock.AddUserToOrg(user.ID.String(), org.ID.String(), "member")

		session := mock.CreateSession(user.ID.String(), org.ID.String())
		ctx := context.Background()
		ctx = mock.WithSession(ctx, session.ID.String())
		ctx = mock.WithUser(ctx, user.ID.String())
		ctx = mock.WithOrg(ctx, org.ID.String())

		profile, err := service.GetProfile(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Custom Org", profile["org_name"])
		assert.Equal(t, "custom-org", profile["org_slug"])
	})
}

func TestUserProfileService_UpdateProfile(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	service := &UserProfileService{}

	t.Run("verified user can update profile", func(t *testing.T) {
		scenarios := mock.NewCommonScenarios()
		scenario := scenarios.AuthenticatedUser()

		err := service.UpdateProfile(scenario.Context, "New Name")
		assert.NoError(t, err)
	})

	t.Run("unverified user cannot update profile", func(t *testing.T) {
		scenarios := mock.NewCommonScenarios()
		scenario := scenarios.UnverifiedUser()

		err := service.UpdateProfile(scenario.Context, "New Name")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email must be verified")
	})

	t.Run("unauthenticated user cannot update profile", func(t *testing.T) {
		ctx := context.Background()
		err := service.UpdateProfile(ctx, "New Name")
		assert.Equal(t, authsometesting.ErrNotAuthenticated, err)
	})
}

// Test Suite: Admin Service

func TestAdminService_DeleteUser(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	service := &AdminService{}

	t.Run("admin can delete users", func(t *testing.T) {
		scenarios := mock.NewCommonScenarios()
		adminScenario := scenarios.AdminUser()

		targetUser := mock.CreateUser("target@example.com", "Target User")

		err := service.DeleteUser(adminScenario.Context, mock, targetUser.ID.String())
		assert.NoError(t, err)
	})

	t.Run("regular user cannot delete users", func(t *testing.T) {
		scenarios := mock.NewCommonScenarios()
		userScenario := scenarios.AuthenticatedUser()

		targetUser := mock.CreateUser("target@example.com", "Target User")

		err := service.DeleteUser(userScenario.Context, mock, targetUser.ID.String())
		assert.Equal(t, authsometesting.ErrInsufficientPermissions, err)
	})

	t.Run("unauthenticated user cannot delete users", func(t *testing.T) {
		ctx := context.Background()
		targetUser := mock.CreateUser("target@example.com", "Target User")

		err := service.DeleteUser(ctx, mock, targetUser.ID.String())
		assert.Error(t, err)
	})
}

// Test Suite: Organization Service

func TestOrganizationService_ListUserOrganizations(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	service := &OrganizationService{}

	t.Run("user can list their organizations", func(t *testing.T) {
		scenarios := mock.NewCommonScenarios()
		scenario := scenarios.AuthenticatedUser()

		orgs, err := service.ListUserOrganizations(scenario.Context, mock)
		require.NoError(t, err)
		assert.Len(t, orgs, 1) // Default org
		assert.Equal(t, "Test Organization", orgs[0].Name)
	})

	t.Run("multi-org user sees all organizations", func(t *testing.T) {
		scenarios := mock.NewCommonScenarios()
		scenario := scenarios.MultiOrgUser()

		orgs, err := service.ListUserOrganizations(scenario.Context, mock)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(orgs), 2) // At least default org + second org
	})

	t.Run("unauthenticated user cannot list organizations", func(t *testing.T) {
		ctx := context.Background()
		_, err := service.ListUserOrganizations(ctx, mock)
		assert.Equal(t, authsometesting.ErrNotAuthenticated, err)
	})
}

func TestOrganizationService_SwitchOrganization(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	service := &OrganizationService{}

	t.Run("user can switch to member organization", func(t *testing.T) {
		user := mock.CreateUser("user@example.com", "Test User")
		org1 := mock.GetDefaultOrg()
		org2 := mock.CreateOrganization("Second Org", "second-org")
		mock.AddUserToOrg(user.ID.String(), org2.ID.String(), "member")

		ctx := mock.NewTestContextWithUser(user)

		// Initially in org1
		currentOrgID, _ := authsometesting.GetCurrentOrgID(ctx)
		assert.Equal(t, org1.ID.String(), currentOrgID)

		// Switch to org2
		newCtx, err := service.SwitchOrganization(ctx, mock, org2.ID.String())
		require.NoError(t, err)

		// Verify switched
		newOrgID, ok := authsometesting.GetCurrentOrgID(newCtx)
		require.True(t, ok)
		assert.Equal(t, org2.ID.String(), newOrgID)
	})

	t.Run("user cannot switch to non-member organization", func(t *testing.T) {
		user := mock.CreateUser("user@example.com", "Test User")
		org2 := mock.CreateOrganization("Other Org", "other-org")
		// User is NOT added to org2

		ctx := mock.NewTestContextWithUser(user)

		_, err := service.SwitchOrganization(ctx, mock, org2.ID.String())
		assert.Equal(t, authsometesting.ErrNotOrgMember, err)
	})

	t.Run("unauthenticated user cannot switch organization", func(t *testing.T) {
		ctx := context.Background()
		org := mock.GetDefaultOrg()

		_, err := service.SwitchOrganization(ctx, mock, org.ID.String())
		assert.Equal(t, authsometesting.ErrNotAuthenticated, err)
	})
}

// Test Suite: Complex Scenarios

func TestComplexScenario_UserJourney(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	profileService := &UserProfileService{}
	orgService := &OrganizationService{}

	// Step 1: User signs up (simulated)
	user := mock.CreateUser("newuser@example.com", "New User")
	user.EmailVerified = false
	ctx := mock.NewTestContextWithUser(user)

	// Step 2: Try to update profile (should fail - email not verified)
	err := profileService.UpdateProfile(ctx, "Updated Name")
	assert.Error(t, err)

	// Step 3: Verify email (simulated)
	user.EmailVerified = true

	// Step 4: Now can update profile
	err = profileService.UpdateProfile(ctx, "Updated Name")
	assert.NoError(t, err)

	// Step 5: Get profile
	profile, err := profileService.GetProfile(ctx)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), profile["user_id"])

	// Step 6: Create and join second organization
	org2 := mock.CreateOrganization("Work Org", "work-org")
	mock.AddUserToOrg(user.ID.String(), org2.ID.String(), "member")

	// Step 7: List organizations
	orgs, err := orgService.ListUserOrganizations(ctx, mock)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)

	// Step 8: Switch to work org
	workCtx, err := orgService.SwitchOrganization(ctx, mock, org2.ID.String())
	require.NoError(t, err)

	// Step 9: Verify current org changed
	currentOrg, ok := authsometesting.GetCurrentOrg(workCtx)
	require.True(t, ok)
	assert.Equal(t, "Work Org", currentOrg.Name)
}

func TestComplexScenario_AdminManagement(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	adminService := &AdminService{}
	profileService := &UserProfileService{}

	// Create admin and regular user
	admin := mock.CreateUserWithRole("admin@example.com", "Admin", "admin")
	user := mock.CreateUserWithRole("user@example.com", "Regular User", "member")

	adminCtx := mock.NewTestContextWithUser(admin)
	userCtx := mock.NewTestContextWithUser(user)

	// Admin can view any user's profile
	profile, err := profileService.GetProfile(adminCtx)
	require.NoError(t, err)
	assert.Equal(t, admin.ID.String(), profile["user_id"])

	// Admin can delete users
	err = adminService.DeleteUser(adminCtx, mock, user.ID.String())
	assert.NoError(t, err)

	// Regular user cannot delete users
	targetUser := mock.CreateUser("target@example.com", "Target")
	err = adminService.DeleteUser(userCtx, mock, targetUser.ID.String())
	assert.Equal(t, authsometesting.ErrInsufficientPermissions, err)
}

// Benchmark: Test performance of mock operations

func BenchmarkMock_CreateUser(b *testing.B) {
	mock := authsometesting.NewMock(&testing.T{})
	defer mock.Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.CreateUser(fmt.Sprintf("user%d@example.com", i), "Test User")
	}
}

func BenchmarkMock_GetLoggedInUser(b *testing.B) {
	mock := authsometesting.NewMock(&testing.T{})
	defer mock.Reset()

	ctx := mock.NewTestContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authsometesting.GetLoggedInUser(ctx)
	}
}

func BenchmarkMock_RequireAuth(b *testing.B) {
	mock := authsometesting.NewMock(&testing.T{})
	defer mock.Reset()

	ctx := mock.NewTestContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.RequireAuth(ctx)
	}
}
