// Package testing provides comprehensive mocking utilities for testing applications
// that integrate with the AuthSome authentication framework.
//
// Version 2.0: Now with full multi-tenancy support, core/contexts integration,
// and Forge HTTP handler testing.
//
// # Overview
//
// This package is designed for external users who need to test their applications
// that depend on AuthSome without setting up a full AuthSome instance with database,
// Redis, and other infrastructure components.
//
// # Quick Start
//
// Import the package and create a mock instance:
//
//	import (
//	    "testing"
//	    authsometesting "github.com/xraph/authsome/testing"
//	)
//
//	func TestMyHandler(t *testing.T) {
//	    mock := authsometesting.NewMock(t)
//	    defer mock.Reset()
//
//	    // Create authenticated context with full tenant hierarchy
//	    ctx := mock.NewTestContext()
//
//	    // Your test code here
//	}
//
// # Core Features
//
//   - Multi-Tenancy: Full App → Environment → Organization hierarchy support
//   - Core Contexts: Uses AuthSome's actual context system from core/contexts
//   - Mock Users: Create test users with various states (verified, unverified, active, inactive)
//   - Mock Sessions: Create active or expired sessions for testing authentication flows
//   - Mock Organizations: Create organizations and manage memberships
//   - Mock Services: UserService, SessionService, OrganizationService with full CRUD operations
//   - Context Helpers: Easily add authentication data to context with typed keys
//   - Common Scenarios: Pre-configured scenarios for typical test cases
//   - Authorization Helpers: Test role-based access control
//   - Forge Mocking: Mock Forge contexts for HTTP handler testing
//   - Builder Pattern: Fluent API for creating complex test data
//   - Thread-Safe: Safe for concurrent use in tests
//
// # Creating Test Data
//
// Create users:
//
//	user := mock.CreateUser("user@example.com", "Test User")
//	adminUser := mock.CreateUserWithRole("admin@example.com", "Admin", "admin")
//
// Create sessions:
//
//	session := mock.CreateSession(user.ID, org.ID)
//	expiredSession := mock.CreateExpiredSession(user.ID, org.ID)
//
// Create organizations:
//
//	org := mock.CreateOrganization("My Org", "my-org")
//	member := mock.AddUserToOrg(user.ID, org.ID, "member")
//
// # Working with Context
//
// Create authenticated contexts:
//
//	// Quick way - creates user, org, and session automatically
//	ctx := mock.NewTestContext()
//
//	// With specific user
//	user := mock.CreateUser("user@example.com", "Test User")
//	ctx := mock.NewTestContextWithUser(user)
//
//	// Manual setup with full control
//	ctx := context.Background()
//	ctx = mock.WithApp(ctx, app.ID)
//	ctx = mock.WithEnvironment(ctx, env.ID)
//	ctx = mock.WithOrganization(ctx, org.ID)
//	ctx = mock.WithSession(ctx, session.ID)
//
// Retrieve from context (using core/contexts):
//
//	userID, ok := authsometesting.GetUserID(ctx)
//	appID, ok := authsometesting.GetAppID(ctx)
//	envID, ok := authsometesting.GetEnvironmentID(ctx)
//	orgID, ok := authsometesting.GetOrganizationID(ctx)
//	session, ok := authsometesting.GetSession(ctx)
//
//	// Get full entities from context
//	user, err := mock.GetUserFromContext(ctx)
//	org, err := mock.GetOrganizationFromContext(ctx)
//
// # Authorization Testing
//
// Test authentication and authorization:
//
//	// Require authentication
//	user, err := mock.RequireAuth(ctx)
//
//	// Require organization membership
//	member, err := mock.RequireOrgMember(ctx, orgID)
//
//	// Require specific role
//	member, err := mock.RequireOrgRole(ctx, orgID, "admin")
//
// # Common Scenarios
//
// Use pre-configured scenarios for typical test cases:
//
//	scenarios := mock.NewCommonScenarios()
//
//	// Various scenarios available:
//	authUser := scenarios.AuthenticatedUser()
//	adminUser := scenarios.AdminUser()
//	unverifiedUser := scenarios.UnverifiedUser()
//	multiOrgUser := scenarios.MultiOrgUser()
//	expiredSession := scenarios.ExpiredSession()
//	unauthenticated := scenarios.UnauthenticatedUser()
//	inactiveUser := scenarios.InactiveUser()
//
//	// Use scenario in tests
//	user := authUser.User
//	ctx := authUser.Context
//
// # Service Methods
//
// Test with mock services that implement the same interfaces as real services:
//
//	// User service
//	user, err := mock.UserService.GetByEmail(ctx, "user@example.com")
//	user, err := mock.UserService.GetByID(ctx, userID)
//	user, err := mock.UserService.Create(ctx, req)
//	user, err := mock.UserService.Update(ctx, userID, req)
//
//	// Session service
//	session, err := mock.SessionService.GetByToken(ctx, token)
//	session, err := mock.SessionService.Validate(ctx, token)
//	err := mock.SessionService.Delete(ctx, sessionID)
//
//	// Organization service
//	org, err := mock.OrganizationService.GetByID(ctx, orgID)
//	org, err := mock.OrganizationService.GetBySlug(ctx, "my-org")
//	membersResp, err := mock.OrganizationService.GetMembers(ctx, orgID)
//	orgs, err := mock.OrganizationService.GetUserOrganizations(ctx, userID)
//
// # Builder Pattern
//
// Create complex test data with fluent API:
//
//	app := mock.NewApp().
//	    WithName("Production App").
//	    WithSlug("prod-app").
//	    Build()
//
//	user := mock.NewUser().
//	    WithEmail("admin@example.com").
//	    WithName("Admin User").
//	    WithRole("admin").
//	    Build()
//
// # HTTP Handler Testing
//
// Test HTTP handlers with mock Forge contexts:
//
//	// Quick authenticated context
//	forgeCtx := mock.QuickAuthenticatedForgeContext("GET", "/api/profile")
//
//	// With specific user
//	user := mock.CreateUser("user@example.com", "Test User")
//	forgeCtx := mock.QuickAuthenticatedForgeContextWithUser("POST", "/api/data", user)
//
//	// Full control
//	req := httptest.NewRequest("GET", "/api/profile", nil)
//	session := mock.CreateSession(user.ID, org.ID)
//	forgeCtx := mock.MockAuthenticatedForgeContext(req, user, app, env, org, session)
//
// # Complete Example
//
// Here's a complete example testing a handler that requires authentication:
//
//	func TestGetUserProfile(t *testing.T) {
//	    mock := authsometesting.NewMock(t)
//	    defer mock.Reset()
//
//	    // Handler being tested
//	    getUserProfile := func(ctx context.Context, mock *authsometesting.Mock) (map[string]string, error) {
//	        userID, ok := authsometesting.GetUserID(ctx)
//	        if !ok || userID.IsNil() {
//	            return nil, authsometesting.ErrNotAuthenticated
//	        }
//	        user, err := mock.GetUserFromContext(ctx)
//	        if err != nil {
//	            return nil, err
//	        }
//	        return map[string]string{
//	            "id":    user.ID.String(),
//	            "email": user.Email,
//	            "name":  user.Name,
//	        }, nil
//	    }
//
//	    t.Run("authenticated", func(t *testing.T) {
//	        ctx := mock.NewTestContext()
//	        profile, err := getUserProfile(ctx, mock)
//	        require.NoError(t, err)
//	        assert.NotEmpty(t, profile["id"])
//	    })
//
//	    t.Run("unauthenticated", func(t *testing.T) {
//	        ctx := context.Background()
//	        _, err := getUserProfile(ctx, mock)
//	        assert.Equal(t, authsometesting.ErrNotAuthenticated, err)
//	    })
//	}
//
// # Migration from v1.x
//
// Key changes in v2.0:
//
//   - All ID parameters now use xid.ID instead of string
//   - Context helpers renamed to match core/contexts (GetUserID vs GetLoggedInUser)
//   - WithOrg renamed to WithOrganization
//   - Added App and Environment support
//   - Added builder pattern for test data creation
//   - Added Forge context mocking
//   - Member storage now uses OrganizationMember schema
//
// # Best Practices
//
//   - Always call defer mock.Reset() to clean up between tests
//   - Use common scenarios for typical test cases
//   - Test both success and failure cases
//   - Use table-driven tests for multiple scenarios
//   - Verify context values before using them (check ok return values)
//   - Test full tenant hierarchy (App/Environment/Organization) for multi-tenant features
//   - Use builder pattern for complex test data
//   - Leverage Forge mocking for HTTP handler tests
//
// # Thread Safety
//
// The mock implementation is thread-safe and can be used across multiple goroutines
// in your tests.
//
// # Limitations
//
// This is a testing mock and does not:
//   - Connect to a real database
//   - Implement full RBAC policy evaluation
//   - Support all AuthSome plugins
//   - Provide rate limiting or caching
//   - Validate complex business logic
//
// For integration testing with real services, use the full AuthSome setup.
//
// See the README.md file in this package for more detailed documentation and examples.
package testing
