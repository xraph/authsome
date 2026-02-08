package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// UserEditData contains data for editing a user
type UserEditData struct {
	UserID        string
	Name          string
	Email         string
	Username      string
	EmailVerified bool
}

// UserEditPageData contains data for the user edit page
type UserEditPageData struct {
	User      UserEditData
	BasePath  string
	CSRFToken string
}

// UserEditPage renders the complete user edit page
func UserEditPage(data UserEditPageData) g.Node {
	return Div(Class("max-w-4xl"),
		// Header
		editHeader(data),

		// Edit Form
		editForm(data),
	)
}

func editHeader(data UserEditPageData) g.Node {
	return Div(Class("mb-6"),
		Div(Class("flex items-center justify-between mb-2"),
			Div(
				H1(Class("text-2xl font-semibold text-gray-900 dark:text-white"),
					g.Text("Edit User"),
				),
				P(Class("text-sm text-gray-600 dark:text-gray-400 mt-1"),
					g.Text("Update user information and permissions"),
				),
			),
			A(
				Href(data.BasePath+"/users/"+data.User.UserID),
				Class("px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"),
				g.Text("Cancel"),
			),
		),
	)
}

func editForm(data UserEditPageData) g.Node {
	return FormEl(
		Method("POST"),
		Action(data.BasePath+"/users/"+data.User.UserID+"/edit"),
		Class("space-y-6"),

		Input(Type("hidden"), Name("csrf_token"), Value(data.CSRFToken)),

		// Basic Information Card
		basicInfoCard(data.User),

		// Email Verification Card
		emailVerificationCard(data.User.EmailVerified),

		// Action Buttons
		formActions(data),
	)
}

func basicInfoCard(user UserEditData) g.Node {
	return Div(Class("bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden"),
		Div(Class("p-6 border-b border-gray-200 dark:border-gray-700"),
			H2(Class("text-lg font-semibold text-gray-900 dark:text-white"),
				g.Text("Basic Information"),
			),
		),
		Div(Class("p-6 space-y-4"),
			// User ID (Read-only)
			Div(
				Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
					g.Text("User ID"),
				),
				Input(
					Type("text"),
					Value(user.UserID),
					Disabled(),
					Class("w-full px-3 py-2 font-mono text-sm bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg text-gray-500 dark:text-gray-500 cursor-not-allowed"),
				),
			),

			// Name
			Div(
				Label(
					For("name"),
					Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
					g.Text("Name "),
					Span(Class("text-red-500"), g.Text("*")),
				),
				Input(
					Type("text"),
					ID("name"),
					Name("name"),
					Value(user.Name),
					Required(),
					Class("w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"),
				),
			),

			// Email
			Div(
				Label(
					For("email"),
					Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
					g.Text("Email "),
					Span(Class("text-red-500"), g.Text("*")),
				),
				Input(
					Type("email"),
					ID("email"),
					Name("email"),
					Value(user.Email),
					Required(),
					Class("w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"),
				),
			),

			// Username
			Div(
				Label(
					For("username"),
					Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
					g.Text("Username"),
				),
				Input(
					Type("text"),
					ID("username"),
					Name("username"),
					Value(user.Username),
					Class("w-full px-3 py-2 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"),
				),
			),
		),
	)
}

func emailVerificationCard(isVerified bool) g.Node {
	return Div(Class("bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden"),
		Div(Class("p-6 border-b border-gray-200 dark:border-gray-700"),
			H2(Class("text-lg font-semibold text-gray-900 dark:text-white"),
				g.Text("Email Verification"),
			),
		),
		Div(Class("p-6"),
			Div(Class("flex items-center"),
				Input(
					Type("checkbox"),
					ID("email_verified"),
					Name("email_verified"),
					Value("true"),
					g.If(isVerified, Checked()),
					Class("w-4 h-4 text-blue-600 bg-white dark:bg-gray-900 border-gray-300 dark:border-gray-600 rounded focus:ring-blue-500 focus:ring-2"),
				),
				Label(
					For("email_verified"),
					Class("ml-2 text-sm font-medium text-gray-700 dark:text-gray-300"),
					g.Text("Email is verified"),
				),
			),
			P(Class("mt-2 text-xs text-gray-500 dark:text-gray-400"),
				g.Text("Check this box to manually verify the user's email address"),
			),
		),
	)
}

func formActions(data UserEditPageData) g.Node {
	return Div(Class("flex items-center justify-between pt-4"),
		Div(Class("flex gap-3"),
			Button(
				Type("submit"),
				Class("inline-flex items-center gap-2 px-6 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 transition-colors"),
				lucide.Check(Class("h-4 w-4")),
				g.Text("Save Changes"),
			),
			A(
				Href(data.BasePath+"/users/"+data.User.UserID),
				Class("px-6 py-2.5 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 text-sm font-medium border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"),
				g.Text("Cancel"),
			),
		),
	)
}
