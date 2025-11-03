# AuthSome + Forge Database Extension Integration

This example demonstrates how to integrate AuthSome with Forge's database extension instead of manually managing database connections.

## Benefits of Using Forge Database Extension

1. **Unified Database Management** - Centralized connection pooling and lifecycle management
2. **Multi-Database Support** - Easily work with multiple databases
3. **Health Checks** - Built-in database health monitoring
4. **Metrics** - Automatic database metrics collection
5. **Configuration** - YAML-based database configuration
6. **Connection Pooling** - Optimized connection management

## Migration Registry

AuthSome now uses Forge's migration registry:

```go
// migrations/migrations.go
package migrations

import (
    forgemigrate "github.com/xraph/forge/extensions/database/migrate"
)

// Migrations is the global migration registry
// Now using Forge's database extension migration registry
var Migrations = forgemigrate.Migrations
```

This means all AuthSome migrations are registered in Forge's global migration system, allowing you to run all migrations (app + authsome) together.

## Integration Methods

### Method 1: WithDatabaseFromForge() - Recommended

This is the simplest approach. It automatically resolves the database from Forge's DI container:

```go
app := forge.NewApp(forge.AppConfig{...})

// Register database extension
dbExt := forgedb.NewExtension(
    forgedb.WithDatabase(forgedb.DatabaseConfig{
        Name: "default",
        Type: forgedb.TypeSQLite,
        DSN:  "authsome.db",
    }),
)
app.RegisterExtension(dbExt)

// Initialize AuthSome
auth := authsome.New(
    authsome.WithForgeApp(app),
    authsome.WithDatabaseFromForge(), // Automatically uses Forge's database
    authsome.WithMode(authsome.ModeStandalone),
)
```

### Method 2: WithDatabaseManager() - Advanced

Use this when you need more control or want to use a specific database from the manager:

```go
// Get DatabaseManager from DI
manager, err := authsome.ResolveDatabaseManager(app.Container())
if err != nil {
    log.Fatal(err)
}

// Use specific database
auth := authsome.New(
    authsome.WithForgeApp(app),
    authsome.WithDatabaseManager(manager, "default"), // or "analytics", "logs", etc.
    authsome.WithMode(authsome.ModeStandalone),
)
```

### Method 3: WithDatabase() - Traditional (Backwards Compatible)

This maintains backwards compatibility with existing code:

```go
// Create database connection manually
sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
db := bun.NewDB(sqldb, pgdialect.New())

// Pass directly to AuthSome
auth := authsome.New(
    authsome.WithForgeApp(app),
    authsome.WithDatabase(db), // Traditional approach
    authsome.WithMode(authsome.ModeStandalone),
)
```

## Running the Example

```bash
# Using SQLite (default)
go run main.go

# Using PostgreSQL
DATABASE_URL="postgres://user:pass@localhost/authsome" go run main.go

# Using MySQL
DATABASE_URL="mysql://user:pass@localhost/authsome" go run main.go
```

## Configuration File Example

You can also configure databases via YAML:

```yaml
# config.yaml
database:
  default: "main"
  databases:
    - name: "main"
      type: "postgres"
      dsn: "postgres://localhost/authsome"
      pool:
        maxOpenConns: 25
        maxIdleConns: 5
        connMaxLifetime: "5m"
    
    - name: "analytics"
      type: "postgres"
      dsn: "postgres://localhost/analytics"
      pool:
        maxOpenConns: 10
        maxIdleConns: 2
```

Then use it:

```go
app := forge.NewApp(forge.AppConfig{
    ConfigFile: "config.yaml",
})

dbExt := forgedb.NewExtension()
app.RegisterExtension(dbExt)

// AuthSome will use the default database
auth := authsome.New(
    authsome.WithForgeApp(app),
    authsome.WithDatabaseFromForge(),
)
```

## Running Migrations

Migrations are now unified in Forge's migration registry:

```bash
# Using AuthSome CLI (still works)
authsome migrate up

# Or using Forge CLI (if available)
forge db migrate up

# Or programmatically
import (
    "github.com/uptrace/bun/migrate"
    forgemigrate "github.com/xraph/forge/extensions/database/migrate"
)

migrator := migrate.NewMigrator(db, forgemigrate.Migrations)
group, err := migrator.Migrate(ctx)
```

## Container Helpers

AuthSome provides helpers to resolve database components from Forge's DI:

```go
// Resolve database
db, err := authsome.ResolveDatabase(app.Container())

// Resolve database manager
manager, err := authsome.ResolveDatabaseManager(app.Container())

// Resolve AuthSome services
userService, err := authsome.ResolveUserService(app.Container())
```

## Multi-Database Scenarios

Using DatabaseManager allows AuthSome to use one database while your app uses others:

```go
// config.yaml
database:
  databases:
    - name: "auth"
      type: "postgres"
      dsn: "postgres://localhost/auth_db"
    
    - name: "analytics"
      type: "postgres"
      dsn: "postgres://localhost/analytics_db"

// main.go
manager, _ := authsome.ResolveDatabaseManager(app.Container())

// AuthSome uses "auth" database
auth := authsome.New(
    authsome.WithForgeApp(app),
    authsome.WithDatabaseManager(manager, "auth"),
)

// Your app uses "analytics" database
analyticsDB, _ := manager.SQL("analytics")
```

## Health Checks

Forge's database extension provides health checks:

```go
// Check all databases
statuses := manager.HealthCheckAll(ctx)
for name, status := range statuses {
    if !status.Healthy {
        log.Printf("Database %s unhealthy: %s", name, status.Message)
    }
}
```

## Metrics

Database metrics are automatically collected:

- Connection pool stats
- Query durations
- Error rates
- Open/close operations

## Benefits Summary

| Feature | Traditional | Forge Extension |
|---------|------------|-----------------|
| Connection Management | Manual | Automatic |
| Multi-DB Support | Complex | Built-in |
| Health Checks | Manual | Built-in |
| Metrics | Manual | Automatic |
| Configuration | Code | YAML + Code |
| Lifecycle | Manual | Managed |
| Migration Registry | Separate | Unified |

## Backwards Compatibility

All existing code using `WithDatabase()` continues to work without changes. The new methods are additive and optional.

## Recommended Approach

For new projects: **Use `WithDatabaseFromForge()`**

For existing projects: **Keep using `WithDatabase()`** or migrate gradually

For complex multi-database setups: **Use `WithDatabaseManager()`**

