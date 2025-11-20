# OIDC Provider Test Client

A comprehensive test client for validating the OIDC Provider plugin implementation.

## Features Tested

✅ OIDC Discovery (/.well-known/openid-configuration)
✅ Client Registration (RFC 7591)
✅ Authorization Code Flow with PKCE
✅ Token Exchange
✅ UserInfo Endpoint
✅ Token Introspection (RFC 7662)
✅ Token Revocation (RFC 7009)

## Prerequisites

1. **AuthSome server running** on `http://localhost:3001`
2. **User account created** for testing
3. **Optional**: Admin token for dynamic client registration

## Usage

### Quick Start

```bash
cd plugins/oidcprovider/test_client
go run main.go
```

### Manual Testing Flow

The test client will:

1. **Discover OIDC Endpoints** - Fetch `.well-known/openid-configuration`
2. **Use Client Credentials** - Either register new or use existing client
3. **Generate Authorization URL** - With PKCE challenge
4. **Wait for User Input** - You manually complete the authorization flow
5. **Exchange Code for Tokens** - Once you provide the auth code
6. **Fetch User Info** - Using the access token
7. **Introspect Token** - Verify token metadata (if confidential client)
8. **Revoke Token** - Revoke the access token
9. **Verify Revocation** - Confirm token no longer works

### Example Output

```
🚀 OIDC Provider Test Client
================================

📡 Discovering OIDC endpoints...
✅ Discovery successful!
   Issuer: http://localhost:3001
   Authorization: http://localhost:3001/oauth2/authorize
   Token: http://localhost:3001/oauth2/token
   UserInfo: http://localhost:3001/oauth2/userinfo
   JWKS: http://localhost:3001/oauth2/jwks
   Scopes: [openid profile email phone address offline_access]

📌 Using existing client: client_test123

🔗 Authorization URL:
http://localhost:3001/oauth2/authorize?client_id=client_test123&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback&response_type=code&scope=openid+profile+email&state=abc123&code_challenge=xyz789&code_challenge_method=S256

📋 Manual Steps:
1. Open the above URL in a browser
2. Log in if not already authenticated
3. Grant consent if prompted
4. Copy the authorization code from the redirect URL

Enter the authorization code: def456789

🔄 Exchanging authorization code for tokens...
✅ Tokens received successfully!
   Access Token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOi...
   Token Type: Bearer
   Expires In: 3600 seconds
   Refresh Token: refresh_01HZ...
   ID Token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOi...
   Scope: openid profile email

👤 Fetching user info...
✅ User info retrieved!
   Subject: user_01HZ...
   Email: test@example.com
   Email Verified: true
   Name: Test User

⏳ Waiting 3 seconds before revocation...

🗑️  Revoking token...
✅ Token revoked successfully!

🔍 Verifying token revocation...
✅ Token is properly revoked (UserInfo failed as expected)

================================
✨ OIDC flow test complete!
```

## Configuration

### Using Existing Client

Edit `main.go` line 427:

```go
// Use your pre-registered client
client.UseExistingClient("your-client-id", "your-client-secret")
```

### Registering New Client

Uncomment lines 417-423 in `main.go` and provide admin token:

```go
adminToken := "your-admin-token-here"
if err := client.RegisterClient(adminToken); err != nil {
    log.Printf("⚠️  Client registration failed: %v", err)
    log.Println("   Using manual client configuration...")
    client.UseExistingClient("fallback-client-id", "")
}
```

### Server URL

Change `baseURL` constant at the top of `main.go`:

```go
const baseURL = "https://your-server.com"
```

## Pre-registering a Test Client

You can pre-register a client using curl:

```bash
# Register a public client (SPA) with PKCE
curl -X POST http://localhost:3001/oauth2/register \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "client_name": "Test Client",
    "redirect_uris": ["http://localhost:8080/callback"],
    "application_type": "spa",
    "token_endpoint_auth_method": "none",
    "require_pkce": true
  }'

# Response includes client_id to use in test
```

## Testing Different Client Types

### Public Client (SPA/Native) with PKCE

```go
reqBody := ClientRegistrationRequest{
    ClientName:              "SPA Client",
    RedirectURIs:            []string{"http://localhost:8080/callback"},
    ApplicationType:         "spa",
    TokenEndpointAuthMethod: "none",
    RequirePKCE:             true,
}
```

### Confidential Client (Web App)

```go
reqBody := ClientRegistrationRequest{
    ClientName:              "Web App Client",
    RedirectURIs:            []string{"https://app.example.com/callback"},
    ApplicationType:         "web",
    TokenEndpointAuthMethod: "client_secret_basic",
    RequirePKCE:             false,
}
```

## Troubleshooting

### "Discovery endpoint returned 404"

- Ensure AuthSome server is running on the correct port
- Check that OIDC provider plugin is registered and initialized

### "Client registration failed with status 403"

- You need an admin token to register clients
- Use a pre-registered client instead

### "Token exchange failed with status 400"

- Check PKCE code_verifier matches the challenge
- Verify authorization code hasn't expired (10 minute TTL)
- Ensure redirect_uri matches exactly

### "Invalid authorization code"

- Code may have expired (they're valid for 10 minutes)
- Code can only be used once
- Generate a new authorization URL and try again

### "Token introspection failed"

- Introspection requires a confidential client
- Public clients (PKCE-only) cannot introspect tokens
- Use a client with client_secret

## Advanced Testing

### Testing Consent Flow

1. Clear consent for your test client in the database
2. Run the test client
3. Verify consent screen is displayed
4. Grant consent
5. Run test again - should skip consent

### Testing Token Revocation Cascade

```sql
-- Create a session with multiple tokens
-- Revoke by session_id
-- Verify all tokens are revoked
```

### Testing Org-Specific Clients

1. Set organization context in your test environment
2. Register client with org context
3. Verify client is org-specific
4. Test hierarchy resolution (org → app fallback)

## Integration with CI/CD

```bash
#!/bin/bash
# test-oidc-flow.sh

# Start AuthSome server in background
./authsome &
SERVER_PID=$!

# Wait for server to be ready
sleep 5

# Run test client with pre-configured client
echo "test_auth_code" | go run main.go

# Check exit code
if [ $? -eq 0 ]; then
    echo "✅ OIDC flow test passed"
    kill $SERVER_PID
    exit 0
else
    echo "❌ OIDC flow test failed"
    kill $SERVER_PID
    exit 1
fi
```

## Next Steps

After successful testing:

1. ✅ Verify all endpoints return correct status codes
2. ✅ Confirm PKCE validation works correctly
3. ✅ Test consent flow persistence
4. ✅ Validate token revocation cascade
5. ✅ Check introspection for confidential clients
6. 📝 Write automated integration tests
7. 🔒 Perform security audit
8. 🚀 Deploy to staging environment

