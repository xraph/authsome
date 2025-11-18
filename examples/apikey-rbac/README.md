# API Key RBAC Integration Example

This example demonstrates the hybrid RBAC permission system for API keys in AuthSome.

## Features Demonstrated

1. **API Key Types** (pk/sk/rk) with default scopes
2. **RBAC Role Assignment** to API keys
3. **Permission Delegation** (inherit creator's permissions)
4. **Effective Permission Calculation** (hybrid approach)
5. **Middleware Usage** (RequireRBACPermission, RequireCanAccess)
6. **Scope-to-RBAC Mapping** (backward compatibility)

## Three Permission Patterns

### 1. Independent API Key (Default - Clerk Style)
```bash
# Create API key with its own roles - most secure
curl -X POST http://localhost:8080/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Backend Service Key",
    "keyType": "sk",
    "scopes": ["analytics:write"],
    "roleIDs": ["editor_role_id", "analytics_role_id"]
  }'
```

The key has ONLY the roles assigned, independent of creator.

### 2. Delegated Permissions (GitHub PAT Style)
```bash
# Create API key that inherits creator's permissions
curl -X POST http://localhost:8080/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Personal Access Token",
    "keyType": "rk",
    "delegateUserPermissions": true
  }'
```

The key has its own scopes + creator's full permissions.

### 3. User Impersonation (Advanced)
```bash
# Create API key that acts as a specific user
curl -X POST http://localhost:8080/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Support Tool Key",
    "keyType": "sk",
    "roleIDs": ["user_impersonation_role_id"],
    "impersonateUserID": "target_user_id"
  }'
```

Must have "user-impersonation" permission. Acts as the target user.

## Assigning Roles to API Keys

```bash
# Assign a role to an existing API key
curl -X POST http://localhost:8080/api-keys/{key_id}/roles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {session_token}" \
  -d '{
    "roleID": "editor_role_id"
  }'

# Get all roles for an API key
curl -X GET http://localhost:8080/api-keys/{key_id}/roles \
  -H "Authorization: Bearer {session_token}"

# Get effective permissions (includes delegated)
curl -X GET http://localhost:8080/api-keys/{key_id}/permissions \
  -H "Authorization: Bearer {session_token}"

# Unassign a role
curl -X DELETE http://localhost:8080/api-keys/{key_id}/roles/{role_id} \
  -H "Authorization: Bearer {session_token}"
```

## Using RBAC Middleware

```go
// Require specific RBAC permission (strict - only RBAC)
app.GET("/users",
    auth.RequireAPIKey(),
    auth.RequireRBACPermission("view", "users"),
    handleListUsers,
)

// Flexible check - accepts scope OR RBAC (recommended)
app.POST("/users",
    auth.RequireAPIKey(),
    auth.RequireCanAccess("create", "users"),
    handleCreateUser,
)

// Require any of several permissions
app.GET("/dashboard",
    auth.RequireAuth(),
    auth.RequireAnyPermission("view:analytics", "view:reports"),
    handleDashboard,
)

// Require all permissions
app.POST("/critical-operation",
    auth.RequireAuth(),
    auth.RequireAllPermissions("edit:config", "admin:full"),
    handleCriticalOp,
)
```

## Runtime Permission Checks

```go
func handleAction(c forge.Context) error {
    authCtx, _ := contexts.GetAuthContext(c.Request().Context())
    
    // Check RBAC permission
    if !authCtx.HasRBACPermission("delete", "users") {
        return c.JSON(403, "Access denied")
    }
    
    // Flexible check (scopes OR RBAC)
    if !authCtx.CanAccess("edit", "posts") {
        return c.JSON(403, "Access denied")
    }
    
    // Check if delegating permissions
    if authCtx.IsDelegatingCreatorPermissions() {
        // Special handling for delegated keys
    }
    
    // Check if impersonating
    if authCtx.IsImpersonating() {
        impersonatedUserID := authCtx.GetImpersonatedUserID()
        // Audit impersonation
    }
    
    // ... perform action ...
}
```

## Permission Priority

API Key permission check follows this order:

1. **API Key's own permissions** (scopes + roles) ← Always checked
2. **If delegation ON**: Add creator's permissions ← Optional
3. **If user session present**: Add session user permissions ← Context-dependent
4. **If impersonation**: Use target user permissions ← Advanced

Result: Union of all applicable permissions

## Security Considerations

### Delegation Safety
- **Disabled by default** - opt-in only
- Only enable for trusted use cases (Personal Access Tokens)
- Document security implications clearly

### Impersonation Safety
- **Requires explicit permission** ("user-impersonation")
- All impersonation attempts are audited
- Logs which user is being impersonated

### Best Practices
1. Use **independent API keys** (no delegation) by default
2. Enable delegation only for trusted PATs
3. Require MFA for impersonation permission grants
4. Regularly audit API key permissions
5. Rotate keys after privilege escalation

## Testing the Implementation

Run the integration tests:

```bash
go test ./examples/apikey-rbac/...
```

## Migration from Scopes to RBAC

Use the scope mapper for backward compatibility:

```go
// Convert existing scopes to RBAC permissions
scopes := []string{"users:read", "users:write", "admin:full"}
rbacPerms := apikey.ConvertScopesToRBAC(scopes)

// Suggest appropriate role based on scopes
suggestedRole := apikey.GenerateSuggestedRole(scopes)
```

## Complete Flow Example

1. Create an organization with roles
2. Create an API key with role assignments
3. Make authenticated requests
4. Check permissions in handlers
5. View effective permissions

See `main.go` for a complete working example.

