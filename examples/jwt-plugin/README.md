# JWT Plugin Example

This example demonstrates how to use the JWT plugin with AuthSome.

## Overview

As of the architectural fix, JWT functionality is now provided through a plugin that must be explicitly registered. This makes JWT truly optional and follows the same pattern as other plugins.

## Usage

```go
package main

import (
    "context"
    "log"

    "github.com/xraph/authsome"
    jwtplugin "github.com/xraph/authsome/plugins/jwt"
    "github.com/xraph/forge"
)

func main() {
    // Create Forge app
    app := forge.New()

    // Create AuthSome instance
    auth := authsome.New(
        authsome.WithDatabase(app.DB()),
        authsome.WithBasePath("/api/auth"),
    )

    // Register JWT plugin
    if err := auth.RegisterPlugin(jwtplugin.NewPlugin()); err != nil {
        log.Fatalf("Failed to register JWT plugin: %v", err)
    }

    // Initialize AuthSome
    if err := auth.Initialize(context.Background()); err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    // Mount routes (JWT routes will be registered automatically by the plugin)
    if err := auth.Mount(app.Router(), "/api/auth"); err != nil {
        log.Fatalf("Failed to mount: %v", err)
    }

    // Start server
    if err := app.Start(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

## Plugin Configuration

Configure the JWT plugin via config file or environment variables:

```yaml
# config.yaml
auth:
  jwt:
    issuer: "my-app"
    accessExpirySeconds: 3600      # 1 hour
    refreshExpirySeconds: 2592000  # 30 days
    signingAlgorithm: "HS256"
    includeAppIDClaim: true
```

Or programmatically:

```go
auth.RegisterPlugin(jwtplugin.NewPlugin(
    jwtplugin.WithIssuer("my-app"),
    jwtplugin.WithAccessExpiry(3600),
    jwtplugin.WithRefreshExpiry(2592000),
    jwtplugin.WithSigningAlgorithm("HS256"),
))
```

## Available Endpoints

Once the JWT plugin is registered, the following routes are available:

### Key Management
- `POST /api/auth/jwt/keys` - Create a new JWT signing key
- `GET /api/auth/jwt/keys` - List JWT signing keys

### Token Operations
- `POST /api/auth/jwt/generate` - Generate a JWT token
- `POST /api/auth/jwt/verify` - Verify a JWT token
- `GET /api/auth/jwt/jwks` - Get JSON Web Key Set (JWKS)

## Example Requests

### Create JWT Key
```bash
curl -X POST http://localhost:8080/api/auth/jwt/keys \
  -H "Content-Type: application/json" \
  -d '{
    "algorithm": "RS256",
    "keyType": "RSA"
  }'
```

### Generate Token
```bash
curl -X POST http://localhost:8080/api/auth/jwt/generate \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user_123",
    "tokenType": "access",
    "scopes": ["read:users", "write:posts"]
  }'
```

### Verify Token
```bash
curl -X POST http://localhost:8080/api/auth/jwt/verify \
  -H "Content-Type: application/json" \
  -d '{
    "token": "eyJhbGc..."
  }'
```

### Get JWKS
```bash
curl http://localhost:8080/api/auth/jwt/jwks
```

## Migration from Pre-Plugin Version

If you were using JWT before the plugin architecture fix, update your code:

**Before:**
```go
auth := authsome.New(...)
auth.Initialize(ctx)
auth.Mount(router, "/api/auth")
// JWT routes were available automatically
```

**After:**
```go
auth := authsome.New(...)
auth.RegisterPlugin(jwtplugin.NewPlugin())  // ← Add this line
auth.Initialize(ctx)
auth.Mount(router, "/api/auth")
```

## Notes

- The JWT core service is still available for internal use even without the plugin
- The plugin only provides HTTP routes for JWT operations
- JWT keys are now scoped to apps (not organizations) - see app-scoped migration docs

