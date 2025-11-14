# Admin Plugin

Cross-cutting administrative operations for AuthSome platform management.

## Purpose

The admin plugin provides **platform-level administrative APIs** that span multiple services:
- User lifecycle management
- Security operations (ban, impersonation)
- Session oversight
- Aggregated statistics
- Centralized audit logs

## Scope

### What belongs in admin plugin (cross-cutting):
- User CRUD operations
- User impersonation (security-sensitive)
- Session management across all users
- Platform-wide statistics
- Centralized audit log access
- Role assignments (RBAC integration)

### What belongs in individual plugins (plugin-specific):
- OAuth provider configuration → social plugin
- 2FA settings management → mfa plugin
- Passkey configuration → passkey plugin
- Email template management → notification plugin
- Rate limit overrides → respective plugins

## Architecture Decision

Per architecture decision 1b, 2a, 3a:
- Admin plugin handles cross-cutting operations
- Individual plugins expose their own admin endpoints for plugin-specific features
- Impersonation remains centralized in admin plugin (security-sensitive)

## Usage

```go
// Admin plugin is typically used internally, not registered as a plugin
adminSvc := admin.NewService(config, userSvc, sessionSvc, rbacSvc, auditSvc, banSvc)
adminHandler := admin.NewHandler(adminSvc)

// Mount admin routes
router.POST("/admin/users", adminHandler.CreateUser)
router.GET("/admin/users", adminHandler.ListUsers)
// ... other cross-cutting operations
```

## Available Endpoints

### User Management
- `POST /admin/users` - Create a new user
- `GET /admin/users` - List users with filtering and pagination
- `DELETE /admin/users/:id` - Delete a user

### Security Operations
- `POST /admin/users/:id/ban` - Ban a user
- `POST /admin/users/:id/unban` - Unban a user
- `POST /admin/users/:id/impersonate` - Impersonate a user (security-sensitive)

### Session Management
- `GET /admin/sessions` - List all active sessions
- `DELETE /admin/sessions/:id` - Revoke a session

### Role Management
- `POST /admin/users/:id/role` - Assign role to user

### Statistics & Monitoring
- `GET /admin/stats` - Get platform-wide statistics
- `GET /admin/audit-logs` - View centralized audit logs

## Plugin-Specific Admin Endpoints Pattern

Individual plugins should expose their own admin endpoints:

```go
// Example: MFA plugin admin endpoints
router.GET("/mfa/admin/policies", mfaHandler.ListPolicies)       // Plugin-specific
router.PUT("/mfa/admin/policies/:id", mfaHandler.UpdatePolicy)    // Plugin-specific

// Example: Social plugin admin endpoints  
router.POST("/social/admin/providers", socialHandler.AddProvider) // Plugin-specific
router.GET("/social/admin/providers", socialHandler.ListProviders) // Plugin-specific
```

See plugin-specific documentation for admin endpoint implementations.

## Security & Permissions

### Role-Based Access Control

The admin plugin registers the following roles and permissions during initialization:

**Admin Role** (`admin`):
- Priority: 80 (platform-level)
- Inherits from: `member`
- Permissions:
  - `admin:user:create on admin:*` - Create users
  - `admin:user:read on admin:*` - List and view users
  - `admin:user:update on admin:*` - Update user information
  - `admin:user:delete on admin:*` - Delete users
  - `admin:user:ban on admin:*` - Ban/unban users
  - `admin:session:read on admin:*` - View sessions
  - `admin:session:revoke on admin:*` - Revoke sessions
  - `admin:role:assign on admin:*` - Assign roles
  - `admin:stats:read on admin:*` - View statistics
  - `admin:audit:read on admin:*` - View audit logs

**Superadmin Role** (`superadmin`):
- Priority: 100 (highest platform-level)
- Inherits from: `admin`
- Additional Permissions:
  - `admin:user:impersonate on admin:*` - Impersonate users (security-sensitive)
  - `* on admin:*` - Full wildcard access to admin operations

### Permission Checks

All admin endpoints:
- Check permissions via RBAC before performing operations
- Validate app/org context
- Log all administrative actions to audit service
- Use rate limiting
- Require CSRF protection for state-changing operations

### Bootstrapping

Permissions are automatically registered during plugin initialization via the `RegisterRoles` interface:

```go
// Automatic during server startup
plugin := admin.NewPlugin()
plugin.RegisterRoles(roleRegistry)
```

## See Also

- [Plugin Admin Endpoint Guidelines](../../docs/PLUGIN_ADMIN_ENDPOINTS.md) - How to add admin endpoints to your plugin
- [RBAC Plugin](../permissions/README.md) - Role-based access control
- [Audit Plugin](../../core/audit/README.md) - Audit logging

