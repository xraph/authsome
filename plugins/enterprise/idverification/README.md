# Identity Verification (KYC) Plugin

Enterprise-grade identity verification and Know Your Customer (KYC) compliance plugin for AuthSome. Supports multiple verification providers including Onfido, Jumio, and Stripe Identity.

## Features

### Core Capabilities
- **Document Verification**: Passport, driver's license, national ID verification
- **Liveness Detection**: Facial recognition and liveness checks
- **Age Verification**: Automated age verification with configurable minimum age
- **AML/Sanctions Screening**: Check users against sanctions lists and PEP databases
- **Multi-Provider Support**: Onfido, Jumio, and Stripe Identity integrations
- **Webhook Integration**: Real-time verification status updates
- **Document Retention**: Configurable document retention policies for compliance
- **Risk Scoring**: Automated risk assessment with configurable thresholds
- **Multi-Tenancy**: Organization-scoped configurations and verification

### Compliance & Security
- **GDPR Compliant**: Built-in data retention and deletion policies
- **Audit Logging**: Complete audit trail of all verification activities
- **Encryption**: Sensitive data encryption at rest
- **Webhook Verification**: Cryptographic signature verification for webhooks
- **Rate Limiting**: Protection against abuse and excessive verification attempts
- **Data Residency**: Configurable data storage locations (US, EU, UK, Global)

### Enterprise Features
- **Manual Review**: Support for manual review of failed verifications
- **Re-verification**: Configurable re-verification workflows
- **User Blocking**: Block users from verification based on risk or compliance
- **Custom Fields**: Extensible metadata support
- **Analytics**: Comprehensive verification statistics and reporting
- **Admin API**: Full administrative control over verifications

## Installation

### 1. Install the Plugin

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/enterprise/idverification"
)

func main() {
    auth, err := authsome.New(
        authsome.WithDatabase(db),
        authsome.WithConfig(config),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Register the identity verification plugin
    plugin := idverification.NewPlugin()
    if err := auth.RegisterPlugin(plugin); err != nil {
        log.Fatal(err)
    }
    
    // Run migrations
    if err := plugin.Migrate(); err != nil {
        log.Fatal(err)
    }
}
```

### 2. Configuration

Add the following to your configuration file:

```yaml
auth:
  idverification:
    enabled: true
    defaultProvider: "onfido"  # onfido, jumio, stripe_identity
    
    # Session configuration
    sessionExpiryDuration: 24h
    verificationExpiry: 8760h  # 1 year
    
    # Required checks
    requireDocumentVerification: true
    requireLivenessDetection: true
    requireAgeVerification: false
    requireAMLScreening: false
    minimumAge: 18
    
    # Accepted documents and countries
    acceptedDocuments:
      - "passport"
      - "drivers_license"
      - "national_id"
    acceptedCountries: []  # Empty = all countries
    
    # Risk configuration
    maxAllowedRiskScore: 70  # 0-100
    autoRejectHighRisk: true
    minConfidenceScore: 80
    
    # Document retention
    retainDocuments: true
    documentRetentionPeriod: 2160h  # 90 days
    autoDeleteAfterExpiry: true
    
    # Webhooks
    webhooksEnabled: true
    webhookUrl: "https://your-app.com/webhooks/verification"
    webhookEvents:
      - "verification.completed"
      - "verification.failed"
      - "verification.expired"
    webhookSecret: "your-webhook-secret"
    webhookRetryCount: 3
    
    # Features
    enableManualReview: true
    enableReverification: true
    maxVerificationAttempts: 3
    
    # Compliance
    enableAuditLog: true
    complianceMode: "standard"  # standard, strict, custom
    gdprCompliant: true
    dataResidency: "eu"  # us, eu, uk, global
    
    # Rate limiting
    rateLimitEnabled: true
    maxVerificationsPerHour: 10
    maxVerificationsPerDay: 50
    
    # Provider configurations
    onfido:
      enabled: true
      apiToken: "your-onfido-api-token"
      region: "eu"  # us, eu, ca
      webhookToken: "your-onfido-webhook-token"
      documentCheck:
        enabled: true
        validateExpiry: true
        validateDataConsistency: true
        extractData: true
      facialCheck:
        enabled: true
        variant: "video"  # standard, video
        motionCapture: true
      includeDocumentReport: true
      includeFacialReport: true
      includeWatchlistReport: true
    
    jumio:
      enabled: false
      apiToken: "your-jumio-api-token"
      apiSecret: "your-jumio-api-secret"
      dataCenter: "us"  # us, eu, sg
      verificationType: "identity"
      enableLiveness: true
      enableAMLScreening: false
      enableExtraction: true
    
    stripeIdentity:
      enabled: false
      apiKey: "your-stripe-api-key"
      webhookSecret: "your-stripe-webhook-secret"
      requireLiveCapture: true
      allowedTypes:
        - "document"
      requireMatchingSelfie: true
```

## Middleware

The plugin provides comprehensive middleware for protecting endpoints based on verification status.

### Available Middleware

```go
// Load verification status into context (non-blocking)
plugin.Middleware().LoadVerificationStatus

// Require user to be verified
middleware.RequireVerified()

// Require specific verification level (none, basic, enhanced, full)
middleware.RequireVerificationLevel("full")

// Require specific checks
middleware.RequireDocumentVerified()
middleware.RequireLivenessVerified()
middleware.RequireAMLClear()
middleware.RequireAge(18)

// Ensure user is not blocked
middleware.RequireNotBlocked()
```

### Usage Examples

#### Protect a route requiring full verification

```go
// Get middleware from plugin
verifyMW := plugin.GetMiddleware()

// Protect endpoint
router.GET("/sensitive/endpoint", 
    verifyMW.RequireVerified(),
    handler.SensitiveOperation,
)
```

#### Require specific verification level

```go
// Enhanced verification for medium-risk operations
router.POST("/transfer", 
    verifyMW.RequireVerificationLevel("enhanced"),
    handler.Transfer,
)

// Full verification for high-risk operations
router.POST("/withdraw", 
    verifyMW.RequireVerificationLevel("full"),
    handler.Withdraw,
)
```

#### Combine multiple requirements

```go
// Financial operations require document + AML screening
router.POST("/investment", 
    verifyMW.RequireDocumentVerified(),
    verifyMW.RequireAMLClear(),
    handler.Investment,
)

// Age-restricted content
router.GET("/premium/content", 
    verifyMW.RequireAge(21),
    handler.PremiumContent,
)
```

#### Load status for conditional logic

```go
// Load status into context (non-blocking)
router.Use(plugin.Middleware().LoadVerificationStatus)

// In handler, check verification status
func (h *Handler) SomeEndpoint(c forge.Context) error {
    if idverification.IsVerified(c) {
        // User is verified, show premium features
        return showPremiumFeatures(c)
    }
    
    // User not verified, show basic features
    return showBasicFeatures(c)
}

// Or get full status
status, ok := idverification.GetVerificationStatus(c)
if ok {
    level := status.VerificationLevel
    // Conditional logic based on level
}
```

### Middleware Response Codes

| Status | Code | Meaning |
|--------|------|---------|
| 401 | AUTHENTICATION_REQUIRED | User not authenticated |
| 403 | VERIFICATION_REQUIRED | User not verified |
| 403 | VERIFICATION_NOT_FOUND | No verification status found |
| 403 | INSUFFICIENT_VERIFICATION_LEVEL | Verification level too low |
| 403 | DOCUMENT_VERIFICATION_REQUIRED | Document check not passed |
| 403 | LIVENESS_VERIFICATION_REQUIRED | Liveness check not passed |
| 403 | AML_SCREENING_REQUIRED | AML screening not done |
| 403 | AML_SCREENING_FAILED | AML screening found issues |
| 403 | AGE_VERIFICATION_REQUIRED | Age verification needed |
| 403 | USER_BLOCKED | User blocked from verification |
| 403 | REVERIFICATION_REQUIRED | Re-verification needed |

## Usage

### Create a Verification Session

```go
// Create a verification session for a user
session, err := service.CreateVerificationSession(ctx, &idverification.CreateSessionRequest{
    UserID:         "user_123",
    OrganizationID: "org_456",
    Provider:       "onfido",  // Optional, uses default if not specified
    RequiredChecks: []string{"document", "liveness"},
    SuccessURL:     "https://your-app.com/verification/success",
    CancelURL:      "https://your-app.com/verification/cancel",
    Metadata: map[string]interface{}{
        "purpose": "account_verification",
    },
    IPAddress: "1.2.3.4",
    UserAgent: "Mozilla/5.0...",
})

// Redirect user to session.SessionURL to complete verification
```

### Via HTTP API

```bash
# Create verification session
curl -X POST https://your-app.com/auth/verification/sessions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "onfido",
    "requiredChecks": ["document", "liveness"],
    "successUrl": "https://your-app.com/verification/success",
    "cancelUrl": "https://your-app.com/verification/cancel"
  }'

# Get user verification status
curl -X GET https://your-app.com/auth/verification/me/status \
  -H "Authorization: Bearer YOUR_TOKEN"

# Get user verifications
curl -X GET https://your-app.com/auth/verification/me \
  -H "Authorization: Bearer YOUR_TOKEN"

# Request re-verification
curl -X POST https://your-app.com/auth/verification/me/reverify \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Document expired"
  }'
```

### Check Verification Status

```go
// Get user verification status
status, err := service.GetUserVerificationStatus(ctx, "user_123")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Verified: %v\n", status.IsVerified)
fmt.Printf("Level: %s\n", status.VerificationLevel)
fmt.Printf("Risk Level: %s\n", status.OverallRiskLevel)
fmt.Printf("Document Verified: %v\n", status.DocumentVerified)
fmt.Printf("Liveness Verified: %v\n", status.LivenessVerified)
fmt.Printf("AML Clear: %v\n", status.AMLClear)
```

### Handle Webhooks

```go
// Webhook endpoint is automatically registered at:
// POST /auth/verification/webhook/:provider

// The handler will:
// 1. Verify the webhook signature
// 2. Parse the webhook payload
// 3. Update the verification record
// 4. Update user verification status
// 5. Send internal webhooks to your app
```

### Admin Operations

```go
// Block a user from verification
err := service.BlockUser(ctx, "user_123", "org_456", "Suspicious activity detected")

// Unblock a user
err := service.UnblockUser(ctx, "user_123", "org_456")

// Get verification stats
stats, err := repo.GetVerificationStats(ctx, "org_456", from, to)
fmt.Printf("Total: %d\n", stats.TotalVerifications)
fmt.Printf("Successful: %d\n", stats.SuccessfulVerifications)
fmt.Printf("High Risk: %d\n", stats.HighRiskCount)
```

## API Reference

### Endpoints

#### User Endpoints

- `POST /verification/sessions` - Create a verification session
- `GET /verification/sessions/:id` - Get session details
- `GET /verification/me` - Get user's verifications
- `GET /verification/me/status` - Get user's verification status
- `POST /verification/me/reverify` - Request re-verification
- `GET /verification/:id` - Get specific verification

#### Admin Endpoints

- `POST /verification/admin/users/:userId/block` - Block user from verification
- `POST /verification/admin/users/:userId/unblock` - Unblock user
- `GET /verification/admin/users/:userId/status` - Get any user's status
- `GET /verification/admin/users/:userId/verifications` - Get any user's verifications

#### Webhook Endpoints

- `POST /verification/webhook/:provider` - Receive provider webhooks

## Provider Integration

### Onfido

1. Sign up at [Onfido](https://onfido.com)
2. Get your API token from the dashboard
3. Configure webhook URL in Onfido dashboard
4. Copy webhook token for signature verification

```yaml
onfido:
  enabled: true
  apiToken: "test_xxxxx"
  region: "eu"
  webhookToken: "webhook_xxxxx"
```

### Jumio

1. Sign up at [Jumio](https://www.jumio.com)
2. Get API credentials (token + secret)
3. Configure callback URL
4. Set data center region

```yaml
jumio:
  enabled: true
  apiToken: "your_token"
  apiSecret: "your_secret"
  dataCenter: "us"
```

### Stripe Identity

1. Enable Stripe Identity in your [Stripe Dashboard](https://dashboard.stripe.com)
2. Get your API key
3. Configure webhook endpoint
4. Copy webhook signing secret

```yaml
stripeIdentity:
  enabled: true
  apiKey: "sk_test_xxxxx"
  webhookSecret: "whsec_xxxxx"
```

## Verification Flow

1. **Session Creation**: Application creates a verification session
2. **User Redirect**: User is redirected to provider's verification page
3. **Document Upload**: User uploads documents and completes checks
4. **Provider Processing**: Provider verifies documents, checks liveness, screens AML
5. **Webhook Callback**: Provider sends results via webhook
6. **Status Update**: AuthSome updates verification status
7. **User Notification**: User is redirected to success/failure URL
8. **Application Check**: Application checks verification status

## Database Schema

### identity_verifications

Stores individual verification attempts.

| Column | Type | Description |
|--------|------|-------------|
| id | varchar(255) | Unique verification ID |
| user_id | varchar(255) | User being verified |
| organization_id | varchar(255) | Organization context |
| provider | varchar(50) | Provider used (onfido, jumio, etc.) |
| provider_check_id | varchar(255) | Provider's check ID |
| verification_type | varchar(50) | Type (document, liveness, age, aml) |
| status | varchar(50) | Status (pending, completed, failed, expired) |
| is_verified | boolean | Verification result |
| risk_score | int | Risk score (0-100) |
| risk_level | varchar(20) | Risk level (low, medium, high) |
| confidence_score | int | Confidence score (0-100) |

### identity_verification_documents

Stores uploaded documents.

| Column | Type | Description |
|--------|------|-------------|
| id | varchar(255) | Document ID |
| verification_id | varchar(255) | Related verification |
| document_side | varchar(20) | Side (front, back, selfie) |
| file_url | text | Encrypted storage URL |
| file_hash | varchar(64) | SHA-256 hash |
| processing_status | varchar(50) | Processing status |

### identity_verification_sessions

Tracks verification sessions.

| Column | Type | Description |
|--------|------|-------------|
| id | varchar(255) | Session ID |
| user_id | varchar(255) | User ID |
| session_url | text | Provider verification URL |
| required_checks | jsonb | Required checks |
| status | varchar(50) | Session status |
| expires_at | timestamptz | Expiration time |

### user_verification_status

Tracks overall user verification status.

| Column | Type | Description |
|--------|------|-------------|
| id | varchar(255) | Status ID |
| user_id | varchar(255) | User ID (unique) |
| is_verified | boolean | Overall verification status |
| verification_level | varchar(50) | Verification level |
| document_verified | boolean | Document check passed |
| liveness_verified | boolean | Liveness check passed |
| age_verified | boolean | Age check passed |
| aml_screened | boolean | AML screening completed |
| aml_clear | boolean | AML screening clear |
| is_blocked | boolean | User blocked from verification |

## Error Handling

The plugin defines comprehensive error types:

- `ErrVerificationNotFound` - Verification record not found
- `ErrVerificationExpired` - Verification has expired
- `ErrMaxAttemptsReached` - Maximum verification attempts exceeded
- `ErrRateLimitExceeded` - Rate limit exceeded
- `ErrHighRiskDetected` - High risk score detected
- `ErrSanctionsListMatch` - User found on sanctions list
- `ErrPEPDetected` - Politically exposed person detected
- `ErrAgeBelowMinimum` - Age below minimum requirement
- `ErrDocumentNotSupported` - Document type not supported
- `ErrCountryNotSupported` - Country not supported

## Security Considerations

1. **API Credentials**: Store provider API keys securely (use environment variables or secrets manager)
2. **Webhook Verification**: Always verify webhook signatures
3. **Data Encryption**: Sensitive data (document numbers, tokens) are encrypted
4. **Rate Limiting**: Enable rate limiting to prevent abuse
5. **GDPR Compliance**: Configure appropriate retention periods
6. **Access Control**: Use RBAC to control access to verification data
7. **Audit Logging**: Enable audit logging for compliance

## Testing

### Unit Tests

```bash
go test ./plugins/idverification/...
```

### Integration Tests

```bash
# Set test provider credentials
export ONFIDO_API_TOKEN="test_xxxxx"
export JUMIO_API_TOKEN="test_xxxxx"
export JUMIO_API_SECRET="test_xxxxx"
export STRIPE_API_KEY="sk_test_xxxxx"

# Run integration tests
go test ./plugins/idverification/... -tags=integration
```

### Mock Provider

For testing without real provider accounts:

```go
// Use mock provider
mockProvider := &MockProvider{
    SessionResponse: &ProviderSession{
        ID:  "mock_session_123",
        URL: "https://mock-provider.com/verify/mock_session_123",
    },
}

service.providers["mock"] = mockProvider
```

## Use Cases

### Fintech Applications
- KYC compliance for banking and financial services
- Age verification for investment platforms
- AML screening for cryptocurrency exchanges

### Healthcare
- Patient identity verification
- Telemedicine identity confirmation
- Prescription validation

### Age-Restricted Content
- Alcohol/tobacco sales verification
- Adult content access control
- Gaming and gambling platforms

### Regulated Industries
- Securities trading platforms
- Insurance applications
- Real estate transactions

## Best Practices

1. **Choose the Right Provider**: Each provider has strengths
   - **Onfido**: Global coverage, excellent document support
   - **Jumio**: Strong in US market, good liveness detection
   - **Stripe Identity**: Easy integration, good for existing Stripe users

2. **Configure Appropriate Checks**: Balance security and user experience
   - Document verification: Essential for most use cases
   - Liveness detection: Prevents photo attacks
   - AML screening: Required for regulated industries
   - Age verification: Specific use cases only

3. **Set Reasonable Thresholds**: Don't make verification too strict
   - Risk score: 70 is a good starting point
   - Confidence score: 80 is reasonable
   - Max attempts: 3-5 attempts recommended

4. **Handle Failures Gracefully**: Provide clear error messages and support
   - Manual review for edge cases
   - Clear rejection reasons
   - Support contact information

5. **Monitor and Optimize**: Track verification metrics
   - Completion rates
   - Failure reasons
   - Provider performance
   - User feedback

## Troubleshooting

### Common Issues

**Verification Fails Immediately**
- Check provider API credentials
- Verify webhook configuration
- Check rate limits

**Webhook Not Received**
- Verify webhook URL is publicly accessible
- Check webhook secret configuration
- Review webhook logs in provider dashboard

**High Failure Rate**
- Review risk score thresholds
- Check document type requirements
- Verify country restrictions

**Session Expired**
- Increase session expiry duration
- Send reminder emails before expiry
- Allow re-verification

## Support

For issues or questions:
1. Check the [AuthSome Documentation](https://docs.authsome.dev)
2. Review provider documentation (Onfido, Jumio, Stripe)
3. Open an issue on GitHub
4. Contact support@authsome.dev

## License

This plugin is part of the AuthSome project and is licensed under the MIT License.

