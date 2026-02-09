package pages

import (
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// IndexPage - Dashboard entry point that handles app selection or redirect.
func (p *PagesManager) IndexPage(ctx *router.PageContext) (g.Node, error) {
	// In standalone mode, we would redirect to default app
	// In multiapp mode, show app selection
	// For now, we'll show the app selection interface
	// The bridge will determine available apps based on mode
	return primitives.Container(
		Div(
			Class("p-8"),
			g.Attr("x-data", `{
				apps: [],
				loading: true,
				isMultiApp: false,
				defaultAppId: null,
				error: null,
				async init() {
					await this.loadApps();
					// If standalone mode with a default app, redirect automatically
					if (!this.isMultiApp && this.defaultAppId && this.apps.length === 1) {
						window.location.href = '`+p.baseUIPath+`/app/' + this.defaultAppId;
					}
				},
				async loadApps() {
					this.loading = true;
					this.error = null;
					try {
						// Check if bridge is available
						if (typeof $go === 'undefined') {
							console.error('Bridge ($go) not available');
							this.error = 'Bridge not initialized';
							return;
						}
						
						const result = await $go('getAppsList', {});
						console.log('Apps loaded:', result);
						this.apps = result.apps || [];
						this.isMultiApp = result.isMultiApp || false;
						this.defaultAppId = result.defaultAppId || null;
					} catch (err) {
						console.error('Failed to load apps:', err);
						this.error = err.message || 'Failed to load applications';
					} finally {
						this.loading = false;
					}
				},
				getAppGradient(name) {
					// Generate gradient color based on app name
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
			g.Attr("x-init", "init()"),

			// Header (only show if not redirecting)
			Div(
				g.Attr("x-show", "!loading || (loading && apps.length > 0)"),
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

			// Error state
			Div(
				g.Attr("x-show", "error && !loading"),
				Class("max-w-md mx-auto"),
				Div(
					Class("rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4"),
					Div(
						Class("flex items-start gap-3"),
						icons.AlertCircle(icons.WithSize(20), icons.WithClass("text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5")),
						Div(
							Class("flex-1"),
							H3(
								Class("text-sm font-medium text-red-800 dark:text-red-300 mb-1"),
								g.Text("Error Loading Applications"),
							),
							P(
								Class("text-sm text-red-700 dark:text-red-400"),
								g.Attr("x-text", "error"),
							),
						),
					),
				),
			),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("text-center py-12"),
				Div(
					Class("animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto"),
				),
				P(
					Class("mt-4 text-gray-600 dark:text-gray-400"),
					g.Text("Loading applications..."),
				),
			),

			// Apps Grid
			Div(
				g.Attr("x-show", "!loading && apps.length > 0"),
				Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
				g.El("template", g.Attr("x-for", "app in apps"),
					p.renderAppCard(),
				),
			),

			// Empty state
			Div(
				g.Attr("x-show", "!loading && apps.length === 0"),
				Class("text-center py-12"),
				icons.Box(icons.WithSize(48), icons.WithClass("mx-auto text-gray-400 mb-4")),
				H3(
					Class("text-lg font-medium text-gray-900 dark:text-white mb-2"),
					g.Text("No Applications Available"),
				),
				P(
					Class("text-gray-600 dark:text-gray-400 mb-6"),
					g.Text("Contact your administrator to get access to applications"),
				),
			),
		),
	), nil
}

// renderAppCard renders a single app card for the grid.
func (p *PagesManager) renderAppCard() g.Node {
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
