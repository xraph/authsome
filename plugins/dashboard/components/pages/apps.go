package pages

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// AppsManagementData contains data for the apps management list page
type AppsManagementData struct {
	Apps          []*app.App
	Page          int
	TotalPages    int
	Total         int
	CanCreateApps bool // Based on multiapp plugin being enabled
}

// AppsManagementPage renders the apps management page (admin only)
func AppsManagementPage(data AppsManagementData, currentAppIDStr string) g.Node {
	return Div(
		Class("space-y-6"),

		// Header with Create button (conditional)
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Your Apps"),
				),
				P(
					Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Select an app to manage users, sessions, and settings"),
				),
			),
			g.If(data.CanCreateApps,
				A(
					Href(fmt.Sprintf("/dashboard/app/%s/apps-management/create", currentAppIDStr)),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 active:bg-violet-800 transition-colors"),
					lucide.Plus(Class("h-4 w-4")),
					g.Text("Create App"),
				),
			),
		),

		// Apps Grid
		appsManagementGrid(data, currentAppIDStr),
	)
}

func appsManagementGrid(data AppsManagementData, currentAppIDStr string) g.Node {
	if len(data.Apps) == 0 {
		return emptyAppsManagementState(currentAppIDStr, data.CanCreateApps)
	}

	return Div(
		Class("space-y-6"),
		// Apps Cards Grid
		Div(
			Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),

			// Render app cards
			g.Group(renderAppManagementCards(data.Apps, currentAppIDStr)),

			// Create App Card (if enabled)
			g.If(data.CanCreateApps,
				createAppManagementCard(currentAppIDStr),
			),
		),

		// Pagination (if needed)
		g.If(data.TotalPages > 1,
			appsManagementPagination(data.Page, data.TotalPages, currentAppIDStr),
		),
	)
}

func renderAppManagementCards(apps []*app.App, currentAppIDStr string) []g.Node {
	nodes := make([]g.Node, len(apps))
	for i, appItem := range apps {
		nodes[i] = appManagementCard(appItem, currentAppIDStr)
	}
	return nodes
}

func appManagementCard(appItem *app.App, currentAppIDStr string) g.Node {
	appURL := fmt.Sprintf("/dashboard/app/%s/apps-management/%s", currentAppIDStr, appItem.ID.String())
	editURL := fmt.Sprintf("/dashboard/app/%s/apps-management/%s/edit", currentAppIDStr, appItem.ID.String())

	// Generate gradient colors based on app name
	gradientClass := getAppGradient(appItem.Name)
	firstLetter := string([]rune(appItem.Name)[0])

	return A(
		Href(appURL),
		Class("group relative block rounded-2xl border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden transition-all duration-200 hover:shadow-xl hover:-translate-y-1 hover:border-violet-300 dark:hover:border-violet-700"),

		// Card Content
		Div(
			Class("p-6"),

			// Header with Icon and Edit Button
			Div(
				Class("flex items-start justify-between mb-4"),

				// App Icon
				Div(
					Class("flex items-center gap-3 flex-1 min-w-0"),
					Div(
						Class("flex-shrink-0 w-14 h-14 rounded-xl "+gradientClass+" flex items-center justify-center shadow-lg"),
						Span(
							Class("text-2xl font-bold text-white"),
							g.Text(firstLetter),
						),
					),
					Div(
						Class("flex-1 min-w-0"),
						H3(
							Class("text-lg font-semibold text-slate-900 dark:text-white truncate"),
							g.Text(appItem.Name),
						),
						P(
							Class("text-sm text-slate-500 dark:text-gray-400 truncate"),
							g.Text("@"+appItem.Slug),
						),
					),
				),

				// Edit Button (separate clickable area)
				A(
					Href(editURL),
					Class("flex-shrink-0 p-2 rounded-lg text-slate-400 hover:text-violet-600 hover:bg-violet-50 dark:hover:bg-violet-900/20 transition-colors"),
					g.Attr("onclick", "event.stopPropagation(); event.preventDefault(); window.location.href='"+editURL+"'"),
					lucide.Pencil(Class("h-4 w-4")),
				),
			),

			// Type Badge
			Div(
				Class("mb-4"),
				g.If(appItem.IsPlatform,
					Span(
						Class("inline-flex items-center gap-1.5 rounded-full bg-purple-100 dark:bg-purple-900/30 px-3 py-1 text-xs font-medium text-purple-700 dark:text-purple-400"),
						lucide.Shield(Class("h-3 w-3")),
						g.Text("Platform"),
					),
				),
				g.If(!appItem.IsPlatform,
					Span(
						Class("inline-flex items-center gap-1.5 rounded-full bg-blue-100 dark:bg-blue-900/30 px-3 py-1 text-xs font-medium text-blue-700 dark:text-blue-400"),
						lucide.Layers(Class("h-3 w-3")),
						g.Text("Tenant"),
					),
				),
			),

			// Stats
			Div(
				Class("flex items-center gap-4 text-xs text-slate-600 dark:text-gray-400 pb-4 border-b border-slate-100 dark:border-gray-800"),
				Div(
					Class("flex items-center gap-1.5"),
					lucide.Calendar(Class("h-3.5 w-3.5")),
					g.Text(formatAppMgmtTime(appItem.CreatedAt)),
				),
			),

			// Action area
			Div(
				Class("mt-4 flex items-center justify-between"),
				Span(
					Class("text-xs text-slate-500 dark:text-gray-500"),
					g.Text("ID: "+appItem.ID.String()[:8]+"..."),
				),
				Div(
					Class("flex items-center gap-1 text-violet-600 dark:text-violet-400 text-sm font-medium group-hover:gap-2 transition-all"),
					g.Text("Open Dashboard"),
					lucide.ArrowRight(Class("h-4 w-4")),
				),
			),
		),
	)
}

func createAppManagementCard(currentAppIDStr string) g.Node {
	createURL := fmt.Sprintf("/dashboard/app/%s/apps-management/create", currentAppIDStr)

	return A(
		Href(createURL),
		Class("group relative block rounded-2xl border-2 border-dashed border-slate-300 dark:border-gray-700 bg-slate-50/50 dark:bg-gray-900/50 overflow-hidden transition-all duration-200 hover:border-violet-400 dark:hover:border-violet-600 hover:bg-violet-50/50 dark:hover:bg-violet-900/10"),

		Div(
			Class("p-6 flex flex-col items-center justify-center text-center min-h-[240px]"),

			// Plus Icon
			Div(
				Class("w-16 h-16 rounded-2xl bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center mb-4 shadow-lg group-hover:scale-110 transition-transform"),
				lucide.Plus(Class("h-8 w-8 text-white")),
			),

			H3(
				Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"),
				g.Text("Create New App"),
			),

			P(
				Class("text-sm text-slate-600 dark:text-gray-400 max-w-[200px]"),
				g.Text("Set up a new app to manage users and authentication"),
			),
		),
	)
}

func emptyAppsManagementState(currentAppIDStr string, canCreate bool) g.Node {
	return Div(
		Class("flex flex-col items-center justify-center py-16 px-4 bg-slate-50 dark:bg-gray-900/50 rounded-2xl border border-slate-200 dark:border-gray-800"),

		Div(
			Class("w-20 h-20 rounded-full bg-gradient-to-br from-violet-100 to-purple-100 dark:from-violet-900/20 dark:to-purple-900/20 flex items-center justify-center mb-6"),
			lucide.Layers(Class("h-10 w-10 text-violet-600 dark:text-violet-400")),
		),

		H3(
			Class("text-xl font-semibold text-slate-900 dark:text-white mb-2"),
			g.Text("No apps"),
		),

		P(
			Class("text-sm text-slate-600 dark:text-gray-400 text-center max-w-md mb-6"),
			g.If(canCreate,
				g.Text("Get started by creating your first app to manage users and authentication."),
			),
			g.If(!canCreate,
				g.Text("Apps are managed by the system administrator. Contact your admin for access."),
			),
		),

		g.If(canCreate,
			A(
				Href(fmt.Sprintf("/dashboard/app/%s/apps-management/create", currentAppIDStr)),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-6 py-3 text-sm font-semibold text-white hover:bg-violet-700 active:bg-violet-800 transition-colors shadow-lg hover:shadow-xl"),
				lucide.Plus(Class("h-4 w-4")),
				g.Text("Create Your First App"),
			),
		),
	)
}

// AppManagementDetailData contains data for the app detail page
type AppManagementDetailData struct {
	App         *app.App
	MemberCount int
}

// AppManagementDetailPage renders the app detail page
func AppManagementDetailPage(data AppManagementDetailData, currentAppIDStr string) g.Node {
	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				A(
					Href(fmt.Sprintf("/dashboard/app/%s/apps-management", currentAppIDStr)),
					Class("inline-flex items-center gap-2 text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400 mb-4"),
					lucide.ArrowLeft(Class("h-4 w-4")),
					g.Text("Back to Apps"),
				),
				H1(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(data.App.Name),
				),
				P(
					Class("mt-1 text-sm text-slate-600 dark:text-gray-400 flex items-center gap-2"),
					Code(
						Class("rounded bg-slate-100 dark:bg-gray-800 px-2 py-1 text-xs font-mono"),
						g.Text(data.App.Slug),
					),
					g.Text("•"),
					g.If(data.App.IsPlatform,
						Span(
							Class("inline-flex items-center rounded-full bg-purple-100 dark:bg-purple-900/30 px-2.5 py-0.5 text-xs font-medium text-purple-800 dark:text-purple-400"),
							g.Text("Platform App"),
						),
					),
					g.If(!data.App.IsPlatform,
						Span(
							Class("inline-flex items-center rounded-full bg-blue-100 dark:bg-blue-900/30 px-2.5 py-0.5 text-xs font-medium text-blue-800 dark:text-blue-400"),
							g.Text("Tenant App"),
						),
					),
				),
			),
			Div(
				Class("flex items-center gap-2"),
				A(
					Href(fmt.Sprintf("/dashboard/app/%s/apps-management/%s/edit", currentAppIDStr, data.App.ID.String())),
					Class("inline-flex items-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 px-4 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
					lucide.Pencil(Class("h-4 w-4")),
					g.Text("Edit"),
				),
			),
		),

		// App Info
		Div(
			Class("rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6"),
			H2(
				Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("App Details"),
			),
			Dl(
				Class("grid grid-cols-1 gap-4 sm:grid-cols-2"),
				appManagementDetailItem("ID", data.App.ID.String()),
				appManagementDetailItem("Name", data.App.Name),
				appManagementDetailItem("Slug", data.App.Slug),
				appManagementDetailItem("Members", fmt.Sprintf("%d", data.MemberCount)),
				appManagementDetailItem("Created", formatAppMgmtTime(data.App.CreatedAt)),
				appManagementDetailItem("Updated", formatAppMgmtTime(data.App.UpdatedAt)),
			),
		),
	)
}

func appManagementDetailItem(label, value string) g.Node {
	return Div(
		Dt(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(label)),
		Dd(Class("mt-1 text-sm text-slate-900 dark:text-white"), g.Text(value)),
	)
}

// AppManagementCreatePage renders the app creation form
func AppManagementCreatePage(currentAppIDStr, csrfToken string) g.Node {
	return Div(
		Class("space-y-6"),
		Div(
			A(
				Href(fmt.Sprintf("/dashboard/app/%s/apps-management", currentAppIDStr)),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400"),
				lucide.ArrowLeft(Class("h-4 w-4")),
				g.Text("Back to Apps"),
			),
			H1(
				Class("mt-2 text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Create App"),
			),
		),
		appManagementForm(fmt.Sprintf("/dashboard/app/%s/apps-management/create", currentAppIDStr), currentAppIDStr, nil, csrfToken),
	)
}

// AppManagementEditData contains data for the app edit page
type AppManagementEditData struct {
	App *app.App
}

// AppManagementEditPage renders the app edit form
func AppManagementEditPage(data AppManagementEditData, currentAppIDStr, csrfToken string) g.Node {
	return Div(
		Class("space-y-6"),
		Div(
			A(
				Href(fmt.Sprintf("/dashboard/app/%s/apps-management/%s", currentAppIDStr, data.App.ID.String())),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400"),
				lucide.ArrowLeft(Class("h-4 w-4")),
				g.Text("Back to App"),
			),
			H1(
				Class("mt-2 text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Edit App"),
			),
		),
		appManagementForm(fmt.Sprintf("/dashboard/app/%s/apps-management/%s/edit", currentAppIDStr, data.App.ID.String()), currentAppIDStr, data.App, csrfToken),
	)
}

func appManagementForm(action string, currentAppIDStr string, appItem *app.App, csrfToken string) g.Node {
	name := ""
	slug := ""
	if appItem != nil {
		name = appItem.Name
		slug = appItem.Slug
	}

	return FormEl(
		Method("POST"),
		Action(action),
		Class("rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 space-y-6"),
		Input(
			Type("hidden"),
			Name("csrf_token"),
			Value(csrfToken),
		),
		Input(
			Type("hidden"),
			Name("redirect"),
			Value(action),
		),
		Div(
			Label(
				For("name"),
				Class("block text-sm font-medium text-slate-900 dark:text-white"),
				g.Text("Name"),
			),
			Input(
				Type("text"),
				ID("name"),
				Name("name"),
				Value(name),
				Required(),
				Class("mt-1 block w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-gray-500 focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20 dark:focus:ring-violet-400/20 sm:text-sm transition-colors"),
				g.Attr("placeholder", "My Application"),
			),
		),
		g.If(appItem == nil,
			Div(
				Label(
					For("slug"),
					Class("block text-sm font-medium text-slate-900 dark:text-white"),
					g.Text("Slug"),
				),
				Input(
					Type("text"),
					ID("slug"),
					Name("slug"),
					Value(slug),
					Required(),
					Class("mt-1 block w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-gray-500 focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20 dark:focus:ring-violet-400/20 sm:text-sm transition-colors"),
					g.Attr("placeholder", "my-app"),
				),
				P(
					Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
					g.Text("URL-friendly identifier (alphanumeric, hyphens, underscores)"),
				),
			),
		),
		Div(
			Class("flex items-center justify-end gap-3"),
			A(
				Href(func() string {
					if appItem != nil {
						return fmt.Sprintf("/dashboard/app/%s/apps-management/%s", currentAppIDStr, appItem.ID.String())
					}
					return fmt.Sprintf("/dashboard/app/%s/apps-management", currentAppIDStr)
				}()),
				Class("rounded-lg border border-slate-200 dark:border-gray-700 px-4 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-800 transition-colors"),
				g.Text("Cancel"),
			),
			Button(
				Type("submit"),
				Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 active:bg-violet-800 transition-colors"),
				g.If(appItem == nil, g.Text("Create App")),
				g.If(appItem != nil, g.Text("Update App")),
			),
		),
	)
}

func formatAppMgmtTime(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

func appsManagementPagination(page, totalPages int, currentAppIDStr string) g.Node {
	if totalPages <= 1 {
		return g.Text("")
	}

	baseURL := fmt.Sprintf("/dashboard/app/%s/apps-management", currentAppIDStr)

	return Div(
		Class("flex items-center justify-center gap-2 pt-2"),
		// Previous button
		g.If(page > 1,
			A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, page-1)),
				Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-3 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
				lucide.ChevronLeft(Class("h-4 w-4")),
				g.Text("Previous"),
			),
		),

		// Page numbers
		g.Group(func() []g.Node {
			var pages []g.Node
			for i := 1; i <= totalPages; i++ {
				if i == page {
					pages = append(pages, Span(
						Class("inline-flex items-center rounded-lg border border-violet-200 dark:border-violet-800 bg-violet-50 dark:bg-violet-900/20 px-3 py-2 text-sm font-semibold text-violet-800 dark:text-violet-400"),
						g.Text(fmt.Sprintf("%d", i)),
					))
				} else {
					pages = append(pages, A(
						Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
						Class("inline-flex items-center rounded-lg border border-slate-200 dark:border-gray-700 px-3 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
						g.Text(fmt.Sprintf("%d", i)),
					))
				}
			}
			return pages
		}()),

		// Next button
		g.If(page < totalPages,
			A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, page+1)),
				Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-3 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
				g.Text("Next"),
				lucide.ChevronRight(Class("h-4 w-4")),
			),
		),
	)
}

// getAppGradient returns a gradient class based on the app name
func getAppGradient(name string) string {
	gradients := []string{
		"bg-gradient-to-br from-violet-500 to-purple-600",
		"bg-gradient-to-br from-blue-500 to-cyan-600",
		"bg-gradient-to-br from-pink-500 to-rose-600",
		"bg-gradient-to-br from-orange-500 to-amber-600",
		"bg-gradient-to-br from-green-500 to-emerald-600",
		"bg-gradient-to-br from-indigo-500 to-blue-600",
		"bg-gradient-to-br from-red-500 to-pink-600",
		"bg-gradient-to-br from-teal-500 to-cyan-600",
	}

	// Use the first character of the name to determine gradient
	hash := 0
	for _, c := range name {
		hash += int(c)
	}

	return gradients[hash%len(gradients)]
}
