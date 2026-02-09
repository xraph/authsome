package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/input"
)

// EditTeamPage renders the edit team form page.
func EditTeamPage(currentApp *app.App, orgID, teamID, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Class("space-y-6"),

		// Alpine.js data
		Div(
			g.Attr("x-data", editTeamData(currentApp.ID.String(), orgID, teamID)),
			g.Attr("x-init", "loadTeam()"),

			// Back link
			BackLink(baseURL+"/teams", "Back to Teams"),

			// Page header
			PageHeader(
				"Edit Team",
				"Update team details",
			),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				LoadingSpinner(),
			),

			// Error state
			ErrorMessage("error && !loading"),

			// Form
			Div(
				g.Attr("x-show", "!loading && !error"),
				g.Attr("x-cloak", ""),
				Card(
					Class("p-6"),
					Form(
						g.Attr("@submit.prevent", "updateTeam()"),
						Class("space-y-6"),

						// Team Name
						Div(
							Class("space-y-2"),
							Label(
								For("name"),
								Class("text-sm font-medium"),
								g.Text("Team Name"),
								Span(Class("text-destructive"), g.Text("*")),
							),
							input.Input(
								input.WithType("text"),
								input.WithID("name"),
								input.WithName("name"),
								input.WithPlaceholder("Engineering Team"),
								input.WithAttrs(
									Required(),
									g.Attr("x-model", "form.name"),
								),
							),
							P(Class("text-xs text-muted-foreground"), g.Text("The display name of the team")),
						),

						// Slug (read-only)
						Div(
							Class("space-y-2"),
							Label(
								For("slug"),
								Class("text-sm font-medium"),
								g.Text("Slug"),
							),
							input.Input(
								input.WithType("text"),
								input.WithID("slug"),
								input.WithName("slug"),
								input.WithAttrs(
									g.Attr("x-model", "form.slug"),
									g.Attr("disabled", ""),
								),
							),
							P(Class("text-xs text-muted-foreground"), g.Text("URL-friendly identifier (cannot be changed)")),
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
								Placeholder("Describe the team's purpose and responsibilities..."),
								Rows("4"),
								Class("flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"),
								g.Attr("x-model", "form.description"),
							),
							P(Class("text-xs text-muted-foreground"), g.Text("Brief description of the team")),
						),

						// Metadata
						Div(
							Class("space-y-2"),
							Label(
								For("metadata"),
								Class("text-sm font-medium"),
								g.Text("Metadata (JSON)"),
							),
							Textarea(
								ID("metadata"),
								Name("metadata"),
								Placeholder(`{"key": "value"}`),
								Rows("4"),
								Class("flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"),
								g.Attr("x-model", "form.metadataJson"),
							),
							P(Class("text-xs text-muted-foreground"), g.Text("Additional metadata in JSON format")),
						),

						// Actions
						Div(
							Class("flex justify-end gap-2 pt-4"),
							button.Button(
								g.Text("Cancel"),
								button.WithVariant("outline"),
								button.WithAttrs(
									Type("button"),
									g.Attr("onclick", fmt.Sprintf("window.location.href='%s/teams'", baseURL)),
									g.Attr(":disabled", "saving"),
								),
							),
							button.Button(
								Div(
									Span(
										g.Attr("x-show", "saving"),
										Class("inline-flex items-center gap-2"),
										Div(Class("animate-spin rounded-full h-4 w-4 border-b-2 border-current")),
										g.Text("Saving..."),
									),
									Span(
										g.Attr("x-show", "!saving"),
										Div(
											lucide.Save(Class("size-4")),
											g.Text("Save Changes"),
										),
									),
								),
								button.WithVariant("default"),
								button.WithAttrs(
									Type("submit"),
									g.Attr(":disabled", "saving"),
								),
							),
						),
					),
				),
			),
		),
	)
}

// editTeamData returns the Alpine.js data object for edit team.
func editTeamData(appID, orgID, teamID string) string {
	return fmt.Sprintf(`{
		team: null,
		form: {
			name: '',
			slug: '',
			description: '',
			metadataJson: ''
		},
		loading: true,
		error: null,
		saving: false,
		
		async loadTeam() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('organization.getTeams', {
					appId: '%s',
					orgId: '%s'
				});
				
				// Find the team by ID
				this.team = result.teams.find(t => t.id === '%s');
				if (!this.team) {
					throw new Error('Team not found');
				}
				
				this.form.name = this.team.name;
				this.form.slug = this.team.slug;
				this.form.description = this.team.description || '';
				this.form.metadataJson = this.team.metadata ? JSON.stringify(this.team.metadata, null, 2) : '';
			} catch (err) {
				console.error('Failed to load team:', err);
				this.error = err.message || 'Failed to load team';
			} finally {
				this.loading = false;
			}
		},
		
		async updateTeam() {
			this.saving = true;
			this.error = null;
			try {
				// Parse metadata JSON
				let metadata = null;
				if (this.form.metadataJson.trim()) {
					try {
						metadata = JSON.parse(this.form.metadataJson);
					} catch (e) {
						throw new Error('Invalid JSON in metadata field');
					}
				}
				
				await $bridge.call('organization.updateTeam', {
					appId: '%s',
					orgId: '%s',
					teamId: '%s',
					name: this.form.name,
					description: this.form.description,
					metadata: metadata
				});
				
				// Redirect back to teams list
				window.location.href = '/api/identity/ui/app/%s/organizations/%s/teams';
			} catch (err) {
				console.error('Failed to update team:', err);
				this.error = err.message || 'Failed to update team';
			} finally {
				this.saving = false;
			}
		}
	}`, appID, orgID, teamID, appID, orgID, teamID, appID, orgID)
}
