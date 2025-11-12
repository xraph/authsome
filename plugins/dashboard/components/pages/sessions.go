package pages

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SessionData represents a user session
type SessionData struct {
	ID        string
	UserID    string
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionsPageData contains data for the sessions page
type SessionsPageData struct {
	Sessions  []SessionData
	Query     string
	BasePath  string
	CSRFToken string
	// Statistics
	AvgDuration      string // e.g., "2.5h", "45m"
	SessionsToday    int
	SessionsThisWeek int
}

// SessionsPage renders the full sessions management page
func SessionsPage(data SessionsPageData) g.Node {
	return Div(Class("space-y-6"),
		// Search Bar
		searchBar(data),

		// Sessions Table
		sessionsTable(data),

		// Statistics Cards
		statisticsCards(data),
	)
}

func searchBar(data SessionsPageData) g.Node {
	return FormEl(
		Method("GET"),
		Action(data.BasePath+"/dashboard/sessions"),
		Class("flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4"),

		// Search Input
		Div(Class("flex-1 w-full max-w-lg"),
			Label(For("search"), Class("sr-only"), g.Text("Search sessions")),
			Div(Class("relative"),
				Div(Class("pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3"),
					lucide.Search(Class("h-5 w-5 text-slate-400 dark:text-gray-500")),
				),
				Input(
					Name("q"),
					ID("search"),
					Type("search"),
					Value(data.Query),
					Placeholder("Search by user ID or IP address..."),
					Class("block w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 py-2 pl-10 pr-3 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-gray-500 focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20 dark:focus:ring-violet-400/20 sm:text-sm transition-colors"),
				),
			),
		),

		// Action Buttons
		Div(Class("flex items-center gap-2"),
			g.If(data.Query != "",
				A(
					Href(data.BasePath+"/dashboard/sessions"),
					Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 text-sm font-medium text-slate-600 dark:text-gray-400 hover:text-slate-900 dark:hover:text-white transition-colors"),
					lucide.X(Class("h-4 w-4")),
					g.Text("Clear"),
				),
			),
			Span(Class("inline-flex items-center rounded-full bg-emerald-100 dark:bg-emerald-500/20 px-3 py-1 text-xs leading-4 font-semibold text-emerald-800 dark:text-emerald-400"),
				g.Textf("%d Active", len(data.Sessions)),
			),
		),
	)
}

func sessionsTable(data SessionsPageData) g.Node {
	return Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden"),
		Div(Class("flex flex-col items-center justify-between gap-4 border-b border-slate-100 dark:border-gray-800 p-5 text-center sm:flex-row sm:text-start"),
			Div(
				H2(Class("mb-0.5 font-semibold text-slate-900 dark:text-white"),
					g.Text("Active Sessions"),
				),
				H3(Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
					g.Text("Manage and monitor user sessions across all devices"),
				),
			),
		),

		g.If(len(data.Sessions) > 0,
			Div(Class("p-5"),
				Div(Class("min-w-full overflow-x-auto rounded-sm"),
					Table(Class("min-w-full align-middle text-sm"),
						// Table Header
						THead(
							Tr(Class("border-b-2 border-slate-100 dark:border-gray-800"),
								Th(Class("min-w-[180px] py-3 pe-3 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
									g.Text("User"),
								),
								Th(Class("min-w-[140px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
									g.Text("IP Address"),
								),
								Th(Class("min-w-[180px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
									g.Text("Device"),
								),
								Th(Class("min-w-[140px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
									g.Text("Created"),
								),
								Th(Class("min-w-[140px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
									g.Text("Expires"),
								),
								Th(Class("min-w-[100px] py-2 ps-3 text-end text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
									g.Text("Actions"),
								),
							),
						),

						// Table Body
						TBody(
							g.Group(sessionRows(data)),
						),
					),
				),
			),
		),

		// Empty State
		g.If(len(data.Sessions) == 0,
			Div(Class("p-12 text-center"),
				Div(Class("mx-auto h-16 w-16 rounded-full bg-slate-100 dark:bg-gray-800 flex items-center justify-center mb-4"),
					lucide.ShieldCheck(Class("h-8 w-8 text-slate-400 dark:text-gray-500")),
				),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("No Active Sessions"),
				),
				P(Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("There are currently no active user sessions in the system."),
				),
				P(Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
					g.Text("Sessions will appear here once users log in to their accounts."),
				),
			),
		),
	)
}

func sessionRows(data SessionsPageData) []g.Node {
	rows := make([]g.Node, 0, len(data.Sessions))
	for _, session := range data.Sessions {
		rows = append(rows, sessionRow(session, data.BasePath, data.CSRFToken))
	}
	return rows
}

func sessionRow(session SessionData, basePath, csrfToken string) g.Node {
	userInitials := "US"
	if len(session.UserID) >= 2 {
		userInitials = session.UserID[0:2]
	}

	return Tr(Class("border-b border-slate-100 dark:border-gray-800 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
		// User
		Td(Class("py-3 pe-3 text-start"),
			Div(Class("flex items-center gap-3"),
				Div(Class("h-10 w-10 rounded-full bg-violet-50 dark:bg-violet-900/20 flex items-center justify-center ring-2 ring-slate-100 dark:ring-gray-800"),
					Span(Class("text-xs font-bold text-violet-600 dark:text-violet-400"),
						g.Text(userInitials),
					),
				),
				Div(
					Div(Class("font-medium text-slate-900 dark:text-white text-sm"),
						g.Text("User Account"),
					),
					Div(Class("text-xs font-mono text-slate-600 dark:text-gray-400"),
						g.Text(session.UserID),
					),
				),
			),
		),

		// IP Address
		Td(Class("p-3"),
			Span(Class("font-mono text-slate-600 dark:text-gray-400"),
				g.If(session.IPAddress != "",
					g.Text(session.IPAddress),
				),
				g.If(session.IPAddress == "",
					Span(Class("text-slate-400 dark:text-gray-500"), g.Text("N/A")),
				),
			),
		),

		// Device
		Td(Class("p-3 max-w-xs"),
			Span(
				Class("text-slate-600 dark:text-gray-400 truncate block"),
				TitleAttr(session.UserAgent),
				g.If(session.UserAgent != "",
					g.Text(session.UserAgent),
				),
				g.If(session.UserAgent == "",
					Span(Class("text-slate-400 dark:text-gray-500"), g.Text("Unknown Device")),
				),
			),
		),

		// Created
		Td(Class("p-3 text-slate-600 dark:text-gray-400"),
			g.Text(session.CreatedAt.Format("Jan 2, 2006 15:04")),
		),

		// Expires
		Td(Class("p-3 text-slate-600 dark:text-gray-400"),
			g.Text(session.ExpiresAt.Format("Jan 2, 2006 15:04")),
		),

		// Actions
		Td(Class("py-3 ps-3 text-end font-medium"),
			FormEl(
				Method("POST"),
				Action(fmt.Sprintf("%s/dashboard/sessions/%s/revoke", basePath, session.ID)),
				Class("inline"),
				Input(Type("hidden"), Name("csrf_token"), Value(csrfToken)),
				Button(
					Type("submit"),
					Class("group inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 font-medium text-slate-800 dark:text-gray-300 hover:border-rose-300 dark:hover:border-rose-700 hover:text-rose-800 dark:hover:text-rose-400 active:border-slate-200 transition-colors"),
					lucide.Trash2(Class("hi-mini hi-trash inline-block size-4 text-slate-400 dark:text-gray-500 group-hover:text-rose-600 dark:group-hover:text-rose-400 transition-colors")),
					Span(g.Text("Revoke")),
				),
			),
		),
	)
}

func statisticsCards(data SessionsPageData) g.Node {
	// Use real statistics or show "N/A" if no data
	avgDuration := data.AvgDuration
	if avgDuration == "" {
		avgDuration = "N/A"
	}

	return Div(Class("grid grid-cols-1 gap-4 md:grid-cols-3"),
		statCard(avgDuration, "Average Duration", "Per session average", lucide.Clock(Class("hi-outline hi-clock inline-block size-6")), "violet"),
		statCard(fmt.Sprintf("%d", data.SessionsToday), "Today", "New sessions today", lucide.Check(Class("hi-outline hi-check-circle inline-block size-6")), "emerald"),
		statCard(fmt.Sprintf("%d", data.SessionsThisWeek), "This Week", "Last 7 days", lucide.Calendar(Class("hi-outline hi-calendar inline-block size-6")), "blue"),
	)
}

func statCard(value, label, description string, icon g.Node, color string) g.Node {
	colorClasses := map[string][]string{
		"violet":  {"border-violet-100", "dark:border-violet-900/30", "bg-violet-50", "dark:bg-violet-900/20", "text-violet-500", "dark:text-violet-400"},
		"emerald": {"border-emerald-100", "dark:border-emerald-900/30", "bg-emerald-50", "dark:bg-emerald-900/20", "text-emerald-500", "dark:text-emerald-400"},
		"blue":    {"border-blue-100", "dark:border-blue-900/30", "bg-blue-50", "dark:bg-blue-900/20", "text-blue-500", "dark:text-blue-400"},
	}

	classes := colorClasses[color]
	iconClass := fmt.Sprintf("flex size-12 items-center justify-center rounded-xl border %s %s %s %s %s", classes[0], classes[1], classes[2], classes[3], classes[4])

	return Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900"),
		Div(Class("flex grow items-center justify-between p-5"),
			Dl(
				Dt(Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(value),
				),
				Dd(Class("text-sm font-medium text-slate-500 dark:text-gray-400"),
					g.Text(label),
				),
			),
			Div(Class(iconClass),
				icon,
			),
		),
		Div(Class("border-t border-slate-100 dark:border-gray-800 px-5 py-3 text-xs font-medium text-slate-500 dark:text-gray-400"),
			P(g.Text(description)),
		),
	)
}
