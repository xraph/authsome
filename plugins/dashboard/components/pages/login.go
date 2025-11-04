package pages

import (
	"github.com/delaneyj/gomponents-iconify/iconify/tabler"
	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// LoginData contains data specific to the login page
type LoginData struct {
	ShowSignup  bool
	IsFirstUser bool
	Redirect    string
}

// LoginPageData represents all data needed for login page
type LoginPageData struct {
	Title     string
	Error     string
	CSRFToken string
	BasePath  string
	Data      LoginData
}

// Login renders a standalone login page (no base layout)
func Login(data LoginPageData) g.Node {
	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			g.Attr("x-data", "themeData()"),
			g.Attr("x-init", "initTheme()"),
			g.Attr(":class", "{ 'dark': isDark }"),

			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text(data.Title+" - AuthSome Dashboard")),

				// Tailwind CSS CDN
				Script(Src("https://cdn.tailwindcss.com?plugins=forms,typography,aspect-ratio")),
				Script(g.Raw(`tailwind.config = { darkMode: 'class' }`)),

				// Alpine.js
				Script(Defer(), Src("https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js")),

				// Custom CSS
				Link(Rel("stylesheet"), Href(data.BasePath+"/dashboard/static/css/custom.css")),

				// Alpine.js x-cloak style
				StyleEl(g.Raw(`[x-cloak] { display: none !important; }`)),

				// Dashboard JS
				Script(Src(data.BasePath+"/dashboard/static/js/dashboard.js")),
			),

			Body(
				Class("bg-gray-100 dark:bg-gray-900 transition-colors duration-200"),
				Div(
					Class("min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8"),
					Div(
						Class("max-w-md w-full space-y-8"),

						// Theme Toggle
						Div(
							Class("flex justify-end"),
							themeToggleButton(),
						),

						// Header
						Div(
							H2(
								Class("mt-6 text-center text-3xl font-extrabold text-gray-900 dark:text-white"),
								g.Text("AuthSome Dashboard"),
							),
							P(
								Class("mt-2 text-center text-sm text-gray-600 dark:text-gray-400"),
								g.Text("Sign in to access the admin dashboard"),
							),
						),

						// Error Message
						g.If(data.Error != "",
							Div(
								Class("rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4"),
								Div(
									Class("flex"),
									Div(
										Class("flex-shrink-0"),
										tabler.X(Class("h-5 w-5 text-red-400 dark:text-red-500")),
									),
									Div(
										Class("ml-3"),
										H3(
											Class("text-sm font-medium text-red-800 dark:text-red-300"),
											g.Text(data.Error),
										),
									),
								),
							),
						),

						// Login Form
						loginForm(data),

						// Signup Link
						g.If(data.Data.ShowSignup,
							Div(
								Class("text-center"),
								P(
									Class("text-sm text-gray-600 dark:text-gray-400"),
									g.If(data.Data.IsFirstUser,
										Span(
											Class("block mb-2 text-indigo-600 dark:text-indigo-400 font-medium"),
											g.Text("No users found in the system"),
										),
									),
									g.Text("Don't have an account? "),
									A(
										Href(data.BasePath+"/dashboard/signup"),
										Class("font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-500 dark:hover:text-indigo-300"),
										g.If(data.Data.IsFirstUser,
											g.Text("Create admin account"),
										),
										g.If(!data.Data.IsFirstUser,
											g.Text("Sign up"),
										),
									),
								),
							),
						),

						// Footer
						Div(
							Class("text-center text-sm text-gray-600 dark:text-gray-400"),
							P(g.Text("Protected by AuthSome")),
						),
					),
				),
			),
		),
	)
}

func themeToggleButton() g.Node {
	return Button(
		g.Attr("@click", "toggleTheme()"),
		Class("rounded-full bg-white dark:bg-gray-800 p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white border border-gray-300 dark:border-gray-700 shadow-sm hover:shadow-md transition-all"),
		g.Attr(":title", "isDark ? 'Switch to light mode' : 'Switch to dark mode'"),
		Span(Class("sr-only"), g.Text("Toggle theme")),

		// Sun icon (shown in dark mode)
		g.El("div",
			g.Attr("x-show", "isDark"),
			g.Attr("x-cloak", ""),
			lucide.Sun(Class("h-5 w-5")),
		),

		// Moon icon (shown in light mode)
		g.El("div",
			g.Attr("x-show", "!isDark"),
			g.Attr("x-cloak", ""),
			lucide.Moon(Class("h-5 w-5")),
		),
	)
}

func loginForm(data LoginPageData) g.Node {
	return FormEl(
		Class("mt-8 space-y-6"),
		Action(data.BasePath+"/dashboard/login"),
		Method("POST"),
		g.Attr("x-data", "{ loading: false }"),
		g.Attr("@submit", "loading = true"),

		// CSRF Token
		Input(Type("hidden"), Name("csrf_token"), Value(data.CSRFToken)),

		// Redirect field
		g.If(data.Data.Redirect != "",
			Input(Type("hidden"), Name("redirect"), Value(data.Data.Redirect)),
		),

		// Email and Password inputs
		Div(
			Class("rounded-md shadow-sm -space-y-px"),
			Div(
				Label(
					For("email"),
					Class("sr-only"),
					g.Text("Email address"),
				),
				Input(
					ID("email"),
					Name("email"),
					Type("email"),
					g.Attr("autocomplete", "email"),
					Required(),
					Class("appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
					g.Attr("placeholder", "Email address"),
				),
			),
			Div(
				Label(
					For("password"),
					Class("sr-only"),
					g.Text("Password"),
				),
				Input(
					ID("password"),
					Name("password"),
					Type("password"),
					g.Attr("autocomplete", "current-password"),
					Required(),
					Class("appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
					g.Attr("placeholder", "Password"),
				),
			),
		),

		// Submit Button
		Div(
			Button(
				Type("submit"),
				g.Attr(":disabled", "loading"),
				g.Attr(":class", "loading ? 'opacity-50 cursor-not-allowed' : ''"),
				Class("group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors"),
				Span(
					Class("absolute left-0 inset-y-0 flex items-center pl-3"),
					lucide.Lock(Class("h-5 w-5 text-indigo-500 group-hover:text-indigo-400 dark:text-indigo-300")),
				),
				g.El("span", g.Attr("x-show", "!loading"), g.Text("Sign in")),
				g.El("span", g.Attr("x-show", "loading"), g.Text("Signing in...")),
			),
		),
	)
}
