# OIDC Provider Plugin

The OIDC Provider plugin enables AuthSome to act as an OpenID Connect Provider, allowing other applications to authenticate users through AuthSome.

## Features

- Full OpenID Connect Provider implementation
- JWT token signing with RSA keys
- JWKS (JSON Web Key Set) endpoint
- Authorization Code flow with PKCE support
- Consent screen for user authorization
- Configurable token expiry times
- Key rotation support
- File-based or auto-generated RSA keys

## Configuration

### Basic Configuration

```yaml
auth:
  oidcprovider:
    # The issuer URL for the OIDC Provider
    issuer: "https://auth.example.com"
    
    # Token expiry settings
    tokens:
      accessTokenExpiry: "1h"      # Access token lifetime
      idTokenExpiry: "1h"          # ID token lifetime  
      refreshTokenExpiry: "720h"   # Refresh token lifetime (30 days)
```

### Key Configuration

#### Option 1: File-based Keys (Recommended for Production)

```yaml
auth:
  oidcprovider:
    keys:
      # Path to RSA private key file (PEM format)
      privateKeyPath: "/path/to/private-key.pem"
      # Path to RSA public key file (PEM format)
      publicKeyPath: "/path/to/public-key.pem"
      # Key rotation settings
      rotationInterval: "24h"      # How often to rotate keys
      keyLifetime: "168h"          # How long to keep keys valid (7 days)
```

#### Option 2: Auto-generated Keys (Development Only)

If no key paths are specified, the plugin will automatically generate RSA keys in memory. This is suitable for development but not recommended for production.

```yaml
auth:
  oidcprovider:
    keys:
      rotationInterval: "24h"
      keyLifetime: "168h"
```

### Organization-specific Configuration (SaaS Mode)

```yaml
# Organization-specific overrides
orgs:
  org_123:
    auth:
      oidcprovider:
        issuer: "https://org123.auth.example.com"
        keys:
          privateKeyPath: "/path/to/org123-private-key.pem"
          publicKeyPath: "/path/to/org123-public-key.pem"
```

## Key Management

### Generating RSA Keys

Use the provided script to generate RSA key pairs:

```bash
./scripts/generate-keys.sh
```

This creates:
- `./keys/oidc-private.pem` - RSA private key (2048-bit)
- `./keys/oidc-public.pem` - RSA public key

### Manual Key Generation

You can also generate keys manually using OpenSSL:

```bash
# Generate private key
openssl genrsa -out private-key.pem 2048

# Extract public key
openssl rsa -in private-key.pem -pubout -out public-key.pem
```

### Key Formats

The plugin supports RSA keys in PEM format with the following encodings:
- **Private Keys**: PKCS#1 (`-----BEGIN RSA PRIVATE KEY-----`) or PKCS#8 (`-----BEGIN PRIVATE KEY-----`)
- **Public Keys**: PKCS#1 (`-----BEGIN RSA PUBLIC KEY-----`) or X.509 SubjectPublicKeyInfo (`-----BEGIN PUBLIC KEY-----`)

### Key Rotation

Keys are automatically rotated based on the `rotationInterval` setting. During rotation:
1. A new key pair is generated
2. The new key becomes the active signing key
3. Old keys remain valid for verification until they expire
4. Expired keys are automatically cleaned up

## Security Considerations

### Production Deployment

1. **Use File-based Keys**: Always use file-based keys in production
2. **Secure Key Storage**: Store private keys securely with appropriate file permissions (600)
3. **Key Rotation**: Enable automatic key rotation for enhanced security
4. **HTTPS Only**: Always use HTTPS for the issuer URL
5. **Backup Keys**: Maintain secure backups of your private keys

### Development

- Auto-generated keys are acceptable for development
- Never commit private keys to version control
- Use the provided `.gitignore` patterns to exclude key files

## Endpoints

The plugin registers the following endpoints:

- `GET /.well-known/openid-configuration` - OpenID Connect Discovery
- `GET /auth/authorize` - Authorization endpoint
- `POST /auth/token` - Token endpoint
- `GET /auth/jwks` - JSON Web Key Set
- `GET /auth/userinfo` - User info endpoint
- `GET /consent` - Consent screen (HTML)
- `POST /consent` - Consent form submission

## Client Registration

Register OAuth clients programmatically:

```go
client, err := oidcService.RegisterClient(ctx, "My App", "https://myapp.com/callback")
```

Or use the client management endpoints (when available).

## Testing

Run the plugin tests:

```bash
go test ./plugins/oidcprovider -v
```

Test with real key files:

```bash
# Generate test keys first
./scripts/generate-keys.sh

# Run tests
go test ./plugins/oidcprovider -v -run TestNewJWKSServiceFromFiles
```

## Integration Example

```go
// Initialize AuthSome with OIDC Provider
auth := authsome.New(
    authsome.WithDatabase(db),
    authsome.WithForgeConfig(configManager),
    authsome.WithPlugins(
        oidcprovider.NewPlugin(),
    ),
)

// Mount to Forge app
app := forge.New()
auth.Mount(app, "/")
```

## Troubleshooting

### Common Issues

1. **Key Loading Errors**
   - Verify file paths are correct
   - Check file permissions (private key should be readable)
   - Ensure keys are in valid PEM format

2. **Token Validation Failures**
   - Check that the issuer URL matches your configuration
   - Verify JWKS endpoint is accessible
   - Ensure system time is synchronized

3. **Authorization Errors**
   - Verify client registration
   - Check redirect URI configuration
   - Ensure proper scope configuration

### Debug Mode

Enable debug logging to troubleshoot issues:

```yaml
logging:
  level: debug
  components:
    - oidcprovider
```

## Standards Compliance

This plugin implements:
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [OAuth 2.0 Authorization Code Flow](https://tools.ietf.org/html/rfc6749#section-4.1)
- [PKCE (RFC 7636)](https://tools.ietf.org/html/rfc7636)
- [JSON Web Key Set (RFC 7517)](https://tools.ietf.org/html/rfc7517)
- [JSON Web Token (RFC 7519)](https://tools.ietf.org/html/rfc7519)