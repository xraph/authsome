package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/button"
)

// OrganizationDetailPage renders the organization detail/overview page
func OrganizationDetailPage(currentApp *app.App, orgID, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())

	return Div(
		Class("space-y-6"),

		// Alpine.js data
		Div(
			g.Attr("x-data", organizationDetailData(currentApp.ID.String(), orgID, basePath)),
			g.Attr("x-init", "loadData()"),

			// Back link
			BackLink(appBase+"/organizations", "Back to Organizations"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				LoadingSpinner(),
			),

		// Error state
		ErrorMessage("error && !loading"),

		// Empty/Not Found state (when no error but organization is null)
		Div(
			g.Attr("x-show", "!loading && !error && !organization"),
			g.Attr("x-cloak", ""),
			Class("text-center py-12"),
			Card(
				Class("max-w-md mx-auto p-8"),
				EmptyState(
					lucide.Building2(Class("size-16 mx-auto mb-4 opacity-50")),
					"Organization Not Found",
					"The organization you're looking for doesn't exist or you don't have access to it.",
				),
				Div(
					Class("mt-6"),
					button.Button(
						Div(
							lucide.ArrowLeft(Class("size-4")),
							g.Text("Back to Organizations"),
						),
						button.WithVariant("outline"),
						button.WithAttrs(
							g.Attr("onclick", fmt.Sprintf("window.location.href='%s/app/%s/organizations'", basePath, currentApp.ID.String())),
						),
					),
				),
			),
		),

		// Content
		Div(
			g.Attr("x-show", "!loading && !error && organization"),
			g.Attr("x-cloak", ""),
			Class("space-y-6"),

			// Organization header
			organizationHeader(appBase, orgID),

				// Tabs navigation
				organizationTabs(appBase, orgID),

				// Stats cards
				Div(
					Class("grid gap-6 md:grid-cols-3"),
					StatsCard("Members", "stats.memberCount", "emerald"),
					StatsCard("Teams", "stats.teamCount", "blue"),
					StatsCard("Pending Invitations", "stats.invitationCount", "amber"),
				),

				// Quick links section
				quickLinksSection(appBase, orgID),

				// Extension widgets
				extensionWidgetsSection(),
			),

			// Delete confirmation modal
			deleteConfirmationModal(appBase, orgID),
		),
	)
}

// organizationDetailData returns the Alpine.js data object
func organizationDetailData(appID, orgID, basePath string) string {
	return fmt.Sprintf(`{
		organization: null,
		userRole: '',
		stats: {
			memberCount: 0,
			teamCount: 0,
			invitationCount: 0
		},
		extensionData: {
			widgets: [],
			tabs: [],
			actions: [],
			quickLinks: []
		},
		loading: true,
		error: null,
		showDeleteModal: false,
		deleting: false,
		
		get canDelete() {
			return this.userRole === 'owner';
		},
		
		get canManage() {
			return this.userRole === 'owner' || this.userRole === 'admin';
		},
		
		async loadData() {
			this.loading = true;
			this.error = null;
			try {
				// Load organization details
				const result = await $bridge.call('organization.getOrganization', {
					appId: '%s',
					orgId: '%s'
				});
				
				this.organization = result.organization;
				this.userRole = result.userRole || '';
				this.stats = result.stats || { memberCount: 0, teamCount: 0, invitationCount: 0 };
				
				// Load extension data
				await this.loadExtensions();
			} catch (err) {
				console.error('Failed to load organization:', err);
				this.error = err.message || 'Failed to load organization';
			} finally {
				this.loading = false;
			}
		},
		
		async loadExtensions() {
			try {
				const result = await $bridge.call('organization.getExtensionData', {
					appId: '%s',
					orgId: '%s'
				});
				this.extensionData = result || { widgets: [], tabs: [], actions: [], quickLinks: [] };
			} catch (err) {
				console.warn('Failed to load extension data:', err);
			}
		},
		
		async deleteOrganization() {
			if (!this.canDelete) return;
			
			this.deleting = true;
			try {
				await $bridge.call('organization.deleteOrganization', {
					appId: '%s',
					orgId: '%s'
				});
				
				// Redirect to organizations list
				window.location.href = '%s/app/%s/organizations';
			} catch (err) {
				console.error('Failed to delete organization:', err);
				alert('Failed to delete organization: ' + (err.message || 'Unknown error'));
			} finally {
				this.deleting = false;
				this.showDeleteModal = false;
			}
		}
	}`, appID, orgID, appID, orgID, appID, orgID, basePath, appID)
}

// organizationHeader renders the organization header with logo, name, and actions
func organizationHeader(appBase, orgID string) g.Node {
	return Card(
		Class("p-6"),
		Div(
			Class("flex items-center justify-between"),
			// Left side: Logo and info
			Div(
				Class("flex items-center gap-4"),
				// Logo
				Div(
					g.Attr("x-show", "organization?.logo"),
					Img(
						g.Attr(":src", "organization.logo"),
						g.Attr(":alt", "organization.name"),
						Class("size-16 rounded-lg object-cover"),
					),
				),
				Div(
					g.Attr("x-show", "!organization?.logo"),
					Class("size-16 rounded-lg bg-primary/10 flex items-center justify-center"),
					lucide.Building2(Class("size-8 text-primary")),
				),
				// Name and slug
				Div(
					H1(
						Class("text-2xl font-bold"),
						g.Attr("x-text", "organization?.name || 'Loading...'"),
					),
					P(
						Class("text-sm text-muted-foreground"),
						g.Text("@"),
						Span(g.Attr("x-text", "organization?.slug || ''")),
					),
					Div(
						Class("mt-2"),
						Span(
							Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize"),
							g.Attr(":class", `{
								'bg-primary text-primary-foreground': userRole === 'owner',
								'bg-secondary text-secondary-foreground': userRole === 'admin',
								'bg-muted text-muted-foreground': userRole === 'member'
							}`),
							g.Attr("x-text", "userRole"),
						),
					),
				),
			),
			// Right side: Actions
			Div(
				Class("flex items-center gap-2"),
				// Extension actions (dynamic)
				Template(
					g.Attr("x-for", "action in extensionData.actions"),
					g.Attr(":key", "action.id"),
					Button(
						Type("button"),
						g.Attr("@click", "eval(action.action)"),
						Class("inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 h-9 px-4 py-2"),
						g.Attr(":class", `{
							'bg-primary text-primary-foreground hover:bg-primary/90': action.style === 'primary',
							'bg-destructive text-destructive-foreground hover:bg-destructive/90': action.style === 'danger',
							'bg-secondary text-secondary-foreground hover:bg-secondary/80': action.style !== 'primary' && action.style !== 'danger'
						}`),
						Span(g.Attr("x-html", "action.icon")),
						Span(g.Attr("x-text", "action.label")),
					),
				),
				// Edit button (owner/admin)
				button.Button(
					Div(
						lucide.Settings(Class("size-4")),
						g.Text("Settings"),
					),
					button.WithVariant("outline"),
					button.WithAttrs(
						g.Attr("x-show", "canManage"),
						g.Attr("@click", fmt.Sprintf("window.location.href='%s/organizations/%s/update'", appBase, orgID)),
					),
				),
				// Delete button (owner only)
				button.Button(
					Div(
						lucide.Trash2(Class("size-4")),
						g.Text("Delete"),
					),
					button.WithVariant("destructive"),
					button.WithAttrs(
						g.Attr("x-show", "canDelete"),
						g.Attr("@click", "showDeleteModal = true"),
					),
				),
			),
		),
	)
}

// organizationTabs renders the tab navigation for organization pages
func organizationTabs(appBase, orgID string) g.Node {
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Class("border-b border-border"),
		Nav(
			Class("flex space-x-8 overflow-x-auto"),
			g.Attr("aria-label", "Tabs"),
			// Overview tab (always active on this page)
			A(
				Href(baseURL),
				Class("inline-flex items-center gap-2 border-b-2 border-primary px-1 py-4 text-sm font-medium text-primary"),
				g.Attr("aria-current", "page"),
				lucide.LayoutDashboard(Class("size-4")),
				g.Text("Overview"),
			),
			// Members tab
			A(
				Href(baseURL+"/members"),
				Class("inline-flex items-center gap-2 border-b-2 border-transparent px-1 py-4 text-sm font-medium text-muted-foreground hover:text-foreground hover:border-border"),
				lucide.Users(Class("size-4")),
				g.Text("Members"),
			),
			// Teams tab
			A(
				Href(baseURL+"/teams"),
				Class("inline-flex items-center gap-2 border-b-2 border-transparent px-1 py-4 text-sm font-medium text-muted-foreground hover:text-foreground hover:border-border"),
				lucide.UsersRound(Class("size-4")),
				g.Text("Teams"),
			),
			// Roles tab
			A(
				Href(baseURL+"/roles"),
				Class("inline-flex items-center gap-2 border-b-2 border-transparent px-1 py-4 text-sm font-medium text-muted-foreground hover:text-foreground hover:border-border"),
				lucide.ShieldCheck(Class("size-4")),
				g.Text("Roles"),
			),
			// Invitations tab
			A(
				Href(baseURL+"/invitations"),
				Class("inline-flex items-center gap-2 border-b-2 border-transparent px-1 py-4 text-sm font-medium text-muted-foreground hover:text-foreground hover:border-border"),
				lucide.Mail(Class("size-4")),
				g.Text("Invitations"),
			),
			// Extension tabs (dynamic)
			Template(
				g.Attr("x-for", "tab in extensionData.tabs"),
				g.Attr(":key", "tab.id"),
				A(
					g.Attr(":href", fmt.Sprintf("`%s/tabs/${tab.path}`", baseURL)),
					Class("inline-flex items-center gap-2 border-b-2 border-transparent px-1 py-4 text-sm font-medium text-muted-foreground hover:text-foreground hover:border-border"),
					Span(g.Attr("x-html", "tab.icon")),
					Span(g.Attr("x-text", "tab.label")),
				),
			),
		),
	)
}

// quickLinksSection renders the quick access links grid
func quickLinksSection(appBase, orgID string) g.Node {
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Div(
			Class("mb-4"),
			H2(Class("text-lg font-semibold"), g.Text("Quick Links")),
			P(Class("text-sm text-muted-foreground"), g.Text("Navigate to different sections")),
		),
		Div(
			Class("grid gap-4 md:grid-cols-4"),
			// Default quick links
			QuickLinkCard(
				"Members",
				"Manage organization members",
				baseURL+"/members",
				lucide.Users(Class("size-6 text-primary")),
			),
			QuickLinkCard(
				"Teams",
				"Organize members into teams",
				baseURL+"/teams",
				lucide.UsersRound(Class("size-6 text-primary")),
			),
			QuickLinkCard(
				"Roles",
				"Manage roles & permissions",
				baseURL+"/roles",
				lucide.ShieldCheck(Class("size-6 text-primary")),
			),
			QuickLinkCard(
				"Invitations",
				"View pending invitations",
				baseURL+"/invitations",
				lucide.Mail(Class("size-6 text-primary")),
			),
			// Extension quick links (dynamic)
			Template(
				g.Attr("x-for", "link in extensionData.quickLinks"),
				g.Attr(":key", "link.id"),
				A(
					g.Attr(":href", "link.url"),
					Class("group"),
					Card(
						Class("transition-all hover:shadow-md hover:border-primary/50 p-4"),
						Div(
							Class("flex items-start gap-3"),
							Div(
								Class("rounded-lg bg-primary/10 p-3 group-hover:bg-primary/20 transition-colors"),
								Span(g.Attr("x-html", "link.icon")),
							),
							Div(
								Class("flex-1 min-w-0"),
								H3(
									Class("text-sm font-semibold group-hover:text-primary transition-colors"),
									g.Attr("x-text", "link.title"),
								),
								P(
									Class("mt-1 text-xs text-muted-foreground line-clamp-2"),
									g.Attr("x-text", "link.description"),
								),
							),
							lucide.ChevronRight(Class("size-5 text-muted-foreground transition-transform group-hover:translate-x-1")),
						),
					),
				),
			),
		),
	)
}

// extensionWidgetsSection renders the extension widgets
func extensionWidgetsSection() g.Node {
	return Div(
		g.Attr("x-show", "extensionData.widgets.length > 0"),
		Div(
			Class("mb-4"),
			H2(Class("text-lg font-semibold"), g.Text("Overview")),
			P(Class("text-sm text-muted-foreground"), g.Text("Additional information and metrics")),
		),
		Div(
			Class("grid gap-6 md:grid-cols-3"),
			Template(
				g.Attr("x-for", "widget in extensionData.widgets"),
				g.Attr(":key", "widget.id"),
				Div(
					g.Attr(":class", `{
						'md:col-span-1': widget.size === 1,
						'md:col-span-2': widget.size === 2,
						'md:col-span-3': widget.size >= 3
					}`),
					Card(
						Class("p-6"),
						Div(
							Class("flex items-center justify-between mb-4"),
							Div(
								Class("flex items-center gap-2"),
								Span(g.Attr("x-html", "widget.icon")),
								H3(
									Class("text-lg font-semibold"),
									g.Attr("x-text", "widget.title"),
								),
							),
						),
						Div(
							Class("text-muted-foreground"),
							Div(g.Attr("x-html", "widget.content")),
						),
					),
				),
			),
		),
	)
}

// deleteConfirmationModal renders the delete confirmation modal
func deleteConfirmationModal(appBase, orgID string) g.Node {
	return Div(
		g.Attr("x-show", "showDeleteModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", "showDeleteModal = false"),
		Card(
			Class("max-w-md w-full mx-4 p-6 space-y-4"),
			Div(
				Class("flex items-start gap-4"),
				Div(
					Class("rounded-full bg-destructive/10 p-3"),
					lucide.TriangleAlert(Class("size-6 text-destructive")),
				),
				Div(
					Class("flex-1"),
					H3(Class("text-lg font-semibold"), g.Text("Delete Organization")),
					P(
						Class("text-sm text-muted-foreground mt-2"),
						g.Text("Are you sure you want to delete "),
						Span(Class("font-medium"), g.Attr("x-text", "organization?.name")),
						g.Text("? This action cannot be undone. All members, teams, and data will be permanently deleted."),
					),
				),
			),
			Div(
				Class("flex justify-end gap-2"),
				button.Button(
					g.Text("Cancel"),
					button.WithVariant("outline"),
					button.WithAttrs(
						g.Attr("@click", "showDeleteModal = false"),
						g.Attr(":disabled", "deleting"),
					),
				),
				button.Button(
					Div(
						Span(
							g.Attr("x-show", "deleting"),
							Class("inline-flex items-center gap-2"),
							Div(Class("animate-spin rounded-full h-4 w-4 border-b-2 border-current")),
							g.Text("Deleting..."),
						),
						Span(
							g.Attr("x-show", "!deleting"),
							g.Text("Delete Organization"),
						),
					),
					button.WithVariant("destructive"),
					button.WithAttrs(
						g.Attr("@click", "deleteOrganization()"),
						g.Attr(":disabled", "deleting"),
					),
				),
			),
		),
	)
}
