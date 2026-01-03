package admin

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/forge"
)

const (
	PluginID      = "admin"
	PluginName    = "Admin"
	PluginVersion = "1.0.0"
)

// Plugin implements the AuthSome plugin interface for admin operations
type Plugin struct {
	service *Service
	handler *Handler
}

// NewPlugin creates a new admin plugin instance
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the unique plugin identifier
func (p *Plugin) ID() string {
	return PluginID
}

// Name returns the human-readable plugin name
func (p *Plugin) Name() string {
	return PluginName
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return PluginVersion
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Cross-cutting administrative operations for platform management"
}

// Init initializes the plugin
func (p *Plugin) Init(auth interface{}) error {
	// Type assert to get Auth instance
	authInstance, ok := auth.(interface {
		GetServiceRegistry() interface {
			UserService() interface{}
			SessionService() interface{}
			RBACService() *rbac.Service
			AuditService() interface{}
			BanService() interface{}
		}
		GetConfig() interface{}
	})
	if !ok {
		return fmt.Errorf("invalid auth instance type")
	}

	serviceRegistry := authInstance.GetServiceRegistry()

	// Get required services
	rbacService := serviceRegistry.RBACService()
	if rbacService == nil {
		return fmt.Errorf("rbac service not found")
	}

	// Get user service
	userServiceRaw := serviceRegistry.UserService()
	if userServiceRaw == nil {
		return fmt.Errorf("user service not found")
	}

	// Get session service
	sessionServiceRaw := serviceRegistry.SessionService()
	if sessionServiceRaw == nil {
		return fmt.Errorf("session service not found")
	}

	// Get audit service
	auditServiceRaw := serviceRegistry.AuditService()
	if auditServiceRaw == nil {
		return fmt.Errorf("audit service not found")
	}

	// Get ban service
	banServiceRaw := serviceRegistry.BanService()
	if banServiceRaw == nil {
		return fmt.Errorf("ban service not found")
	}

	// Initialize admin service with all dependencies (services are returned as interfaces)
	p.service = NewService(
		DefaultConfig(),
		userServiceRaw,
		sessionServiceRaw,
		rbacService,
		auditServiceRaw,
		banServiceRaw,
	)

	// Initialize handler
	p.handler = NewHandler(p.service)

	return nil
}

// RegisterRoles implements the PluginWithRoles optional interface
// This is called automatically during server initialization to register admin permissions
func (p *Plugin) RegisterRoles(registry interface{}) error {
	roleRegistry, ok := registry.(rbac.RoleRegistryInterface)
	if !ok {
		return fmt.Errorf("invalid role registry type")
	}


	// Register Admin Role - Platform-level administrative operations
	if err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:         rbac.RoleAdmin,
		DisplayName:  "Administrator",
		Description:  rbac.RoleDescAdmin,
		Priority:     rbac.RolePriorityAdmin,
		IsPlatform:   rbac.RoleIsPlatformAdmin,
		InheritsFrom: rbac.RoleMember, // Inherits basic member permissions
		Permissions: []string{
			// User management - cross-cutting operations
			PermUserCreate + " on admin:*",
			PermUserRead + " on admin:*",
			PermUserUpdate + " on admin:*",
			PermUserDelete + " on admin:*",
			PermUserBan + " on admin:*",

			// Session management - cross-cutting operations
			PermSessionRead + " on admin:*",
			PermSessionRevoke + " on admin:*",

			// Role management - cross-cutting operations
			PermRoleAssign + " on admin:*",

			// Platform oversight
			PermStatsRead + " on admin:*",
			PermAuditRead + " on admin:*",
		},
	}); err != nil {
		return fmt.Errorf("failed to register admin role: %w", err)
	}

	// Extend Superadmin Role with impersonation (security-sensitive)
	// Superadmin should already exist from core RBAC bootstrap
	if err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:         rbac.RoleSuperAdmin,
		DisplayName:  "Super Administrator",
		Description:  rbac.RoleDescSuperAdmin,
		Priority:     rbac.RolePrioritySuperAdmin,
		IsPlatform:   rbac.RoleIsPlatformSuperAdmin,
		InheritsFrom: rbac.RoleAdmin, // Inherits all admin permissions
		Permissions: []string{
			// Impersonation is restricted to superadmin only
			PermUserImpersonate + " on admin:*",

			// Full wildcard access to admin operations
			"* on admin:*",
		},
	}); err != nil {
		// If superadmin already exists, that's ok - log it
	}

	return nil
}

// RegisterHooks implements the hooks registration (placeholder)
func (p *Plugin) RegisterHooks(registry interface{}) error {
	return nil
}

// RegisterServiceDecorators implements service decoration (not needed for admin)
func (p *Plugin) RegisterServiceDecorators(registry interface{}) error {
	return nil
}

// Migrate runs database migrations (placeholder)
func (p *Plugin) Migrate() error {
	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Admin API group
	admin := router.Group("/admin")

	// User management
	admin.POST("/users", p.handler.CreateUser)
	admin.GET("/users", p.handler.ListUsers)
	admin.DELETE("/users/:id", p.handler.DeleteUser)

	// Security operations
	admin.POST("/users/:id/ban", p.handler.BanUser)
	admin.POST("/users/:id/unban", p.handler.UnbanUser)
	admin.POST("/users/:id/impersonate", p.handler.ImpersonateUser)

	// Session management
	admin.GET("/sessions", p.handler.ListSessions)
	admin.DELETE("/sessions/:id", p.handler.RevokeSession)

	// Role management
	admin.POST("/users/:id/role", p.handler.SetUserRole)

	// Statistics & monitoring
	admin.GET("/stats", p.handler.GetStats)
	admin.GET("/audit-logs", p.handler.GetAuditLogs)

	return nil
}

// checkPermission is a helper to verify user has the required permission
func (p *Plugin) checkPermission(ctx context.Context, userID xid.ID, permission string) bool {
	if p.service == nil || p.service.rbacService == nil {
		return false
	}
	return p.service.checkAdminPermission(ctx, userID, permission) == nil
}
