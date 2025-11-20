# Phone Authentication Plugin

Enterprise-grade phone authentication for AuthSome with SMS verification codes.

## Features

- 📱 **E.164 Phone Validation** - International phone number format support
- 🔐 **Cryptographically Secure Codes** - Uses `crypto/rand` for OTP generation
- 🚦 **Advanced Rate Limiting** - Redis-backed distributed rate limiting
- 📊 **Comprehensive Audit Logging** - Full audit trail for compliance
- 🔄 **Multi-tenant Support** - App/Organization context isolation
- 🎯 **Implicit Signup** - Optional automatic user creation
- 📞 **Multiple SMS Providers** - Via notification plugin integration
- ⚙️ **Highly Configurable** - Flexible functional options API
- ✅ **Production Ready** - Fully tested and battle-hardened

## Installation

```go
import "github.com/xraph/authsome/plugins/phone"

// Basic usage
phonePlugin := phone.NewPlugin()

// With custom configuration
phonePlugin := phone.NewPlugin(
    phone.WithCodeLength(6),
    phone.WithExpiryMinutes(10),
    phone.WithMaxAttempts(5),
    phone.WithSMSProvider("twilio"),
)
```

## Configuration

### YAML Configuration

```yaml
auth:
  phone:
    # Code settings
    codeLength: 6           # Length of verification code
    expiryMinutes: 10       # Code expiry time
    maxAttempts: 5          # Max verification attempts
    
    # Features
    allowImplicitSignup: true  # Auto-create users
    devExposeCode: false       # Expose codes in dev mode
    
    # SMS provider (handled by notification plugin)
    smsProvider: "twilio"
    
    # Rate limiting
    rateLimit:
      enabled: true
      useRedis: true         # For production/distributed
      redisAddr: "localhost:6379"
      redisPassword: ""
      redisDb: 0
      
      # Per-phone limits
      sendCodePerPhone:
        window: "1m"
        max: 3
      
      # Per-IP limits
      sendCodePerIP:
        window: "1h"
        max: 20
      
      verifyPerPhone:
        window: "5m"
        max: 10
      
      verifyPerIP:
        window: "1h"
        max: 50
```

### Programmatic Configuration

```go
phonePlugin := phone.NewPlugin(
    // Code configuration
    phone.WithCodeLength(8),
    phone.WithExpiryMinutes(15),
    phone.WithMaxAttempts(3),
    
    // Features
    phone.WithAllowImplicitSignup(false),
    phone.WithDevExposeCode(true),
    
    // SMS provider
    phone.WithSMSProvider("aws_sns"),
    
    // Rate limiting
    phone.WithRateLimitSendCodePerPhone(1*time.Minute, 5),
)
```

## API Endpoints

### Send Verification Code

```http
POST /api/auth/phone/send-code
Content-Type: application/json

{
    "phone": "+12345678901"
}
```

**Response (200 OK):**
```json
{
    "status": "sent"
}
```

**Response (dev mode):**
```json
{
    "status": "sent",
    "dev_code": "123456"
}
```

**Rate Limits:**
- 3 requests per minute per phone number
- 20 requests per hour per IP address

### Verify Code

```http
POST /api/auth/phone/verify
Content-Type: application/json

{
    "phone": "+12345678901",
    "code": "123456",
    "email": "user@example.com",
    "remember": true
}
```

**Response (200 OK):**
```json
{
    "user": {
        "id": "...",
        "email": "user@example.com",
        "name": "..."
    },
    "session": {
        "id": "...",
        "token": "...",
        "expiresAt": "2025-11-21T10:00:00Z"
    },
    "token": "session_token_..."
}
```

**Rate Limits:**
- 10 attempts per 5 minutes per phone number
- 50 attempts per hour per IP address

### Sign In (Alias)

```http
POST /api/auth/phone/signin
```

Same as `/phone/verify` - provided for API consistency.

## Phone Number Format

The plugin enforces **E.164** international phone number format:

✅ **Valid:**
- `+12345678901` (US)
- `+442071838750` (UK)
- `+819012345678` (Japan)
- `+33123456789` (France)

❌ **Invalid:**
- `12345678901` (missing +)
- `+0123456789` (country code starts with 0)
- `+1-234-567-8901` (contains dashes)
- `+1 234 567 8901` (contains spaces)
- `+12` (too short - minimum 7 digits)

## Security Features

### Cryptographically Secure Codes

Uses `crypto/rand` for secure OTP generation:
```go
// Generates codes with uniform distribution
// Proper leading zero handling
// Configurable length (default: 6 digits)
```

### Rate Limiting

Multi-layer rate limiting protection:
- **Per-phone limits** - Prevent abuse of specific numbers
- **Per-IP limits** - Prevent distributed attacks
- **Redis-backed** - Distributed rate limiting for production
- **Automatic fallback** - Memory storage if Redis unavailable

### Brute Force Protection

- Maximum attempt tracking per verification code
- Automatic code invalidation after max attempts
- Audit logging of failed attempts
- Progressive delays (implemented at handler level)

### Audit Trail

Every action is logged with structured metadata:
- Code generation and delivery
- Verification attempts (success/failure)
- User creation (implicit signup)
- Session creation
- All errors and failures

## Multi-Tenancy

The plugin fully supports AuthSome's multi-tenant architecture:

- **App-level isolation** - Codes scoped to applications
- **Organization context** - Org-specific verification
- **Environment support** - Dev/staging/prod separation
- **Context propagation** - Proper tenant context throughout

## SMS Provider Integration

The plugin integrates with AuthSome's notification plugin for SMS delivery:

**Supported Providers (via notification plugin):**
- Twilio
- AWS SNS
- Vonage (Nexmo)
- MessageBird
- Custom providers

**Features:**
- Automatic provider selection
- Graceful degradation if SMS fails
- Provider-specific configuration
- Delivery status tracking

## Error Handling

The plugin uses structured error responses:

```go
// Common errors
ErrInvalidPhoneFormat  // Invalid E.164 format
ErrMissingPhone        // Phone required
ErrMissingCode         // Code required
ErrMissingEmail        // Email required
ErrCodeExpired         // Code expired or not found
ErrTooManyAttempts     // Max attempts exceeded
ErrInvalidCode         // Wrong code
```

**Error Response Format:**
```json
{
    "code": "INVALID_PHONE_FORMAT",
    "message": "invalid phone number format, must be E.164 format (e.g., +1234567890)",
    "httpStatus": 400
}
```

## Development Mode

Enable development features for testing:

```go
phonePlugin := phone.NewPlugin(
    phone.WithDevExposeCode(true),
)
```

**Features:**
- Exposes verification codes in API responses
- Useful for automated testing
- **Never enable in production**

## Testing

```bash
# Run tests
go test ./plugins/phone/...

# Run with coverage
go test ./plugins/phone/... -cover

# Run with race detection
go test ./plugins/phone/... -race
```

**Test Coverage:**
- Phone validation (E.164 format)
- Secure code generation
- Configuration options
- Request/response serialization
- Error conditions
- Rate limit configuration

## Production Deployment

### Checklist

- [ ] Set `useRedis: true` for distributed rate limiting
- [ ] Configure Redis connection parameters
- [ ] Set `devExposeCode: false`
- [ ] Configure appropriate rate limits
- [ ] Set up SMS provider credentials (via notification plugin)
- [ ] Enable audit logging
- [ ] Configure max attempts appropriately
- [ ] Test phone number validation
- [ ] Monitor SMS delivery rates
- [ ] Set up alerts for rate limit breaches

### Monitoring

Key metrics to monitor:
- Verification code send rate
- Verification success rate
- Failed verification attempts
- SMS delivery failures
- Rate limit hits
- Average verification time

## Examples

### Basic Phone Auth Flow

```go
// 1. Send verification code
client.POST("/api/auth/phone/send-code", {
    "phone": "+12345678901"
})

// 2. User receives SMS with code

// 3. Verify code and create session
client.POST("/api/auth/phone/verify", {
    "phone": "+12345678901",
    "code": "123456",
    "email": "user@example.com",
    "remember": true
})
```

### Custom Configuration

```go
phonePlugin := phone.NewPlugin(
    // Shorter code for better UX
    phone.WithCodeLength(4),
    
    // Longer expiry for email-based flow
    phone.WithExpiryMinutes(20),
    
    // More lenient attempts
    phone.WithMaxAttempts(10),
    
    // Disable implicit signup
    phone.WithAllowImplicitSignup(false),
    
    // Custom SMS provider
    phone.WithSMSProvider("custom_provider"),
)
```

## Troubleshooting

### SMS Not Sending

1. Check notification plugin is installed and configured
2. Verify SMS provider credentials
3. Check audit logs for `phone_sms_send_failed` events
4. Verify phone number format (E.164)

### Rate Limit Issues

1. Check Redis connection if `useRedis: true`
2. Review rate limit configuration
3. Check audit logs for rate limit events
4. Consider adjusting limits for your use case

### Verification Failures

1. Check code hasn't expired
2. Verify max attempts not exceeded
3. Confirm phone number matches exactly
4. Check audit logs for detailed error info

## License

See main AuthSome license.

## Support

For issues and questions, see main AuthSome documentation.

---

**Version**: 1.0.0 (Enterprise Grade)  
**Status**: Production Ready ✅

