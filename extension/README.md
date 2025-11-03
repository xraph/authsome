# AuthSome Forge Extension

The easiest way to integrate AuthSome into your Forge application.

## Quick Start

### 1. Basic Setup

```go
package main

import (
    "github.com/xraph/forge"
    forgedb "github.com/xraph/forge/extensions/database"
    authext "github.com/xraph/authsome/extension"
)

func main() {
    // Create Forge app
    app := forge.NewApp(forge.AppConfig{
        Name:        "myapp",
        Environment: "development",
        HTTPAddress: ":8080",
    })

    // Register database extension
    app.RegisterExtension(forgedb.NewExtension(
        forgedb.WithDatabase(forgedb.DatabaseConfig{
            Name: "default",
            Type: forgedb.TypeSQLite,
            DSN:  "myapp.db",
        }),
    ))

    // Register AuthSome extension - that's it!
    app.RegisterExtension(authext.NewExtension())

    // Start the app
    app.Run()
}
```

That's all you need! AuthSome will:
- ✅ Auto-discover the database
- ✅ Initialize all services
- ✅ Mount routes at `/api/auth`
- ✅ Register all services in DI

## Configuration

### Programmatic Configuration

```go
import (
    authext "github.com/xraph/authsome/extension"
    "github.com/xraph/authsome"
)

app.RegisterExtension(authext.NewExtension(
    authext.WithMode(authsome.ModeSaaS),
    authext.WithBasePath("/auth"),
    authext.WithSecret("your-secret-key"),
    authext.WithRBACEnforcement(true),
))
```

### YAML Configuration

```yaml
# config.yaml
authsome:
  mode: saas
  basePath: /auth
  secret: ${SECRET_KEY}
  rbacEnforce: true
  trustedOrigins:
    - https://example.com
    - https://app.example.com
  databaseName: auth_db  # Use specific database from DatabaseManager
  security:
    enabled: true
    ipWhitelist:
      - 192.168.1.0/24
  rateLimit:
    enabled: true
    default:
      limit: 100
      window: 60
```

```go
// Load config automatically from file
app := forge.NewApp(forge.AppConfig{
    ConfigFile: "config.yaml",
})

// Extension will load config automatically
app.RegisterExtension(authext.NewExtension())
```

## With Plugins

```go
import (
    authext "github.com/xraph/authsome/extension"
    "github.com/xraph/authsome/plugins/jwt"
    "github.com/xraph/authsome/plugins/apikey"
    "github.com/xraph/authsome/plugins/dashboard"
)

// Method 1: During extension creation
app.RegisterExtension(authext.NewExtension(
    authext.WithPlugins(
        jwt.NewPlugin(),
        apikey.NewPlugin(),
        dashboard.NewPlugin(),
    ),
))

// Method 2: After extension creation
authExt := authext.NewExtension()
authExt.RegisterPlugin(jwt.NewPlugin())
authExt.RegisterPlugin(apikey.NewPlugin())
app.RegisterExtension(authExt)
```

## Configuration Options

### Mode

```go
authext.WithMode(authsome.ModeStandalone)  // Single tenant
authext.WithMode(authsome.ModeSaaS)        // Multi-tenant
```

### Database

```go
// Option 1: Auto-resolve from Forge database extension (default)
authext.NewExtension()

// Option 2: Use specific database from DatabaseManager
authext.NewExtension(
    authext.WithDatabaseName("auth_db"),
)

// Option 3: Provide database directly
db := bun.NewDB(...)
authext.NewExtension(
    authext.WithDatabase(db),
)
```

### Security

```go
authext.NewExtension(
    authext.WithSecurityConfig(security.Config{
        Enabled: true,
        IPWhitelist: []string{"192.168.1.0/24"},
        AllowedCountries: []string{"US", "CA"},
    }),
    authext.WithGeoIPProvider(myGeoIPProvider),
)
```

### Rate Limiting

```go
authext.NewExtension(
    authext.WithRateLimitConfig(ratelimit.Config{
        Enabled: true,
        Default: ratelimit.Rule{
            Limit:  100,
            Window: 60,
        },
    }),
    authext.WithRateLimitStorage(redisStorage),
)
```

## Accessing AuthSome

### From Extension Instance

```go
authExt := authext.NewExtension()
app.RegisterExtension(authExt)

// Start app
app.Start(context.Background())

// Access AuthSome instance
auth := authExt.Auth()
userService := auth.GetServiceRegistry().UserService()
```

### From Forge DI Container

```go
import "github.com/xraph/authsome"

// Resolve services from container
userService, _ := authsome.ResolveUserService(app.Container())
sessionService, _ := authsome.ResolveSessionService(app.Container())
authService, _ := authsome.ResolveAuthService(app.Container())
```

## Complete Example

```go
package main

import (
    "context"
    "log"

    "github.com/xraph/forge"
    forgedb "github.com/xraph/forge/extensions/database"
    authext "github.com/xraph/authsome/extension"
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/jwt"
    "github.com/xraph/authsome/plugins/apikey"
    "github.com/xraph/authsome/plugins/dashboard"
)

func main() {
    // Create app
    app := forge.NewApp(forge.AppConfig{
        Name:        "myapp",
        Environment: "production",
        HTTPAddress: ":8080",
        ConfigFile:  "config.yaml",
    })

    // Database extension
    app.RegisterExtension(forgedb.NewExtension(
        forgedb.WithDatabase(forgedb.DatabaseConfig{
            Name: "main",
            Type: forgedb.TypePostgres,
            DSN:  "postgres://localhost/myapp",
        }),
    ))

    // AuthSome extension with full configuration
    app.RegisterExtension(authext.NewExtension(
        authext.WithMode(authsome.ModeSaaS),
        authext.WithBasePath("/auth"),
        authext.WithSecret("your-secret-key"),
        authext.WithRBACEnforcement(true),
        authext.WithPlugins(
            jwt.NewPlugin(),
            apikey.NewPlugin(),
            dashboard.NewPlugin(),
        ),
    ))

    // Add your app routes
    app.Router().GET("/", func(c forge.Context) error {
        return c.JSON(200, map[string]string{
            "message": "Hello World",
        })
    })

    // Run
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Multi-Database Example

```go
// config.yaml
database:
  databases:
    - name: "main"
      type: "postgres"
      dsn: "postgres://localhost/app_db"
    - name: "auth"
      type: "postgres"
      dsn: "postgres://localhost/auth_db"
    - name: "analytics"
      type: "postgres"
      dsn: "postgres://localhost/analytics_db"

authsome:
  databaseName: "auth"  # Use dedicated auth database

// main.go
app.RegisterExtension(forgedb.NewExtension())
app.RegisterExtension(authext.NewExtension())

// AuthSome uses "auth" database
// Your app can use "main" or "analytics" databases
```

## Benefits

### vs Direct AuthSome Usage

| Feature | Direct Usage | Extension |
|---------|-------------|-----------|
| Setup Code | ~30 lines | ~3 lines |
| Database Setup | Manual | Automatic |
| Route Mounting | Manual | Automatic |
| Configuration | Code-based | YAML + Code |
| Lifecycle | Manual | Managed |
| Health Checks | Manual | Built-in |

### Extension Benefits

✅ **One-line integration** - Just register the extension
✅ **Auto-configuration** - Loads from YAML automatically
✅ **Managed lifecycle** - Start/Stop/Health handled by Forge
✅ **DI Integration** - All services registered automatically
✅ **Database auto-discovery** - Works with Forge database extension
✅ **Plugin support** - Easy plugin registration
✅ **Production ready** - Proper error handling and logging

## Advanced Usage

### Custom Initialization

```go
authExt := authext.NewExtension()
app.RegisterExtension(authExt)

// Start app
app.Start(context.Background())

// Access and customize after initialization
auth := authExt.Auth()

// Add custom middleware
app.Router().Use(func(next func(forge.Context) error) func(forge.Context) error {
    return func(c forge.Context) error {
        // Custom logic
        return next(c)
    }
})
```

### Dynamic Plugin Registration

```go
authExt := authext.NewExtension()

// Register plugins conditionally
if os.Getenv("ENABLE_JWT") == "true" {
    authExt.RegisterPlugin(jwt.NewPlugin())
}

if os.Getenv("ENABLE_DASHBOARD") == "true" {
    authExt.RegisterPlugin(dashboard.NewPlugin())
}

app.RegisterExtension(authExt)
```

### Health Checks

The extension automatically implements Forge's health check interface:

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "healthy",
  "extensions": {
    "authsome": {
      "status": "healthy"
    }
  }
}
```

## Migration from Direct Usage

### Before (Direct Usage)

```go
// 30+ lines of setup code
db := bun.NewDB(...)
auth := authsome.New(
    authsome.WithDatabase(db),
    authsome.WithForgeApp(app),
    authsome.WithMode(authsome.ModeStandalone),
    // ... more options
)
auth.Initialize(ctx)
auth.Mount(app.Router(), "/api/auth")
```

### After (Extension)

```go
// 1 line
app.RegisterExtension(authext.NewExtension())
```

## Best Practices

1. **Use YAML Config** for environment-specific settings
2. **Use Programmatic Options** for code-driven configuration
3. **Register Plugins** during extension creation for clarity
4. **Use DatabaseManager** for multi-database scenarios
5. **Access via DI** for loose coupling in your handlers

## See Also

- [AuthSome Documentation](../README.md)
- [Forge Extensions Guide](https://forge.dev/extensions)
- [Database Extension](../FORGE_DATABASE_INTEGRATION_SUMMARY.md)
- [Plugin System](../plugins/README.md)

