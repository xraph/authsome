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

// ResetPasswordPage renders the reset password page.
func (p *PagesManager) ResetPasswordPage(ctx *router.PageContext) (g.Node, error) {
	token := ctx.Query("token")
	errorParam := ctx.Query("error")

	var errorMessage string

	switch errorParam {
	case "invalid_token":
		errorMessage = "Invalid or expired reset token"
	case "password_mismatch":
		errorMessage = "Passwords do not match"
	case "weak_password":
		errorMessage = "Password must be at least 8 characters long"
	case "failed":
		errorMessage = "Failed to reset password. Please try again."
	}

	if token == "" {
		errorMessage = "Reset token is required"
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
									g.Text("Reset Password"),
								),
								P(
									Class("mt-2 text-center text-sm text-gray-600 dark:text-gray-400"),
									g.Text("Enter your new password below"),
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
							p.resetPasswordForm(token),

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

func (p *PagesManager) resetPasswordForm(token string) g.Node {
	return FormEl(
		Class("mt-8 space-y-6"),
		g.Attr("x-data", "{ loading: false, password: '', confirmPassword: '' }"),
		g.Attr("@submit.prevent", `
			if (password !== confirmPassword) {
				alert('Passwords do not match');
				return;
			}
			if (password.length < 8) {
				alert('Password must be at least 8 characters long');
				return;
			}
			loading = true;
			const result = await $bridge.call('resetPassword', { 
				token: '`+token+`',
				password 
			});
			if (result.success) {
				window.location.href = '`+p.baseUIPath+`/auth/login?success=password_reset';
			} else {
				loading = false;
				alert(result.message || 'Failed to reset password');
			}
		`),

		input.Input(
			input.WithType("hidden"),
			input.WithName("token"),
			input.WithValue(token),
		),

		Div(
			Label(
				For("password"),
				Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
				g.Text("New Password"),
			),
			input.Input(
				input.WithType("password"),
				input.WithName("password"),
				input.WithAttrs(
					g.Attr("x-model", "password"),
					g.Attr("autocomplete", "new-password"),
					g.Attr("required", ""),
					g.Attr("minlength", "8"),
				),
				input.WithClass("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
				input.WithPlaceholder("Enter new password"),
			),
		),

		Div(
			Label(
				For("confirmPassword"),
				Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
				g.Text("Confirm Password"),
			),
			input.Input(
				input.WithType("password"),
				input.WithName("confirmPassword"),
				input.WithAttrs(
					g.Attr("x-model", "confirmPassword"),
					g.Attr("autocomplete", "new-password"),
					g.Attr("required", ""),
					g.Attr("minlength", "8"),
				),
				input.WithClass("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"),
				input.WithPlaceholder("Confirm new password"),
			),
		),

		Div(
			button.Button(
				Div(
					Span(
						Class("absolute left-0 inset-y-0 flex items-center pl-3"),
						icons.Lock(icons.WithSize(20), icons.WithClass("text-indigo-500 group-hover:text-indigo-400 dark:text-indigo-300")),
					),
					g.El("span", g.Attr("x-show", "!loading"), g.Text("Reset Password")),
					g.El("span", g.Attr("x-show", "loading"), g.Text("Resetting...")),
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
