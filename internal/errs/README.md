# Authsome Error Package

A robust, production-ready error handling system for Authsome that builds on Forge's error infrastructure.

## Overview

The `internal/errs` package provides a comprehensive error type system designed for authentication and authorization systems. It offers:

- **Structured Errors**: Rich error context with codes, HTTP status, and metadata
- **Error Wrapping**: Full support for Go 1.13+ error wrapping with `errors.Is` and `errors.As`
- **Forge Integration**: Seamless conversion to Forge's `HTTPError` for handler responses
- **Distributed Tracing**: Built-in trace ID support for request correlation
- **Type Safety**: Strongly-typed error constructors for common scenarios
- **JSON Serialization**: Ready for API responses with proper JSON tags

## Core Type: AuthsomeError

```go
type AuthsomeError struct {
    Code       string         // Error code (e.g., "USER_NOT_FOUND")
    Message    string         // Human-readable message
    HTTPStatus int            // HTTP status code
    Err        error          // Underlying error (if any)
    Context    map[string]any // Additional debug context
    Timestamp  time.Time      // When the error occurred
    TraceID    string         // Distributed tracing ID
    Details    any            // Structured details (e.g., validation errors)
}
```

## Quick Start

### Basic Usage

```go
import "github.com/xraph/authsome/internal/errs"

// Simple error creation
err := errs.UserNotFound()
// Returns: USER_NOT_FOUND: User not found (404)

// With additional context
err := errs.EmailAlreadyExists("user@example.com")
// Returns: EMAIL_ALREADY_EXISTS: Email address already registered (409)
// Context: {"email": "user@example.com"}
```

### Error Wrapping

```go
// Wrap an underlying error
dbErr := errors.New("connection timeout")
err := errs.DatabaseError("SELECT users", dbErr)

// Can still find the original error
if errors.Is(err, dbErr) {
    // Handle database connection issue
}
```

### Adding Context

```go
err := errs.PermissionDenied("delete", "user:123").
    WithContext("user_role", "viewer").
    WithContext("required_role", "admin").
    WithTraceID(requestID)

// Context: {"action": "delete", "resource": "user:123", 
//           "user_role": "viewer", "required_role": "admin"}
```

### Handler Integration

```go
func (h *Handler) GetUser(c forge.Context) error {
    user, err := h.service.FindByID(ctx, userID)
    if err != nil {
        var authErr *errs.AuthsomeError
        if errors.As(err, &authErr) {
            // Convert to Forge HTTPError for automatic response handling
            return authErr.ToHTTPError()
        }
        return err
    }
    
    return c.JSON(200, user)
}
```

## Error Categories

### Authentication Errors (401, 403)

```go
errs.InvalidCredentials()              // Invalid email or password
errs.EmailNotVerified(email)           // Email not verified
errs.AccountLocked(reason)             // Account locked
errs.TwoFactorRequired()               // 2FA required
errs.TokenExpired()                    // Token expired
errs.InvalidOTP()                      // Invalid OTP code
```

### User Errors (400, 404, 409)

```go
errs.UserNotFound()                    // User not found
errs.EmailAlreadyExists(email)         // Email already registered
errs.UsernameAlreadyExists(username)   // Username taken
errs.WeakPassword(reason)              // Password doesn't meet requirements
```

### Session Errors (401, 409)

```go
errs.SessionNotFound()                 // Session not found
errs.SessionExpired()                  // Session expired
errs.SessionRevoked()                  // Session revoked
errs.ConcurrentSessionLimit()          // Too many active sessions
```

### Organization Errors (403, 404, 409)

```go
errs.OrganizationNotFound()            // Organization not found
errs.NotMember()                       // Not an organization member
errs.InsufficientRole(required)        // Insufficient role
errs.SlugAlreadyExists(slug)           // Slug already in use
```

### RBAC Errors (403, 404)

```go
errs.PermissionDenied(action, resource) // Permission denied
errs.RoleNotFound(role)                 // Role not found
errs.PolicyViolation(policy, reason)    // Policy violation
```

### Rate Limiting Errors (429)

```go
errs.RateLimitExceeded(retryAfter)     // Rate limit exceeded
errs.TooManyAttempts(retryAfter)       // Too many failed attempts
```

### Validation Errors (400)

```go
fields := map[string]string{
    "email": "invalid format",
    "password": "too short",
}
errs.ValidationFailed(fields)          // Multiple validation errors

errs.InvalidInput(field, reason)       // Invalid input
errs.RequiredField(field)              // Required field missing
```

### Plugin Errors (404, 500, 503)

```go
errs.PluginNotFound(pluginID)          // Plugin not found
errs.PluginInitFailed(pluginID, err)   // Plugin init failed
errs.PluginDisabled(pluginID)          // Plugin disabled
```

### OAuth/SSO Errors (400, 502)

```go
errs.OAuthFailed(provider, reason)     // OAuth failed
errs.InvalidOAuthState()               // Invalid state parameter
errs.SAMLError(reason)                 // SAML error
```

### General Errors (500, 501)

```go
errs.InternalError(err)                // Internal server error
errs.DatabaseError(operation, err)     // Database error
errs.NotImplemented(feature)           // Feature not implemented
```

## Advanced Usage

### Error Comparison with Sentinel Errors

```go
err := someFunction()

// Using errors.Is for code-based comparison
if errs.Is(err, errs.ErrUserNotFound) {
    // Handle user not found
}

// Using errors.As to extract AuthsomeError
var authErr *errs.AuthsomeError
if errors.As(err, &authErr) {
    log.Printf("Error code: %s, Status: %d", authErr.Code, authErr.HTTPStatus)
}
```

### Creating Custom Errors

```go
// Use New for custom errors
err := errs.New(
    "CUSTOM_ERROR_CODE",
    "Something specific happened",
    http.StatusBadRequest,
).WithContext("custom_field", "value")

// Or wrap existing errors
err := errs.Wrap(
    originalErr,
    "CUSTOM_ERROR_CODE",
    "Custom message",
    http.StatusInternalServerError,
)
```

### Helper Functions

```go
// Extract HTTP status code from any error
status := errs.GetHTTPStatus(err)  // Returns 500 if not found

// Extract error code
code := errs.GetErrorCode(err)     // Returns "INTERNAL_ERROR" if not found

// Check if error is specific type
if errs.GetErrorCode(err) == errs.CodeUserNotFound {
    // Handle user not found
}
```

### Validation Error Details

```go
validationErrors := map[string]string{
    "email":    "Invalid email format",
    "password": "Password must be at least 8 characters",
    "name":     "Name is required",
}

err := errs.ValidationFailed(validationErrors)

// In handler
return c.JSON(err.HTTPStatus, map[string]interface{}{
    "error":   err.Code,
    "message": err.Message,
    "details": err.Details, // Contains the validation errors map
})
```

## Best Practices

### 1. Use Specific Error Constructors

**Good:**
```go
return errs.EmailAlreadyExists(email)
```

**Avoid:**
```go
return errs.New("EMAIL_ALREADY_EXISTS", "Email exists", 409)
```

### 2. Always Add Context

```go
// Good - includes relevant context
return errs.PermissionDenied("delete", "user:"+userID).
    WithContext("actor_role", actorRole).
    WithTraceID(requestID)

// Less helpful
return errs.PermissionDenied("delete", "user")
```

### 3. Wrap Underlying Errors

```go
// Good - preserves error chain
user, err := repo.FindByID(ctx, id)
if err != nil {
    return errs.DatabaseError("FindByID", err)
}

// Bad - loses original error
user, err := repo.FindByID(ctx, id)
if err != nil {
    return errs.InternalError(nil)
}
```

### 4. Use Sentinel Errors for Comparison

```go
// Good - uses sentinel
if errors.Is(err, errs.ErrUserNotFound) {
    // Handle
}

// Avoid - string comparison
if err.Error() == "user not found" {
    // Fragile
}
```

### 5. Convert to HTTPError in Handlers

```go
func (h *Handler) DoSomething(c forge.Context) error {
    result, err := h.service.Process(ctx)
    if err != nil {
        // Check if it's an AuthsomeError
        var authErr *errs.AuthsomeError
        if errors.As(err, &authErr) {
            // Let Forge handle the HTTP response
            return authErr.ToHTTPError()
        }
        // Fallback for unknown errors
        return forge.InternalError(err)
    }
    return c.JSON(200, result)
}
```

## Error Codes Reference

All error codes are constants in the package:

- `CodeInvalidCredentials` - Invalid authentication credentials
- `CodeUserNotFound` - User does not exist
- `CodeSessionExpired` - Session has expired
- `CodePermissionDenied` - Permission denied for action
- `CodeRateLimitExceeded` - Rate limit exceeded
- ...and 50+ more

See `errors.go` for the complete list.

## JSON Response Format

When serialized to JSON, AuthsomeError produces:

```json
{
  "code": "USER_NOT_FOUND",
  "message": "User not found",
  "timestamp": "2024-01-15T10:30:00Z",
  "trace_id": "req-123-abc",
  "context": {
    "user_id": "usr_abc123"
  },
  "details": null
}
```

Note: `Err` and `HTTPStatus` fields are omitted from JSON (marked with `json:"-"`).

## Integration with Existing Code

### Migrating from types.Errors

The package provides sentinel errors that match the existing `types` package:

```go
// Old code using types
import "github.com/xraph/authsome/types"

if err == types.ErrUserNotFound {
    // Handle
}

// New code using errs
import "github.com/xraph/authsome/internal/errs"

if errors.Is(err, errs.ErrUserNotFound) {
    // Handle - uses proper error comparison
}
```

### Service Layer Pattern

```go
// In service
func (s *Service) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    // Check if user exists
    existing, _ := s.repo.FindByEmail(ctx, req.Email)
    if existing != nil {
        return nil, errs.EmailAlreadyExists(req.Email)
    }
    
    // Validate password
    if !isStrongPassword(req.Password) {
        return nil, errs.WeakPassword("must contain uppercase, lowercase, and number")
    }
    
    // Create user
    user, err := s.repo.Create(ctx, req)
    if err != nil {
        return nil, errs.DatabaseError("Create", err).
            WithContext("email", req.Email)
    }
    
    return user, nil
}
```

## Testing

The package includes comprehensive tests. To run:

```bash
go test -v ./internal/errs/
```

All 40+ test cases cover:
- Error creation and wrapping
- Error comparison with `errors.Is`
- Error extraction with `errors.As`
- Context and detail handling
- Conversion methods
- Sentinel error behavior
- Error chaining

## Performance Considerations

- Context maps are pre-allocated but start small
- Timestamps use `time.Now()` which is fast
- Error constructors are lightweight
- No reflection or heavy allocations
- Suitable for high-throughput APIs

## Thread Safety

`AuthsomeError` is **not** thread-safe for mutation. Once created and configured (using `WithContext`, etc.), treat it as immutable.

For concurrent operations, create errors independently per goroutine.

