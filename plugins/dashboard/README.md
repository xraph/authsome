# Dashboard Plugin

A server-rendered admin interface for AuthSome built with ForgeUI, gomponents, and Tailwind CSS 4.

## Quick Start

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/dashboard"
)

auth, err := authsome.New(
    authsome.WithPlugins(
        dashboard.NewPlugin(),
    ),
)
```

Then access the dashboard at `http://localhost:8080/api/auth/ui/` (or your base path + `/ui/`). Admin role required.

## Features

- Server-side rendering with ForgeUI and gomponents
- Bridge API for frontend-backend communication
- Built-in auth, RBAC, CSRF, rate limiting, and audit logging
- Responsive, mobile-first design
- Real-time statistics and user management
- Dark mode support with system preference detection and localStorage persistence
- Plugin extension system for adding custom dashboard pages and widgets  

## Access Control

The dashboard implements **production-grade security** with:

- **Fast Permission Checking**: Role-based access control with 5-minute cache (< 100µs per check)
- **CSRF Protection**: Session-bound tokens with HMAC signatures
- **First-User Admin**: First user automatically gets admin role

### Assigning Admin Role

```bash
# Using the CLI
authsome-cli user assign-role --user-id=<id> --role=admin

# Or programmatically
rbacSvc.AssignRole(ctx, userID, roleID, orgID)
```

### Permission System

```go
// Check permissions with expressive fluent API
checker := dashboard.NewPermissionChecker(rbacSvc, userRoleRepo)

// Simple check
canView := checker.Can(ctx, userID, "view", "users")

// Fluent API
user := checker.For(ctx, userID)
if user.Dashboard().CanAccess() {
    // Grant access
}
```

## Documentation

- **[EXTENSION_GUIDE.md](./EXTENSION_GUIDE.md)** - How to add dashboard extensions from plugins
- **BRIDGE_ARCHITECTURE.md**, **BRIDGE_EXTENSIONS.md** (if present) - Bridge API and extension docs

## Pages and Routes

The dashboard is served under the ForgeUI base path (default: `{authBasePath}/ui`). All routes are app-scoped.

**Dashboard Index:** App selection (multiapp mode) or redirect to default app (standalone).

**App-scoped routes** (under `/ui/app/:appId/` or equivalent) include: home, users, sessions, organizations, apps management, environments, settings, and plugins. Plugin extensions register their own routes and navigation items.

## Dark Mode

The dashboard includes a built-in dark mode switcher located in the top-right header.

### Features

- **System Preference Detection**: Automatically detects and respects OS-level dark mode preferences
- **localStorage Persistence**: User preference is saved locally and persists across sessions
- **Smooth Transitions**: All theme changes are animated with smooth CSS transitions
- **Complete Coverage**: All components, forms, tables, and UI elements are fully styled for dark mode

### How It Works

1. **Initial Load**: Checks localStorage for saved preference, falls back to system preference
2. **Toggle Button**: Click the sun/moon icon in the header to switch themes manually
3. **Automatic Updates**: Listens for system preference changes when no manual preference is set
4. **CSS Classes**: Uses Tailwind's `dark:` prefix for conditional dark mode styling

### Technical Implementation

- **Alpine.js Component**: `themeData()` component manages state and persistence
- **Tailwind CSS**: `darkMode: 'class'` configuration for class-based toggling
- **localStorage Key**: `theme` (values: `'light'` or `'dark'`)
- **CSS Variables**: Custom scrollbar colors for both light and dark themes

## Development

```bash
# Build
go build ./plugins/dashboard/...

# Test
go test ./plugins/dashboard/... -v

# Lint
golangci-lint run ./plugins/dashboard/...
```

## Premium React Dashboard

A premium React-based dashboard with advanced features is available separately at `frontend/dashboard-premium/`. See its README for details.

## License

Part of AuthSome. See main LICENSE file.
