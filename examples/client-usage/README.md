# Client Usage Examples

This directory contains examples of using the generated AuthSome clients in Go, TypeScript, and Rust.

## Prerequisites

1. **Generate the clients:**
   ```bash
   cd ../..
   make generate-clients
   # or
   authsome generate client --lang all
   ```

2. **Run the AuthSome server (optional):**
   ```bash
   cd ../../examples/comprehensive
   go run .
   ```

   Or modify the `baseURL` in each example to point to your own server.

## Examples

Each example directory has its own README with detailed instructions.

### Go Example

```bash
cd go-example
go mod download
go run main.go
```

See [go-example/README.md](go-example/README.md) for details.

### TypeScript Example

```bash
cd typescript-example
npm install
npm start
```

See [typescript-example/README.md](typescript-example/README.md) for details.

### Rust Example

```bash
cd rust-example
cargo run
```

See [rust-example/README.md](rust-example/README.md) for details.

## What These Examples Demonstrate

Each example shows:

1. **Client initialization** with plugin composition
2. **User registration** (SignUp / sign_up / signUp)
3. **Session management** (GetSession / get_session / getSession)
4. **Profile updates** (UpdateUser / update_user / updateUser)
5. **Plugin usage** (Social OAuth, 2FA)
6. **Device management** (ListDevices / list_devices / listDevices)
7. **Sign out** (SignOut / sign_out / signOut)
8. **Error handling** with typed errors

## Project Structure

```
client-usage/
├── go-example/
│   ├── go.mod              # Go module with local client reference
│   ├── main.go             # Example code
│   └── README.md
├── typescript-example/
│   ├── package.json        # NPM package with local client reference
│   ├── index.ts            # Example code
│   └── README.md
├── rust-example/
│   ├── Cargo.toml          # Cargo package with local client reference
│   ├── src/main.rs         # Example code
│   └── README.md
└── README.md               # This file
```

## Local Client References

Each example uses local path references to the generated clients:

- **Go**: `replace` directive in `go.mod`
- **TypeScript**: `file:` dependency in `package.json`
- **Rust**: `path` dependency in `Cargo.toml`

This allows testing the examples immediately after generation without publishing.

## Customizing

- Change `baseURL` to point to your AuthSome server
- Add/remove plugins as needed
- Modify credentials and test data
- Extend with your own use cases

## Regenerating Clients

To regenerate all clients:
```bash
cd ../..
make generate-clients
```

To regenerate a specific language:
```bash
make generate-go
make generate-typescript
make generate-rust
```

