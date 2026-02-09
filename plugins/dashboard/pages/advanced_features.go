package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/table"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ConfigViewerPage shows system configuration.
func (p *PagesManager) ConfigViewerPage(ctx *router.PageContext) (g.Node, error) {
	return primitives.Container(
		Div(
			Class("space-y-2"),

			H1(Class("text-3xl font-bold"), g.Text("Configuration Viewer")),

			Div(
				g.Attr("x-data", `{
					config: {},
					loading: true,
					async loadConfig() {
						this.loading = true;
						try {
							const result = await $bridge.call('getSystemConfig', {});
							this.config = result;
						} catch (err) {
							console.error('Failed to load config:', err);
						} finally {
							this.loading = false;
						}
					}
				}`),
				g.Attr("x-init", "loadConfig()"),

				card.Card(
					card.Header(
						card.Title("System Configuration"),
						card.Description("View current system configuration (read-only)"),
					),
					card.Content(
						Pre(
							Class("bg-gray-50 dark:bg-gray-900 p-4 rounded-lg overflow-auto text-sm"),
							Code(g.Attr("x-text", "JSON.stringify(config, null, 2)")),
						),
					),
				),
			),
		),
	), nil
}

// AuditLogViewerPage shows audit logs.
func (p *PagesManager) AuditLogViewerPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),

			// Header
			Div(
				H1(Class("text-3xl font-bold"), g.Text("Audit Logs")),
				P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("View system audit logs and activity")),
			),

			// Audit Logs Table
			Div(
				g.Attr("x-data", `{
					logs: [],
					loading: true,
					filters: {
						action: '',
						user: '',
						startDate: '',
						endDate: ''
					},
					async loadLogs() {
						this.loading = true;
						try {
							const result = await $bridge.call('getAuditLogs', {
								appId: '`+appID+`',
								...this.filters
							});
							this.logs = result.logs || [];
						} catch (err) {
							console.error('Failed to load logs:', err);
						} finally {
							this.loading = false;
						}
					}
				}`),
				g.Attr("x-init", "loadLogs()"),

				card.Card(
					card.Content(
						table.Table()(
							table.TableHeader()(
								table.TableRow()(
									table.TableHeaderCell()(g.Text("Timestamp")),
									table.TableHeaderCell()(g.Text("Action")),
									table.TableHeaderCell()(g.Text("User")),
									table.TableHeaderCell()(g.Text("Resource")),
									table.TableHeaderCell()(g.Text("Status")),
								),
							),
							table.TableBody()(
								g.El("template", g.Attr("x-for", "log in logs"),
									g.El("tr",
										g.El("td", Class("px-6 py-4"), g.Attr("x-text", "new Date(log.timestamp).toLocaleString()")),
										g.El("td", Class("px-6 py-4"), g.Attr("x-text", "log.action")),
										g.El("td", Class("px-6 py-4"), g.Attr("x-text", "log.userEmail || 'System'")),
										g.El("td", Class("px-6 py-4"), g.Attr("x-text", "log.resource")),
										g.El("td", Class("px-6 py-4"), g.Attr("x-text", "log.status")),
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

// ErrorPage shows custom error page.
func (p *PagesManager) ErrorPage(ctx *router.PageContext) (g.Node, error) {
	errorCode := ctx.Query("code")
	errorMessage := ctx.Query("message")

	if errorCode == "" {
		errorCode = "500"
	}

	if errorMessage == "" {
		errorMessage = "An unexpected error occurred"
	}

	return primitives.Container(
		Div(
			Class("min-h-screen flex items-center justify-center p-8"),
			Div(
				Class("text-center"),
				icons.AlertCircle(icons.WithSize(64), icons.WithClass("mx-auto text-red-500 mb-4")),
				H1(Class("text-6xl font-bold mb-4"), g.Text(errorCode)),
				P(Class("text-xl text-gray-600 dark:text-gray-400 mb-8"), g.Text(errorMessage)),
				button.Button(
					Div(
						Class("flex items-center gap-2"),
						icons.Home(icons.WithSize(16)),
						Span(g.Text("Back to Dashboard")),
					),
					button.WithAttrs(g.Attr("onclick", "window.location.href = '"+p.baseUIPath+"'")),
				),
			),
		),
	), nil
}

// ForbiddenPage displays a 403 access denied page.
func (p *PagesManager) ForbiddenPage(loginURL string) g.Node {
	return primitives.Container(
		Div(
			Class("min-h-screen flex items-center justify-center p-8"),
			Div(
				Class("max-w-md mx-auto text-center"),
				icons.ShieldAlert(icons.WithSize(64), icons.WithClass("mx-auto text-red-500 mb-6")),
				H1(Class("text-6xl font-bold mb-4"), g.Text("403")),
				H2(Class("text-2xl font-semibold mb-4"), g.Text("Access Denied")),
				P(Class("text-lg text-gray-600 dark:text-gray-400 mb-8"),
					g.Text("You don't have permission to access the dashboard. Please contact your administrator to request access.")),
				Div(
					Class("flex flex-col sm:flex-row gap-4 justify-center"),
					button.Button(
						Div(
							Class("flex items-center gap-2"),
							icons.ArrowLeft(icons.WithSize(16)),
							Span(g.Text("Back to Login")),
						),
						button.WithAttrs(g.Attr("onclick", "window.location.href = '"+loginURL+"'")),
					),
				),
			),
		),
	)
}
