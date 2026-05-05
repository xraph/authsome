package rbac

// Platform role slugs. These are ONLY created under the platform app,
// never for regular tenant apps. They grant cross-app access.
//
// Slugs are kebab-case so they pass the warden DSL slug convention regex
// (^[a-z][a-z0-9-]{0,62}$). Existing deployments that have the old
// underscore-separated rows in their roles table will need to either
// re-bootstrap or migrate the rows manually — the bootstrap is
// idempotent at the slug level, so old rows aren't touched and the new
// rows live alongside them until cleaned up.
const (
	PlatformUserSlug  = "platform-user"
	PlatformAdminSlug = "platform-admin"
	PlatformOwnerSlug = "platform-owner"

	// SuperAdminSlug documents the intended slug for a future super-admin
	// role that spans all platform capabilities. It is declared here to
	// prevent accidental reuse of the name in unrelated roles and to serve
	// as a single source of truth when the role is eventually introduced.
	SuperAdminSlug = "super-admin"
)

// App-scoped role slugs. Created for every app during bootstrap.
const (
	AppOwnerSlug = "owner"
	AppAdminSlug = "admin"
	AppUserSlug  = "user"
)

// IsPlatformRole returns true if the slug is a platform-scoped role.
func IsPlatformRole(slug string) bool {
	switch slug {
	case PlatformUserSlug, PlatformAdminSlug, PlatformOwnerSlug, SuperAdminSlug:
		return true
	}
	return false
}
