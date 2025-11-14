package authsome

import (
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/plugins"
	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/database"
	forgedb "github.com/xraph/forge/extensions/database"
)

// ContainerHelpers provides type-safe service resolution from the Forge DI container
// These helper functions are provided for plugins and handlers to easily access services

// ResolveDatabase resolves the database from the container
// First tries AuthSome's registered database, then falls back to Forge's database extension
func ResolveDatabase(container forge.Container) (*bun.DB, error) {
	// Try AuthSome's registered database first (backwards compatibility)
	svc, err := container.Resolve(ServiceDatabase)
	if err == nil {
		db, ok := svc.(*bun.DB)
		if ok {
			return db, nil
		}
	}

	// Fall back to Forge's database extension
	db, err := database.GetSQL(container)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database from container: %w", err)
	}

	return db, nil
}

// ResolveDatabaseManager resolves Forge's DatabaseManager from the container
// This is useful for plugins that need access to multiple databases
func ResolveDatabaseManager(container forge.Container) (*forgedb.DatabaseManager, error) {
	manager, err := database.GetManager(container)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database manager: %w", err)
	}

	return manager, nil
}

// ResolveUserService resolves the user service from the container
func ResolveUserService(container forge.Container) (user.ServiceInterface, error) {
	svc, err := container.Resolve(ServiceUser)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user service: %w", err)
	}
	userSvc, ok := svc.(user.ServiceInterface)
	if !ok {
		return nil, fmt.Errorf("user service has invalid type")
	}
	return userSvc, nil
}

// ResolveSessionService resolves the session service from the container
func ResolveSessionService(container forge.Container) (session.ServiceInterface, error) {
	svc, err := container.Resolve(ServiceSession)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve session service: %w", err)
	}
	sessionSvc, ok := svc.(session.ServiceInterface)
	if !ok {
		return nil, fmt.Errorf("session service has invalid type")
	}
	return sessionSvc, nil
}

// ResolveAuthService resolves the auth service from the container
func ResolveAuthService(container forge.Container) (auth.ServiceInterface, error) {
	svc, err := container.Resolve(ServiceAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth service: %w", err)
	}
	authSvc, ok := svc.(auth.ServiceInterface)
	if !ok {
		return nil, fmt.Errorf("auth service has invalid type")
	}
	return authSvc, nil
}

// ResolveAppService resolves the app service from the container
func ResolveAppService(container forge.Container) (*app.ServiceImpl, error) {
	svc, err := container.Resolve(ServiceApp)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve app service: %w", err)
	}
	appSvc, ok := svc.(*app.ServiceImpl)
	if !ok {
		return nil, fmt.Errorf("app service has invalid type")
	}
	return appSvc, nil
}

// ResolveRateLimitService resolves the rate limit service from the container
func ResolveRateLimitService(container forge.Container) (*ratelimit.Service, error) {
	svc, err := container.Resolve(ServiceRateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve rate limit service: %w", err)
	}
	rateLimitSvc, ok := svc.(*ratelimit.Service)
	if !ok {
		return nil, fmt.Errorf("rate limit service has invalid type")
	}
	return rateLimitSvc, nil
}

// ResolveDeviceService resolves the device service from the container
func ResolveDeviceService(container forge.Container) (*device.Service, error) {
	svc, err := container.Resolve(ServiceDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve device service: %w", err)
	}
	deviceSvc, ok := svc.(*device.Service)
	if !ok {
		return nil, fmt.Errorf("device service has invalid type")
	}
	return deviceSvc, nil
}

// ResolveSecurityService resolves the security service from the container
func ResolveSecurityService(container forge.Container) (*security.Service, error) {
	svc, err := container.Resolve(ServiceSecurity)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve security service: %w", err)
	}
	securitySvc, ok := svc.(*security.Service)
	if !ok {
		return nil, fmt.Errorf("security service has invalid type")
	}
	return securitySvc, nil
}

// ResolveAuditService resolves the audit service from the container
func ResolveAuditService(container forge.Container) (*audit.Service, error) {
	svc, err := container.Resolve(ServiceAudit)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve audit service: %w", err)
	}
	auditSvc, ok := svc.(*audit.Service)
	if !ok {
		return nil, fmt.Errorf("audit service has invalid type")
	}
	return auditSvc, nil
}

// ResolveRBACService resolves the RBAC service from the container
func ResolveRBACService(container forge.Container) (*rbac.Service, error) {
	svc, err := container.Resolve(ServiceRBAC)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve RBAC service: %w", err)
	}
	rbacSvc, ok := svc.(*rbac.Service)
	if !ok {
		return nil, fmt.Errorf("RBAC service has invalid type")
	}
	return rbacSvc, nil
}

// ResolveWebhookService resolves the webhook service from the container
func ResolveWebhookService(container forge.Container) (*webhook.Service, error) {
	svc, err := container.Resolve(ServiceWebhook)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve webhook service: %w", err)
	}
	webhookSvc, ok := svc.(*webhook.Service)
	if !ok {
		return nil, fmt.Errorf("webhook service has invalid type")
	}
	return webhookSvc, nil
}

// ResolveNotificationService resolves the notification service from the container
func ResolveNotificationService(container forge.Container) (*notification.Service, error) {
	svc, err := container.Resolve(ServiceNotification)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve notification service: %w", err)
	}
	notificationSvc, ok := svc.(*notification.Service)
	if !ok {
		return nil, fmt.Errorf("notification service has invalid type")
	}
	return notificationSvc, nil
}

// ResolveJWTService resolves the JWT service from the container
func ResolveJWTService(container forge.Container) (*jwt.Service, error) {
	svc, err := container.Resolve(ServiceJWT)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve JWT service: %w", err)
	}
	jwtSvc, ok := svc.(*jwt.Service)
	if !ok {
		return nil, fmt.Errorf("JWT service has invalid type")
	}
	return jwtSvc, nil
}

// ResolveAPIKeyService resolves the API key service from the container
func ResolveAPIKeyService(container forge.Container) (*apikey.Service, error) {
	svc, err := container.Resolve(ServiceAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve API key service: %w", err)
	}
	apikeySvc, ok := svc.(*apikey.Service)
	if !ok {
		return nil, fmt.Errorf("API key service has invalid type")
	}
	return apikeySvc, nil
}

// ResolveHookRegistry resolves the hook registry from the container
func ResolveHookRegistry(container forge.Container) (*hooks.HookRegistry, error) {
	svc, err := container.Resolve(ServiceHookRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve hook registry: %w", err)
	}
	hookRegistry, ok := svc.(*hooks.HookRegistry)
	if !ok {
		return nil, fmt.Errorf("hook registry has invalid type")
	}
	return hookRegistry, nil
}

// ResolvePluginRegistry resolves the plugin registry from the container
func ResolvePluginRegistry(container forge.Container) (*plugins.Registry, error) {
	svc, err := container.Resolve(ServicePluginRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve plugin registry: %w", err)
	}
	pluginRegistry, ok := svc.(*plugins.Registry)
	if !ok {
		return nil, fmt.Errorf("plugin registry has invalid type")
	}
	return pluginRegistry, nil
}

// PluginDependencies is a convenience struct for plugins to get all common dependencies
type PluginDependencies struct {
	Container      forge.Container
	Database       *bun.DB
	UserService    user.ServiceInterface
	SessionService session.ServiceInterface
	AuthService    auth.ServiceInterface
	AuditService   *audit.Service
	RBACService    *rbac.Service
	HookRegistry   *hooks.HookRegistry
}

// ResolvePluginDependencies resolves all common plugin dependencies from the container
func ResolvePluginDependencies(container forge.Container) (*PluginDependencies, error) {
	db, err := ResolveDatabase(container)
	if err != nil {
		return nil, err
	}

	userSvc, err := ResolveUserService(container)
	if err != nil {
		return nil, err
	}

	sessionSvc, err := ResolveSessionService(container)
	if err != nil {
		return nil, err
	}

	authSvc, err := ResolveAuthService(container)
	if err != nil {
		return nil, err
	}

	auditSvc, err := ResolveAuditService(container)
	if err != nil {
		return nil, err
	}

	rbacSvc, err := ResolveRBACService(container)
	if err != nil {
		return nil, err
	}

	hookRegistry, err := ResolveHookRegistry(container)
	if err != nil {
		return nil, err
	}

	return &PluginDependencies{
		Container:      container,
		Database:       db,
		UserService:    userSvc,
		SessionService: sessionSvc,
		AuthService:    authSvc,
		AuditService:   auditSvc,
		RBACService:    rbacSvc,
		HookRegistry:   hookRegistry,
	}, nil
}
