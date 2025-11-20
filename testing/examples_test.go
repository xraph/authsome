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
	session := mock.CreateSession(user.ID, mock.GetDefaultOrg().ID)
	assert.NotNil(t, session)
	assert.Equal(t, user.ID, session.UserID)

	// Create authenticated context
	ctx := mock.WithSession(context.Background(), session.ID)
	ctx = mock.WithUser(ctx, user.ID)

	// Verify user can be retrieved from context
	userID, ok := authsometesting.GetUserID(ctx)
	require.True(t, ok)
	assert.Equal(t, user.ID, userID)

	// Verify we can get the full user from context
	retrievedUser, err := mock.GetUserFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)
}

// Example: Using NewTestContext convenience method
func TestExample_QuickAuth(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Quick way to get a fully authenticated context
	ctx := mock.NewTestContext()

	// Get user ID from context
	userID, ok := authsometesting.GetUserID(ctx)
	require.True(t, ok)
	assert.False(t, userID.IsNil())

	// Get org ID from context
	orgID, ok := authsometesting.GetOrganizationID(ctx)
	require.True(t, ok)
	assert.False(t, orgID.IsNil())

	// Get session from context
	session, ok := authsometesting.GetSession(ctx)
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
	mock.AddUserToOrg(user.ID, org2.ID, "admin")

	// Verify user is in both orgs
	orgs, err := mock.GetUserOrgs(user.ID)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)

	// Create session for first org
	session1 := mock.CreateSession(user.ID, org1.ID)
	ctx1 := mock.WithSession(context.Background(), session1.ID)
	ctx1 = mock.WithOrganization(ctx1, org1.ID)

	orgID1, ok := authsometesting.GetOrganizationID(ctx1)
	require.True(t, ok)
	assert.Equal(t, org1.ID, orgID1)

	// Create session for second org
	session2 := mock.CreateSession(user.ID, org2.ID)
	ctx2 := mock.WithSession(context.Background(), session2.ID)
	ctx2 = mock.WithOrganization(ctx2, org2.ID)

	orgID2, ok := authsometesting.GetOrganizationID(ctx2)
	require.True(t, ok)
	assert.Equal(t, org2.ID, orgID2)
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

	member, err := mock.RequireOrgRole(adminCtx, org.ID, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", member.Role)

	// Test member access (should fail for admin role)
	memberCtx := mock.NewTestContextWithUser(memberUser)

	_, err = mock.RequireOrgRole(memberCtx, org.ID, "admin")
	assert.Error(t, err)
	assert.Equal(t, authsometesting.ErrInsufficientPermissions, err)
}

// Example: Testing session expiration
func TestExample_SessionExpiration(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	user := mock.CreateUser("user@example.com", "Test User")

	// Create an expired session
	expiredSession := mock.CreateExpiredSession(user.ID, mock.GetDefaultOrg().ID)

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
		orgs, err := mock.GetUserOrgs(scenario.User.ID)
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
		userID, ok := authsometesting.GetUserID(scenario.Context)
		assert.False(t, ok)
		assert.True(t, userID.IsNil())
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
		user, err = mock.UserService.GetByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("session service", func(t *testing.T) {
		user := mock.CreateUser("session@example.com", "Session User")
		session := mock.CreateSession(user.ID, mock.GetDefaultOrg().ID)

		// Get by ID
		retrieved, err := mock.SessionService.GetByID(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrieved.ID)

		// Get by token
		retrieved, err = mock.SessionService.GetByToken(ctx, session.Token)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrieved.ID)

		// Validate
		validated, err := mock.SessionService.Validate(ctx, session.Token)
		require.NoError(t, err)
		assert.Equal(t, session.ID, validated.ID)

		// Delete
		err = mock.SessionService.Delete(ctx, session.ID)
		require.NoError(t, err)

		_, err = mock.SessionService.GetByID(ctx, session.ID)
		assert.Error(t, err)
	})

	t.Run("organization service", func(t *testing.T) {
		// Create org
		org := mock.CreateOrganization("Test Org", "test-org-2")

		// Get by ID
		retrieved, err := mock.OrganizationService.GetByID(ctx, org.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Org", retrieved.Name)

		// Get by slug
		retrieved, err = mock.OrganizationService.GetBySlug(ctx, "test-org-2")
		require.NoError(t, err)
		assert.Equal(t, org.ID, retrieved.ID)

		// Add member
		user := mock.CreateUser("member@example.com", "Member User")
		member := mock.AddUserToOrg(user.ID, org.ID, "member")
		assert.NotNil(t, member)

		// Get members
		membersResp, err := mock.OrganizationService.GetMembers(ctx, org.ID)
		require.NoError(t, err)
		assert.Len(t, membersResp.Data, 1)

		// Get user organizations
		orgs, err := mock.OrganizationService.GetUserOrganizations(ctx, user.ID)
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
	getUserProfile := func(ctx context.Context, mock *authsometesting.Mock) (map[string]interface{}, error) {
		userID, ok := authsometesting.GetUserID(ctx)
		if !ok || userID.IsNil() {
			return nil, authsometesting.ErrNotAuthenticated
		}

		user, err := mock.GetUserFromContext(ctx)
		if err != nil {
			return nil, authsometesting.ErrUserNotFound
		}

		org, err := mock.GetOrganizationFromContext(ctx)
		if err != nil {
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
		profile, err := getUserProfile(ctx, mock)
		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), profile["user_id"])
		assert.Equal(t, "Test User", profile["user_name"])
	})

	t.Run("unauthenticated request", func(t *testing.T) {
		// No authentication
		ctx := context.Background()

		// Should fail
		_, err := getUserProfile(ctx, mock)
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
	t := &testing.T{}
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.CreateUser(fmt.Sprintf("user%d@example.com", i), "Test User")
	}
}

func BenchmarkMock_GetUserID(b *testing.B) {
	t := &testing.T{}
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	ctx := mock.NewTestContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authsometesting.GetUserID(ctx)
	}
}

func BenchmarkMock_RequireAuth(b *testing.B) {
	t := &testing.T{}
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	ctx := mock.NewTestContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.RequireAuth(ctx)
	}
}
