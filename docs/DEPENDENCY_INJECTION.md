# Dependency Injection in AuthSome

## Overview

AuthSome integrates with the **Forge DI Container** to provide enterprise-grade dependency injection for all services, plugins, and handlers. This enables loose coupling, testability, and clean architecture.

## Architecture

```
┌─────────────────────────────────────────────┐
│         Forge Application                   │
│  ┌──────────────────────────────────────┐  │
│  │      Forge DI Container              │  │
│  │  ┌────────────────────────────────┐  │  │
│  │  │   AuthSome Services            │  │  │
│  │  │  - User Service                │  │  │
│  │  │  - Session Service             │  │  │
│  │  │  - Auth Service                │  │  │
│  │  │  - Audit Service               │  │  │
│  │  │  - RBAC Service                │  │  │
│  │  │  - Webhook Service             │  │  │
│  │  │  - Notification Service        │  │  │
│  │  │  - JWT Service                 │  │  │
│  │  │  - API Key Service             │  │  │
│  │  │  - Organization Service        │  │  │
│  │  │  - Rate Limit Service          │  │  │
│  │  │  - Device Service              │  │  │
│  │  │  - Security Service            │  │  │
│  │  └────────────────────────────────┘  │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
           │
           ├──> Handlers (resolve services)
           ├──> Plugins (resolve services)
           └──> Middleware (resolve services)
```

## Service Registration

All AuthSome services are automatically registered into the Forge DI container during initialization.

### Service Names

Services are registered with consistent naming:

```go
const (
    ServiceDatabase      = "authsome.database"
    ServiceUser          = "authsome.user"
    ServiceSession       = "authsome.session"
    ServiceAuth          = "authsome.auth"
    ServiceOrganization  = "authsome.organization"
    ServiceRateLimit     = "authsome.ratelimit"
    ServiceDevice        = "authsome.device"
    ServiceSecurity      = "authsome.security"
    ServiceAudit         = "authsome.audit"
    ServiceRBAC          = "authsome.rbac"
    ServiceWebhook       = "authsome.webhook"
    ServiceNotification  = "authsome.notification"
    ServiceJWT           = "authsome.jwt"
    ServiceAPIKey        = "authsome.apikey"
    ServiceHookRegistry  = "authsome.hooks"
    ServicePluginRegistry = "authsome.plugins"
)
```

### Lifecycle

All services are registered as **singletons** by default, ensuring:
- Single instance per application
- Efficient memory usage
- Consistent state across the application

## Resolving Services

### In Handlers

Handlers can resolve services from the Forge context:

```go
func MyHandler(c forge.Context) error {
    container := c.Container()
    
    // Resolve user service
    userSvc, err := authsome.ResolveUserService(container)
    if err != nil {
        return err
    }
    
    // Use the service
    user, err := userSvc.FindByEmail(c.Request().Context(), "user@example.com")
    if err != nil {
        return err
    }
    
    return c.JSON(200, user)
}
```

### In Plugins

Plugins receive the Auth instance during initialization and can access the container:

```go
func (p *MyPlugin) Init(auth interface{}) error {
    // Type assert to get Auth instance
    type authInterface interface {
        GetDB() interface{}
        GetForgeApp() interface{}
        GetServiceRegistry() *registry.ServiceRegistry
    }
    
    authInstance, ok := auth.(authInterface)
    if !ok {
        return fmt.Errorf("invalid auth instance")
    }
    
    // Get Forge app
    forgeApp := authInstance.GetForgeApp()
    
    // Get container
    type containerGetter interface {
        Container() forge.Container
    }
    
    app, ok := forgeApp.(containerGetter)
    if !ok {
        return fmt.Errorf("forge app does not support container")
    }
    
    container := app.Container()
    
    // Resolve services
    userSvc, err := authsome.ResolveUserService(container)
    if err != nil {
        return err
    }
    
    auditSvc, err := authsome.ResolveAuditService(container)
    if err != nil {
        return err
    }
    
    // Initialize plugin with resolved services
    p.userService = userSvc
    p.auditService = auditSvc
    
    return nil
}
```

### Helper Functions

AuthSome provides type-safe helper functions for resolving services:

```go
// Core services
userSvc, err := authsome.ResolveUserService(container)
sessionSvc, err := authsome.ResolveSessionService(container)
authSvc, err := authsome.ResolveAuthService(container)

// Security services
auditSvc, err := authsome.ResolveAuditService(container)
rbacSvc, err := authsome.ResolveRBACService(container)
securitySvc, err := authsome.ResolveSecurityService(container)

// Integration services
webhookSvc, err := authsome.ResolveWebhookService(container)
notificationSvc, err := authsome.ResolveNotificationService(container)
jwtSvc, err := authsome.ResolveJWTService(container)
apikeySvc, err := authsome.ResolveAPIKeyService(container)

// Infrastructure services
rateLimitSvc, err := authsome.ResolveRateLimitService(container)
deviceSvc, err := authsome.ResolveDeviceService(container)
orgSvc, err := authsome.ResolveOrganizationService(container)

// Database
db, err := authsome.ResolveDatabase(container)

// Registries
hookRegistry, err := authsome.ResolveHookRegistry(container)
pluginRegistry, err := authsome.ResolvePluginRegistry(container)
```

### Bulk Resolution

For plugins that need multiple services, use the convenience helper:

```go
deps, err := authsome.ResolvePluginDependencies(container)
if err != nil {
    return err
}

// Access all common dependencies
_ = deps.Database
_ = deps.UserService
_ = deps.SessionService
_ = deps.AuthService
_ = deps.AuditService
_ = deps.RBACService
_ = deps.HookRegistry
```

## Example: Complete Plugin with DI

```go
package myplugin

import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/core/audit"
    "github.com/xraph/forge"
)

type Plugin struct {
    userService  user.ServiceInterface
    auditService *audit.Service
}

func NewPlugin() *Plugin {
    return &Plugin{}
}

func (p *Plugin) ID() string {
    return "myplugin"
}

func (p *Plugin) Init(auth interface{}) error {
    // Get Auth instance
    type authInterface interface {
        GetForgeApp() interface{}
    }
    
    authInstance, ok := auth.(authInterface)
    if !ok {
        return fmt.Errorf("invalid auth instance")
    }
    
    // Get container
    forgeApp := authInstance.GetForgeApp()
    type containerGetter interface {
        Container() forge.Container
    }
    
    app, ok := forgeApp.(containerGetter)
    if !ok {
        return fmt.Errorf("forge app does not support container")
    }
    
    container := app.Container()
    
    // Resolve dependencies
    var err error
    p.userService, err = authsome.ResolveUserService(container)
    if err != nil {
        return fmt.Errorf("failed to resolve user service: %w", err)
    }
    
    p.auditService, err = authsome.ResolveAuditService(container)
    if err != nil {
        return fmt.Errorf("failed to resolve audit service: %w", err)
    }
    
    return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
    router.GET("/myplugin/example", p.handleExample)
    return nil
}

func (p *Plugin) handleExample(c forge.Context) error {
    // Services are already resolved and ready to use
    users, err := p.userService.List(c.Request().Context(), 10, 0)
    if err != nil {
        return err
    }
    
    return c.JSON(200, users)
}

func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
    return nil
}

func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
    return nil
}

func (p *Plugin) Migrate() error {
    return nil
}
```

## Example: Handler with DI

```go
package myhandler

import (
    "github.com/xraph/authsome"
    "github.com/xraph/forge"
)

func HandleUserProfile(c forge.Context) error {
    // Resolve user service from container
    userSvc, err := authsome.ResolveUserService(c.Container())
    if err != nil {
        return c.JSON(500, map[string]string{
            "error": "Failed to resolve user service",
        })
    }
    
    // Resolve audit service
    auditSvc, err := authsome.ResolveAuditService(c.Container())
    if err != nil {
        return c.JSON(500, map[string]string{
            "error": "Failed to resolve audit service",
        })
    }
    
    // Use services
    ctx := c.Request().Context()
    userID := c.Param("id")
    
    user, err := userSvc.FindByID(ctx, userID)
    if err != nil {
        return c.JSON(404, map[string]string{
            "error": "User not found",
        })
    }
    
    // Log audit event
    _ = auditSvc.Log(ctx, &audit.Event{
        UserID: userID,
        Action: "user.profile.view",
    })
    
    return c.JSON(200, user)
}
```

## Testing with DI

The DI container makes testing easier:

```go
func TestMyHandler(t *testing.T) {
    // Create mock container
    container := di.NewContainer()
    
    // Register mock services
    mockUserService := &MockUserService{}
    container.Register("authsome.user", func(c forge.Container) (interface{}, error) {
        return mockUserService, nil
    })
    
    // Create test context with container
    ctx := createTestContext(container)
    
    // Test handler
    err := HandleUserProfile(ctx)
    
    // Assert results
    assert.NoError(t, err)
    assert.True(t, mockUserService.FindByIDCalled)
}
```

## Best Practices

### 1. Always Use Helper Functions

❌ **Don't** resolve services manually:
```go
svc, _ := container.Resolve("authsome.user")
userSvc := svc.(user.ServiceInterface) // Unsafe
```

✅ **Do** use type-safe helpers:
```go
userSvc, err := authsome.ResolveUserService(container)
if err != nil {
    return err
}
```

### 2. Check Errors

❌ **Don't** ignore resolution errors:
```go
userSvc, _ := authsome.ResolveUserService(container)
```

✅ **Do** handle errors properly:
```go
userSvc, err := authsome.ResolveUserService(container)
if err != nil {
    return fmt.Errorf("failed to resolve user service: %w", err)
}
```

### 3. Resolve Once

❌ **Don't** resolve in every request:
```go
func (p *Plugin) HandleRequest(c forge.Context) error {
    userSvc, _ := authsome.ResolveUserService(c.Container())
    // ...
}
```

✅ **Do** resolve during initialization:
```go
type Plugin struct {
    userService user.ServiceInterface
}

func (p *Plugin) Init(auth interface{}) error {
    // Resolve once
    userSvc, err := authsome.ResolveUserService(container)
    if err != nil {
        return err
    }
    p.userService = userSvc
    return nil
}

func (p *Plugin) HandleRequest(c forge.Context) error {
    // Use cached service
    users, _ := p.userService.List(...)
}
```

### 4. Use Bulk Resolution for Multiple Services

❌ **Don't** resolve services one by one:
```go
userSvc, _ := authsome.ResolveUserService(container)
sessionSvc, _ := authsome.ResolveSessionService(container)
authSvc, _ := authsome.ResolveAuthService(container)
// ...
```

✅ **Do** use bulk resolution:
```go
deps, err := authsome.ResolvePluginDependencies(container)
if err != nil {
    return err
}

p.userService = deps.UserService
p.sessionService = deps.SessionService
p.authService = deps.AuthService
```

## Container Inspection

You can inspect the container to see registered services:

```go
// List all registered services
services := container.Services()
for _, name := range services {
    fmt.Println("Service:", name)
}

// Check if a service is registered
if container.Has("authsome.user") {
    fmt.Println("User service is registered")
}

// Get service info
info := container.Inspect("authsome.user")
fmt.Printf("Service: %s, Type: %s, Lifecycle: %s\n",
    info.Name, info.Type, info.Lifecycle)
```

## Troubleshooting

### Service Not Found

If you get "service not found" errors:

1. Ensure AuthSome is properly initialized:
   ```go
   auth := authsome.New(
       authsome.WithForgeApp(app),
       authsome.WithDatabase(db),
   )
   err := auth.Initialize(ctx)
   ```

2. Check that the Forge app has a container:
   ```go
   container := app.Container()
   if container == nil {
       // Container not available
   }
   ```

### Type Assertion Failures

If you get type assertion errors, use the helper functions instead of manual resolution.

### Circular Dependencies

Avoid circular dependencies:
- Service A depends on Service B
- Service B depends on Service A

Use interfaces and dependency inversion to break cycles.

## Migration from Service Registry

If you're migrating from the old service registry pattern:

**Old:**
```go
serviceRegistry := auth.GetServiceRegistry()
userSvc := serviceRegistry.UserService()
```

**New:**
```go
container := auth.GetForgeApp().Container()
userSvc, err := authsome.ResolveUserService(container)
```

Both patterns are supported for backward compatibility. The DI container approach is recommended for new code.

## Performance Considerations

- **Singleton services**: Instantiated once, reused across requests
- **Resolution overhead**: Minimal - services are cached after first resolution
- **Memory usage**: Single instance per service reduces memory footprint
- **Concurrency**: All services are thread-safe and can be used concurrently

## Related Documentation

- [Plugin Development Guide](./PLUGIN_DEVELOPMENT.md)
- [Service Architecture](./SERVICE_ARCHITECTURE.md)
- [Testing Guide](./TESTING.md)
- [Forge DI Documentation](https://forge.dev/docs/di)

## Summary

AuthSome's integration with Forge DI Container provides:

✅ **Type-safe service resolution**  
✅ **Loose coupling between components**  
✅ **Easy testing with mocks**  
✅ **Consistent service lifecycle**  
✅ **Enterprise-grade dependency management**  

Use the provided helper functions, follow best practices, and enjoy clean, maintainable code!

