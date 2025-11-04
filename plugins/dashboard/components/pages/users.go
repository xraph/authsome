package pages

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/user"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// UsersData contains data for the users list page
type UsersData struct {
	Users      []*user.User
	Page       int
	TotalPages int
	Total      int
	Query      string
}

// UsersPage renders the users list page
func UsersPage(data UsersData, basePath string) g.Node {
	return Div(
		Class("space-y-6"),

		// Search and Actions
		searchForm(data, basePath),

		// Users Table
		usersTable(data, basePath),
	)
}

func searchForm(data UsersData, basePath string) g.Node {
	return FormEl(
		Method("GET"),
		Action(basePath+"/dashboard/users"),
		Class("flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4"),

		Div(
			Class("flex-1 w-full max-w-lg"),
			Label(For("search"), Class("sr-only"), g.Text("Search users")),
			Div(
				Class("relative"),
				Div(
					Class("pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3"),
					lucide.Search(Class("h-5 w-5 text-slate-400 dark:text-gray-500")),
				),
				Input(
					Name("q"),
					ID("search"),
					Type("search"),
					Value(data.Query),
					g.Attr("placeholder", "Search users by name or email..."),
					Class("block w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 py-2 pl-10 pr-3 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-gray-500 focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20 dark:focus:ring-violet-400/20 sm:text-sm transition-colors"),
				),
			),
		),
		Div(
			Class("flex items-center gap-2"),
			g.If(data.Query != "",
				A(
					Href(basePath+"/dashboard/users"),
					Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 text-sm font-medium text-slate-600 dark:text-gray-400 hover:text-slate-900 dark:hover:text-white transition-colors"),
					lucide.X(Class("h-4 w-4")),
					g.Text("Clear"),
				),
			),
			Span(
				Class("inline-flex items-center rounded-full bg-violet-100 dark:bg-violet-500/20 px-3 py-1 text-xs leading-4 font-semibold text-violet-800 dark:text-violet-400"),
				g.Text(fmt.Sprintf("%d %s", data.Total, pluralize(data.Total, "User", "Users"))),
			),
		),
	)
}

func usersTable(data UsersData, basePath string) g.Node {
	return Div(
		Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden"),
		g.If(len(data.Users) > 0,
			g.Group([]g.Node{
				Div(
					Class("p-5"),
					Div(
						Class("min-w-full overflow-x-auto rounded-sm"),
						Table(
							Class("min-w-full align-middle text-sm"),
							usersTableHead(),
							usersTableBody(data.Users, basePath),
						),
					),
				),
				g.If(data.TotalPages > 1,
					pagination(data, basePath),
				),
			}),
		),
		g.If(len(data.Users) == 0,
			emptyUsersState(),
		),
	)
}

func usersTableHead() g.Node {
	return THead(
		Tr(
			Class("border-b-2 border-slate-100 dark:border-gray-800"),
			Th(
				Class("min-w-[250px] py-3 pe-3 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
				g.Text("User"),
			),
			Th(
				Class("min-w-[120px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
				g.Text("Status"),
			),
			Th(
				Class("min-w-[140px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
				g.Text("Created"),
			),
			Th(
				Class("min-w-[100px] py-2 ps-3 text-end text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
				g.Text("Actions"),
			),
		),
	)
}

func usersTableBody(users []*user.User, basePath string) g.Node {
	rows := make([]g.Node, len(users))
	for i, u := range users {
		rows[i] = userTableRow(u, basePath)
	}
	return TBody(g.Group(rows))
}

func userTableRow(u *user.User, basePath string) g.Node {
	initial := "?"
	if len(u.Email) > 0 {
		initial = string(u.Email[0])
	}

	displayName := u.Name
	if displayName == "" {
		displayName = "No Name"
	}

	return Tr(
		Class("border-b border-slate-100 dark:border-gray-800 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
		Td(
			Class("py-3 pe-3 text-start"),
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("h-10 w-10 rounded-full bg-violet-50 dark:bg-violet-900/20 flex items-center justify-center ring-2 ring-slate-100 dark:ring-gray-800"),
					Span(Class("text-xs font-bold text-violet-600 dark:text-violet-400"), g.Text(initial)),
				),
				Div(
					Div(Class("font-medium text-slate-900 dark:text-white text-sm"), g.Text(displayName)),
					Div(Class("text-xs text-slate-600 dark:text-gray-400"), g.Text(u.Email)),
				),
			),
		),
		Td(
			Class("p-3"),
			g.If(u.EmailVerified,
				Span(
					Class("inline-block rounded-full bg-emerald-100 dark:bg-emerald-500/20 px-2.5 py-0.5 text-xs leading-4 font-semibold text-emerald-800 dark:text-emerald-400"),
					g.Text("Verified"),
				),
			),
			g.If(!u.EmailVerified,
				Span(
					Class("inline-block rounded-full bg-yellow-100 dark:bg-yellow-500/20 px-2.5 py-0.5 text-xs leading-4 font-semibold text-yellow-800 dark:text-yellow-400"),
					g.Text("Unverified"),
				),
			),
		),
		Td(
			Class("p-3 text-slate-600 dark:text-gray-400"),
			g.Text(formatDate(u.CreatedAt)),
		),
		Td(
			Class("py-3 ps-3 text-end font-medium"),
			A(
				Href(basePath+"/dashboard/users/"+u.ID.String()),
				Class("group inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 font-medium text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 hover:text-violet-800 dark:hover:text-violet-400 active:border-slate-200 transition-colors"),
				Span(g.Text("View")),
				lucide.ArrowRight(Class("inline-block size-4 text-slate-400 dark:text-gray-500 group-hover:text-violet-600 dark:group-hover:text-violet-400 group-active:translate-x-0.5 transition-all")),
			),
		),
	)
}

func emptyUsersState() g.Node {
	return Div(
		Class("p-12 text-center"),
		Div(
			Class("mx-auto h-16 w-16 rounded-full bg-slate-100 dark:bg-gray-800 flex items-center justify-center mb-4"),
			lucide.Users(Class("h-8 w-8 text-slate-400 dark:text-gray-500")),
		),
		H3(
			Class("text-lg font-semibold text-slate-900 dark:text-white"),
			g.Text("No Users Found"),
		),
		P(
			Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
			g.Text("There are currently no users in the system."),
		),
		P(
			Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
			g.Text("Users will appear here once they sign up."),
		),
	)
}

func pagination(data UsersData, basePath string) g.Node {
	queryParam := ""
	if data.Query != "" {
		queryParam = "&q=" + data.Query
	}

	return Div(
		Class("flex items-center justify-between border-t border-slate-100 dark:border-gray-800 bg-white dark:bg-gray-900 px-5 py-3"),

		// Mobile Pagination
		Div(
			Class("flex flex-1 justify-between sm:hidden"),
			g.If(data.Page > 1,
				A(
					Href(fmt.Sprintf("?page=%d%s", data.Page-1, queryParam)),
					Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm leading-5 font-semibold text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 active:border-slate-200 transition-colors"),
					g.Text("Previous"),
				),
			),
			g.If(data.Page < data.TotalPages,
				A(
					Href(fmt.Sprintf("?page=%d%s", data.Page+1, queryParam)),
					Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm leading-5 font-semibold text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 active:border-slate-200 transition-colors"),
					g.Text("Next"),
				),
			),
		),

		// Desktop Pagination
		Div(
			Class("hidden sm:flex sm:flex-1 sm:items-center sm:justify-between"),
			Div(
				P(
					Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Page "),
					Span(Class("font-semibold text-slate-900 dark:text-white"), g.Text(fmt.Sprintf("%d", data.Page))),
					g.Text(" of "),
					Span(Class("font-semibold text-slate-900 dark:text-white"), g.Text(fmt.Sprintf("%d", data.TotalPages))),
				),
			),
			Div(
				Nav(
					Class("isolate inline-flex -space-x-px rounded-lg shadow-sm"),
					g.If(data.Page > 1,
						A(
							Href(fmt.Sprintf("?page=%d%s", data.Page-1, queryParam)),
							Class("relative inline-flex items-center rounded-l-lg px-2 py-2 text-slate-400 dark:text-gray-500 ring-1 ring-inset ring-slate-200 dark:ring-gray-700 hover:bg-slate-50 dark:hover:bg-gray-800 focus:z-20 transition-colors"),
							Span(Class("sr-only"), g.Text("Previous")),
							lucide.ChevronLeft(Class("h-5 w-5")),
						),
					),
					paginationNumbers(data, queryParam),
					g.If(data.Page < data.TotalPages,
						A(
							Href(fmt.Sprintf("?page=%d%s", data.Page+1, queryParam)),
							Class("relative inline-flex items-center rounded-r-lg px-2 py-2 text-slate-400 dark:text-gray-500 ring-1 ring-inset ring-slate-200 dark:ring-gray-700 hover:bg-slate-50 dark:hover:bg-gray-800 focus:z-20 transition-colors"),
							Span(Class("sr-only"), g.Text("Next")),
							lucide.ChevronRight(Class("h-5 w-5")),
						),
					),
				),
			),
		),
	)
}

func paginationNumbers(data UsersData, queryParam string) g.Node {
	nodes := make([]g.Node, 0, data.TotalPages)
	for i := 0; i < data.TotalPages; i++ {
		pageNum := i + 1
		// Show first, last, and pages around current page
		if pageNum == 1 || pageNum == data.TotalPages || (pageNum >= data.Page-1 && pageNum <= data.Page+1) {
			activeClass := "text-slate-900 dark:text-gray-300 ring-1 ring-inset ring-slate-200 dark:ring-gray-700 hover:bg-slate-50 dark:hover:bg-gray-800"
			if pageNum == data.Page {
				activeClass = "z-10 bg-violet-600 dark:bg-violet-500 text-white ring-1 ring-inset ring-violet-600 dark:ring-violet-500"
			}

			nodes = append(nodes, A(
				Href(fmt.Sprintf("?page=%d%s", pageNum, queryParam)),
				Class("relative inline-flex items-center px-4 py-2 text-sm font-semibold transition-colors "+activeClass),
				g.Text(fmt.Sprintf("%d", pageNum)),
			))
		}
	}
	return g.Group(nodes)
}

// Helper functions
func formatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
