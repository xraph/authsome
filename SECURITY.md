# Security Policy

## Overview

AuthSome is an authentication framework handling sensitive user credentials and session data. Security is our highest priority.

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x     | :white_check_mark: |

## Reporting a Vulnerability

**DO NOT** open a public GitHub issue for security vulnerabilities.

### Reporting Process

1. **Email**: Send details to security@authsome.dev (or your security contact)
2. **Include**:
   - Description of the vulnerability
   - Steps to reproduce
   - Affected versions
   - Potential impact
   - Suggested fix (if any)
3. **Response Time**: We aim to respond within 48 hours
4. **Disclosure**: We follow coordinated disclosure (90 days)

### Security Vulnerability Severity

We use CVSS 3.1 scoring:

- **Critical** (9.0-10.0): Immediate patch required
- **High** (7.0-8.9): Patch within 7 days
- **Medium** (4.0-6.9): Patch within 30 days
- **Low** (0.1-3.9): Patch in next minor release

## Security Measures

### Authentication & Session Management

- **Password Hashing**: bcrypt (cost factor 12)
- **Session Tokens**: Cryptographically secure random (32 bytes)
- **Session Storage**: Cookie (HttpOnly, Secure, SameSite=Lax) + Redis cache
- **Token Expiry**: Configurable (default 7 days)
- **Refresh Tokens**: Rotation on use with family tracking

### Rate Limiting

- **Login Attempts**: 5 per 15 minutes per IP
- **Registration**: 3 per hour per IP
- **Password Reset**: 3 per hour per email
- **2FA Attempts**: 5 per 15 minutes per user
- **API Endpoints**: 100 requests per minute per IP

### Data Protection

- **Encryption at Rest**: Database-level encryption recommended
- **Encryption in Transit**: TLS 1.3 required in production
- **PII Handling**: Minimal collection, encrypted storage
- **Audit Logging**: All authentication events logged
- **Secret Management**: Environment variables, no hardcoded secrets

### Input Validation

- **SQL Injection**: Parameterized queries (Bun ORM)
- **XSS Prevention**: Output encoding, CSP headers
- **CSRF Protection**: SameSite cookies, CSRF tokens
- **Path Traversal**: Strict path validation
- **Command Injection**: No shell execution with user input

### Dependency Management

- **Vulnerability Scanning**: Daily automated scans
- **Dependency Updates**: Weekly review and updates
- **Supply Chain**: SBOM generation, license compliance
- **Pinned Versions**: Go modules with checksums

### Security Testing

- **SAST**: gosec, golangci-lint with security linters
- **Dependency Scanning**: govulncheck, trivy, nancy
- **Secret Detection**: gitleaks pre-commit hooks
- **Penetration Testing**: Annual third-party assessment
- **Security Reviews**: Mandatory for all PRs

## Security Best Practices for Users

### Deployment

```yaml
# Production configuration requirements
server:
  https_only: true
  tls_min_version: "1.3"
  
session:
  secure_cookie: true
  same_site: "lax"
  http_only: true
  
rate_limit:
  enabled: true
  store: "redis"  # Distributed rate limiting
  
security:
  csrf_protection: true
  cors_strict: true
```

### Environment Variables

Never commit these to version control:

```bash
# Required secrets
AUTH_SECRET=<generate with: make generate-secret>
DATABASE_URL=postgresql://...
REDIS_URL=redis://...

# OAuth provider secrets (if used)
GOOGLE_CLIENT_SECRET=...
GITHUB_CLIENT_SECRET=...

# Email/SMS provider keys
SENDGRID_API_KEY=...
TWILIO_AUTH_TOKEN=...

# JWT signing keys
JWT_PRIVATE_KEY_PATH=/path/to/private.pem
JWT_PUBLIC_KEY_PATH=/path/to/public.pem
```

### Key Generation

```bash
# Generate secure secrets
make generate-secret

# Generate RSA keys for JWT/OIDC
make generate-keys

# Rotate secrets regularly (quarterly recommended)
```

### Database Security

```sql
-- Use separate database users with minimal privileges
CREATE USER authsome_app WITH PASSWORD '...';
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO authsome_app;

-- Enable SSL connections
ALTER SYSTEM SET ssl = on;

-- Enable audit logging
ALTER SYSTEM SET log_statement = 'mod';
```

### Network Security

- Deploy behind reverse proxy (Nginx, Caddy)
- Use Web Application Firewall (WAF)
- Enable DDoS protection (Cloudflare, AWS Shield)
- Restrict database access to application servers only
- Use VPC/private networks

### Monitoring & Alerting

Monitor these security events:

- Failed login attempts (5+ in 5 minutes)
- Password reset requests (3+ in 1 hour)
- New user registrations (spike detection)
- API rate limit hits
- Database connection failures
- Unusual geographic access patterns

## Security Scanning Tools

We use the following tools (see Makefile):

```bash
# Run all security checks
make security-audit

# Individual checks
make security-gosec          # Static application security
make security-vuln           # Vulnerability scanning
make security-secrets        # Secret detection
make security-deps           # Dependency audit
make security-licenses       # License compliance
```

## Known Security Considerations

### Session Management

- Sessions are stored in cookies AND Redis for distributed systems
- Session fixation: New session ID on authentication
- Session hijacking: Bind sessions to User-Agent + IP (optional)
- Concurrent sessions: Configurable limit per user

### OAuth/OIDC

- State parameter: CSRF protection for OAuth flows
- PKCE: Required for public clients
- Token validation: Signature, expiry, audience, issuer
- Provider trust: Only verified providers allowed

### Multi-tenancy

- Organization isolation: Row-level security
- Data leakage: Strict org_id validation
- Cross-tenant access: Explicit role checks
- Configuration isolation: Per-org config overrides

### Cryptography

- Random generation: crypto/rand (not math/rand)
- Hash algorithms: bcrypt, SHA-256 (no MD5, SHA-1)
- Key derivation: PBKDF2 or Argon2
- Token comparison: constant-time comparison

## Compliance

### GDPR

- Right to erasure: User deletion endpoint
- Data portability: Export user data
- Consent management: Explicit opt-in
- Data minimization: Collect only necessary fields

### SOC 2

- Access controls: RBAC with audit logging
- Encryption: TLS + database encryption
- Monitoring: Real-time security alerts
- Incident response: Documented procedures

### OWASP Top 10

We address all OWASP Top 10 vulnerabilities:

1. ✅ Broken Access Control → RBAC with policy engine
2. ✅ Cryptographic Failures → Strong encryption, secure key management
3. ✅ Injection → Parameterized queries, input validation
4. ✅ Insecure Design → Security-first architecture
5. ✅ Security Misconfiguration → Secure defaults, config validation
6. ✅ Vulnerable Components → Automated scanning, updates
7. ✅ Authentication Failures → Enterprise-grade auth patterns
8. ✅ Software & Data Integrity → SBOM, checksums, signatures
9. ✅ Security Logging → Comprehensive audit trails
10. ✅ SSRF → URL validation, allowlist approach

## Security Roadmap

### Planned Enhancements

- [ ] Hardware security module (HSM) integration
- [ ] Biometric authentication (WebAuthn enhancement)
- [ ] Behavioral analytics for anomaly detection
- [ ] Advanced threat protection (ATP) integration
- [ ] Zero-trust architecture patterns
- [ ] Secrets rotation automation
- [ ] Security chaos engineering tests

## Contact

- Security Issues: security@authsome.dev
- General Questions: support@authsome.dev
- GitHub: https://github.com/xraph/authsome

## Acknowledgments

We appreciate responsible disclosure from security researchers. Hall of fame coming soon.

---

**Last Updated**: 2024-10-16
**Security Team**: security@authsome.dev

