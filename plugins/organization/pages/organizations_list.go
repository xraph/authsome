package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/table"
)

// OrganizationsListPage renders the organizations list page with dynamic data loading
func OrganizationsListPage(currentApp *app.App, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())

	return Div(
		Class("space-y-6"),

		// Page header
		PageHeader(
			"Organizations",
			"Manage user organizations and their members",
			PrimaryButton(
				appBase+"/organizations/create",
				"Create Organization",
				lucide.Plus(Class("size-4")),
			),
		),

		// Dynamic content with Alpine.js
		Div(
			g.Attr("x-data", organizationsListData(currentApp.ID.String())),
			g.Attr("x-init", "loadOrganizations()"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				LoadingSpinner(),
			),

			// Error state
			ErrorMessage("error && !loading"),

			// Content
			Div(
				g.Attr("x-show", "!loading && !error"),
				g.Attr("x-cloak", ""),
				Class("space-y-6"),

				// Stats cards
				Div(
					Class("grid gap-6 md:grid-cols-3"),
					StatsCard("Total Organizations", "stats.totalOrganizations", "violet"),
					StatsCard("Total Members", "stats.totalMembers", "emerald"),
					StatsCard("Total Teams", "stats.totalTeams", "blue"),
				),

				// Search bar
				Div(
					Class("flex items-center gap-4"),
					SearchInput("Search organizations...", "", ""),
				),

				// Organizations table
				organizationsTable(appBase),

				// Pagination
				Div(
					g.Attr("x-show", "pagination.totalPages > 1"),
					Class("flex items-center justify-center gap-2 mt-6"),
					button.Button(
						Div(
							lucide.ChevronLeft(Class("size-4")),
							g.Text("Previous"),
						),
						button.WithVariant("outline"),
						button.WithSize("sm"),
						button.WithAttrs(
							g.Attr("x-show", "pagination.currentPage > 1"),
							g.Attr("@click", "filters.page--; loadOrganizations()"),
						),
					),
					Span(
						Class("text-sm text-muted-foreground"),
						g.Attr("x-text", "`Page ${pagination.currentPage} of ${pagination.totalPages}`"),
					),
					button.Button(
						Div(
							g.Text("Next"),
							lucide.ChevronRight(Class("size-4")),
						),
						button.WithVariant("outline"),
						button.WithSize("sm"),
						button.WithAttrs(
							g.Attr("x-show", "pagination.currentPage < pagination.totalPages"),
							g.Attr("@click", "filters.page++; loadOrganizations()"),
						),
					),
				),
			),
		),
	)
}

// organizationsListData returns the Alpine.js data object for organizations list
func organizationsListData(appID string) string {
	return fmt.Sprintf(`{
		organizations: [],
		stats: {
			totalOrganizations: 0,
			totalMembers: 0,
			totalTeams: 0
		},
		pagination: {
			currentPage: 1,
			pageSize: 20,
			totalItems: 0,
			totalPages: 0
		},
		filters: {
			search: '',
			page: 1,
			limit: 20
		},
		loading: true,
		error: null,
		
		async loadOrganizations() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('organization.getOrganizations', {
					appId: '%s',
					search: this.filters.search,
					page: this.filters.page,
					limit: this.filters.limit
				});
				
				this.organizations = result.data || [];
				this.stats = result.stats || { totalOrganizations: 0, totalMembers: 0, totalTeams: 0 };
				this.pagination = result.pagination || { currentPage: 1, pageSize: 20, totalItems: 0, totalPages: 0 };
			} catch (err) {
				console.error('Failed to load organizations:', err);
				this.error = err.message || 'Failed to load organizations';
			} finally {
				this.loading = false;
			}
		},
		
		formatDate(dateStr) {
			if (!dateStr) return 'N/A';
			const date = new Date(dateStr);
			return date.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
		}
	}`, appID)
}

// organizationsTable renders the organizations table with Alpine.js templating using ForgeUI
func organizationsTable(appBase string) g.Node {
	return table.Table()(
		table.TableHeader()(
			table.TableRow()(
				table.TableHeaderCell()(g.Text("Organization")),
				table.TableHeaderCell()(g.Text("Members")),
				table.TableHeaderCell()(g.Text("Teams")),
				table.TableHeaderCell()(g.Text("Your Role")),
				table.TableHeaderCell()(g.Text("Created")),
				table.TableHeaderCell(table.WithAlign(table.AlignRight))(g.Text("Actions")),
			),
		),
		table.TableBody()(
			// Empty state
			Tr(
				g.Attr("x-show", "organizations.length === 0"),
				Td(
					g.Attr("colspan", "6"),
					Class("text-center py-8"),
					EmptyState(
						lucide.Building2(Class("size-12 mx-auto mb-2 opacity-50")),
						"No organizations found",
						"Create your first organization to get started",
					),
				),
			),
			// Template for each organization
			Template(
				g.Attr("x-for", "org in organizations"),
				g.Attr(":key", "org.id"),
				table.TableRow()(
					table.TableCell()(
						Div(
							Class("flex items-center gap-3"),
							// Logo
							Div(
								g.Attr("x-show", "org.logo"),
								Img(
									g.Attr(":src", "org.logo"),
									g.Attr(":alt", "org.name"),
									Class("size-10 rounded-lg object-cover"),
								),
							),
							Div(
								g.Attr("x-show", "!org.logo"),
								Class("size-10 rounded-lg bg-primary/10 flex items-center justify-center"),
								lucide.Building2(Class("size-5 text-primary")),
							),
							// Name and slug
							Div(
								A(
									g.Attr(":href", fmt.Sprintf("`%s/organizations/${org.id}`", appBase)),
									Class("font-medium hover:underline"),
									g.Attr("x-text", "org.name"),
								),
								P(
									Class("text-sm text-muted-foreground"),
									g.Text("@"),
									Span(g.Attr("x-text", "org.slug")),
								),
							),
						),
					),
					table.TableCell()(
						Span(g.Attr("x-text", "org.memberCount")),
					),
					table.TableCell()(
						Span(g.Attr("x-text", "org.teamCount")),
					),
					table.TableCell()(
						Span(
							Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize"),
							g.Attr(":class", `{
								'bg-primary text-primary-foreground': org.userRole === 'owner',
								'bg-secondary text-secondary-foreground': org.userRole === 'admin',
								'bg-muted text-muted-foreground': org.userRole === 'member'
							}`),
							g.Attr("x-text", "org.userRole"),
						),
					),
					table.TableCell()(
						Span(g.Attr("x-text", "formatDate(org.createdAt)")),
					),
					table.TableCell(table.WithAlign(table.AlignRight))(
						A(
							g.Attr(":href", fmt.Sprintf("`%s/organizations/${org.id}`", appBase)),
							Class("inline-flex items-center gap-1 text-sm text-primary hover:underline"),
							g.Text("View"),
							lucide.ArrowRight(Class("size-3")),
						),
					),
				),
			),
		),
	)
}
