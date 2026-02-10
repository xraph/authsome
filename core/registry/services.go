package registry

import (
	"fmt"
	"sync"

	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/forms"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/internal/errs"
)

// ServiceRegistry manages all core services and allows plugins to replace them.
type ServiceRegistry struct {
	// Core services (using interfaces to allow decoration)
	appService          *app.ServiceImpl
	userService         user.ServiceInterface
	sessionService      session.ServiceInterface
	authService         auth.ServiceInterface
	jwtService          *jwt.Service
	apikeyService       *apikey.Service
	formsService        *forms.Service
	auditService        *audit.Service
	webhookService      *webhook.Service
	notificationService *notification.Service
	deviceService       *device.Service
	rbacService         *rbac.Service
	ratelimitService    *ratelimit.Service

	// Hook registry
	hookRegistry *hooks.HookRegistry

	// Role registry for role bootstrap system
	roleRegistry *rbac.RoleRegistry

	// Plugin-provided services (for multi-tenancy)
	organizationService *organization.Service
	configService       *app.ServiceImpl               // Will be set by multi-tenancy plugin
	environmentService  environment.EnvironmentService // will be set by multi-tenancy plugin

	// External services registered by key (for plugins and external integrations)
	externalServices map[string]any
	externalMu       sync.RWMutex
}

// NewServiceRegistry creates a new service registry.
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		hookRegistry:     hooks.NewHookRegistry(),
		roleRegistry:     rbac.NewRoleRegistry(),
		externalServices: make(map[string]any),
	}
}

// SetUserService sets the user service.
func (r *ServiceRegistry) SetUserService(svc user.ServiceInterface) {
	r.userService = svc
}

func (r *ServiceRegistry) SetSessionService(svc session.ServiceInterface) {
	r.sessionService = svc
}

func (r *ServiceRegistry) SetAuthService(svc auth.ServiceInterface) {
	r.authService = svc
}

func (r *ServiceRegistry) SetJWTService(svc *jwt.Service) {
	r.jwtService = svc
}

func (r *ServiceRegistry) SetAPIKeyService(svc *apikey.Service) {
	r.apikeyService = svc
}

func (r *ServiceRegistry) SetFormsService(svc *forms.Service) {
	r.formsService = svc
}

func (r *ServiceRegistry) SetAuditService(svc *audit.Service) {
	r.auditService = svc
}

func (r *ServiceRegistry) SetWebhookService(svc *webhook.Service) {
	r.webhookService = svc
}

func (r *ServiceRegistry) SetNotificationService(svc *notification.Service) {
	r.notificationService = svc
}

func (r *ServiceRegistry) SetDeviceService(svc *device.Service) {
	r.deviceService = svc
}

func (r *ServiceRegistry) SetRBACService(svc *rbac.Service) {
	r.rbacService = svc
}

func (r *ServiceRegistry) SetRateLimitService(svc *ratelimit.Service) {
	r.ratelimitService = svc
}

// UserService returns the user service.
func (r *ServiceRegistry) UserService() user.ServiceInterface {
	return r.userService
}

func (r *ServiceRegistry) SessionService() session.ServiceInterface {
	return r.sessionService
}

func (r *ServiceRegistry) AuthService() auth.ServiceInterface {
	return r.authService
}

func (r *ServiceRegistry) JWTService() *jwt.Service {
	return r.jwtService
}

func (r *ServiceRegistry) APIKeyService() *apikey.Service {
	return r.apikeyService
}

func (r *ServiceRegistry) FormsService() *forms.Service {
	return r.formsService
}

func (r *ServiceRegistry) AuditService() *audit.Service {
	return r.auditService
}

func (r *ServiceRegistry) WebhookService() *webhook.Service {
	return r.webhookService
}

func (r *ServiceRegistry) NotificationService() *notification.Service {
	return r.notificationService
}

func (r *ServiceRegistry) DeviceService() *device.Service {
	return r.deviceService
}

func (r *ServiceRegistry) RBACService() *rbac.Service {
	return r.rbacService
}

func (r *ServiceRegistry) RateLimitService() *ratelimit.Service {
	return r.ratelimitService
}

// HookRegistry returns the hook registry.
func (r *ServiceRegistry) HookRegistry() *hooks.HookRegistry {
	return r.hookRegistry
}

// RoleRegistry returns the role registry.
func (r *ServiceRegistry) RoleRegistry() *rbac.RoleRegistry {
	return r.roleRegistry
}

// ReplaceUserService replaces the user service (used by plugins to decorate services).
func (r *ServiceRegistry) ReplaceUserService(svc user.ServiceInterface) {
	r.userService = svc
}

func (r *ServiceRegistry) ReplaceSessionService(svc session.ServiceInterface) {
	r.sessionService = svc
}

func (r *ServiceRegistry) ReplaceAuthService(svc auth.ServiceInterface) {
	r.authService = svc
}

func (r *ServiceRegistry) ReplaceJWTService(svc *jwt.Service) {
	r.jwtService = svc
}

func (r *ServiceRegistry) ReplaceAPIKeyService(svc *apikey.Service) {
	r.apikeyService = svc
}

func (r *ServiceRegistry) ReplaceFormsService(svc *forms.Service) {
	r.formsService = svc
}

// SetOrganizationService sets the organization service.
func (r *ServiceRegistry) SetOrganizationService(svc *organization.Service) {
	r.organizationService = svc
}

func (r *ServiceRegistry) SetAppService(svc *app.ServiceImpl) {
	r.appService = svc
}

func (r *ServiceRegistry) SetConfigService(svc *app.ServiceImpl) {
	r.configService = svc
}

func (r *ServiceRegistry) SetEnvironmentService(svc environment.EnvironmentService) {
	r.environmentService = svc
}

// OrganizationService returns the organization service.
func (r *ServiceRegistry) OrganizationService() *organization.Service {
	return r.organizationService
}

func (r *ServiceRegistry) AppService() *app.ServiceImpl {
	return r.appService
}

func (r *ServiceRegistry) ConfigService() *app.ServiceImpl {
	return r.configService
}

func (r *ServiceRegistry) EnvironmentService() environment.EnvironmentService {
	return r.environmentService
}

// HasOrganizationService checks if the organization service is available.
func (r *ServiceRegistry) HasOrganizationService() bool {
	return r.organizationService != nil
}

func (r *ServiceRegistry) HasAppService() bool {
	return r.organizationService != nil
}

func (r *ServiceRegistry) HasConfigService() bool {
	return r.configService != nil
}

func (r *ServiceRegistry) HasEnvironmentService() bool {
	return r.environmentService != nil
}

// IsMultiTenant returns true if the multi-tenancy plugin is active.
func (r *ServiceRegistry) IsMultiTenant() bool {
	return r.HasAppService()
}

// Register registers an external service by key
// This allows plugins and external integrations to register custom services
// that can be retrieved later by other plugins or components.
func (r *ServiceRegistry) Register(key string, service any) error {
	if key == "" {
		return errs.RequiredField("key")
	}

	if service == nil {
		return errs.RequiredField("service")
	}

	r.externalMu.Lock()
	defer r.externalMu.Unlock()

	// Check if key already exists
	if _, exists := r.externalServices[key]; exists {
		return errs.Conflict(fmt.Sprintf("service with key '%s' already registered", key))
	}

	r.externalServices[key] = service

	return nil
}

// Get retrieves an external service by key
// Returns the service and a boolean indicating if it was found
// The caller must perform type assertion to use the service.
func (r *ServiceRegistry) Get(key string) (any, bool) {
	if key == "" {
		return nil, false
	}

	r.externalMu.RLock()
	defer r.externalMu.RUnlock()

	service, exists := r.externalServices[key]

	return service, exists
}

// GetAs retrieves an external service by key and performs type assertion
// Returns the service as the requested type and an error if not found or type assertion fails
// This is a generic function that provides type-safe service retrieval
// Usage: service, err := registry.GetAs[*MyService](registry, "my-service-key").
func GetAs[T any](r *ServiceRegistry, key string) (T, error) {
	var zero T

	service, exists := r.Get(key)
	if !exists {
		return zero, errs.NotFound(fmt.Sprintf("service with key '%s' not found", key))
	}

	typed, ok := service.(T)
	if !ok {
		return zero, errs.InternalServerErrorWithMessage(fmt.Sprintf("service with key '%s' is not of type %T", key, zero))
	}

	return typed, nil
}

// Has checks if a service is registered for the given key.
func (r *ServiceRegistry) Has(key string) bool {
	if key == "" {
		return false
	}

	r.externalMu.RLock()
	defer r.externalMu.RUnlock()

	_, exists := r.externalServices[key]

	return exists
}

// Unregister removes a service from the registry by key.
func (r *ServiceRegistry) Unregister(key string) error {
	if key == "" {
		return errs.RequiredField("key")
	}

	r.externalMu.Lock()
	defer r.externalMu.Unlock()

	if _, exists := r.externalServices[key]; !exists {
		return errs.NotFound(fmt.Sprintf("service with key '%s' not found", key))
	}

	delete(r.externalServices, key)

	return nil
}

// ListKeys returns all registered service keys.
func (r *ServiceRegistry) ListKeys() []string {
	r.externalMu.RLock()
	defer r.externalMu.RUnlock()

	keys := make([]string, 0, len(r.externalServices))
	for key := range r.externalServices {
		keys = append(keys, key)
	}

	return keys
}

// RBACRegistry returns the RBAC role registry.
func (r *ServiceRegistry) RBACRegistry() *rbac.RoleRegistry {
	return r.roleRegistry
}
