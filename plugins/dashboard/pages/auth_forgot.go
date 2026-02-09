package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ForgotPasswordPage renders the forgot password page.
func (p *PagesManager) ForgotPasswordPage(ctx *router.PageContext) (g.Node, error) {
	errorParam := ctx.Query("error")
	successParam := ctx.Query("success")

	var (
		errorMessage   string
		successMessage string
	)

	switch errorParam {
	case "invalid_email":
		errorMessage = "Please enter a valid email address"
	case "user_not_found":
		errorMessage = "No account found with that email address"
	case "failed":
		errorMessage = "Failed to send reset email. Please try again."
	}

	if successParam == "1" {
		successMessage = "Password reset instructions have been sent to your email"
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
									g.Text("Forgot Password"),
								),
								P(
									Class("mt-2 text-center text-sm text-gray-600 dark:text-gray-400"),
									g.Text("Enter your email address and we'll send you instructions to reset your password"),
								),
							),

							// Success Message
							g.If(successMessage != "",
								Div(
									Class("rounded-md bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 p-4"),
									Div(
										Class("flex"),
										Div(
											Class("flex-shrink-0"),
											icons.CheckCircle(icons.WithSize(20), icons.WithClass("text-green-400 dark:text-green-500")),
										),
										Div(
											Class("ml-3"),
											P(
												Class("text-sm font-medium text-green-800 dark:text-green-300"),
												g.Text(successMessage),
											),
										),
									),
								),
							),

							// Error Message
							g.If(errorMessage != "",
								Div(
									Class("rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4"),
									Div(
										Class("flex"),
										Div(
											Class("flex-shrink-0"),
											icons.XCircle(icons.WithSize(20), icons.WithClass("text-red-400 dark:text-red-500")),
										),
										Div(
											Class("ml-3"),
											P(
												Class("text-sm font-medium text-red-800 dark:text-red-300"),
												g.Text(errorMessage),
											),
										),
									),
								),
							),

							// Form
							p.forgotPasswordForm(),

							// Back to login link
							Div(
								Class("text-center"),
								A(
									Href(p.baseUIPath+"/auth/login"),
									Class("font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-500 dark:hover:text-indigo-300"),
									g.Text("← Back to login"),
								),
							),
						),
					),
				),
			),
		),
	), nil
}

func (p *PagesManager) forgotPasswordForm() g.Node {
	return FormEl(
		Class("mt-8 space-y-6"),
		g.Attr("x-data", "{ loading: false, email: '' }"),
		g.Attr("@submit.prevent", `
			loading = true;
			const result = await $bridge.call('requestPasswordReset', { email });
			if (result.success) {
				window.location.href = '`+p.baseUIPath+`/auth/forgot-password?success=1';
			} else {
				loading = false;
				alert(result.message || 'Failed to send reset email');
			}
		`),

		Div(
			Label(
				For("email"),
				Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
				g.Text("Email address"),
			),
			input.Input(
				input.WithType("email"),
				input.WithName("email"),
				input.WithAttrs(
					g.Attr("x-model", "email"),
					g.Attr("autocomplete", "email"),
					g.Attr("required", ""),
				),
				input.WithClass("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
				input.WithPlaceholder("you@example.com"),
			),
		),

		Div(
			button.Button(
				Div(
					Span(
						Class("absolute left-0 inset-y-0 flex items-center pl-3"),
						icons.Mail(icons.WithSize(20), icons.WithClass("text-indigo-500 group-hover:text-indigo-400 dark:text-indigo-300")),
					),
					g.El("span", g.Attr("x-show", "!loading"), g.Text("Send Reset Instructions")),
					g.El("span", g.Attr("x-show", "loading"), g.Text("Sending...")),
				),
				button.WithType("submit"),
				button.WithAttrs(
					g.Attr(":disabled", "loading"),
					g.Attr(":class", "loading ? 'opacity-50 cursor-not-allowed' : ''"),
				),
				button.WithClass("group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors"),
			),
		),
	)
}
