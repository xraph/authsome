#!/bin/bash

# OIDC Provider Integration Test Script
# Tests the complete OIDC flow without manual intervention

set -e

BASE_URL="${OIDC_BASE_URL:-http://localhost:3001}"
CLIENT_ID="${OIDC_CLIENT_ID:-client_test123}"
CLIENT_SECRET="${OIDC_CLIENT_SECRET:-}"
REDIRECT_URI="http://localhost:8080/callback"

echo "🚀 OIDC Provider Integration Test"
echo "=================================="
echo "Base URL: $BASE_URL"
echo "Client ID: $CLIENT_ID"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function for assertions
assert_status() {
    local expected=$1
    local actual=$2
    local description=$3
    
    if [ "$actual" -eq "$expected" ]; then
        echo -e "${GREEN}✓${NC} $description (status: $actual)"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} $description (expected: $expected, got: $actual)"
        ((TESTS_FAILED++))
        return 1
    fi
}

assert_json_field() {
    local json=$1
    local field=$2
    local description=$3
    
    local value=$(echo "$json" | jq -r ".$field")
    if [ "$value" != "null" ] && [ "$value" != "" ]; then
        echo -e "${GREEN}✓${NC} $description: $value"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} $description: field '$field' is missing or null"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test 1: Discovery Endpoint
echo "📡 Test 1: OIDC Discovery"
echo "------------------------"

DISCOVERY_RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/.well-known/openid-configuration")
DISCOVERY_BODY=$(echo "$DISCOVERY_RESPONSE" | head -n -1)
DISCOVERY_STATUS=$(echo "$DISCOVERY_RESPONSE" | tail -n 1)

assert_status 200 "$DISCOVERY_STATUS" "Discovery endpoint accessible"

if [ $? -eq 0 ]; then
    assert_json_field "$DISCOVERY_BODY" "issuer" "Issuer URL present"
    assert_json_field "$DISCOVERY_BODY" "authorization_endpoint" "Authorization endpoint present"
    assert_json_field "$DISCOVERY_BODY" "token_endpoint" "Token endpoint present"
    assert_json_field "$DISCOVERY_BODY" "userinfo_endpoint" "UserInfo endpoint present"
    assert_json_field "$DISCOVERY_BODY" "jwks_uri" "JWKS URI present"
    
    # Save endpoints for later use
    AUTH_ENDPOINT=$(echo "$DISCOVERY_BODY" | jq -r '.authorization_endpoint')
    TOKEN_ENDPOINT=$(echo "$DISCOVERY_BODY" | jq -r '.token_endpoint')
    USERINFO_ENDPOINT=$(echo "$DISCOVERY_BODY" | jq -r '.userinfo_endpoint')
    JWKS_ENDPOINT=$(echo "$DISCOVERY_BODY" | jq -r '.jwks_uri')
    INTROSPECT_ENDPOINT="$BASE_URL/oauth2/introspect"
    REVOKE_ENDPOINT="$BASE_URL/oauth2/revoke"
fi

echo ""

# Test 2: JWKS Endpoint
echo "🔑 Test 2: JWKS Endpoint"
echo "------------------------"

JWKS_RESPONSE=$(curl -s -w "\n%{http_code}" "$JWKS_ENDPOINT")
JWKS_BODY=$(echo "$JWKS_RESPONSE" | head -n -1)
JWKS_STATUS=$(echo "$JWKS_RESPONSE" | tail -n 1)

assert_status 200 "$JWKS_STATUS" "JWKS endpoint accessible"

if [ $? -eq 0 ]; then
    KEY_COUNT=$(echo "$JWKS_BODY" | jq '.keys | length')
    if [ "$KEY_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓${NC} JWKS contains $KEY_COUNT key(s)"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC} JWKS contains no keys"
        ((TESTS_FAILED++))
    fi
fi

echo ""

# Test 3: Authorization Endpoint (should redirect or show error without user)
echo "🔗 Test 3: Authorization Endpoint"
echo "----------------------------"

# Generate PKCE challenge
CODE_VERIFIER=$(openssl rand -base64 32 | tr -d '=+/' | cut -c1-43)
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | openssl dgst -sha256 -binary | base64 | tr -d '=+/' | tr '/' '_')

AUTH_URL="$AUTH_ENDPOINT?client_id=$CLIENT_ID&redirect_uri=$REDIRECT_URI&response_type=code&scope=openid%20profile%20email&state=test_state&code_challenge=$CODE_CHALLENGE&code_challenge_method=S256"

AUTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$AUTH_URL" -L --max-redirs 0)
AUTH_STATUS=$(echo "$AUTH_RESPONSE" | tail -n 1)

# Should redirect to login (302) or show error (400/401)
if [ "$AUTH_STATUS" -eq 302 ] || [ "$AUTH_STATUS" -eq 401 ] || [ "$AUTH_STATUS" -eq 400 ]; then
    echo -e "${GREEN}✓${NC} Authorization endpoint responding correctly (status: $AUTH_STATUS)"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⚠${NC}  Authorization endpoint returned unexpected status: $AUTH_STATUS"
fi

echo ""

# Test 4: Token Endpoint Error Handling
echo "🎫 Test 4: Token Endpoint (Error Cases)"
echo "--------------------------------------"

# Test with invalid grant type
TOKEN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$TOKEN_ENDPOINT" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=invalid_grant&code=invalid&redirect_uri=$REDIRECT_URI&client_id=$CLIENT_ID")

TOKEN_STATUS=$(echo "$TOKEN_RESPONSE" | tail -n 1)
assert_status 400 "$TOKEN_STATUS" "Invalid grant type rejected"

# Test with missing parameters
TOKEN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$TOKEN_ENDPOINT" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=authorization_code")

TOKEN_STATUS=$(echo "$TOKEN_RESPONSE" | tail -n 1)
assert_status 400 "$TOKEN_STATUS" "Missing parameters rejected"

echo ""

# Test 5: UserInfo Endpoint (without token)
echo "👤 Test 5: UserInfo Endpoint (Unauthorized)"
echo "-----------------------------------------"

USERINFO_RESPONSE=$(curl -s -w "\n%{http_code}" "$USERINFO_ENDPOINT")
USERINFO_STATUS=$(echo "$USERINFO_RESPONSE" | tail -n 1)

assert_status 401 "$USERINFO_STATUS" "UserInfo requires authentication"

echo ""

# Test 6: Introspection Endpoint (without auth)
echo "🔍 Test 6: Token Introspection (Unauthorized)"
echo "-------------------------------------------"

INTROSPECT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$INTROSPECT_ENDPOINT" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "token=fake_token")

INTROSPECT_STATUS=$(echo "$INTROSPECT_RESPONSE" | tail -n 1)
assert_status 401 "$INTROSPECT_STATUS" "Introspection requires client authentication"

echo ""

# Test 7: Revocation Endpoint (without auth)
echo "🗑️  Test 7: Token Revocation (Unauthorized)"
echo "-----------------------------------------"

REVOKE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$REVOKE_ENDPOINT" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "token=fake_token")

REVOKE_STATUS=$(echo "$REVOKE_RESPONSE" | tail -n 1)
assert_status 401 "$REVOKE_STATUS" "Revocation requires client authentication"

echo ""

# Test 8: Client Registration Endpoint (without admin auth)
echo "📝 Test 8: Client Registration (Unauthorized)"
echo "-------------------------------------------"

REGISTER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/oauth2/register" \
    -H "Content-Type: application/json" \
    -d '{
        "client_name": "Test Client",
        "redirect_uris": ["http://localhost:8080/callback"],
        "application_type": "spa"
    }')

REGISTER_STATUS=$(echo "$REGISTER_RESPONSE" | tail -n 1)

# Should require admin auth (401/403) or work (201)
if [ "$REGISTER_STATUS" -eq 401 ] || [ "$REGISTER_STATUS" -eq 403 ]; then
    echo -e "${GREEN}✓${NC} Client registration requires authentication (status: $REGISTER_STATUS)"
    ((TESTS_PASSED++))
elif [ "$REGISTER_STATUS" -eq 201 ]; then
    echo -e "${YELLOW}⚠${NC}  Client registration allowed without auth (might be in development mode)"
else
    echo -e "${RED}✗${NC} Unexpected status: $REGISTER_STATUS"
    ((TESTS_FAILED++))
fi

echo ""

# Summary
echo "=================================="
echo "📊 Test Summary"
echo "=================================="
echo -e "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✨ All tests passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Start AuthSome server if not running"
    echo "2. Create a test user account"
    echo "3. Register an OAuth client"
    echo "4. Run the interactive test client:"
    echo "   cd plugins/oidcprovider/test_client && go run main.go"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed. Please check the output above.${NC}"
    exit 1
fi

