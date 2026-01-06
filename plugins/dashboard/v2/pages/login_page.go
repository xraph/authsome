package pages

import (
	"github.com/delaneyj/gomponents-iconify/iconify/tabler"
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/plugins/dashboard/components/pages"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ModernHomePage - Simple handler returning content only
func (p *PagesManager) LoginPage(ctx *router.PageContext) (g.Node, error) {

	// Check if already authenticated (check session cookie directly since no auth middleware)
	if user := p.checkExistingPageSession(ctx); user != nil {
		// Already logged in, redirect to dashboard
		redirect := ctx.Query("redirect")
		if redirect == "" {
			redirect = p.baseUIPath
		}
		// return ctx.Redirect(http.StatusFound, redirect)
	}

	redirect := ctx.Query("redirect")
	errorParam := ctx.Query("error")

	// Check if this is the first user (show signup prominently)
	isFirstUser, _ := p.isFirstUser(ctx.Request.Context())

	// Map error codes to user-friendly messages
	var errorMessage string
	switch errorParam {
	case "admin_required":
		errorMessage = "Admin access required to view dashboard"
	case "invalid_session":
		errorMessage = "Your session is invalid. Please log in again"
	case "insufficient_permissions":
		errorMessage = "You don't have permission to access the dashboard"
	case "session_expired":
		errorMessage = "Your session has expired. Please log in again"
	}

	loginData := pages.LoginPageData{
		Title:     "Login",
		CSRFToken: p.generateCSRFToken(),
		BasePath:  p.baseUIPath,
		Error:     errorMessage,
		Data: pages.LoginData{
			Redirect:    redirect,
			ShowSignup:  true,
			IsFirstUser: isFirstUser,
		},
	}

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

							// Error Message
							g.If(loginData.Error != "",
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
												g.Text(loginData.Error),
											),
										),
									),
								),
							),

							// Login Form
							p.loginForm(loginData),

							// Signup Link
							g.If(loginData.Data.ShowSignup,
								Div(
									Class("text-center"),
									P(
										Class("text-sm text-gray-600 dark:text-gray-400"),
										g.If(loginData.Data.IsFirstUser,
											Span(
												Class("block mb-2 text-indigo-600 dark:text-indigo-400 font-medium"),
												g.Text("No users found in the system"),
											),
										),
										g.Text("Don't have an account? "),
										A(
											Href(loginData.BasePath+"/dashboard/signup"),
											Class("font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-500 dark:hover:text-indigo-300"),
											g.If(loginData.Data.IsFirstUser,
												g.Text("Create admin account"),
											),
											g.If(!loginData.Data.IsFirstUser,
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

func (p *PagesManager) loginForm(data pages.LoginPageData) g.Node {
	return FormEl(
		Class("mt-8 space-y-6"),
		Action(data.BasePath+"/dashboard/login"),
		Method("POST"),
		g.Attr("x-data", "{ loading: false }"),
		g.Attr("@submit", "loading = true"),

		// CSRF Token
		input.Input(input.WithType("hidden"), input.WithName("csrf_token"), input.WithValue(data.CSRFToken)),

		// Redirect field
		g.If(data.Data.Redirect != "",
			input.Input(input.WithType("hidden"), input.WithName("redirect"), input.WithValue(data.Data.Redirect)),
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
				input.Input(
					input.WithType("email"),
					input.WithName("email"),
					// input.WithAutoComplete("email"),
					input.WithAttrs(g.Attr("autocomplete", "email")),
					input.WithClass("appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
					input.WithPlaceholder("Email address"),
				),
				// 	ID("email"),
				// 	Name("email"),
				// 	Type("email"),
				// 	g.Attr("autocomplete", "email"),
				// 	Required(),
				// 	Class("appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
				// 	g.Attr("placeholder", "Email address"),
				// ),
			),
			Div(
				Label(
					For("password"),
					Class("sr-only"),
					g.Text("Password"),
				),
				input.Input(
					input.WithType("password"),
					input.WithName("password"),
					input.WithAttrs(g.Attr("autocomplete", "current-password")),
					input.WithClass("appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
					input.WithPlaceholder("Password"),
				),
				// 	ID("password"),
				// 	Name("password"),
				// 	Type("password"),
				// 	g.Attr("autocomplete", "current-password"),
				// 	Required(),
				// 	Class("appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
				// 	g.Attr("placeholder", "Password"),
				// ),
			),
		),

		// Submit Button
		Div(
			button.Button(
				Div(
					Span(
						Class("absolute left-0 inset-y-0 flex items-center pl-3"),
						lucide.Lock(Class("h-5 w-5 text-indigo-500 group-hover:text-indigo-400 dark:text-indigo-300")),
					),
					g.El("span", g.Attr("x-show", "!loading"), g.Text("Sign in")),
					g.El("span", g.Attr("x-show", "loading"), g.Text("Signing in...")),
				),
				button.WithType("submit"),
				button.WithAttrs(g.Attr(":disabled", "loading")),
				button.WithAttrs(g.Attr(":class", "loading ? 'opacity-50 cursor-not-allowed' : ''")),
				button.WithClass("group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors"),
			),
		),
	)
	// 			Type("submit"),
	// 			g.Attr(":disabled", "loading"),
	// 			g.Attr(":class", "loading ? 'opacity-50 cursor-not-allowed' : ''"),
	// 			Class("group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors"),
	// 			Span(
	// 				Class("absolute left-0 inset-y-0 flex items-center pl-3"),
	// 				lucide.Lock(Class("h-5 w-5 text-indigo-500 group-hover:text-indigo-400 dark:text-indigo-300")),
	// 			),
	// 			g.El("span", g.Attr("x-show", "!loading"), g.Text("Sign in")),
	// 			g.El("span", g.Attr("x-show", "loading"), g.Text("Signing in...")),
	// 		),
	// 	),
	// )
}
