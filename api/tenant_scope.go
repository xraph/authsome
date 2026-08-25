package api

import (
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
)

// The three helpers below delegate to the plugin package, which is where the
// tenancy boundary now lives so plugin route groups enforce the same rule.
// They stay as methods on *API because ~27 handlers call them that way, and
// because a single definition is the only thing that stops the core API and a
// plugin from disagreeing about what "another app's resource" returns.

// callerAppID returns the app the authenticated caller is bound to, as resolved
// onto the request context by the auth / publishable-key middleware. This is
// the tenant boundary used to scope app-owned admin resources (webhooks,
// environments, per-app config) to the caller.
func (a *API) callerAppID(ctx forge.Context) (id.AppID, bool) {
	return plugin.CallerAppID(ctx)
}

// scopedAppID resolves the caller's tenant app for create/list operations,
// rejecting any request that explicitly targets a different app than the caller
// is bound to. A caller may therefore only ever create or enumerate resources
// within their own app, never one supplied in the request body/query.
func (a *API) scopedAppID(ctx forge.Context, requested string) (id.AppID, error) {
	return plugin.ScopedAppID(ctx, requested)
}

// assertAppScope verifies that a loaded resource belongs to the caller's tenant
// app. It returns a 404 (not 403) on mismatch so the endpoint never discloses
// the existence of another app's resource. A missing caller app yields 401.
func (a *API) assertAppScope(ctx forge.Context, resourceAppID id.AppID) error {
	return plugin.AssertAppScope(ctx, resourceAppID)
}
