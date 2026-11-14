package authsome

import (
	"context"
	"errors"
	"fmt"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// Codes a refused sign-up carries in SignUpRefusedError.Code. Browser
// callbacks hand the code back to the frontend as a query parameter, so
// these are part of the API.
const (
	// SignUpRefusedDisabled means the app has sign-up switched off.
	SignUpRefusedDisabled = "signup_disabled"
	// SignUpRefusedWaitlist means the waitlist plugin refused: the address
	// has no approved entry.
	SignUpRefusedWaitlist = "waitlist_required"
	// SignUpRefusedHook means some other BeforeSignUp hook refused.
	SignUpRefusedHook = "signup_refused"
)

// SignUpRefusal is implemented by an error a BeforeSignUp hook returns when
// it wants the refusal reported under its own code rather than
// SignUpRefusedHook. The code should be a short snake_case token.
type SignUpRefusal interface {
	SignUpRefusalCode() string
}

// SignUpRefusedError is returned by GateFederatedSignUp when an account may
// not be created: the app has sign-up switched off, or a BeforeSignUp hook
// refused. It answers 403 with type "signup_refused" and the code as
// "reason", and never echoes the hook's own message, which may carry
// internal detail.
type SignUpRefusedError struct {
	// Code is one of the SignUpRefused* constants, or a hook's own code.
	Code string
	// Err is the underlying refusal.
	Err error
}

// Error describes the refusal.
func (e *SignUpRefusedError) Error() string {
	if e.Err == nil {
		return "authsome: sign-up refused: " + e.Code
	}
	return "authsome: sign-up refused: " + e.Err.Error()
}

// Unwrap exposes the hook's error.
func (e *SignUpRefusedError) Unwrap() error { return e.Err }

// StatusCode renders the refusal as a 403.
func (e *SignUpRefusedError) StatusCode() int { return 403 }

// ResponseBody is the JSON envelope for callers that are not redirected.
func (e *SignUpRefusedError) ResponseBody() any {
	return map[string]any{
		"error":  "sign-up is not allowed",
		"code":   403,
		"type":   "signup_refused",
		"reason": e.Code,
	}
}

// signUpDisabled reports whether the app has switched sign-up off.
func (e *Engine) signUpDisabled(ctx context.Context, appID id.AppID) bool {
	cfg, err := e.store.GetAppClientConfig(ctx, appID)
	return err == nil && cfg != nil && cfg.SignupEnabled != nil && !*cfg.SignupEnabled
}

// GateFederatedSignUp runs the checks an account created outside SignUp has
// to pass, for a first sign-in through a social provider or an SSO
// connection: the app's sign-up switch, then every BeforeSignUp plugin hook
// (waitlist, IP reputation, geofencing, a host's own gates). Call it before
// the user is written, and only when one is about to be; an existing account
// signing in is never gated.
//
// What SignUp checks on a password is left out: there is no password, so no
// policy or breach check. Captcha is a route-level control on the sign-up
// form and has no place here either; the identity provider round trip is the
// proof a person is present. req.AuthMethod tells hooks which path this is.
//
// A refusal comes back as *SignUpRefusedError.
func (e *Engine) GateFederatedSignUp(ctx context.Context, req *account.SignUpRequest) error {
	if req == nil {
		return fmt.Errorf("authsome: gate sign-up: nil request")
	}
	if e.signUpDisabled(ctx, req.AppID) {
		return &SignUpRefusedError{Code: SignUpRefusedDisabled, Err: errors.New("signup is disabled for this application")}
	}
	if err := e.plugins.EmitBeforeSignUp(ctx, req); err != nil {
		code := SignUpRefusedHook
		var r SignUpRefusal
		if errors.As(err, &r) && r.SignUpRefusalCode() != "" {
			code = r.SignUpRefusalCode()
		}
		return &SignUpRefusedError{Code: code, Err: err}
	}
	return nil
}

// FinishFederatedSignUp fires the AfterSignUp plugin hooks for an account
// GateFederatedSignUp let through, once it is written. sess may be nil, as it
// is when the session is minted later or a second factor is still owed.
func (e *Engine) FinishFederatedSignUp(ctx context.Context, u *user.User, sess *session.Session) {
	if u == nil {
		return
	}
	e.plugins.EmitAfterSignUp(ctx, u, sess)
}
