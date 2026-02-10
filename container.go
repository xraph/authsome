package authsome

import (
	"fmt"
	"net/http"

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
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins"
	"github.com/xraph/forge"
	forgedb "github.com/xraph/forge/extensions/database"
)

// ContainerHelpers provides type-safe service resolution from the Forge DI container
// These helper functions are provided for plugins and handlers to easily access services

// ResolveDatabase resolves the database from the container
// First tries AuthSome's registered database, then falls back to Forge's database extension.
func ResolveDatabase(container forge.Container) (*bun.DB, error) {
	// Try AuthSome's registered database first (backwards compatibility)
	svc, err := container.Resolve(ServiceDatabase)
	if err == nil {
		db, ok := svc.(*bun.DB)
		if ok {
			return db, nil
		}
	}

	manager, err := forgedb.GetManager(container)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database manager: %w", err)
	}

	dbName := manager.DefaultName()

	return manager.SQL(dbName)
}

// ResolveDatabaseManager resolves Forge's DatabaseManager from the container
// This is useful for plugins that need access to multiple databases.
func ResolveDatabaseManager(container forge.Container) (*forgedb.DatabaseManager, error) {
	manager, err := forgedb.GetManager(container)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database manager: %w", err)
	}

	return manager, nil
}

// ResolveUserService resolves the user service from the container.
func ResolveUserService(container forge.Container) (user.ServiceInterface, error) {
	userSvc, err := forge.InjectType[user.ServiceInterface](container)
	if userSvc != nil {
		return userSvc, nil
	}

	svc, err := container.Resolve(ServiceUser)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user service: %w", err)
	}

	userSvc, ok := svc.(user.ServiceInterface)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "user service has invalid type", http.StatusBadRequest)
	}

	return userSvc, nil
}

// ResolveSessionService resolves the session service from the container.
func ResolveSessionService(container forge.Container) (session.ServiceInterface, error) {
	sessionSvc, err := forge.InjectType[session.ServiceInterface](container)
	if sessionSvc != nil {
		return sessionSvc, nil
	}

	svc, err := container.Resolve(ServiceSession)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve session service: %w", err)
	}

	sessionSvc, ok := svc.(session.ServiceInterface)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "session service has invalid type", http.StatusBadRequest)
	}

	return sessionSvc, nil
}

// ResolveAuthService resolves the auth service from the container.
func ResolveAuthService(container forge.Container) (auth.ServiceInterface, error) {
	authSvc, err := forge.InjectType[auth.ServiceInterface](container)
	if authSvc != nil {
		return authSvc, nil
	}

	svc, err := container.Resolve(ServiceAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth service: %w", err)
	}

	authSvc, ok := svc.(auth.ServiceInterface)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "auth service has invalid type", http.StatusBadRequest)
	}

	return authSvc, nil
}

// ResolveAppService resolves the app service from the container.
func ResolveAppService(container forge.Container) (*app.ServiceImpl, error) {
	appSvc, err := forge.InjectType[*app.ServiceImpl](container)
	if appSvc != nil {
		return appSvc, nil
	}

	svc, err := container.Resolve(ServiceApp)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve app service: %w", err)
	}

	appSvc, ok := svc.(*app.ServiceImpl)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "app service has invalid type", http.StatusBadRequest)
	}

	return appSvc, nil
}

// ResolveRateLimitService resolves the rate limit service from the container.
func ResolveRateLimitService(container forge.Container) (*ratelimit.Service, error) {
	rateLimitSvc, err := forge.InjectType[*ratelimit.Service](container)
	if rateLimitSvc != nil {
		return rateLimitSvc, nil
	}

	svc, err := container.Resolve(ServiceRateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve rate limit service: %w", err)
	}

	rateLimitSvc, ok := svc.(*ratelimit.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "rate limit service has invalid type", http.StatusBadRequest)
	}

	return rateLimitSvc, nil
}

// ResolveDeviceService resolves the device service from the container.
func ResolveDeviceService(container forge.Container) (*device.Service, error) {
	deviceSvc, err := forge.InjectType[*device.Service](container)
	if deviceSvc != nil {
		return deviceSvc, nil
	}

	svc, err := container.Resolve(ServiceDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve device service: %w", err)
	}

	deviceSvc, ok := svc.(*device.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "device service has invalid type", http.StatusBadRequest)
	}

	return deviceSvc, nil
}

// ResolveSecurityService resolves the security service from the container.
func ResolveSecurityService(container forge.Container) (*security.Service, error) {
	securitySvc, err := forge.InjectType[*security.Service](container)
	if securitySvc != nil {
		return securitySvc, nil
	}

	svc, err := container.Resolve(ServiceSecurity)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve security service: %w", err)
	}

	securitySvc, ok := svc.(*security.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "security service has invalid type", http.StatusBadRequest)
	}

	return securitySvc, nil
}

// ResolveAuditService resolves the audit service from the container.
func ResolveAuditService(container forge.Container) (*audit.Service, error) {
	auditSvc, err := forge.InjectType[*audit.Service](container)
	if auditSvc != nil {
		return auditSvc, nil
	}

	svc, err := container.Resolve(ServiceAudit)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve audit service: %w", err)
	}

	auditSvc, ok := svc.(*audit.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "audit service has invalid type", http.StatusBadRequest)
	}

	return auditSvc, nil
}

// ResolveRBACService resolves the RBAC service from the container.
func ResolveRBACService(container forge.Container) (*rbac.Service, error) {
	rbacSvc, err := forge.InjectType[*rbac.Service](container)
	if rbacSvc != nil {
		return rbacSvc, nil
	}

	svc, err := container.Resolve(ServiceRBAC)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve RBAC service: %w", err)
	}

	rbacSvc, ok := svc.(*rbac.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "RBAC service has invalid type", http.StatusBadRequest)
	}

	return rbacSvc, nil
}

// ResolveWebhookService resolves the webhook service from the container.
func ResolveWebhookService(container forge.Container) (*webhook.Service, error) {
	webhookSvc, err := forge.InjectType[*webhook.Service](container)
	if webhookSvc != nil {
		return webhookSvc, nil
	}

	svc, err := container.Resolve(ServiceWebhook)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve webhook service: %w", err)
	}

	webhookSvc, ok := svc.(*webhook.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "webhook service has invalid type", http.StatusBadRequest)
	}

	return webhookSvc, nil
}

// ResolveNotificationService resolves the notification service from the container.
func ResolveNotificationService(container forge.Container) (*notification.Service, error) {
	notificationSvc, err := forge.InjectType[*notification.Service](container)
	if notificationSvc != nil {
		return notificationSvc, nil
	}

	svc, err := container.Resolve(ServiceNotification)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve notification service: %w", err)
	}

	notificationSvc, ok := svc.(*notification.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "notification service has invalid type", http.StatusBadRequest)
	}

	return notificationSvc, nil
}

// ResolveJWTService resolves the JWT service from the container.
func ResolveJWTService(container forge.Container) (*jwt.Service, error) {
	jwtSvc, err := forge.InjectType[*jwt.Service](container)
	if jwtSvc != nil {
		return jwtSvc, nil
	}

	svc, err := container.Resolve(ServiceJWT)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve JWT service: %w", err)
	}

	jwtSvc, ok := svc.(*jwt.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "JWT service has invalid type", http.StatusBadRequest)
	}

	return jwtSvc, nil
}

// ResolveAPIKeyService resolves the API key service from the container.
func ResolveAPIKeyService(container forge.Container) (*apikey.Service, error) {
	apikeySvc, err := forge.InjectType[*apikey.Service](container)
	if apikeySvc != nil {
		return apikeySvc, nil
	}

	svc, err := container.Resolve(ServiceAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve API key service: %w", err)
	}

	apikeySvc, ok := svc.(*apikey.Service)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "API key service has invalid type", http.StatusBadRequest)
	}

	return apikeySvc, nil
}

// ResolveHookRegistry resolves the hook registry from the container.
func ResolveHookRegistry(container forge.Container) (*hooks.HookRegistry, error) {
	hookRegistry, err := forge.InjectType[*hooks.HookRegistry](container)
	if hookRegistry != nil {
		return hookRegistry, nil
	}

	svc, err := container.Resolve(ServiceHookRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve hook registry: %w", err)
	}

	hookRegistry, ok := svc.(*hooks.HookRegistry)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "hook registry has invalid type", http.StatusBadRequest)
	}

	return hookRegistry, nil
}

// ResolvePluginRegistry resolves the plugin registry from the container.
func ResolvePluginRegistry(container forge.Container) (*plugins.Registry, error) {
	pluginRegistry, err := forge.InjectType[*plugins.Registry](container)
	if pluginRegistry != nil {
		return pluginRegistry, nil
	}

	svc, err := container.Resolve(ServicePluginRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve plugin registry: %w", err)
	}

	pluginRegistry, ok := svc.(*plugins.Registry)
	if !ok {
		return nil, errs.New(errs.CodeInvalidInput, "plugin registry has invalid type", http.StatusBadRequest)
	}

	return pluginRegistry, nil
}

// =============================================================================
// Type-Based Injection Helpers (Preferred)
// =============================================================================
//
// These functions use Forge's type-based dependency injection via vessel.
// They are cleaner and type-safe compared to string-based resolution.
//
// Use these in new code; the ResolveXxx functions above are kept for backward compatibility.

// InjectDatabase resolves the database using type-based injection.
func InjectDatabase(container forge.Container) (*bun.DB, error) {
	return forge.InjectType[*bun.DB](container)
}

// InjectUserService resolves the user service using type-based injection.
func InjectUserService(container forge.Container) (user.ServiceInterface, error) {
	return forge.InjectType[user.ServiceInterface](container)
}

// InjectSessionService resolves the session service using type-based injection.
func InjectSessionService(container forge.Container) (session.ServiceInterface, error) {
	return forge.InjectType[session.ServiceInterface](container)
}

// InjectAuthService resolves the auth service using type-based injection.
func InjectAuthService(container forge.Container) (auth.ServiceInterface, error) {
	return forge.InjectType[auth.ServiceInterface](container)
}

// InjectAppService resolves the app service using type-based injection.
func InjectAppService(container forge.Container) (*app.ServiceImpl, error) {
	return forge.InjectType[*app.ServiceImpl](container)
}

// InjectRateLimitService resolves the rate limit service using type-based injection.
func InjectRateLimitService(container forge.Container) (*ratelimit.Service, error) {
	return forge.InjectType[*ratelimit.Service](container)
}

// InjectDeviceService resolves the device service using type-based injection.
func InjectDeviceService(container forge.Container) (*device.Service, error) {
	return forge.InjectType[*device.Service](container)
}

// InjectSecurityService resolves the security service using type-based injection.
func InjectSecurityService(container forge.Container) (*security.Service, error) {
	return forge.InjectType[*security.Service](container)
}

// InjectAuditService resolves the audit service using type-based injection.
func InjectAuditService(container forge.Container) (*audit.Service, error) {
	return forge.InjectType[*audit.Service](container)
}

// InjectRBACService resolves the RBAC service using type-based injection.
func InjectRBACService(container forge.Container) (*rbac.Service, error) {
	return forge.InjectType[*rbac.Service](container)
}

// InjectWebhookService resolves the webhook service using type-based injection.
func InjectWebhookService(container forge.Container) (*webhook.Service, error) {
	return forge.InjectType[*webhook.Service](container)
}

// InjectNotificationService resolves the notification service using type-based injection.
func InjectNotificationService(container forge.Container) (*notification.Service, error) {
	return forge.InjectType[*notification.Service](container)
}

// InjectJWTService resolves the JWT service using type-based injection.
func InjectJWTService(container forge.Container) (*jwt.Service, error) {
	return forge.InjectType[*jwt.Service](container)
}

// InjectAPIKeyService resolves the API key service using type-based injection.
func InjectAPIKeyService(container forge.Container) (*apikey.Service, error) {
	return forge.InjectType[*apikey.Service](container)
}

// InjectHookRegistry resolves the hook registry using type-based injection.
func InjectHookRegistry(container forge.Container) (*hooks.HookRegistry, error) {
	return forge.InjectType[*hooks.HookRegistry](container)
}

// InjectPluginRegistry resolves the plugin registry using type-based injection.
func InjectPluginRegistry(container forge.Container) (plugins.PluginRegistry, error) {
	return forge.InjectType[plugins.PluginRegistry](container)
}

// PluginDependencies is a convenience struct for plugins to get all common dependencies.
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

// ResolvePluginDependencies resolves all common plugin dependencies from the container.
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
