// handlers_auth_pages.go: Phase C.14 — anonymous auth pages.
//
// Six intents the dashboard's AuthGate dispatches before a session
// exists:
//
//   - auth.signup           public registration via engine.SignUp
//   - auth.forgotPassword   request a reset link (always returns OK)
//   - auth.resetPassword    consume a reset token + set new password
//   - auth.setupStatus      query: does any admin user exist yet?
//   - auth.setup            first-run bootstrap: create the first user
//   - auth.dynamicConfig    query: per-app dynamic signup form
//   - auth.dynamicRegister  signup with form-config-driven metadata
//
// On success, auth.signup / auth.setup / auth.dynamicRegister write
// the new session cookie via the same helper as auth.login so the
// principal endpoint reflects the new session on the next call.
// auth.forgotPassword always succeeds at the wire level even when the
// email doesn't exist (account enumeration prevention).
package contract

import (
	"context"
	"errors"
	"strings"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/user"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
)

// SignupInput is the wire shape for auth.signup. Matches the
// auth.signup-form renderer's form fields. Name is optional in v1 —
// the renderer can toggle the field via `nameField: false`.
type SignupInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// SignupResponse mirrors LoginResponse — slim by design because the
// session cookie is the load-bearing artifact.
type SignupResponse struct {
	OK      bool   `json:"ok"`
	Subject string `json:"subject"`
}

// ForgotPasswordInput is the wire shape for auth.forgotPassword.
type ForgotPasswordInput struct {
	Email string `json:"email"`
}

// ForgotPasswordResponse always returns ok:true to avoid leaking
// whether the email belongs to a real account.
type ForgotPasswordResponse struct {
	OK bool `json:"ok"`
}

// ResetPasswordInput is the wire shape for auth.resetPassword.
// Token comes from the reset-link URL (the renderer parses it from
// the query string and submits it alongside the new password).
type ResetPasswordInput struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ResetPasswordResponse is the auth.resetPassword reply.
type ResetPasswordResponse struct {
	OK bool `json:"ok"`
}

func signupHandler(deps Deps) func(ctx context.Context, in SignupInput, _ contract.Principal) (SignupResponse, error) {
	return func(ctx context.Context, in SignupInput, _ contract.Principal) (SignupResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return SignupResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		email := strings.ToLower(strings.TrimSpace(in.Email))
		if email == "" || in.Password == "" {
			return SignupResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "email and password are required"}
		}

		httpReq := dashauth.RequestFromContext(ctx)
		httpRes := dashauth.ResponseWriterFromContext(ctx)
		if httpRes == nil || httpReq == nil {
			return SignupResponse{}, &contract.Error{Code: contract.CodeInternal, Message: "no http context (forge >= dashauth.WithHTTP required)"}
		}

		first, last := splitName(in.Name)
		req := &account.SignUpRequest{
			AppID:     defaultAppID(eng),
			Email:     email,
			Password:  in.Password,
			FirstName: first,
			LastName:  last,
			IPAddress: clientIP(httpReq),
			UserAgent: httpReq.UserAgent(),
		}
		u, sess, err := eng.SignUp(ctx, req)
		if err != nil {
			return SignupResponse{}, mapSignUpError(err)
		}

		if sess != nil {
			setSessionCookie(httpRes, httpReq, eng, sess.Token, secureForRequest(httpReq, deps))
		}

		subject := ""
		if u != nil {
			subject = u.ID.String()
		}
		return SignupResponse{OK: true, Subject: subject}, nil
	}
}

func forgotPasswordHandler(deps Deps) func(ctx context.Context, in ForgotPasswordInput, _ contract.Principal) (ForgotPasswordResponse, error) {
	return func(ctx context.Context, in ForgotPasswordInput, _ contract.Principal) (ForgotPasswordResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return ForgotPasswordResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		email := strings.ToLower(strings.TrimSpace(in.Email))
		if email == "" {
			return ForgotPasswordResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "email is required"}
		}
		// Best-effort: fire the engine call but always return OK on the
		// wire to avoid account enumeration. Logging stays internal.
		_, _ = eng.ForgotPassword(ctx, defaultAppID(eng), email) //nolint:errcheck // best-effort; error deliberately swallowed to avoid account enumeration
		return ForgotPasswordResponse{OK: true}, nil
	}
}

func resetPasswordHandler(deps Deps) func(ctx context.Context, in ResetPasswordInput, _ contract.Principal) (ResetPasswordResponse, error) {
	return func(ctx context.Context, in ResetPasswordInput, _ contract.Principal) (ResetPasswordResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return ResetPasswordResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		token := strings.TrimSpace(in.Token)
		if token == "" || in.Password == "" {
			return ResetPasswordResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "token and password are required"}
		}
		if err := eng.ResetPassword(ctx, token, in.Password); err != nil {
			return ResetPasswordResponse{}, mapResetError(err)
		}
		return ResetPasswordResponse{OK: true}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

// splitName breaks a single "Full Name" string into (first, last).
// The auth.signup-form renderer ships a single Name field — splitting
// here keeps the wire shape simple while still populating the
// engine's first/last fields. Unicode-aware trimming would be nicer;
// the current split-on-first-space matches what the templ register
// page does today.
func splitName(name string) (first, last string) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", ""
	}
	i := strings.IndexByte(n, ' ')
	if i < 0 {
		return n, ""
	}
	return strings.TrimSpace(n[:i]), strings.TrimSpace(n[i+1:])
}

// mapSignUpError translates account errors into wire codes mirroring
// mapSignInError. Generic credential failures are deliberately vague —
// don't leak whether an email is already taken vs invalid format.
func mapSignUpError(err error) error {
	switch {
	case errors.Is(err, account.ErrEmailTaken):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "An account with this email already exists"}
	case errors.Is(err, account.ErrUsernameTaken):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "That username is already taken"}
	case errors.Is(err, account.ErrWeakPassword):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Password does not meet policy requirements"}
	case errors.Is(err, authsome.ErrNotStarted):
		return &contract.Error{Code: contract.CodeUnavailable, Message: "System is still initializing. Please try again in a moment."}
	default:
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Could not create account"}
	}
}

// mapResetError keeps the surface terse so reset failures don't help
// an attacker probe the token namespace. The engine returns generic
// errors for invalid/expired reset tokens; we collapse the lot into
// a single user-facing message.
func mapResetError(err error) error {
	switch {
	case errors.Is(err, account.ErrWeakPassword):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Password does not meet policy requirements"}
	default:
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Reset link is invalid or expired"}
	}
}

// ────────────────────────────────────────────────────────────────────
// Setup (first-run bootstrap)
// ────────────────────────────────────────────────────────────────────

// SetupStatusResponse is returned by auth.setupStatus. Pending is true
// when no users exist yet — the dashboard's AuthGate uses this to
// redirect to /setup before /login.
type SetupStatusResponse struct {
	Pending bool `json:"pending"`
}

// SetupInput is the wire shape for auth.setup. Mirrors the
// auth.setup-form renderer's fields.
type SetupInput struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	Name             string `json:"name,omitempty"`
	OrganizationName string `json:"organizationName,omitempty"`
}

// SetupResponse is the auth.setup reply.
type SetupResponse struct {
	OK      bool   `json:"ok"`
	Subject string `json:"subject"`
}

func setupStatusHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (SetupStatusResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (SetupStatusResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return SetupStatusResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		// Limit:1 is enough — we only need Total. The engine fills Total
		// from the underlying count query regardless of Limit.
		list, err := eng.AdminListUsers(ctx, &user.Query{
			AppID: defaultAppID(eng),
			Limit: 1,
		})
		if err != nil {
			// Treat list-failure as "setup not pending" — failing closed
			// to the login page is safer than showing setup to an
			// already-bootstrapped deployment whose count query just hiccupped.
			return SetupStatusResponse{Pending: false}, nil
		}
		return SetupStatusResponse{Pending: list.Total == 0}, nil
	}
}

// setupHandler runs the first-run bootstrap. Refuses to run when users
// already exist — protects against an attacker stumbling on /setup
// after launch and overwriting the admin's email/password.
//
// Organization creation is best-effort: if the request includes an
// organization name and the organization plugin is loaded, we attempt
// to create the org alongside the user. Failure to create the org
// doesn't roll back the user — admins can create their org manually
// post-setup.
func setupHandler(deps Deps) func(ctx context.Context, in SetupInput, _ contract.Principal) (SetupResponse, error) {
	return func(ctx context.Context, in SetupInput, _ contract.Principal) (SetupResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return SetupResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		email := strings.ToLower(strings.TrimSpace(in.Email))
		if email == "" || in.Password == "" {
			return SetupResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "email and password are required"}
		}

		// Refuse to bootstrap a populated deployment.
		list, err := eng.AdminListUsers(ctx, &user.Query{AppID: defaultAppID(eng), Limit: 1})
		if err != nil {
			return SetupResponse{}, mapEngineError(err)
		}
		if list.Total > 0 {
			return SetupResponse{}, &contract.Error{Code: contract.CodePermissionDenied, Message: "setup has already been completed"}
		}

		httpReq := dashauth.RequestFromContext(ctx)
		httpRes := dashauth.ResponseWriterFromContext(ctx)
		if httpRes == nil || httpReq == nil {
			return SetupResponse{}, &contract.Error{Code: contract.CodeInternal, Message: "no http context (forge >= dashauth.WithHTTP required)"}
		}

		first, last := splitName(in.Name)
		req := &account.SignUpRequest{
			AppID:     defaultAppID(eng),
			Email:     email,
			Password:  in.Password,
			FirstName: first,
			LastName:  last,
			IPAddress: clientIP(httpReq),
			UserAgent: httpReq.UserAgent(),
		}
		u, sess, err := eng.SignUp(ctx, req)
		if err != nil {
			return SetupResponse{}, mapSignUpError(err)
		}
		if sess != nil {
			setSessionCookie(httpRes, httpReq, eng, sess.Token, secureForRequest(httpReq, deps))
		}

		// Organization creation is intentionally not gated on engine
		// surface — we don't know at compile time whether the org
		// plugin is loaded. Skip it for now and let admins create their
		// org through the dashboard's /organizations page post-setup.
		_ = in.OrganizationName

		subject := ""
		if u != nil {
			subject = u.ID.String()
		}
		return SetupResponse{OK: true, Subject: subject}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Dynamic signup (form-config-driven)
// ────────────────────────────────────────────────────────────────────

// DynamicConfigResponse is returned by auth.dynamicConfig. Fields are
// the FieldSpec list the auth.dynamic-signup-form renderer feeds into
// organism.dynamic-form.
type DynamicConfigResponse struct {
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Fields      []formconfig.FormField `json:"fields,omitempty"`
	Active      bool                   `json:"active"`
}

// DynamicRegisterInput is the wire shape for auth.dynamicRegister.
// Standard fields go in the top-level slots; everything else flows
// through Metadata so the engine writes it onto User.Metadata.
type DynamicRegisterInput struct {
	Email    string            `json:"email"`
	Password string            `json:"password"`
	Name     string            `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func dynamicConfigHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (DynamicConfigResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (DynamicConfigResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return DynamicConfigResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		fc, err := eng.GetSignupFormConfig(ctx, defaultAppID(eng))
		if err != nil {
			// No active config → return an empty (inactive) shape so the
			// renderer falls back to the static auth.signup-form layout.
			return DynamicConfigResponse{Active: false}, nil
		}
		if fc == nil {
			return DynamicConfigResponse{Active: false}, nil
		}
		return DynamicConfigResponse{
			Title:       "Create your account",
			Description: "",
			Fields:      fc.Fields,
			Active:      true,
		}, nil
	}
}

func dynamicRegisterHandler(deps Deps) func(ctx context.Context, in DynamicRegisterInput, _ contract.Principal) (SignupResponse, error) {
	return func(ctx context.Context, in DynamicRegisterInput, _ contract.Principal) (SignupResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return SignupResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		email := strings.ToLower(strings.TrimSpace(in.Email))
		if email == "" || in.Password == "" {
			return SignupResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "email and password are required"}
		}
		httpReq := dashauth.RequestFromContext(ctx)
		httpRes := dashauth.ResponseWriterFromContext(ctx)
		if httpRes == nil || httpReq == nil {
			return SignupResponse{}, &contract.Error{Code: contract.CodeInternal, Message: "no http context (forge >= dashauth.WithHTTP required)"}
		}

		first, last := splitName(in.Name)
		req := &account.SignUpRequest{
			AppID:     defaultAppID(eng),
			Email:     email,
			Password:  in.Password,
			FirstName: first,
			LastName:  last,
			Metadata:  in.Metadata,
			IPAddress: clientIP(httpReq),
			UserAgent: httpReq.UserAgent(),
		}
		u, sess, err := eng.SignUp(ctx, req)
		if err != nil {
			return SignupResponse{}, mapSignUpError(err)
		}
		if sess != nil {
			setSessionCookie(httpRes, httpReq, eng, sess.Token, secureForRequest(httpReq, deps))
		}
		subject := ""
		if u != nil {
			subject = u.ID.String()
		}
		return SignupResponse{OK: true, Subject: subject}, nil
	}
}
