package components

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// BaseLayout renders a minimal HTML structure (used for error pages and fallbacks)
func BaseLayout(data PageData, content g.Node) g.Node {
	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Class("h-full"),
			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text(data.Title+" - AuthSome Dashboard")),
				Link(Rel("stylesheet"), Href(data.BasePath+"/static/css/dashboard.css")),
			),
			Body(
				Class("min-h-screen bg-gray-50 dark:bg-gray-950 p-4"),
				Main(Class("container mx-auto"), content),
			),
		),
	)
}

// EmptyLayout renders a minimal layout without header/footer
func EmptyLayout(data PageData, content g.Node) g.Node {
	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Class("h-full"),
			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text(data.Title+" - AuthSome Dashboard")),
				Link(Rel("stylesheet"), Href(data.BasePath+"/static/css/dashboard.css")),
			),
			Body(
				Class("min-h-screen bg-gray-50 dark:bg-gray-950 p-4"),
				content,
			),
		),
	)
}

// BaseSidebarLayout renders a minimal sidebar layout (used when handler renders with layout)
func BaseSidebarLayout(data PageData, content g.Node) g.Node {
	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Class("h-full"),
			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text(data.Title+" - AuthSome Dashboard")),
				Link(Rel("stylesheet"), Href(data.BasePath+"/static/css/dashboard.css")),
			),
			Body(
				Class("min-h-screen bg-gray-50 dark:bg-gray-950"),
				Main(Class("p-4"), content),
			),
		),
	)
}
