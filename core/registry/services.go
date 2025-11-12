package registry

import (
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/forms"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
)

// ServiceRegistry manages all core services and allows plugins to replace them
type ServiceRegistry struct {
	// Core services (using interfaces to allow decoration)
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
	organizationService interface{} // Will be set by multi-tenancy plugin
	configService       interface{} // Will be set by multi-tenancy plugin
	environmentService  interface{} // Will be set by multi-tenancy plugin
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		hookRegistry: hooks.NewHookRegistry(),
		roleRegistry: rbac.NewRoleRegistry(),
	}
}

// Core service setters (used during initialization)
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

// Core service getters
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

// Hook registry getter
func (r *ServiceRegistry) HookRegistry() *hooks.HookRegistry {
	return r.hookRegistry
}

// Role registry getter
func (r *ServiceRegistry) RoleRegistry() *rbac.RoleRegistry {
	return r.roleRegistry
}

// Service replacement methods (used by plugins to decorate services)
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

// Plugin service setters (for multi-tenancy plugin)
func (r *ServiceRegistry) SetOrganizationService(svc interface{}) {
	r.organizationService = svc
}

func (r *ServiceRegistry) SetAppService(svc interface{}) {
	r.organizationService = svc // organizationService field stores app service (renamed terminology)
}

func (r *ServiceRegistry) SetConfigService(svc interface{}) {
	r.configService = svc
}

func (r *ServiceRegistry) SetEnvironmentService(svc interface{}) {
	r.environmentService = svc
}

// Plugin service getters (for multi-tenancy plugin)
func (r *ServiceRegistry) OrganizationService() interface{} {
	return r.organizationService
}

func (r *ServiceRegistry) AppService() interface{} {
	return r.organizationService // organizationService field stores app service (renamed terminology)
}

func (r *ServiceRegistry) ConfigService() interface{} {
	return r.configService
}

func (r *ServiceRegistry) EnvironmentService() interface{} {
	return r.environmentService
}

// Utility methods for plugins
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

// IsMultiTenant returns true if the multi-tenancy plugin is active
func (r *ServiceRegistry) IsMultiTenant() bool {
	return r.HasAppService()
}
