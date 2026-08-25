package agentauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/session"
)

// httpError is agentauth's own carrier for a status code plus response
// headers. forge's HTTPError (github.com/xraph/forge/errors, backed by
// github.com/xraph/go-utils/errs) only carries a status code and a JSON body
// — neither forge nor go-utils exposes a header on an error anywhere in the
// dependency chain, so there is no forge.NewHTTPError(...).WithHeader(...) to
// call. httpError fills that gap while still satisfying the same
// StatusCode()/ResponseBody() shape forge's router already knows how to turn
// into a response, so an error returned here also degrades correctly if it
// ever reaches forge's generic handler directly instead of through Guard.
type httpError struct {
	status  int
	message string
	headers map[string]string
}

func newHTTPError(status int, message string) *httpError {
	return &httpError{status: status, message: message}
}

// withHeader sets a response header on the error and returns it, so
// construction reads as one expression at each call site.
func (e *httpError) withHeader(name, value string) *httpError {
	if e.headers == nil {
		e.headers = make(map[string]string, 1)
	}
	e.headers[name] = value
	return e
}

func (e *httpError) Error() string { return e.message }

// StatusCode matches the interface forge's router already checks for
// (StatusCode() int, ResponseBody() any) before falling back to a generic
// 500, so this error renders correctly even outside Guard.
func (e *httpError) StatusCode() int { return e.status }

func (e *httpError) ResponseBody() any {
	return map[string]any{"error": e.message, "code": e.status}
}

// Header returns a response header the error carries, or "" if unset.
func (e *httpError) Header(name string) string { return e.headers[name] }

// StatusOf returns the HTTP status an error carries, or 0 if it carries none.
func StatusOf(err error) int {
	var he *httpError
	if errors.As(err, &he) {
		return he.status
	}
	return 0
}

// HeaderOf returns a response header the error carries, or "" if it carries
// none or the header was never set.
func HeaderOf(err error, name string) string {
	var he *httpError
	if errors.As(err, &he) {
		return he.Header(name)
	}
	return ""
}

// opaqueDenial is the response for every failure that must not be
// distinguishable from any other: a user-gate refusal, a missing permission
// checker, and any other error Authorize did not name as a sentinel. All
// three share this single constructor call so their status, body, and
// headers are structurally identical, not just similar by convention.
//
// Its rendered body is also, deliberately, byte-identical on the wire to
// forge.Forbidden("insufficient permissions") as go-utils/errs renders it
// (both produce {"error":"insufficient permissions","code":403}) — not
// because opaqueDenial reuses that constructor (it can't; forge's type
// carries no headers, see httpError's doc comment above), but because
// matching its shape by hand means an agent cannot tell an agentauth denial
// apart from a plain middleware.RequirePermission denial on a route that
// stacks both. A host app composing the two gates does not thereby hand an
// agent a way to identify which gate refused it.
func opaqueDenial() *httpError {
	return newHTTPError(http.StatusForbidden, "insufficient permissions")
}

// AuthorizeHTTP wraps Authorize and maps its sentinels onto the responses an
// agent developer can act on.
//
// The scope case names the scope that would satisfy the route, which is safe
// because it describes the route rather than the owner. Every other failure
// — an inactive grant aside, which is a distinct re-authorize signal — stays
// opaque on purpose: reporting a user-gate refusal in the same shape as an
// RBAC outage or any other denial would let an agent enumerate its owner's
// permissions one request at a time.
//
// The returned error's headers (the WWW-Authenticate on the 401 and 403
// cases) only reach the client through Guard, or through a caller that reads
// HeaderOf itself and sets them by hand — forge has no header support on any
// error type it recognizes (see httpError's doc comment), so a caller that
// returns this error directly from a route handler gets the right status and
// body but a silently dropped header.
func (p *Plugin) AuthorizeHTTP(ctx context.Context, sess *session.Session, action, resource string) error {
	err := p.Authorize(ctx, sess, action, resource)
	switch {
	case err == nil:
		return nil

	case errors.Is(err, ErrGrantInactive):
		return newHTTPError(http.StatusUnauthorized, "agent grant is no longer valid").
			withHeader("WWW-Authenticate",
				`Bearer error="invalid_token", error_description="agent grant revoked or expired"`)

	case errors.Is(err, ErrInsufficientScope):
		return newHTTPError(http.StatusForbidden, "insufficient scope").
			withHeader("WWW-Authenticate", fmt.Sprintf(
				`Bearer error="insufficient_scope", scope=%q`, p.scopeFor(action, resource)))

	case errors.Is(err, ErrNoPermissionChecker):
		// Fail closed. Do not tell the caller why: an RBAC outage is not an
		// agent's business, and the answer must look identical to a plain
		// user-gate refusal below.
		return opaqueDenial()

	case errors.Is(err, ErrUserNotPermitted):
		// The user gate refused. Same shape as ErrNoPermissionChecker and the
		// default case: this is the response the security property in
		// AuthorizeHTTP's own doc comment exists to protect.
		return opaqueDenial()

	default:
		// Anything Authorize returns that is not one of the named sentinels
		// above (a wrapped store or RBAC transport error, for instance) gets
		// the same opaque treatment rather than leaking its text to the
		// client. Unlike the named sentinels, this branch's "why" is not
		// already implied by which case matched — a Mongo outage on the
		// grant load and a Warden transport failure both land here — so it
		// is logged server-side (matching middleware/rbac.go:36-43's
		// "log the check error, still deny the request" shape) rather than
		// discarded. httpError carries no Unwrap on purpose: logging here,
		// once, is how this text is surfaced, not letting it resurface later
		// through some caller's err.Error() on the returned httpError.
		if p.logger != nil {
			p.logger.Warn("agentauth: authorization check failed", log.Error(err))
		}
		return opaqueDenial()
	}
}

// scopeFor returns the registered scope that confers a permission, so the
// insufficient_scope response can name it. Falls back to "action:resource"
// when the host app registered no scope for it — that string is still safe
// to report, since it only echoes the route's own requirement.
//
// If two scopes happen to confer the same Permission, the lexicographically
// smallest scope name wins, deterministically. Ranging over p.scopes.scopes
// (a Go map) and returning the first hit visited would instead vary randomly
// request to request — the same route naming a different scope on every
// retry is its own kind of confusing response, even though it leaks nothing.
func (p *Plugin) scopeFor(action, resource string) string {
	p.scopes.mu.RLock()
	defer p.scopes.mu.RUnlock()
	best := ""
	for scope, perm := range p.scopes.scopes {
		if perm.Action != action || perm.Resource != resource {
			continue
		}
		if best == "" || scope < best {
			best = scope
		}
	}
	if best == "" {
		return action + ":" + resource
	}
	return best
}
