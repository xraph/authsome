# Step-Up Authentication Plugin

Enterprise-grade step-up authentication for AuthSome with context-aware verification requirements.

## Overview

The step-up authentication plugin provides **adaptive security** for high-value operations by requiring additional verification based on:

- 🛣️ **Route patterns** - Sensitive endpoints require re-authentication
- 💰 **Transaction amounts** - Higher amounts need stronger verification  
- 📦 **Resource sensitivity** - Critical resources require step-up
- ⏰ **Time-based rules** - Re-auth after inactivity periods
- 🎯 **Risk scores** - Adaptive security based on risk assessment
- 🏢 **Multi-tenancy** - Organization-scoped policies and overrides

## Key Features

### Security Levels

Four graduated security levels with configurable verification methods:

- **Low** - Basic authentication (logged in)
- **Medium** - Re-authentication within 15 minutes (password)
- **High** - Strong re-auth within 5 minutes (password + 2FA)
- **Critical** - Immediate verification (password + biometric/WebAuthn)

### Rule Types

1. **Route Rules** - Pattern-based route protection
2. **Amount Rules** - Transaction value thresholds
3. **Resource Rules** - Sensitivity-based access control
4. **Time-Based Rules** - Age-based re-authentication
5. **Context Rules** - Custom condition evaluation
6. **Risk-Based Rules** - Adaptive security levels

### User Experience

- ✅ Device remembering (24-hour default)
- ✅ Grace periods (30 seconds)
- ✅ Clear prompts and error messages
- ✅ Multiple verification methods
- ✅ Graceful degradation

### Multi-Tenancy Support

- Organization-scoped rules and policies
- Per-org configuration overrides
- Global and organization-specific rules
- User-specific policy overrides

## Installation

### 1. Register the Plugin

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/stepup"
)

// Create AuthSome instance
auth := authsome.New(
    authsome.WithDatabase(db),
    authsome.WithForgeApp(app),
)

// Register step-up plugin
stepupPlugin := stepup.NewPlugin(nil) // Use default config
auth.RegisterPlugin(stepupPlugin)

// Initialize
if err := auth.Initialize(ctx); err != nil {
    log.Fatal(err)
}

// Mount routes
auth.Mount(router, "/api/auth")
```

### 2. Run Migrations

Migrations run automatically during plugin initialization, creating these tables:

- `stepup_verifications` - Verification records
- `stepup_requirements` - Pending requirements
- `stepup_remembered_devices` - Trusted devices
- `stepup_attempts` - Verification attempts
- `stepup_policies` - Organization policies
- `stepup_audit_logs` - Audit trail

## Configuration

### Basic Configuration

```go
config := &stepup.Config{
    Enabled: true,
    
    // Time windows
    MediumAuthWindow:   15 * time.Minute,
    HighAuthWindow:     5 * time.Minute,
    CriticalAuthWindow: 0, // Immediate
    
    // Verification methods per level
    LowMethods:      []stepup.VerificationMethod{stepup.MethodPassword},
    MediumMethods:   []stepup.VerificationMethod{stepup.MethodPassword},
    HighMethods:     []stepup.VerificationMethod{stepup.MethodPassword, stepup.MethodTOTP},
    CriticalMethods: []stepup.VerificationMethod{stepup.MethodPassword, stepup.MethodWebAuthn},
    
    // Device remembering
    RememberStepUp:   true,
    RememberDuration: 24 * time.Hour,
    
    // Risk-based adaptive security
    RiskBasedEnabled: true,
}

plugin := stepup.NewPlugin(config)
```

### Route Rules

```go
config := stepup.DefaultConfig()
config.RouteRules = []stepup.RouteRule{
    {
        Pattern:       "/api/user/email",
        Method:        "PUT",
        SecurityLevel: stepup.SecurityLevelMedium,
        Description:   "Changing email requires re-authentication",
    },
    {
        Pattern:       "/api/user/password",
        Method:        "PUT",
        SecurityLevel: stepup.SecurityLevelHigh,
        Description:   "Changing password requires strong authentication",
    },
    {
        Pattern:       "/api/payment/*",
        Method:        "POST",
        SecurityLevel: stepup.SecurityLevelMedium,
        Description:   "Payment operations require verification",
    },
}
```

### Amount Rules

```go
config.AmountRules = []stepup.AmountRule{
    {
        MinAmount:     0,
        MaxAmount:     1000,
        Currency:      "USD",
        SecurityLevel: stepup.SecurityLevelMedium,
        Description:   "Amounts under $1,000 require medium security",
    },
    {
        MinAmount:     1000,
        MaxAmount:     10000,
        Currency:      "USD",
        SecurityLevel: stepup.SecurityLevelHigh,
        Description:   "Amounts $1,000-$10,000 require high security",
    },
    {
        MinAmount:     10000,
        MaxAmount:     0, // Unlimited
        Currency:      "USD",
        SecurityLevel: stepup.SecurityLevelCritical,
        Description:   "Amounts over $10,000 require critical security",
    },
}
```

### Resource Rules

```go
config.ResourceRules = []stepup.ResourceRule{
    {
        ResourceType:  "user",
        Action:        "delete",
        SecurityLevel: stepup.SecurityLevelHigh,
        Sensitivity:   "high",
        Description:   "Deleting user account requires high security",
    },
    {
        ResourceType:  "settings",
        Action:        "update",
        SecurityLevel: stepup.SecurityLevelMedium,
        Sensitivity:   "medium",
        Description:   "Updating security settings requires verification",
    },
}
```

### Organization-Specific Rules

Rules can be scoped to specific organizations:

```go
config.RouteRules = []stepup.RouteRule{
    {
        Pattern:       "/api/admin/*",
        Method:        "POST",
        SecurityLevel: stepup.SecurityLevelHigh,
        OrgID:         "org_enterprise_123", // Only for this org
        Description:   "Enterprise org requires high security for admin operations",
    },
}
```

## Usage Examples

### 1. Route-Based Protection

Automatically enforce step-up for specific routes:

```go
// Apply to all routes (checks configured rules)
router.Use(stepupPlugin.Middleware().RequireForRoute())

// Or apply to specific route groups
adminRoutes := router.Group("/api/admin")
adminRoutes.Use(stepupPlugin.Middleware().RequireLevel(stepup.SecurityLevelHigh))
adminRoutes.POST("/users/delete", deleteUserHandler)
```

### 2. Amount-Based Protection

For financial transactions:

```go
func transferHandler(c forge.Context) error {
    var req TransferRequest
    c.BindJSON(&req)
    
    // Apply amount-based middleware
    middleware := stepupPlugin.Middleware().RequireForAmount(req.Amount, req.Currency)
    
    // Check if step-up is required
    if err := middleware(func(c forge.Context) error {
        // Process transfer
        return processTransfer(c, req)
    })(c); err != nil {
        return err
    }
    
    return nil
}
```

### 3. Resource-Based Protection

For sensitive resource operations:

```go
func deleteAccountHandler(c forge.Context) error {
    // Require high security for account deletion
    middleware := stepupPlugin.Middleware().RequireForResource("user", "delete")
    
    return middleware(func(c forge.Context) error {
        // Delete account
        return deleteAccount(c)
    })(c)
}
```

### 4. Manual Evaluation

For custom logic:

```go
func sensitiveOperation(c forge.Context) error {
    service := stepupPlugin.Service()
    
    evalCtx := &stepup.EvaluationContext{
        UserID:    getUserID(c),
        OrgID:     getOrgID(c),
        Route:     c.Request().URL.Path,
        Method:    c.Request().Method,
        Amount:    15000.00,
        Currency:  "USD",
        IP:        c.Request().RemoteAddr,
        UserAgent: c.Request().Header.Get("User-Agent"),
        DeviceID:  getDeviceID(c),
    }
    
    result, err := service.EvaluateRequirement(c.Request().Context(), evalCtx)
    if err != nil {
        return c.JSON(500, map[string]interface{}{"error": "Evaluation failed"})
    }
    
    if result.Required {
        return c.JSON(403, map[string]interface{}{
            "error":           "Step-up required",
            "requirement_id":  result.RequirementID,
            "challenge_token": result.ChallengeToken,
            "allowed_methods": result.AllowedMethods,
        })
    }
    
    // Continue with operation
    return performOperation(c)
}
```

### 5. Verification Flow

Client-side verification process:

```javascript
// 1. Attempt operation
const response = await fetch('/api/user/email', {
  method: 'PUT',
  body: JSON.stringify({ email: 'new@example.com' })
});

// 2. Check if step-up is required
if (response.status === 403) {
  const data = await response.json();
  
  if (data.error === 'Step-up authentication required') {
    // Show step-up dialog
    showStepUpDialog({
      level: data.security_level,
      reason: data.reason,
      methods: data.allowed_methods,
      challengeToken: data.challenge_token
    });
  }
}

// 3. User enters verification (e.g., password)
async function verifyStepUp(password, challengeToken, rememberDevice) {
  const response = await fetch('/api/auth/stepup/verify', {
    method: 'POST',
    body: JSON.stringify({
      challenge_token: challengeToken,
      method: 'password',
      credential: password,
      remember_device: rememberDevice,
      device_id: getDeviceID()
    })
  });
  
  if (response.ok) {
    // Step-up successful, retry original operation
    retryOriginalOperation();
  }
}
```

## API Endpoints

### Evaluation

**POST /stepup/evaluate**

Evaluate if step-up is required for an operation:

```bash
curl -X POST /api/auth/stepup/evaluate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "route": "/api/payment/transfer",
    "method": "POST",
    "amount": 5000,
    "currency": "USD"
  }'
```

Response:
```json
{
  "required": true,
  "security_level": "high",
  "current_level": "medium",
  "matched_rules": ["Amount: 5000.00 USD"],
  "reason": "Amounts $1,000-$10,000 require high security",
  "requirement_id": "req_abc123",
  "challenge_token": "token_xyz789",
  "allowed_methods": ["password", "totp"],
  "expires_at": "2025-11-01T12:30:00Z",
  "can_remember": true
}
```

### Verification

**POST /stepup/verify**

Verify step-up authentication:

```bash
curl -X POST /api/auth/stepup/verify \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "challenge_token": "token_xyz789",
    "method": "password",
    "credential": "user_password",
    "remember_device": true,
    "device_id": "device_123"
  }'
```

Response:
```json
{
  "success": true,
  "verification_id": "ver_def456",
  "security_level": "high",
  "expires_at": "2025-11-01T12:35:00Z",
  "device_remembered": true
}
```

### Status

**GET /stepup/status**

Get current step-up status:

```bash
curl /api/auth/stepup/status \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "enabled": true,
  "current_level": "medium",
  "pending_count": 0,
  "remembered_devices": 2,
  "remember_enabled": true,
  "risk_based_enabled": true
}
```

### Requirements

**GET /stepup/requirements/pending**

List pending step-up requirements:

```bash
curl /api/auth/stepup/requirements/pending \
  -H "Authorization: Bearer $TOKEN"
```

**GET /stepup/requirements/:id**

Get specific requirement details.

### Verifications

**GET /stepup/verifications**

List verification history with pagination:

```bash
curl "/api/auth/stepup/verifications?limit=20&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

### Remembered Devices

**GET /stepup/devices**

List remembered devices:

```bash
curl /api/auth/stepup/devices \
  -H "Authorization: Bearer $TOKEN"
```

**DELETE /stepup/devices/:id**

Forget a remembered device:

```bash
curl -X DELETE /api/auth/stepup/devices/device_123 \
  -H "Authorization: Bearer $TOKEN"
```

### Policies

**POST /stepup/policies**

Create organization-specific policy:

```bash
curl -X POST /api/auth/stepup/policies \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "High-Value Transactions",
    "description": "Require critical security for transactions over $50k",
    "enabled": true,
    "priority": 100,
    "rules": {
      "amount_threshold": 50000,
      "security_level": "critical"
    }
  }'
```

**GET /stepup/policies**

List organization policies.

**PUT /stepup/policies/:id**

Update policy.

**DELETE /stepup/policies/:id**

Delete policy.

### Audit Logs

**GET /stepup/audit**

Get audit logs with pagination:

```bash
curl "/api/auth/stepup/audit?limit=50&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

## Real-World Examples

### Example 1: Email Change

```go
// Changing email requires re-authentication
router.PUT("/api/user/email", func(c forge.Context) error {
    // Step-up middleware checks automatically
    var req struct {
        NewEmail string `json:"new_email"`
    }
    c.BindJSON(&req)
    
    // If we get here, step-up was satisfied
    return updateEmail(c, req.NewEmail)
}, stepupPlugin.Middleware().RequireLevel(stepup.SecurityLevelMedium))
```

### Example 2: Large Transfer

```go
// Transfer $10k requires biometric
func transferMoney(c forge.Context) error {
    var req TransferRequest
    c.BindJSON(&req)
    
    service := stepupPlugin.Service()
    result, _ := service.EvaluateRequirement(c.Request().Context(), &stepup.EvaluationContext{
        UserID:   getUserID(c),
        OrgID:    getOrgID(c),
        Amount:   req.Amount,
        Currency: req.Currency,
    })
    
    if result.Required {
        return c.JSON(403, map[string]interface{}{
            "error":          "Additional verification required",
            "required_level": result.SecurityLevel,
            "reason":         fmt.Sprintf("Transferring %.2f %s requires %s security", 
                req.Amount, req.Currency, result.SecurityLevel),
            "challenge_token": result.ChallengeToken,
            "allowed_methods": result.AllowedMethods,
        })
    }
    
    return processTransfer(c, req)
}
```

### Example 3: Account Deletion

```go
// Deleting account requires password + 2FA
router.DELETE("/api/user/account", 
    stepupPlugin.Middleware().RequireForResource("user", "delete"),
    func(c forge.Context) error {
        // Delete account
        return deleteUserAccount(c)
    },
)
```

## Advanced Features

### Risk-Based Adaptive Security

Enable risk-based evaluation:

```go
config := stepup.DefaultConfig()
config.RiskBasedEnabled = true
config.RiskThresholdLow = 0.3    // Risk score > 0.3 = medium security
config.RiskThresholdMedium = 0.6 // Risk score > 0.6 = high security
config.RiskThresholdHigh = 0.8   // Risk score > 0.8 = critical security
```

Then provide risk scores in evaluation:

```go
evalCtx := &stepup.EvaluationContext{
    UserID:    userID,
    OrgID:     orgID,
    RiskScore: 0.75, // High risk - will require high security
    // ... other fields
}
```

### Device Remembering

Users can remember their device for 24 hours:

```javascript
// During verification, user checks "Remember this device"
await fetch('/api/auth/stepup/verify', {
  method: 'POST',
  body: JSON.stringify({
    challenge_token: token,
    method: 'password',
    credential: password,
    remember_device: true,
    device_id: generateDeviceID(), // Persistent device identifier
    device_name: 'Chrome on MacBook Pro'
  })
});
```

### Cleanup Scheduler

Start automatic cleanup of expired records:

```go
// Start cleanup every hour
stepupPlugin.StartCleanupScheduler(1 * time.Hour)
```

### Organization Overrides

Organizations can override global rules:

```go
// Create org-specific policy via API
POST /api/auth/stepup/policies
{
  "name": "Enterprise Security Policy",
  "description": "Stricter rules for enterprise org",
  "enabled": true,
  "priority": 200, // Higher priority than global rules
  "rules": {
    "route_rules": [{
      "pattern": "/api/*",
      "security_level": "high"
    }]
  }
}
```

## Security Considerations

### Best Practices

1. **Start Conservative** - Begin with stricter rules, relax as needed
2. **Monitor Patterns** - Review audit logs for false positives
3. **User Education** - Clearly communicate why step-up is required
4. **Graceful Degradation** - Allow degraded experience if verification unavailable
5. **Device Fingerprinting** - Use robust device identification
6. **Rate Limiting** - Limit verification attempts
7. **Audit Everything** - Log all step-up events

### Integration with MFA

Step-up works alongside MFA plugin:

```go
// Register both plugins
auth.RegisterPlugin(mfaPlugin)
auth.RegisterPlugin(stepupPlugin)

// Configure step-up to require MFA for high security
config := stepup.DefaultConfig()
config.HighMethods = []stepup.VerificationMethod{
    stepup.MethodPassword,
    stepup.MethodTOTP, // Requires MFA plugin
}
```

## Performance

### Caching

Current security levels are cached to avoid repeated database queries:

- Verifications are cached until expiry
- Remembered devices checked first
- Redis recommended for distributed systems

### Indexes

All tables have optimized indexes for:
- User/org lookups
- Status queries
- Time-based queries
- Token lookups

### Cleanup

Automatic cleanup prevents table bloat:
- Expired requirements (default: 10 minutes)
- Expired verifications (default: per security level)
- Expired remembered devices (default: 24 hours)

## Monitoring

### Audit Events

All step-up events are logged:

- `stepup.required` - Step-up was required
- `stepup.initiated` - User initiated verification
- `stepup.verified` - Verification succeeded
- `stepup.failed` - Verification failed
- `stepup.bypassed` - Step-up bypassed (remembered device)
- `stepup.device_forgotten` - User forgot device

### Metrics to Track

Recommended metrics:

- Step-up requirement rate
- Verification success rate
- Average time to verify
- Remembered device usage
- Failed attempt rate
- Rule match distribution

## Troubleshooting

### Step-Up Always Required

Check current authentication level:

```bash
curl /api/auth/stepup/status
```

Possible causes:
- User session expired
- Verification expired
- Too strict rules

### Verification Failing

Check:
1. Requirement not expired
2. Correct verification method
3. Valid credentials
4. Not rate limited

### Device Not Remembered

Verify:
- `remember_device` set to true
- Consistent device_id across requests
- Device cookie not cleared
- Device not expired (24h default)

## License

This plugin is part of the AuthSome framework and follows the same license.

## Support

For issues and questions:
- GitHub Issues: https://github.com/xraph/authsome/issues
- Documentation: https://authsome.dev/docs/plugins/stepup

