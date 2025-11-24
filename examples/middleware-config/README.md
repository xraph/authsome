# Middleware Configuration Example

This example demonstrates how to customize the authentication middleware configuration using the `WithAuthMiddlewareConfig` option.

## Overview

The authentication middleware handles:
- **API key authentication** (pk/sk/rk keys)
- **Session-based authentication** (cookies + bearer tokens)
- **Dual authentication** (both API key and user session)
- **Context population** (app/environment from headers or API key)

## Default Behavior (Backwards Compatible)

Without explicit configuration, the middleware uses sensible defaults:

```go
auth, err := authsome.New(
    authsome.WithSecret("my-secret-key"),
)
```

Default configuration:
- `SessionCookieName`: "authsome_session"
- `Optional`: `true` (allows unauthenticated requests)
- `AllowAPIKeyInQuery`: `false` (security best practice)
- `AllowSessionInQuery`: `false` (security best practice)
- `APIKeyHeaders`: `["Authorization", "X-API-Key"]`

## Custom Configuration

You can customize any aspect of the middleware:

```go
auth, err := authsome.New(
    authsome.WithSecret("my-secret-key"),
    authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
        SessionCookieName:   "my_custom_session",
        Optional:            false, // Require authentication
        AllowAPIKeyInQuery:  false,
        AllowSessionInQuery: false,
        APIKeyHeaders:       []string{"Authorization", "X-API-Key", "X-Custom-Key"},
        Context: authsome.ContextConfig{
            AutoDetectFromAPIKey: true,  // Infer app/env from API key
            AutoDetectFromConfig: false, // Don't auto-detect from config
            AppIDHeader:          "X-App-ID",
            EnvironmentIDHeader:  "X-Environment-ID",
        },
    }),
)
```

## Configuration Options

### AuthMiddlewareConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `SessionCookieName` | string | "authsome_session" | Cookie name for session token |
| `Optional` | bool | true | If false, returns 401 for unauthenticated requests |
| `AllowAPIKeyInQuery` | bool | false | Allow API key in query params (NOT recommended) |
| `AllowSessionInQuery` | bool | false | Allow session token in query params (NOT recommended) |
| `APIKeyHeaders` | []string | ["Authorization", "X-API-Key"] | Headers to check for API keys |
| `Context` | ContextConfig | See below | Configuration for context population |

### ContextConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `DefaultAppID` | string | "" | Default app ID to use when not detected (xid format) |
| `DefaultEnvironmentID` | string | "" | Default environment ID to use when not detected (xid format) |
| `AutoDetectFromAPIKey` | bool | true | Infer app/env from verified API key |
| `AutoDetectFromConfig` | bool | false | Auto-detect from AuthSome config in standalone mode |
| `AppIDHeader` | string | "X-App-ID" | Header name for app ID |
| `EnvironmentIDHeader` | string | "X-Environment-ID" | Header name for environment ID |

## Common Use Cases

### 1. Require Authentication Globally

```go
authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
    Optional: false, // Block unauthenticated requests
})
```

### 2. Custom Session Cookie Name

```go
authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
    SessionCookieName: "my_app_session",
})
```

### 3. Additional API Key Headers

```go
authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
    APIKeyHeaders: []string{
        "Authorization",
        "X-API-Key",
        "X-Custom-Api-Key",
        "X-Service-Key",
    },
})
```

### 4. Custom Context Headers

```go
authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
    Context: authsome.ContextConfig{
        AutoDetectFromAPIKey: true,
        AppIDHeader:          "X-Application-ID",
        EnvironmentIDHeader:  "X-Env-ID",
    },
})
```

### 5. Standalone Mode with Auto-Detection

```go
authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
    Context: authsome.ContextConfig{
        AutoDetectFromAPIKey: true,
        AutoDetectFromConfig: true, // Enable for standalone mode
    },
})
```

### 6. Set Default App and Environment

```go
authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
    Context: authsome.ContextConfig{
        DefaultAppID:         "c7ndh411g9k8pdunveeg",
        DefaultEnvironmentID: "c7ndh412g9k8pdunveeh",
        AutoDetectFromAPIKey: true, // Still try API key first
    },
})
```

## Security Considerations

### Production Best Practices

✅ **DO:**
- Keep `AllowAPIKeyInQuery` as `false`
- Keep `AllowSessionInQuery` as `false`
- Use HTTPS in production
- Use secure cookie settings (HttpOnly, Secure, SameSite)

❌ **DON'T:**
- Enable query param authentication in production (risk of leaking in logs)
- Set `Optional: true` for endpoints requiring authentication
- Use custom API key headers without proper documentation

### Query Param Authentication Risk

Enabling `AllowAPIKeyInQuery` or `AllowSessionInQuery` is dangerous because:
- Query params appear in server logs
- Query params appear in browser history
- Query params can leak via Referer headers
- Query params are visible in monitoring tools

**Only use query param authentication for development/testing purposes.**

## Testing

Run the example:

```bash
cd examples/middleware-config
go run main.go
```

## Integration with Forge

When mounted on a Forge app, the middleware is automatically applied to all AuthSome routes:

```go
app := forge.New()

auth, err := authsome.New(
    authsome.WithForgeApp(app),
    authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
        Optional: true,
        SessionCookieName: "my_session",
    }),
)

auth.Mount(app)
```

## Further Reading

- [Authentication Context Documentation](../../docs/AUTHENTICATION_CONTEXT.md)
- [API Key Authentication](../../docs/API_KEY_AUTHENTICATION.md)
- [Session Management](../../docs/SESSION_MANAGEMENT.md)
- [Security Best Practices](../../docs/SECURITY.md)

