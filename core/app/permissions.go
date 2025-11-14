package app

import "github.com/xraph/authsome/core/rbac"

// RegisterAppPermissions registers app-level role definitions with the RBAC registry.
// This should be called during application bootstrap to set up default app roles.
func RegisterAppPermissions(registry rbac.RoleRegistryInterface) error {
	// App Owner Role - Full control over the app
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:        "app_owner",
		Description: "App owner with full control over all app resources",
		Priority:    90,
		IsPlatform:  false,
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

	// App Admin Role - Can manage members, teams, and invitations
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:         "app_admin",
		Description:  "App administrator with management capabilities",
		Priority:     70,
		IsPlatform:   false,
		InheritsFrom: "app_member", // Inherits basic member permissions
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

	// App Member Role - Basic read access
	if err := registry.RegisterRole(&rbac.RoleDefinition{
		Name:        "app_member",
		Description: "Regular app member with read-only access",
		Priority:    50,
		IsPlatform:  false,
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
