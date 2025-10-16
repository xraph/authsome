# 🔒 AuthSome Security Infrastructure

> Enterprise-grade security auditing for authentication systems

## Overview

AuthSome includes comprehensive security infrastructure with **6 automated scanners**, **GitHub Actions integration**, and **pre-commit hooks** to ensure the highest security standards for authentication systems.

## 🚀 Quick Start

```bash
# 1. Install security tools (one-time setup)
make security-install-tools

# 2. Install trivy manually
brew install trivy  # macOS
# or apt-get install trivy  # Linux

# 3. Install pre-commit hooks
pip install pre-commit
pre-commit install

# 4. Run your first security audit
make security-audit

# 5. View results
cat .security-reports/REPORT-*.md
```

## 📋 Daily Workflow

```bash
# Before committing (automatic with hooks)
git commit -m "feat: new feature"
# → Pre-commit hooks run automatically

# Manual quick check
make security-pre-commit

# Before pushing
make pre-push

# Before releasing
make release-prep
```

## 🛠️ Security Tools

| Tool | Purpose | Command |
|------|---------|---------|
| **gosec** | Static security analysis | `make security-gosec` |
| **govulncheck** | Vulnerability scanning | `make security-vuln` |
| **trivy** | Comprehensive scanning | `make security-vuln` |
| **gitleaks** | Secret detection | `make security-secrets` |
| **go-licenses** | License compliance | `make security-licenses` |
| **CodeQL** | Semantic analysis | (GitHub Actions) |

## 📊 What Gets Detected

### Security Issues
- ✅ SQL Injection
- ✅ Hardcoded Credentials
- ✅ Weak Cryptography (MD5, SHA1)
- ✅ Weak Random Number Generation
- ✅ Command Injection
- ✅ Path Traversal
- ✅ Known CVEs in Dependencies

### Secrets
- ✅ API Keys (AWS, GitHub, Stripe, etc.)
- ✅ OAuth Tokens
- ✅ JWT Tokens
- ✅ Database Credentials
- ✅ Private Keys

### Compliance
- ✅ OWASP Top 10 (100% coverage)
- ✅ SOC 2 Requirements
- ✅ GDPR Compliance
- ✅ License Compliance

## 🎯 Makefile Targets

### Complete Audit
```bash
make security-audit          # Full security audit (all scanners)
make security-ci             # Fast CI checks
make security-pre-commit     # Quick pre-commit check
```

### Individual Scanners
```bash
make security-gosec          # Static application security testing
make security-vuln           # Vulnerability scanning
make security-deps           # Dependency audit
make security-secrets        # Secret detection
make security-sbom           # Software Bill of Materials
make security-licenses       # License compliance check
```

### Utilities
```bash
make security-install-tools  # Install all security tools
make security-report         # Generate summary report
make security-clean          # Remove security reports
make clean-all              # Remove all artifacts
```

## 🤖 Automated Security

### GitHub Actions
- ✅ Runs on every push and PR
- ✅ Daily scheduled scans (2 AM UTC)
- ✅ Weekly CodeQL analysis (Monday 3 AM UTC)
- ✅ Results in GitHub Security tab

### Pre-commit Hooks
- ✅ Go formatting and linting
- ✅ Secret scanning (gitleaks)
- ✅ YAML/JSON validation
- ✅ Quick security check
- ✅ Short test suite

### Dependabot
- ✅ Weekly dependency updates
- ✅ Security patches prioritized
- ✅ Auto-labeled PRs

## 📖 Documentation

| Document | Purpose |
|----------|---------|
| **SECURITY.md** | Main security policy and reporting |
| **docs/SECURITY_AUDIT_GUIDE.md** | Comprehensive 500+ line guide |
| **SECURITY_IMPLEMENTATION.md** | Technical implementation details |
| **.github/SECURITY_QUICK_REFERENCE.md** | One-page quick reference |

## 🚨 Emergency Response

### If Secret Exposed
```bash
# 1. Immediately rotate the secret
export NEW_SECRET=$(make generate-secret)

# 2. Scan for other secrets
make security-secrets

# 3. Remove from git history if needed
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch path/to/file' \
  --prune-empty --tag-name-filter cat -- --all
```

### If Vulnerability Found
```bash
# 1. Scan for vulnerabilities
make security-vuln

# 2. Update affected package
go get package@latest
go mod tidy

# 3. Test
make test

# 4. Deploy hotfix if critical
git tag v1.2.3-hotfix
```

## 📈 Security Levels

| Severity | Response Time | Action |
|----------|---------------|--------|
| **Critical** | < 24 hours | Immediate hotfix |
| **High** | < 7 days | Next patch release |
| **Medium** | < 30 days | Next minor release |
| **Low** | Next release | Add to backlog |

## 🎓 Common Fixes

### SQL Injection
```go
❌ query := "SELECT * FROM users WHERE id = " + id
✅ db.NewSelect().Model(&user).Where("id = ?", id)
```

### Hardcoded Secret
```go
❌ const apiKey = "sk_live_123456"
✅ apiKey := os.Getenv("API_KEY")
```

### Weak Random
```go
❌ import "math/rand"; token := rand.Int()
✅ import "crypto/rand"; rand.Read(token)
```

### Weak Crypto
```go
❌ hash := md5.Sum(password)
✅ hash, _ := bcrypt.GenerateFromPassword(password, 12)
```

## 📞 Support

- **Security Issues**: security@authsome.dev (private reporting)
- **General Support**: support@authsome.dev
- **GitHub Security**: https://github.com/xraph/authsome/security

## 🔗 Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Database](https://cwe.mitre.org/)
- [Go Security](https://go.dev/security/)
- [gosec Rules](https://github.com/securego/gosec#available-rules)

---

**For detailed information, see [docs/SECURITY_AUDIT_GUIDE.md](docs/SECURITY_AUDIT_GUIDE.md)**

