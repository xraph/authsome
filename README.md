# AuthSome

[![CI](https://github.com/xraph/farp/workflows/CI/badge.svg)](https://github.com/xraph/farp/actions?query=workflow%3ACI)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/xraph/authsome)](https://goreportcard.com/report/github.com/xraph/authsome)
[![GoDoc](https://godoc.org/github.com/xraph/authsome?status.svg)](https://godoc.org/github.com/xraph/authsome)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-available-blue.svg)](https://authsome.xraph.dev)
[![Release](https://img.shields.io/github/v/release/xraph/authsome)](https://github.com/xraph/authsome/releases)

**Authors**: [Rex Raphael](githib.com/juicycleff)
**Last Updated**: 2025-11-01

> A comprehensive, pluggable authentication framework for Go, inspired by better-auth. Enterprise-grade authentication with multi-tenancy support, designed to integrate seamlessly with the Forge framework.

## 🚀 Features

### Core Authentication
- **Email/Password Authentication** - Secure user registration and login
- **Session Management** - Cookie-based sessions with Redis caching support
- **Multi-Factor Authentication** - TOTP, SMS, and email-based 2FA
- **Social Authentication** - OAuth integration with major providers
- **Passwordless Authentication** - Magic links and WebAuthn/Passkeys

### Enterprise Features
- **Multi-Tenancy** - Organization-scoped authentication and configuration
- **Role-Based Access Control (RBAC)** - Fine-grained permissions and policies
- **Audit Logging** - Comprehensive security event tracking
- **Rate Limiting** - Configurable request throttling and abuse prevention
- **Device Management** - Track and manage user devices and sessions
- **Security Monitoring** - IP filtering, geolocation, and anomaly detection

### Developer Experience
- **Plugin Architecture** - Extensible authentication methods
- **Clean Architecture** - Service-oriented design with repository pattern
- **Type Safety** - Full Go type safety with comprehensive error handling
- **Database Agnostic** - PostgreSQL, MySQL, and SQLite support
- **Configuration Management** - Flexible YAML/JSON configuration with environment overrides
- **Comprehensive Testing** - Unit and integration test coverage

### Deployment Modes
- **Standalone Mode** - Single-tenant applications
- **SaaS Mode** - Multi-tenant platforms with organization isolation

## 📦 Installation

### Prerequisites

- Go 1.25 or later
- Supported database (PostgreSQL, MySQL, or SQLite)
- Redis (optional, for distributed session storage)

### Install AuthSome

```bash
go get github.com/xraph/authsome
```

### Install Dependencies

```bash
# Core dependencies
go get github.com/xraph/forge
go get github.com/uptrace/bun

# Database drivers (choose one)
go get github.com/lib/pq                    # PostgreSQL
go get github.com/go-sql-driver/mysql       # MySQL
go get github.com/mattn/go-sqlite3          # SQLite

# Optional: Redis for session storage
go get github.com/redis/go-redis/v9
```

## 🏃‍♂️ Quick Start

### Basic Setup

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/xraph/authsome"
    "github.com/xraph/forge"
    "github.com/uptrace/bun"
    "github.com/uptrace/bun/dialect/pgdialect"
    "github.com/uptrace/bun/driver/pgdriver"
)

func main() {
    // Create Forge app
    app := forge.New()

    // Setup database
    db := bun.NewDB(pgdriver.NewConnector(
        pgdriver.WithDSN(os.Getenv("DATABASE_URL")),
    ), pgdialect.New())

    // Initialize AuthSome
    auth := authsome.New(
        authsome.WithDatabase(db),
        authsome.WithForgeConfig(app.Config()),
        authsome.WithMode(authsome.ModeStandalone),
    )

    // Initialize services
    if err := auth.Initialize(context.Background()); err != nil {
        log.Fatal("Failed to initialize AuthSome:", err)
    }

    // Mount AuthSome routes
    if err := auth.Mount(app, "/auth"); err != nil {
        log.Fatal("Failed to mount AuthSome:", err)
    }

    // Start server
    log.Println("Server starting on :8080")
    log.Fatal(app.Listen(":8080"))
}
```

### Environment Configuration

Create a `.env` file:

```bash
# Database
DATABASE_URL=postgres://user:password@localhost/myapp?sslmode=disable

# Session
SESSION_SECRET=your-super-secret-session-key

# Email (optional)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# Redis (optional)
REDIS_URL=redis://localhost:6379
```

## 🔧 Usage

### User Registration

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123"
  }'
```

### User Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123"
  }'
```

### Protected Routes

```go
// Middleware for protected routes
func requireAuth(auth *authsome.Auth) forge.MiddlewareFunc {
    return func(c *forge.Context) error {
        session := auth.GetSession(c)
        if session == nil {
            return c.JSON(401, map[string]string{
                "error": "Authentication required",
            })
        }
        return c.Next()
    }
}

// Protected route example
app.GET("/api/profile", requireAuth(auth), func(c *forge.Context) error {
    user := auth.GetUser(c)
    return c.JSON(200, user)
})
```

### Plugin Usage

```go
import (
    "github.com/xraph/authsome/plugins/twofa"
    "github.com/xraph/authsome/plugins/username"
    "github.com/xraph/authsome/plugins/magiclink"
)

// Initialize with plugins
auth := authsome.New(
    authsome.WithDatabase(db),
    authsome.WithForgeConfig(app.Config()),
    authsome.WithPlugins(
        twofa.NewPlugin(),
        username.NewPlugin(),
        magiclink.NewPlugin(),
    ),
)
```

## ⚙️ Configuration

### YAML Configuration

```yaml
# config.yaml
auth:
  mode: "standalone"  # or "saas"
  basePath: "/auth"
  secret: "your-session-secret"
  rbacEnforce: false

  # Session configuration
  session:
    maxAge: 86400      # 24 hours
    secure: true
    httpOnly: true
    sameSite: "strict"

  # Rate limiting
  rateLimit:
    enabled: true
    requests: 100
    window: "1m"
    storage: "memory"  # or "redis"

  # Email configuration
  email:
    provider: "smtp"
    smtp:
      host: "smtp.gmail.com"
      port: 587
      username: "your-email@gmail.com"
      password: "your-app-password"

  # Security settings
  security:
    enabled: true
    ipWhitelist: []
    ipBlacklist: []
    allowedCountries: ["US", "CA", "GB"]
    blockedCountries: ["CN", "RU"]

  # Plugin configurations
  plugins:
    twofa:
      enabled: true
      issuer: "MyApp"
      digits: 6
      period: 30

    username:
      enabled: true
      minLength: 3
      maxLength: 30
      allowSpecialChars: false

    magiclink:
      enabled: true
      tokenExpiry: "15m"
      maxAttempts: 3
```

### Environment Variables

```bash
# Core settings
AUTHSOME_MODE=standalone
AUTHSOME_SECRET=your-session-secret
AUTHSOME_BASE_PATH=/auth

# Database
DATABASE_URL=postgres://user:pass@localhost/db

# Session
SESSION_MAX_AGE=86400
SESSION_SECURE=true

# Rate limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Email
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# Redis (optional)
REDIS_URL=redis://localhost:6379

# Security
SECURITY_ENABLED=true
ALLOWED_COUNTRIES=US,CA,GB
BLOCKED_COUNTRIES=CN,RU
```

## 📚 API Reference

### Authentication Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/register` | Register a new user |
| `POST` | `/auth/login` | Authenticate user |
| `POST` | `/auth/logout` | End user session |
| `POST` | `/auth/refresh` | Refresh session |
| `GET` | `/auth/me` | Get current user |
| `PUT` | `/auth/me` | Update user profile |
| `POST` | `/auth/change-password` | Change user password |
| `POST` | `/auth/forgot-password` | Request password reset |
| `POST` | `/auth/reset-password` | Reset password with token |

### Two-Factor Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/2fa/setup` | Setup 2FA for user |
| `POST` | `/auth/2fa/verify` | Verify 2FA token |
| `POST` | `/auth/2fa/disable` | Disable 2FA |
| `GET` | `/auth/2fa/backup-codes` | Get backup codes |

### Organization Management (SaaS Mode)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/orgs` | List user organizations |
| `POST` | `/api/orgs` | Create organization |
| `GET` | `/api/orgs/{id}` | Get organization details |
| `PUT` | `/api/orgs/{id}` | Update organization |
| `DELETE` | `/api/orgs/{id}` | Delete organization |
| `GET` | `/api/orgs/{id}/members` | List organization members |
| `POST` | `/api/orgs/{id}/invite` | Invite user to organization |
| `DELETE` | `/api/orgs/{id}/members/{userId}` | Remove member |

### Session Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/auth/sessions` | List user sessions |
| `DELETE` | `/auth/sessions/{id}` | Revoke specific session |
| `DELETE` | `/auth/sessions/all` | Revoke all sessions |

### Audit & Security

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/auth/audit` | Get audit logs |
| `GET` | `/auth/devices` | List user devices |
| `DELETE` | `/auth/devices/{id}` | Remove device |

### Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/auth/webhooks` | List webhooks |
| `POST` | `/auth/webhooks` | Create webhook |
| `PUT` | `/auth/webhooks/{id}` | Update webhook |
| `DELETE` | `/auth/webhooks/{id}` | Delete webhook |

### API Keys

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/auth/api-keys` | List API keys |
| `POST` | `/auth/api-keys` | Create API key |
| `DELETE` | `/auth/api-keys/{id}` | Revoke API key |

## 🤝 Contributing

We welcome contributions to AuthSome! Please follow these guidelines:

### Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/your-username/authsome.git
   cd authsome
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up development database**
   ```bash
   # Using Docker
   docker run --name authsome-postgres -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres:15
   
   # Create test database
   createdb -h localhost -U postgres authsome_test
   ```

4. **Run tests**
   ```bash
   go test ./...
   ```

5. **Run integration tests**
   ```bash
   make test-integration
   ```

### Code Style

- Follow standard Go conventions and use `gofmt`
- Write comprehensive tests for new features
- Add function-level comments for exported functions
- Use meaningful variable and function names
- Follow the existing architecture patterns

### Submitting Changes

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Write tests for new functionality
   - Update documentation as needed
   - Ensure all tests pass

3. **Commit your changes**
   ```bash
   git commit -m "feat: add your feature description"
   ```

4. **Push and create a pull request**
   ```bash
   git push origin feature/your-feature-name
   ```

### Pull Request Guidelines

- Provide a clear description of the changes
- Include tests for new functionality
- Update documentation if needed
- Ensure CI passes
- Link to any relevant issues

### Reporting Issues

When reporting issues, please include:

- Go version
- AuthSome version
- Database type and version
- Minimal reproduction case
- Error messages and stack traces

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2024 AuthSome Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## 🆘 Support

### Documentation

- **Official Documentation**: [https://authsome.dev](https://authsome.dev)
- **API Reference**: [https://authsome.dev/api](https://authsome.dev/api)
- **Examples**: [https://github.com/xraph/authsome/tree/main/examples](https://github.com/xraph/authsome/tree/main/examples)

### Community

- **GitHub Discussions**: [https://github.com/xraph/authsome/discussions](https://github.com/xraph/authsome/discussions)
- **Discord Server**: [https://discord.gg/authsome](https://discord.gg/authsome)
- **Stack Overflow**: Tag your questions with `authsome-go`

### Commercial Support

For enterprise support, consulting, and custom development:

- **Email**: support@authsome.dev
- **Enterprise**: enterprise@authsome.dev
- **Website**: [https://authsome.dev/enterprise](https://authsome.dev/enterprise)

### Security Issues

For security-related issues, please email: security@authsome.dev

**Do not report security issues through public GitHub issues.**

---

<div align="center">

**[Website](https://authsome.dev)** • **[Documentation](https://authsome.dev/docs)** • **[Examples](https://github.com/xraph/authsome/tree/main/examples)** • **[Contributing](CONTRIBUTING.md)**

Made with ❤️ by the AuthSome team

</div>