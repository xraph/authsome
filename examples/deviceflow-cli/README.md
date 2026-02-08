# Device Flow CLI Example

A simple command-line application demonstrating the OAuth 2.0 Device Authorization Grant (RFC 8628) flow.

## Overview

This example shows how to implement device flow authentication in a CLI application. The device flow is ideal for:

- Command-line tools
- IoT devices
- Smart TVs and streaming devices
- Any application with limited input capabilities

## How It Works

1. **Device requests authorization** - CLI calls `/oauth2/device/authorize`
2. **User code is displayed** - CLI shows the verification URL and code
3. **User authorizes** - User visits URL on another device and enters code
4. **CLI polls for tokens** - CLI continuously polls `/oauth2/token`
5. **Tokens received** - Once authorized, CLI receives access and refresh tokens

## Prerequisites

- Authsome server running with OIDC provider plugin enabled
- Device flow enabled in configuration
- A registered OAuth client

## Usage

### Run the Example

```bash
go run main.go --server http://localhost:3001 --client your_client_id --scope "openid profile email"
```

### Command-Line Flags

- `--server` - Auth server base URL (default: `http://localhost:3001`)
- `--client` - OAuth client ID (required)
- `--scope` - OAuth scope (default: `openid profile email`)

### Example Output

```
🔐 Device Flow Authentication Demo
====================================

Step 1: Requesting device code...

✅ Device authorization initiated successfully!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📱 Please visit: http://localhost:3001/device
🔢 Enter code: WDJB-MJHT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⏱️  Code expires in 600 seconds
🔄 Polling interval: 5 seconds

Direct link: http://localhost:3001/device?user_code=WDJB-MJHT

Step 3: Waiting for authorization...
(Polling every 5 seconds...)

  [1] Polling... ⏳ Pending
  [2] Polling... ⏳ Pending
  [3] Polling... ✅ Authorized!

✅ Authorization successful!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎉 Tokens Received:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Token Type: Bearer
Expires In: 3600 seconds
Scope: openid profile email

Access Token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6...
Refresh Token: refresh_abc123...
ID Token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6...

✨ You can now use these tokens to access protected resources!
```

## Implementation Details

### 1. Device Authorization Request

```go
data := url.Values{}
data.Set("client_id", clientID)
data.Set("scope", scope)

resp, err := http.Post(
    baseURL+"/oauth2/device/authorize",
    "application/x-www-form-urlencoded",
    bytes.NewBufferString(data.Encode()),
)
```

### 2. Token Polling

```go
data := url.Values{}
data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
data.Set("device_code", deviceCode)
data.Set("client_id", clientID)

resp, err := http.Post(
    baseURL+"/oauth2/token",
    "application/x-www-form-urlencoded",
    bytes.NewBufferString(data.Encode()),
)
```

### 3. Error Handling

The example handles all RFC 8628 error codes:

- `authorization_pending` - Continues polling
- `slow_down` - Increases polling interval by 5 seconds
- `expired_token` - Exits with error
- `access_denied` - User denied, exits with error

## Testing

1. Start the Authsome server with device flow enabled:

```yaml
auth:
  oidcprovider:
    deviceFlow:
      enabled: true
```

2. Register an OAuth client (or use an existing one)

3. Run the CLI example:

```bash
go run main.go --client your_client_id
```

4. Open the verification URL in a browser and enter the code

5. The CLI will automatically receive the tokens

## Production Considerations

### Token Storage

In a production CLI tool, you should:

- Store tokens securely (use OS keychain/credential manager)
- Implement automatic refresh token rotation
- Handle token expiration gracefully

### Error Handling

- Implement retry logic for network failures
- Handle server errors gracefully
- Provide clear user feedback

### User Experience

- Consider opening the verification URL automatically
- Display a QR code for mobile scanning
- Show progress indicators during polling
- Implement graceful cancellation (Ctrl+C)

### Security

- Validate TLS certificates in production
- Use PKCE even for public clients
- Clear sensitive data from memory after use
- Log authentication events for audit

## See Also

- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://tools.ietf.org/html/rfc8628)
- [Device Flow Plugin Documentation](../../plugins/oidcprovider/deviceflow/README.md)
- [OIDC Provider Plugin](../../plugins/oidcprovider/README.md)
