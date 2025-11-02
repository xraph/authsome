# SaaS Integration Guide: Using AuthSome for External Applications

## Overview

AuthSome provides **enterprise-grade authentication and RBAC** that can be easily integrated into external SaaS applications. This allows your SaaS to:

- ✅ **Delegate authentication** to AuthSome
- ✅ **Use organization-scoped permissions** for multi-tenant access control
- ✅ **Leverage fast permission checking** (10ns cache hits, 240µs cache misses)
- ✅ **Manage users, roles, and organizations** through AuthSome APIs
- ✅ **Customize per-organization settings** (OAuth, forms, RBAC policies)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Your SaaS Application                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────────┐   │
│  │  Frontend  │  │  Backend   │  │  API Services  │   │
│  └────────────┘  └────────────┘  └────────────────┘   │
│         │              │                  │             │
│         └──────────────┴──────────────────┘             │
│                        │                                │
│                 ┌──────▼──────┐                         │
│                 │  Auth Client │                        │
│                 └──────┬──────┘                         │
└────────────────────────┼────────────────────────────────┘
                         │
                         │ HTTP/REST API
                         │
┌────────────────────────▼────────────────────────────────┐
│                  AuthSome Instance                       │
│  ┌─────────────────────────────────────────────────┐   │
│  │            Multi-Tenant Organizations            │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐         │   │
│  │  │ Org A   │  │ Org B   │  │ Org C   │   ...   │   │
│  │  │ Users   │  │ Users   │  │ Users   │         │   │
│  │  │ Roles   │  │ Roles   │  │ Roles   │         │   │
│  │  │ Perms   │  │ Perms   │  │ Perms   │         │   │
│  │  └─────────┘  └─────────┘  └─────────┘         │   │
│  └─────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │  RBAC Engine │  │  Session Mgmt│  │  Audit Logs  │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
└──────────────────────────────────────────────────────────┘
```

## Integration Methods

### Method 1: Client Library (Recommended)

Use the official Go, TypeScript, or Rust client libraries:

```go
package main

import (
    "context"
    "fmt"
    
    authsome "github.com/xraph/authsome/clients/go"
)

func main() {
    // Initialize AuthSome client
    client := authsome.NewClient(authsome.ClientConfig{
        BaseURL: "https://auth.yourdomain.com",
        APIKey:  "your-api-key",
    })
    
    // Authenticate user
    session, err := client.Auth.SignIn(context.Background(), &authsome.SignInRequest{
        Email:    "user@example.com",
        Password: "password",
        OrgID:    "org_123", // Organization context
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("User authenticated: %s\n", session.User.ID)
    
    // Check permissions
    canManage := client.RBAC.Can(context.Background(), &authsome.PermissionCheck{
        UserID:   session.User.ID,
        Action:   "manage",
        Resource: "projects",
        OrgID:    "org_123",
    })
    
    if canManage {
        fmt.Println("User can manage projects")
    }
}
```

### Method 2: Direct API Integration

Make HTTP requests to AuthSome APIs:

```typescript
// TypeScript/JavaScript example
class AuthSomeClient {
    constructor(private baseURL: string, private apiKey: string) {}
    
    async signIn(email: string, password: string, orgId: string) {
        const response = await fetch(`${this.baseURL}/auth/signin`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': this.apiKey,
                'X-Organization-ID': orgId,
            },
            body: JSON.stringify({ email, password }),
        });
        
        return response.json();
    }
    
    async checkPermission(userId: string, action: string, resource: string, orgId: string) {
        const response = await fetch(`${this.baseURL}/rbac/check`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': this.apiKey,
                'X-Organization-ID': orgId,
            },
            body: JSON.stringify({
                user_id: userId,
                action,
                resource,
            }),
        });
        
        const data = await response.json();
        return data.allowed;
    }
}

// Usage
const auth = new AuthSomeClient('https://auth.yourdomain.com', 'your-api-key');

// Sign in user
const session = await auth.signIn('user@example.com', 'password', 'org_123');

// Check permission
const canDelete = await auth.checkPermission(
    session.user.id,
    'delete',
    'projects',
    'org_123'
);
```

## Multi-Tenant Setup

### 1. Configure AuthSome in SaaS Mode

```yaml
# authsome-config.yaml
mode: saas

server:
  host: auth.yourdomain.com
  port: 8080

database:
  driver: postgres
  host: localhost
  port: 5432
  database: authsome
  username: authsome
  password: secret

auth:
  session:
    lifetime: 24h
    cookie:
      name: authsome_session
      domain: .yourdomain.com  # Share across subdomains
      secure: true
      httponly: true
      samesite: lax

  rbac:
    enabled: true
    cache_ttl: 5m  # Fast permission checking with caching

# Per-organization config overrides
organizations:
  enabled: true
  default_roles:
    - name: owner
      permissions: ["*"]
    - name: admin
      permissions: ["org:*", "members:*", "projects:*", "settings:*"]
    - name: member
      permissions: ["org:read", "projects:read"]
    - name: viewer
      permissions: ["org:read", "projects:read"]
```

### 2. Create Organizations for Your SaaS Customers

```bash
# Using AuthSome CLI
authsome-cli org create \
  --name "Acme Corp" \
  --slug "acme" \
  --owner-email "admin@acme.com"

# Or via API
curl -X POST https://auth.yourdomain.com/api/organizations \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "name": "Acme Corp",
    "slug": "acme",
    "metadata": {
      "plan": "enterprise",
      "max_users": 100
    }
  }'
```

### 3. Define Custom Roles and Permissions

```bash
# Create custom roles for your SaaS
authsome-cli role create \
  --org "acme" \
  --name "project-manager" \
  --permissions "projects:*,tasks:*,comments:*"

authsome-cli role create \
  --org "acme" \
  --name "developer" \
  --permissions "projects:read,tasks:*,code:*"

# Assign roles to users
authsome-cli user assign-role \
  --user-id "user_123" \
  --role "project-manager" \
  --org "acme"
```

## RBAC Integration Patterns

### Pattern 1: Middleware-Based Authorization

```go
// Middleware to check permissions before handler execution
func RequirePermission(action, resource string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get session from request
            session := getSessionFromRequest(r)
            if session == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            // Get organization from context
            orgID := r.Header.Get("X-Organization-ID")
            
            // Check permission using AuthSome client
            allowed, err := authClient.RBAC.Can(r.Context(), &authsome.PermissionCheck{
                UserID:   session.UserID,
                Action:   action,
                Resource: resource,
                OrgID:    orgID,
            })
            
            if err != nil || !allowed {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Usage in your SaaS
http.Handle("/api/projects", RequirePermission("read", "projects")(projectsHandler))
http.Handle("/api/projects/create", RequirePermission("create", "projects")(createProjectHandler))
http.Handle("/api/projects/delete", RequirePermission("delete", "projects")(deleteProjectHandler))
```

### Pattern 2: Service-Level Authorization

```go
type ProjectService struct {
    authClient *authsome.Client
    db         *sql.DB
}

func (s *ProjectService) CreateProject(ctx context.Context, userID, orgID, name string) (*Project, error) {
    // Check permission before action
    allowed, err := s.authClient.RBAC.Can(ctx, &authsome.PermissionCheck{
        UserID:   userID,
        Action:   "create",
        Resource: "projects",
        OrgID:    orgID,
    })
    
    if err != nil {
        return nil, fmt.Errorf("permission check failed: %w", err)
    }
    
    if !allowed {
        return nil, fmt.Errorf("permission denied: user %s cannot create projects", userID)
    }
    
    // Perform the action
    project := &Project{
        ID:      generateID(),
        Name:    name,
        OrgID:   orgID,
        Created: time.Now(),
    }
    
    if err := s.db.CreateProject(ctx, project); err != nil {
        return nil, err
    }
    
    return project, nil
}

func (s *ProjectService) GetProject(ctx context.Context, userID, orgID, projectID string) (*Project, error) {
    // Check read permission
    allowed, err := s.authClient.RBAC.Can(ctx, &authsome.PermissionCheck{
        UserID:   userID,
        Action:   "read",
        Resource: fmt.Sprintf("projects:%s", projectID),
        OrgID:    orgID,
    })
    
    if err != nil || !allowed {
        return nil, fmt.Errorf("permission denied")
    }
    
    return s.db.GetProject(ctx, projectID)
}
```

### Pattern 3: Frontend Permission Checks

```typescript
// React component with permission checks
import { useAuthSome } from '@authsome/react';

function ProjectActions({ project }) {
    const { checkPermission } = useAuthSome();
    
    const canEdit = checkPermission('edit', `projects:${project.id}`);
    const canDelete = checkPermission('delete', `projects:${project.id}`);
    const canShare = checkPermission('share', `projects:${project.id}`);
    
    return (
        <div className="actions">
            {canEdit && <button onClick={handleEdit}>Edit</button>}
            {canDelete && <button onClick={handleDelete}>Delete</button>}
            {canShare && <button onClick={handleShare}>Share</button>}
        </div>
    );
}
```

## Organization-Scoped Configurations

AuthSome allows per-organization customization:

```yaml
# Default configuration
auth:
  oauth:
    google:
      client_id: "default-google-client-id"
      client_secret: "default-google-secret"

# Organization-specific override
organizations:
  org_acme:
    auth:
      oauth:
        google:
          client_id: "acme-google-client-id"
          client_secret: "acme-google-secret"
          
  org_techcorp:
    auth:
      session:
        lifetime: 48h  # Extended session for this org
      oauth:
        azure_ad:
          tenant_id: "techcorp-tenant"
          client_id: "techcorp-client"
```

## Advanced RBAC Patterns

### Hierarchical Permissions

```bash
# Define hierarchical permission structure
authsome-cli rbac add-policy \
  --expression "role:admin can * on projects:*"

authsome-cli rbac add-policy \
  --expression "role:project-manager can read,write on projects:*"

authsome-cli rbac add-policy \
  --expression "role:developer can read on projects:*"

# Resource-specific permissions
authsome-cli rbac add-policy \
  --expression "user:user_123 can delete on projects:proj_456"
```

### Conditional Permissions

```bash
# Permissions with conditions
authsome-cli rbac add-policy \
  --expression "role:member can edit on projects:* where owner = true"

authsome-cli rbac add-policy \
  --expression "role:member can delete on comments:* where author = true"
```

### Time-Based Permissions

```go
// Grant temporary elevated permissions
err := authClient.RBAC.GrantTemporaryPermission(ctx, &authsome.TemporaryPermission{
    UserID:    "user_123",
    Action:    "admin",
    Resource:  "billing",
    OrgID:     "org_acme",
    ExpiresAt: time.Now().Add(1 * time.Hour),
})
```

## Session Management

### Single Sign-On (SSO)

```go
// User signs in once, works across all your SaaS services
session, err := authClient.Auth.SignIn(ctx, &authsome.SignInRequest{
    Email:    "user@acme.com",
    Password: "password",
    OrgID:    "org_acme",
})

// Session token works across subdomains
// - app.yourdomain.com
// - api.yourdomain.com
// - dashboard.yourdomain.com
```

### Session Validation

```go
// Validate session on each request
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get session token from cookie
        cookie, err := r.Cookie("authsome_session")
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Validate session
        session, err := authClient.Session.Validate(r.Context(), cookie.Value)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Add session to context
        ctx := context.WithValue(r.Context(), "session", session)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Performance Optimization

### 1. Enable Permission Caching

```go
// AuthSome automatically caches role lookups (5 min default)
// Cache hit: ~10ns
// Cache miss: ~240µs

// For distributed systems, use Redis cache
client := authsome.NewClient(authsome.ClientConfig{
    BaseURL: "https://auth.yourdomain.com",
    Cache: &authsome.RedisCacheConfig{
        Host: "redis:6379",
        TTL:  5 * time.Minute,
    },
})
```

### 2. Batch Permission Checks

```go
// Check multiple permissions in one call
permissions := []authsome.PermissionCheck{
    {UserID: "user_123", Action: "read", Resource: "projects"},
    {UserID: "user_123", Action: "write", Resource: "projects"},
    {UserID: "user_123", Action: "delete", Resource: "projects"},
}

results, err := authClient.RBAC.CheckBatch(ctx, permissions)
// results: map[string]bool
```

### 3. Prefetch User Permissions

```go
// Load all user permissions on login
userPerms, err := authClient.RBAC.GetUserPermissions(ctx, "user_123", "org_acme")

// Cache in your application
cache.Set("permissions:user_123:org_acme", userPerms, 5*time.Minute)

// Check permissions locally
if contains(userPerms, "projects:read") {
    // Allow access
}
```

## Audit Logging

AuthSome automatically logs all authentication and permission events:

```go
// Query audit logs
logs, err := authClient.Audit.Query(ctx, &authsome.AuditQuery{
    OrgID:     "org_acme",
    UserID:    "user_123",
    Actions:   []string{"signin", "permission_check", "role_assigned"},
    StartDate: time.Now().Add(-24 * time.Hour),
    EndDate:   time.Now(),
})

// Export audit logs for compliance
csv, err := authClient.Audit.Export(ctx, &authsome.ExportRequest{
    OrgID:  "org_acme",
    Format: "csv",
    Period: "last_30_days",
})
```

## Webhooks Integration

Subscribe to AuthSome events in your SaaS:

```yaml
# Configure webhooks
webhooks:
  - name: "User Registration"
    url: "https://api.yoursaas.com/webhooks/auth/user-registered"
    events: ["user.created"]
    
  - name: "Role Changes"
    url: "https://api.yoursaas.com/webhooks/auth/role-changed"
    events: ["role.assigned", "role.removed"]
    
  - name: "Organization Events"
    url: "https://api.yoursaas.com/webhooks/auth/org-events"
    events: ["org.created", "org.updated", "member.added", "member.removed"]
```

```go
// Handle webhooks in your SaaS
func HandleAuthWebhook(w http.ResponseWriter, r *http.Request) {
    var event authsome.WebhookEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid payload", http.StatusBadRequest)
        return
    }
    
    // Verify webhook signature
    if !authClient.Webhook.Verify(r, event) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    
    switch event.Type {
    case "user.created":
        // Create user profile in your SaaS
        createUserProfile(event.Data)
        
    case "role.assigned":
        // Update user permissions cache
        invalidateUserPermissionsCache(event.Data.UserID)
        
    case "org.created":
        // Initialize organization resources
        initializeOrgResources(event.Data.OrgID)
    }
    
    w.WriteHeader(http.StatusOK)
}
```

## Example: Complete SaaS Integration

### Project Management SaaS Example

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
    authsome "github.com/xraph/authsome/clients/go"
)

type ProjectManagementApp struct {
    auth *authsome.Client
}

func main() {
    app := &ProjectManagementApp{
        auth: authsome.NewClient(authsome.ClientConfig{
            BaseURL: "https://auth.projectapp.com",
            APIKey:  "your-api-key",
        }),
    }
    
    // Setup routes with permission checks
    http.HandleFunc("/api/projects", app.RequireAuth(app.RequirePermission("read", "projects")(app.ListProjects)))
    http.HandleFunc("/api/projects/create", app.RequireAuth(app.RequirePermission("create", "projects")(app.CreateProject)))
    http.HandleFunc("/api/projects/delete", app.RequireAuth(app.RequirePermission("delete", "projects")(app.DeleteProject)))
    
    http.ListenAndServe(":8080", nil)
}

func (app *ProjectManagementApp) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        session, err := app.auth.Session.FromRequest(r)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        ctx := context.WithValue(r.Context(), "session", session)
        next(w, r.WithContext(ctx))
    }
}

func (app *ProjectManagementApp) RequirePermission(action, resource string) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            session := r.Context().Value("session").(*authsome.Session)
            orgID := r.Header.Get("X-Organization-ID")
            
            allowed, err := app.auth.RBAC.Can(r.Context(), &authsome.PermissionCheck{
                UserID:   session.UserID,
                Action:   action,
                Resource: resource,
                OrgID:    orgID,
            })
            
            if err != nil || !allowed {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            
            next(w, r)
        }
    }
}

func (app *ProjectManagementApp) ListProjects(w http.ResponseWriter, r *http.Request) {
    // Implementation
}

func (app *ProjectManagementApp) CreateProject(w http.ResponseWriter, r *http.Request) {
    // Implementation
}

func (app *ProjectManagementApp) DeleteProject(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

## Best Practices

### 1. Organization Context
Always include organization ID in requests:
- Header: `X-Organization-ID`
- Subdomain: `acme.yourapp.com`
- JWT claim: `org_id`

### 2. Permission Granularity
Use hierarchical resources:
```
projects:*                    # All projects
projects:proj_123            # Specific project
projects:proj_123:tasks:*    # All tasks in project
```

### 3. Role Design
Create roles that match your SaaS's business logic:
- `owner` - Full access
- `admin` - Administrative operations
- `manager` - Team management
- `member` - Standard access
- `viewer` - Read-only access

### 4. Caching Strategy
- Cache user permissions on login (5 min)
- Invalidate on role changes
- Use Redis for distributed systems

### 5. Error Handling
Return appropriate HTTP status codes:
- `401 Unauthorized` - No session/invalid token
- `403 Forbidden` - Authenticated but no permission
- `404 Not Found` - Resource doesn't exist or no access

## Migration from Existing Auth

### Step 1: Run AuthSome Alongside Existing Auth
```go
// Dual authentication during migration
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Try AuthSome first
        session, err := authsomeClient.Session.FromRequest(r)
        if err == nil {
            ctx := context.WithValue(r.Context(), "session", session)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }
        
        // Fallback to legacy auth
        legacySession := validateLegacySession(r)
        if legacySession != nil {
            ctx := context.WithValue(r.Context(), "session", legacySession)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }
        
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
    })
}
```

### Step 2: Migrate Users Gradually
```go
// On user login, migrate to AuthSome
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    // Validate with legacy system
    legacyUser := validateLegacyCredentials(email, password)
    
    // Create in AuthSome
    authsomeUser, err := authsomeClient.User.Create(ctx, &authsome.CreateUserRequest{
        Email:    legacyUser.Email,
        Password: password,  // Will be hashed by AuthSome
        OrgID:    legacyUser.OrgID,
    })
    
    // Migrate roles
    for _, role := range legacyUser.Roles {
        authsomeClient.RBAC.AssignRole(ctx, authsomeUser.ID, role, legacyUser.OrgID)
    }
    
    // Create session
    session, _ := authsomeClient.Session.Create(ctx, authsomeUser.ID)
    
    // Set cookie
    http.SetCookie(w, &http.Cookie{
        Name:  "authsome_session",
        Value: session.Token,
    })
}
```

### Step 3: Remove Legacy Auth
After all users migrated, remove legacy auth system.

## Support & Resources

- **Documentation**: https://docs.authsome.dev
- **API Reference**: https://docs.authsome.dev/api
- **Client Libraries**: https://github.com/xraph/authsome/tree/main/clients
- **Examples**: https://github.com/xraph/authsome/tree/main/examples
- **Community**: https://discord.gg/authsome
- **Enterprise Support**: enterprise@authsome.dev

## Conclusion

AuthSome provides a **production-ready, performant, and feature-rich** authentication and RBAC system that can be easily integrated into any SaaS application. With:

- ✅ Multi-tenant organization support out of the box
- ✅ Fast permission checking (< 100µs with caching)
- ✅ Flexible RBAC with custom roles and policies
- ✅ Organization-scoped configurations
- ✅ Comprehensive audit logging
- ✅ Webhook integrations
- ✅ Client libraries for Go, TypeScript, and Rust

Your SaaS can focus on core business logic while AuthSome handles all authentication, authorization, and user management complexity.

