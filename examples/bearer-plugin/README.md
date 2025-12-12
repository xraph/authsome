# Bearer Plugin Example

This example demonstrates how to use the Bearer authentication plugin with AuthSome.

## What is the Bearer Plugin?

The Bearer plugin enables Bearer token authentication via the `Authorization: Bearer <token>` header. It implements a pluggable authentication strategy that:

1. Extracts bearer tokens from the Authorization header
2. Validates them as session tokens
3. Populates the authentication context with user information

## Architecture

The Bearer plugin uses AuthSome's **pluggable authentication strategy** system:

```
Request → Middleware → Try Strategies (by priority) → Auth Context
                       ├── API Key (priority 10)
                       ├── Bearer Token (priority 20) ← Bearer Plugin
                       └── Cookie Session (priority 30)
```

Strategies are tried in priority order until one successfully authenticates.

## Running the Example

```bash
# From the authsome root directory
cd examples/bearer-plugin
go run main.go
```

## Usage Flow

### 1. Create a User and Sign In

```bash
# Sign up
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }'

# Sign in to get session token
curl -X POST http://localhost:8080/api/auth/signin \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

Response:
```json
{
  "token": "abc123def456...",
  "user": {
    "id": "usr_xyz",
    "email": "user@example.com"
  }
}
```

### 2. Use Bearer Token for Authentication

```bash
# Access protected route with bearer token
curl http://localhost:8080/api/protected \
  -H "Authorization: Bearer abc123def456..."
```

Response:
```json
{
  "message": "You are authenticated via bearer token!",
  "user": "John Doe"
}
```

## Configuration

Configure the bearer plugin with functional options:

```go
bearerPlugin := bearerplugin.NewPlugin(
    // Token prefix (default: "Bearer")
    bearerplugin.WithTokenPrefix("Bearer"),
    
    // Case-sensitive matching (default: false)
    // If false, "bearer", "Bearer", "BEARER" all work
    bearerplugin.WithCaseSensitive(false),
    
    // Validate issuer (default: false)
    bearerplugin.WithValidateIssuer(false),
    
    // Required scopes (default: none)
    bearerplugin.WithRequireScopes([]string{"read", "write"}),
)
```

Or use YAML configuration:

```yaml
auth:
  bearer:
    tokenPrefix: "Bearer"
    caseSensitive: false
    validateIssuer: false
    requireScopes: []
```

## How It Works

### 1. Plugin Registration

When you register the bearer plugin:

```go
auth.RegisterPlugin(bearerPlugin)
```

The plugin's `Init()` method:
- Gets session and user services from the registry
- Creates a `BearerStrategy` instance
- Registers the strategy with AuthSome: `auth.RegisterAuthStrategy(strategy)`

### 2. Strategy Execution

During a request, the auth middleware:
1. Tries each registered strategy in priority order
2. Bearer strategy checks for `Authorization: Bearer <token>`
3. If found, validates token as a session token
4. Returns populated `AuthContext` with user info

### 3. Authentication Context

Once authenticated, your handlers can access the auth context:

```go
func myHandler(c forge.Context) error {
    ctx := c.Request().Context()
    
    // Get full auth context
    authCtx, _ := contexts.GetAuthContext(ctx)
    if !authCtx.IsAuthenticated {
        return c.JSON(401, map[string]interface{}{
            "error": "Unauthorized",
        })
    }
    
    // Access user info
    user := authCtx.User
    session := authCtx.Session
    
    return c.JSON(200, map[string]interface{}{
        "user": user.Email,
        "authenticated_via": authCtx.Method, // "session"
    })
}
```

## Benefits of Strategy Pattern

### 1. True Modularity
- Authentication methods are opt-in via plugins
- No hardcoded auth logic in core
- Clean separation of concerns

### 2. Extensibility
- Add custom authentication strategies easily
- Third-party plugins can contribute strategies
- Strategies are composable

### 3. Priority Control
- Control which auth method takes precedence
- API keys (priority 10) beat bearer tokens (priority 20)
- Customize priorities per strategy

### 4. Testability
- Test strategies independently
- Mock strategies for testing
- No coupling to middleware internals

## Creating Custom Strategies

You can create your own authentication strategies:

```go
type MyStrategy struct {
    // your fields
}

func (s *MyStrategy) ID() string {
    return "my-auth"
}

func (s *MyStrategy) Priority() int {
    return 25 // Between bearer (20) and cookie (30)
}

func (s *MyStrategy) Extract(c forge.Context) (interface{}, bool) {
    // Extract credentials from request
    token := c.Request().Header.Get("X-My-Token")
    if token == "" {
        return nil, false
    }
    return token, true
}

func (s *MyStrategy) Authenticate(ctx context.Context, credentials interface{}) (*contexts.AuthContext, error) {
    // Validate credentials and build auth context
    token := credentials.(string)
    
    // Your validation logic here
    user := validateMyToken(token)
    
    return &contexts.AuthContext{
        User:            user,
        IsAuthenticated: true,
        IsUserAuth:      true,
        Method:          contexts.AuthMethodSession,
    }, nil
}

// Register your strategy
auth.RegisterAuthStrategy(&MyStrategy{})
```

## API Reference

### Plugin Options

- `WithTokenPrefix(prefix string)` - Set bearer token prefix (default: "Bearer")
- `WithCaseSensitive(sensitive bool)` - Enable case-sensitive matching
- `WithValidateIssuer(validate bool)` - Validate token issuer
- `WithRequireScopes(scopes []string)` - Require specific scopes

### Middleware Methods

The bearer plugin also provides standalone middleware handlers if you want fine-grained control:

```go
// Populate auth context (optional)
app.Use(bearerPlugin.AuthenticateHandler(next))

// Require authentication (mandatory)
api.Use(bearerPlugin.RequireAuthHandler(next))
```

However, the **recommended approach** is to use AuthSome's global middleware, which automatically uses all registered strategies:

```go
// This will try all strategies: API key, bearer, cookie
api.Use(auth.AuthMiddleware())
```

## Comparison: Bearer vs Session Cookie

| Feature | Bearer Token | Session Cookie |
|---------|-------------|----------------|
| **Storage** | Client manages (localStorage, memory) | Browser handles automatically |
| **CSRF Protection** | Not needed (no automatic sending) | Required (SameSite, CSRF tokens) |
| **Mobile Apps** | ✅ Easy to use | ⚠️ Requires cookie management |
| **SPAs** | ✅ Explicit control | ⚠️ Cookie setup complexity |
| **Server-to-Server** | ✅ Perfect | ❌ Not suitable |
| **Priority** | 20 (medium-high) | 30 (medium) |

Both methods use the same session tokens under the hood—the only difference is the transport mechanism.

## Best Practices

### 1. Use HTTPS in Production
Bearer tokens are sensitive credentials. Always use HTTPS to prevent interception.

### 2. Short-Lived Tokens
Configure short session expiry times:

```go
auth := authsome.New(
    authsome.WithSessionExpiry(15 * time.Minute),
)
```

### 3. Refresh Token Flow
Implement token refresh to avoid frequent re-authentication:

```go
// When token expires
POST /api/auth/refresh
Authorization: Bearer <refresh_token>
```

### 4. Token Revocation
Revoke sessions when needed:

```go
// Sign out invalidates the session
DELETE /api/auth/signout
Authorization: Bearer <token>
```

### 5. Rate Limiting
Protect auth endpoints:

```go
auth := authsome.New(
    authsome.WithRateLimit(10, time.Minute), // 10 requests per minute
)
```

## Troubleshooting

### Token Not Recognized

**Problem**: Bearer token is sent but auth fails

**Check**:
1. Is the token in the correct format: `Authorization: Bearer <token>`?
2. Is the session still valid (not expired)?
3. Is the bearer plugin registered before `auth.Initialize()`?

### Strategy Not Executing

**Problem**: Bearer strategy doesn't seem to run

**Check**:
1. Is a higher-priority strategy (e.g., API key) succeeding first?
2. Check strategy registration: `auth.RegisterPlugin(bearerPlugin)`
3. Enable debug logging to see which strategies are tried

## Related Examples

- [JWT Plugin](../jwt-plugin) - Use JWT tokens instead of session tokens
- [API Key Authentication](../apikey-example) - Machine-to-machine auth
- [Multi-App](../multiapp) - Multiple apps with different auth strategies

## Learn More

- [Authentication Strategy Pattern](../../docs/auth-strategies.md)
- [Plugin Development Guide](../../docs/plugin-development.md)
- [Security Best Practices](../../docs/security.md)

