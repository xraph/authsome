package pages

import (
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// PluginsPage renders the plugins management page
func (p *PagesManager) PluginsPage(ctx *router.PageContext) (g.Node, error) {
	// Get appId from context for bridge calls
	appIDStr := ctx.Param("appId")

	return primitives.Container(
		html.Div(
			html.Class("space-y-2"),
			g.Attr("x-data", `{
			plugins: [],
			loading: true,
			get totalPlugins() { return this.plugins.length; },
			get enabledPlugins() { return this.plugins.filter(p => p.enabled).length; },
			get disabledPlugins() { return this.plugins.filter(p => !p.enabled).length; },
			async loadPlugins() {
				try {
					const result = await $go('getPluginsList', { appId: '`+appIDStr+`' });
					this.plugins = result.plugins || [];
				} catch (err) {
					console.error('Failed to load plugins:', err);
				} finally {
					this.loading = false;
				}
			}
		}`),
			g.Attr("x-init", "loadPlugins()"),

			// Page Header
			html.Div(
				html.Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
				html.Div(
					html.H1(
						html.Class("text-2xl font-bold text-slate-900 dark:text-white"),
						g.Text("Plugins"),
					),
					html.P(
						html.Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
						g.Text("Manage and configure authentication plugins"),
					),
				),
			),

			// Stats Cards - Dynamic
			html.Div(
				html.Class("grid grid-cols-1 gap-4 md:grid-cols-3"),
				p.pluginStatCardDynamic("Total Plugins", "totalPlugins", "violet"),
				p.pluginStatCardDynamic("Enabled", "enabledPlugins", "emerald"),
				p.pluginStatCardDynamic("Disabled", "disabledPlugins", "slate"),
			),

			// Plugins Grid
			html.Div(
				html.Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"),

				// Loading state
				html.Template(
					g.Attr("x-if", "loading"),
					html.Div(
						html.Class("col-span-full flex items-center justify-center py-12"),
						html.Div(
							html.Class("animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"),
						),
					),
				),

				// Empty state
				html.Template(
					g.Attr("x-if", "!loading && plugins.length === 0"),
					html.Div(
						html.Class("col-span-full flex flex-col items-center justify-center py-12"),
						html.Div(
							html.Class("h-16 w-16 rounded-full bg-slate-100 dark:bg-gray-800 flex items-center justify-center mb-4"),
							icons.Box(icons.WithSize(32)),
						),
						html.H3(
							html.Class("text-lg font-semibold text-slate-900 dark:text-white"),
							g.Text("No plugins found"),
						),
						html.P(
							html.Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
							g.Text("Plugins will appear here when configured"),
						),
					),
				),

				// Plugins List
				html.Template(
					g.Attr("x-if", "!loading && plugins.length > 0"),
					html.Div(
						html.Class("contents"),
						html.Template(
							g.Attr("x-for", "plugin in plugins"),
							g.Attr("x-bind:key", "plugin.id"),
							p.pluginCard(),
						),
					),
				),
			),
		),
	), nil
}

func (p *PagesManager) pluginStatCardDynamic(label, alpineValue, color string) g.Node {
	colorClasses := map[string]string{
		"violet":  "border-violet-100 dark:border-violet-900/30 bg-white dark:bg-gray-900",
		"emerald": "border-emerald-100 dark:border-emerald-900/30 bg-white dark:bg-gray-900",
		"slate":   "border-slate-100 dark:border-slate-900/30 bg-white dark:bg-gray-900",
	}

	borderClass := colorClasses[color]
	if borderClass == "" {
		borderClass = colorClasses["violet"]
	}

	return html.Div(
		html.Class("rounded-lg border "+borderClass+" p-5"),
		html.Div(
			html.Class("flex items-center justify-between"),
			html.Div(
				html.Div(
					html.Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Attr("x-text", alpineValue),
				),
				html.Div(
					html.Class("text-sm font-medium text-slate-500 dark:text-gray-400"),
					g.Text(label),
				),
			),
		),
	)
}

func (p *PagesManager) pluginCard() g.Node {
	return html.Div(
		html.Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:border-violet-300 dark:hover:border-violet-700 transition-all"),

		// Card Header
		html.Div(
			html.Class("p-5 border-b border-slate-100 dark:border-gray-800"),
			html.Div(
				html.Class("flex items-start gap-4"),
				html.Div(
					html.Class("flex-shrink-0 h-12 w-12 rounded-lg flex items-center justify-center"),
					g.Attr("x-bind:class", `{
						'bg-violet-50 dark:bg-violet-900/20 text-violet-600 dark:text-violet-400 border border-violet-200 dark:border-violet-900/30': plugin.category?.toLowerCase() === 'core',
						'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-900/30': plugin.category?.toLowerCase() === 'authentication',
						'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-900/30': plugin.category?.toLowerCase() === 'security',
						'bg-amber-50 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400 border border-amber-200 dark:border-amber-900/30': plugin.category?.toLowerCase() === 'session',
						'bg-slate-50 dark:bg-slate-900/20 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-900/30': !plugin.category
					}`),
					icons.Box(icons.WithSize(24)),
				),
				html.Div(
					html.Class("flex-1 min-w-0"),
					html.H3(
						html.Class("text-lg font-semibold text-slate-900 dark:text-white truncate"),
						g.Attr("x-text", "plugin.name"),
					),
					html.Div(
						html.Class("mt-1 flex items-center gap-2"),
						html.Span(
							html.Class("text-xs font-medium text-slate-500 dark:text-gray-400 capitalize"),
							g.Attr("x-text", "plugin.category || 'other'"),
						),
						// Status badge
						html.Span(
							html.Class("inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-semibold"),
							g.Attr("x-bind:class", `plugin.enabled 
								? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-800 dark:text-emerald-400' 
								: 'bg-slate-100 dark:bg-slate-500/20 text-slate-600 dark:text-slate-400'`),
							g.Attr("x-text", "plugin.enabled ? 'Enabled' : 'Disabled'"),
						),
					),
				),
			),
		),

		// Card Body
		html.Div(
			html.Class("p-5 flex-1"),
			html.P(
				html.Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Attr("x-text", "plugin.description || 'No description available'"),
			),
		),

		// Card Footer
		html.Div(
			html.Class("px-5 py-3 border-t border-slate-100 dark:border-gray-800 bg-slate-50 dark:bg-gray-800/50 flex items-center justify-between"),
			html.Span(
				html.Class("text-xs text-slate-500 dark:text-gray-400"),
				html.Span(g.Text("ID: ")),
				html.Span(g.Attr("x-text", "plugin.id")),
			),
			html.Div(
				html.Class("flex items-center gap-1 text-xs"),
				g.Attr("x-bind:class", `plugin.enabled 
					? 'text-emerald-600 dark:text-emerald-400' 
					: 'text-slate-400 dark:text-gray-500'`),
				html.Template(
					g.Attr("x-if", "plugin.enabled"),
					html.Span(
						icons.Activity(icons.WithSize(12)),
						g.Text(" Active"),
					),
				),
				html.Template(
					g.Attr("x-if", "!plugin.enabled"),
					html.Span(g.Text("Not configured")),
				),
			),
		),
	)
}
