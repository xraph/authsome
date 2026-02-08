package components

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SettingsNavItem represents a navigation item in the settings sidebar
type SettingsNavItem struct {
	ID            string
	Label         string
	Icon          g.Node
	URL           string
	Category      string // "general", "security", "communication", "integrations", "advanced"
	RequirePlugin string // Optional plugin requirement
}

// SettingsLayoutData contains data for the settings layout
type SettingsLayoutData struct {
	NavItems    []SettingsNavItem
	ActivePage  string
	BasePath    string
	CurrentApp  *app.App
	PageContent g.Node
}

// SettingsLayout renders the settings page with sidebar navigation
func SettingsLayout(data SettingsLayoutData) g.Node {
	return Div(
		Class("settings-layout flex min-h-screen"),
		g.Attr("x-data", "{ sidebarOpen: false }"),

		// Mobile sidebar overlay
		Div(
			g.Attr("x-show", "sidebarOpen"),
			g.Attr("x-cloak", ""),
			g.Attr("@click", "sidebarOpen = false"),
			Class("fixed inset-0 bg-gray-900/50 z-40 lg:hidden"),
		),

		// Sidebar
		settingsSidebar(data),

		// Main content area
		Div(
			Class("flex-1 flex flex-col min-w-0"),

			// Mobile header with menu button
			Div(
				Class("lg:hidden sticky top-0 z-30 flex items-center gap-x-6 bg-white dark:bg-gray-900 px-4 py-4 shadow-sm sm:px-6 border-b border-gray-200 dark:border-gray-800"),
				Button(
					Type("button"),
					g.Attr("@click", "sidebarOpen = true"),
					Class("-m-2.5 p-2.5 text-gray-700 dark:text-gray-300 lg:hidden"),
					Span(Class("sr-only"), g.Text("Open sidebar")),
					lucide.Menu(Class("h-6 w-6")),
				),
				Div(
					Class("flex-1 text-sm font-semibold leading-6 text-gray-900 dark:text-white"),
					g.Text("Settings"),
				),
			),

			// Page content
			Div(
				Class("flex-1 px-4 py-6 sm:px-6 lg:px-8"),
				data.PageContent,
			),
		),
	)
}

// settingsSidebar renders the settings navigation sidebar
func settingsSidebar(data SettingsLayoutData) g.Node {
	// Group nav items by category
	categorizedItems := make(map[string][]SettingsNavItem)
	for _, item := range data.NavItems {
		category := item.Category
		if category == "" {
			category = "general"
		}
		categorizedItems[category] = append(categorizedItems[category], item)
	}

	// Define category order and labels
	categories := []struct {
		ID    string
		Label string
	}{
		{"general", "GENERAL"},
		{"security", "SECURITY"},
		{"communication", "COMMUNICATION"},
		{"integrations", "INTEGRATIONS"},
		{"advanced", "ADVANCED"},
	}

	return Div(
		// Sidebar container - responsive
		Class("settings-sidebar"),
		g.Attr(":class", "sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'"),
		Class("fixed inset-y-0 left-0 z-50 w-72 flex flex-col bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 transition-transform duration-300 ease-in-out lg:static lg:z-auto"),

		// Sidebar header
		Div(
			Class("flex h-16 shrink-0 items-center gap-x-4 border-b border-gray-200 dark:border-gray-800 px-6"),
			lucide.Settings(Class("h-6 w-6 text-gray-400")),
			H2(
				Class("text-lg font-semibold text-gray-900 dark:text-white"),
				g.Text("Settings"),
			),
			// Close button for mobile
			Button(
				Type("button"),
				g.Attr("@click", "sidebarOpen = false"),
				Class("ml-auto lg:hidden -m-2.5 p-2.5 text-gray-400 hover:text-gray-500"),
				Span(Class("sr-only"), g.Text("Close sidebar")),
				lucide.X(Class("h-6 w-6")),
			),
		),

		// Navigation
		Nav(
			Class("flex-1 overflow-y-auto px-3 py-4"),
			g.Group(renderCategorizedNav(categories, categorizedItems, data.ActivePage)),
		),
	)
}

// renderCategorizedNav renders navigation items grouped by category
func renderCategorizedNav(categories []struct{ ID, Label string }, items map[string][]SettingsNavItem, activePage string) []g.Node {
	nodes := make([]g.Node, 0)

	for _, cat := range categories {
		catItems, exists := items[cat.ID]
		if !exists || len(catItems) == 0 {
			continue
		}

		// Category header
		nodes = append(nodes, Div(
			Class("px-3 mb-2 mt-4 first:mt-0"),
			Div(
				Class("text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider"),
				g.Text(cat.Label),
			),
		))

		// Category items
		for _, item := range catItems {
			nodes = append(nodes, renderNavItem(item, activePage))
		}
	}

	return nodes
}

// renderNavItem renders a single navigation item
func renderNavItem(item SettingsNavItem, activePage string) g.Node {
	isActive := item.ID == activePage

	return A(
		Href(item.URL),
		Class("group flex items-center gap-x-3 rounded-md px-3 py-2 text-sm font-medium transition-colors"),
		g.If(isActive,
			Class("bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400"),
		),
		g.If(!isActive,
			Class("text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white"),
		),

		// Icon
		Div(
			Class("flex-shrink-0"),
			g.If(isActive,
				g.Group([]g.Node{
					Div(Class("text-blue-600 dark:text-blue-400"), item.Icon),
				}),
			),
			g.If(!isActive,
				g.Group([]g.Node{
					Div(Class("text-gray-400 group-hover:text-gray-500 dark:group-hover:text-gray-300"), item.Icon),
				}),
			),
		),

		// Label
		Span(g.Text(item.Label)),

		// Active indicator
		g.If(isActive,
			Div(
				Class("ml-auto h-1.5 w-1.5 rounded-full bg-blue-600 dark:bg-blue-400"),
			),
		),
	)
}

// SettingsPageHeader renders a consistent header for settings pages
func SettingsPageHeader(title, description string) g.Node {
	return Div(
		Class("mb-6 pb-5 border-b border-gray-200 dark:border-gray-800"),
		H1(
			Class("text-2xl font-bold text-gray-900 dark:text-white"),
			g.Text(title),
		),
		g.If(description != "",
			P(
				Class("mt-2 text-sm text-gray-600 dark:text-gray-400"),
				g.Text(description),
			),
		),
	)
}

// SettingsSection renders a settings section card
func SettingsSection(title, description string, content g.Node) g.Node {
	return Div(
		Class("bg-white dark:bg-gray-900 shadow sm:rounded-lg border border-gray-200 dark:border-gray-800 mb-6"),

		// Section header
		g.If(title != "" || description != "",
			Div(
				Class("px-4 py-5 sm:px-6 border-b border-gray-200 dark:border-gray-800"),
				g.If(title != "",
					H3(
						Class("text-base font-semibold leading-6 text-gray-900 dark:text-white"),
						g.Text(title),
					),
				),
				g.If(description != "",
					P(
						Class("mt-1 text-sm text-gray-500 dark:text-gray-400"),
						g.Text(description),
					),
				),
			),
		),

		// Section content
		Div(
			Class("px-4 py-5 sm:p-6"),
			content,
		),
	)
}

// SettingsFormField renders a form field with label and description
func SettingsFormField(label, description, fieldID string, field g.Node) g.Node {
	return Div(
		Class("sm:grid sm:grid-cols-3 sm:items-start sm:gap-4 py-4 first:pt-0 last:pb-0"),

		// Label column
		Div(
			Class("sm:col-span-1"),
			Label(
				g.Attr("for", fieldID),
				Class("block text-sm font-medium text-gray-900 dark:text-white"),
				g.Text(label),
			),
			g.If(description != "",
				P(
					Class("mt-1 text-sm text-gray-500 dark:text-gray-400"),
					g.Text(description),
				),
			),
		),

		// Field column
		Div(
			Class("mt-2 sm:col-span-2 sm:mt-0"),
			field,
		),
	)
}

// SettingsActions renders action buttons for settings forms
func SettingsActions(saveText string, cancelURL string, extraActions ...g.Node) g.Node {
	if saveText == "" {
		saveText = "Save Changes"
	}

	return Div(
		Class("flex items-center justify-end gap-x-3 pt-5 border-t border-gray-200 dark:border-gray-800"),

		g.Group(extraActions),

		g.If(cancelURL != "",
			A(
				Href(cancelURL),
				Class("rounded-md bg-white dark:bg-gray-800 px-3 py-2 text-sm font-semibold text-gray-900 dark:text-white shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700"),
				g.Text("Cancel"),
			),
		),

		Button(
			Type("submit"),
			Class("rounded-md bg-blue-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"),
			g.Text(saveText),
		),
	)
}

// BuildSettingsURL builds a settings page URL
func BuildSettingsURL(basePath string, appID, page string) string {
	if appID == "" {
		return fmt.Sprintf("%s/settings/%s", basePath, page)
	}
	return fmt.Sprintf("%s/app/%s/settings/%s", basePath, appID, page)
}
