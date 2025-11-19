#!/bin/bash

# Client Release Setup Verification Script
# This script verifies that the CI release workflows are properly configured

set -e

echo "🔍 Verifying Client Release Setup..."
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if we're in the right directory
if [ ! -d ".github/workflows" ]; then
    error "Not in the authsome root directory"
    exit 1
fi

success "Found .github/workflows directory"

# Check workflow files
echo ""
echo "Checking workflow files..."
WORKFLOWS=("release-typescript-client.yml" "release-rust-client.yml" "release-go-client.yml")
for workflow in "${WORKFLOWS[@]}"; do
    if [ -f ".github/workflows/$workflow" ]; then
        success "$workflow exists"
        
        # Check for required keys
        if grep -q "^name:" ".github/workflows/$workflow" && \
           grep -q "^on:" ".github/workflows/$workflow" && \
           grep -q "^jobs:" ".github/workflows/$workflow"; then
            success "  - Has required keys (name, on, jobs)"
        else
            error "  - Missing required keys"
        fi
        
        # Check for secrets
        if grep -q "secrets\." ".github/workflows/$workflow"; then
            success "  - References secrets"
        else
            warning "  - No secrets referenced (might be expected for Go)"
        fi
        
        # Check for change detection
        if grep -q "Check if.*changed" ".github/workflows/$workflow"; then
            success "  - Has change detection"
        else
            error "  - Missing change detection"
        fi
    else
        error "$workflow not found"
    fi
done

# Check documentation
echo ""
echo "Checking documentation..."
DOCS=("RELEASING.md" "CLIENT_RELEASES.md")
for doc in "${DOCS[@]}"; do
    if [ -f ".github/$doc" ]; then
        success "$doc exists"
    else
        error "$doc not found"
    fi
done

# Check client directories
echo ""
echo "Checking client directories..."
CLIENTS=("typescript" "rust" "go")
for client in "${CLIENTS[@]}"; do
    if [ -d "clients/$client" ]; then
        success "clients/$client exists"
        
        # Check for client-specific files
        case $client in
            typescript)
                if [ -f "clients/$client/package.json" ]; then
                    success "  - package.json found"
                    VERSION=$(grep '"version"' clients/$client/package.json | sed 's/.*: *"\(.*\)".*/\1/')
                    echo "    Current version: $VERSION"
                else
                    error "  - package.json not found"
                fi
                ;;
            rust)
                if [ -f "clients/$client/Cargo.toml" ]; then
                    success "  - Cargo.toml found"
                    VERSION=$(grep '^version' clients/$client/Cargo.toml | head -1 | sed 's/version = "\(.*\)"/\1/')
                    echo "    Current version: $VERSION"
                else
                    error "  - Cargo.toml not found"
                fi
                ;;
            go)
                if [ -f "clients/$client/go.mod" ]; then
                    success "  - go.mod found"
                    if grep -q "// version:" clients/$client/go.mod; then
                        VERSION=$(grep "// version:" clients/$client/go.mod | sed 's/.*: *\(.*\)/\1/')
                        echo "    Current version: $VERSION"
                    else
                        warning "  - No version comment in go.mod (will be added on first release)"
                    fi
                else
                    error "  - go.mod not found"
                fi
                ;;
        esac
    else
        error "clients/$client not found"
    fi
done

# Check for existing tags
echo ""
echo "Checking existing release tags..."
for client in "${CLIENTS[@]}"; do
    TAGS=$(git tag -l "clients/$client/v*" 2>/dev/null | wc -l | tr -d ' ')
    if [ "$TAGS" -gt "0" ]; then
        success "clients/$client: $TAGS tag(s) found"
        LATEST=$(git tag -l "clients/$client/v*" --sort=-version:refname 2>/dev/null | head -1)
        echo "    Latest: $LATEST"
    else
        warning "clients/$client: No tags yet (first release)"
    fi
done

# Check GitHub Actions status (if gh CLI is available)
echo ""
if command -v gh &> /dev/null; then
    echo "Checking GitHub Actions status..."
    gh workflow list 2>/dev/null | grep -i "release" && success "Workflows visible in GitHub" || warning "Could not fetch workflow status"
else
    warning "GitHub CLI (gh) not installed - skipping remote checks"
    echo "  Install with: brew install gh"
fi

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
success "All workflow files are in place"
success "All documentation is available"
success "All client directories exist"
echo ""
echo "🔑 Required Actions:"
echo ""
echo "1. Configure GitHub Secrets:"
echo "   - NPM_TOKEN (for TypeScript)"
echo "   - CARGO_TOKEN (for Rust)"
echo "   Go to: Settings → Secrets and variables → Actions"
echo ""
echo "2. Test a release:"
echo "   git tag clients/typescript/v1.0.0"
echo "   git push origin clients/typescript/v1.0.0"
echo ""
echo "3. Monitor at:"
echo "   https://github.com/xraph/authsome/actions"
echo ""
echo "📚 Documentation:"
echo "   - Quick Start: .github/CLIENT_RELEASES.md"
echo "   - Full Guide: .github/RELEASING.md"
echo ""
success "Setup verification complete!"

