package api

import (
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// callerAppID returns the app the authenticated caller is bound to, as resolved
// onto the request context by the auth / publishable-key middleware. This is
// the tenant boundary used to scope app-owned admin resources (webhooks,
// environments, per-app config) to the caller.
func (a *API) callerAppID(ctx forge.Context) (id.AppID, bool) {
	return middleware.AppIDFrom(ctx.Context())
}

// scopedAppID resolves the caller's tenant app for create/list operations,
// rejecting any request that explicitly targets a different app than the caller
// is bound to. A caller may therefore only ever create or enumerate resources
// within their own app, never one supplied in the request body/query.
func (a *API) scopedAppID(ctx forge.Context, requested string) (id.AppID, error) {
	var zero id.AppID
	appID, ok := a.callerAppID(ctx)
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

// assertAppScope verifies that a loaded resource belongs to the caller's tenant
// app. It returns a 404 (not 403) on mismatch so the endpoint never discloses
// the existence of another app's resource. A missing caller app yields 401.
func (a *API) assertAppScope(ctx forge.Context, resourceAppID id.AppID) error {
	appID, ok := a.callerAppID(ctx)
	if !ok {
		return forge.Unauthorized("authentication required")
	}
	if resourceAppID.String() != appID.String() {
		return forge.NotFound("not found")
	}
	return nil
}
