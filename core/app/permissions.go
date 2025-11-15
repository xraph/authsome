package app

import "github.com/xraph/authsome/core/rbac"

// RegisterAppPermissions registers app-level role definitions with the RBAC registry.
// This extends platform roles (owner, admin, member) with app-specific permissions.
func RegisterAppPermissions(registry rbac.RoleRegistryInterface) error {
	// Extend Owner Role with app-specific permissions
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:        rbac.RoleOwner,
		Description: rbac.RoleDescOwner,
		Priority:    rbac.RolePriorityOwner,
		IsPlatform:  rbac.RoleIsPlatformOwner,
		Permissions: []string{
			// Full wildcard access to all app resources
			"* on app:*",
			"* on member:app/*",
			"* on team:app/*",
			"* on invitation:app/*",

			// Special owner-only actions
			"transfer on app:*",
			"delete on app:*",
			"configure on app:*",
		},
	}); err != nil {
		return err
	}

	// Extend Admin Role with app-specific permissions
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:         rbac.RoleAdmin,
		Description:  rbac.RoleDescAdmin,
		Priority:     rbac.RolePriorityAdmin,
		IsPlatform:   rbac.RoleIsPlatformAdmin,
		InheritsFrom: rbac.RoleMember, // Inherits basic member permissions
		Permissions: []string{
			// Member management (except owner operations)
			"read on member:app/*",
			"create on member:app/*",
			"update on member:app/*",
			"delete on member:app/*",
			"update_role on member:app/*",

			// Team management
			"read on team:app/*",
			"create on team:app/*",
			"update on team:app/*",
			"delete on team:app/*",
			"add_member on team:app/*",
			"remove_member on team:app/*",

			// Invitation management
			"read on invitation:app/*",
			"create on invitation:app/*",
			"cancel on invitation:app/*",
			"resend on invitation:app/*",

			// App settings (limited)
			"read on app:*",
			"update on app:*",
		},
	}); err != nil {
		return err
	}

	// Extend Member Role with app-specific permissions
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:        rbac.RoleMember,
		Description: rbac.RoleDescMember,
		Priority:    rbac.RolePriorityMember,
		IsPlatform:  rbac.RoleIsPlatformMember,
		Permissions: []string{
			// Read-only access to app resources
			"read on app:*",
			"read on member:app/*",
			"read on team:app/*",

			// Can accept invitations sent to them
			"accept on invitation:app/*",
			"decline on invitation:app/*",

			// Can view their own membership
			"read on member:app/*",
		},
	}); err != nil {
		return err
	}

	return nil
}

// Action constants for app resources
const (
	// Generic actions
	ActionRead   = "read"
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"

	// Member-specific actions
	ActionUpdateRole = "update_role"
	ActionInvite     = "invite"

	// Team-specific actions
	ActionAddMember    = "add_member"
	ActionRemoveMember = "remove_member"

	// Invitation-specific actions
	ActionAccept  = "accept"
	ActionDecline = "decline"
	ActionCancel  = "cancel"
	ActionResend  = "resend"

	// App-specific actions
	ActionConfigure = "configure"
	ActionTransfer  = "transfer"
)

// Resource type constants for app resources
const (
	ResourceTypeApp        = "app"
	ResourceTypeMember     = "member"
	ResourceTypeTeam       = "team"
	ResourceTypeInvitation = "invitation"
)

// BuildAppResource builds a resource identifier for app-scoped resources
func BuildAppResource(resourceType, resourceID string) string {
	return resourceType + ":app/" + resourceID
}

// BuildAppWildcard builds a wildcard resource identifier for all app resources of a type
func BuildAppWildcard(resourceType string) string {
	return resourceType + ":app/*"
}
