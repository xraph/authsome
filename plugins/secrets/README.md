# Secrets Plugin for AuthSome

The Secrets Plugin provides enterprise-grade secrets management for AuthSome applications with encryption at rest, versioning, and Forge ConfigSource integration.

## Features

- **Hierarchical Path Structure**: Organize secrets with Consul-like paths (e.g., `database/postgres/password`)
- **Encryption at Rest**: AES-256-GCM encryption with Argon2 key derivation per tenant
- **Multi-Value Type Support**: Plain text, JSON, YAML, and binary values
- **JSON Schema Validation**: Optional schema validation for structured secrets
- **Version History**: Track changes and rollback to previous versions
- **Forge ConfigSource**: Integrate secrets directly into Forge's configuration system
- **Audit Logging**: Track all access to secrets for compliance
- **Dashboard UI**: Full-featured web interface for managing secrets

## Installation

The secrets plugin is included in AuthSome. Enable it by adding to your configuration:

```yaml
auth:
  secrets:
    encryption:
      masterKey: ${AUTHSOME_SECRETS_MASTER_KEY}
```

Generate a master key:

```bash
# Generate a secure 32-byte key
openssl rand -base64 32
```

## Configuration

```yaml
auth:
  secrets:
    # Encryption settings
    encryption:
      masterKey: ${AUTHSOME_SECRETS_MASTER_KEY}  # Required: 32-byte base64-encoded key
      rotateKeyAfter: 8760h                       # Warn about rotation after 1 year
      testOnStartup: true                         # Test encryption on startup
    
    # Forge ConfigSource integration
    configSource:
      enabled: false                              # Enable config source integration
      prefix: ""                                  # Path prefix filter
      refreshInterval: 5m                         # Cache refresh interval
      autoRefresh: true                           # Auto-refresh on secret changes
      priority: 100                               # Config source priority
    
    # Access control
    access:
      requireAuthentication: true                 # Require auth for all access
      requireRbac: true                           # Enable RBAC checks
      allowApiAccess: true                        # Allow REST API access
      allowDashboardAccess: true                  # Allow dashboard access
      rateLimitPerMinute: 0                       # Rate limit (0 = disabled)
    
    # Versioning
    versioning:
      maxVersions: 50                             # Max versions to keep
      retentionDays: 90                           # Version retention period
      autoCleanup: true                           # Auto-cleanup old versions
    
    # Audit logging
    audit:
      enableAccessLog: true                       # Enable access logging
      logReads: false                             # Log read access (verbose)
      logWrites: true                             # Log write access
      retentionDays: 365                          # Log retention period
```

## Quick Start

### 1. Initialize the Plugin

```go
import "github.com/xraph/authsome/plugins/secrets"

// Create plugin with options
secretsPlugin := secrets.NewPlugin(
    secrets.WithConfigSourceEnabled(true),
    secrets.WithMaxVersions(100),
)

// Register with AuthSome
auth := authsome.New(
    authsome.WithPlugins(secretsPlugin),
)
```

### 2. Create a Secret

```bash
curl -X POST http://localhost:8080/api/auth/secrets \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "path": "database/postgres/password",
    "value": "supersecret123",
    "valueType": "plain",
    "description": "PostgreSQL production password",
    "tags": ["production", "database"]
  }'
```

### 3. Retrieve a Secret

```bash
# Get metadata
curl http://localhost:8080/api/auth/secrets/path/database/postgres/password \
  -H "Authorization: Bearer $TOKEN"

# Get decrypted value
curl http://localhost:8080/api/auth/secrets/abc123/value \
  -H "Authorization: Bearer $TOKEN"
```

## API Reference

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/secrets` | List secrets with filtering |
| POST | `/secrets` | Create a new secret |
| GET | `/secrets/:id` | Get secret metadata |
| GET | `/secrets/:id/value` | Get decrypted value |
| PUT | `/secrets/:id` | Update a secret |
| DELETE | `/secrets/:id` | Delete a secret |
| GET | `/secrets/:id/versions` | Get version history |
| POST | `/secrets/:id/rollback/:version` | Rollback to version |
| GET | `/secrets/path/*path` | Get secret by path |
| GET | `/secrets/stats` | Get statistics |
| GET | `/secrets/tree` | Get tree structure |

### Query Parameters

For `GET /secrets`:
- `prefix` - Filter by path prefix
- `search` - Search in path and description
- `valueType` - Filter by type (plain, json, yaml, binary)
- `tags` - Filter by tags (comma-separated)
- `page` - Page number (default: 1)
- `pageSize` - Items per page (default: 20, max: 100)
- `sortBy` - Sort field (path, created_at, updated_at)
- `sortOrder` - Sort order (asc, desc)

### Request/Response Examples

**Create Secret with JSON Value:**
```json
{
  "path": "services/stripe/config",
  "value": {
    "apiKey": "sk_live_...",
    "webhookSecret": "whsec_..."
  },
  "valueType": "json",
  "schema": {
    "type": "object",
    "required": ["apiKey", "webhookSecret"]
  },
  "description": "Stripe API configuration",
  "tags": ["payments", "production"]
}
```

**Secret Response:**
```json
{
  "id": "cnpq7jkg3k8g00f45e40",
  "path": "services/stripe/config",
  "key": "config",
  "valueType": "json",
  "description": "Stripe API configuration",
  "tags": ["payments", "production"],
  "version": 1,
  "isActive": true,
  "hasSchema": true,
  "hasExpiry": false,
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T10:30:00Z"
}
```

## Value Types

### Plain Text
Simple string values. Best for passwords, API keys, tokens.

```json
{
  "path": "api/openai/key",
  "value": "sk-proj-abc123...",
  "valueType": "plain"
}
```

### JSON
Structured JSON objects or arrays. Supports JSON Schema validation.

```json
{
  "path": "database/postgres/config",
  "value": {
    "host": "db.example.com",
    "port": 5432,
    "database": "myapp",
    "sslMode": "require"
  },
  "valueType": "json",
  "schema": {
    "type": "object",
    "required": ["host", "port", "database"]
  }
}
```

### YAML
YAML documents. Useful for configuration files.

```json
{
  "path": "kubernetes/secrets/app-config",
  "value": "database:\n  host: localhost\n  port: 5432\nredis:\n  host: cache.local",
  "valueType": "yaml"
}
```

### Binary
Base64-encoded binary data. For certificates, keys, images.

```json
{
  "path": "tls/server/certificate",
  "value": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...",
  "valueType": "binary"
}
```

## Forge ConfigSource Integration

When enabled, secrets become available through Forge's configuration system:

```go
// Secrets are accessible via dot notation
// Path: database/postgres/password
// Config key: database.postgres.password

password := configManager.GetString("database.postgres.password")
```

### How It Works

1. Secret paths are converted to config keys (`/` → `.`)
2. Values are cached in memory for performance
3. Cache is refreshed periodically or on secret changes
4. Supports hot-reload via callbacks

## Security

### Encryption Architecture

1. **Master Key**: A 32-byte key (base64 encoded) stored securely
2. **Key Derivation**: Argon2id derives unique keys per app/environment
3. **Encryption**: AES-256-GCM with random nonces per operation
4. **Isolation**: Different tenants cannot decrypt each other's secrets

### Best Practices

- Store master key in environment variable, not config files
- Rotate master key periodically
- Use RBAC to control access
- Enable audit logging in production
- Set appropriate expiration dates

## RBAC Roles

The plugin registers two default roles:

- `secrets_admin`: Full CRUD access to all secrets
- `secrets_viewer`: Read-only access to secret metadata (no values)

## Dashboard

Access the secrets dashboard at:
```
/dashboard/app/{appId}/secrets
```

Features:
- List/search secrets with tree view
- Create/edit secrets with syntax hints
- Reveal values with auto-hide
- Version history and rollback
- Statistics overview

## Troubleshooting

### Common Issues

**"secrets master key is not configured"**
Set the `AUTHSOME_SECRETS_MASTER_KEY` environment variable with a valid 32-byte base64-encoded key.

**"invalid master key"**
Ensure the key is exactly 32 bytes when decoded from base64.

**"secret not found"**
Check that the path is correct and normalized (lowercase, no leading/trailing slashes).

**"validation failed"**
The value doesn't match the JSON Schema. Check the schema requirements.

### Debug Mode

Enable debug logging:
```yaml
logging:
  level: debug
```

## License

Part of the AuthSome project. See main LICENSE file.

