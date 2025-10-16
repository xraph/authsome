# AuthSome Routing Architecture

## Single forge.App Instance Guarantee

AuthSome ensures that **only ONE routing instance** is used per application, regardless of how many plugins are registered.

## How It Works

### With Real Forge (`*forge.App`)

```go
app := forge.New()           // ONE App instance
auth := authsome.New(...)
auth.Mount(app, "/api/auth")

// Internal behavior:
// 1. Core routes: app.Group("/api/auth")
// 2. Plugin routes: app.Group("/api/auth")
// All plugins share the SAME app instance via Groups
```

**Result:** All routes (core + plugins) use the same `*forge.App` instance.

### With http.ServeMux (`*http.ServeMux`)

```go
mux := http.NewServeMux()    // ONE ServeMux
auth := authsome.New(...)
auth.Mount(mux, "/api/auth")

// Internal behavior:
// 1. Create ONE forge.App wrapper: f := forge.NewApp(mux)
// 2. f registers catch-all "/" handler on mux
// 3. Core routes: f.Group("/api/auth")
// 4. Plugin routes: mux.HandleFunc("/dashboard/", ...)
```

**Result:** 
- ONE forge.App wrapper is created for core routes
- Plugins register directly on the same underlying ServeMux
- http.ServeMux dispatches:
  - Specific patterns → Direct plugin handlers
  - Catch-all "/" → forge.App internal routing

## Request Flow Examples

### Real Forge Framework

```
Request: GET /api/auth/dashboard/assets/file.js
  ↓
forge.App (single instance)
  ↓
Route matching in app.routes[]
  ↓
Dashboard plugin handler
```

### http.ServeMux

```
Request: GET /dashboard/assets/file.js
  ↓
http.ServeMux
  ↓
Checks registered patterns:
  - /dashboard/ (specific, registered by plugin) ← MATCHES
  - / (catch-all, registered by forge.App)
  ↓
Dashboard plugin handler (direct, bypasses forge.App)

Request: GET /api/auth/session
  ↓
http.ServeMux
  ↓
Checks registered patterns:
  - /dashboard/ (doesn't match)
  - / (catch-all) ← MATCHES
  ↓
forge.App catch-all dispatcher
  ↓
Route matching in forge.App.routes[]
  ↓
Core auth handler
```

## Why This Design?

### Advantages

1. **Performance**: Plugin routes with http.ServeMux go directly to handlers
2. **Simplicity**: Plugins don't need to understand internal forge implementation
3. **Compatibility**: Works with standard library http.ServeMux
4. **No Conflicts**: http.ServeMux's pattern matching prevents route collisions

### Trade-offs

1. **BasePath Support**: Plugins using http.ServeMux can't automatically inherit basePath
   - **Solution**: Use real Forge framework for full basePath support
2. **Separate Routing**: Core routes and plugin routes use different dispatch mechanisms
   - **Impact**: Minimal - both are efficient and coexist cleanly

## Verification

You can verify the single-instance guarantee by adding debug logging:

```go
// authsome.go - Mount method
case *http.ServeMux:
    f := forge.NewApp(v)
    log.Printf("Created forge.App instance: %p", f)  // Printed ONCE
    
    // ... route registration ...
    
    for _, p := range a.pluginRegistry.List() {
        log.Printf("Registering plugin %s", p.ID())
        _ = p.RegisterRoutes(v)  // Receives same v
    }
```

## Best Practices

### For Full Feature Support
**Use the real Forge framework:**
```go
import "github.com/xraph/forge"

app := forge.New()
auth.Mount(app, "/api/auth")
// ✅ One app instance shared across all routes
// ✅ Full basePath support for plugins
// ✅ Consistent routing system
```

### For Standard Library
**http.ServeMux is fine for simple cases:**
```go
mux := http.NewServeMux()
auth.Mount(mux, "/api/auth")
// ✅ One forge.App wrapper for core routes
// ✅ Plugins register directly on mux
// ⚠️  Plugins don't inherit basePath
```

## Summary

| Aspect | Real Forge | http.ServeMux |
|--------|------------|---------------|
| **Instance Count** | ONE App | ONE App wrapper + raw mux |
| **Plugin Routing** | Via Groups | Direct registration |
| **BasePath Support** | ✅ Full | ⚠️ Manual |
| **Performance** | Excellent | Excellent |
| **Consistency** | Single system | Hybrid (efficient) |

**Bottom Line:** AuthSome guarantees only ONE primary routing instance per application, whether using Forge or http.ServeMux. The architecture is designed to be efficient, compatible, and conflict-free.

