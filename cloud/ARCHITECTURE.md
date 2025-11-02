# AuthSome Cloud Architecture

**Deep dive into the control plane system design**

## Table of Contents

- [System Overview](#system-overview)
- [Core Components](#core-components)
- [Data Models](#data-models)
- [Application Lifecycle](#application-lifecycle)
- [Request Flow](#request-flow)
- [Database Architecture](#database-architecture)
- [Security Model](#security-model)
- [Scaling Strategy](#scaling-strategy)
- [Failure Modes](#failure-modes)

## System Overview

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                     Customer Applications                         │
│  (SDK, cURL, Browser → api.authsome.cloud/v1/app_xxx/*)         │
└────────────────────────┬─────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────────┐
│                      API Gateway (Cloudflare)                     │
│  • DDoS Protection  • Rate Limiting  • TLS Termination           │
└────────────────────────┬─────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Load Balancer (K8s Ingress)                  │
└────────────────────────┬─────────────────────────────────────────┘
                         │
         ┌───────────────┴─────────────────┐
         ▼                                 ▼
┌─────────────────────┐          ┌─────────────────────┐
│   Control Plane API │          │    Proxy Service    │
│                     │          │                     │
│ • Workspace CRUD    │          │ • API Key Auth      │
│ • App provisioning  │          │ • Request routing   │
│ • Team management   │          │ • Response caching  │
│ • Billing endpoints │          │                     │
└──────────┬──────────┘          └──────────┬──────────┘
           │                                │
           │                                │ Routes to
           ▼                                │
┌─────────────────────┐                    │
│  Control Plane DB   │                    │
│   (PostgreSQL)      │                    │
│                     │                    │
│ • Workspaces        │                    │
│ • Applications      │                    │
│ • Usage metrics     │                    │
└─────────────────────┘                    │
                                           │
           ┌───────────────────────────────┘
           ▼
┌──────────────────────────────────────────────────────────────────┐
│            Customer AuthSome Instances (K8s Namespaces)           │
│                                                                   │
│  ┌───────────────────┐  ┌───────────────────┐                   │
│  │  app-abc123       │  │  app-def456       │                   │
│  │                   │  │                   │                   │
│  │  ┌─────────────┐  │  │  ┌─────────────┐  │                   │
│  │  │ AuthSome    │  │  │  │ AuthSome    │  │                   │
│  │  │ Deployment  │  │  │  │ Deployment  │  │                   │
│  │  └──────┬──────┘  │  │  └──────┬──────┘  │                   │
│  │         │         │  │         │         │                   │
│  │         ▼         │  │         ▼         │                   │
│  │  ┌─────────────┐  │  │  ┌─────────────┐  │                   │
│  │  │ PostgreSQL  │  │  │  │ PostgreSQL  │  │                   │
│  │  │  (Isolated) │  │  │  │  (Isolated) │  │                   │
│  │  └─────────────┘  │  │  └─────────────┘  │                   │
│  │  ┌─────────────┐  │  │  ┌─────────────┐  │                   │
│  │  │    Redis    │  │  │  │    Redis    │  │                   │
│  │  │  (Isolated) │  │  │  │  (Isolated) │  │                   │
│  │  └─────────────┘  │  │  └─────────────┘  │                   │
│  └───────────────────┘  └───────────────────┘                   │
└──────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Control Plane API

**Responsibility**: Workspace and application management

```go
// cmd/control-plane/main.go
package main

import (
    "github.com/xraph/forge"
    "github.com/xraph/authsome-cloud/control"
)

func main() {
    app := forge.New()
    
    // Middleware
    app.Use(authMiddleware())      // JWT auth for dashboard users
    app.Use(auditMiddleware())     // Log all control plane actions
    app.Use(rateLimitMiddleware()) // Rate limiting
    
    // Control services
    workspaceSvc := control.NewWorkspaceService(db, k8sClient)
    appSvc := control.NewApplicationService(db, k8sClient, provisioner)
    teamSvc := control.NewTeamService(db, emailSvc)
    billingSvc := control.NewBillingService(db, stripe)
    
    // API routes
    v1 := app.Group("/api/v1")
    
    // Workspace management
    v1.POST("/workspaces", workspaceSvc.Create)
    v1.GET("/workspaces", workspaceSvc.List)
    v1.GET("/workspaces/:id", workspaceSvc.Get)
    v1.PATCH("/workspaces/:id", workspaceSvc.Update)
    v1.DELETE("/workspaces/:id", workspaceSvc.Delete)
    
    // Application management
    v1.POST("/workspaces/:workspaceId/applications", appSvc.Create)
    v1.GET("/workspaces/:workspaceId/applications", appSvc.List)
    v1.GET("/applications/:id", appSvc.Get)
    v1.PATCH("/applications/:id", appSvc.Update)
    v1.DELETE("/applications/:id", appSvc.Delete)
    v1.POST("/applications/:id/restart", appSvc.Restart)
    
    // Team management
    v1.POST("/workspaces/:id/members", teamSvc.Invite)
    v1.GET("/workspaces/:id/members", teamSvc.List)
    v1.DELETE("/workspaces/:id/members/:userId", teamSvc.Remove)
    
    // Billing
    v1.GET("/workspaces/:id/usage", billingSvc.GetUsage)
    v1.GET("/workspaces/:id/invoices", billingSvc.ListInvoices)
    v1.POST("/workspaces/:id/payment-method", billingSvc.UpdatePaymentMethod)
    
    app.Run(":8080")
}
```

### 2. Proxy Service

**Responsibility**: Route customer API requests to correct AuthSome instance

```go
// cmd/proxy/main.go
package main

import (
    "github.com/xraph/forge"
    "github.com/xraph/authsome-cloud/internal/proxy"
)

func main() {
    app := forge.New()
    
    proxyService := proxy.New(
        appRepo,        // Application repository
        cache,          // Redis cache for routing table
        metrics,        // Prometheus metrics
    )
    
    // All customer API requests
    app.Any("/v1/:appId/*", proxyService.Forward)
    
    app.Run(":8081")
}

// internal/proxy/proxy.go
type Service struct {
    appRepo  repository.ApplicationRepository
    cache    *redis.Client
    metrics  *prometheus.Registry
    transport http.RoundTripper
}

func (s *Service) Forward(c *forge.Context) error {
    appID := c.Param("appId")
    
    // 1. Verify API key
    apiKey := extractAPIKey(c.Request())
    app, err := s.verifyAPIKey(apiKey, appID)
    if err != nil {
        return c.JSON(401, ErrorResponse{Message: "Invalid API key"})
    }
    
    // 2. Check application status
    if app.Status != "active" {
        return c.JSON(503, ErrorResponse{Message: "Application unavailable"})
    }
    
    // 3. Get target URL (cached)
    targetURL, err := s.getApplicationURL(appID)
    if err != nil {
        return c.JSON(500, ErrorResponse{Message: "Service unavailable"})
    }
    
    // 4. Track usage (async)
    go s.trackUsage(app.ID, c.Request())
    
    // 5. Forward request
    return s.forward(c, targetURL)
}

func (s *Service) getApplicationURL(appID string) (string, error) {
    // Try cache first
    cached, err := s.cache.Get(ctx, "app:url:"+appID).Result()
    if err == nil {
        return cached, nil
    }
    
    // Fetch from database
    app, err := s.appRepo.GetByID(ctx, appID)
    if err != nil {
        return "", err
    }
    
    // Kubernetes internal service URL
    url := fmt.Sprintf("http://authsome.authsome-%s.svc.cluster.local", appID)
    
    // Cache for 5 minutes
    s.cache.Set(ctx, "app:url:"+appID, url, 5*time.Minute)
    
    return url, nil
}
```

### 3. Provisioner Service

**Responsibility**: Asynchronous application provisioning

```go
// cmd/provisioner/main.go
package main

import (
    "github.com/xraph/authsome-cloud/control/application"
    "github.com/nats-io/nats.go"
)

func main() {
    // Connect to NATS for job queue
    nc, _ := nats.Connect("nats://nats:4222")
    
    provisioner := application.NewProvisioner(
        k8sClient,
        dbManager,
        cacheManager,
        monitoringManager,
    )
    
    // Subscribe to provisioning events
    nc.QueueSubscribe("app.provision", "provisioners", func(m *nats.Msg) {
        var req ProvisionRequest
        json.Unmarshal(m.Data, &req)
        
        err := provisioner.Provision(context.Background(), &req)
        if err != nil {
            log.Error("Provisioning failed", "appId", req.AppID, "error", err)
            // Update app status to "failed"
            return
        }
        
        log.Info("Provisioning complete", "appId", req.AppID)
    })
    
    select {} // Block forever
}
```

### 4. Management Dashboard

**Responsibility**: Web UI for workspace/application management

```typescript
// cmd/dashboard/app/page.tsx
import { WorkspaceList } from '@/components/workspace-list'
import { CreateWorkspaceDialog } from '@/components/create-workspace-dialog'

export default async function HomePage() {
  // Server Component - fetch on server
  const workspaces = await fetch('http://control-plane:8080/api/v1/workspaces', {
    headers: {
      'Authorization': `Bearer ${cookies().get('auth_token')?.value}`
    }
  }).then(r => r.json())
  
  return (
    <div>
      <h1>Your Workspaces</h1>
      <CreateWorkspaceDialog />
      <WorkspaceList workspaces={workspaces} />
    </div>
  )
}

// cmd/dashboard/app/workspace/[id]/applications/page.tsx
export default async function ApplicationsPage({ params }: { params: { id: string } }) {
  const applications = await fetch(
    `http://control-plane:8080/api/v1/workspaces/${params.id}/applications`
  ).then(r => r.json())
  
  return (
    <div>
      <ApplicationList applications={applications} />
      <CreateApplicationButton workspaceId={params.id} />
    </div>
  )
}
```

## Data Models

### Control Plane Database Schema

```go
// schema/workspace.go
type Workspace struct {
    ID          string                 `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
    Name        string                 `bun:"name,notnull"`
    Slug        string                 `bun:"slug,unique,notnull"`
    OwnerID     string                 `bun:"owner_id,notnull"`
    Plan        string                 `bun:"plan,notnull"` // free, pro, enterprise
    Status      string                 `bun:"status,notnull"` // active, suspended, deleted
    Metadata    map[string]interface{} `bun:"metadata,type:jsonb"`
    CreatedAt   time.Time              `bun:"created_at,notnull,default:now()"`
    UpdatedAt   time.Time              `bun:"updated_at,notnull,default:now()"`
    DeletedAt   *time.Time             `bun:"deleted_at,soft_delete"`
}

// schema/application.go
type Application struct {
    ID              string                 `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
    WorkspaceID     string                 `bun:"workspace_id,notnull"`
    Name            string                 `bun:"name,notnull"`
    Slug            string                 `bun:"slug,notnull"` // unique within workspace
    Environment     string                 `bun:"environment,notnull"` // production, staging, development
    Status          string                 `bun:"status,notnull"` // provisioning, active, suspended, deleted
    
    // Deployment details
    Region          string                 `bun:"region,notnull"` // us-east-1, eu-west-1
    K8sNamespace    string                 `bun:"k8s_namespace,notnull"`
    DatabaseURL     string                 `bun:"database_url,notnull"` // Encrypted
    RedisURL        string                 `bun:"redis_url,notnull"` // Encrypted
    InternalURL     string                 `bun:"internal_url,notnull"` // K8s service URL
    
    // API keys
    PublicKey       string                 `bun:"public_key,unique,notnull"` // pk_live_xxx
    SecretKeyHash   string                 `bun:"secret_key_hash,notnull"` // Hashed sk_live_xxx
    
    // Configuration
    Config          map[string]interface{} `bun:"config,type:jsonb"` // AuthSome config overrides
    
    // Resources
    CPULimit        string                 `bun:"cpu_limit"` // 1000m
    MemoryLimit     string                 `bun:"memory_limit"` // 2Gi
    StorageLimit    string                 `bun:"storage_limit"` // 10Gi
    
    // Metadata
    Metadata        map[string]interface{} `bun:"metadata,type:jsonb"`
    CreatedAt       time.Time              `bun:"created_at,notnull,default:now()"`
    UpdatedAt       time.Time              `bun:"updated_at,notnull,default:now()"`
    DeletedAt       *time.Time             `bun:"deleted_at,soft_delete"`
    
    // Relations
    Workspace       *Workspace             `bun:"rel:belongs-to,join:workspace_id=id"`
}

// schema/team_member.go
type TeamMember struct {
    ID          string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
    WorkspaceID string     `bun:"workspace_id,notnull"`
    UserID      string     `bun:"user_id,notnull"`
    Email       string     `bun:"email,notnull"`
    Role        string     `bun:"role,notnull"` // owner, admin, developer, billing
    Status      string     `bun:"status,notnull"` // active, invited, suspended
    InvitedBy   string     `bun:"invited_by"`
    InvitedAt   time.Time  `bun:"invited_at"`
    JoinedAt    *time.Time `bun:"joined_at"`
    CreatedAt   time.Time  `bun:"created_at,notnull,default:now()"`
    UpdatedAt   time.Time  `bun:"updated_at,notnull,default:now()"`
}

// schema/usage.go
type UsageRecord struct {
    ID              string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
    ApplicationID   string    `bun:"application_id,notnull"`
    WorkspaceID     string    `bun:"workspace_id,notnull"`
    Period          string    `bun:"period,notnull"` // 2025-11
    
    // Metrics
    MAU             int       `bun:"mau"`                   // Monthly active users
    APIRequests     int64     `bun:"api_requests"`          // Total API requests
    StorageBytes    int64     `bun:"storage_bytes"`         // Database storage
    BandwidthBytes  int64     `bun:"bandwidth_bytes"`       // Network egress
    
    // Calculated at period end
    Cost            float64   `bun:"cost"`                  // Calculated cost
    Billable        bool      `bun:"billable"`              // Whether to bill
    
    CreatedAt       time.Time `bun:"created_at,notnull,default:now()"`
    UpdatedAt       time.Time `bun:"updated_at,notnull,default:now()"`
}

// schema/api_key.go
type APIKey struct {
    ID              string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
    ApplicationID   string     `bun:"application_id,notnull"`
    Type            string     `bun:"type,notnull"` // public, secret
    Key             string     `bun:"key,unique,notnull"`
    KeyHash         string     `bun:"key_hash,notnull"` // For secret keys
    Name            string     `bun:"name"`
    Scopes          []string   `bun:"scopes,array"`
    LastUsedAt      *time.Time `bun:"last_used_at"`
    ExpiresAt       *time.Time `bun:"expires_at"`
    RevokedAt       *time.Time `bun:"revoked_at"`
    CreatedAt       time.Time  `bun:"created_at,notnull,default:now()"`
}
```

## Application Lifecycle

### Provisioning Flow

```
1. User creates application via dashboard/API
   └─→ Control Plane API receives request

2. Control Plane validates and creates record
   ├─→ Generate application ID: app_abc123
   ├─→ Generate API keys: pk_live_abc123, sk_live_abc123
   ├─→ Set status: "provisioning"
   └─→ Publish to NATS: "app.provision"

3. Provisioner worker picks up job
   ├─→ Create Kubernetes namespace: authsome-app-abc123
   ├─→ Provision PostgreSQL database
   │   ├─→ Create database: authsome_app_abc123
   │   ├─→ Generate credentials
   │   └─→ Enable encryption at rest
   ├─→ Provision Redis instance
   │   ├─→ Create Redis instance
   │   └─→ Generate credentials
   ├─→ Deploy AuthSome instance
   │   ├─→ Apply Deployment (2 replicas)
   │   ├─→ Apply Service (ClusterIP)
   │   ├─→ Apply ConfigMap (AuthSome config)
   │   ├─→ Apply Secret (DB credentials)
   │   └─→ Wait for rollout complete
   ├─→ Run database migrations
   │   └─→ Initialize AuthSome schema
   ├─→ Setup monitoring
   │   ├─→ Create ServiceMonitor (Prometheus)
   │   ├─→ Create Grafana dashboard
   │   └─→ Configure alerts
   └─→ Update status: "active"

4. Application ready
   └─→ User can make API requests
```

### Scaling Flow

```go
// Auto-scaling based on metrics
type ScalingDecision struct {
    AppID           string
    CurrentReplicas int
    DesiredReplicas int
    Reason          string
}

func (s *Scaler) Evaluate(app *Application, metrics *Metrics) *ScalingDecision {
    // Scale up if:
    // - CPU > 70% for 5 minutes
    // - Memory > 80% for 5 minutes
    // - Request latency > 500ms p95
    
    // Scale down if:
    // - CPU < 30% for 15 minutes
    // - Memory < 40% for 15 minutes
    // - Min replicas: 2 (for HA)
    
    if metrics.CPUPercent > 70 && app.Replicas < app.MaxReplicas {
        return &ScalingDecision{
            AppID:           app.ID,
            CurrentReplicas: app.Replicas,
            DesiredReplicas: app.Replicas + 1,
            Reason:          "High CPU usage",
        }
    }
    
    return nil
}
```

### Deletion Flow

```
1. User deletes application
   └─→ Control Plane API receives request

2. Soft delete and grace period
   ├─→ Set status: "deleting"
   ├─→ Set deleted_at: now() + 7 days grace period
   ├─→ Disable API keys (requests fail)
   └─→ Send email notification

3. Grace period allows recovery
   └─→ User can "undelete" within 7 days

4. After grace period, provisioner hard deletes
   ├─→ Delete Kubernetes namespace (cascade deletes)
   ├─→ Delete PostgreSQL database
   ├─→ Delete Redis instance
   ├─→ Archive usage records
   └─→ Remove from routing cache
```

## Request Flow

### Customer API Request Path

```
1. Client makes request:
   POST https://api.authsome.cloud/v1/app_abc123/users
   Authorization: Bearer sk_live_abc123_xyz...
   
2. Cloudflare (Edge)
   ├─→ DDoS protection
   ├─→ Rate limiting (per API key)
   ├─→ TLS termination
   └─→ Forward to load balancer

3. Kubernetes Ingress (Load Balancer)
   └─→ Route to Proxy Service

4. Proxy Service
   ├─→ Extract app ID from path: app_abc123
   ├─→ Extract API key from header
   ├─→ Verify API key belongs to app
   │   ├─→ Check cache (Redis)
   │   └─→ Fallback to database
   ├─→ Check application status: active
   ├─→ Get internal URL from cache
   │   └─→ http://authsome.authsome-app-abc123.svc.cluster.local
   ├─→ Track usage (async to NATS)
   └─→ Forward request to AuthSome instance

5. AuthSome Instance (Customer's)
   ├─→ Process authentication request
   ├─→ Query customer's PostgreSQL
   ├─→ Return response
   └─→ Cache session in customer's Redis

6. Proxy Service
   ├─→ Receive response
   ├─→ Add headers (X-AuthSome-App-ID, etc.)
   └─→ Return to client

7. Client receives response
```

### Dashboard Request Path

```
1. User accesses dashboard:
   GET https://dashboard.authsome.cloud/workspace/ws_xyz

2. Next.js Server Component
   ├─→ Verify dashboard session (cookie)
   ├─→ Fetch from Control Plane API
   │   GET http://control-plane:8080/api/v1/workspaces/ws_xyz
   │   Authorization: Bearer <dashboard-jwt>
   └─→ Render server-side

3. Control Plane API
   ├─→ Verify JWT token
   ├─→ Check user has access to workspace
   ├─→ Query control plane database
   └─→ Return workspace data

4. Dashboard renders HTML
   └─→ Streamed to user's browser
```

## Database Architecture

### Control Plane Database

```sql
-- Single PostgreSQL cluster for control plane
-- High availability with streaming replication

CREATE DATABASE authsome_control;

-- Tables (see schema above):
-- workspaces, applications, team_members, usage_records, api_keys

-- Indexes for performance
CREATE INDEX idx_applications_workspace_id ON applications(workspace_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_applications_public_key ON applications(public_key);
CREATE UNIQUE INDEX idx_applications_workspace_slug ON applications(workspace_id, slug);

CREATE INDEX idx_usage_records_application_period ON usage_records(application_id, period);
CREATE INDEX idx_usage_records_workspace_period ON usage_records(workspace_id, period);
```

### Customer Databases

```sql
-- Each application gets isolated database
-- Naming: authsome_app_{app_id}

CREATE DATABASE authsome_app_abc123;

-- Standard AuthSome schema (from core framework):
-- users, sessions, organizations, members, audit_logs, etc.

-- Connection pooling per application:
Max connections: 100 (configurable per plan)
Connection timeout: 30s
Idle timeout: 10m
```

### Database Isolation Strategies

```go
// Option A: Separate database per app (RECOMMENDED)
// Pros: Complete isolation, easy to backup/restore, independent scaling
// Cons: More databases to manage
type DatabaseManager struct {
    host string
}

func (m *DatabaseManager) CreateDatabase(appID string) (string, error) {
    dbName := fmt.Sprintf("authsome_app_%s", appID)
    
    _, err := m.adminConn.Exec(ctx, fmt.Sprintf(`
        CREATE DATABASE %s 
        WITH ENCODING 'UTF8' 
        LC_COLLATE='en_US.UTF-8' 
        LC_CTYPE='en_US.UTF-8'
    `, dbName))
    
    // Create dedicated user
    username := fmt.Sprintf("app_%s", appID)
    password := generateSecurePassword()
    
    _, err = m.adminConn.Exec(ctx, fmt.Sprintf(`
        CREATE USER %s WITH PASSWORD '%s'
    `, username, password))
    
    _, err = m.adminConn.Exec(ctx, fmt.Sprintf(`
        GRANT ALL PRIVILEGES ON DATABASE %s TO %s
    `, dbName, username))
    
    connectionString := fmt.Sprintf(
        "postgres://%s:%s@%s:5432/%s?sslmode=require",
        username, password, m.host, dbName,
    )
    
    return connectionString, nil
}

// Option B: Shared database with schemas (Alternative for high-density)
// Pros: Fewer databases, easier to manage
// Cons: Less isolation, noisy neighbor issues
func (m *DatabaseManager) CreateSchema(appID string) (string, error) {
    schemaName := fmt.Sprintf("app_%s", appID)
    
    _, err := m.conn.Exec(ctx, fmt.Sprintf(`
        CREATE SCHEMA %s
    `, schemaName))
    
    // Set search_path in connection string
    connectionString := fmt.Sprintf(
        "postgres://user:pass@host:5432/authsome_shared?sslmode=require&search_path=%s",
        schemaName,
    )
    
    return connectionString, nil
}
```

## Security Model

### API Key Security

```go
// API key format:
// pk_live_abc123def456        (public key - safe to expose in frontend)
// sk_live_abc123def456ghi789  (secret key - server-side only)

type APIKeyType string

const (
    PublicKey APIKeyType = "public"
    SecretKey APIKeyType = "secret"
)

type APIKeyGenerator struct {
    random io.Reader
}

func (g *APIKeyGenerator) Generate(appID string, env string, keyType APIKeyType) (string, error) {
    prefix := fmt.Sprintf("%s_%s_%s", keyType, env, appID[:8])
    
    // Generate cryptographically secure random suffix
    suffix := make([]byte, 32)
    if _, err := io.ReadFull(g.random, suffix); err != nil {
        return "", err
    }
    
    key := fmt.Sprintf("%s_%s", prefix, base62Encode(suffix))
    return key, nil
}

// Secret keys are hashed before storage
func (g *APIKeyGenerator) HashSecretKey(key string) string {
    hash := sha256.Sum256([]byte(key))
    return base64.StdEncoding.EncodeToString(hash[:])
}

// Verification uses constant-time comparison
func (s *Service) VerifyAPIKey(providedKey, storedHash string) bool {
    providedHash := sha256.Sum256([]byte(providedKey))
    providedHashB64 := base64.StdEncoding.EncodeToString(providedHash[:])
    
    return subtle.ConstantTimeCompare(
        []byte(providedHashB64),
        []byte(storedHash),
    ) == 1
}
```

### Network Isolation

```yaml
# Kubernetes NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: authsome-app-abc123-policy
  namespace: authsome-app-abc123
spec:
  podSelector:
    matchLabels:
      app: authsome
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Only allow proxy service to reach app
  - from:
    - namespaceSelector:
        matchLabels:
          name: authsome-system
      podSelector:
        matchLabels:
          app: proxy
    ports:
    - protocol: TCP
      port: 8080
  egress:
  # Allow DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
  # Allow PostgreSQL
  - to:
    - podSelector:
        matchLabels:
          app: postgresql
    ports:
    - protocol: TCP
      port: 5432
  # Allow Redis
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
  # Allow HTTPS egress (for OAuth, webhooks)
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443
```

### Secrets Management

```go
// Use HashiCorp Vault for sensitive data
type VaultManager struct {
    client *vault.Client
}

func (v *VaultManager) StoreCredentials(appID string, creds *Credentials) error {
    path := fmt.Sprintf("secret/data/applications/%s", appID)
    
    _, err := v.client.Logical().Write(path, map[string]interface{}{
        "data": map[string]interface{}{
            "database_url":     creds.DatabaseURL,
            "redis_url":        creds.RedisURL,
            "secret_key":       creds.SecretKey,
            "encryption_key":   creds.EncryptionKey,
        },
    })
    
    return err
}

// Kubernetes gets secrets via Vault Agent sidecar
// Secrets never stored in etcd plaintext
```

## Scaling Strategy

### Control Plane Scaling

```
Control Plane API: Stateless, horizontal scaling
├─→ Target: 80% CPU utilization
├─→ Min replicas: 3 (across AZs)
└─→ Max replicas: 20

Proxy Service: Stateless, horizontal scaling
├─→ Target: 70% CPU utilization (latency-sensitive)
├─→ Min replicas: 5
└─→ Max replicas: 50

Provisioner Workers: Queue-based scaling
├─→ Scale based on NATS queue depth
├─→ Min workers: 2
└─→ Max workers: 10
```

### Customer Application Scaling

```yaml
# Each customer app auto-scales independently
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: authsome
  namespace: authsome-app-abc123
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: authsome
  minReplicas: 2  # HA minimum
  maxReplicas: 10 # Plan-based limit
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Pods
        value: 1
        periodSeconds: 60
```

## Failure Modes

### Application Deployment Failure

```
Symptom: AuthSome instance won't start
Causes:
- Database connection failure
- Invalid configuration
- Resource limits exceeded
- Image pull failure

Recovery:
1. Provisioner sets status: "failed"
2. Logs error to application.error_log
3. Sends alert to ops team
4. Dashboard shows error with remediation steps
5. Automatic retry with exponential backoff (3 attempts)
6. User notified via email if all retries fail

Prevention:
- Pre-flight checks before provisioning
- Configuration validation
- Resource quota checks
```

### Proxy Service Failure

```
Symptom: API requests failing to route
Causes:
- Proxy pods crashed
- Cache (Redis) unavailable
- K8s DNS resolution issues

Recovery:
1. Load balancer removes unhealthy pods
2. Requests route to healthy pods
3. Cache miss falls back to database
4. Auto-healing recreates failed pods

High Availability:
- Min 5 replicas across 3 AZs
- PodDisruptionBudget ensures min 3 available during updates
- Graceful shutdown (30s drain period)
```

### Database Failure

```
Symptom: Customer application can't access database
Causes:
- Database server down
- Network partition
- Disk full
- Connection pool exhausted

Recovery:
1. AuthSome returns 503 Service Unavailable
2. Monitoring alerts ops team
3. Automated failover to standby (30s RTO)
4. Customer dashboard shows incident

Prevention:
- PostgreSQL streaming replication
- Automated backups (hourly, retained 30 days)
- Connection pool limits
- Disk space monitoring + alerts
```

### Control Plane Database Failure

```
Symptom: Can't create/manage workspaces or applications
Impact: Existing apps continue working (only control plane affected)

Recovery:
1. Failover to standby database (RPO < 1 minute)
2. Provisioning queued in NATS (durable queue)
3. Operations resume when database restored

High Availability:
- Primary + synchronous standby
- Automatic failover with Patroni
- Backups every 6 hours, retained 90 days
```

---

## Next Steps

- **[API Reference](./API.md)**: Complete API documentation
- **[Deployment Guide](./DEPLOYMENT.md)**: How to deploy control plane
- **[Billing Guide](./BILLING.md)**: Usage tracking and billing implementation
- **[Security Model](./SECURITY.md)**: Detailed security architecture

