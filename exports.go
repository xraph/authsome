package authsome

import (
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	aud "github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	dev "github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/middleware"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/organization"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/responses"
	sec "github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/plugins"
	"github.com/xraph/authsome/schema"
)

// ============================================================================
// Type Exports from Core Packages
// ============================================================================

// App Package Exports.
type (
	// AppService is the service interface for app operations.
	AppService = app.AppService

	// AppConfig holds app service configuration.
	AppConfig = app.Config

	// App represents an application entity.
	App = app.App

	// Member represents an app member.
	Member = app.Member

	// Team represents a team within an app.
	Team = app.Team

	// TeamMember represents a team member.
	TeamMember = app.TeamMember

	// Invitation represents an app invitation.
	Invitation = app.Invitation

	// Environment represents an environment.
	Environment = environment.Environment

	// CreateAppRequest is the request for creating an app.
	CreateAppRequest = app.CreateAppRequest

	// UpdateAppRequest is the request for updating an app.
	UpdateAppRequest = app.UpdateAppRequest

	// CreateTeamRequest is the request for creating a team.
	CreateTeamRequest = app.CreateTeamRequest

	// UpdateTeamRequest is the request for updating a team.
	UpdateTeamRequest = app.UpdateTeamRequest

	// UpdateMemberRequest is the request for updating a member.
	UpdateMemberRequest = app.UpdateMemberRequest

	// InviteMemberRequest is the request for inviting a member.
	InviteMemberRequest = app.InviteMemberRequest

	// AppRepository defines the app repository interface.
	AppRepository = app.AppRepository

	// MemberRepository defines the member repository interface.
	MemberRepository = app.MemberRepository

	// TeamRepository defines the team repository interface.
	TeamRepository = app.TeamRepository

	// InvitationRepository defines the invitation repository interface.
	InvitationRepository = app.InvitationRepository

	// EnvironmentRepository defines the environment repository interface.
	EnvironmentRepository = environment.Repository
)

// Auth Package Exports.
type (
	// AuthService is the authentication service interface.
	AuthService = auth.ServiceInterface

	// AuthConfig holds auth service configuration.
	AuthConfig = auth.Config

	// SignInRequest is the request for signing in.
	SignInRequest = auth.SignInRequest

	// SignUpRequest is the request for signing up.
	SignUpRequest = auth.SignUpRequest

	// AuthResponse is the response from authentication operations.
	AuthResponse = responses.AuthResponse
)

// User Package Exports.
type (
	// UserService is the user service interface.
	UserService = user.ServiceInterface

	// UserConfig holds user service configuration.
	UserConfig = user.Config

	// User represents a user entity.
	User = user.User

	// CreateUserRequest is the request for creating a user.
	CreateUserRequest = user.CreateUserRequest

	// UpdateUserRequest is the request for updating a user.
	UpdateUserRequest = user.UpdateUserRequest
)

// Session Package Exports.
type (
	// SessionService is the session service interface.
	SessionService = session.ServiceInterface

	// SessionConfig holds session service configuration.
	SessionConfig = session.Config

	// Session represents a session entity.
	Session = session.Session

	// CreateSessionRequest is the request for creating a session.
	CreateSessionRequest = session.CreateSessionRequest
)

// JWT Package Exports.
type (
	// JWTService is the JWT service.
	JWTService = jwt.Service

	// JWTConfig holds JWT service configuration.
	JWTConfig = jwt.Config

	// JWTKey represents a JWT key entity.
	JWTKey = jwt.JWTKey

	// CreateJWTKeyRequest is the request for creating a JWT key.
	CreateJWTKeyRequest = jwt.CreateJWTKeyRequest

	// GenerateTokenRequest is the request for generating a JWT token.
	GenerateTokenRequest = jwt.GenerateTokenRequest
)

// APIKey Package Exports.
type (
	// APIKeyService is the API key service.
	APIKeyService = apikey.Service

	// APIKeyConfig holds API key service configuration.
	APIKeyConfig = apikey.Config

	// APIKey represents an API key entity.
	APIKey = apikey.APIKey

	// CreateAPIKeyRequest is the request for creating an API key.
	CreateAPIKeyRequest = apikey.CreateAPIKeyRequest
)

// Webhook Package Exports.
type (
	// WebhookService is the webhook service.
	WebhookService = webhook.Service

	// WebhookConfig holds webhook service configuration.
	WebhookConfig = webhook.Config

	// Webhook represents a webhook entity.
	Webhook = webhook.Webhook

	// CreateWebhookRequest is the request for creating a webhook.
	CreateWebhookRequest = webhook.CreateWebhookRequest

	// WebhookEvent represents a webhook event.
	WebhookEvent = webhook.Event

	// WebhookDelivery represents a webhook delivery.
	WebhookDelivery = webhook.Delivery
)

// Notification Package Exports.
type (
	// NotificationService is the notification service.
	NotificationService = notification.Service

	// NotificationConfig holds notification service configuration.
	NotificationConfig = notification.Config

	// Notification represents a notification entity.
	Notification = notification.Notification

	// NotificationTemplate represents a notification template.
	NotificationTemplate = notification.Template
)

// Audit Package Exports.
type (
	// AuditService is the audit service.
	AuditService = aud.Service
)

// RBAC Package Exports.
type (
	// RBACService is the RBAC service.
	RBACService = rbac.Service

	// Role represents a role entity.
	Role = rbac.Role

	// Permission represents a permission.
	Permission = rbac.Permission

	// Policy represents a policy.
	Policy = rbac.Policy

	// RoleRegistry is the role registry for registering roles.
	RoleRegistry = rbac.RoleRegistry
)

// Device Package Exports.
type (
	// DeviceService is the device service.
	DeviceService = dev.Service

	// Device represents a device entity.
	Device = dev.Device
)

// Security Package Exports.
type (
	// SecurityService is the security service.
	SecurityService = sec.Service

	// SecurityConfig holds security service configuration.
	SecurityConfig = sec.Config

	// GeoIPProvider is the interface for GeoIP providers.
	GeoIPProvider = sec.GeoIPProvider
)

// Rate Limit Package Exports.
type (
	// RateLimitService is the rate limit service.
	RateLimitService = rl.Service

	// RateLimitConfig holds rate limit service configuration.
	RateLimitConfig = rl.Config

	// RateLimitStorage is the interface for rate limit storage.
	RateLimitStorage = rl.Storage
)

// Organization Package Exports.
type (
	// OrganizationService is the organization service interface.
	OrganizationService = organization.OrganizationService

	// OrganizationConfig holds organization service configuration.
	OrganizationConfig = organization.Config

	// Organization represents an organization entity.
	Organization = organization.Organization
)

// Contexts Package Exports.
type (
	// AuthContext holds complete authentication state for a request.
	AuthContext = contexts.AuthContext

	// AuthMethod indicates how the request was authenticated.
	AuthMethod = contexts.AuthMethod
)

// AuthMethod constants.
const (
	// AuthMethodNone indicates no authentication.
	AuthMethodNone = contexts.AuthMethodNone

	// AuthMethodSession indicates session-based authentication.
	AuthMethodSession = contexts.AuthMethodSession

	// AuthMethodAPIKey indicates API key authentication.
	AuthMethodAPIKey = contexts.AuthMethodAPIKey

	// AuthMethodBoth indicates both session and API key authentication.
	AuthMethodBoth = contexts.AuthMethodBoth
)

// Hook Package Exports.
type (
	// HookRegistry is the registry for registering hooks.
	HookRegistry = hooks.HookRegistry
)

// Schema Package Exports - Database Models.
type (
	// SchemaApp is the database model for apps.
	SchemaApp = schema.App

	// SchemaMember is the database model for members.
	SchemaMember = schema.Member

	// SchemaTeam is the database model for teams.
	SchemaTeam = schema.Team

	// SchemaTeamMember is the database model for team members.
	SchemaTeamMember = schema.TeamMember

	// SchemaInvitation is the database model for invitations.
	SchemaInvitation = schema.Invitation

	// SchemaUser is the database model for users.
	SchemaUser = schema.User

	// SchemaSession is the database model for sessions.
	SchemaSession = schema.Session

	// SchemaRole is the database model for roles.
	SchemaRole = schema.Role

	// SchemaUserRole is the database model for user roles.
	SchemaUserRole = schema.UserRole

	// SchemaWebhook is the database model for webhooks.
	SchemaWebhook = schema.Webhook

	// SchemaNotification is the database model for notifications.
	SchemaNotification = schema.Notification

	// SchemaAPIKey is the database model for API keys.
	SchemaAPIKey = schema.APIKey

	// SchemaJWTKey is the database model for JWT keys.
	SchemaJWTKey = schema.JWTKey

	// SchemaDevice is the database model for devices.
	SchemaDevice = schema.Device
)

// Plugin Package Exports.
type (
	// Plugin is the interface that all plugins must implement.
	Plugin = plugins.Plugin

	// PluginRegistry is the registry for managing plugins.
	PluginRegistry = plugins.PluginRegistry
)

// ServiceImpl Registry Export.
type (
	// ServiceRegistry manages all core services and allows plugins to replace them.
	ServiceRegistry = registry.ServiceRegistry
)

// ============================================================================
// Function Exports from Core Packages
// ============================================================================

// RBAC Functions.
var (
	// RegisterDefaultPlatformRoles registers default platform roles.
	RegisterDefaultPlatformRoles = rbac.RegisterDefaultPlatformRoles
)

// Contexts Functions.
var (
	// GetAppID gets the app ID from context.
	GetAppID     = contexts.GetAppID
	SetAppID     = contexts.SetAppID
	RequireAppID = contexts.RequireAppID

	// GetEnvironmentID gets the environment ID from context.
	GetEnvironmentID     = contexts.GetEnvironmentID
	SetEnvironmentID     = contexts.SetEnvironmentID
	RequireEnvironmentID = contexts.RequireEnvironmentID

	// GetOrganizationID gets the organization ID from context.
	GetOrganizationID     = contexts.GetOrganizationID
	SetOrganizationID     = contexts.SetOrganizationID
	RequireOrganizationID = contexts.RequireOrganizationID

	// GetUserID gets the user ID from context.
	GetUserID     = contexts.GetUserID
	SetUserID     = contexts.SetUserID
	RequireUserID = contexts.RequireUserID

	// WithAppAndOrganization creates a context with app and organization IDs.
	WithAppAndOrganization            = contexts.WithAppAndOrganization
	WithAppAndUser                    = contexts.WithAppAndUser
	WithAppEnvironmentAndOrganization = contexts.WithAppEnvironmentAndOrganization
	WithAll                           = contexts.WithAll

	// SetAuthContext sets the auth context.
	SetAuthContext     = contexts.SetAuthContext
	GetAuthContext     = contexts.GetAuthContext
	RequireAuthContext = contexts.RequireAuthContext
	RequireUser        = contexts.RequireUser
	RequireAPIKey      = contexts.RequireAPIKey
	GetUser            = contexts.GetUser
	GetAPIKey          = contexts.GetAPIKey
	GetSession         = contexts.GetSession
)

type (
	// AfterOrganizationCreateHook registers a user lifecycle hook.
	AfterOrganizationCreateHook = hooks.AfterOrganizationCreateHook

	// AfterSessionCreateHook registers a session lifecycle hook.
	AfterSessionCreateHook = hooks.AfterSessionCreateHook

	// AfterSignUpHook registers an authentication lifecycle hook.
	AfterSignUpHook = hooks.AfterSignUpHook

	// AfterSignInHook registers an authentication lifecycle hook.
	AfterSignInHook = hooks.AfterSignInHook

	// AfterSignOutHook registers an authentication lifecycle hook.
	AfterSignOutHook = hooks.AfterSignOutHook

	// AfterMemberAddHook registers an organization lifecycle hook.
	AfterMemberAddHook = hooks.AfterMemberAddHook
)

// Schema Enums - Type aliases for cleaner API (re-exported from core/app).
type (
	// MemberRole is a member role type.
	MemberRole   = app.MemberRole
	MemberStatus = app.MemberStatus

	// InvitationStatus is an invitation status type.
	InvitationStatus = app.InvitationStatus
)

// Enum constants exported for convenience.
const (
	// MemberRoleOwner is the owner member role.
	MemberRoleOwner  = app.MemberRoleOwner
	MemberRoleAdmin  = app.MemberRoleAdmin
	MemberRoleMember = app.MemberRoleMember

	// MemberStatusActive is the active member status.
	MemberStatusActive    = app.MemberStatusActive
	MemberStatusSuspended = app.MemberStatusSuspended
	MemberStatusPending   = app.MemberStatusPending

	// InvitationStatusPending is the pending invitation status.
	InvitationStatusPending   = app.InvitationStatusPending
	InvitationStatusAccepted  = app.InvitationStatusAccepted
	InvitationStatusExpired   = app.InvitationStatusExpired
	InvitationStatusCancelled = app.InvitationStatusCancelled
	InvitationStatusDeclined  = app.InvitationStatusDeclined

	// RoleOwner is a backward compatibility alias for MemberRoleOwner.
	RoleOwner       = app.MemberRoleOwner
	RoleAdmin       = app.MemberRoleAdmin
	RoleMember      = app.MemberRoleMember
	StatusActive    = app.MemberStatusActive
	StatusSuspended = app.MemberStatusSuspended
	StatusPending   = app.MemberStatusPending
)

// ============================================================================
// Error Exports from Core Packages
// ============================================================================

// Context Errors.
var (
	// ErrAppContextRequired is returned when app context is required but not found.
	ErrAppContextRequired = contexts.ErrAppContextRequired

	// ErrEnvironmentContextRequired is returned when environment context is required but not found.
	ErrEnvironmentContextRequired = contexts.ErrEnvironmentContextRequired

	// ErrOrganizationContextRequired is returned when organization context is required but not found.
	ErrOrganizationContextRequired = contexts.ErrOrganizationContextRequired

	// ErrUserContextRequired is returned when user context is required but not found.
	ErrUserContextRequired = contexts.ErrUserContextRequired

	// ErrAuthContextRequired is returned when auth context is required but not found.
	ErrAuthContextRequired = contexts.ErrAuthContextRequired

	// ErrUserAuthRequired is returned when user authentication is required.
	ErrUserAuthRequired = contexts.ErrUserAuthRequired

	// ErrAPIKeyRequired is returned when API key authentication is required.
	ErrAPIKeyRequired = contexts.ErrAPIKeyRequired

	// ErrInsufficientScope is returned when API key lacks required scope.
	ErrInsufficientScope = contexts.ErrInsufficientScope

	// ErrInsufficientPermission is returned when lacking required RBAC permission.
	ErrInsufficientPermission = contexts.ErrInsufficientPermission
)

// ============================================================================
// Middleware Package Exports
// ============================================================================

// Middleware Configuration.
type (
	// AuthMiddleware is the authentication middleware.
	AuthMiddleware = middleware.AuthMiddleware

	// AuthMiddlewareConfig configures the authentication middleware behavior.
	AuthMiddlewareConfig = middleware.AuthMiddlewareConfig

	// ContextConfig configures how app and environment context is populated.
	ContextConfig = middleware.ContextConfig

	// ContextResolution tracks how context values were resolved.
	ContextResolution = middleware.ContextResolution

	// ContextSource indicates where the context value came from.
	ContextSource = middleware.ContextSource
)

// Context Source Constants.
const (
	// ContextSourceNone indicates no context source.
	ContextSourceNone = middleware.ContextSourceNone

	// ContextSourceExisting indicates context already exists in request.
	ContextSourceExisting = middleware.ContextSourceExisting

	// ContextSourceHeader indicates context from HTTP header.
	ContextSourceHeader = middleware.ContextSourceHeader

	// ContextSourceAPIKey indicates context from verified API key.
	ContextSourceAPIKey = middleware.ContextSourceAPIKey

	// ContextSourceDefault indicates context from default config.
	ContextSourceDefault = middleware.ContextSourceDefault

	// ContextSourceAutoDetect indicates context from AuthSome config.
	ContextSourceAutoDetect = middleware.ContextSourceAutoDetect
)

// Middleware Config Functions.
var (
	// NewAuthMiddleware creates a new authentication middleware.
	NewAuthMiddleware = middleware.NewAuthMiddleware

	// DefaultContextConfig returns a ContextConfig with sensible defaults.
	DefaultContextConfig = middleware.DefaultContextConfig
)
