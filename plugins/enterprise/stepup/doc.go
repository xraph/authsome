// Package stepup provides context-aware step-up authentication for AuthSome.
//
// The step-up authentication plugin adds adaptive security to AuthSome by requiring
// additional verification for high-value or sensitive operations. Unlike always-on
// 2FA, step-up authentication is context-aware and only triggers when needed based
// on configurable rules.
//
// # Key Features
//
//   - Context-aware verification (route, amount, resource-based)
//   - Four graduated security levels (Low, Medium, High, Critical)
//   - Multiple verification methods (Password, TOTP, SMS, Email, Biometric, WebAuthn)
//   - Device remembering for trusted devices
//   - Risk-based adaptive security
//   - Multi-tenant organization-scoped policies
//   - Time-based re-authentication requirements
//   - Comprehensive audit trail
//
// # Quick Start
//
// Basic setup with default configuration:
//
//	auth := authsome.New(
//	    authsome.WithDatabase(db),
//	    authsome.WithForgeApp(app),
//	)
//
//	stepupPlugin := stepup.NewPlugin(nil) // Use default config
//	auth.RegisterPlugin(stepupPlugin)
//	auth.Initialize(ctx)
//	auth.Mount(app.Router(), "/api/auth")
//
// # Route Protection
//
// Protect routes with middleware:
//
//	// Automatic route-based protection
//	router.Use(stepupPlugin.Middleware().RequireForRoute())
//
//	// Specific security level requirement
//	adminRoutes := router.Group("/api/admin")
//	adminRoutes.Use(stepupPlugin.Middleware().RequireLevel(stepup.SecurityLevelHigh))
//	adminRoutes.POST("/users/delete", deleteUserHandler)
//
// # Manual Evaluation
//
// For custom logic, manually evaluate step-up requirements:
//
//	service := stepupPlugin.Service()
//	result, err := service.EvaluateRequirement(ctx, &stepup.EvaluationContext{
//	    UserID:    userID,
//	    OrgID:     orgID,
//	    Route:     "/api/transfer",
//	    Method:    "POST",
//	    Amount:    5000,
//	    Currency:  "USD",
//	})
//
//	if result.Required {
//	    return c.JSON(403, map[string]interface{}{
//	        "error":           "step_up_required",
//	        "challenge_token": result.ChallengeToken,
//	        "allowed_methods": result.AllowedMethods,
//	    })
//	}
//
// # Configuration
//
// Customize configuration for your needs:
//
//	config := &stepup.Config{
//	    Enabled: true,
//
//	    // Time windows for re-authentication
//	    MediumAuthWindow:   15 * time.Minute,
//	    HighAuthWindow:     5 * time.Minute,
//	    CriticalAuthWindow: 0, // Immediate
//
//	    // Route-based rules
//	    RouteRules: []stepup.RouteRule{
//	        {
//	            Pattern:       "/api/user/email",
//	            Method:        "PUT",
//	            SecurityLevel: stepup.SecurityLevelMedium,
//	            Description:   "Email changes require password",
//	        },
//	        {
//	            Pattern:       "/api/user/password",
//	            Method:        "PUT",
//	            SecurityLevel: stepup.SecurityLevelHigh,
//	            Description:   "Password changes require 2FA",
//	        },
//	    },
//
//	    // Amount-based rules
//	    AmountRules: []stepup.AmountRule{
//	        {
//	            MinAmount:     10000,
//	            MaxAmount:     0,
//	            Currency:      "USD",
//	            SecurityLevel: stepup.SecurityLevelCritical,
//	            Description:   "Large transfers require biometric",
//	        },
//	    },
//
//	    // Device remembering
//	    RememberStepUp:   true,
//	    RememberDuration: 24 * time.Hour,
//
//	    // Risk-based security
//	    RiskBasedEnabled: true,
//	}
//
//	plugin := stepup.NewPlugin(config)
//
// # Security Levels
//
// Four graduated security levels with different requirements:
//
//   - Low: Basic authentication (user is logged in)
//   - Medium: Re-authentication within time window (e.g., password)
//   - High: Strong authentication (e.g., password + TOTP)
//   - Critical: Immediate strong authentication (e.g., password + biometric)
//
// Each level can require different verification methods:
//
//	config.HighMethods = []stepup.VerificationMethod{
//	    stepup.MethodPassword,
//	    stepup.MethodTOTP,
//	}
//
// # Rule Types
//
// Five types of rules for different scenarios:
//
// 1. Route Rules - Pattern-based route protection:
//
//	RouteRule{
//	    Pattern:       "/api/admin/*",
//	    Method:        "*",
//	    SecurityLevel: SecurityLevelHigh,
//	}
//
// 2. Amount Rules - Transaction value thresholds:
//
//	AmountRule{
//	    MinAmount:     1000,
//	    MaxAmount:     10000,
//	    Currency:      "USD",
//	    SecurityLevel: SecurityLevelHigh,
//	}
//
// 3. Resource Rules - Sensitivity-based access:
//
//	ResourceRule{
//	    ResourceType:  "user",
//	    Action:        "delete",
//	    SecurityLevel: SecurityLevelHigh,
//	}
//
// 4. Time-Based Rules - Authentication age limits:
//
//	TimeBasedRule{
//	    Operation:     "admin_action",
//	    MaxAge:        5 * time.Minute,
//	    SecurityLevel: SecurityLevelHigh,
//	}
//
// 5. Context Rules - Custom condition evaluation:
//
//	ContextRule{
//	    Name:          "suspicious_activity",
//	    Condition:     "risk_score > 0.8",
//	    SecurityLevel: SecurityLevelCritical,
//	}
//
// # Multi-Tenancy
//
// Full support for organization-scoped rules and policies:
//
//	// Global rule (applies to all organizations)
//	RouteRule{
//	    Pattern:       "/api/admin/*",
//	    SecurityLevel: SecurityLevelMedium,
//	    OrgID:         "", // Empty = global
//	}
//
//	// Organization-specific rule
//	RouteRule{
//	    Pattern:       "/api/admin/*",
//	    SecurityLevel: SecurityLevelHigh,
//	    OrgID:         "org_enterprise_123",
//	}
//
// Organizations can create custom policies:
//
//	policy := &stepup.StepUpPolicy{
//	    OrgID:       "org_123",
//	    Name:        "Enterprise Security Policy",
//	    Priority:    100, // Higher priority = evaluated first
//	    Enabled:     true,
//	    Rules:       {...},
//	}
//
// # Device Remembering
//
// Users can remember trusted devices for 24 hours (configurable):
//
//	verifyReq := &stepup.VerifyRequest{
//	    ChallengeToken: challengeToken,
//	    Method:         stepup.MethodPassword,
//	    Credential:     password,
//	    RememberDevice: true,
//	    DeviceID:       "device_xyz",
//	    DeviceName:     "Chrome on MacBook Pro",
//	}
//
//	response, err := service.VerifyStepUp(ctx, verifyReq)
//
// # Risk-Based Adaptive Security
//
// Automatically adjust security requirements based on risk scores:
//
//	config := stepup.DefaultConfig()
//	config.RiskBasedEnabled = true
//	config.RiskThresholdLow = 0.3
//	config.RiskThresholdMedium = 0.6
//	config.RiskThresholdHigh = 0.8
//
//	evalCtx := &stepup.EvaluationContext{
//	    UserID:    userID,
//	    RiskScore: 0.75, // High risk - will require high security
//	}
//
// # Client-Side Integration
//
// Example JavaScript client for handling step-up flow:
//
//	async function transferMoney(amount) {
//	    const response = await fetch('/api/transfer', {
//	        method: 'POST',
//	        body: JSON.stringify({ amount })
//	    });
//
//	    if (response.status === 403) {
//	        const data = await response.json();
//
//	        if (data.error === 'step_up_required') {
//	            // Show step-up dialog
//	            const credential = await showStepUpDialog(data);
//
//	            // Verify step-up
//	            await fetch('/api/auth/stepup/verify', {
//	                method: 'POST',
//	                body: JSON.stringify({
//	                    challenge_token: data.challenge_token,
//	                    method: 'password',
//	                    credential: credential,
//	                    remember_device: true
//	                })
//	            });
//
//	            // Retry original request
//	            return transferMoney(amount);
//	        }
//	    }
//
//	    return response.json();
//	}
//
// # API Endpoints
//
// The plugin registers the following endpoints:
//
//   - POST   /stepup/evaluate          - Evaluate if step-up is required
//   - POST   /stepup/verify            - Verify step-up authentication
//   - GET    /stepup/status            - Get current step-up status
//   - GET    /stepup/requirements/:id  - Get requirement details
//   - GET    /stepup/requirements/pending - List pending requirements
//   - GET    /stepup/verifications     - List verification history
//   - GET    /stepup/devices           - List remembered devices
//   - DELETE /stepup/devices/:id       - Forget device
//   - POST   /stepup/policies          - Create organization policy
//   - GET    /stepup/policies          - List policies
//   - GET    /stepup/policies/:id      - Get policy
//   - PUT    /stepup/policies/:id      - Update policy
//   - DELETE /stepup/policies/:id      - Delete policy
//   - GET    /stepup/audit             - Get audit logs
//
// # Audit and Monitoring
//
// All step-up events are audited:
//
//   - stepup.required - Step-up was required
//   - stepup.initiated - User initiated verification
//   - stepup.verified - Verification succeeded
//   - stepup.failed - Verification failed
//   - stepup.bypassed - Step-up bypassed (remembered device)
//   - stepup.device_forgotten - User forgot device
//
// Enable audit logging in configuration:
//
//	config := stepup.DefaultConfig()
//	config.AuditEnabled = true
//	config.AuditEvents = []string{
//	    "stepup.required",
//	    "stepup.verified",
//	    "stepup.failed",
//	}
//
// # Cleanup Scheduler
//
// Start automatic cleanup of expired records:
//
//	stepupPlugin.StartCleanupScheduler(1 * time.Hour)
//
// This removes:
//   - Expired requirements (10 minute TTL)
//   - Expired verifications (per security level TTL)
//   - Expired remembered devices (24 hour default)
//
// # Performance Considerations
//
// The plugin is optimized for production use:
//
//   - Database indexes on all query patterns
//   - Minimal database queries (cached verifications)
//   - Efficient rule evaluation (O(n) where n = number of rules)
//   - Background cleanup (doesn't block requests)
//   - Stateless design (works with load balancers)
//
// # Testing
//
// The plugin includes comprehensive tests. Example:
//
//	func TestStepUpFlow(t *testing.T) {
//	    // Setup
//	    repo := new(mockRepository)
//	    service := stepup.NewService(repo, stepup.DefaultConfig(), nil)
//
//	    // Test evaluation
//	    result, err := service.EvaluateRequirement(ctx, &stepup.EvaluationContext{
//	        UserID:   "user123",
//	        Amount:   5000,
//	        Currency: "USD",
//	    })
//
//	    assert.NoError(t, err)
//	    assert.True(t, result.Required)
//	    assert.Equal(t, stepup.SecurityLevelHigh, result.SecurityLevel)
//	}
//
// # Error Handling
//
// All errors are wrapped with context:
//
//	result, err := service.EvaluateRequirement(ctx, evalCtx)
//	if err != nil {
//	    return fmt.Errorf("failed to evaluate step-up: %w", err)
//	}
//
// Error responses include helpful information:
//
//	{
//	    "error": "step_up_required",
//	    "security_level": "high",
//	    "reason": "Transferring $5,000.00 USD requires high security",
//	    "challenge_token": "token_xyz",
//	    "allowed_methods": ["password", "totp"]
//	}
//
// # Best Practices
//
//  1. Start Conservative - Begin with stricter rules, relax as needed
//  2. Monitor Patterns - Review audit logs for optimization opportunities
//  3. User Communication - Clearly explain why verification is required
//  4. Graceful Degradation - Handle failures gracefully
//  5. Test Thoroughly - Test all security levels and edge cases
//  6. Regular Review - Review and update rules periodically
//  7. Performance - Use caching for high-traffic routes
//
// # Documentation
//
// See the following files for more information:
//
//   - README.md - Comprehensive guide with API documentation
//   - EXAMPLE.md - Practical examples with client code
//   - INTEGRATION.md - Integration guide for different patterns
//   - SUMMARY.md - Implementation summary and architecture
//
// # Support
//
//   - GitHub: https://github.com/xraph/authsome
//   - Documentation: https://authsome.dev/docs/plugins/stepup
//   - Issues: https://github.com/xraph/authsome/issues
package stepup
