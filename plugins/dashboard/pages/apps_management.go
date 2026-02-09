package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// AppsManagementPage shows platform-level apps management (admin only).
func (p *PagesManager) AppsManagementPage(ctx *router.PageContext) (g.Node, error) {
	return primitives.Container(
		Div(
			Class("space-y-2"),

			// Header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold"), g.Text("Applications Management")),
					P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Platform-level application management")),
				),
				button.Button(
					Div(
						Class("flex items-center gap-2"),
						icons.Plus(icons.WithSize(16)),
						Span(g.Text("Create App")),
					),
					button.WithAttrs(g.Attr("@click", "showCreateDialog = true")),
				),
			),

			// Apps Grid
			Div(
				g.Attr("x-data", `{
					apps: [],
					loading: true,
					showCreateDialog: false,
					newApp: {name: '', description: ''},
					async loadApps() {
						this.loading = true;
						try {
							const result = await $bridge.call('getAppsList', {});
							this.apps = result.apps || [];
						} catch (err) {
							console.error('Failed to load apps:', err);
						} finally {
							this.loading = false;
						}
					},
					async createApp() {
						try {
							await $bridge.call('createApp', this.newApp);
							this.showCreateDialog = false;
							this.newApp = {name: '', description: ''};
							await this.loadApps();
						} catch (err) {
							alert('Failed to create app');
						}
					}
				}`),
				g.Attr("x-init", "loadApps()"),

				// Create Dialog (simple modal)
				g.El("div", g.Attr("x-show", "showCreateDialog"),
					Class("fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"),
					g.Attr("@click.self", "showCreateDialog = false"),
					Div(
						Class("bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full"),
						H2(Class("text-xl font-bold mb-4"), g.Text("Create New App")),
						Div(
							Class("space-y-4"),
							Div(
								Label(Class("block font-semibold mb-2"), g.Text("Name")),
								input.Input(input.WithAttrs(g.Attr("x-model", "newApp.name"))),
							),
							Div(
								Label(Class("block font-semibold mb-2"), g.Text("Description")),
								input.Input(input.WithAttrs(g.Attr("x-model", "newApp.description"))),
							),
							Div(
								Class("flex gap-2 justify-end"),
								button.Button(
									g.Text("Cancel"),
									button.WithVariant("outline"),
									button.WithAttrs(g.Attr("@click", "showCreateDialog = false")),
								),
								button.Button(
									g.Text("Create"),
									button.WithAttrs(g.Attr("@click", "createApp()")),
								),
							),
						),
					),
				),

				// Apps Grid
				Div(
					Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
					g.El("template", g.Attr("x-for", "app in apps"),
						card.Card(
							card.Header(
								card.Title("", card.WithAttrs(g.Attr("x-text", "app.name"))),
								card.Description("", card.WithAttrs(g.Attr("x-text", "app.description || 'No description'"))),
							),
							card.Content(
								Div(
									Class("space-y-2 text-sm"),
									Div(
										Span(Class("font-semibold"), g.Text("Users: ")),
										Span(g.Attr("x-text", "app.userCount || 0")),
									),
									Div(
										Span(Class("font-semibold"), g.Text("Status: ")),
										Span(g.Attr("x-text", "app.status")),
									),
								),
							),
							card.Footer(
								Div(
									Class("flex gap-2"),
									button.Button(
										g.Text("View"),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(g.Attr("@click", "window.location.href = `"+p.baseUIPath+"/app/${app.id}`")),
									),
									button.Button(
										g.Text("Settings"),
										button.WithVariant("outline"),
										button.WithSize("sm"),
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
