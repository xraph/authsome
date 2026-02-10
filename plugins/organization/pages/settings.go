package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library

	"github.com/xraph/forgeui/components/badge"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/primitives"
)

// OrganizationSettingsPage renders the organization settings page with ForgeUI components.
func OrganizationSettingsPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()
	appBase := fmt.Sprintf("%s/app/%s", basePath, appID)

	return Div(
		g.Attr("x-data", settingsPageData(appID)),
		g.Attr("x-init", "loadSettings()"),
		Class("space-y-6"),

		// Back link
		BackLink(appBase+"/organizations", "Back to Organizations"),

		// Page header with save button
		Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
			primitives.VStack("gap-1",
				H1(Class("text-2xl font-bold"), g.Text("Organization Settings")),
				P(Class("text-sm text-muted-foreground"), g.Text("Configure organization-wide settings and preferences")),
			),
			Div(
				Class("flex items-center gap-2"),
				// Success message
				Div(
					g.Attr("x-show", "saveSuccess"),
					g.Attr("x-transition", ""),
					Class("flex items-center gap-2 text-sm text-emerald-600"),
					lucide.CircleCheck(Class("size-4")),
					g.Text("Settings saved"),
				),
				button.Button(
					Div(
						g.Attr("x-show", "!saving"),
						Class("flex items-center gap-2"),
						lucide.Save(Class("size-4")),
						g.Text("Save Changes"),
					),
					button.WithVariant("default"),
					button.WithAttrs(
						g.Attr("@click", "saveSettings()"),
						g.Attr(":disabled", "saving || loading"),
						g.Attr("x-show", "hasChanges"),
					),
				),
				Div(
					g.Attr("x-show", "saving"),
					Class("flex items-center gap-2 text-sm text-muted-foreground"),
					Div(Class("animate-spin rounded-full h-4 w-4 border-b-2 border-primary")),
					g.Text("Saving..."),
				),
			),
		),

		// Loading state
		Div(
			g.Attr("x-show", "loading"),
			Class("flex items-center justify-center py-12"),
			Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
		),

		// Error state
		Div(
			g.Attr("x-show", "error && !loading"),
			g.Attr("x-cloak", ""),
			Class("bg-destructive/10 border border-destructive/20 rounded-lg p-4"),
			Div(
				Class("flex items-center gap-2 text-destructive"),
				lucide.TriangleAlert(Class("size-5")),
				Span(g.Attr("x-text", "error"), g.Text("An error occurred")),
			),
		),

		// Settings content
		Div(
			g.Attr("x-show", "!loading"),
			g.Attr("x-cloak", ""),
			Class("space-y-6"),

			// General Settings Section
			generalSettingsSectionV2(appID),

			// Creation Settings Section
			creationSettingsSection(appID),

			// Membership Settings Section
			membershipSettingsSection(appID),

			// Role Templates Section
			roleTemplatesSectionV2(currentApp, basePath),
		),
	)
}

// generalSettingsSectionV2 renders the general settings card with Alpine.js bindings.
func generalSettingsSectionV2(appID string) g.Node {
	return card.Card(
		card.Header(
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("rounded-lg bg-primary/10 p-2"),
					lucide.Settings(Class("size-5 text-primary")),
				),
				Div(
					card.Title("General Settings"),
					card.Description("Basic organization plugin configuration"),
				),
			),
		),
		card.Content(
			Class("space-y-6"),

			// Organizations Enabled Toggle
			Div(
				Class("flex items-center justify-between py-3 border-b"),
				Div(
					Class("space-y-1"),
					Label(Class("text-sm font-medium"), g.Text("Enable Organizations")),
					P(Class("text-xs text-muted-foreground"), g.Text("Allow users to create and manage organizations in this app")),
				),
				Div(
					Class("relative"),
					// Toggle switch
					Label(
						Class("relative inline-flex items-center cursor-pointer"),
						Input(
							Type("checkbox"),
							Class("sr-only peer"),
							g.Attr("x-model", "settings.enabled"),
							g.Attr("@change", "hasChanges = true"),
						),
						Div(
							Class("w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/20 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-primary"),
						),
					),
				),
			),

			// Allow User Creation Toggle
			Div(
				Class("flex items-center justify-between py-3 border-b"),
				Div(
					Class("space-y-1"),
					Label(Class("text-sm font-medium"), g.Text("Allow User Organization Creation")),
					P(Class("text-xs text-muted-foreground"), g.Text("Allow regular users to create their own organizations")),
				),
				Div(
					Class("relative"),
					Label(
						Class("relative inline-flex items-center cursor-pointer"),
						Input(
							Type("checkbox"),
							Class("sr-only peer"),
							g.Attr("x-model", "settings.allowUserCreation"),
							g.Attr("@change", "hasChanges = true"),
						),
						Div(
							Class("w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/20 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-primary"),
						),
					),
				),
			),

			// Default Organization Role
			Div(
				Class("space-y-2 py-3"),
				Label(
					For("defaultRole"),
					Class("text-sm font-medium"),
					g.Text("Default Member Role"),
				),
				Select(
					ID("defaultRole"),
					Class("flex h-10 w-full max-w-xs rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"),
					g.Attr("x-model", "settings.defaultRole"),
					g.Attr("@change", "hasChanges = true"),
					Option(Value("member"), g.Text("Member")),
					Option(Value("admin"), g.Text("Admin")),
					Option(Value("viewer"), g.Text("Viewer")),
				),
				P(Class("text-xs text-muted-foreground"), g.Text("Default role assigned to new organization members")),
			),
		),
	)
}

// creationSettingsSection renders the organization creation settings.
func creationSettingsSection(appID string) g.Node {
	return card.Card(
		card.Header(
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("rounded-lg bg-blue-500/10 p-2"),
					lucide.Building2(Class("size-5 text-blue-500")),
				),
				Div(
					card.Title("Creation Limits"),
					card.Description("Control how many organizations users can create"),
				),
			),
		),
		card.Content(
			Class("space-y-6"),

			// Max Organizations per User
			Div(
				Class("space-y-2"),
				Label(
					For("maxOrgsPerUser"),
					Class("text-sm font-medium"),
					g.Text("Max Organizations per User"),
				),
				Div(
					Class("flex items-center gap-3"),
					input.Input(
						input.WithType("number"),
						input.WithID("maxOrgsPerUser"),
						input.WithName("maxOrgsPerUser"),
						input.WithAttrs(
							Class("max-w-[120px]"),
							Min("1"),
							Max("100"),
							g.Attr("x-model.number", "settings.maxOrgsPerUser"),
							g.Attr("@input", "hasChanges = true"),
						),
					),
					Span(Class("text-sm text-muted-foreground"), g.Text("organizations")),
				),
				P(Class("text-xs text-muted-foreground"), g.Text("Maximum number of organizations a single user can create (1-100)")),
			),

			// Max Members per Organization
			Div(
				Class("space-y-2"),
				Label(
					For("maxMembersPerOrg"),
					Class("text-sm font-medium"),
					g.Text("Max Members per Organization"),
				),
				Div(
					Class("flex items-center gap-3"),
					input.Input(
						input.WithType("number"),
						input.WithID("maxMembersPerOrg"),
						input.WithName("maxMembersPerOrg"),
						input.WithAttrs(
							Class("max-w-[120px]"),
							Min("1"),
							Max("10000"),
							g.Attr("x-model.number", "settings.maxMembersPerOrg"),
							g.Attr("@input", "hasChanges = true"),
						),
					),
					Span(Class("text-sm text-muted-foreground"), g.Text("members")),
				),
				P(Class("text-xs text-muted-foreground"), g.Text("Maximum number of members allowed in each organization")),
			),

			// Max Teams per Organization
			Div(
				Class("space-y-2"),
				Label(
					For("maxTeamsPerOrg"),
					Class("text-sm font-medium"),
					g.Text("Max Teams per Organization"),
				),
				Div(
					Class("flex items-center gap-3"),
					input.Input(
						input.WithType("number"),
						input.WithID("maxTeamsPerOrg"),
						input.WithName("maxTeamsPerOrg"),
						input.WithAttrs(
							Class("max-w-[120px]"),
							Min("0"),
							Max("1000"),
							g.Attr("x-model.number", "settings.maxTeamsPerOrg"),
							g.Attr("@input", "hasChanges = true"),
						),
					),
					Span(Class("text-sm text-muted-foreground"), g.Text("teams")),
				),
				P(Class("text-xs text-muted-foreground"), g.Text("Maximum number of teams allowed in each organization (0 for unlimited)")),
			),
		),
	)
}

// membershipSettingsSection renders the membership-related settings.
func membershipSettingsSection(appID string) g.Node {
	return card.Card(
		card.Header(
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("rounded-lg bg-emerald-500/10 p-2"),
					lucide.Users(Class("size-5 text-emerald-500")),
				),
				Div(
					card.Title("Membership Settings"),
					card.Description("Configure invitation and membership behavior"),
				),
			),
		),
		card.Content(
			Class("space-y-6"),

			// Require Invitation Toggle
			Div(
				Class("flex items-center justify-between py-3 border-b"),
				Div(
					Class("space-y-1"),
					Label(Class("text-sm font-medium"), g.Text("Require Invitation")),
					P(Class("text-xs text-muted-foreground"), g.Text("Users must be invited to join organizations")),
				),
				Div(
					Class("relative"),
					Label(
						Class("relative inline-flex items-center cursor-pointer"),
						Input(
							Type("checkbox"),
							Class("sr-only peer"),
							g.Attr("x-model", "settings.requireInvitation"),
							g.Attr("@change", "hasChanges = true"),
						),
						Div(
							Class("w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/20 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-primary"),
						),
					),
				),
			),

			// Invitation Expiry
			Div(
				Class("space-y-2 py-3 border-b"),
				Label(
					For("invitationExpiryDays"),
					Class("text-sm font-medium"),
					g.Text("Invitation Expiry"),
				),
				Div(
					Class("flex items-center gap-3"),
					input.Input(
						input.WithType("number"),
						input.WithID("invitationExpiryDays"),
						input.WithName("invitationExpiryDays"),
						input.WithAttrs(
							Class("max-w-[120px]"),
							Min("1"),
							Max("365"),
							g.Attr("x-model.number", "settings.invitationExpiryDays"),
							g.Attr("@input", "hasChanges = true"),
						),
					),
					Span(Class("text-sm text-muted-foreground"), g.Text("days")),
				),
				P(Class("text-xs text-muted-foreground"), g.Text("Number of days before pending invitations expire")),
			),

			// Allow Multiple Memberships Toggle
			Div(
				Class("flex items-center justify-between py-3"),
				Div(
					Class("space-y-1"),
					Label(Class("text-sm font-medium"), g.Text("Allow Multiple Memberships")),
					P(Class("text-xs text-muted-foreground"), g.Text("Users can be members of multiple organizations")),
				),
				Div(
					Class("relative"),
					Label(
						Class("relative inline-flex items-center cursor-pointer"),
						Input(
							Type("checkbox"),
							Class("sr-only peer"),
							g.Attr("x-model", "settings.allowMultipleMemberships"),
							g.Attr("@change", "hasChanges = true"),
						),
						Div(
							Class("w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/20 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-primary"),
						),
					),
				),
			),
		),
	)
}

// roleTemplatesSectionV2 renders the role templates section with improved UI.
func roleTemplatesSectionV2(currentApp *app.App, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())

	return card.Card(
		card.Header(
			Div(
				Class("flex items-center justify-between w-full"),
				Div(
					Class("flex items-center gap-3"),
					Div(
						Class("rounded-lg bg-violet-500/10 p-2"),
						lucide.ShieldCheck(Class("size-5 text-violet-500")),
					),
					Div(
						card.Title("Role Templates"),
						card.Description("Default role templates for new organizations"),
					),
				),
				button.Button(
					Div(
						Class("flex items-center gap-2"),
						lucide.Plus(Class("size-4")),
						g.Text("Create Template"),
					),
					button.WithVariant("default"),
					button.WithSize("sm"),
					button.WithAttrs(
						Type("button"),
						g.Attr("@click", fmt.Sprintf("window.location.href='%s/settings/roles/create'", appBase)),
					),
				),
			),
		),
		card.Content(
			// Loading state for templates
			Div(
				g.Attr("x-show", "templatesLoading"),
				Class("flex items-center justify-center py-8"),
				Div(Class("animate-spin rounded-full h-6 w-6 border-b-2 border-primary")),
			),

			// Empty state
			Div(
				g.Attr("x-show", "!templatesLoading && templates.length === 0"),
				g.Attr("x-cloak", ""),
				Class("text-center py-8"),
				lucide.ShieldCheck(Class("size-12 mx-auto mb-3 text-muted-foreground/50")),
				P(Class("text-sm font-medium text-muted-foreground"), g.Text("No role templates")),
				P(Class("text-xs text-muted-foreground mt-1"), g.Text("Create role templates to standardize permissions across organizations")),
			),

			// Templates list
			Div(
				g.Attr("x-show", "!templatesLoading && templates.length > 0"),
				g.Attr("x-cloak", ""),
				Class("space-y-3"),

				Template(
					g.Attr("x-for", "template in templates"),
					g.Attr(":key", "template.id"),
					Div(
						Class("flex items-center justify-between p-4 border rounded-lg hover:bg-accent/50 transition-colors group"),
						Div(
							Class("flex items-center gap-4 flex-1 min-w-0"),
							// Role icon
							Div(
								Class("rounded-lg bg-muted p-2"),
								lucide.Shield(Class("size-5 text-muted-foreground")),
							),
							// Role info
							Div(
								Class("flex-1 min-w-0"),
								H4(
									Class("text-sm font-medium truncate"),
									g.Attr("x-text", "template.name"),
								),
								P(
									Class("text-xs text-muted-foreground mt-0.5 truncate"),
									g.Attr("x-text", "template.description || 'No description'"),
								),
								Div(
									Class("flex items-center gap-2 mt-2"),
									badge.Badge(
										"",
										badge.WithClass("text-xs"),
										badge.WithAttrs(
											g.Attr("x-text", "(template.permissions?.length || 0) + ' permissions'"),
										),
									),
								),
							),
						),
						// Actions
						Div(
							Class("flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"),
							button.Button(
								lucide.Pencil(Class("size-4")),
								button.WithVariant("ghost"),
								button.WithSize("icon"),
								button.WithAttrs(
									Type("button"),
									g.Attr("@click", fmt.Sprintf("window.location.href='%s/settings/roles/' + template.id + '/edit'", appBase)),
									Title("Edit template"),
								),
							),
							button.Button(
								lucide.Trash2(Class("size-4")),
								button.WithVariant("ghost"),
								button.WithSize("icon"),
								button.WithAttrs(
									Type("button"),
									Class("text-destructive hover:text-destructive"),
									g.Attr("@click", "deleteTemplate(template.id)"),
									Title("Delete template"),
								),
							),
						),
					),
				),
			),
		),
	)
}

// settingsPageData returns the Alpine.js data object for the settings page.
func settingsPageData(appID string) string {
	return fmt.Sprintf(`{
		appId: '%s',
		loading: true,
		saving: false,
		error: null,
		saveSuccess: false,
		hasChanges: false,
		
		settings: {
			enabled: true,
			allowUserCreation: true,
			defaultRole: 'member',
			maxOrgsPerUser: 10,
			maxMembersPerOrg: 100,
			maxTeamsPerOrg: 20,
			requireInvitation: true,
			invitationExpiryDays: 7,
			allowMultipleMemberships: true
		},
		
		templates: [],
		templatesLoading: true,
		
		async loadSettings() {
			this.loading = true;
			this.error = null;
			try {
				// Load settings
				const result = await $bridge.call('organization.getSettings', {
					appId: this.appId
				});
				if (result && result.settings) {
					this.settings = { ...this.settings, ...result.settings };
				}
			} catch (err) {
				console.error('Failed to load settings:', err);
				// Use defaults if no settings exist
			} finally {
				this.loading = false;
			}
			
			// Load templates in parallel
			this.loadTemplates();
		},
		
		async loadTemplates() {
			this.templatesLoading = true;
			try {
				const result = await $bridge.call('organization.getRoleTemplates', {
					appId: this.appId
				});
				this.templates = result.templates || [];
			} catch (err) {
				console.error('Failed to load templates:', err);
			} finally {
				this.templatesLoading = false;
			}
		},
		
		async saveSettings() {
			this.saving = true;
			this.error = null;
			this.saveSuccess = false;
			try {
				await $bridge.call('organization.updateSettings', {
					appId: this.appId,
					settings: this.settings
				});
				this.saveSuccess = true;
				this.hasChanges = false;
				setTimeout(() => { this.saveSuccess = false; }, 3000);
			} catch (err) {
				console.error('Failed to save settings:', err);
				this.error = err.message || 'Failed to save settings';
			} finally {
				this.saving = false;
			}
		},
		
		async deleteTemplate(templateId) {
			if (!confirm('Are you sure you want to delete this role template? This action cannot be undone.')) {
				return;
			}
			
			try {
				await $bridge.call('organization.deleteRoleTemplate', {
					appId: this.appId,
					templateId: templateId
				});
				await this.loadTemplates();
			} catch (err) {
				console.error('Failed to delete template:', err);
				alert('Failed to delete template: ' + (err.message || 'Unknown error'));
			}
		}
	}`, appID)
}

// RoleTemplateFormPage renders the create/edit role template form page.
func RoleTemplateFormPage(currentApp *app.App, templateID, basePath string, isEdit bool) g.Node {
	appID := currentApp.ID.String()
	appBase := fmt.Sprintf("%s/app/%s", basePath, appID)

	title := "Create Role Template"
	if isEdit {
		title = "Edit Role Template"
	}

	return Div(
		g.Attr("x-data", roleTemplateFormData(appID, templateID, isEdit, appBase)),
		g.If(isEdit, g.Attr("x-init", "loadTemplate()")),
		Class("space-y-6"),

		// Back link
		BackLink(appBase+"/settings/organizations", "Back to Settings"),

		// Page header
		PageHeader(
			title,
			"Define permissions for this role template",
		),

		// Loading state (for edit mode)
		g.If(isEdit, func() g.Node {
			return Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-12"),
				Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
			)
		}()),

		// Error state
		Div(
			g.Attr("x-show", "error"),
			g.Attr("x-cloak", ""),
			Class("bg-destructive/10 border border-destructive/20 rounded-lg p-4"),
			Div(
				Class("flex items-center gap-2 text-destructive"),
				lucide.TriangleAlert(Class("size-5")),
				Span(g.Attr("x-text", "error"), g.Text("An error occurred")),
			),
		),

		// Form
		Div(
			g.If(isEdit, g.Attr("x-show", "!loading")),
			g.Attr("x-cloak", ""),

			card.Card(
				card.Header(
					card.Title("Template Details"),
					card.Description("Basic information about this role template"),
				),
				card.Content(
					Form(
						g.Attr("@submit.prevent", "saveTemplate()"),
						Class("space-y-6"),

						// Template Name
						Div(
							Class("space-y-2"),
							Label(
								For("name"),
								Class("text-sm font-medium"),
								g.Text("Template Name"),
								Span(Class("text-destructive ml-1"), g.Text("*")),
							),
							input.Input(
								input.WithType("text"),
								input.WithID("name"),
								input.WithName("name"),
								input.WithPlaceholder("e.g., Administrator, Editor, Viewer"),
								input.WithAttrs(
									Required(),
									g.Attr("x-model", "form.name"),
								),
							),
							P(Class("text-xs text-muted-foreground"), g.Text("A descriptive name for this role template")),
						),

						// Description
						Div(
							Class("space-y-2"),
							Label(
								For("description"),
								Class("text-sm font-medium"),
								g.Text("Description"),
							),
							Textarea(
								ID("description"),
								Name("description"),
								Placeholder("Describe the responsibilities and access level of this role..."),
								Rows("3"),
								Class("flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"),
								g.Attr("x-model", "form.description"),
							),
							P(Class("text-xs text-muted-foreground"), g.Text("Brief description of the role's responsibilities")),
						),

						// Permissions Section
						Div(
							Class("space-y-4 pt-4 border-t"),
							Div(
								Class("flex items-center justify-between"),
								Div(
									Label(Class("text-sm font-medium"), g.Text("Permissions")),
									P(Class("text-xs text-muted-foreground"), g.Text("Select the permissions for this role")),
								),
								Div(
									Class("flex gap-2"),
									button.Button(
										g.Text("Select All"),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(
											Type("button"),
											g.Attr("@click", "selectAllPermissions()"),
										),
									),
									button.Button(
										g.Text("Clear All"),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(
											Type("button"),
											g.Attr("@click", "clearAllPermissions()"),
										),
									),
								),
							),

							// Permission groups
							Div(
								Class("grid gap-4 md:grid-cols-2"),

								// Member Management
								permissionGroup("Members", []settingsPermissionItem{
									{ID: "manage_members", Label: "Manage Members", Description: "Add, remove, and update organization members"},
									{ID: "invite_members", Label: "Invite Members", Description: "Send invitations to new members"},
									{ID: "view_members", Label: "View Members", Description: "View member list and details"},
								}),

								// Team Management
								permissionGroup("Teams", []settingsPermissionItem{
									{ID: "manage_teams", Label: "Manage Teams", Description: "Create, update, and delete teams"},
									{ID: "assign_team_members", Label: "Assign Team Members", Description: "Add or remove team members"},
									{ID: "view_teams", Label: "View Teams", Description: "View team list and details"},
								}),

								// Role Management
								permissionGroup("Roles", []settingsPermissionItem{
									{ID: "manage_roles", Label: "Manage Roles", Description: "Assign and modify member roles"},
									{ID: "view_roles", Label: "View Roles", Description: "View role assignments"},
								}),

								// Organization Management
								permissionGroup("Organization", []settingsPermissionItem{
									{ID: "manage_settings", Label: "Manage Settings", Description: "Update organization settings"},
									{ID: "view_audit_logs", Label: "View Audit Logs", Description: "Access organization audit logs"},
									{ID: "delete_organization", Label: "Delete Organization", Description: "Permanently delete the organization"},
								}),
							),
						),

						// Actions
						Div(
							Class("flex justify-end gap-3 pt-6 border-t"),
							button.Button(
								g.Text("Cancel"),
								button.WithVariant("outline"),
								button.WithAttrs(
									Type("button"),
									g.Attr("@click", fmt.Sprintf("window.location.href='%s/settings/organizations'", appBase)),
									g.Attr(":disabled", "saving"),
								),
							),
							button.Button(
								Div(
									g.Attr("x-show", "!saving"),
									Class("flex items-center gap-2"),
									lucide.Save(Class("size-4")),
									g.Text("Save Template"),
								),
								button.WithVariant("default"),
								button.WithAttrs(
									Type("submit"),
									g.Attr(":disabled", "saving"),
								),
							),
							Div(
								g.Attr("x-show", "saving"),
								Class("flex items-center gap-2 text-sm text-muted-foreground"),
								Div(Class("animate-spin rounded-full h-4 w-4 border-b-2 border-primary")),
								g.Text("Saving..."),
							),
						),
					),
				),
			),
		),
	)
}

// settingsPermissionItem represents a single permission checkbox for settings page.
type settingsPermissionItem struct {
	ID          string
	Label       string
	Description string
}

// permissionGroup renders a group of related permissions.
func permissionGroup(groupName string, items []settingsPermissionItem) g.Node {
	checkboxes := make([]g.Node, len(items))
	for i, item := range items {
		checkboxes[i] = Div(
			Class("flex items-start gap-3 py-2"),
			Input(
				Type("checkbox"),
				ID(item.ID),
				Value(item.ID),
				Class("mt-1 h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"),
				g.Attr("x-model", "form.permissions"),
			),
			Label(
				For(item.ID),
				Class("flex-1 cursor-pointer"),
				Div(
					Class("text-sm font-medium"),
					g.Text(item.Label),
				),
				P(
					Class("text-xs text-muted-foreground"),
					g.Text(item.Description),
				),
			),
		)
	}

	return Div(
		Class("rounded-lg border p-4"),
		H4(Class("text-sm font-semibold mb-3 text-muted-foreground"), g.Text(groupName)),
		Div(Class("space-y-1"), g.Group(checkboxes)),
	)
}

// roleTemplateFormData returns the Alpine.js data object for role template form.
func roleTemplateFormData(appID, templateID string, isEdit bool, appBase string) string {
	return fmt.Sprintf(`{
		form: {
			name: '',
			description: '',
			permissions: []
		},
		loading: %t,
		error: null,
		saving: false,
		allPermissions: [
			'manage_members', 'invite_members', 'view_members',
			'manage_teams', 'assign_team_members', 'view_teams',
			'manage_roles', 'view_roles',
			'manage_settings', 'view_audit_logs', 'delete_organization'
		],
		
		async loadTemplate() {
			if (!'%s') return;
			
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('organization.getRoleTemplate', {
					appId: '%s',
					templateId: '%s'
				});
				
				this.form.name = result.template.name;
				this.form.description = result.template.description || '';
				this.form.permissions = result.template.permissions || [];
			} catch (err) {
				console.error('Failed to load role template:', err);
				this.error = err.message || 'Failed to load role template';
			} finally {
				this.loading = false;
			}
		},
		
		selectAllPermissions() {
			this.form.permissions = [...this.allPermissions];
		},
		
		clearAllPermissions() {
			this.form.permissions = [];
		},
		
		async saveTemplate() {
			if (!this.form.name.trim()) {
				this.error = 'Template name is required';
				return;
			}
			
			this.saving = true;
			this.error = null;
			try {
				const action = '%s' ? 'organization.updateRoleTemplate' : 'organization.createRoleTemplate';
				const params = {
					appId: '%s',
					name: this.form.name,
					description: this.form.description,
					permissions: this.form.permissions
				};
				
				if ('%s') {
					params.templateId = '%s';
				}
				
				await $bridge.call(action, params);
				
				// Redirect back to settings
				window.location.href = '%s/settings/organizations';
			} catch (err) {
				console.error('Failed to save role template:', err);
				this.error = err.message || 'Failed to save role template';
			} finally {
				this.saving = false;
			}
		}
	}`, isEdit, templateID, appID, templateID, templateID, appID, templateID, templateID, appBase)
}
