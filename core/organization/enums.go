package organization

import "github.com/xraph/authsome/schema"

// Re-export organization member roles from schema
const (
	RoleOwner  = schema.OrgMemberRoleOwner
	RoleAdmin  = schema.OrgMemberRoleAdmin
	RoleMember = schema.OrgMemberRoleMember
)

// Re-export organization member statuses from schema
const (
	StatusActive    = schema.OrgMemberStatusActive
	StatusSuspended = schema.OrgMemberStatusSuspended
	StatusPending   = schema.OrgMemberStatusPending
)

// Re-export organization invitation statuses from schema
const (
	InvitationStatusPending   = schema.OrgInvitationStatusPending
	InvitationStatusAccepted  = schema.OrgInvitationStatusAccepted
	InvitationStatusExpired   = schema.OrgInvitationStatusExpired
	InvitationStatusCancelled = schema.OrgInvitationStatusCancelled
	InvitationStatusDeclined  = schema.OrgInvitationStatusDeclined
)

// ValidRoles returns the list of valid member roles
func ValidRoles() []string {
	return schema.ValidOrgMemberRoles()
}

// ValidStatuses returns the list of valid member statuses
func ValidStatuses() []string {
	return schema.ValidOrgMemberStatuses()
}

// ValidInvitationStatuses returns the list of valid invitation statuses
func ValidInvitationStatuses() []string {
	return schema.ValidOrgInvitationStatuses()
}

// IsValidRole checks if a role is valid
func IsValidRole(role string) bool {
	return schema.IsValidOrgMemberRole(role)
}

// IsValidStatus checks if a status is valid
func IsValidStatus(status string) bool {
	return schema.IsValidOrgMemberStatus(status)
}

// IsValidInvitationStatus checks if an invitation status is valid
func IsValidInvitationStatus(status string) bool {
	return schema.IsValidOrgInvitationStatus(status)
}
