# Go Client Example

This example demonstrates using the generated AuthSome Go client library.

## Prerequisites

1. **Generate the Go client:**
   ```bash
   cd ../../..
   make generate-go
   # or
   authsome generate client --lang go
   ```

2. **Ensure the AuthSome server is running** (optional, for testing against live server)

## Running the Example

```bash
# Download dependencies
go mod download

# Run the example
go run main.go

# Or build and run
go build -o go-example .
./go-example
```

## What This Example Demonstrates

1. **Client Initialization** with custom options and plugins
2. **User Registration** (`SignUp`)
3. **Session Management** (`GetSession`)
4. **Profile Updates** (`UpdateUser`)
5. **Plugin Usage** (Social OAuth, 2FA)
6. **Device Management** (`ListDevices`)
7. **Sign Out** (`SignOut`)
8. **Error Handling** with typed errors

## Module Configuration

This example uses a `go.mod` with a replace directive to use the locally generated client:

```go
replace github.com/xraph/authsome/clients/go => ../../../clients/go
```

This allows the example to work immediately after generating the client, without needing to publish it.

## Customizing

- Change `baseURL` in `main.go` to point to your server
- Add/remove plugins from the client initialization
- Modify the example flows to test different scenarios
- Use the `handleError` function for better error messages

## Generated Client Location

The client is generated to: `../../../clients/go/`

To regenerate it, run:
```bash
make generate-go
```

