package multiapp

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/multiapp/config"
	"github.com/xraph/authsome/plugins/multiapp/decorators"
	"github.com/xraph/authsome/plugins/multiapp/handlers"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the multi-tenancy plugin.
type Plugin struct {
	authInstance core.Authsome
	// Core services
	appService         *app.ServiceImpl
	configService      *config.Service
	environmentService environment.EnvironmentService

	// Handlers
	appHandler    *handlers.AppHandler
	memberHandler *handlers.MemberHandler
	teamHandler   *handlers.TeamHandler

	// Database
	db *bun.DB

	// Configuration
	config        Config
	defaultConfig Config

	// Logger
	logger forge.Logger
}

// Config holds the multi-tenancy plugin configuration.
type Config struct {
	// PlatformAppID is the ID of the platform app
	PlatformAppID xid.ID `json:"platformAppId"`

	// DefaultAppName is the name of the default app in standalone mode
	DefaultAppName string `json:"defaultAppName"`

	// EnableAppCreation allows users to create new apps (multitenancy mode)
	EnableAppCreation bool `json:"enableAppCreation"`

	// MaxMembersPerApp limits the number of members per app
	MaxMembersPerApp int `json:"maxMembersPerApp"`

	// MaxTeamsPerApp limits the number of teams per app
	MaxTeamsPerApp int `json:"maxTeamsPerApp"`

	// RequireInvitation requires invitation for joining apps
	RequireInvitation bool `json:"requireInvitation"`

	// InvitationExpiryHours sets how long invitations are valid
	InvitationExpiryHours int `json:"invitationExpiryHours"`

	// AutoCreateDefaultApp auto-creates default app on server start
	AutoCreateDefaultApp bool `json:"autoCreateDefaultApp"`

	// DefaultEnvironmentName is the name of the default dev environment
	DefaultEnvironmentName string `json:"defaultEnvironmentName"`
}

// PluginOption is a functional option for configuring the plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithPlatformAppID sets the platform app ID.
func WithPlatformAppID(id xid.ID) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.PlatformAppID = id
	}
}

// WithDefaultAppName sets the default app name.
func WithDefaultAppName(name string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DefaultAppName = name
	}
}

// WithEnableAppCreation sets whether app creation is enabled.
func WithEnableAppCreation(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.EnableAppCreation = enabled
	}
}

// WithMaxMembersPerApp sets the maximum members per app.
func WithMaxMembersPerApp(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxMembersPerApp = max
	}
}

// WithMaxTeamsPerApp sets the maximum teams per app.
func WithMaxTeamsPerApp(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxTeamsPerApp = max
	}
}

// WithRequireInvitation sets whether invitation is required.
func WithRequireInvitation(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireInvitation = required
	}
}

// WithInvitationExpiryHours sets the invitation expiry hours.
func WithInvitationExpiryHours(hours int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.InvitationExpiryHours = hours
	}
}

// NewPlugin creates a new multi-tenancy plugin instance with optional configuration.
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: Config{
			DefaultAppName:         "Platform App",
			MaxMembersPerApp:       1000,
			MaxTeamsPerApp:         100,
			InvitationExpiryHours:  72,
			AutoCreateDefaultApp:   true,
			DefaultEnvironmentName: "Development",
		},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string {
	return "multiapp"
}

// Init initializes the plugin with dependencies.
func (p *Plugin) Init(authInstance core.Authsome) error {
	if authInstance == nil {
		return errs.InternalServerError("invalid auth instance", nil)
	}

	p.authInstance = authInstance

	p.db = authInstance.GetDB()
	forgeApp := authInstance.GetForgeApp()
	configManager := forgeApp.Config()
	serviceRegistry := authInstance.GetServiceRegistry()

	// Get logger from Forge app
	p.logger = forgeApp.Logger().With(forge.F("plugin", "multiapp"))

	// Register models with Bun for relationships to work
	// Register TeamMember first as it's the join table for m2m relationships
	p.db.RegisterModel((*schema.TeamMember)(nil))
	p.db.RegisterModel(
		(*schema.App)(nil),
		(*schema.Member)(nil),
		(*schema.Team)(nil),
		(*schema.Invitation)(nil),
	)

	// Try to bind plugin configuration using Forge ConfigManager with provided defaults
	if err := configManager.BindWithDefault("auth.multiapp", &p.config, p.defaultConfig); err != nil {
		// Log but don't fail - use defaults
		p.logger.Warn("failed to bind multiapp config", forge.F("error", err.Error()))
	}

	// Set default values
	if p.config.DefaultAppName == "" {
		p.config.DefaultAppName = "Default App"
	}

	if p.config.DefaultEnvironmentName == "" {
		p.config.DefaultEnvironmentName = "Development"
	}

	if p.config.MaxMembersPerApp == 0 {
		p.config.MaxMembersPerApp = 100
	}

	if p.config.MaxTeamsPerApp == 0 {
		p.config.MaxTeamsPerApp = 10
	}

	if p.config.InvitationExpiryHours == 0 {
		p.config.InvitationExpiryHours = 72 // 3 days
	}

	// Create repositories using the consolidated repository package
	appRepo := authInstance.Repository().App()
	memberRepo := authInstance.Repository().App()
	teamRepo := authInstance.Repository().App()
	invitationRepo := authInstance.Repository().App()
	roleRepo := authInstance.Repository().Role()         // NEW: Role repository for RBAC
	userRoleRepo := authInstance.Repository().UserRole() // NEW: UserRole repository for RBAC

	// Create app service config
	appConfig := app.Config{
		PlatformAppID:         p.config.PlatformAppID,
		DefaultAppName:        p.config.DefaultAppName,
		EnableAppCreation:     p.config.EnableAppCreation,
		MaxMembersPerApp:      p.config.MaxMembersPerApp,
		MaxTeamsPerApp:        p.config.MaxTeamsPerApp,
		RequireInvitation:     p.config.RequireInvitation,
		InvitationExpiryHours: p.config.InvitationExpiryHours,
	}

	// Get RBAC service from registry
	rbacSvc := serviceRegistry.RBACService()
	if rbacSvc == nil {
		return errs.InternalServerError("RBAC service not available", nil)
	}

	// Use core app service directly
	p.appService = app.NewService(appRepo, memberRepo, teamRepo, invitationRepo, roleRepo, userRoleRepo, appConfig, rbacSvc)

	// Register the service with the service registry
	serviceRegistry.SetAppService(p.appService)

	// Create config service for app-specific config management
	// This wraps Forge's ConfigManager to provide multi-tenant configuration
	p.configService = config.NewService(configManager)
	// Note: Config service is not registered in ServiceRegistry as it has a different purpose
	// It provides app-scoped configuration overrides, not app entity management

	// Get environment repository for bootstrap
	envRepo := authInstance.Repository().Environment()
	if envRepo == nil {
		return errs.InternalServerError("environment repository not available", nil)
	}

	// Get environment service from core (initialized in authsome.go)
	// The core already initialized environment service with default config
	// We can optionally replace it with plugin-specific config if needed
	p.environmentService = serviceRegistry.EnvironmentService()
	if p.environmentService == nil {
		// Fallback: create new service if somehow not available (shouldn't happen)
		envConfig := environment.Config{
			AutoCreateDev:                  true,
			DefaultDevName:                 p.config.DefaultEnvironmentName,
			AllowPromotion:                 true,
			RequireConfirmationForDataCopy: true,
			MaxEnvironmentsPerApp:          10,
		}
		p.environmentService = environment.NewService(envRepo, envConfig)
		serviceRegistry.SetEnvironmentService(p.environmentService)
		p.logger.Warn("environment service was not in registry - created new instance")
	}

	// Bootstrap default app and environment on first initialization
	if p.config.AutoCreateDefaultApp {
		ctx := context.Background()

		bootstrap := environment.NewBootstrap(
			&appRepositoryAdapter{repo: appRepo}, // Adapt to environment.AppRepository interface
			envRepo,
			environment.BootstrapConfig{
				DefaultAppName:       p.config.DefaultAppName,
				DefaultAppSlug:       "platform",
				AutoCreateDefaultApp: true,
				MultitenancyEnabled:  p.config.EnableAppCreation,
			},
		)

		defaultApp, defaultEnv, err := bootstrap.EnsureDefaultApp(ctx)
		if err != nil {
			p.logger.Error("failed to bootstrap default app",
				forge.F("error", err.Error()))

			return fmt.Errorf("bootstrap failed: %w", err)
		}

		p.logger.Info("default app bootstrap complete",
			forge.F("app_id", defaultApp.ID.String()),
			forge.F("app_name", defaultApp.Name),
			forge.F("env_id", defaultEnv.ID.String()),
			forge.F("env_name", defaultEnv.Name))

		// Update platform app ID if not set
		if p.config.PlatformAppID.IsNil() {
			p.config.PlatformAppID = defaultApp.ID
		}
	}

	// Create handlers
	p.appHandler = handlers.NewAppHandler(p.appService)
	p.memberHandler = handlers.NewMemberHandler(p.appService)
	p.teamHandler = handlers.NewTeamHandler(p.appService)

	return nil
}

// RegisterRoutes registers the plugin's HTTP routes.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	gopts := p.authInstance.GetGlobalGroupRoutesOptions()

	// App management routes
	appGroup := router.Group("/apps", gopts...)
	{
		if err := appGroup.POST("", p.appHandler.CreateApp,
			forge.WithName("multitenancy.apps.create"),
			forge.WithSummary("Create app"),
			forge.WithDescription("Create a new app in multi-tenant mode"),
			forge.WithResponseSchema(200, "App created", app.App{}),
			forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
			forge.WithTags("Multiapp", "Apps"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}

		if err := appGroup.GET("", p.appHandler.ListApps,
			forge.WithName("multitenancy.apps.list"),
			forge.WithSummary("List apps"),
			forge.WithDescription("List all apps the user has access to"),
			forge.WithResponseSchema(200, "Apps retrieved", AppsListResponse{}),
			forge.WithResponseSchema(500, "Internal server error", MultitenancyErrorResponse{}),
			forge.WithTags("Multiapp", "Apps"),
		
		); err != nil {
			return err
		}

		if err := appGroup.GET("/:appId", p.appHandler.GetApp,
			forge.WithName("multitenancy.apps.get"),
			forge.WithSummary("Get app"),
			forge.WithDescription("Retrieve a specific app by ID"),
			forge.WithResponseSchema(200, "App retrieved", app.App{}),
			forge.WithResponseSchema(404, "App not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multiapp", "Apps"),
		
		); err != nil {
			return err
		}

		if err := appGroup.PUT("/:appId", p.appHandler.UpdateApp,
			forge.WithName("multitenancy.apps.update"),
			forge.WithSummary("Update app"),
			forge.WithDescription("Update app details (name, metadata, settings)"),
			forge.WithResponseSchema(200, "App updated", app.App{}),
			forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
			forge.WithResponseSchema(404, "App not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multiapp", "Apps"),
			forge.WithValidation(true),
		
		); err != nil {
			return err
		}

		if err := appGroup.DELETE("/:appId", p.appHandler.DeleteApp,
			forge.WithName("multitenancy.apps.delete"),
			forge.WithSummary("Delete app"),
			forge.WithDescription("Delete an app and all associated data. This action is irreversible."),
			forge.WithResponseSchema(200, "App deleted", MultitenancyStatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
			forge.WithResponseSchema(404, "App not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multiapp", "Apps"),
		
		); err != nil {
			return err
		}

		// Member management
		memberGroup := appGroup.Group("/:appId/members")
		{
			if err := memberGroup.GET("", p.memberHandler.ListMembers,
				forge.WithName("multitenancy.members.list"),
				forge.WithSummary("List app members"),
				forge.WithDescription("List all members of an app with their roles and status"),
				forge.WithResponseSchema(200, "Members retrieved", MembersListResponse{}),
				forge.WithResponseSchema(404, "App not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Members"),
			
			); err != nil {
				return err
			}

			if err := memberGroup.POST("/invite", p.memberHandler.InviteMember,
				forge.WithName("multitenancy.members.invite"),
				forge.WithSummary("Invite member to app"),
				forge.WithDescription("Send an invitation to a user to join the app"),
				forge.WithResponseSchema(200, "Invitation sent", app.Invitation{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Members"),
				forge.WithValidation(true),
			
			); err != nil {
				return err
			}

			if err := memberGroup.PUT("/:memberId", p.memberHandler.UpdateMember,
				forge.WithName("multitenancy.members.update"),
				forge.WithSummary("Update member"),
				forge.WithDescription("Update member role or status within the app"),
				forge.WithResponseSchema(200, "Member updated", app.Member{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Member not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Members"),
				forge.WithValidation(true),
			
			); err != nil {
				return err
			}

			if err := memberGroup.DELETE("/:memberId", p.memberHandler.RemoveMember,
				forge.WithName("multitenancy.members.remove"),
				forge.WithSummary("Remove member"),
				forge.WithDescription("Remove a member from the app"),
				forge.WithResponseSchema(200, "Member removed", MultitenancyStatusResponse{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Member not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Members"),
			
			); err != nil {
				return err
			}
		}

		// Team management
		teamGroup := appGroup.Group("/:appId/teams")
		{
			if err := teamGroup.GET("", p.teamHandler.ListTeams,
				forge.WithName("multitenancy.teams.list"),
				forge.WithSummary("List teams"),
				forge.WithDescription("List all teams within the app"),
				forge.WithResponseSchema(200, "Teams retrieved", TeamsListResponse{}),
				forge.WithResponseSchema(404, "App not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Teams"),
			
			); err != nil {
				return err
			}

			if err := teamGroup.POST("", p.teamHandler.CreateTeam,
				forge.WithName("multitenancy.teams.create"),
				forge.WithSummary("Create team"),
				forge.WithDescription("Create a new team within the app"),
				forge.WithResponseSchema(200, "Team created", app.Team{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Teams"),
				forge.WithValidation(true),
			
			); err != nil {
				return err
			}

			if err := teamGroup.GET("/:teamId", p.teamHandler.GetTeam,
				forge.WithName("multitenancy.teams.get"),
				forge.WithSummary("Get team"),
				forge.WithDescription("Retrieve a specific team by ID"),
				forge.WithResponseSchema(200, "Team retrieved", app.Team{}),
				forge.WithResponseSchema(404, "Team not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Teams"),
			
			); err != nil {
				return err
			}

			if err := teamGroup.PUT("/:teamId", p.teamHandler.UpdateTeam,
				forge.WithName("multitenancy.teams.update"),
				forge.WithSummary("Update team"),
				forge.WithDescription("Update team details (name, description, etc.)"),
				forge.WithResponseSchema(200, "Team updated", app.Team{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Team not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Teams"),
				forge.WithValidation(true),
			
			); err != nil {
				return err
			}

			if err := teamGroup.DELETE("/:teamId", p.teamHandler.DeleteTeam,
				forge.WithName("multitenancy.teams.delete"),
				forge.WithSummary("Delete team"),
				forge.WithDescription("Delete a team from the app"),
				forge.WithResponseSchema(200, "Team deleted", MultitenancyStatusResponse{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Team not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Teams"),
			
			); err != nil {
				return err
			}

			if err := teamGroup.POST("/:teamId/members", p.teamHandler.AddTeamMember,
				forge.WithName("multitenancy.teams.members.add"),
				forge.WithSummary("Add team member"),
				forge.WithDescription("Add a member to a team"),
				forge.WithResponseSchema(200, "Team member added", MultitenancyStatusResponse{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Team or member not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Teams"),
				forge.WithValidation(true),
			
			); err != nil {
				return err
			}

			if err := teamGroup.DELETE("/:teamId/members/:memberId", p.teamHandler.RemoveTeamMember,
				forge.WithName("multitenancy.teams.members.remove"),
				forge.WithSummary("Remove team member"),
				forge.WithDescription("Remove a member from a team"),
				forge.WithResponseSchema(200, "Team member removed", MultitenancyStatusResponse{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Team or member not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multiapp", "Apps", "Teams"),
			
			); err != nil {
				return err
			}
		}
	}

	// Invitation routes
	inviteGroup := router.Group("/invitations")
	{
		if err := inviteGroup.GET("/:token", p.memberHandler.GetInvitation,
			forge.WithName("multitenancy.invitations.get"),
			forge.WithSummary("Get invitation"),
			forge.WithDescription("Retrieve invitation details by token"),
			forge.WithResponseSchema(200, "Invitation retrieved", app.Invitation{}),
			forge.WithResponseSchema(404, "Invitation not found or expired", MultitenancyErrorResponse{}),
			forge.WithTags("Multiapp", "Invitations"),
		
		); err != nil {
			return err
		}

		if err := inviteGroup.POST("/:token/accept", p.memberHandler.AcceptInvitation,
			forge.WithName("multitenancy.invitations.accept"),
			forge.WithSummary("Accept invitation"),
			forge.WithDescription("Accept an app invitation and become a member"),
			forge.WithResponseSchema(200, "Invitation accepted", MultitenancyStatusResponse{}),
			forge.WithResponseSchema(400, "Invalid or expired invitation", MultitenancyErrorResponse{}),
			forge.WithResponseSchema(404, "Invitation not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multiapp", "Invitations"),
		
		); err != nil {
			return err
		}

		if err := inviteGroup.POST("/:token/decline", p.memberHandler.DeclineInvitation,
			forge.WithName("multitenancy.invitations.decline"),
			forge.WithSummary("Decline invitation"),
			forge.WithDescription("Decline an app invitation"),
			forge.WithResponseSchema(200, "Invitation declined", MultitenancyStatusResponse{}),
			forge.WithResponseSchema(404, "Invitation not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multiapp", "Invitations"),
		
		); err != nil {
			return err
		}
	}

	return nil
}

// DTOs for multitenancy routes

// MultitenancyErrorResponse represents an error response.
type MultitenancyErrorResponse struct {
	Error string `example:"Error message" json:"error"`
}

// MultitenancyStatusResponse represents a status response.
type MultitenancyStatusResponse struct {
	Status string `example:"success" json:"status"`
}

// AppsListResponse represents a list of apps.
type AppsListResponse []app.App

// MembersListResponse represents a list of members.
type MembersListResponse []app.Member

// TeamsListResponse represents a list of teams.
type TeamsListResponse []app.Team

// RegisterHooks registers the plugin's hooks.
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	// Register app-related hooks
	hooks.RegisterAfterUserCreate(p.handleUserCreated)
	hooks.RegisterAfterUserDelete(p.handleUserDeleted)
	hooks.RegisterAfterSessionCreate(p.handleSessionCreated)

	return nil
}

// RegisterServiceDecorators replaces core services with multi-tenant aware versions.
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Decorate user service with multi-tenancy support
	if userService := services.UserService(); userService != nil {
		decoratedUserService := decorators.NewMultiTenantUserService(userService, p.appService)
		services.ReplaceUserService(decoratedUserService)
	}

	// Decorate session service with multi-tenancy support
	if sessionService := services.SessionService(); sessionService != nil {
		decoratedSessionService := decorators.NewMultiTenantSessionService(sessionService, p.appService)
		services.ReplaceSessionService(decoratedSessionService)
	}

	// Decorate auth service with multi-tenancy support
	if authService := services.AuthService(); authService != nil {
		decoratedAuthService := decorators.NewMultiTenantAuthService(authService, p.appService)
		services.ReplaceAuthService(decoratedAuthService)
	}

	// TODO: Implement JWT, API Key, and Forms decorators when needed
	// These will follow the same pattern as above

	return nil
}

// Migrate runs the plugin's database migrations.
func (p *Plugin) Migrate() error {
	ctx := context.Background()

	// Create app table
	if _, err := p.db.NewCreateTable().
		Model((*app.App)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create apps table: %w", err)
	}

	// Create member table
	if _, err := p.db.NewCreateTable().
		Model((*app.Member)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create members table: %w", err)
	}

	// Create team table
	if _, err := p.db.NewCreateTable().
		Model((*app.Team)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create teams table: %w", err)
	}

	// Create team member table
	if _, err := p.db.NewCreateTable().
		Model((*app.TeamMember)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create team_members table: %w", err)
	}

	// Create invitation table
	if _, err := p.db.NewCreateTable().
		Model((*app.Invitation)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create invitations table: %w", err)
	}

	return nil
}

// Hook handlers

// handleUserCreated is called when a user is created
// This ensures app membership in both standalone and SaaS modes.
func (p *Plugin) handleUserCreated(ctx context.Context, u *user.User) error {
	// Check if this is the first user (no apps exist yet)
	response, err := p.appService.ListApps(ctx, &app.ListAppsFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 1,
		},
	})
	if err != nil || len(response.Data) == 0 {
		// This is the FIRST user - create the platform app
		// In both standalone and SaaS modes, this becomes the foundational app
		p.logger.Info("creating platform app for first user", forge.F("email", u.Email))

		platformSlug := "platform"
		if p.config.DefaultAppName != "" {
			platformSlug = p.config.DefaultAppName
		}

		platformApp, err := p.appService.CreateApp(ctx, &app.CreateAppRequest{
			Name: "Platform App",
			Slug: platformSlug,
		})
		if err != nil {
			return fmt.Errorf("failed to create platform app: %w", err)
		}

		// Add first user as OWNER of platform app (not just member)
		_, err = p.appService.Member.CreateMember(ctx, &app.Member{
			AppID:  platformApp.ID,
			UserID: u.ID,
			Role:   app.MemberRoleOwner,
			Status: app.MemberStatusActive,
		})
		if err != nil {
			return fmt.Errorf("failed to add first user as platform owner: %w", err)
		}

		p.logger.Info("platform app created",
			forge.F("name", platformApp.Name),
			forge.F("id", platformApp.ID.String()))
		p.logger.Info("first user is now platform owner",
			forge.F("email", u.Email),
			forge.F("role", "owner"))

		return nil
	}

	// Not the first user - behavior depends on mode
	// Get the platform/default app
	platformApp, err := p.appService.FindAppBySlug(ctx, "platform")
	if err != nil {
		// Fallback to default app if platform not found
		platformApp, err = p.appService.GetPlatformApp(ctx)
		if err != nil {
			return fmt.Errorf("failed to get platform/default app: %w", err)
		}
	}

	// In standalone mode, add all users to the platform app
	// In SaaS mode, users will create/join their own apps
	// For now, we'll add them to platform app in both modes (can be refined later)
	_, err = p.appService.Member.CreateMember(ctx, &app.Member{
		AppID:  platformApp.ID,
		UserID: u.ID,
		Role:   app.MemberRoleMember,
		Status: app.MemberStatusActive,
	})
	if err != nil {
		// Check if already a member
		if err.Error() != "user is already a member of this app" {
			return fmt.Errorf("failed to add user to platform app: %w", err)
		}

		p.logger.Info("user already member of platform app", forge.F("email", u.Email))
	} else {
		p.logger.Info("user added to platform app as member", forge.F("email", u.Email))
	}

	return nil
}

// handleUserDeleted is called when a user is deleted.
func (p *Plugin) handleUserDeleted(ctx context.Context, userID xid.ID) error {
	// Note: Member cleanup should be handled at the repository level with cascade delete
	// or we need to add a DeleteMembersByUserID method to the AppService interface
	// For now, we rely on database constraints
	p.logger.Info("user deleted - member cleanup handled by database constraints",
		forge.F("user_id", userID.String()))

	return nil
}

// handleSessionCreated is called when a session is created.
func (p *Plugin) handleSessionCreated(ctx context.Context, s *session.Session) error {
	// App context is handled by the session service decorator
	// This hook is mainly for logging/auditing purposes
	return nil
}

// appRepositoryAdapter adapts app.AppRepository to environment.AppRepository.
type appRepositoryAdapter struct {
	repo app.AppRepository
}

func (a *appRepositoryAdapter) Count(ctx context.Context) (int, error) {
	return a.repo.CountApps(ctx)
}

func (a *appRepositoryAdapter) Create(ctx context.Context, appEntity *schema.App) error {
	return a.repo.CreateApp(ctx, appEntity)
}

func (a *appRepositoryAdapter) FindBySlug(ctx context.Context, slug string) (*schema.App, error) {
	return a.repo.FindAppBySlug(ctx, slug)
}
