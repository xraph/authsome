package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// AppsListPageData holds data for the apps list page
type AppsListPageData struct {
	Apps               []*AppCardData
	BasePath           string
	CanCreateApps      bool
	ShowCreateAppCard  bool
}

// AppCardData represents data for a single app card
type AppCardData struct {
	App        *app.App
	Role       string // owner, admin, member
	MemberCount int
}

// AppsListPage renders the apps list page with cards
func AppsListPage(data AppsListPageData) g.Node {
	return Div(
		Class("w-full"),
		
		// Page Header
		Div(
			Class("mb-8"),
			H1(
				Class("text-3xl font-bold text-slate-900 dark:text-white mb-2"),
				g.Text("Your Apps"),
			),
			P(
				Class("text-slate-600 dark:text-gray-400"),
				g.Text("Select an app to manage users, sessions, and settings"),
			),
		),

		// Apps Grid
		Div(
			Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
			
			// App Cards
			g.Group(renderAppCards(data.Apps, data.BasePath)),
			
			// Create App Card (if enabled)
			g.If(data.ShowCreateAppCard,
				createAppCard(data.BasePath),
			),
		),
		
		// Empty State (if no apps)
		g.If(len(data.Apps) == 0 && !data.ShowCreateAppCard,
			appsListEmptyState(data.BasePath),
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
	roleColor := getRoleBadgeColor(data.Role)
	appURL := basePath + "/dashboard/app/" + data.App.ID.String()
	
	return Div(
		Class("card bg-base-100 dark:bg-gray-800 shadow-xl hover:shadow-2xl transition-all duration-200 hover:-translate-y-1 border border-slate-200 dark:border-gray-700"),
		Div(
			Class("card-body"),
			
			// App Icon & Title
			Div(
				Class("flex items-start justify-between mb-3"),
				Div(
					Class("flex items-center gap-3"),
					Div(
						Class("avatar placeholder"),
						Div(
							Class("bg-primary text-primary-content w-12 h-12 rounded-lg"),
							Span(
								Class("text-xl font-bold"),
								g.Text(string(data.App.Name[0])),
							),
						),
					),
					Div(
						H2(
							Class("card-title text-slate-900 dark:text-white text-lg"),
							g.Text(data.App.Name),
						),
						P(
							Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("@"+data.App.Slug),
						),
					),
				),
				// Role Badge
				Span(
					Class("badge "+roleColor+" badge-sm"),
					g.Text(data.Role),
				),
			),
			
			// App Stats
			Div(
				Class("flex items-center gap-4 py-3 text-sm text-slate-600 dark:text-gray-400"),
				Div(
					Class("flex items-center gap-2"),
					lucide.Users(Class("w-4 h-4")),
					Span(g.Textf("%d members", data.MemberCount)),
				),
				Div(
					Class("flex items-center gap-2"),
					lucide.Calendar(Class("w-4 h-4")),
					Span(g.Text(formatAppDate(data.App.CreatedAt))),
				),
			),
			
			// Actions
			Div(
				Class("card-actions justify-end mt-4 pt-4 border-t border-slate-200 dark:border-gray-700"),
				A(
					Href(appURL),
					Class("btn btn-primary btn-sm"),
					lucide.ArrowRight(Class("w-4 h-4 mr-1")),
					g.Text("Open Dashboard"),
				),
			),
		),
	)
}

func createAppCard(basePath string) g.Node {
	return Div(
		Class("card bg-base-100 dark:bg-gray-800 shadow-xl border-2 border-dashed border-slate-300 dark:border-gray-600 hover:border-primary dark:hover:border-primary transition-all duration-200"),
		Div(
			Class("card-body items-center justify-center text-center min-h-[280px]"),
			
			// Plus Icon
			Div(
				Class("w-16 h-16 rounded-full bg-primary/10 dark:bg-primary/20 flex items-center justify-center mb-4"),
				lucide.Plus(Class("w-8 h-8 text-primary")),
			),
			
			H3(
				Class("text-xl font-bold text-slate-900 dark:text-white mb-2"),
				g.Text("Create New App"),
			),
			
			P(
				Class("text-sm text-slate-600 dark:text-gray-400 mb-4"),
				g.Text("Set up a new app to manage users and authentication"),
			),
			
			// Create Button - links to multiapp plugin API or shows modal
			A(
				Href("#create-app"),
				Class("btn btn-outline btn-primary btn-sm"),
				g.Attr("onclick", "alert('Use the multiapp API to create apps: POST /api/auth/apps')"),
				lucide.Plus(Class("w-4 h-4 mr-1")),
				g.Text("Create App"),
			),
		),
	)
}

func appsListEmptyState(basePath string) g.Node {
	return Div(
		Class("col-span-full flex flex-col items-center justify-center py-16 px-4"),
		
		Div(
			Class("w-24 h-24 rounded-full bg-slate-100 dark:bg-gray-800 flex items-center justify-center mb-6"),
			lucide.Inbox(Class("w-12 h-12 text-slate-400 dark:text-gray-600")),
		),
		
		H3(
			Class("text-2xl font-bold text-slate-900 dark:text-white mb-2"),
			g.Text("No Apps Found"),
		),
		
		P(
			Class("text-slate-600 dark:text-gray-400 text-center max-w-md mb-6"),
			g.Text("You don't have access to any apps yet. Contact your administrator or create a new app to get started."),
		),
		
		A(
			Href(basePath+"/dashboard/login"),
			Class("btn btn-primary"),
			lucide.LogIn(Class("w-4 h-4 mr-2")),
			g.Text("Back to Login"),
		),
	)
}

func getRoleBadgeColor(role string) string {
	switch role {
	case "owner":
		return "badge-error"
	case "admin":
		return "badge-warning"
	case "member":
		return "badge-info"
	default:
		return "badge-ghost"
	}
}

func formatAppDate(t any) string {
	// Simple date formatting - can be enhanced
	return "Recent"
}

