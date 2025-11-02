# AuthSome Testing Package

A comprehensive testing package for developers building applications with AuthSome authentication. This package provides mocked services, helpers, and utilities to test your integration without requiring a full AuthSome setup.

## Overview

The testing package allows you to:
- Create mock users, organizations, and sessions
- Simulate authenticated contexts
- Test authorization scenarios (roles, permissions)
- Verify session handling and expiration
- Test multi-tenancy scenarios
- Use pre-configured common scenarios

## Quick Start

```go
import (
    "testing"
    authsometesting "github.com/xraph/authsome/testing"
)

func TestMyHandler(t *testing.T) {
    // Create a mock instance
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    // Quick setup: create authenticated context
    ctx := mock.NewTestContext()

    // Your test code here
    result := myHandler(ctx)
    // ... assertions
}
```

## Core Components

### Mock Instance

The `Mock` struct is the main entry point for testing. It provides:
- Mock user, session, and organization services
- In-memory storage for test data
- Helper methods for creating test scenarios
- Context manipulation utilities

```go
mock := authsometesting.NewMock(t)
defer mock.Reset() // Clean up after tests
```

### Creating Test Data

#### Users

```go
// Create a basic user
user := mock.CreateUser("user@example.com", "Test User")

// Create a user with specific role
adminUser := mock.CreateUserWithRole("admin@example.com", "Admin", "admin")

// Get user
user, err := mock.GetUser(userID)
```

#### Organizations

```go
// Get default organization (auto-created)
org := mock.GetDefaultOrg()

// Create additional organization
org := mock.CreateOrganization("My Org", "my-org")

// Add user to organization
member := mock.AddUserToOrg(userID, orgID, "member")
```

#### Sessions

```go
// Create active session
session := mock.CreateSession(userID, orgID)

// Create expired session (for testing expiration)
expiredSession := mock.CreateExpiredSession(userID, orgID)

// Get session
session, err := mock.GetSession(sessionID)
```

### Working with Context

#### Authenticated Context

```go
// Method 1: Quick way - creates user, org, and session automatically
ctx := mock.NewTestContext()

// Method 2: With specific user
user := mock.CreateUser("user@example.com", "Test User")
ctx := mock.NewTestContextWithUser(user)

// Method 3: Manual setup with full control
user := mock.CreateUser("user@example.com", "Test User")
session := mock.CreateSession(user.ID, mock.GetDefaultOrg().ID)
ctx := context.Background()
ctx = mock.WithSession(ctx, session.ID)
ctx = mock.WithUser(ctx, user.ID)
ctx = mock.WithOrg(ctx, mock.GetDefaultOrg().ID)
```

#### Retrieving from Context

```go
// Get logged-in user
user, ok := authsometesting.GetLoggedInUser(ctx)
if !ok {
    // Not authenticated
}

// Get user ID
userID, ok := authsometesting.GetLoggedInUserID(ctx)

// Get current organization
org, ok := authsometesting.GetCurrentOrg(ctx)

// Get organization ID
orgID, ok := authsometesting.GetCurrentOrgID(ctx)

// Get session
session, ok := authsometesting.GetCurrentSession(ctx)

// Get session ID
sessionID, ok := authsometesting.GetCurrentSessionID(ctx)
```

### Authorization Helpers

```go
// Require authentication
user, err := mock.RequireAuth(ctx)
if err != nil {
    // Handle: ErrNotAuthenticated, ErrInvalidSession, ErrUserNotFound, ErrUserInactive
}

// Require organization membership
member, err := mock.RequireOrgMember(ctx, orgID)
if err != nil {
    // Handle: ErrOrgNotFound, ErrNotOrgMember
}

// Require specific role
member, err := mock.RequireOrgRole(ctx, orgID, "admin")
if err != nil {
    // Handle: ErrInsufficientPermissions
}
```

### Common Test Errors

The package provides standard test errors:

```go
authsometesting.ErrNotAuthenticated        // User not authenticated
authsometesting.ErrInvalidSession          // Session invalid or expired
authsometesting.ErrUserNotFound            // User not found
authsometesting.ErrUserInactive            // User account inactive
authsometesting.ErrOrgNotFound             // Organization not found
authsometesting.ErrNotOrgMember            // User not a member
authsometesting.ErrInsufficientPermissions // User lacks required role
```

## Common Scenarios

The package includes pre-configured scenarios for typical test cases:

```go
scenarios := mock.NewCommonScenarios()

// Authenticated user with verified email
scenario := scenarios.AuthenticatedUser()

// Admin user with elevated privileges
scenario := scenarios.AdminUser()

// User with unverified email
scenario := scenarios.UnverifiedUser()

// User belonging to multiple organizations
scenario := scenarios.MultiOrgUser()

// User with expired session
scenario := scenarios.ExpiredSession()

// No authentication
scenario := scenarios.UnauthenticatedUser()

// Inactive user account
scenario := scenarios.InactiveUser()

// Use scenario in tests
user := scenario.User
org := scenario.Org
session := scenario.Session
ctx := scenario.Context
```

## Mock Services

The package provides mock implementations of core services:

### UserService

```go
// Create user
user, err := mock.UserService.Create(ctx, &user.CreateUserRequest{
    Email: "user@example.com",
    Name:  "Test User",
})

// Get user by ID
user, err := mock.UserService.GetByID(ctx, userID)

// Get user by email
user, err := mock.UserService.GetByEmail(ctx, "user@example.com")

// Update user
user, err := mock.UserService.Update(ctx, userID, &user.UpdateUserRequest{
    Name: &newName,
})

// Delete user
err := mock.UserService.Delete(ctx, userID)
```

### SessionService

```go
// Create session
session, err := mock.SessionService.Create(ctx, &session.CreateSessionRequest{
    UserID:         userID,
    OrganizationID: orgID,
})

// Get session by ID
session, err := mock.SessionService.GetByID(ctx, sessionID)

// Get session by token
session, err := mock.SessionService.GetByToken(ctx, token)

// Validate session
session, err := mock.SessionService.Validate(ctx, token)

// Delete session
err := mock.SessionService.Delete(ctx, sessionID)
```

### OrganizationService

```go
// Create organization
org, err := mock.OrganizationService.Create(ctx, &organization.CreateOrganizationRequest{
    Name: "My Org",
    Slug: "my-org",
})

// Get organization by ID
org, err := mock.OrganizationService.GetByID(ctx, orgID)

// Get organization by slug
org, err := mock.OrganizationService.GetBySlug(ctx, "my-org")

// Add member
member, err := mock.OrganizationService.AddMember(ctx, &organization.AddMemberRequest{
    OrganizationID: orgID,
    UserID:         userID,
    Role:           "member",
})

// Get members
members, err := mock.OrganizationService.GetMembers(ctx, orgID)

// Get user organizations
orgs, err := mock.OrganizationService.GetUserOrganizations(ctx, userID)
```

## Testing Forge Handlers

For testing HTTP handlers that use Forge context:

```go
import (
    "net/http"
    "net/http/httptest"
)

func TestHandler(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    // Create test user
    user := mock.CreateUser("user@example.com", "Test User")
    org := mock.GetDefaultOrg()
    session := mock.CreateSession(user.ID, org.ID)

    // Create authenticated request
    req := httptest.NewRequest("GET", "/api/profile", nil)
    forgeCtx := mock.MockAuthenticatedForgeContext(req, user, org, session)

    // Test your handler
    err := yourHandler.HandleProfile(forgeCtx)
    assert.NoError(t, err)
}
```

## Complete Examples

### Example 1: Testing User Profile Handler

```go
func TestGetUserProfile(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    // Handler being tested
    getUserProfile := func(ctx context.Context) (map[string]string, error) {
        user, ok := authsometesting.GetLoggedInUser(ctx)
        if !ok {
            return nil, authsometesting.ErrNotAuthenticated
        }
        return map[string]string{
            "id":    user.ID,
            "email": user.Email,
            "name":  user.Name,
        }, nil
    }

    t.Run("authenticated user", func(t *testing.T) {
        ctx := mock.NewTestContext()
        profile, err := getUserProfile(ctx)
        require.NoError(t, err)
        assert.NotEmpty(t, profile["id"])
    })

    t.Run("unauthenticated", func(t *testing.T) {
        ctx := context.Background()
        _, err := getUserProfile(ctx)
        assert.Equal(t, authsometesting.ErrNotAuthenticated, err)
    })
}
```

### Example 2: Testing Role-Based Access

```go
func TestAdminOnlyAction(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    // Handler requiring admin role
    adminAction := func(ctx context.Context) error {
        user, err := mock.RequireAuth(ctx)
        if err != nil {
            return err
        }

        orgID, _ := authsometesting.GetCurrentOrgID(ctx)
        _, err = mock.RequireOrgRole(ctx, orgID, "admin")
        return err
    }

    t.Run("admin user succeeds", func(t *testing.T) {
        scenarios := mock.NewCommonScenarios()
        scenario := scenarios.AdminUser()
        err := adminAction(scenario.Context)
        assert.NoError(t, err)
    })

    t.Run("regular user fails", func(t *testing.T) {
        scenarios := mock.NewCommonScenarios()
        scenario := scenarios.AuthenticatedUser()
        err := adminAction(scenario.Context)
        assert.Equal(t, authsometesting.ErrInsufficientPermissions, err)
    })
}
```

### Example 3: Testing Multi-Org Scenarios

```go
func TestMultiOrgAccess(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    // Create user and multiple orgs
    user := mock.CreateUser("user@example.com", "Test User")
    org1 := mock.GetDefaultOrg()
    org2 := mock.CreateOrganization("Second Org", "second-org")
    org3 := mock.CreateOrganization("Third Org", "third-org")

    // Add to different orgs with different roles
    mock.AddUserToOrg(user.ID, org2.ID, "admin")
    mock.AddUserToOrg(user.ID, org3.ID, "viewer")

    // Test access to each org
    orgs, err := mock.GetUserOrgs(user.ID)
    require.NoError(t, err)
    assert.Len(t, orgs, 3)

    // Verify roles
    ctx := mock.NewTestContextWithUser(user)
    
    _, err = mock.RequireOrgRole(ctx, org1.ID, "member")
    assert.NoError(t, err)
    
    _, err = mock.RequireOrgRole(ctx, org2.ID, "admin")
    assert.NoError(t, err)
    
    _, err = mock.RequireOrgRole(ctx, org3.ID, "admin")
    assert.Error(t, err) // User is only a viewer
}
```

### Example 4: Testing Session Expiration

```go
func TestSessionExpiration(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    user := mock.CreateUser("user@example.com", "Test User")

    t.Run("valid session", func(t *testing.T) {
        session := mock.CreateSession(user.ID, mock.GetDefaultOrg().ID)
        validated, err := mock.SessionService.Validate(context.Background(), session.Token)
        require.NoError(t, err)
        assert.Equal(t, session.ID, validated.ID)
    })

    t.Run("expired session", func(t *testing.T) {
        session := mock.CreateExpiredSession(user.ID, mock.GetDefaultOrg().ID)
        _, err := mock.SessionService.Validate(context.Background(), session.Token)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "expired")
    })
}
```

## Best Practices

### 1. Always Reset Between Tests

```go
func TestSomething(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset() // Clean up
    
    // Your test code
}
```

### 2. Use Common Scenarios for Typical Cases

```go
// Instead of manually creating everything
scenarios := mock.NewCommonScenarios()
scenario := scenarios.AuthenticatedUser()

// Use the pre-configured scenario
result := myHandler(scenario.Context)
```

### 3. Test Both Success and Failure Cases

```go
t.Run("success", func(t *testing.T) {
    ctx := mock.NewTestContext()
    // Test success case
})

t.Run("unauthenticated", func(t *testing.T) {
    ctx := context.Background()
    // Test failure case
})
```

### 4. Use Table-Driven Tests for Multiple Scenarios

```go
tests := []struct {
    name     string
    setup    func() context.Context
    wantErr  error
}{
    {
        name:    "authenticated",
        setup:   func() context.Context { return mock.NewTestContext() },
        wantErr: nil,
    },
    {
        name:    "unauthenticated",
        setup:   func() context.Context { return context.Background() },
        wantErr: authsometesting.ErrNotAuthenticated,
    },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        ctx := tt.setup()
        err := myHandler(ctx)
        assert.Equal(t, tt.wantErr, err)
    })
}
```

### 5. Verify Context Values

```go
ctx := mock.NewTestContext()

// Always verify context has what you expect
user, ok := authsometesting.GetLoggedInUser(ctx)
require.True(t, ok, "expected user in context")
assert.NotNil(t, user)
```

## Thread Safety

The mock implementation uses `sync.RWMutex` for thread-safe operations. You can safely use the same mock instance across multiple goroutines in your tests:

```go
mock := authsometesting.NewMock(t)
defer mock.Reset()

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(i int) {
        defer wg.Done()
        user := mock.CreateUser(fmt.Sprintf("user%d@example.com", i), "User")
        // ... test code
    }(i)
}
wg.Wait()
```

## Limitations

This testing package provides mocks for common scenarios but does not:
- Connect to a real database
- Implement full RBAC policy evaluation
- Support all AuthSome plugins
- Provide rate limiting or caching
- Validate complex business logic

For integration testing with real services, see the [Integration Testing Guide](../docs/INTEGRATION_TESTING.md).

## Support

For questions, issues, or contributions:
- GitHub Issues: https://github.com/xraph/authsome/issues
- Documentation: https://github.com/xraph/authsome/docs
- Examples: https://github.com/xraph/authsome/examples

## License

Same as AuthSome - see LICENSE file in repository root.

