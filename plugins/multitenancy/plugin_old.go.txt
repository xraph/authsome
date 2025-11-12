package multitenancy

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/multitenancy/config"
	"github.com/xraph/authsome/plugins/multitenancy/decorators"
	"github.com/xraph/authsome/plugins/multitenancy/handlers"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/authsome/plugins/multitenancy/repository"
	"github.com/xraph/forge"
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
	config        Config
	defaultConfig Config

	// Logger
	logger forge.Logger
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

// PluginOption is a functional option for configuring the plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithPlatformOrganizationID sets the platform organization ID
func WithPlatformOrganizationID(id string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.PlatformOrganizationID = id
	}
}

// WithDefaultOrganizationName sets the default organization name
func WithDefaultOrganizationName(name string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DefaultOrganizationName = name
	}
}

// WithEnableOrganizationCreation sets whether organization creation is enabled
func WithEnableOrganizationCreation(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.EnableOrganizationCreation = enabled
	}
}

// WithMaxMembersPerOrganization sets the maximum members per organization
func WithMaxMembersPerOrganization(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxMembersPerOrganization = max
	}
}

// WithMaxTeamsPerOrganization sets the maximum teams per organization
func WithMaxTeamsPerOrganization(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxTeamsPerOrganization = max
	}
}

// WithRequireInvitation sets whether invitation is required
func WithRequireInvitation(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireInvitation = required
	}
}

// WithInvitationExpiryHours sets the invitation expiry hours
func WithInvitationExpiryHours(hours int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.InvitationExpiryHours = hours
	}
}

// NewPlugin creates a new multi-tenancy plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: Config{
			DefaultOrganizationName:   "Platform Organization",
			MaxMembersPerOrganization: 1000,
			MaxTeamsPerOrganization:   100,
			InvitationExpiryHours:     72,
		},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
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
		GetForgeApp() forge.App
		GetServiceRegistry() *registry.ServiceRegistry
	})
	if !ok {
		return fmt.Errorf("invalid auth instance type")
	}

	p.db = authInstance.GetDB()
	forgeApp := authInstance.GetForgeApp()
	configManager := forgeApp.Config()
	serviceRegistry := authInstance.GetServiceRegistry()

	// Get logger from Forge app
	p.logger = forgeApp.Logger().With(forge.F("plugin", "multitenancy"))

	// Register models with Bun for relationships to work
	// Register TeamMember first as it's the join table for m2m relationships
	p.db.RegisterModel((*organization.TeamMember)(nil))
	p.db.RegisterModel(
		(*organization.Organization)(nil),
		(*organization.Member)(nil),
		(*organization.Team)(nil),
		(*organization.Invitation)(nil),
	)

	// Try to bind plugin configuration using Forge ConfigManager with provided defaults
	if err := configManager.BindWithDefault("auth.multitenancy", &p.config, p.defaultConfig); err != nil {
		// Log but don't fail - use defaults
		p.logger.Warn("failed to bind multitenancy config", forge.F("error", err.Error()))
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
		PlatformOrganizationID:     p.config.PlatformOrganizationID,
		DefaultOrganizationName:    p.config.DefaultOrganizationName,
		EnableOrganizationCreation: p.config.EnableOrganizationCreation,
		MaxMembersPerOrganization:  p.config.MaxMembersPerOrganization,
		MaxTeamsPerOrganization:    p.config.MaxTeamsPerOrganization,
		RequireInvitation:          p.config.RequireInvitation,
		InvitationExpiryHours:      p.config.InvitationExpiryHours,
	}

	// Create services
	p.orgService = organization.NewService(orgConfig, orgRepo, memberRepo, teamRepo, invitationRepo)

	// Create config service for org-specific config management
	// This wraps Forge's ConfigManager to provide multi-tenant configuration
	p.configService = config.NewService(configManager)

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
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Organization management routes
	orgGroup := router.Group("/organizations")
	{
		orgGroup.POST("", p.orgHandler.CreateOrganization,
			forge.WithName("multitenancy.organizations.create"),
			forge.WithSummary("Create organization"),
			forge.WithDescription("Create a new organization in multi-tenant mode"),
			forge.WithResponseSchema(200, "Organization created", organization.Organization{}),
			forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
			forge.WithTags("Multitenancy", "Organizations"),
			forge.WithValidation(true),
		)

		orgGroup.GET("", p.orgHandler.ListOrganizations,
			forge.WithName("multitenancy.organizations.list"),
			forge.WithSummary("List organizations"),
			forge.WithDescription("List all organizations the user has access to"),
			forge.WithResponseSchema(200, "Organizations retrieved", OrganizationsListResponse{}),
			forge.WithResponseSchema(500, "Internal server error", MultitenancyErrorResponse{}),
			forge.WithTags("Multitenancy", "Organizations"),
		)

		orgGroup.GET("/:orgId", p.orgHandler.GetOrganization,
			forge.WithName("multitenancy.organizations.get"),
			forge.WithSummary("Get organization"),
			forge.WithDescription("Retrieve a specific organization by ID"),
			forge.WithResponseSchema(200, "Organization retrieved", organization.Organization{}),
			forge.WithResponseSchema(404, "Organization not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multitenancy", "Organizations"),
		)

		orgGroup.PUT("/:orgId", p.orgHandler.UpdateOrganization,
			forge.WithName("multitenancy.organizations.update"),
			forge.WithSummary("Update organization"),
			forge.WithDescription("Update organization details (name, metadata, settings)"),
			forge.WithResponseSchema(200, "Organization updated", organization.Organization{}),
			forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
			forge.WithResponseSchema(404, "Organization not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multitenancy", "Organizations"),
			forge.WithValidation(true),
		)

		orgGroup.DELETE("/:orgId", p.orgHandler.DeleteOrganization,
			forge.WithName("multitenancy.organizations.delete"),
			forge.WithSummary("Delete organization"),
			forge.WithDescription("Delete an organization and all associated data. This action is irreversible."),
			forge.WithResponseSchema(200, "Organization deleted", MultitenancyStatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
			forge.WithResponseSchema(404, "Organization not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multitenancy", "Organizations"),
		)

		// Member management
		memberGroup := orgGroup.Group("/:orgId/members")
		{
			memberGroup.GET("", p.memberHandler.ListMembers,
				forge.WithName("multitenancy.members.list"),
				forge.WithSummary("List organization members"),
				forge.WithDescription("List all members of an organization with their roles and status"),
				forge.WithResponseSchema(200, "Members retrieved", MembersListResponse{}),
				forge.WithResponseSchema(404, "Organization not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Members"),
			)

			memberGroup.POST("/invite", p.memberHandler.InviteMember,
				forge.WithName("multitenancy.members.invite"),
				forge.WithSummary("Invite member to organization"),
				forge.WithDescription("Send an invitation to a user to join the organization"),
				forge.WithResponseSchema(200, "Invitation sent", organization.Invitation{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Members"),
				forge.WithValidation(true),
			)

			memberGroup.PUT("/:memberId", p.memberHandler.UpdateMember,
				forge.WithName("multitenancy.members.update"),
				forge.WithSummary("Update member"),
				forge.WithDescription("Update member role or status within the organization"),
				forge.WithResponseSchema(200, "Member updated", organization.Member{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Member not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Members"),
				forge.WithValidation(true),
			)

			memberGroup.DELETE("/:memberId", p.memberHandler.RemoveMember,
				forge.WithName("multitenancy.members.remove"),
				forge.WithSummary("Remove member"),
				forge.WithDescription("Remove a member from the organization"),
				forge.WithResponseSchema(200, "Member removed", MultitenancyStatusResponse{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Member not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Members"),
			)
		}

		// Team management
		teamGroup := orgGroup.Group("/:orgId/teams")
		{
			teamGroup.GET("", p.teamHandler.ListTeams,
				forge.WithName("multitenancy.teams.list"),
				forge.WithSummary("List teams"),
				forge.WithDescription("List all teams within the organization"),
				forge.WithResponseSchema(200, "Teams retrieved", TeamsListResponse{}),
				forge.WithResponseSchema(404, "Organization not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Teams"),
			)

			teamGroup.POST("", p.teamHandler.CreateTeam,
				forge.WithName("multitenancy.teams.create"),
				forge.WithSummary("Create team"),
				forge.WithDescription("Create a new team within the organization"),
				forge.WithResponseSchema(200, "Team created", organization.Team{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Teams"),
				forge.WithValidation(true),
			)

			teamGroup.GET("/:teamId", p.teamHandler.GetTeam,
				forge.WithName("multitenancy.teams.get"),
				forge.WithSummary("Get team"),
				forge.WithDescription("Retrieve a specific team by ID"),
				forge.WithResponseSchema(200, "Team retrieved", organization.Team{}),
				forge.WithResponseSchema(404, "Team not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Teams"),
			)

			teamGroup.PUT("/:teamId", p.teamHandler.UpdateTeam,
				forge.WithName("multitenancy.teams.update"),
				forge.WithSummary("Update team"),
				forge.WithDescription("Update team details (name, description, etc.)"),
				forge.WithResponseSchema(200, "Team updated", organization.Team{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Team not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Teams"),
				forge.WithValidation(true),
			)

			teamGroup.DELETE("/:teamId", p.teamHandler.DeleteTeam,
				forge.WithName("multitenancy.teams.delete"),
				forge.WithSummary("Delete team"),
				forge.WithDescription("Delete a team from the organization"),
				forge.WithResponseSchema(200, "Team deleted", MultitenancyStatusResponse{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Team not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Teams"),
			)

			teamGroup.POST("/:teamId/members", p.teamHandler.AddTeamMember,
				forge.WithName("multitenancy.teams.members.add"),
				forge.WithSummary("Add team member"),
				forge.WithDescription("Add a member to a team"),
				forge.WithResponseSchema(200, "Team member added", MultitenancyStatusResponse{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Team or member not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Teams"),
				forge.WithValidation(true),
			)

			teamGroup.DELETE("/:teamId/members/:memberId", p.teamHandler.RemoveTeamMember,
				forge.WithName("multitenancy.teams.members.remove"),
				forge.WithSummary("Remove team member"),
				forge.WithDescription("Remove a member from a team"),
				forge.WithResponseSchema(200, "Team member removed", MultitenancyStatusResponse{}),
				forge.WithResponseSchema(400, "Invalid request", MultitenancyErrorResponse{}),
				forge.WithResponseSchema(404, "Team or member not found", MultitenancyErrorResponse{}),
				forge.WithTags("Multitenancy", "Organizations", "Teams"),
			)
		}
	}

	// Invitation routes
	inviteGroup := router.Group("/invitations")
	{
		inviteGroup.GET("/:token", p.memberHandler.GetInvitation,
			forge.WithName("multitenancy.invitations.get"),
			forge.WithSummary("Get invitation"),
			forge.WithDescription("Retrieve invitation details by token"),
			forge.WithResponseSchema(200, "Invitation retrieved", organization.Invitation{}),
			forge.WithResponseSchema(404, "Invitation not found or expired", MultitenancyErrorResponse{}),
			forge.WithTags("Multitenancy", "Invitations"),
		)

		inviteGroup.POST("/:token/accept", p.memberHandler.AcceptInvitation,
			forge.WithName("multitenancy.invitations.accept"),
			forge.WithSummary("Accept invitation"),
			forge.WithDescription("Accept an organization invitation and become a member"),
			forge.WithResponseSchema(200, "Invitation accepted", MultitenancyStatusResponse{}),
			forge.WithResponseSchema(400, "Invalid or expired invitation", MultitenancyErrorResponse{}),
			forge.WithResponseSchema(404, "Invitation not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multitenancy", "Invitations"),
		)

		inviteGroup.POST("/:token/decline", p.memberHandler.DeclineInvitation,
			forge.WithName("multitenancy.invitations.decline"),
			forge.WithSummary("Decline invitation"),
			forge.WithDescription("Decline an organization invitation"),
			forge.WithResponseSchema(200, "Invitation declined", MultitenancyStatusResponse{}),
			forge.WithResponseSchema(404, "Invitation not found", MultitenancyErrorResponse{}),
			forge.WithTags("Multitenancy", "Invitations"),
		)
	}

	return nil
}

// DTOs for multitenancy routes

// MultitenancyErrorResponse represents an error response
type MultitenancyErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

// MultitenancyStatusResponse represents a status response
type MultitenancyStatusResponse struct {
	Status string `json:"status" example:"success"`
}

// OrganizationsListResponse represents a list of organizations
type OrganizationsListResponse []organization.Organization

// MembersListResponse represents a list of members
type MembersListResponse []organization.Member

// TeamsListResponse represents a list of teams
type TeamsListResponse []organization.Team

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
	// Decorate user service with multi-tenancy support
	if userService := services.UserService(); userService != nil {
		decoratedUserService := decorators.NewMultiTenantUserService(userService, p.orgService)
		services.ReplaceUserService(decoratedUserService)
	}

	// Decorate session service with multi-tenancy support
	if sessionService := services.SessionService(); sessionService != nil {
		decoratedSessionService := decorators.NewMultiTenantSessionService(sessionService, p.orgService)
		services.ReplaceSessionService(decoratedSessionService)
	}

	// Decorate auth service with multi-tenancy support
	if authService := services.AuthService(); authService != nil {
		decoratedAuthService := decorators.NewMultiTenantAuthService(authService, p.orgService)
		services.ReplaceAuthService(decoratedAuthService)
	}

	// TODO: Implement JWT, API Key, and Forms decorators when needed
	// These will follow the same pattern as above

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
// This ensures organization membership in both standalone and SaaS modes
func (p *Plugin) handleUserCreated(ctx context.Context, u *user.User) error {
	// Check if this is the first user (no organizations exist yet)
	orgs, err := p.orgService.ListOrganizations(ctx, 1, 0)
	if err != nil || len(orgs) == 0 {
		// This is the FIRST user - create the platform organization
		// In both standalone and SaaS modes, this becomes the foundational organization
		p.logger.Info("creating platform organization for first user", forge.F("email", u.Email))

		platformSlug := p.config.PlatformOrganizationID
		if platformSlug == "" {
			platformSlug = "platform"
		}

		platformOrg, err := p.orgService.CreateOrganization(ctx, &organization.CreateOrganizationRequest{
			Name: "Platform Organization",
			Slug: platformSlug,
		}, u.ID)
		if err != nil {
			return fmt.Errorf("failed to create platform organization: %w", err)
		}

		// Add first user as OWNER of platform organization (not just member)
		_, err = p.orgService.AddMember(ctx, platformOrg.ID, u.ID, organization.RoleOwner)
		if err != nil {
			return fmt.Errorf("failed to add first user as platform owner: %w", err)
		}

		p.logger.Info("platform organization created",
			forge.F("name", platformOrg.Name),
			forge.F("id", platformOrg.ID.String()))
		p.logger.Info("first user is now platform owner",
			forge.F("email", u.Email),
			forge.F("role", string(organization.RoleOwner)))
		return nil
	}

	// Not the first user - behavior depends on mode
	// Get the platform/default organization
	platformOrg, err := p.orgService.GetOrganizationBySlug(ctx, "platform")
	if err != nil {
		// Fallback to default organization if platform not found
		platformOrg, err = p.orgService.GetDefaultOrganization(ctx)
		if err != nil {
			return fmt.Errorf("failed to get platform/default organization: %w", err)
		}
	}

	// In standalone mode, add all users to the platform organization
	// In SaaS mode, users will create/join their own organizations
	// For now, we'll add them to platform org in both modes (can be refined later)
	_, err = p.orgService.AddMember(ctx, platformOrg.ID, u.ID, organization.RoleMember)
	if err != nil {
		// Check if already a member
		if err.Error() != "user is already a member of this organization" {
			return fmt.Errorf("failed to add user to platform organization: %w", err)
		}
		p.logger.Info("user already member of platform organization", forge.F("email", u.Email))
	} else {
		p.logger.Info("user added to platform organization as member", forge.F("email", u.Email))
	}

	return nil
}

// handleUserDeleted is called when a user is deleted
func (p *Plugin) handleUserDeleted(ctx context.Context, userID xid.ID) error {
	// Remove user from all organizations
	return p.orgService.RemoveUserFromAllOrganizations(ctx, userID)
}

// handleSessionCreated is called when a session is created
func (p *Plugin) handleSessionCreated(ctx context.Context, s *session.Session) error {
	// Organization context is handled by the session service decorator
	// This hook is mainly for logging/auditing purposes
	return nil
}
