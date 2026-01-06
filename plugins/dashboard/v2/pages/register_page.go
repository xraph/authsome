package pages

import (
	"github.com/delaneyj/gomponents-iconify/iconify/tabler"
	"github.com/xraph/authsome/plugins/dashboard/components/pages"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ModernHomePage - Simple handler returning content only
func (p *PagesManager) RegisterPage(ctx *router.PageContext) (g.Node, error) {

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
