# AuthSome Testing Package Usage Example

This example demonstrates how to use the AuthSome testing package to test your application code that depends on AuthSome authentication.

## Overview

This example shows real-world usage patterns for:
- Testing user profile operations
- Testing admin-only operations with RBAC
- Testing multi-tenant organization switching
- Complex user journey scenarios
- Performance benchmarking

## Running the Tests

```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestUserProfileService_GetProfile

# Run benchmarks
go test -bench=. -benchmem

# Run with coverage
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Example Services Tested

### 1. UserProfileService
A service for managing user profiles with authentication requirements:
- `GetProfile()` - Retrieve user profile (requires auth)
- `UpdateProfile()` - Update profile (requires verified email)

### 2. AdminService
Admin-only operations with role-based access control:
- `DeleteUser()` - Delete users (requires admin role)

### 3. OrganizationService
Multi-tenant organization management:
- `ListUserOrganizations()` - List user's organizations
- `SwitchOrganization()` - Switch active organization context

## Test Scenarios Covered

### Basic Authentication Tests
- ✅ Authenticated user can access protected resources
- ✅ Unauthenticated user is blocked
- ✅ User profile contains correct data

### Email Verification Tests
- ✅ Verified user can perform actions
- ✅ Unverified user is blocked from sensitive actions

### Role-Based Access Control
- ✅ Admin can perform admin operations
- ✅ Regular user cannot perform admin operations
- ✅ Proper error messages for insufficient permissions

### Multi-Tenancy Tests
- ✅ User can belong to multiple organizations
- ✅ User can switch between organizations
- ✅ User cannot access non-member organizations
- ✅ Organization context is properly maintained

### Complex Scenarios
- ✅ Complete user journey (signup → verify → update profile → join org → switch org)
- ✅ Admin management workflow
- ✅ Session expiration handling

## Key Patterns Demonstrated

### 1. Using Common Scenarios

```go
mock := authsometesting.NewMock(t)
scenarios := mock.NewCommonScenarios()

// Quick setup for common cases
authUser := scenarios.AuthenticatedUser()
adminUser := scenarios.AdminUser()
multiOrgUser := scenarios.MultiOrgUser()
```

### 2. Testing Authorization

```go
// Test admin-only operation
err := adminService.DeleteUser(userCtx, mock, targetUserID)
assert.Equal(t, authsometesting.ErrInsufficientPermissions, err)
```

### 3. Context Management

```go
// Retrieve authenticated user
user, ok := authsometesting.GetLoggedInUser(ctx)
require.True(t, ok)

// Get current organization
org, ok := authsometesting.GetCurrentOrg(ctx)
require.True(t, ok)
```

### 4. Multi-Org Testing

```go
// Create user in multiple orgs
user := mock.CreateUser("user@example.com", "User")
org1 := mock.GetDefaultOrg()
org2 := mock.CreateOrganization("Second Org", "second-org")
mock.AddUserToOrg(user.ID, org2.ID, "member")

// Switch organization context
newCtx, err := service.SwitchOrganization(ctx, mock, org2.ID)
```

## Best Practices Shown

1. **Always clean up**: Use `defer mock.Reset()`
2. **Test both success and failure**: Cover happy path and error cases
3. **Use descriptive test names**: Clear what scenario is being tested
4. **Organize by service**: Group related tests together
5. **Test complex flows**: Simulate real user journeys
6. **Benchmark critical paths**: Ensure mock performance is acceptable

## Expected Test Output

```
=== RUN   TestUserProfileService_GetProfile
=== RUN   TestUserProfileService_GetProfile/authenticated_user_can_get_profile
=== RUN   TestUserProfileService_GetProfile/unauthenticated_user_cannot_get_profile
=== RUN   TestUserProfileService_GetProfile/profile_includes_organization_info
--- PASS: TestUserProfileService_GetProfile (0.00s)
    --- PASS: TestUserProfileService_GetProfile/authenticated_user_can_get_profile (0.00s)
    --- PASS: TestUserProfileService_GetProfile/unauthenticated_user_cannot_get_profile (0.00s)
    --- PASS: TestUserProfileService_GetProfile/profile_includes_organization_info (0.00s)

=== RUN   TestUserProfileService_UpdateProfile
=== RUN   TestUserProfileService_UpdateProfile/verified_user_can_update_profile
=== RUN   TestUserProfileService_UpdateProfile/unverified_user_cannot_update_profile
=== RUN   TestUserProfileService_UpdateProfile/unauthenticated_user_cannot_update_profile
--- PASS: TestUserProfileService_UpdateProfile (0.00s)
    --- PASS: TestUserProfileService_UpdateProfile/verified_user_can_update_profile (0.00s)
    --- PASS: TestUserProfileService_UpdateProfile/unverified_user_cannot_update_profile (0.00s)
    --- PASS: TestUserProfileService_UpdateProfile/unauthenticated_user_cannot_update_profile (0.00s)

... (more tests)

PASS
coverage: 100.0% of statements
ok      github.com/xraph/authsome/examples/testing-usage    0.123s
```

## Adapting for Your Application

To use this testing approach in your own application:

1. **Import the testing package**:
   ```go
   import authsometesting "github.com/xraph/authsome/testing"
   ```

2. **Create mock in your tests**:
   ```go
   func TestMyService(t *testing.T) {
       mock := authsometesting.NewMock(t)
       defer mock.Reset()
       // ... your tests
   }
   ```

3. **Use context helpers in your services**:
   ```go
   func (s *MyService) DoSomething(ctx context.Context) error {
       user, ok := authsometesting.GetLoggedInUser(ctx)
       if !ok {
           return authsometesting.ErrNotAuthenticated
       }
       // ... your logic
   }
   ```

4. **Test with various scenarios**:
   ```go
   scenarios := mock.NewCommonScenarios()
   
   t.Run("as authenticated user", func(t *testing.T) {
       scenario := scenarios.AuthenticatedUser()
       err := service.DoSomething(scenario.Context)
       assert.NoError(t, err)
   })
   ```

## Further Reading

- [Testing Package Documentation](../../testing/README.md)
- [Testing Package API Reference](../../testing/doc.go)
- [AuthSome Main Documentation](../../README.md)

## Support

For questions or issues with the testing package:
- Open an issue: https://github.com/xraph/authsome/issues
- Check documentation: https://github.com/xraph/authsome/docs

