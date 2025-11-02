# AuthSome Cloud Architecture Diagrams

**Visual representations of the system architecture**

---

## High-Level System Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│                           Internet / Customer Apps                          │
└──────────────────────────────────┬─────────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                     Cloudflare (Edge Layer)                                 │
│  ┌──────────────┬──────────────┬──────────────┬──────────────┐            │
│  │ DDoS         │ WAF          │ Rate         │ TLS          │            │
│  │ Protection   │ Rules        │ Limiting     │ Termination  │            │
│  └──────────────┴──────────────┴──────────────┴──────────────┘            │
└──────────────────────────────────┬─────────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster (EKS/GKE/AKS)                        │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐ │
│  │                    System Namespace (authsome-system)                 │ │
│  │                                                                       │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐     │ │
│  │  │ Control Plane   │  │ Proxy Service   │  │ Provisioner     │     │ │
│  │  │ API (3 pods)    │  │ (5 pods)        │  │ Workers (3)     │     │ │
│  │  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘     │ │
│  │           │                    │                    │               │ │
│  │           └────────────────────┼────────────────────┘               │ │
│  │                                │                                    │ │
│  │  ┌─────────────────────────────┼─────────────────────────────────┐ │ │
│  │  │                             │                                   │ │ │
│  │  │  ┌────────────────┐  ┌──────▼──────┐  ┌──────────────┐       │ │ │
│  │  │  │ PostgreSQL     │  │ Redis       │  │ NATS         │       │ │ │
│  │  │  │ (Control DB)   │  │ (Cache)     │  │ (Queue)      │       │ │ │
│  │  │  └────────────────┘  └─────────────┘  └──────────────┘       │ │ │
│  │  └───────────────────────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                           │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │              Application Namespaces (Customer Isolation)           │  │
│  │                                                                    │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │ Namespace: authsome-app-abc123 (Customer A - Production)    │  │  │
│  │  │                                                              │  │  │
│  │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │  │  │
│  │  │  │ AuthSome     │──│ PostgreSQL   │  │ Redis        │     │  │  │
│  │  │  │ (2 pods)     │  │ (Dedicated)  │  │ (Dedicated)  │     │  │  │
│  │  │  └──────────────┘  └──────────────┘  └──────────────┘     │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │                                                                    │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │ Namespace: authsome-app-def456 (Customer B - Production)    │  │  │
│  │  │                                                              │  │  │
│  │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │  │  │
│  │  │  │ AuthSome     │──│ PostgreSQL   │  │ Redis        │     │  │  │
│  │  │  │ (2 pods)     │  │ (Dedicated)  │  │ (Dedicated)  │     │  │  │
│  │  │  └──────────────┘  └──────────────┘  └──────────────┘     │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │                                                                    │  │
│  │  ... (More customer namespaces)                                   │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Customer API Request Flow

```
┌─────────────────┐
│ Customer App    │ 1. POST /v1/app_abc123/auth/signup
│ (e.g. Mobile)   │    Authorization: Bearer sk_live_abc123_xyz...
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Cloudflare Edge │ 2. DDoS check, WAF rules, rate limit
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Load Balancer   │ 3. Route to Proxy Service
└────────┬────────┘
         │
         ▼
┌────────────────────────────────────────────┐
│ Proxy Service                               │
│                                            │
│ 4a. Extract app_abc123 from URL           │
│ 4b. Extract sk_live_abc123 from header    │
│ 4c. Verify API key (cache or DB)          │
│ 4d. Check application status: active      │
│ 4e. Get internal URL from cache           │
│     → http://authsome.authsome-app-abc123.svc.cluster.local
│ 4f. Track usage (async to NATS)           │
└────────┬───────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────────┐
│ AuthSome Instance (app-abc123 namespace)   │
│                                            │
│ 5a. Process authentication request         │
│ 5b. Hash password                          │
│ 5c. Query PostgreSQL (dedicated DB)       │
│     INSERT INTO users ...                  │
│ 5d. Create session                         │
│ 5e. Cache session in Redis (dedicated)    │
│ 5f. Return user + session token            │
└────────┬───────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────────┐
│ Proxy Service                               │
│                                            │
│ 6a. Receive response                       │
│ 6b. Add headers:                           │
│     X-AuthSome-App-ID: app_abc123         │
│     X-AuthSome-Request-ID: req_xyz        │
│ 6c. Return to client                       │
└────────┬───────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│ Customer App    │ 7. Receive response with user + session
└─────────────────┘
```

---

## Application Provisioning Flow

```
┌──────────────────┐
│ Customer creates │ 1. POST /control/v1/workspaces/ws_xyz/applications
│ app via Dashboard│    { name: "Production", environment: "production" }
└────────┬─────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Control Plane API                               │
│                                                 │
│ 2a. Validate request                           │
│ 2b. Generate app ID: app_abc123                │
│ 2c. Generate API keys:                         │
│     - Public: pk_live_abc123def456             │
│     - Secret: sk_live_abc123def456ghi789       │
│ 2d. Create application record:                 │
│     - Status: provisioning                     │
│ 2e. Publish to NATS: "app.provision"          │
│ 2f. Return 202 Accepted                        │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ NATS Queue                                      │
│ Topic: app.provision                            │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────┐
│ Provisioner Worker                              │
│                                                 │
│ 3. Process provisioning job                    │
└────────┬────────────────────────────────────────┘
         │
         ├──► 4a. Create Kubernetes Namespace
         │    └─→ kubectl create namespace authsome-app-abc123
         │
         ├──► 4b. Provision PostgreSQL Database
         │    ├─→ CREATE DATABASE authsome_app_abc123
         │    ├─→ CREATE USER app_abc123_user
         │    └─→ GRANT ALL PRIVILEGES
         │
         ├──► 4c. Provision Redis Instance
         │    └─→ Deploy Redis StatefulSet
         │
         ├──► 4d. Deploy AuthSome Instance
         │    ├─→ Create ConfigMap (AuthSome config)
         │    ├─→ Create Secret (DB credentials)
         │    ├─→ Deploy AuthSome (2 replicas)
         │    ├─→ Create Service (ClusterIP)
         │    └─→ Wait for rollout complete
         │
         ├──► 4e. Run Database Migrations
         │    └─→ Initialize AuthSome schema
         │
         ├──► 4f. Setup Monitoring
         │    ├─→ Create ServiceMonitor (Prometheus)
         │    ├─→ Create Grafana dashboard
         │    └─→ Configure alerts
         │
         └──► 4g. Update Application Status
              └─→ Status: active
         
         ▼
┌─────────────────────────────────────────────────┐
│ Control Plane DB                                │
│                                                 │
│ UPDATE applications                             │
│ SET status = 'active',                          │
│     internal_url = 'http://authsome.authsome-app-abc123.svc.cluster.local'
│ WHERE id = 'app_abc123'                         │
└────────┬────────────────────────────────────────┘
         │
         ▼
┌──────────────────┐
│ Email to Customer│ 5. Application ready!
│                  │    API URL: https://api.authsome.cloud/v1/app_abc123
└──────────────────┘
```

---

## Multi-Tenant Isolation

```
┌────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster                      │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Namespace: authsome-app-abc123 (Customer A)               │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────────┐ │ │
│  │  │ Network Policy: Strict Ingress/Egress                │ │ │
│  │  │ ┌──────────────────────────────────────────────────┐ │ │ │
│  │  │ │ Ingress: Only from proxy service                 │ │ │ │
│  │  │ │ Egress: Only to DB, Redis, DNS, HTTPS           │ │ │ │
│  │  │ └──────────────────────────────────────────────────┘ │ │ │
│  │  └──────────────────────────────────────────────────────┘ │ │
│  │                                                            │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │ │
│  │  │ AuthSome Pod │  │ PostgreSQL   │  │ Redis Pod    │   │ │
│  │  │              │──│ Pod          │  │              │   │ │
│  │  │ Resources:   │  │              │  │ Resources:   │   │ │
│  │  │ CPU: 1 core  │  │ Database:    │  │ Memory: 1Gi  │   │ │
│  │  │ RAM: 2Gi     │  │ authsome_    │  │              │   │ │
│  │  │              │  │ app_abc123   │  │              │   │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │ │
│  │                                                            │ │
│  │  ServiceAccount: app-abc123-sa (no cluster-wide access)   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ────────────────────── ISOLATED ──────────────────────────    │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Namespace: authsome-app-def456 (Customer B)               │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────────┐ │ │
│  │  │ Network Policy: Strict Ingress/Egress                │ │ │
│  │  │ (Independent from Customer A)                        │ │ │
│  │  └──────────────────────────────────────────────────────┘ │ │
│  │                                                            │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │ │
│  │  │ AuthSome Pod │  │ PostgreSQL   │  │ Redis Pod    │   │ │
│  │  │              │──│ Pod          │  │              │   │ │
│  │  │ Resources:   │  │              │  │ Resources:   │   │ │
│  │  │ CPU: 1 core  │  │ Database:    │  │ Memory: 1Gi  │   │ │
│  │  │ RAM: 2Gi     │  │ authsome_    │  │              │   │ │
│  │  │              │  │ app_def456   │  │              │   │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │ │
│  │                                                            │ │
│  │  ServiceAccount: app-def456-sa (no cluster-wide access)   │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘

Key Isolation Points:
✓ Network: Namespace-scoped NetworkPolicies
✓ Compute: ResourceQuotas per namespace
✓ Storage: Dedicated databases per application
✓ Identity: Separate ServiceAccounts
✓ No cross-namespace communication allowed
```

---

## Data Model Relationships

```
┌──────────────────────────────────────────────────────────────────┐
│                     Control Plane Database                        │
└──────────────────────────────────────────────────────────────────┘

┌────────────────┐
│   User         │
│                │ Authenticates to dashboard
│ - id           │
│ - email        │
│ - password     │
│ - totp_enabled │
└───────┬────────┘
        │
        │ has many
        ▼
┌───────────────────┐
│ TeamMember        │ belongs to
│                   ├──────────┐
│ - workspace_id    │          │
│ - user_id         │          │
│ - role            │          │
│   (owner/admin)   │          │
└───────────────────┘          │
                               │
                               ▼
                     ┌───────────────────┐
                     │  Workspace        │
                     │                   │
                     │ - id              │
                     │ - name            │
                     │ - plan            │
                     │ - stripe_customer │
                     └────────┬──────────┘
                              │
                              │ has many
                              ▼
                     ┌────────────────────────┐
                     │  Application           │
                     │                        │
                     │ - id                   │
                     │ - workspace_id         │
                     │ - name                 │
                     │ - environment          │
                     │ - status               │
                     │ - database_url         │
                     │ - redis_url            │
                     │ - k8s_namespace        │
                     └────────┬───────────────┘
                              │
                              │ has many
                              ▼
                     ┌────────────────────────┐
                     │  APIKey                │
                     │                        │
                     │ - application_id       │
                     │ - type (public/secret) │
                     │ - key_hash             │
                     │ - scopes               │
                     │ - last_used_at         │
                     └────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│              Customer Application Database                        │
│              (One per application - Complete isolation)           │
└──────────────────────────────────────────────────────────────────┘

Database: authsome_app_abc123

┌────────────────┐
│   User         │ ← Customer's end-users
│                │
│ - id           │
│ - email        │
│ - password     │
└───────┬────────┘
        │
        │ has many
        ▼
┌───────────────────┐
│ Session           │
│                   │
│ - user_id         │
│ - token           │
│ - expires_at      │
└───────────────────┘

┌───────────────────┐
│ Organization      │ ← Customer's organizations (SaaS mode)
│                   │
│ - id              │
│ - name            │
└────────┬──────────┘
         │
         │ has many
         ▼
┌───────────────────┐
│ Member            │
│                   │
│ - org_id          │
│ - user_id         │
│ - role            │
└───────────────────┘
```

---

## Billing Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                         Usage Tracking                           │
└─────────────────────────────────────────────────────────────────┘

Step 1: Track User Activity (Every Auth Request)
┌──────────────┐
│ Proxy Service│ → Track MAU
└──────┬───────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ Redis HyperLogLog                       │
│                                         │
│ Key: mau:app_abc123:2025-11            │
│ PFADD mau:app_abc123:2025-11 user_789  │
│                                         │
│ Efficient unique counting (12KB memory) │
└─────────────────────────────────────────┘

Step 2: Track API Requests (Every Request)
┌──────────────┐
│ Proxy Service│ → Publish event
└──────┬───────┘
       │
       ▼
┌─────────────────────┐
│ NATS               │
│ Topic: billing.req │
└──────┬──────────────┘
       │
       ▼
┌────────────────────────────────────┐
│ Request Aggregator Worker          │
│                                    │
│ Batch INSERT into PostgreSQL:     │
│ usage_requests table               │
└────────────────────────────────────┘

Step 3: Calculate Storage (Hourly)
┌────────────────────────────────────┐
│ Storage Calculator Cron            │
│                                    │
│ FOR EACH application:              │
│   size = pg_database_size(db)     │
│   INSERT INTO usage_storage        │
└────────────────────────────────────┘

Step 4: Generate Invoice (Monthly - 1st of month)
┌────────────────────────────────────┐
│ Invoice Generator                  │
│                                    │
│ FOR EACH workspace:                │
│   - Get MAU from Redis             │
│   - Get storage from DB            │
│   - Calculate costs                │
│   - Create Stripe invoice          │
│   - Email customer                 │
└────────────────────────────────────┘

Step 5: Process Payment
┌────────────────────────────────────┐
│ Stripe                             │
│ - Charge customer                  │
│ - Send webhook: invoice.paid       │
└──────┬─────────────────────────────┘
       │
       ▼
┌────────────────────────────────────┐
│ Webhook Handler                    │
│ - Update invoice status            │
│ - Send receipt email               │
└────────────────────────────────────┘
```

---

## Monitoring & Alerting

```
┌────────────────────────────────────────────────────────────────┐
│                      Observability Stack                        │
└────────────────────────────────────────────────────────────────┘

Metrics Collection:
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Control Plane│  │ Proxy Service│  │ AuthSome Pods│
│              │  │              │  │              │
│ /metrics     │  │ /metrics     │  │ /metrics     │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       └─────────────────┼─────────────────┘
                         │
                         ▼ scrape (15s interval)
                 ┌───────────────┐
                 │  Prometheus   │
                 │               │
                 │ - 30d storage │
                 │ - PromQL      │
                 └───────┬───────┘
                         │
                         ▼ query
                 ┌───────────────┐
                 │   Grafana     │
                 │               │
                 │ - Dashboards  │
                 │ - Alerts      │
                 └───────────────┘

Log Collection:
┌──────────────┐  ┌──────────────┐
│ All Pods     │  │ All Pods     │
│              │  │              │
│ stdout/stderr│  │ stdout/stderr│
└──────┬───────┘  └──────┬───────┘
       │                 │
       └─────────────────┘
                │
                ▼ collect
        ┌───────────────┐
        │  Loki         │
        │               │
        │ - 30d storage │
        │ - LogQL       │
        └───────┬───────┘
                │
                ▼ query
        ┌───────────────┐
        │   Grafana     │
        │               │
        │ - Log explorer│
        └───────────────┘

Distributed Tracing:
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Proxy        │→ │ AuthSome Pod │→ │ PostgreSQL   │
│ trace_id: 123│  │ trace_id: 123│  │ trace_id: 123│
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       └─────────────────┼─────────────────┘
                         │
                         ▼ send spans
                 ┌───────────────┐
                 │   Jaeger      │
                 │               │
                 │ - Trace UI    │
                 │ - Span search │
                 └───────────────┘

Alerting:
┌───────────────┐
│  Prometheus   │
│  Alert Rules  │
└───────┬───────┘
        │ evaluate
        ▼
┌───────────────────────────────────┐
│ Alertmanager                      │
│ - Deduplication                   │
│ - Grouping                        │
│ - Routing                         │
└───────┬───────────────────────────┘
        │
        ├─→ PagerDuty (critical)
        ├─→ Slack (warning)
        └─→ Email (info)
```

---

**For more diagrams, see individual documentation files or request specific visualizations.**

