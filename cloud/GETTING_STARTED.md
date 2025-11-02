# Getting Started with AuthSome Cloud

**Quick start guide for understanding and contributing to AuthSome Cloud**

## What is AuthSome Cloud?

AuthSome Cloud is a managed control plane that orchestrates multiple isolated AuthSome deployments, providing a **Clerk-like experience** for customers who want managed authentication infrastructure.

### Key Concepts

```
┌─────────────────────────────────────────────────────────────┐
│  Customer (e.g., Acme Corp)                                  │
│  └─→ Workspace                                               │
│      └─→ Applications (Production, Staging, Dev)            │
│          └─→ Isolated AuthSome Instances                    │
│              ├─→ Dedicated Database                          │
│              ├─→ Dedicated Redis                             │
│              ├─→ API Keys (pk_live_xxx, sk_live_xxx)        │
│              └─→ Organizations & Users (customer's end-users)│
└─────────────────────────────────────────────────────────────┘
```

**Think of it as:**
- **Workspace** = Your company's AuthSome Cloud account
- **Application** = One environment (like prod, staging, dev)
- **AuthSome Instance** = Running AuthSome core framework with your data

## Architecture Overview

### Components

```
1. Control Plane API
   └─→ Manages workspaces, applications, billing
   └─→ CRUD operations for cloud resources

2. Proxy Service
   └─→ Routes customer API requests
   └─→ /v1/app_abc123/* → Customer's AuthSome instance

3. Provisioner Workers
   └─→ Automated application provisioning
   └─→ Creates K8s namespace, database, Redis, AuthSome deployment

4. Management Dashboard
   └─→ Next.js web UI
   └─→ Workspace/application management
   └─→ Usage metrics and billing
```

### Request Flow

```
Customer App
    ↓
    ↓ POST /v1/app_abc123/auth/signup
    ↓
Cloudflare (DDoS + WAF)
    ↓
Load Balancer
    ↓
Proxy Service
    ├─→ Verify API key
    ├─→ Check rate limits
    └─→ Route to app_abc123's AuthSome instance
        ↓
        AuthSome Instance (K8s namespace: authsome-app-abc123)
        ├─→ PostgreSQL (authsome_app_abc123)
        └─→ Redis (app-abc123-cache)
```

## Documentation Index

### 📘 For Understanding

- **[README.md](./README.md)** - Project overview and quick reference
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - Deep technical architecture
- **[ROADMAP.md](./ROADMAP.md)** - Implementation phases and timeline

### 📗 For Implementation

- **[API.md](./API.md)** - Complete API specification
- **[DEPLOYMENT.md](./DEPLOYMENT.md)** - Infrastructure and deployment guide
- **[BILLING.md](./BILLING.md)** - Usage tracking and billing implementation
- **[SECURITY.md](./SECURITY.md)** - Security model and compliance

### 📁 Code Structure

```
cloud/
├── README.md                    # Overview
├── ARCHITECTURE.md              # Architecture deep dive
├── API.md                       # API reference
├── DEPLOYMENT.md                # Deployment guide
├── BILLING.md                   # Billing implementation
├── SECURITY.md                  # Security model
├── ROADMAP.md                   # Implementation roadmap
├── GETTING_STARTED.md           # This file
│
├── cmd/
│   ├── control-plane/           # Main control plane API
│   ├── proxy/                   # Request proxy service
│   ├── provisioner/             # Provisioning worker
│   └── dashboard/               # Management UI (Next.js)
│
├── control/                     # Business logic
│   ├── workspace/               # Workspace management
│   ├── application/             # Application orchestration
│   ├── team/                    # Team member management
│   └── billing/                 # Billing service
│
├── internal/                    # Internal packages
│   ├── k8s/                     # Kubernetes client
│   ├── database/                # Database provisioning
│   ├── cache/                   # Redis provisioning
│   ├── monitoring/              # Metrics and alerting
│   ├── proxy/                   # Proxy logic
│   └── events/                  # Event bus
│
├── schema/                      # Data models
├── repository/                  # Data access
├── migrations/                  # Database migrations
├── k8s/                         # Kubernetes manifests
└── docs/                        # Additional documentation
```

## Quick Start (for Developers)

### Prerequisites

```bash
# Required tools
- Go 1.21+
- Docker
- kubectl
- Helm 3
- Node.js 18+ (for dashboard)
```

### Local Development Setup

```bash
# 1. Clone repository
git clone https://github.com/xraph/authsome-cloud
cd authsome-cloud

# 2. Start dependencies (PostgreSQL, Redis, NATS)
docker-compose up -d

# 3. Run migrations
go run cmd/migrate/main.go up

# 4. Start control plane API
cd cmd/control-plane
go run main.go

# 5. Start dashboard (separate terminal)
cd cmd/dashboard
npm install
npm run dev

# 6. Access dashboard
open http://localhost:3000
```

### Development Workflow

```bash
# Run tests
go test ./...

# Run linters
golangci-lint run

# Build images
docker build -t authsome/control-plane -f cmd/control-plane/Dockerfile .

# Deploy to local K8s
kubectl apply -f k8s/development/
```

## Understanding the Codebase

### Key Files to Read First

1. **[cloud/README.md](./README.md)** - Start here
2. **[cloud/ARCHITECTURE.md](./ARCHITECTURE.md)** - System design
3. **`control/application/service.go`** - Application lifecycle
4. **`internal/k8s/provisioner.go`** - Kubernetes orchestration
5. **`cmd/proxy/main.go`** - Request routing

### Core Abstractions

```go
// Workspace = Customer's cloud account
type Workspace struct {
    ID     string
    Name   string
    Plan   string  // free, pro, enterprise
}

// Application = One environment (prod, staging, dev)
type Application struct {
    ID          string
    WorkspaceID string
    Name        string
    Environment string
    Status      string  // provisioning, active, suspended
    
    // Deployment details
    DatabaseURL   string
    RedisURL      string
    PublicKey     string  // pk_live_xxx
    SecretKeyHash string  // hashed sk_live_xxx
}

// Provisioning orchestrates the creation
func (s *Service) Provision(app *Application) error {
    // 1. Create K8s namespace
    // 2. Create PostgreSQL database
    // 3. Create Redis instance
    // 4. Deploy AuthSome
    // 5. Run migrations
    // 6. Setup monitoring
}
```

## Common Tasks

### Adding a New API Endpoint

```go
// 1. Add route in cmd/control-plane/routes.go
app.POST("/api/v1/workspaces/:id/custom", handler.CustomAction)

// 2. Implement handler
func (h *Handler) CustomAction(c *forge.Context) error {
    workspaceID := c.Param("id")
    
    // Business logic
    result, err := h.service.DoSomething(c.Context(), workspaceID)
    if err != nil {
        return c.JSON(500, ErrorResponse{Message: err.Error()})
    }
    
    return c.JSON(200, result)
}

// 3. Add tests
func TestCustomAction(t *testing.T) {
    // Test implementation
}

// 4. Update API.md documentation
```

### Adding Usage Tracking

```go
// Track new metric
func (t *Tracker) TrackCustomMetric(appID string, value int64) error {
    // Store in PostgreSQL or Redis
    query := `
        INSERT INTO usage_custom (application_id, period, value)
        VALUES ($1, $2, $3)
        ON CONFLICT (application_id, period)
        DO UPDATE SET value = usage_custom.value + EXCLUDED.value
    `
    
    period := time.Now().Format("2006-01")
    _, err := t.db.Exec(ctx, query, appID, period, value)
    return err
}

// Include in invoice
func (g *Generator) generateInvoice(ws *Workspace) *Invoice {
    // ... existing metrics
    
    // Add new metric
    customValue, _ := g.usageRepo.GetCustomMetric(ws.ID)
    if customValue > threshold {
        invoice.AddLineItem("Custom Feature", customValue * pricePerUnit)
    }
}
```

### Debugging Application Provisioning

```bash
# Check provisioner logs
kubectl logs -n authsome-system -l app=provisioner -f

# Check application status
kubectl get all -n authsome-app-abc123

# Check AuthSome logs
kubectl logs -n authsome-app-abc123 -l app=authsome -f

# Check database connection
kubectl exec -n authsome-app-abc123 deploy/authsome -- \
  psql $DATABASE_URL -c "SELECT 1"

# Check provisioning events
kubectl get events -n authsome-app-abc123 --sort-by='.lastTimestamp'
```

## FAQ

### When should we build AuthSome Cloud?

**After AuthSome core framework reaches v1.0.** The cloud control plane is built ON TOP of the stable core framework.

### Can customers use both self-hosted and cloud?

**Yes!** Self-hosted remains a first-class option. Cloud is for customers who want managed infrastructure.

### How is this different from Clerk/Auth0?

- **Clerk/Auth0:** Closed-source SaaS only
- **AuthSome:** Open-source core + optional managed cloud
- **Benefit:** Customers can start self-hosted and migrate to cloud later

### What about data residency?

Enterprise customers can choose:
- Region (US, EU, APAC)
- VPC peering (data never leaves their VPC)
- Dedicated instances (no shared infrastructure)

### How do we make money?

```
Self-Hosted: Free (open source)
  └─→ Revenue: Enterprise support contracts, consulting

Cloud:
  ├─→ Free tier: $0/month (10K MAU, 3 apps)
  ├─→ Pro tier: $25/month + usage
  └─→ Enterprise: Custom pricing
```

### What's the hardest part?

1. **Automated provisioning** - Reliable K8s orchestration
2. **Usage tracking accuracy** - Correct billing is critical
3. **Multi-tenant isolation** - Security is paramount
4. **Cost management** - Infrastructure costs can spiral

### How do we ensure security?

- Database-per-application (complete isolation)
- Network policies (strict ingress/egress)
- Encryption at rest and in transit
- SOC 2 Type II compliance
- Regular security audits
- Bug bounty program

## Getting Help

### Documentation
- **Architecture questions:** Read [ARCHITECTURE.md](./ARCHITECTURE.md)
- **API questions:** Read [API.md](./API.md)
- **Deployment questions:** Read [DEPLOYMENT.md](./DEPLOYMENT.md)
- **Security questions:** Read [SECURITY.md](./SECURITY.md)

### Support Channels
- **GitHub Issues:** Bug reports and feature requests
- **GitHub Discussions:** General questions
- **Discord:** Real-time chat with team
- **Email:** cloud@authsome.dev

### Contributing
- Read CONTRIBUTING.md (to be created)
- Check ROADMAP.md for current focus
- Pick issues labeled "good first issue"
- Submit PR with tests and documentation

## Next Steps

### If you're new to the codebase:
1. ✅ Read this file (you're here!)
2. ⬜ Read [ARCHITECTURE.md](./ARCHITECTURE.md) for system design
3. ⬜ Read [API.md](./API.md) to understand the API
4. ⬜ Run local development setup
5. ⬜ Pick a "good first issue" to work on

### If you're planning deployment:
1. ⬜ Read [DEPLOYMENT.md](./DEPLOYMENT.md)
2. ⬜ Review [SECURITY.md](./SECURITY.md)
3. ⬜ Provision infrastructure (EKS/GKE)
4. ⬜ Deploy control plane
5. ⬜ Test with beta users

### If you're building features:
1. ⬜ Check [ROADMAP.md](./ROADMAP.md) for priorities
2. ⬜ Design the feature (create RFC if major)
3. ⬜ Implement with tests
4. ⬜ Update documentation
5. ⬜ Submit PR for review

---

**Welcome to AuthSome Cloud! Let's build the future of authentication infrastructure together.** 🚀

