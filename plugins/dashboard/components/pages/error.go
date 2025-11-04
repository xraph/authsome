package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ErrorPage renders an error page content (used within base layout)
func ErrorPage(errorMessage string, basePath string) g.Node {
	return Div(
		Class("min-h-[400px] flex items-center justify-center"),
		Div(
			Class("text-center"),
			lucide.CircleAlert(Class("mx-auto h-16 w-16 text-red-400")),
			H1(
				Class("mt-4 text-3xl font-bold tracking-tight text-gray-900 dark:text-white"),
				g.Text("Error"),
			),
			P(
				Class("mt-2 text-base text-gray-500 dark:text-gray-400"),
				g.Text(errorMessage),
			),
			Div(
				Class("mt-6"),
				A(
					Href(basePath+"/dashboard/"),
					Class("inline-flex items-center rounded-md bg-primary-600 px-3.5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-primary-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600"),
					lucide.House(Class("mr-2 h-5 w-5")),
					g.Text("Go back to dashboard"),
				),
			),
		),
	)
}
