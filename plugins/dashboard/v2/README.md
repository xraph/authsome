# Dashboard Plugin

A lightweight, server-rendered admin interface for AuthSome built with Alpine.js and Tailwind CSS 4 CDN.

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

Then access the dashboard at `http://localhost:8080/dashboard/` (requires admin role).

## Features

✅ Server-side rendering with Go templates  
✅ ~40KB total bundle size (Alpine.js + Tailwind CSS CDN)  
✅ Built-in auth, RBAC, CSRF, rate limiting, and audit logging  
✅ Responsive, mobile-first design  
✅ Real-time statistics and user management  
✅ **Dark mode support** with system preference detection and localStorage persistence  

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

See [DASHBOARD_STATUS.md](./DASHBOARD_STATUS.md) for detailed security documentation.

## Documentation

- **[DASHBOARD_STATUS.md](./DASHBOARD_STATUS.md)** - Complete current state, architecture, features, and deployment guide
- **[components/README.md](./components/README.md)** - Component usage and development guide

## Pages

### App-Based Route Structure (Breaking Change v2.0)

All dashboard routes are now app-scoped. You must select an app before accessing dashboard features.

**Dashboard Index:**
- `/dashboard/` - App selection (multiapp mode) or auto-redirect to default app (standalone mode)

**App-Scoped Routes:**
- `/dashboard/app/:appId/` - Statistics and quick actions for the app
- `/dashboard/app/:appId/users` - User management within the app
- `/dashboard/app/:appId/users/:id` - User details
- `/dashboard/app/:appId/organizations` - User-created organizations (workspaces) within the app
- `/dashboard/app/:appId/organizations/:orgId` - Organization details
- `/dashboard/app/:appId/apps-management` - Platform apps management (admin only)
- `/dashboard/app/:appId/apps-management/:targetAppId` - App details
- `/dashboard/app/:appId/apps-management/create` - Create new app (requires multiapp plugin)
- `/dashboard/app/:appId/sessions` - Active sessions in the app
- `/dashboard/app/:appId/settings` - App-specific settings
- `/dashboard/app/:appId/plugins` - Plugin management for the app

**App Switcher:**
In multiapp mode, a dropdown appears in the header allowing quick switching between apps you belong to.

### Breaking Changes from v1.x

**⚠️ IMPORTANT:** This release introduces breaking changes to the URL structure.

1. **Old routes removed:**
   - `/dashboard/users` → `/dashboard/app/{appId}/users`
   - `/dashboard/sessions` → `/dashboard/app/{appId}/sessions`
   - All routes now require an `appId` in the URL path

2. **App vs. Organization distinction:**
   - **Apps** (platform-level tenants) can be managed via the dashboard "Apps Management" section
     - List, view, edit, and delete apps are always available
     - Create new apps is only available when `multiapp` plugin is enabled
   - **Organizations** (user-created workspaces) are managed via the dashboard "Organizations" section
   - The dashboard `/` now displays app cards for navigation
   - Within each app, you can create and manage user organizations

3. **Context requirement:**
   - All dashboard features now operate within an app context
   - Users must be members of an app to access its dashboard
   - Data is automatically scoped to the selected app

### Migration Guide

**If you have bookmarks or hardcoded links:**
- Update `/dashboard/users` to `/dashboard/app/{appId}/users`
- Pattern: `/dashboard/{page}` → `/dashboard/app/{appId}/{page}`

**For multiapp mode:**
- Visit `/dashboard/` to see all your apps
- Click on an app card to enter its dashboard

**For standalone mode:**
- `/dashboard/` will auto-redirect to the default app
- URLs will automatically include the app ID

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
