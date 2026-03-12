package extension

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forgeui/router"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/dashboard/auth"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

// authPages implements dashauth.AuthPageProvider for the authsome extension.
type authPages struct {
	engine   *authsome.Engine
	basePath string // dashboard base path (e.g. "/dashboard")
}

// Ensure authPages implements AuthPageProvider at compile time.
var _ dashauth.AuthPageProvider = (*authPages)(nil)

// AuthPages returns the list of auth pages this provider contributes.
func (a *authPages) AuthPages() []dashauth.AuthPageDescriptor {
	return []dashauth.AuthPageDescriptor{
		{
			Type:  dashauth.PageLogin,
			Path:  "/login",
			Title: "Sign In",
			Icon:  "shield-check",
		},
		{
			Type:  dashauth.PageRegister,
			Path:  "/register",
			Title: "Sign Up",
			Icon:  "user-plus",
		},
		{
			Type:  dashauth.PageForgotPassword,
			Path:  "/forgot-password",
			Title: "Forgot Password",
			Icon:  "key-round",
		},
		{
			Type:  dashauth.PageLogout,
			Path:  "/logout",
			Title: "Sign Out",
			Icon:  "log-out",
		},
	}
}

// authLinks computes sibling auth page paths using the dashboard base path.
// e.g., if basePath is "/dashboard", it returns /dashboard/login, /dashboard/register, /dashboard/forgot-password.
func authLinks(basePath string) (loginPath, registerPath, forgotPath string) {
	return basePath + "/login", basePath + "/register", basePath + "/forgot-password"
}

// loginLinks builds LoginPageLinks from the dashboard base path.
func loginLinks(basePath string) auth.LoginPageLinks {
	_, registerPath, forgotPath := authLinks(basePath)
	return auth.LoginPageLinks{
		RegisterPath:       registerPath,
		ForgotPasswordPath: forgotPath,
	}
}

// registerLinks builds RegisterPageLinks from the dashboard base path.
func registerLinks(basePath string) auth.RegisterPageLinks {
	loginPath, _, _ := authLinks(basePath)
	return auth.RegisterPageLinks{
		LoginPath: loginPath,
	}
}

// forgotPasswordLinks builds ForgotPasswordPageLinks from the dashboard base path.
func forgotPasswordLinks(basePath string) auth.ForgotPasswordPageLinks {
	loginPath, _, _ := authLinks(basePath)
	return auth.ForgotPasswordPageLinks{
		LoginPath: loginPath,
	}
}

// RenderAuthPage renders the templ component for an auth page.
func (a *authPages) RenderAuthPage(ctx *router.PageContext, pageType dashauth.AuthPageType) (templ.Component, error) {
	switch pageType {
	case dashauth.PageLogin:
		return auth.LoginPage(loginLinks(a.basePath)), nil
	case dashauth.PageRegister:
		return a.renderRegisterPage(ctx, "", nil, nil)
	case dashauth.PageForgotPassword:
		return auth.ForgotPasswordPage(forgotPasswordLinks(a.basePath)), nil
	default:
		return nil, nil
	}
}

// renderRegisterPage renders either the dynamic or static register page.
func (a *authPages) renderRegisterPage(_ *router.PageContext, errorMsg string, values, fieldErrs map[string]string) (templ.Component, error) {
	links := registerLinks(a.basePath)
	appID := a.defaultAppID()

	// Try to load a dynamic form config for this app.
	fc, err := a.engine.GetSignupFormConfig(context.Background(), appID)
	if err != nil || fc == nil || !fc.Active || len(fc.Fields) == 0 {
		// No active form config — fall back to static page.
		if errorMsg != "" {
			return auth.RegisterError(errorMsg, links), nil
		}
		return auth.RegisterPage(links), nil
	}

	// Optionally load org branding.
	var branding *formconfig.BrandingConfig
	// Branding is per-org; for the default register page we skip it unless
	// an org context is available in the future.

	if values == nil {
		values = make(map[string]string)
	}
	if fieldErrs == nil {
		fieldErrs = make(map[string]string)
	}

	return auth.DynamicRegisterPage(auth.DynamicRegisterProps{
		Links:     links,
		Fields:    fc.Fields,
		Branding:  branding,
		ErrorMsg:  errorMsg,
		Values:    values,
		FieldErrs: fieldErrs,
	}), nil
}

// HandleAuthAction handles form submissions for auth pages.
func (a *authPages) HandleAuthAction(ctx *router.PageContext, pageType dashauth.AuthPageType) (string, templ.Component, error) {
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

// defaultAppID resolves the default app ID from the engine config.
// When bootstrap is configured, the platform app ID takes precedence.
func (a *authPages) defaultAppID() id.AppID {
	if platformID := a.engine.PlatformAppID(); !platformID.IsNil() {
		return platformID
	}
	appID, _ := id.ParseAppID(a.engine.Config().AppID)
	return appID
}

func (a *authPages) handleLogin(ctx *router.PageContext) (string, templ.Component, error) {
	r := ctx.Request
	links := loginLinks(a.basePath)

	if err := r.ParseForm(); err != nil {
		return "", auth.LoginError("Invalid form data", links), nil
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		return "", auth.LoginError("Email and password are required", links), nil
	}

	_, sess, err := a.engine.SignIn(r.Context(), &account.SignInRequest{
		AppID:     a.defaultAppID(),
		Email:     email,
		Password:  password,
		IPAddress: clientIPFromRequest(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		return "", auth.LoginError("Invalid email or password", links), nil
	}

	// Set the session token as a cookie for dashboard auth.
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "auth_token",
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecureRequest(r),
	})

	return a.basePath + "/", nil, nil
}

func (a *authPages) handleRegister(ctx *router.PageContext) (string, templ.Component, error) {
	r := ctx.Request

	if err := r.ParseForm(); err != nil {
		comp, _ := a.renderRegisterPage(ctx, "Invalid form data", nil, nil)
		return "", comp, nil
	}

	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Collect meta_* fields into metadata map.
	metadata := make(map[string]string)
	for key, vals := range r.Form {
		if strings.HasPrefix(key, "meta_") && len(vals) > 0 {
			metadata[strings.TrimPrefix(key, "meta_")] = vals[0]
		}
	}

	// Build values map for re-render on error.
	values := map[string]string{
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
	}
	for k, v := range metadata {
		values[k] = v
	}

	if email == "" || password == "" {
		comp, _ := a.renderRegisterPage(ctx, "Email and password are required", values, nil)
		return "", comp, nil
	}

	req := &account.SignUpRequest{
		AppID:     a.defaultAppID(),
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		IPAddress: clientIPFromRequest(r),
		UserAgent: r.UserAgent(),
	}
	if len(metadata) > 0 {
		req.Metadata = metadata
	}

	_, sess, err := a.engine.SignUp(r.Context(), req)
	if err != nil {
		comp, _ := a.renderRegisterPage(ctx, err.Error(), values, nil)
		return "", comp, nil
	}

	// Set the session token as a cookie for dashboard auth.
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "auth_token",
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecureRequest(r),
	})

	return a.basePath + "/", nil, nil
}

func (a *authPages) handleForgotPassword(ctx *router.PageContext) (string, templ.Component, error) {
	r := ctx.Request
	links := forgotPasswordLinks(a.basePath)

	if err := r.ParseForm(); err != nil {
		return "", auth.ForgotPasswordError("Invalid form data", links), nil
	}

	email := r.FormValue("email")
	if email == "" {
		return "", auth.ForgotPasswordError("Email is required", links), nil
	}

	// Always show success to prevent email enumeration.
	_, _ = a.engine.ForgotPassword(r.Context(), a.defaultAppID(), email)

	return "", auth.ForgotPasswordSuccess(links), nil
}

func (a *authPages) handleLogout(ctx *router.PageContext) (string, templ.Component, error) {
	r := ctx.Request

	// Resolve and terminate the server-side session.
	token := extractToken(r)
	if token != "" {
		if sess, err := a.engine.ResolveSessionByToken(token); err == nil {
			_ = a.engine.SignOut(r.Context(), sess.ID)
		}
	}

	// Clear the auth cookie.
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecureRequest(r),
	})

	return a.basePath + "/login", nil, nil
}

// authChecker implements dashauth.AuthChecker for the authsome extension.
type authChecker struct {
	engine *authsome.Engine
}

// Ensure authChecker implements AuthChecker at compile time.
var _ dashauth.AuthChecker = (*authChecker)(nil)

// CheckAuth inspects the request and returns a UserInfo if authenticated.
func (c *authChecker) CheckAuth(ctx context.Context, r *http.Request) (*dashauth.UserInfo, error) {
	token := extractToken(r)
	if token == "" {
		return nil, nil
	}

	sess, err := c.engine.ResolveSessionByToken(token)
	if err != nil {
		return nil, nil
	}

	u, err := c.engine.ResolveUser(sess.UserID.String())
	if err != nil {
		return nil, nil
	}

	return userToUserInfo(u), nil
}

func userToUserInfo(u *user.User) *dashauth.UserInfo {
	return &dashauth.UserInfo{
		Subject:     u.ID.String(),
		DisplayName: u.Name(),
		Email:       u.Email,
		AvatarURL:   u.Image,
	}
}

func extractToken(r *http.Request) string {
	// Check Authorization header first.
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
	}

	// Fall back to cookie.
	if cookie, err := r.Cookie("auth_token"); err == nil {
		return cookie.Value
	}

	return ""
}

func isSecureRequest(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func clientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}
