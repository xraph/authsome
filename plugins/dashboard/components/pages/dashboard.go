package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardStats represents statistics for the dashboard
type DashboardStats struct {
	TotalUsers     int
	ActiveUsers    int
	NewUsersToday  int
	TotalSessions  int
	ActiveSessions int
	FailedLogins   int
	UserGrowth     float64
	SessionGrowth  float64
	RecentActivity []ActivityItem
	SystemStatus   []StatusItem
	Plugins        []PluginItem
}

// ActivityItem represents a recent activity entry
type ActivityItem struct {
	Title       string
	Description string
	Time        string
	Type        string // success, warning, error, info
}

// StatusItem represents a system status entry
type StatusItem struct {
	Name   string
	Status string // operational, degraded, down
	Color  string // green, yellow, red
}

// PluginItem represents a plugin entry
type PluginItem struct {
	ID          string
	Name        string
	Description string
	Category    string
	Status      string // enabled, disabled
	Icon        string // lucide icon name
}

// DashboardPage renders the dashboard stats page content
func DashboardPage(stats *DashboardStats, basePath string, appIDStr string, extensionWidgets []g.Node) g.Node {
	return Div(
		Class("min-h-screen"),
		g.Group([]g.Node{
			// Stats Grid
			statsGrid(stats, basePath, appIDStr),

			// Content Grid
			Div(
				Class("grid grid-cols-1 gap-6 lg:grid-cols-2 mt-6"),
				recentActivityCard(stats.RecentActivity),
				systemStatusCard(stats.SystemStatus),
			),

			// Extension Widgets Section
			g.If(len(extensionWidgets) > 0,
				extensionWidgetsSection(extensionWidgets),
			),

			// Plugins Overview
			pluginsOverviewCard(stats.Plugins, basePath, appIDStr),

			// Quick Actions
			quickActionsCard(basePath, appIDStr),
		}),
	)
}

// extensionWidgetsSection renders the extension widgets in a grid
func extensionWidgetsSection(widgets []g.Node) g.Node {
	return Div(
		Class("mt-6"),
		Div(
			Class("grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3"),
			g.Group(widgets),
		),
	)
}

func statsGrid(stats *DashboardStats, basePath string, appIDStr string) g.Node {
	return Div(
		Class("grid grid-cols-1 gap-4 md:grid-cols-3 mb-6"),
		statsCard(
			"Total Users",
			stats.TotalUsers,
			stats.UserGrowth,
			basePath+"/app/"+appIDStr+"/users",
			"violet",
			lucide.Users(Class("h-6 w-6")),
		),
		statsCard(
			"Active Sessions",
			stats.ActiveSessions,
			stats.SessionGrowth,
			basePath+"/app/"+appIDStr+"/sessions",
			"emerald",
			lucide.ShieldCheck(Class("h-6 w-6")),
		),
		statsCard(
			"Failed Logins",
			stats.FailedLogins,
			0,
			"",
			"rose",
			lucide.TriangleAlert(Class("h-6 w-6")),
		),
	)
}

func statsCard(title string, value int, growth float64, href, colorScheme string, icon g.Node) g.Node {
	borderColor := "border-slate-200 dark:border-gray-800 hover:border-slate-300 dark:hover:border-gray-700"
	iconBg := "bg-slate-50 dark:bg-gray-800"
	iconBorder := "border-slate-100 dark:border-gray-700"
	iconColor := "text-slate-500 dark:text-gray-400"
	growthColor := "text-emerald-600 dark:text-emerald-400"

	if colorScheme == "violet" {
		borderColor = "border-slate-200 dark:border-gray-800 hover:border-slate-300 dark:hover:border-gray-700 active:border-violet-300 dark:active:border-violet-700"
		iconBg = "bg-violet-50 dark:bg-violet-900/20"
		iconBorder = "border-violet-100 dark:border-violet-900/30"
		iconColor = "text-violet-500 dark:text-violet-400"
	} else if colorScheme == "emerald" {
		borderColor = "border-slate-200 dark:border-gray-800 hover:border-slate-300 dark:hover:border-gray-700 active:border-emerald-300 dark:active:border-emerald-700"
		iconBg = "bg-emerald-50 dark:bg-emerald-900/20"
		iconBorder = "border-emerald-100 dark:border-emerald-900/30"
		iconColor = "text-emerald-500 dark:text-emerald-400"
	} else if colorScheme == "rose" {
		iconBg = "bg-rose-50 dark:bg-rose-900/20"
		iconBorder = "border-rose-100 dark:border-rose-900/30"
		iconColor = "text-rose-500 dark:text-rose-400"
	}

	cardElement := Div(
		Class("flex flex-col rounded-lg border "+borderColor+" bg-white dark:bg-gray-900 transition-colors"),
		Div(
			Class("flex grow items-center justify-between p-5"),
			Dl(
				Dt(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text(fmt.Sprintf("%d", value))),
				Dd(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text(title)),
			),
			Div(
				Class("flex size-12 items-center justify-center rounded-xl border "+iconBorder+" "+iconBg+" "+iconColor),
				icon,
			),
		),
		Div(
			Class("border-t border-slate-100 dark:border-gray-800 px-5 py-3 text-xs font-medium text-slate-500 dark:text-gray-400"),
			g.If(growth > 0,
				P(
					Class("inline-flex items-center gap-1 "+growthColor),
					lucide.TrendingUp(Class("h-3 w-3")),
					g.Text(fmt.Sprintf("%.1f%% growth", growth)),
				),
			),
			g.If(growth == 0 && title == "Failed Logins",
				P(g.Text("Last 24 hours")),
			),
			g.If(growth == 0 && title != "Failed Logins",
				g.Group([]g.Node{
					g.If(title == "Total Users",
						P(g.Text("All registered accounts")),
					),
					g.If(title == "Active Sessions",
						P(g.Text("Current active sessions")),
					),
				}),
			),
		),
	)

	if href != "" {
		return A(Href(href), cardElement)
	}
	return cardElement
}

func recentActivityCard(activities []ActivityItem) g.Node {
	return Div(
		Class("bg-white dark:bg-gray-800 h-[400px] rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden"),
		Div(
			Class("px-6 py-2 border-b border-gray-200 dark:border-gray-700"),
			H3(Class("text-lg font-semibold text-gray-900 dark:text-white"), g.Text("Recent Activity")),
		),
		Div(
			Class("px-6 py-5 h-full overflow-y-scroll "),
			Div(
				Class("flow-root"),
				Ul(
					g.Attr("role", "list"),
					Class("-mb-8"),
					activityList(activities),
				),
			),
		),
	)
}

func activityList(activities []ActivityItem) g.Node {
	nodes := make([]g.Node, len(activities))
	for i, activity := range activities {
		isLast := i == len(activities)-1
		nodes[i] = Li(
			Div(
				Class("relative pb-8"),
				g.If(!isLast,
					Span(Class("absolute left-4 top-4 -ml-px h-full w-0.5 bg-gray-200 dark:bg-gray-700")),
				),
				Div(
					Class("relative flex space-x-3"),
					activityIcon(activity.Type),
					Div(
						Class("flex min-w-0 flex-1 justify-between space-x-4 pt-1.5"),
						Div(
							P(Class("text-sm font-medium text-gray-900 dark:text-white"), g.Text(activity.Title)),
							P(Class("text-sm text-gray-500 dark:text-gray-400"), g.Text(activity.Description)),
						),
						Div(
							Class("whitespace-nowrap text-right text-sm text-gray-500 dark:text-gray-400"),
							g.Text(activity.Time),
						),
					),
				),
			),
		)
	}
	return g.Group(nodes)
}

func activityIcon(activityType string) g.Node {
	bgColor := "bg-blue-500"
	if activityType == "success" {
		bgColor = "bg-green-500"
	} else if activityType == "warning" {
		bgColor = "bg-yellow-500"
	} else if activityType == "error" {
		bgColor = "bg-red-500"
	}

	return Div(
		Span(
			Class("h-8 w-8 rounded-full flex items-center justify-center ring-8 ring-white dark:ring-gray-800 "+bgColor),
			lucide.CircleCheck(Class("h-5 w-5 text-white")),
		),
	)
}

func systemStatusCard(statuses []StatusItem) g.Node {
	return Div(
		Class("bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden"),
		Div(
			Class("px-6 py-2 border-b border-gray-200 dark:border-gray-700"),
			H3(Class("text-lg font-semibold text-gray-900 dark:text-white"), g.Text("System Status")),
		),
		Div(
			Class("px-6 py-5"),
			Div(
				Class("space-y-4"),
				statusList(statuses),
			),
		),
	)
}

func statusList(statuses []StatusItem) g.Node {
	nodes := make([]g.Node, len(statuses))
	for i, status := range statuses {
		nodes[i] = statusRow(status)
	}
	return g.Group(nodes)
}

func statusRow(status StatusItem) g.Node {
	dotColor := "bg-gray-500"
	badgeBg := "bg-gray-100 dark:bg-gray-500/20"
	badgeText := "text-gray-800 dark:text-gray-400"

	if status.Color == "green" {
		dotColor = "bg-green-500"
		badgeBg = "bg-green-100 dark:bg-green-500/20"
		badgeText = "text-green-800 dark:text-green-400"
	} else if status.Color == "yellow" {
		dotColor = "bg-yellow-500"
		badgeBg = "bg-yellow-100 dark:bg-yellow-500/20"
		badgeText = "text-yellow-800 dark:text-yellow-400"
	} else if status.Color == "red" {
		dotColor = "bg-red-500"
		badgeBg = "bg-red-100 dark:bg-red-500/20"
		badgeText = "text-red-800 dark:text-red-400"
	}

	return Div(
		Class("flex items-center justify-between"),
		Div(
			Class("flex items-center gap-2"),
			Span(Class("h-2 w-2 rounded-full "+dotColor)),
			Span(Class("text-sm font-medium text-gray-900 dark:text-white"), g.Text(status.Name)),
		),
		Span(
			Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "+badgeBg+" "+badgeText),
			g.Text(status.Status),
		),
	)
}

func quickActionsCard(basePath string, appIDStr string) g.Node {
	return Div(
		Class("mt-6 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden"),
		Div(
			Class("px-6 py-5 border-b border-gray-200 dark:border-gray-700"),
			H3(Class("text-lg font-semibold text-gray-900 dark:text-white"), g.Text("Quick Actions")),
		),
		Div(
			Class("px-6 py-5"),
			Div(
				Class("grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4"),
				quickActionButton("Manage Users", "View and manage accounts", basePath+"/app/"+appIDStr+"/users", "violet", lucide.Users(Class("h-6 w-6"))),
				quickActionButton("View Sessions", "Monitor active sessions", basePath+"/app/"+appIDStr+"/sessions", "emerald", lucide.ShieldCheck(Class("h-6 w-6"))),
				quickActionButton("Security Settings", "Coming soon", "", "slate", lucide.Settings(Class("h-6 w-6"))),
				quickActionButton("View Analytics", "Coming soon", "", "slate", lucide.ChartBar(Class("h-6 w-6"))),
			),
		),
	)
}

func quickActionButton(title, subtitle, href, colorScheme string, icon g.Node) g.Node {
	isDisabled := href == ""

	classes := "group relative rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 px-6 py-5 hover:border-violet-300 dark:hover:border-violet-700 hover:shadow-md transition-all flex flex-col items-center space-y-3"
	iconBg := "bg-violet-50 dark:bg-violet-900/20"
	iconHover := "group-hover:bg-violet-600 group-hover:scale-110"
	iconColor := "text-violet-600 dark:text-violet-400 group-hover:text-white"
	titleColor := "text-slate-900 dark:text-white group-hover:text-violet-600 dark:group-hover:text-violet-400"
	subtitleColor := "text-slate-500 dark:text-gray-400"

	if colorScheme == "emerald" {
		classes = "group relative rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 px-6 py-5 hover:border-emerald-300 dark:hover:border-emerald-700 hover:shadow-md transition-all flex flex-col items-center space-y-3"
		iconBg = "bg-emerald-50 dark:bg-emerald-900/20"
		iconHover = "group-hover:bg-emerald-600 group-hover:scale-110"
		iconColor = "text-emerald-600 dark:text-emerald-400 group-hover:text-white"
		titleColor = "text-slate-900 dark:text-white group-hover:text-emerald-600 dark:group-hover:text-emerald-400"
	} else if colorScheme == "slate" {
		classes = "relative rounded-lg border border-dashed border-slate-200 dark:border-gray-800 bg-slate-50 dark:bg-gray-900/50 px-6 py-5 flex flex-col items-center space-y-3 opacity-60 cursor-not-allowed"
		iconBg = "bg-slate-100 dark:bg-gray-800"
		iconHover = ""
		iconColor = "text-slate-400 dark:text-gray-500"
		titleColor = "text-slate-500 dark:text-gray-400"
		subtitleColor = "text-slate-400 dark:text-gray-500"
	}

	content := g.Group([]g.Node{
		Div(
			Class("h-12 w-12 rounded-lg "+iconBg+" flex items-center justify-center "+iconHover+" transition-all"),
			g.El("div", Class(iconColor+" transition-colors"), icon),
		),
		Div(
			Class("text-center"),
			P(Class("text-sm font-semibold "+titleColor+" transition-colors"), g.Text(title)),
			P(Class("text-xs "+subtitleColor), g.Text(subtitle)),
		),
	})

	if isDisabled {
		return Div(Class(classes), content)
	}

	return A(Href(href), Class(classes), content)
}

func pluginsOverviewCard(plugins []PluginItem, basePath string, appIDStr string) g.Node {
	// Count enabled plugins
	enabledCount := 0
	for _, p := range plugins {
		if p.Status == "enabled" {
			enabledCount++
		}
	}

	// Get up to 6 enabled plugins for preview
	enabledPlugins := make([]PluginItem, 0)
	for _, p := range plugins {
		if p.Status == "enabled" && len(enabledPlugins) < 6 {
			enabledPlugins = append(enabledPlugins, p)
		}
	}

	return Div(
		Class("mt-6 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden"),
		Div(
			Class("px-6 py-2 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between"),
			H3(Class("text-lg font-semibold text-gray-900 dark:text-white"), g.Text("Plugins")),
			Span(
				Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium bg-violet-100 dark:bg-violet-500/20 text-violet-800 dark:text-violet-400"),
				g.Textf("%d enabled", enabledCount),
			),
		),
		Div(
			Class("px-6 py-5"),
			g.If(len(enabledPlugins) > 0,
				Div(
					Class("grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4"),
					g.Group(pluginCards(enabledPlugins)),
				),
			),
			g.If(len(enabledPlugins) == 0,
				Div(
					Class("text-center py-8"),
					P(Class("text-sm text-gray-500 dark:text-gray-400"), g.Text("No plugins enabled")),
				),
			),
			Div(
				Class("mt-4 pt-4 border-t border-gray-200 dark:border-gray-700"),
				A(
					Href(basePath+"/app/"+appIDStr+"/plugins"),
					Class("inline-flex items-center gap-2 text-sm font-medium text-violet-600 dark:text-violet-400 hover:text-violet-700 dark:hover:text-violet-300 transition-colors"),
					g.Text("View all plugins"),
					lucide.ArrowRight(Class("h-4 w-4")),
				),
			),
		),
	)
}

func pluginCards(plugins []PluginItem) []g.Node {
	cards := make([]g.Node, len(plugins))
	for i, plugin := range plugins {
		cards[i] = pluginCard(plugin)
	}
	return cards
}

func pluginCard(plugin PluginItem) g.Node {
	// Map category to color
	colorClasses := map[string]string{
		"core":           "bg-violet-50 dark:bg-violet-900/20 text-violet-600 dark:text-violet-400",
		"authentication": "bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400",
		"security":       "bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400",
		"session":        "bg-amber-50 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400",
		"administration": "bg-rose-50 dark:bg-rose-900/20 text-rose-600 dark:text-rose-400",
		"communication":  "bg-sky-50 dark:bg-sky-900/20 text-sky-600 dark:text-sky-400",
		"integration":    "bg-purple-50 dark:bg-purple-900/20 text-purple-600 dark:text-purple-400",
		"enterprise":     "bg-slate-50 dark:bg-slate-900/20 text-slate-600 dark:text-slate-400",
	}

	colorClass := colorClasses[plugin.Category]
	if colorClass == "" {
		colorClass = colorClasses["core"]
	}

	// Get lucide icon
	icon := getPluginIcon(plugin.Icon)

	return Div(
		Class("flex flex-col rounded-lg border border-gray-200 dark:border-gray-700 p-4 hover:border-violet-300 dark:hover:border-violet-700 transition-colors"),
		Div(
			Class("flex items-start gap-3"),
			Div(
				Class("flex-shrink-0 h-10 w-10 rounded-lg flex items-center justify-center "+colorClass),
				icon,
			),
			Div(
				Class("flex-1 min-w-0"),
				H4(Class("text-sm font-semibold text-gray-900 dark:text-white truncate"), g.Text(plugin.Name)),
				P(Class("text-xs text-gray-500 dark:text-gray-400 mt-1 line-clamp-2"), g.Text(plugin.Description)),
			),
		),
	)
}

func getPluginIcon(iconName string) g.Node {
	// Map icon names to lucide components
	iconMap := map[string]func(children ...g.Node) g.Node{
		"LayoutDashboard": lucide.LayoutDashboard,
		"User":            lucide.User,
		"ShieldCheck":     lucide.ShieldCheck,
		"Shield":          lucide.Shield,
		"UserCircle":      lucide.User, // Use User instead
		"Building2":       lucide.Building2,
		"Mail":            lucide.Mail,
		"Link":            lucide.Link,
		"Phone":           lucide.Phone,
		"Fingerprint":     lucide.Fingerprint,
		"LogIn":           lucide.LogIn,
		"Share2":          lucide.Share2,
		"Layers":          lucide.Layers,
		"Key":             lucide.Key,
		"FileJson":        lucide.FileJson,
		"Hash":            lucide.Hash,
		"KeyRound":        lucide.KeyRound,
		"Users":           lucide.Users,
		"ShieldAlert":     lucide.ShieldAlert,
		"Bell":            lucide.Bell,
		"Server":          lucide.Server,
		"Archive":         lucide.Archive,
		"FileCheck":       lucide.FileCheck,
		"ClipboardCheck":  lucide.ClipboardCheck,
		"MapPin":          lucide.MapPin,
		"BadgeCheck":      lucide.BadgeCheck,
		"Network":         lucide.Network,
		"Lock":            lucide.Lock,
	}

	iconFunc, ok := iconMap[iconName]
	if !ok {
		iconFunc = lucide.Package // Default icon
	}

	return iconFunc(Class("h-5 w-5"))
}
