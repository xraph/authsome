# Database Abstraction Layer

## Overview

AuthSome provides a schema-driven database abstraction layer that allows you to use any ORM or database driver while maintaining a consistent database schema. This architecture separates the schema definition from the ORM implementation, giving you the flexibility to choose the best tools for your stack.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    AuthSome Framework                         │
│  ┌───────────────────────────────────────────────────────┐  │
│  │          authsome-schema.json                          │  │
│  │       (Single Source of Truth)                         │  │
│  └────────────────────┬──────────────────────────────────┘  │
│                       │                                       │
│         ┌─────────────┴──────────────┐                      │
│         │   Schema Generators         │                      │
│         └─────────────┬──────────────┘                      │
│                       │                                       │
│      ┌────────┬──────┼──────┬─────────┐                    │
│      │        │      │      │         │                      │
│   ┌──▼──┐ ┌──▼──┐ ┌─▼──┐ ┌─▼───┐ ┌──▼──┐                 │
│   │ SQL │ │ Bun │ │GORM│ │Prisma│ │Diesel│                 │
│   └──┬──┘ └──┬──┘ └─┬──┘ └─┬───┘ └──┬──┘                 │
└──────┼───────┼──────┼──────┼────────┼────────────────────┘
       │       │      │      │        │
    ┌──▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐  ┌─▼──┐
    │Postgres│ │Go │ │Go │ │TypeScript│ │Rust│
    │MySQL  │ │App│ │App│ │App   │ │App │
    │SQLite │ └────┘ └────┘ └──────┘ └────┘
    └───────┘
```

## Key Benefits

### 1. **ORM Independence**
- Use Bun, GORM, Prisma, Diesel, or any other ORM
- Switch ORMs without changing AuthSome
- Existing Bun implementation continues to work

### 2. **Multi-Language Support**
- Go: Bun, GORM
- TypeScript: Prisma, Drizzle
- Rust: Diesel, SeaORM
- Raw SQL: Works with any driver

### 3. **Version Control**
- Schema versioned with code
- Migration history tracked
- Easy rollback capabilities

### 4. **Customization**
- Extend schema for your needs
- Add application-specific fields
- Maintain compatibility with AuthSome

## How It Works

### 1. Schema Definition

AuthSome ships with `authsome-schema.json` defining all required tables:

```json
{
  "version": "1.0",
  "description": "AuthSome Database Schema",
  "models": {
    "User": {
      "name": "User",
      "table": "users",
      "fields": [
        {
          "name": "ID",
          "column": "id",
          "type": "string",
          "primary": true,
          "length": 20
        },
        {
          "name": "Email",
          "column": "email",
          "type": "string",
          "unique": true,
          "required": true
        }
      ]
    }
  }
}
```

### 2. Migration Generation

Generate migrations for your chosen ORM:

```bash
# PostgreSQL with raw SQL
authsome generate migrations --orm=sql --dialect=postgres --output=./migrations

# Bun (Go)
authsome generate migrations --orm=bun --output=./migrations

# GORM (Go)
authsome generate migrations --orm=gorm --output=./migrations

# Prisma (TypeScript)
authsome generate migrations --orm=prisma --output=./prisma

# Diesel (Rust)
authsome generate migrations --orm=diesel --output=./migrations
```

### 3. Apply Migrations

Use your ORM's standard tooling:

```bash
# SQL migrations (any driver)
psql < migrations/001_initial_up.sql

# Bun
go run migrations/*.go

# GORM
# Import and call AutoMigrate in your code

# Prisma
cd prisma && npx prisma migrate dev

# Diesel
diesel migration run
```

## Supported ORMs

### SQL (Raw SQL)

**Dialects:** PostgreSQL, MySQL, SQLite

**Output:**
- `001_initial_up.sql` - Create tables
- `001_initial_down.sql` - Drop tables

**Use case:** Maximum control, no ORM overhead, works with any SQL driver

### Bun (Go)

**Output:**
- `001_initial.go` - Bun migration file

**Use case:** Default for AuthSome, best integration

**Example:**
```go
import "github.com/xraph/authsome/migrations"

// Run migrations
migrator := migrations.NewMigrator(db)
if err := migrator.Run(ctx); err != nil {
    log.Fatal(err)
}
```

### GORM (Go)

**Output:**
- `models.go` - GORM model structs
- `migrate.go` - AutoMigrate function

**Use case:** Popular Go ORM with active community

**Example:**
```go
import "your-app/migrations"

// Run migrations
if err := migrations.AutoMigrate(db); err != nil {
    log.Fatal(err)
}
```

### Prisma (TypeScript)

**Output:**
- `schema.prisma` - Prisma schema file

**Use case:** TypeScript applications, excellent DX

**Example:**
```bash
npx prisma migrate dev --name initial
npx prisma generate
```

```typescript
import { PrismaClient } from '@prisma/client'

const prisma = new PrismaClient()
```

### Diesel (Rust)

**Output:**
- `YYYY-MM-DD-HHMMSS_authsome_initial/`
  - `up.sql`
  - `down.sql`

**Use case:** Rust applications, compile-time query checking

**Example:**
```bash
diesel migration run
diesel print-schema > src/schema.rs
```

```rust
use diesel::prelude::*;
```

## Workflow Examples

### Starting a New Project

```bash
# 1. Initialize your project
authsome init --orm=prisma

# 2. This creates:
#    - authsome.yaml (config)
#    - authsome-schema.json (schema definition)
#    - prisma/schema.prisma (generated schema)

# 3. Apply migrations
cd prisma
npx prisma migrate dev

# 4. Use AuthSome API
# Your database is now ready!
```

### Switching ORMs

```bash
# Current: Using Bun
# Want to: Switch to GORM

# 1. Generate GORM migrations from schema
authsome generate migrations --orm=gorm --output=./models

# 2. Review generated models.go and migrate.go
# 3. Update your application to use GORM
# 4. Database schema remains identical
```

### Adding Custom Fields

```bash
# 1. Extract current schema
authsome schema extract --output=my-schema.json

# 2. Edit my-schema.json, add custom fields:
{
  "models": {
    "User": {
      "fields": [
        {
          "name": "CustomField",
          "column": "custom_field",
          "type": "string"
        }
      ]
    }
  }
}

# 3. Generate migrations with custom schema
authsome generate migrations --schema=my-schema.json --orm=sql --dialect=postgres

# 4. Apply migrations
```

## CLI Commands

### Schema Management

```bash
# Extract schema from Go structs
authsome schema extract --input=./schema --output=authsome-schema.json

# Validate schema
authsome schema validate --schema=authsome-schema.json

# Show schema info
authsome schema info --schema=authsome-schema.json

# Compare schemas
authsome schema diff --from=v1-schema.json --to=v2-schema.json
```

### Migration Generation

```bash
# Generate migrations
authsome generate migrations \
  --orm=<sql|bun|gorm|prisma|diesel> \
  --schema=authsome-schema.json \
  --output=./migrations \
  --dialect=<postgres|mysql|sqlite>  # Required for SQL
```

### Initialization

```bash
# Initialize new project
authsome init --orm=bun --dialect=postgres
```

## Best Practices

### 1. **Version Control**
- Commit `authsome-schema.json` with your code
- Commit generated migrations
- Never modify generated files directly

### 2. **Schema Updates**
- Always regenerate from schema
- Test migrations in development first
- Use versioned migration files

### 3. **Custom Extensions**
- Fork the schema for custom fields
- Document your changes
- Consider contributing back

### 4. **Testing**
- Test migrations against fresh database
- Verify rollback works
- Check data integrity after migration

### 5. **Production**
- Back up database before migrations
- Run migrations during maintenance window
- Have rollback plan ready

## Backward Compatibility

AuthSome maintains backward compatibility:

1. **Existing Bun code continues to work**
   - Current `schema/` directory unchanged
   - Existing migrations still valid
   - No breaking changes

2. **Gradual adoption**
   - Use schema-driven approach for new projects
   - Migrate existing projects at your pace
   - Both approaches coexist

3. **Schema versioning**
   - Schema version tracked in JSON
   - Migration paths between versions
   - Clear upgrade documentation

## Troubleshooting

### Schema Extraction Fails

```bash
# Check Go syntax
go build ./schema

# Verify Bun tags present
grep -r "bun:\"" ./schema
```

### Generated Migration Invalid

```bash
# Validate schema first
authsome schema validate --schema=authsome-schema.json

# Check for syntax errors in schema
```

### ORM-Specific Issues

**Bun:** Ensure models match schema exactly

**GORM:** Check AutoMigrate compatibility with your DB

**Prisma:** Run `npx prisma validate`

**Diesel:** Verify PostgreSQL syntax

## Contributing

Want to add support for a new ORM?

1. Implement `generator.Generator` interface
2. Add generator package under `pkg/schema/generator/`
3. Register in `cmd/authsome-cli/generate_migrations.go`
4. Add tests and documentation
5. Submit PR

See `pkg/schema/generator/sql/generator.go` for reference implementation.

## Related Documentation

- [Schema Format](SCHEMA_FORMAT.md) - JSON schema specification
- [Custom ORM Guide](CUSTOM_ORM.md) - Adding new generators
- [Migration Guide](MIGRATION_GUIDE.md) - Migrating from pure Bun

## Support

- GitHub Issues: https://github.com/xraph/authsome/issues
- Documentation: https://authsome.dev
- Examples: `/examples/orm-*` directories

