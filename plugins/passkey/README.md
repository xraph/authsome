# Passkey Plugin

⚠️ **EXPERIMENTAL / BETA** ⚠️

**Status:** This plugin is currently in experimental/beta status. The WebAuthn implementation is a basic stub and not production-ready.

## Current Limitations

### ❌ Not Implemented

The following critical WebAuthn features are **NOT** currently implemented:

1. **WebAuthn Challenge Generation** - Uses timestamp instead of cryptographic challenge
2. **Attestation Processing** - No verification of authenticator attestation
3. **Public Key Storage** - Credential public keys are not stored
4. **Signature Verification** - Authentication responses are not cryptographically verified
5. **Challenge-Response Validation** - No actual WebAuthn protocol validation

### ⚠️ Current Implementation

The current implementation provides:

- ✅ Basic API structure for passkey registration and login
- ✅ Database schema for passkey storage
- ✅ Audit logging for passkey events
- ✅ List and delete passkey operations
- ❌ Actual WebAuthn cryptographic operations

### Example of Stub Implementation

```go
// BeginRegistration returns a simple challenge payload (stub for WebAuthn options)
func (s *Service) BeginRegistration(_ context.Context, userID string) (map[string]any, error) {
    // In a full implementation, generate WebAuthn options and store session
    return map[string]any{
        "challenge": time.Now().UnixNano(), // ❌ NOT cryptographically secure
        "rpId":      s.config.RPID,
        "userId":    userID,
    }, nil
}
```

## Production Requirements

To make this plugin production-ready, the following must be implemented:

### 1. WebAuthn Library Integration

Integrate a proper WebAuthn library such as:
- **go-webauthn/webauthn** (recommended)
- **duo-labs/webauthn**

```go
import "github.com/go-webauthn/webauthn/webauthn"
```

### 2. Proper Challenge Generation

```go
// Generate cryptographically secure challenge
challenge, err := webauthn.CreateChallenge()
sessionData := webauthn.SessionData{
    Challenge:            challenge,
    UserID:               userID,
    UserVerification:     protocol.VerificationRequired,
    Extensions:           protocol.AuthenticationExtensions{},
}
// Store session data for verification
```

### 3. Credential Storage

Store the following for each credential:
- Credential ID (unique identifier)
- Public Key (for signature verification)
- Sign Count (replay attack prevention)
- Authenticator AAGUID
- User ID association

### 4. Attestation Verification

```go
// Verify attestation during registration
credential, err := webauthn.ParseCredentialCreationResponse(response)
if err != nil {
    return err
}

// Validate attestation
attestation := credential.Response.AttestationObject.AttStatement
// Verify attestation based on format (packed, fido-u2f, tpm, etc.)
```

### 5. Authentication Verification

```go
// Verify authentication assertion
credential, err := webauthn.ParseCredentialRequestResponse(response)
if err != nil {
    return err
}

// Verify signature using stored public key
valid := credential.Verify(storedPublicKey, clientDataHash, authenticatorData)
```

## Current Usage (Testing Only)

⚠️ **DO NOT USE IN PRODUCTION**

For testing/development purposes only:

### Registration Flow

```go
// Begin registration (returns stub challenge)
challenge, err := passkeyService.BeginRegistration(ctx, userID)

// Finish registration (stores credential ID only - no verification)
err = passkeyService.FinishRegistration(ctx, userID, credentialID, ip, userAgent)
```

### Login Flow

```go
// Begin login (returns stub challenge)
challenge, err := passkeyService.BeginLogin(ctx, userID)

// Finish login (creates session without verification)
authResponse, err := passkeyService.FinishLogin(ctx, userID, remember, ip, userAgent)
```

## Roadmap to Production

### Phase 1: Library Integration (2-3 weeks)
- [ ] Integrate go-webauthn/webauthn library
- [ ] Update schema to store public keys
- [ ] Implement proper challenge generation

### Phase 2: Registration Flow (1 week)
- [ ] Implement proper BeginRegistration with WebAuthn options
- [ ] Process and verify attestation in FinishRegistration
- [ ] Store public keys and credential metadata

### Phase 3: Authentication Flow (1 week)
- [ ] Implement proper BeginLogin with WebAuthn options
- [ ] Verify authentication assertions in FinishLogin
- [ ] Implement replay attack prevention (sign count)

### Phase 4: Security Hardening (1 week)
- [ ] Add timeout for challenges
- [ ] Implement user presence/verification requirements
- [ ] Add resident key support
- [ ] Security audit and penetration testing

### Phase 5: Testing & Documentation (1 week)
- [ ] Comprehensive unit tests
- [ ] Integration tests with real authenticators
- [ ] Browser compatibility testing
- [ ] Production deployment guide

**Total Estimated Time:** 6-8 weeks

## Security Considerations

### Current Security Issues

1. **No Cryptographic Verification** - Anyone can claim any credential
2. **Timestamp as Challenge** - Predictable, not random
3. **No Replay Protection** - Same response can be used multiple times
4. **No User Verification** - No proof that user is present
5. **Public Key Not Stored** - Cannot verify signatures

### Required Security Measures

1. **Strong Challenge Generation** - 32+ bytes of cryptographic randomness
2. **Challenge Timeout** - Challenges expire after 5 minutes
3. **Sign Count Tracking** - Detect cloned authenticators
4. **User Verification** - Require PIN/biometric on authenticator
5. **Origin Validation** - Verify requests come from expected RP
6. **HTTPS Only** - WebAuthn requires secure context

## References

### Specifications
- [WebAuthn Level 2](https://www.w3.org/TR/webauthn-2/)
- [FIDO2 CTAP](https://fidoalliance.org/specs/fido-v2.0-ps-20190130/fido-client-to-authenticator-protocol-v2.0-ps-20190130.html)

### Implementation Guides
- [go-webauthn/webauthn Documentation](https://github.com/go-webauthn/webauthn)
- [WebAuthn Guide (MDN)](https://developer.mozilla.org/en-US/docs/Web/API/Web_Authentication_API)
- [WebAuthn Awesome List](https://github.com/herrjemand/awesome-webauthn)

### Security Resources
- [WebAuthn Security Considerations](https://www.w3.org/TR/webauthn-2/#sctn-security-considerations)
- [FIDO Security Reference](https://fidoalliance.org/specifications/download/)

## Contributing

If you'd like to help implement full WebAuthn support, please:

1. Review the roadmap above
2. Check existing issues/PRs related to passkey
3. Create an issue discussing your implementation approach
4. Submit a PR with comprehensive tests

## Support

For questions or issues:

- **GitHub Issues**: Report bugs or request features
- **Documentation**: See main AuthSome docs for integration
- **Security Issues**: Email security@example.com (use GPG key)

## License

Same as main AuthSome project - see LICENSE file.

---

**Last Updated:** October 30, 2025  
**Status:** EXPERIMENTAL - Not Production Ready  
**Maintainers:** AuthSome Core Team

