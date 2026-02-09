package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/primitives"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// OverviewPage renders the main notifications overview/dashboard.
func OverviewPage(currentApp *app.App, basePath string) g.Node {
	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Page header
			PageHeader(
				"Notifications",
				"Manage templates, providers, and view analytics",
			),

			// Main content with Alpine.js data
			Div(
				g.Attr("x-data", overviewData(currentApp.ID.String())),
				g.Attr("x-init", "await loadStats()"),
				Class("space-y-6"),

				// Loading state
				Div(
					g.Attr("x-show", "loading"),
					LoadingSpinner(),
				),

				// Error message
				ErrorMessage("error && !loading"),

				// Stats overview
				Div(
					g.Attr("x-show", "!loading && !error && stats"),
					Class("grid gap-4 md:grid-cols-4"),

					// Total Sent Card
					card.Card(
						card.Content(
							Class("p-6"),
							Div(
								Class("flex items-center justify-between"),
								Div(
									Class("space-y-1"),
									P(Class("text-sm font-medium text-muted-foreground"), g.Text("Total Sent")),
									H3(
										Class("text-2xl font-bold"),
										g.Attr("x-text", "stats.totalSent"),
									),
								),
								Div(
									Class("flex h-12 w-12 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900/20"),
									lucide.Mail(Class("h-6 w-6 text-blue-600 dark:text-blue-400")),
								),
							),
						),
					),

					// Delivery Rate Card
					card.Card(
						card.Content(
							Class("p-6"),
							Div(
								Class("flex items-center justify-between"),
								Div(
									Class("space-y-1"),
									P(Class("text-sm font-medium text-muted-foreground"), g.Text("Delivery Rate")),
									H3(
										Class("text-2xl font-bold"),
										g.Attr("x-text", "stats.deliveryRate.toFixed(1) + '%'"),
									),
								),
								Div(
									Class("flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/20"),
									lucide.Check(Class("h-6 w-6 text-green-600 dark:text-green-400")),
								),
							),
						),
					),

					// Open Rate Card
					card.Card(
						card.Content(
							Class("p-6"),
							Div(
								Class("flex items-center justify-between"),
								Div(
									Class("space-y-1"),
									P(Class("text-sm font-medium text-muted-foreground"), g.Text("Open Rate")),
									H3(
										Class("text-2xl font-bold"),
										g.Attr("x-text", "stats.openRate.toFixed(1) + '%'"),
									),
								),
								Div(
									Class("flex h-12 w-12 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/20"),
									lucide.Eye(Class("h-6 w-6 text-violet-600 dark:text-violet-400")),
								),
							),
						),
					),

					// Click Rate Card
					card.Card(
						card.Content(
							Class("p-6"),
							Div(
								Class("flex items-center justify-between"),
								Div(
									Class("space-y-1"),
									P(Class("text-sm font-medium text-muted-foreground"), g.Text("Click Rate")),
									H3(
										Class("text-2xl font-bold"),
										g.Attr("x-text", "stats.clickRate.toFixed(1) + '%'"),
									),
								),
								Div(
									Class("flex h-12 w-12 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/20"),
									lucide.MousePointer2(Class("h-6 w-6 text-amber-600 dark:text-amber-400")),
								),
							),
						),
					),
				),

				// Quick Actions Grid
				Div(
					g.Attr("x-show", "!loading"),
					Class("grid gap-4 md:grid-cols-3"),

					// Templates Card
					A(
						Href(fmt.Sprintf("%s/app/%s/notifications/templates", basePath, currentApp.ID.String())),
						card.Card(
							card.Content(
								Class("p-6 transition-shadow hover:shadow-lg"),
								Div(
									Class("flex items-center gap-4"),
									Div(
										Class("flex h-12 w-12 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/20"),
										lucide.FileText(Class("h-6 w-6 text-violet-600 dark:text-violet-400")),
									),
									Div(
										Class("space-y-1"),
										H3(Class("font-semibold"), g.Text("Templates")),
										P(Class("text-sm text-muted-foreground"), g.Text("Manage email and SMS templates")),
									),
								),
							),
						),
					),

					// History Card
					A(
						Href(fmt.Sprintf("%s/app/%s/notifications/history", basePath, currentApp.ID.String())),
						card.Card(
							card.Content(
								Class("p-6 transition-shadow hover:shadow-lg"),
								Div(
									Class("flex items-center gap-4"),
									Div(
										Class("flex h-12 w-12 items-center justify-center rounded-lg bg-amber-100 dark:bg-amber-900/20"),
										lucide.Clock(Class("h-6 w-6 text-amber-600 dark:text-amber-400")),
									),
									Div(
										Class("space-y-1"),
										H3(Class("font-semibold"), g.Text("History")),
										P(Class("text-sm text-muted-foreground"), g.Text("View sent notifications")),
									),
								),
							),
						),
					),

					// Providers Card
					A(
						Href(fmt.Sprintf("%s/app/%s/settings/notification/providers", basePath, currentApp.ID.String())),
						card.Card(
							card.Content(
								Class("p-6 transition-shadow hover:shadow-lg"),
								Div(
									Class("flex items-center gap-4"),
									Div(
										Class("flex h-12 w-12 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/20"),
										lucide.Send(Class("h-6 w-6 text-green-600 dark:text-green-400")),
									),
									Div(
										Class("space-y-1"),
										H3(Class("font-semibold"), g.Text("Providers")),
										P(Class("text-sm text-muted-foreground"), g.Text("Configure delivery providers")),
									),
								),
							),
						),
					),

					// Analytics Card
					A(
						Href(fmt.Sprintf("%s/app/%s/notifications/analytics", basePath, currentApp.ID.String())),
						card.Card(
							card.Content(
								Class("p-6 transition-shadow hover:shadow-lg"),
								Div(
									Class("flex items-center gap-4"),
									Div(
										Class("flex h-12 w-12 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/20"),
										lucide.TrendingUp(Class("h-6 w-6 text-blue-600 dark:text-blue-400")),
									),
									Div(
										Class("space-y-1"),
										H3(Class("font-semibold"), g.Text("Analytics")),
										P(Class("text-sm text-muted-foreground"), g.Text("View performance metrics")),
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

func overviewData(appID string) string {
	return `{
		stats: {
			totalSent: 0,
			totalDelivered: 0,
			totalOpened: 0,
			totalClicked: 0,
			totalBounced: 0,
			totalFailed: 0,
			deliveryRate: 0,
			openRate: 0,
			clickRate: 0,
			bounceRate: 0
		},
		loading: true,
		error: null,

		async loadStats() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('notification.getOverviewStats', {
					days: 30
				});
				if (result && result.stats) {
					this.stats = result.stats;
				} else {
					this.error = 'Invalid response format';
				}
			} catch (err) {
				console.error('Failed to load overview stats:', err);
				this.error = err.message || 'Failed to load statistics';
			} finally {
				this.loading = false;
			}
		}
	}`
}
