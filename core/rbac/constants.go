package rbac

// Role name constants
// These define the standard role names used across the platform
const (
	RoleSuperAdmin = "superadmin"
	RoleOwner      = "owner"
	RoleAdmin      = "admin"
	RoleMember     = "member"
)

// Role priority constants
// Higher priority roles override lower priority roles in the hierarchy
const (
	RolePrioritySuperAdmin = 100
	RolePriorityOwner      = 80
	RolePriorityAdmin      = 60
	RolePriorityMember     = 40
)

// Role description constants
const (
	RoleDescSuperAdmin = "System Superadministrator (Platform Owner)"
	RoleDescOwner      = "Organization Owner"
	RoleDescAdmin      = "Organization Administrator"
	RoleDescMember     = "Regular User"
)

// Role platform flag constants
// Platform roles can only be assigned in the platform app
const (
	RoleIsPlatformSuperAdmin = true
	RoleIsPlatformOwner      = false
	RoleIsPlatformAdmin      = false
	RoleIsPlatformMember     = false
)
