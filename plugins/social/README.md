# Social OAuth Plugin

Comprehensive social authentication plugin for AuthSome, supporting 17+ OAuth providers including Google, GitHub, Microsoft, Apple, Facebook, Discord, and more.

## Supported Providers

- ✅ **Google** - OAuth 2.0 with refresh tokens
- ✅ **GitHub** - OAuth 2.0
- ✅ **Microsoft** - Azure AD OAuth 2.0
- ✅ **Apple** - Sign in with Apple
- ✅ **Facebook** - OAuth 2.0
- ✅ **Discord** - OAuth 2.0
- ✅ **Twitter/X** - OAuth 2.0
- ✅ **LinkedIn** - OAuth 2.0
- ✅ **Spotify** - OAuth 2.0
- ✅ **Twitch** - OAuth 2.0
- ✅ **Dropbox** - OAuth 2.0
- ✅ **GitLab** - OAuth 2.0
- ✅ **LINE** - OAuth 2.0 / OpenID Connect
- ✅ **Reddit** - OAuth 2.0
- ✅ **Slack** - OAuth 2.0
- ✅ **Bitbucket** - OAuth 2.0
- ✅ **Notion** - OAuth 2.0

## Features

- 🔐 **Production-ready** - Complete OAuth 2.0 implementation
- 🏢 **Multi-tenant** - Organization-scoped provider configurations
- 🔗 **Account Linking** - Link multiple providers to one user
- 🔄 **Token Refresh** - Automatic token refresh for supported providers
- 📊 **Scope Management** - Dynamic scope requests (like better-auth)
- 🎯 **ID Token Support** - Direct sign-in with ID tokens (Google One Tap)
- 🛡️ **Security** - CSRF protection via state parameter

## Installation

The social plugin is included in AuthSome. Simply enable it in your configuration:

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/social"
)

auth := authsome.New(
    authsome.WithDatabase(db),
    authsome.WithPlugins(
        social.NewPlugin(),
    ),
)
```

## Configuration

### YAML Configuration

```yaml
auth:
  social:
    baseUrl: "https://your-app.com"
    allowAccountLinking: true
    autoCreateUser: true
    requireEmailVerified: false
    providers:
      google:
        enabled: true
        clientId: "${GOOGLE_CLIENT_ID}"
        clientSecret: "${GOOGLE_CLIENT_SECRET}"
        accessType: "offline"  # For refresh tokens
        prompt: "select_account consent"
      github:
        enabled: true
        clientId: "${GITHUB_CLIENT_ID}"
        clientSecret: "${GITHUB_CLIENT_SECRET}"
      microsoft:
        enabled: true
        clientId: "${MICROSOFT_CLIENT_ID}"
        clientSecret: "${MICROSOFT_CLIENT_SECRET}"
```

### Programmatic Configuration

```go
config := social.Config{
    BaseURL:              "https://your-app.com",
    AllowAccountLinking:  true,
    AutoCreateUser:       true,
    RequireEmailVerified: false,
    Providers: social.ProvidersConfig{
        Google: &providers.ProviderConfig{
            ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
            ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
            Enabled:      true,
            AccessType:   "offline",
            Prompt:       "select_account consent",
        },
        GitHub: &providers.ProviderConfig{
            ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
            ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
            Enabled:      true,
        },
    },
}

plugin := social.NewPlugin()
plugin.SetConfig(config)
```

## Provider Setup Instructions

### Google

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Navigate to **APIs & Services > Credentials**
4. Click **Create Credentials > OAuth 2.0 Client ID**
5. Set application type to **Web application**
6. Add authorized redirect URI: `https://your-app.com/api/auth/callback/google`
7. Copy the Client ID and Client Secret

**Scopes:**
- `openid` - Required for OIDC
- `https://www.googleapis.com/auth/userinfo.email` - Email address
- `https://www.googleapis.com/auth/userinfo.profile` - Profile info

**Getting Refresh Tokens:**
```yaml
google:
  accessType: "offline"
  prompt: "select_account consent"
```

### GitHub

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click **New OAuth App**
3. Set **Authorization callback URL**: `https://your-app.com/api/auth/callback/github`
4. Copy the Client ID and generate a Client Secret

**Scopes:**
- `user:email` - Read user email
- `read:user` - Read user profile

### Microsoft

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to **Azure Active Directory > App registrations**
3. Click **New registration**
4. Set redirect URI: `https://your-app.com/api/auth/callback/microsoft`
5. Go to **Certificates & secrets** and create a new client secret
6. Note the Application (client) ID and client secret

**Scopes:**
- `User.Read` - Read user profile
- `openid` - OpenID Connect
- `profile` - Profile info
- `email` - Email address

### Apple

1. Go to [Apple Developer Portal](https://developer.apple.com/account/)
2. Navigate to **Certificates, Identifiers & Profiles**
3. Create a new **Services ID**
4. Enable **Sign in with Apple**
5. Configure domains and redirect URLs: `https://your-app.com/api/auth/callback/apple`
6. Create a **Key** for Sign in with Apple
7. Download the private key and note the Key ID and Team ID

**Note:** Apple requires JWT-based client secrets. Use the `GenerateAppleClientSecret` helper.

### Facebook

1. Go to [Facebook Developers](https://developers.facebook.com/)
2. Create a new app
3. Add **Facebook Login** product
4. Configure **Valid OAuth Redirect URIs**: `https://your-app.com/api/auth/callback/facebook`
5. Copy the App ID and App Secret

**Scopes:**
- `email` - User email
- `public_profile` - Public profile info

### Discord

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Navigate to **OAuth2**
4. Add redirect: `https://your-app.com/api/auth/callback/discord`
5. Copy the Client ID and Client Secret

**Scopes:**
- `identify` - User profile
- `email` - User email

### Twitter/X

1. Go to [Twitter Developer Portal](https://developer.twitter.com/en/portal/dashboard)
2. Create a new app
3. Navigate to **User authentication settings**
4. Set **Callback URI**: `https://your-app.com/api/auth/callback/twitter`
5. Copy the Client ID and Client Secret

**Note:** Use OAuth 2.0, not OAuth 1.0a

### LinkedIn

1. Go to [LinkedIn Developers](https://www.linkedin.com/developers/)
2. Create a new app
3. Add **Sign In with LinkedIn** product
4. Set redirect URLs: `https://your-app.com/api/auth/callback/linkedin`
5. Copy the Client ID and Client Secret

**Scopes:**
- `r_liteprofile` - Basic profile
- `r_emailaddress` - Email address

### Spotify

1. Go to [Spotify Dashboard](https://developer.spotify.com/dashboard/)
2. Create a new app
3. Set **Redirect URIs**: `https://your-app.com/api/auth/callback/spotify`
4. Copy the Client ID and Client Secret

### Twitch

1. Go to [Twitch Developers](https://dev.twitch.tv/console/apps)
2. Register a new application
3. Set **OAuth Redirect URLs**: `https://your-app.com/api/auth/callback/twitch`
4. Copy the Client ID and Client Secret

### Other Providers

Similar setup steps apply for:
- **Dropbox**: [Dropbox App Console](https://www.dropbox.com/developers/apps)
- **GitLab**: [GitLab Applications](https://gitlab.com/-/profile/applications)
- **LINE**: [LINE Developers](https://developers.line.biz/)
- **Reddit**: [Reddit Apps](https://www.reddit.com/prefs/apps)
- **Slack**: [Slack Apps](https://api.slack.com/apps)
- **Bitbucket**: [Bitbucket OAuth](https://confluence.atlassian.com/bitbucket/oauth-on-bitbucket-cloud-238027431.html)
- **Notion**: [Notion Integrations](https://www.notion.so/my-integrations)

## API Endpoints

### Sign In with Social Provider

**Request:**
```http
POST /api/auth/signin/social
Content-Type: application/json

{
  "provider": "google",
  "scopes": ["additional-scope"]  // Optional
}
```

**Response:**
```json
{
  "url": "https://accounts.google.com/o/oauth2/v2/auth?..."
}
```

### OAuth Callback

**Request:**
```http
GET /api/auth/callback/google?code=...&state=...
```

**Response:**
```json
{
  "user": {
    "id": "...",
    "email": "user@example.com",
    "name": "John Doe"
  },
  "isNewUser": true,
  "action": "signup"
}
```

### Link Social Account

**Request:**
```http
POST /api/auth/account/link
Authorization: Bearer <session-token>
Content-Type: application/json

{
  "provider": "github",
  "scopes": ["repo"]  // Optional: request additional scopes
}
```

### Unlink Social Account

**Request:**
```http
DELETE /api/auth/account/unlink/github
Authorization: Bearer <session-token>
```

### List Available Providers

**Request:**
```http
GET /api/auth/providers
```

**Response:**
```json
{
  "providers": ["google", "github", "microsoft", "discord"]
}
```

## Admin Endpoints

Requires admin role and `social:admin` permission.

### List Configured Providers

**Request:**
```http
GET /social/admin/providers
Authorization: Bearer <admin-token>
```

**Response:**
```json
{
  "providers": ["google", "github", "microsoft", "discord"],
  "appId": "app_123"
}
```

### Add/Configure Provider

**Request:**
```http
POST /social/admin/providers
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "appId": "app_123",
  "provider": "google",
  "clientId": "your-client-id",
  "clientSecret": "your-client-secret",
  "scopes": ["openid", "email", "profile"],
  "enabled": true
}
```

**Response:**
```json
{
  "message": "Provider configured successfully",
  "provider": "google",
  "appId": "app_123"
}
```

### Update Provider Configuration

**Request:**
```http
PUT /social/admin/providers/google
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "clientId": "new-client-id",
  "clientSecret": "new-client-secret",
  "enabled": true
}
```

**Response:**
```json
{
  "message": "Provider updated successfully",
  "provider": "google",
  "appId": "app_123"
}
```

### Remove Provider Configuration

**Request:**
```http
DELETE /social/admin/providers/google
Authorization: Bearer <admin-token>
```

**Response:**
```json
{
  "message": "Provider removed successfully",
  "provider": "google",
  "appId": "app_123"
}
```

**Note:** Admin endpoints are currently placeholders. Full implementation requires:
- Database schema for app-specific provider configurations
- RBAC integration for permission checks
- Credential encryption for secure storage
- Audit logging for administrative actions

See [Plugin Admin Endpoint Guidelines](../../docs/PLUGIN_ADMIN_ENDPOINTS.md) for implementation details.

## Usage Examples

### Client-Side (React/Next.js)

```typescript
// Sign in with Google
const signInWithGoogle = async () => {
  const response = await fetch('/api/auth/signin/social', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ provider: 'google' })
  });
  
  const { url } = await response.json();
  window.location.href = url; // Redirect to Google
};

// Link GitHub account (for logged-in users)
const linkGitHub = async () => {
  const response = await fetch('/api/auth/account/link', {
    method: 'POST',
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${sessionToken}`
    },
    body: JSON.stringify({ 
      provider: 'github',
      scopes: ['repo', 'read:org'] // Request additional scopes
    })
  });
  
  const { url } = await response.json();
  window.location.href = url;
};
```

### Go Client

```go
// Get authorization URL
authURL, err := socialService.GetAuthorizationURL(
    ctx,
    "google",
    organizationID,
    []string{"https://www.googleapis.com/auth/drive.file"}, // Extra scopes
)

// Handle callback
result, err := socialService.HandleCallback(ctx, "google", state, code)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("User: %+v\n", result.User)
fmt.Printf("Is new user: %v\n", result.IsNewUser)
fmt.Printf("Action: %s\n", result.Action) // "signin", "signup", or "linked"
```

## Multi-Tenancy

The social plugin is multi-tenant aware. Organizations can have their own OAuth configurations:

```yaml
# Default configuration
auth:
  social:
    providers:
      google:
        clientId: "default-client-id"

# Organization-specific override
orgs:
  org_acme:
    auth:
      social:
        providers:
          google:
            clientId: "acme-specific-client-id"
            clientSecret: "acme-specific-secret"
```

## Security Considerations

1. **State Parameter**: CSRF protection via cryptographically secure random state
2. **HTTPS Only**: Always use HTTPS in production
3. **Token Storage**: Tokens are encrypted at rest in the database
4. **Scope Validation**: Validate requested scopes against allowed list
5. **Email Verification**: Option to require verified emails from providers
6. **Account Linking**: Optional - can be disabled for security

## Database Schema

```sql
CREATE TABLE social_accounts (
    id VARCHAR(20) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_id VARCHAR(20) NOT NULL,
    organization_id VARCHAR(20) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    name VARCHAR(255),
    avatar TEXT,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(50) DEFAULT 'Bearer',
    expires_at TIMESTAMP,
    refresh_expires_at TIMESTAMP,
    scope TEXT,
    id_token TEXT,
    raw_user_info JSONB,
    revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMP
);

CREATE INDEX idx_social_accounts_user_id ON social_accounts(user_id);
CREATE INDEX idx_social_accounts_provider ON social_accounts(provider);
CREATE INDEX idx_social_accounts_provider_provider_id_org ON social_accounts(provider, provider_id, organization_id);
```

## Testing

Run the migration:
```bash
./authsome migrate
```

Test with a provider:
```bash
# Set environment variables
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"

# Start server
go run cmd/dev/main.go

# Open browser to http://localhost:3000/api/auth/signin/social
```

## Roadmap

- [ ] OAuth 1.0a support (legacy Twitter)
- [ ] PKCE support for public clients
- [ ] Token refresh automation
- [ ] Redis-based state storage (for distributed systems)
- [ ] Webhooks for provider events
- [ ] Admin UI for provider configuration
- [ ] Provider health monitoring

## Troubleshooting

### "Provider not configured"
Ensure the provider is enabled in your configuration and environment variables are set.

### "State not found" or "State expired"
State tokens expire after 15 minutes. In production, use Redis for distributed state storage.

### "Email not verified"
If `requireEmailVerified: true`, the provider must confirm email verification. Some providers (Twitter) don't provide email verification status.

### "User with email already exists"
If `allowAccountLinking: false`, users cannot link providers to existing accounts. Enable it or handle manually.

## License

Part of the AuthSome project. See main LICENSE file.

## Contributing

Contributions welcome! To add a new provider:

1. Create `providers/<provider>.go`
2. Implement the `Provider` interface
3. Add configuration to `ProvidersConfig`
4. Initialize in `service.go`
5. Add setup instructions to this README
6. Submit PR

---

**Need help?** Check the [main AuthSome documentation](../../README.md) or open an issue.

