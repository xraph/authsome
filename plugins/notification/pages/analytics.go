package pages

import (
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/primitives"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// AnalyticsPage renders the analytics dashboard
func AnalyticsPage(currentApp *app.App, basePath string) g.Node {
	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Page header
			PageHeader(
				"Notification Analytics",
				"View detailed performance metrics and insights",
			),

			// Main content with Alpine.js
			Div(
				g.Attr("x-data", analyticsData(currentApp.ID.String())),
				g.Attr("x-init", "await loadAnalytics()"),
				Class("space-y-6"),

				// Loading state
				Div(
					g.Attr("x-show", "loading"),
					LoadingSpinner(),
				),

				// Error message
				ErrorMessage("error && !loading"),

				// Date range selector
				Div(
					g.Attr("x-show", "!loading && !error"),
					Class("flex items-center gap-2"),
					Button(
						Type("button"),
						g.Attr("@click", "changeDateRange(7)"),
						g.Attr(":class", "days === 7 ? 'bg-accent' : ''"),
						Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-4 py-2 text-sm hover:bg-accent hover:text-accent-foreground"),
						g.Text("7 days"),
					),
					Button(
						Type("button"),
						g.Attr("@click", "changeDateRange(30)"),
						g.Attr(":class", "days === 30 ? 'bg-accent' : ''"),
						Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-4 py-2 text-sm hover:bg-accent hover:text-accent-foreground"),
						g.Text("30 days"),
					),
					Button(
						Type("button"),
						g.Attr("@click", "changeDateRange(90)"),
						g.Attr(":class", "days === 90 ? 'bg-accent' : ''"),
						Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-4 py-2 text-sm hover:bg-accent hover:text-accent-foreground"),
						g.Text("90 days"),
					),
				),

				// Overview stats
				Div(
					g.Attr("x-show", "!loading && !error && analytics"),
					Class("grid gap-4 md:grid-cols-4"),

					// Total Sent
					card.Card(
						card.Content(
							Class("p-6"),
							Div(
								Class("space-y-2"),
								P(Class("text-sm font-medium text-muted-foreground"), g.Text("Total Sent")),
								H3(
									Class("text-2xl font-bold"),
									g.Attr("x-text", "analytics.overview.totalSent"),
								),
							),
						),
					),

					// Delivery Rate
					card.Card(
						card.Content(
							Class("p-6"),
							Div(
								Class("space-y-2"),
								P(Class("text-sm font-medium text-muted-foreground"), g.Text("Delivery Rate")),
								H3(
									Class("text-2xl font-bold text-green-600 dark:text-green-400"),
									g.Attr("x-text", "analytics.overview.deliveryRate.toFixed(1) + '%'"),
								),
							),
						),
					),

					// Open Rate
					card.Card(
						card.Content(
							Class("p-6"),
							Div(
								Class("space-y-2"),
								P(Class("text-sm font-medium text-muted-foreground"), g.Text("Open Rate")),
								H3(
									Class("text-2xl font-bold text-violet-600 dark:text-violet-400"),
									g.Attr("x-text", "analytics.overview.openRate.toFixed(1) + '%'"),
								),
							),
						),
					),

					// Click Rate
					card.Card(
						card.Content(
							Class("p-6"),
							Div(
								Class("space-y-2"),
								P(Class("text-sm font-medium text-muted-foreground"), g.Text("Click Rate")),
								H3(
									Class("text-2xl font-bold text-amber-600 dark:text-amber-400"),
									g.Attr("x-text", "analytics.overview.clickRate.toFixed(1) + '%'"),
								),
							),
						),
					),
				),

				// Template Performance
				Div(
					g.Attr("x-show", "!loading && !error && analytics && analytics.byTemplate && analytics.byTemplate.length > 0"),
					card.Card(
						card.Header(
							card.Title("Template Performance"),
							card.Description("Performance metrics by template"),
						),
						card.Content(
							Class("p-0"),
							Div(
								Class("overflow-x-auto"),
								Table(
									Class("w-full"),
									THead(
										Class("border-b"),
										Tr(
											Th(Class("p-4 text-left text-sm font-medium"), g.Text("Template")),
											Th(Class("p-4 text-left text-sm font-medium"), g.Text("Sent")),
											Th(Class("p-4 text-left text-sm font-medium"), g.Text("Delivered")),
											Th(Class("p-4 text-left text-sm font-medium"), g.Text("Opened")),
											Th(Class("p-4 text-left text-sm font-medium"), g.Text("Clicked")),
											Th(Class("p-4 text-left text-sm font-medium"), g.Text("Delivery Rate")),
											Th(Class("p-4 text-left text-sm font-medium"), g.Text("Open Rate")),
										),
									),
									TBody(
										Template(
											g.Attr("x-for", "tmpl in analytics.byTemplate"),
											g.Attr(":key", "tmpl.templateId"),
											Tr(
												Class("border-b hover:bg-muted/50"),
												Td(Class("p-4 text-sm"), g.Attr("x-text", "tmpl.templateName")),
												Td(Class("p-4 text-sm"), g.Attr("x-text", "tmpl.totalSent")),
												Td(Class("p-4 text-sm"), g.Attr("x-text", "tmpl.totalDelivered")),
												Td(Class("p-4 text-sm"), g.Attr("x-text", "tmpl.totalOpened")),
												Td(Class("p-4 text-sm"), g.Attr("x-text", "tmpl.totalClicked")),
												Td(Class("p-4 text-sm text-green-600 dark:text-green-400"),
													g.Attr("x-text", "tmpl.deliveryRate.toFixed(1) + '%'")),
												Td(Class("p-4 text-sm text-violet-600 dark:text-violet-400"),
													g.Attr("x-text", "tmpl.openRate.toFixed(1) + '%'")),
											),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}

func analyticsData(appID string) string {
	return `{
		analytics: {
			overview: {
				totalSent: 0,
				totalDelivered: 0,
				totalOpened: 0,
				totalClicked: 0,
				deliveryRate: 0,
				openRate: 0,
				clickRate: 0
			},
			byTemplate: [],
			daily: []
		},
		loading: true,
		error: null,
		days: 30,

		async loadAnalytics() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('notification.getAnalytics', {
					days: this.days
				});
				if (result && result.analytics) {
					this.analytics = result.analytics;
				} else {
					this.error = 'Invalid response format';
				}
			} catch (err) {
				console.error('Failed to load analytics:', err);
				this.error = err.message || 'Failed to load analytics';
			} finally {
				this.loading = false;
			}
		},

		async changeDateRange(days) {
			this.days = days;
			await this.loadAnalytics();
		}
	}`
}
