# Multiapp Plugin

The Multiapp plugin provides multi-tenancy support for AuthSome, enabling applications to support multiple isolated tenant apps with their own users, teams, and configurations.

## Overview

This plugin enables both **Standalone Mode** and **SaaS Mode**:

- **Standalone Mode**: Single default app with all users belonging to it
- **SaaS Mode**: Multiple tenant apps with isolated data and configurations

## Features

- **App Management**: Create, update, and manage tenant apps
- **Member Management**: Add users to apps with role-based access (owner, admin, member)
- **Team Management**: Organize members into teams within apps
- **Invitation System**: Invite users to join apps via secure tokens
- **Environment Support**: Multiple environments per app (development, staging, production)
- **Config Overrides**: App-specific configuration overrides
- **Service Decoration**: Enhances core services with multi-tenancy awareness

## Architecture

The plugin follows a clean architecture pattern:

```
plugins/multiapp/
├── service.go          # AppService implementation
├── plugin.go           # Plugin registration and initialization
├── handlers/           # HTTP request handlers
│   ├── app.go
│   ├── member.go
│   └── team.go
├── decorators/         # Service decorators for multi-tenancy
│   ├── user.go
│   ├── session.go
│   └── auth.go
├── config/             # Config service for app-specific overrides
└── middleware/         # App context middleware
```

## Data Model

### Core Entities

All entities use `schema.*` types from the global schema package:

- **App** (`schema.App`): Tenant application
- **Member** (`schema.Member`): User membership in an app
- **Team** (`schema.Team`): Team within an app
- **TeamMember** (`schema.TeamMember`): Team membership
- **Invitation** (`schema.Invitation`): App invitation
- **Environment** (`schema.Environment`): App environment (dev, staging, prod)

### Repository Layer

Repositories are consolidated in the main `repository/` package:

- `repository.AppRepository`: App data access
- `repository.MemberRepository`: Member data access
- `repository.TeamRepository`: Team data access
- `repository.InvitationRepository`: Invitation data access
- `repository.EnvironmentRepository`: Environment data access

## Usage

### Installation

Add the plugin to your AuthSome initialization:

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/multiapp"
)

auth := authsome.New(
    authsome.WithDatabase(db),
    authsome.WithPlugins(
        multiapp.NewPlugin(
            multiapp.WithEnableAppCreation(true),
            multiapp.WithMaxMembersPerApp(100),
            multiapp.WithMaxTeamsPerApp(10),
        ),
    ),
)
```

### Configuration

```yaml
auth:
  multiapp:
    platformAppId: ""  # Auto-set during initialization
    defaultAppName: "Platform App"
    enableAppCreation: true  # Enable SaaS mode
    maxMembersPerApp: 1000
    maxTeamsPerApp: 100
    requireInvitation: false
    invitationExpiryHours: 72
    autoCreateDefaultApp: true
    defaultEnvironmentName: "Development"
```

### API Routes

The plugin registers the following routes:

#### Apps
- `POST /apps` - Create new app
- `GET /apps` - List apps
- `GET /apps/:appId` - Get app details
- `PUT /apps/:appId` - Update app
- `DELETE /apps/:appId` - Delete app

#### Members
- `GET /apps/:appId/members` - List app members
- `POST /apps/:appId/members/invite` - Invite member
- `PUT /apps/:appId/members/:memberId` - Update member role/status
- `DELETE /apps/:appId/members/:memberId` - Remove member

#### Teams
- `GET /apps/:appId/teams` - List teams
- `POST /apps/:appId/teams` - Create team
- `GET /apps/:appId/teams/:teamId` - Get team details
- `PUT /apps/:appId/teams/:teamId` - Update team
- `DELETE /apps/:appId/teams/:teamId` - Delete team
- `POST /apps/:appId/teams/:teamId/members` - Add team member
- `DELETE /apps/:appId/teams/:teamId/members/:memberId` - Remove team member

#### Invitations
- `GET /invitations/:token` - Get invitation details
- `POST /invitations/:token/accept` - Accept invitation
- `POST /invitations/:token/decline` - Decline invitation

## Service Decorators

The plugin decorates core services to add multi-tenancy awareness:

### User Service Decorator
```go
// Automatically adds new users to the appropriate app
// Handles first user as platform owner
decoratedUserService := decorators.NewMultiTenantUserService(
    userService,
    appService,
)
```

### Session Service Decorator
```go
// Adds app context to sessions
// Validates app membership on session creation
decoratedSessionService := decorators.NewMultiTenantSessionService(
    sessionService,
    appService,
)
```

### Auth Service Decorator
```go
// Ensures authentication respects app boundaries
// Validates app membership on sign-in
decoratedAuthService := decorators.NewMultiTenantAuthService(
    authService,
    appService,
)
```

## Hooks

The plugin registers hooks to handle multi-tenancy events:

- **AfterUserCreate**: Assigns new users to appropriate apps
- **AfterUserDelete**: Removes user from all apps
- **AfterSessionCreate**: Logs session creation for audit

## Environment Management

Each app can have multiple environments:

```go
// Bootstrap creates a default "Development" environment
// Additional environments can be created via the Environment service

envService := registry.EnvironmentService()
env, err := envService.Create(ctx, &environment.CreateEnvironmentRequest{
    AppID: appID,
    Name: "Production",
    Type: "production",
})
```

## App Context

Use context helpers to work with app context in handlers:

```go
import "github.com/xraph/authsome/core/interfaces"

// Get app ID from context
appID, err := interfaces.GetAppID(ctx)

// Set app ID in context
ctx = interfaces.WithAppID(ctx, appID)
```

## Best Practices

1. **Always validate app membership** before allowing access to app resources
2. **Use service decorators** instead of modifying core services directly
3. **Leverage app context** for multi-tenant aware operations
4. **Configure appropriate limits** for members and teams per app
5. **Enable invitations** for controlled app access in SaaS mode

## Migration from Legacy Organization System

If migrating from the old organization-based system:

1. The `organizations` table has been renamed to `apps`
2. All references to "organization" in code now use "app"
3. Schema models are now in the global `schema` package
4. Repository implementations are in the main `repository` package
5. The plugin ID changed from "multitenancy" to "multiapp"

## License

See main AuthSome LICENSE file.

