# Backup Authentication & Recovery Plugin

Enterprise-grade backup authentication and account recovery system with multiple verification methods.

## Overview

The Backup Authentication plugin provides comprehensive account recovery mechanisms to ensure users can regain access to their accounts even when they lose their primary authentication credentials (devices, passwords, etc.).

## Features

### Core Recovery Methods

1. **Recovery Codes** (10+ one-time use codes)
2. **Security Questions** (customizable challenge questions)
3. **Trusted Contacts** (emergency contact verification)
4. **Email Verification** (code-based verification)
5. **SMS Verification** (code-based verification)
6. **Video Verification** (live video session with admin)
7. **Document Verification** (ID upload with OCR/AI verification)

### Multi-Step Recovery Flows

- Risk-based step requirements (low, medium, high risk)
- Configurable minimum steps
- User choice between available methods
- Session-based recovery process

### Security Features

- **Risk Assessment**: Analyze device, location, IP, velocity, history
- **Rate Limiting**: Prevent brute-force recovery attempts
- **Admin Review**: Optional manual approval for high-risk recoveries
- **Audit Trail**: Immutable logs of all recovery attempts
- **Session Expiry**: Time-limited recovery sessions

### Multi-Tenancy Support

- Organization-scoped configurations
- Per-organization recovery settings
- Tenant isolation for all recovery data

## Installation

```go
import (
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/enterprise/backupauth"
)

func main() {
	auth := authsome.New()
	
	// Register plugin
	plugin := backupauth.NewPlugin()
	auth.RegisterPlugin(plugin)
	
	// Configure providers (optional)
	plugin.SetEmailProvider(myEmailProvider)
	plugin.SetSMSProvider(mySMSProvider)
	plugin.SetDocumentProvider(myDocProvider)
	
	auth.Mount("/auth", app)
}
```

## Configuration

### YAML Configuration

```yaml
auth:
  backupauth:
    enabled: true
    
    # Recovery codes
    recoveryCodes:
      enabled: true
      codeCount: 10
      codeLength: 12
      autoRegenerate: true
      format: "alphanumeric"
      
    # Security questions
    securityQuestions:
      enabled: true
      minimumQuestions: 3
      requiredToRecover: 2
      allowCustomQuestions: true
      caseSensitive: false
      maxAttempts: 3
      lockoutDuration: 30m
      
    # Trusted contacts
    trustedContacts:
      enabled: true
      minimumContacts: 1
      maximumContacts: 5
      requiredToRecover: 1
      requireVerification: true
      verificationExpiry: 168h # 7 days
      cooldownPeriod: 1h
      
    # Email verification
    emailVerification:
      enabled: true
      codeExpiry: 15m
      codeLength: 6
      maxAttempts: 5
      
    # SMS verification
    smsVerification:
      enabled: true
      codeExpiry: 10m
      codeLength: 6
      maxAttempts: 3
      provider: "twilio"
      maxSmsPerDay: 5
      cooldownPeriod: 5m
      
    # Video verification
    videoVerification:
      enabled: false # Enterprise feature
      provider: "zoom"
      requireScheduling: true
      minScheduleAdvance: 2h
      sessionDuration: 30m
      requireLivenessCheck: true
      livenessThreshold: 0.85
      recordSessions: true
      recordingRetention: 2160h # 90 days
      requireAdminReview: true
      
    # Document verification
    documentVerification:
      enabled: false # Enterprise feature
      provider: "stripe_identity"
      acceptedDocuments: ["passport", "drivers_license", "national_id"]
      requireSelfie: true
      requireBothSides: true
      minConfidenceScore: 0.85
      requireManualReview: false
      storageProvider: "s3"
      storagePath: "/var/lib/authsome/backup/documents"
      retentionPeriod: 2160h # 90 days
      encryptAtRest: true
      
    # Multi-step recovery
    multiStepRecovery:
      enabled: true
      minimumSteps: 2
      lowRiskSteps: ["recovery_codes", "email_verification"]
      mediumRiskSteps: ["security_questions", "email_verification", "sms_verification"]
      highRiskSteps: ["security_questions", "trusted_contact", "video_verification"]
      allowUserChoice: true
      sessionExpiry: 30m
      allowStepSkip: false
      requireAdminApproval: false
      
    # Risk assessment
    riskAssessment:
      enabled: true
      newDeviceWeight: 0.25
      newLocationWeight: 0.20
      newIpWeight: 0.15
      velocityWeight: 0.20
      historyWeight: 0.20
      lowRiskThreshold: 30.0
      mediumRiskThreshold: 60.0
      highRiskThreshold: 80.0
      blockHighRisk: false
      requireReviewAbove: 85.0
      
    # Rate limiting
    rateLimiting:
      enabled: true
      maxAttemptsPerHour: 5
      maxAttemptsPerDay: 10
      lockoutAfterAttempts: 5
      lockoutDuration: 24h
      exponentialBackoff: true
      maxAttemptsPerIp: 20
      ipCooldownPeriod: 1h
      
    # Audit
    audit:
      enabled: true
      logAllAttempts: true
      logSuccessful: true
      logFailed: true
      immutableLogs: true
      retentionDays: 2555 # 7 years
      archiveOldLogs: true
      archiveInterval: 2160h # 90 days
      logIpAddress: true
      logUserAgent: true
      logDeviceInfo: true
      
    # Notifications
    notifications:
      enabled: true
      notifyOnRecoveryStart: true
      notifyOnRecoveryComplete: true
      notifyOnRecoveryFailed: true
      notifyAdminOnHighRisk: true
      notifyAdminOnReviewNeeded: true
      channels: ["email"]
      securityOfficerEmail: "security@example.com"
```

## Usage

### 1. Setup Recovery Methods (User Configuration)

#### Generate Recovery Codes

```bash
POST /auth/recovery-codes/generate
Authorization: Bearer <token>
Content-Type: application/json

{
  "count": 10,
  "format": "alphanumeric"
}
```

Response:
```json
{
  "codes": [
    "ABCD-1234-EFGH",
    "IJKL-5678-MNOP",
    ...
  ],
  "count": 10,
  "generatedAt": "2024-01-01T00:00:00Z",
  "warning": "Store these codes securely. Each can only be used once."
}
```

#### Setup Security Questions

```bash
POST /auth/security-questions/setup
Authorization: Bearer <token>
Content-Type: application/json

{
  "questions": [
    {
      "questionId": 1,
      "answer": "fluffy"
    },
    {
      "questionId": 4,
      "answer": "springfield elementary"
    },
    {
      "customText": "What is your favorite programming language?",
      "answer": "golang"
    }
  ]
}
```

#### Add Trusted Contact

```bash
POST /auth/trusted-contacts/add
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com",
  "relationship": "friend"
}
```

### 2. Account Recovery Flow

#### Step 1: Start Recovery

```bash
POST /auth/recovery/start
Content-Type: application/json

{
  "userId": "user_123",
  "email": "user@example.com",
  "preferredMethod": "recovery_codes"
}
```

Response:
```json
{
  "sessionId": "session_abc123",
  "status": "pending",
  "availableMethods": [
    "recovery_codes",
    "security_questions",
    "email_verification",
    "trusted_contact"
  ],
  "requiredSteps": 2,
  "completedSteps": 0,
  "expiresAt": "2024-01-01T00:30:00Z",
  "riskScore": 45.5,
  "requiresReview": false
}
```

#### Step 2: Continue with Method

```bash
POST /auth/recovery/continue
Content-Type: application/json

{
  "sessionId": "session_abc123",
  "method": "recovery_codes"
}
```

Response:
```json
{
  "sessionId": "session_abc123",
  "method": "recovery_codes",
  "currentStep": 1,
  "totalSteps": 2,
  "instructions": "Enter one of your recovery codes",
  "data": {},
  "expiresAt": "2024-01-01T00:30:00Z"
}
```

#### Step 3: Verify Recovery Code

```bash
POST /auth/recovery-codes/verify
Content-Type: application/json

{
  "sessionId": "session_abc123",
  "code": "ABCD-1234-EFGH"
}
```

Response:
```json
{
  "valid": true,
  "message": "Recovery code verified successfully"
}
```

#### Step 4: Complete Additional Steps (if required)

Repeat steps 2-3 for each required method.

#### Step 5: Complete Recovery

```bash
POST /auth/recovery/complete
Content-Type: application/json

{
  "sessionId": "session_abc123"
}
```

Response:
```json
{
  "sessionId": "session_abc123",
  "status": "completed",
  "completedAt": "2024-01-01T00:25:00Z",
  "token": "recovery_token_xyz",
  "message": "Recovery completed successfully. Use the token to reset your password."
}
```

### 3. Admin Operations

#### List Recovery Sessions

```bash
GET /auth/admin/sessions?status=pending&requiresReview=true
Authorization: Bearer <admin_token>
```

#### Approve Recovery

```bash
POST /auth/admin/sessions/{sessionId}/approve
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "notes": "Verified identity via video call"
}
```

#### Get Recovery Statistics

```bash
GET /auth/admin/stats?startDate=2024-01-01&endDate=2024-01-31
Authorization: Bearer <admin_token>
```

## Security Considerations

### Best Practices

1. **Always enable rate limiting** to prevent brute-force attacks
2. **Use multi-step recovery** for sensitive accounts
3. **Enable audit logging** and monitor recovery attempts
4. **Configure risk assessment** to detect suspicious activity
5. **Require admin review** for high-risk recoveries
6. **Store recovery codes securely** (encrypted at rest)
7. **Implement MFA** before allowing recovery code generation

### Risk Factors

The plugin assesses risk based on:
- **Device**: New or unknown device
- **Location**: Geographic anomaly
- **IP Address**: Known VPN/proxy, suspicious region
- **Velocity**: Multiple rapid attempts
- **History**: Past recovery failures

### Admin Review

Enable admin review for:
- Risk score > 85
- First recovery attempt
- Multiple failed verifications
- Document verification required
- Video verification enabled

## Provider Integration

### Email Provider

```go
type MyEmailProvider struct{}

func (p *MyEmailProvider) SendVerificationEmail(ctx context.Context, to, code string, expiresIn time.Duration) error {
	// Implement email sending
	return nil
}

func (p *MyEmailProvider) SendRecoveryNotification(ctx context.Context, to, subject, body string) error {
	// Implement notification sending
	return nil
}

// Register provider
plugin.SetEmailProvider(&MyEmailProvider{})
```

### SMS Provider

```go
type MySMSProvider struct{}

func (p *MySMSProvider) SendVerificationSMS(ctx context.Context, to, code string, expiresIn time.Duration) error {
	// Implement SMS sending via Twilio, Vonage, etc.
	return nil
}

// Register provider
plugin.SetSMSProvider(&MySMSProvider{})
```

### Document Verification Provider

```go
type MyDocumentProvider struct{}

func (p *MyDocumentProvider) VerifyDocument(ctx context.Context, req *DocumentVerificationRequest) (*DocumentVerificationResult, error) {
	// Integrate with Stripe Identity, Onfido, Jumio, etc.
	return &DocumentVerificationResult{
		VerificationID: "verify_123",
		Status: "verified",
		ConfidenceScore: 0.95,
	}, nil
}

// Register provider
plugin.SetDocumentProvider(&MyDocumentProvider{})
```

## Testing

```go
func TestRecoveryFlow(t *testing.T) {
	// Initialize test environment
	plugin := backupauth.NewPlugin()
	
	// Setup test user with recovery methods
	userID := xid.New()
	orgID := "test_org"
	
	// Generate recovery codes
	codes, err := plugin.Service().GenerateRecoveryCodes(ctx, userID, orgID, &GenerateRecoveryCodesRequest{
		Count: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 10, len(codes.Codes))
	
	// Start recovery
	session, err := plugin.Service().StartRecovery(ctx, &StartRecoveryRequest{
		UserID: userID.String(),
	})
	assert.NoError(t, err)
	assert.NotNil(t, session)
	
	// Verify recovery code
	result, err := plugin.Service().VerifyRecoveryCode(ctx, &VerifyRecoveryCodeRequest{
		SessionID: session.SessionID,
		Code: codes.Codes[0],
	})
	assert.NoError(t, err)
	assert.True(t, result.Valid)
}
```

## API Reference

See [API.md](./API.md) for complete API documentation.

## Compliance

The plugin is designed to support:
- **GDPR**: Right to data portability, audit trail
- **SOC 2**: Security controls, audit logging
- **HIPAA**: Access controls, audit trail
- **PCI DSS**: Strong authentication, audit logging

## Performance

- **Database Queries**: Optimized with indexes on user_id, organization_id, status
- **Session Storage**: In-database with automatic cleanup
- **Rate Limiting**: In-memory + Redis for distributed systems
- **Audit Logging**: Async writes to avoid blocking
- **Scalability**: Supports millions of users per organization

## Troubleshooting

### Common Issues

**Recovery session expired:**
- Increase `multiStepRecovery.sessionExpiry` in config
- User needs to start a new recovery session

**Rate limit exceeded:**
- User attempted recovery too many times
- Wait for lockout period to expire
- Admin can manually reset lockout

**Method not available:**
- User hasn't configured the recovery method
- Method is disabled in configuration
- Check `availableMethods` in start recovery response

**High risk detected:**
- Risk score exceeds threshold
- Enable admin review or video verification
- Adjust risk assessment weights in config

## Roadmap

- [ ] WebAuthn recovery (biometric fallback)
- [ ] Hardware security key recovery
- [ ] Social recovery (friends verify identity)
- [ ] Blockchain-based recovery
- [ ] AI-powered risk assessment
- [ ] Machine learning anomaly detection
- [ ] Integration with identity verification services
- [ ] Mobile app deep linking

## License

Enterprise Plugin - See LICENSE file

## Support

For issues and feature requests, please contact enterprise-support@authsome.dev

## Contributing

See CONTRIBUTING.md for guidelines on contributing to this plugin.

