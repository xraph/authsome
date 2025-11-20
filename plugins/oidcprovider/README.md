# OIDC Provider Plugin - Enterprise Edition

Comprehensive enterprise-grade OpenID Connect Provider plugin for AuthSome with multi-tenancy support, org-specific configurations, and RFC-compliant OAuth2/OIDC implementation.

## Features

### Core OIDC/OAuth2
- ✅ OAuth2 Authorization Code Flow with PKCE
- ✅ OpenID Connect ID Tokens (JWT)
- ✅ Refresh Tokens
- ✅ Automatic Key Rotation
- ✅ JWKS Endpoint
- ✅ UserInfo Endpoint
- ✅ Discovery Endpoint (.well-known/openid-configuration)

### Enterprise Features (RFC-Compliant)
- ✅ **RFC 7591**: Dynamic Client Registration
- ✅ **RFC 7662**: Token Introspection
- ✅ **RFC 7009**: Token Revocation (with cascade support)
- ✅ Client Authentication (client_secret_basic, client_secret_post, none/PKCE)
- ✅ Persistent Consent Management
- ✅ Org-specific OAuth Clients with hierarchy fallback
- ✅ Session-linked Token Lifecycle
- ✅ Admin Client Management Endpoints

### Multi-Tenancy Architecture
- **App-Level Clients**: Default OAuth clients available to all organizations
- **Org-Level Clients**: Organization-specific OAuth clients with custom configurations
- **Hierarchy Resolution**: Org-specific clients override app-level clients
- **Context-Aware**: Full integration with App/Environment/Organization contexts

## Architecture

### Schema Updates

#### OAuthClient
```go
type OAuthClient struct {
    // Context
    AppID          xid.ID
    EnvironmentID  xid.ID
    OrganizationID *xid.ID  // null = app-level, set = org-specific
    
    // OAuth2/OIDC Config
    RedirectURIs            []string
    PostLogoutRedirectURIs  []string
    GrantTypes              []string
    ResponseTypes           []string
    AllowedScopes           []string
    
    // Security
    TokenEndpointAuthMethod string  // client_secret_basic, client_secret_post, none
    ApplicationType         string  // web, native, spa
    RequirePKCE             bool
    RequireConsent          bool
    TrustedClient           bool
    
    // RFC 7591 Metadata
    LogoURI    string
    PolicyURI  string
    TosURI     string
    Contacts   []string
    Metadata   map[string]interface{}
}
```

#### AuthorizationCode
```go
type AuthorizationCode struct {
    AppID          xid.ID
    EnvironmentID  xid.ID
    OrganizationID *xid.ID
    SessionID      *xid.ID     // Links to user session
    
    ConsentGranted bool
    ConsentScopes  string
    AuthTime       time.Time   // For max_age checks
}
```

#### OAuthToken
```go
type OAuthToken struct {
    AppID          xid.ID
    EnvironmentID  xid.ID
    OrganizationID *xid.ID
    SessionID      *xid.ID     // For session-based revocation
    
    IDToken    string
    TokenClass string          // access_token, refresh_token, id_token
    
    // JWT Claims
    JTI       string           // For revocation by ID
    Issuer    string
    Audience  []string
    NotBefore *time.Time
    
    // Authentication Context
    AuthTime *time.Time
    ACR      string
    AMR      []string
}
```

#### OAuthConsent (New)
```go
type OAuthConsent struct {
    AppID          xid.ID
    EnvironmentID  xid.ID
    OrganizationID *xid.ID
    UserID         xid.ID
    ClientID       string
    Scopes         []string
    ExpiresAt      *time.Time  // Optional consent expiration
}
```

## API Endpoints

### Public Endpoints

#### GET `/.well-known/openid-configuration`
Returns OIDC discovery document with all supported endpoints and capabilities.

#### GET `/oauth2/jwks`
Returns JSON Web Key Set for token verification.

#### GET `/oauth2/authorize`
OAuth2/OIDC authorization endpoint. Initiates authorization flow.

**Query Parameters:**
- `client_id` (required)
- `redirect_uri` (required)
- `response_type` (required): `code`
- `scope`: Space-separated scopes (e.g., `openid profile email`)
- `state`: Opaque value for CSRF protection
- `nonce`: For ID token replay protection
- `code_challenge`: PKCE challenge
- `code_challenge_method`: `S256` or `plain`

#### POST `/oauth2/token`
Exchanges authorization code for tokens.

**Request:**
```json
{
  "grant_type": "authorization_code",
  "code": "...",
  "redirect_uri": "...",
  "client_id": "...",
  "client_secret": "...",  // Not required for public clients
  "code_verifier": "..."   // Required for PKCE
}
```

**Response:**
```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "...",
  "id_token": "...",        // Only if openid scope requested
  "scope": "openid profile email"
}
```

#### GET `/oauth2/userinfo`
Returns user information. Requires Bearer token.

**Response:**
```json
{
  "sub": "user_id",
  "email": "user@example.com",
  "email_verified": true,
  "name": "John Doe",
  "preferred_username": "johndoe",
  "picture": "https://..."
}
```

### Enterprise Endpoints

#### POST `/oauth2/introspect` (RFC 7662)
Introspects access or refresh token. Requires client authentication.

**Request:**
```json
{
  "token": "...",
  "token_type_hint": "access_token"  // Optional: access_token or refresh_token
}
```

**Response:**
```json
{
  "active": true,
  "scope": "openid profile",
  "client_id": "client_123",
  "username": "johndoe",
  "token_type": "Bearer",
  "exp": 1609459200,
  "iat": 1609455600,
  "sub": "user_id",
  "aud": ["https://api.example.com"],
  "iss": "https://auth.example.com",
  "jti": "token_id"
}
```

#### POST `/oauth2/revoke` (RFC 7009)
Revokes access or refresh token. Requires client authentication.

**Request:**
```json
{
  "token": "...",
  "token_type_hint": "access_token"  // Optional
}
```

**Response:**
```json
{
  "status": "revoked"
}
```

### Admin Endpoints

#### POST `/oauth2/register` (RFC 7591)
Registers a new OAuth client (admin only).

**Request:**
```json
{
  "client_name": "My Application",
  "redirect_uris": ["https://app.example.com/callback"],
  "post_logout_redirect_uris": ["https://app.example.com"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "application_type": "web",  // web, native, spa
  "token_endpoint_auth_method": "client_secret_basic",
  "logo_uri": "https://app.example.com/logo.png",
  "policy_uri": "https://app.example.com/privacy",
  "tos_uri": "https://app.example.com/terms",
  "contacts": ["admin@example.com"],
  "scope": "openid profile email",
  "require_pkce": true,
  "require_consent": true,
  "trusted_client": false
}
```

**Response:**
```json
{
  "client_id": "client_01HZ...",
  "client_secret": "secret_...",  // Omitted for public clients
  "client_id_issued_at": 1609459200,
  "client_secret_expires_at": 0,  // 0 = never expires
  "client_name": "My Application",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "application_type": "web",
  "token_endpoint_auth_method": "client_secret_basic",
  ...
}
```

#### GET `/oauth2/clients`
Lists all OAuth clients (admin only).

**Query Parameters:**
- `page`: Page number (default: 1)
- `page_size`: Results per page (default: 20, max: 100)

#### GET `/oauth2/clients/:clientId`
Gets OAuth client details (admin only).

#### PUT `/oauth2/clients/:clientId`
Updates OAuth client (admin only).

#### DELETE `/oauth2/clients/:clientId`
Deletes OAuth client and revokes all tokens (admin only).

## Configuration

```yaml
auth:
  oidcprovider:
    issuer: "https://auth.example.com"
    
    keys:
      privateKeyPath: "/path/to/private_key.pem"
      publicKeyPath: "/path/to/public_key.pem"
      rotationInterval: "24h"
      keyLifetime: "168h"  # 7 days
    
    tokens:
      accessTokenExpiry: "1h"
      idTokenExpiry: "1h"
      refreshTokenExpiry: "720h"  # 30 days
```

## Usage

### Initialize Plugin

```go
import "github.com/xraph/authsome/plugins/oidcprovider"

plugin := oidcprovider.NewPlugin(
    oidcprovider.WithIssuer("https://auth.example.com"),
)

authsome.RegisterPlugin(plugin)
```

### Register App-Level Client

```go
// Via API (admin authenticated)
POST /oauth2/register
{
  "client_name": "Platform App",
  "redirect_uris": ["https://app.example.com/callback"],
  "application_type": "web"
}
```

### Register Org-Specific Client

```go
// Set organization context, then register
// Client will be scoped to the organization
POST /oauth2/register
{
  "client_name": "Org Custom App",
  "redirect_uris": ["https://org.example.com/callback"]
}
```

### Authorization Flow

1. **Redirect to Authorization Endpoint:**
```
GET /oauth2/authorize?
  client_id=client_123&
  redirect_uri=https://app.example.com/callback&
  response_type=code&
  scope=openid profile email&
  state=random_state&
  code_challenge=challenge&
  code_challenge_method=S256
```

2. **User Authenticates & Consents**
   - If not logged in, redirected to login
   - If consent required, shown consent screen
   - Consent stored for future requests

3. **Redirect Back with Code:**
```
https://app.example.com/callback?code=auth_code&state=random_state
```

4. **Exchange Code for Tokens:**
```json
POST /oauth2/token
{
  "grant_type": "authorization_code",
  "code": "auth_code",
  "redirect_uri": "https://app.example.com/callback",
  "client_id": "client_123",
  "client_secret": "secret",
  "code_verifier": "verifier"
}
```

5. **Use Access Token:**
```
GET /oauth2/userinfo
Authorization: Bearer access_token
```

## Security Considerations

### PKCE (Proof Key for Code Exchange)
- **Mandatory** for native and SPA applications
- **Recommended** for all authorization code flows
- Prevents authorization code interception attacks
- Use S256 method (SHA256) over plain

### Client Authentication
- **Confidential Clients** (web apps): Use client_secret_basic or client_secret_post
- **Public Clients** (native/SPA): Use none with mandatory PKCE
- Client secrets never expire (rotate manually if compromised)

### Token Security
- Access tokens expire after 1 hour
- Refresh tokens expire after 30 days
- Tokens linked to sessions (revoked when session ends)
- Support revocation by token, session, user, or client
- JWT IDs (jti) enable revocation by ID

### Consent Management
- Persistent consent storage
- Optional consent expiration
- Trusted clients skip consent
- Consent can be revoked by user

## Migration Guide

### From Basic OIDC to Enterprise

1. **Update Schema:**
```bash
# Run migrations to add new fields and tables
authsome migrate
```

2. **Update Existing Clients:**
   - Set `application_type` based on client type
   - Configure `token_endpoint_auth_method`
   - Enable `require_pkce` for public clients
   - Set `require_consent` based on trust level

3. **Update Client Applications:**
   - Implement PKCE for public clients
   - Handle consent screens
   - Use introspection for resource servers
   - Implement token revocation on logout

## Advanced Features

### Session-Based Token Lifecycle
Tokens are linked to user sessions. When a session is terminated:
- All associated tokens are automatically revoked
- Prevents orphaned tokens after logout

### Org-Specific Configurations
Organizations can override app-level OAuth clients:
```
App Level: Default Google OAuth (app client_id)
  ↓
Org Level: Custom Google OAuth (org client_id)
```

Resolution order:
1. Try org-specific client first
2. Fall back to app-level client
3. Return not found if neither exists

### Token Revocation Strategies
- **By Token**: Revoke specific access/refresh token
- **By JWT ID**: Revoke by jti claim
- **By Session**: Cascade revoke all session tokens
- **By User**: Revoke all user tokens in org
- **By Client**: Revoke all client tokens

## Testing

```bash
# Run OIDC provider tests
go test -v ./plugins/oidcprovider/...

# Test with coverage
go test -v -cover ./plugins/oidcprovider/...

# Integration tests
go test -v -tags=integration ./plugins/oidcprovider/...
```

## Troubleshooting

### Common Issues

**"Invalid redirect_uri"**
- Ensure redirect URI is exactly registered (including trailing slashes)
- Check for HTTPS requirement (except localhost)

**"PKCE required for this client"**
- Client configured with `require_pkce: true`
- Must provide `code_challenge` and `code_challenge_method` in authorize request
- Must provide `code_verifier` in token request

**"Invalid client credentials"**
- Check client authentication method matches client configuration
- For Basic auth: Properly encode credentials in Authorization header
- For POST: Include client_id and client_secret in request body

**"Token expired or revoked"**
- Access tokens expire after 1 hour
- Use refresh token to obtain new access token
- Check if session was terminated

## License

Copyright (c) 2025 AuthSome. All rights reserved.
