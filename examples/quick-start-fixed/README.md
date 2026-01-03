# Quick Start (Fixed) - Forge Database Extension

This example demonstrates the **correct way** to use the Forge database extension with AuthSome.

## The Problem

If you try to register the database extension without configuration:

```go
// ❌ WRONG: This will fail!
app.RegisterExtension(forgedb.NewExtension())
```

You'll get this error:
```
database: failed to load required configuration. 
Error: config required but no ConfigManager available: ConfigManager not registered
```

## The Solution

**Always provide database configuration when registering the database extension:**

```go
// ✅ CORRECT: Provide database config
app.RegisterExtension(forgedb.NewExtension(
    forgedb.WithDatabase(forgedb.DatabaseConfig{
        Name: "default",
        Type: forgedb.TypeSQLite,
        DSN:  "file:myapp.db?cache=shared&_fk=1",
    }),
))
```

## Running This Example

```bash
cd examples/quick-start-fixed
go run main.go
```

Then visit:
- 🏠 Home: http://localhost:8080/
- 🔐 Auth API: http://localhost:8080/api/auth
- 📊 Dashboard: http://localhost:8080/api/auth/dashboard

## Key Takeaways

1. **Always provide database configuration** via `forgedb.WithDatabase()`
2. **Support environment variables** for flexibility
3. **Don't rely on ConfigManager** unless you explicitly register it
4. **This approach works everywhere** - no config files needed

## Alternative: Using Config Files

If you prefer config files:

```go
import "github.com/xraph/forge/extensions/config"

// Register config extension FIRST
app.RegisterExtension(config.NewExtension(
    config.WithConfigFile("config.yaml"),
))

// Then database extension can load from config
app.RegisterExtension(forgedb.NewExtension())
```

But the programmatic approach (shown in `main.go`) is simpler and more explicit.

