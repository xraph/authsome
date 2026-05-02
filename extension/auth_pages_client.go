package extension

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forgeui/router"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/dashboard/auth"
	authclient "github.com/xraph/authsome/sdk/go"
)

// clientAuthPages implements dashauth.AuthPageProvider when authsome runs as
// a client of a remote authsome service. All form submissions are proxied to
// the remote via the SDK; no local engine is required.
type clientAuthPages struct {
	client   *authclient.Client
	basePath string
}

var _ dashauth.AuthPageProvider = (*clientAuthPages)(nil)

func (a *clientAuthPages) AuthPages() []dashauth.AuthPageDescriptor {
	return []dashauth.AuthPageDescriptor{
		{Type: dashauth.PageLogin, Path: "/login", Title: "Sign In", Icon: "shield-check"},
		{Type: dashauth.PageRegister, Path: "/register", Title: "Sign Up", Icon: "user-plus"},
		{Type: dashauth.PageForgotPassword, Path: "/forgot-password", Title: "Forgot Password", Icon: "key-round"},
		{Type: dashauth.PageLogout, Path: "/logout", Title: "Sign Out", Icon: "log-out"},
	}
}

func (a *clientAuthPages) RenderAuthPage(ctx *router.PageContext, pageType dashauth.AuthPageType) (templ.Component, error) {
	switch pageType {
	case dashauth.PageLogin:
		links := loginLinks(a.basePath)
		links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)
		return auth.LoginPage(links), nil
	case dashauth.PageRegister:
		// Dynamic form configs aren't exposed via the SDK yet; static page only.
		links := registerLinks(a.basePath)
		links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)
		return auth.RegisterPage(links), nil
	case dashauth.PageForgotPassword:
		links := forgotPasswordLinks(a.basePath)
		links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)
		return auth.ForgotPasswordPage(links), nil
	default:
		return nil, nil
	}
}

func (a *clientAuthPages) HandleAuthAction(ctx *router.PageContext, pageType dashauth.AuthPageType) (string, templ.Component, error) {
	switch pageType {
	case dashauth.PageLogin:
		return a.handleLogin(ctx)
	case dashauth.PageRegister:
		return a.handleRegister(ctx)
	case dashauth.PageForgotPassword:
		return a.handleForgotPassword(ctx)
	case dashauth.PageLogout:
		return a.handleLogout(ctx)
	default:
		return "", nil, nil
	}
}

func (a *clientAuthPages) handleLogin(ctx *router.PageContext) (string, templ.Component, error) {
	r := ctx.Request
	links := loginLinks(a.basePath)

	if err := r.ParseForm(); err != nil {
		links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)
		return "", auth.LoginError("Invalid form data", links), nil
	}

	if !dashboard.VerifyFormCSRFToken(r, r.FormValue(dashboard.FormCSRFFormField)) {
		return a.basePath + "/login", nil, nil
	}

	links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		return "", auth.LoginError("Email and password are required", links), nil
	}

	resp, err := a.client.SignIn(r.Context(), &authclient.SignInRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "", auth.LoginError("Invalid email or password", links), nil
	}

	setSessionCookie(ctx, resp.SessionToken, isSecureRequest(r))

	return a.basePath + "/", nil, nil
}

func (a *clientAuthPages) handleRegister(ctx *router.PageContext) (string, templ.Component, error) {
	r := ctx.Request
	links := registerLinks(a.basePath)

	if err := r.ParseForm(); err != nil {
		links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)
		return "", auth.RegisterError("Invalid form data", links), nil
	}

	if !dashboard.VerifyFormCSRFToken(r, r.FormValue(dashboard.FormCSRFFormField)) {
		return a.basePath + "/register", nil, nil
	}

	links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)

	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Collect meta_* fields into a metadata map, matching the engine path.
	metadata := make(map[string]any)
	for key, vals := range r.Form {
		if strings.HasPrefix(key, "meta_") && len(vals) > 0 {
			metadata[strings.TrimPrefix(key, "meta_")] = vals[0]
		}
	}

	if email == "" || password == "" {
		return "", auth.RegisterError("Email and password are required", links), nil
	}

	req := &authclient.SignUpRequest{
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
	}
	if len(metadata) > 0 {
		req.Metadata = metadata
	}

	resp, err := a.client.SignUp(r.Context(), req)
	if err != nil {
		return "", auth.RegisterError(err.Error(), links), nil
	}

	setSessionCookie(ctx, resp.SessionToken, isSecureRequest(r))

	return a.basePath + "/", nil, nil
}

func (a *clientAuthPages) handleForgotPassword(ctx *router.PageContext) (string, templ.Component, error) {
	r := ctx.Request
	links := forgotPasswordLinks(a.basePath)

	if err := r.ParseForm(); err != nil {
		links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)
		return "", auth.ForgotPasswordError("Invalid form data", links), nil
	}

	if !dashboard.VerifyFormCSRFToken(r, r.FormValue(dashboard.FormCSRFFormField)) {
		return a.basePath + "/forgot-password", nil, nil
	}

	links.CSRFToken = dashboard.GenerateFormCSRFToken(ctx.ResponseWriter)

	email := r.FormValue("email")
	if email == "" {
		return "", auth.ForgotPasswordError("Email is required", links), nil
	}

	// Always show success to prevent email enumeration; ignore errors.
	_, _ = a.client.ForgotPassword(r.Context(), &authclient.ForgotPasswordRequest{ //nolint:errcheck // best-effort send
		Email: email,
	})

	return "", auth.ForgotPasswordSuccess(links), nil
}

func (a *clientAuthPages) handleLogout(ctx *router.PageContext) (string, templ.Component, error) {
	r := ctx.Request

	// Best-effort SignOut on the remote — the server identifies the session
	// from the cookie/header forwarded by the client SDK.
	if token := extractToken(r); token != "" {
		_, _ = a.client.SignOut(r.Context(), &authclient.SignOutRequest{}) //nolint:errcheck // best-effort sign out
	}

	clearSessionCookie(ctx, isSecureRequest(r))

	return a.basePath + "/login", nil, nil
}

// clientAuthChecker validates dashboard sessions by introspecting the auth
// token against the remote authsome service.
type clientAuthChecker struct {
	client *authclient.Client
}

var _ dashauth.AuthChecker = (*clientAuthChecker)(nil)

func (c *clientAuthChecker) CheckAuth(ctx context.Context, r *http.Request) (*dashauth.UserInfo, error) {
	token := extractToken(r)
	if token == "" {
		return nil, nil
	}

	resp, err := c.client.Introspect(ctx, token)
	if err != nil || resp == nil || !resp.Active || resp.User == nil {
		return nil, nil
	}

	display := strings.TrimSpace(resp.User.FirstName + " " + resp.User.LastName)
	if display == "" {
		display = resp.User.Email
	}

	return &dashauth.UserInfo{
		Subject:     resp.User.ID,
		DisplayName: display,
		Email:       resp.User.Email,
	}, nil
}
