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

// EnvironmentDetailPage shows detailed information about an environment
func (p *PagesManager) EnvironmentDetailPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")
	envID := ctx.Param("envId")

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				environment: null,
				loading: true,
				error: '',
				showDeleteConfirm: false,
				showPromoteModal: false,
				deleting: false,
				promoting: false,
				switching: false,
				allEnvironments: [],
				selectedTargetEnv: '',
				async loadEnvironment() {
					this.loading = true;
					this.error = '';
					try {
						const result = await $go('getEnvironmentDetail', {
							appId: '`+appID+`',
							envId: '`+envID+`'
						});
						this.environment = result;
						// Also load all environments for promote modal
						await this.loadAllEnvironments();
					} catch (err) {
						console.error('Failed to load environment:', err);
						this.error = err.message || 'Failed to load environment';
					} finally {
						this.loading = false;
					}
				},
				async loadAllEnvironments() {
					try {
						const result = await $go('getEnvironmentsList', {
							appId: '`+appID+`'
						});
						this.allEnvironments = (result.environments || []).filter(e => e.id !== '`+envID+`');
					} catch (err) {
						console.error('Failed to load environments:', err);
					}
				},
				async switchToEnvironment() {
					this.switching = true;
					try {
						const result = await $go('switchEnvironment', {
							appId: '`+appID+`',
							envId: this.environment.id
						});
						if (result.message) {
							alert(result.message);
						}
					} catch (err) {
						alert(err.message || 'Failed to switch environment');
					} finally {
						this.switching = false;
					}
				},
				async deleteEnvironment() {
					this.deleting = true;
					try {
						const result = await $go('deleteEnvironment', {
							appId: '`+appID+`',
							envId: this.environment.id
						});
						if (result.message) {
							alert(result.message);
						}
						window.location.href = '`+p.baseUIPath+`/app/`+appID+`/environments';
					} catch (err) {
						alert(err.message || 'Failed to delete environment');
						this.deleting = false;
					}
				},
				async promoteEnvironment() {
					if (!this.selectedTargetEnv) {
						alert('Please select a target environment');
						return;
					}
					this.promoting = true;
					try {
						const result = await $go('promoteEnvironment', {
							appId: '`+appID+`',
							sourceEnvId: this.environment.id,
							targetEnvId: this.selectedTargetEnv
						});
						if (result.message) {
							alert(result.message);
						}
						this.showPromoteModal = false;
						this.selectedTargetEnv = '';
						await this.loadEnvironment();
					} catch (err) {
						alert(err.message || 'Failed to promote environment');
					} finally {
						this.promoting = false;
					}
				},
				cloneEnvironment() {
					// Navigate to create page - the user can fill in the form with similar settings
					window.location.href = '`+p.baseUIPath+`/app/`+appID+`/environments?clone=' + this.environment.id;
				},
				formatJSON(obj) {
					if (!obj) return '{}';
					return JSON.stringify(obj, null, 2);
				},
				getTypeBadgeVariant(type) {
					const variants = {
						production: 'destructive',
						staging: 'default',
						development: 'secondary',
						preview: 'outline',
						test: 'outline'
					};
					return variants[type] || 'default';
				},
				getStatusBadgeVariant(status) {
					const variants = {
						active: 'default',
						inactive: 'secondary',
						maintenance: 'outline'
					};
					return variants[status] || 'default';
				}
			}`),
			g.Attr("x-init", "loadEnvironment()"),

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
				Class("space-y-4"),
				card.Card(
					card.Header(
						Div(
							Class("flex items-center gap-3 text-red-600 dark:text-red-400"),
							icons.AlertCircle(icons.WithSize(24)),
							card.Title("Environment Not Found"),
						),
						card.Description("", card.WithAttrs(g.Attr("x-text", "error"))),
					),
					card.Footer(
						Div(
							Class("flex gap-2"),
							button.Button(
								g.Text("Back to Environments"),
								button.WithVariant("outline"),
								button.WithAttrs(
									g.Attr("@click", "window.location.href = '"+p.baseUIPath+"/app/"+appID+"/environments'"),
								),
							),
							button.Button(
								g.Text("Retry"),
								button.WithAttrs(
									g.Attr("@click", "loadEnvironment()"),
								),
							),
						),
					),
				),
			),

			// Environment detail content - wrapped in template with x-if to prevent evaluation when null
			g.El("template",
				g.Attr("x-if", "!loading && !error && environment"),
				Div(
					Class("space-y-6"),

					// Header with actions
					Div(
						Class("flex items-start justify-between"),
						Div(
							Class("space-y-2"),
							// Back button
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									icons.ArrowLeft(icons.WithSize(16)),
									Span(g.Text("Back to Environments")),
								),
								button.WithVariant("ghost"),
								button.WithSize("sm"),
								button.WithAttrs(
									g.Attr("@click", "window.location.href = '"+p.baseUIPath+"/app/"+appID+"/environments'"),
								),
							),
							// Environment name and badges
							Div(
								Class("flex items-center gap-3"),
								H1(Class("text-3xl font-bold"), g.Attr("x-text", "environment.name")),
								Div(
									Class("flex items-center gap-2"),
									badge.Badge(
										"",
										badge.WithAttrs(
											g.Attr("x-text", "environment.type"),
											g.Attr(":class", `{
											'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200': environment.type === 'production',
											'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200': environment.type === 'staging',
											'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200': environment.type === 'development',
											'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200': environment.type === 'preview',
											'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200': environment.type === 'test'
										}`),
										),
									),
									badge.Badge(
										"",
										badge.WithAttrs(
											g.Attr("x-text", "environment.status"),
											g.Attr(":class", `{
											'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200': environment.status === 'active',
											'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200': environment.status === 'inactive',
											'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200': environment.status === 'maintenance'
										}`),
										),
									),
									badge.Badge(
										"Default",
										badge.WithAttrs(
											g.Attr("x-show", "environment.isDefault"),
											g.Attr("class", "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200"),
										),
									),
								),
							),
							P(Class("text-gray-600 dark:text-gray-400"), g.Attr("x-text", "`Slug: ${environment.slug}`")),
						),
						// Action buttons
						Div(
							Class("flex gap-2"),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									icons.Settings(icons.WithSize(16)),
									Span(g.Text("Edit")),
								),
								button.WithVariant("outline"),
								button.WithAttrs(
									g.Attr("@click", "window.location.href = '"+p.baseUIPath+"/app/"+appID+"/environments/"+envID+"/edit'"),
								),
							),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									icons.Trash(icons.WithSize(16)),
									Span(g.Text("Delete")),
								),
								button.WithVariant("destructive"),
								button.WithAttrs(
									g.Attr("@click", "showDeleteConfirm = true"),
									g.Attr(":disabled", "environment.isDefault"),
									g.Attr("x-show", "!environment.isDefault"),
								),
							),
						),
					),

					// Two column grid for info cards
					Div(
						Class("grid grid-cols-1 lg:grid-cols-2 gap-6"),

						// Overview Card
						card.Card(
							card.Header(
								Div(
									Class("flex items-center gap-2"),
									icons.Info(icons.WithSize(20)),
									card.Title("Overview"),
								),
							),
							card.Content(
								Div(
									Class("space-y-4"),
									// Environment ID
									Div(
										Span(Class("text-sm font-semibold text-gray-700 dark:text-gray-300"), g.Text("Environment ID")),
										P(Class("text-sm text-gray-600 dark:text-gray-400 font-mono"), g.Attr("x-text", "environment.id")),
									),
									// App ID
									Div(
										Span(Class("text-sm font-semibold text-gray-700 dark:text-gray-300"), g.Text("Application ID")),
										P(Class("text-sm text-gray-600 dark:text-gray-400 font-mono"), g.Attr("x-text", "environment.appId")),
									),
									// Slug
									Div(
										Span(Class("text-sm font-semibold text-gray-700 dark:text-gray-300"), g.Text("Slug")),
										P(Class("text-sm text-gray-600 dark:text-gray-400"), g.Attr("x-text", "environment.slug")),
									),
									// Created
									Div(
										Span(Class("text-sm font-semibold text-gray-700 dark:text-gray-300"), g.Text("Created")),
										P(Class("text-sm text-gray-600 dark:text-gray-400"), g.Attr("x-text", "new Date(environment.createdAt).toLocaleString()")),
									),
									// Updated
									Div(
										Span(Class("text-sm font-semibold text-gray-700 dark:text-gray-300"), g.Text("Last Updated")),
										P(Class("text-sm text-gray-600 dark:text-gray-400"), g.Attr("x-text", "new Date(environment.updatedAt).toLocaleString()")),
									),
								),
							),
						),

						// Actions Card
						card.Card(
							card.Header(
								Div(
									Class("flex items-center gap-2"),
									icons.Zap(icons.WithSize(20)),
									card.Title("Quick Actions"),
								),
							),
							card.Content(
								Div(
									Class("space-y-2"),
									button.Button(
										Div(
											Class("flex items-center gap-2 w-full justify-start"),
											icons.RefreshCw(icons.WithSize(16)),
											Span(
												g.Attr("x-show", "!switching"),
												g.Text("Switch to This Environment"),
											),
											Span(
												g.Attr("x-show", "switching"),
												g.Text("Switching..."),
											),
										),
										button.WithVariant("outline"),
										button.WithAttrs(
											g.Attr("class", "w-full"),
											g.Attr("@click", "switchToEnvironment()"),
											g.Attr(":disabled", "switching"),
										),
									),
									button.Button(
										Div(
											Class("flex items-center gap-2 w-full justify-start"),
											icons.Copy(icons.WithSize(16)),
											Span(g.Text("Clone Environment")),
										),
										button.WithVariant("outline"),
										button.WithAttrs(
											g.Attr("class", "w-full"),
											g.Attr("@click", "cloneEnvironment()"),
										),
									),
									button.Button(
										Div(
											Class("flex items-center gap-2 w-full justify-start"),
											icons.ArrowUp(icons.WithSize(16)),
											Span(g.Text("Promote to Another Environment")),
										),
										button.WithVariant("outline"),
										button.WithAttrs(
											g.Attr("class", "w-full"),
											g.Attr("@click", "showPromoteModal = true"),
											g.Attr(":disabled", "allEnvironments.length === 0"),
										),
									),
								),
							),
						),
					),

					// Configuration Card (full width)
					card.Card(
						card.Header(
							Div(
								Class("flex items-center gap-2"),
								icons.Code(icons.WithSize(20)),
								card.Title("Configuration"),
							),
							card.Description("Environment-specific configuration settings"),
						),
						card.Content(
							Pre(
								Class("bg-gray-50 dark:bg-gray-900 p-4 rounded-lg overflow-x-auto text-sm"),
								Code(
									Class("text-gray-800 dark:text-gray-200"),
									g.Attr("x-text", "formatJSON(environment.config)"),
								),
							),
						),
					),

					// Promotion History Card (full width)
					card.Card(
						card.Header(
							Div(
								Class("flex items-center gap-2"),
								icons.GitBranch(icons.WithSize(20)),
								card.Title("Promotion History"),
							),
							card.Description("Recent environment promotions"),
						),
						card.Content(
							// Empty state
							Div(
								g.Attr("x-show", "!environment.promotions || environment.promotions.length === 0"),
								Class("text-center py-8 text-gray-500 dark:text-gray-400"),
								P(g.Text("No promotion history available")),
							),

							// Promotions list
							Div(
								g.Attr("x-show", "environment.promotions && environment.promotions.length > 0"),
								Class("space-y-3"),
								g.El("template", g.Attr("x-for", "promo in environment.promotions"),
									Div(
										Class("flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg"),
										Div(
											Class("flex items-center gap-3"),
											Div(
												Class("flex items-center gap-2 text-sm"),
												Span(Class("font-medium"), g.Attr("x-text", "promo.fromEnvName")),
												icons.ArrowRight(icons.WithSize(16)),
												Span(Class("font-medium"), g.Attr("x-text", "promo.toEnvName")),
											),
										),
										Div(
											Class("flex items-center gap-3"),
											badge.Badge(
												"",
												badge.WithAttrs(
													g.Attr("x-text", "promo.status"),
													g.Attr(":class", `{
													'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200': promo.status === 'completed',
													'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200': promo.status === 'in_progress',
													'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200': promo.status === 'pending',
													'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200': promo.status === 'failed'
												}`),
												),
											),
											Span(
												Class("text-sm text-gray-500 dark:text-gray-400"),
												g.Attr("x-text", "new Date(promo.createdAt).toLocaleDateString()"),
											),
										),
									),
								),
							),
						),
					),
				),
			), // End of template x-if

			// Delete Confirmation Modal
			Div(
				g.Attr("x-show", "showDeleteConfirm"),
				g.Attr("x-cloak", ""),
				Class("fixed inset-0 z-50 overflow-y-auto"),
				g.Attr("aria-labelledby", "delete-modal-title"),
				g.Attr("role", "dialog"),
				g.Attr("aria-modal", "true"),

				// Backdrop
				Div(
					Class("fixed inset-0 bg-black/50 transition-opacity"),
					g.Attr("@click", "showDeleteConfirm = false"),
				),

				// Modal panel
				Div(
					Class("flex min-h-full items-center justify-center p-4"),
					Div(
						Class("relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6"),
						g.Attr("@click.stop", ""),

						// Header
						Div(
							Class("flex items-center gap-3 mb-4"),
							Div(
								Class("flex items-center justify-center w-12 h-12 rounded-full bg-red-100 dark:bg-red-900/30"),
								icons.AlertCircle(icons.WithSize(24), icons.WithClass("text-red-600 dark:text-red-400")),
							),
							Div(
								H3(
									Class("text-lg font-semibold text-gray-900 dark:text-gray-100"),
									g.Attr("id", "delete-modal-title"),
									g.Text("Delete Environment"),
								),
								P(
									Class("text-sm text-gray-500 dark:text-gray-400"),
									g.Text("This action cannot be undone."),
								),
							),
						),

						// Content
						P(
							Class("text-gray-600 dark:text-gray-300 mb-6"),
							g.Text("Are you sure you want to delete the environment "),
							Span(Class("font-semibold"), g.Attr("x-text", "environment?.name")),
							g.Text("? All configuration and settings will be permanently removed."),
						),

						// Actions
						Div(
							Class("flex justify-end gap-3"),
							button.Button(
								g.Text("Cancel"),
								button.WithVariant("outline"),
								button.WithAttrs(
									g.Attr("@click", "showDeleteConfirm = false"),
								),
							),
							button.Button(
								g.Group([]g.Node{
									Span(
										g.Attr("x-show", "!deleting"),
										g.Text("Delete Environment"),
									),
									Span(
										g.Attr("x-show", "deleting"),
										g.Text("Deleting..."),
									),
								}),
								button.WithVariant("destructive"),
								button.WithAttrs(
									g.Attr("@click", "deleteEnvironment()"),
									g.Attr(":disabled", "deleting"),
								),
							),
						),
					),
				),
			),

			// Promote Modal
			Div(
				g.Attr("x-show", "showPromoteModal"),
				g.Attr("x-cloak", ""),
				Class("fixed inset-0 z-50 overflow-y-auto"),
				g.Attr("aria-labelledby", "promote-modal-title"),
				g.Attr("role", "dialog"),
				g.Attr("aria-modal", "true"),

				// Backdrop
				Div(
					Class("fixed inset-0 bg-black/50 transition-opacity"),
					g.Attr("@click", "showPromoteModal = false; selectedTargetEnv = ''"),
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
								g.Attr("id", "promote-modal-title"),
								g.Text("Promote Environment"),
							),
							button.Button(
								icons.X(icons.WithSize(20)),
								button.WithVariant("ghost"),
								button.WithSize("sm"),
								button.WithAttrs(g.Attr("@click", "showPromoteModal = false; selectedTargetEnv = ''")),
							),
						),

						// Content
						Div(
							Class("space-y-4"),
							P(
								Class("text-gray-600 dark:text-gray-300"),
								g.Text("Promote configuration from "),
								Span(Class("font-semibold"), g.Attr("x-text", "environment?.name")),
								g.Text(" to another environment."),
							),

							// Target environment select
							Div(
								Label(
									Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
									g.Attr("for", "target-env"),
									g.Text("Target Environment"),
								),
								Select(
									g.Attr("id", "target-env"),
									g.Attr("x-model", "selectedTargetEnv"),
									Class("w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"),
									Option(Value(""), g.Text("Select an environment...")),
									g.El("template", g.Attr("x-for", "env in allEnvironments"),
										Option(
											g.Attr(":value", "env.id"),
											g.Attr("x-text", "env.name + ' (' + env.type + ')'"),
										),
									),
								),
							),
						),

						// Actions
						Div(
							Class("flex justify-end gap-3 mt-6"),
							button.Button(
								g.Text("Cancel"),
								button.WithVariant("outline"),
								button.WithAttrs(
									g.Attr("@click", "showPromoteModal = false; selectedTargetEnv = ''"),
								),
							),
							button.Button(
								g.Group([]g.Node{
									Span(
										g.Attr("x-show", "!promoting"),
										g.Text("Promote"),
									),
									Span(
										g.Attr("x-show", "promoting"),
										g.Text("Promoting..."),
									),
								}),
								button.WithAttrs(
									g.Attr("@click", "promoteEnvironment()"),
									g.Attr(":disabled", "promoting || !selectedTargetEnv"),
								),
							),
						),
					),
				),
			),
		),
	), nil
}
