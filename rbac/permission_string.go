package rbac

// PermissionString renders an action on a resource as the single string a
// route declares to Forge.
//
// Permissions are a pair everywhere else in authsome: RequirePermission takes
// (action, resource), and rbac.Permission stores them in two columns. Forge's
// WithAllPermissions takes one string per permission, because that string ends
// up in the OpenAPI document as x-forge-authz and then as a constant in every
// generated client. Flattening it in one function keeps the two sides from
// drifting into "manage:user" here and "user:manage" three files away, which
// nothing would catch: both are valid strings, and a client generated from one
// silently fails to match a server checking the other.
//
// Action first, matching the argument order of RequirePermission, so a
// declaration reads in the same order as the middleware call it mirrors.
func PermissionString(action, resource string) string {
	return action + ":" + resource
}
