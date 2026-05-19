package contract

import (
	"context"
	"errors"
	"net/http"
	"strings"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
)

// dashboardCookieName mirrors the constant in extension/auth_pages.go. It's
// duplicated here rather than exported because the legacy templ flow and
// the contract flow both need to write the SAME cookie (the contract
// log-in must satisfy a subsequent /principal call going through the
// existing authChecker).
const dashboardCookieName = "auth_token"

// LoginInput is the wire shape for the auth.login command. The React
// shell's form.edit at /login submits {email, password}; password is
// never echoed back, only consumed.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is what the shell receives on success. The principal
// endpoint will reflect the new session on the next call so this is
// intentionally lean — the cookie is the load-bearing artifact.
type LoginResponse struct {
	OK      bool   `json:"ok"`
	Subject string `json:"subject"`
}

// LogoutResponse mirrors LoginResponse.
type LogoutResponse struct {
	OK bool `json:"ok"`
}

func loginHandler(deps Deps) func(ctx context.Context, in LoginInput, p contract.Principal) (LoginResponse, error) {
	return func(ctx context.Context, in LoginInput, _ contract.Principal) (LoginResponse, error) {
		email := strings.TrimSpace(in.Email)
		if email == "" || in.Password == "" {
			return LoginResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "email and password are required"}
		}
		eng := deps.Engine
		if eng == nil {
			return LoginResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}

		httpReq := dashauth.RequestFromContext(ctx)
		httpRes := dashauth.ResponseWriterFromContext(ctx)
		if httpRes == nil || httpReq == nil {
			// The transport is supposed to populate these via dashauth.WithHTTP
			// before dispatching commands. If we don't get them we can't write
			// the session cookie — fail loudly rather than silently issuing a
			// session the shell can't see.
			return LoginResponse{}, &contract.Error{Code: contract.CodeInternal, Message: "no http context (forge >= dashauth.WithHTTP required)"}
		}

		signInReq := &account.SignInRequest{
			AppID:     defaultAppID(eng),
			Email:     strings.ToLower(email),
			Password:  in.Password,
			IPAddress: clientIP(httpReq),
			UserAgent: httpReq.UserAgent(),
		}
		u, sess, err := eng.SignIn(ctx, signInReq)
		if err != nil {
			return LoginResponse{}, mapSignInError(err)
		}

		setSessionCookie(httpRes, httpReq, eng, sess.Token, secureForRequest(httpReq, deps))

		subject := ""
		if u != nil {
			subject = u.ID.String()
		}
		return LoginResponse{OK: true, Subject: subject}, nil
	}
}

func logoutHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (LogoutResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (LogoutResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return LogoutResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}

		httpReq := dashauth.RequestFromContext(ctx)
		httpRes := dashauth.ResponseWriterFromContext(ctx)
		if httpRes == nil || httpReq == nil {
			return LogoutResponse{}, &contract.Error{Code: contract.CodeInternal, Message: "no http context"}
		}

		// Resolve the live session token (cookie or bearer) and terminate the
		// server-side session. Best-effort: even if SignOut errors we still
		// clear the client cookie so the shell stops thinking it's signed in.
		if token := extractToken(httpReq); token != "" {
			if sess, err := eng.ResolveSessionByToken(token); err == nil && sess != nil {
				_ = eng.SignOut(ctx, sess.ID) //nolint:errcheck // best-effort sign out
			}
		}
		clearSessionCookie(httpRes, httpReq, eng, secureForRequest(httpReq, deps))
		return LogoutResponse{OK: true}, nil
	}
}

// mapSignInError translates authsome's domain errors into wire codes the
// React shell's LoginScreen / form.edit can render. The legacy templ flow
// surfaces the same set; the message strings stay short so they fit in
// the inline error block.
func mapSignInError(err error) error {
	switch {
	case errors.Is(err, account.ErrInvalidCredentials):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Invalid email or password"}
	case errors.Is(err, account.ErrEmailNotVerified):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Please verify your email address before signing in"}
	case errors.Is(err, account.ErrAccountLocked):
		return &contract.Error{Code: contract.CodePermissionDenied, Message: "Account temporarily locked due to too many failed attempts"}
	case errors.Is(err, account.ErrUserBanned):
		return &contract.Error{Code: contract.CodePermissionDenied, Message: "Account has been suspended"}
	case errors.Is(err, account.ErrPasswordExpired):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Password has expired and must be changed"}
	case errors.Is(err, authsome.ErrNotStarted):
		return &contract.Error{Code: contract.CodeUnavailable, Message: "System is still initializing. Please try again in a moment."}
	default:
		// Don't leak the underlying message — generic credential failure.
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Invalid email or password"}
	}
}

func defaultAppID(eng *authsome.Engine) id.AppID {
	if platformID := eng.PlatformAppID(); !platformID.IsNil() {
		return platformID
	}
	parsed, _ := id.ParseAppID(eng.Config().AppID) //nolint:errcheck // best-effort parse
	return parsed
}

func clientIP(r *http.Request) string {
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

func extractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
	}
	if c, err := r.Cookie(dashboardCookieName); err == nil {
		return c.Value
	}
	// Try with the __Host- prefix variant.
	if c, err := r.Cookie("__Host-" + dashboardCookieName); err == nil {
		return c.Value
	}
	return ""
}

func secureForRequest(r *http.Request, deps Deps) bool {
	if deps.CookieSecure != nil {
		return *deps.CookieSecure
	}
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// setSessionCookie writes the dashboard auth_token cookie matching the
// templ flow's attributes (resolved through SessionCookieTemplate). The
// __Host- prefix is honoured when SettingCookieUseHostPrefix is on.
func setSessionCookie(w http.ResponseWriter, r *http.Request, eng *authsome.Engine, token string, secure bool) {
	c := dashboardCookieTemplate(r.Context(), eng, secure) // #nosec G124 -- template sets HttpOnly+SameSite+Secure
	c.Value = token
	http.SetCookie(w, c)
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request, eng *authsome.Engine, secure bool) {
	c := dashboardCookieTemplate(r.Context(), eng, secure) // #nosec G124 -- template sets HttpOnly+SameSite+Secure
	c.Value = ""
	c.MaxAge = -1
	http.SetCookie(w, c)
}

func dashboardCookieTemplate(ctx context.Context, eng *authsome.Engine, secure bool) *http.Cookie {
	if eng == nil {
		return &http.Cookie{ // #nosec G124 -- HttpOnly+SameSite+Secure set below
			Name:     dashboardCookieName,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   secure,
		}
	}
	mgr := eng.Settings()
	if mgr == nil {
		return &http.Cookie{ // #nosec G124 -- HttpOnly+SameSite+Secure set below
			Name:     dashboardCookieName,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   secure,
		}
	}
	c := authsome.SessionCookieTemplate(ctx, mgr, "", secure) // #nosec G124 -- template sets HttpOnly+SameSite+Secure
	c.Name = applyHostPrefix(c.Name, dashboardCookieName)
	return c
}

func applyHostPrefix(templated, baseName string) string {
	const hostPrefix = "__Host-"
	if strings.HasPrefix(templated, hostPrefix) {
		return hostPrefix + baseName
	}
	return baseName
}
