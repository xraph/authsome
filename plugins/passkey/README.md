# Passkey Plugin

✅ **PRODUCTION READY** - Full WebAuthn/FIDO2 Implementation

The passkey plugin provides production-ready WebAuthn/FIDO2 passwordless authentication with cryptographic verification, supporting both standalone authentication and MFA integration.

## Status

**Production Ready** - This plugin now includes:
- ✅ Complete WebAuthn cryptographic implementation
- ✅ Challenge generation with secure randomness
- ✅ Attestation verification during registration
- ✅ Signature verification during authentication
- ✅ Sign count tracking for replay attack prevention
- ✅ Resident key / discoverable credential support
- ✅ Passkey naming and management
- ✅ App and organization scoping
- ✅ MFA integration support
- ✅ Comprehensive test coverage

## Features

### Core WebAuthn Support
- **Platform Authenticators**: Touch ID, Windows Hello, Face ID
- **Cross-Platform Authenticators**: YubiKey, Google Titan, Feitian
- **Discoverable Credentials**: Usernameless authentication
- **User Verification**: Biometric or PIN requirements
- **Attestation Formats**: Packed, FIDO U2F, TPM, Android SafetyNet

### Security
- Cryptographic challenge generation (32+ bytes)
- Public key storage and signature verification
- Sign count tracking for cloned authenticator detection
- Challenge timeout (5 minutes default)
- Replay attack prevention
- App and organization scoping

### Device Management
- User-friendly passkey naming
- Last used timestamp tracking
- Authenticator type identification (platform vs cross-platform)
- AAGUID for hardware key identification
- Multiple passkeys per user

## Installation

Add the passkey plugin to your AuthSome setup:

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/passkey"
)

func main() {
    // Create AuthSome instance
    auth := authsome.New(
        db,
        authsome.WithPlugins(
            passkey.NewPlugin(
                passkey.WithRPID("example.com"),
                passkey.WithRPName("My App"),
                passkey.WithTimeout(300000), // 5 minutes
                passkey.WithUserVerification("preferred"),
            ),
        ),
    )
}
```

## Configuration

### YAML Configuration

```yaml
auth:
  passkey:
    rpid: "example.com"              # Relying Party ID (your domain)
    rpname: "My Application"         # Display name
    rporigins:                       # Allowed origins
      - "https://example.com"
      - "https://app.example.com"
    timeout: 300000                  # Challenge timeout (milliseconds)
    userverification: "preferred"    # required, preferred, or discouraged
    attestationtype: "none"          # none, indirect, or direct
    requireresidentkey: false        # Require resident keys
    authenticatorattachment: ""      # platform, cross-platform, or empty
    challengestorage: "memory"       # memory or redis
```

### Programmatic Configuration

```go
plugin := passkey.NewPlugin(
    passkey.WithRPID("example.com"),
    passkey.WithRPName("My App"),
    passkey.WithTimeout(300000),
    passkey.WithUserVerification("required"),
    passkey.WithAttestationType("direct"),
)
```

## Usage

### Standalone Passwordless Authentication

#### Registration Flow

```go
// 1. Begin registration
POST /auth/passkey/register/begin
{
  "userId": "cjld2cjxh0000qzrmn831i7rn",
  "name": "MacBook Pro Touch ID",
  "authenticatorType": "platform",  // optional: "platform" or "cross-platform"
  "requireResidentKey": false,
  "userVerification": "preferred"   // optional: "required", "preferred", "discouraged"
}

// Response:
{
  "options": { /* WebAuthn PublicKeyCredentialCreationOptions */ },
  "challenge": "base64url_encoded_challenge",
  "userId": "cjld2cjxh0000qzrmn831i7rn",
  "timeout": 300000
}

// 2. Client calls navigator.credentials.create() with options
// 3. Finish registration with credential response

POST /auth/passkey/register/finish
{
  "userId": "cjld2cjxh0000qzrmn831i7rn",
  "name": "MacBook Pro Touch ID",
  "response": { /* WebAuthn PublicKeyCredential */ }
}

// Response:
{
  "passkeyId": "cjld2cjxh0001qzrmn831i7rn",
  "name": "MacBook Pro Touch ID",
  "status": "registered",
  "createdAt": "2025-01-15T10:30:00Z",
  "credentialId": "base64url_credential_id"
}
```

#### Authentication Flow

```go
// 1. Begin authentication
POST /auth/passkey/login/begin
{
  "userId": "cjld2cjxh0000qzrmn831i7rn",  // optional for discoverable credentials
  "userVerification": "preferred"
}

// Response:
{
  "options": { /* WebAuthn PublicKeyCredentialRequestOptions */ },
  "challenge": "base64url_encoded_challenge",
  "timeout": 300000
}

// 2. Client calls navigator.credentials.get() with options
// 3. Finish authentication with assertion

POST /auth/passkey/login/finish
{
  "response": { /* WebAuthn PublicKeyCredential assertion */ },
  "remember": true
}

// Response:
{
  "user": { /* user object */ },
  "session": { /* session object */ },
  "token": "session_token",
  "passkeyUsed": "cjld2cjxh0001qzrmn831i7rn"
}
```

### Discoverable Credentials (Usernameless)

```javascript
// Begin login without userID
POST /auth/passkey/login/begin
{
  // No userId - supports autofill/conditional UI
}

// Client side with conditional UI
const publicKeyCredential = await navigator.credentials.get({
  publicKey: options,
  mediation: 'conditional'  // Enables autofill
});
```

### Passkey Management

```go
// List user's passkeys
GET /auth/passkey/list?userId=cjld2cjxh0000qzrmn831i7rn

// Response:
{
  "passkeys": [
    {
      "id": "cjld2cjxh0001qzrmn831i7rn",
      "name": "MacBook Pro Touch ID",
      "credentialId": "base64url_id",
      "aaguid": "base64url_aaguid",
      "authenticatorType": "platform",
      "createdAt": "2025-01-15T10:30:00Z",
      "lastUsedAt": "2025-01-20T14:22:00Z",
      "signCount": 42,
      "isResidentKey": true
    }
  ],
  "count": 1
}

// Update passkey name
PUT /auth/passkey/:id
{
  "name": "New Name for Security Key"
}

// Delete passkey
DELETE /auth/passkey/:id
```

## MFA Integration

The passkey plugin seamlessly integrates with the MFA plugin:

```go
// Setup both plugins
auth := authsome.New(
    db,
    authsome.WithPlugins(
        passkey.NewPlugin(),
        mfa.NewPlugin(),
    ),
)

// Passkeys can then be enrolled as an MFA factor
POST /auth/mfa/factors/enroll
{
  "type": "webauthn",
  "name": "YubiKey 5",
  "metadata": {
    "authenticatorType": "cross-platform"
  }
}

// And verified during MFA challenges
POST /auth/mfa/verify
{
  "challengeId": "...",
  "factorId": "...",
  "data": {
    "credentialResponse": { /* WebAuthn assertion */ }
  }
}
```

## Client-Side Integration

### Basic JavaScript Example

```javascript
// Registration
async function registerPasskey(userId, name) {
  // 1. Get registration options
  const beginResp = await fetch('/auth/passkey/register/begin', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId, name })
  });
  const { options } = await beginResp.json();

  // 2. Create credential
  const credential = await navigator.credentials.create({
    publicKey: options
  });

  // 3. Finish registration
  const finishResp = await fetch('/auth/passkey/register/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      userId,
      name,
      response: credential
    })
  });

  return await finishResp.json();
}

// Authentication
async function loginWithPasskey(userId) {
  // 1. Get authentication options
  const beginResp = await fetch('/auth/passkey/login/begin', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId })
  });
  const { options } = await beginResp.json();

  // 2. Get credential
  const credential = await navigator.credentials.get({
    publicKey: options
  });

  // 3. Finish authentication
  const finishResp = await fetch('/auth/passkey/login/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      response: credential,
      remember: true
    })
  });

  return await finishResp.json();
}
```

### Conditional UI (Autofill)

```javascript
// Enable passkey autofill in login form
async function setupPasskeyAutofill() {
  if (!window.PublicKeyCredential?.isConditionalMediationAvailable) {
    return;
  }

  const available = await PublicKeyCredential.isConditionalMediationAvailable();
  if (!available) return;

  // Get options for discoverable login
  const resp = await fetch('/auth/passkey/login/begin', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({})  // No userId
  });
  const { options } = await resp.json();

  // Start conditional mediation
  const credential = await navigator.credentials.get({
    publicKey: options,
    mediation: 'conditional'
  });

  // Finish login
  const loginResp = await fetch('/auth/passkey/login/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      response: credential,
      remember: true
    })
  });

  const result = await loginResp.json();
  window.location.href = '/dashboard';
}

// HTML form with autocomplete
<input
  type="text"
  name="username"
  autocomplete="username webauthn"
  placeholder="Email or passkey"
/>
```

## Architecture

The passkey plugin follows clean architecture principles:

```
plugin.go           → Plugin registration and initialization
service.go          → Business logic and WebAuthn operations
handlers.go         → HTTP request handling
webauthn.go         → WebAuthn library wrapper
user_adapter.go     → WebAuthn User interface implementation
challenge_store.go  → Challenge session management
request_types.go    → Request DTOs with validation
response_types.go   → Response DTOs
```

### Multi-Tenancy

Passkeys are automatically scoped to:
- **App**: Platform tenant
- **Organization**: User-created workspace (optional)

This ensures complete data isolation in multi-tenant deployments.

## Security Considerations

### Best Practices

1. **HTTPS Required**: WebAuthn only works over HTTPS (or localhost for development)
2. **RP ID Must Match**: Set RPID to your domain (e.g., "example.com")
3. **Origins Whitelist**: Configure all allowed origins in RPOrigins
4. **User Verification**: Use "required" for sensitive operations
5. **Sign Count Tracking**: Monitor for cloned authenticators
6. **Attestation**: Use "direct" for high-security environments

### Threat Model

The passkey plugin protects against:
- ✅ Phishing (origin-bound credentials)
- ✅ Credential stuffing (no passwords)
- ✅ Replay attacks (sign count tracking)
- ✅ Man-in-the-middle (cryptographic binding)
- ✅ Cloned authenticators (sign count validation)
- ✅ Database breaches (public keys only)

## Troubleshooting

### Common Issues

**Issue**: Registration fails with "rpID mismatch"
**Solution**: Ensure RPID matches your domain exactly. For `https://app.example.com`, use `example.com` as RPID.

**Issue**: "NotAllowedError: The operation either timed out or was not allowed"
**Solution**: Check user verification requirements. For Touch ID/Windows Hello, ensure "preferred" or "discouraged" is used.

**Issue**: Challenge expired
**Solution**: Increase timeout in configuration. Default is 5 minutes (300000ms).

**Issue**: Passkeys not appearing in autofill
**Solution**: Ensure `requireResidentKey: true` during registration and use conditional mediation.

## Browser Support

| Browser | Platform Auth | Cross-Platform | Conditional UI |
|---------|---------------|----------------|----------------|
| Chrome 67+ | ✅ | ✅ | ✅ (108+) |
| Safari 14+ | ✅ | ✅ | ✅ (16+) |
| Firefox 60+ | ✅ | ✅ | ❌ |
| Edge 18+ | ✅ | ✅ | ✅ (108+) |

## Migration from Beta

If you were using the beta version, run the database migration:

```bash
# The migration adds required WebAuthn fields
./authsome-cli migrate up
```

**Note**: Existing passkey records from beta will need to be re-registered as they lack cryptographic public keys.

## References

### Specifications
- [WebAuthn Level 2](https://www.w3.org/TR/webauthn-2/)
- [WebAuthn Level 3 (Draft)](https://www.w3.org/TR/webauthn-3/)
- [FIDO2 CTAP](https://fidoalliance.org/specs/fido-v2.0-ps-20190130/fido-client-to-authenticator-protocol-v2.0-ps-20190130.html)

### Guides
- [WebAuthn Guide (MDN)](https://developer.mozilla.org/en-US/docs/Web/API/Web_Authentication_API)
- [WebAuthn Awesome List](https://github.com/herrjemand/awesome-webauthn)
- [go-webauthn Documentation](https://github.com/go-webauthn/webauthn)

### Security
- [WebAuthn Security Considerations](https://www.w3.org/TR/webauthn-2/#sctn-security-considerations)
- [FIDO Security Reference](https://fidoalliance.org/specifications/download/)

## Support

For questions, issues, or feature requests:
- **GitHub Issues**: [Report bugs or request features](https://github.com/xraph/authsome/issues)
- **Documentation**: See main AuthSome docs for additional integration examples
- **Security Issues**: Please report security vulnerabilities privately

## License

Same as main AuthSome project - see LICENSE file.

---

**Last Updated:** November 20, 2025  
**Status:** ✅ Production Ready  
**Maintainers:** AuthSome Core Team
