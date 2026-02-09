package pages

import (
	"fmt"
	"net/http"
	"time"

	"github.com/delaneyj/gomponents-iconify/iconify/tabler"
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard/components/pages"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ModernHomePage - Simple handler returning content only.
func (p *PagesManager) RegisterPage(ctx *router.PageContext) (g.Node, error) {
	// // Check if already authenticated (check session cookie directly since no auth middleware)
	// if user := p.checkExistingPageSession(ctx); user != nil {
	// 	// Already logged in, redirect to dashboard
	// 	redirect := ctx.Query("redirect")
	// 	if redirect == "" {
	// 		redirect = p.baseUIPath
	// 	}
	// 	// return ctx.Redirect(http.StatusFound, redirect)
	// }
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

	signupData := pages.SignupPageData{
		Title:     "Sign Up",
		CSRFToken: p.generateCSRFToken(),
		BasePath:  p.baseUIPath,
		Error:     errorMessage,
		Data: pages.SignupData{
			Redirect:    redirect,
			IsFirstUser: isFirstUser,
		},
	}

	return p.signupPageContent(signupData), nil
}

// HandleSignup processes the signup form submission.
func (p *PagesManager) HandleSignup(ctx *router.PageContext) (g.Node, error) {
	// Parse form data
	if err := ctx.Request.ParseForm(); err != nil {
		return p.renderSignupError(ctx, "Invalid form data", ctx.Query("redirect"))
	}

	name := ctx.Request.FormValue("name")
	email := ctx.Request.FormValue("email")
	password := ctx.Request.FormValue("password")
	confirmPassword := ctx.Request.FormValue("password_confirm")
	redirect := ctx.Request.FormValue("redirect")
	csrfToken := ctx.Request.FormValue("csrf_token")

	// Validate CSRF token
	if csrfToken == "" {
		return p.renderSignupError(ctx, "Invalid CSRF token", redirect)
	}

	// Validate inputs
	if email == "" || password == "" {
		return p.renderSignupError(ctx, "Email and password are required", redirect)
	}

	if password != confirmPassword {
		return p.renderSignupError(ctx, "Passwords do not match", redirect)
	}

	if len(password) < 8 {
		return p.renderSignupError(ctx, "Password must be at least 8 characters", redirect)
	}

	// Get platform app context
	goCtx := ctx.Request.Context()

	platformApp, err := p.services.AppService().GetPlatformApp(goCtx)
	if err != nil {
		return p.renderSignupError(ctx, "System configuration error. Please contact administrator.", redirect)
	}

	goCtx = contexts.SetAppID(goCtx, platformApp.ID)

	// Get default environment for platform app (if multiapp enabled)
	if p.services.EnvironmentService() != nil {
		defaultEnv, err := p.services.EnvironmentService().GetDefaultEnvironment(goCtx, platformApp.ID)
		if err == nil && defaultEnv != nil {
			goCtx = contexts.SetEnvironmentID(goCtx, defaultEnv.ID)
		}
	}

	// Create user in platform app context
	newUser, err := p.services.UserService().Create(goCtx, &user.CreateUserRequest{
		Email:    email,
		Password: password,
		Name:     name,
		AppID:    platformApp.ID,
	})
	if err != nil {
		return p.renderSignupError(ctx, fmt.Sprintf("Failed to create account: %v", err), redirect)
	}

	// Add user as member of platform app
	// Will be auto-promoted to owner if first user
	_, err = p.services.AppService().CreateMember(goCtx, &app.Member{
		ID:       xid.New(),
		AppID:    platformApp.ID,
		UserID:   newUser.ID,
		Role:     app.MemberRoleMember,
		Status:   app.MemberStatusActive,
		JoinedAt: time.Now(),
	})
	if err != nil {
		// Continue anyway - user is created
	}

	// Create session for the new user
	sess, err := p.services.SessionService().Create(goCtx, &session.CreateSessionRequest{
		UserID:    newUser.ID,
		IPAddress: ctx.Request.RemoteAddr,
		UserAgent: ctx.Request.UserAgent(),
		Remember:  false,
		AppID:     platformApp.ID,
	})
	if err != nil {
		return p.renderSignupError(ctx, "Account created but failed to log you in. Please try logging in.", redirect)
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   ctx.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sess.ExpiresAt.Sub(sess.CreatedAt).Seconds()),
	}
	http.SetCookie(ctx.ResponseWriter, cookie)

	// Redirect to dashboard or specified redirect URL
	if redirect == "" {
		redirect = p.baseUIPath
	}

	ctx.SetHeader("Location", redirect)
	ctx.ResponseWriter.WriteHeader(http.StatusFound)

	return nil, nil
}

// renderSignupError renders the signup page with an error message.
func (p *PagesManager) renderSignupError(ctx *router.PageContext, message string, redirect string) (g.Node, error) {
	isFirstUser, _ := p.isFirstUser(ctx.Request.Context())

	signupData := pages.SignupPageData{
		Title:     "Sign Up",
		CSRFToken: p.generateCSRFToken(),
		BasePath:  p.baseUIPath,
		Error:     message,
		Data: pages.SignupData{
			Redirect:    redirect,
			IsFirstUser: isFirstUser,
		},
	}

	// Re-render signup page with error
	return p.signupPageContent(signupData), nil
}

// signupPageContent returns the signup page content (extracted for reuse).
func (p *PagesManager) signupPageContent(signupData pages.SignupPageData) g.Node {
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
									g.Text("Create Your Account"),
								),
								P(
									Class("mt-2 text-center text-sm text-gray-600 dark:text-gray-400"),
									g.Text("Get started with AuthSome Dashboard"),
								),
							),

							// Error Message
							g.If(signupData.Error != "",
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
												g.Text(signupData.Error),
											),
										),
									),
								),
							),

							// Signup Form
							p.signupForm(signupData),

							// Login Link
							Div(
								Class("text-center"),
								P(
									Class("text-sm text-gray-600 dark:text-gray-400"),
									g.Text("Already have an account? "),
									A(
										Href(signupData.BasePath+"/auth/login"),
										Class("font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-500 dark:hover:text-indigo-300"),
										g.Text("Sign in"),
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
		),
	)
}

func (p *PagesManager) signupForm(data pages.SignupPageData) g.Node {
	return FormEl(
		Class("mt-8 space-y-6"),
		Action(data.BasePath+"/auth/signup"),
		Method("POST"),
		g.Attr("x-data", "{ loading: false, password: '', confirmPassword: '', passwordMatch: true }"),
		g.Attr("@submit", `
			loading = true;
			if (password !== confirmPassword) {
				passwordMatch = false;
				loading = false;
				$event.preventDefault();
			}
		`),

		// CSRF Token
		input.Input(input.WithType("hidden"), input.WithName("csrf_token"), input.WithValue(data.CSRFToken)),

		// Redirect field
		g.If(data.Data.Redirect != "",
			input.Input(input.WithType("hidden"), input.WithName("redirect"), input.WithValue(data.Data.Redirect)),
		),

		// Email, Name, Password, and Confirm Password inputs
		Div(
			Class("space-y-4"),

			// Email field
			Div(
				Label(
					For("email"),
					Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
					g.Text("Email address"),
				),
				input.Input(
					input.WithType("email"),
					input.WithName("email"),
					input.WithAttrs(
						g.Attr("id", "email"),
						g.Attr("required", ""),
						g.Attr("autocomplete", "email"),
					),
					input.WithClass("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent sm:text-sm"),
					input.WithPlaceholder("Enter your email"),
				),
			),

			// Name field
			Div(
				Label(
					For("name"),
					Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
					g.Text("Full name"),
				),
				input.Input(
					input.WithType("text"),
					input.WithName("name"),
					input.WithAttrs(
						g.Attr("id", "name"),
						g.Attr("autocomplete", "name"),
					),
					input.WithClass("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent sm:text-sm"),
					input.WithPlaceholder("Enter your name (optional)"),
				),
			),

			// Password field
			Div(
				Label(
					For("password"),
					Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
					g.Text("Password"),
				),
				input.Input(
					input.WithType("password"),
					input.WithName("password"),
					input.WithAttrs(
						g.Attr("id", "password"),
						g.Attr("required", ""),
						g.Attr("autocomplete", "new-password"),
						g.Attr("x-model", "password"),
						g.Attr("@input", "passwordMatch = true"),
					),
					input.WithClass("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent sm:text-sm"),
					input.WithPlaceholder("Create a password"),
				),
			),

			// Confirm Password field
			Div(
				Label(
					For("password_confirm"),
					Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
					g.Text("Confirm password"),
				),
				input.Input(
					input.WithType("password"),
					input.WithName("password_confirm"),
					input.WithAttrs(
						g.Attr("id", "password_confirm"),
						g.Attr("required", ""),
						g.Attr("autocomplete", "new-password"),
						g.Attr("x-model", "confirmPassword"),
						g.Attr("@input", "passwordMatch = true"),
					),
					input.WithClass("appearance-none relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-800 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent sm:text-sm"),
					input.WithPlaceholder("Confirm your password"),
				),
				// Password mismatch error
				Div(
					Class("mt-1"),
					g.Attr("x-show", "!passwordMatch"),
					P(
						Class("text-sm text-red-600 dark:text-red-400"),
						g.Text("Passwords do not match"),
					),
				),
			),
		),

		// Submit Button
		Div(
			button.Button(
				Div(
					Span(
						Class("absolute left-0 inset-y-0 flex items-center pl-3"),
						lucide.UserPlus(Class("h-5 w-5 text-indigo-500 group-hover:text-indigo-400 dark:text-indigo-300")),
					),
					g.El("span", g.Attr("x-show", "!loading"), g.Text("Create account")),
					g.El("span", g.Attr("x-show", "loading"), g.Text("Creating account...")),
				),
				button.WithType("submit"),
				button.WithAttrs(g.Attr(":disabled", "loading")),
				button.WithAttrs(g.Attr(":class", "loading ? 'opacity-50 cursor-not-allowed' : ''")),
				button.WithClass("group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors"),
			),
		),
	)
}
