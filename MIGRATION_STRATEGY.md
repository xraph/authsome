# AuthSome Multi-Tenancy Plugin Migration Strategy

## Overview

This document outlines the comprehensive strategy for migrating AuthSome from its current architecture to a plugin-based multi-tenancy system. The goal is to transform the current tightly-coupled organization and multi-tenancy features into a unified `plugins/multitenancy` plugin while maintaining backward compatibility.

## Current State Analysis

### Existing Organization Dependencies
Based on codebase analysis, the following services currently have organization dependencies:

1. **Core Services with Organization References:**
   - `core/jwt/service.go` - JWT keys are organization-scoped
   - `core/forms/service.go` - Forms can be organization-specific
   - `core/apikey/service.go` - API keys are organization-scoped
   - `core/session/service.go` - Sessions have organization context (TODO comments)

2. **Organization-Specific Entities:**
   - `core/organization/` - Complete organization service
   - `schema/organization.go`, `schema/member.go`, `schema/team.go` - Database schemas
   - `handlers/organization.go` - HTTP handlers
   - `repository/organization.go` - Database operations

3. **Configuration System:**
   - `config.go` - Mode-based configuration (Standalone vs SaaS)
   - Organization-scoped configuration overrides (documented but not fully implemented)

## Migration Goals

1. **Single-Tenant Core**: Make AuthSome work perfectly without any organization concepts
2. **Plugin-Based Multi-Tenancy**: Move all organization/multi-tenancy features to a plugin
3. **Service Decoration**: Use decorator pattern to extend core services with multi-tenant awareness
4. **Backward Compatibility**: Ensure existing installations can migrate smoothly
5. **Clean Architecture**: Maintain dependency flow and avoid circular dependencies

## Migration Phases

### Phase 1: Core Service Refactoring (Foundation)
**Duration**: 2-3 days
**Goal**: Remove organization dependencies from core services

#### 1.1 Refactor Core Services
- **User Service**: Remove any organization context, make purely single-tenant
- **Session Service**: Remove organization references, focus on user sessions only
- **Auth Service**: Simplify to work without organization context
- **JWT Service**: Extract organization-specific logic to interfaces
- **API Key Service**: Extract organization-specific logic to interfaces
- **Forms Service**: Extract organization-specific logic to interfaces

#### 1.2 Create Plugin Interfaces
```go
// core/interfaces/multitenancy.go
type MultiTenantUserService interface {
    CreateWithOrganization(ctx context.Context, req *CreateUserRequest, orgID string) (*User, error)
    FindByEmailInOrganization(ctx context.Context, email, orgID string) (*User, error)
}

type MultiTenantSessionService interface {
    CreateWithOrganization(ctx context.Context, req *CreateSessionRequest, orgID string) (*Session, error)
    FindByTokenInOrganization(ctx context.Context, token, orgID string) (*Session, error)
}

// Similar interfaces for other services...
```

#### 1.3 Add Hook System
```go
// core/hooks/hooks.go
type HookRegistry struct {
    beforeUserCreate []func(ctx context.Context, req *CreateUserRequest) error
    afterUserCreate  []func(ctx context.Context, user *User) error
    // ... other hooks
}
```

### Phase 2: Plugin Interface Development
**Duration**: 1-2 days
**Goal**: Create the plugin system foundation

#### 2.1 Enhanced Plugin Interface
```go
// plugins/plugin.go
type Plugin interface {
    ID() string
    Init(auth *authsome.Auth) error
    RegisterRoutes(router forge.Router) error
    RegisterHooks(hooks *HookRegistry) error
    RegisterServiceDecorators(services *ServiceRegistry) error // NEW
    Migrate() error
}
```

#### 2.2 Service Registry
```go
// core/registry/services.go
type ServiceRegistry struct {
    userService    *user.Service
    sessionService *session.Service
    authService    *auth.Service
    // ... other services
}

func (r *ServiceRegistry) ReplaceUserService(svc *user.Service) {
    r.userService = svc
}
// ... other replacement methods
```

### Phase 3: Multi-Tenancy Plugin Development
**Duration**: 4-5 days
**Goal**: Create the unified multi-tenancy plugin

#### 3.1 Plugin Structure
```
plugins/multitenancy/
├── plugin.go              # Main plugin implementation
├── entities/
│   ├── organization.go     # Organization entity
│   ├── member.go          # Member entity
│   ├── team.go            # Team entity
│   └── invitation.go      # Invitation entity
├── services/
│   ├── organization.go     # Organization service
│   ├── member.go          # Member service
│   ├── team.go            # Team service
│   └── config.go          # Organization-scoped config
├── decorators/
│   ├── user.go            # Multi-tenant user decorator
│   ├── session.go         # Multi-tenant session decorator
│   ├── auth.go            # Multi-tenant auth decorator
│   ├── jwt.go             # Multi-tenant JWT decorator
│   └── apikey.go          # Multi-tenant API key decorator
├── handlers/
│   ├── organization.go     # Organization HTTP handlers
│   ├── member.go          # Member HTTP handlers
│   └── team.go            # Team HTTP handlers
├── repository/
│   ├── organization.go     # Organization repository
│   ├── member.go          # Member repository
│   └── team.go            # Team repository
├── middleware/
│   └── organization.go     # Organization context middleware
└── schema/
    ├── organization.go     # Database schemas
    ├── member.go
    ├── team.go
    └── invitation.go
```

#### 3.2 Service Decorators
The decorators will wrap core services to add multi-tenant awareness:

```go
// plugins/multitenancy/decorators/user.go
type MultiTenantUserDecorator struct {
    core   *user.Service
    orgSvc *organization.Service
}

func (d *MultiTenantUserDecorator) Create(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
    // Get organization from context
    orgID := GetOrganizationFromContext(ctx)
    if orgID == "" {
        return nil, errors.New("organization context required")
    }
    
    // Validate organization exists
    _, err := d.orgSvc.FindByID(ctx, orgID)
    if err != nil {
        return nil, fmt.Errorf("invalid organization: %w", err)
    }
    
    // Create user with organization context
    user, err := d.core.Create(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // Create organization membership
    member := &Member{
        ID:             xid.New(),
        OrganizationID: orgID,
        UserID:         user.ID,
        Role:           "member",
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }
    
    if err := d.orgSvc.CreateMember(ctx, member); err != nil {
        // Rollback user creation if member creation fails
        d.core.Delete(ctx, user.ID)
        return nil, fmt.Errorf("failed to create organization membership: %w", err)
    }
    
    return user, nil
}
```

### Phase 4: Configuration System Update
**Duration**: 2-3 days
**Goal**: Update configuration to support plugin-based overrides

#### 4.1 Plugin Configuration Interface
```go
// core/config/plugin.go
type PluginConfigProvider interface {
    GetConfig(ctx context.Context, key string, orgID string) (interface{}, error)
    SetConfig(ctx context.Context, key string, orgID string, value interface{}) error
}
```

#### 4.2 Organization-Scoped Configuration
```go
// plugins/multitenancy/services/config.go
type ConfigService struct {
    globalConfig forge.ConfigManager
    repo         ConfigRepository
}

func (s *ConfigService) GetConfig(ctx context.Context, key string, orgID string) (interface{}, error) {
    // Try organization-specific config first
    if orgConfig, err := s.repo.GetOrgConfig(ctx, orgID, key); err == nil {
        return orgConfig, nil
    }
    
    // Fallback to global config
    var result interface{}
    if err := s.globalConfig.Bind(key, &result); err != nil {
        return nil, err
    }
    return result, nil
}
```

### Phase 5: Migration Tools and Testing
**Duration**: 2-3 days
**Goal**: Create migration tools and comprehensive testing

#### 5.1 Migration CLI
```go
// cmd/migrate/main.go
func main() {
    // Migrate existing installations
    // 1. Create default organization for standalone mode
    // 2. Migrate existing users to default organization
    // 3. Update configuration format
    // 4. Migrate database schemas
}
```

#### 5.2 Backward Compatibility Layer
```go
// compat/organization.go
// Provide compatibility shims for existing code
```

## Implementation Timeline

| Phase | Duration | Dependencies | Deliverables |
|-------|----------|--------------|--------------|
| Phase 1 | 2-3 days | None | Refactored core services, plugin interfaces |
| Phase 2 | 1-2 days | Phase 1 | Enhanced plugin system, service registry |
| Phase 3 | 4-5 days | Phase 1-2 | Complete multi-tenancy plugin |
| Phase 4 | 2-3 days | Phase 3 | Updated configuration system |
| Phase 5 | 2-3 days | Phase 1-4 | Migration tools, testing |

**Total Duration**: 11-16 days

## Risk Mitigation

### 1. Breaking Changes
- **Risk**: Existing installations break
- **Mitigation**: Comprehensive migration tools and backward compatibility layer

### 2. Performance Impact
- **Risk**: Service decorators add overhead
- **Mitigation**: Benchmark tests and optimization

### 3. Complexity Increase
- **Risk**: Architecture becomes too complex
- **Mitigation**: Clear documentation and examples

### 4. Plugin Dependencies
- **Risk**: Circular dependencies between plugin and core
- **Mitigation**: Strict interface-based design

## Testing Strategy

### 1. Unit Tests
- Test each decorator independently
- Test plugin initialization and registration
- Test configuration overrides

### 2. Integration Tests
- Test full request/response cycle with plugin active
- Test migration from current to new architecture
- Test both single-tenant and multi-tenant modes

### 3. Performance Tests
- Benchmark service decorator overhead
- Test with large numbers of organizations
- Memory usage analysis

## Rollback Plan

### 1. Feature Flags
- Use feature flags to enable/disable plugin-based multi-tenancy
- Allow gradual rollout

### 2. Database Migrations
- All database migrations must be reversible
- Keep old schemas during transition period

### 3. Configuration Rollback
- Maintain old configuration format support
- Provide conversion tools in both directions

## Success Criteria

1. **Single-Tenant Mode**: AuthSome works perfectly without any plugins
2. **Multi-Tenant Mode**: Full feature parity with current implementation
3. **Performance**: No significant performance degradation
4. **Migration**: Existing installations migrate without data loss
5. **Documentation**: Complete documentation for new architecture

## Next Steps

1. **Review and Approve Strategy**: Get team approval for this migration plan
2. **Create Feature Branch**: `feature/multitenancy-plugin-migration`
3. **Begin Phase 1**: Start with core service refactoring
4. **Continuous Testing**: Test each phase thoroughly before proceeding
5. **Documentation**: Update documentation as we progress

---

**Ready to begin implementation? Let's start with Phase 1!**