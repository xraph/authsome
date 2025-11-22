# @authsome/client

TypeScript/JavaScript client library for AuthSome authentication framework.

## Installation

```bash
npm install @authsome/client
```

## Quick Start

```typescript
import { AuthsomeClient, mfaClient } from '@authsome/client';

// Create client with configuration
const client = new AuthsomeClient({
  baseURL: 'http://localhost:3000',
  basePath: '/api/auth',  // Optional: defaults to ''
  plugins: [mfaClient()]
});

// Sign up a new user
const { user, session } = await client.signUp({
  email: 'user@example.com',
  password: 'securepassword123',
  name: 'John Doe'
});

// Sign in
const { user, session } = await client.signIn({
  email: 'user@example.com',
  password: 'securepassword123'
});
```

## Features

### Multiple Authentication Methods

The client supports multiple authentication methods that can be used simultaneously:

- **Cookies**: Automatically sent with every request (session-based auth)
- **Bearer Token**: JWT tokens sent in Authorization header when `auth: true`
- **API Key**: Sent with every request for server-to-server auth

#### API Key Authentication

```typescript
// Frontend: Publishable key (safe to expose)
const client = new AuthsomeClient({
  baseURL: 'https://api.example.com',
  basePath: '/api/auth'
});
client.setPublishableKey('pk_your_publishable_key');

// Backend: Secret key (NEVER expose in client-side code!)
const adminClient = new AuthsomeClient({
  baseURL: 'https://api.example.com',
  basePath: '/api/auth'
});
adminClient.setSecretKey('sk_your_secret_key');
```

### Configurable Base Path

Configure the base path for all API routes:

```typescript
// Option 1: Set at initialization
const client = new AuthsomeClient({
  baseURL: 'http://localhost:3000',
  basePath: '/api/auth'  // All routes prefixed with /api/auth
});

// Option 2: Change at runtime
client.setBasePath('/v2/auth');
```

**How it works:**
- Core methods like `/signup` become `http://localhost:3000/api/auth/signup`
- Plugin methods like `/mfa/factors` become `http://localhost:3000/api/auth/mfa/factors`

### Type-Safe Plugin Access

Two ways to access plugins with full type safety:

#### Option 1: Type-Safe Registry (Recommended)

```typescript
import { AuthsomeClient, MfaPlugin, mfaClient } from '@authsome/client';

const client = new AuthsomeClient({
  baseURL: 'http://localhost:3000',
  basePath: '/api/auth',
  plugins: [mfaClient()]
});

// Access via type-safe registry
const mfa = client.$plugins.mfa();
if (mfa) {
  await mfa.enrollFactor({
    type: 'totp',
    name: 'My Authenticator',
    priority: 'primary'
  });
}
```

#### Option 2: Generic Method

```typescript
import { MfaPlugin } from '@authsome/client';

const mfa = client.getPlugin<MfaPlugin>('mfa');
if (mfa) {
  await mfa.listFactors();
}
```

## Available Plugins

### Security Plugins
- **mfa** - Multi-factor authentication
- **twofa** - Two-factor authentication
- **passkey** - WebAuthn/Passkey authentication
- **backupauth** - Backup authentication methods

### Social & OAuth
- **social** - Social login (Google, GitHub, etc.)
- **sso** - Single Sign-On
- **oidcprovider** - OpenID Connect Provider

### Communication
- **emailotp** - Email-based OTP
- **phone** - Phone/SMS authentication
- **magiclink** - Magic link authentication
- **notification** - Notification management

### Enterprise Features
- **compliance** - Compliance and data governance
- **consent** - User consent management
- **idverification** - Identity verification
- **stepup** - Step-up authentication

### User Management
- **username** - Username-based authentication
- **anonymous** - Anonymous sessions
- **impersonation** - User impersonation
- **organization** - Organization management
- **multiapp** - Multi-application support

### Developer Tools
- **admin** - Administrative operations
- **apikey** - API key management
- **jwt** - JWT token management
- **webhook** - Webhook management
- **multisession** - Multiple session support

## Usage Examples

### Basic Authentication

```typescript
// Sign up
const { user, session } = await client.signUp({
  email: 'user@example.com',
  password: 'password123',
  name: 'John Doe'
});

// Sign in
const result = await client.signIn({
  email: 'user@example.com',
  password: 'password123'
});

// Check if 2FA is required
if (result.requiresTwoFactor) {
  // Handle 2FA flow
}

// Sign out
await client.signOut();

// Get current session
const { user, session } = await client.getSession();
```

### Multi-Factor Authentication

```typescript
import { mfaClient } from '@authsome/client';

const client = new AuthsomeClient({
  baseURL: 'http://localhost:3000',
  basePath: '/api/auth',
  plugins: [mfaClient()]
});

const mfa = client.$plugins.mfa();

// Enroll a new factor
await mfa.enrollFactor({
  type: 'totp',
  name: 'Google Authenticator',
  priority: 'primary'
});

// List enrolled factors
const { factors } = await mfa.listFactors();

// Verify a factor
await mfa.verifyFactor({
  factorId: 'factor_123',
  code: '123456'
});

// Trust a device
await mfa.trustDevice({
  deviceId: 'device_123',
  name: 'My Laptop'
});
```

### Social Authentication

```typescript
import { socialClient } from '@authsome/client';

const client = new AuthsomeClient({
  baseURL: 'http://localhost:3000',
  basePath: '/api/auth',
  plugins: [socialClient()]
});

const social = client.$plugins.social();

// Get OAuth URL
const { url } = await social.getAuthUrl({
  provider: 'google',
  redirectUri: 'http://localhost:3000/callback'
});

// Redirect user to OAuth provider
window.location.href = url;

// Handle callback
const { user, session } = await social.callback({
  provider: 'google',
  code: 'oauth_code_from_callback'
});
```

### Organization Management

```typescript
import { organizationClient } from '@authsome/client';

const client = new AuthsomeClient({
  baseURL: 'http://localhost:3000',
  basePath: '/api/auth',
  plugins: [organizationClient()]
});

const org = client.$plugins.organization();

// Create organization
const { organization } = await org.createOrganization({
  name: 'Acme Corp',
  slug: 'acme'
});

// List user's organizations
const { organizations } = await org.listOrganizations();

// Switch active organization
await org.switchOrganization({
  organizationId: 'org_123'
});
```

### Admin Operations

```typescript
import { adminClient } from '@authsome/client';

// Use secret key for admin operations
const client = new AuthsomeClient({
  baseURL: 'http://localhost:3000',
  basePath: '/api/auth',
  plugins: [adminClient()]
});
client.setSecretKey('sk_your_secret_key');

const admin = client.$plugins.admin();

// List all users
const { users } = await admin.listUsers();

// Ban a user
await admin.banUser({
  userId: 'user_123',
  reason: 'Violation of terms',
  expiresAt: '2025-12-31'
});

// Impersonate a user
const { session } = await admin.impersonateUser({
  userId: 'user_123'
});
```

## Configuration Options

```typescript
interface AuthsomeClientConfig {
  /** Base URL of the AuthSome API */
  baseURL: string;
  
  /** Base path prefix for all API routes (default: '') */
  basePath?: string;
  
  /** Plugin instances to initialize */
  plugins?: ClientPlugin[];
  
  /** JWT/Bearer token for user authentication (sent only when auth: true) */
  token?: string;
  
  /** API key for server-to-server auth (pk_* or sk_*, sent with all requests) */
  apiKey?: string;
  
  /** Custom header name for API key (default: 'X-API-Key') */
  apiKeyHeader?: string;
  
  /** Custom headers to include with all requests */
  headers?: Record<string, string>;
}
```

## Error Handling

The client throws typed errors that you can catch and handle:

```typescript
import { 
  UnauthorizedError, 
  ValidationError, 
  NotFoundError,
  RateLimitError 
} from '@authsome/client';

try {
  await client.signIn({ email, password });
} catch (error) {
  if (error instanceof UnauthorizedError) {
    console.error('Invalid credentials');
  } else if (error instanceof ValidationError) {
    console.error('Invalid input:', error.fields);
  } else if (error instanceof RateLimitError) {
    console.error('Too many requests, please wait');
  }
}
```

## TypeScript Support

The client is written in TypeScript and provides full type definitions:

```typescript
import type { 
  User, 
  Session, 
  Device,
  MessageResponse 
} from '@authsome/client';

// All API responses are fully typed
const result: { user: User; session: Session } = await client.signUp({
  email: 'user@example.com',
  password: 'password123'
});
```

## Version History

See [CHANGELOG.md](./CHANGELOG.md) for detailed version history.

## License

MIT

