package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SignupPageData contains data for signup page
type SignupPageData struct {
	Title     string
	CSRFToken string
	BasePath  string
	Error     string
	Data      SignupData
}

// SignupData contains form data
type SignupData struct {
	Redirect    string
	IsFirstUser bool
}

// Signup renders a standalone signup page (no base layout)
func Signup(data SignupPageData) g.Node {
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

				// Theme management
				Script(g.Raw(`
					function themeData() {
						return {
							isDark: localStorage.getItem('theme') === 'dark' || 
								(!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches),
							initTheme() {
								if (this.isDark) {
									document.documentElement.classList.add('dark');
								} else {
									document.documentElement.classList.remove('dark');
								}
							},
							toggleTheme() {
								this.isDark = !this.isDark;
								if (this.isDark) {
									document.documentElement.classList.add('dark');
									localStorage.setItem('theme', 'dark');
								} else {
									document.documentElement.classList.remove('dark');
									localStorage.setItem('theme', 'light');
								}
							}
						}
					}
				`)),
			),

			Body(Class("antialiased"),
				Div(Class("min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 py-12 px-4 sm:px-6 lg:px-8"),
		Div(Class("max-w-md w-full space-y-8"),
			// Logo and Title
			Div(Class("text-center"),
				Div(Class("flex justify-center mb-6"),
					lucide.ShieldCheck(Class("h-16 w-16 text-violet-600")),
				),
				H2(Class("text-3xl font-bold text-gray-900 dark:text-white"),
					g.Text("Create your account"),
				),
				P(Class("mt-2 text-sm text-gray-600 dark:text-gray-400"),
					g.Text("Get started with your dashboard"),
				),
			),

			// Theme toggle (using Alpine.js)
			Div(Class("flex justify-center"),
				Button(
					Type("button"),
					g.Attr("@click", "toggleTheme()"),
					Class("p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"),
					Span(g.Attr("x-show", "!isDark"),
						lucide.Moon(Class("h-5 w-5 text-gray-600")),
					),
					Span(g.Attr("x-show", "isDark"),
						lucide.Sun(Class("h-5 w-5 text-gray-400")),
					),
				),
			),

			// Error message
			g.If(data.Error != "",
				Div(Class("rounded-md bg-red-50 dark:bg-red-900/20 p-4 border border-red-200 dark:border-red-800"),
					Div(Class("flex"),
						Div(Class("flex-shrink-0"),
							lucide.X(Class("h-5 w-5 text-red-400")),
						),
						Div(Class("ml-3"),
							P(Class("text-sm font-medium text-red-800 dark:text-red-200"),
								g.Text(data.Error),
							),
						),
					),
				),
			),

			// Signup form
			FormEl(Class("mt-8 space-y-6"),
				Action(data.BasePath+"/dashboard/signup"),
				Method("POST"),
				Input(Type("hidden"), Name("csrf_token"), Value(data.CSRFToken)),
				Input(Type("hidden"), Name("redirect"), Value(data.Data.Redirect)),

				Div(Class("space-y-4"),
					// Email field
					Div(
						Label(For("email"),
							Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
							g.Text("Email address"),
						),
						Input(
							Type("email"),
							ID("email"),
							Name("email"),
							Required(),
							AutoComplete("email"),
							Class("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-lg placeholder-gray-400 dark:placeholder-gray-500 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent sm:text-sm"),
							Placeholder("Enter your email"),
						),
					),

					// Name field
					Div(
						Label(For("name"),
							Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
							g.Text("Full name"),
						),
						Input(
							Type("text"),
							ID("name"),
							Name("name"),
							Required(),
							AutoComplete("name"),
							Class("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-lg placeholder-gray-400 dark:placeholder-gray-500 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent sm:text-sm"),
							Placeholder("Enter your name"),
						),
					),

					// Password field
					Div(
						Label(For("password"),
							Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
							g.Text("Password"),
						),
						Input(
							Type("password"),
							ID("password"),
							Name("password"),
							Required(),
							AutoComplete("new-password"),
							Class("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-lg placeholder-gray-400 dark:placeholder-gray-500 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent sm:text-sm"),
							Placeholder("Create a password"),
						),
					),

					// Confirm password field
					Div(
						Label(For("password_confirm"),
							Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
							g.Text("Confirm password"),
						),
						Input(
							Type("password"),
							ID("password_confirm"),
							Name("password_confirm"),
							Required(),
							AutoComplete("new-password"),
							Class("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-700 rounded-lg placeholder-gray-400 dark:placeholder-gray-500 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent sm:text-sm"),
							Placeholder("Confirm your password"),
						),
					),
				),

				// Submit button
				Div(
					Button(
						Type("submit"),
						Class("group relative w-full flex justify-center py-2.5 px-4 border border-transparent text-sm font-medium rounded-lg text-white bg-violet-600 hover:bg-violet-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-violet-500 transition-colors"),
						g.Text("Create account"),
					),
				),

				// Login link
				Div(Class("text-center"),
					P(Class("text-sm text-gray-600 dark:text-gray-400"),
						g.Text("Already have an account? "),
						A(Href(data.BasePath+"/dashboard/login"),
							Class("font-medium text-violet-600 hover:text-violet-500 dark:text-violet-400 dark:hover:text-violet-300"),
							g.Text("Sign in"),
						),
					),
				),
			),
		),
				),
			),
		),
	)
}

