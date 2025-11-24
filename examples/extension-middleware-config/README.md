# Extension Middleware Configuration Example

This example demonstrates how to customize the authentication middleware configuration when using AuthSome as a Forge extension.

## Overview

The AuthSome extension for Forge allows you to configure the authentication middleware behavior at extension creation time, giving you full control over authentication requirements, session management, and context resolution.

## Usage Patterns

### 1. Default Configuration (Backwards Compatible)

```go
app := forge.New()

ext := extension.NewExtension(
    extension.WithBasePath("/auth"),
    extension.WithSecret("my-secret-key"),
)

app.RegisterExtension(ext)
```

Uses sensible defaults:
- `SessionCookieName`: "authsome_session"
- `Optional`: `true` (allows unauthenticated requests)
- `AllowAPIKeyInQuery`: `false` (secure)
- `AllowSessionInQuery`: `false` (secure)

### 2. Custom Middleware Configuration

```go
ext := extension.NewExtension(
    extension.WithBasePath("/api/auth"),
    extension.WithSecret("my-secret-key"),
    extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
        SessionCookieName:   "my_custom_session",
        Optional:            false, // Require authentication
        AllowAPIKeyInQuery:  false,
        AllowSessionInQuery: false,
        APIKeyHeaders:       []string{"Authorization", "X-API-Key", "X-Custom-Key"},
        Context: middleware.ContextConfig{
            AutoDetectFromAPIKey: true,
            AutoDetectFromConfig: false,
            AppIDHeader:          "X-App-ID",
            EnvironmentIDHeader:  "X-Environment-ID",
        },
    }),
)
```

### 3. Partial Configuration

Override only what you need:

```go
ext := extension.NewExtension(
    extension.WithBasePath("/auth"),
    extension.WithSecret("my-secret-key"),
    extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
        Optional: false, // Require auth, rest uses defaults
        Context: middleware.ContextConfig{
            AutoDetectFromAPIKey: true,
            AutoDetectFromConfig: true,
        },
    }),
)
```

### 4. Security-First Configuration

Production-ready secure configuration:

```go
ext := extension.NewExtension(
    extension.WithBasePath("/auth"),
    extension.WithSecret("production-secret-key"),
    extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
        SessionCookieName:   "secure_session",
        Optional:            false, // Block unauthenticated requests
        AllowAPIKeyInQuery:  false, // Never in production
        AllowSessionInQuery: false, // Never in production
        Context: middleware.ContextConfig{
            AutoDetectFromAPIKey: true,
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

## YAML Configuration

You can also configure via YAML:

```yaml
authsome:
  basePath: "/api/auth"
  secret: "my-secret-key"
  authMiddleware:
    sessionCookieName: "my_session"
    optional: false
    allowAPIKeyInQuery: false
    allowSessionInQuery: false
    context:
      defaultAppID: "c7ndh411g9k8pdunveeg"
      defaultEnvironmentID: "c7ndh412g9k8pdunveeh"
      autoDetectFromAPIKey: true
      autoDetectFromConfig: true
      appIDHeader: "X-App-ID"
      environmentIDHeader: "X-Environment-ID"
```

Then load it:

```go
ext := extension.NewExtension()
// Config is automatically loaded from Forge's config system
app.RegisterExtension(ext)
```

## Complete Example

```go
package main

import (
    "context"
    "log"

    "github.com/xraph/authsome/core/middleware"
    "github.com/xraph/authsome/extension"
    "github.com/xraph/forge"
)

func main() {
    app := forge.New()

    // Create extension with custom middleware config
    authExt := extension.NewExtension(
        extension.WithBasePath("/api/auth"),
        extension.WithSecret("my-secret-key"),
        extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
            SessionCookieName: "app_session",
            Optional:          true,
            Context: middleware.ContextConfig{
                AutoDetectFromAPIKey: true,
                AppIDHeader:          "X-App-ID",
                EnvironmentIDHeader:  "X-Environment-ID",
            },
        }),
    )

    // Register extension
    if err := app.RegisterExtension(authExt); err != nil {
        log.Fatal(err)
    }

    // Start app
    ctx := context.Background()
    if err := app.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Add your routes
    app.Router().GET("/", func(c forge.Context) error {
        return c.JSON(200, map[string]string{
            "message": "Hello!",
        })
    })

    // Run server
    if err := app.Run(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

## Security Considerations

### Production Best Practices

✅ **DO:**
- Keep `AllowAPIKeyInQuery` as `false`
- Keep `AllowSessionInQuery` as `false`
- Use HTTPS in production
- Use strong secrets (environment variables)
- Set `Optional: false` for endpoints requiring auth

❌ **DON'T:**
- Enable query param authentication in production
- Hard-code secrets in source code
- Use weak or default secrets

### Query Param Authentication Risk

Enabling query param authentication is dangerous because:
- Tokens appear in server logs
- Tokens appear in browser history
- Tokens leak via Referer headers
- Tokens visible in monitoring tools

**Only use for local development/testing.**

## Integration with Forge Extensions

The middleware config works seamlessly with other Forge extensions:

```go
app := forge.New()

// Register database extension
dbExt := database.NewExtension(/* config */)
app.RegisterExtension(dbExt)

// Register AuthSome with middleware config
authExt := extension.NewExtension(
    extension.WithBasePath("/auth"),
    extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
        Optional: false,
    }),
)
app.RegisterExtension(authExt)

// Register other extensions...
```

## Testing

Run the example:

```bash
cd examples/extension-middleware-config
go run main.go
```

## Accessing the Auth Instance

After registration, access the AuthSome instance:

```go
auth := ext.Auth()
serviceRegistry := ext.GetServiceRegistry()
pluginRegistry := ext.GetPluginRegistry()
basePath := ext.GetBasePath()
db := ext.GetDB()
```

## Further Reading

- [Extension Documentation](../../extension/README.md)
- [Middleware Configuration](../middleware-config/README.md)
- [Authentication Context](../../docs/AUTHENTICATION_CONTEXT.md)
- [Security Best Practices](../../docs/SECURITY.md)

