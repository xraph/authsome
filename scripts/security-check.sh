#!/bin/bash
# Quick security check script for AuthSome
# Run before committing code

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔒 AuthSome Security Quick Check"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

ERRORS=0

# Check 1: Secret scanning
echo "→ Scanning for hardcoded secrets..."
if command -v gitleaks >/dev/null 2>&1; then
    if gitleaks detect --config .gitleaks.toml --no-git --quiet 2>/dev/null; then
        echo -e "${GREEN}✓ No secrets detected${NC}"
    else
        echo -e "${RED}✗ SECRETS DETECTED! Review and remove immediately.${NC}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${YELLOW}⚠ gitleaks not installed, skipping${NC}"
fi
echo ""

# Check 2: Go formatting
echo "→ Checking Go code formatting..."
UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v "vendor/" | grep -v "clients/go/" | grep -v "clients/typescript/" | grep -v "clients/rust/" || true)
if [ -z "$UNFORMATTED" ]; then
    echo -e "${GREEN}✓ Code is formatted${NC}"
else
    echo -e "${RED}✗ Unformatted files:${NC}"
    echo "$UNFORMATTED"
    echo "Run: go fmt ./..."
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 3: Go vet
echo "→ Running go vet..."
if go vet ./... 2>&1 | grep -v "vendor/" | grep -v "clients/go/" | grep -v "clients/typescript/" | grep -v "clients/rust/" >/dev/null; then
    echo -e "${RED}✗ go vet found issues${NC}"
    go vet ./... 2>&1 | grep -v "vendor/" | grep -v "clients/go/" | grep -v "clients/typescript/" | grep -v "clients/rust/"
    ERRORS=$((ERRORS + 1))
else
    echo -e "${GREEN}✓ No issues found${NC}"
fi
echo ""

# Check 4: Common security patterns
echo "→ Checking for common security anti-patterns..."

# Check for math/rand in security-critical code
WEAK_RANDOM=$(grep -r "math/rand" --include="*.go" --exclude-dir={vendor,clients} . || true)
if [ -n "$WEAK_RANDOM" ]; then
    echo -e "${YELLOW}⚠ Found math/rand usage (use crypto/rand for security):${NC}"
    echo "$WEAK_RANDOM"
fi

# Check for hardcoded passwords/secrets patterns
HARDCODED=$(grep -ri "password\s*=\s*\"" --include="*.go" --exclude-dir={vendor,clients,examples} . || true)
if [ -n "$HARDCODED" ]; then
    echo -e "${RED}✗ Potential hardcoded credentials:${NC}"
    echo "$HARDCODED"
    ERRORS=$((ERRORS + 1))
fi

# Check for SQL string concatenation
SQL_CONCAT=$(grep -r "SELECT.*+.*" --include="*.go" --exclude-dir={vendor,clients} . || true)
if [ -n "$SQL_CONCAT" ]; then
    echo -e "${RED}✗ Potential SQL injection (string concatenation):${NC}"
    echo "$SQL_CONCAT"
    ERRORS=$((ERRORS + 1))
fi

if [ -z "$WEAK_RANDOM" ] && [ -z "$HARDCODED" ] && [ -z "$SQL_CONCAT" ]; then
    echo -e "${GREEN}✓ No obvious anti-patterns found${NC}"
fi
echo ""

# Check 5: Test coverage on modified files
if [ -n "$1" ]; then
    echo "→ Running tests..."
    if go test -short ./... >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Tests passed${NC}"
    else
        echo -e "${RED}✗ Tests failed${NC}"
        ERRORS=$((ERRORS + 1))
    fi
    echo ""
fi

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✓ Security quick check passed!${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 0
else
    echo -e "${RED}✗ Security check failed with $ERRORS error(s)${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Fix the issues above before committing."
    echo "For detailed security audit, run: make security-audit"
    exit 1
fi

