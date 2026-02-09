package pages

import (
	"github.com/xraph/forgeui/components/badge"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// PluginsManagementPage shows plugin management.
func (p *PagesManager) PluginsManagementPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),

			// Header
			Div(
				H1(Class("text-3xl font-bold"), g.Text("Plugins Management")),
				P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Manage authentication plugins and extensions")),
			),

			// Plugins List
			Div(
				g.Attr("x-data", `{
					plugins: [],
					loading: true,
					async loadPlugins() {
						this.loading = true;
						try {
							const result = await $bridge.call('getPluginsList', {
								appId: '`+appID+`'
							});
							this.plugins = result.plugins || [];
						} catch (err) {
							console.error('Failed to load plugins:', err);
						} finally {
							this.loading = false;
						}
					},
					async togglePlugin(pluginId, enabled) {
						try {
							await $bridge.call('togglePlugin', {
								appId: '`+appID+`',
								pluginId,
								enabled: !enabled
							});
							await this.loadPlugins();
						} catch (err) {
							alert('Failed to toggle plugin');
						}
					}
				}`),
				g.Attr("x-init", "loadPlugins()"),

				Div(
					Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
					g.El("template", g.Attr("x-for", "plugin in plugins"),
						card.Card(
							card.Header(
								Div(
									Class("flex items-center justify-between"),
									card.Title("", card.WithAttrs(g.Attr("x-text", "plugin.name"))),
									badge.Badge(
										"",
										badge.WithAttrs(
											g.Attr("x-text", "plugin.enabled ? 'Enabled' : 'Disabled'"),
											g.Attr(":class", "plugin.enabled ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-gray-100 text-gray-800'"),
										),
									),
								),
								card.Description("", card.WithAttrs(g.Attr("x-text", "plugin.description"))),
							),
							card.Content(
								Div(
									Class("space-y-2 text-sm"),
									Div(
										Span(Class("font-semibold"), g.Text("Version: ")),
										Span(g.Attr("x-text", "plugin.version")),
									),
									Div(
										Span(Class("font-semibold"), g.Text("Category: ")),
										Span(g.Attr("x-text", "plugin.category")),
									),
								),
							),
							card.Footer(
								Div(
									Class("flex gap-2"),
									button.Button(
										g.Attr("x-text", "plugin.enabled ? 'Disable' : 'Enable'"),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(g.Attr("@click", "togglePlugin(plugin.id, plugin.enabled)")),
									),
									button.Button(
										g.Text("Configure"),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(g.Attr(":disabled", "!plugin.enabled")),
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
