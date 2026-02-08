package pages

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/badge"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/separator"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardHomePage shows the main dashboard for an app with all widgets
func (p *PagesManager) DashboardHomePage(ctx *router.PageContext) (g.Node, error) {
	appIDStr := ctx.Param("appId")
	if appIDStr == "" {
		// Redirect to index
		ctx.SetHeader("Location", p.baseUIPath)
		ctx.ResponseWriter.WriteHeader(302)
		return nil, nil
	}

	// Parse app ID and get current app for extension widgets
	var currentApp *app.App
	appID, err := xid.FromString(appIDStr)
	if err == nil && p.services != nil {
		appSvc := p.services.AppService()
		if appSvc != nil {
			goCtx := ctx.Context()
			currentApp, _ = appSvc.FindAppByID(goCtx, appID)
		}
	}

	// Fetch extension widgets server-side
	var extensionWidgets []g.Node
	if p.extensionRegistry != nil {
		widgets := p.extensionRegistry.GetDashboardWidgets()
		for _, widget := range widgets {
			if widget.Renderer != nil {
				// Render widget server-side with currentApp
				extensionWidgets = append(extensionWidgets, widget.Renderer(p.baseUIPath, currentApp))
			}
		}
	}

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				appId: '`+appIDStr+`',
				loading: {
					stats: true,
					activity: true,
					systemStatus: true,
					plugins: true
				},
				stats: {
					totalUsers: 0,
					activeUsers: 0,
					totalSessions: 0,
					activeSessions: 0,
					newUsersToday: 0,
					newUsersWeek: 0,
					growthRate: 0,
					userGrowthData: []
				},
				activities: [],
				systemStatus: [],
				plugins: {
					enabledCount: 0,
					totalCount: 0,
					plugins: []
				},
				error: {
					stats: null,
					activity: null,
					systemStatus: null,
					plugins: null
				},
				async init() {
					await Promise.all([
						this.loadStats(),
						this.loadActivity(),
						this.loadSystemStatus(),
						this.loadPlugins()
					]);
				},
				async loadStats() {
					this.loading.stats = true;
					this.error.stats = null;
					try {
						const result = await $go('getDashboardStats', { appId: this.appId });
						this.stats = result;
					} catch (err) {
						console.error('Failed to load stats:', err);
						this.error.stats = err.message || 'Failed to load statistics';
					} finally {
						this.loading.stats = false;
					}
				},
				async loadActivity() {
					this.loading.activity = true;
					this.error.activity = null;
					try {
						const result = await $go('getRecentActivity', { 
							appId: this.appId,
							limit: 10
						});
						this.activities = result.activities || [];
					} catch (err) {
						console.error('Failed to load activity:', err);
						this.error.activity = err.message || 'Failed to load activity';
					} finally {
						this.loading.activity = false;
					}
				},
				async loadSystemStatus() {
					this.loading.systemStatus = true;
					this.error.systemStatus = null;
					try {
						const result = await $go('getSystemStatus', { appId: this.appId });
						this.systemStatus = result.components || [];
					} catch (err) {
						console.error('Failed to load system status:', err);
						this.error.systemStatus = err.message || 'Failed to load status';
					} finally {
						this.loading.systemStatus = false;
					}
				},
				async loadPlugins() {
					this.loading.plugins = true;
					this.error.plugins = null;
					try {
						const result = await $go('getPluginsOverview', { appId: this.appId });
						this.plugins = result;
					} catch (err) {
						console.error('Failed to load plugins:', err);
						this.error.plugins = err.message || 'Failed to load plugins';
					} finally {
						this.loading.plugins = false;
					}
				},
				async refresh() {
					await Promise.all([
						this.loadStats(),
						this.loadActivity(),
						this.loadSystemStatus(),
						this.loadPlugins()
					]);
				},
				formatTimestamp(timestamp) {
					const date = new Date(timestamp);
					const now = new Date();
					const diff = now - date;
					const seconds = Math.floor(diff / 1000);
					const minutes = Math.floor(seconds / 60);
					const hours = Math.floor(minutes / 60);
					const days = Math.floor(hours / 24);
					
					if (seconds < 60) return 'just now';
					if (minutes < 60) return minutes + 'm ago';
					if (hours < 24) return hours + 'h ago';
					if (days < 7) return days + 'd ago';
					return date.toLocaleDateString();
				}
			}`),
			g.Attr("x-init", "init()"),
			// Header with title and refresh button
			Div(
				Class("flex items-center justify-between mb-8"),
				Div(
					H1(
						Class("text-2xl font-semibold tracking-tight text-foreground"),
						g.Text("Dashboard"),
					),
					P(
						Class("text-sm text-muted-foreground mt-1"),
						g.Text("Monitor your application performance and activity"),
					),
				),
				button.Button(
					g.Group([]g.Node{
						icons.RefreshCw(icons.WithSize(16)),
						Span(g.Text("Refresh")),
					}),
					button.WithVariant("outline"),
					button.WithSize("sm"),
					button.WithClass("gap-2"),
					button.WithAttrs(g.Attr("@click", "refresh()")),
				),
			),

			Div(
				Class("space-y-2"),

				// Stats Grid
				p.renderStatsGrid(),

				// Activity and Status Row
				Div(
					Class("grid grid-cols-1 lg:grid-cols-2 gap-6"),
					p.renderRecentActivity(),
					p.renderSystemStatus(),
				),

				// Growth Chart
				p.renderGrowthChart(),

				// Extension Widgets (if any) - rendered server-side
				g.If(len(extensionWidgets) > 0,
					Div(
						Class("grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4"),
						g.Group(extensionWidgets),
					),
				),

				// Plugins Overview
				p.renderPluginsOverview(),

				// Quick Actions
				p.renderQuickActions(appIDStr),
			),
		),
	), nil
}

// renderStatsGrid renders the statistics cards grid
func (p *PagesManager) renderStatsGrid() g.Node {
	return Div(
		Class("grid gap-4 md:grid-cols-3"),

		// Total Users Card
		p.renderStatCard(
			"Total Users",
			"stats.totalUsers",
			icons.Users(icons.WithSize(18)),
			"All time",
		),

		// Active Sessions Card
		p.renderStatCard(
			"Active Sessions",
			"stats.activeSessions",
			icons.Activity(icons.WithSize(18)),
			"Current",
		),

		// New Users Card
		p.renderStatCard(
			"New Today",
			"stats.newUsersToday",
			icons.TrendingUp(icons.WithSize(18)),
			"Last 24h",
		),
	)
}

// renderStatCard renders a single statistics card with minimal design
func (p *PagesManager) renderStatCard(title, valueBinding string, icon g.Node, subtitle string) g.Node {
	return card.Card(
		card.Header(
			Div(
				Class("flex flex-row items-center justify-between space-y-0 pb-2"),
				card.Title(title, card.WithClass("text-sm font-medium text-muted-foreground")),
				Div(
					Class("text-muted-foreground"),
					icon,
				),
			),
		),
		card.Content(
			Div(
				Class("space-y-1"),
				// Value
				Div(
					Class("text-2xl font-bold tracking-tight"),
					g.Attr("x-show", "!loading.stats"),
					g.Attr("x-text", valueBinding),
				),
				// Loading skeleton
				Div(
					g.Attr("x-show", "loading.stats"),
					Class("h-8 w-16 bg-muted animate-pulse rounded"),
				),
				// Subtitle
				P(
					Class("text-xs text-muted-foreground"),
					g.Text(subtitle),
				),
			),
		),
	)
}

// renderRecentActivity renders the recent activity widget
func (p *PagesManager) renderRecentActivity() g.Node {
	return card.Card(
		card.Header(
			card.Title("Recent Activity", card.WithClass("text-base font-semibold")),
			card.Description("Latest events and actions"),
		),
		card.Content(
			Div(
				Class("h-80 overflow-y-auto"),

				// Loading state
				Div(
					g.Attr("x-show", "loading.activity"),
					Class("space-y-3"),
					g.Group([]g.Node{
						p.renderLoadingSkeleton(),
						p.renderLoadingSkeleton(),
						p.renderLoadingSkeleton(),
					}),
				),

				// Error state
				Div(
					g.Attr("x-show", "!loading.activity && error.activity"),
					Class("text-center py-8"),
					p.renderErrorState("error.activity", "loadActivity()"),
				),

				// Empty state
				Div(
					g.Attr("x-show", "!loading.activity && !error.activity && activities.length === 0"),
					Class("flex flex-col items-center justify-center py-8 text-center"),
					icons.Inbox(icons.WithSize(32), icons.WithClass("text-muted-foreground mb-2")),
					P(
						Class("text-sm text-muted-foreground"),
						g.Text("No recent activity"),
					),
				),

				// Activity list
				Div(
					g.Attr("x-show", "!loading.activity && activities.length > 0"),
					Class("space-y-2"),
					g.El("template", g.Attr("x-for", "activity in activities"),
						Div(
							Class("flex items-start gap-3 p-2 rounded-md hover:bg-accent transition-colors"),
							Div(
								Class("flex-shrink-0 w-8 h-8 rounded-full bg-muted flex items-center justify-center"),
								icons.Activity(icons.WithSize(14), icons.WithClass("text-muted-foreground")),
							),
							Div(
								Class("flex-1 min-w-0"),
								P(
									Class("text-sm font-medium leading-none"),
									g.Attr("x-text", "activity.description"),
								),
								P(
									Class("text-xs text-muted-foreground mt-1"),
									g.Attr("x-text", "activity.userEmail || 'System'"),
								),
							),
							Div(
								Class("text-xs text-muted-foreground whitespace-nowrap"),
								g.Attr("x-text", "formatTimestamp(activity.timestamp)"),
							),
						),
					),
				),
			),
		),
	)
}

// renderSystemStatus renders the system status widget
func (p *PagesManager) renderSystemStatus() g.Node {
	return card.Card(
		card.Header(
			card.Title("System Status", card.WithClass("text-base font-semibold")),
			card.Description("Service health monitoring"),
		),
		card.Content(

			// Loading state
			Div(
				g.Attr("x-show", "loading.systemStatus"),
				Class("space-y-3"),
				g.Group([]g.Node{
					p.renderLoadingSkeleton(),
					p.renderLoadingSkeleton(),
				}),
			),

			// Error state
			Div(
				g.Attr("x-show", "!loading.systemStatus && error.systemStatus"),
				Class("text-center py-8"),
				p.renderErrorState("error.systemStatus", "loadSystemStatus()"),
			),

			// Status list
			Div(
				g.Attr("x-show", "!loading.systemStatus && systemStatus.length > 0"),
				Class("space-y-3"),
				g.El("template", g.Attr("x-for", "component in systemStatus"),
					Div(
						Class("flex items-center justify-between py-2"),
						Div(
							Class("flex items-center gap-2"),
							Div(
								g.Attr(":class", `{
									'h-2 w-2 rounded-full': true,
									'bg-green-500': component.color === 'green',
									'bg-yellow-500': component.color === 'yellow',
									'bg-destructive': component.color === 'red'
								}`),
							),
							Span(
								Class("text-sm font-medium"),
								g.Attr("x-text", "component.name"),
							),
						),
						Div(
							Span(
								g.Attr("x-show", "component.color === 'green'"),
								badge.Badge(
									"Operational",
									badge.WithVariant("secondary"),
									badge.WithClass("text-xs"),
								),
							),
							Span(
								g.Attr("x-show", "component.color === 'yellow'"),
								badge.Badge(
									"Degraded",
									badge.WithVariant("outline"),
									badge.WithClass("text-xs border-yellow-500 text-yellow-600 dark:text-yellow-400"),
								),
							),
							Span(
								g.Attr("x-show", "component.color === 'red'"),
								badge.Badge(
									"Down",
									badge.WithVariant("destructive"),
									badge.WithClass("text-xs"),
								),
							),
						),
					),
				),
			),
		),
	)
}

// renderGrowthChart renders the user growth chart widget
func (p *PagesManager) renderGrowthChart() g.Node {
	return card.Card(
		card.Header(
			card.Title("User Growth", card.WithClass("text-base font-semibold")),
			card.Description("User registration trend"),
		),
		card.Content(
			Div(
				Class("h-64 flex items-center justify-center"),
				Div(
					Class("text-center"),
					Div(
						Class("inline-flex items-center justify-center w-12 h-12 rounded-full bg-muted mb-3"),
						icons.TrendingUp(icons.WithSize(24), icons.WithClass("text-muted-foreground")),
					),
					P(
						Class("text-sm font-medium mb-1"),
						g.Text("Chart Visualization"),
					),
					P(
						Class("text-xs text-muted-foreground"),
						g.Text("Integrate charting library for visual data"),
					),
				),
			),
		),
	)
}

// renderPluginsOverview renders the plugins overview widget
func (p *PagesManager) renderPluginsOverview() g.Node {
	return card.Card(
		card.Header(
			Div(
				Class("flex flex-row items-center justify-between"),
				card.Title("Plugins", card.WithClass("text-base font-semibold")),
				Span(
					g.Attr("x-show", "!loading.plugins"),
					Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium bg-secondary text-secondary-foreground"),
					g.Attr("x-text", "plugins.enabledCount + ' enabled'"),
				),
			),
		),
		card.Content(

			// Loading state
			Div(
				g.Attr("x-show", "loading.plugins"),
				Class("grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3"),
				g.Group([]g.Node{
					p.renderLoadingSkeleton(),
					p.renderLoadingSkeleton(),
					p.renderLoadingSkeleton(),
				}),
			),

			// Error state
			Div(
				g.Attr("x-show", "!loading.plugins && error.plugins"),
				Class("text-center py-8"),
				p.renderErrorState("error.plugins", "loadPlugins()"),
			),

			// Plugins grid (show first 6)
			Div(
				g.Attr("x-show", "!loading.plugins && plugins.plugins.length > 0"),
				Div(
					Class("grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3"),
					g.El("template", g.Attr("x-for", "(plugin, index) in plugins.plugins.slice(0, 6)"),
						Div(
							Class("flex gap-3 p-3 rounded-lg border hover:bg-accent transition-colors"),
							Div(
								Class("flex-shrink-0 w-9 h-9 rounded-md bg-muted flex items-center justify-center"),
								icons.Box(icons.WithSize(16), icons.WithClass("text-muted-foreground")),
							),
							Div(
								Class("flex-1 min-w-0"),
								H4(
									Class("text-sm font-medium truncate"),
									g.Attr("x-text", "plugin.name"),
								),
								P(
									Class("text-xs text-muted-foreground line-clamp-1"),
									g.Attr("x-text", "plugin.description"),
								),
							),
						),
					),
				),
				separator.Separator(separator.WithClass("my-4")),
				A(
					g.Attr(":href", "`"+p.baseUIPath+"/app/${appId}/plugins`"),
					Class("inline-flex items-center gap-1 text-sm font-medium hover:underline"),
					g.Text("View all plugins"),
					icons.ArrowRight(icons.WithSize(14)),
				),
			),

			// Empty state
			Div(
				g.Attr("x-show", "!loading.plugins && plugins.plugins.length === 0"),
				Class("flex flex-col items-center justify-center py-8 text-center"),
				icons.Box(icons.WithSize(32), icons.WithClass("text-muted-foreground mb-2")),
				P(
					Class("text-sm text-muted-foreground"),
					g.Text("No plugins enabled"),
				),
			),
		),
	)
}

// renderQuickActions renders the quick actions widget
func (p *PagesManager) renderQuickActions(appID string) g.Node {
	return card.Card(
		card.Header(
			card.Title("Quick Actions", card.WithClass("text-base font-semibold")),
			card.Description("Common tasks and shortcuts"),
		),
		card.Content(
			Div(
				Class("grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3"),
				p.renderQuickActionCard(
					"Manage Users",
					"View and manage accounts",
					icons.Users(icons.WithSize(18)),
					p.baseUIPath+"/app/"+appID+"/users",
					false,
				),
				p.renderQuickActionCard(
					"View Sessions",
					"Monitor active sessions",
					icons.Activity(icons.WithSize(18)),
					p.baseUIPath+"/app/"+appID+"/sessions",
					false,
				),
				p.renderQuickActionCard(
					"Security Settings",
					"Coming soon",
					icons.Settings(icons.WithSize(18)),
					"",
					true,
				),
				p.renderQuickActionCard(
					"View Analytics",
					"Coming soon",
					icons.TrendingUp(icons.WithSize(18)),
					"",
					true,
				),
			),
		),
	)
}

// renderQuickActionCard renders a single quick action card with minimal design
func (p *PagesManager) renderQuickActionCard(title, subtitle string, icon g.Node, href string, disabled bool) g.Node {
	content := Div(
		Class("flex flex-col gap-2 p-4 rounded-lg border hover:bg-accent transition-colors"),
		g.If(disabled, g.Attr("class", "flex flex-col gap-2 p-4 rounded-lg border border-dashed opacity-50 cursor-not-allowed")),
		Div(
			Class("flex items-center gap-3"),
			Div(
				Class("flex-shrink-0 w-9 h-9 rounded-md bg-muted flex items-center justify-center"),
				Div(Class("text-muted-foreground"), icon),
			),
			Div(
				Class("flex-1 min-w-0"),
				P(
					Class("text-sm font-medium"),
					g.Text(title),
				),
				P(
					Class("text-xs text-muted-foreground"),
					g.Text(subtitle),
				),
			),
		),
	)

	if disabled {
		return content
	}

	return A(
		Href(href),
		content,
	)
}

// renderLoadingSkeleton renders a loading skeleton with minimal design
func (p *PagesManager) renderLoadingSkeleton() g.Node {
	return Div(
		Class("animate-pulse space-y-2"),
		Div(
			Class("h-4 bg-muted rounded w-3/4"),
		),
		Div(
			Class("h-3 bg-muted rounded w-1/2"),
		),
	)
}

// renderErrorState renders an error state with retry button
func (p *PagesManager) renderErrorState(errorBinding, retryFunc string) g.Node {
	return Div(
		Class("flex flex-col items-center justify-center"),
		Div(
			Class("inline-flex items-center justify-center w-12 h-12 rounded-full bg-destructive/10 mb-3"),
			icons.AlertCircle(icons.WithSize(24), icons.WithClass("text-destructive")),
		),
		P(
			Class("text-sm text-muted-foreground mb-3"),
			g.Attr("x-text", errorBinding),
		),
		button.Button(
			g.Text("Retry"),
			button.WithVariant("outline"),
			button.WithSize("sm"),
			button.WithAttrs(g.Attr("@click", retryFunc)),
		),
	)
}
