package testing_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authsometesting "github.com/xraph/authsome/testing"
)

// Example: Basic user creation and authentication
func TestExample_BasicAuth(t *testing.T) {
	// Create a new mock instance
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Create a test user
	user := mock.CreateUser("test@example.com", "Test User")
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.True(t, user.EmailVerified)

	// Create a session
	session := mock.CreateSession(user.ID.String(), mock.GetDefaultOrg().ID.String())
	assert.NotNil(t, session)
	assert.Equal(t, user.ID.String(), session.UserID.String())

	// Create authenticated context
	ctx := mock.WithSession(context.Background(), session.ID.String())
	ctx = mock.WithUser(ctx, user.ID.String())

	// Verify user can be retrieved from context
	retrievedUser, ok := authsometesting.GetLoggedInUser(ctx)
	require.True(t, ok)
	assert.Equal(t, user.ID.String(), retrievedUser.ID.String())
}

// Example: Using NewTestContext convenience method
func TestExample_QuickAuth(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Quick way to get a fully authenticated context
	ctx := mock.NewTestContext()

	// Get user from context
	user, ok := authsometesting.GetLoggedInUser(ctx)
	require.True(t, ok)
	assert.NotNil(t, user)

	// Get org from context
	org, ok := authsometesting.GetCurrentOrg(ctx)
	require.True(t, ok)
	assert.NotNil(t, org)

	// Get session from context
	session, ok := authsometesting.GetCurrentSession(ctx)
	require.True(t, ok)
	assert.NotNil(t, session)
}

// Example: Testing with multiple organizations
func TestExample_MultiOrg(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Create user and multiple organizations
	user := mock.CreateUser("user@example.com", "Test User")
	org1 := mock.GetDefaultOrg()
	org2 := mock.CreateOrganization("Second Org", "second-org")

	// Add user to second org with admin role
	mock.AddUserToOrg(user.ID.String(), org2.ID.String(), "admin")

	// Verify user is in both orgs
	orgs, err := mock.GetUserOrgs(user.ID.String())
	require.NoError(t, err)
	assert.Len(t, orgs, 2)

	// Create session for first org
	session1 := mock.CreateSession(user.ID.String(), org1.ID.String())
	ctx1 := mock.WithSession(context.Background(), session1.ID.String())
	ctx1 = mock.WithOrg(ctx1, org1.ID.String())

	orgID1, ok := authsometesting.GetCurrentOrgID(ctx1)
	require.True(t, ok)
	assert.Equal(t, org1.ID.String(), orgID1)

	// Create session for second org
	session2 := mock.CreateSession(user.ID.String(), org2.ID.String())
	ctx2 := mock.WithSession(context.Background(), session2.ID.String())
	ctx2 = mock.WithOrg(ctx2, org2.ID.String())

	orgID2, ok := authsometesting.GetCurrentOrgID(ctx2)
	require.True(t, ok)
	assert.Equal(t, org2.ID.String(), orgID2)
}

// Example: Testing authorization with roles
func TestExample_RoleBasedAuth(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Create users with different roles
	adminUser := mock.CreateUserWithRole("admin@example.com", "Admin User", "admin")
	memberUser := mock.CreateUserWithRole("member@example.com", "Member User", "member")

	org := mock.GetDefaultOrg()

	// Test admin access
	adminCtx := mock.NewTestContextWithUser(adminUser)

	member, err := mock.RequireOrgRole(adminCtx, org.ID.String(), "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", member.Role)

	// Test member access (should fail for admin role)
	memberCtx := mock.NewTestContextWithUser(memberUser)

	_, err = mock.RequireOrgRole(memberCtx, org.ID.String(), "admin")
	assert.Error(t, err)
	assert.Equal(t, authsometesting.ErrInsufficientPermissions, err)
}

// Example: Testing session expiration
func TestExample_SessionExpiration(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	user := mock.CreateUser("user@example.com", "Test User")

	// Create an expired session
	expiredSession := mock.CreateExpiredSession(user.ID.String(), mock.GetDefaultOrg().ID.String())

	// Try to validate the expired session
	_, err := mock.SessionService.Validate(context.Background(), expiredSession.Token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// Example: Using common scenarios
func TestExample_CommonScenarios(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	scenarios := mock.NewCommonScenarios()

	t.Run("authenticated user", func(t *testing.T) {
		scenario := scenarios.AuthenticatedUser()
		assert.NotNil(t, scenario.User)
		assert.NotNil(t, scenario.Session)
		assert.True(t, scenario.User.EmailVerified)
	})

	t.Run("admin user", func(t *testing.T) {
		scenario := scenarios.AdminUser()
		assert.NotNil(t, scenario.User)
		assert.Equal(t, "admin@example.com", scenario.User.Email)
	})

	t.Run("unverified user", func(t *testing.T) {
		scenario := scenarios.UnverifiedUser()
		assert.NotNil(t, scenario.User)
		assert.False(t, scenario.User.EmailVerified)
	})

	t.Run("multi-org user", func(t *testing.T) {
		scenario := scenarios.MultiOrgUser()
		orgs, err := mock.GetUserOrgs(scenario.User.ID.String())
		require.NoError(t, err)
		assert.Len(t, orgs, 2)
	})

	t.Run("expired session", func(t *testing.T) {
		scenario := scenarios.ExpiredSession()
		_, err := mock.SessionService.Validate(context.Background(), scenario.Session.Token)
		assert.Error(t, err)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		scenario := scenarios.UnauthenticatedUser()
		assert.Nil(t, scenario.User)
		assert.Nil(t, scenario.Session)
		_, ok := authsometesting.GetLoggedInUser(scenario.Context)
		assert.False(t, ok)
	})

	t.Run("inactive user", func(t *testing.T) {
		scenario := scenarios.InactiveUser()
		assert.NotNil(t, scenario.User.DeletedAt)
	})
}

// Example: Testing service methods
func TestExample_ServiceMethods(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	ctx := context.Background()

	t.Run("user service", func(t *testing.T) {
		// Test user creation
		user, err := mock.UserService.GetByEmail(ctx, "test@example.com")
		assert.Error(t, err) // Should not exist yet

		// Create user
		created := mock.CreateUser("test@example.com", "Test User")
		assert.NotNil(t, created)

		// Get by email
		user, err = mock.UserService.GetByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, created.ID.String(), user.ID.String())

		// Get by ID
		user, err = mock.UserService.GetByID(ctx, created.ID.String())
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("session service", func(t *testing.T) {
		user := mock.CreateUser("session@example.com", "Session User")
		session := mock.CreateSession(user.ID.String(), mock.GetDefaultOrg().ID.String())

		// Get by ID
		retrieved, err := mock.SessionService.GetByID(ctx, session.ID.String())
		require.NoError(t, err)
		assert.Equal(t, session.ID.String(), retrieved.ID.String())

		// Get by token
		retrieved, err = mock.SessionService.GetByToken(ctx, session.Token)
		require.NoError(t, err)
		assert.Equal(t, session.ID.String(), retrieved.ID.String())

		// Validate
		validated, err := mock.SessionService.Validate(ctx, session.Token)
		require.NoError(t, err)
		assert.Equal(t, session.ID.String(), validated.ID.String())

		// Delete
		err = mock.SessionService.Delete(ctx, session.ID.String())
		require.NoError(t, err)

		_, err = mock.SessionService.GetByID(ctx, session.ID.String())
		assert.Error(t, err)
	})

	t.Run("organization service", func(t *testing.T) {
		// Create org
		org := mock.CreateOrganization("Test Org", "test-org-2")

		// Get by ID
		retrieved, err := mock.OrganizationService.GetByID(ctx, org.ID.String())
		require.NoError(t, err)
		assert.Equal(t, "Test Org", retrieved.Name)

		// Get by slug
		retrieved, err = mock.OrganizationService.GetBySlug(ctx, "test-org-2")
		require.NoError(t, err)
		assert.Equal(t, org.ID.String(), retrieved.ID.String())

		// Add member
		user := mock.CreateUser("member@example.com", "Member User")
		member := mock.AddUserToOrg(user.ID.String(), org.ID.String(), "member")
		assert.NotNil(t, member)

		// Get members
		members, err := mock.OrganizationService.GetMembers(ctx, org.ID.String())
		require.NoError(t, err)
		assert.Len(t, members, 1)

		// Get user organizations
		orgs, err := mock.OrganizationService.GetUserOrganizations(ctx, user.ID.String())
		require.NoError(t, err)
		// User should be in default org and the new org
		assert.GreaterOrEqual(t, len(orgs), 2)
	})
}

// Example: Real-world scenario - testing a handler that requires auth
func TestExample_HandlerWithAuth(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Your application handler
	getUserProfile := func(ctx context.Context) (map[string]interface{}, error) {
		user, ok := authsometesting.GetLoggedInUser(ctx)
		if !ok {
			return nil, authsometesting.ErrNotAuthenticated
		}

		org, ok := authsometesting.GetCurrentOrg(ctx)
		if !ok {
			return nil, authsometesting.ErrOrgNotFound
		}

		return map[string]interface{}{
			"user_id":   user.ID.String(),
			"user_name": user.Name,
			"org_id":    org.ID.String(),
			"org_name":  org.Name,
		}, nil
	}

	t.Run("authenticated request", func(t *testing.T) {
		// Set up authenticated context
		user := mock.CreateUser("user@example.com", "Test User")
		ctx := mock.NewTestContextWithUser(user)

		// Call handler
		profile, err := getUserProfile(ctx)
		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), profile["user_id"])
		assert.Equal(t, "Test User", profile["user_name"])
	})

	t.Run("unauthenticated request", func(t *testing.T) {
		// No authentication
		ctx := context.Background()

		// Should fail
		_, err := getUserProfile(ctx)
		assert.Error(t, err)
		assert.Equal(t, authsometesting.ErrNotAuthenticated, err)
	})
}

// Example: Testing with custom metadata
func TestExample_Metadata(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Create organization with metadata
	org := mock.CreateOrganization("Test Org", "test-org-meta")
	org.Metadata = map[string]interface{}{
		"industry": "Technology",
		"size":     "Enterprise",
		"region":   "US-West",
	}

	// Verify metadata
	assert.Equal(t, "Technology", org.Metadata["industry"])
	assert.Equal(t, "Enterprise", org.Metadata["size"])
	assert.Equal(t, "US-West", org.Metadata["region"])
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
