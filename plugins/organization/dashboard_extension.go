package organization

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	coreorg "github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements the ui.DashboardExtension interface
// This allows the organization plugin to add its own screens to the dashboard
type DashboardExtension struct {
	plugin   *Plugin
	registry *dashboard.ExtensionRegistry
	basePath string
}

// NewDashboardExtension creates a new dashboard extension for organization
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{plugin: plugin}
}

// SetRegistry sets the extension registry reference (called by dashboard after registration)
func (e *DashboardExtension) SetRegistry(registry *dashboard.ExtensionRegistry) {
	e.registry = registry
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "organization"
}

// NavigationItems returns navigation items to register
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:    "organizations",
			Label: "Organizations",
			Icon: lucide.Building2(
				Class("size-4"),
			),
			Position: ui.NavPositionMain,
			Order:    45, // Between Users (30) and Sessions (40)
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp != nil {
					return basePath + "/dashboard/app/" + currentApp.ID.String() + "/organizations"
				}
				return basePath + "/dashboard/"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "organizations"
			},
			RequiresPlugin: "organization",
		},
	}
}

// Routes returns routes to register under /dashboard/app/:appId/
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Organization List
		{
			Method:       "GET",
			Path:         "/organizations",
			Handler:      e.ServeOrganizationsListPage,
			Name:         "dashboard.organizations.list",
			Summary:      "Organizations list",
			Description:  "View and manage user organizations in the app",
			Tags:         []string{"Dashboard", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Organization Page
		{
			Method:       "GET",
			Path:         "/organizations/create",
			Handler:      e.ServeCreateOrganizationPage,
			Name:         "dashboard.organizations.create",
			Summary:      "Create organization",
			Description:  "Create a new user organization",
			Tags:         []string{"Dashboard", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Create Organization Action
		{
			Method:       "POST",
			Path:         "/organizations/create",
			Handler:      e.CreateOrganization,
			Name:         "dashboard.organizations.create.submit",
			Summary:      "Submit create organization",
			Description:  "Process organization creation form",
			Tags:         []string{"Dashboard", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// =====================================================================
		// IMPORTANT: More specific routes with sub-paths MUST come BEFORE
		// the generic :orgId route to ensure proper route matching
		// =====================================================================

		// Members List (must be before /organizations/:orgId)
		{
			Method:       "GET",
			Path:         "/organizations/:orgId/members",
			Handler:      e.ServeOrganizationMembersPage,
			Name:         "dashboard.organizations.members",
			Summary:      "Organization members",
			Description:  "View and manage organization members",
			Tags:         []string{"Dashboard", "Organizations", "Members"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Invite Member
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/members/invite",
			Handler:      e.InviteMember,
			Name:         "dashboard.organizations.members.invite",
			Summary:      "Invite member",
			Description:  "Invite a user to join the organization",
			Tags:         []string{"Dashboard", "Organizations", "Members"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Update Member Role
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/members/:memberId/update-role",
			Handler:      e.UpdateMemberRole,
			Name:         "dashboard.organizations.members.update-role",
			Summary:      "Update member role",
			Description:  "Change a member's role in the organization",
			Tags:         []string{"Dashboard", "Organizations", "Members"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Remove Member
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/members/:memberId/remove",
			Handler:      e.RemoveMember,
			Name:         "dashboard.organizations.members.remove",
			Summary:      "Remove member",
			Description:  "Remove a member from the organization",
			Tags:         []string{"Dashboard", "Organizations", "Members"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Teams List (must be before /organizations/:orgId)
		{
			Method:       "GET",
			Path:         "/organizations/:orgId/teams",
			Handler:      e.ServeOrganizationTeamsPage,
			Name:         "dashboard.organizations.teams",
			Summary:      "Organization teams",
			Description:  "View and manage organization teams",
			Tags:         []string{"Dashboard", "Organizations", "Teams"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Create Team
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/teams/create",
			Handler:      e.CreateTeam,
			Name:         "dashboard.organizations.teams.create",
			Summary:      "Create team",
			Description:  "Create a new team in the organization",
			Tags:         []string{"Dashboard", "Organizations", "Teams"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Update Team
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/teams/:teamId/update",
			Handler:      e.UpdateTeam,
			Name:         "dashboard.organizations.teams.update",
			Summary:      "Update team",
			Description:  "Update team details",
			Tags:         []string{"Dashboard", "Organizations", "Teams"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Delete Team
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/teams/:teamId/delete",
			Handler:      e.DeleteTeam,
			Name:         "dashboard.organizations.teams.delete",
			Summary:      "Delete team",
			Description:  "Delete a team from the organization",
			Tags:         []string{"Dashboard", "Organizations", "Teams"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Roles List (must be before /organizations/:orgId)
		{
			Method:       "GET",
			Path:         "/organizations/:orgId/roles",
			Handler:      e.ServeOrganizationRolesPage,
			Name:         "dashboard.organizations.roles",
			Summary:      "Organization roles",
			Description:  "View and manage organization roles",
			Tags:         []string{"Dashboard", "Organizations", "Roles"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Invitations List (must be before /organizations/:orgId)
		{
			Method:       "GET",
			Path:         "/organizations/:orgId/invitations",
			Handler:      e.ServeOrganizationInvitationsPage,
			Name:         "dashboard.organizations.invitations",
			Summary:      "Organization invitations",
			Description:  "View and manage pending invitations",
			Tags:         []string{"Dashboard", "Organizations", "Invitations"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Cancel Invitation
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/invitations/:inviteId/cancel",
			Handler:      e.CancelInvitation,
			Name:         "dashboard.organizations.invitations.cancel",
			Summary:      "Cancel invitation",
			Description:  "Cancel a pending invitation",
			Tags:         []string{"Dashboard", "Organizations", "Invitations"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Update Organization (must be before /organizations/:orgId)
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/update",
			Handler:      e.UpdateOrganization,
			Name:         "dashboard.organizations.update",
			Summary:      "Update organization",
			Description:  "Update organization details",
			Tags:         []string{"Dashboard", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Delete Organization (must be before /organizations/:orgId)
		{
			Method:       "POST",
			Path:         "/organizations/:orgId/delete",
			Handler:      e.DeleteOrganization,
			Name:         "dashboard.organizations.delete",
			Summary:      "Delete organization",
			Description:  "Delete an organization (owner only)",
			Tags:         []string{"Dashboard", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Organization Tab Content (extension tabs)
		{
			Method:       "GET",
			Path:         "/organizations/:orgId/tabs/:tabId",
			Handler:      e.ServeOrganizationTabContent,
			Name:         "dashboard.organizations.tab",
			Summary:      "Organization tab content",
			Description:  "View organization extension tab content",
			Tags:         []string{"Dashboard", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Organization Detail (generic :orgId route - must come AFTER all specific routes)
		{
			Method:       "GET",
			Path:         "/organizations/:orgId",
			Handler:      e.ServeOrganizationDetailPage,
			Name:         "dashboard.organizations.detail",
			Summary:      "Organization detail",
			Description:  "View organization overview and details",
			Tags:         []string{"Dashboard", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: false,
		},
		// Plugin Settings
		{
			Method:       "POST",
			Path:         "/organizations/plugin-settings",
			Handler:      e.SavePluginSettings,
			Name:         "dashboard.organizations.settings",
			Summary:      "Save plugin settings",
			Description:  "Update organization plugin configuration",
			Tags:         []string{"Dashboard", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Settings Pages - Role Templates
		{
			Method:       "GET",
			Path:         "/settings/roles",
			Handler:      e.ServeRoleTemplatesSettings,
			Name:         "dashboard.settings.role-templates",
			Summary:      "Role templates settings",
			Description:  "Manage role templates for organizations",
			Tags:         []string{"Dashboard", "Settings", "Roles"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/roles/create",
			Handler:      e.ServeCreateRoleTemplate,
			Name:         "dashboard.settings.role-templates.create",
			Summary:      "Create role template",
			Description:  "Create a new role template",
			Tags:         []string{"Dashboard", "Settings", "Roles"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/roles/create",
			Handler:      e.CreateRoleTemplate,
			Name:         "dashboard.settings.role-templates.create.submit",
			Summary:      "Submit create role template",
			Description:  "Process role template creation",
			Tags:         []string{"Dashboard", "Settings", "Roles"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/roles/create/add-permission",
			Handler:      e.AddCustomPermission,
			Name:         "dashboard.settings.role-templates.add-permission",
			Summary:      "Add custom permission",
			Description:  "Create a new custom permission",
			Tags:         []string{"Dashboard", "Settings", "Permissions"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/roles/:roleId/edit",
			Handler:      e.ServeEditRoleTemplate,
			Name:         "dashboard.settings.role-templates.edit",
			Summary:      "Edit role template",
			Description:  "Edit an existing role template",
			Tags:         []string{"Dashboard", "Settings", "Roles"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/roles/:roleId/edit",
			Handler:      e.UpdateRoleTemplate,
			Name:         "dashboard.settings.role-templates.edit.submit",
			Summary:      "Submit update role template",
			Description:  "Process role template update",
			Tags:         []string{"Dashboard", "Settings", "Roles"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/roles/:roleId/edit/add-permission",
			Handler:      e.AddCustomPermission,
			Name:         "dashboard.settings.role-templates.edit.add-permission",
			Summary:      "Add custom permission (edit)",
			Description:  "Create a new custom permission while editing",
			Tags:         []string{"Dashboard", "Settings", "Permissions"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/roles/:roleId/delete",
			Handler:      e.DeleteRoleTemplate,
			Name:         "dashboard.settings.role-templates.delete",
			Summary:      "Delete role template",
			Description:  "Delete a role template",
			Tags:         []string{"Dashboard", "Settings", "Roles"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Settings Pages - Organization Settings
		{
			Method:       "GET",
			Path:         "/settings/organizations",
			Handler:      e.ServeOrganizationSettings,
			Name:         "dashboard.settings.organizations",
			Summary:      "Organization settings",
			Description:  "Configure organization behavior and limits",
			Tags:         []string{"Dashboard", "Settings", "Organizations"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections to add to the settings page
// Deprecated: Use SettingsPages() instead
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return []ui.SettingsSection{
		{
			ID:          "organization-settings",
			Title:       "Organization Plugin Configuration",
			Description: "Configure user organization behavior and limits",
			Icon: lucide.Building2(
				Class("size-5"),
			),
			Order: 55,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderSettingsSection(basePath, currentApp)
			},
		},
	}
}

// SettingsPages returns full settings pages for the new sidebar layout
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "role-templates",
			Label:         "Role Templates",
			Description:   "Manage role templates for new organizations",
			Icon:          lucide.Shield(Class("h-5 w-5")),
			Category:      "security",
			Order:         10,
			Path:          "roles",
			RequirePlugin: "organization",
			RequireAdmin:  true,
		},
		{
			ID:            "organization-settings",
			Label:         "Organizations",
			Description:   "Configure organization behavior and limits",
			Icon:          lucide.Building2(Class("h-5 w-5")),
			Category:      "general",
			Order:         15,
			Path:          "organizations",
			RequirePlugin: "organization",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns widgets to show on the main dashboard
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "organizations-stats",
			Title: "Organizations",
			Icon: lucide.Building2(
				Class("size-5"),
			),
			Order: 25,
			Size:  1, // 1 column
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderDashboardWidget(basePath, currentApp)
			},
		},
	}
}

// ServeOrganizationsListPage renders the organizations list page
func (e *DashboardExtension) ServeOrganizationsListPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	// Build minimal PageData
	pageData := components.PageData{
		Title:      "Organizations",
		User:       currentUser,
		ActivePage: "organizations",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	// Render page content
	content := e.renderOrganizationsListContent(c, currentApp, currentUser, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// renderOrganizationsListContent renders the main content for organizations list
func (e *DashboardExtension) renderOrganizationsListContent(c forge.Context, currentApp *app.App, currentUser *user.User, basePath string) g.Node {
	ctx := c.Request().Context()

	// Get current environment
	envID, err := e.extractEnvironmentFromURL(c, currentApp.ID)
	if err != nil {
		// Log error but continue with empty list
	}

	// Fetch organizations for the current app and environment
	filter := &coreorg.ListOrganizationsFilter{
		AppID:         currentApp.ID,
		EnvironmentID: envID,
		PaginationParams: pagination.PaginationParams{
			Limit: 100,
		},
	}

	orgsResp, err := e.plugin.orgService.ListOrganizations(ctx, filter)

	var orgs []*coreorg.Organization
	var totalOrgs int64
	if err == nil && orgsResp != nil {
		orgs = orgsResp.Data
		if orgsResp.Pagination != nil {
			totalOrgs = orgsResp.Pagination.Total
		}
	}

	// Calculate stats
	totalMembers := int64(0)
	totalTeams := int64(0)
	for _, org := range orgs {
		// Count members for each org
		membersResp, _ := e.plugin.orgService.ListMembers(ctx, &coreorg.ListMembersFilter{
			OrganizationID:   org.ID,
			PaginationParams: pagination.PaginationParams{Limit: 1},
		})
		if membersResp != nil && membersResp.Pagination != nil {
			totalMembers += membersResp.Pagination.Total
		}

		// Count teams for each org
		teamsResp, _ := e.plugin.orgService.ListTeams(ctx, &coreorg.ListTeamsFilter{
			OrganizationID:   org.ID,
			PaginationParams: pagination.PaginationParams{Limit: 1},
		})
		if teamsResp != nil && teamsResp.Pagination != nil {
			totalTeams += teamsResp.Pagination.Total
		}
	}

	return Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Organizations")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage user organizations and their members")),
			),
			A(
				Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/create"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 focus:outline-none focus:ring-2 focus:ring-violet-500"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Organization"),
			),
		),

		// Stats cards
		Div(
			Class("grid gap-6 md:grid-cols-3"),
			e.statsCard("Total Organizations", fmt.Sprintf("%d", totalOrgs), lucide.Building2(Class("size-5"))),
			e.statsCard("Total Members", fmt.Sprintf("%d", totalMembers), lucide.Users(Class("size-5"))),
			e.statsCard("Total Teams", fmt.Sprintf("%d", totalTeams), lucide.UsersRound(Class("size-5"))),
		),

		// Organizations table
		e.renderOrganizationsTable(ctx, orgs, currentApp, currentUser, basePath),
	)
}

// ServeCreateOrganizationPage renders the create organization page
func (e *DashboardExtension) ServeCreateOrganizationPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	pageData := components.PageData{
		Title:      "Create Organization",
		User:       currentUser,
		ActivePage: "organizations",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderCreateOrganizationForm(currentApp, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// renderCreateOrganizationForm renders the organization creation form
func (e *DashboardExtension) renderCreateOrganizationForm(currentApp *app.App, basePath string) g.Node {
	return Div(
		Class("space-y-6"),

		// Back button and header
		Div(
			A(
				Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations"),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white mb-4"),
				lucide.ArrowLeft(Class("size-4")),
				g.Text("Back to Organizations"),
			),
			H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
				g.Text("Create Organization")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Create a new organization workspace for your team")),
		),

		// Form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/create"),
				Class("space-y-6"),

				// Name field
				Div(
					Label(
						For("name"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Organization Name"),
					),
					Input(
						Type("text"),
						Name("name"),
						ID("name"),
						Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("Acme Corporation"),
					),
					P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
						g.Text("The display name for your organization")),
				),

				// Slug field
				Div(
					Label(
						For("slug"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("URL Slug"),
					),
					Input(
						Type("text"),
						Name("slug"),
						ID("slug"),
						Required(),
						Pattern("[a-z0-9-]+"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("acme"),
					),
					P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Lowercase letters, numbers, and hyphens only")),
				),

				// Logo URL field (optional)
				Div(
					Label(
						For("logo"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Logo URL (Optional)"),
					),
					Input(
						Type("url"),
						Name("logo"),
						ID("logo"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("https://example.com/logo.png"),
					),
				),

				// Submit buttons
				Div(
					Class("flex justify-end gap-3"),
					A(
						Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations"),
						Class("rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 focus:outline-none focus:ring-2 focus:ring-violet-500"),
						g.Text("Create Organization"),
					),
				),
			),
		),
	)
}

// ServeOrganizationDetailPage renders the organization detail page
func (e *DashboardExtension) ServeOrganizationDetailPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	// Get organization from URL
	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	pageData := components.PageData{
		Title:      org.Name,
		User:       currentUser,
		ActivePage: "organizations",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderOrganizationDetailContent(c, org, currentApp, currentUser, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// renderOrganizationDetailContent renders the organization detail content
func (e *DashboardExtension) renderOrganizationDetailContent(c forge.Context, org *coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string) g.Node {
	ctx := c.Request().Context()

	// Get user's role in this organization
	userRole := e.getUserRole(ctx, org.ID, currentUser.ID)
	isAdmin := e.isUserAdmin(ctx, org.ID, currentUser.ID)

	// Get member count
	membersResp, _ := e.plugin.orgService.ListMembers(ctx, &coreorg.ListMembersFilter{
		OrganizationID:   org.ID,
		PaginationParams: pagination.PaginationParams{Limit: 1},
	})
	memberCount := int64(0)
	if membersResp != nil && membersResp.Pagination != nil {
		memberCount = membersResp.Pagination.Total
	}

	// Get team count
	teamsResp, _ := e.plugin.orgService.ListTeams(ctx, &coreorg.ListTeamsFilter{
		OrganizationID:   org.ID,
		PaginationParams: pagination.PaginationParams{Limit: 1},
	})
	teamCount := int64(0)
	if teamsResp != nil && teamsResp.Pagination != nil {
		teamCount = teamsResp.Pagination.Total
	}

	// Create extension context
	extCtx := ui.OrgExtensionContext{
		OrgID:    org.ID,
		AppID:    currentApp.ID,
		BasePath: basePath,
		Request:  c.Request(),
		GetOrg: func() (interface{}, error) {
			return org, nil
		},
		IsAdmin: isAdmin,
	}

	// Get extension registry
	registry := e.plugin.GetOrganizationUIRegistry()
	var extensionWidgets []ui.OrganizationWidget
	var extensionActions []ui.OrganizationAction
	var extensionQuickLinks []ui.OrganizationQuickLink
	var tabs []ui.OrganizationTab

	if registry != nil {
		extensionWidgets = registry.GetWidgets(extCtx)
		extensionActions = registry.GetActions(extCtx)
		extensionQuickLinks = registry.GetQuickLinks(extCtx)
		tabs = registry.GetTabs(extCtx)
	}

	baseURL := fmt.Sprintf("%s/dashboard/app/%s/organizations/%s", basePath, currentApp.ID.String(), org.ID.String())

	// Build default quick links
	defaultQuickLinks := []ui.OrganizationQuickLink{
		{
			ID:          "members",
			Title:       "Members",
			Description: fmt.Sprintf("Manage %d members", memberCount),
			Icon:        lucide.Users(Class("size-6")),
			Order:       10,
			URLBuilder: func(bp string, orgID, appID xid.ID) string {
				return fmt.Sprintf("%s/dashboard/app/%s/organizations/%s/members", bp, appID.String(), orgID.String())
			},
		},
		{
			ID:          "teams",
			Title:       "Teams",
			Description: fmt.Sprintf("Manage %d teams", teamCount),
			Icon:        lucide.UsersRound(Class("size-6")),
			Order:       20,
			URLBuilder: func(bp string, orgID, appID xid.ID) string {
				return fmt.Sprintf("%s/dashboard/app/%s/organizations/%s/teams", bp, appID.String(), orgID.String())
			},
		},
		{
			ID:          "roles",
			Title:       "Roles",
			Description: "Manage roles & permissions",
			Icon:        lucide.ShieldCheck(Class("size-6")),
			Order:       30,
			URLBuilder: func(bp string, orgID, appID xid.ID) string {
				return fmt.Sprintf("%s/dashboard/app/%s/organizations/%s/roles", bp, appID.String(), orgID.String())
			},
		},
		{
			ID:          "invitations",
			Title:       "Invitations",
			Description: "View pending invitations",
			Icon:        lucide.Mail(Class("size-6")),
			Order:       40,
			URLBuilder: func(bp string, orgID, appID xid.ID) string {
				return fmt.Sprintf("%s/dashboard/app/%s/organizations/%s/invitations", bp, appID.String(), orgID.String())
			},
		},
	}

	// Merge quick links
	allQuickLinks := MergeQuickLinks(defaultQuickLinks, extensionQuickLinks)

	return Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations"),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Organizations"),
		),

		// Organization header
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between"),
				Div(
					Class("flex items-center gap-4"),
					g.If(org.Logo != "",
						Img(
							Src(org.Logo),
							Alt(org.Name),
							Class("size-16 rounded-lg object-cover"),
						),
					),
					Div(
						H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
							g.Text(org.Name)),
						P(Class("text-sm text-slate-600 dark:text-gray-400"),
							g.Text("@"+org.Slug)),
						Div(
							Class("mt-2"),
							e.renderRoleBadge(userRole),
						),
					),
				),
				Div(
					Class("flex gap-2"),
					// Extension actions
					g.If(len(extensionActions) > 0, ActionButtons(extensionActions)),
					// Delete button (owner only)
					g.If(userRole == "owner",
						g.Group([]g.Node{
							Button(
								Type("button"),
								Class("rounded-lg border border-red-600 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"),
								g.Attr("onclick", "if(confirm('Are you sure you want to delete this organization? This action cannot be undone.')) { document.getElementById('delete-form-"+org.ID.String()+"').submit(); }"),
								g.Text("Delete Organization"),
							),
							Form(
								ID("delete-form-"+org.ID.String()),
								Method("POST"),
								Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/delete"),
								Style("display: none;"),
							),
						}),
					),
				),
			),
		),

		// Tab navigation (if tabs exist)
		g.If(len(tabs) > 0, TabNavigation(tabs, "", baseURL)),

		// Stats cards (default + extensions)
		Div(
			Class("grid gap-6 md:grid-cols-3"),
			e.statsCard("Members", fmt.Sprintf("%d", memberCount), lucide.Users(Class("size-5"))),
			e.statsCard("Teams", fmt.Sprintf("%d", teamCount), lucide.UsersRound(Class("size-5"))),
			e.statsCard("Created", org.CreatedAt.Format("Jan 2, 2006"), lucide.Calendar(Class("size-5"))),
		),

		// Extension widgets
		g.If(len(extensionWidgets) > 0, WidgetGrid(extensionWidgets, extCtx)),

		// Quick links (merged)
		QuickLinkGrid(allQuickLinks, basePath, org.ID.String(), currentApp.ID.String()),
	)
}

// ServeOrganizationTabContent renders extension tab content
func (e *DashboardExtension) ServeOrganizationTabContent(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	// Get organization from URL
	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	// Get tab ID from URL
	tabID := c.Param("tabId")
	if tabID == "" {
		return c.String(http.StatusBadRequest, "Tab ID required")
	}

	// Check if user is admin
	isAdmin := e.isUserAdmin(c.Request().Context(), org.ID, currentUser.ID)

	// Create extension context
	extCtx := ui.OrgExtensionContext{
		OrgID:    org.ID,
		AppID:    currentApp.ID,
		BasePath: basePath,
		Request:  c.Request(),
		GetOrg: func() (interface{}, error) {
			return org, nil
		},
		IsAdmin: isAdmin,
	}

	// Get tab from registry
	registry := e.plugin.GetOrganizationUIRegistry()
	if registry == nil {
		return c.String(http.StatusInternalServerError, "UI registry not available")
	}

	tab := registry.GetTabByPath(extCtx, tabID)
	if tab == nil {
		return c.String(http.StatusNotFound, "Tab not found")
	}

	pageData := components.PageData{
		Title:      org.Name + " - " + tab.Label,
		User:       currentUser,
		ActivePage: "organizations",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	// Render tab content with navigation
	content := e.renderTabContentWithNav(c, org, currentApp, currentUser, basePath, tabID, tab, extCtx)

	return handler.RenderWithLayout(c, pageData, content)
}

// renderTabContentWithNav renders tab content with tab navigation
func (e *DashboardExtension) renderTabContentWithNav(c forge.Context, org *coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string, activeTab string, tab *ui.OrganizationTab, extCtx ui.OrgExtensionContext) g.Node {
	// Get all tabs from registry
	registry := e.plugin.GetOrganizationUIRegistry()
	tabs := []ui.OrganizationTab{}
	if registry != nil {
		tabs = registry.GetTabs(extCtx)
	}

	baseURL := fmt.Sprintf("%s/dashboard/app/%s/organizations/%s", basePath, currentApp.ID.String(), org.ID.String())

	return Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(baseURL),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Organization"),
		),

		// Organization header (minimal)
		Div(
			Class("flex items-center gap-3"),
			g.If(org.Logo != "",
				Img(
					Src(org.Logo),
					Alt(org.Name),
					Class("size-10 rounded-lg object-cover"),
				),
			),
			Div(
				H1(Class("text-xl font-bold text-slate-900 dark:text-white"), g.Text(org.Name)),
				P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("@"+org.Slug)),
			),
		),

		// Tab navigation
		TabNavigation(tabs, activeTab, baseURL),

		// Tab content
		Div(
			Class("mt-6"),
			tab.Renderer(extCtx),
		),
	)
}

// ServeOrganizationMembersPage renders the members management page
func (e *DashboardExtension) ServeOrganizationMembersPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	pageData := components.PageData{
		Title:      org.Name + " - Members",
		User:       currentUser,
		ActivePage: "organizations",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderMembersPageContent(c, org, currentApp, currentUser, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// renderMembersPageContent renders the members management content
func (e *DashboardExtension) renderMembersPageContent(c forge.Context, org *coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string) g.Node {
	ctx := c.Request().Context()

	// Get user's role in the organization
	userRole := e.getUserRole(ctx, org.ID, currentUser.ID)

	// Check management permission using RBAC
	canManage := e.canManageOrganization(ctx, org.ID, currentUser.ID)

	// Fetch members
	membersResp, _ := e.plugin.orgService.ListMembers(ctx, &coreorg.ListMembersFilter{
		OrganizationID:   org.ID,
		PaginationParams: pagination.PaginationParams{Limit: 100},
	})

	members := []*coreorg.Member{}
	if membersResp != nil {
		members = membersResp.Data
	}

	return Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Organization"),
		),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Members")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage members and their roles in "+org.Name)),
			),
			g.If(canManage,
				Button(
					Type("button"),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.Attr("onclick", "document.getElementById('invite-modal').style.display='block'"),
					lucide.UserPlus(Class("size-4")),
					g.Text("Invite Member"),
				),
			),
		),

		// Members table
		e.renderMembersTable(ctx, members, org, currentApp, userRole, canManage, basePath),

		// Invite modal (if can manage)
		g.If(canManage, e.renderInviteMemberModal(org, currentApp, basePath)),
	)
}

// ServeOrganizationTeamsPage renders the teams management page
func (e *DashboardExtension) ServeOrganizationTeamsPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	pageData := components.PageData{
		Title:      org.Name + " - Teams",
		User:       currentUser,
		ActivePage: "organizations",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderTeamsPageContent(c, org, currentApp, currentUser, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// renderTeamsPageContent renders the teams management content
func (e *DashboardExtension) renderTeamsPageContent(c forge.Context, org *coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string) g.Node {
	ctx := c.Request().Context()

	// Get user's role in the organization
	userRole := e.getUserRole(ctx, org.ID, currentUser.ID)

	// Check management permission using RBAC
	canManage := e.canManageOrganization(ctx, org.ID, currentUser.ID)

	// Fetch teams
	teamsResp, _ := e.plugin.orgService.ListTeams(ctx, &coreorg.ListTeamsFilter{
		OrganizationID:   org.ID,
		PaginationParams: pagination.PaginationParams{Limit: 100},
	})

	teams := []*coreorg.Team{}
	if teamsResp != nil {
		teams = teamsResp.Data
	}

	return Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Organization"),
		),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Teams")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage teams in "+org.Name)),
			),
			g.If(canManage,
				Button(
					Type("button"),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.Attr("onclick", "document.getElementById('team-modal').style.display='block'"),
					lucide.Plus(Class("size-4")),
					g.Text("Create Team"),
				),
			),
		),

		// Teams table
		e.renderTeamsTable(ctx, teams, org, currentApp, userRole, canManage, basePath),

		// Create team modal (if can manage)
		g.If(canManage, e.renderCreateTeamModal(org, currentApp, basePath)),
	)
}

// ServeOrganizationRolesPage renders the roles page
func (e *DashboardExtension) ServeOrganizationRolesPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	pageData := components.PageData{
		Title:      org.Name + " - Roles",
		User:       currentUser,
		ActivePage: "organizations",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderRolesPageContent(c, org, currentApp, currentUser, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// renderRolesPageContent renders the roles management content
func (e *DashboardExtension) renderRolesPageContent(c forge.Context, org *coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string) g.Node {
	ctx := c.Request().Context()

	// Check management permission using RBAC
	canManage := e.canManageOrganization(ctx, org.ID, currentUser.ID)

	// Fetch organization-specific roles from RBAC service
	var roles []*schema.Role
	if e.plugin.rbacService != nil {
		roles, _ = e.plugin.rbacService.GetOrgRoles(ctx, org.ID, org.EnvironmentID)
	}

	return Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Organization"),
		),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Roles")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage roles and permissions in "+org.Name)),
			),
		),

		// Roles table
		e.renderRolesTable(ctx, roles, org, currentApp, canManage, basePath),
	)
}

// renderRolesTable renders the roles table
func (e *DashboardExtension) renderRolesTable(ctx context.Context, roles []*schema.Role, org *coreorg.Organization, currentApp *app.App, canManage bool, basePath string) g.Node {
	if len(roles) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-8 text-center dark:border-gray-700 dark:bg-gray-800"),
			lucide.ShieldCheck(Class("mx-auto size-12 text-slate-400 dark:text-gray-500")),
			H3(Class("mt-4 text-lg font-medium text-slate-900 dark:text-white"),
				g.Text("No roles defined")),
			P(Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
				g.Text("This organization doesn't have any custom roles defined yet. Default roles from the platform are used.")),
		)
	}

	return Div(
		Class("overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-800"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-gray-700"),
			THead(
				Class("bg-slate-50 dark:bg-gray-900"),
				Tr(
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
						g.Text("Role")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
						g.Text("Description")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
						g.Text("Type")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
						g.Text("Source")),
				),
			),
			TBody(
				Class("divide-y divide-slate-200 bg-white dark:divide-gray-700 dark:bg-gray-800"),
				g.Group(g.Map(roles, func(role *schema.Role) g.Node {
					return e.renderRoleRow(role, org, currentApp, canManage, basePath)
				})),
			),
		),
	)
}

// renderRoleRow renders a single role row
func (e *DashboardExtension) renderRoleRow(role *schema.Role, org *coreorg.Organization, currentApp *app.App, canManage bool, basePath string) g.Node {
	// Determine role type
	roleType := "Custom"
	roleTypeClass := "bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-300"
	if role.TemplateID != nil {
		roleType = "From Template"
		roleTypeClass = "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300"
	}
	if role.IsOwnerRole {
		roleType = "Owner Role"
		roleTypeClass = "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300"
	}

	// Determine source
	source := "Organization"
	if role.IsTemplate {
		source = "Template"
	} else if role.TemplateID != nil {
		source = "Cloned"
	}

	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("flex items-center"),
				Div(Class("flex-shrink-0 size-10 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center"),
					lucide.Shield(Class("size-5 text-white")),
				),
				Div(Class("ml-4"),
					Div(Class("text-sm font-medium text-slate-900 dark:text-white"),
						g.Text(role.Name)),
				),
			),
		),
		Td(Class("px-6 py-4"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400 max-w-xs truncate"),
				g.Text(role.Description),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Span(
				Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "+roleTypeClass),
				g.Text(roleType),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(source),
			),
		),
	)
}

// ServeOrganizationInvitationsPage renders the invitations page
func (e *DashboardExtension) ServeOrganizationInvitationsPage(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	pageData := components.PageData{
		Title:      org.Name + " - Invitations",
		User:       currentUser,
		ActivePage: "organizations",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	content := e.renderInvitationsPageContent(c, org, currentApp, currentUser, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// renderInvitationsPageContent renders the invitations content
func (e *DashboardExtension) renderInvitationsPageContent(c forge.Context, org *coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string) g.Node {
	ctx := c.Request().Context()

	// Fetch invitations
	invitationsResp, _ := e.plugin.orgService.ListInvitations(ctx, &coreorg.ListInvitationsFilter{
		OrganizationID:   org.ID,
		PaginationParams: pagination.PaginationParams{Limit: 100},
	})

	invitations := []*coreorg.Invitation{}
	if invitationsResp != nil {
		invitations = invitationsResp.Data
	}

	return Div(
		Class("space-y-6"),

		// Back button
		A(
			Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()),
			Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
			lucide.ArrowLeft(Class("size-4")),
			g.Text("Back to Organization"),
		),

		// Header
		H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
			g.Text("Invitations")),
		P(Class("text-slate-600 dark:text-gray-400"),
			g.Text("Pending invitations for "+org.Name)),

		// Invitations table
		e.renderInvitationsTable(ctx, invitations, org, currentApp, basePath),
	)
}

// Action Handlers

// CreateOrganization handles organization creation
func (e *DashboardExtension) CreateOrganization(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	// Get current environment
	envID, err := e.extractEnvironmentFromURL(c, currentApp.ID)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid environment context: "+err.Error())
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form
	name := form.Get("name")
	slug := form.Get("slug")
	logoURL := form.Get("logo")

	if name == "" || slug == "" {
		return c.String(http.StatusBadRequest, "Name and slug are required")
	}

	// Create organization
	req := &coreorg.CreateOrganizationRequest{
		Name: name,
		Slug: slug,
	}
	if logoURL != "" {
		req.Logo = &logoURL
	}

	ctx := c.Request().Context()
	_, err = e.plugin.orgService.CreateOrganization(ctx, req, currentUser.ID, currentApp.ID, envID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create organization: "+err.Error())
	}

	// Redirect to organizations list
	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations")
}

// UpdateOrganization handles organization updates
func (e *DashboardExtension) UpdateOrganization(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	// Check permission (owner or admin)
	ctx := c.Request().Context()
	if !e.checkOrgAdmin(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Insufficient permissions")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form
	name := form.Get("name")
	req := &coreorg.UpdateOrganizationRequest{
		Name: &name,
	}
	if logo := form.Get("logo"); logo != "" {
		req.Logo = &logo
	}

	_, err = e.plugin.orgService.UpdateOrganization(ctx, org.ID, req)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update organization: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String())
}

// DeleteOrganization handles organization deletion
func (e *DashboardExtension) DeleteOrganization(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	// Check permission (owner only)
	ctx := c.Request().Context()
	if !e.checkOrgOwner(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Only owners can delete organizations")
	}

	err = e.plugin.orgService.DeleteOrganization(ctx, org.ID, currentUser.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete organization: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations")
}

// InviteMember handles member invitation
func (e *DashboardExtension) InviteMember(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	// Check permission
	ctx := c.Request().Context()
	if !e.checkOrgAdmin(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Insufficient permissions")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form
	email := form.Get("email")
	role := form.Get("role")

	if email == "" || role == "" {
		return c.String(http.StatusBadRequest, "Email and role are required")
	}

	req := &coreorg.InviteMemberRequest{
		Email: email,
		Role:  role,
	}

	_, err = e.plugin.orgService.InviteMember(ctx, org.ID, req, currentUser.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to invite member: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/members")
}

// UpdateMemberRole handles member role updates
func (e *DashboardExtension) UpdateMemberRole(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	memberIDStr := c.Param("memberId")
	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid member ID")
	}

	// Check permission
	ctx := c.Request().Context()
	if !e.checkOrgAdmin(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Insufficient permissions")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	newRole := c.Request().Form.Get("role")
	if newRole == "" {
		return c.String(http.StatusBadRequest, "Role is required")
	}

	req := &coreorg.UpdateMemberRequest{
		Role: &newRole,
	}

	_, err = e.plugin.orgService.UpdateMember(ctx, memberID, req, currentUser.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update member: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/members")
}

// RemoveMember handles member removal
func (e *DashboardExtension) RemoveMember(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	memberIDStr := c.Param("memberId")
	memberID, err := xid.FromString(memberIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid member ID")
	}

	// Check permission
	ctx := c.Request().Context()
	if !e.checkOrgAdmin(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Insufficient permissions")
	}

	err = e.plugin.orgService.RemoveMember(ctx, memberID, currentUser.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to remove member: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/members")
}

// CreateTeam handles team creation
func (e *DashboardExtension) CreateTeam(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	// Check permission
	ctx := c.Request().Context()
	if !e.checkOrgAdmin(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Insufficient permissions")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form
	name := form.Get("name")
	description := form.Get("description")

	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	req := &coreorg.CreateTeamRequest{
		Name: name,
	}
	if description != "" {
		req.Description = &description
	}

	_, err = e.plugin.orgService.CreateTeam(ctx, org.ID, req, currentUser.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create team: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/teams")
}

// UpdateTeam handles team updates
func (e *DashboardExtension) UpdateTeam(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	teamIDStr := c.Param("teamId")
	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid team ID")
	}

	// Check permission
	ctx := c.Request().Context()
	if !e.checkOrgAdmin(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Insufficient permissions")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form
	name := form.Get("name")
	description := form.Get("description")

	req := &coreorg.UpdateTeamRequest{
		Name: &name,
	}
	if description != "" {
		req.Description = &description
	}

	_, err = e.plugin.orgService.UpdateTeam(ctx, teamID, req, currentUser.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update team: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/teams")
}

// DeleteTeam handles team deletion
func (e *DashboardExtension) DeleteTeam(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	teamIDStr := c.Param("teamId")
	teamID, err := xid.FromString(teamIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid team ID")
	}

	// Check permission
	ctx := c.Request().Context()
	if !e.checkOrgAdmin(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Insufficient permissions")
	}

	err = e.plugin.orgService.DeleteTeam(ctx, teamID, currentUser.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete team: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/teams")
}

// CancelInvitation handles invitation cancellation
func (e *DashboardExtension) CancelInvitation(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	org, err := e.getCurrentOrganization(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid organization")
	}

	inviteIDStr := c.Param("inviteId")
	inviteID, err := xid.FromString(inviteIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid invitation ID")
	}

	// Check permission
	ctx := c.Request().Context()
	if !e.checkOrgAdmin(ctx, org.ID, currentUser.ID) {
		return c.String(http.StatusForbidden, "Insufficient permissions")
	}

	err = e.plugin.orgService.CancelInvitation(ctx, inviteID, currentUser.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to cancel invitation: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/invitations")
}

// ServeRoleTemplatesSettings renders the role templates settings page
func (e *DashboardExtension) ServeRoleTemplatesSettings(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := c.Request().Context()
	content := e.renderRoleTemplatesContent(ctx, currentApp, handler.GetBasePath())

	// Use the settings layout with sidebar navigation
	return handler.RenderSettingsPage(c, "role-templates", content)
}

// ServeCreateRoleTemplate renders the create role template form
func (e *DashboardExtension) ServeCreateRoleTemplate(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	ctx := c.Request().Context()
	content := e.renderCreateRoleTemplateForm(ctx, currentApp, handler.GetBasePath(), nil)

	return handler.RenderSettingsPage(c, "role-templates", content)
}

// ServeEditRoleTemplate renders the edit role template form
func (e *DashboardExtension) ServeEditRoleTemplate(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	roleIDStr := c.Param("roleId")
	roleID, err := xid.FromString(roleIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid role ID")
	}

	ctx := c.Request().Context()
	content := e.renderEditRoleTemplateForm(ctx, currentApp, roleID, handler.GetBasePath(), nil)

	return handler.RenderSettingsPage(c, "role-templates", content)
}

// ServeOrganizationSettings renders the organization settings page
func (e *DashboardExtension) ServeOrganizationSettings(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	content := e.renderOrganizationSettingsContent(currentApp, handler.GetBasePath())

	// Use the settings layout with sidebar navigation
	return handler.RenderSettingsPage(c, "organization-settings", content)
}

// SavePluginSettings handles plugin settings updates
func (e *DashboardExtension) SavePluginSettings(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form

	// Parse numeric fields
	maxOrgsPerUser, _ := strconv.Atoi(form.Get("maxOrganizationsPerUser"))
	if maxOrgsPerUser <= 0 || maxOrgsPerUser > 100 {
		maxOrgsPerUser = 5
	}

	maxMembersPerOrg, _ := strconv.Atoi(form.Get("maxMembersPerOrganization"))
	if maxMembersPerOrg <= 0 || maxMembersPerOrg > 1000 {
		maxMembersPerOrg = 50
	}

	maxTeamsPerOrg, _ := strconv.Atoi(form.Get("maxTeamsPerOrganization"))
	if maxTeamsPerOrg <= 0 || maxTeamsPerOrg > 100 {
		maxTeamsPerOrg = 20
	}

	invitationExpiry, _ := strconv.Atoi(form.Get("invitationExpiryHours"))
	if invitationExpiry <= 0 || invitationExpiry > 720 {
		invitationExpiry = 72
	}

	// Parse checkboxes
	enableUserCreation := form.Get("enableUserCreation") == "true"
	requireInvitation := form.Get("requireInvitation") == "true"

	// Update plugin config (in-memory)
	e.plugin.config.MaxOrganizationsPerUser = maxOrgsPerUser
	e.plugin.config.MaxMembersPerOrganization = maxMembersPerOrg
	e.plugin.config.MaxTeamsPerOrganization = maxTeamsPerOrg
	e.plugin.config.EnableUserCreation = enableUserCreation
	e.plugin.config.RequireInvitation = requireInvitation
	e.plugin.config.InvitationExpiryHours = invitationExpiry

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/settings/organizations?saved=true")
}

// CreateRoleTemplate handles role template creation
func (e *DashboardExtension) CreateRoleTemplate(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	ctx := c.Request().Context()
	form := c.Request().Form

	name := form.Get("name")
	description := form.Get("description")
	isOwnerRole := form.Get("isOwnerRole") == "true"
	permissionIDs := form["permissionIDs[]"]

	// Validate
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Role name is required"
	}
	if len(permissionIDs) == 0 {
		errors["permissions"] = "At least one permission must be selected"
	}

	if len(errors) > 0 {
		handler := e.registry.GetHandler()
		content := e.renderCreateRoleTemplateForm(ctx, currentApp, handler.GetBasePath(), errors)
		return handler.RenderSettingsPage(c, "role-templates", content)
	}

	// Convert permission IDs
	permIDs := make([]xid.ID, 0, len(permissionIDs))
	for _, pidStr := range permissionIDs {
		pid, err := xid.FromString(pidStr)
		if err == nil {
			permIDs = append(permIDs, pid)
		}
	}

	// Get default environment for the app
	var defaultEnvID xid.ID
	err = e.plugin.db.NewSelect().
		Table("environments").
		Column("id").
		Where("app_id = ?", currentApp.ID).
		Where("is_default = ?", true).
		Limit(1).
		Scan(ctx, &defaultEnvID)

	if err != nil {
		// If no default environment, try to get the first one
		err = e.plugin.db.NewSelect().
			Table("environments").
			Column("id").
			Where("app_id = ?", currentApp.ID).
			Order("created_at ASC").
			Limit(1).
			Scan(ctx, &defaultEnvID)

		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to get environment: "+err.Error())
		}
	}

	// Create role template via RBAC service
	// Display name will be auto-generated from name if empty string is passed
	_, err = e.plugin.rbacService.CreateRoleTemplate(ctx, currentApp.ID, defaultEnvID, name, "", description, isOwnerRole, permIDs)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create role template: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/settings/roles?created=true")
}

// UpdateRoleTemplate handles role template updates
func (e *DashboardExtension) UpdateRoleTemplate(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	roleIDStr := c.Param("roleId")
	roleID, err := xid.FromString(roleIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid role ID")
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	ctx := c.Request().Context()
	form := c.Request().Form

	name := form.Get("name")
	description := form.Get("description")
	isOwnerRole := form.Get("isOwnerRole") == "true"
	permissionIDs := form["permissionIDs[]"]

	// Validate
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Role name is required"
	}
	if len(permissionIDs) == 0 {
		errors["permissions"] = "At least one permission must be selected"
	}

	if len(errors) > 0 {
		handler := e.registry.GetHandler()
		content := e.renderEditRoleTemplateForm(ctx, currentApp, roleID, handler.GetBasePath(), errors)
		return handler.RenderSettingsPage(c, "role-templates", content)
	}

	// Convert permission IDs
	permIDs := make([]xid.ID, 0, len(permissionIDs))
	for _, pidStr := range permissionIDs {
		pid, err := xid.FromString(pidStr)
		if err == nil {
			permIDs = append(permIDs, pid)
		}
	}

	// Update role template via RBAC service
	// Display name will be auto-generated from name if empty string is passed
	_, err = e.plugin.rbacService.UpdateRoleTemplate(ctx, roleID, name, "", description, isOwnerRole, permIDs)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update role template: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/settings/roles?updated=true")
}

// DeleteRoleTemplate handles role template deletion
func (e *DashboardExtension) DeleteRoleTemplate(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, e.registry.GetHandler().GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	roleIDStr := c.Param("roleId")
	roleID, err := xid.FromString(roleIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid role ID")
	}

	ctx := c.Request().Context()

	// Delete role template via RBAC service
	if err := e.plugin.rbacService.DeleteRoleTemplate(ctx, roleID); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete role template: "+err.Error())
	}

	basePath := e.registry.GetHandler().GetBasePath()
	return c.Redirect(http.StatusFound, basePath+"/dashboard/app/"+currentApp.ID.String()+"/settings/roles?deleted=true")
}

// AddCustomPermission handles creating custom permissions inline
func (e *DashboardExtension) AddCustomPermission(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid app context"})
	}

	// Parse JSON request
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Validate
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Permission name is required"})
	}

	ctx := c.Request().Context()

	// Create permission
	appID := currentApp.ID
	permission := &schema.Permission{
		ID:          xid.New(),
		AppID:       &appID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		IsCustom:    true,
	}

	if _, err := e.plugin.db.NewInsert().Model(permission).Exec(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create permission"})
	}

	return c.JSON(http.StatusOK, permission)
}

// Helper Methods

// Context helpers
func (e *DashboardExtension) getUserFromContext(c forge.Context) *user.User {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil
	}
	return handler.GetUserFromContext(c)
}

func (e *DashboardExtension) extractAppFromURL(c forge.Context) (*app.App, error) {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "handler not available")
	}

	currentApp, err := handler.GetCurrentApp(c)
	if err != nil {
		return nil, err
	}

	return currentApp, nil
}

func (e *DashboardExtension) extractEnvironmentFromURL(c forge.Context, appID xid.ID) (xid.ID, error) {
	handler := e.registry.GetHandler()
	if handler == nil {
		return xid.NilID(), forge.NewHTTPError(http.StatusInternalServerError, "handler not available")
	}

	currentEnv, err := handler.GetCurrentEnvironment(c, appID)
	if err != nil {
		return xid.NilID(), err
	}

	return currentEnv.ID, nil
}

func (e *DashboardExtension) getCurrentOrganization(c forge.Context) (*coreorg.Organization, error) {
	orgIDStr := c.Param("orgId")
	if orgIDStr == "" {
		return nil, forge.NewHTTPError(http.StatusBadRequest, "organization ID required")
	}

	orgID, err := xid.FromString(orgIDStr)
	if err != nil {
		return nil, forge.NewHTTPError(http.StatusBadRequest, "invalid organization ID")
	}

	ctx := c.Request().Context()
	org, err := e.plugin.orgService.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return nil, forge.NewHTTPError(http.StatusNotFound, "organization not found")
	}

	return org, nil
}

// Permission helpers
func (e *DashboardExtension) checkOrgAccess(ctx context.Context, orgID, userID xid.ID) bool {
	isMember, _ := e.plugin.orgService.IsMember(ctx, orgID, userID)
	return isMember
}

func (e *DashboardExtension) checkOrgAdmin(ctx context.Context, orgID, userID xid.ID) bool {
	isAdmin, _ := e.plugin.orgService.IsAdmin(ctx, orgID, userID)
	isOwner, _ := e.plugin.orgService.IsOwner(ctx, orgID, userID)
	return isAdmin || isOwner
}

func (e *DashboardExtension) checkOrgOwner(ctx context.Context, orgID, userID xid.ID) bool {
	isOwner, _ := e.plugin.orgService.IsOwner(ctx, orgID, userID)
	return isOwner
}

func (e *DashboardExtension) getUserRole(ctx context.Context, orgID, userID xid.ID) string {
	member, err := e.plugin.orgService.FindMember(ctx, orgID, userID)
	if err != nil || member == nil {
		return ""
	}
	return member.Role
}

// isUserAdmin checks if the user has admin privileges in the organization
func (e *DashboardExtension) isUserAdmin(ctx context.Context, orgID, userID xid.ID) bool {
	role := e.getUserRole(ctx, orgID, userID)
	return role == "owner" || role == "admin"
}

// canManageOrganization checks if a user can manage an organization using RBAC
// It checks:
// 1. App-level role check (owner/admin/superadmin can manage all orgs in their app)
// 2. Dynamic RBAC permission (create on members)
// 3. Organization-level role check (owner/admin)
func (e *DashboardExtension) canManageOrganization(ctx context.Context, orgID, userID xid.ID) bool {
	// First, check if user is an app owner/admin/superadmin
	// App owners should be able to manage all organizations in their app
	if e.isAppAdmin(ctx, userID) {
		return true
	}

	// Try RBAC permission check for "create on members" (management permission)
	hasPermission, err := e.plugin.orgService.CheckPermission(ctx, orgID, userID, "create", "members")
	if err == nil && hasPermission {
		return true
	}

	// Fallback to organization-level role check
	userRole := e.getUserRole(ctx, orgID, userID)
	if userRole == "owner" || userRole == "admin" {
		return true
	}

	return false
}

// isAppAdmin checks if a user has owner/admin/superadmin role at the app level
// This allows app admins to manage all organizations within their app
func (e *DashboardExtension) isAppAdmin(ctx context.Context, userID xid.ID) bool {
	handler := e.registry.GetHandler()
	if handler == nil {
		return false
	}

	// Get the current app from context
	db := e.plugin.db
	if db == nil {
		return false
	}

	// Query user's app-level role from user_roles table
	var userRoles []struct {
		RoleName string `bun:"role__name"`
	}

	err := db.NewSelect().
		TableExpr("user_roles").
		ColumnExpr("roles.name AS role__name").
		Join("LEFT JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Where("user_roles.deleted_at IS NULL").
		Scan(ctx, &userRoles)

	if err != nil || len(userRoles) == 0 {
		return false
	}

	// Check if user has any admin-level role
	for _, ur := range userRoles {
		roleName := strings.ToLower(ur.RoleName)
		if roleName == "owner" || roleName == "admin" || roleName == "superadmin" {
			return true
		}
	}

	return false
}

// Settings rendering
func (e *DashboardExtension) RenderSettingsSection(basePath string, currentApp *app.App) g.Node {
	cfg := e.plugin.config

	return Form(
		Method("POST"),
		Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/plugin-settings"),
		Class("space-y-6"),

		// Max organizations per user
		Div(
			Label(
				For("maxOrganizationsPerUser"),
				Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Text("Max Organizations Per User"),
			),
			Input(
				Type("number"),
				Name("maxOrganizationsPerUser"),
				ID("maxOrganizationsPerUser"),
				Value(strconv.Itoa(cfg.MaxOrganizationsPerUser)),
				g.Attr("min", "1"),
				g.Attr("max", "100"),
				Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
				g.Text("Maximum number of organizations a user can create")),
		),

		// Max members per organization
		Div(
			Label(
				For("maxMembersPerOrganization"),
				Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Text("Max Members Per Organization"),
			),
			Input(
				Type("number"),
				Name("maxMembersPerOrganization"),
				ID("maxMembersPerOrganization"),
				Value(strconv.Itoa(cfg.MaxMembersPerOrganization)),
				g.Attr("min", "1"),
				g.Attr("max", "1000"),
				Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
				g.Text("Maximum number of members per organization")),
		),

		// Max teams per organization
		Div(
			Label(
				For("maxTeamsPerOrganization"),
				Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Text("Max Teams Per Organization"),
			),
			Input(
				Type("number"),
				Name("maxTeamsPerOrganization"),
				ID("maxTeamsPerOrganization"),
				Value(strconv.Itoa(cfg.MaxTeamsPerOrganization)),
				g.Attr("min", "1"),
				g.Attr("max", "100"),
				Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
				g.Text("Maximum number of teams per organization")),
		),

		// Invitation expiry hours
		Div(
			Label(
				For("invitationExpiryHours"),
				Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Text("Invitation Expiry (Hours)"),
			),
			Input(
				Type("number"),
				Name("invitationExpiryHours"),
				ID("invitationExpiryHours"),
				Value(strconv.Itoa(cfg.InvitationExpiryHours)),
				g.Attr("min", "1"),
				g.Attr("max", "720"),
				Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
				g.Text("How long invitations remain valid")),
		),

		// Enable user creation
		Div(
			Label(
				Class("flex items-center space-x-3"),
				Input(
					Type("checkbox"),
					Name("enableUserCreation"),
					ID("enableUserCreation"),
					Value("true"),
					g.If(cfg.EnableUserCreation, Checked()),
					Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700"),
				),
				Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Enable User Creation")),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400 ml-6"),
				g.Text("Allow users to create their own organizations")),
		),

		// Require invitation
		Div(
			Label(
				Class("flex items-center space-x-3"),
				Input(
					Type("checkbox"),
					Name("requireInvitation"),
					ID("requireInvitation"),
					Value("true"),
					g.If(cfg.RequireInvitation, Checked()),
					Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700"),
				),
				Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Require Invitation")),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400 ml-6"),
				g.Text("Require invitation to join organizations")),
		),

		// Save button
		Div(
			Class("flex justify-end"),
			Button(
				Type("submit"),
				Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2"),
				g.Text("Save Settings"),
			),
		),
	)
}

// Dashboard widget rendering
func (e *DashboardExtension) RenderDashboardWidget(basePath string, currentApp *app.App) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),

		// Header
		Div(
			Class("flex items-center justify-between mb-4"),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text("Organizations")),
			lucide.Building2(
				Class("size-5 text-violet-600 dark:text-violet-400"),
			),
		),

		// Stats
		Div(
			Class("space-y-3"),
			Div(
				Div(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("—")),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("User organizations")),
			),
			P(Class("text-xs text-slate-500 dark:text-gray-500"),
				g.Text("View detailed stats on the organizations page")),
		),

		// View more link
		Div(
			Class("mt-4 pt-4 border-t border-slate-200 dark:border-gray-800"),
			A(
				Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations"),
				Class("text-sm font-medium text-violet-600 hover:text-violet-700 dark:text-violet-400 dark:hover:text-violet-300"),
				g.Text("View all organizations →"),
			),
		),
	)
}

// Component rendering helpers

func (e *DashboardExtension) statsCard(label, value string, icon g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				P(Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
					g.Text(label)),
				Div(
					Class("mt-2 text-2xl font-semibold text-slate-900 dark:text-white"),
					g.Text(value),
				),
			),
			Div(
				Class("text-violet-600 dark:text-violet-400"),
				icon,
			),
		),
	)
}

func (e *DashboardExtension) quickLinkCard(title, description, href string, icon g.Node) g.Node {
	return A(
		Href(href),
		Class("block rounded-lg border border-slate-200 bg-white p-6 shadow-sm hover:border-violet-300 hover:shadow-md transition dark:border-gray-800 dark:bg-gray-900 dark:hover:border-violet-700"),
		Div(
			Class("flex items-start gap-4"),
			Div(
				Class("text-violet-600 dark:text-violet-400"),
				icon,
			),
			Div(
				H3(Class("font-semibold text-slate-900 dark:text-white"),
					g.Text(title)),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text(description)),
			),
		),
	)
}

func (e *DashboardExtension) renderRoleBadge(role string) g.Node {
	var classes string
	var badgeIcon g.Node

	switch role {
	case "owner":
		classes = "inline-flex items-center gap-1 rounded-full bg-violet-100 px-2.5 py-0.5 text-xs font-semibold text-violet-800 dark:bg-violet-900/30 dark:text-violet-300"
		badgeIcon = lucide.Crown(Class("size-3"))
	case "admin":
		classes = "inline-flex items-center gap-1 rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-semibold text-blue-800 dark:bg-blue-900/30 dark:text-blue-300"
		badgeIcon = lucide.Shield(Class("size-3"))
	case "member":
		classes = "inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-800 dark:bg-gray-800 dark:text-gray-300"
		badgeIcon = lucide.User(Class("size-3"))
	default:
		classes = "inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-800 dark:bg-gray-800 dark:text-gray-300"
		badgeIcon = nil
	}

	return Span(
		Class(classes),
		badgeIcon,
		g.Text(strings.ToUpper(role[:1])+role[1:]),
	)
}

func (e *DashboardExtension) renderStatusBadge(status string) g.Node {
	var classes string

	switch status {
	case "active":
		classes = "inline-flex rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-semibold text-green-800 dark:bg-green-900/30 dark:text-green-300"
	case "pending":
		classes = "inline-flex rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs font-semibold text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300"
	case "suspended", "declined", "cancelled":
		classes = "inline-flex rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-semibold text-red-800 dark:bg-red-900/30 dark:text-red-300"
	case "expired":
		classes = "inline-flex rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-800 dark:bg-gray-800 dark:text-gray-300"
	default:
		classes = "inline-flex rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-800 dark:bg-gray-800 dark:text-gray-300"
	}

	return Span(
		Class(classes),
		g.Text(strings.ToUpper(status[:1])+status[1:]),
	)
}

// Table rendering methods

func (e *DashboardExtension) renderOrganizationsTable(ctx context.Context, orgs []*coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string) g.Node {
	if len(orgs) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 shadow-sm dark:border-gray-800 dark:bg-gray-900 text-center"),
			lucide.Building2(
				Class("mx-auto size-12 text-slate-400 dark:text-gray-600 mb-4"),
			),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"),
				g.Text("No Organizations Yet")),
			P(Class("text-slate-600 dark:text-gray-400 mb-4"),
				g.Text("Get started by creating your first organization")),
			A(
				Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/create"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Organization"),
			),
		)
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900 overflow-hidden"),
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("w-full"),
				THead(
					Class("bg-slate-50 dark:bg-gray-800/50"),
					Tr(
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Organization")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Slug")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Members")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Teams")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Your Role")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Created")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Actions")),
					),
				),
				TBody(
					Class("bg-white dark:bg-gray-900 divide-y divide-slate-200 dark:divide-gray-800"),
					g.Group(e.renderOrganizationRows(ctx, orgs, currentApp, currentUser, basePath)),
				),
			),
		),
	)
}

func (e *DashboardExtension) renderOrganizationRows(ctx context.Context, orgs []*coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string) []g.Node {
	rows := make([]g.Node, 0, len(orgs))
	for _, org := range orgs {
		rows = append(rows, e.renderOrganizationRow(ctx, org, currentApp, currentUser, basePath))
	}
	return rows
}

func (e *DashboardExtension) renderOrganizationRow(ctx context.Context, org *coreorg.Organization, currentApp *app.App, currentUser *user.User, basePath string) g.Node {
	// Get member count
	membersResp, _ := e.plugin.orgService.ListMembers(ctx, &coreorg.ListMembersFilter{
		OrganizationID:   org.ID,
		PaginationParams: pagination.PaginationParams{Limit: 1},
	})
	memberCount := 0
	if membersResp != nil && membersResp.Pagination != nil {
		memberCount = int(membersResp.Pagination.Total)
	}

	// Get team count
	teamsResp, _ := e.plugin.orgService.ListTeams(ctx, &coreorg.ListTeamsFilter{
		OrganizationID:   org.ID,
		PaginationParams: pagination.PaginationParams{Limit: 1},
	})
	teamCount := 0
	if teamsResp != nil && teamsResp.Pagination != nil {
		teamCount = int(teamsResp.Pagination.Total)
	}

	// Get user's role
	userRole := e.getUserRole(ctx, org.ID, currentUser.ID)

	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("flex items-center gap-3"),
				g.If(org.Logo != "",
					Img(
						Src(org.Logo),
						Alt(org.Name),
						Class("size-8 rounded object-cover"),
					),
				),
				Div(
					Class("text-sm font-medium text-slate-900 dark:text-white"),
					g.Text(org.Name),
				),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text("@"+org.Slug),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-900 dark:text-white"),
				g.Text(fmt.Sprintf("%d", memberCount)),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-900 dark:text-white"),
				g.Text(fmt.Sprintf("%d", teamCount)),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			e.renderRoleBadge(userRole),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(org.CreatedAt.Format("Jan 2, 2006")),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap text-right text-sm font-medium"),
			A(
				Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()),
				Class("text-violet-600 hover:text-violet-900 dark:text-violet-400 dark:hover:text-violet-300"),
				g.Text("View"),
			),
		),
	)
}

func (e *DashboardExtension) renderMembersTable(ctx context.Context, members []*coreorg.Member, org *coreorg.Organization, currentApp *app.App, userRole string, canManage bool, basePath string) g.Node {
	if len(members) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 shadow-sm dark:border-gray-800 dark:bg-gray-900 text-center"),
			lucide.Users(
				Class("mx-auto size-12 text-slate-400 dark:text-gray-600 mb-4"),
			),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"),
				g.Text("No Members Yet")),
			P(Class("text-slate-600 dark:text-gray-400"),
				g.Text("Invite members to collaborate in this organization")),
		)
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900 overflow-hidden"),
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("w-full"),
				THead(
					Class("bg-slate-50 dark:bg-gray-800/50"),
					Tr(
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("User")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Role")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Status")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Joined")),
						g.If(canManage,
							Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
								g.Text("Actions")),
						),
					),
				),
				TBody(
					Class("bg-white dark:bg-gray-900 divide-y divide-slate-200 dark:divide-gray-800"),
					g.Group(e.renderMemberRows(members, org, currentApp, userRole, canManage, basePath)),
				),
			),
		),
	)
}

func (e *DashboardExtension) renderMemberRows(members []*coreorg.Member, org *coreorg.Organization, currentApp *app.App, userRole string, canManage bool, basePath string) []g.Node {
	rows := make([]g.Node, 0, len(members))
	for _, member := range members {
		rows = append(rows, e.renderMemberRow(member, org, currentApp, userRole, canManage, basePath))
	}
	return rows
}

func (e *DashboardExtension) renderMemberRow(member *coreorg.Member, org *coreorg.Organization, currentApp *app.App, userRole string, canManage bool, basePath string) g.Node {
	// Can't manage owner
	canModify := canManage && member.Role != "owner"

	// Get user display info
	userName := "Unknown User"
	userEmail := ""
	userImage := ""
	if member.User != nil {
		if member.User.Name != "" {
			userName = member.User.Name
		} else if member.User.Email != "" {
			userName = member.User.Email
		}
		userEmail = member.User.Email
		userImage = member.User.Image
	}

	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("flex items-center gap-3"),
				// Avatar
				g.If(userImage != "",
					Img(
						Src(userImage),
						Alt(userName),
						Class("size-10 rounded-full object-cover"),
					),
				),
				g.If(userImage == "",
					Div(
						Class("size-10 rounded-full bg-violet-100 dark:bg-violet-900/30 flex items-center justify-center"),
						Span(
							Class("text-sm font-medium text-violet-600 dark:text-violet-400"),
							g.Text(string([]rune(userName)[0:1])),
						),
					),
				),
				Div(
					Div(
						Class("text-sm font-medium text-slate-900 dark:text-white"),
						g.Text(userName),
					),
					g.If(userEmail != "" && userEmail != userName,
						Div(
							Class("text-xs text-slate-500 dark:text-gray-400"),
							g.Text(userEmail),
						),
					),
				),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			e.renderRoleBadge(member.Role),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			e.renderStatusBadge(member.Status),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(member.CreatedAt.Format("Jan 2, 2006")),
			),
		),
		g.If(canManage,
			Td(Class("px-6 py-4 whitespace-nowrap text-right text-sm font-medium"),
				g.If(canModify,
					Div(
						Class("flex items-center justify-end gap-2"),
						Form(
							Method("POST"),
							Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/members/"+member.ID.String()+"/remove"),
							g.Attr("onsubmit", "return confirm('Are you sure you want to remove this member?')"),
							Class("inline"),
							Button(
								Type("submit"),
								Class("text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"),
								g.Text("Remove"),
							),
						),
					),
				),
			),
		),
	)
}

func (e *DashboardExtension) renderTeamsTable(ctx context.Context, teams []*coreorg.Team, org *coreorg.Organization, currentApp *app.App, userRole string, canManage bool, basePath string) g.Node {
	if len(teams) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 shadow-sm dark:border-gray-800 dark:bg-gray-900 text-center"),
			lucide.UsersRound(
				Class("mx-auto size-12 text-slate-400 dark:text-gray-600 mb-4"),
			),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"),
				g.Text("No Teams Yet")),
			P(Class("text-slate-600 dark:text-gray-400"),
				g.Text("Create teams to organize members")),
		)
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900 overflow-hidden"),
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("w-full"),
				THead(
					Class("bg-slate-50 dark:bg-gray-800/50"),
					Tr(
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Name")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Description")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Members")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Created")),
						g.If(canManage,
							Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
								g.Text("Actions")),
						),
					),
				),
				TBody(
					Class("bg-white dark:bg-gray-900 divide-y divide-slate-200 dark:divide-gray-800"),
					g.Group(e.renderTeamRows(ctx, teams, org, currentApp, canManage, basePath)),
				),
			),
		),
	)
}

func (e *DashboardExtension) renderTeamRows(ctx context.Context, teams []*coreorg.Team, org *coreorg.Organization, currentApp *app.App, canManage bool, basePath string) []g.Node {
	rows := make([]g.Node, 0, len(teams))
	for _, team := range teams {
		rows = append(rows, e.renderTeamRow(ctx, team, org, currentApp, canManage, basePath))
	}
	return rows
}

func (e *DashboardExtension) renderTeamRow(ctx context.Context, team *coreorg.Team, org *coreorg.Organization, currentApp *app.App, canManage bool, basePath string) g.Node {
	// Get member count for team
	memberCount := 0

	description := team.Description

	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm font-medium text-slate-900 dark:text-white"),
				g.Text(team.Name),
			),
		),
		Td(Class("px-6 py-4"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400 max-w-xs truncate"),
				g.Text(description),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-900 dark:text-white"),
				g.Text(fmt.Sprintf("%d", memberCount)),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(team.CreatedAt.Format("Jan 2, 2006")),
			),
		),
		g.If(canManage,
			Td(Class("px-6 py-4 whitespace-nowrap text-right text-sm font-medium"),
				Form(
					Method("POST"),
					Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/teams/"+team.ID.String()+"/delete"),
					g.Attr("onsubmit", "return confirm('Are you sure you want to delete this team?')"),
					Class("inline"),
					Button(
						Type("submit"),
						Class("text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"),
						g.Text("Delete"),
					),
				),
			),
		),
	)
}

func (e *DashboardExtension) renderInvitationsTable(ctx context.Context, invitations []*coreorg.Invitation, org *coreorg.Organization, currentApp *app.App, basePath string) g.Node {
	if len(invitations) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-12 shadow-sm dark:border-gray-800 dark:bg-gray-900 text-center"),
			lucide.Mail(
				Class("mx-auto size-12 text-slate-400 dark:text-gray-600 mb-4"),
			),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"),
				g.Text("No Pending Invitations")),
			P(Class("text-slate-600 dark:text-gray-400"),
				g.Text("All invitations have been processed")),
		)
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900 overflow-hidden"),
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("w-full"),
				THead(
					Class("bg-slate-50 dark:bg-gray-800/50"),
					Tr(
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Email")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Role")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Status")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Expires")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Actions")),
					),
				),
				TBody(
					Class("bg-white dark:bg-gray-900 divide-y divide-slate-200 dark:divide-gray-800"),
					g.Group(e.renderInvitationRows(invitations, org, currentApp, basePath)),
				),
			),
		),
	)
}

func (e *DashboardExtension) renderInvitationRows(invitations []*coreorg.Invitation, org *coreorg.Organization, currentApp *app.App, basePath string) []g.Node {
	rows := make([]g.Node, 0, len(invitations))
	for _, invite := range invitations {
		rows = append(rows, e.renderInvitationRow(invite, org, currentApp, basePath))
	}
	return rows
}

func (e *DashboardExtension) renderInvitationRow(invite *coreorg.Invitation, org *coreorg.Organization, currentApp *app.App, basePath string) g.Node {
	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm font-medium text-slate-900 dark:text-white"),
				g.Text(invite.Email),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			e.renderRoleBadge(invite.Role),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			e.renderStatusBadge(invite.Status),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(invite.ExpiresAt.Format("Jan 2, 2006")),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap text-right text-sm font-medium"),
			g.If(invite.Status == "pending",
				Form(
					Method("POST"),
					Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/invitations/"+invite.ID.String()+"/cancel"),
					g.Attr("onsubmit", "return confirm('Are you sure you want to cancel this invitation?')"),
					Class("inline"),
					Button(
						Type("submit"),
						Class("text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"),
						g.Text("Cancel"),
					),
				),
			),
		),
	)
}

// Modal rendering methods

func (e *DashboardExtension) renderInviteMemberModal(org *coreorg.Organization, currentApp *app.App, basePath string) g.Node {
	return Div(
		ID("invite-modal"),
		Class("fixed inset-0 z-50 hidden items-center justify-center bg-black bg-opacity-50"),
		g.Attr("onclick", "if(event.target === this) this.style.display='none'"),
		Div(
			Class("bg-white dark:bg-gray-900 rounded-lg p-6 max-w-md w-full mx-4"),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Invite Member")),
			Form(
				Method("POST"),
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/members/invite"),
				Class("space-y-4"),
				Div(
					Label(
						For("invite-email"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Email Address"),
					),
					Input(
						Type("email"),
						Name("email"),
						ID("invite-email"),
						Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("user@example.com"),
					),
				),
				Div(
					Label(
						For("invite-role"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Role"),
					),
					Select(
						Name("role"),
						ID("invite-role"),
						Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value("member"), g.Text("Member")),
						Option(Value("admin"), g.Text("Admin")),
					),
				),
				Div(
					Class("flex justify-end gap-3"),
					Button(
						Type("button"),
						Class("rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"),
						g.Attr("onclick", "document.getElementById('invite-modal').style.display='none'"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Text("Send Invitation"),
					),
				),
			),
		),
	)
}

func (e *DashboardExtension) renderCreateTeamModal(org *coreorg.Organization, currentApp *app.App, basePath string) g.Node {
	return Div(
		ID("team-modal"),
		Class("fixed inset-0 z-50 hidden items-center justify-center bg-black bg-opacity-50"),
		g.Attr("onclick", "if(event.target === this) this.style.display='none'"),
		Div(
			Class("bg-white dark:bg-gray-900 rounded-lg p-6 max-w-md w-full mx-4"),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Create Team")),
			Form(
				Method("POST"),
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/"+org.ID.String()+"/teams/create"),
				Class("space-y-4"),
				Div(
					Label(
						For("team-name"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Team Name"),
					),
					Input(
						Type("text"),
						Name("name"),
						ID("team-name"),
						Required(),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("Engineering Team"),
					),
				),
				Div(
					Label(
						For("team-description"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Description (Optional)"),
					),
					Textarea(
						Name("description"),
						ID("team-description"),
						Rows("3"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Placeholder("Describe the team's purpose..."),
					),
				),
				Div(
					Class("flex justify-end gap-3"),
					Button(
						Type("button"),
						Class("rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"),
						g.Attr("onclick", "document.getElementById('team-modal').style.display='none'"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Text("Create Team"),
					),
				),
			),
		),
	)
}

// Settings content rendering

func (e *DashboardExtension) renderRoleTemplatesContent(ctx context.Context, currentApp *app.App, basePath string) g.Node {
	// Fetch role templates
	var roleTemplates []*schema.Role
	if e.plugin.rbacService == nil {
		roleTemplates = []*schema.Role{}
	} else {
		// Get default environment for the app
		var defaultEnvID xid.ID
		err := e.plugin.db.NewSelect().
			Table("environments").
			Column("id").
			Where("app_id = ?", currentApp.ID).
			Where("is_default = ?", true).
			Limit(1).
			Scan(ctx, &defaultEnvID)

		if err != nil {
			// If no default environment, try to get the first one
			err = e.plugin.db.NewSelect().
				Table("environments").
				Column("id").
				Where("app_id = ?", currentApp.ID).
				Order("created_at ASC").
				Limit(1).
				Scan(ctx, &defaultEnvID)
		}

		if err != nil {
			roleTemplates = []*schema.Role{}
		} else {
			roleTemplates, err = e.plugin.rbacService.GetRoleTemplates(ctx, currentApp.ID, defaultEnvID)
			if err != nil {
				roleTemplates = []*schema.Role{}
			} else {
			}
		}
	}

	return ui.RoleManagementInterface(ui.RoleManagementInterfaceData{
		Title:         "Role Templates",
		Description:   "Manage role templates that can be used when creating new organizations. Each template can have custom permissions and access levels.",
		Roles:         roleTemplates,
		IsTemplate:    true,
		BasePath:      basePath + "/dashboard/app/" + currentApp.ID.String(),
		CreateRoleURL: basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/roles/create",
		AppID:         currentApp.ID,
		OrgID:         nil,
		ShowActions:   true,
	})
}

func (e *DashboardExtension) renderCreateRoleTemplateForm(ctx context.Context, currentApp *app.App, basePath string, errors map[string]string) g.Node {
	// Fetch all available permissions
	permissions, err := e.plugin.rbacService.GetAppPermissions(ctx, currentApp.ID)
	if err != nil {
		permissions = []*schema.Permission{}
	}

	if errors == nil {
		errors = make(map[string]string)
	}

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			A(
				Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/settings/roles"),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white mb-4"),
				lucide.ArrowLeft(Class("size-4")),
				g.Text("Back to Role Templates"),
			),
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Create Role Template")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Define a new role template that can be used when creating organizations")),
		),

		// Form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/settings/roles/create"),
				ui.RoleForm(ui.RoleFormData{
					Role:            nil,
					Permissions:     permissions,
					SelectedPermIDs: make(map[xid.ID]bool),
					IsTemplate:      true,
					CanSetOwnerRole: true,
					Errors:          errors,
					ActionURL:       basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/roles/create",
					CancelURL:       basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/roles",
				}),
			),
		),
	)
}

func (e *DashboardExtension) renderEditRoleTemplateForm(ctx context.Context, currentApp *app.App, roleID xid.ID, basePath string, errors map[string]string) g.Node {
	// Fetch the role template with permissions via RBAC service
	roleWithPerms, err := e.plugin.rbacService.GetRoleTemplateWithPermissions(ctx, roleID)
	if err != nil {
		return Div(
			Class("text-red-600"),
			g.Text("Error: Role template not found"),
		)
	}

	// Fetch all available permissions
	permissions, err := e.plugin.rbacService.GetAppPermissions(ctx, currentApp.ID)
	if err != nil {
		permissions = []*schema.Permission{}
	}

	// Build selected permission IDs map from role's current permissions
	selectedPermIDs := make(map[xid.ID]bool)
	for _, perm := range roleWithPerms.Permissions {
		selectedPermIDs[perm.ID] = true
	}

	if errors == nil {
		errors = make(map[string]string)
	}

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			A(
				Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/settings/roles"),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white mb-4"),
				lucide.ArrowLeft(Class("size-4")),
				g.Text("Back to Role Templates"),
			),
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Edit Role Template")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Modify the role template settings and permissions")),
		),

		// Form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/settings/roles/"+roleID.String()+"/edit"),
				ui.RoleForm(ui.RoleFormData{
					Role:            roleWithPerms.Role,
					Permissions:     permissions,
					SelectedPermIDs: selectedPermIDs,
					IsTemplate:      true,
					CanSetOwnerRole: true,
					Errors:          errors,
					ActionURL:       basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/roles/" + roleID.String() + "/edit",
					CancelURL:       basePath + "/dashboard/app/" + currentApp.ID.String() + "/settings/roles",
				}),
			),
		),
	)
}

func (e *DashboardExtension) renderOrganizationSettingsContent(currentApp *app.App, basePath string) g.Node {
	cfg := e.plugin.config

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Organization Settings")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Configure how organizations work in your application")),
		),

		// Settings form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(basePath+"/dashboard/app/"+currentApp.ID.String()+"/organizations/plugin-settings"),
				Class("space-y-6"),

				// Limits section
				Div(
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
						g.Text("Resource Limits")),

					// Max organizations per user
					Div(
						Label(
							For("maxOrganizationsPerUser"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Max Organizations Per User"),
						),
						Input(
							Type("number"),
							Name("maxOrganizationsPerUser"),
							ID("maxOrganizationsPerUser"),
							Value(strconv.Itoa(cfg.MaxOrganizationsPerUser)),
							g.Attr("min", "1"),
							g.Attr("max", "100"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Maximum number of organizations a user can create")),
					),

					// Max members per organization
					Div(
						Class("mt-4"),
						Label(
							For("maxMembersPerOrganization"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Max Members Per Organization"),
						),
						Input(
							Type("number"),
							Name("maxMembersPerOrganization"),
							ID("maxMembersPerOrganization"),
							Value(strconv.Itoa(cfg.MaxMembersPerOrganization)),
							g.Attr("min", "1"),
							g.Attr("max", "1000"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Maximum number of members per organization")),
					),

					// Max teams per organization
					Div(
						Class("mt-4"),
						Label(
							For("maxTeamsPerOrganization"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Max Teams Per Organization"),
						),
						Input(
							Type("number"),
							Name("maxTeamsPerOrganization"),
							ID("maxTeamsPerOrganization"),
							Value(strconv.Itoa(cfg.MaxTeamsPerOrganization)),
							g.Attr("min", "1"),
							g.Attr("max", "100"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Maximum number of teams per organization")),
					),
				),

				// Invitation settings section
				Div(
					Class("mt-8 pt-6 border-t border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
						g.Text("Invitation Settings")),

					// Invitation expiry hours
					Div(
						Label(
							For("invitationExpiryHours"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Invitation Expiry (Hours)"),
						),
						Input(
							Type("number"),
							Name("invitationExpiryHours"),
							ID("invitationExpiryHours"),
							Value(strconv.Itoa(cfg.InvitationExpiryHours)),
							g.Attr("min", "1"),
							g.Attr("max", "720"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
							g.Text("How long invitations remain valid")),
					),
				),

				// Behavior settings section
				Div(
					Class("mt-8 pt-6 border-t border-slate-200 dark:border-gray-800"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
						g.Text("Behavior Settings")),

					// Enable user creation
					Div(
						Label(
							Class("flex items-center space-x-3"),
							Input(
								Type("checkbox"),
								Name("enableUserCreation"),
								ID("enableUserCreation"),
								Value("true"),
								g.If(cfg.EnableUserCreation, Checked()),
								Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700"),
							),
							Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Enable User Creation")),
						),
						P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400 ml-6"),
							g.Text("Allow users to create their own organizations")),
					),

					// Require invitation
					Div(
						Class("mt-4"),
						Label(
							Class("flex items-center space-x-3"),
							Input(
								Type("checkbox"),
								Name("requireInvitation"),
								ID("requireInvitation"),
								Value("true"),
								g.If(cfg.RequireInvitation, Checked()),
								Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700"),
							),
							Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Require Invitation")),
						),
						P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400 ml-6"),
							g.Text("Require invitation to join organizations")),
					),
				),

				// Save button
				Div(
					Class("flex justify-end pt-6"),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2"),
						g.Text("Save Settings"),
					),
				),
			),
		),
	)
}
