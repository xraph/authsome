#!/usr/bin/env bash

# AuthSome CLI Comprehensive Test Suite
# Tests all 25+ CLI commands systematically
# Date: 2025-10-16

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test database
TEST_DB="./test_cli_comprehensive.db"
KEYS_DIR="./test-keys"
CONFIG_FILE="./test_config.yaml"

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Track test categories using simple counters
BUILD_TESTS=0
BUILD_PASSED=0
BUILD_FAILED=0
MIGRATE_TESTS=0
MIGRATE_PASSED=0
MIGRATE_FAILED=0
GENERATE_TESTS=0
GENERATE_PASSED=0
GENERATE_FAILED=0
CONFIG_TESTS=0
CONFIG_PASSED=0
CONFIG_FAILED=0
ORG_TESTS=0
ORG_PASSED=0
ORG_FAILED=0
USER_TESTS=0
USER_PASSED=0
USER_FAILED=0
ORG_MEMBERS_TESTS=0
ORG_MEMBERS_PASSED=0
ORG_MEMBERS_FAILED=0
SEED_TESTS=0
SEED_PASSED=0
SEED_FAILED=0
ADVANCED_TESTS=0
ADVANCED_PASSED=0
ADVANCED_FAILED=0
CLEANUP_TESTS=0
CLEANUP_PASSED=0
CLEANUP_FAILED=0

# Helper functions
print_header() {
echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo ""
}

print_test() {
    echo -e "${YELLOW}▶ Test $TESTS_RUN: $1${NC}"
}

test_passed() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "${GREEN}  ✓ PASSED${NC}"
echo ""
}

test_failed() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "${RED}  ✗ FAILED: $1${NC}"
echo ""
}

update_category_counters() {
    local category="$1"
    local passed="$2"
    
    case "$category" in
        "BUILD")
            BUILD_TESTS=$((BUILD_TESTS + 1))
            [ "$passed" = "true" ] && BUILD_PASSED=$((BUILD_PASSED + 1)) || BUILD_FAILED=$((BUILD_FAILED + 1))
            ;;
        "MIGRATE")
            MIGRATE_TESTS=$((MIGRATE_TESTS + 1))
            [ "$passed" = "true" ] && MIGRATE_PASSED=$((MIGRATE_PASSED + 1)) || MIGRATE_FAILED=$((MIGRATE_FAILED + 1))
            ;;
        "GENERATE")
            GENERATE_TESTS=$((GENERATE_TESTS + 1))
            [ "$passed" = "true" ] && GENERATE_PASSED=$((GENERATE_PASSED + 1)) || GENERATE_FAILED=$((GENERATE_FAILED + 1))
            ;;
        "CONFIG")
            CONFIG_TESTS=$((CONFIG_TESTS + 1))
            [ "$passed" = "true" ] && CONFIG_PASSED=$((CONFIG_PASSED + 1)) || CONFIG_FAILED=$((CONFIG_FAILED + 1))
            ;;
        "ORG")
            ORG_TESTS=$((ORG_TESTS + 1))
            [ "$passed" = "true" ] && ORG_PASSED=$((ORG_PASSED + 1)) || ORG_FAILED=$((ORG_FAILED + 1))
            ;;
        "USER")
            USER_TESTS=$((USER_TESTS + 1))
            [ "$passed" = "true" ] && USER_PASSED=$((USER_PASSED + 1)) || USER_FAILED=$((USER_FAILED + 1))
            ;;
        "ORG_MEMBERS")
            ORG_MEMBERS_TESTS=$((ORG_MEMBERS_TESTS + 1))
            [ "$passed" = "true" ] && ORG_MEMBERS_PASSED=$((ORG_MEMBERS_PASSED + 1)) || ORG_MEMBERS_FAILED=$((ORG_MEMBERS_FAILED + 1))
            ;;
        "SEED")
            SEED_TESTS=$((SEED_TESTS + 1))
            [ "$passed" = "true" ] && SEED_PASSED=$((SEED_PASSED + 1)) || SEED_FAILED=$((SEED_FAILED + 1))
            ;;
        "ADVANCED")
            ADVANCED_TESTS=$((ADVANCED_TESTS + 1))
            [ "$passed" = "true" ] && ADVANCED_PASSED=$((ADVANCED_PASSED + 1)) || ADVANCED_FAILED=$((ADVANCED_FAILED + 1))
            ;;
        "CLEANUP")
            CLEANUP_TESTS=$((CLEANUP_TESTS + 1))
            [ "$passed" = "true" ] && CLEANUP_PASSED=$((CLEANUP_PASSED + 1)) || CLEANUP_FAILED=$((CLEANUP_FAILED + 1))
            ;;
    esac
}

run_test() {
    local category="$1"
    local description="$2"
    local command="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    print_test "$description"
    
    if eval "$command" > /tmp/test_output.log 2>&1; then
        test_passed
        update_category_counters "$category" "true"
        return 0
    else
        test_failed "Command failed"
        cat /tmp/test_output.log
        update_category_counters "$category" "false"
        return 1
    fi
}

run_test_with_output() {
    local category="$1"
    local description="$2"
    local command="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    print_test "$description"
    
    if OUTPUT=$(eval "$command" 2>&1); then
        echo "$OUTPUT"
        test_passed
        update_category_counters "$category" "true"
        return 0
    else
        echo "$OUTPUT"
        test_failed "Command failed"
        update_category_counters "$category" "false"
        return 1
    fi
}

extract_id() {
    # Use sed for macOS compatibility (BSD grep doesn't support -P)
    sed -n 's/.*ID: *\([a-z0-9-]*\).*/\1/p' | head -1
}

# Cleanup function
cleanup() {
echo ""
    echo "Cleaning up test artifacts..."
    rm -f "$TEST_DB"
    rm -rf "$KEYS_DIR"
    rm -f "$CONFIG_FILE"
    rm -f /tmp/test_output.log
    rm -f ./authsome-cli
}

# Set cleanup trap
trap cleanup EXIT

# Clean up any existing test artifacts first
echo "Cleaning up any previous test artifacts..."
rm -f "$TEST_DB"
rm -rf "$KEYS_DIR"
rm -f "$CONFIG_FILE"
rm -f ./test_saas_config.yaml
rm -f /tmp/test_output.log

# Start tests
print_header "AuthSome CLI Comprehensive Test Suite"
echo "Testing all 25+ CLI commands"
echo "Database: $TEST_DB"
echo "Start time: $(date)"
echo ""

# ============================================================================
# PHASE 1: BUILD AND SETUP
# ============================================================================
print_header "PHASE 1: Build and Setup"

# Test 1: Build CLI
run_test "BUILD" "Build CLI tool" \
    "go build -o authsome-cli ./cmd/authsome-cli"

# Test 2: Verify CLI binary
run_test "BUILD" "Verify CLI binary exists" \
    "[ -f ./authsome-cli ] && [ -x ./authsome-cli ]"

# Test 3: Check version
run_test "BUILD" "Check CLI version" \
    "./authsome-cli --version"

# Test 4: Check help
run_test "BUILD" "Check CLI help" \
    "./authsome-cli --help"

# Set database URL
export DATABASE_URL="$TEST_DB"

# ============================================================================
# PHASE 2: DATABASE MIGRATIONS
# ============================================================================
print_header "PHASE 2: Database Migrations"

# Test 5: Migrate status (before migration)
run_test_with_output "MIGRATE" "Check migration status (before)" \
    "./authsome-cli migrate status"

# Test 6: Migrate up
run_test "MIGRATE" "Run migrations" \
    "./authsome-cli migrate up"

# Test 7: Migrate status (after migration)
run_test_with_output "MIGRATE" "Check migration status (after)" \
    "./authsome-cli migrate status"

# Test 8: Verify database file created
run_test "MIGRATE" "Verify database file created" \
    "[ -f $TEST_DB ]"

# ============================================================================
# PHASE 3: CODE GENERATION
# ============================================================================
print_header "PHASE 3: Code Generation"

# Test 9: Generate keys
run_test "GENERATE" "Generate RSA key pair" \
    "./authsome-cli generate keys --output $KEYS_DIR"

# Test 10: Verify private key
run_test "GENERATE" "Verify private key exists" \
    "[ -f $KEYS_DIR/private.pem ]"

# Test 11: Verify public key
run_test "GENERATE" "Verify public key exists" \
    "[ -f $KEYS_DIR/public.pem ]"

# Test 12: Generate secret
run_test_with_output "GENERATE" "Generate secret" \
    "./authsome-cli generate secret"

# Test 13: Generate secret with custom length
run_test_with_output "GENERATE" "Generate secret (length 64)" \
    "./authsome-cli generate secret --length 64"

# ============================================================================
# PHASE 4: CONFIGURATION MANAGEMENT
# ============================================================================
print_header "PHASE 4: Configuration Management"

# Test 14: Initialize standalone config
run_test "CONFIG" "Initialize standalone configuration" \
    "./authsome-cli config init --mode standalone --output $CONFIG_FILE --force"

# Test 15: Verify config file created
run_test "CONFIG" "Verify config file exists" \
    "[ -f $CONFIG_FILE ]"

# Test 16: Validate config
run_test "CONFIG" "Validate configuration" \
    "./authsome-cli config validate $CONFIG_FILE"

# Test 17: Show config
run_test_with_output "CONFIG" "Show configuration" \
    "./authsome-cli config show $CONFIG_FILE"

# ============================================================================
# PHASE 5: ORGANIZATION MANAGEMENT
# ============================================================================
print_header "PHASE 5: Organization Management"

# Test 18: List organizations (empty)
run_test_with_output "ORG" "List organizations (empty)" \
    "./authsome-cli org list"

# Test 19: Create organization
TESTS_RUN=$((TESTS_RUN + 1))
print_test "Create organization"
if ORG_OUTPUT=$(./authsome-cli org create --name "Test Corporation" --slug "testcorp" 2>&1); then
    echo "$ORG_OUTPUT"
    ORG_ID=$(echo "$ORG_OUTPUT" | extract_id)
    if [ -n "$ORG_ID" ]; then
        echo "  Organization ID: $ORG_ID"
        test_passed
        update_category_counters "ORG" "true"
    else
        test_failed "Could not extract organization ID"
        update_category_counters "ORG" "false"
        ORG_ID="dummy-org-id"
    fi
else
echo "$ORG_OUTPUT"
    test_failed "Failed to create organization"
    update_category_counters "ORG" "false"
    ORG_ID="dummy-org-id"
fi

# Test 20: List organizations (should have 1)
run_test_with_output "ORG" "List organizations (with data)" \
    "./authsome-cli org list"

# Test 21: Show organization details
if [ "$ORG_ID" != "dummy-org-id" ]; then
    run_test_with_output "ORG" "Show organization details" \
        "./authsome-cli org show $ORG_ID"
else
    echo "Skipping org show test (no valid org ID)"
fi

# Test 22: Create second organization
TESTS_RUN=$((TESTS_RUN + 1))
print_test "Create second organization"
if ORG2_OUTPUT=$(./authsome-cli org create --name "Acme Inc" --slug "acme" 2>&1); then
    echo "$ORG2_OUTPUT"
    ORG2_ID=$(echo "$ORG2_OUTPUT" | extract_id)
    if [ -n "$ORG2_ID" ]; then
        echo "  Organization 2 ID: $ORG2_ID"
        test_passed
        update_category_counters "ORG" "true"
    else
        test_failed "Could not extract organization ID"
        update_category_counters "ORG" "false"
        ORG2_ID="dummy-org2-id"
    fi
else
    echo "$ORG2_OUTPUT"
    test_failed "Failed to create second organization"
    update_category_counters "ORG" "false"
    ORG2_ID="dummy-org2-id"
fi

# ============================================================================
# PHASE 6: USER MANAGEMENT
# ============================================================================
print_header "PHASE 6: User Management"

# Test 23: List users (empty)
run_test_with_output "USER" "List users (empty)" \
    "./authsome-cli user list"

# Test 24: Create user
TESTS_RUN=$((TESTS_RUN + 1))
print_test "Create user"
if [ "$ORG_ID" != "dummy-org-id" ]; then
    if USER_OUTPUT=$(./authsome-cli user create \
        --email "admin@testcorp.com" \
        --password "SecurePass123!" \
        --first-name "Admin" \
        --last-name "User" \
  --org "$ORG_ID" \
        --verified 2>&1); then
        echo "$USER_OUTPUT"
        USER_ID=$(echo "$USER_OUTPUT" | extract_id)
        if [ -n "$USER_ID" ]; then
            echo "  User ID: $USER_ID"
            test_passed
            update_category_counters "USER" "true"
        else
            test_failed "Could not extract user ID"
            update_category_counters "USER" "false"
            USER_ID="dummy-user-id"
        fi
    else
echo "$USER_OUTPUT"
        test_failed "Failed to create user"
        update_category_counters "USER" "false"
        USER_ID="dummy-user-id"
    fi
else
    echo "Skipping user creation (no valid org ID)"
    test_failed "No valid org ID"
    update_category_counters "USER" "false"
    USER_ID="dummy-user-id"
fi

# Test 25: List users (should have 1)
run_test_with_output "USER" "List users (with data)" \
    "./authsome-cli user list"

# Test 26: Show user details
if [ "$USER_ID" != "dummy-user-id" ]; then
    run_test_with_output "USER" "Show user details" \
        "./authsome-cli user show $USER_ID"
else
    echo "Skipping user show test (no valid user ID)"
fi

# Test 27: List users by organization
if [ "$ORG_ID" != "dummy-org-id" ]; then
    run_test_with_output "USER" "List users by organization" \
        "./authsome-cli user list --org $ORG_ID"
else
    echo "Skipping user list by org test (no valid org ID)"
fi

# Test 28: Create second user
TESTS_RUN=$((TESTS_RUN + 1))
print_test "Create second user"
if [ "$ORG_ID" != "dummy-org-id" ]; then
    if USER2_OUTPUT=$(./authsome-cli user create \
        --email "john@testcorp.com" \
        --password "Password456!" \
        --first-name "John" \
        --last-name "Doe" \
        --org "$ORG_ID" 2>&1); then
        echo "$USER2_OUTPUT"
        USER2_ID=$(echo "$USER2_OUTPUT" | extract_id)
        if [ -n "$USER2_ID" ]; then
            echo "  User 2 ID: $USER2_ID"
            test_passed
            update_category_counters "USER" "true"
        else
            test_failed "Could not extract user ID"
            update_category_counters "USER" "false"
            USER2_ID="dummy-user2-id"
        fi
    else
        echo "$USER2_OUTPUT"
        test_failed "Failed to create second user"
        update_category_counters "USER" "false"
        USER2_ID="dummy-user2-id"
    fi
else
    echo "Skipping second user creation (no valid org ID)"
    test_failed "No valid org ID"
    update_category_counters "USER" "false"
    USER2_ID="dummy-user2-id"
fi

# Test 29: Verify user email
if [ "$USER2_ID" != "dummy-user2-id" ]; then
    run_test "USER" "Verify user email" \
        "./authsome-cli user verify $USER2_ID"
else
    echo "Skipping user verify test (no valid user ID)"
fi

# Test 30: Update user password
if [ "$USER_ID" != "dummy-user-id" ]; then
    run_test "USER" "Update user password" \
        "./authsome-cli user password $USER_ID --password 'NewSecurePass789!'"
else
    echo "Skipping password update test (no valid user ID)"
fi

# ============================================================================
# PHASE 7: ORGANIZATION MEMBER MANAGEMENT
# ============================================================================
print_header "PHASE 7: Organization Member Management"

# Test 31: List organization members
if [ "$ORG_ID" != "dummy-org-id" ]; then
    run_test_with_output "ORG_MEMBERS" "List organization members" \
        "./authsome-cli org members $ORG_ID"
else
    echo "Skipping members list test (no valid org ID)"
fi

# Test 32: Add member to organization
if [ "$ORG2_ID" != "dummy-org2-id" ] && [ "$USER_ID" != "dummy-user-id" ]; then
    run_test "ORG_MEMBERS" "Add member to organization" \
        "./authsome-cli org add-member $ORG2_ID $USER_ID --role member"
else
    echo "Skipping add member test (no valid IDs)"
fi

# Test 33: List members of second organization
if [ "$ORG2_ID" != "dummy-org2-id" ]; then
    run_test_with_output "ORG_MEMBERS" "List members of second org" \
        "./authsome-cli org members $ORG2_ID"
else
    echo "Skipping second org members list test (no valid org ID)"
fi

# ============================================================================
# PHASE 8: DATA SEEDING
# ============================================================================
print_header "PHASE 8: Data Seeding"

# Test 34: Seed basic data
run_test_with_output "SEED" "Seed basic data" \
    "./authsome-cli seed basic"

# Test 35: List users after seeding
run_test_with_output "SEED" "List users after basic seed" \
    "./authsome-cli user list"

# Test 36: List organizations after seeding
run_test_with_output "SEED" "List organizations after basic seed" \
    "./authsome-cli org list"

# Get an org ID for user seeding
SEED_ORG_OUTPUT=$(./authsome-cli org list 2>&1)
SEED_ORG_ID=$(echo "$SEED_ORG_OUTPUT" | sed -n 's/.*ID: *\([a-z0-9-]*\).*/\1/p' | head -1)

# Test 37: Seed users
if [ -n "$SEED_ORG_ID" ]; then
    run_test_with_output "SEED" "Seed additional users" \
        "./authsome-cli seed users --count 5 --org $SEED_ORG_ID"
else
    echo "Skipping user seeding (no org ID found)"
fi

# Test 38: List users after user seeding
run_test_with_output "SEED" "List users after user seed" \
    "./authsome-cli user list"

# Test 39: Seed organizations
run_test_with_output "SEED" "Seed additional organizations" \
    "./authsome-cli seed orgs --count 3"

# Test 40: List organizations after org seeding
run_test_with_output "SEED" "List organizations after org seed" \
    "./authsome-cli org list"

# ============================================================================
# PHASE 9: ADVANCED OPERATIONS
# ============================================================================
print_header "PHASE 9: Advanced Operations"

# Test 41: Generate another config (SaaS mode)
run_test "ADVANCED" "Generate SaaS configuration" \
    "./authsome-cli config init --mode saas --output ./test_saas_config.yaml --force"

# Test 42: Validate SaaS config
run_test "ADVANCED" "Validate SaaS configuration" \
    "./authsome-cli config validate ./test_saas_config.yaml"

# Test 43: Show SaaS config
run_test_with_output "ADVANCED" "Show SaaS configuration" \
    "./authsome-cli config show ./test_saas_config.yaml"

# Cleanup SaaS config
rm -f ./test_saas_config.yaml

# Test 44: Test verbose flag
run_test "ADVANCED" "Test verbose flag" \
    "./authsome-cli --verbose org list"

# ============================================================================
# PHASE 10: CLEANUP AND DELETION TESTS
# ============================================================================
print_header "PHASE 10: Cleanup and Deletion Tests"

# Test 45: Remove member from organization
if [ "$ORG2_ID" != "dummy-org2-id" ] && [ "$USER_ID" != "dummy-user-id" ]; then
    run_test "CLEANUP" "Remove member from organization" \
        "./authsome-cli org remove-member $ORG2_ID $USER_ID"
else
    echo "Skipping remove member test (no valid IDs)"
fi

# Test 46: Delete user
if [ "$USER2_ID" != "dummy-user2-id" ]; then
    run_test "CLEANUP" "Delete user" \
        "./authsome-cli user delete $USER2_ID --force"
else
    echo "Skipping user deletion test (no valid user ID)"
fi

# Test 47: Verify user deleted
if [ "$USER2_ID" != "dummy-user2-id" ]; then
    TESTS_RUN=$((TESTS_RUN + 1))
    print_test "Verify user deleted"
    if ./authsome-cli user show "$USER2_ID" 2>&1 | grep -q "not found"; then
        test_passed
        update_category_counters "CLEANUP" "true"
    else
        test_failed "User still exists"
        update_category_counters "CLEANUP" "false"
    fi
else
    echo "Skipping user deletion verify test (no valid user ID)"
fi

# Test 48: Delete organization
if [ "$ORG2_ID" != "dummy-org2-id" ]; then
    run_test "CLEANUP" "Delete organization" \
        "./authsome-cli org delete $ORG2_ID --confirm"
else
    echo "Skipping org deletion test (no valid org ID)"
fi

# Test 49: Verify organization deleted
if [ "$ORG2_ID" != "dummy-org2-id" ]; then
    TESTS_RUN=$((TESTS_RUN + 1))
    print_test "Verify organization deleted"
    if ./authsome-cli org show "$ORG2_ID" 2>&1 | grep -q "not found"; then
        test_passed
        update_category_counters "CLEANUP" "true"
    else
        test_failed "Organization still exists"
        update_category_counters "CLEANUP" "false"
    fi
else
    echo "Skipping org deletion verify test (no valid org ID)"
fi

# Test 50: Clear seeded data
run_test "CLEANUP" "Clear seeded data" \
    "./authsome-cli seed clear --confirm"

# ============================================================================
# PHASE 11: DATABASE MIGRATION ROLLBACK
# ============================================================================
print_header "PHASE 11: Database Migration Rollback"

# Test 51: Migration down
run_test "MIGRATE" "Rollback migration" \
    "./authsome-cli migrate down"

# Test 52: Check migration status after rollback
run_test_with_output "MIGRATE" "Check migration status after rollback" \
    "./authsome-cli migrate status"

# Test 53: Migration up again
run_test "MIGRATE" "Re-run migrations" \
    "./authsome-cli migrate up"

# Test 54: Final migration status
run_test_with_output "MIGRATE" "Final migration status" \
    "./authsome-cli migrate status"

# ============================================================================
# FINAL SUMMARY
# ============================================================================
print_header "Test Results Summary"

echo "Total Tests Run: $TESTS_RUN"
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
else
    echo -e "${GREEN}Tests Failed: $TESTS_FAILED${NC}"
fi

PASS_RATE=0
if [ $TESTS_RUN -gt 0 ]; then
    PASS_RATE=$((TESTS_PASSED * 100 / TESTS_RUN))
fi
echo "Pass Rate: ${PASS_RATE}%"

echo ""
echo "Results by Category:"
echo "─────────────────────────────────────────────────────────────"

print_category() {
    local name="$1"
    local tests="$2"
    local passed="$3"
    local failed="$4"
    
    if [ $tests -eq 0 ]; then
        return
    fi
    
    local rate=$((passed * 100 / tests))
    
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}$name: $passed/$tests ($rate%)${NC}"
    else
        echo -e "${RED}$name: $passed/$tests ($rate%) - $failed failed${NC}"
    fi
}

print_category "BUILD" $BUILD_TESTS $BUILD_PASSED $BUILD_FAILED
print_category "MIGRATE" $MIGRATE_TESTS $MIGRATE_PASSED $MIGRATE_FAILED
print_category "GENERATE" $GENERATE_TESTS $GENERATE_PASSED $GENERATE_FAILED
print_category "CONFIG" $CONFIG_TESTS $CONFIG_PASSED $CONFIG_FAILED
print_category "ORG" $ORG_TESTS $ORG_PASSED $ORG_FAILED
print_category "USER" $USER_TESTS $USER_PASSED $USER_FAILED
print_category "ORG_MEMBERS" $ORG_MEMBERS_TESTS $ORG_MEMBERS_PASSED $ORG_MEMBERS_FAILED
print_category "SEED" $SEED_TESTS $SEED_PASSED $SEED_FAILED
print_category "ADVANCED" $ADVANCED_TESTS $ADVANCED_PASSED $ADVANCED_FAILED
print_category "CLEANUP" $CLEANUP_TESTS $CLEANUP_PASSED $CLEANUP_FAILED

echo ""
echo "End time: $(date)"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                                                           ║${NC}"
    echo -e "${GREEN}║  ✓ ALL TESTS PASSED - CLI IS FULLY FUNCTIONAL ✓          ║${NC}"
    echo -e "${GREEN}║                                                           ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${RED}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║                                                           ║${NC}"
    echo -e "${RED}║  ✗ SOME TESTS FAILED - REVIEW OUTPUT ABOVE ✗             ║${NC}"
    echo -e "${RED}║                                                           ║${NC}"
    echo -e "${RED}╚═══════════════════════════════════════════════════════════╝${NC}"
    exit 1
fi
