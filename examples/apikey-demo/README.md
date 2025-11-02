# API Key Demo Application

This example demonstrates the API Key plugin for AuthSome, showing how to implement secure external API access with multi-tenant support.

## Features Demonstrated

- ✅ API Key plugin registration and initialization
- ✅ Multiple authentication methods (Authorization header, X-API-Key header)
- ✅ Optional vs. required authentication
- ✅ Scope-based authorization
- ✅ Permission-based authorization
- ✅ Rate limiting per API key
- ✅ Organization context injection for multi-tenancy
- ✅ Usage tracking and analytics
- ✅ Programmatic API key management

## Prerequisites

- Go 1.21+
- PostgreSQL database
- AuthSome framework

## Setup

1. **Set up database:**

```bash
# Create database
createdb authsome_dev

# Set database URL
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/authsome_dev?sslmode=disable"
```

2. **Run migrations:**

```bash
cd ../..
go run cmd/migrate/main.go up
```

3. **Run the demo:**

```bash
cd examples/apikey-demo
go run main.go
```

The server will start on `http://localhost:3000` and automatically create a demo API key.

## API Endpoints

### Public Endpoints

```bash
# Health check
curl http://localhost:3000/health

# API info
curl http://localhost:3000/

# Public API endpoint (auth optional)
curl http://localhost:3000/api/v1/public
```

### API Key Management

```bash
# Create new API key
curl -X POST http://localhost:3000/api/auth/api-keys \
  -H 'Content-Type: application/json' \
  -d '{
    "org_id": "demo_org",
    "user_id": "demo_user",
    "name": "My API Key",
    "scopes": ["users:read", "users:write"],
    "rate_limit": 1000
  }'

# List API keys
curl 'http://localhost:3000/api/auth/api-keys?org_id=demo_org&user_id=demo_user'

# Get specific key
curl 'http://localhost:3000/api/auth/api-keys/KEY_ID?org_id=demo_org&user_id=demo_user'

# Update API key
curl -X PUT 'http://localhost:3000/api/auth/api-keys/KEY_ID?org_id=demo_org&user_id=demo_user' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Updated Key Name",
    "rate_limit": 2000
  }'

# Rotate API key
curl -X POST http://localhost:3000/api/auth/api-keys/KEY_ID/rotate \
  -H 'Content-Type: application/json' \
  -d '{
    "org_id": "demo_org",
    "user_id": "demo_user"
  }'

# Delete API key
curl -X DELETE 'http://localhost:3000/api/auth/api-keys/KEY_ID?org_id=demo_org&user_id=demo_user'
```

### Protected Endpoints (Require API Key)

Use the API key shown in the console output after starting the server.

#### Method 1: Authorization Header with ApiKey scheme

```bash
curl -H 'Authorization: ApiKey YOUR_API_KEY_HERE' \
     http://localhost:3000/api/v1/users
```

#### Method 2: Authorization Header with Bearer scheme

```bash
curl -H 'Authorization: Bearer YOUR_API_KEY_HERE' \
     http://localhost:3000/api/v1/users
```

#### Method 3: X-API-Key Header

```bash
curl -H 'X-API-Key: YOUR_API_KEY_HERE' \
     http://localhost:3000/api/v1/users
```

### Scope-Based Endpoints

```bash
# Requires 'admin' scope
curl -H 'Authorization: ApiKey YOUR_API_KEY_HERE' \
     http://localhost:3000/api/v1/admin/users

# Requires 'resources:read' scope
curl -H 'Authorization: ApiKey YOUR_API_KEY_HERE' \
     http://localhost:3000/api/v1/resources/read

# Requires 'resources:write' scope
curl -X POST -H 'Authorization: ApiKey YOUR_API_KEY_HERE' \
     http://localhost:3000/api/v1/resources/write
```

### Permission-Based Endpoints

```bash
# Requires 'settings:write' permission
curl -X POST -H 'Authorization: ApiKey YOUR_API_KEY_HERE' \
     http://localhost:3000/api/v1/settings
```

## Expected Responses

### Successful Request

```json
{
  "message": "User list endpoint",
  "org_id": "demo_org",
  "api_key_name": "Demo API Key",
  "scopes": ["users:read", "users:write", "resources:read", "admin"],
  "user_id": "demo_user",
  "data": [
    {"id": "1", "name": "Alice", "email": "alice@example.com"},
    {"id": "2", "name": "Bob", "email": "bob@example.com"}
  ]
}
```

### Missing API Key

```json
{
  "error": "API key authentication required",
  "code": "MISSING_API_KEY"
}
```

### Insufficient Scope

```json
{
  "error": "Missing required scope: admin",
  "code": "INSUFFICIENT_SCOPE"
}
```

### Rate Limit Exceeded

```json
{
  "error": "rate limit exceeded",
  "message": "API key rate limit of 1000 requests per window exceeded"
}
```

## Configuration

The demo uses default configuration. To customize, create a config file:

```yaml
# config.yaml
auth:
  apikey:
    default_rate_limit: 1000
    max_rate_limit: 10000
    default_expiry: 8760h  # 1 year
    max_keys_per_user: 10
    max_keys_per_org: 100
    
    rate_limiting:
      enabled: true
      window: 1h
    
    # Allow API key in query param (not recommended for production)
    allow_query_param: false
```

## Code Structure

```
examples/apikey-demo/
├── main.go          # Main application
└── README.md        # This file
```

### Key Components

1. **Plugin Registration**
```go
plugin := apikeyPlugin.NewPlugin()
auth.Use(plugin)
```

2. **Middleware Application**
```go
// Optional authentication
apiV1.Use(plugin.Middleware())

// Required authentication
protected.Use(plugin.RequireAPIKey())

// Scope-based
admin.Use(plugin.RequireAPIKey("admin"))

// Permission-based
endpoint.Use(plugin.RequirePermission("settings:write"))
```

3. **Context Extraction**
```go
apiKey := apikeyPlugin.GetAPIKey(c)
orgID := apikeyPlugin.GetOrgID(c)
user := apikeyPlugin.GetUser(c)
scopes := apikeyPlugin.GetScopes(c)
```

## Testing

### Test Authentication

```bash
# Should succeed with valid key
curl -H 'Authorization: ApiKey YOUR_KEY' \
     http://localhost:3000/api/v1/users

# Should fail with invalid key
curl -H 'Authorization: ApiKey invalid_key' \
     http://localhost:3000/api/v1/users
```

### Test Scopes

```bash
# Should succeed if key has 'admin' scope
curl -H 'Authorization: ApiKey YOUR_KEY' \
     http://localhost:3000/api/v1/admin/users

# Create a key without 'admin' scope and test - should fail
```

### Test Rate Limiting

```bash
# Run in a loop to test rate limits
for i in {1..1100}; do
  curl -H 'Authorization: ApiKey YOUR_KEY' \
       http://localhost:3000/api/v1/users
done
# Should start failing after 1000 requests in the time window
```

## Multi-Tenancy

The API key plugin automatically integrates with multi-tenancy:

1. Each API key is scoped to an organization (`org_id`)
2. Organization context is automatically injected into requests
3. All operations respect organization boundaries
4. User belongs to the key's organization

```go
// Organization ID is available in handlers
orgID := apikeyPlugin.GetOrgID(c)

// Use it for scoped queries
users, err := userService.ListByOrganization(ctx, orgID, limit, offset)
```

## Production Considerations

### Security

1. **HTTPS Only**: Always use HTTPS in production
2. **Secure Storage**: Store keys in secrets manager (Vault, AWS Secrets Manager)
3. **Rotation**: Implement regular key rotation policies
4. **IP Whitelisting**: Enable for server-to-server keys
5. **Monitoring**: Set up alerts for suspicious activity

### Performance

1. **Rate Limiting**: Tune limits based on usage patterns
2. **Caching**: API key verification results can be cached
3. **Connection Pooling**: Configure database connection pool
4. **Load Balancing**: Distribute API requests across instances

### Observability

1. **Logging**: Log API key usage (sanitize keys in logs)
2. **Metrics**: Track request count, error rates, rate limit hits
3. **Tracing**: Use distributed tracing for API calls
4. **Alerting**: Alert on rate limit violations, suspicious patterns

## Troubleshooting

### "Invalid auth instance type" Error

Ensure AuthSome is properly initialized before registering the plugin:

```go
auth := authsome.New(authsome.WithDatabase(db), authsome.WithForgeApp(app))
auth.Use(plugin)  // Plugin registration
auth.Initialize(ctx)  // Initialize after registration
```

### API Key Not Working

1. Check key format: `ak_<prefix>.<secret>`
2. Verify key is active: `active = true`
3. Check expiration: `expires_at` is in future
4. Verify scopes match requirements
5. Check rate limits haven't been exceeded

### Rate Limit Issues

Check current usage:
```bash
curl 'http://localhost:3000/api/auth/api-keys/KEY_ID?user_id=USER&org_id=ORG'
```

Response includes `usage_count` and `rate_limit` fields.

## Learn More

- [API Key Plugin Documentation](../../plugins/apikey/README.md)
- [AuthSome Documentation](https://authsome.dev/docs)
- [Multi-Tenancy Guide](https://authsome.dev/docs/guides/multi-tenancy)
- [Security Best Practices](https://authsome.dev/docs/security)

## License

Part of AuthSome - See LICENSE file in repository root.

