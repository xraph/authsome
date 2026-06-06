package api

import (
	"errors"
	"net/http"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/store"
)

// codedHTTPError is a forge IHTTPError that carries a stable string error
// type alongside the numeric HTTP status code. It widens the default
// {"error": message, "code": <int>} response body with a "type" field so
// SDK consumers can branch on the failure mode (e.g. email_not_verified)
// without parsing human-readable messages or relying on the HTTP status
// alone.
//
// Forge's router serialises any error implementing
// {error, StatusCode() int, ResponseBody() any} via its handleError
// path (forge/internal/router/handler.go), so satisfying that interface
// is enough to swap the wire shape.
type codedHTTPError struct {
	status  int
	typeStr string
	message string
	extras  map[string]any
}

func (e *codedHTTPError) Error() string   { return e.message }
func (e *codedHTTPError) StatusCode() int { return e.status }
func (e *codedHTTPError) ResponseBody() any {
	body := map[string]any{
		"error": e.message,
		"code":  e.status,
		"type":  e.typeStr,
	}
	for k, v := range e.extras {
		// Reserved keys take precedence so callers can't accidentally
		// shadow the canonical envelope fields.
		if _, reserved := body[k]; reserved {
			continue
		}
		body[k] = v
	}
	return body
}

// newCodedError returns a forge-compatible HTTP error carrying a stable
// string type code for SDK consumers to branch on.
func newCodedError(status int, typeStr, message string) error {
	return &codedHTTPError{status: status, typeStr: typeStr, message: message}
}

// newCodedErrorWithExtras returns a coded HTTP error with additional
// response-body fields merged alongside the canonical {error, code,
// type} envelope. Used for errors whose remediation requires
// out-of-band data — e.g. mfa_required carries the ticket and the
// available methods so the client can complete the round-trip.
func newCodedErrorWithExtras(status int, typeStr, message string, extras map[string]any) error {
	return &codedHTTPError{status: status, typeStr: typeStr, message: message, extras: extras}
}

// mapError converts domain errors into Forge HTTP errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return forge.NotFound(err.Error())
	}
	if errors.Is(err, account.ErrInvalidCredentials) {
		return forge.Unauthorized("invalid credentials")
	}
	if errors.Is(err, account.ErrTooManyAttempts) {
		return forge.NewHTTPError(http.StatusTooManyRequests, "too many verification attempts; request a new code")
	}
	if errors.Is(err, account.ErrEmailTaken) {
		return forge.NewHTTPError(http.StatusConflict, "email already taken")
	}
	if errors.Is(err, account.ErrUsernameTaken) {
		return forge.NewHTTPError(http.StatusConflict, "username already taken")
	}
	if errors.Is(err, store.ErrConflict) {
		// Generic backend uniqueness violation — the store layer didn't
		// recognize the constraint so we return a vague 409 rather than
		// leaking the index name and key value via the raw driver error.
		return forge.NewHTTPError(http.StatusConflict, "conflict")
	}
	if errors.Is(err, account.ErrMFARequired) {
		// Surface the ticket + available methods so the UI can
		// transition from sign-in to challenge without a second
		// round-trip to look the methods up.
		var mfaErr *authsome.MFARequiredError
		if errors.As(err, &mfaErr) {
			return newCodedErrorWithExtras(http.StatusForbidden, "mfa_required",
				"multi-factor authentication is required",
				map[string]any{
					"mfa_ticket":        mfaErr.Ticket,
					"available_methods": mfaErr.AvailableMethods,
				})
		}
		// Defensive fallback: an MFA error without the wrapper means
		// IssueSession wasn't on this code path. Still surface 403 so
		// the SDK can branch correctly even without ticket data.
		return newCodedError(http.StatusForbidden, "mfa_required",
			"multi-factor authentication is required")
	}
	if errors.Is(err, account.ErrEmailNotVerified) {
		// Phase 2A: SettingRequireEmailVerification defaults to true,
		// so unverified accounts now hit this path on every signin.
		// Map to 403 with a stable string code matching the existing
		// dashboard / extension handling so SDK consumers can prompt
		// the user to verify rather than reporting an opaque 500.
		return newCodedError(http.StatusForbidden, "email_not_verified",
			"please verify your email address before signing in")
	}
	if errors.Is(err, account.ErrUserBanned) {
		return forge.Forbidden("user is banned")
	}
	if errors.Is(err, account.ErrSessionExpired) {
		return forge.Unauthorized("session expired")
	}
	if errors.Is(err, account.ErrWeakPassword) {
		return forge.BadRequest(err.Error())
	}
	return forge.InternalError(err)
}
