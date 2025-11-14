package organization

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/registry"
	orgrepo "github.com/xraph/authsome/repository/organization"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the user-created organizations plugin
type Plugin struct {
	// Core services
	orgService *organization.Service

	// Handlers
	orgHandler *OrganizationHandler

	// Database
	db *bun.DB

	// Configuration
	config        Config
	defaultConfig Config

	// Logger
	logger forge.Logger
}

// PluginOption is a functional option for configuring the plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithMaxOrganizationsPerUser sets the maximum organizations per user
func WithMaxOrganizationsPerUser(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxOrganizationsPerUser = max
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

// WithEnableUserCreation sets whether user creation is enabled
func WithEnableUserCreation(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.EnableUserCreation = enabled
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

// NewPlugin creates a new organization plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: Config{
			MaxOrganizationsPerUser:   5,
			MaxMembersPerOrganization: 50,
			MaxTeamsPerOrganization:   20,
			EnableUserCreation:        true,
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
	return "organization"
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
	p.logger = forgeApp.Logger().With(forge.F("plugin", "organization"))

	// Get RBAC service from registry
	rbacSvc := serviceRegistry.RBACService()
	if rbacSvc == nil {
		p.logger.Warn("RBAC service not available, authorization checks may not work properly")
	}

	// Register schema models with Bun for relationships to work
	// Register OrganizationTeamMember first as it's the join table for m2m relationships
	p.db.RegisterModel((*schema.OrganizationTeamMember)(nil))
	p.db.RegisterModel(
		(*schema.Organization)(nil),
		(*schema.OrganizationMember)(nil),
		(*schema.OrganizationTeam)(nil),
		(*schema.OrganizationInvitation)(nil),
	)

	// Try to bind plugin configuration using Forge ConfigManager with provided defaults
	if err := configManager.BindWithDefault("auth.organization", &p.config, p.defaultConfig); err != nil {
		// Log but don't fail - use defaults
		p.logger.Warn("failed to bind organization config", forge.F("error", err.Error()))
	}

	// Set default values
	if p.config.MaxOrganizationsPerUser == 0 {
		p.config.MaxOrganizationsPerUser = 5
	}
	if p.config.MaxMembersPerOrganization == 0 {
		p.config.MaxMembersPerOrganization = 50
	}
	if p.config.MaxTeamsPerOrganization == 0 {
		p.config.MaxTeamsPerOrganization = 20
	}
	if p.config.InvitationExpiryHours == 0 {
		p.config.InvitationExpiryHours = 72 // 3 days
	}

	// Create repositories
	orgRepo := orgrepo.NewOrganizationRepository(p.db)
	memberRepo := orgrepo.NewOrganizationMemberRepository(p.db)
	teamRepo := orgrepo.NewOrganizationTeamRepository(p.db)
	invitationRepo := orgrepo.NewOrganizationInvitationRepository(p.db)

	// Create organization service config
	orgConfig := Config{
		MaxOrganizationsPerUser:   p.config.MaxOrganizationsPerUser,
		MaxMembersPerOrganization: p.config.MaxMembersPerOrganization,
		MaxTeamsPerOrganization:   p.config.MaxTeamsPerOrganization,
		EnableUserCreation:        p.config.EnableUserCreation,
		RequireInvitation:         p.config.RequireInvitation,
		InvitationExpiryHours:     p.config.InvitationExpiryHours,
	}

	// Create services with actual repositories and RBAC service
	p.orgService = NewService(
		orgRepo,
		memberRepo,
		teamRepo,
		invitationRepo,
		orgConfig,
		rbacSvc,
	)

	// Create handlers
	p.orgHandler = &OrganizationHandler{
		orgService: p.orgService,
	}

	p.logger.Info("organization plugin initialized",
		forge.F("max_orgs_per_user", p.config.MaxOrganizationsPerUser),
		forge.F("max_members_per_org", p.config.MaxMembersPerOrganization))

	return nil
}

// RegisterRoutes registers the plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Organization management routes
	orgGroup := router.Group("/organizations")
	{
		orgGroup.POST("", p.orgHandler.CreateOrganization,
			forge.WithName("organization.create"),
			forge.WithSummary("Create organization"),
			forge.WithDescription("Create a new user organization (workspace)"),
			forge.WithResponseSchema(201, "Organization created", schema.Organization{}),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithTags("Organizations"),
			forge.WithValidation(true),
		)

		orgGroup.GET("", p.orgHandler.ListOrganizations,
			forge.WithName("organization.list"),
			forge.WithSummary("List user organizations"),
			forge.WithDescription("List all organizations the current user is a member of"),
			forge.WithResponseSchema(200, "Organizations retrieved", schema.Organization{}),
			forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
			forge.WithTags("Organizations"),
		)

		orgGroup.GET("/:id", p.orgHandler.GetOrganization,
			forge.WithName("organization.get"),
			forge.WithSummary("Get organization"),
			forge.WithDescription("Retrieve a specific organization by ID"),
			forge.WithResponseSchema(200, "Organization retrieved", schema.Organization{}),
			forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
			forge.WithTags("Organizations"),
		)

		orgGroup.GET("/slug/:slug", p.orgHandler.GetOrganizationBySlug,
			forge.WithName("organization.get_by_slug"),
			forge.WithSummary("Get organization by slug"),
			forge.WithDescription("Retrieve a specific organization by its slug"),
			forge.WithResponseSchema(200, "Organization retrieved", schema.Organization{}),
			forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
			forge.WithTags("Organizations"),
		)

		orgGroup.PATCH("/:id", p.orgHandler.UpdateOrganization,
			forge.WithName("organization.update"),
			forge.WithSummary("Update organization"),
			forge.WithDescription("Update organization details (name, logo, metadata)"),
			forge.WithResponseSchema(200, "Organization updated", schema.Organization{}),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
			forge.WithTags("Organizations"),
			forge.WithValidation(true),
		)

		orgGroup.DELETE("/:id", p.orgHandler.DeleteOrganization,
			forge.WithName("organization.delete"),
			forge.WithSummary("Delete organization"),
			forge.WithDescription("Delete an organization (owner only). This action is irreversible."),
			forge.WithResponseSchema(204, "Organization deleted", nil),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
			forge.WithTags("Organizations"),
		)

		// Member management
		memberGroup := orgGroup.Group("/:id/members")
		{
			memberGroup.GET("", p.orgHandler.ListMembers,
				forge.WithName("organization.members.list"),
				forge.WithSummary("List organization members"),
				forge.WithDescription("List all members of an organization with their roles and status"),
				forge.WithResponseSchema(200, "Members retrieved", schema.OrganizationMember{}),
				forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
				forge.WithTags("Organizations", "Members"),
			)

			memberGroup.POST("/invite", p.orgHandler.InviteMember,
				forge.WithName("organization.members.invite"),
				forge.WithSummary("Invite member to organization"),
				forge.WithDescription("Send an invitation to a user to join the organization"),
				forge.WithResponseSchema(201, "Invitation sent", schema.OrganizationInvitation{}),
				forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
				forge.WithTags("Organizations", "Members"),
				forge.WithValidation(true),
			)

			memberGroup.PATCH("/:memberId", p.orgHandler.UpdateMember,
				forge.WithName("organization.members.update"),
				forge.WithSummary("Update member"),
				forge.WithDescription("Update member role or status within the organization"),
				forge.WithResponseSchema(200, "Member updated", schema.OrganizationMember{}),
				forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
				forge.WithResponseSchema(404, "Member not found", ErrorResponse{}),
				forge.WithTags("Organizations", "Members"),
				forge.WithValidation(true),
			)

			memberGroup.DELETE("/:memberId", p.orgHandler.RemoveMember,
				forge.WithName("organization.members.remove"),
				forge.WithSummary("Remove member"),
				forge.WithDescription("Remove a member from the organization"),
				forge.WithResponseSchema(204, "Member removed", nil),
				forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
				forge.WithResponseSchema(404, "Member not found", ErrorResponse{}),
				forge.WithTags("Organizations", "Members"),
			)
		}

		// Team management
		teamGroup := orgGroup.Group("/:id/teams")
		{
			teamGroup.GET("", p.orgHandler.ListTeams,
				forge.WithName("organization.teams.list"),
				forge.WithSummary("List teams"),
				forge.WithDescription("List all teams within the organization"),
				forge.WithResponseSchema(200, "Teams retrieved", schema.OrganizationTeam{}),
				forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
				forge.WithTags("Organizations", "Teams"),
			)

			teamGroup.POST("", p.orgHandler.CreateTeam,
				forge.WithName("organization.teams.create"),
				forge.WithSummary("Create team"),
				forge.WithDescription("Create a new team within the organization"),
				forge.WithResponseSchema(201, "Team created", schema.OrganizationTeam{}),
				forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
				forge.WithTags("Organizations", "Teams"),
				forge.WithValidation(true),
			)

			teamGroup.PATCH("/:teamId", p.orgHandler.UpdateTeam,
				forge.WithName("organization.teams.update"),
				forge.WithSummary("Update team"),
				forge.WithDescription("Update team details (name, description, etc.)"),
				forge.WithResponseSchema(200, "Team updated", schema.OrganizationTeam{}),
				forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
				forge.WithResponseSchema(404, "Team not found", ErrorResponse{}),
				forge.WithTags("Organizations", "Teams"),
				forge.WithValidation(true),
			)

			teamGroup.DELETE("/:teamId", p.orgHandler.DeleteTeam,
				forge.WithName("organization.teams.delete"),
				forge.WithSummary("Delete team"),
				forge.WithDescription("Delete a team from the organization"),
				forge.WithResponseSchema(204, "Team deleted", nil),
				forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
				forge.WithResponseSchema(404, "Team not found", ErrorResponse{}),
				forge.WithTags("Organizations", "Teams"),
			)
		}
	}

	// Invitation routes
	inviteGroup := router.Group("/organization-invitations")
	{
		inviteGroup.POST("/:token/accept", p.orgHandler.AcceptInvitation,
			forge.WithName("organization.invitations.accept"),
			forge.WithSummary("Accept organization invitation"),
			forge.WithDescription("Accept an organization invitation and become a member"),
			forge.WithResponseSchema(200, "Invitation accepted", schema.OrganizationMember{}),
			forge.WithResponseSchema(400, "Invalid or expired invitation", ErrorResponse{}),
			forge.WithResponseSchema(404, "Invitation not found", ErrorResponse{}),
			forge.WithTags("Organizations", "Invitations"),
		)

		inviteGroup.POST("/:token/decline", p.orgHandler.DeclineInvitation,
			forge.WithName("organization.invitations.decline"),
			forge.WithSummary("Decline organization invitation"),
			forge.WithDescription("Decline an organization invitation"),
			forge.WithResponseSchema(200, "Invitation declined", StatusResponse{}),
			forge.WithResponseSchema(404, "Invitation not found", ErrorResponse{}),
			forge.WithTags("Organizations", "Invitations"),
		)
	}

	return nil
}

// RegisterHooks registers the plugin's hooks
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	// Register organization-related hooks
	// hooks.RegisterAfterUserDelete(p.handleUserDeleted)

	return nil
}

// Migrate runs the plugin's database migrations
func (p *Plugin) Migrate() error {
	ctx := context.Background()

	// Create organization table
	if _, err := p.db.NewCreateTable().
		Model((*schema.Organization)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create organizations table: %w", err)
	}

	// Create organization member table
	if _, err := p.db.NewCreateTable().
		Model((*schema.OrganizationMember)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create organization_members table: %w", err)
	}

	// Create organization team table
	if _, err := p.db.NewCreateTable().
		Model((*schema.OrganizationTeam)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create organization_teams table: %w", err)
	}

	// Create organization team member table
	if _, err := p.db.NewCreateTable().
		Model((*schema.OrganizationTeamMember)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create organization_team_members table: %w", err)
	}

	// Create organization invitation table
	if _, err := p.db.NewCreateTable().
		Model((*schema.OrganizationInvitation)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create organization_invitations table: %w", err)
	}

	return nil
}

// DTOs for organization routes

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

// StatusResponse represents a status response
type StatusResponse struct {
	Status string `json:"status" example:"success"`
}

// OrganizationsListResponse represents a list of organizations
type OrganizationsListResponse []schema.Organization

// MembersListResponse represents a list of members
type MembersListResponse []schema.OrganizationMember

// TeamsListResponse represents a list of teams
type TeamsListResponse []schema.OrganizationTeam
