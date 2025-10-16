# Auto-Generating Manifests via Code Introspection

Instead of manually maintaining YAML manifests, AuthSome can introspect your Go code to automatically generate manifests. This ensures manifests always match the actual server implementation.

## Overview

The introspector analyzes:

1. **Handler Functions** - Extracts request/response types
2. **Route Registrations** - Gets HTTP methods and paths
3. **Struct Definitions** - Extracts type information with JSON tags
4. **Plugin Metadata** - Reads plugin IDs and descriptions
5. **Comments** - Captures documentation

## Usage

### Introspect a Single Plugin

```bash
authsome-cli generate introspect --plugin social
```

**Output:**
```
Introspecting plugin: social
✓ Generated manifest: ./clients/manifest/data/social.yaml
  - 5 routes
  - 1 types
```

### Introspect All Plugins

```bash
authsome-cli generate introspect --plugin all
```

This discovers all plugins in `plugins/` and generates manifests for each.

### Introspect Core Handlers

```bash
authsome-cli generate introspect --core
```

Analyzes `handlers/` directory for core authentication routes.

### Dry Run (Preview Only)

```bash
authsome-cli generate introspect --plugin social --dry-run
```

Prints the generated manifest without writing files.

### Custom Output Directory

```bash
authsome-cli generate introspect --plugin social --output ./my-manifests
```

## How It Works

### 1. Handler Analysis

The introspector parses Go AST to find handler methods:

```go
// This handler method...
func (h *AuthHandler) SignUp(c *forge.Context) error {
    var req SignUpRequest
    if err := c.BindJSON(&req); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    
    res, err := h.auth.SignUp(c.Request().Context(), &req)
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    
    return c.JSON(200, res)
}
```

**Extracts:**
- Handler name: `SignUp`
- Request type: `SignUpRequest` (from `c.BindJSON`)
- Response type: `SignUpResponse` (from return type)
- Possible error codes: 400, 200

### 2. Route Registration Analysis

Parses `routes/*.go` files to find route registrations:

```go
// This route registration...
auth.POST("/signup", h.SignUp)
```

**Extracts:**
- HTTP method: `POST`
- Path: `/signup`
- Handler: `h.SignUp` → links to handler method

### 3. Type Extraction

Analyzes struct definitions:

```go
// This struct...
type SignUpRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
    Name     string `json:"name,omitempty"`
}
```

**Generates:**
```yaml
request:
  email: string!       # Required (no omitempty)
  password: string!    # Required
  name: string         # Optional (has omitempty)
```

### 4. Manifest Generation

Combines all extracted information into a complete manifest:

```yaml
plugin_id: social
version: 1.0.0
description: Social OAuth authentication
base_path: /api/auth

types:
  - name: SocialAccount
    fields:
      id: string!
      provider: string!
      email: string

routes:
  - name: SignIn
    method: POST
    path: /signin/social
    request:
      provider: string!
      scopes: string[]
    response:
      url: string!
```

## Advantages Over Manual Manifests

### 1. Always in Sync ✅

Manual manifests can drift from code. Introspection ensures they match exactly.

### 2. Single Source of Truth ✅

Code is the source of truth. Manifests are generated from it.

### 3. Reduced Maintenance ✅

No need to manually update YAML files when adding routes.

### 4. Type Safety ✅

Struct tags and Go types are authoritative - no transcription errors.

### 5. Documentation from Code ✅

Comments in Go code become manifest descriptions automatically.

## Workflow: Introspection → Generation

### Development Flow

```bash
# 1. Write your handler
vim plugins/social/handlers.go

# 2. Register routes
vim plugins/social/plugin.go

# 3. Auto-generate manifest
authsome-cli generate introspect --plugin social

# 4. Generate clients from manifest
authsome-cli generate client --lang all

# Done! Clients are now in sync with code
```

### CI/CD Integration

```bash
#!/bin/bash
# scripts/generate-clients.sh

# Introspect all plugins
authsome-cli generate introspect --plugin all

# Validate generated manifests
authsome-cli generate client --validate

# Generate all clients
authsome-cli generate client --lang all

# Clients are ready for distribution
```

## Limitations & Workarounds

### Current Limitations

1. **Complex Response Types** - Only simple struct responses detected
2. **Dynamic Routes** - Routes with variables require annotation
3. **Custom Validation** - Validation rules need manual addition
4. **Error Responses** - Error details inferred, not extracted

### Workarounds

#### 1. Annotation Comments

Add special comments for complex cases:

```go
// @route POST /users/{id}
// @param id string! User ID
// @response User!
// @error 404 User not found
func (h *Handler) GetUser(c *forge.Context) error {
    // ...
}
```

#### 2. Manifest Augmentation

Generate base manifest, then manually add:
- Detailed error descriptions
- Validation rules
- Additional documentation

```bash
# Generate base
authsome-cli generate introspect --plugin social

# Edit to add details
vim clients/manifest/data/social.yaml

# Then generate clients
authsome-cli generate client --lang all
```

#### 3. Partial Introspection

Introspect what you can, manually define edge cases:

```yaml
# Auto-generated section
routes:
  - name: SignIn
    method: POST
    path: /signin/social
    # ... auto-generated fields

# Manually added
    errors:
      - code: 400
        description: Provider configuration invalid
      - code: 403
        description: Provider disabled for organization
```

## Examples

### Example 1: Simple Plugin

**Go Code:**
```go
// plugins/example/handlers.go
package example

type CreateRequest struct {
    Name string `json:"name"`
    Value int   `json:"value"`
}

type CreateResponse struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

func (h *Handler) Create(c *forge.Context) error {
    var req CreateRequest
    if err := c.BindJSON(&req); err != nil {
        return c.JSON(400, ErrorResponse{Error: "invalid"})
    }
    
    resp := CreateResponse{ID: "123", Name: req.Name}
    return c.JSON(201, resp)
}
```

**Route Registration:**
```go
// plugins/example/plugin.go
func (p *Plugin) RegisterRoutes(router interface{}) error {
    grp := router.(*forge.Group)
    h := NewHandler()
    grp.POST("/create", h.Create)
    return nil
}
```

**Generated Manifest:**
```yaml
plugin_id: example
version: 1.0.0

types:
  - name: CreateRequest
    fields:
      name: string!
      value: int!

  - name: CreateResponse
    fields:
      id: string!
      name: string!

routes:
  - name: Create
    method: POST
    path: /create
    request:
      name: string!
      value: int!
    response:
      id: string!
      name: string!
```

### Example 2: Plugin with Multiple Routes

```bash
$ authsome-cli generate introspect --plugin social --dry-run

--- social.yaml ---
plugin_id: social
version: 1.0.0
description: Social OAuth authentication
base_path: /api/auth

routes:
  - name: SignIn
    description: Initiate OAuth flow for sign-in
    method: POST
    path: /signin/social
    request:
      provider: string!
      scopes: string[]
      redirectUrl: string
    response:
      url: string!

  - name: Callback
    description: Handle OAuth provider callback
    method: GET
    path: /callback/{provider}
    params:
      provider: string!
    query:
      code: string!
      state: string!
    response:
      user: User!
      session: Session!

types:
  - name: SocialAccount
    fields:
      id: string!
      userId: string!
      provider: string!
      providerAccountId: string!
```

## Best Practices

### 1. Use Descriptive Comments

Comments become manifest descriptions:

```go
// SignUp creates a new user account with email and password
func (h *Handler) SignUp(c *forge.Context) error {
    // ...
}
```

Generates:
```yaml
- name: SignUp
  description: Creates a new user account with email and password
```

### 2. Consistent Naming

Use clear, consistent names for types and methods:

```go
// Good
type SignUpRequest struct {}
type SignUpResponse struct {}
func (h *Handler) SignUp(...) {}

// Less clear
type Input struct {}
type Output struct {}
func (h *Handler) DoStuff(...) {}
```

### 3. JSON Tags

Always include JSON tags - they become field names in manifests:

```go
type User struct {
    ID    string `json:"id"`        // ✓ Good
    Email string `json:"email"`     // ✓ Good
    Name  string                    // ✗ No JSON tag - skipped
}
```

### 4. Omitempty for Optional

Use `omitempty` to mark optional fields:

```go
type Request struct {
    Required string  `json:"required"`           // Required
    Optional string  `json:"optional,omitempty"` // Optional
}
```

Generates:
```yaml
request:
  required: string!   # ! = required
  optional: string    # optional
```

### 5. Regenerate Regularly

```bash
# After code changes
authsome-cli generate introspect --plugin all

# Check diff
git diff clients/manifest/data/

# Regenerate clients
authsome-cli generate client --lang all
```

## Future Enhancements

Planned improvements for introspection:

1. **Annotation Support** - Parse special comments for metadata
2. **Validation Rules** - Extract from struct tags (`validate:`)
3. **OpenAPI Export** - Generate OpenAPI specs directly from code
4. **Error Inference** - Better error code detection
5. **Watch Mode** - Auto-regenerate on code changes
6. **Plugin Auto-Discovery** - Scan for new plugins automatically

## Troubleshooting

### "Failed to parse file"

Ensure Go code compiles:
```bash
go build ./...
```

### "No routes found"

Check route registration uses standard patterns:
```go
app.POST("/path", handler)  // ✓ Detected
app.Handle("POST", "/path", handler)  // ✗ Not detected yet
```

### "Type not found"

Ensure types are in same package or imported:
```go
// Same file - works
type Request struct {}
func (h *Handler) Method(...) {}

// Different package - may need import path
```

### "Manifest validation failed"

Generated manifest may need manual fixes:
```bash
# Generate
authsome-cli generate introspect --plugin social

# Validate
authsome-cli generate client --validate

# Fix any errors in the YAML
vim clients/manifest/data/social.yaml
```

## Summary

**Introspection automates manifest generation by analyzing Go code:**

✅ Handlers → Request/response types  
✅ Routes → HTTP methods and paths  
✅ Structs → Type definitions  
✅ Comments → Documentation  
✅ Always in sync with code  

**Use it in your workflow:**

```bash
# Write code
vim plugins/myfeature/

# Generate manifest
authsome-cli generate introspect --plugin myfeature

# Generate clients
authsome-cli generate client --lang all

# Done!
```

---

**Introspection + Generation = Always-Synchronized Clients** 🎯

