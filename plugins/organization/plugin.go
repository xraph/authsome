package organization

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
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

	// Dashboard (lazy initialized)
	dashboardExtension     *DashboardExtension
	dashboardExtensionOnce sync.Once

	// UI Registry for organization page extensions
	uiRegistry *OrganizationUIRegistry

	// Database
	db *bun.DB

	// Configuration
	config        Config
	defaultConfig Config

	// Logger
	logger forge.Logger

	// RBAC service
	rbacService *rbac.Service

	// Notification adapter
	notifAdapter interface{}

	// Auth instance (for accessing service registry and hooks)
	authInst core.Authsome

	// Flag to prevent double initialization
	initialized bool
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

// WithAllowAppLevelRoles sets whether app-level (global) RBAC roles can be used for organization membership
func WithAllowAppLevelRoles(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowAppLevelRoles = allow
	}
}

// WithInvitationExpiryHours sets the invitation expiry hours
func WithInvitationExpiryHours(hours int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.InvitationExpiryHours = hours
	}
}

// WithEnforceUniqueSlug sets whether to enforce unique slugs within app+environment scope
func WithEnforceUniqueSlug(enforce bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.EnforceUniqueSlug = enforce
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
			EnforceUniqueSlug:         true,
			AllowAppLevelRoles:        true, // Allow app-level roles by default
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
func (p *Plugin) Init(authInstance core.Authsome) error {
	if authInstance == nil {
		return fmt.Errorf("invalid auth instance type")
	}

	// Prevent double initialization
	if p.initialized {
		return nil
	}

	// Store auth instance for later use (hooks)
	p.authInst = authInstance

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
	p.rbacService = rbacSvc

	// Get notification adapter from service registry
	if adapter, exists := serviceRegistry.Get("notification.adapter"); exists {
		if typedAdapter, ok := adapter.(*notificationPlugin.Adapter); ok {
			p.notifAdapter = typedAdapter
			p.logger.Debug("retrieved notification adapter from service registry")
		} else {
			p.logger.Warn("notification adapter type assertion failed")
		}
	} else {
		p.logger.Debug("notification adapter not available in service registry (graceful degradation)")
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

	// Apply defaults from p.defaultConfig for any zero values
	// This ensures functional options (WithAllowAppLevelRoles, etc.) are respected
	// even when not explicitly set in config file
	if p.config.MaxOrganizationsPerUser == 0 {
		p.config.MaxOrganizationsPerUser = p.defaultConfig.MaxOrganizationsPerUser
		if p.config.MaxOrganizationsPerUser == 0 {
			p.config.MaxOrganizationsPerUser = 5
		}
	}
	if p.config.MaxMembersPerOrganization == 0 {
		p.config.MaxMembersPerOrganization = p.defaultConfig.MaxMembersPerOrganization
		if p.config.MaxMembersPerOrganization == 0 {
			p.config.MaxMembersPerOrganization = 50
		}
	}
	if p.config.MaxTeamsPerOrganization == 0 {
		p.config.MaxTeamsPerOrganization = p.defaultConfig.MaxTeamsPerOrganization
		if p.config.MaxTeamsPerOrganization == 0 {
			p.config.MaxTeamsPerOrganization = 20
		}
	}
	if p.config.InvitationExpiryHours == 0 {
		p.config.InvitationExpiryHours = p.defaultConfig.InvitationExpiryHours
		if p.config.InvitationExpiryHours == 0 {
			p.config.InvitationExpiryHours = 72 // 3 days
		}
	}

	// For boolean fields, we need special handling since false is a valid value
	// If AllowAppLevelRoles was set via functional option (in defaultConfig), use it
	// unless explicitly overridden in config file
	// Since we can't distinguish between "not set" and "false" in config file,
	// we prioritize the defaultConfig value set by functional options
	if p.defaultConfig.AllowAppLevelRoles {
		p.config.AllowAppLevelRoles = true
	}

	// Create repositories
	orgRepo := orgrepo.NewOrganizationRepository(p.db)
	memberRepo := orgrepo.NewOrganizationMemberRepository(p.db)
	teamRepo := orgrepo.NewOrganizationTeamRepository(p.db)
	invitationRepo := orgrepo.NewOrganizationInvitationRepository(p.db)

	// Get role repository for RBAC role validation
	roleRepo := authInstance.Repository().Role()

	// Create organization service config
	orgConfig := Config{
		MaxOrganizationsPerUser:   p.config.MaxOrganizationsPerUser,
		MaxMembersPerOrganization: p.config.MaxMembersPerOrganization,
		MaxTeamsPerOrganization:   p.config.MaxTeamsPerOrganization,
		EnableUserCreation:        p.config.EnableUserCreation,
		RequireInvitation:         p.config.RequireInvitation,
		InvitationExpiryHours:     p.config.InvitationExpiryHours,
		EnforceUniqueSlug:         p.config.EnforceUniqueSlug,
		AllowAppLevelRoles:        p.config.AllowAppLevelRoles,
	}

	// Log configuration for debugging
	p.logger.Debug("initializing organization service",
		forge.F("allow_app_level_roles", orgConfig.AllowAppLevelRoles),
		forge.F("max_orgs_per_user", orgConfig.MaxOrganizationsPerUser),
		forge.F("max_members_per_org", orgConfig.MaxMembersPerOrganization),
	)

	// Create services with actual repositories and RBAC service
	p.orgService = NewService(
		orgRepo,
		memberRepo,
		teamRepo,
		invitationRepo,
		orgConfig,
		rbacSvc,
		roleRepo,
	)

	// Set hook registry on organization service
	hookRegistry := authInstance.GetHookRegistry()
	if hookRegistry != nil {
		p.orgService.SetHookRegistry(hookRegistry)
	}

	// Create handlers
	p.orgHandler = &OrganizationHandler{
		orgService: p.orgService,
		plugin:     p, // Pass plugin reference for notifications
	}

	// Dashboard extension is lazy-initialized when first accessed via DashboardExtension()

	// Initialize organization UI registry
	p.uiRegistry = NewOrganizationUIRegistry()

	// Discover and register organization UI extensions from other plugins
	// This happens after plugin initialization, so we need to defer actual registration
	// until all plugins are loaded. We'll register extensions in a second pass.
	if pluginRegistry := authInstance.GetPluginRegistry(); pluginRegistry != nil {
		plugins := pluginRegistry.List()
		p.logger.Debug("scanning for organization UI extensions", forge.F("plugin_count", len(plugins)))

		for _, plugin := range plugins {
			// Skip self
			if plugin.ID() == p.ID() {
				continue
			}

			// Check if plugin implements OrganizationUIExtension
			if orgUIExt, ok := plugin.(ui.OrganizationUIExtension); ok {
				if err := p.uiRegistry.Register(orgUIExt); err != nil {
					p.logger.Warn("failed to register organization UI extension",
						forge.F("plugin", plugin.ID()),
						forge.F("error", err.Error()))
				} else {
					p.logger.Debug("registered organization UI extension",
						forge.F("plugin", plugin.ID()),
						forge.F("extension_id", orgUIExt.ExtensionID()))
				}
			}
		}
	}

	// Register services in DI container if available
	if container := forgeApp.Container(); container != nil {
		if err := p.RegisterServices(container); err != nil {
			p.logger.Warn("failed to register organization services in DI container", forge.F("error", err.Error()))
		}
	}

	// Mark as initialized
	p.initialized = true

	p.logger.Info("organization plugin initialized",
		forge.F("max_orgs_per_user", p.config.MaxOrganizationsPerUser),
		forge.F("max_members_per_org", p.config.MaxMembersPerOrganization),
		forge.F("enforce_unique_slug", p.config.EnforceUniqueSlug))

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
			forge.WithResponseSchema(201, "Organization created", organization.Organization{}),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithTags("Organizations"),
			forge.WithValidation(true),
		)

		orgGroup.GET("", p.orgHandler.ListOrganizations,
			forge.WithName("organization.list"),
			forge.WithSummary("List user organizations"),
			forge.WithDescription("List all organizations the current user is a member of"),
			forge.WithResponseSchema(200, "Organizations retrieved", organization.Organization{}),
			forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
			forge.WithTags("Organizations"),
		)

		orgGroup.GET("/:id", p.orgHandler.GetOrganization,
			forge.WithName("organization.get"),
			forge.WithSummary("Get organization"),
			forge.WithDescription("Retrieve a specific organization by ID"),
			forge.WithResponseSchema(200, "Organization retrieved", organization.Organization{}),
			forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
			forge.WithTags("Organizations"),
		)

		orgGroup.GET("/slug/:slug", p.orgHandler.GetOrganizationBySlug,
			forge.WithName("organization.get_by_slug"),
			forge.WithSummary("Get organization by slug"),
			forge.WithDescription("Retrieve a specific organization by its slug"),
			forge.WithResponseSchema(200, "Organization retrieved", organization.Organization{}),
			forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
			forge.WithTags("Organizations"),
		)

		orgGroup.PATCH("/:id", p.orgHandler.UpdateOrganization,
			forge.WithName("organization.update"),
			forge.WithSummary("Update organization"),
			forge.WithDescription("Update organization details (name, logo, metadata)"),
			forge.WithResponseSchema(200, "Organization updated", organization.Organization{}),
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
				forge.WithResponseSchema(200, "Members retrieved", map[string]interface{}{}),
				forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
				forge.WithTags("Organizations", "Members"),
			)

			memberGroup.POST("/invite", p.orgHandler.InviteMember,
				forge.WithName("organization.members.invite"),
				forge.WithSummary("Invite member to organization"),
				forge.WithDescription("Send an invitation to a user to join the organization"),
				forge.WithResponseSchema(201, "Invitation sent", map[string]interface{}{}),
				forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
				forge.WithTags("Organizations", "Members"),
				forge.WithValidation(true),
			)

			memberGroup.PATCH("/:memberId", p.orgHandler.UpdateMember,
				forge.WithName("organization.members.update"),
				forge.WithSummary("Update member"),
				forge.WithDescription("Update member role or status within the organization"),
				forge.WithResponseSchema(200, "Member updated", map[string]interface{}{}),
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
				forge.WithResponseSchema(200, "Teams retrieved", map[string]interface{}{}),
				forge.WithResponseSchema(404, "Organization not found", ErrorResponse{}),
				forge.WithTags("Organizations", "Teams"),
			)

			teamGroup.POST("", p.orgHandler.CreateTeam,
				forge.WithName("organization.teams.create"),
				forge.WithSummary("Create team"),
				forge.WithDescription("Create a new team within the organization"),
				forge.WithResponseSchema(201, "Team created", map[string]interface{}{}),
				forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
				forge.WithTags("Organizations", "Teams"),
				forge.WithValidation(true),
			)

			teamGroup.PATCH("/:teamId", p.orgHandler.UpdateTeam,
				forge.WithName("organization.teams.update"),
				forge.WithSummary("Update team"),
				forge.WithDescription("Update team details (name, description, etc.)"),
				forge.WithResponseSchema(200, "Team updated", map[string]interface{}{}),
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
			forge.WithResponseSchema(200, "Invitation accepted", map[string]interface{}{}),
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
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if hookRegistry == nil || p.notifAdapter == nil {
		return nil
	}

	adapter, ok := p.notifAdapter.(*notificationPlugin.Adapter)
	if !ok {
		p.logger.Warn("notification adapter type assertion failed in RegisterHooks")
		return nil
	}

	// Create wrapper for invitation with notification
	// We'll wrap the InviteMember call in the plugin to send notifications
	p.logger.Debug("organization plugin will handle invitation notifications via service wrapper")

	// Hook: After member added - send notification
	hookRegistry.RegisterAfterMemberAdd(func(ctx context.Context, memberInterface interface{}) error {
		member, ok := memberInterface.(*organization.Member)
		if !ok {
			p.logger.Warn("invalid member type in AfterMemberAdd hook")
			return nil
		}

		// Get app context
		appID, ok := contexts.GetAppID(ctx)
		if !ok || appID.IsNil() {
			p.logger.Warn("app context not available in after member add hook")
			return nil
		}

		// Get organization details
		org, err := p.orgService.FindOrganizationByID(ctx, member.OrganizationID)
		if err != nil {
			p.logger.Error("failed to get organization in member add hook", forge.F("error", err.Error()))
			return nil
		}

		// Get user details for the new member
		userSvc := p.authInst.GetServiceRegistry().UserService()
		if userSvc == nil {
			p.logger.Warn("user service not available")
			return nil
		}
		newMember, err := userSvc.FindByID(ctx, member.UserID)
		if err != nil || newMember == nil {
			p.logger.Error("failed to get user details in member add hook", forge.F("error", err.Error()))
			return nil
		}

		// Send notification to the new member
		userName := newMember.Name
		if userName == "" {
			userName = newMember.Email
		}

		err = adapter.SendOrgMemberAdded(ctx, appID, newMember.Email, userName, userName, org.Name, member.Role)
		if err != nil {
			p.logger.Error("failed to send member added notification",
				forge.F("error", err.Error()),
				forge.F("org_id", org.ID.String()))
		}

		return nil
	})

	// Hook: After member removed - send notification
	hookRegistry.RegisterAfterMemberRemove(func(ctx context.Context, orgID xid.ID, userID xid.ID, memberName string) error {
		// Get app context
		appID, ok := contexts.GetAppID(ctx)
		if !ok || appID.IsNil() {
			p.logger.Warn("app context not available in after member remove hook")
			return nil
		}

		// Get organization details
		org, err := p.orgService.FindOrganizationByID(ctx, orgID)
		if err != nil {
			p.logger.Error("failed to get organization in member remove hook", forge.F("error", err.Error()))
			return nil
		}

		// Get user details
		userSvc := p.authInst.GetServiceRegistry().UserService()
		if userSvc == nil {
			p.logger.Warn("user service not available")
			return nil
		}
		removedUser, err := userSvc.FindByID(ctx, userID)
		if err != nil || removedUser == nil {
			p.logger.Error("failed to get user details in member remove hook", forge.F("error", err.Error()))
			return nil
		}

		// Send notification to the removed member
		userName := removedUser.Name
		if userName == "" {
			userName = removedUser.Email
		}

		timestamp := time.Now().Format(time.RFC3339)
		err = adapter.SendOrgMemberRemoved(ctx, appID, removedUser.Email, userName, memberName, org.Name, timestamp)
		if err != nil {
			p.logger.Error("failed to send member removed notification",
				forge.F("error", err.Error()),
				forge.F("org_id", org.ID.String()))
		}

		return nil
	})

	// Hook: After member role changed - send notification
	hookRegistry.RegisterAfterMemberRoleChange(func(ctx context.Context, orgID xid.ID, userID xid.ID, oldRole string, newRole string) error {
		// Get app context
		appID, ok := contexts.GetAppID(ctx)
		if !ok || appID.IsNil() {
			p.logger.Warn("app context not available in after member role change hook")
			return nil
		}

		// Get organization details
		org, err := p.orgService.FindOrganizationByID(ctx, orgID)
		if err != nil {
			p.logger.Error("failed to get organization in role change hook", forge.F("error", err.Error()))
			return nil
		}

		// Get user details
		userSvc := p.authInst.GetServiceRegistry().UserService()
		if userSvc == nil {
			p.logger.Warn("user service not available")
			return nil
		}
		member, err := userSvc.FindByID(ctx, userID)
		if err != nil || member == nil {
			p.logger.Error("failed to get user details in role change hook", forge.F("error", err.Error()))
			return nil
		}

		// Send notification to the member
		userName := member.Name
		if userName == "" {
			userName = member.Email
		}

		err = adapter.SendOrgRoleChanged(ctx, appID, member.Email, userName, org.Name, oldRole, newRole)
		if err != nil {
			p.logger.Error("failed to send role changed notification",
				forge.F("error", err.Error()),
				forge.F("org_id", org.ID.String()))
		}

		return nil
	})

	// Hook: After organization deleted - send notifications to all members
	hookRegistry.RegisterAfterOrganizationDelete(func(ctx context.Context, orgID xid.ID, orgName string) error {
		// Get app context
		appID, ok := contexts.GetAppID(ctx)
		if !ok || appID.IsNil() {
			p.logger.Warn("app context not available in after org delete hook")
			return nil
		}

		// Get all members (before deletion, they should be in context or passed)
		// For now, we'll log a warning that this needs member list passed in context
		p.logger.Warn("organization deleted notification needs member list - implement in service layer")

		// TODO: The service layer should pass member list in context before deleting org
		// For now, just return nil
		return nil
	})

	p.logger.Debug("registered organization notification hooks")
	return nil
}

// SendInvitationNotification sends an org.invite notification when an invitation is created
// This should be called by handlers after creating an invitation
func (p *Plugin) SendInvitationNotification(ctx context.Context, invitation *organization.Invitation, inviter *user.User, org *organization.Organization) error {
	if p.notifAdapter == nil {
		return nil
	}

	adapter, ok := p.notifAdapter.(*notificationPlugin.Adapter)
	if !ok {
		return nil
	}

	// Get app context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		p.logger.Warn("app context not available for invitation notification")
		return nil
	}

	// Build invite URL (this should be configurable)
	inviteURL := fmt.Sprintf("/invite/%s", invitation.Token)

	// Calculate expiry duration
	expiresIn := fmt.Sprintf("%d hours", p.config.InvitationExpiryHours)

	inviterName := inviter.Name
	if inviterName == "" {
		inviterName = inviter.Email
	}

	// Send invitation email
	err := adapter.SendOrgInvite(
		ctx,
		appID,
		invitation.Email,
		invitation.Email, // userName for recipient (they may not be a user yet)
		inviterName,
		org.Name,
		invitation.Role,
		inviteURL,
		expiresIn,
	)

	if err != nil {
		p.logger.Error("failed to send invitation notification",
			forge.F("error", err.Error()),
			forge.F("org_id", org.ID.String()))
		return err
	}

	return nil
}

// RegisterServiceDecorators registers service decorators
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	services.SetOrganizationService(p.orgService)

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

// DashboardExtension returns the dashboard extension interface implementation
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	p.dashboardExtensionOnce.Do(func() {
		p.dashboardExtension = NewDashboardExtension(p)
	})
	return p.dashboardExtension
}

// RegisterRoles implements the PluginWithRoles interface
// This registers organization-related permissions for platform roles
func (p *Plugin) RegisterRoles(reg interface{}) error {
	roleRegistry, ok := reg.(*rbac.RoleRegistry)
	if !ok {
		return fmt.Errorf("invalid role registry type")
	}

	// Extend Owner role with full organization management permissions
	if err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        rbac.RoleOwner,
		DisplayName: "Owner",
		Description: rbac.RoleDescOwner,
		IsPlatform:  rbac.RoleIsPlatformOwner,
		Priority:    rbac.RolePriorityOwner,
		Permissions: []string{
			// Full organization management
			"* on organizations",
			"* on organization.*",
			// Members management
			"create on members",
			"view on members",
			"update on members",
			"delete on members",
			"invite on members",
			// Teams management
			"create on teams",
			"view on teams",
			"update on teams",
			"delete on teams",
			// Invitations management
			"create on invitations",
			"view on invitations",
			"cancel on invitations",
			// Roles management
			"view on roles",
			"manage on roles",
		},
	}); err != nil {
		return fmt.Errorf("failed to register owner organization permissions: %w", err)
	}

	// Extend Admin role with organization management permissions (except delete org)
	if err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:         rbac.RoleAdmin,
		DisplayName:  "Administrator",
		Description:  rbac.RoleDescAdmin,
		IsPlatform:   rbac.RoleIsPlatformAdmin,
		InheritsFrom: rbac.RoleMember,
		Priority:     rbac.RolePriorityAdmin,
		Permissions: []string{
			// Organization view/update (not delete)
			"view on organizations",
			"update on organizations",
			// Members management
			"create on members",
			"view on members",
			"update on members",
			"delete on members",
			"invite on members",
			// Teams management
			"create on teams",
			"view on teams",
			"update on teams",
			"delete on teams",
			// Invitations management
			"create on invitations",
			"view on invitations",
			"cancel on invitations",
			// Roles view
			"view on roles",
		},
	}); err != nil {
		return fmt.Errorf("failed to register admin organization permissions: %w", err)
	}

	// Extend Member role with basic organization access
	if err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        rbac.RoleMember,
		DisplayName: "Member",
		Description: rbac.RoleDescMember,
		IsPlatform:  rbac.RoleIsPlatformMember,
		Priority:    rbac.RolePriorityMember,
		Permissions: []string{
			// Basic organization access
			"view on organizations",
			// View members and teams
			"view on members",
			"view on teams",
			// View roles
			"view on roles",
		},
	}); err != nil {
		return fmt.Errorf("failed to register member organization permissions: %w", err)
	}

	return nil
}

// GetOrganizationUIRegistry returns the UI registry for accessing registered extensions
// This is used by the dashboard extension to render extension widgets, tabs, and actions
func (p *Plugin) GetOrganizationUIRegistry() *OrganizationUIRegistry {
	return p.uiRegistry
}

// DTOs for organization routes - use shared responses from core
type ErrorResponse = responses.ErrorResponse
type StatusResponse = responses.StatusResponse

// OrganizationsListResponse represents a list of organizations
type OrganizationsListResponse []schema.Organization

// MembersListResponse represents a list of members
type MembersListResponse []schema.OrganizationMember

// TeamsListResponse represents a list of teams
type TeamsListResponse []schema.OrganizationTeam
