# Impersonation Plugin

The Impersonation Plugin enables secure admin-to-user impersonation for troubleshooting and support purposes, with comprehensive audit logging, RBAC integration, and time-limited sessions.

## Features

✅ **Secure Impersonation**
- Admin can impersonate any user with proper permissions
- Cannot impersonate yourself
- Only one active impersonation per admin at a time

✅ **Time-Limited Sessions**
- Configurable default and maximum durations
- Auto-expiry after time limit
- Manual session termination

✅ **Permission-Based Access**
- RBAC integration with configurable permission
- Default permission: `impersonate:user`
- Permission checked before impersonation starts

✅ **Comprehensive Audit Logging**
- Every impersonation event logged to core audit system
- Dedicated impersonation audit table for detailed tracking
- Records: start, end, duration, reason, ticket number
- Optional: audit all actions during impersonation

✅ **Reason & Ticket Tracking**
- Required reason (min 10 characters by default)
- Optional ticket number for compliance
- Stored with audit trail

✅ **Visible Impersonation Indicator**
- Response headers indicate impersonation status
- UI can display banner: "⚠️ You are currently impersonating another user"
- Custom indicator message

✅ **Multi-Tenant Support**
- Organization-scoped impersonation
- Complete tenant isolation
- Org-specific audit logs

✅ **Auto-Cleanup**
- Background task expires old sessions
- Configurable cleanup interval (default: 15 minutes)
- Graceful shutdown

✅ **Regulatory Compliance**
- HIPAA break-glass support
- Complete audit trail
- Reason and ticket tracking
- Time-limited access

## Installation

Register the plugin when initializing AuthSome:

```go
import (
    "github.com/xraph/authsome"
    impersonationPlugin "github.com/xraph/authsome/plugins/impersonation"
)

func main() {
    auth := authsome.New(
        authsome.WithDatabase(db),
        authsome.WithForgeApp(app),
    )

    // Register impersonation plugin
    impersonationPlugin := impersonationPlugin.NewPlugin()
    auth.RegisterPlugin(impersonationPlugin)

    // Initialize
    auth.Initialize(context.Background())

    // Mount routes
    auth.Mount(app.Router(), "/api/auth")
}
```

## Configuration

Configuration can be provided via YAML config file or environment variables:

```yaml
auth:
  plugins:
    impersonation:
      # Time limits
      default_duration_minutes: 30      # Default session duration
      max_duration_minutes: 480         # Maximum allowed duration (8 hours)
      min_duration_minutes: 1           # Minimum allowed duration

      # Security
      require_reason: true              # Require reason for impersonation
      require_ticket: false             # Require ticket number
      min_reason_length: 10             # Minimum reason length

      # RBAC
      require_permission: true          # Check RBAC permission
      impersonate_permission: "impersonate:user"  # Required permission

      # Audit
      audit_all_actions: true           # Log all actions during impersonation

      # Auto-cleanup
      auto_cleanup_enabled: true        # Enable automatic cleanup
      cleanup_interval: "15m"           # Cleanup interval

      # UI Indicator
      show_indicator: true              # Show impersonation banner
      indicator_message: "⚠️ You are currently impersonating another user"

      # Webhooks (optional)
      webhook_on_start: true
      webhook_on_end: true
      webhook_urls:
        - "https://your-webhook.com/impersonation"
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `default_duration_minutes` | int | 30 | Default impersonation session duration |
| `max_duration_minutes` | int | 480 | Maximum allowed duration (8 hours) |
| `min_duration_minutes` | int | 1 | Minimum allowed duration |
| `require_reason` | bool | true | Require reason for impersonation |
| `require_ticket` | bool | false | Require support ticket number |
| `min_reason_length` | int | 10 | Minimum characters for reason |
| `require_permission` | bool | true | Check RBAC permission before allowing |
| `impersonate_permission` | string | "impersonate:user" | RBAC permission required |
| `audit_all_actions` | bool | true | Log every action during impersonation |
| `auto_cleanup_enabled` | bool | true | Enable automatic cleanup of expired sessions |
| `cleanup_interval` | duration | 15m | How often to run cleanup task |
| `show_indicator` | bool | true | Show impersonation indicator in UI |
| `indicator_message` | string | "⚠️ You are..." | Message shown during impersonation |

## API Endpoints

### Start Impersonation

**POST** `/impersonation/start`

Start impersonating a user.

**Request Body:**
```json
{
  "organization_id": "org_abc123",
  "impersonator_id": "user_admin123",
  "target_user_id": "user_target456",
  "reason": "Customer reported login issue, investigating - Ticket #12345",
  "ticket_number": "SUPPORT-12345",
  "duration_minutes": 60
}
```

**Response:**
```json
{
  "impersonation_id": "imp_xyz789",
  "session_id": "ses_new123",
  "session_token": "eyJhbGc...",
  "expires_at": "2024-01-15T15:30:00Z",
  "message": "Impersonating user@example.com until 2024-01-15T15:30:00Z"
}
```

**Errors:**
- `400` - Invalid request, invalid reason, invalid duration
- `403` - Permission denied
- `409` - Already impersonating another user

### End Impersonation

**POST** `/impersonation/end`

End an active impersonation session.

**Request Body:**
```json
{
  "impersonation_id": "imp_xyz789",
  "organization_id": "org_abc123",
  "impersonator_id": "user_admin123",
  "reason": "manual"
}
```

**Response:**
```json
{
  "success": true,
  "impersonation_id": "imp_xyz789",
  "ended_at": "2024-01-15T14:45:00Z",
  "message": "Impersonation session ended successfully"
}
```

### Get Impersonation Session

**GET** `/impersonation/:id?org_id=org_abc123`

Retrieve details of an impersonation session.

**Response:**
```json
{
  "id": "imp_xyz789",
  "organization_id": "org_abc123",
  "impersonator_id": "user_admin123",
  "target_user_id": "user_target456",
  "reason": "Customer reported login issue...",
  "ticket_number": "SUPPORT-12345",
  "active": true,
  "expires_at": "2024-01-15T15:30:00Z",
  "created_at": "2024-01-15T14:30:00Z",
  "impersonator_email": "admin@example.com",
  "impersonator_name": "Admin User",
  "target_email": "user@example.com",
  "target_name": "Target User"
}
```

### List Impersonation Sessions

**GET** `/impersonation?org_id=org_abc123&active_only=true&limit=20&offset=0`

List impersonation sessions with filters.

**Query Parameters:**
- `org_id` (required) - Organization ID
- `impersonator_id` (optional) - Filter by impersonator
- `target_user_id` (optional) - Filter by target user
- `active_only` (optional) - Only show active sessions
- `limit` (optional) - Page size (default: 20)
- `offset` (optional) - Page offset (default: 0)

**Response:**
```json
{
  "sessions": [
    {
      "id": "imp_xyz789",
      "organization_id": "org_abc123",
      "impersonator_id": "user_admin123",
      "target_user_id": "user_target456",
      "reason": "Customer reported login issue...",
      "active": true,
      "expires_at": "2024-01-15T15:30:00Z",
      "created_at": "2024-01-15T14:30:00Z"
    }
  ],
  "total": 15,
  "limit": 20,
  "offset": 0
}
```

### Verify Impersonation

**POST** `/impersonation/verify`

Check if a session is an impersonation session.

**Request Body:**
```json
{
  "session_id": "ses_new123"
}
```

**Response:**
```json
{
  "is_impersonating": true,
  "impersonation_id": "imp_xyz789",
  "impersonator_id": "user_admin123",
  "target_user_id": "user_target456",
  "expires_at": "2024-01-15T15:30:00Z"
}
```

### List Audit Events

**GET** `/impersonation/audit?org_id=org_abc123&limit=50&offset=0`

List impersonation audit events.

**Query Parameters:**
- `org_id` (required) - Organization ID
- `impersonation_id` (optional) - Filter by impersonation session
- `event_type` (optional) - Filter by event type (started, ended, action_performed)
- `limit` (optional) - Page size (default: 50)
- `offset` (optional) - Page offset (default: 0)

**Response:**
```json
{
  "events": [
    {
      "id": "evt_123",
      "impersonation_id": "imp_xyz789",
      "organization_id": "org_abc123",
      "event_type": "started",
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "details": {
        "target_user_id": "user_target456",
        "impersonator_id": "user_admin123",
        "reason": "Customer reported login issue...",
        "duration_minutes": "60"
      },
      "created_at": "2024-01-15T14:30:00Z"
    }
  ],
  "total": 10,
  "limit": 50,
  "offset": 0
}
```

## Middleware

The plugin provides middleware for impersonation context and protection.

### Add Impersonation Context

```go
// Get the middleware from the plugin
impersonationPlugin := /* your plugin instance */
middleware := impersonationPlugin.GetMiddleware()

// Add to your routes
router.Use(middleware.Handle())
```

This middleware:
- Checks if current session is impersonating
- Adds impersonation context to request
- Sets response headers: `X-Impersonating`, `X-Impersonator-ID`, `X-Target-User-ID`

### Protect Sensitive Endpoints

```go
// Prevent impersonation on sensitive endpoints
router.POST("/api/settings/delete-account", 
    middleware.RequireNoImpersonation(),
    handler.DeleteAccount,
)
```

### Require Impersonation

```go
// Only allow during impersonation
router.GET("/api/debug/impersonation-info",
    middleware.RequireImpersonation(),
    handler.GetImpersonationInfo,
)
```

### Helper Functions

```go
import "github.com/xraph/authsome/plugins/impersonation"

func MyHandler(c forge.Context) error {
    // Check if impersonating
    if impersonation.IsImpersonating(c) {
        // Handle impersonation case
    }

    // Get impersonator ID
    impersonatorID := impersonation.GetImpersonatorID(c)
    if impersonatorID != nil {
        // Use impersonator ID
    }

    // Get target user ID
    targetUserID := impersonation.GetTargetUserID(c)

    // Get full context
    impCtx := impersonation.GetImpersonationContext(c)
    if impCtx != nil && impCtx.IsImpersonating {
        // Access impersonation details
    }

    return nil
}
```

## RBAC Integration

The plugin integrates with AuthSome's RBAC system. You need to grant the impersonation permission to admins:

```go
// Create a policy that allows impersonation
policy := "role:admin:impersonate:user on user:*"
rbacService.CreatePolicy(ctx, policy)

// Or using the permissions plugin
permissionService.CreatePolicy(ctx, &permissions.Policy{
    Name: "admin-impersonation",
    Expression: `
        subject.role == "admin" && 
        action == "impersonate:user" && 
        resource.type == "user"
    `,
})
```

## UI Integration

### Show Impersonation Banner

The middleware sets response headers when impersonating. Use these in your frontend:

```javascript
// Check response headers
const isImpersonating = response.headers.get('X-Impersonating') === 'true';
const impersonatorId = response.headers.get('X-Impersonator-ID');
const targetUserId = response.headers.get('X-Target-User-ID');

// Show banner
if (isImpersonating) {
    showBanner('⚠️ You are currently impersonating another user');
}
```

### End Impersonation Button

```javascript
async function endImpersonation(impersonationId) {
    const response = await fetch('/impersonation/end', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            impersonation_id: impersonationId,
            organization_id: currentOrgId,
            impersonator_id: currentUserId,
            reason: 'manual'
        })
    });

    if (response.ok) {
        // Redirect to admin dashboard
        window.location.href = '/dashboard';
    }
}
```

## Database Schema

### Impersonation Sessions Table

```sql
CREATE TABLE impersonation_sessions (
    id VARCHAR(20) PRIMARY KEY,
    organization_id VARCHAR(20) NOT NULL,
    impersonator_id VARCHAR(20) NOT NULL,
    target_user_id VARCHAR(20) NOT NULL,
    original_session VARCHAR(20),
    new_session_id VARCHAR(20),
    reason TEXT NOT NULL,
    ticket_number VARCHAR(100),
    ip_address VARCHAR(45),
    user_agent TEXT,
    metadata JSONB,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    end_reason VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
    FOREIGN KEY (impersonator_id) REFERENCES users(id),
    FOREIGN KEY (target_user_id) REFERENCES users(id)
);

CREATE INDEX idx_impersonation_org ON impersonation_sessions(organization_id);
CREATE INDEX idx_impersonation_impersonator ON impersonation_sessions(impersonator_id);
CREATE INDEX idx_impersonation_target ON impersonation_sessions(target_user_id);
CREATE INDEX idx_impersonation_active ON impersonation_sessions(active, expires_at) WHERE active = TRUE;
CREATE INDEX idx_impersonation_session ON impersonation_sessions(new_session_id) WHERE new_session_id IS NOT NULL;
```

### Impersonation Audit Table

```sql
CREATE TABLE impersonation_audit (
    id VARCHAR(20) PRIMARY KEY,
    impersonation_id VARCHAR(20) NOT NULL,
    organization_id VARCHAR(20) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    action VARCHAR(100),
    resource VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (impersonation_id) REFERENCES impersonation_sessions(id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE INDEX idx_impersonation_audit_session ON impersonation_audit(impersonation_id);
CREATE INDEX idx_impersonation_audit_org ON impersonation_audit(organization_id);
CREATE INDEX idx_impersonation_audit_event ON impersonation_audit(event_type);
CREATE INDEX idx_impersonation_audit_created ON impersonation_audit(created_at DESC);
```

## Security Considerations

1. **Permission Control**: Always require RBAC permission for impersonation
2. **Audit Everything**: Enable comprehensive audit logging
3. **Time Limits**: Set reasonable max duration (default 8 hours)
4. **Reason Required**: Enforce reason with minimum length
5. **Ticket Tracking**: Consider requiring ticket numbers for compliance
6. **Protect Sensitive Operations**: Use `RequireNoImpersonation()` middleware on critical endpoints
7. **Auto-Expiry**: Enable auto-cleanup to prevent stale sessions
8. **Visible Indicators**: Always show impersonation status in UI
9. **Single Session**: Enforce one active impersonation per admin
10. **Regular Audits**: Review impersonation logs regularly

## Compliance

### HIPAA Break-Glass

The plugin supports HIPAA break-glass requirements:

- ✅ Reason tracking (required)
- ✅ Complete audit trail
- ✅ Time-limited access
- ✅ Ticket number tracking (optional)
- ✅ All actions logged
- ✅ Cannot be retroactively hidden
- ✅ Regular review capability

### SOC 2

- ✅ Access logging
- ✅ Permission-based access
- ✅ Audit trail retention
- ✅ Time-bounded sessions

## Troubleshooting

### Impersonation fails with "permission denied"

Check RBAC policies:
```go
// Verify user has impersonate permission
hasPermission, _ := rbacService.Evaluate(ctx, &rbac.EvaluateRequest{
    Principal: rbac.Principal{Type: "user", ID: adminUserID},
    Action: "impersonate:user",
    Resource: rbac.Resource{Type: "user", ID: targetUserID},
})
```

### Cleanup not running

Check configuration:
```yaml
auto_cleanup_enabled: true
cleanup_interval: "15m"
```

Check logs for errors from cleanup goroutine.

### Sessions not expiring

Cleanup runs on interval. Force expiry:
```go
service := plugin.GetService()
count, err := service.ExpireSessions(ctx)
fmt.Printf("Expired %d sessions\n", count)
```

## Example Usage

See `examples/impersonation/` for complete examples:

- Basic impersonation flow
- RBAC integration
- UI integration with React
- Audit log reporting
- Compliance reporting

## License

MIT License - see LICENSE file for details

