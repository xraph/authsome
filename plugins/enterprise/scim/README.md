# SCIM 2.0 Provisioning Plugin

Enterprise-grade SCIM 2.0 implementation for automated user and group provisioning from identity providers like Okta, Azure AD, OneLogin, and others.

## Overview

The SCIM (System for Cross-domain Identity Management) 2.0 plugin enables enterprise customers to:

- **Automate User Provisioning**: Create, update, and deactivate users automatically
- **Group Synchronization**: Sync SCIM groups to AuthSome teams/roles
- **Real-time Updates**: Immediate synchronization of identity changes
- **Bulk Operations**: Efficiently provision multiple users at once
- **Custom Attribute Mapping**: Flexible schema mapping for custom attributes
- **JIT Provisioning**: Just-In-Time user creation on first login
- **Enterprise Integration**: Works with Okta, Azure AD, OneLogin, Google Workspace, etc.

## Features

### ✅ SCIM 2.0 Standard Compliance

- Full RFC 7643 and RFC 7644 compliance
- Core User schema (RFC 7643 Section 4.1)
- Enterprise User extension (RFC 7643 Section 4.3)
- Group schema (RFC 7643 Section 4.2)
- Service Provider Configuration endpoint
- Resource Type and Schema discovery

### 🔐 Security

- Bearer token authentication
- Organization-scoped isolation
- IP whitelisting support
- Rate limiting per organization
- Complete audit logging
- Secure token generation and storage

### 📊 Operations Supported

**User Operations:**
- `POST /scim/v2/Users` - Create user
- `GET /scim/v2/Users` - List users with filtering
- `GET /scim/v2/Users/:id` - Get user by ID
- `PUT /scim/v2/Users/:id` - Replace user (full update)
- `PATCH /scim/v2/Users/:id` - Update user (partial)
- `DELETE /scim/v2/Users/:id` - Deactivate/delete user

**Group Operations:**
- `POST /scim/v2/Groups` - Create group
- `GET /scim/v2/Groups` - List groups
- `GET /scim/v2/Groups/:id` - Get group by ID
- `PUT /scim/v2/Groups/:id` - Replace group
- `PATCH /scim/v2/Groups/:id` - Update group
- `DELETE /scim/v2/Groups/:id` - Delete group

**Bulk Operations:**
- `POST /scim/v2/Bulk` - Process multiple operations in a single request

**Search:**
- `POST /scim/v2/.search` - Search across resources with filtering

**Discovery:**
- `GET /scim/v2/ServiceProviderConfig` - Service provider capabilities
- `GET /scim/v2/ResourceTypes` - Supported resource types
- `GET /scim/v2/Schemas` - Supported schemas

## Installation

### 1. Add Plugin to Your AuthSome Instance

```go
package main

import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/enterprise/scim"
)

func main() {
    // Create AuthSome instance
    auth := authsome.New(
        authsome.WithMode(authsome.ModeSaaS),
        authsome.WithDatabase(db),
        authsome.WithForgeApp(app),
    )
    
    // Register SCIM plugin
    scimPlugin := scim.NewPlugin()
    if err := auth.RegisterPlugin(scimPlugin); err != nil {
        panic(err)
    }
    
    // Initialize AuthSome (this will initialize all plugins)
    if err := auth.Initialize(ctx); err != nil {
        panic(err)
    }
    
    // Mount routes
    if err := auth.Mount(router, "/api/auth"); err != nil {
        panic(err)
    }
}
```

### 2. Configure Plugin

```yaml
# config.yaml
auth:
  plugins:
    scim:
      enabled: true
      auth_method: "bearer"
      token_expiry: "90d"
      
      rate_limit:
        enabled: true
        requests_per_min: 600  # 10 req/sec
        burst_size: 100
      
      user_provisioning:
        enabled: true
        auto_activate: true
        send_welcome_email: false
        default_role: "member"
        prevent_duplicates: true
        soft_delete_on_deprovision: true
      
      group_sync:
        enabled: true
        sync_to_teams: true
        sync_to_roles: false
        create_missing_groups: true
      
      bulk_operations:
        enabled: true
        max_operations: 100
        max_payload_bytes: 1048576  # 1MB
      
      search:
        max_results: 1000
        default_results: 50
      
      security:
        require_https: true
        audit_all_operations: true
        mask_sensitive_data: true
        require_org_validation: true
```

### 3. Run Database Migrations

Migrations run automatically when the plugin is initialized. You can also run them manually:

```go
if err := scimPlugin.Migrate(); err != nil {
    panic(err)
}
```

## Usage

### Creating a Provisioning Token

Before you can use SCIM, create a provisioning token for your identity provider:

```bash
curl -X POST http://localhost:8080/api/scim-admin/tokens \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Okta Production",
    "description": "SCIM token for Okta production environment",
    "scopes": ["scim:read", "scim:write"],
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "id": "cm3abc123def456ghi789",
  "name": "Okta Production",
  "message": "Store this token securely. It will not be shown again."
}
```

⚠️ **Important**: Save the token immediately. It will not be displayed again.

### Configuring Your Identity Provider

#### Okta

1. Navigate to **Applications > Applications** in Okta Admin Console
2. Click **Browse App Catalog**
3. Search for **SCIM 2.0 Test App (Header Auth)**
4. Click **Add Integration**
5. Configure:
   - **SCIM 2.0 Base URL**: `https://your-domain.com/scim/v2`
   - **Unique identifier field for users**: `userName`
   - **Supported provisioning actions**: Check all
   - **Authentication Mode**: `HTTP Header`
   - **Authorization**: `Bearer YOUR_SCIM_TOKEN`

#### Azure AD

1. Navigate to **Azure Active Directory > Enterprise Applications**
2. Click **New Application > Create your own application**
3. Select **Integrate any other application you don't find in the gallery (Non-gallery)**
4. Go to **Provisioning > Automatic**
5. Configure:
   - **Tenant URL**: `https://your-domain.com/scim/v2`
   - **Secret Token**: `YOUR_SCIM_TOKEN`
6. Test connection and save

#### OneLogin

1. Navigate to **Applications > Custom Connectors**
2. Click **New Connector**
3. Select **SCIM 2.0**
4. Configure:
   - **SCIM Base URL**: `https://your-domain.com/scim/v2`
   - **SCIM Bearer Token**: `YOUR_SCIM_TOKEN`
   - **SCIM Version**: `2.0`
5. Enable provisioning features

### Example: Provisioning a User

When you assign a user in your IdP, it will automatically create the user in AuthSome:

**SCIM Request (from IdP):**
```http
POST /scim/v2/Users HTTP/1.1
Host: your-domain.com
Authorization: Bearer YOUR_SCIM_TOKEN
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "bjensen@example.com",
  "name": {
    "givenName": "Barbara",
    "familyName": "Jensen"
  },
  "emails": [{
    "value": "bjensen@example.com",
    "type": "work",
    "primary": true
  }],
  "displayName": "Barbara Jensen",
  "active": true,
  "externalId": "okta_user_12345"
}
```

**Response:**
```http
HTTP/1.1 201 Created
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "id": "cm3xyz789abc123def456",
  "externalId": "okta_user_12345",
  "userName": "bjensen@example.com",
  "name": {
    "givenName": "Barbara",
    "familyName": "Jensen",
    "formatted": "Barbara Jensen"
  },
  "emails": [{
    "value": "bjensen@example.com",
    "type": "work",
    "primary": true
  }],
  "displayName": "Barbara Jensen",
  "active": true,
  "meta": {
    "resourceType": "User",
    "created": "2024-01-15T10:30:00Z",
    "lastModified": "2024-01-15T10:30:00Z",
    "location": "/scim/v2/Users/cm3xyz789abc123def456"
  }
}
```

### Example: Updating a User

**SCIM Request (PATCH):**
```http
PATCH /scim/v2/Users/cm3xyz789abc123def456 HTTP/1.1
Host: your-domain.com
Authorization: Bearer YOUR_SCIM_TOKEN
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "replace",
      "path": "active",
      "value": false
    }
  ]
}
```

### Example: Bulk Provisioning

```http
POST /scim/v2/Bulk HTTP/1.1
Host: your-domain.com
Authorization: Bearer YOUR_SCIM_TOKEN
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
  "failOnErrors": 1,
  "Operations": [
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "qwerty",
      "data": {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": "alice@example.com",
        "name": {
          "givenName": "Alice",
          "familyName": "Smith"
        },
        "emails": [{
          "value": "alice@example.com",
          "type": "work"
        }]
      }
    },
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "ytrewq",
      "data": {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": "bob@example.com",
        "name": {
          "givenName": "Bob",
          "familyName": "Johnson"
        },
        "emails": [{
          "value": "bob@example.com",
          "type": "work"
        }]
      }
    }
  ]
}
```

## Attribute Mapping

The plugin supports flexible attribute mapping to adapt to your schema:

### Default Mappings

| SCIM Attribute | AuthSome Field |
|----------------|----------------|
| `userName` | `email` |
| `emails[0].value` | `email` |
| `name.givenName` | `name` (first part) |
| `name.familyName` | `name` (last part) |
| `displayName` | `name` |
| `active` | `email_verified` |
| `externalId` | `metadata.scim_external_id` |

### Enterprise Extension Mappings

| SCIM Attribute | AuthSome Field |
|----------------|----------------|
| `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber` | `metadata.employee_number` |
| `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department` | `metadata.department` |
| `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager.value` | `metadata.manager_id` |

### Custom Mappings

Configure custom attribute mappings:

```yaml
attribute_mapping:
  enabled: true
  custom_mapping:
    "customAttribute1": "metadata.custom_field_1"
    "customAttribute2": "metadata.custom_field_2"
```

## Group Synchronization

SCIM groups can be automatically synced to AuthSome teams or roles:

### Configuration

```yaml
group_sync:
  enabled: true
  sync_to_teams: true           # Sync to teams
  sync_to_roles: false          # Or sync to roles
  create_missing_groups: true   # Auto-create teams/roles
  delete_empty_groups: false    # Keep empty groups
```

### Example: Creating a Group

```http
POST /scim/v2/Groups HTTP/1.1
Host: your-domain.com
Authorization: Bearer YOUR_SCIM_TOKEN
Content-Type: application/scim+json

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
  "displayName": "Engineering",
  "externalId": "okta_group_engineering",
  "members": [
    {
      "value": "cm3xyz789abc123def456",
      "display": "Barbara Jensen"
    }
  ]
}
```

## Monitoring & Audit

### Viewing Provisioning Logs

```bash
curl -X GET "http://localhost:8080/api/scim-admin/logs?start_date=2024-01-01&limit=50" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

### Provisioning Statistics

```bash
curl -X GET "http://localhost:8080/api/scim-admin/stats?start_date=2024-01-01&end_date=2024-01-31" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

Response:
```json
{
  "total_operations": 1250,
  "successful_operations": 1238,
  "failed_operations": 12,
  "success_rate": 99.04,
  "operations_by_type": {
    "CREATE_USER": 500,
    "UPDATE_USER": 650,
    "DELETE_USER": 100
  },
  "average_duration_ms": 125.5
}
```

## Webhooks

Send provisioning events to external systems:

```yaml
webhooks:
  enabled: true
  notify_on_create: true
  notify_on_update: true
  notify_on_delete: true
  notify_on_group_sync: true
  webhook_urls:
    - "https://your-webhook-endpoint.com/scim-events"
  retry_attempts: 3
  timeout_seconds: 10
```

## Troubleshooting

### Common Issues

#### 1. "Invalid or expired token"

**Cause**: Bearer token is invalid, expired, or revoked.

**Solution**: Generate a new provisioning token and update your IdP configuration.

#### 2. "Rate limit exceeded"

**Cause**: Too many requests in a short period.

**Solution**: 
- Increase `rate_limit.requests_per_min` in config
- Implement exponential backoff in your IdP
- Use bulk operations for multiple users

#### 3. "User with email already exists"

**Cause**: Duplicate email address.

**Solution**:
- Set `user_provisioning.prevent_duplicates: false` to allow duplicates
- Or ensure unique emails in your IdP

#### 4. "Group synchronization failed"

**Cause**: Group sync is disabled or team/role doesn't exist.

**Solution**:
- Enable group sync in configuration
- Set `group_sync.create_missing_groups: true`

### Testing SCIM Endpoints

Use the SCIM 2.0 Test Tool:

```bash
# Test Service Provider Config
curl http://localhost:8080/scim/v2/ServiceProviderConfig \
  -H "Authorization: Bearer YOUR_SCIM_TOKEN"

# Test User Listing
curl http://localhost:8080/scim/v2/Users \
  -H "Authorization: Bearer YOUR_SCIM_TOKEN"

# Test User Creation
curl -X POST http://localhost:8080/scim/v2/Users \
  -H "Authorization: Bearer YOUR_SCIM_TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"test@example.com"}'
```

## Performance

### Benchmarks

- **User creation**: ~100ms p99 (including database write)
- **User lookup**: ~10ms p99 (cached)
- **Bulk operations**: ~500ms for 100 users
- **Rate limit**: 600 requests/min per organization (configurable)

### Optimization Tips

1. **Enable Bulk Operations**: Use bulk endpoint for provisioning multiple users
2. **Increase Rate Limits**: Adjust based on your needs
3. **Use Caching**: Redis caching for attribute lookups
4. **Index Database**: Ensure indexes on `email`, `external_id`

## Security Best Practices

1. **Use HTTPS**: Always use HTTPS in production
2. **Rotate Tokens**: Regularly rotate provisioning tokens
3. **IP Whitelist**: Restrict access to known IdP IPs
4. **Audit Logs**: Monitor provisioning logs for suspicious activity
5. **Least Privilege**: Use minimal scopes for tokens
6. **Soft Delete**: Enable soft delete to prevent accidental data loss

## Support

For issues, feature requests, or questions:

- **GitHub**: https://github.com/xraph/authsome/issues
- **Documentation**: https://docs.authsome.dev/plugins/scim
- **Email**: support@authsome.dev

## License

This plugin is part of the AuthSome framework and follows the same license.

---

**Made with ❤️ by the AuthSome Team**

