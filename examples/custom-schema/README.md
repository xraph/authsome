# Custom Database Schema Example

This example demonstrates how to configure AuthSome to store all authentication tables in a custom PostgreSQL schema (e.g., `auth`) instead of the default `public` schema.

## Use Case

When mounting AuthSome into an existing SaaS application, you may want to:
- **Separate auth tables** from your application tables
- **Organize your database** with clear schema boundaries
- **Apply different permissions** to auth vs application data
- **Backup/restore** auth data independently

This is NOT multi-tenancy—all users share the same schema. It's purely organizational.

## What This Example Shows

1. ✅ Creating AuthSome with `WithDatabaseSchema("auth")`
2. ✅ Automatic schema creation on initialization
3. ✅ All tables created in the `auth` schema
4. ✅ Plugins automatically use the custom schema
5. ✅ Health check that queries the custom schema
6. ✅ Schema info endpoint to inspect database structure

## Prerequisites

- Go 1.21+
- PostgreSQL 12+ running locally or remote
- Database accessible at the configured URL

## Setup

### 1. Create Database

```bash
# Using psql
createdb authsome_custom_schema

# Or using SQL
psql -U postgres -c "CREATE DATABASE authsome_custom_schema;"
```

### 2. Set Environment Variables (Optional)

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/authsome_custom_schema?sslmode=disable"
```

If not set, the example uses the default URL above.

### 3. Run the Example

```bash
cd examples/custom-schema
go run main.go
```

Expected output:
```
🚀 Server starting on :8080
📊 AuthSome tables will be created in 'auth' schema
📍 Endpoints:
   - http://localhost:8080/
   - http://localhost:8080/health
   - http://localhost:8080/schema-info
   - http://localhost:8080/api/auth/*

[AuthSome] Resolved database from Forge DatabaseManager: default
[AuthSome] ✅ Applied custom database schema: auth
[AuthSome] Successfully registered all services into Forge DI container
```

## Testing

### Health Check
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "schema": "auth",
  "users": 0
}
```

### Schema Info
```bash
curl http://localhost:8080/schema-info
```

Response shows which tables are in which schema:
```json
{
  "configured_schema": "auth",
  "schemas": {
    "auth": [
      "users",
      "sessions",
      "organizations",
      "members",
      "api_keys",
      "audit_events",
      "..."
    ],
    "public": []
  },
  "message": "All AuthSome tables should be in 'auth' schema"
}
```

### Database Inspection

Connect to the database and verify:

```bash
psql -U postgres authsome_custom_schema
```

```sql
-- List all schemas
\dn

-- List tables in auth schema
\dt auth.*

-- Example: Query users table
SELECT * FROM auth.users;

-- Show search path (should include auth)
SHOW search_path;
```

Expected output:
```
 Schema |           Name            | Type  |  Owner
--------+---------------------------+-------+----------
 auth   | api_keys                  | table | postgres
 auth   | audit_events              | table | postgres
 auth   | members                   | table | postgres
 auth   | organizations             | table | postgres
 auth   | sessions                  | table | postgres
 auth   | users                     | table | postgres
 ...
```

## Code Walkthrough

### Key Configuration

```go
auth := authsome.New(
    authsome.WithForgeApp(app),
    authsome.WithDatabaseManager(dbExt.Manager(), "default"),
    authsome.WithDatabaseSchema("auth"), // 🔑 This line!
)
```

### What Happens

1. **Schema Creation**
   ```sql
   CREATE SCHEMA IF NOT EXISTS auth;
   ```

2. **Search Path Configuration**
   ```sql
   SET search_path TO auth, public;
   ```

3. **Table Creation**
   All migrations create tables in the `auth` schema:
   ```sql
   CREATE TABLE auth.users (...);
   CREATE TABLE auth.sessions (...);
   -- etc.
   ```

4. **Queries Work Seamlessly**
   Because of search_path, you can query without schema prefix:
   ```go
   db.NewSelect().Model(&schema.User{}).Scan(ctx)
   // Automatically queries auth.users
   ```

## Real-World Integration

Here's how you'd integrate this with your SaaS app:

```go
package main

import (
    "github.com/xraph/authsome"
    "github.com/xraph/forge"
    "github.com/xraph/forge/extensions/database"
)

func main() {
    app := forge.New()
    
    // Shared database for both app and auth
    dbExt := database.New(app, database.Config{
        Driver: "postgres",
        DSN:    os.Getenv("DATABASE_URL"),
    })
    dbExt.Initialize(app.Context())
    
    // Your SaaS models (use public schema)
    // db.NewCreateTable().Model(&Product{}).Exec(ctx)
    // db.NewCreateTable().Model(&Order{}).Exec(ctx)
    
    // AuthSome (use auth schema)
    auth := authsome.New(
        authsome.WithForgeApp(app),
        authsome.WithDatabaseManager(dbExt.Manager(), "default"),
        authsome.WithDatabaseSchema("auth"),
    )
    auth.Initialize(app.Context())
    auth.Mount(app.Router(), "/api/auth")
    
    // Now you have clean separation:
    // - public.products, public.orders (your app)
    // - auth.users, auth.sessions (AuthSome)
}
```

## Schema Isolation Benefits

### 1. Clear Boundaries
```
your_database/
├── auth/          <- AuthSome tables
│   ├── users
│   ├── sessions
│   └── ...
└── public/        <- Your application tables
    ├── products
    ├── orders
    └── ...
```

### 2. Independent Backups
```bash
# Backup auth data only
pg_dump -n auth authsome_custom_schema > auth_backup.sql

# Backup app data only
pg_dump -n public authsome_custom_schema > app_backup.sql
```

### 3. Schema-Level Permissions
```sql
-- Auth service user (restricted)
CREATE USER authsome_svc WITH PASSWORD 'secure';
GRANT USAGE ON SCHEMA auth TO authsome_svc;
GRANT ALL ON ALL TABLES IN SCHEMA auth TO authsome_svc;

-- App service user (no auth access)
CREATE USER app_svc WITH PASSWORD 'secure';
GRANT USAGE ON SCHEMA public TO app_svc;
GRANT ALL ON ALL TABLES IN SCHEMA public TO app_svc;
```

## Troubleshooting

### Tables in Wrong Schema

If tables already exist in `public`, move them:

```sql
-- Backup first!
ALTER TABLE public.users SET SCHEMA auth;
ALTER TABLE public.sessions SET SCHEMA auth;
-- ... etc
```

Or drop and recreate:

```sql
-- ⚠️ WARNING: Deletes all data
DROP SCHEMA public CASCADE;
```

Then restart the application to recreate in the correct schema.

### Permission Denied

Ensure your database user can create schemas:

```sql
GRANT CREATE ON DATABASE authsome_custom_schema TO your_user;
```

### Tables Not Found

Make sure you're using the configured schema:

```go
// ✅ Correct - uses configured schema automatically
db.NewSelect().Model(&schema.User{}).Scan(ctx)

// ❌ Incorrect - explicitly uses public
db.NewSelect().Table("public.users").Scan(ctx)

// ✅ Correct - explicitly uses auth
db.NewSelect().Table("auth.users").Scan(ctx)
```

## Cleanup

```bash
# Drop the database
dropdb authsome_custom_schema

# Or using SQL
psql -U postgres -c "DROP DATABASE authsome_custom_schema;"
```

## Related Examples

- [Basic Standalone](../standalone/) - Simple AuthSome setup
- [SaaS Mode](../saas/) - Multi-tenant configuration
- [Custom Config](../custom-config/) - Advanced configuration

## Documentation

See [DATABASE_SCHEMA.md](../../docs/DATABASE_SCHEMA.md) for complete documentation.

## Questions?

Open an issue: https://github.com/xraph/authsome/issues

