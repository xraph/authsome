# Email Verification Plugin

The Email Verification plugin provides complete email verification workflow for AuthSome applications. It handles sending verification emails, validating tokens, and marking users as verified.

## Features

- **Automatic Verification Emails**: Optionally send verification emails after user signup
- **Secure Token Generation**: Cryptographically secure random tokens
- **Rate Limiting**: Prevents abuse with configurable limits (default: 3 per hour)
- **Token Expiry**: Configurable expiration time (default: 24 hours)
- **Auto-Login**: Optionally create session after successful verification
- **Resend Support**: Users can request new verification emails
- **Status Checking**: Check verification status for authenticated users
- **Notification Integration**: Uses AuthSome notification system for email delivery

## Installation

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/emailverification"
)

func main() {
    // Create AuthSome instance
    auth := authsome.New(/* ... */)
    
    // Register email verification plugin
    emailVerif := emailverification.NewPlugin(
        emailverification.WithAutoSendOnSignup(true),
        emailverification.WithExpiryHours(24),
        emailverification.WithMaxResendPerHour(3),
    )
    
    auth.RegisterPlugin(emailVerif)
}
```

## Configuration

### Via Code (Functional Options)

```go
emailVerif := emailverification.NewPlugin(
    emailverification.WithTokenLength(32),              // Token length in bytes
    emailverification.WithExpiryHours(24),              // Token expiry time
    emailverification.WithMaxResendPerHour(3),          // Rate limit
    emailverification.WithAutoSendOnSignup(true),       // Auto-send on signup
    emailverification.WithAutoLoginAfterVerify(true),   // Auto-login after verify
    emailverification.WithVerificationURL("https://myapp.com/verify"), // Frontend URL
)
```

### Via Configuration File

```yaml
auth:
  auth:
    requireEmailVerification: true  # Block sign-in until verified
    
  emailverification:
    tokenLength: 32
    expiryHours: 24
    maxResendPerHour: 3
    autoSendOnSignup: true
    autoLoginAfterVerify: true
    verificationURL: "https://myapp.com/verify"
    devExposeToken: false  # Set to true in development to see tokens
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `tokenLength` | int | 32 | Length of verification token in bytes |
| `expiryHours` | int | 24 | Token expiration time in hours |
| `maxResendPerHour` | int | 3 | Maximum resend requests per hour per user |
| `autoSendOnSignup` | bool | true | Automatically send verification email after signup |
| `autoLoginAfterVerify` | bool | true | Create session after successful verification |
| `verificationURL` | string | "" | Frontend URL template for verification links |
| `devExposeToken` | bool | false | Expose token in response (development only) |

## API Endpoints

### Verify Email

Verifies an email address using a token from the verification link.

```http
GET /api/auth/email-verification/verify?token={token}
```

**Response (200 OK):**
```json
{
  "success": true,
  "user": {
    "id": "abc123",
    "email": "user@example.com",
    "emailVerified": true,
    "emailVerifiedAt": "2024-12-13T15:45:00Z"
  },
  "session": {
    "id": "session123",
    "token": "session_token_abc",
    "expiresAt": "2024-12-14T15:45:00Z"
  },
  "token": "session_token_abc"
}
```

**Errors:**
- `404` - Token not found or invalid
- `410` - Token expired or already used
- `400` - Email already verified

### Resend Verification Email

Requests a new verification email to be sent.

```http
POST /api/auth/email-verification/resend
Content-Type: application/json

{
  "email": "user@example.com"
}
```

**Response (200 OK):**
```json
{
  "status": "sent"
}
```

**Errors:**
- `404` - User not found
- `400` - Email already verified
- `429` - Rate limit exceeded (too many requests)

### Send Verification Email

Manually sends a verification email (useful for admin tools).

```http
POST /api/auth/email-verification/send
Content-Type: application/json

{
  "email": "user@example.com"
}
```

**Response (200 OK):**
```json
{
  "status": "sent",
  "devToken": "abc123xyz789"  // Only in dev mode
}
```

### Check Verification Status

Returns the email verification status for the current authenticated user.

```http
GET /api/auth/email-verification/status
Cookie: session_token=...
```

**Response (200 OK):**
```json
{
  "emailVerified": true,
  "emailVerifiedAt": "2024-12-13T15:45:00Z"
}
```

## User Flow

### 1. Sign Up Flow

```
User signs up → User created (EmailVerified=false) → 
Verification email sent automatically → User receives email
```

### 2. Verification Flow

```
User clicks link in email → GET /verify?token=xyz → 
Token validated → User marked as verified → 
Optional: Session created (auto-login) → Success
```

### 3. Sign In Flow

```
User attempts sign in → Password validated → 
Check EmailVerified → If false: Reject with ErrEmailNotVerified → 
User must verify email first
```

## Integration with Auth Service

The plugin works seamlessly with AuthSome's core authentication:

```go
// When RequireEmailVerification is enabled in auth config
auth.Config{
    RequireEmailVerification: true,
}

// Sign-in will automatically check if user is verified
// and reject with types.ErrEmailNotVerified if not verified
```

## Email Templates

The plugin uses the notification system's built-in `auth.verify_email` template with these variables:

```go
{
    "userName":        "John Doe",
    "verificationURL": "https://myapp.com/verify?token=abc123",
    "code":            "abc123",  // For manual entry
    "appName":         "MyApp"
}
```

## Frontend Implementation

### React Example

```typescript
// Verification page component
const VerifyEmailPage = () => {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [error, setError] = useState<string>('');
  
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get('token');
    
    if (!token) {
      setStatus('error');
      setError('No verification token provided');
      return;
    }
    
    // Verify email
    fetch(`/api/auth/email-verification/verify?token=${token}`)
      .then(res => res.json())
      .then(data => {
        if (data.success) {
          setStatus('success');
          // Optionally redirect to dashboard if auto-login enabled
          if (data.session) {
            window.location.href = '/dashboard';
          }
        } else {
          setStatus('error');
          setError(data.error || 'Verification failed');
        }
      })
      .catch(() => {
        setStatus('error');
        setError('Network error');
      });
  }, []);
  
  if (status === 'loading') return <div>Verifying...</div>;
  if (status === 'success') return <div>Email verified successfully!</div>;
  return <div>Error: {error}</div>;
};
```

### Resend Verification

```typescript
const resendVerification = async (email: string) => {
  const response = await fetch('/api/auth/email-verification/resend', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  });
  
  if (response.ok) {
    alert('Verification email sent!');
  } else {
    const data = await response.json();
    alert(data.error || 'Failed to send email');
  }
};
```

## Security Considerations

1. **Secure Token Generation**: Uses `crypto/rand` for cryptographically secure tokens
2. **Rate Limiting**: Prevents abuse with max 3 requests per hour per user
3. **Token Expiry**: Tokens expire after 24 hours by default
4. **One-Time Use**: Tokens are marked as used and cannot be reused
5. **HTTPS**: Verification links should always use HTTPS in production
6. **Email Ownership**: Only sends to registered email addresses

## Error Handling

The plugin defines specific error codes:

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `TOKEN_NOT_FOUND` | 404 | Token not found or invalid |
| `TOKEN_EXPIRED` | 410 | Token has expired |
| `TOKEN_USED` | 410 | Token already used |
| `ALREADY_VERIFIED` | 400 | Email already verified |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `USER_NOT_FOUND` | 404 | User not found |

## Testing

### Development Mode

Enable `devExposeToken` in development to receive tokens in API responses:

```yaml
emailverification:
  devExposeToken: true
```

This allows testing without email delivery.

### Example Test

```go
func TestEmailVerification(t *testing.T) {
    // Setup
    plugin := emailverification.NewPlugin(
        emailverification.WithDevExposeToken(true),
    )
    
    // Send verification
    resp := sendVerification("test@example.com")
    token := resp.DevToken
    
    // Verify
    verifyResp := verify(token)
    assert.True(t, verifyResp.Success)
    assert.True(t, verifyResp.User.EmailVerified)
}
```

## Database Schema

Uses the existing `verifications` table:

```sql
CREATE TABLE verifications (
    id VARCHAR(20) PRIMARY KEY,
    app_id VARCHAR(20) NOT NULL,
    user_id VARCHAR(20) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,  -- 'email' for this plugin
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Cleanup

The plugin provides a method to clean up expired tokens:

```go
// Run periodically (e.g., daily cron job)
count, err := plugin.Service().CleanupExpiredTokens(ctx)
if err != nil {
    log.Printf("Failed to cleanup: %v", err)
} else {
    log.Printf("Cleaned up %d expired tokens", count)
}
```

## Troubleshooting

### Email Not Received

1. Check notification plugin is configured
2. Verify email provider settings
3. Check spam folder
4. Enable `devExposeToken` to test without email

### Token Invalid

1. Check token hasn't expired (default: 24 hours)
2. Verify token hasn't been used already
3. Ensure token is complete (no truncation)

### Rate Limit Hit

1. Default: 3 requests per hour per user
2. Increase `maxResendPerHour` if needed
3. Wait before retrying

## Best Practices

1. **Always use HTTPS** for verification links in production
2. **Set appropriate expiry** - Balance security vs. user experience
3. **Monitor rate limits** - Adjust based on legitimate user patterns
4. **Graceful degradation** - Don't block signup if email fails to send
5. **Clear error messages** - Help users understand what went wrong
6. **Provide resend option** - Make it easy to get a new verification email

## License

Part of AuthSome - see main LICENSE file.
