# Social OAuth Plugin

Enterprise-grade social authentication plugin for AuthSome, providing OAuth 2.0 integration with popular identity providers.

## Features

✅ **Multi-Provider Support**
- Google, GitHub, Facebook, Microsoft, Apple, and more
- Extensible provider system
- Dynamic provider configuration per app

✅ **Account Management**
- OAuth sign-in and sign-up
- Account linking (multiple providers per user)
- Account unlinking
- Provider discovery

✅ **Enterprise Features**
- Redis-backed OAuth state storage
- Distributed rate limiting
- Comprehensive audit logging
- Multi-tenancy support (apps & orgs)
- CSRF protection via state tokens
- Email verification checks

✅ **Production Ready**
- Type-safe request/response handling
- Graceful error handling
- Connection pooling
- Horizontal scaling support
- Zero-downtime configuration updates

## Quick Start

### 1. Configure Providers

```yaml
auth:
  social:
    baseUrl: "https://your-domain.com"
    allowAccountLinking: true
    autoCreateUser: true
    requireEmailVerified: false
    trustEmailVerified: true
    
    # State storage configuration
    stateStorage:
      useRedis: true
      redisAddr: "localhost:6379"
      redisPassword: ""
      redisDb: 0
      stateTtl: "15m"
    
    # Provider configurations
    providers:
      google:
        clientId: "${GOOGLE_CLIENT_ID}"
        clientSecret: "${GOOGLE_CLIENT_SECRET}"
        redirectUrl: "https://your-domain.com/api/auth/callback/google"
        scopes: ["email", "profile"]
        enabled: true
        
      github:
        clientId: "${GITHUB_CLIENT_ID}"
        clientSecret: "${GITHUB_CLIENT_SECRET}"
        redirectUrl: "https://your-domain.com/api/auth/callback/github"
        scopes: ["user:email"]
        enabled: true
```

### 2. Register Plugin

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/social"
    "github.com/xraph/authsome/plugins/social/providers"
)

// Create AuthSome instance
auth := authsome.New(db, config)

// Create and register social plugin
socialPlugin := social.NewPlugin(
    social.WithProvider("google", 
        os.Getenv("GOOGLE_CLIENT_ID"),
        os.Getenv("GOOGLE_CLIENT_SECRET"),
        "https://your-domain.com/api/auth/callback/google",
        []string{"email", "profile"},
    ),
    social.WithAutoCreateUser(true),
    social.WithAllowLinking(true),
    social.WithTrustEmailVerified(true),
)

auth.RegisterPlugin(socialPlugin)
```

### 3. Use in Application

#### Sign In with Google
```bash
POST /api/auth/signin/social
Content-Type: application/json

{
  "provider": "google",
  "scopes": ["email", "profile"],
  "redirectUrl": "https://app.example.com/auth/callback"
}

# Response
{
  "url": "https://accounts.google.com/o/oauth2/v2/auth?client_id=..."
}
```

#### OAuth Callback
```bash
GET /api/auth/callback/google?code=xxx&state=yyy

# Response
{
  "user": {
    "id": "c9h7b3j2k1m4n5p6",
    "email": "user@example.com",
    "emailVerified": true,
    "name": "John Doe"
  },
  "isNewUser": true,
  "action": "signup"
}
```

#### Link Account
```bash
POST /api/auth/account/link
Content-Type: application/json
X-User-ID: c9h7b3j2k1m4n5p6

{
  "provider": "github",
  "scopes": ["user:email"]
}

# Response
{
  "url": "https://github.com/login/oauth/authorize?..."
}
```

#### Unlink Account
```bash
DELETE /api/auth/account/unlink/github
X-User-ID: c9h7b3j2k1m4n5p6

# Response
{
  "message": "Account unlinked successfully"
}
```

#### List Providers
```bash
GET /api/auth/providers

# Response
{
  "providers": ["google", "github", "facebook"]
}
```

## Architecture

### State Management

OAuth state tokens are stored for CSRF protection and callback validation:

- **Development**: In-memory storage (single instance)
- **Production**: Redis-backed storage (distributed)
- **TTL**: 15 minutes (configurable)
- **One-time use**: State deleted after verification

```go
type StateStore interface {
    Set(ctx context.Context, key string, state *OAuthState, ttl time.Duration) error
    Get(ctx context.Context, key string) (*OAuthState, error)
    Delete(ctx context.Context, key string) error
}
```

### Rate Limiting

Distributed rate limiting via Redis:

| Action | Limit | Window |
|--------|-------|--------|
| `oauth_signin` | 10 requests | 1 minute |
| `oauth_callback` | 20 requests | 1 minute |
| `oauth_link` | 5 requests | 1 minute |
| `oauth_unlink` | 5 requests | 1 minute |

Rate limits are customizable:
```go
rateLimiter.SetLimit("oauth_signin", 20, 2*time.Minute)
```

### Audit Logging

Comprehensive audit events:
- `social_signin_initiated` - OAuth flow started
- `social_callback_received` - Callback received
- `social_state_invalid` - Invalid/expired state
- `social_token_exchange_success` - Token exchanged
- `social_userinfo_fetched` - User info retrieved
- `social_link_initiated` - Account linking started
- `social_provider_not_found` - Provider not configured

## Supported Providers

### Built-in Providers
- ✅ Google
- ✅ GitHub
- ✅ Facebook
- ✅ Microsoft
- ✅ Apple
- ✅ Twitter/X
- ✅ LinkedIn
- ✅ Discord

### Adding Custom Providers

```go
package providers

import (
    "context"
    "golang.org/x/oauth2"
)

type CustomProvider struct {
    config *oauth2.Config
}

func NewCustomProvider(config *ProviderConfig) Provider {
    return &CustomProvider{
        config: &oauth2.Config{
            ClientID:     config.ClientID,
            ClientSecret: config.ClientSecret,
            RedirectURL:  config.RedirectURL,
            Scopes:       config.Scopes,
            Endpoint: oauth2.Endpoint{
                AuthURL:  "https://provider.com/oauth/authorize",
                TokenURL: "https://provider.com/oauth/token",
            },
        },
    }
}

func (p *CustomProvider) GetOAuth2Config() *oauth2.Config {
    return p.config
}

func (p *CustomProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
    // Fetch user info from provider API
    // ...
    return &UserInfo{
        ProviderUserID: "...",
        Email:          "...",
        EmailVerified:  true,
        Name:           "...",
        Picture:        "...",
    }, nil
}
```

## Configuration Options

### Plugin Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `baseUrl` | string | `http://localhost:3000` | Base URL for OAuth callbacks |
| `allowAccountLinking` | bool | `true` | Allow users to link multiple providers |
| `autoCreateUser` | bool | `true` | Auto-create user on OAuth sign-in |
| `requireEmailVerified` | bool | `false` | Require email verification from provider |
| `trustEmailVerified` | bool | `true` | Trust email verification from provider |

### State Storage Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `useRedis` | bool | `false` | Use Redis for state storage |
| `redisAddr` | string | `localhost:6379` | Redis server address |
| `redisPassword` | string | `""` | Redis password |
| `redisDb` | int | `0` | Redis database number |
| `stateTtl` | duration | `15m` | State expiration time |

### Provider Configuration

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `clientId` | string | ✅ | OAuth client ID |
| `clientSecret` | string | ✅ | OAuth client secret |
| `redirectUrl` | string | ✅ | OAuth callback URL |
| `scopes` | []string | ❌ | OAuth scopes (default: provider-specific) |
| `enabled` | bool | ❌ | Enable/disable provider (default: `true`) |

## Security

### CSRF Protection
- Crypto-secure random state tokens (32 bytes)
- One-time use state tokens
- State expiration (15 minutes)
- State-provider binding validation

### Rate Limiting
- Distributed via Redis
- Per-IP rate limiting
- Configurable limits per action
- Graceful degradation

### Email Verification
- Configurable email verification checks
- Trust provider email verification
- Optional manual verification flow

### Audit Logging
- All OAuth events logged
- User attribution
- Error tracking
- Compliance support

## Testing

```bash
# Run all tests
go test ./plugins/social/... -v

# Run with coverage
go test ./plugins/social/... -cover

# Run specific test
go test ./plugins/social/... -run TestService_ListProviders -v

# Run rate limiter tests
go test ./plugins/social/... -run TestRateLimiter -v
```

### Test Coverage

- ✅ Request/response serialization
- ✅ State store operations (memory & Redis)
- ✅ State expiration
- ✅ Rate limiting
- ✅ Configuration validation
- ✅ Handler logic
- ✅ Mock implementations

## Performance

### Benchmarks

```bash
go test ./plugins/social/... -bench=. -benchmem
```

### Optimization Tips

1. **Use Redis for state storage** in production
2. **Enable rate limiting** to prevent abuse
3. **Configure connection pooling** for Redis
4. **Use async audit logging** for high-volume scenarios
5. **Cache provider configurations** per app

## Troubleshooting

### Common Issues

#### "state not found or expired"
- Check Redis connectivity
- Verify `stateTtl` configuration
- Check clock synchronization across servers

#### "rate limit exceeded"
- Increase rate limits in configuration
- Check Redis for stuck keys
- Verify client IP extraction

#### "provider not configured"
- Verify provider credentials
- Check provider `enabled` flag
- Verify app-specific configuration

### Debug Mode

Enable debug logging:
```go
socialPlugin.SetLogLevel("debug")
```

## Migration from v1

See [MIGRATION_SUMMARY.md](./MIGRATION_SUMMARY.md) for detailed migration guide.

## License

See [LICENSE](../../LICENSE) file in the root directory.

## Support

- **Documentation**: https://docs.authsome.dev/plugins/social
- **GitHub Issues**: https://github.com/xraph/authsome/issues
- **Discord**: https://discord.gg/authsome

## Contributing

Contributions welcome! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md) for version history.
