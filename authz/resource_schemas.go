// Package authz provides authorization primitives for authsome, including
// resource type schemas and a centralized Warden-backed authorization checker.
package authz

import "github.com/xraph/warden/resourcetype"

// ResourceSchema pairs a resource name with its Warden schema definition.
type ResourceSchema struct {
	Name        string
	Description string
	Relations   []resourcetype.RelationDef
	Permissions []resourcetype.PermissionDef
}

// DefaultSchemas returns the authsome resource type schemas with their
// relations and permissions for Warden's ReBAC model.
func DefaultSchemas() []ResourceSchema {
	return []ResourceSchema{
		{
			Name:        "user",
			Description: "User accounts",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner or app"},
				{Name: "update", Expression: "owner"},
				{Name: "delete", Expression: "owner"},
				{Name: "ban", Expression: "app"},
				{Name: "impersonate", Expression: "app"},
			},
		},
		{
			Name:        "session",
			Description: "Login sessions",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner or app"},
				{Name: "create", Expression: "owner or app"},
				{Name: "delete", Expression: "owner or app"},
				{Name: "revoke", Expression: "owner or app"},
			},
		},
		{
			Name:        "device",
			Description: "User devices",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner or app"},
				{Name: "create", Expression: "owner or app"},
				{Name: "update", Expression: "owner or app"},
				{Name: "delete", Expression: "owner or app"},
				{Name: "trust", Expression: "app"},
			},
		},
		{
			Name:        "app",
			Description: "Applications",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "admin", AllowedSubjects: []string{"user"}},
				{Name: "member", AllowedSubjects: []string{"user"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "member or admin or owner"},
				{Name: "update", Expression: "admin or owner"},
				{Name: "delete", Expression: "owner"},
				{Name: "manage_roles", Expression: "admin or owner"},
				{Name: "manage_envs", Expression: "admin or owner"},
			},
		},
		{
			Name:        "organization",
			Description: "Organizations",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "admin", AllowedSubjects: []string{"user"}},
				{Name: "member", AllowedSubjects: []string{"user"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "member or admin or owner"},
				{Name: "update", Expression: "admin or owner"},
				{Name: "delete", Expression: "owner"},
				{Name: "manage_members", Expression: "admin or owner"},
			},
		},
		{
			Name:        "role",
			Description: "Roles",
			Relations: []resourcetype.RelationDef{
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "app"},
				{Name: "create", Expression: "app"},
				{Name: "update", Expression: "app"},
				{Name: "delete", Expression: "app"},
				{Name: "assign", Expression: "app"},
				{Name: "unassign", Expression: "app"},
			},
		},
		{
			Name:        "permission",
			Description: "Permissions",
			Relations: []resourcetype.RelationDef{
				{Name: "role", AllowedSubjects: []string{"role"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "role"},
				{Name: "create", Expression: "role"},
				{Name: "delete", Expression: "role"},
			},
		},
		{
			Name:        "environment",
			Description: "Environments",
			Relations: []resourcetype.RelationDef{
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "app"},
				{Name: "create", Expression: "app"},
				{Name: "update", Expression: "app"},
				{Name: "delete", Expression: "app"},
				{Name: "clone", Expression: "app"},
			},
		},
		{
			Name:        "webhook",
			Description: "Webhooks",
			Relations: []resourcetype.RelationDef{
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "app"},
				{Name: "create", Expression: "app"},
				{Name: "update", Expression: "app"},
				{Name: "delete", Expression: "app"},
			},
		},
		{
			Name:        "apikey",
			Description: "API keys",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner or app"},
				{Name: "create", Expression: "owner or app"},
				{Name: "revoke", Expression: "owner or app"},
			},
		},
		{
			Name:        "passkey",
			Description: "Passkeys",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner"},
				{Name: "create", Expression: "owner"},
				{Name: "delete", Expression: "owner"},
			},
		},
		{
			Name:        "team",
			Description: "Teams",
			Relations: []resourcetype.RelationDef{
				{Name: "org", AllowedSubjects: []string{"organization"}},
				{Name: "admin", AllowedSubjects: []string{"user"}},
				{Name: "member", AllowedSubjects: []string{"user"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "member or admin or org"},
				{Name: "update", Expression: "admin or org"},
				{Name: "delete", Expression: "admin or org"},
			},
		},
		{
			Name:        "invitation",
			Description: "Invitations",
			Relations: []resourcetype.RelationDef{
				{Name: "org", AllowedSubjects: []string{"organization"}},
				{Name: "inviter", AllowedSubjects: []string{"user"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "inviter or org"},
				{Name: "create", Expression: "org"},
				{Name: "accept", Expression: "org"},
				{Name: "revoke", Expression: "inviter or org"},
			},
		},
		{
			Name:        "member",
			Description: "Organization members",
			Relations: []resourcetype.RelationDef{
				{Name: "org", AllowedSubjects: []string{"organization"}},
				{Name: "user", AllowedSubjects: []string{"user"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "user or org"},
				{Name: "update", Expression: "org"},
				{Name: "remove", Expression: "org"},
			},
		},
		// Plugin resource types
		{
			Name:        "scim_config",
			Description: "SCIM configurations",
			Relations: []resourcetype.RelationDef{
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "app"},
				{Name: "create", Expression: "app"},
				{Name: "update", Expression: "app"},
				{Name: "delete", Expression: "app"},
			},
		},
		{
			Name:        "subscription",
			Description: "Subscriptions",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner or app"},
				{Name: "create", Expression: "owner or app"},
				{Name: "update", Expression: "owner or app"},
				{Name: "cancel", Expression: "owner or app"},
			},
		},
		{
			Name:        "notification",
			Description: "Notifications",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner or app"},
				{Name: "create", Expression: "app"},
				{Name: "send", Expression: "app"},
			},
		},
		{
			Name:        "consent",
			Description: "User consent records",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner or app"},
				{Name: "grant", Expression: "owner"},
				{Name: "revoke", Expression: "owner"},
			},
		},
		{
			Name:        "mfa",
			Description: "MFA configuration",
			Relations: []resourcetype.RelationDef{
				{Name: "owner", AllowedSubjects: []string{"user"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "owner"},
				{Name: "enable", Expression: "owner"},
				{Name: "disable", Expression: "owner"},
			},
		},
		{
			Name:        "social_provider",
			Description: "Social login providers",
			Relations: []resourcetype.RelationDef{
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "app"},
				{Name: "create", Expression: "app"},
				{Name: "update", Expression: "app"},
				{Name: "delete", Expression: "app"},
			},
		},
		{
			Name:        "sso_config",
			Description: "SSO configurations",
			Relations: []resourcetype.RelationDef{
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "app"},
				{Name: "create", Expression: "app"},
				{Name: "update", Expression: "app"},
				{Name: "delete", Expression: "app"},
			},
		},
		{
			Name:        "settings",
			Description: "App settings",
			Relations: []resourcetype.RelationDef{
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "app"},
				{Name: "update", Expression: "app"},
			},
		},
		{
			Name:        "security_event",
			Description: "Security audit events",
			Relations: []resourcetype.RelationDef{
				{Name: "app", AllowedSubjects: []string{"app"}},
			},
			Permissions: []resourcetype.PermissionDef{
				{Name: "read", Expression: "app"},
			},
		},
	}
}
