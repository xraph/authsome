package authsome

import (
	"context"
	"errors"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// ErrDPoPBindingRequired reports a session refused because the app mandates
// DPoP (dpop.mode = required) and the issuing path resolved no thumbprint to
// bind it to.
//
// This exists because enforcement follows the token, not the app: the
// middleware's enforceDPoP returns nil the moment it sees an empty thumbprint,
// so an unbound session is exempt from proof-of-possession for its whole life
// however the app is configured. A mint site that forgets to resolve a binding
// therefore doesn't produce a weaker session, it produces one the mandate
// never applies to at all. Refusing at the mint is the only place that
// distinction can still be made.
var ErrDPoPBindingRequired = errors.New("authsome: this app requires a DPoP-bound session")

// DPoPRequiredError renders ErrDPoPBindingRequired over HTTP in the shape RFC
// 9449 clients already watch for, matching what the OAuth2 token endpoint
// returns when a required-mode client presents no proof.
type DPoPRequiredError struct{}

// Error returns the underlying sentinel's message.
func (*DPoPRequiredError) Error() string { return ErrDPoPBindingRequired.Error() }

// Unwrap exposes the sentinel for errors.Is checks.
func (*DPoPRequiredError) Unwrap() error { return ErrDPoPBindingRequired }

// StatusCode lets forge render this directly, without every plugin handler
// needing its own mapping.
func (*DPoPRequiredError) StatusCode() int { return http.StatusBadRequest }

// ResponseBody returns the JSON envelope the client receives.
func (*DPoPRequiredError) ResponseBody() any {
	return map[string]any{
		"error":             "invalid_dpop_proof",
		"error_description": "this app requires a DPoP-bound token; present a DPoP proof",
	}
}

// dpopNonceChallengeError is the first-party equivalent of the OAuth2
// plugin's use_dpop_nonce response (RFC 9449 §8): 400, with the error code a
// DPoP client looks for to know it should retry carrying the DPoP-Nonce
// header value the caller already set on the response before constructing
// this. Unlike bindDPoP's caller, the paths that return this have no
// single-use artifact to protect, so nothing needs reordering, just the same
// response shape.
type dpopNonceChallengeError struct{}

func (dpopNonceChallengeError) Error() string { return "use_dpop_nonce" }

func (dpopNonceChallengeError) StatusCode() int { return http.StatusBadRequest }

func (dpopNonceChallengeError) ResponseBody() any {
	return map[string]any{
		"error":             "use_dpop_nonce",
		"error_description": "a server-provided nonce is required",
	}
}

// DPoPBindingForRequest validates any DPoP proof on a first-party auth request
// and returns the thumbprint to bind the new session to. An empty string with
// a nil error means "no binding, and that is allowed here".
//
// Mirrors the OAuth2 plugin's bindDPoP. There is no OAuth client in a
// first-party flow, so only the app's mode applies.
//
// This lives on the Engine rather than in the api package because every
// issuing path needs it: the first-party handlers, and the magic-link,
// passkey, phone and MFA plugins, none of which can reach an unexported api
// method. Seven copies of proof validation is seven chances to get one of
// them subtly wrong.
func (e *Engine) DPoPBindingForRequest(ctx forge.Context, appID id.AppID) (string, error) {
	mode := e.DPoPModeForApp(ctx.Context(), appID)
	if mode == dpop.ModeOff {
		return "", nil
	}

	raw := ctx.Request().Header.Get("DPoP")
	if raw == "" {
		if mode == dpop.ModeRequired {
			return "", &DPoPRequiredError{}
		}
		return "", nil
	}

	proof, err := dpop.Parse(raw)
	if err != nil {
		return "", forge.BadRequest("invalid DPoP proof")
	}

	nonceRequired := e.DPoPNonceRequiredForApp(ctx.Context(), appID)

	if err := e.DPoPValidator().Validate(ctx.Context(), proof, dpop.Expectation{
		Method:        ctx.Request().Method,
		URL:           middleware.RequestURL(ctx.Request()),
		NonceRequired: nonceRequired,
	}); err != nil {
		if errors.Is(err, dpop.ErrNonceRequired) || errors.Is(err, dpop.ErrNonceMismatch) {
			// A pre-authentication client has no other endpoint to obtain a
			// nonce from, so unlike a resource-server 401 this must be a
			// retryable challenge: 400 with the code the client watches for,
			// plus a fresh nonce it can carry on the very next attempt.
			// Without this, a nonce-required app rejects every DPoP-presenting
			// client permanently, and under mode required that's a total
			// lockout on every first-party sign-in route.
			if signer := e.DPoPNonceSigner(); signer != nil {
				ctx.Response().Header().Set("DPoP-Nonce", signer.Issue(proof.JKT))
			}
			return "", dpopNonceChallengeError{}
		}
		return "", forge.BadRequest("invalid DPoP proof")
	}
	return proof.JKT, nil
}

// DPoPBindingUnavailable returns a refusal when appID mandates DPoP but the
// caller structurally cannot obtain a proof.
//
// The IdP-redirect callbacks are the case: at a social or SSO callback the
// request is the identity provider redirecting the user agent, so the client
// holding the DPoP key is not the caller and has no opportunity to present
// anything. IssueSession's gate would refuse these anyway, but only after the
// authorization code had been redeemed and burned. Calling this first turns a
// consumed single-use artifact into a clean error.
//
// The refusal is deliberate rather than an exemption: an app on required is
// stating a mandate, and a path that quietly issued unbound sessions would let
// anyone who can pick the sign-in method opt out of it. An operator who needs
// social or SSO on such an app lowers the app to optional, which is explicit,
// visible and audited.
func (e *Engine) DPoPBindingUnavailable(ctx context.Context, appID id.AppID) error {
	if e.DPoPModeForApp(ctx, appID) == dpop.ModeRequired {
		return &DPoPRequiredError{}
	}
	return nil
}

// ImpersonateOption configures Engine.Impersonate. Variadic so the existing
// three-argument call shape keeps compiling.
type ImpersonateOption func(*impersonateOpts)

type impersonateOpts struct {
	dpopJKT string
}

// WithImpersonationDPoPBinding binds the impersonation session to a
// client-held key.
//
// Impersonation mints outside IssueSession, so it needs the mandate applied
// explicitly. It is worth applying: an impersonation session acts as another
// user, which makes it the one session you would least want exempt from
// proof-of-possession on an app that requires it. The admin console asking for
// the impersonation is an ordinary HTTP client and can present a proof like
// any other.
func WithImpersonationDPoPBinding(jkt string) ImpersonateOption {
	return func(o *impersonateOpts) { o.dpopJKT = jkt }
}
