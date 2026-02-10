package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// SCIMOverviewPage renders the SCIM overview page with Alpine.js.
func SCIMOverviewPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()
	appBase := fmt.Sprintf("%s/app/%s", basePath, appID)

	return Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
			Div(
				H1(Class("text-2xl font-bold tracking-tight"), g.Text("SCIM Provisioning")),
				P(Class("text-muted-foreground"), g.Text("Manage identity provider integrations and user provisioning")),
			),
			Div(
				Class("flex items-center gap-2"),
				button.Button(
					Div(Class("flex items-center gap-2"), lucide.RefreshCw(Class("size-4")), g.Text("Refresh")),
					button.WithVariant("outline"),
					button.WithAttrs(g.Attr("@click", "loadOverview()")),
				),
				A(
					Href(appBase+"/scim/providers/add"),
					button.Button(
						Div(Class("flex items-center gap-2"), lucide.Plus(Class("size-4")), g.Text("Add Provider")),
						button.WithVariant("default"),
					),
				),
			),
		),

		// Alpine.js container
		Div(
			g.Attr("x-data", scimOverviewData(appID)),
			g.Attr("x-init", "loadOverview()"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-12"),
				Div(
					Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary"),
				),
			),

			// Error state
			g.El("template",
				g.Attr("x-if", "error && !loading"),
				card.Card(
					card.Content(
						Class("flex items-center gap-3 text-destructive"),
						lucide.CircleAlert(Class("size-5")),
						Span(g.Attr("x-text", "error")),
					),
				),
			),

			// Content
			Div(
				g.Attr("x-show", "!loading && !error"),
				g.Attr("x-cloak", ""),
				Class("space-y-6"),

				// Stats cards
				Div(
					Class("grid gap-4 sm:grid-cols-2 lg:grid-cols-4"),
					statsCard("Active Providers", "stats.activeProviders", "stats.totalProviders", "bg-blue-500", lucide.Server(Class("size-5 text-blue-500"))),
					statsCard("Active Tokens", "stats.activeTokens", "stats.totalTokens", "bg-emerald-500", lucide.Key(Class("size-5 text-emerald-500"))),
					statsCard("Users Provisioned", "stats.usersProvisioned", "", "bg-violet-500", lucide.Users(Class("size-5 text-violet-500"))),
					statsCard("Sync Errors", "stats.syncErrors", "", "bg-red-500", lucide.TriangleAlert(Class("size-5 text-red-500"))),
				),

				// Main content grid
				Div(
					Class("grid gap-6 lg:grid-cols-3"),

					// Providers section
					Div(
						Class("lg:col-span-2"),
						card.Card(
							card.Header(
								Class("flex flex-row items-center justify-between"),
								card.Title("Identity Providers"),
								A(
									Href(appBase+"/scim/providers"),
									Class("text-sm text-primary hover:underline"),
									g.Text("View all"),
								),
							),
							card.Content(
								// Empty state
								Div(
									g.Attr("x-show", "providers.length === 0"),
									Class("text-center py-8"),
									lucide.Server(Class("size-12 mx-auto text-muted-foreground mb-4")),
									P(Class("text-muted-foreground"), g.Text("No providers configured")),
									A(
										Href(appBase+"/scim/providers/add"),
										Class("text-sm text-primary hover:underline mt-2 inline-block"),
										g.Text("Add your first provider"),
									),
								),

								// Providers list
								Div(
									g.Attr("x-show", "providers.length > 0"),
									Class("space-y-3"),
									g.El("template",
										g.Attr("x-for", "provider in providers"),
										g.Attr(":key", "provider.id"),
										Div(
											Class("flex items-center justify-between p-3 rounded-lg border bg-card hover:bg-accent/50 transition-colors"),
											Div(
												Class("flex items-center gap-3"),
												Div(
													Class("p-2 rounded-lg bg-primary/10"),
													lucide.Server(Class("size-4 text-primary")),
												),
												Div(
													Div(
														Class("font-medium"),
														g.Attr("x-text", "provider.name"),
													),
													Div(
														Class("text-sm text-muted-foreground flex items-center gap-2"),
														Span(g.Attr("x-text", "provider.type")),
														Span(g.Text("•")),
														Span(g.Attr("x-text", "provider.userCount + ' users'")),
													),
												),
											),
											Div(
												Class("flex items-center gap-2"),
												Span(
													Class("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium"),
													g.Attr(":class", "provider.status === 'active' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' : provider.status === 'error' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' : 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'"),
													g.Attr("x-text", "provider.status"),
												),
												A(
													g.Attr(":href", fmt.Sprintf("'%s/scim/providers/' + provider.id", appBase)),
													Class("p-1.5 rounded-lg hover:bg-accent"),
													lucide.ChevronRight(Class("size-4")),
												),
											),
										),
									),
								),
							),
						),
					),

					// Quick actions & recent activity
					Div(
						Class("space-y-6"),

						// Quick actions
						card.Card(
							card.Header(
								card.Title("Quick Actions"),
							),
							card.Content(
								Class("grid gap-2"),
								A(
									Href(appBase+"/scim/providers/add"),
									Class("flex items-center gap-3 p-3 rounded-lg hover:bg-accent transition-colors"),
									Div(
										Class("p-2 rounded-lg bg-blue-100 dark:bg-blue-900/30"),
										lucide.Plus(Class("size-4 text-blue-600 dark:text-blue-400")),
									),
									Div(
										Div(Class("font-medium text-sm"), g.Text("Add Provider")),
										Div(Class("text-xs text-muted-foreground"), g.Text("Configure a new IdP")),
									),
								),
								A(
									Href(appBase+"/scim/tokens"),
									Class("flex items-center gap-3 p-3 rounded-lg hover:bg-accent transition-colors"),
									Div(
										Class("p-2 rounded-lg bg-emerald-100 dark:bg-emerald-900/30"),
										lucide.Key(Class("size-4 text-emerald-600 dark:text-emerald-400")),
									),
									Div(
										Div(Class("font-medium text-sm"), g.Text("Manage Tokens")),
										Div(Class("text-xs text-muted-foreground"), g.Text("Create or rotate tokens")),
									),
								),
								A(
									Href(appBase+"/scim/logs"),
									Class("flex items-center gap-3 p-3 rounded-lg hover:bg-accent transition-colors"),
									Div(
										Class("p-2 rounded-lg bg-violet-100 dark:bg-violet-900/30"),
										lucide.FileText(Class("size-4 text-violet-600 dark:text-violet-400")),
									),
									Div(
										Div(Class("font-medium text-sm"), g.Text("View Logs")),
										Div(Class("text-xs text-muted-foreground"), g.Text("Monitor sync events")),
									),
								),
								A(
									Href(appBase+"/scim/config"),
									Class("flex items-center gap-3 p-3 rounded-lg hover:bg-accent transition-colors"),
									Div(
										Class("p-2 rounded-lg bg-amber-100 dark:bg-amber-900/30"),
										lucide.Settings(Class("size-4 text-amber-600 dark:text-amber-400")),
									),
									Div(
										Div(Class("font-medium text-sm"), g.Text("Configuration")),
										Div(Class("text-xs text-muted-foreground"), g.Text("Adjust settings")),
									),
								),
							),
						),

						// Recent activity
						card.Card(
							card.Header(
								Class("flex flex-row items-center justify-between"),
								card.Title("Recent Activity"),
								A(
									Href(appBase+"/scim/logs"),
									Class("text-sm text-primary hover:underline"),
									g.Text("View all"),
								),
							),
							card.Content(
								// Empty state
								Div(
									g.Attr("x-show", "recentActivity.length === 0"),
									Class("text-center py-6"),
									lucide.Activity(Class("size-8 mx-auto text-muted-foreground mb-2")),
									P(Class("text-sm text-muted-foreground"), g.Text("No recent activity")),
								),

								// Activity list
								Div(
									g.Attr("x-show", "recentActivity.length > 0"),
									Class("space-y-3"),
									g.El("template",
										g.Attr("x-for", "activity in recentActivity.slice(0, 5)"),
										g.Attr(":key", "activity.id"),
										Div(
											Class("flex items-start gap-3 text-sm"),
											Div(
												Class("p-1.5 rounded-full"),
												g.Attr(":class", "activity.status === 'success' ? 'bg-emerald-100 dark:bg-emerald-900/30' : activity.status === 'error' ? 'bg-red-100 dark:bg-red-900/30' : 'bg-blue-100 dark:bg-blue-900/30'"),
												lucide.Activity(
													Class("size-3"),
													g.Attr(":class", "activity.status === 'success' ? 'text-emerald-600' : activity.status === 'error' ? 'text-red-600' : 'text-blue-600'"),
												),
											),
											Div(
												Class("flex-1 min-w-0"),
												Div(
													Class("font-medium truncate"),
													g.Attr("x-text", "activity.description"),
												),
												Div(
													Class("text-xs text-muted-foreground"),
													g.Attr("x-text", "activity.provider + ' • ' + formatRelativeTime(activity.timestamp)"),
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
		),
	)
}

// scimOverviewData returns the Alpine.js data for the overview page.
func scimOverviewData(appID string) string {
	return fmt.Sprintf(`{
		stats: {
			totalProviders: 0,
			activeProviders: 0,
			totalTokens: 0,
			activeTokens: 0,
			usersProvisioned: 0,
			groupsSynced: 0,
			lastSyncTime: '',
			syncErrors: 0
		},
		providers: [],
		recentActivity: [],
		loading: true,
		error: null,
		
		async loadOverview() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('scim.getOverview', {
					appId: '%s'
				});
				this.stats = result.stats || {};
				this.providers = result.providers || [];
				this.recentActivity = result.recentActivity || [];
			} catch (err) {
				console.error('Failed to load SCIM overview:', err);
				this.error = err.message || 'Failed to load overview';
			} finally {
				this.loading = false;
			}
		},
		
		formatRelativeTime(timestamp) {
			if (!timestamp) return '';
			const date = new Date(timestamp);
			const now = new Date();
			const diff = Math.floor((now - date) / 1000);
			
			if (diff < 60) return 'just now';
			if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
			if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
			return Math.floor(diff / 86400) + 'd ago';
		}
	}`, appID)
}

// statsCard creates a stats card component.
func statsCard(title, value, total, bgColor string, icon g.Node) g.Node {
	return card.Card(
		card.Content(
			Class("flex items-center gap-4"),
			Div(
				Class("p-3 rounded-lg bg-muted"),
				g.Group([]g.Node{icon}),
			),
			Div(
				Div(
					Class("text-2xl font-bold"),
					Span(g.Attr("x-text", value)),
					g.If(total != "", func() g.Node {
						return Span(
							Class("text-sm font-normal text-muted-foreground"),
							g.Text(" / "),
							Span(g.Attr("x-text", total)),
						)
					}()),
				),
				Div(
					Class("text-sm text-muted-foreground"),
					g.Text(title),
				),
			),
		),
	)
}
