package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardIndexPage shows app selection or redirects to default app.
func (p *PagesManager) DashboardIndexPage(ctx *router.PageContext) (g.Node, error) {
	// // Check if user is authenticated
	// user := p.checkExistingPageSession(ctx)
	// if user == nil {
	// 	// Redirect to login
	// 	ctx.SetHeader("Location", p.baseUIPath+"/auth/login")
	// 	ctx.ResponseWriter.WriteHeader(302)
	// 	return nil, nil
	// }
	return primitives.Container(
		Div(
			g.Attr("x-data", `{
				apps: [],
				loading: true,
				isMultiApp: false,
				showCreateDialog: false,
				creating: false,
				newApp: { name: '', description: '' },
				async fetchApps() {
					try {
						const result = await $go('getAppsList', {});
						this.apps = result.apps || [];
						this.isMultiApp = result.isMultiApp || false;
					} catch (err) {
						console.error('Failed to load apps:', err);
					} finally {
						this.loading = false;
					}
				},
				async createApp() {
					if (!this.newApp.name.trim()) {
						alert('App name is required');
						return;
					}
					this.creating = true;
					try {
						const result = await $go('createApp', {
							name: this.newApp.name,
							description: this.newApp.description
						});
						if (result.message) {
							alert(result.message);
						}
						this.showCreateDialog = false;
						this.newApp = { name: '', description: '' };
						await this.fetchApps();
					} catch (err) {
						alert(err.message || 'Failed to create app');
					} finally {
						this.creating = false;
					}
				},
				getAppGradient(name) {
					const gradients = [
						'bg-gradient-to-br from-violet-500 to-purple-600',
						'bg-gradient-to-br from-blue-500 to-cyan-600',
						'bg-gradient-to-br from-emerald-500 to-teal-600',
						'bg-gradient-to-br from-orange-500 to-red-600',
						'bg-gradient-to-br from-pink-500 to-rose-600',
						'bg-gradient-to-br from-indigo-500 to-blue-600',
					];
					const hash = name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
					return gradients[hash % gradients.length];
				}
			}`),
			g.Attr("x-init", `
			fetchApps();
			// Check if we should open create dialog from URL param
			const urlParams = new URLSearchParams(window.location.search);
			if (urlParams.get('create') === 'true') {
				showCreateDialog = true;
				// Clean up URL
				window.history.replaceState({}, '', window.location.pathname);
			}
		`),

			// Header
			Div(
				Class("mb-8"),
				H1(
					Class("text-3xl font-bold text-gray-900 dark:text-white"),
					g.Text("Select an Application"),
				),
				P(
					Class("mt-2 text-gray-600 dark:text-gray-400"),
					g.Text("Choose an application to manage"),
				),
			),

			// Create App Dialog
			Div(
				g.Attr("x-show", "showCreateDialog"),
				g.Attr("x-cloak", ""),
				Class("fixed inset-0 z-50 overflow-y-auto"),
				g.Attr("aria-labelledby", "create-app-title"),
				g.Attr("role", "dialog"),
				g.Attr("aria-modal", "true"),

				// Backdrop
				Div(
					Class("fixed inset-0 bg-black/50 transition-opacity"),
					g.Attr("@click", "showCreateDialog = false"),
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
								g.Attr("id", "create-app-title"),
								g.Text("Create New Application"),
							),
							button.Button(
								icons.X(icons.WithSize(20)),
								button.WithVariant("ghost"),
								button.WithSize("sm"),
								button.WithAttrs(g.Attr("@click", "showCreateDialog = false")),
							),
						),

						// Form
						Form(
							g.Attr("@submit.prevent", "createApp()"),
							Class("space-y-4"),

							// Name field
							Div(
								Label(
									Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
									g.Attr("for", "app-name"),
									g.Text("Application Name"),
								),
								input.Input(
									input.WithAttrs(
										g.Attr("id", "app-name"),
										g.Attr("x-model", "newApp.name"),
										g.Attr("placeholder", "e.g., My Application"),
										g.Attr("required", ""),
									),
								),
							),

							// Description field
							Div(
								Label(
									Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
									g.Attr("for", "app-description"),
									g.Text("Description (optional)"),
								),
								Textarea(
									g.Attr("id", "app-description"),
									g.Attr("x-model", "newApp.description"),
									Class("w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"),
									g.Attr("rows", "3"),
									g.Attr("placeholder", "Describe this application..."),
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
										g.Attr("@click", "showCreateDialog = false"),
									),
								),
								button.Button(
									g.Group([]g.Node{
										Span(
											g.Attr("x-show", "!creating"),
											g.Text("Create Application"),
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

			// Apps Grid
			Div(
				Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),

				// Loading state
				g.El("template", g.Attr("x-if", "loading"),
					Div(
						Class("col-span-3 text-center py-12"),
						Div(
							Class("animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto"),
						),
						P(
							Class("mt-4 text-gray-600 dark:text-gray-400"),
							g.Text("Loading applications..."),
						),
					),
				),

				// Create App Card (shown first in multi-app mode)
				g.El("template", g.Attr("x-if", "!loading && isMultiApp"),
					p.createAppCard(),
				),

				// Apps list
				g.El("template", g.Attr("x-if", "!loading"),
					g.El("template", g.Attr("x-for", "app in apps"),
						p.appCard(),
					),
				),

				// Empty state (only when no apps AND not multi-app mode)
				g.El("template", g.Attr("x-if", "!loading && apps.length === 0 && !isMultiApp"),
					Div(
						Class("col-span-3 text-center py-12"),
						icons.Box(icons.WithSize(48), icons.WithClass("mx-auto text-gray-400")),
						P(
							Class("mt-4 text-gray-600 dark:text-gray-400"),
							g.Text("No applications available"),
						),
					),
				),
			),
		),
	), nil
}

// createAppCard renders a card for creating a new app.
func (p *PagesManager) createAppCard() g.Node {
	return Div(
		Button(
			g.Attr("type", "button"),
			g.Attr("@click", "showCreateDialog = true"),
			Class("group relative block w-full h-full min-h-[200px] rounded-2xl border-2 border-dashed border-slate-300 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden transition-all duration-200 hover:border-violet-400 dark:hover:border-violet-600 hover:bg-violet-50 dark:hover:bg-violet-900/10 cursor-pointer"),

			Div(
				Class("flex flex-col items-center justify-center h-full p-6"),

				// Plus icon with circle
				Div(
					Class("flex h-16 w-16 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/30 text-violet-600 dark:text-violet-400 mb-4 group-hover:bg-violet-200 dark:group-hover:bg-violet-900/50 transition-colors"),
					icons.Plus(icons.WithSize(32)),
				),

				// Text
				H3(
					Class("text-lg font-semibold text-slate-700 dark:text-gray-300 mb-1"),
					g.Text("Create New Application"),
				),
				P(
					Class("text-sm text-slate-500 dark:text-gray-400 text-center"),
					g.Text("Add a new application to your platform"),
				),
			),
		),
	)
}

// CreateAppPage redirects to the index page with the create dialog open.
func (p *PagesManager) CreateAppPage(ctx *router.PageContext) (g.Node, error) {
	// Redirect to the index page with a query param to open the dialog
	ctx.ResponseWriter.Header().Set("Location", p.baseUIPath+"/?create=true")
	ctx.ResponseWriter.WriteHeader(302)

	return nil, nil
}

func (p *PagesManager) appCard() g.Node {
	return Div(
		g.Attr(":key", "app.id"),
		A(
			g.Attr(":href", "`"+p.baseUIPath+"/app/${app.id}`"),
			Class("group relative block rounded-2xl border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden transition-all duration-200 hover:shadow-xl hover:-translate-y-1 hover:border-violet-300 dark:hover:border-violet-700"),

			// Card Content
			Div(
				Class("p-6"),

				// Header with Icon and Status Badge
				Div(
					Class("flex items-start justify-between mb-4"),

					// App Icon and Name
					Div(
						Class("flex items-center gap-3 flex-1 min-w-0"),
						// App Icon with gradient and first letter
						Div(
							g.Attr(":class", `'flex-shrink-0 w-14 h-14 rounded-xl flex items-center justify-center shadow-lg ' + getAppGradient(app.name)`),
							Span(
								Class("text-2xl font-bold text-white"),
								g.Attr("x-text", "app.name.charAt(0).toUpperCase()"),
							),
						),
						Div(
							Class("flex-1 min-w-0"),
							H3(
								Class("text-lg font-semibold text-slate-900 dark:text-white truncate"),
								g.Attr("x-text", "app.name"),
							),
							P(
								Class("text-sm text-slate-500 dark:text-gray-400 truncate"),
								g.Attr("x-text", "app.id ? '@' + app.id.substring(0, 8) : ''"),
							),
						),
					),

					// Status Badge
					Span(
						g.Attr(":class", `app.status === 'active' ? 'px-2.5 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'px-2.5 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'`),
						g.Attr("x-text", "app.status"),
					),
				),

				// Stats Row
				Div(
					Class("flex items-center gap-4 text-xs text-slate-600 dark:text-gray-400 pb-4 border-b border-slate-100 dark:border-gray-800"),
					Div(
						Class("flex items-center gap-1.5"),
						icons.Users(icons.WithSize(14), icons.WithClass("text-inherit")),
						Span(g.Attr("x-text", "(app.userCount || 0) + ' members'")),
					),
					Div(
						Class("flex items-center gap-1.5"),
						icons.Calendar(icons.WithSize(14), icons.WithClass("text-inherit")),
						Span(g.Attr("x-text", "new Date(app.createdAt).toLocaleDateString()")),
					),
				),

				// Action area with arrow
				Div(
					Class("flex items-center justify-between pt-4"),
					P(
						Class("text-sm text-slate-600 dark:text-gray-400 line-clamp-1"),
						g.Attr("x-text", "app.description || 'No description'"),
					),
					icons.ArrowRight(
						icons.WithSize(16),
						icons.WithClass("text-violet-600 dark:text-violet-400 opacity-0 group-hover:opacity-100 transition-opacity"),
					),
				),
			),
		),
	)
}
