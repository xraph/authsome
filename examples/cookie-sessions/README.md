# Cookie Sessions Example

This example demonstrates how to use AuthSome with cookie-based session management.

## Features

- ✅ Automatic cookie setting on authentication
- ✅ Global cookie configuration
- ✅ Per-app cookie customization
- ✅ Secure cookie attributes (HttpOnly, Secure, SameSite)
- ✅ Auto-detection of HTTPS for Secure flag

## Running the Example

```bash
cd examples/cookie-sessions
go run main.go
```

## Testing Cookie Authentication

### 1. Sign Up (Creates User + Sets Cookie)

```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!",
    "name": "Test User"
  }' \
  -c cookies.txt
```

### 2. Sign In (Authenticates + Sets Cookie)

```bash
curl -X POST http://localhost:8080/auth/signin \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }' \
  -c cookies.txt
```

### 3. Access Protected Route (Using Cookie)

```bash
curl http://localhost:8080/api/me \
  -b cookies.txt
```

## Configuration

### Global Cookie Configuration

Edit `config.yaml`:

```yaml
auth:
  sessionCookie:
    enabled: true           # Enable/disable cookie setting
    name: "authsome_session" # Cookie name
    path: "/"              # Cookie path
    httpOnly: true         # HttpOnly flag (recommended: true)
    sameSite: "Lax"       # "Strict", "Lax", or "None"
    secure: true          # HTTPS only (auto-detects if not set)
    domain: ".example.com" # Cookie domain (optional)
    maxAge: 86400         # Max age in seconds (optional)
```

### Per-App Cookie Configuration

You can override cookie settings for specific apps via the API:

```bash
# Get current cookie config for an app
curl http://localhost:8080/auth/apps/{appId}/cookie-config \
  -H "Authorization: Bearer $TOKEN"

# Update cookie config for an app
curl -X PUT http://localhost:8080/auth/apps/{appId}/cookie-config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "enabled": true,
    "name": "custom_session",
    "path": "/",
    "httpOnly": true,
    "secure": true,
    "sameSite": "Strict",
    "domain": ".example.com"
  }'

# Delete app-specific config (revert to global)
curl -X DELETE http://localhost:8080/auth/apps/{appId}/cookie-config \
  -H "Authorization: Bearer $TOKEN"
```

## Cookie Security Best Practices

1. **HttpOnly**: Always set to `true` to prevent JavaScript access
2. **Secure**: Set to `true` in production (HTTPS only)
3. **SameSite**: Use `"Strict"` or `"Lax"` to prevent CSRF attacks
4. **Domain**: Set appropriately for subdomain sharing
5. **Path**: Restrict to necessary paths only

## Token-Based vs Cookie-Based

AuthSome supports both patterns simultaneously:

- **Cookie-Based**: Browser applications, traditional web apps
- **Token-Based**: Mobile apps, SPAs with localStorage, API clients

When cookies are enabled, both the token (in JSON response) and the cookie are provided, allowing clients to choose their preferred method.

## Debugging

To see cookies in browser DevTools:
1. Open Application/Storage tab
2. Navigate to Cookies → http://localhost:8080
3. Look for `authsome_session` cookie

## Learn More

- [Cookie Sessions Documentation](../../docs/COOKIE_SESSIONS.md)
- [Session Management](https://github.com/xraph/authsome)
- [Security Best Practices](../../docs/SECURITY.md)

