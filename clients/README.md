# AuthSome Client Generator

This directory contains the client code generator for AuthSome, which produces type-safe client libraries in Go, TypeScript, and Rust.

## Overview

The generator reads API manifests (YAML files) and produces complete, production-ready client libraries that mirror the server's API structure and plugin architecture.

## Directory Structure

```
clients/
├── manifest/
│   ├── data/              # API route manifests (YAML)
│   ├── schema.go          # Manifest data structures
│   └── parser.go          # Manifest parser
├── generator/
│   ├── generator.go       # Main generator coordinator
│   ├── go.go             # Go client generator
│   ├── typescript.go      # TypeScript client generator
│   └── rust.go            # Rust client generator
├── generated/             # Generated output (gitignored)
│   ├── go/
│   ├── typescript/
│   └── rust/
└── README.md             # This file
```

## Manifests

Manifests define API routes in a language-agnostic format. Each manifest represents either the core API or a plugin.

### Core Manifests
- `core.yaml` - Core authentication (signup, signin, session, etc.)
- `organization.yaml` - Organization management
- `webhook.yaml` - Webhook management

### Plugin Manifests
- `social.yaml` - Social OAuth authentication
- `twofa.yaml` - Two-factor authentication
- `magiclink.yaml` - Magic link authentication
- `passkey.yaml` - Passkey (WebAuthn) authentication
- ... and more

### Manifest Format

```yaml
plugin_id: social
version: 1.0.0
description: Social OAuth authentication
base_path: /api/auth

types:
  - name: SocialAccount
    description: Linked social account
    fields:
      id: string!
      provider: string!
      email: string

routes:
  - name: SignIn
    description: Initiate OAuth flow
    method: POST
    path: /signin/social
    request:
      provider: string!
      scopes: string[]
    response:
      url: string!
    errors:
      - code: 400
        description: Invalid provider
```

**Field Type Syntax:**
- `string` - String type
- `int`, `int64`, `float64`, `bool` - Numeric/boolean types
- `object` - Generic object/map
- `!` suffix - Required field (e.g., `string!`)
- `[]` suffix - Array type (e.g., `string[]`)
- Custom types reference types defined in `types` section

## Usage

### Generate All Clients

```bash
authsome-cli generate client --lang all
```

### Generate Specific Language

```bash
# TypeScript
authsome-cli generate client --lang typescript

# Go
authsome-cli generate client --lang go

# Rust
authsome-cli generate client --lang rust
```

### Generate with Specific Plugins

```bash
authsome-cli generate client --lang typescript --plugins core,social,twofa
```

### Custom Output Directory

```bash
authsome-cli generate client --lang typescript --output ./my-app/src/lib/authsome
```

### Validate Manifests

```bash
authsome-cli generate client --validate
```

### List Available Plugins

```bash
authsome-cli generate client --list
```

## Generated Clients

### TypeScript

**Structure:**
```
typescript/
├── src/
│   ├── index.ts           # Main exports
│   ├── client.ts          # AuthsomeClient class
│   ├── types.ts           # Type definitions
│   ├── errors.ts          # Error classes
│   ├── plugin.ts          # Plugin interface
│   └── plugins/
│       ├── social.ts
│       ├── twofa.ts
│       └── ...
├── package.json
└── tsconfig.json
```

**Usage:**
```typescript
import { AuthsomeClient, socialClient, twofaClient } from '@authsome/client';

const client = new AuthsomeClient({
  baseURL: 'https://api.example.com',
  plugins: [
    socialClient(),
    twofaClient(),
  ]
});

// Core methods
const { user, session } = await client.signUp({
  email: 'user@example.com',
  password: 'secure123'
});

// Plugin methods
const plugin = client.getPlugin<SocialPlugin>('social');
const { url } = await plugin.signIn({ provider: 'google' });
```

### Go

**Structure:**
```
go/
├── client.go              # Main client
├── types.go               # Type definitions
├── errors.go              # Error types
├── plugin.go              # Plugin interface
├── plugins/
│   ├── social/
│   │   └── social.go
│   └── twofa/
│       └── twofa.go
└── go.mod
```

**Usage:**
```go
import (
    "github.com/xraph/authsome-client"
    "github.com/xraph/authsome-client/plugins/social"
    "github.com/xraph/authsome-client/plugins/twofa"
)

client := authsome.NewClient("https://api.example.com",
    authsome.WithPlugins(
        social.NewPlugin(),
        twofa.NewPlugin(),
    ),
)

// Core methods
resp, err := client.SignUp(ctx, &authsome.SignUpRequest{
    Email:    "user@example.com",
    Password: "secure123",
})

// Plugin methods
socialPlugin, _ := client.GetPlugin("social").(*social.Plugin)
url, err := socialPlugin.SignIn(ctx, &social.SignInRequest{
    Provider: "google",
})
```

### Rust

**Structure:**
```
rust/
├── src/
│   ├── lib.rs             # Library exports
│   ├── client.rs          # AuthsomeClient
│   ├── types.rs           # Type definitions
│   ├── error.rs           # Error types
│   ├── plugin.rs          # Plugin trait
│   └── plugins/
│       ├── mod.rs
│       ├── social.rs
│       └── twofa.rs
└── Cargo.toml
```

**Usage:**
```rust
use authsome_client::{AuthsomeClient, Result};
use authsome_client::plugins::social::SocialPlugin;

#[tokio::main]
async fn main() -> Result<()> {
    let client = AuthsomeClient::builder()
        .base_url("https://api.example.com")
        .build()?;

    // Core methods
    let response = client.sign_up(SignUpRequest {
        email: "user@example.com".into(),
        password: "secure123".into(),
    }).await?;

    // Plugin methods
    let social = SocialPlugin::new();
    let url = social.sign_in(SignInRequest {
        provider: "google".into(),
        scopes: vec![],
    }).await?;

    Ok(())
}
```

## Features

### Type Safety
- Full type checking at compile time
- Required vs optional fields enforced
- Array types properly handled
- Custom type references resolved

### Plugin Architecture
- Clients mirror server plugin system
- Plugins are composable and optional
- Tree-shakeable in TypeScript
- No runtime overhead for unused plugins

### Error Handling
- HTTP status codes mapped to semantic errors
- Error messages preserved from server
- Stack traces in development

### Authentication
- Bearer token support
- Cookie-based sessions
- Automatic token injection when configured

### Validation
- Request validation before API calls
- Type-safe parameters
- Required field checking

## Adding New Routes

1. **Create/update manifest:**
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

3. **Use in your app:**
```typescript
const result = await client.myfeature.doSomething({ data: "test" });
```

## Development

### Adding a New Language

1. Create `clients/generator/{language}.go`
2. Implement generator following existing patterns
3. Update `generator.go` to register new language
4. Add tests and documentation

### Extending Manifest Schema

1. Update `clients/manifest/schema.go`
2. Update parser in `clients/manifest/parser.go`
3. Update all generators to handle new fields
4. Regenerate and test all clients

## Testing

```bash
# Run generator tests
go test ./clients/...

# Generate test clients
authsome-cli generate client --lang all --output ./test-output

# Validate all manifests
authsome-cli generate client --validate
```

## Best Practices

1. **Keep manifests in sync with server code**
2. **Version manifests appropriately**
3. **Document all custom types**
4. **Test generated clients before releasing**
5. **Use semantic versioning for breaking changes**

## Troubleshooting

### Generation Fails

- Check manifest syntax with `--validate`
- Ensure all type references are defined
- Verify YAML indentation

### Type Errors in Generated Code

- Check field type mappings in generators
- Ensure custom types are defined in manifest
- Verify required/optional markers

### Plugin Methods Not Available

- Ensure plugin is included in generation
- Check plugin initialization in client code
- Verify plugin ID matches manifest

## Contributing

When adding new features:

1. Update relevant manifests
2. Regenerate all clients
3. Test in all three languages
4. Update documentation
5. Add examples

## License

MIT

