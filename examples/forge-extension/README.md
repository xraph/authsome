# AuthSome Forge Extension Example

This example demonstrates the easiest way to use AuthSome in a Forge application: as a Forge extension.

## What This Shows

- ✅ One-line AuthSome integration
- ✅ Automatic database discovery
- ✅ Plugin registration
- ✅ YAML-based configuration
- ✅ Production-ready setup

## Quick Start

### 1. Run with defaults (in-memory SQLite)

```bash
go run main.go
```

### 2. Run with PostgreSQL

```bash
DATABASE_URL="postgres://localhost/authsome" go run main.go
```

### 3. Run with configuration file

```bash
# Edit config.yaml first
go run main.go
```

## Code Comparison

### Before (Direct Usage) - ~40 lines

```go
// Manual database setup
sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
db := bun.NewDB(sqldb, pgdialect.New())
defer db.Close()

if err := db.Ping(); err != nil {
    log.Fatal(err)
}

// Create Forge app
app := forge.New()

// Initialize AuthSome
auth := authsome.New(
    authsome.WithDatabase(db),
    authsome.WithForgeApp(app),
    authsome.WithMode(authsome.ModeStandalone),
    authsome.WithSecret("secret"),
)

// Register plugins
auth.RegisterPlugin(jwt.NewPlugin())
auth.RegisterPlugin(apikey.NewPlugin())
auth.RegisterPlugin(dashboard.NewPlugin())

// Initialize
ctx := context.Background()
if err := auth.Initialize(ctx); err != nil {
    log.Fatal(err)
}

// Mount routes
if err := auth.Mount(app.Router(), "/api/auth"); err != nil {
    log.Fatal(err)
}

// Start server
app.Run()
```

### After (Extension) - 3 lines!

```go
app := forge.NewApp(forge.AppConfig{...})

app.RegisterExtension(forgedb.NewExtension(...))

app.RegisterExtension(authext.NewExtension(
    authext.WithPlugins(
        jwt.NewPlugin(),
        apikey.NewPlugin(),
        dashboard.NewPlugin(),
    ),
))

app.Run()
```

## Configuration Options

### Programmatic (in code)

```go
app.RegisterExtension(authext.NewExtension(
    authext.WithMode(authsome.ModeSaaS),
    authext.WithBasePath("/auth"),
    authext.WithSecret("your-secret"),
    authext.WithRBACEnforcement(true),
    authext.WithPlugins(
        jwt.NewPlugin(),
        apikey.NewPlugin(),
    ),
))
```

### YAML-based (config.yaml)

```yaml
authsome:
  mode: saas
  basePath: /auth
  secret: ${SECRET_KEY}
  rbacEnforce: true
  trustedOrigins:
    - https://example.com
  security:
    enabled: true
  rateLimit:
    enabled: true
```

```go
// Loads configuration automatically
app := forge.NewApp(forge.AppConfig{
    ConfigFile: "config.yaml",
})
app.RegisterExtension(authext.NewExtension())
```

## Available Endpoints

After running, these endpoints are available:

- `GET /` - Home
- `GET /health` - Health check
- `POST /api/auth/signup` - User registration
- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - User logout
- `GET /api/auth/session` - Get current session
- `GET /api/auth/dashboard` - Admin dashboard (dashboard plugin)
- And many more...

## Testing

### 1. Check Health

```bash
curl http://localhost:8080/health
```

### 2. Register a User

```bash
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "name": "Test User"
  }'
```

### 3. Login

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'
```

### 4. Access Dashboard

```bash
open http://localhost:8080/api/auth/dashboard
```

## Multi-Database Configuration

```yaml
# config.yaml
database:
  databases:
    - name: main
      type: postgres
      dsn: postgres://localhost/app_db
    - name: auth
      type: postgres
      dsn: postgres://localhost/auth_db
    - name: analytics
      type: postgres
      dsn: postgres://localhost/analytics_db

authsome:
  databaseName: auth  # AuthSome uses dedicated database
```

## Environment Variables

```bash
# Database URL
DATABASE_URL="postgres://localhost/authsome"

# Secret key
SECRET_KEY="your-secret-key"

# Mode
AUTHSOME_MODE="saas"
```

## Production Checklist

- [ ] Set strong `SECRET_KEY` environment variable
- [ ] Configure production database (PostgreSQL)
- [ ] Enable RBAC enforcement if needed
- [ ] Configure rate limiting
- [ ] Set up security rules (IP whitelist, country restrictions)
- [ ] Configure CORS trusted origins
- [ ] Set up monitoring and logging
- [ ] Enable HTTPS
- [ ] Configure session timeouts
- [ ] Set up backup strategy

## Benefits

### Development
- ✅ Fast setup - 3 lines of code
- ✅ Hot reload support
- ✅ Easy testing
- ✅ Quick iterations

### Production
- ✅ Proper lifecycle management
- ✅ Health checks built-in
- ✅ Metrics collection
- ✅ Configuration management
- ✅ Error handling
- ✅ Graceful shutdown

## Next Steps

1. Explore the [extension package](../../extension/)
2. Check [available plugins](../../plugins/)
3. Read [AuthSome documentation](../../README.md)
4. Review [Forge extensions guide](https://forge.dev/extensions)

## Troubleshooting

### Database not found

```
Error: failed to resolve database from Forge DI
```

**Solution:** Make sure database extension is registered before AuthSome extension:

```go
app.RegisterExtension(forgedb.NewExtension(...))  // Register FIRST
app.RegisterExtension(authext.NewExtension())      // Then AuthSome
```

### Config not loading

```
Warning: using default/programmatic config
```

**Solution:** Ensure config file exists and is properly formatted:

```go
app := forge.NewApp(forge.AppConfig{
    ConfigFile: "config.yaml",  // Must exist
})
```

### Plugins not working

**Solution:** Register plugins during extension creation or before app starts:

```go
authExt := authext.NewExtension()
authExt.RegisterPlugin(jwt.NewPlugin())
app.RegisterExtension(authExt)
```

## See Also

- [Extension Package Documentation](../../extension/README.md)
- [Forge Database Integration](../../FORGE_DATABASE_INTEGRATION_SUMMARY.md)
- [Plugin System](../../plugins/README.md)

