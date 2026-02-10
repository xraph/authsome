package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/input"
)

// CreateTeamPage renders the create team form page.
func CreateTeamPage(currentApp *app.App, orgID, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Class("space-y-6"),

		// Alpine.js data
		Div(
			g.Attr("x-data", createTeamData(currentApp.ID.String(), orgID)),

			// Back link
			BackLink(baseURL+"/teams", "Back to Teams"),

			// Page header
			PageHeader(
				"Create Team",
				"Create a new team within the organization",
			),

			// Error message
			ErrorMessage("error"),

			// Form
			Card(
				Class("p-6"),
				Form(
					g.Attr("@submit.prevent", "createTeam()"),
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

					// Slug
					Div(
						Class("space-y-2"),
						Label(
							For("slug"),
							Class("text-sm font-medium"),
							g.Text("Slug"),
							Span(Class("text-destructive"), g.Text("*")),
						),
						input.Input(
							input.WithType("text"),
							input.WithID("slug"),
							input.WithName("slug"),
							input.WithPlaceholder("engineering-team"),
							input.WithAttrs(
								Required(),
								g.Attr("pattern", "[a-z0-9-]+"),
								g.Attr("x-model", "form.slug"),
							),
						),
						P(Class("text-xs text-muted-foreground"), g.Text("URL-friendly identifier (lowercase letters, numbers, and hyphens only)")),
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
						P(Class("text-xs text-muted-foreground"), g.Text("Optional: Brief description of the team")),
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
						P(Class("text-xs text-muted-foreground"), g.Text("Optional: Additional metadata in JSON format")),
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
								g.Attr(":disabled", "creating"),
							),
						),
						button.Button(
							Div(
								Span(
									g.Attr("x-show", "creating"),
									Class("inline-flex items-center gap-2"),
									Div(Class("animate-spin rounded-full h-4 w-4 border-b-2 border-current")),
									g.Text("Creating..."),
								),
								Span(
									g.Attr("x-show", "!creating"),
									Div(
										lucide.Plus(Class("size-4")),
										g.Text("Create Team"),
									),
								),
							),
							button.WithVariant("default"),
							button.WithAttrs(
								Type("submit"),
								g.Attr(":disabled", "creating"),
							),
						),
					),
				),
			),
		),
	)
}

// createTeamData returns the Alpine.js data object for create team.
func createTeamData(appID, orgID string) string {
	return fmt.Sprintf(`{
		form: {
			name: '',
			slug: '',
			description: '',
			metadataJson: ''
		},
		error: null,
		creating: false,
		
		async createTeam() {
			this.creating = true;
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
				
				await $bridge.call('organization.createTeam', {
					appId: '%s',
					orgId: '%s',
					name: this.form.name,
					slug: this.form.slug,
					description: this.form.description,
					metadata: metadata
				});
				
				// Redirect back to teams list
				window.location.href = '/api/identity/ui/app/%s/organizations/%s/teams';
			} catch (err) {
				console.error('Failed to create team:', err);
				this.error = err.message || 'Failed to create team';
			} finally {
				this.creating = false;
			}
		}
	}`, appID, orgID, appID, orgID)
}
