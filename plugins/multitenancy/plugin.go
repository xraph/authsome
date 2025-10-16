package multitenancy

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/spf13/viper"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/multitenancy/config"
	"github.com/xraph/authsome/plugins/multitenancy/handlers"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/authsome/plugins/multitenancy/repository"
	"github.com/xraph/forge"
	"github.com/uptrace/bun"
)

// Plugin implements the multi-tenancy plugin
type Plugin struct {
	// Core services
	orgService    *organization.Service
	configService *config.Service

	// Handlers
	orgHandler    *handlers.OrganizationHandler
	memberHandler *handlers.MemberHandler
	teamHandler   *handlers.TeamHandler

	// Database
	db *bun.DB

	// Configuration
	config Config
}

// Config holds the multi-tenancy plugin configuration
type Config struct {
	// PlatformOrganizationID is the ID of the platform organization
	PlatformOrganizationID string `json:"platformOrganizationId"`

	// DefaultOrganizationName is the name of the default organization in standalone mode
	DefaultOrganizationName string `json:"defaultOrganizationName"`

	// EnableOrganizationCreation allows users to create new organizations
	EnableOrganizationCreation bool `json:"enableOrganizationCreation"`

	// MaxMembersPerOrganization limits the number of members per organization
	MaxMembersPerOrganization int `json:"maxMembersPerOrganization"`

	// MaxTeamsPerOrganization limits the number of teams per organization
	MaxTeamsPerOrganization int `json:"maxTeamsPerOrganization"`

	// RequireInvitation requires invitation for joining organizations
	RequireInvitation bool `json:"requireInvitation"`

	// InvitationExpiryHours sets how long invitations are valid
	InvitationExpiryHours int `json:"invitationExpiryHours"`
}

// NewPlugin creates a new multi-tenancy plugin instance
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "multitenancy"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(auth interface{}) error {
	// Type assert to get the auth instance
	authInstance, ok := auth.(interface {
		GetDB() *bun.DB
		GetConfigManager() interface{}
		GetServiceRegistry() *registry.ServiceRegistry
	})
	if !ok {
		return fmt.Errorf("invalid auth instance type")
	}

	p.db = authInstance.GetDB()
	configManager := authInstance.GetConfigManager()
	serviceRegistry := authInstance.GetServiceRegistry()
	
	// Register models with Bun for relationships to work
	// Register TeamMember first as it's the join table for m2m relationships
	p.db.RegisterModel((*organization.TeamMember)(nil))
	p.db.RegisterModel(
		(*organization.Organization)(nil),
		(*organization.Member)(nil),
		(*organization.Team)(nil),
		(*organization.Invitation)(nil),
	)
	
	// Type assert to viper.Viper
	viperConfig, ok := configManager.(*viper.Viper)
	if !ok {
		return fmt.Errorf("config manager is not a viper instance")
	}
	
	// Bind plugin configuration
	if err := viperConfig.UnmarshalKey("auth.multitenancy", &p.config); err != nil {
		return fmt.Errorf("failed to bind multitenancy config: %w", err)
	}

	// Set default values
	if p.config.DefaultOrganizationName == "" {
		p.config.DefaultOrganizationName = "Default Organization"
	}
	if p.config.MaxMembersPerOrganization == 0 {
		p.config.MaxMembersPerOrganization = 100
	}
	if p.config.MaxTeamsPerOrganization == 0 {
		p.config.MaxTeamsPerOrganization = 10
	}
	if p.config.InvitationExpiryHours == 0 {
		p.config.InvitationExpiryHours = 72 // 3 days
	}

	// Create repositories
	orgRepo := repository.NewOrganizationRepository(p.db)
	memberRepo := repository.NewMemberRepository(p.db)
	teamRepo := repository.NewTeamRepository(p.db)
	invitationRepo := repository.NewInvitationRepository(p.db)

	// Create organization service config
	orgConfig := organization.Config{
		PlatformOrganizationID:    p.config.PlatformOrganizationID,
		DefaultOrganizationName:   p.config.DefaultOrganizationName,
		EnableOrganizationCreation: p.config.EnableOrganizationCreation,
		MaxMembersPerOrganization: p.config.MaxMembersPerOrganization,
		MaxTeamsPerOrganization:   p.config.MaxTeamsPerOrganization,
		RequireInvitation:         p.config.RequireInvitation,
		InvitationExpiryHours:     p.config.InvitationExpiryHours,
	}

	// Create services
	p.orgService = organization.NewService(orgConfig, orgRepo, memberRepo, teamRepo, invitationRepo)
	p.configService = config.NewService(viperConfig)

	// Create handlers
	p.orgHandler = handlers.NewOrganizationHandler(p.orgService)
	p.memberHandler = handlers.NewMemberHandler(p.orgService)
	p.teamHandler = handlers.NewTeamHandler(p.orgService)

	// Register services in the registry
	serviceRegistry.SetOrganizationService(p.orgService)
	serviceRegistry.SetConfigService(p.configService)

	return nil
}

// RegisterRoutes registers the plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router interface{}) error {
	forgeRouter, ok := router.(forge.Router)
	if !ok {
		return fmt.Errorf("invalid router type")
	}

	// Organization management routes
	orgGroup := forgeRouter.Group("/organizations")
	{
		orgGroup.POST("", p.orgHandler.CreateOrganization)
		orgGroup.GET("", p.orgHandler.ListOrganizations)
		orgGroup.GET("/:orgId", p.orgHandler.GetOrganization)
		orgGroup.PUT("/:orgId", p.orgHandler.UpdateOrganization)
		orgGroup.DELETE("/:orgId", p.orgHandler.DeleteOrganization)

		// Member management
		memberGroup := orgGroup.Group("/:orgId/members")
		{
			memberGroup.GET("", p.memberHandler.ListMembers)
			memberGroup.POST("/invite", p.memberHandler.InviteMember)
			memberGroup.PUT("/:memberId", p.memberHandler.UpdateMember)
			memberGroup.DELETE("/:memberId", p.memberHandler.RemoveMember)
		}

		// Team management
		teamGroup := orgGroup.Group("/:orgId/teams")
		{
			teamGroup.GET("", p.teamHandler.ListTeams)
			teamGroup.POST("", p.teamHandler.CreateTeam)
			teamGroup.GET("/:teamId", p.teamHandler.GetTeam)
			teamGroup.PUT("/:teamId", p.teamHandler.UpdateTeam)
			teamGroup.DELETE("/:teamId", p.teamHandler.DeleteTeam)
			teamGroup.POST("/:teamId/members", p.teamHandler.AddTeamMember)
			teamGroup.DELETE("/:teamId/members/:memberId", p.teamHandler.RemoveTeamMember)
		}
	}

	// Invitation routes
	inviteGroup := forgeRouter.Group("/invitations")
	{
		inviteGroup.GET("/:token", p.memberHandler.GetInvitation)
		inviteGroup.POST("/:token/accept", p.memberHandler.AcceptInvitation)
		inviteGroup.POST("/:token/decline", p.memberHandler.DeclineInvitation)
	}

	return nil
}

// RegisterHooks registers the plugin's hooks
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	// Register organization-related hooks
	hooks.RegisterAfterUserCreate(p.handleUserCreated)
	hooks.RegisterAfterUserDelete(p.handleUserDeleted)
	hooks.RegisterAfterSessionCreate(p.handleSessionCreated)

	return nil
}

// RegisterServiceDecorators replaces core services with multi-tenant aware versions
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// TODO: Implement decorators in future phases
	// These decorators will wrap core services to add multi-tenant functionality
	
	// // Decorate user service
	// if userService := services.UserService(); userService != nil {
	// 	decoratedUserService := decorators.NewMultiTenantUserDecorator(userService, p.orgService, p.configService)
	// 	services.ReplaceUserService(decoratedUserService)
	// }

	// // Decorate session service
	// if sessionService := services.SessionService(); sessionService != nil {
	// 	decoratedSessionService := decorators.NewMultiTenantSessionDecorator(sessionService, p.orgService, p.configService)
	// 	services.ReplaceSessionService(decoratedSessionService)
	// }

	// // Decorate auth service
	// if authService := services.AuthService(); authService != nil {
	// 	decoratedAuthService := decorators.NewMultiTenantAuthDecorator(authService, p.orgService, p.configService)
	// 	services.ReplaceAuthService(decoratedAuthService)
	// }

	// // Decorate JWT service
	// if jwtService := services.JWTService(); jwtService != nil {
	// 	decoratedJWTService := decorators.NewMultiTenantJWTDecorator(jwtService, p.orgService, p.configService)
	// 	services.ReplaceJWTService(decoratedJWTService)
	// }

	// // Decorate API Key service
	// if apikeyService := services.APIKeyService(); apikeyService != nil {
	// 	decoratedAPIKeyService := decorators.NewMultiTenantAPIKeyDecorator(apikeyService, p.orgService, p.configService)
	// 	services.ReplaceAPIKeyService(decoratedAPIKeyService)
	// }

	// // Decorate Forms service
	// if formsService := services.FormsService(); formsService != nil {
	// 	decoratedFormsService := decorators.NewMultiTenantFormsDecorator(formsService, p.orgService, p.configService)
	// 	services.ReplaceFormsService(decoratedFormsService)
	// }

	return nil
}

// Migrate runs the plugin's database migrations
func (p *Plugin) Migrate() error {
	ctx := context.Background()

	// Create organization table
	if _, err := p.db.NewCreateTable().
		Model((*organization.Organization)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create organizations table: %w", err)
	}

	// Create member table
	if _, err := p.db.NewCreateTable().
		Model((*organization.Member)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create members table: %w", err)
	}

	// Create team table
	if _, err := p.db.NewCreateTable().
		Model((*organization.Team)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create teams table: %w", err)
	}

	// Create team member table
	if _, err := p.db.NewCreateTable().
		Model((*organization.TeamMember)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create team_members table: %w", err)
	}

	// Create invitation table
	if _, err := p.db.NewCreateTable().
		Model((*organization.Invitation)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create invitations table: %w", err)
	}

	return nil
}

// Hook handlers

// handleUserCreated is called when a user is created
func (p *Plugin) handleUserCreated(ctx context.Context, u *user.User) error {
	// Add user to default organization
	defaultOrg, err := p.orgService.GetDefaultOrganization(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default organization: %w", err)
	}
	
	// Add user as member of default organization
	_, err = p.orgService.AddMember(ctx, defaultOrg.ID, u.ID.String(), "member")
	if err != nil {
		return fmt.Errorf("failed to add user to default organization: %w", err)
	}

	return nil
}

// handleUserDeleted is called when a user is deleted
func (p *Plugin) handleUserDeleted(ctx context.Context, userID xid.ID) error {
	// Remove user from all organizations
	return p.orgService.RemoveUserFromAllOrganizations(ctx, userID.String())
}

// handleSessionCreated is called when a session is created
func (p *Plugin) handleSessionCreated(ctx context.Context, s *session.Session) error {
	// Organization context is handled by the session service decorator
	// This hook is mainly for logging/auditing purposes
	return nil
}