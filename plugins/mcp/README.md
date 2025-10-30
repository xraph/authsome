# MCP Plugin for AuthSome

The MCP (Model Context Protocol) plugin exposes AuthSome's data and operations to AI assistants through Anthropic's standardized protocol.

## What is MCP?

Model Context Protocol (MCP) is an open protocol developed by Anthropic that standardizes how AI assistants interact with external systems. It provides:

- **Resources**: Read-only context (configuration, schemas, data)
- **Tools**: Interactive operations (queries, checks, modifications)
- **Security**: Built-in authorization and sanitization
- **Transport-agnostic**: Works over stdio, HTTP, or SSE

## Benefits for AuthSome

### 1. Developer Experience
- **Natural language queries**: "Show me users created in the last week"
- **Debug authentication flows**: "Why did user@example.com's login fail?"
- **Schema exploration**: "What fields does the User model have?"
- **API documentation**: Generates integration code from actual configuration

### 2. Security & Auditing
- **AI-assisted analysis**: "Find suspicious login patterns from IP 203.0.113.45"
- **RBAC verification**: "Does user X have permission to delete organization Y?"
- **Audit log search**: Semantic search across security events

### 3. Administrative Tasks
- **User management**: "Create a test user in organization XYZ"
- **Session control**: "Show all active sessions for user@example.com"
- **Multi-tenant operations**: "List organizations with failed payment status"

## Architecture

```
┌─────────────────┐
│  AI Assistant   │
│  (Claude, etc)  │
└────────┬────────┘
         │ MCP Protocol (JSON-RPC)
         │
┌────────▼────────┐
│   MCP Server    │
│  (this plugin)  │
├─────────────────┤
│  Resources      │  ← Read-only context
│  - config       │
│  - schema       │
│  - routes       │
│  - users        │
│  - sessions     │
│  - audit logs   │
├─────────────────┤
│  Tools          │  ← Interactive operations
│  - query_user   │
│  - check_perm   │
│  - search_audit │
├─────────────────┤
│  Security       │
│  - Sanitization │
│  - RBAC         │
│  - Rate limit   │
└────────┬────────┘
         │
┌────────▼────────┐
│   AuthSome      │
│   Core Services │
└─────────────────┘
```

## Configuration

### Standalone Mode (Development)
```yaml
mode: standalone

plugins:
  mcp:
    enabled: true
    mode: development  # Allow test data creation
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
        - query_user
        - search_audit_logs
        - check_permission
    rate_limit:
      requests_per_minute: 30
```

## Operation Modes

### 1. `readonly` (Default, Safest)
- Only read operations and read-only tools
- No data modification
- Recommended for production

**Allowed:**
- `query_user` - Find users
- `list_sessions` - View sessions
- `check_permission` - Verify RBAC
- `search_audit_logs` - Search audit trail
- `explain_route` - Get API documentation
- `validate_policy` - Check RBAC syntax

### 2. `admin` (Controlled Write)
- Read operations + administrative writes
- Requires API key with admin role
- For authorized administrators

**Additional:**
- `revoke_session` - Terminate sessions
- `rotate_api_key` - Rotate keys

### 3. `development` (Full Access)
- All operations including dangerous ones
- Create test data
- **NEVER use in production**

**Additional:**
- `create_test_user` - Generate test users
- `create_test_org` - Generate test organizations
- `seed_data` - Populate test database

## Transport Options

### Stdio (Default)
- Uses stdin/stdout for communication
- Perfect for local development
- Inherits process permissions
- No network exposure

```bash
# Start MCP server with stdio
authsome mcp serve --config=config.yaml

# AI assistant connects via stdio
claude-desktop mcp://authsome-mcp-serve
```

### HTTP
- REST API over HTTPS
- For remote AI assistants
- Requires API key authentication
- Can be rate-limited

```bash
# Start HTTP MCP server
authsome mcp serve --config=config.yaml --transport=http --port=9090

# AI assistant connects via HTTP
curl -X POST http://localhost:9090/api/mcp \
  -H "X-API-Key: your-api-key" \
  -d '{"method": "resources/list"}'
```

## CLI Usage

### Start MCP Server

```bash
# Read-only mode (default)
authsome mcp serve --config=config.yaml

# Admin mode
authsome mcp serve --config=config.yaml --mode=admin

# Development mode (INSECURE)
authsome mcp serve --config=config.yaml --mode=development --no-auth

# HTTP transport
authsome mcp serve --config=config.yaml --transport=http --port=9090
```

### Flags

- `--config=PATH` - Configuration file (required)
- `--mode=MODE` - Operation mode: `readonly`, `admin`, `development`
- `--transport=TYPE` - Transport: `stdio`, `http`
- `--port=PORT` - HTTP port (default: 9090)
- `--no-auth` - Disable API key requirement (INSECURE)

## Available Resources

Resources provide read-only context to AI assistants.

### `authsome://config`
Sanitized configuration (secrets removed)

```json
{
  "mode": "saas",
  "base_path": "/api/auth",
  "plugins": ["multitenancy", "twofa", "sso"]
}
```

### `authsome://schema/[entity]`
Database schema documentation

```json
{
  "user": {
    "fields": {
      "id": {"type": "string", "description": "User UUID"},
      "email": {"type": "string", "description": "Email address"},
      "name": {"type": "string", "description": "Display name"}
    }
  }
}
```

### `authsome://routes`
Registered API routes

```json
{
  "routes": [
    {
      "method": "POST",
      "path": "/api/auth/signup",
      "description": "User registration",
      "auth": "none"
    }
  ]
}
```

### `authsome://users?org_id=X&limit=50`
User list (sanitized, respects RBAC)

### `authsome://sessions?user_id=X&active=true`
Active/recent sessions

### `authsome://audit?action=login&limit=100`
Audit log entries

### `authsome://rbac/policies`
RBAC policies and roles

## Available Tools

Tools allow AI to perform operations.

### Read-Only Tools

#### `query_user`
Find user by email/ID/username

```json
{
  "name": "query_user",
  "arguments": {
    "email": "test@example.com"
  }
}
```

Returns sanitized user data (no password hashes).

#### `check_permission`
Verify if user has permission

```json
{
  "name": "check_permission",
  "arguments": {
    "user_id": "user-123",
    "action": "delete",
    "resource": "organization:org-456"
  }
}
```

Returns boolean + explanation.

#### `search_audit_logs`
Search audit logs

```json
{
  "name": "search_audit_logs",
  "arguments": {
    "query": "failed login attempts from suspicious IP",
    "limit": 50
  }
}
```

### Admin Tools (Require Admin Mode)

#### `revoke_session`
Revoke specific session

```json
{
  "name": "revoke_session",
  "arguments": {
    "session_id": "session-789"
  }
}
```

#### `rotate_api_key`
Rotate API key

```json
{
  "name": "rotate_api_key",
  "arguments": {
    "key_id": "key-101"
  }
}
```

### Development Tools (Require Development Mode)

#### `create_test_user`
Create test user

```json
{
  "name": "create_test_user",
  "arguments": {
    "email": "test@example.com",
    "organization_id": "org-123"
  }
}
```

## Security

### Data Sanitization

**Always Removed:**
- `password_hash`
- `twofa_secret`
- `backup_codes`
- `oauth_tokens`
- API key secrets

**Conditionally Masked (readonly mode):**
- Email addresses (partial)
- Phone numbers (partial)
- IP addresses (partial)

### Authorization Layers

1. **Transport Security**
   - Stdio: Process isolation
   - HTTP: API key + HTTPS

2. **Operation Authorization**
   - Mode-based operation filtering
   - Role-based access (admin operations)

3. **RBAC Integration**
   - Resource queries respect AuthSome's RBAC
   - Multi-tenant isolation enforced

4. **Rate Limiting**
   - Configurable requests per minute
   - Uses AuthSome's rate limiter

5. **Audit Logging**
   - All tool invocations logged
   - Includes user context and outcome

## Usage Examples

### Example 1: Developer Debugging

**Scenario**: User can't log in

```
AI: "Show me the user with email test@example.com"
→ Tool: query_user(email="test@example.com")
→ Result: User found, emailVerified=false

AI: "That explains it - email not verified. Check recent verification emails."
→ Resource: authsome://audit?user_id=X&action=email.sent
```

### Example 2: Security Analysis

**Scenario**: Suspicious activity detected

```
AI: "Show login attempts from IP 203.0.113.45 in last 24h"
→ Resource: authsome://audit?ip=203.0.113.45&action=login.attempt
→ Analysis: 47 failed attempts, different user accounts
→ AI: "Credential stuffing attack detected. Recommend blocking IP."
```

### Example 3: API Integration Help

**Scenario**: Developer needs to integrate

```
AI: "How do I create a user with 2FA enabled?"
→ Resource: authsome://routes (find user creation endpoint)
→ Resource: authsome://schema/user (get required fields)
→ AI generates:
   POST /api/auth/signup
   {
     "email": "user@example.com",
     "password": "...",
     "twoFactorEnabled": true
   }
```

## Testing

### Manual Testing

```bash
# Start MCP server
authsome mcp serve --config=test-config.yaml --mode=development

# In another terminal, send MCP requests
echo '{"jsonrpc":"2.0","id":1,"method":"resources/list","params":{}}' | \
  authsome mcp serve --config=test-config.yaml
```

### Integration Testing

```go
func TestMCPPlugin(t *testing.T) {
    // Create test auth instance
    auth := authsome.New(...)
    
    // Register MCP plugin
    mcpPlugin := mcp.NewPlugin(mcp.DefaultConfig())
    auth.RegisterPlugin(mcpPlugin)
    
    // Initialize
    auth.Initialize(context.Background())
    
    // Test resource reading
    server := mcpPlugin.GetServer()
    content, err := server.ReadResource(ctx, "authsome://config")
    assert.NoError(t, err)
    assert.Contains(t, content, "mode")
}
```

## Troubleshooting

### "API key required but not provided"
- Solution: Add `--no-auth` flag (development only) or provide valid API key

### "Operation not allowed"
- Solution: Change mode (e.g., `--mode=admin` for admin operations)

### "Resource not found"
- Solution: Check resource URI syntax, ensure plugin initialized

### "RBAC service not available"
- Solution: Ensure AuthSome fully initialized before starting MCP

## Performance Considerations

- **Caching**: Resource responses can be cached (use ETags in HTTP mode)
- **Pagination**: Large result sets are automatically paginated
- **Rate Limiting**: Prevents abuse, configurable per deployment
- **Connection Pooling**: HTTP mode reuses database connections

## Roadmap

### v0.2 (Next Release)
- [ ] HTTP transport implementation
- [ ] Streaming resource updates (Server-Sent Events)
- [ ] More tools: `list_sessions`, `search_audit_logs`
- [ ] Custom resource filters via query params

### v0.3 (Future)
- [ ] WebSocket transport
- [ ] Real-time subscription support
- [ ] Organization-scoped MCP sessions
- [ ] Plugin for custom resources/tools

## Contributing

To add new resources or tools:

1. **Create Resource Handler**
```go
type MyResource struct{}

func (r *MyResource) Describe() ResourceDescription {
    return ResourceDescription{
        URI: "authsome://myresource",
        Name: "My Resource",
        Description: "...",
    }
}

func (r *MyResource) Read(ctx context.Context, uri string, plugin *Plugin) (string, error) {
    // Implementation
}
```

2. **Register in server.go**
```go
func (s *Server) registerResources() {
    s.resources.Register("authsome://myresource", &MyResource{})
}
```

3. **Add tests**
4. **Update documentation**

## References

- [MCP Specification](https://modelcontextprotocol.io/)
- [Anthropic MCP Documentation](https://docs.anthropic.com/mcp)
- [AuthSome Plugin Architecture](../../docs/plugins.md)

## License

Same as AuthSome project

