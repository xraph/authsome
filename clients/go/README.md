# AuthSome Go Client SDK

Enterprise-grade Go client for AuthSome authentication with automatic middleware support for Forge and standard HTTP clients.

## Features

- 🔐 **Multiple Authentication Methods**: API keys, session tokens, and cookies
- 🔄 **Auto-detection**: Automatically chooses the right auth method
- 🎯 **Forge Integration**: First-class support for Forge framework
- 🌐 **Standard HTTP**: Works with any `http.Client` via `http.RoundTripper`
- 📦 **Context Management**: Built-in app and environment context handling
- 🍪 **Cookie Support**: Automatic session cookie management
- 🔗 **Plugin System**: Extensible plugin architecture

## Installation

```bash
go get github.com/xraph/authsome/clients/go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    authsome "github.com/xraph/authsome/clients/go"
)

func main() {
    // Create client with session token
    client := authsome.NewClient(
        "https://api.example.com",
        authsome.WithToken("your-session-token"),
    )
    
    // Sign in
    resp, err := client.SignIn(context.Background(), &authsome.SignInRequest{
        Email:    "user@example.com",
        Password: "password123",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Signed in as: %s", resp.User.Email)
}
```

## Authentication Methods

### 1. API Key Authentication

Recommended for server-to-server communication:

```go
client := authsome.NewClient(
    "https://api.example.com",
    authsome.WithAPIKey("sk_live_abc123"),
)
```

API keys automatically add `Authorization: ApiKey {key}` header.

### 2. Session Token Authentication

For authenticated user requests:

```go
client := authsome.NewClient(
    "https://api.example.com",
    authsome.WithToken("user-session-token"),
)
```

Session tokens automatically add `Authorization: Bearer {token}` header.

### 3. Cookie-Based Authentication

For browser-like session management:

```go
import "net/http/cookiejar"

jar, _ := cookiejar.New(nil)
client := authsome.NewClient(
    "https://api.example.com",
    authsome.WithCookieJar(jar),
)
```

Cookies are automatically sent with requests.

### 4. Dual Authentication

Combine API key and session token:

```go
client := authsome.NewClient(
    "https://api.example.com",
    authsome.WithAPIKey("pk_live_xyz"),
    authsome.WithToken("user-session-token"),
)
```

Priority: API Key > Session Token > Cookies

## Forge Framework Integration

### Setting Up Forge Middleware

```go
package main

import (
    "github.com/xraph/forge"
    authsome "github.com/xraph/authsome/clients/go"
)

func main() {
    app := forge.New()
    
    // Create AuthSome client
    authClient := authsome.NewClient(
        "https://auth.example.com",
        authsome.WithAPIKey("sk_live_abc123"),
    )
    
    // Apply authentication middleware globally
    app.Use(authClient.ForgeMiddleware())
    
    // Protected routes
    api := app.Group("/api")
    api.Use(authClient.RequireAuth())
    
    api.GET("/profile", func(c forge.Context) error {
        // Get user from context
        session, ok := authsome.GetSessionFromContext(c.Request().Context())
        if !ok {
            return c.JSON(401, map[string]string{"error": "not authenticated"})
        }
        
        return c.JSON(200, session)
    })
    
    app.Listen(":3000")
}
```

### Middleware Functions

- **`ForgeMiddleware()`**: Optional authentication - populates context
- **`RequireAuth()`**: Blocks unauthenticated requests
- **`OptionalAuth()`**: Alias for `ForgeMiddleware()`

## Standard HTTP Client Integration

### Using RoundTripper

```go
package main

import (
    "net/http"
    authsome "github.com/xraph/authsome/clients/go"
)

func main() {
    // Create AuthSome client
    authClient := authsome.NewClient(
        "https://api.example.com",
        authsome.WithAPIKey("sk_live_abc123"),
    )
    
    // Create HTTP client with auth injection
    httpClient := &http.Client{
        Transport: authClient.RoundTripper(),
    }
    
    // All requests automatically include authentication
    resp, err := httpClient.Get("https://other-service.com/api/data")
    // ... handle response
}
```

### Convenience Method

```go
// Create fully configured HTTP client
httpClient := authClient.NewHTTPClientWithAuth()
```

## Context Management

### Setting App and Environment Context

```go
client := authsome.NewClient(
    "https://api.example.com",
    authsome.WithAPIKey("sk_live_abc123"),
    authsome.WithAppContext("app_123", "env_prod"),
)

// Or set dynamically
client.SetAppContext("app_456", "env_staging")

// Get current context
appID, envID := client.GetAppContext()
```

Context headers are automatically added:
- `X-App-ID`: Application identifier
- `X-Environment-ID`: Environment identifier

### Context Helper Functions

```go
import (
    "context"
    authsome "github.com/xraph/authsome/clients/go"
)

ctx := context.Background()

// Add app ID
ctx = authsome.WithAppID(ctx, "app_123")

// Add environment ID
ctx = authsome.WithEnvironmentID(ctx, "env_prod")

// Add both
ctx = authsome.WithAppContext(ctx, "app_123", "env_prod")

// Retrieve values
appID, ok := authsome.GetAppID(ctx)
envID, ok := authsome.GetEnvironmentID(ctx)
```

## User and Session Helpers

### Get Current User

```go
user, err := client.GetCurrentUser(context.Background())
if err != nil {
    log.Fatal(err)
}
log.Printf("User: %s (%s)", user.Name, user.Email)
```

### Get Current Session

```go
session, err := client.GetCurrentSession(context.Background())
if err != nil {
    log.Fatal(err)
}
log.Printf("Session expires: %s", session.ExpiresAt)
```

## Custom Headers

```go
client := authsome.NewClient(
    "https://api.example.com",
    authsome.WithHeaders(map[string]string{
        "X-Custom-Header": "value",
        "User-Agent":      "MyApp/1.0",
    }),
)
```

## Plugin System

### Using Plugins

```go
import (
    authsome "github.com/xraph/authsome/clients/go"
    "github.com/xraph/authsome/clients/go/plugins/social"
)

client := authsome.NewClient(
    "https://api.example.com",
    authsome.WithPlugins(
        social.NewPlugin(),
    ),
)

// Access plugin
socialPlugin, ok := client.GetPlugin("social")
if ok {
    // Use plugin methods
}
```

## Advanced Configuration

### Complete Example

```go
package main

import (
    "context"
    "log"
    "net/http"
    "net/http/cookiejar"
    
    authsome "github.com/xraph/authsome/clients/go"
)

func main() {
    // Create cookie jar
    jar, err := cookiejar.New(nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create custom HTTP client
    httpClient := &http.Client{
        Timeout: 30 * time.Second,
        Jar:     jar,
    }
    
    // Create AuthSome client with all options
    client := authsome.NewClient(
        "https://api.example.com",
        authsome.WithHTTPClient(httpClient),
        authsome.WithAPIKey("sk_live_abc123"),
        authsome.WithCookieJar(jar),
        authsome.WithAppContext("app_prod_123", "env_production"),
        authsome.WithHeaders(map[string]string{
            "User-Agent": "MyApp/1.0.0",
        }),
    )
    
    // Use client
    session, err := client.GetCurrentSession(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Authenticated as: %s", session.User.Email)
}
```

## Error Handling

```go
resp, err := client.SignIn(context.Background(), &authsome.SignInRequest{
    Email:    "user@example.com",
    Password: "wrong-password",
})

if err != nil {
    if authErr, ok := err.(*authsome.Error); ok {
        switch authErr.StatusCode {
        case 401:
            log.Println("Invalid credentials")
        case 403:
            log.Println("Account locked")
        case 429:
            log.Println("Rate limited")
        default:
            log.Printf("Error: %s (code: %s)", authErr.Message, authErr.Code)
        }
    }
    return
}
```

## Best Practices

### 1. API Key Security

```go
// ✅ DO: Load from environment
apiKey := os.Getenv("AUTHSOME_API_KEY")

// ❌ DON'T: Hardcode in source
apiKey := "sk_live_abc123" // Never do this!
```

### 2. Context Propagation

```go
// ✅ DO: Pass context through call chain
func handleRequest(ctx context.Context, client *authsome.Client) error {
    return client.GetSession(ctx)
}

// ❌ DON'T: Create new context
func handleRequest(client *authsome.Client) error {
    return client.GetSession(context.Background())
}
```

### 3. Resource Cleanup

```go
// ✅ DO: Reuse client instances
var authClient *authsome.Client

func init() {
    authClient = authsome.NewClient(...)
}

// ❌ DON'T: Create new clients for each request
func handleRequest() {
    client := authsome.NewClient(...) // Wasteful!
}
```

## Regenerating the Client

When the AuthSome API changes, regenerate the client:

```bash
cd /path/to/authsome
authsome generate client --lang go --output ./clients
```

## Support

- Documentation: https://docs.authsome.dev
- Repository: https://github.com/xraph/authsome
- Issues: https://github.com/xraph/authsome/issues

## License

See LICENSE file in the repository root.

