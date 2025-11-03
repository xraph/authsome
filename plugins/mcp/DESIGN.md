# MCP Plugin Design Document

## Overview

The MCP (Model Context Protocol) plugin exposes AuthSome's data and operations to AI assistants through Anthropic's standardized protocol. This enables AI-powered developer tools, security analysis, and administrative assistance.

## Architecture

### Plugin Structure
```
plugins/mcp/
├── plugin.go           # Plugin interface implementation
├── server.go           # MCP server (stdio/HTTP transport)
├── resources.go        # MCP resource handlers
├── tools.go            # MCP tool handlers
├── security.go         # Authorization & sanitization
├── handlers.go         # HTTP endpoints (if enabled)
└── config.go           # Plugin configuration
```

### Integration Pattern

Follows AuthSome's standard plugin lifecycle:
1. **Init**: Access Auth instance, setup MCP server
2. **RegisterHooks**: None needed (read-mostly operations)
3. **RegisterServiceDecorators**: None (doesn't modify core services)
4. **RegisterRoutes**: Optional HTTP transport endpoint
5. **Migrate**: No database schema required

## MCP Resources (Read-Only Context)

Resources provide read-only context to AI assistants:

### 1. `authsome://config`
- **Description**: Sanitized configuration (secrets removed)
- **URI**: `authsome://config/[section]`
- **Examples**: 
  - `authsome://config/general` - Mode, base path
  - `authsome://config/plugins` - Installed plugins
  - `authsome://config/rbac` - RBAC enforcement settings

### 2. `authsome://schema`
- **Description**: Database schema documentation
- **URI**: `authsome://schema/[entity]`
- **Examples**:
  - `authsome://schema/user` - User model fields
  - `authsome://schema/organization` - Organization structure

### 3. `authsome://routes`
- **Description**: Registered API routes
- **URI**: `authsome://routes/[method]`
- **Returns**: All routes, grouped by plugin/handler

### 4. `authsome://audit`
- **Description**: Audit log entries (with pagination)
- **URI**: `authsome://audit?user_id=X&limit=100`
- **Filters**: user_id, org_id, action, date_range

### 5. `authsome://users`
- **Description**: User list (respects RBAC)
- **URI**: `authsome://users?org_id=X&limit=50`
- **Sanitized**: No password hashes or sensitive fields

### 6. `authsome://organizations`
- **Description**: Organization list (SaaS mode only)
- **URI**: `authsome://organizations?limit=50`

### 7. `authsome://sessions`
- **Description**: Active/recent sessions
- **URI**: `authsome://sessions?user_id=X&active=true`

### 8. `authsome://rbac`
- **Description**: RBAC policies and roles
- **URI**: `authsome://rbac/policies` or `authsome://rbac/roles`

## MCP Tools (Interactive Operations)

Tools allow AI to perform controlled actions:

### Read-Only Tools

1. **`query_user`**
   - Find user by email/ID/username
   - Returns sanitized user data
   
2. **`list_sessions`**
   - List sessions with filters (user, org, active status)
   
3. **`check_permission`**
   - Verify if user has permission: `check_permission(user_id, action, resource)`
   
4. **`search_audit_logs`**
   - Search audit logs with semantic queries
   - Returns relevant security events
   
5. **`explain_route`**
   - Explain what a route does, required auth, parameters
   
6. **`validate_policy`**
   - Check if RBAC policy syntax is valid

### Write Tools (Require Authorization)

7. **`create_test_user`**
   - Create test users (development mode only)
   
8. **`revoke_session`**
   - Revoke specific session (requires admin permission)
   
9. **`rotate_api_key`**
   - Rotate API key (requires key ownership or admin)

## Security & Authorization

### Multi-Level Security

1. **Plugin Configuration**
   ```yaml
   plugins:
     mcp:
       enabled: true
       mode: "readonly"  # readonly | admin | development
       transport: "stdio" # stdio | http
       http_port: 9090
       authorization:
         require_api_key: true
         allowed_operations:
           - "query_user"
           - "search_audit_logs"
         admin_operations:  # Requires admin role
           - "create_test_user"
           - "revoke_session"
   ```

2. **Authorization Modes**
   - **readonly**: Only read resources and read-only tools
   - **admin**: + write tools (requires API key with admin role)
   - **development**: + dangerous operations like creating test data

3. **Data Sanitization**
   - **Always Remove**: password_hash, 2FA secrets, OAuth tokens
   - **Conditionally Remove**: email, phone (based on RBAC)
   - **Mask**: IP addresses (partial), API keys (show prefix only)

4. **Rate Limiting**
   - Use AuthSome's existing rate limiter
   - Default: 60 requests/minute per API key

5. **Audit Logging**
   - Log all MCP tool invocations
   - Include: operation, user context, outcome

## Transport Options

### 1. Stdio Transport (Default)
- **Use Case**: Local development, CLI tools
- **Security**: Inherits process permissions
- **Example**: `authsome mcp serve --stdio`

### 2. HTTP Transport
- **Use Case**: Remote AI assistants, team tools
- **Security**: API key required, HTTPS enforced
- **Endpoint**: `POST /api/mcp` (MCP protocol over HTTP)

## Configuration Examples

### Standalone Mode (Development)
```yaml
mode: standalone
plugins:
  mcp:
    enabled: true
    mode: development
    transport: stdio
    authorization:
      require_api_key: false  # Trust local process
```

### SaaS Mode (Production)
```yaml
mode: saas
plugins:
  mcp:
    enabled: true
    mode: readonly  # Strict read-only
    transport: http
    http_port: 9090
    authorization:
      require_api_key: true
      allowed_operations:
        - "query_user"
        - "search_audit_logs"
        - "check_permission"
    rate_limit:
      requests_per_minute: 30
```

## Implementation Phases

### Phase 1: Core Infrastructure (3-4 hours)
- [ ] Plugin skeleton (plugin.go)
- [ ] MCP server (stdio transport only)
- [ ] Basic resource: `authsome://config`
- [ ] Basic tool: `query_user`
- [ ] Security layer (sanitization)

### Phase 2: Essential Resources (2-3 hours)
- [ ] `authsome://schema`
- [ ] `authsome://routes`
- [ ] `authsome://users`
- [ ] `authsome://sessions`

### Phase 3: RBAC & Audit (2 hours)
- [ ] `authsome://rbac`
- [ ] `authsome://audit`
- [ ] Tools: `check_permission`, `search_audit_logs`

### Phase 4: HTTP Transport (2 hours)
- [ ] HTTP server implementation
- [ ] API key authentication
- [ ] CORS & security headers

### Phase 5: Admin Tools (1-2 hours)
- [ ] `create_test_user`
- [ ] `revoke_session`
- [ ] `rotate_api_key`

### Phase 6: Testing & Documentation (2 hours)
- [ ] Unit tests
- [ ] Integration tests with actual AI assistant
- [ ] CLI command: `authsome mcp serve`
- [ ] Usage examples

**Total Estimate: 12-15 hours**

## Usage Examples

### Example 1: Developer Debugging
```
AI: "Show me the user with email test@example.com"
→ Calls query_user tool
→ Returns sanitized user data

AI: "Why can't this user access /api/admin/users?"
→ Calls check_permission tool
→ Explains missing role or policy
```

### Example 2: Security Analysis
```
AI: "Show login attempts from IP 203.0.113.45 in the last 24h"
→ Reads authsome://audit resource with filters
→ Analyzes patterns, flags suspicious activity
```

### Example 3: API Integration Help
```
AI: "How do I create a user with 2FA enabled?"
→ Reads authsome://routes resource
→ Reads authsome://schema/user
→ Generates correct API call with example
```

## Benefits Over Direct Database Access

1. **RBAC Enforcement**: All queries respect AuthSome's permissions
2. **Multi-Tenancy**: Automatic org context, no data leakage
3. **Sanitization**: Sensitive data never exposed
4. **Audit Trail**: Every AI action logged
5. **Type Safety**: Structured data, not raw SQL
6. **Future Proof**: Schema changes don't break AI tools

## Open Questions

1. **Organization Context**: How should AI specify target org in SaaS mode?
   - Option A: Require `org_id` parameter in all tools
   - Option B: Create org-scoped MCP sessions
   
2. **Pagination**: MCP doesn't have built-in pagination. Use cursor-based?

3. **Real-time Updates**: Should resources support streaming/subscriptions?

4. **CLI Integration**: Should `authsome` include an MCP REPL?

## References

- [MCP Specification](https://modelcontextprotocol.io/)
- [Anthropic MCP SDK (Go)](https://github.com/mark3labs/mcp-go)
- [AuthSome Plugin Architecture](../../docs/plugins.md)

