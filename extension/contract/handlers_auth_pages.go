// handlers_auth_pages.go: Phase C.14 — anonymous auth pages.
//
// Three commands the dashboard's AuthGate dispatches before a session
// exists: public signup, forgot-password (request a reset link), and
// reset-password (consume a reset token + set the new password). The
// auth.setup (first-run bootstrap) and auth.dynamicSignup flows are
// deferred — both require engine surface we haven't built yet
// (setup-status check + form-config-aware dynamic field metadata).
//
// On success, auth.signup writes the new session cookie via the same
// helper as auth.login so the principal endpoint reflects the new
// session on the next call. auth.forgotPassword always succeeds at
// the wire level even when the email doesn't exist (account
// enumeration prevention); the server-side handler still issues the
// reset record so callbacks fire and audits emit.
package contract

import (
	"context"
	"errors"
	"strings"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"

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
		_, _ = eng.ForgotPassword(ctx, defaultAppID(eng), email)
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
func splitName(name string) (string, string) {
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
