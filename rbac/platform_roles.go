package rbac

// Platform role slugs. These are ONLY created under the platform app,
// never for regular tenant apps. They grant cross-app access.
const (
	PlatformUserSlug  = "platform_user"
	PlatformAdminSlug = "platform_admin"
	PlatformOwnerSlug = "platform_owner"
)

// IsPlatformRole returns true if the slug is a platform-scoped role.
func IsPlatformRole(slug string) bool {
	switch slug {
	case PlatformUserSlug, PlatformAdminSlug, PlatformOwnerSlug:
		return true
	}
	return false
}
