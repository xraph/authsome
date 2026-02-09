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

// UpdateOrganizationPage renders the update organization form page.
func UpdateOrganizationPage(currentApp *app.App, orgID, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Class("space-y-6"),

		// Alpine.js data
		Div(
			g.Attr("x-data", updateOrganizationData(currentApp.ID.String(), orgID)),
			g.Attr("x-init", "loadOrganization()"),

			// Back link
			BackLink(baseURL, "Back to Organization"),

			// Page header
			PageHeader(
				"Organization Settings",
				"Update your organization details",
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
						g.Attr("@submit.prevent", "updateOrganization()"),
						Class("space-y-6"),

						// Organization Name
						Div(
							Class("space-y-2"),
							Label(
								For("name"),
								Class("text-sm font-medium"),
								g.Text("Organization Name"),
								Span(Class("text-destructive"), g.Text("*")),
							),
							input.Input(
								input.WithType("text"),
								input.WithID("name"),
								input.WithName("name"),
								input.WithPlaceholder("Acme Inc."),
								input.WithAttrs(
									Required(),
									g.Attr("x-model", "form.name"),
								),
							),
							P(Class("text-xs text-muted-foreground"), g.Text("The display name of your organization")),
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

						// Logo URL
						Div(
							Class("space-y-2"),
							Label(
								For("logo"),
								Class("text-sm font-medium"),
								g.Text("Logo URL"),
							),
							input.Input(
								input.WithType("url"),
								input.WithID("logo"),
								input.WithName("logo"),
								input.WithPlaceholder("https://example.com/logo.png"),
								input.WithAttrs(
									g.Attr("x-model", "form.logo"),
								),
							),
							P(Class("text-xs text-muted-foreground"), g.Text("URL to your organization's logo")),
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
									g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", baseURL)),
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

// updateOrganizationData returns the Alpine.js data object for update organization.
func updateOrganizationData(appID, orgID string) string {
	return fmt.Sprintf(`{
		organization: null,
		form: {
			name: '',
			slug: '',
			logo: '',
			metadataJson: ''
		},
		loading: true,
		error: null,
		saving: false,
		
		async loadOrganization() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('organization.getOrganization', {
					appId: '%s',
					orgId: '%s'
				});
				
				this.organization = result.organization;
				this.form.name = result.organization.name;
				this.form.slug = result.organization.slug;
				this.form.logo = result.organization.logo || '';
				this.form.metadataJson = result.organization.metadata ? JSON.stringify(result.organization.metadata, null, 2) : '';
			} catch (err) {
				console.error('Failed to load organization:', err);
				this.error = err.message || 'Failed to load organization';
			} finally {
				this.loading = false;
			}
		},
		
		async updateOrganization() {
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
				
				await $bridge.call('organization.updateOrganization', {
					appId: '%s',
					orgId: '%s',
					name: this.form.name,
					logo: this.form.logo,
					metadata: metadata
				});
				
				// Redirect back to organization detail
				window.location.href = '/api/identity/ui/app/%s/organizations/%s';
			} catch (err) {
				console.error('Failed to update organization:', err);
				this.error = err.message || 'Failed to update organization';
			} finally {
				this.saving = false;
			}
		}
	}`, appID, orgID, appID, orgID, appID, orgID)
}
