# Authentication Context Example

This example demonstrates the production-grade authentication context system with pk/sk/rk API key support, following patterns from Clerk and other modern auth platforms.

## Features

### 1. **API Key Types**
- **`pk_`** - Publishable keys (frontend-safe, limited scopes)
- **`sk_`** - Secret keys (backend-only, full admin access)
- **`rk_`** - Restricted keys (backend-only, custom scopes)

### 2. **Dual Authentication Context**
- API Key authentication (identifies the app/platform)
- User Session authentication (identifies the end-user)
- Both can be present simultaneously

### 3. **Comprehensive Middleware**
- `AuthMiddleware()` - Optional, populates context
- `RequireAuth()` - Requires any authentication
- `RequireUser()` - Requires user session
- `RequireAPIKey()` - Requires API key
- `RequireScope(scope)` - Requires specific scope
- `RequireAdmin()` - Requires admin privileges

## API Key Examples

### Creating Publishable Key (Frontend)
```bash
curl -X POST http://localhost:8080/api/admin/api-keys \
  -H "Authorization: Bearer sk_prod_admin_key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Frontend Key",
    "keyType": "pk",
    "scopes": ["app:identify", "sessions:create", "users:verify"]
  }'

# Response: pk_prod_a1b2c3d4e5f6
```

### Creating Secret Key (Backend Admin)
```bash
curl -X POST http://localhost:8080/api/admin/api-keys \
  -H "Authorization: Bearer sk_prod_admin_key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Backend Admin Key",
    "keyType": "sk",
    "scopes": ["admin:full"]
  }'

# Response: sk_prod_x9y8z7w6v5u4
```

### Creating Restricted Key (Analytics Service)
```bash
curl -X POST http://localhost:8080/api/admin/api-keys \
  -H "Authorization: Bearer sk_prod_admin_key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Analytics Service Key",
    "keyType": "rk",
    "scopes": ["analytics:write", "users:read"]
  }'

# Response: rk_prod_m3n2o1p0q9r8
```

## Usage Patterns

### Pattern 1: Public Endpoint with Optional Auth
```go
app.GET("/api/public/status", handleStatus)

func handleStatus(c forge.Context) error {
    authCtx, _ := contexts.GetAuthContext(c.Request().Context())
    
    if authCtx != nil && authCtx.IsAuthenticated {
        // User is authenticated, show personalized content
    } else {
        // User is anonymous, show public content
    }
}
```

### Pattern 2: Protected User Endpoint
```go
userGroup := app.Group("/api/user")
userGroup.Use(auth.RequireUser())
{
    userGroup.GET("/me", handleGetMe)
}

func handleGetMe(c forge.Context) error {
    user, _ := contexts.RequireUser(c.Request().Context())
    return c.JSON(200, user)
}
```

### Pattern 3: Admin Endpoint
```go
adminGroup := app.Group("/api/admin")
adminGroup.Use(auth.RequireSecretKey())
adminGroup.Use(auth.RequireAdmin())
{
    adminGroup.GET("/users", handleListAllUsers)
}
```

### Pattern 4: Scoped Backend Endpoint
```go
analyticsGroup := app.Group("/api/analytics")
analyticsGroup.Use(auth.RequireAPIKey())
analyticsGroup.Use(auth.RequireScope("analytics:write"))
{
    analyticsGroup.POST("/events", handleTrackEvent)
}
```

### Pattern 5: Dual Authentication
```go
// User logged in + API key present
func handleUpdateProfile(c forge.Context) error {
    user, _ := contexts.RequireUser(c.Request().Context())
    authCtx, _ := contexts.GetAuthContext(c.Request().Context())
    
    // Check if API key also present
    if authCtx.HasAPIKey() {
        // Verify API key has required scope
        if !authCtx.HasScope("users:write") {
            return c.JSON(403, "API key lacks users:write scope")
        }
    }
    
    // Update user profile
    // ...
}
```

## Testing Requests

### 1. Public Endpoint (No Auth)
```bash
curl http://localhost:8080/api/public/status
# Works without authentication
```

### 2. User Endpoint (Session Cookie)
```bash
curl http://localhost:8080/api/user/me \
  -H "Cookie: authsome_session=sess_token_here"
# Requires valid user session
```

### 3. Backend Endpoint (API Key)
```bash
curl http://localhost:8080/api/backend/stats \
  -H "Authorization: Bearer rk_prod_analytics_key"
# Works with any API key
```

### 4. Admin Endpoint (Secret Key)
```bash
curl http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer sk_prod_admin_key"
# Requires secret key with admin:full scope
```

### 5. Dual Authentication
```bash
curl http://localhost:8080/api/user/me \
  -H "Cookie: authsome_session=sess_token_here" \
  -H "Authorization: Bearer rk_prod_backend_key"
# Both session and API key are validated
```

## Auth Context Structure

```go
type AuthContext struct {
    // API Key Authentication
    APIKey       *apikey.APIKey
    APIKeyScopes []string
    
    // User Session Authentication
    Session *session.Session
    User    *user.User
    
    // Resolved Context
    AppID          xid.ID
    EnvironmentID  xid.ID
    OrganizationID *xid.ID
    
    // Metadata
    Method          AuthMethod // none/session/apikey/both
    IsAuthenticated bool
    IsAPIKeyAuth    bool
    IsUserAuth      bool
    IPAddress       string
    UserAgent       string
}
```

## Helper Methods

```go
// Check authentication type
authCtx.HasAPIKey()
authCtx.HasSession()
authCtx.IsPublishableKey()
authCtx.IsSecretKey()
authCtx.IsRestrictedKey()

// Check scopes
authCtx.HasScope("users:write")
authCtx.HasAnyScopeOf("users:read", "users:write")
authCtx.HasAllScopesOf("data:read", "data:export")

// Check privileges
authCtx.IsAdmin()
authCtx.CanPerformAdminOp()
authCtx.CanAccessUserData(userID)
authCtx.CanAccessOrgData(orgID)

// Get effective context
authCtx.GetEffectiveOrgID()
authCtx.GetEffectiveAppID()
authCtx.GetEffectiveEnvironmentID()
```

## Security Best Practices

1. **Never expose secret keys in frontend**
   - Only use `pk_` keys in client-side code
   - Keep `sk_` and `rk_` keys on backend

2. **Use appropriate scopes**
   - Publishable keys: Only safe scopes (sessions:create, users:verify)
   - Restricted keys: Only what's needed for the service
   - Secret keys: Full admin access

3. **Validate scopes in handlers**
   - Middleware checks general authentication
   - Handlers check specific permissions

4. **Never allow API keys in query params**
   - Use Authorization header or X-API-Key header
   - Query params are logged and visible in browser history

5. **Implement rate limiting**
   - Configure per-key rate limits
   - Monitor usage patterns

## Running the Example

```bash
# From the repository root
cd examples/auth-context
go run main.go

# Server starts on :8080
```

## Related Documentation

- [Authentication Architecture](../../docs/authentication.md)
- [API Key Management](../../docs/api-keys.md)
- [Context System](../../docs/contexts.md)
- [Middleware Guide](../../docs/middleware.md)

