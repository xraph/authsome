package scim

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// Configuration Management Handlers

// ServeConfigPage renders the SCIM configuration page.
func (e *DashboardExtension) ServeConfigPage(ctx *router.PageContext) (g.Node, error) {
	currentUser := e.getUserFromContext(ctx)
	if currentUser == nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, e.baseUIPath+"/login", http.StatusFound)

		return nil, nil
	}

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get organization if in org mode
	orgID, _ := e.getOrgFromContext(ctx)

	content := e.renderConfigPageContent(currentApp, orgID)

	// Return content directly (ForgeUI applies layout automatically)
	return content, nil
}

// renderConfigPageContent renders the configuration page content.
func (e *DashboardExtension) renderConfigPageContent(currentApp any, orgID *xid.ID) g.Node {
	mode := e.detectMode()
	config := e.plugin.config

	scopeLabel := "App"
	if mode == "organization" && orgID != nil {
		scopeLabel = "Organization"
	}

	basePath := e.getBasePath()
	app := currentApp.(*app.App)
	appID := app.ID

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
				g.Text("SCIM Configuration")),
			P(Class("mt-1 text-slate-600 dark:text-gray-400"),
				g.Textf("Configure user provisioning and group sync settings (%s scope)", scopeLabel)),
		),

		// Mode indicator for organization mode
		g.If(mode == "organization" && orgID != nil,
			alertBox("info", "Organization Overrides",
				"These settings can override app-level defaults for this organization. Inherited values are shown with a badge."),
		),

		// User Provisioning Section
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("border-b border-slate-200 p-6 dark:border-gray-800"),
				Div(
					Class("flex items-center gap-3"),
					Div(
						Class("rounded-full bg-violet-100 p-2 dark:bg-violet-900/30"),
						lucide.Users(Class("size-5 text-violet-600 dark:text-violet-400")),
					),
					Div(
						H2(Class("text-xl font-semibold text-slate-900 dark:text-white"),
							g.Text("User Provisioning")),
						P(Class("text-sm text-slate-600 dark:text-gray-400"),
							g.Text("Control how users are provisioned from your identity provider")),
					),
				),
			),
			Form(
				Method("POST"),
				Action(fmt.Sprintf("%s/app/%s/settings/scim-config/user-provisioning", basePath, appID.String())),
				Class("p-6 space-y-4"),

				// Auto-activate users
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("auto_activate"),
							ID("auto_activate"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.UserProvisioning.AutoActivate, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("auto_activate"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Auto-activate users"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Automatically activate users when they are provisioned")),
					),
				),

				// Send welcome email
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("send_welcome_email"),
							ID("send_welcome_email"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.UserProvisioning.SendWelcomeEmail, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("send_welcome_email"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Send welcome email"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Send welcome email when users are first provisioned")),
					),
				),

				// Prevent duplicates
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("prevent_duplicates"),
							ID("prevent_duplicates"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.UserProvisioning.PreventDuplicates, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("prevent_duplicates"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Prevent duplicate emails"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Reject provisioning if a user with the same email already exists")),
					),
				),

				// Soft delete on deprovision
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("soft_delete"),
							ID("soft_delete"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.UserProvisioning.SoftDeleteOnDeProvision, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("soft_delete"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Soft delete on deprovision"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Mark users as inactive instead of permanently deleting them")),
					),
				),

				// Default role
				Div(
					Label(
						For("default_role"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Default Role"),
					),
					Input(
						Type("text"),
						Name("default_role"),
						ID("default_role"),
						Value(config.UserProvisioning.DefaultRole),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						g.Attr("placeholder", "member"),
					),
					P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Default role assigned to provisioned users")),
				),

				// Submit button
				Div(
					Class("flex justify-end pt-4"),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Text("Save User Provisioning Settings"),
					),
				),
			),
		),

		// Group Sync Section
		Div(
			Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("border-b border-slate-200 p-6 dark:border-gray-800"),
				Div(
					Class("flex items-center gap-3"),
					Div(
						Class("rounded-full bg-blue-100 p-2 dark:bg-blue-900/30"),
						lucide.Users(Class("size-5 text-blue-600 dark:text-blue-400")),
					),
					Div(
						H2(Class("text-xl font-semibold text-slate-900 dark:text-white"),
							g.Text("Group Synchronization")),
						P(Class("text-sm text-slate-600 dark:text-gray-400"),
							g.Text("Configure how SCIM groups are synced to teams and roles")),
					),
				),
			),
			Form(
				Method("POST"),
				Action(fmt.Sprintf("%s/app/%s/settings/scim-config/group-sync", basePath, appID.String())),
				Class("p-6 space-y-4"),

				// Enable group sync
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("enabled"),
							ID("group_sync_enabled"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.GroupSync.Enabled, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("group_sync_enabled"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Enable group synchronization"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Sync SCIM groups to teams and/or roles")),
					),
				),

				// Sync to teams
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("sync_to_teams"),
							ID("sync_to_teams"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.GroupSync.SyncToTeams, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("sync_to_teams"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Sync groups to teams"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Create or update teams based on SCIM groups")),
					),
				),

				// Sync to roles
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("sync_to_roles"),
							ID("sync_to_roles"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.GroupSync.SyncToRoles, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("sync_to_roles"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Sync groups to roles"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Create or update roles based on SCIM groups")),
					),
				),

				// Create missing groups
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("create_missing"),
							ID("create_missing"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.GroupSync.CreateMissingGroups, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("create_missing"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Auto-create missing groups"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Automatically create teams/roles if they don't exist")),
					),
				),

				// Delete empty groups
				Div(
					Class("flex items-start"),
					Div(
						Class("flex h-5 items-center"),
						Input(
							Type("checkbox"),
							Name("delete_empty"),
							ID("delete_empty"),
							Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
							g.If(config.GroupSync.DeleteEmptyGroups, g.Attr("checked", "")),
						),
					),
					Div(
						Class("ml-3"),
						Label(
							For("delete_empty"),
							Class("text-sm font-medium text-slate-900 dark:text-white"),
							g.Text("Delete empty groups"),
						),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Remove teams/roles when all members are deprovisioned")),
					),
				),

				// Submit button
				Div(
					Class("flex justify-end pt-4"),
					Button(
						Type("submit"),
						Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Text("Save Group Sync Settings"),
					),
				),
			),
		),

		// Attribute Mapping Section
		e.renderAttributeMappingSection(basePath, &appID, config),

		// Security & Rate Limiting Section
		e.renderSecuritySection(basePath, &appID, config),
	)
}

// renderAttributeMappingSection renders the attribute mapping configuration section.
func (e *DashboardExtension) renderAttributeMappingSection(basePath string, appID *xid.ID, config *Config) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("border-b border-slate-200 p-6 dark:border-gray-800"),
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("rounded-full bg-green-100 p-2 dark:bg-green-900/30"),
					lucide.Map(Class("size-5 text-green-600 dark:text-green-400")),
				),
				Div(
					H2(Class("text-xl font-semibold text-slate-900 dark:text-white"),
						g.Text("Attribute Mapping")),
					P(Class("text-sm text-slate-600 dark:text-gray-400"),
						g.Text("Map SCIM attributes to AuthSome user fields")),
				),
			),
		),
		Form(
			Method("POST"),
			Action(fmt.Sprintf("%s/app/%s/settings/scim-config/attribute-mapping", basePath, appID.String())),
			Class("p-6 space-y-4"),

			// Standard mappings
			Div(
				Class("grid grid-cols-2 gap-4"),

				e.renderMappingField("Username Field", "username_field", config.AttributeMapping.UserNameField),
				e.renderMappingField("Email Field", "email_field", config.AttributeMapping.EmailField),
				e.renderMappingField("Given Name", "given_name_field", config.AttributeMapping.GivenNameField),
				e.renderMappingField("Family Name", "family_name_field", config.AttributeMapping.FamilyNameField),
				e.renderMappingField("Display Name", "display_name_field", config.AttributeMapping.DisplayNameField),
				e.renderMappingField("Active Status", "active_field", config.AttributeMapping.ActiveField),
			),

			// Enterprise extension mappings
			Div(
				Class("pt-4 border-t border-slate-200 dark:border-gray-800"),
				H3(Class("text-sm font-semibold text-slate-900 dark:text-white mb-3"),
					g.Text("Enterprise Extension Mappings")),
				Div(
					Class("grid grid-cols-2 gap-4"),
					e.renderMappingField("Employee Number", "employee_number_field", config.AttributeMapping.EmployeeNumberField),
					e.renderMappingField("Department", "department_field", config.AttributeMapping.DepartmentField),
					e.renderMappingField("Manager", "manager_field", config.AttributeMapping.ManagerField),
				),
			),

			// Submit button
			Div(
				Class("flex justify-end pt-4"),
				Button(
					Type("submit"),
					Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.Text("Save Attribute Mappings"),
				),
			),
		),
	)
}

// renderMappingField renders a single mapping field.
func (e *DashboardExtension) renderMappingField(label, name, value string) g.Node {
	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
			g.Text(label),
		),
		Input(
			Type("text"),
			Name(name),
			ID(name),
			Value(value),
			Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
		),
	)
}

// renderSecuritySection renders the security and rate limiting section.
func (e *DashboardExtension) renderSecuritySection(basePath string, appID *xid.ID, config *Config) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("border-b border-slate-200 p-6 dark:border-gray-800"),
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("rounded-full bg-red-100 p-2 dark:bg-red-900/30"),
					lucide.Shield(Class("size-5 text-red-600 dark:text-red-400")),
				),
				Div(
					H2(Class("text-xl font-semibold text-slate-900 dark:text-white"),
						g.Text("Security & Rate Limiting")),
					P(Class("text-sm text-slate-600 dark:text-gray-400"),
						g.Text("Configure security and performance settings")),
				),
			),
		),
		Form(
			Method("POST"),
			Action(fmt.Sprintf("%s/app/%s/settings/scim-config/security", basePath, appID.String())),
			Class("p-6 space-y-4"),

			// Rate limiting
			Div(
				Class("flex items-start"),
				Div(
					Class("flex h-5 items-center"),
					Input(
						Type("checkbox"),
						Name("rate_limit_enabled"),
						ID("rate_limit_enabled"),
						Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
						g.If(config.RateLimit.Enabled, g.Attr("checked", "")),
					),
				),
				Div(
					Class("ml-3"),
					Label(
						For("rate_limit_enabled"),
						Class("text-sm font-medium text-slate-900 dark:text-white"),
						g.Text("Enable rate limiting"),
					),
					P(Class("text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Protect against excessive requests")),
				),
			),

			// Requests per minute
			Div(
				Label(
					For("requests_per_min"),
					Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Requests Per Minute"),
				),
				Input(
					Type("number"),
					Name("requests_per_min"),
					ID("requests_per_min"),
					Value(strconv.Itoa(config.RateLimit.RequestsPerMin)),
					Min("1"),
					Max("10000"),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
				),
				P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
					g.Text("Maximum number of requests allowed per minute")),
			),

			// Burst size
			Div(
				Label(
					For("burst_size"),
					Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Burst Size"),
				),
				Input(
					Type("number"),
					Name("burst_size"),
					ID("burst_size"),
					Value(strconv.Itoa(config.RateLimit.BurstSize)),
					Min("1"),
					Max("1000"),
					Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
				),
				P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
					g.Text("Maximum burst of requests allowed")),
			),

			// Require HTTPS
			Div(
				Class("flex items-start"),
				Div(
					Class("flex h-5 items-center"),
					Input(
						Type("checkbox"),
						Name("require_https"),
						ID("require_https"),
						Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
						g.If(config.Security.RequireHTTPS, g.Attr("checked", "")),
					),
				),
				Div(
					Class("ml-3"),
					Label(
						For("require_https"),
						Class("text-sm font-medium text-slate-900 dark:text-white"),
						g.Text("Require HTTPS"),
					),
					P(Class("text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Reject non-HTTPS requests in production")),
				),
			),

			// Audit all operations
			Div(
				Class("flex items-start"),
				Div(
					Class("flex h-5 items-center"),
					Input(
						Type("checkbox"),
						Name("audit_all"),
						ID("audit_all"),
						Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
						g.If(config.Security.AuditAllOperations, g.Attr("checked", "")),
					),
				),
				Div(
					Class("ml-3"),
					Label(
						For("audit_all"),
						Class("text-sm font-medium text-slate-900 dark:text-white"),
						g.Text("Audit all operations"),
					),
					P(Class("text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Log all SCIM operations for compliance")),
				),
			),

			// Mask sensitive data
			Div(
				Class("flex items-start"),
				Div(
					Class("flex h-5 items-center"),
					Input(
						Type("checkbox"),
						Name("mask_sensitive"),
						ID("mask_sensitive"),
						Class("h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500"),
						g.If(config.Security.MaskSensitiveData, g.Attr("checked", "")),
					),
				),
				Div(
					Class("ml-3"),
					Label(
						For("mask_sensitive"),
						Class("text-sm font-medium text-slate-900 dark:text-white"),
						g.Text("Mask sensitive data in logs"),
					),
					P(Class("text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Mask emails, phone numbers in audit logs")),
				),
			),

			// Submit button
			Div(
				Class("flex justify-end pt-4"),
				Button(
					Type("submit"),
					Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					g.Text("Save Security Settings"),
				),
			),
		),
	)
}

// HandleUpdateUserProvisioning handles user provisioning settings update.
func (e *DashboardExtension) HandleUpdateUserProvisioning(ctx *router.PageContext) (g.Node, error) {
	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	autoActivate := ctx.Request.FormValue("auto_activate") == "on"
	sendWelcomeEmail := ctx.Request.FormValue("send_welcome_email") == "on"
	preventDuplicates := ctx.Request.FormValue("prevent_duplicates") == "on"
	softDelete := ctx.Request.FormValue("soft_delete") == "on"
	defaultRole := ctx.Request.FormValue("default_role")

	// Update config (in production, persist to config store)
	e.plugin.config.UserProvisioning.AutoActivate = autoActivate
	e.plugin.config.UserProvisioning.SendWelcomeEmail = sendWelcomeEmail
	e.plugin.config.UserProvisioning.PreventDuplicates = preventDuplicates
	e.plugin.config.UserProvisioning.SoftDeleteOnDeProvision = softDelete
	e.plugin.config.UserProvisioning.DefaultRole = defaultRole

	// Extract app from URL for redirect
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Redirect back to config page with success message
	redirectURL := fmt.Sprintf("%s/app/%s/settings/scim-config?success=user_provisioning",
		e.baseUIPath, currentApp.ID.String())
	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL, http.StatusSeeOther)

	return nil, nil
}

// HandleUpdateGroupSync handles group sync settings update.
func (e *DashboardExtension) HandleUpdateGroupSync(ctx *router.PageContext) (g.Node, error) {
	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	enabled := ctx.Request.FormValue("enabled") == "on"
	syncToTeams := ctx.Request.FormValue("sync_to_teams") == "on"
	syncToRoles := ctx.Request.FormValue("sync_to_roles") == "on"
	createMissing := ctx.Request.FormValue("create_missing") == "on"
	deleteEmpty := ctx.Request.FormValue("delete_empty") == "on"

	// Update config
	e.plugin.config.GroupSync.Enabled = enabled
	e.plugin.config.GroupSync.SyncToTeams = syncToTeams
	e.plugin.config.GroupSync.SyncToRoles = syncToRoles
	e.plugin.config.GroupSync.CreateMissingGroups = createMissing
	e.plugin.config.GroupSync.DeleteEmptyGroups = deleteEmpty

	// Extract app from URL for redirect
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Redirect back to config page with success message
	redirectURL := fmt.Sprintf("%s/app/%s/settings/scim-config?success=group_sync",
		e.baseUIPath, currentApp.ID.String())
	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL, http.StatusSeeOther)

	return nil, nil
}

// HandleUpdateAttributeMapping handles attribute mapping update.
func (e *DashboardExtension) HandleUpdateAttributeMapping(ctx *router.PageContext) (g.Node, error) {
	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	e.plugin.config.AttributeMapping.UserNameField = ctx.Request.FormValue("username_field")
	e.plugin.config.AttributeMapping.EmailField = ctx.Request.FormValue("email_field")
	e.plugin.config.AttributeMapping.GivenNameField = ctx.Request.FormValue("given_name_field")
	e.plugin.config.AttributeMapping.FamilyNameField = ctx.Request.FormValue("family_name_field")
	e.plugin.config.AttributeMapping.DisplayNameField = ctx.Request.FormValue("display_name_field")
	e.plugin.config.AttributeMapping.ActiveField = ctx.Request.FormValue("active_field")
	e.plugin.config.AttributeMapping.EmployeeNumberField = ctx.Request.FormValue("employee_number_field")
	e.plugin.config.AttributeMapping.DepartmentField = ctx.Request.FormValue("department_field")
	e.plugin.config.AttributeMapping.ManagerField = ctx.Request.FormValue("manager_field")

	// Extract app from URL for redirect
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Redirect back to config page with success message
	redirectURL := fmt.Sprintf("%s/app/%s/settings/scim-config?success=attribute_mapping",
		e.baseUIPath, currentApp.ID.String())
	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL, http.StatusSeeOther)

	return nil, nil
}

// HandleUpdateSecurity handles security settings update.
func (e *DashboardExtension) HandleUpdateSecurity(ctx *router.PageContext) (g.Node, error) {
	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	rateLimitEnabled := ctx.Request.FormValue("rate_limit_enabled") == "on"
	requestsPerMin, _ := strconv.Atoi(ctx.Request.FormValue("requests_per_min"))
	burstSize, _ := strconv.Atoi(ctx.Request.FormValue("burst_size"))
	requireHTTPS := ctx.Request.FormValue("require_https") == "on"
	auditAll := ctx.Request.FormValue("audit_all") == "on"
	maskSensitive := ctx.Request.FormValue("mask_sensitive") == "on"

	// Update config
	e.plugin.config.RateLimit.Enabled = rateLimitEnabled
	if requestsPerMin > 0 {
		e.plugin.config.RateLimit.RequestsPerMin = requestsPerMin
	}

	if burstSize > 0 {
		e.plugin.config.RateLimit.BurstSize = burstSize
	}

	e.plugin.config.Security.RequireHTTPS = requireHTTPS
	e.plugin.config.Security.AuditAllOperations = auditAll
	e.plugin.config.Security.MaskSensitiveData = maskSensitive

	// Extract app from URL for redirect
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Redirect back to config page with success message
	redirectURL := fmt.Sprintf("%s/app/%s/settings/scim-config?success=security",
		e.baseUIPath, currentApp.ID.String())
	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL, http.StatusSeeOther)

	return nil, nil
}

// HandleUpdateWebhooks handles webhook configuration update.
func (e *DashboardExtension) HandleUpdateWebhooks(ctx *router.PageContext) (g.Node, error) {
	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	enabled := ctx.Request.FormValue("enabled") == "on"
	notifyOnCreate := ctx.Request.FormValue("notify_on_create") == "on"
	notifyOnUpdate := ctx.Request.FormValue("notify_on_update") == "on"
	notifyOnDelete := ctx.Request.FormValue("notify_on_delete") == "on"
	webhookURLs := ctx.Request.FormValue("webhook_urls")

	// Parse URLs
	urls := []string{}
	if webhookURLs != "" {
		urls = strings.Split(webhookURLs, "\n")
	}

	// Update config
	e.plugin.config.Webhooks.Enabled = enabled
	e.plugin.config.Webhooks.NotifyOnCreate = notifyOnCreate
	e.plugin.config.Webhooks.NotifyOnUpdate = notifyOnUpdate
	e.plugin.config.Webhooks.NotifyOnDelete = notifyOnDelete
	e.plugin.config.Webhooks.WebhookURLs = urls

	// Extract app from URL for redirect
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Redirect back to config page with success message
	redirectURL := fmt.Sprintf("%s/app/%s/settings/scim-config?success=webhooks",
		e.baseUIPath, currentApp.ID.String())
	http.Redirect(ctx.ResponseWriter, ctx.Request, redirectURL, http.StatusSeeOther)

	return nil, nil
}
