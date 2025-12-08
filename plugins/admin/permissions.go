package admin

// Admin Permissions
// These permission strings are registered with the RBAC system during plugin initialization
const (
	PermUserCreate      = "admin:user:create"
	PermUserRead        = "admin:user:read"
	PermUserUpdate      = "admin:user:update"
	PermUserDelete      = "admin:user:delete"
	PermUserBan         = "admin:user:ban"
	PermUserImpersonate = "admin:user:impersonate"
	PermSessionRead     = "admin:session:read"
	PermSessionRevoke   = "admin:session:revoke"
	PermRoleAssign      = "admin:role:assign"
	PermStatsRead       = "admin:stats:read"
	PermAuditRead       = "admin:audit:read"
)
