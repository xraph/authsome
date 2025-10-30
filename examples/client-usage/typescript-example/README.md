# TypeScript Client Example

This example demonstrates using the generated AuthSome TypeScript client library.

## Prerequisites

1. **Generate the TypeScript client:**
   ```bash
   cd ../../..
   make generate-typescript
   # or
   authsome generate client --lang typescript
   ```

2. **Ensure the AuthSome server is running** (optional, for testing against live server)

## Running the Example

```bash
# Install dependencies
npm install

# Run the example
npm start

# Or build TypeScript
npm run build
```

## What This Example Demonstrates

1. **Client Initialization** with plugins
2. **User Registration** (`signUp`)
3. **Session Management** (`getSession`)
4. **Profile Updates** (`updateUser`)
5. **Plugin Usage** (Social OAuth, 2FA)
6. **Device Management** (`listDevices`)
7. **Sign Out** (`signOut`)
8. **Error Handling** with `AuthsomeError`

## Package Configuration

This example uses a local file reference in `package.json`:

```json
{
  "dependencies": {
    "@authsome/client": "file:../../../clients/typescript"
  }
}
```

This allows the example to work immediately after generating the client.

## Customizing

- Change `baseURL` in `index.ts` to point to your server
- Add/remove plugins from the client initialization
- Modify the example flows to test different scenarios

## Generated Client Location

The client is generated to: `../../../clients/typescript/`

To regenerate it, run:
```bash
make generate-typescript
```

