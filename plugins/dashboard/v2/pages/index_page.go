package pages

import (
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// IndexPage - Simple handler returning content only
func (p *PagesManager) IndexPage(ctx *router.PageContext) (g.Node, error) {

	return primitives.Container(
		primitives.Container(
			primitives.Box(
				primitives.WithChildren(
					Div(
						Class("min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8"),
						Div(
							Class("max-w-md w-full space-y-8"),

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
						),
					),
				),
			),

			// // Actions
			// primitives.HStack("4",
			// 	button.Primary(
			// 		g.Group([]g.Node{
			// 			icons.ChevronRight(icons.WithSize(16)),
			// 			g.Text("Get Started"),
			// 		}),
			// 		button.WithSize(forgeui.SizeLG),
			// 	),
			// 	button.Secondary(
			// 		g.Group([]g.Node{
			// 			icons.Book(icons.WithSize(16)),
			// 			g.Text("Documentation"),
			// 		}),
			// 		button.WithSize(forgeui.SizeLG),
			// 	),
			// ),
		),
	), nil
}
