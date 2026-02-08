package pages

import (
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// AppsListPageData holds data for the apps list page
type AppsListPageData struct {
	Apps              []*AppCardData
	BasePath          string
	CanCreateApps     bool
	ShowCreateAppCard bool
}

// AppCardData represents data for a single app card
type AppCardData struct {
	App         *app.App
	Role        string // owner, admin, member
	MemberCount int
}

// AppsListPage renders the apps list page with cards
func AppsListPage(data AppsListPageData) g.Node {
	return Div(
		Class("w-full space-y-6"),

		// Page Header
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
		),

		// Apps Grid or Empty State
		g.If(len(data.Apps) > 0,
			Div(
				Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
				// App Cards
				g.Group(renderAppCards(data.Apps, data.BasePath)),
				// Create App Card (if enabled)
				g.If(data.ShowCreateAppCard,
					createAppCard(data.BasePath),
				),
			),
		),

		// Empty State (if no apps)
		g.If(len(data.Apps) == 0,
			appsListEmptyState(data.BasePath, data.ShowCreateAppCard),
		),
	)
}

func renderAppCards(apps []*AppCardData, basePath string) []g.Node {
	nodes := make([]g.Node, len(apps))
	for i, appData := range apps {
		nodes[i] = appCard(appData, basePath)
	}
	return nodes
}

func appCard(data *AppCardData, basePath string) g.Node {
	appURL := basePath + "app/" + data.App.ID.String()

	// Generate gradient colors based on app name
	gradientClass := getAppGradient(data.App.Name)
	firstLetter := string([]rune(data.App.Name)[0])

	return A(
		Href(appURL),
		Class("group relative block rounded-2xl border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden transition-all duration-200 hover:shadow-xl hover:-translate-y-1 hover:border-violet-300 dark:hover:border-violet-700"),

		// Card Content
		Div(
			Class("p-6"),

			// Header with Icon and Role Badge
			Div(
				Class("flex items-start justify-between mb-4"),

				// App Icon and Name
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
							g.Text(data.App.Name),
						),
						P(
							Class("text-sm text-slate-500 dark:text-gray-400 truncate"),
							g.Text("@"+data.App.Slug),
						),
					),
				),

				// Role Badge
				Span(
					Class(getRoleBadgeClasses(data.Role)),
					g.Text(data.Role),
				),
			),

			// Stats Row
			Div(
				Class("flex items-center gap-4 text-xs text-slate-600 dark:text-gray-400 pb-4 border-b border-slate-100 dark:border-gray-800"),
				Div(
					Class("flex items-center gap-1.5"),
					lucide.Users(Class("h-3.5 w-3.5")),
					g.Textf("%d members", data.MemberCount),
				),
				Div(
					Class("flex items-center gap-1.5"),
					lucide.Calendar(Class("h-3.5 w-3.5")),
					g.Text(formatAppDate(data.App.CreatedAt)),
				),
			),

			// Action area with arrow
			Div(
				Class("mt-4 flex items-center justify-between"),
				Span(
					Class("text-xs text-slate-500 dark:text-gray-500"),
					g.Text("ID: "+data.App.ID.String()[:8]+"..."),
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

func createAppCard(basePath string) g.Node {
	return A(
		Href("#create-app"),
		Class("group relative block rounded-2xl border-2 border-dashed border-slate-300 dark:border-gray-700 bg-slate-50/50 dark:bg-gray-900/50 overflow-hidden transition-all duration-200 hover:border-violet-400 dark:hover:border-violet-600 hover:bg-violet-50/50 dark:hover:bg-violet-900/10"),
		g.Attr("onclick", "alert('Use the multiapp API to create apps: POST /api/auth/apps')"),

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

func appsListEmptyState(basePath string, canCreate bool) g.Node {
	return Div(
		Class("flex flex-col items-center justify-center py-16 px-4 bg-slate-50 dark:bg-gray-900/50 rounded-2xl border border-slate-200 dark:border-gray-800"),

		Div(
			Class("w-20 h-20 rounded-full bg-gradient-to-br from-violet-100 to-purple-100 dark:from-violet-900/20 dark:to-purple-900/20 flex items-center justify-center mb-6"),
			lucide.Layers(Class("h-10 w-10 text-violet-600 dark:text-violet-400")),
		),

		H3(
			Class("text-xl font-semibold text-slate-900 dark:text-white mb-2"),
			g.Text("No Apps Found"),
		),

		P(
			Class("text-sm text-slate-600 dark:text-gray-400 text-center max-w-md mb-6"),
			g.If(canCreate,
				g.Text("You don't have access to any apps yet. Create your first app to get started."),
			),
			g.If(!canCreate,
				g.Text("You don't have access to any apps yet. Contact your administrator to get access."),
			),
		),

		g.If(canCreate,
			A(
				Href("#create-app"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-6 py-3 text-sm font-semibold text-white hover:bg-violet-700 active:bg-violet-800 transition-colors shadow-lg hover:shadow-xl"),
				g.Attr("onclick", "alert('Use the multiapp API to create apps: POST /api/auth/apps')"),
				lucide.Plus(Class("h-4 w-4")),
				g.Text("Create Your First App"),
			),
		),
		g.If(!canCreate,
			A(
				Href(basePath+"login"),
				Class("inline-flex items-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 px-6 py-3 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-800 transition-colors"),
				lucide.LogIn(Class("h-4 w-4")),
				g.Text("Back to Login"),
			),
		),
	)
}

func getRoleBadgeClasses(role string) string {
	switch role {
	case "owner":
		return "inline-flex items-center rounded-full bg-rose-100 dark:bg-rose-900/30 px-3 py-1 text-xs font-medium text-rose-700 dark:text-rose-400"
	case "admin":
		return "inline-flex items-center rounded-full bg-amber-100 dark:bg-amber-900/30 px-3 py-1 text-xs font-medium text-amber-700 dark:text-amber-400"
	case "member":
		return "inline-flex items-center rounded-full bg-blue-100 dark:bg-blue-900/30 px-3 py-1 text-xs font-medium text-blue-700 dark:text-blue-400"
	default:
		return "inline-flex items-center rounded-full bg-slate-100 dark:bg-slate-900/30 px-3 py-1 text-xs font-medium text-slate-700 dark:text-slate-400"
	}
}

func formatAppDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}
