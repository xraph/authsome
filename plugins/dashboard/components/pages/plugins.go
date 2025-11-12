package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// PluginsPageData contains data for the plugins page
type PluginsPageData struct {
	Plugins        []PluginItem
	FilterStatus   string // "all", "enabled", "disabled"
	FilterCategory string
	BasePath       string
	CSRFToken      string
}

// PluginsPage renders the full plugins management page
func PluginsPage(data PluginsPageData) g.Node {
	return Div(Class("space-y-6"),
		// Page Header
		pluginsPageHeader(),

		// Filter Bar
		filterBar(data),

		// Plugin Stats
		pluginStatsCards(data.Plugins),

		// Plugins Grid
		pluginsGrid(data),
	)
}

func pluginsPageHeader() g.Node {
	return Div(Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
		Div(
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Plugins"),
			),
			P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
				g.Text("Manage and configure authentication plugins"),
			),
		),
	)
}

func filterBar(data PluginsPageData) g.Node {
	return Div(Class("flex flex-col sm:flex-row items-start sm:items-center gap-4"),
		// Status Filter
		Div(Class("flex items-center gap-2"),
			Label(For("status-filter"), Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Text("Status:"),
			),
			Select(
				ID("status-filter"),
				Name("status"),
				Class("rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-slate-900 dark:text-white focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20"),
				g.Attr("onchange", "this.form.submit()"),
				Option(Value("all"), g.If(data.FilterStatus == "all", Selected()), g.Text("All")),
				Option(Value("enabled"), g.If(data.FilterStatus == "enabled", Selected()), g.Text("Enabled")),
				Option(Value("disabled"), g.If(data.FilterStatus == "disabled", Selected()), g.Text("Disabled")),
			),
		),

		// Category Filter
		Div(Class("flex items-center gap-2"),
			Label(For("category-filter"), Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Text("Category:"),
			),
			Select(
				ID("category-filter"),
				Name("category"),
				Class("rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-slate-900 dark:text-white focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20"),
				g.Attr("onchange", "this.form.submit()"),
				Option(Value(""), g.If(data.FilterCategory == "", Selected()), g.Text("All Categories")),
				Option(Value("core"), g.If(data.FilterCategory == "core", Selected()), g.Text("Core")),
				Option(Value("authentication"), g.If(data.FilterCategory == "authentication", Selected()), g.Text("Authentication")),
				Option(Value("security"), g.If(data.FilterCategory == "security", Selected()), g.Text("Security")),
				Option(Value("session"), g.If(data.FilterCategory == "session", Selected()), g.Text("Session")),
				Option(Value("administration"), g.If(data.FilterCategory == "administration", Selected()), g.Text("Administration")),
				Option(Value("communication"), g.If(data.FilterCategory == "communication", Selected()), g.Text("Communication")),
				Option(Value("integration"), g.If(data.FilterCategory == "integration", Selected()), g.Text("Integration")),
				Option(Value("enterprise"), g.If(data.FilterCategory == "enterprise", Selected()), g.Text("Enterprise")),
			),
		),
	)
}

func pluginStatsCards(plugins []PluginItem) g.Node {
	enabledCount := 0
	disabledCount := 0
	categoryCount := make(map[string]int)

	for _, p := range plugins {
		if p.Status == "enabled" {
			enabledCount++
		} else {
			disabledCount++
		}
		categoryCount[p.Category]++
	}

	return Div(Class("grid grid-cols-1 gap-4 md:grid-cols-3"),
		pluginStatCard("Total Plugins", len(plugins), "violet"),
		pluginStatCard("Enabled", enabledCount, "emerald"),
		pluginStatCard("Disabled", disabledCount, "slate"),
	)
}

func pluginStatCard(label string, value int, color string) g.Node {
	colorClasses := map[string][]string{
		"violet":  {"border-violet-100", "dark:border-violet-900/30", "bg-violet-50", "dark:bg-violet-900/20", "text-violet-500", "dark:text-violet-400"},
		"emerald": {"border-emerald-100", "dark:border-emerald-900/30", "bg-emerald-50", "dark:bg-emerald-900/20", "text-emerald-500", "dark:text-emerald-400"},
		"slate":   {"border-slate-100", "dark:border-slate-900/30", "bg-slate-50", "dark:bg-slate-900/20", "text-slate-500", "dark:text-slate-400"},
	}

	classes := colorClasses[color]
	if classes == nil {
		classes = colorClasses["violet"]
	}

	return Div(
		Class("rounded-lg border "+classes[0]+" "+classes[1]+" p-5 bg-white dark:bg-gray-900"),
		Div(Class("flex items-center justify-between"),
			Div(
				Div(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Textf("%d", value)),
				Div(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text(label)),
			),
		),
	)
}

func pluginsGrid(data PluginsPageData) g.Node {
	// Filter plugins
	filtered := filterPlugins(data.Plugins, data.FilterStatus, data.FilterCategory)

	return Div(
		Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"),
		g.If(len(filtered) > 0,
			g.Group(fullPluginCards(filtered)),
		),
		g.If(len(filtered) == 0,
			Div(
				Class("col-span-full flex flex-col items-center justify-center py-12"),
				Div(Class("h-16 w-16 rounded-full bg-slate-100 dark:bg-gray-800 flex items-center justify-center mb-4"),
					lucide.Package(Class("h-8 w-8 text-slate-400 dark:text-gray-500")),
				),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("No plugins found")),
				P(Class("mt-2 text-sm text-slate-600 dark:text-gray-400"), g.Text("Try adjusting your filters")),
			),
		),
	)
}

func filterPlugins(plugins []PluginItem, status, category string) []PluginItem {
	filtered := make([]PluginItem, 0)

	for _, p := range plugins {
		// Filter by status
		if status != "all" && status != "" {
			if p.Status != status {
				continue
			}
		}

		// Filter by category
		if category != "" {
			if p.Category != category {
				continue
			}
		}

		filtered = append(filtered, p)
	}

	return filtered
}

func fullPluginCards(plugins []PluginItem) []g.Node {
	cards := make([]g.Node, len(plugins))
	for i, plugin := range plugins {
		cards[i] = fullPluginCard(plugin)
	}
	return cards
}

func fullPluginCard(plugin PluginItem) g.Node {
	// Map category to color
	colorClasses := map[string]string{
		"core":           "bg-violet-50 dark:bg-violet-900/20 text-violet-600 dark:text-violet-400 border-violet-200 dark:border-violet-900/30",
		"authentication": "bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 border-blue-200 dark:border-blue-900/30",
		"security":       "bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 border-emerald-200 dark:border-emerald-900/30",
		"session":        "bg-amber-50 dark:bg-amber-900/20 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-900/30",
		"administration": "bg-rose-50 dark:bg-rose-900/20 text-rose-600 dark:text-rose-400 border-rose-200 dark:border-rose-900/30",
		"communication":  "bg-sky-50 dark:bg-sky-900/20 text-sky-600 dark:text-sky-400 border-sky-200 dark:border-sky-900/30",
		"integration":    "bg-purple-50 dark:bg-purple-900/20 text-purple-600 dark:text-purple-400 border-purple-200 dark:border-purple-900/30",
		"enterprise":     "bg-slate-50 dark:bg-slate-900/20 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-900/30",
	}

	colorClass := colorClasses[plugin.Category]
	if colorClass == "" {
		colorClass = colorClasses["core"]
	}

	// Status badge
	statusBadge := g.If(plugin.Status == "enabled",
		Span(
			Class("inline-flex items-center gap-1 rounded-full bg-emerald-100 dark:bg-emerald-500/20 px-2.5 py-0.5 text-xs font-semibold text-emerald-800 dark:text-emerald-400"),
			lucide.Check(Class("h-3 w-3")),
			g.Text("Enabled"),
		),
	)

	if plugin.Status != "enabled" {
		statusBadge = Span(
			Class("inline-flex items-center gap-1 rounded-full bg-slate-100 dark:bg-slate-500/20 px-2.5 py-0.5 text-xs font-semibold text-slate-600 dark:text-slate-400"),
			lucide.Minus(Class("h-3 w-3")),
			g.Text("Disabled"),
		)
	}

	// Get lucide icon
	icon := getPluginIcon(plugin.Icon)

	return Div(
		Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:border-violet-300 dark:hover:border-violet-700 transition-all"),
		// Card Header
		Div(
			Class("p-5 border-b border-slate-100 dark:border-gray-800"),
			Div(Class("flex items-start gap-4"),
				Div(
					Class("flex-shrink-0 h-12 w-12 rounded-lg flex items-center justify-center "+colorClass),
					icon,
				),
				Div(Class("flex-1 min-w-0"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white truncate"), g.Text(plugin.Name)),
					Div(Class("mt-1 flex items-center gap-2"),
						Span(Class("text-xs font-medium text-slate-500 dark:text-gray-400 capitalize"), g.Text(plugin.Category)),
						statusBadge,
					),
				),
			),
		),

		// Card Body
		Div(
			Class("p-5 flex-1"),
			P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text(plugin.Description)),
		),

		// Card Footer
		Div(
			Class("px-5 py-3 border-t border-slate-100 dark:border-gray-800 bg-slate-50 dark:bg-gray-800/50 flex items-center justify-between"),
			Span(Class("text-xs text-slate-500 dark:text-gray-400"), g.Textf("ID: %s", plugin.ID)),
			g.If(plugin.Status == "disabled",
				Span(Class("text-xs text-slate-400 dark:text-gray-500"), g.Text("Not configured")),
			),
			g.If(plugin.Status == "enabled",
				Div(Class("flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400"),
					lucide.Activity(Class("h-3 w-3")),
					g.Text("Active"),
				),
			),
		),
	)
}
