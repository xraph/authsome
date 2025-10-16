# Quick Start: AuthSome Client Generator

Generate type-safe authentication clients in Go, TypeScript, or Rust with a single command.

## TL;DR

```bash
# Generate TypeScript client
authsome-cli generate client --lang typescript

# Generate all languages
authsome-cli generate client --lang all

# List available plugins
authsome-cli generate client --list
```

## Installation

The client generator is built into the AuthSome CLI tool:

```bash
# Build the CLI
go build -o authsome-cli ./cmd/authsome-cli

# Or use go run
go run ./cmd/authsome-cli generate client --help
```

## Basic Usage

### Generate TypeScript Client

```bash
authsome-cli generate client --lang typescript --output ./my-app/src/lib/authsome

cd ./my-app/src/lib/authsome
npm install
npm run build
```

**Use it:**
```typescript
import { AuthsomeClient, socialClient, twofaClient } from './lib/authsome';

const client = new AuthsomeClient({
  baseURL: 'https://api.example.com',
  plugins: [socialClient(), twofaClient()]
});

// Sign up
const { user, session } = await client.signUp({
  email: 'user@example.com',
  password: 'secure123'
});

// Use plugin
const social = client.getPlugin('social');
const { url } = await social.signIn({ provider: 'google' });
```

### Generate Go Client

```bash
authsome-cli generate client --lang go --output ./clients

cd ./clients/go
go mod tidy
```

**Use it:**
```go
import (
    authsome "github.com/xraph/authsome-client"
    "github.com/xraph/authsome-client/plugins/social"
)

client := authsome.NewClient("https://api.example.com",
    authsome.WithPlugins(social.NewPlugin()),
)

resp, err := client.SignUp(ctx, &authsome.SignUpRequest{
    Email:    "user@example.com",
    Password: "secure123",
})
```

### Generate Rust Client

```bash
authsome-cli generate client --lang rust --output ./clients

cd ./clients/rust
cargo build
```

**Use it:**
```rust
use authsome_client::{AuthsomeClient, Result};

let client = AuthsomeClient::builder()
    .base_url("https://api.example.com")
    .build()?;

let response = client.sign_up(SignUpRequest {
    email: "user@example.com".into(),
    password: "secure123".into(),
}).await?;
```

## Advanced Usage

### Generate with Specific Plugins Only

```bash
authsome-cli generate client \
  --lang typescript \
  --plugins core,social,twofa
```

### Custom Manifest Directory

```bash
authsome-cli generate client \
  --lang go \
  --manifest-dir ./custom-manifests
```

### Validate Manifests

```bash
authsome-cli generate client --validate
```

## Available Plugins

Run this command to see all available plugins:

```bash
authsome-cli generate client --list
```

**Current plugins:**
- `core` - Authentication basics (signup, signin, session)
- `social` - Social OAuth (Google, GitHub, etc.)
- `twofa` - Two-factor authentication
- `organization` - Organization management
- `webhook` - Webhook configuration
- `magiclink` - Passwordless magic links
- `passkey` - WebAuthn/Passkeys

## Plugin Architecture

Clients use the same plugin pattern as the server:

```typescript
// Only include what you need
const client = new AuthsomeClient({
  baseURL: 'https://api.example.com',
  plugins: [
    socialClient(),    // Add social OAuth
    twofaClient(),     // Add 2FA
    // Other plugins as needed
  ]
});
```

**Benefits:**
- Tree-shakeable (only bundle what you use)
- Type-safe plugin APIs
- Matches server architecture
- Easy to extend

## Adding New Routes

### Option 1: Auto-Generate from Code (Recommended)

```bash
# 1. Write your handler code
vim plugins/myfeature/handlers.go

# 2. Auto-generate manifest from code
authsome-cli generate introspect --plugin myfeature

# 3. Regenerate clients
authsome-cli generate client --lang all

# Done!
```

See `clients/INTROSPECTION.md` for details on code introspection.

### Option 2: Manual Manifest

1. **Create manifest** in `clients/manifest/data/`:

```yaml
# clients/manifest/data/myfeature.yaml
plugin_id: myfeature
version: 1.0.0
routes:
  - name: DoSomething
    method: POST
    path: /something
    request:
      data: string!
    response:
      result: string!
```

2. **Regenerate clients:**

```bash
authsome-cli generate client --lang all
```

3. **Use immediately:**

```typescript
const result = await client.myfeature.doSomething({ data: "test" });
```

## Type System

Manifests use a simple type syntax:

- `string`, `int`, `bool` - Basic types
- `string!` - Required field (append `!`)
- `string[]` - Array type (append `[]`)
- `User`, `Session` - Custom types (defined in manifest)

**Example:**
```yaml
request:
  email: string!         # Required string
  name: string           # Optional string
  tags: string[]         # Array of strings
  metadata: object       # Generic object
```

## Error Handling

All generated clients map HTTP status codes to semantic errors:

**TypeScript:**
```typescript
try {
  await client.signUp({ email, password });
} catch (error) {
  if (error instanceof ValidationError) {
    console.error('Invalid input:', error.message);
  } else if (error instanceof UnauthorizedError) {
    console.error('Not authorized:', error.message);
  }
}
```

**Go:**
```go
if err != nil {
    if authErr, ok := err.(*authsome.Error); ok {
        fmt.Printf("Error %d: %s\n", authErr.StatusCode, authErr.Message)
    }
}
```

**Rust:**
```rust
match client.sign_up(req).await {
    Ok(response) => println!("Success"),
    Err(AuthsomeError::Validation(msg)) => eprintln!("Validation: {}", msg),
    Err(AuthsomeError::Unauthorized(msg)) => eprintln!("Auth: {}", msg),
    Err(e) => eprintln!("Error: {:?}", e),
}
```

## Authentication

Generated clients handle authentication automatically:

```typescript
// Sign in
const { session } = await client.signIn({ email, password });

// Store token
client.setToken(session.token);

// All subsequent requests include the token automatically
const userData = await client.getSession(); // ✓ Authenticated

// Sign out
await client.signOut();
```

## Examples

Complete working examples are in `examples/client-usage/`:

```bash
# TypeScript example
cd examples/client-usage/typescript-example
npm install
npm start

# Go example
cd examples/client-usage/go-example
go run main.go

# Rust example
cd examples/client-usage/rust-example
cargo run
```

## Troubleshooting

### "Manifest validation failed"

Run validation to see specific errors:
```bash
authsome-cli generate client --validate
```

### "Failed to load manifests"

Check that manifest directory exists:
```bash
ls -la clients/manifest/data/
```

### Generated TypeScript has errors

Ensure TypeScript version is 5.0+:
```bash
cd clients/generated/typescript
npm install typescript@latest
npm run build
```

### Go client import errors

Run `go mod tidy` in the generated directory:
```bash
cd clients/generated/go
go mod tidy
```

## Documentation

- **Full Guide:** `clients/README.md`
- **Implementation Details:** `CLIENT_GENERATOR_SUMMARY.md`
- **Manifest Format:** `clients/manifest/schema.go`

## Support

For issues or questions:
1. Check the examples in `examples/client-usage/`
2. Review the comprehensive README in `clients/README.md`
3. Validate your manifests with `--validate`
4. Ensure you're using the latest version

## What's Next?

Once you have your client generated:

1. **Integrate into your app** - Import and use the client
2. **Customize as needed** - Modify manifests for your routes
3. **Keep in sync** - Regenerate when server APIs change
4. **Deploy confidently** - Type safety prevents runtime errors

---

**Ready to go!** Generate your client and start building.

```bash
authsome-cli generate client --lang typescript
```

