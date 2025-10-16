# Security Audit Guide for AuthSome

## Overview

This guide provides comprehensive instructions for running security audits on AuthSome, an authentication framework handling sensitive credentials. Given the critical nature of authentication systems, regular security audits are mandatory.

## Quick Start

```bash
# Install all security tools
make security-install-tools

# Run complete security audit
make security-audit

# Quick pre-commit check
make security-pre-commit

# CI/CD security checks
make security-ci
```

## Security Scanning Tools

### 1. gosec - Static Application Security Testing (SAST)

**Purpose**: Detects common security issues in Go code.

```bash
# Run gosec
make security-gosec

# Direct invocation
gosec -fmt=json -out=gosec-results.json ./...
```

**What it detects**:
- SQL injection vulnerabilities
- Hardcoded credentials
- Weak cryptography (MD5, DES)
- Unsafe random number generation (math/rand)
- Path traversal vulnerabilities
- Command injection
- Insecure TLS configurations
- Unhandled errors in security-critical code

**Common issues in auth systems**:
- G101: Hardcoded credentials
- G104: Unhandled errors (especially in crypto operations)
- G201/G202: SQL query string building (SQL injection)
- G401: Use of weak crypto (MD5, SHA1)
- G404: Weak random number generator

### 2. govulncheck - Vulnerability Scanning

**Purpose**: Checks for known vulnerabilities in dependencies.

```bash
# Run govulncheck
make security-vuln

# Direct invocation
govulncheck ./...
```

**What it detects**:
- Known CVEs in Go standard library
- Vulnerabilities in third-party dependencies
- Only reports vulnerabilities in code you actually use

**Response strategy**:
- **Critical**: Patch immediately, release hotfix
- **High**: Patch within 7 days
- **Medium**: Patch within 30 days
- **Low**: Patch in next release

### 3. trivy - Comprehensive Vulnerability Scanner

**Purpose**: Multi-purpose scanner for vulnerabilities, misconfigurations, and secrets.

```bash
# Install trivy
brew install trivy  # macOS
# or apt-get install trivy  # Debian/Ubuntu

# Run trivy
trivy fs --severity CRITICAL,HIGH .
```

**What it detects**:
- OS package vulnerabilities
- Language-specific package vulnerabilities
- IaC misconfigurations
- Exposed secrets
- License compliance issues

### 4. gitleaks - Secret Detection

**Purpose**: Prevents credential leaks in code and git history.

```bash
# Run gitleaks
make security-secrets

# Scan git history
gitleaks detect --config .gitleaks.toml --verbose

# Scan files only (no git)
gitleaks detect --config .gitleaks.toml --no-git
```

**What it detects**:
- API keys (AWS, GitHub, Stripe, etc.)
- OAuth tokens
- JWT tokens
- Database connection strings
- Private keys
- Passwords in code/config

**If secrets are detected**:
1. **Immediately** rotate the compromised secret
2. Remove from git history: `git filter-branch` or BFG Repo-Cleaner
3. Review access logs for unauthorized use
4. Document in security incident log
5. Update .gitleaks.toml to prevent recurrence

### 5. License Compliance

**Purpose**: Ensure dependencies use acceptable licenses.

```bash
# Check licenses
make security-licenses

# Generate SBOM
make security-sbom
```

**Disallowed license types**:
- AGPL (copyleft, requires source disclosure)
- GPL (copyleft)
- Proprietary/Commercial licenses without agreement

**Acceptable licenses**:
- MIT, BSD, Apache 2.0
- ISC, MPL 2.0
- CC0, Unlicense

## Automated Security Scanning

### GitHub Actions

Security scans run automatically on:
- Every push to `main` or `develop`
- Every pull request
- Daily at 2 AM UTC (scheduled scan)
- Manual trigger via Actions tab

**Workflows**:
- `.github/workflows/security.yml` - Main security suite
- `.github/workflows/codeql.yml` - CodeQL analysis

**SARIF upload**: Results appear in GitHub Security tab.

### Pre-commit Hooks

Install pre-commit hooks to catch issues before commit:

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

**Hooks enabled**:
- Go formatting (gofmt, goimports)
- Secret scanning (gitleaks)
- YAML/JSON validation
- Quick security check
- Short tests

### Dependabot

Automatic dependency updates configured in `.github/dependabot.yml`:
- Weekly Go module updates
- Security patches applied immediately
- GitHub Actions version updates

## Security Audit Workflow

### Monthly Audit (Required)

```bash
# 1. Full security audit
make security-audit

# 2. Review reports
ls -la .security-reports/

# 3. Triage findings
# - Create GitHub issues for vulnerabilities
# - Assign severity labels
# - Set fix deadlines based on severity

# 4. Document in security log
# - Update SECURITY.md
# - Note any accepted risks
# - Record remediation actions
```

### Pre-release Audit (Required)

```bash
# Run complete release preparation with security
make release-prep

# This runs:
# - All tests
# - Client generation
# - Validation
# - Full security audit

# Review checklist:
# ✓ No critical/high vulnerabilities
# ✓ No exposed secrets
# ✓ All dependencies up to date
# ✓ License compliance verified
# ✓ SBOM generated
# ✓ Security documentation updated
```

### Incident Response Audit

If a security incident occurs:

```bash
# 1. Immediate full scan
make security-audit

# 2. Check for similar vulnerabilities
gosec -include=G<rule-number> ./...

# 3. Review audit logs
# Check who accessed what and when

# 4. Generate incident report
# Use template in SECURITY.md
```

## Interpreting Results

### gosec Output

```json
{
  "Issues": [
    {
      "severity": "HIGH",
      "confidence": "HIGH",
      "rule_id": "G401",
      "details": "Use of weak cryptographic primitive",
      "file": "/path/to/file.go",
      "line": "42"
    }
  ]
}
```

**Action**: Fix HIGH severity issues immediately. Review MEDIUM/LOW based on context.

### govulncheck Output

```
Vulnerability #1: GO-2023-1234
    Use of vulnerable package
    
  More info: https://pkg.go.dev/vuln/GO-2023-1234
  
  Module: github.com/example/vulnerable
  Found in: example.com/myapp@v1.2.3
  Fixed in: example.com/myapp@v1.2.4
  
  Call stacks:
    main.go:42:10: vulnerable.Function
```

**Action**: Upgrade to fixed version: `go get github.com/example/vulnerable@v1.2.4`

### trivy Output

```
Total: 5 (CRITICAL: 2, HIGH: 3)

┌─────────────┬─────────────┬──────────┬─────────────────┐
│  Library    │ Vulnerability│ Severity │ Installed Ver.  │
├─────────────┼─────────────┼──────────┼─────────────────┤
│ go-jose     │ CVE-2023-123│ CRITICAL │ v2.5.0          │
└─────────────┴─────────────┴──────────┴─────────────────┘
```

**Action**: Update vulnerable libraries immediately.

## Common Vulnerabilities in Auth Systems

### 1. SQL Injection

**Detection**: gosec G201/G202

```go
// VULNERABLE
query := "SELECT * FROM users WHERE email = '" + email + "'"

// SECURE
query := db.NewSelect().Model(&user).Where("email = ?", email)
```

### 2. Hardcoded Secrets

**Detection**: gosec G101, gitleaks

```go
// VULNERABLE
const apiKey = "sk_live_1234567890abcdef"

// SECURE
apiKey := os.Getenv("API_KEY")
```

### 3. Weak Random

**Detection**: gosec G404

```go
// VULNERABLE
import "math/rand"
token := rand.Int()

// SECURE
import "crypto/rand"
token := make([]byte, 32)
rand.Read(token)
```

### 4. Weak Crypto

**Detection**: gosec G401

```go
// VULNERABLE
import "crypto/md5"
hash := md5.Sum(password)

// SECURE
import "golang.org/x/crypto/bcrypt"
hash, _ := bcrypt.GenerateFromPassword(password, 12)
```

### 5. Timing Attacks

**Detection**: Manual review

```go
// VULNERABLE
if userToken == dbToken { ... }

// SECURE
import "crypto/subtle"
if subtle.ConstantTimeCompare(userToken, dbToken) == 1 { ... }
```

## Security Metrics

Track these metrics monthly:

- **Mean Time to Remediate (MTTR)**:
  - Critical: < 24 hours
  - High: < 7 days
  - Medium: < 30 days

- **Vulnerability Density**: Vulnerabilities per 1000 lines of code
- **Secret Exposure Rate**: Secrets detected per commit
- **Dependency Freshness**: % of dependencies on latest version
- **Test Coverage**: Aim for 80%+ on security-critical code

## Best Practices

### Development

1. **Never commit secrets** - Use environment variables
2. **Review dependencies** before adding
3. **Run pre-commit hooks** on every commit
4. **Use parameterized queries** for all SQL
5. **Constant-time comparison** for tokens/passwords
6. **Strong crypto only** - bcrypt/argon2, no MD5/SHA1
7. **Validate all inputs** at boundaries
8. **Log security events** (login, password change, etc.)

### Review

1. **Peer review** all security-related code
2. **Threat modeling** for new features
3. **Security checklist** in PR template
4. **Automated scanning** in CI/CD
5. **Quarterly penetration testing**

### Operations

1. **TLS everywhere** - HTTPS only in production
2. **Rate limiting** on all endpoints
3. **Monitor security logs** for anomalies
4. **Rotate secrets** quarterly
5. **Keep dependencies updated** weekly
6. **Backup encryption keys** securely

## Compliance

### OWASP Top 10

- [x] A01: Broken Access Control → RBAC with policy engine
- [x] A02: Cryptographic Failures → Strong crypto, key management
- [x] A03: Injection → Parameterized queries, input validation
- [x] A04: Insecure Design → Security-first architecture
- [x] A05: Security Misconfiguration → Secure defaults
- [x] A06: Vulnerable Components → Automated scanning
- [x] A07: Authentication Failures → Enterprise auth patterns
- [x] A08: Data Integrity → SBOM, checksums
- [x] A09: Security Logging → Comprehensive audit
- [x] A10: SSRF → URL validation, allowlists

### SOC 2

- Access controls: RBAC with audit logging
- Encryption: TLS 1.3, AES-256
- Monitoring: Real-time security alerts
- Incident response: Documented in SECURITY.md

### GDPR

- Data minimization: Collect only necessary fields
- Right to erasure: User deletion endpoint
- Data portability: Export user data
- Consent: Explicit opt-in

## Tools Installation

### macOS

```bash
# Homebrew
brew install golangci-lint trivy

# Go tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/gitleaks/gitleaks/v8@latest
go install github.com/google/go-licenses@latest

# Or use make
make security-install-tools
```

### Linux (Ubuntu/Debian)

```bash
# APT
sudo apt-get update
sudo apt-get install -y trivy

# Go tools
make security-install-tools
```

### CI/CD (GitHub Actions)

Already configured in `.github/workflows/security.yml`.

## Resources

- **OWASP**: https://owasp.org/
- **CWE**: https://cwe.mitre.org/
- **Go Security**: https://go.dev/security/
- **gosec**: https://github.com/securego/gosec
- **govulncheck**: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
- **trivy**: https://github.com/aquasecurity/trivy
- **gitleaks**: https://github.com/gitleaks/gitleaks

## Contact

- Security issues: security@authsome.dev
- General support: support@authsome.dev
- GitHub: https://github.com/xraph/authsome

---

**Last Updated**: 2024-10-16  
**Next Review**: Monthly  
**Owner**: Security Team

