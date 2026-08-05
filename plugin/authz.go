// Package plugin: authz.go — shared route-group guards for plugins.
//
// Plugins register their own routes (see RouteProvider), which means each
// plugin is responsible for gating its own admin and account surfaces. The
// helpers here package the same chain the core API uses for /v1/admin so a
// plugin group can't accidentally ship unauthenticated.
package plugin

import (
	"github.com/xraph/forge"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/middleware"
)

// SessionGuard returns the middleware chain that requires a resolved session
// on every route in a group. Use it for plugin surfaces that act on behalf of
// the calling user (creating a key for yourself, reading your own billing).
//
// Pair it with forge.WithGroupAuth("session") so the OpenAPI document
// advertises the requirement alongside the enforcement:
//
//	g := router.Group("/v1/things",
//	    forge.WithGroupAuth("session"),
//	    forge.WithGroupMiddleware(plugin.SessionGuard(engine)...),
//	)
//
// Returns nil when engine is nil or exposes no auth registry (minimal test
// wiring), so tests that construct a bare plugin still register routes.
func SessionGuard(engine Engine) []forge.Middleware {
	if engine == nil {
		return nil
	}
	reg := engine.AuthRegistry()
	if reg == nil {
		return nil
	}
	return []forge.Middleware{authprovider.RegistryMiddleware(reg, "session")}
}

// PermissionGuard returns just the RBAC permission check, without the session
// middleware. Use it on individual routes inside a group that already carries
// SessionGuard — typically to hold mutating routes to a higher bar than the
// reads they sit beside.
//
// Returns nil when the engine doesn't implement PermissionChecker, so an
// engine without RBAC falls back to the group's session requirement rather
// than failing closed on every request.
func PermissionGuard(engine Engine, action, resource string) []forge.Middleware {
	if engine == nil {
		return nil
	}
	pc, ok := engine.(PermissionChecker)
	if !ok {
		return nil
	}
	return []forge.Middleware{middleware.RequirePermission(pc, action, resource)}
}

// PermissionRouteOptions is PermissionGuard adapted to per-route options, for
// use alongside forge.WithSummary and friends in a route registration.
func PermissionRouteOptions(engine Engine, action, resource string) []forge.RouteOption {
	guards := PermissionGuard(engine, action, resource)
	opts := make([]forge.RouteOption, 0, len(guards))
	for _, mw := range guards {
		opts = append(opts, forge.WithMiddleware(mw))
	}
	return opts
}

// AdminGuard returns SessionGuard plus an RBAC permission check, mirroring the
// core API's /v1/admin chain (middleware.RequireAuth + RequirePermission). Use
// it for plugin admin groups that manage tenant-wide configuration —
// OAuth2 clients, SSO connections, provider credentials, billing plans.
//
// The permission check is appended only when the engine implements
// PermissionChecker. An engine without RBAC still gets session enforcement,
// which is the property that matters most: the group is never anonymous.
func AdminGuard(engine Engine, action, resource string) []forge.Middleware {
	return append(SessionGuard(engine), PermissionGuard(engine, action, resource)...)
}
