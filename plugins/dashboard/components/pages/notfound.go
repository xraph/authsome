package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// NotFound renders a 404 page (standalone, no base layout)
func NotFound(basePath string) g.Node {
	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Class("h-full"),
			g.Attr("x-data", "themeData()"),
			g.Attr("x-init", "initTheme()"),
			g.Attr(":class", "{ 'dark': isDark }"),

			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text("404 - Page Not Found")),

				// Tailwind CSS CDN
				Script(Src("https://cdn.tailwindcss.com?plugins=forms,typography")),

				// Alpine.js x-cloak style
				StyleEl(g.Raw(`[x-cloak] { display: none !important; }`)),

				// Tailwind Configuration
				Script(g.Raw(`
                    tailwind.config = {
                        darkMode: 'class',
                        theme: {
                            extend: {
                                colors: {
                                    primary: {
                                        DEFAULT: 'rgb(124 58 237)',
                                        500: 'rgb(168 85 247)',
                                        600: 'rgb(147 51 234)',
                                    }
                                }
                            }
                        }
                    }
                `)),

				// Load component functions BEFORE Alpine.js
				Script(Src(basePath+"/dashboard/static/js/pines-components.js")),
				Script(Src(basePath+"/dashboard/static/js/dashboard.js")),

				// Alpine.js - Load LAST
				Script(Defer(), Src("https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js")),
			),

			Body(
				Class("h-full bg-slate-50 dark:bg-gray-950"),
				Div(
					Class("flex min-h-full flex-col items-center justify-center px-6 py-12 lg:px-8"),
					Div(
						Class("text-center max-w-2xl"),

						// 404 Illustration
						Div(
							Class("mb-8"),
							lucide.CircleHelp(Class("mx-auto h-64 w-64 text-violet-600 dark:text-violet-400")),
						),

						// 404 Text
						H1(
							Class("text-9xl font-black text-slate-900 dark:text-white mb-4"),
							g.Text("404"),
						),
						H2(
							Class("text-3xl font-bold text-slate-900 dark:text-white mb-4"),
							g.Text("Page Not Found"),
						),
						P(
							Class("text-lg text-slate-600 dark:text-gray-400 mb-8 max-w-md mx-auto"),
							g.Text("Sorry, we couldn't find the page you're looking for. The page might have been moved or deleted."),
						),

						// Actions
						Div(
							Class("flex flex-col sm:flex-row gap-4 justify-center items-center"),
							A(
								Href(basePath+"/dashboard/"),
								Class("inline-flex items-center justify-center gap-2 rounded-lg bg-violet-600 hover:bg-violet-700 px-6 py-3 text-sm font-semibold text-white shadow-sm transition-colors"),
								lucide.House(Class("h-5 w-5")),
								g.Text("Back to Dashboard"),
							),
							Button(
								g.Attr("onclick", "history.back()"),
								Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-6 py-3 text-sm font-semibold text-slate-900 dark:text-white hover:bg-slate-50 dark:hover:bg-gray-700 shadow-sm transition-colors"),
								lucide.ArrowLeft(Class("h-5 w-5")),
								g.Text("Go Back"),
							),
						),

						// Dark Mode Toggle
						Div(
							Class("mt-8"),
							Button(
								g.Attr("@click", "toggleTheme()"),
								Class("inline-flex items-center gap-2 text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400 transition-colors"),
								g.El("div",
									g.Attr("x-show", "!isDark"),
									lucide.Moon(Class("h-5 w-5")),
								),
								g.El("div",
									g.Attr("x-show", "isDark"),
									lucide.Sun(Class("h-5 w-5")),
								),
								g.El("span", g.Attr("x-text", "isDark ? 'Light Mode' : 'Dark Mode'")),
							),
						),

						// Help Links
						Div(
							Class("mt-12 pt-8 border-t border-slate-200 dark:border-gray-800"),
							P(
								Class("text-sm text-slate-600 dark:text-gray-400 mb-4"),
								g.Text("Need help?"),
							),
							Div(
								Class("flex flex-wrap gap-6 justify-center text-sm"),
								A(
									Href(basePath+"/dashboard/users"),
									Class("text-violet-600 dark:text-violet-400 hover:text-violet-700 dark:hover:text-violet-300 transition-colors"),
									g.Text("Users"),
								),
								A(
									Href(basePath+"/dashboard/sessions"),
									Class("text-violet-600 dark:text-violet-400 hover:text-violet-700 dark:hover:text-violet-300 transition-colors"),
									g.Text("Sessions"),
								),
								A(
									Href(basePath+"/dashboard/settings"),
									Class("text-violet-600 dark:text-violet-400 hover:text-violet-700 dark:hover:text-violet-300 transition-colors"),
									g.Text("Settings"),
								),
							),
						),
					),
				),
			),
		),
	)
}
