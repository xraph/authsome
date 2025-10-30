# Rust Client Example

This example demonstrates using the generated AuthSome Rust client library.

## Prerequisites

1. **Generate the Rust client:**
   ```bash
   cd ../../..
   make generate-rust
   # or
   authsome generate client --lang rust
   ```

2. **Ensure the AuthSome server is running** (optional, for testing against live server)

## Running the Example

```bash
# Run the example
cargo run

# Or build and run
cargo build
./target/debug/authsome-client-example
```

## What This Example Demonstrates

1. **Client Initialization** with builder pattern
2. **User Registration** (`sign_up`)
3. **Session Management** (`get_session`)
4. **Profile Updates** (`update_user`)
5. **Plugin Usage** (Social OAuth, 2FA)
6. **Device Management** (`list_devices`)
7. **Sign Out** (`sign_out`)
8. **Error Handling** with Rust `Result` types

## Cargo Configuration

This example uses a path dependency in `Cargo.toml`:

```toml
[dependencies]
authsome-client = { path = "../../../clients/rust" }
```

This allows the example to work immediately after generating the client.

## Customizing

- Change `base_url` when building the client
- Add/remove plugins from the client initialization
- Modify the example flows to test different scenarios

## Generated Client Location

The client is generated to: `../../../clients/rust/`

To regenerate it, run:
```bash
make generate-rust
```

