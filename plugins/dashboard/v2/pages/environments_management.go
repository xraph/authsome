package pages

import (
	"github.com/xraph/forgeui/components/badge"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// EnvironmentsManagementPage shows environment management (if multiapp plugin is enabled)
func (p *PagesManager) EnvironmentsManagementPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				environments: [],
				currentEnv: null,
				loading: true,
				error: '',
				showNewEnvForm: false,
				creating: false,
				newEnv: {
					name: '',
					slug: '',
					type: 'development',
					description: ''
				},
				async loadEnvironments() {
					this.loading = true;
					this.error = '';
					try {
						const result = await $go('getEnvironmentsList', {
							appId: '`+appID+`'
						});
						this.environments = result.environments || [];
						this.currentEnv = result.current || null;
					} catch (err) {
						console.error('Failed to load environments:', err);
						this.error = err.message || 'Failed to load environments';
					} finally {
						this.loading = false;
					}
				},
				async switchEnvironment(envId) {
					try {
						const result = await $go('switchEnvironment', {
							appId: '`+appID+`',
							envId
						});
						if (result.message) {
							alert(result.message);
						}
						await this.loadEnvironments();
					} catch (err) {
						alert(err.message || 'Failed to switch environment');
					}
				},
				async createEnvironment() {
					if (!this.newEnv.name.trim()) {
						alert('Environment name is required');
						return;
					}
					this.creating = true;
					try {
						const result = await $go('createEnvironment', {
							appId: '`+appID+`',
							name: this.newEnv.name,
							slug: this.newEnv.slug || this.newEnv.name.toLowerCase().replace(/\s+/g, '-'),
							type: this.newEnv.type,
							description: this.newEnv.description
						});
						if (result.message) {
							alert(result.message);
						}
						this.showNewEnvForm = false;
						this.newEnv = { name: '', slug: '', type: 'development', description: '' };
						await this.loadEnvironments();
					} catch (err) {
						alert(err.message || 'Failed to create environment');
					} finally {
						this.creating = false;
					}
				},
				closeForm() {
					this.showNewEnvForm = false;
					this.newEnv = { name: '', slug: '', type: 'development', description: '' };
				},
				async cloneFromEnv(envId) {
					try {
						const result = await $go('getEnvironmentDetail', {
							appId: '`+appID+`',
							envId: envId
						});
						if (result) {
							this.newEnv = {
								name: (result.name || '') + ' (Copy)',
								slug: (result.slug || '') + '-copy',
								type: result.type || 'development',
								description: ''
							};
							this.showNewEnvForm = true;
						}
					} catch (err) {
						console.error('Failed to load source environment:', err);
						alert('Failed to load environment for cloning: ' + (err.message || 'Unknown error'));
					}
				}
			}`),
			g.Attr("x-init", `
				loadEnvironments();
				const urlParams = new URLSearchParams(window.location.search);
				const cloneId = urlParams.get('clone');
				if (cloneId) {
					cloneFromEnv(cloneId);
					window.history.replaceState({}, '', window.location.pathname);
				}
			`),

			// Header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold"), g.Text("Environments")),
					P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Manage application environments")),
				),
				button.Button(
					Div(
						Class("flex items-center gap-2"),
						icons.Plus(icons.WithSize(16)),
						Span(g.Text("New Environment")),
					),
					button.WithAttrs(
						g.Attr("@click", "showNewEnvForm = true"),
					),
				),
			),

			// New Environment Form Modal
			Div(
				g.Attr("x-show", "showNewEnvForm"),
				g.Attr("x-cloak", ""),
				Class("fixed inset-0 z-50 overflow-y-auto"),
				g.Attr("aria-labelledby", "modal-title"),
				g.Attr("role", "dialog"),
				g.Attr("aria-modal", "true"),

				// Backdrop
				Div(
					Class("fixed inset-0 bg-black/50 transition-opacity"),
					g.Attr("@click", "closeForm()"),
				),

				// Modal panel
				Div(
					Class("flex min-h-full items-center justify-center p-4"),
					Div(
						Class("relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6"),
						g.Attr("@click.stop", ""),

						// Header
						Div(
							Class("flex items-center justify-between mb-4"),
							H3(
								Class("text-lg font-semibold text-gray-900 dark:text-gray-100"),
								g.Attr("id", "modal-title"),
								g.Text("Create New Environment"),
							),
							button.Button(
								icons.X(icons.WithSize(20)),
								button.WithVariant("ghost"),
								button.WithSize("sm"),
								button.WithAttrs(g.Attr("@click", "closeForm()")),
							),
						),

						// Form
						Form(
							g.Attr("@submit.prevent", "createEnvironment()"),
							Class("space-y-4"),

							// Name field
							Div(
								Label(
									Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
									g.Attr("for", "env-name"),
									g.Text("Name"),
								),
								Input(
									Type("text"),
									g.Attr("id", "env-name"),
									g.Attr("x-model", "newEnv.name"),
									Class("w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"),
									g.Attr("placeholder", "e.g., Staging"),
									g.Attr("required", ""),
								),
							),

							// Slug field
							Div(
								Label(
									Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
									g.Attr("for", "env-slug"),
									g.Text("Slug (optional)"),
								),
								Input(
									Type("text"),
									g.Attr("id", "env-slug"),
									g.Attr("x-model", "newEnv.slug"),
									Class("w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"),
									g.Attr("placeholder", "e.g., staging (auto-generated if empty)"),
								),
							),

							// Type field
							Div(
								Label(
									Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
									g.Attr("for", "env-type"),
									g.Text("Type"),
								),
								Select(
									g.Attr("id", "env-type"),
									g.Attr("x-model", "newEnv.type"),
									Class("w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"),
									Option(Value("development"), g.Text("Development")),
									Option(Value("staging"), g.Text("Staging")),
									Option(Value("production"), g.Text("Production")),
									Option(Value("preview"), g.Text("Preview")),
									Option(Value("test"), g.Text("Test")),
								),
							),

							// Description field
							Div(
								Label(
									Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
									g.Attr("for", "env-description"),
									g.Text("Description (optional)"),
								),
								Textarea(
									g.Attr("id", "env-description"),
									g.Attr("x-model", "newEnv.description"),
									Class("w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"),
									g.Attr("rows", "3"),
									g.Attr("placeholder", "Describe this environment..."),
								),
							),

							// Actions
							Div(
								Class("flex justify-end gap-3 pt-4"),
								button.Button(
									g.Text("Cancel"),
									button.WithVariant("outline"),
									button.WithAttrs(
										g.Attr("type", "button"),
										g.Attr("@click", "closeForm()"),
									),
								),
								button.Button(
									g.Group([]g.Node{
										Span(
											g.Attr("x-show", "!creating"),
											g.Text("Create Environment"),
										),
										Span(
											g.Attr("x-show", "creating"),
											g.Text("Creating..."),
										),
									}),
									button.WithAttrs(
										g.Attr("type", "submit"),
										g.Attr(":disabled", "creating"),
									),
								),
							),
						),
					),
				),
			),

			// Environments List
			Div(

				// Loading state
				Div(
					g.Attr("x-show", "loading"),
					Class("flex items-center justify-center py-12"),
					Div(
						Class("animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"),
					),
				),

				// Error state
				Div(
					g.Attr("x-show", "!loading && error"),
					Class("p-4 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400"),
					P(g.Attr("x-text", "error")),
				),

				// Empty state
				Div(
					g.Attr("x-show", "!loading && !error && environments.length === 0"),
					Class("text-center py-12"),
					Div(
						Class("text-gray-400 dark:text-gray-600 mb-4"),
						icons.Database(icons.WithSize(48)),
					),
					H3(Class("text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2"), g.Text("No environments found")),
					P(Class("text-gray-600 dark:text-gray-400"), g.Text("Create your first environment to get started")),
				),

				// Environments grid
				Div(
					g.Attr("x-show", "!loading && !error && environments.length > 0"),
					Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
					g.El("template", g.Attr("x-for", "env in environments"),
						card.Card(
							card.Header(
								Div(
									Class("flex items-center justify-between"),
									card.Title("", card.WithAttrs(g.Attr("x-text", "env.name"))),
									badge.Badge(
										"",
										badge.WithAttrs(
											g.Attr("x-show", "currentEnv && env.id === currentEnv.id"),
											g.Attr("x-text", "'Active'"),
										),
									),
								),
								card.Description("", card.WithAttrs(g.Attr("x-text", "env.description || 'No description'"))),
							),
							card.Content(
								Div(
									Class("space-y-2 text-sm"),
									Div(
										Span(Class("font-semibold"), g.Text("Type: ")),
										Span(g.Attr("x-text", "env.type || 'standard'")),
									),
									Div(
										Span(Class("font-semibold"), g.Text("Created: ")),
										Span(g.Attr("x-text", "new Date(env.createdAt).toLocaleDateString()")),
									),
								),
							),
							card.Footer(
								Div(
									Class("flex gap-2"),
									button.Button(
										g.Text("Switch"),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(
											g.Attr("@click", "switchEnvironment(env.id)"),
											g.Attr(":disabled", "currentEnv && env.id === currentEnv.id"),
										),
									),
									button.Button(
										g.Text("View Details"),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(
											g.Attr("@click", "window.location.href = '/api/identity/ui/app/"+appID+"/environments/' + env.id"),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	), nil
}
