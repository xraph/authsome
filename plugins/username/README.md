# Username Authentication Plugin

Enterprise-grade username/password authentication for AuthSome with advanced security features.

## Features

- 🔐 **Password Strength Validation** - Configurable password requirements
- 🚫 **Account Lockout** - Automatic lockout after failed attempts
- 📜 **Password History** - Prevents password reuse
- ⏰ **Password Expiry** - Enforces periodic password changes
- 🚦 **Advanced Rate Limiting** - Redis-backed distributed rate limiting
- 📊 **Comprehensive Audit Logging** - Full audit trail for compliance
- 🔄 **Multi-tenant Support** - App/Organization context isolation
- 🎯 **2FA Integration** - Seamless two-factor authentication support
- ⚙️ **Highly Configurable** - Flexible functional options API
- ✅ **Production Ready** - Fully tested and battle-hardened

## Installation

```go
import "github.com/xraph/authsome/plugins/username"

// Basic usage
usernamePlugin := username.NewPlugin()

// With custom configuration
usernamePlugin := username.NewPlugin(
    username.WithMinPasswordLength(12),
    username.WithRequireUppercase(true),
    username.WithRequireNumber(true),
    username.WithRequireSpecialChar(true),
    username.WithLockoutEnabled(true),
    username.WithMaxFailedAttempts(5),
)
```

## Configuration

### YAML Configuration

```yaml
auth:
  username:
    # Password requirements
    minPasswordLength: 8
    maxPasswordLength: 128
    requireUppercase: true
    requireLowercase: true
    requireNumber: true
    requireSpecialChar: true
    allowUsernameLogin: true
    
    # Account lockout
    lockoutEnabled: true
    maxFailedAttempts: 5
    lockoutDuration: "15m"         # 15 minutes
    failedAttemptWindow: "10m"     # 10 minutes
    
    # Password history
    passwordHistorySize: 5
    preventPasswordReuse: true
    
    # Password expiry
    passwordExpiryEnabled: false
    passwordExpiryDays: 90
    passwordExpiryWarningDays: 7
    
    # Rate limiting
    rateLimit:
      enabled: true
      useRedis: true               # For production/distributed
      redisAddr: "localhost:6379"
      redisPassword: ""
      redisDb: 0
      
      # Per-IP limits
      signupPerIp:
        window: "1h"
        max: 10
      
      signinPerIp:
        window: "15m"
        max: 20
      
      signinPerUser:
        window: "5m"
        max: 5
```

### Programmatic Configuration

```go
usernamePlugin := username.NewPlugin(
    // Password requirements
    username.WithMinPasswordLength(12),
    username.WithMaxPasswordLength(128),
    username.WithRequireUppercase(true),
    username.WithRequireLowercase(true),
    username.WithRequireNumber(true),
    username.WithRequireSpecialChar(true),
    
    // Account lockout
    username.WithLockoutEnabled(true),
    username.WithMaxFailedAttempts(3),
    username.WithLockoutDuration(30*time.Minute),
    
    // Password history
    username.WithPasswordHistorySize(10),
    username.WithPreventPasswordReuse(true),
    
    // Password expiry
    username.WithPasswordExpiryEnabled(true),
    username.WithPasswordExpiryDays(60),
)
```

## API Endpoints

### Sign Up

```http
POST /api/auth/username/signup
Content-Type: application/json

{
    "username": "johndoe",
    "password": "SecureP@ss123"
}
```

**Response (201 Created):**
```json
{
    "status": "created",
    "message": "User created successfully"
}
```

**Error Responses:**
- `400` - Invalid request (validation failed)
- `409` - Username already exists
- `429` - Rate limit exceeded

**Rate Limits:**
- 10 signups per hour per IP address

### Sign In

```http
POST /api/auth/username/signin
Content-Type: application/json

{
    "username": "johndoe",
    "password": "SecureP@ss123",
    "remember": true
}
```

**Response (200 OK):**
```json
{
    "user": {
        "id": "...",
        "email": "...",
        "username": "johndoe",
        "name": "John Doe"
    },
    "session": {
        "id": "...",
        "token": "...",
        "expiresAt": "2025-11-21T10:00:00Z"
    },
    "token": "session_token_..."
}
```

**Response (200 OK - 2FA Required):**
```json
{
    "user": {
        "id": "...",
        "email": "...",
        "username": "johndoe"
    },
    "require_twofa": true,
    "device_id": "device_fingerprint_..."
}
```

**Response (403 Forbidden - Account Locked):**
```json
{
    "code": "ACCOUNT_LOCKED",
    "message": "Account locked due to too many failed login attempts",
    "locked_until": "2025-11-20T12:15:00Z",
    "locked_minutes": 15
}
```

**Error Responses:**
- `400` - Invalid request
- `401` - Invalid credentials
- `403` - Account locked
- `429` - Rate limit exceeded

**Rate Limits:**
- 20 signin attempts per 15 minutes per IP address
- 5 signin attempts per 5 minutes per username

## Security Features

### Password Strength Validation

Configurable password requirements:
- Minimum and maximum length
- Uppercase letters
- Lowercase letters
- Numbers
- Special characters

### Account Lockout

Automatic account lockout protection:
- **Failed Attempt Tracking**: Records all failed login attempts with IP and user agent
- **Configurable Threshold**: Lock account after N failed attempts (default: 5)
- **Time Window**: Failed attempts counted within a window (default: 10 minutes)
- **Lockout Duration**: Account locked for specified duration (default: 15 minutes)
- **Auto-unlock**: Accounts automatically unlock after lockout duration expires
- **Attempt Reset**: Failed attempts cleared after successful login

### Password History

Prevents password reuse:
- **History Tracking**: Stores hashed passwords in history
- **Configurable Size**: Track last N passwords (default: 5)
- **Reuse Prevention**: Blocks using any password from history
- **Secure Storage**: Only password hashes stored, never plain text
- **Auto-cleanup**: Old history entries automatically removed

### Password Expiry

Enforces periodic password changes:
- **Configurable Expiry**: Set password expiry period (default: 90 days)
- **Warning Period**: Warn users before expiry (default: 7 days)
- **Grace Period**: Optional grace period after expiry
- **Creation Date Fallback**: Uses account creation date if no password change recorded

### Rate Limiting

Multi-layer rate limiting protection:
- **Per-IP Limits**: Prevent abuse from single IPs
- **Per-Username Limits**: Prevent targeted attacks
- **Redis-backed**: Distributed rate limiting for production
- **Automatic Fallback**: Memory storage if Redis unavailable
- **Configurable Rules**: Separate limits for signup and signin

### Audit Trail

Every action is logged with structured metadata:
- User registration attempts
- Login attempts (success/failure)
- Account lockouts
- Failed attempt recordings
- Password validation failures
- All errors and security events

## Multi-Tenancy

The plugin fully supports AuthSome's multi-tenant architecture:

- **App-level Isolation**: Usernames scoped to applications
- **Organization Context**: Org-specific authentication
- **Environment Support**: Dev/staging/prod separation
- **Context Propagation**: Proper tenant context throughout
- **Temp Email Scoping**: Generated emails include app ID to prevent collisions

## Audit Logging Events

### SignUp Events
- `username_signup_attempt` - Signup initiated
- `username_signup_success` - User created successfully
- `username_signup_failed` - Signup failed
- `username_already_exists` - Username collision
- `username_weak_password` - Weak password rejected

### SignIn Events
- `username_signin_attempt` - Login initiated
- `username_signin_success` - Successful authentication
- `username_signin_failed` - Failed authentication
- `username_invalid_credentials` - Wrong username/password
- `username_account_locked` - Attempted login on locked account
- `username_password_expired` - Login with expired password
- `username_2fa_required` - 2FA challenge issued

### Security Events
- `username_failed_attempt_recorded` - Failed attempt logged
- `username_account_locked_auto` - Auto-locked after max attempts
- `username_account_unlocked` - Manual unlock
- `username_failed_attempts_cleared` - Attempts reset after success
- `username_password_reuse_blocked` - Blocked password from history

## Error Handling

The plugin uses structured error responses:

```go
// Authentication errors
errs.InvalidCredentials()          // Wrong username/password
errs.WeakPassword(reason)          // Password doesn't meet requirements
errs.UsernameAlreadyExists(username) // Username taken
errs.AccountLocked(reason)         // Account locked
errs.PasswordExpired()             // Password needs changing

// Validation errors
errs.RequiredField(field)          // Missing required field
errs.BadRequest(message)           // Invalid request

// Rate limiting
errs.RateLimitExceeded(retryAfter) // Too many requests
```

**Error Response Format:**
```json
{
    "code": "WEAK_PASSWORD",
    "message": "Password does not meet security requirements",
    "httpStatus": 400,
    "context": {
        "reason": "password must contain at least one uppercase letter"
    }
}
```

## Testing

```bash
# Run tests
go test ./plugins/username/...

# Run with coverage
go test ./plugins/username/... -cover

# Run with race detection
go test ./plugins/username/... -race
```

**Test Coverage:**
- Password validation (all requirements)
- Configuration options
- Plugin initialization
- Request/response serialization
- Password expiry calculations
- Account lockout error handling

## Production Deployment

### Checklist

- [ ] Configure strong password requirements
- [ ] Enable account lockout (`lockoutEnabled: true`)
- [ ] Set appropriate lockout threshold (3-5 attempts)
- [ ] Configure rate limiting with Redis (`useRedis: true`)
- [ ] Enable password history tracking
- [ ] Consider enabling password expiry for sensitive apps
- [ ] Set up audit logging
- [ ] Monitor failed login attempts
- [ ] Configure alerting for suspicious activity
- [ ] Test account lockout behavior
- [ ] Document password policies for users

### Monitoring

Key metrics to monitor:
- Signup rate and success rate
- Login success/failure rates
- Account lockout frequency
- Password validation failures
- Rate limit hits
- Average authentication time
- Failed attempt patterns

### Security Best Practices

1. **Password Requirements**: Balance security with usability
   - Minimum 12 characters for sensitive applications
   - Require mix of character types
   - Don't make requirements too complex

2. **Account Lockout**: Protect against brute force
   - 3-5 failed attempts is recommended
   - 15-30 minute lockout duration
   - Monitor for patterns of attacks

3. **Password History**: Prevent reuse
   - Track 5-10 previous passwords
   - More for highly sensitive applications

4. **Password Expiry**: Balance security and UX
   - 60-90 days for sensitive data
   - Consider not enforcing for low-risk apps
   - Provide warning period

5. **Rate Limiting**: Essential for production
   - Always use Redis in distributed systems
   - Set conservative limits initially
   - Adjust based on legitimate traffic patterns

## Examples

### Basic Username Auth Flow

```go
// 1. User signs up
client.POST("/api/auth/username/signup", {
    "username": "johndoe",
    "password": "SecureP@ss123"
})
// Response: {"status": "created"}

// 2. User signs in
client.POST("/api/auth/username/signin", {
    "username": "johndoe",
    "password": "SecureP@ss123",
    "remember": true
})
// Response: {user, session, token}
```

### Enterprise Configuration

```go
usernamePlugin := username.NewPlugin(
    // Strong password policy
    username.WithMinPasswordLength(14),
    username.WithRequireUppercase(true),
    username.WithRequireLowercase(true),
    username.WithRequireNumber(true),
    username.WithRequireSpecialChar(true),
    
    // Aggressive lockout
    username.WithMaxFailedAttempts(3),
    username.WithLockoutDuration(30*time.Minute),
    
    // Extensive password history
    username.WithPasswordHistorySize(10),
    username.WithPreventPasswordReuse(true),
    
    // Enforce password rotation
    username.WithPasswordExpiryEnabled(true),
    username.WithPasswordExpiryDays(60),
)
```

### Handling Account Lockouts

When an account is locked, the API returns:

```json
{
    "code": "ACCOUNT_LOCKED",
    "message": "Account is locked",
    "context": {
        "reason": "locked for 15 minutes",
        "lockedUntil": "2025-12-28T20:15:00Z",
        "lockedMinutes": 15
    },
    "timestamp": "2025-12-28T20:00:00Z"
}
```

The account will automatically unlock after the `lockedMinutes` period expires. The `lockedUntil` timestamp shows the exact time when the account will be unlocked.

## Troubleshooting

### Account Locked Issues

1. Check account lockout status in database
2. Verify lockout configuration is appropriate
3. Review audit logs for failed attempt patterns
4. Consider manual unlock if legitimate user

### Password Validation Failures

1. Review password requirements configuration
2. Check audit logs for `username_weak_password` events
3. Provide clear error messages to users
4. Consider adjusting requirements if too strict

### Rate Limit Issues

1. Check Redis connection if `useRedis: true`
2. Review rate limit configuration
3. Check audit logs for rate limit events
4. Whitelist trusted IPs if necessary
5. Adjust limits for your traffic patterns

## License

See main AuthSome license.

## Support

For issues and questions, see main AuthSome documentation.

---

**Version**: 1.0.0 (Enterprise Grade)  
**Status**: Production Ready ✅

