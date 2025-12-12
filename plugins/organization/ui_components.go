package organization

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/ui"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// TabNavigation renders a tab navigation bar for organization pages
// The activeTab parameter should match the Path of the active tab
func TabNavigation(tabs []ui.OrganizationTab, activeTab string, baseURL string) g.Node {
	if len(tabs) == 0 {
		return nil
	}

	tabItems := make([]g.Node, 0, len(tabs)+1)

	// Add "Overview" tab (always present)
	overviewActive := activeTab == "" || activeTab == "overview"
	tabItems = append(tabItems, renderTabItem("Overview", lucide.LayoutDashboard(Class("size-4")), baseURL, overviewActive))

	// Add extension tabs
	for _, tab := range tabs {
		isActive := activeTab == tab.Path
		tabURL := fmt.Sprintf("%s/tabs/%s", baseURL, tab.Path)
		tabItems = append(tabItems, renderTabItem(tab.Label, tab.Icon, tabURL, isActive))
	}

	return Div(
		Class("border-b border-slate-200 dark:border-gray-800"),
		Nav(
			Class("flex space-x-8"),
			g.Attr("aria-label", "Tabs"),
			g.Group(tabItems),
		),
	)
}

// renderTabItem renders a single tab navigation item
func renderTabItem(label string, icon g.Node, url string, isActive bool) g.Node {
	classes := "group inline-flex items-center gap-2 border-b-2 px-1 py-4 text-sm font-medium"
	if isActive {
		classes += " border-violet-500 text-violet-600 dark:text-violet-400"
	} else {
		classes += " border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700 dark:text-gray-400 dark:hover:text-gray-300"
	}

	attrs := []g.Node{
		Href(url),
		Class(classes),
		icon,
		g.Text(label),
	}
	
	if isActive {
		attrs = append(attrs, g.Attr("aria-current", "page"))
	}
	
	return A(attrs...)
}

// WidgetGrid renders a grid of organization widgets with proper sizing
// Widgets with Size 1 take 1/3 width, Size 2 takes 2/3, Size 3 takes full width
func WidgetGrid(widgets []ui.OrganizationWidget, ctx ui.OrgExtensionContext) g.Node {
	if len(widgets) == 0 {
		return nil
	}

	widgetNodes := make([]g.Node, len(widgets))
	for i, widget := range widgets {
		widgetNodes[i] = renderWidget(widget, ctx)
	}

	return Div(
		Class("grid gap-6 md:grid-cols-3"),
		g.Group(widgetNodes),
	)
}

// renderWidget renders a single widget with appropriate column span
func renderWidget(widget ui.OrganizationWidget, ctx ui.OrgExtensionContext) g.Node {
	// Determine column span based on size
	colSpan := "md:col-span-1"
	if widget.Size == 2 {
		colSpan = "md:col-span-2"
	} else if widget.Size >= 3 {
		colSpan = "md:col-span-3"
	}

	return Div(
		Class(colSpan),
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between mb-4"),
				Div(
					Class("flex items-center gap-2"),
					g.If(widget.Icon != nil, widget.Icon),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text(widget.Title)),
				),
			),
			Div(
				Class("text-slate-600 dark:text-gray-400"),
				widget.Renderer(ctx),
			),
		),
	)
}

// ActionButtons renders action buttons in the organization header
func ActionButtons(actions []ui.OrganizationAction) g.Node {
	if len(actions) == 0 {
		return nil
	}

	buttonNodes := make([]g.Node, len(actions))
	for i, action := range actions {
		buttonNodes[i] = renderActionButton(action)
	}

	return Div(
		Class("flex gap-2"),
		g.Group(buttonNodes),
	)
}

// renderActionButton renders a single action button
func renderActionButton(action ui.OrganizationAction) g.Node {
	// Determine button style classes
	var btnClasses string
	switch action.Style {
	case "primary":
		btnClasses = "rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 dark:bg-violet-500 dark:hover:bg-violet-600"
	case "danger":
		btnClasses = "rounded-lg border border-red-600 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
	default: // secondary
		btnClasses = "rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
	}

	return Button(
		Type("button"),
		Class(btnClasses+" inline-flex items-center gap-2"),
		g.Attr("onclick", action.Action),
		g.If(action.Icon != nil, action.Icon),
		g.Text(action.Label),
	)
}

// QuickLinkGrid renders a grid of quick access cards
// Requires context to build URLs
func QuickLinkGrid(links []ui.OrganizationQuickLink, basePath string, orgID, appID string) g.Node {
	if len(links) == 0 {
		return nil
	}

	linkNodes := make([]g.Node, len(links))
	for i, link := range links {
		linkNodes[i] = renderQuickLink(link, basePath, orgID, appID)
	}

	return Div(
		Class("grid gap-4 md:grid-cols-4"),
		g.Group(linkNodes),
	)
}

// renderQuickLink renders a single quick access card
func renderQuickLink(link ui.OrganizationQuickLink, basePath string, orgID, appID string) g.Node {
	// Parse IDs
	parsedOrgID, _ := parseXID(orgID)
	parsedAppID, _ := parseXID(appID)
	url := link.URLBuilder(basePath, parsedOrgID, parsedAppID)
	
	return A(
		Href(url),
		Class("group rounded-lg border border-slate-200 bg-white p-4 shadow-sm transition-all hover:shadow-md dark:border-gray-800 dark:bg-gray-900 dark:hover:border-gray-700"),
		Div(
			Class("flex items-start gap-3"),
			g.If(link.Icon != nil,
				Div(
					Class("rounded-lg bg-violet-100 p-3 dark:bg-violet-900/30"),
					link.Icon,
				),
			),
			Div(
				Class("flex-1"),
				H3(
					Class("text-sm font-semibold text-slate-900 group-hover:text-violet-600 dark:text-white dark:group-hover:text-violet-400"),
					g.Text(link.Title),
				),
				P(
					Class("mt-1 text-xs text-slate-600 dark:text-gray-400"),
					g.Text(link.Description),
				),
			),
			lucide.ChevronRight(Class("size-5 text-slate-400 transition-transform group-hover:translate-x-1")),
		),
	)
}

// MergeQuickLinks merges default quick links with extension quick links
// Returns a combined, sorted list
func MergeQuickLinks(defaultLinks []ui.OrganizationQuickLink, extensionLinks []ui.OrganizationQuickLink) []ui.OrganizationQuickLink {
	merged := make([]ui.OrganizationQuickLink, 0, len(defaultLinks)+len(extensionLinks))
	merged = append(merged, defaultLinks...)
	merged = append(merged, extensionLinks...)
	return merged
}

// EmptyStateMessage renders an empty state message for tabs with no content
func EmptyStateMessage(icon g.Node, title, description string) g.Node {
	return Div(
		Class("flex flex-col items-center justify-center py-12"),
		Div(
			Class("rounded-full bg-slate-100 p-4 dark:bg-gray-800"),
			icon,
		),
		H3(Class("mt-4 text-lg font-semibold text-slate-900 dark:text-white"), g.Text(title)),
		P(Class("mt-2 text-sm text-slate-600 dark:text-gray-400 text-center max-w-sm"), g.Text(description)),
	)
}

// ErrorWidget renders an error state for a widget
func ErrorWidget(message string) g.Node {
	return Div(
		Class("rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20"),
		Div(
			Class("flex items-center gap-2 text-red-800 dark:text-red-400"),
			lucide.X(Class("size-5")),
			Span(Class("text-sm font-medium"), g.Text(message)),
		),
	)
}

// LoadingWidget renders a loading state for a widget
func LoadingWidget() g.Node {
	return Div(
		Class("flex items-center justify-center py-8"),
		Div(
			Class("animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"),
		),
	)
}

// parseXID safely parses an xid.ID from string
func parseXID(id string) (xid.ID, error) {
	return xid.FromString(id)
}

