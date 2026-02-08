# Device Flow (RFC 8628)

OAuth 2.0 Device Authorization Grant implementation for input-constrained devices.

## Overview

The device flow allows devices with limited input capabilities (smart TVs, IoT devices, CLI tools) to authenticate users through a secondary device with better input capabilities (like a smartphone or computer).

## How It Works

1. **Device Initiates**: Device requests a device code and user code
2. **User Receives Code**: Device displays the user code to the user
3. **User Authorizes**: User visits verification URI on another device and enters the code
4. **Device Polls**: Device polls the token endpoint until user completes authorization
5. **Tokens Issued**: Once authorized, device receives access and refresh tokens

## Configuration

Enable device flow in your OIDC provider configuration:

```yaml
auth:
  oidcprovider:
    deviceFlow:
      enabled: true              # Enable device flow
      codeExpiry: "10m"         # Device code lifetime (default: 10 minutes)
      userCodeLength: 8         # Number of characters in user code (default: 8)
      userCodeFormat: "XXXX-XXXX" # User code format (default: XXXX-XXXX)
      pollingInterval: 5        # Minimum seconds between polls (default: 5)
      verificationUri: "/device" # Verification page path (default: /device)
      maxPollAttempts: 120      # Maximum poll attempts (default: 120)
      cleanupInterval: "5m"     # Cleanup job interval (default: 5 minutes)
```

## API Endpoints

### Device Authorization

**POST** `/oauth2/device/authorize`

Initiates device authorization flow.

**Request:**
```bash
curl -X POST https://auth.example.com/oauth2/device/authorize \
  -d "client_id=your_client_id" \
  -d "scope=openid profile email"
```

**Response:**
```json
{
  "device_code": "GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS",
  "user_code": "WDJB-MJHT",
  "verification_uri": "https://auth.example.com/device",
  "verification_uri_complete": "https://auth.example.com/device?user_code=WDJB-MJHT",
  "expires_in": 600,
  "interval": 5
}
```

### Token Exchange

**POST** `/oauth2/token`

Device polls this endpoint to exchange device code for tokens.

**Request:**
```bash
curl -X POST https://auth.example.com/oauth2/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
  -d "device_code=GmRhmhcxhwAzkoEqiMEg_DnyEysNkuNhszIySk9eS" \
  -d "client_id=your_client_id"
```

**Success Response (200):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "refresh_abc123...",
  "id_token": "eyJhbGciOiJSUzI1NiIs...",
  "scope": "openid profile email"
}
```

**Pending Response (400):**
```json
{
  "error": "authorization_pending",
  "error_description": "user has not yet authorized the device"
}
```

**Slow Down Response (400):**
```json
{
  "error": "slow_down",
  "error_description": "polling too frequently, slow down by 5 seconds"
}
```

## User Verification Flow

### 1. Code Entry Page

**GET** `/device`

Shows HTML form where user enters the device code.

Optional query parameter: `user_code` (pre-fills the code)

### 2. Code Verification

**POST** `/device/verify`

Validates the user code and shows consent screen if valid.

**Request:**
- `user_code`: The code displayed on the device

### 3. Authorization Decision

**POST** `/device/authorize`

Handles user's authorization decision.

**Request:**
- `user_code`: The code being authorized
- `action`: Either `approve` or `deny`

## Error Codes

Device flow uses OAuth 2.0 error codes:

- `authorization_pending` - User hasn't authorized yet (device should continue polling)
- `slow_down` - Device is polling too frequently (add 5 seconds to interval)
- `expired_token` - Device code has expired (device must restart flow)
- `access_denied` - User denied the authorization request
- `invalid_grant` - Invalid device code or already consumed

## Security Considerations

### User Code Generation

- Uses base20 charset (BCDFGHJKLMNPQRSTVWXZ) to avoid ambiguous characters
- Collision detection with retry mechanism
- Configurable length and format

### Rate Limiting

- Enforces minimum polling interval (default 5 seconds)
- Returns `slow_down` error if device polls too frequently
- Tracks poll count and enforces maximum attempts

### Code Expiration

- Device codes expire after configured duration (default 10 minutes)
- Background job cleans up expired codes every 5 minutes
- Old consumed codes are purged after 7 days

### Brute Force Protection

- User code format makes brute force attacks impractical
- Device code is long and cryptographically secure (32 bytes)
- Single-use codes are marked as consumed after token exchange

## Implementation Details

### Database Schema

The `device_codes` table stores:

- Device code (long, secure, URL-safe)
- User code (short, human-typable)
- Client ID and scope
- Status (pending, authorized, denied, expired, consumed)
- User and session IDs (set upon authorization)
- Poll count and last polled timestamp
- PKCE support (optional code challenge)

### Status Transitions

```
pending → authorized → consumed (success)
        → denied (user rejected)
        → expired (timeout)
```

### Polling Behavior

1. Device polls every `interval` seconds
2. If too frequent, returns `slow_down` error
3. If pending, returns `authorization_pending`
4. If authorized, issues tokens and marks as consumed
5. If denied/expired, returns appropriate error

## Testing

Run device flow tests:

```bash
go test ./plugins/oidcprovider/deviceflow/...
```

### Key Test Coverage

- Code generation (uniqueness, format)
- Device code lifecycle (status transitions)
- Rate limiting (polling intervals)
- Expiration handling
- Configuration defaults

## Example Usage

See the CLI example in `/examples/deviceflow-cli/` for a complete implementation demonstrating:

- Device authorization initiation
- User code display
- Token polling with proper intervals
- Error handling
- Token refresh

## References

- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://tools.ietf.org/html/rfc8628)
- [OAuth 2.0 Best Current Practice](https://tools.ietf.org/html/draft-ietf-oauth-security-topics)
