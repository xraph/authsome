// Package plugin: tenant_scope.go — the app tenancy boundary, shared between
// the core API and plugin route groups.
//
// AdminGuard answers "may this caller manage clients?". It cannot answer
// "is this client one of theirs?", because that is a property of the caller
// AND a row the handler has not loaded yet. No middleware can see that row, so
// the second check necessarily lives in handlers. These helpers are what keeps
// every handler answering it the same way.
package plugin

import (
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// CallerAppID returns the app the authenticated caller is bound to. This is
// the tenant boundary used to scope app-owned admin resources (webhooks,
// environments, OAuth2 clients, per-app config) to the caller.
//
// The credential decides: the session's app first, then a machine
// principal's app. The context app id set by the publishable-key middleware
// is consulted only when no credential is present (public routes), because
// a publishable key is public by design and must never move an authenticated
// caller into another tenant.
func CallerAppID(ctx forge.Context) (id.AppID, bool) {
	goCtx := ctx.Context()
	if sess, ok := middleware.SessionFrom(goCtx); ok && sess != nil && !sess.AppID.IsNil() {
		return sess.AppID, true
	}
	if p, ok := middleware.PrincipalFrom(goCtx); ok && p != nil && !p.AppID.IsNil() {
		return p.AppID, true
	}
	return middleware.AppIDFrom(goCtx)
}

// ScopedAppID resolves the caller's tenant app for create/list operations,
// rejecting any request that explicitly targets a different app than the caller
// is bound to. A caller may therefore only ever create or enumerate resources
// within their own app, never one supplied in the request body/query.
func ScopedAppID(ctx forge.Context, requested string) (id.AppID, error) {
	var zero id.AppID
	appID, ok := CallerAppID(ctx)
	if !ok {
		return zero, forge.Unauthorized("authentication required")
	}
	if requested != "" {
		reqID, err := id.ParseAppID(requested)
		if err != nil {
			return zero, forge.BadRequest("invalid app_id")
		}
		if reqID.String() != appID.String() {
			return zero, forge.Forbidden("cannot act on another app's resources")
		}
	}
	return appID, nil
}

// AssertAppScope verifies that a loaded resource belongs to the caller's tenant
// app. It returns a 404 (not 403) on mismatch so the endpoint never discloses
// the existence of another app's resource. A missing caller app yields 401.
func AssertAppScope(ctx forge.Context, resourceAppID id.AppID) error {
	appID, ok := CallerAppID(ctx)
	if !ok {
		return forge.Unauthorized("authentication required")
	}
	if resourceAppID.String() != appID.String() {
		return forge.NotFound("not found")
	}
	return nil
}
