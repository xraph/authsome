package pages

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// UserDetailData contains user detail information
type UserDetailData struct {
	ID            string
	Email         string
	Name          string
	Username      string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UserDetailPageData contains data for the user detail page
type UserDetailPageData struct {
	User      UserDetailData
	Sessions  []SessionData // Active sessions for this user
	BasePath  string
	CSRFToken string
}

// UserDetailPage renders the complete user detail page
func UserDetailPage(data UserDetailPageData) g.Node {
	return Div(Class("space-y-6"),
		// Back Button
		backButton(data.BasePath),

		// User Profile Card
		userProfileCard(data.User),

		// Actions Card
		actionsCard(data),

		// Sessions Card with Real Data
		userSessionsCard(data),
	)
}

func backButton(basePath string) g.Node {
	return Div(
		A(
			Href(basePath+"/dashboard/users"),
			Class("inline-flex items-center gap-2 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"),
			lucide.ArrowLeft(Class("h-5 w-5")),
			g.Text("Back to users"),
		),
	)
}

func userProfileCard(user UserDetailData) g.Node {
	displayName := user.Email
	if user.Name != "" {
		displayName = user.Name
	}

	initial := "U"
	if len(user.Email) > 0 {
		initial = string(user.Email[0])
	}

	return Div(Class("bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden"),
		// Header with Avatar
		Div(Class("px-6 py-8 border-b border-gray-200 dark:border-gray-700"),
			Div(Class("flex items-center gap-6"),
				// Avatar
				Div(Class("h-20 w-20 flex-shrink-0"),
					Div(Class("h-20 w-20 rounded-full bg-primary/10 dark:bg-primary/20 flex items-center justify-center"),
						Span(Class("text-3xl font-semibold text-primary dark:text-primary-foreground"),
							g.Text(initial),
						),
					),
				),

				// User Info
				Div(Class("flex-1"),
					H2(Class("text-2xl font-bold text-gray-900 dark:text-white"),
						g.Text(displayName),
					),
					P(Class("mt-1 text-sm text-gray-500 dark:text-gray-400"),
						g.Text(user.Email),
					),
					Div(Class("mt-3"),
						g.If(user.EmailVerified,
							Span(Class("inline-flex items-center gap-1.5 rounded-full bg-green-50 dark:bg-green-500/20 px-3 py-1 text-xs font-medium text-green-700 dark:text-green-400 ring-1 ring-inset ring-green-600/20 dark:ring-green-500/30"),
								lucide.Check(Class("h-3.5 w-3.5")),
								g.Text("Email Verified"),
							),
						),
						g.If(!user.EmailVerified,
							Span(Class("inline-flex items-center gap-1.5 rounded-full bg-yellow-50 dark:bg-yellow-500/20 px-3 py-1 text-xs font-medium text-yellow-700 dark:text-yellow-400 ring-1 ring-inset ring-yellow-600/20 dark:ring-yellow-500/30"),
								lucide.Info(Class("h-3.5 w-3.5")),
								g.Text("Email Not Verified"),
							),
						),
					),
				),
			),
		),

		// User Details
		Div(Class("px-6 py-6"),
			H3(Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
				g.Text("Account Information"),
			),
			Dl(Class("grid grid-cols-1 gap-6 sm:grid-cols-2"),
				detailItem("User ID", user.ID, true),
				detailItem("Email Address", user.Email, false),
				detailItemDynamic("Full Name", user.Name),
				detailItemDynamic("Username", user.Username),
				detailItem("Created At", user.CreatedAt.Format("Jan 2, 2006 15:04"), false),
				detailItem("Last Updated", user.UpdatedAt.Format("Jan 2, 2006 15:04"), false),
			),
		),
	)
}

func detailItem(label, value string, isMono bool) g.Node {
	valueClass := "text-sm text-gray-900 dark:text-white"
	if isMono {
		valueClass = "text-sm text-gray-900 dark:text-white font-mono bg-gray-50 dark:bg-gray-800/50 rounded-lg px-3 py-2"
	}

	return Div(Class("space-y-1"),
		Dt(Class("text-sm font-medium text-gray-500 dark:text-gray-400"),
			g.Text(label),
		),
		Dd(Class(valueClass),
			g.Raw(value),
		),
	)
}

func detailItemDynamic(label, value string) g.Node {
	return Div(Class("space-y-1"),
		Dt(Class("text-sm font-medium text-gray-500 dark:text-gray-400"),
			g.Text(label),
		),
		Dd(Class("text-sm text-gray-900 dark:text-white"),
			g.If(value != "",
				g.Text(value),
			),
			g.If(value == "",
				Span(Class("text-gray-400 dark:text-gray-500"), g.Text("Not set")),
			),
		),
	)
}

func actionsCard(data UserDetailPageData) g.Node {
	return Div(
		Class("bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden"),
		StyleAttr("position: relative;"),
		g.Attr("x-data", "{ showDeleteModal: false }"),

		// Header
		Div(Class("px-6 py-5 border-b border-gray-200 dark:border-gray-700"),
			H3(Class("text-lg font-semibold text-gray-900 dark:text-white"),
				g.Text("Account Actions"),
			),
		),

		// Actions
		Div(Class("px-6 py-6"),
			Div(Class("grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3"),
				// Edit User
				A(
					Href(fmt.Sprintf("%s/dashboard/users/%s/edit", data.BasePath, data.User.ID)),
					Class("inline-flex items-center justify-center gap-2 rounded-lg bg-white dark:bg-gray-800/50 border border-gray-300 dark:border-gray-600 px-4 py-3 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"),
					lucide.PencilLine(Class("h-4 w-4")),
					g.Text("Edit User"),
				),

				// Reset Password (disabled)
				Button(
					Type("button"),
					Disabled(),
					Class("inline-flex items-center justify-center gap-2 rounded-lg bg-white dark:bg-gray-800/50 border border-gray-300 dark:border-gray-600 px-4 py-3 text-sm font-medium text-gray-700 dark:text-gray-300 opacity-60 cursor-not-allowed"),
					lucide.Lock(Class("h-4 w-4")),
					g.Text("Reset Password"),
					Span(Class("text-xs text-gray-500 dark:text-gray-400"), g.Text("(Soon)")),
				),

				// Delete User
				Button(
					Type("button"),
					g.Attr("@click", "showDeleteModal = true"),
					Class("inline-flex items-center justify-center gap-2 rounded-lg bg-white dark:bg-gray-800/50 border border-red-300 dark:border-red-800 px-4 py-3 text-sm font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"),
					lucide.Trash2(Class("h-4 w-4")),
					g.Text("Delete User"),
				),
			),
		),

		// Delete Modal
		deleteModal(data),
	)
}

func deleteModal(data UserDetailPageData) g.Node {
	return Div(
		g.Attr("x-show", "showDeleteModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("@keydown.escape.window", "showDeleteModal = false"),

		Div(Class("flex min-h-screen items-center justify-center p-4"),
			// Backdrop
			Div(
				g.Attr("x-show", "showDeleteModal"),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0"),
				g.Attr("x-transition:enter-end", "opacity-100"),
				g.Attr("x-transition:leave", "ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100"),
				g.Attr("x-transition:leave-end", "opacity-0"),
				Class("fixed inset-0 bg-black/50 dark:bg-black/70"),
				g.Attr("@click", "showDeleteModal = false"),
			),

			// Modal Content
			Div(
				g.Attr("x-show", "showDeleteModal"),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
				g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
				g.Attr("x-transition:leave", "ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
				g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
				Class("relative bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-md w-full p-6"),

				H3(Class("text-lg font-semibold text-gray-900 dark:text-white mb-2"),
					g.Text("Delete User"),
				),
				P(Class("text-sm text-gray-600 dark:text-gray-400 mb-6"),
					g.Text("Are you sure you want to delete "),
					Strong(Class("text-gray-900 dark:text-white"), g.Text(data.User.Email)),
					g.Text("? This action cannot be undone."),
				),

				Div(Class("flex gap-3 justify-end"),
					Button(
						Type("button"),
						g.Attr("@click", "showDeleteModal = false"),
						Class("px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"),
						g.Text("Cancel"),
					),
					FormEl(
						Method("POST"),
						Action(fmt.Sprintf("%s/dashboard/users/%s/delete", data.BasePath, data.User.ID)),
						Class("inline"),
						Input(Type("hidden"), Name("csrf_token"), Value(data.CSRFToken)),
						Button(
							Type("submit"),
							Class("px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"),
							g.Text("Delete User"),
						),
					),
				),
			),
		),
	)
}

func userSessionsCard(data UserDetailPageData) g.Node {
	// Compute plural form
	plural := "s"
	if len(data.Sessions) == 1 {
		plural = ""
	}

	return Div(Class("bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden"),
		// Header
		Div(Class("flex items-center justify-between px-6 py-5 border-b border-gray-200 dark:border-gray-700"),
			Div(
				H3(Class("text-lg font-semibold text-gray-900 dark:text-white"),
					g.Text("Active Sessions"),
				),
				P(Class("mt-1 text-sm text-gray-600 dark:text-gray-400"),
					g.Textf("%d active session%s", len(data.Sessions), plural),
				),
			),
			g.If(len(data.Sessions) > 0,
				A(
					Href(fmt.Sprintf("%s/dashboard/sessions?user_id=%s", data.BasePath, data.User.ID)),
					Class("text-sm font-medium text-violet-600 dark:text-violet-400 hover:text-violet-700 dark:hover:text-violet-300 transition-colors"),
					g.Text("View All →"),
				),
			),
		),

		// Sessions List or Empty State
		g.If(len(data.Sessions) > 0,
			userSessionsList(data),
		),
		g.If(len(data.Sessions) == 0,
			Div(Class("px-6 py-12 text-center"),
				Div(Class("mx-auto h-12 w-12 rounded-full bg-slate-100 dark:bg-gray-800 flex items-center justify-center mb-4"),
					lucide.ShieldCheck(Class("h-6 w-6 text-slate-400 dark:text-gray-500")),
				),
				H4(Class("text-sm font-semibold text-gray-900 dark:text-white"),
					g.Text("No Active Sessions"),
				),
				P(Class("mt-1 text-xs text-gray-600 dark:text-gray-400"),
					g.Text("This user has no active sessions"),
				),
			),
		),
	)
}

func userSessionsList(data UserDetailPageData) g.Node {
	items := make([]g.Node, 0, len(data.Sessions))
	for _, session := range data.Sessions {
		items = append(items, userSessionItem(session, data.BasePath, data.CSRFToken))
	}

	return Div(Class("divide-y divide-gray-100 dark:divide-gray-800"),
		g.Group(items),
	)
}

func userSessionItem(session SessionData, basePath, csrfToken string) g.Node {
	// Determine if session is current based on some criteria
	// For now, we'll show all as regular sessions

	return Div(Class("px-6 py-4 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"),
		Div(Class("flex items-start justify-between"),
			// Session Info
			Div(Class("flex-1 min-w-0"),
				// Device and Browser
				Div(Class("flex items-center gap-2 mb-2"),
					Div(Class("h-8 w-8 rounded-lg bg-violet-50 dark:bg-violet-900/20 flex items-center justify-center flex-shrink-0"),
						lucide.Monitor(Class("h-4 w-4 text-violet-600 dark:text-violet-400")),
					),
					Div(Class("flex-1 min-w-0"),
						P(Class("text-sm font-medium text-gray-900 dark:text-white truncate"),
							g.If(session.UserAgent != "",
								g.Text(truncateUserAgent(session.UserAgent)),
							),
							g.If(session.UserAgent == "",
								g.Text("Unknown Device"),
							),
						),
						P(Class("text-xs text-gray-600 dark:text-gray-400"),
							g.If(session.IPAddress != "",
								g.Text(session.IPAddress),
							),
							g.If(session.IPAddress == "",
								g.Text("Unknown IP"),
							),
						),
					),
				),

				// Timestamps
				Div(Class("flex items-center gap-4 text-xs text-gray-500 dark:text-gray-500 mt-2"),
					Div(Class("flex items-center gap-1"),
						lucide.Clock(Class("h-3 w-3")),
						g.Text("Created: "+session.CreatedAt.Format("Jan 2, 15:04")),
					),
					Div(Class("flex items-center gap-1"),
						lucide.Calendar(Class("h-3 w-3")),
						g.Text("Expires: "+session.ExpiresAt.Format("Jan 2, 15:04")),
					),
				),
			),

			// Actions
			Div(Class("flex items-center gap-2 ml-4 flex-shrink-0"),
				FormEl(
					Method("POST"),
					Action(fmt.Sprintf("%s/dashboard/sessions/%s/revoke", basePath, session.ID)),
					Class("inline"),
					Input(Type("hidden"), Name("csrf_token"), Value(csrfToken)),
					Button(
						Type("submit"),
						Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 text-xs font-medium text-slate-800 dark:text-gray-300 hover:border-rose-300 dark:hover:border-rose-700 hover:text-rose-800 dark:hover:text-rose-400 transition-colors"),
						lucide.Trash2(Class("h-3 w-3")),
						g.Text("Revoke"),
					),
				),
			),
		),
	)
}

// truncateUserAgent shortens long user agent strings
func truncateUserAgent(ua string) string {
	if len(ua) > 60 {
		return ua[:57] + "..."
	}
	return ua
}
