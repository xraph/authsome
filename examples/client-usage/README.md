# Client Usage Examples

This directory contains examples of using the generated AuthSome clients in Go, TypeScript, and Rust.

## Prerequisites

1. **Generate the clients:**
   ```bash
   cd ../..
   authsome-cli generate client --lang all
   ```

2. **Run the AuthSome server:**
   ```bash
   cd examples/comprehensive
   ./comprehensive-server
   ```

## Examples

### TypeScript Example

```bash
cd typescript-example
npm install
npm start
```

### Go Example

```bash
cd go-example
go run main.go
```

### Rust Example

```bash
cd rust-example
cargo run
```

## What These Examples Do

Each example demonstrates:

1. **Client initialization** with plugin composition
2. **User registration** (SignUp)
3. **User authentication** (SignIn)
4. **Session management** (GetSession)
5. **Plugin usage** (Social OAuth, 2FA)
6. **Error handling**

## Customizing Examples

- Change `baseURL` to point to your AuthSome server
- Add/remove plugins as needed
- Modify credentials and test data
- Add your own use cases

