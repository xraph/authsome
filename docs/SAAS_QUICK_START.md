# Quick Start: Integrate AuthSome RBAC with Your SaaS

## 5-Minute Integration

### Step 1: Install Client Library

```bash
# Go
go get github.com/xraph/authsome/clients/go

# TypeScript/Node.js
npm install @authsome/client

# Rust
cargo add authsome-client
```

### Step 2: Initialize Client

```go
// main.go
import authsome "github.com/xraph/authsome/clients/go"

client := authsome.NewClient(authsome.ClientConfig{
    BaseURL: "https://auth.yourdomain.com",
    APIKey:  "your-api-key",
})
```

### Step 3: Create Organizations

```bash
# Create organization for each customer
curl -X POST https://auth.yourdomain.com/api/organizations \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "name": "Acme Corp",
    "slug": "acme"
  }'
```

### Step 4: Define Roles

```go
// Define roles for your SaaS
roles := []authsome.Role{
    {
        Name: "admin",
        Permissions: []string{
            "projects:*",
            "members:*",
            "settings:*",
        },
    },
    {
        Name: "member",
        Permissions: []string{
            "projects:read",
            "projects:create",
        },
    },
    {
        Name: "viewer",
        Permissions: []string{
            "projects:read",
        },
    },
}

for _, role := range roles {
    client.RBAC.CreateRole(ctx, "org_acme", &role)
}
```

### Step 5: Add Permission Middleware

```go
func RequirePermission(action, resource string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            session := getSession(r)
            orgID := r.Header.Get("X-Organization-ID")
            
            // Check permission using AuthSome
            allowed, _ := client.RBAC.Can(r.Context(), &authsome.PermissionCheck{
                UserID:   session.UserID,
                Action:   action,
                Resource: resource,
                OrgID:    orgID,
            })
            
            if !allowed {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Protect your routes
http.Handle("/api/projects", 
    RequirePermission("read", "projects")(projectsHandler))
http.Handle("/api/projects/create", 
    RequirePermission("create", "projects")(createHandler))
```

## Real-World Example

```go
package main

import (
    "context"
    "net/http"
    
    authsome "github.com/xraph/authsome/clients/go"
)

func main() {
    // Initialize AuthSome client
    auth := authsome.NewClient(authsome.ClientConfig{
        BaseURL: "https://auth.myapp.com",
        APIKey:  "sk_live_xxx",
    })
    
    // Your SaaS routes
    http.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
        // Get session
        session, err := auth.Session.FromRequest(r)
        if err != nil {
            http.Error(w, "Unauthorized", 401)
            return
        }
        
        // Get organization from request
        orgID := r.Header.Get("X-Organization-ID")
        
        // Check permission
        canRead, err := auth.RBAC.Can(r.Context(), &authsome.PermissionCheck{
            UserID:   session.UserID,
            Action:   "read",
            Resource: "projects",
            OrgID:    orgID,
        })
        
        if err != nil || !canRead {
            http.Error(w, "Forbidden", 403)
            return
        }
        
        // Return projects
        projects := getProjectsForOrg(orgID)
        json.NewEncoder(w).Encode(projects)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## Organization Context

### Method 1: Header (Recommended)

```bash
curl https://api.yoursaas.com/projects \
  -H "Authorization: Bearer session_token" \
  -H "X-Organization-ID: org_acme"
```

### Method 2: Subdomain

```bash
# AuthSome automatically extracts org from subdomain
curl https://acme.yoursaas.com/api/projects \
  -H "Authorization: Bearer session_token"
```

### Method 3: JWT Claim

```go
// Extract from JWT token
token := extractJWT(r)
claims := parseJWT(token)
orgID := claims["org_id"]
```

## Permission Patterns

### Basic Permissions

```go
// Simple resource check
canView := client.RBAC.Can(ctx, &authsome.PermissionCheck{
    UserID:   "user_123",
    Action:   "view",
    Resource: "dashboards",
    OrgID:    "org_acme",
})
```

### Hierarchical Permissions

```go
// Check access to specific resource
canEdit := client.RBAC.Can(ctx, &authsome.PermissionCheck{
    UserID:   "user_123",
    Action:   "edit",
    Resource: "projects:proj_456",  // Specific project
    OrgID:    "org_acme",
})

// Check wildcard permission
canManageAll := client.RBAC.Can(ctx, &authsome.PermissionCheck{
    UserID:   "user_123",
    Action:   "manage",
    Resource: "projects:*",  // All projects
    OrgID:    "org_acme",
})
```

### Conditional Permissions

```bash
# Only allow if user is owner of resource
authsome-cli rbac add-policy \
  --expression "role:member can delete on projects:* where owner = true"
```

## Performance Tips

### 1. Cache User Permissions

```go
// On user login, fetch and cache all permissions
userPerms, err := client.RBAC.GetUserPermissions(ctx, userID, orgID)

// Store in cache (Redis, Memcached, or in-memory)
cache.Set(fmt.Sprintf("perms:%s:%s", userID, orgID), userPerms, 5*time.Minute)

// Check locally
if hasPermission(userPerms, "projects:read") {
    // Allow access
}
```

### 2. Use AuthSome's Built-in Cache

```go
// AuthSome caches role lookups automatically
// Cache hit: ~10ns
// Cache miss: ~240µs
// Default TTL: 5 minutes

// No additional code needed - it just works!
```

### 3. Batch Permission Checks

```go
// Check multiple permissions at once
checks := []authsome.PermissionCheck{
    {UserID: "user_123", Action: "read", Resource: "projects", OrgID: "org_acme"},
    {UserID: "user_123", Action: "write", Resource: "projects", OrgID: "org_acme"},
    {UserID: "user_123", Action: "delete", Resource: "projects", OrgID: "org_acme"},
}

results, err := client.RBAC.CheckBatch(ctx, checks)
// Returns: map[string]bool
```

## Common Patterns

### Pattern 1: API Gateway

```go
// Centralized permission checking at API gateway
func ApiGateway(w http.ResponseWriter, r *http.Request) {
    // Extract route info
    action, resource := parseRoute(r.URL.Path)
    
    // Check permission
    allowed, _ := client.RBAC.Can(r.Context(), &authsome.PermissionCheck{
        UserID:   getSessionUser(r),
        Action:   action,
        Resource: resource,
        OrgID:    getOrgID(r),
    })
    
    if !allowed {
        http.Error(w, "Forbidden", 403)
        return
    }
    
    // Forward to backend service
    proxyRequest(w, r)
}
```

### Pattern 2: Service Mesh

```yaml
# Envoy/Istio external authorization
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: authsome-authz
spec:
  action: CUSTOM
  provider:
    name: authsome-ext-authz
  rules:
  - to:
    - operation:
        paths: ["/api/*"]
```

### Pattern 3: Microservices

```go
// Each microservice validates permissions
type ProjectService struct {
    auth *authsome.Client
}

func (s *ProjectService) CreateProject(ctx context.Context, req *CreateProjectRequest) error {
    // Validate permission
    if !s.checkPermission(ctx, req.UserID, "create", "projects", req.OrgID) {
        return errors.New("permission denied")
    }
    
    // Business logic
    return s.db.CreateProject(ctx, req)
}

func (s *ProjectService) checkPermission(ctx context.Context, userID, action, resource, orgID string) bool {
    allowed, _ := s.auth.RBAC.Can(ctx, &authsome.PermissionCheck{
        UserID:   userID,
        Action:   action,
        Resource: resource,
        OrgID:    orgID,
    })
    return allowed
}
```

## Frontend Integration

### React Example

```typescript
import { useAuthSome } from '@authsome/react';

function ProjectList() {
    const { user, checkPermission } = useAuthSome();
    
    const canCreate = checkPermission('create', 'projects');
    const canDelete = checkPermission('delete', 'projects');
    
    return (
        <div>
            <h1>Projects</h1>
            {canCreate && <button onClick={handleCreate}>New Project</button>}
            
            <ul>
                {projects.map(project => (
                    <li key={project.id}>
                        {project.name}
                        {canDelete && (
                            <button onClick={() => handleDelete(project.id)}>
                                Delete
                            </button>
                        )}
                    </li>
                ))}
            </ul>
        </div>
    );
}
```

### Vue Example

```vue
<template>
    <div>
        <h1>Projects</h1>
        <button v-if="canCreate" @click="createProject">New Project</button>
        
        <ul>
            <li v-for="project in projects" :key="project.id">
                {{ project.name }}
                <button v-if="canDelete" @click="deleteProject(project.id)">
                    Delete
                </button>
            </li>
        </ul>
    </div>
</template>

<script>
import { useAuthSome } from '@authsome/vue';

export default {
    setup() {
        const { checkPermission } = useAuthSome();
        
        const canCreate = checkPermission('create', 'projects');
        const canDelete = checkPermission('delete', 'projects');
        
        return { canCreate, canDelete };
    }
}
</script>
```

## Troubleshooting

### Issue: "Forbidden" even with correct role

**Check:**
1. Organization ID is correct
2. Role is assigned in the right organization
3. Permission is spelled correctly
4. Cache is not stale

```bash
# Verify user roles
authsome-cli user get-roles --user-id=user_123 --org=org_acme

# Verify role permissions
authsome-cli role get --name=admin --org=org_acme

# Clear cache
authsome-cli cache clear --user=user_123 --org=org_acme
```

### Issue: Slow permission checks

**Solutions:**
1. Enable caching (enabled by default)
2. Use batch checks for multiple permissions
3. Cache user permissions in your app
4. Use Redis for distributed caching

### Issue: Organization not found

**Check:**
1. Organization exists: `authsome-cli org get --id=org_acme`
2. Header/subdomain is correct
3. User is member of organization

## Next Steps

- [Full Integration Guide](./SAAS_INTEGRATION_GUIDE.md)
- [API Reference](https://docs.authsome.dev/api)
- [Client Libraries](https://github.com/xraph/authsome/tree/main/clients)
- [Examples](https://github.com/xraph/authsome/tree/main/examples)

## Support

- **Documentation**: https://docs.authsome.dev
- **Discord**: https://discord.gg/authsome
- **Email**: support@authsome.dev
- **Enterprise**: enterprise@authsome.dev

