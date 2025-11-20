# AuthSome Testing Package

A comprehensive testing package for developers building applications with AuthSome authentication. This package provides mocked services, helpers, and utilities to test your integration without requiring a full AuthSome setup.

**✨ Version 2.0** - Now with full multi-tenancy support, `core/contexts` integration, and Forge HTTP handler testing.

## Overview

The testing package allows you to:
- Create mock users, organizations, apps, and environments
- Simulate authenticated contexts with full multi-tenant hierarchy
- Test authorization scenarios (roles, permissions)
- Verify session handling and expiration
- Test multi-tenancy scenarios across apps and environments
- Use pre-configured common scenarios
- Test HTTP handlers with mock Forge contexts

## What's New in v2.0

- **Multi-Tenancy Support**: Full App → Environment → Organization hierarchy
- **Core Contexts Integration**: Uses AuthSome's actual context system from `core/contexts`
- **Forge HTTP Testing**: Mock Forge contexts for testing HTTP handlers
- **Builder Pattern**: Fluent API for creating complex test data
- **OrganizationService Mock**: Complete organization service implementation
- **Updated API**: All methods now use `xid.ID` instead of strings

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

    // Quick setup: create authenticated context with full tenant hierarchy
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
- In-memory storage for test data (Apps, Environments, Organizations, Users, Sessions)
- Helper methods for creating test scenarios
- Context manipulation utilities
- Default entities (app, environment, organization) auto-created

```go
mock := authsometesting.NewMock(t)
defer mock.Reset() // Clean up after tests

// Access default entities
defaultApp := mock.GetDefaultApp()
defaultEnv := mock.GetDefaultEnvironment()
defaultOrg := mock.GetDefaultOrg()
```

### Multi-Tenancy Architecture

The testing framework mirrors AuthSome's multi-tenant architecture:

```
App (e.g., "My SaaS Platform")
  └── Environment (e.g., "production", "staging")
      └── Organization (e.g., "Acme Corp", "TechStart Inc")
          └── Users/Members
```

Every authenticated context includes:
- **App ID**: Top-level tenant
- **Environment ID**: Per-app environment
- **Organization ID**: User-created workspace
- **User ID**: Individual user

## Creating Test Data

### Simple Methods

#### Users

```go
// Create a basic user (auto-added to default org)
user := mock.CreateUser("user@example.com", "Test User")

// Create a user with specific role
adminUser := mock.CreateUserWithRole("admin@example.com", "Admin", "admin")

// Get user
user, err := mock.GetUser(userID)
```

#### Apps and Environments

```go
// Get defaults (auto-created)
app := mock.GetDefaultApp()
env := mock.GetDefaultEnvironment()

// Access via getters
app, err := mock.GetApp(appID)
env, err := mock.GetEnvironment(envID)
```

#### Organizations

```go
// Get default organization (auto-created)
org := mock.GetDefaultOrg()

// Create additional organization
org := mock.CreateOrganization("My Org", "my-org")

// Get organization
org, err := mock.GetOrganization(orgID)

// Add user to organization
member := mock.AddUserToOrg(userID, orgID, "member")

// Get user's organizations
orgs, err := mock.GetUserOrgs(userID)
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

### Builder Pattern (Fluent API)

For more complex test data creation:

```go
// Build a custom app
app := mock.NewApp().
    WithName("Custom App").
    WithSlug("custom-app").
    Build()

// Build a custom environment
env := mock.NewEnvironment(app.ID).
    WithName("staging").
    WithSlug("staging").
    Build()

// Build a custom organization
org := mock.NewOrganization().
    WithName("Acme Corp").
    WithSlug("acme").
    WithApp(app.ID).
    WithEnvironment(env.ID).
    Build()

// Build a custom user
user := mock.NewUser().
    WithEmail("john@acme.com").
    WithName("John Doe").
    WithRole("admin").
    WithEmailVerified(true).
    Build()
```

## Working with Context

### Creating Authenticated Contexts

```go
// Method 1: Quick way - creates user, org, session with all tenant levels
ctx := mock.NewTestContext()

// Method 2: With specific user
user := mock.CreateUser("user@example.com", "Test User")
ctx := mock.NewTestContextWithUser(user)

// Method 3: Manual setup with full control
user := mock.CreateUser("user@example.com", "Test User")
session := mock.CreateSession(user.ID, mock.GetDefaultOrg().ID)
ctx := context.Background()
ctx = mock.WithApp(ctx, mock.GetDefaultApp().ID)
ctx = mock.WithEnvironment(ctx, mock.GetDefaultEnvironment().ID)
ctx = mock.WithOrganization(ctx, mock.GetDefaultOrg().ID)
ctx = mock.WithSession(ctx, session.ID)
```

### Retrieving from Context

The testing package uses AuthSome's actual context system (`core/contexts`):

```go
// Get user ID
userID, ok := authsometesting.GetUserID(ctx)
if !ok || userID.IsNil() {
    // Not authenticated
}

// Get app ID
appID, ok := authsometesting.GetAppID(ctx)

// Get environment ID
envID, ok := authsometesting.GetEnvironmentID(ctx)

// Get organization ID
orgID, ok := authsometesting.GetOrganizationID(ctx)

// Get session
session, ok := authsometesting.GetSession(ctx)

// Get full entities from context using Mock
user, err := mock.GetUserFromContext(ctx)
app, err := mock.GetAppFromContext(ctx)
env, err := mock.GetEnvironmentFromContext(ctx)
org, err := mock.GetOrganizationFromContext(ctx)
```

## Authorization Helpers

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

## Mock Services

### UserService

```go
// Create user
user, err := mock.UserService.Create(ctx, &user.CreateUserRequest{
    Email: "test@example.com",
    Name:  "Test User",
})

// Get user by ID
user, err := mock.UserService.GetByID(ctx, userID)

// Get user by email
user, err := mock.UserService.GetByEmail(ctx, "test@example.com")

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
// Get organization by ID
org, err := mock.OrganizationService.GetByID(ctx, orgID)

// Get organization by slug
org, err := mock.OrganizationService.GetBySlug(ctx, "my-org")

// List organizations
orgsResp, err := mock.OrganizationService.ListOrganizations(ctx, filter)

// Get members
membersResp, err := mock.OrganizationService.GetMembers(ctx, orgID)

// Get user organizations
orgs, err := mock.OrganizationService.GetUserOrganizations(ctx, userID)
```

## Testing HTTP Handlers

### Mock Forge Contexts

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

    // Quick authenticated Forge context
    forgeCtx := mock.QuickAuthenticatedForgeContext("GET", "/api/profile")

    // Or with specific user
    forgeCtx := mock.QuickAuthenticatedForgeContextWithUser("POST", "/api/data", user)

    // Or full control
    req := httptest.NewRequest("GET", "/api/profile", nil)
    session := mock.CreateSession(user.ID, mock.GetDefaultOrg().ID)
    forgeCtx := mock.MockAuthenticatedForgeContext(
        req,
        user,
        mock.GetDefaultApp(),
        mock.GetDefaultEnvironment(),
        mock.GetDefaultOrg(),
        session,
    )

    // Test your handler
    err := yourHandler.HandleProfile(forgeCtx)
    assert.NoError(t, err)
    assert.Equal(t, 200, forgeCtx.GetStatus())
}
```

## Common Scenarios

Pre-configured scenarios for common testing needs:

```go
scenarios := mock.CommonScenarios()

// Authenticated user
scenario := scenarios.AuthenticatedUser()
ctx := scenario.Context
user := scenario.User
app := scenario.App
env := scenario.Environment
org := scenario.Org

// Admin user
scenario := scenarios.AdminUser()

// Unverified user
scenario := scenarios.UnverifiedUser()

// Multi-org user
scenario := scenarios.MultiOrgUser()

// Expired session
scenario := scenarios.ExpiredSession()

// Unauthenticated (no auth data)
scenario := scenarios.UnauthenticatedUser()

// Inactive user (deleted/suspended)
scenario := scenarios.InactiveUser()
```

Each scenario includes:
- `Name`: Scenario identifier
- `Description`: What the scenario represents
- `User`: Test user (if applicable)
- `App`: Test app
- `Environment`: Test environment
- `Org`: Test organization
- `Session`: Test session (if applicable)
- `Context`: Pre-configured context

## Complete Examples

### Example 1: Testing User Profile Handler

```go
func TestGetUserProfile(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    // Create test data
    user := mock.CreateUser("john@example.com", "John Doe")
    ctx := mock.NewTestContextWithUser(user)

    // Test handler
    profile, err := getUserProfile(ctx, mock)

    // Assertions
    require.NoError(t, err)
    assert.Equal(t, "John Doe", profile.Name)
    assert.Equal(t, "john@example.com", profile.Email)
}
```

### Example 2: Testing Multi-Org Scenario

```go
func TestMultiOrgAccess(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    // Create user and multiple orgs
    user := mock.CreateUser("user@example.com", "Test User")
    org1 := mock.GetDefaultOrg()
    org2 := mock.CreateOrganization("Second Org", "second-org")
    
    // Add user to second org as admin
    mock.AddUserToOrg(user.ID, org2.ID, "admin")

    // Test access to org1 (as member)
    ctx1 := context.Background()
    ctx1 = mock.WithApp(ctx1, mock.GetDefaultApp().ID)
    ctx1 = mock.WithEnvironment(ctx1, mock.GetDefaultEnvironment().ID)
    ctx1 = mock.WithOrganization(ctx1, org1.ID)
    ctx1 = mock.WithUser(ctx1, user.ID)
    
    member, err := mock.RequireOrgMember(ctx1, org1.ID)
    require.NoError(t, err)
    assert.Equal(t, "member", member.Role)

    // Test access to org2 (as admin)
    ctx2 := context.Background()
    ctx2 = mock.WithApp(ctx2, mock.GetDefaultApp().ID)
    ctx2 = mock.WithEnvironment(ctx2, mock.GetDefaultEnvironment().ID)
    ctx2 = mock.WithOrganization(ctx2, org2.ID)
    ctx2 = mock.WithUser(ctx2, user.ID)
    
    member, err = mock.RequireOrgRole(ctx2, org2.ID, "admin")
    require.NoError(t, err)
    assert.Equal(t, "admin", member.Role)
}
```

### Example 3: Testing Session Expiration

```go
func TestSessionExpiration(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    user := mock.CreateUser("user@example.com", "Test User")
    expiredSession := mock.CreateExpiredSession(user.ID, mock.GetDefaultOrg().ID)

    // Try to use expired session
    _, err := mock.SessionService.Validate(context.Background(), expiredSession.Token)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "expired")
}
```

### Example 4: Testing with Builder Pattern

```go
func TestComplexScenario(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    // Build custom app structure
    app := mock.NewApp().
        WithName("Production App").
        WithSlug("prod-app").
        Build()

    env := mock.NewEnvironment(app.ID).
        WithName("production").
        WithSlug("prod").
        Build()

    org := mock.NewOrganization().
        WithName("Enterprise Corp").
        WithSlug("enterprise").
        WithApp(app.ID).
        WithEnvironment(env.ID).
        Build()

    user := mock.NewUser().
        WithEmail("cto@enterprise.com").
        WithName("Jane Smith").
        WithRole("owner").
        Build()

    // Create context with all custom entities
    session := mock.CreateSession(user.ID, org.ID)
    ctx := context.Background()
    ctx = mock.WithApp(ctx, app.ID)
    ctx = mock.WithEnvironment(ctx, env.ID)
    ctx = mock.WithOrganization(ctx, org.ID)
    ctx = mock.WithSession(ctx, session.ID)

    // Test your application logic
    // ...
}
```

## Error Constants

Common errors you can test against:

```go
authsometesting.ErrNotAuthenticated      // User not authenticated
authsometesting.ErrInvalidSession        // Session invalid or expired
authsometesting.ErrUserNotFound          // User doesn't exist
authsometesting.ErrOrgNotFound          // Organization doesn't exist
authsometesting.ErrNotOrgMember         // User not member of org
authsometesting.ErrInsufficientPermissions // User lacks required permissions
authsometesting.ErrUserInactive         // User account inactive/deleted
```

## Thread Safety

All Mock operations are thread-safe and can be used in concurrent tests:

```go
func TestConcurrent(t *testing.T) {
    mock := authsometesting.NewMock(t)
    defer mock.Reset()

    t.Run("user1", func(t *testing.T) {
        t.Parallel()
        user := mock.CreateUser("user1@example.com", "User 1")
        // ... test with user1
    })

    t.Run("user2", func(t *testing.T) {
        t.Parallel()
        user := mock.CreateUser("user2@example.com", "User 2")
        // ... test with user2
    })
}
```

## Best Practices

1. **Always call Reset()**: Use `defer mock.Reset()` to ensure test isolation
2. **Use scenarios for common cases**: Leverage `CommonScenarios()` for standard tests
3. **Test full hierarchy**: When testing multi-tenancy, include App/Environment/Organization
4. **Use builders for complex data**: Builder pattern makes complex test data readable
5. **Verify contexts**: Always check context retrieval with `ok` return values
6. **Test error paths**: Use error constants to test failure scenarios
7. **Parallel tests**: Mock is thread-safe, use `t.Parallel()` where appropriate

## Migration from v1.x

If you're upgrading from v1.x:

### API Changes

```go
// Old (v1.x)
user.ID.String()                       // String IDs
authsometesting.GetLoggedInUser(ctx)   // Old context helpers
mock.WithOrg(ctx, orgID)               // Old method name

// New (v2.0)
user.ID                                // xid.ID directly
authsometesting.GetUserID(ctx)         // New context helpers
mock.WithOrganization(ctx, orgID)      // New method name
```

### New Features

- App and Environment support
- Builder pattern for test data
- Forge context mocking
- OrganizationService mock
- Core contexts integration

### Breaking Changes

- All ID parameters now use `xid.ID` instead of `string`
- Context helpers renamed to match `core/contexts`
- `WithOrg` renamed to `WithOrganization`
- Member storage now uses `OrganizationMember` schema

## Support

For issues, questions, or contributions, please visit the [AuthSome repository](https://github.com/xraph/authsome).
