package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
)

// TeamsPage renders the organization teams management page.
func TeamsPage(currentApp *app.App, orgID, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Class("space-y-6"),

		// Alpine.js data
		Div(
			g.Attr("x-data", teamsPageData(currentApp.ID.String(), orgID)),
			g.Attr("x-init", "loadTeams()"),

			// Back link
			BackLink(baseURL, "Back to Organization"),

			// Page header
			PageHeader(
				"Teams",
				"Organize members into teams for better collaboration",
				button.Button(
					Div(
						lucide.Plus(Class("size-4")),
						g.Text("Create Team"),
					),
					button.WithVariant("default"),
					button.WithAttrs(
						g.Attr("x-show", "canManage"),
						g.Attr("@click", "showCreateModal = true"),
					),
				),
			),

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

				// Search bar
				Div(
					Class("flex items-center gap-4"),
					SearchInput("Search teams...", "", ""),
				),

				// Teams grid
				teamsGrid(),

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
							g.Attr("@click", "filters.page--; loadTeams()"),
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
							g.Attr("@click", "filters.page++; loadTeams()"),
						),
					),
				),
			),

			// Create team modal
			createTeamModal(),

			// Edit team modal
			editTeamModal(),

			// Delete team confirmation
			deleteTeamModal(),
		),
	)
}

// teamsPageData returns the Alpine.js data object.
func teamsPageData(appID, orgID string) string {
	return fmt.Sprintf(`{
		teams: [],
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
		canManage: false,
		loading: true,
		error: null,
		showCreateModal: false,
		showEditModal: false,
		showDeleteModal: false,
		selectedTeam: null,
		teamForm: {
			name: '',
			description: '',
			submitting: false,
			error: null
		},
		
		async loadTeams() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('organization.getTeams', {
					appId: '%s',
					orgId: '%s',
					search: this.filters.search,
					page: this.filters.page,
					limit: this.filters.limit
				});
				
				this.teams = result.data || [];
				this.pagination = result.pagination || { currentPage: 1, pageSize: 20, totalItems: 0, totalPages: 0 };
				this.canManage = result.canManage || false;
			} catch (err) {
				console.error('Failed to load teams:', err);
				this.error = err.message || 'Failed to load teams';
			} finally {
				this.loading = false;
			}
		},
		
		async createTeam() {
			this.teamForm.submitting = true;
			this.teamForm.error = null;
			try {
				await $bridge.call('organization.createTeam', {
					appId: '%s',
					orgId: '%s',
					name: this.teamForm.name,
					description: this.teamForm.description
				});
				
				// Reset form and close modal
				this.teamForm.name = '';
				this.teamForm.description = '';
				this.showCreateModal = false;
				
				// Reload teams
				await this.loadTeams();
			} catch (err) {
				console.error('Failed to create team:', err);
				this.teamForm.error = err.message || 'Failed to create team';
			} finally {
				this.teamForm.submitting = false;
			}
		},
		
		editTeam(team) {
			this.selectedTeam = team;
			this.teamForm.name = team.name;
			this.teamForm.description = team.description || '';
			this.showEditModal = true;
		},
		
		async updateTeam() {
			if (!this.selectedTeam) return;
			
			this.teamForm.submitting = true;
			this.teamForm.error = null;
			try {
				await $bridge.call('organization.updateTeam', {
					appId: '%s',
					orgId: '%s',
					teamId: this.selectedTeam.id,
					name: this.teamForm.name,
					description: this.teamForm.description
				});
				
				// Reset form and close modal
				this.teamForm.name = '';
				this.teamForm.description = '';
				this.selectedTeam = null;
				this.showEditModal = false;
				
				// Reload teams
				await this.loadTeams();
			} catch (err) {
				console.error('Failed to update team:', err);
				this.teamForm.error = err.message || 'Failed to update team';
			} finally {
				this.teamForm.submitting = false;
			}
		},
		
		confirmDeleteTeam(team) {
			this.selectedTeam = team;
			this.showDeleteModal = true;
		},
		
		async deleteTeam() {
			if (!this.selectedTeam) return;
			
			try {
				await $bridge.call('organization.deleteTeam', {
					appId: '%s',
					orgId: '%s',
					teamId: this.selectedTeam.id
				});
				
				this.showDeleteModal = false;
				this.selectedTeam = null;
				
				// Reload teams
				await this.loadTeams();
			} catch (err) {
				console.error('Failed to delete team:', err);
				alert('Failed to delete team: ' + (err.message || 'Unknown error'));
			}
		},
		
		formatDate(dateStr) {
			if (!dateStr) return 'N/A';
			const date = new Date(dateStr);
			return date.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
		}
	}`, appID, orgID, appID, orgID, appID, orgID, appID, orgID)
}

// teamsGrid renders teams as a grid of cards.
func teamsGrid() g.Node {
	return Div(
		// Empty state
		Div(
			g.Attr("x-show", "teams.length === 0"),
			Class("text-center py-12"),
			lucide.UsersRound(Class("size-12 mx-auto mb-4 opacity-50 text-muted-foreground")),
			P(Class("text-lg font-medium"), g.Text("No teams found")),
			P(Class("text-sm text-muted-foreground"), g.Text("Create your first team to get started")),
		),
		// Teams grid
		Div(
			g.Attr("x-show", "teams.length > 0"),
			Class("grid gap-4 md:grid-cols-2 lg:grid-cols-3"),
			Template(
				g.Attr("x-for", "team in teams"),
				g.Attr(":key", "team.id"),
				card.Card(
					Class("hover:shadow-md transition-shadow"),
					card.Header(
						Div(
							Class("flex items-start justify-between"),
							Div(
								Class("flex-1"),
								card.Title("", card.WithAttrs(g.Attr("x-text", "team.name"))),
								P(
									Class("text-sm text-muted-foreground mt-1"),
									g.Attr("x-text", "team.description || 'No description'"),
								),
							),
						),
					),
					card.Content(
						Div(
							Class("space-y-3"),
							Div(
								Class("flex items-center gap-2 text-sm"),
								lucide.Users(Class("size-4 text-muted-foreground")),
								Span(g.Attr("x-text", "`${team.memberCount} member${team.memberCount !== 1 ? 's' : ''}`")),
							),
							Div(
								Class("flex items-center gap-2 text-sm text-muted-foreground"),
								lucide.Calendar(Class("size-4")),
								Span(g.Attr("x-text", "`Created ${formatDate(team.createdAt)}`")),
							),
						),
					),
					card.Footer(
						Class("flex justify-end gap-2"),
						button.Button(
							Div(
								lucide.Pencil(Class("size-3")),
								g.Text("Edit"),
							),
							button.WithVariant("outline"),
							button.WithSize("sm"),
							button.WithAttrs(
								g.Attr("x-show", "canManage"),
								g.Attr("@click", "editTeam(team)"),
							),
						),
						button.Button(
							Div(
								lucide.Trash2(Class("size-3")),
								g.Text("Delete"),
							),
							button.WithVariant("destructive"),
							button.WithSize("sm"),
							button.WithAttrs(
								g.Attr("x-show", "canManage"),
								g.Attr("@click", "confirmDeleteTeam(team)"),
							),
						),
					),
				),
			),
		),
	)
}

// createTeamModal renders the create team modal.
func createTeamModal() g.Node {
	return Div(
		g.Attr("x-show", "showCreateModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", "showCreateModal = false"),
		Div(
			Class("max-w-md w-full mx-4 rounded-lg border bg-card shadow-lg"),
			Form(
				g.Attr("@submit.prevent", "createTeam()"),
				Div(
					Class("p-6 space-y-4"),
					H3(Class("text-lg font-semibold"), g.Text("Create Team")),
					P(Class("text-sm text-muted-foreground"), g.Text("Create a new team to organize members")),

					// Error message
					Div(
						g.Attr("x-show", "teamForm.error"),
						Class("bg-destructive/10 border border-destructive/20 rounded-lg p-3"),
						P(Class("text-sm text-destructive"), g.Attr("x-text", "teamForm.error")),
					),

					// Name field
					FormField("team-name", "Team Name", "text", "name", "Engineering Team", true, ""),

					// Description field
					TextareaField("team-description", "Description", "description", "Brief description of the team", 3, false, ""),

					// Actions
					Div(
						Class("flex justify-end gap-2 pt-4"),
						button.Button(
							g.Text("Cancel"),
							button.WithVariant("outline"),
							button.WithType("button"),
							button.WithAttrs(
								g.Attr("@click", "showCreateModal = false"),
								g.Attr(":disabled", "teamForm.submitting"),
							),
						),
						button.Button(
							Div(
								Span(
									g.Attr("x-show", "teamForm.submitting"),
									Class("inline-flex items-center gap-2"),
									Div(Class("animate-spin rounded-full h-4 w-4 border-b-2 border-current")),
									g.Text("Creating..."),
								),
								Span(
									g.Attr("x-show", "!teamForm.submitting"),
									g.Text("Create Team"),
								),
							),
							button.WithVariant("default"),
							button.WithType("submit"),
							button.WithAttrs(
								g.Attr(":disabled", "teamForm.submitting"),
							),
						),
					),
				),
			),
		),
	)
}

// editTeamModal renders the edit team modal.
func editTeamModal() g.Node {
	return Div(
		g.Attr("x-show", "showEditModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", "showEditModal = false"),
		Div(
			Class("max-w-md w-full mx-4 rounded-lg border bg-card shadow-lg"),
			Form(
				g.Attr("@submit.prevent", "updateTeam()"),
				Div(
					Class("p-6 space-y-4"),
					H3(Class("text-lg font-semibold"), g.Text("Edit Team")),

					// Error message
					Div(
						g.Attr("x-show", "teamForm.error"),
						Class("bg-destructive/10 border border-destructive/20 rounded-lg p-3"),
						P(Class("text-sm text-destructive"), g.Attr("x-text", "teamForm.error")),
					),

					// Name field with x-model
					Div(
						Class("space-y-2"),
						Label(For("edit-team-name"), Class("text-sm font-medium"), g.Text("Team Name")),
						Input(
							Type("text"),
							ID("edit-team-name"),
							g.Attr("x-model", "teamForm.name"),
							Required(),
							Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
						),
					),

					// Description field with x-model
					Div(
						Class("space-y-2"),
						Label(For("edit-team-description"), Class("text-sm font-medium"), g.Text("Description")),
						Textarea(
							ID("edit-team-description"),
							g.Attr("x-model", "teamForm.description"),
							Rows("3"),
							Class("flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
						),
					),

					// Actions
					Div(
						Class("flex justify-end gap-2 pt-4"),
						button.Button(
							g.Text("Cancel"),
							button.WithVariant("outline"),
							button.WithType("button"),
							button.WithAttrs(
								g.Attr("@click", "showEditModal = false"),
								g.Attr(":disabled", "teamForm.submitting"),
							),
						),
						button.Button(
							Div(
								Span(
									g.Attr("x-show", "teamForm.submitting"),
									Class("inline-flex items-center gap-2"),
									Div(Class("animate-spin rounded-full h-4 w-4 border-b-2 border-current")),
									g.Text("Updating..."),
								),
								Span(
									g.Attr("x-show", "!teamForm.submitting"),
									g.Text("Update Team"),
								),
							),
							button.WithVariant("default"),
							button.WithType("submit"),
							button.WithAttrs(
								g.Attr(":disabled", "teamForm.submitting"),
							),
						),
					),
				),
			),
		),
	)
}

// deleteTeamModal renders the delete team confirmation modal.
func deleteTeamModal() g.Node {
	return Div(
		g.Attr("x-show", "showDeleteModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", "showDeleteModal = false"),
		Div(
			Class("max-w-md w-full mx-4 rounded-lg border bg-card shadow-lg"),
			Div(
				Class("p-6 space-y-4"),
				Div(
					Class("flex items-start gap-4"),
					Div(
						Class("rounded-full bg-destructive/10 p-3"),
						lucide.Trash2(Class("size-6 text-destructive")),
					),
					Div(
						Class("flex-1"),
						H3(Class("text-lg font-semibold"), g.Text("Delete Team")),
						P(
							Class("text-sm text-muted-foreground mt-2"),
							g.Text("Are you sure you want to delete "),
							Span(Class("font-medium"), g.Attr("x-text", "selectedTeam?.name")),
							g.Text("? This action cannot be undone."),
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
						),
					),
					button.Button(
						g.Text("Delete Team"),
						button.WithVariant("destructive"),
						button.WithAttrs(
							g.Attr("@click", "deleteTeam()"),
						),
					),
				),
			),
		),
	)
}
