# AuthSome Cloud Security Model

**Comprehensive security architecture and compliance framework**

## Table of Contents

- [Security Principles](#security-principles)
- [Architecture Security](#architecture-security)
- [Data Security](#data-security)
- [Network Security](#network-security)
- [Access Control](#access-control)
- [Compliance](#compliance)
- [Incident Response](#incident-response)
- [Auditing](#auditing)

## Security Principles

### Core Tenets

1. **Defense in Depth**: Multiple layers of security controls
2. **Least Privilege**: Minimal access rights for all entities
3. **Zero Trust**: Never trust, always verify
4. **Secure by Default**: Security configured out-of-the-box
5. **Transparency**: Clear communication about security practices

### Shared Responsibility Model

```
┌─────────────────────────────────────────────┐
│ Customer Responsibilities                    │
├─────────────────────────────────────────────┤
│ • Application-level security                 │
│ • User access management (within their app) │
│ • Secure coding practices                    │
│ • API key management                         │
│ • OAuth configuration                        │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ AuthSome Cloud Responsibilities              │
├─────────────────────────────────────────────┤
│ • Infrastructure security                    │
│ • Network isolation                          │
│ • Data encryption                            │
│ • Platform access control                    │
│ • Compliance certifications                  │
│ • Security monitoring & incident response    │
│ • Patches & updates                          │
└─────────────────────────────────────────────┘
```

## Architecture Security

### Multi-Tenant Isolation

#### Database Isolation

```
Complete database isolation per application:

app_abc123 → postgres://db1.internal/authsome_app_abc123
  ├─→ Dedicated connection pool (max 100 connections)
  ├─→ Separate credentials (app_abc123_user)
  ├─→ No shared tables or schemas
  └─→ Independent backups

app_def456 → postgres://db2.internal/authsome_app_def456
  ├─→ Dedicated connection pool
  ├─→ Separate credentials (app_def456_user)
  ├─→ No shared tables or schemas
  └─→ Independent backups
```

**Benefits:**
- Complete data isolation
- No risk of cross-tenant data leakage
- Independent scaling and performance
- Customer-specific backup/restore

#### Network Isolation

```yaml
# Kubernetes NetworkPolicy per application namespace
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: application-isolation
  namespace: authsome-app-abc123
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Only allow proxy service
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
  # Allow PostgreSQL (same namespace only)
  - to:
    - podSelector:
        matchLabels:
          app: postgresql
    ports:
    - protocol: TCP
      port: 5432
  # Allow Redis (same namespace only)
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
  # Allow HTTPS egress (OAuth, webhooks)
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443
```

#### Compute Isolation

```
Each application gets:
- Isolated Kubernetes namespace
- Resource quotas (CPU, memory, storage)
- Separate service accounts
- No cross-namespace communication

Example:
namespace: authsome-app-abc123
  ├─→ ResourceQuota: 4 CPU, 8Gi RAM
  ├─→ LimitRange: Min/Max per pod
  ├─→ ServiceAccount: app-abc123-sa
  └─→ NetworkPolicy: Strict ingress/egress
```

## Data Security

### Encryption at Rest

#### Database Encryption

```
PostgreSQL Transparent Data Encryption (TDE):
- All data files encrypted (AES-256)
- Encryption keys managed by AWS KMS / Google Cloud KMS
- Automatic key rotation (annual)
- Backup files also encrypted

Configuration:
RDS: storage_encrypted = true
GCP Cloud SQL: disk_encryption_configuration = { kms_key_name }
```

#### Backup Encryption

```go
// Backups encrypted before storage
type BackupService struct {
    kms        KMSClient
    storage    ObjectStorage
}

func (s *BackupService) Backup(ctx context.Context, appID string) error {
    // 1. Dump database
    dump, err := s.dumpDatabase(ctx, appID)
    if err != nil {
        return err
    }
    
    // 2. Compress
    compressed := gzip.Compress(dump)
    
    // 3. Encrypt with KMS
    dataKey, encryptedKey, err := s.kms.GenerateDataKey(ctx)
    if err != nil {
        return err
    }
    
    encrypted := s.encryptWithKey(compressed, dataKey)
    
    // 4. Upload to S3 with server-side encryption
    key := fmt.Sprintf("backups/%s/%s.sql.gz.enc", appID, time.Now().Format("2006-01-02"))
    
    return s.storage.Put(ctx, key, encrypted, &PutOptions{
        ServerSideEncryption: "aws:kms",
        KMSKeyID:            s.kmsKeyID,
        Metadata: map[string]string{
            "encrypted_data_key": base64.StdEncoding.EncodeToString(encryptedKey),
        },
    })
}
```

#### Secrets Management

```
All secrets stored in HashiCorp Vault:
- Database credentials
- Redis passwords
- API keys
- Encryption keys
- OAuth secrets

Access via:
- Vault Agent sidecar in Kubernetes pods
- Dynamic secret generation
- Automatic rotation
- Audit logging
```

```yaml
# Pod with Vault Agent sidecar
apiVersion: v1
kind: Pod
metadata:
  name: authsome-app
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "authsome-app"
    vault.hashicorp.com/agent-inject-secret-database: "secret/data/apps/app_abc123/database"
spec:
  serviceAccountName: app-abc123-sa
  containers:
  - name: authsome
    image: authsome/authsome:latest
    env:
    - name: DATABASE_URL
      value: file:///vault/secrets/database
```

### Encryption in Transit

#### TLS Everywhere

```
All connections encrypted with TLS 1.3:

External:
- Client → Cloudflare → Load Balancer (TLS 1.3)
- Minimum cipher: TLS_AES_128_GCM_SHA256

Internal:
- Proxy → Application (TLS 1.3)
- Application → PostgreSQL (sslmode=require)
- Application → Redis (TLS enabled)
- Application → NATS (TLS enabled)

Certificate Management:
- Let's Encrypt (external domains)
- cert-manager (internal services)
- Automatic renewal
- 90-day rotation
```

#### mTLS for Service-to-Service

```yaml
# Service mesh (Linkerd) for mTLS
apiVersion: v1
kind: Namespace
metadata:
  name: authsome-app-abc123
  annotations:
    linkerd.io/inject: enabled
---
# Automatic mTLS between all pods in namespace
# No configuration needed - transparent
```

### Data Classification

```
Level 1: Public
- API documentation
- Marketing materials
- Public dashboards

Level 2: Internal
- Application metrics
- Log aggregates
- Usage statistics

Level 3: Confidential
- Customer configuration
- API keys (hashed)
- Billing information

Level 4: Restricted
- User passwords (hashed)
- OAuth secrets
- Database credentials
- Encryption keys

Handling:
- Level 1-2: Standard encryption
- Level 3: Encrypted + access controls
- Level 4: Encrypted + Vault + audit logging + access controls
```

## Network Security

### Edge Protection

#### DDoS Protection

```
Cloudflare DDoS Protection:
- Always-on protection
- Automatic mitigation
- Rate limiting per IP/API key
- Challenge page for suspicious traffic

Configuration:
- Layer 3/4: Automatic
- Layer 7: Custom rules
  └─→ Rate limit: 100 req/sec per IP
  └─→ Challenge after 1000 req/min
  └─→ Block after 5 failed challenges
```

#### WAF Rules

```
Cloudflare WAF:
- OWASP Core Rule Set
- Custom rules for AuthSome

Custom Rules:
1. Block SQL injection attempts
   └─→ Path contains: UNION, SELECT, DROP, etc.
   
2. Block XSS attempts
   └─→ Body/Query contains: <script>, javascript:, etc.
   
3. Rate limiting by API key
   └─→ Free: 10 req/sec
   └─→ Pro: 100 req/sec
   └─→ Enterprise: Custom
   
4. Geographic restrictions (if enabled)
   └─→ Allow only specified countries
```

### Internal Network Security

#### VPC Configuration

```
Private VPC:
- CIDR: 10.0.0.0/16
- Public subnets: 10.0.100.0/24, 10.0.101.0/24, 10.0.102.0/24
  └─→ NAT Gateway, Load Balancer only
- Private subnets: 10.0.1.0/24, 10.0.2.0/24, 10.0.3.0/24
  └─→ All application workloads
- Database subnets: 10.0.10.0/24, 10.0.11.0/24, 10.0.12.0/24
  └─→ PostgreSQL, Redis only

Security Groups:
- Load Balancer: Allow 443 from 0.0.0.0/0
- Control Plane: Allow 8080 from LB only
- Proxy: Allow 8081 from LB only
- Applications: Allow 8080 from Proxy only
- PostgreSQL: Allow 5432 from Applications only
- Redis: Allow 6379 from Applications only
```

#### Firewall Rules

```
Ingress:
- Allow HTTPS (443) from Cloudflare IPs only
- Allow SSH (22) from bastion host only (internal IP)
- Deny all other ingress

Egress:
- Allow HTTPS (443) to 0.0.0.0/0 (for OAuth, webhooks)
- Allow PostgreSQL (5432) within VPC
- Allow Redis (6379) within VPC
- Allow DNS (53) to VPC resolver
- Deny all other egress
```

## Access Control

### Control Plane Access

#### Dashboard Authentication

```go
// Multi-factor authentication required for dashboard
type AuthService struct {
    userRepo repository.UserRepository
    totpService *TOTPService
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*Session, error) {
    // 1. Verify email/password
    user, err := s.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }
    
    if !s.verifyPassword(user.PasswordHash, password) {
        return nil, ErrInvalidCredentials
    }
    
    // 2. Check if 2FA enabled
    if user.TOTPEnabled {
        // Require TOTP code (sent in second request)
        return nil, ErrTOTPRequired
    }
    
    // 3. Create session
    session := s.createSession(ctx, user)
    
    return session, nil
}

func (s *AuthService) VerifyTOTP(ctx context.Context, userID, code string) (*Session, error) {
    user, _ := s.userRepo.FindByID(ctx, userID)
    
    if !s.totpService.Verify(user.TOTPSecret, code) {
        return nil, ErrInvalidTOTPCode
    }
    
    return s.createSession(ctx, user), nil
}
```

#### Role-Based Access Control

```go
// Workspace roles
type Role string

const (
    RoleOwner     Role = "owner"     // Full access, can delete workspace
    RoleAdmin     Role = "admin"     // Manage apps, members, billing
    RoleDeveloper Role = "developer" // Manage apps only
    RoleBilling   Role = "billing"   // View billing only
    RoleViewer    Role = "viewer"    // Read-only access
)

// Permission checks
func (s *AuthorizationService) CanManageApplications(ctx context.Context, userID, workspaceID string) bool {
    member, _ := s.repo.GetMember(ctx, workspaceID, userID)
    
    return member.Role == RoleOwner || 
           member.Role == RoleAdmin || 
           member.Role == RoleDeveloper
}

func (s *AuthorizationService) CanManageBilling(ctx context.Context, userID, workspaceID string) bool {
    member, _ := s.repo.GetMember(ctx, workspaceID, userID)
    
    return member.Role == RoleOwner || 
           member.Role == RoleAdmin || 
           member.Role == RoleBilling
}
```

### Application API Keys

#### Key Management

```go
// API key format and security
type APIKeyService struct {
    repo   repository.APIKeyRepository
    crypto CryptoService
}

func (s *APIKeyService) Generate(ctx context.Context, appID string, keyType APIKeyType) (*APIKey, error) {
    // Generate cryptographically secure random key
    randomBytes := make([]byte, 32)
    if _, err := rand.Read(randomBytes); err != nil {
        return nil, err
    }
    
    prefix := s.getKeyPrefix(keyType)
    key := fmt.Sprintf("%s_%s_%s", prefix, s.getEnv(appID), base62.Encode(randomBytes))
    
    // Hash secret keys before storage
    var keyHash string
    if keyType == SecretKey {
        keyHash = s.crypto.HashAPIKey(key)
    }
    
    apiKey := &APIKey{
        ID:            generateID(),
        ApplicationID: appID,
        Type:          keyType,
        Key:           key,         // Shown once
        KeyHash:       keyHash,     // Stored
        Scopes:        []string{"*"}, // Full access by default
        CreatedAt:     time.Now(),
    }
    
    err := s.repo.Save(ctx, apiKey)
    if err != nil {
        return nil, err
    }
    
    return apiKey, nil
}

// Verify API key (constant-time)
func (s *APIKeyService) Verify(ctx context.Context, providedKey string) (*APIKey, error) {
    // Extract app ID from key
    appID := s.extractAppID(providedKey)
    
    // Get keys for this application
    keys, err := s.repo.ListByApplication(ctx, appID)
    if err != nil {
        return nil, err
    }
    
    // Constant-time comparison
    providedHash := s.crypto.HashAPIKey(providedKey)
    
    for _, key := range keys {
        if key.RevokedAt != nil {
            continue
        }
        
        if subtle.ConstantTimeCompare([]byte(providedHash), []byte(key.KeyHash)) == 1 {
            // Update last used
            s.repo.UpdateLastUsed(ctx, key.ID)
            return key, nil
        }
    }
    
    return nil, ErrInvalidAPIKey
}
```

#### Scoped Permissions

```go
// API keys can have limited scopes
type Scope string

const (
    ScopeUsersRead       Scope = "users:read"
    ScopeUsersWrite      Scope = "users:write"
    ScopeSessionsManage  Scope = "sessions:manage"
    ScopeOrgsRead        Scope = "organizations:read"
    ScopeOrgsWrite       Scope = "organizations:write"
    ScopeAuditRead       Scope = "audit:read"
)

// Check if API key has required scope
func (s *APIKeyService) HasScope(key *APIKey, required Scope) bool {
    // Wildcard grants all permissions
    for _, scope := range key.Scopes {
        if scope == "*" {
            return true
        }
        if Scope(scope) == required {
            return true
        }
    }
    return false
}

// Middleware to check scopes
func RequireScope(scope Scope) forge.MiddlewareFunc {
    return func(next forge.HandlerFunc) forge.HandlerFunc {
        return func(c *forge.Context) error {
            apiKey := c.Get("apiKey").(*APIKey)
            
            if !apiKeyService.HasScope(apiKey, scope) {
                return c.JSON(403, ErrorResponse{
                    Message: "Insufficient permissions",
                })
            }
            
            return next(c)
        }
    }
}
```

### Infrastructure Access

#### Bastion Host

```
SSH access only via bastion host:
- Bastion in public subnet
- SSH keys only (no passwords)
- MFA required (Duo/Google Authenticator)
- Session recording
- Time-limited access

Process:
1. Request access via PagerDuty/Vault
2. Temporary SSH certificate issued (valid 8 hours)
3. SSH to bastion
4. SSH to target server from bastion
5. All sessions logged to S3
```

#### Kubernetes RBAC

```yaml
# Limited RBAC for operators
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: authsome-operator
rules:
# Read-only access to most resources
- apiGroups: [""]
  resources: ["pods", "services", "configmaps"]
  verbs: ["get", "list", "watch"]

# Can restart pods
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "patch"]

# Can view logs
- apiGroups: [""]
  resources: ["pods/log"]
  verbs: ["get"]

# Cannot delete production resources
# Cannot access secrets
```

## Compliance

### SOC 2 Type II

```
Security Controls:
✓ Encryption at rest and in transit
✓ Access controls and MFA
✓ Audit logging
✓ Incident response plan
✓ Vendor management
✓ Change management
✓ Backup and recovery
✓ Security monitoring

Availability Controls:
✓ 99.9% uptime SLA
✓ Multi-AZ deployment
✓ Automated failover
✓ Load balancing
✓ Performance monitoring

Audit Schedule:
- Annual SOC 2 Type II audit
- Quarterly internal audits
- Continuous monitoring
```

### GDPR Compliance

```
Data Subject Rights:
✓ Right to access (API provided)
✓ Right to erasure (hard delete)
✓ Right to rectification (update API)
✓ Right to data portability (export API)
✓ Right to restrict processing (suspend account)

Technical Measures:
✓ Data encryption
✓ Access controls
✓ Audit logging
✓ Data minimization
✓ Privacy by design

Organizational Measures:
✓ DPA with customers
✓ Privacy policy
✓ Breach notification (<72 hours)
✓ DPIA for high-risk processing
✓ Staff training
```

### HIPAA (for healthcare customers)

```
Enterprise Tier Add-on:

Technical Safeguards:
✓ Access controls
✓ Audit controls
✓ Integrity controls
✓ Transmission security
✓ Encryption

Physical Safeguards:
✓ AWS/GCP SOC 2 compliance
✓ Data center security
✓ Workstation security

Administrative Safeguards:
✓ BAA with customers
✓ Security management
✓ Workforce security
✓ Information access management
✓ Security awareness training

Configuration:
- Extended audit retention (7 years)
- Additional encryption
- Dedicated instances available
- Region restrictions (US only)
```

## Incident Response

### Security Incident Response Plan

```
Phase 1: Detection (0-15 minutes)
- Automated alerts via PagerDuty
- On-call engineer notified
- Initial assessment

Phase 2: Containment (15-60 minutes)
- Isolate affected systems
- Preserve evidence
- Notify security team

Phase 3: Investigation (1-4 hours)
- Analyze logs and forensics
- Identify root cause
- Determine scope

Phase 4: Eradication (4-24 hours)
- Remove threat
- Patch vulnerabilities
- Reset compromised credentials

Phase 5: Recovery (24-48 hours)
- Restore services
- Monitor for recurrence
- Verify security

Phase 6: Post-Incident (1-2 weeks)
- Write incident report
- Customer notification (if required)
- Implement preventive measures
- Update runbooks
```

### Breach Notification

```go
// Automated breach detection and notification
type BreachDetector struct {
    alertService AlertService
    emailService EmailService
}

func (d *BreachDetector) DetectUnauthorizedAccess(ctx context.Context, event *SecurityEvent) error {
    // Criteria for breach:
    // - Unauthorized database access
    // - API key leak detected
    // - Mass data export
    // - Privilege escalation
    
    if d.isBreachEvent(event) {
        // 1. Alert security team immediately
        d.alertService.Alert(ctx, "SECURITY BREACH DETECTED", event)
        
        // 2. Log for forensics
        d.logBreach(ctx, event)
        
        // 3. Start incident response
        d.startIncidentResponse(ctx, event)
        
        // 4. Notify affected customers (within 72 hours for GDPR)
        d.scheduleCustomerNotification(ctx, event)
    }
    
    return nil
}
```

## Auditing

### Audit Logging

```go
// Comprehensive audit logging
type AuditLogger struct {
    repo repository.AuditRepository
}

type AuditEvent struct {
    ID          string                 `json:"id"`
    Timestamp   time.Time              `json:"timestamp"`
    Actor       string                 `json:"actor"`       // User or service
    ActorType   string                 `json:"actorType"`   // user, api_key, service
    Action      string                 `json:"action"`      // create, update, delete, access
    Resource    string                 `json:"resource"`    // workspace, application, user
    ResourceID  string                 `json:"resourceId"`
    Changes     map[string]interface{} `json:"changes"`     // Before/after
    IPAddress   string                 `json:"ipAddress"`
    UserAgent   string                 `json:"userAgent"`
    Status      string                 `json:"status"`      // success, failure
    Metadata    map[string]interface{} `json:"metadata"`
}

// Log all sensitive operations
func (l *AuditLogger) LogWorkspaceCreated(ctx context.Context, actor string, ws *Workspace) {
    l.repo.Save(ctx, &AuditEvent{
        ID:         generateID(),
        Timestamp:  time.Now(),
        Actor:      actor,
        ActorType:  "user",
        Action:     "create",
        Resource:   "workspace",
        ResourceID: ws.ID,
        Status:     "success",
    })
}

func (l *AuditLogger) LogAPIKeyAccess(ctx context.Context, apiKey *APIKey, request *http.Request) {
    l.repo.Save(ctx, &AuditEvent{
        ID:         generateID(),
        Timestamp:  time.Now(),
        Actor:      apiKey.ID,
        ActorType:  "api_key",
        Action:     "access",
        Resource:   "api",
        ResourceID: apiKey.ApplicationID,
        IPAddress:  getClientIP(request),
        UserAgent:  request.UserAgent(),
        Status:     "success",
    })
}
```

### Retention Policy

```
Audit Logs:
- Free plan: 7 days
- Pro plan: 30 days
- Enterprise: 90 days - 7 years (configurable)

Security Logs:
- All plans: 1 year minimum
- Extended retention available

Backup Retention:
- Daily backups: 30 days
- Weekly backups: 90 days
- Monthly backups: 1 year
```

### Monitoring and Alerting

```yaml
# Critical security alerts
alerts:
  - name: UnauthorizedAPIAccess
    condition: api_errors{code="401"} > 100 per 5m
    severity: high
    action: PagerDuty + Email
    
  - name: SuspiciousDataExport
    condition: data_export_size > 10GB per 1h
    severity: critical
    action: PagerDuty + Block
    
  - name: FailedLoginAttempts
    condition: failed_logins{user=~".+"} > 10 per 5m
    severity: medium
    action: Email + Temporary block
    
  - name: PrivilegeEscalation
    condition: role_change{from!="owner",to="owner"}
    severity: critical
    action: PagerDuty + Email + Audit
    
  - name: DatabaseAccessAnomaly
    condition: database_queries{type="SELECT"} > 1000 per 1m
    severity: high
    action: PagerDuty + Investigate
```

## Security Best Practices for Customers

### API Key Security

```
✓ Store API keys in environment variables
✓ Never commit keys to git
✓ Rotate keys every 90 days
✓ Use separate keys for dev/staging/prod
✓ Revoke keys immediately if compromised
✓ Use scoped keys (limit permissions)

✗ Don't hardcode keys in application
✗ Don't expose secret keys in frontend
✗ Don't share keys between environments
✗ Don't log API keys
```

### Application Security

```
✓ Enable HTTPS only
✓ Implement rate limiting
✓ Validate all inputs
✓ Use prepared statements (prevent SQL injection)
✓ Implement CSRF protection
✓ Set secure cookie flags
✓ Enable Content Security Policy
✓ Keep dependencies updated

✗ Don't disable TLS verification
✗ Don't trust user input
✗ Don't store passwords in plaintext
✗ Don't use weak hashing algorithms
```

---

**Security is a shared responsibility. Contact security@authsome.cloud for questions or to report vulnerabilities.**

