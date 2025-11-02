# API Key Plugin

Enterprise-grade API key authentication plugin for external client access to your AuthSome application. Provides secure, multi-tenant API key management with rate limiting, scope-based permissions, and comprehensive audit trails.

## Features

- ✅ **Multi-Tenant Support**: API keys scoped to organizations
- ✅ **Secure Storage**: Keys hashed with SHA-256, never stored in plaintext
- ✅ **Flexible Authentication**: Multiple extraction methods (header, query param)
- ✅ **Granular Permissions**: Scope and permission-based access control
- ✅ **Rate Limiting**: Per-key rate limits with configurable windows
- ✅ **Usage Tracking**: Comprehensive analytics and audit trails
- ✅ **Key Lifecycle**: Create, rotate, update, deactivate, delete operations
- ✅ **Automatic Expiration**: Optional key expiration with cleanup
- ✅ **IP Whitelisting**: Restrict keys to specific IP addresses (optional)
- ✅ **Webhook Notifications**: Alert on key events (optional)

## Installation

```go
import (
    "github.com/xraph/authsome"
    apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
)

// Initialize AuthSome
auth := authsome.New(db)

// Add API Key plugin
apikey := apikeyPlugin.NewPlugin()
auth.Use(apikey)
```

## Configuration

Configure via YAML or environment variables:

```yaml
auth:
  apikey:
    # Service configuration
    default_rate_limit: 1000      # Default requests per hour
    max_rate_limit: 10000          # Maximum allowed rate limit
    default_expiry: 8760h          # Default expiration (1 year)
    max_keys_per_user: 10          # Maximum keys per user
    max_keys_per_org: 100          # Maximum keys per organization
    key_length: 32                 # Key byte length
    
    # Authentication options
    allow_query_param: false       # Allow ?api_key=xxx (not recommended)
    
    # Rate limiting
    rate_limiting:
      enabled: true
      window: 1h                   # Rate limit time window
    
    # IP whitelisting (optional)
    ip_whitelisting:
      enabled: false
      strict_mode: false
    
    # Webhook notifications (optional)
    webhooks:
      enabled: false
      notify_on_created: true
      notify_on_rotated: true
      notify_on_deleted: true
      notify_on_rate_limit: true
      notify_on_expiring: true
      expiry_warning_days: 7
      webhook_urls:
        - "https://your-webhook-url.com/api/events"
```

## Usage

### Creating API Keys

```go
// In your handler or service
req := &apikey.CreateAPIKeyRequest{
    OrgID:       "org_abc123",
    UserID:      "user_xyz789",
    Name:        "Production API Key",
    Description: "Key for production server",
    Scopes:      []string{"users:read", "users:write"},
    Permissions: map[string]string{
        "resources:read": "all",
        "resources:write": "own",
    },
    RateLimit:   5000, // 5000 requests per hour
    ExpiresAt:   &expiryDate,
    Metadata: map[string]string{
        "environment": "production",
        "server": "api-01",
    },
}

apiKey, err := apikeyService.CreateAPIKey(ctx, req)
// apiKey.Key contains the full key: "ak_abc123_xyz789.secret_token"
// Store this securely - it won't be shown again!
```

### Protecting Routes with API Keys

#### Method 1: Global Middleware (Optional Authentication)

```go
// Apply to all routes - sets context if valid key present
router.Use(apikey.Middleware())

// Routes will have API key context if provided
router.GET("/api/v1/users", handler.ListUsers)
```

#### Method 2: Required Authentication

```go
// Require valid API key for specific routes
apiGroup := router.Group("/api/v1")
apiGroup.Use(apikey.RequireAPIKey()) // Any valid API key

apiGroup.GET("/users", handler.ListUsers)
apiGroup.POST("/users", handler.CreateUser)
```

#### Method 3: Scope-Based Authentication

```go
// Require specific scopes
router.GET("/api/v1/users", handler.ListUsers).
    Use(apikey.RequireAPIKey("users:read"))

router.POST("/api/v1/users", handler.CreateUser).
    Use(apikey.RequireAPIKey("users:write"))

router.DELETE("/api/v1/users/:id", handler.DeleteUser).
    Use(apikey.RequireAPIKey("users:delete", "admin"))
```

#### Method 4: Permission-Based Authentication

```go
router.POST("/api/v1/admin/settings", handler.UpdateSettings).
    Use(apikey.RequirePermission("admin:settings:write"))
```

### Extracting API Key Information in Handlers

```go
import apikeyPlugin "github.com/xraph/authsome/plugins/apikey"

func (h *Handler) ListUsers(c forge.Context) error {
    // Check if authenticated via API key
    if !apikeyPlugin.IsAuthenticated(c) {
        return c.JSON(401, map[string]string{
            "error": "Authentication required",
        })
    }
    
    // Get API key details
    apiKey := apikeyPlugin.GetAPIKey(c)
    orgID := apikeyPlugin.GetOrgID(c)
    user := apikeyPlugin.GetUser(c)
    scopes := apikeyPlugin.GetScopes(c)
    
    // Use organization ID for multi-tenant queries
    users, err := h.userService.ListByOrganization(ctx, orgID, limit, offset)
    
    return c.JSON(200, users)
}
```

### Client Usage

Clients can authenticate using any of these methods:

#### Method 1: Authorization Header (Recommended)

```bash
curl -H "Authorization: ApiKey ak_abc123_xyz789.secret_token" \
     https://api.example.com/api/v1/users
```

#### Method 2: Bearer Token Format

```bash
curl -H "Authorization: Bearer ak_abc123_xyz789.secret_token" \
     https://api.example.com/api/v1/users
```

#### Method 3: Custom Header

```bash
curl -H "X-API-Key: ak_abc123_xyz789.secret_token" \
     https://api.example.com/api/v1/users
```

#### Method 4: Query Parameter (If Enabled)

```bash
curl https://api.example.com/api/v1/users?api_key=ak_abc123_xyz789.secret_token
```

⚠️ **Note**: Query parameter method is disabled by default and not recommended for production use.

## Key Management Operations

### Rotate API Key

```go
req := &apikey.RotateAPIKeyRequest{
    ID:     "key_id_123",
    OrgID:  "org_abc123",
    UserID: "user_xyz789",
}

newKey, err := apikeyService.RotateAPIKey(ctx, req)
// Old key is deactivated, new key is created with same settings
```

### Update API Key

```go
active := false
req := &apikey.UpdateAPIKeyRequest{
    Name:   ptr("Updated Key Name"),
    Scopes: []string{"users:read"}, // Reduced permissions
    Active: &active, // Deactivate key
}

updated, err := apikeyService.UpdateAPIKey(ctx, keyID, userID, orgID, req)
```

### Delete API Key

```go
err := apikeyService.DeleteAPIKey(ctx, keyID, userID, orgID)
// Soft delete - key remains in database for audit
```

### Cleanup Expired Keys

```go
// Run periodically (e.g., daily cron job)
count, err := apikeyService.CleanupExpired(ctx)
// Returns number of keys cleaned up
```

## Multi-Tenancy

API keys are automatically scoped to organizations. When authenticated via API key:

- Organization context is automatically injected
- User belongs to the key's organization
- All queries should respect organization boundaries

```go
func (h *Handler) GetResource(c forge.Context) error {
    // Organization ID is automatically available
    orgID := apikeyPlugin.GetOrgID(c)
    
    // Multi-tenant decorators will respect this organization
    resource, err := h.service.FindByID(ctx, resourceID)
    // Service automatically filters by orgID from context
    
    return c.JSON(200, resource)
}
```

## Security Best Practices

### ✅ DO:

1. **Store keys securely** - Use secrets managers (HashiCorp Vault, AWS Secrets Manager)
2. **Use HTTPS only** - Never transmit keys over unencrypted connections
3. **Rotate regularly** - Implement key rotation policies (e.g., every 90 days)
4. **Minimal scopes** - Grant only required permissions
5. **Monitor usage** - Track and alert on suspicious activity
6. **Set expiration** - Use time-limited keys when possible
7. **Use IP whitelisting** - Restrict server-to-server keys to known IPs
8. **Rate limiting** - Protect against abuse

### ❌ DON'T:

1. **Don't commit keys** - Never check keys into version control
2. **Don't share keys** - One key per application/server
3. **Don't use in browsers** - API keys are for server-to-server communication
4. **Don't log keys** - Sanitize logs to prevent key exposure
5. **Don't use query params** - Prefer headers for key transmission
6. **Don't ignore expiration** - Clean up unused keys

## Rate Limiting

Per-key rate limits are enforced automatically:

```go
// Keys have individual rate limits
apiKey.RateLimit = 5000 // 5000 requests per window

// Rate limit window configured globally
config.RateLimiting.Window = time.Hour // 1 hour window

// Exceeded limit returns 429 Too Many Requests
{
  "error": "rate limit exceeded",
  "message": "API key rate limit of 5000 requests per window exceeded"
}
```

## Error Codes

| HTTP Status | Error Code | Description |
|-------------|-----------|-------------|
| 401 | `MISSING_API_KEY` | No API key provided |
| 401 | `INVALID_API_KEY` | Key not found or invalid |
| 401 | `KEY_EXPIRED` | API key has expired |
| 401 | `KEY_DEACTIVATED` | API key has been deactivated |
| 403 | `INSUFFICIENT_SCOPE` | Missing required scope |
| 403 | `INSUFFICIENT_PERMISSION` | Missing required permission |
| 429 | `RATE_LIMIT_EXCEEDED` | Rate limit exceeded |

## Monitoring & Analytics

### Usage Tracking

Each API key tracks:
- Total request count
- Last used timestamp
- Last used IP address
- Last used user agent

```go
apiKey, _ := apikeyService.GetAPIKey(ctx, keyID, userID, orgID)
fmt.Printf("Used %d times, last at %v from %s",
    apiKey.UsageCount,
    apiKey.LastUsedAt,
    apiKey.LastUsedIP,
)
```

### Audit Logs

All key operations are logged via the audit service:
- `api_key.created`
- `api_key.updated`
- `api_key.rotated`
- `api_key.deleted`

## Testing

### Mock Mode

```go
// For development/testing
config := apikey.DefaultConfig()
config.AllowQueryParam = true // Allow query params in dev

// Create test keys
testKey, _ := service.CreateAPIKey(ctx, &apikey.CreateAPIKeyRequest{
    OrgID:  "test_org",
    UserID: "test_user",
    Name:   "Test Key",
    Scopes: []string{"*"}, // All permissions
})
```

### Integration Tests

```go
func TestAPIKeyAuth(t *testing.T) {
    // Create test key
    key, err := service.CreateAPIKey(ctx, req)
    require.NoError(t, err)
    
    // Make authenticated request
    resp := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/v1/users", nil)
    req.Header.Set("Authorization", "ApiKey "+key.Key)
    
    router.ServeHTTP(resp, req)
    assert.Equal(t, 200, resp.Code)
}
```

## Migration from Other Auth Methods

### From Session-Based Auth

API keys complement session-based auth - use both:

```go
// Session auth for web UI
router.Group("/dashboard").Use(sessionAuth.Middleware())

// API key auth for external APIs
router.Group("/api/v1").Use(apikey.RequireAPIKey())
```

### From JWT

API keys are simpler for server-to-server:

```go
// JWT for mobile apps (user authentication)
router.Group("/mobile/v1").Use(jwt.Middleware())

// API keys for server integrations
router.Group("/api/v1").Use(apikey.RequireAPIKey())
```

## Advanced Features

### IP Whitelisting

```go
// Coming soon - restrict keys to specific IPs
apiKey.AllowedIPs = []string{"203.0.113.0/24", "198.51.100.42"}
```

### Webhook Notifications

```go
// Coming soon - notify on key events
config.Webhooks.Enabled = true
config.Webhooks.WebhookURLs = []string{"https://hooks.slack.com/..."}
```

### Hierarchical Scopes

```go
// Coming soon - scope inheritance
"admin:*"       → all admin scopes
"users:*"       → users:read, users:write, users:delete
"resources:read" → read-only access
```

## Troubleshooting

### Key Not Working

1. Check key format: `ak_<prefix>.<secret>`
2. Verify key is active: `apiKey.Active == true`
3. Check expiration: `apiKey.ExpiresAt`
4. Verify scopes match requirements
5. Check rate limits haven't been exceeded

### Rate Limit Issues

```bash
# Check current usage
curl -H "Authorization: ApiKey $API_KEY" \
     https://api.example.com/api-keys/$KEY_ID?user_id=$USER&org_id=$ORG

# Response includes usage_count and rate_limit
```

### Organization Context Missing

Ensure multi-tenancy plugin is loaded before API key plugin:

```go
auth.Use(multitenancy.NewPlugin())
auth.Use(apikey.NewPlugin())
```

## Support

For issues, questions, or feature requests:
- GitHub: https://github.com/xraph/authsome/issues
- Documentation: https://authsome.dev/docs/plugins/apikey

## License

Part of AuthSome - See LICENSE file in repository root.

