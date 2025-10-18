# Schema Format Specification

## Overview

The AuthSome schema format is a JSON-based schema definition language that describes database tables, fields, indexes, and relationships in an ORM-agnostic way. This document specifies the format in detail.

## File Format

**Filename:** `authsome-schema.json`  
**Format:** JSON  
**Encoding:** UTF-8

## Root Schema

```json
{
  "version": "string (required)",
  "description": "string (optional)",
  "models": {
    "ModelName": { Model },
    ...
  }
}
```

### Root Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | Schema version (semver format: "1.0", "1.1", etc.) |
| `description` | string | No | Human-readable schema description |
| `models` | object | Yes | Map of model name to Model definition |

## Model

A Model represents a database table.

```json
{
  "name": "User",
  "table": "users",
  "description": "User account table",
  "fields": [{ Field }, ...],
  "indexes": [{ Index }, ...],
  "relations": [{ Relation }, ...]
}
```

### Model Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Model name (PascalCase, e.g., "User") |
| `table` | string | Yes | Database table name (snake_case, e.g., "users") |
| `description` | string | No | Human-readable description |
| `fields` | array | Yes | Array of Field objects |
| `indexes` | array | No | Array of Index objects |
| `relations` | array | No | Array of Relation objects |

## Field

A Field represents a table column.

```json
{
  "name": "Email",
  "column": "email",
  "type": "string",
  "description": "User email address",
  "primary": false,
  "unique": true,
  "required": true,
  "nullable": false,
  "default": null,
  "length": 255,
  "precision": 0,
  "scale": 0,
  "autoGen": false,
  "references": { Reference }
}
```

### Field Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Field name (PascalCase, e.g., "Email") |
| `column` | string | Yes | Column name (snake_case, e.g., "email") |
| `type` | FieldType | Yes | Data type (see Field Types below) |
| `description` | string | No | Human-readable description |
| `primary` | boolean | No | Is primary key? (default: false) |
| `unique` | boolean | No | Has unique constraint? (default: false) |
| `required` | boolean | No | NOT NULL constraint? (default: false) |
| `nullable` | boolean | No | Explicitly nullable? (default: false) |
| `default` | any | No | Default value |
| `length` | integer | No | Max length for string types |
| `precision` | integer | No | Precision for decimal types |
| `scale` | integer | No | Scale for decimal types |
| `autoGen` | boolean | No | Auto-generated (timestamps, IDs)? |
| `references` | Reference | No | Foreign key reference |

**Note:** `required` and `nullable` are mutually exclusive. A field cannot be both required and nullable.

## Field Types

The following field types are supported:

| Type | Description | SQL Example | Go Type | TypeScript Type |
|------|-------------|-------------|---------|-----------------|
| `string` | Variable-length string | VARCHAR(n) | string | string |
| `text` | Long text | TEXT | string | string |
| `integer` | Integer number | INTEGER | int | number |
| `bigint` | Large integer | BIGINT | int64 | number |
| `float` | Floating point | DOUBLE | float64 | number |
| `decimal` | Fixed-point decimal | DECIMAL(p,s) | decimal | number |
| `boolean` | True/false | BOOLEAN | bool | boolean |
| `timestamp` | Date and time | TIMESTAMP | time.Time | Date |
| `date` | Date only | DATE | time.Time | Date |
| `time` | Time only | TIME | time.Time | Date |
| `uuid` | UUID | UUID | string/uuid.UUID | string |
| `json` | JSON data | JSON | interface{} | any |
| `jsonb` | Binary JSON (PostgreSQL) | JSONB | interface{} | any |
| `binary` | Binary data | BYTEA/BLOB | []byte | Buffer |
| `enum` | Enumeration | ENUM | string | string |

## Reference

A Reference defines a foreign key relationship.

```json
{
  "model": "Organization",
  "field": "ID",
  "onDelete": "CASCADE",
  "onUpdate": "CASCADE"
}
```

### Reference Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | Yes | Referenced model name |
| `field` | string | Yes | Referenced field name |
| `onDelete` | string | No | Action on delete: CASCADE, SET NULL, RESTRICT |
| `onUpdate` | string | No | Action on update: CASCADE, SET NULL, RESTRICT |

## Index

An Index defines a database index.

```json
{
  "name": "idx_users_email",
  "columns": ["email"],
  "unique": false,
  "type": "btree"
}
```

### Index Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Index name |
| `columns` | array | Yes | Array of column names |
| `unique` | boolean | No | Unique index? (default: false) |
| `type` | string | No | Index type: btree, hash, gin, gist |

## Relation

A Relation describes relationships between models (for ORM mapping).

```json
{
  "name": "sessions",
  "type": "hasMany",
  "model": "Session",
  "foreignKey": "user_id",
  "references": "id",
  "through": null
}
```

### Relation Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Relation name |
| `type` | RelationType | Yes | Relationship type |
| `model` | string | Yes | Related model name |
| `foreignKey` | string | No | Foreign key column |
| `references` | string | No | Referenced column |
| `through` | string | No | Join table for many-to-many |

### Relation Types

| Type | Description | Example |
|------|-------------|---------|
| `belongsTo` | Many-to-one | Session belongs to User |
| `hasOne` | One-to-one | User has one Profile |
| `hasMany` | One-to-many | User has many Sessions |
| `manyToMany` | Many-to-many | User has many Roles through UserRoles |

## Complete Example

```json
{
  "version": "1.0",
  "description": "Example Schema",
  "models": {
    "User": {
      "name": "User",
      "table": "users",
      "description": "User account",
      "fields": [
        {
          "name": "ID",
          "column": "id",
          "type": "string",
          "primary": true,
          "required": true,
          "length": 20
        },
        {
          "name": "Email",
          "column": "email",
          "type": "string",
          "unique": true,
          "required": true,
          "length": 255
        },
        {
          "name": "EmailVerified",
          "column": "email_verified",
          "type": "boolean",
          "required": true,
          "default": false
        },
        {
          "name": "CreatedAt",
          "column": "created_at",
          "type": "timestamp",
          "required": true,
          "default": "current_timestamp",
          "autoGen": true
        }
      ],
      "indexes": [
        {
          "name": "idx_users_email",
          "columns": ["email"],
          "unique": false
        }
      ],
      "relations": [
        {
          "name": "sessions",
          "type": "hasMany",
          "model": "Session",
          "foreignKey": "user_id",
          "references": "id"
        }
      ]
    },
    "Session": {
      "name": "Session",
      "table": "sessions",
      "fields": [
        {
          "name": "ID",
          "column": "id",
          "type": "string",
          "primary": true,
          "required": true,
          "length": 20
        },
        {
          "name": "UserID",
          "column": "user_id",
          "type": "string",
          "required": true,
          "length": 20,
          "references": {
            "model": "User",
            "field": "ID",
            "onDelete": "CASCADE"
          }
        },
        {
          "name": "Token",
          "column": "token",
          "type": "string",
          "unique": true,
          "required": true
        },
        {
          "name": "ExpiresAt",
          "column": "expires_at",
          "type": "timestamp",
          "required": true
        }
      ],
      "indexes": [
        {
          "name": "idx_sessions_token",
          "columns": ["token"],
          "unique": false
        },
        {
          "name": "idx_sessions_user_id",
          "columns": ["user_id"],
          "unique": false
        }
      ],
      "relations": [
        {
          "name": "user",
          "type": "belongsTo",
          "model": "User",
          "foreignKey": "user_id",
          "references": "id"
        }
      ]
    }
  }
}
```

## Default Values

### Supported Default Values

| Type | Examples |
|------|----------|
| String literals | `"default value"`, `""` |
| Numeric literals | `0`, `1`, `42`, `3.14` |
| Boolean literals | `true`, `false` |
| SQL functions | `"current_timestamp"`, `"now()"` |
| NULL | `null` |

### Database-Specific Defaults

Some defaults are transformed per dialect:

| Generic | PostgreSQL | MySQL | SQLite |
|---------|------------|-------|--------|
| `current_timestamp` | `CURRENT_TIMESTAMP` | `CURRENT_TIMESTAMP` | `CURRENT_TIMESTAMP` |
| Boolean `true` | `true` | `1` | `1` |
| Boolean `false` | `false` | `0` | `0` |

## Naming Conventions

### Model Names
- **Format:** PascalCase
- **Examples:** `User`, `Session`, `OAuthClient`
- **Rules:** Alphanumeric, start with letter

### Table Names
- **Format:** snake_case, pluralized
- **Examples:** `users`, `sessions`, `oauth_clients`
- **Rules:** Lowercase, alphanumeric + underscore

### Field Names
- **Format:** PascalCase
- **Examples:** `Email`, `CreatedAt`, `UserID`
- **Rules:** Alphanumeric, start with letter

### Column Names
- **Format:** snake_case
- **Examples:** `email`, `created_at`, `user_id`
- **Rules:** Lowercase, alphanumeric + underscore

### Index Names
- **Format:** `idx_<table>_<columns>`
- **Examples:** `idx_users_email`, `idx_sessions_token`
- **Rules:** Descriptive, unique

## Validation Rules

### Schema-Level

1. `version` must be present and non-empty
2. `models` must contain at least one model
3. All model names must be unique
4. All table names must be unique

### Model-Level

1. `name` and `table` are required
2. Must have at least one field
3. Must have exactly one primary key
4. All field names must be unique within model
5. All column names must be unique within model

### Field-Level

1. `name`, `column`, and `type` are required
2. `type` must be a valid FieldType
3. Cannot be both `required` and `nullable`
4. Primary keys must be `required`
5. `length`, `precision`, `scale` must be positive integers
6. Foreign key references must point to existing models and fields

### Index-Level

1. `name` and `columns` are required
2. Must specify at least one column
3. All referenced columns must exist in the model

## Generator Requirements

Generators consuming this schema must:

1. Validate schema before generation
2. Support all field types (or error clearly)
3. Handle nullable vs required correctly
4. Preserve default values
5. Create indexes as specified
6. Honor foreign key constraints
7. Map field types to target language/ORM appropriately

## Schema Evolution

### Adding Fields

```json
{
  "name": "NewField",
  "column": "new_field",
  "type": "string",
  "nullable": true,
  "default": null
}
```

New fields should be nullable or have defaults for backward compatibility.

### Removing Fields

Create a new schema version and document the migration path.

### Modifying Fields

Document breaking changes clearly. Consider creating migration scripts.

## Related Tools

- **Extractor:** Generates schema from Go structs
- **Validator:** Validates schema against specification
- **Generators:** Create ORM-specific migrations

## References

- [Database Abstraction](DATABASE_ABSTRACTION.md) - Architecture overview
- [Custom ORM Guide](CUSTOM_ORM.md) - Implementing generators
- JSON Schema: https://json-schema.org/

