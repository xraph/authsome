# AuthSome Cloud

**Managed multi-tenant control plane for AuthSome framework**

AuthSome Cloud is a SaaS offering that orchestrates multiple isolated AuthSome deployments, providing a Clerk-like experience for customers who want managed authentication infrastructure.

## Architecture Philosophy

```
┌─────────────────────────────────────────────────────────────┐
│  AuthSome Core Framework (github.com/xraph/authsome)       │
│  • Self-hosted: Customers deploy themselves                 │
│  • Single deployment = Single "application"                  │
│  • No cloud concepts in core                                 │
└─────────────────────────────────────────────────────────────┘
                           ▲
                           │ Uses
                           │
┌─────────────────────────────────────────────────────────────┐
│  AuthSome Cloud Control Plane (this repository)             │
│  • Orchestrates multiple AuthSome deployments                │
│  • Workspace → Applications → Isolated instances             │
│  • Billing, monitoring, management dashboard                 │
└─────────────────────────────────────────────────────────────┘
```

## Key Principles

1. **Core Framework Remains Cloud-Agnostic**: No cloud-specific code in main AuthSome repository
2. **Database-per-Application**: Complete isolation between customer applications
3. **Kubernetes-Native**: Each application deployed as isolated namespace
4. **API-Compatible**: Cloud API wraps AuthSome API with workspace/app routing
5. **Migration-Friendly**: Self-hosted users can import existing deployments

## Hierarchy Model

```
Workspace (Customer Account)
├── Team Members (workspace admins)
├── Billing & Subscription
└── Applications (Isolated Environments)
    ├── Production App
    │   ├── AuthSome Deployment (isolated)
    │   ├── Database (dedicated)
    │   ├── Redis (dedicated)
    │   ├── API Keys: pk_live_xxx, sk_live_xxx
    │   └── Organizations & Users (customer's data)
    │
    ├── Staging App
    │   ├── AuthSome Deployment (isolated)
    │   ├── Database (dedicated)
    │   ├── Redis (dedicated)
    │   └── API Keys: pk_test_xxx, sk_test_xxx
    │
    └── Development App
        └── (same structure)
```

## Repository Structure

```
cloud/
├── README.md                    # This file
├── ARCHITECTURE.md              # Detailed architecture
├── API.md                       # API specification
├── DEPLOYMENT.md                # Deployment guide
├── BILLING.md                   # Billing & pricing
├── SECURITY.md                  # Security model
├── cmd/
│   ├── control-plane/           # Control plane API server
│   ├── provisioner/             # Application provisioner worker
│   ├── proxy/                   # API proxy service
│   └── dashboard/               # Management dashboard (Next.js)
├── control/
│   ├── workspace/               # Workspace management
│   ├── application/             # Application orchestration
│   ├── team/                    # Team member management
│   └── billing/                 # Billing service
├── internal/
│   ├── k8s/                     # Kubernetes orchestration
│   ├── database/                # Database provisioning
│   ├── cache/                   # Redis provisioning
│   ├── monitoring/              # Prometheus/Grafana setup
│   ├── proxy/                   # Request routing
│   └── events/                  # Event bus
├── schema/
│   ├── workspace.go             # Workspace models
│   ├── application.go           # Application models
│   └── usage.go                 # Usage tracking models
├── repository/
│   ├── workspace.go             # Workspace repository
│   └── application.go           # Application repository
├── migrations/
│   └── control-plane/           # Control plane DB migrations
├── k8s/
│   ├── templates/               # Kubernetes YAML templates
│   ├── helm/                    # Helm charts
│   └── operators/               # Custom operators
├── docs/
│   ├── getting-started.md       # User documentation
│   ├── api-reference.md         # API docs
│   └── migration-guide.md       # Self-hosted → Cloud
└── examples/
    └── client-libraries/        # SDK examples
```

## Quick Start

### For Cloud Operators

```bash
# Clone control plane
git clone https://github.com/xraph/authsome-cloud
cd authsome-cloud

# Deploy control plane infrastructure
kubectl apply -f k8s/control-plane/

# Deploy control plane API
cd cmd/control-plane
go run main.go

# Deploy management dashboard
cd cmd/dashboard
npm install && npm run dev
```

### For Cloud Customers

```bash
# Sign up at dashboard.authsome.cloud
# Create workspace → Create application
# Get API keys

# Use in your app:
export AUTHSOME_API_KEY=pk_live_abc123
export AUTHSOME_SECRET_KEY=sk_live_abc123
export AUTHSOME_API_URL=https://api.authsome.cloud/v1/app_abc123

# Make requests
curl https://api.authsome.cloud/v1/app_abc123/users \
  -H "Authorization: Bearer $AUTHSOME_SECRET_KEY"
```

## Key Features

### Multi-Tenant Orchestration
- Workspace-scoped isolation
- Application lifecycle management (create, update, delete)
- Automatic provisioning (database, cache, compute)
- Zero-downtime deployments

### Intelligent Routing
- API key-based request routing
- Automatic proxy to customer's AuthSome instance
- Load balancing and health checks
- Edge caching for static responses

### Billing & Usage Tracking
- Monthly Active Users (MAU) tracking
- API request metering
- Storage usage monitoring
- Stripe integration for payments

### Observability
- Per-application metrics (Prometheus)
- Distributed tracing (Jaeger)
- Centralized logging (Loki)
- Alerting for anomalies

### Security
- Isolated network namespaces
- Database encryption at rest
- Secrets management (Vault)
- DDoS protection (Cloudflare)
- SOC 2 Type II compliance

## Pricing Model

```
Free Tier:
- 10,000 MAU
- 3 Applications
- Community support
- 7-day data retention

Pro Tier ($25/month):
- 10,000 MAU included
- $0.02 per additional MAU
- Unlimited applications
- Email support
- 30-day data retention
- Custom domains

Enterprise Tier (Custom):
- Volume pricing
- Dedicated instances
- 99.99% SLA
- Priority support
- Custom data retention
- VPC peering
- SAML SSO for dashboard
```

## Technology Stack

**Control Plane:**
- Go (control plane API, provisioner, proxy)
- PostgreSQL (control plane database)
- Redis (session storage, rate limiting)
- NATS (event bus for async operations)

**Orchestration:**
- Kubernetes (container orchestration)
- Helm (package management)
- Terraform (infrastructure as code)

**Monitoring:**
- Prometheus (metrics)
- Grafana (dashboards)
- Loki (logs)
- Jaeger (tracing)

**Dashboard:**
- Next.js 14 (App Router)
- TypeScript
- Tailwind CSS
- shadcn/ui components

**Billing:**
- Stripe (payments)
- Custom usage tracking

## Development Roadmap

### Phase 1: Control Plane Core (Months 1-3)
- [ ] Workspace CRUD
- [ ] Application provisioning
- [ ] Database/Redis provisioning automation
- [ ] Basic API proxy
- [ ] API key generation/verification

### Phase 2: Dashboard & Billing (Months 4-6)
- [ ] Management dashboard UI
- [ ] Team member management
- [ ] Usage tracking system
- [ ] Stripe integration
- [ ] Invoicing

### Phase 3: Enterprise Features (Months 7-9)
- [ ] Custom domains (app-abc123.authsome.cloud → auth.customer.com)
- [ ] VPC peering
- [ ] Dedicated instances
- [ ] SAML SSO for dashboard
- [ ] Advanced monitoring

### Phase 4: Advanced Operations (Months 10-12)
- [ ] Multi-region support
- [ ] Automated backups & PITR
- [ ] Blue-green deployments
- [ ] Self-service migration from self-hosted
- [ ] Compliance certifications (SOC 2, ISO 27001)

## Documentation

- **[Architecture](./ARCHITECTURE.md)**: Deep dive into system design
- **[API Reference](./API.md)**: Complete API documentation
- **[Deployment Guide](./DEPLOYMENT.md)**: How to deploy control plane
- **[Billing Guide](./BILLING.md)**: Pricing and billing implementation
- **[Security Model](./SECURITY.md)**: Security architecture and compliance

## Contributing

AuthSome Cloud is proprietary software. For contributing to the open-source AuthSome core framework, see [github.com/xraph/authsome](https://github.com/xraph/authsome).

## Support

- **Cloud Status**: https://status.authsome.cloud
- **Documentation**: https://docs.authsome.cloud
- **Support**: support@authsome.cloud
- **Enterprise Sales**: sales@authsome.cloud

## License

Proprietary. © 2025 AuthSome Inc.

---

**Ready to build the future of authentication infrastructure.**

