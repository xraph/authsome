# 🔒 Security Quick Reference

> One-page reference for AuthSome security tools and workflows

## Quick Commands

```bash
# Complete security audit
make security-audit

# Pre-commit check (fast)
make security-pre-commit

# CI/CD checks
make security-ci

# Install all tools
make security-install-tools

# Individual scans
make security-gosec          # Static analysis
make security-vuln           # Vulnerabilities
make security-secrets        # Secret detection
make security-licenses       # License check
```

## Emergency Response

### Secret Exposed
```bash
# 1. Rotate immediately
export NEW_SECRET=$(make generate-secret)

# 2. Scan for secrets
make security-secrets

# 3. Check git history
gitleaks detect --verbose

# 4. Remove from history (if needed)
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch path/to/file' \
  --prune-empty --tag-name-filter cat -- --all
```

### Critical Vulnerability
```bash
# 1. Scan for vulnerabilities
make security-vuln

# 2. Check specific package
govulncheck -pkg github.com/example/package ./...

# 3. Update dependency
go get github.com/example/package@latest
go mod tidy

# 4. Test
make test

# 5. Deploy hotfix
git tag v1.2.3-hotfix
```

## CI/CD Status

| Check | Runs On | Action |
|-------|---------|--------|
| gosec | Every push, PR | Fix immediately |
| Secrets | Every push, PR | Rotate & remove |
| Vulnerabilities | Every push, PR, Daily | Patch ASAP |
| CodeQL | Weekly (Mon 3AM) | Review findings |
| Dependabot | Weekly (Mon) | Review & merge |

## Security Levels

| Severity | Response Time | Action |
|----------|---------------|--------|
| **Critical** | < 24 hours | Hotfix release |
| **High** | < 7 days | Next patch |
| **Medium** | < 30 days | Next minor |
| **Low** | Next release | Backlog |

## Common Issues

### SQL Injection
```go
❌ db.Exec("SELECT * FROM users WHERE id = " + id)
✅ db.NewSelect().Model(&user).Where("id = ?", id)
```

### Hardcoded Secret
```go
❌ const apiKey = "sk_live_123"
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

## Pre-commit Hooks

```bash
# Install once
pip install pre-commit
pre-commit install

# Runs automatically on commit:
✓ Go formatting
✓ Secret scanning
✓ YAML validation
✓ Quick tests
```

## Useful Aliases

Add to `~/.bashrc` or `~/.zshrc`:

```bash
alias security-check='make security-pre-commit'
alias security-full='make security-audit'
alias security-secrets='make security-secrets'
alias security-vuln='make security-vuln'
```

## Reports Location

```
.security-reports/
├── gosec-*.{json,txt,sarif}
├── govulncheck-*.{json,txt}
├── trivy-*.{json,txt,sarif}
├── gitleaks-*.{json,sarif}
├── sbom-*.json
├── licenses-*.txt
└── REPORT-*.md
```

## Documentation

- **SECURITY.md** - Security policy
- **docs/SECURITY_AUDIT_GUIDE.md** - Full guide
- **SECURITY_IMPLEMENTATION.md** - Implementation details

## Contact

- Security Issues: security@authsome.dev
- GitHub Security: https://github.com/xraph/authsome/security

---

**Keep this card handy during development!**

