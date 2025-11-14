package schema

// Organization Member Roles
const (
	OrgMemberRoleOwner  = "owner"
	OrgMemberRoleAdmin  = "admin"
	OrgMemberRoleMember = "member"
)

// Organization Member Statuses
const (
	OrgMemberStatusActive    = "active"
	OrgMemberStatusSuspended = "suspended"
	OrgMemberStatusPending   = "pending"
)

// Organization Invitation Statuses
const (
	OrgInvitationStatusPending   = "pending"
	OrgInvitationStatusAccepted  = "accepted"
	OrgInvitationStatusExpired   = "expired"
	OrgInvitationStatusCancelled = "cancelled"
	OrgInvitationStatusDeclined  = "declined"
)

// ValidOrgMemberRoles returns the list of valid member roles
func ValidOrgMemberRoles() []string {
	return []string{OrgMemberRoleOwner, OrgMemberRoleAdmin, OrgMemberRoleMember}
}

// ValidOrgMemberStatuses returns the list of valid member statuses
func ValidOrgMemberStatuses() []string {
	return []string{OrgMemberStatusActive, OrgMemberStatusSuspended, OrgMemberStatusPending}
}

// ValidOrgInvitationStatuses returns the list of valid invitation statuses
func ValidOrgInvitationStatuses() []string {
	return []string{
		OrgInvitationStatusPending,
		OrgInvitationStatusAccepted,
		OrgInvitationStatusExpired,
		OrgInvitationStatusCancelled,
		OrgInvitationStatusDeclined,
	}
}

// IsValidOrgMemberRole checks if a role is valid
func IsValidOrgMemberRole(role string) bool {
	switch role {
	case OrgMemberRoleOwner, OrgMemberRoleAdmin, OrgMemberRoleMember:
		return true
	default:
		return false
	}
}

// IsValidOrgMemberStatus checks if a status is valid
func IsValidOrgMemberStatus(status string) bool {
	switch status {
	case OrgMemberStatusActive, OrgMemberStatusSuspended, OrgMemberStatusPending:
		return true
	default:
		return false
	}
}

// IsValidOrgInvitationStatus checks if an invitation status is valid
func IsValidOrgInvitationStatus(status string) bool {
	switch status {
	case OrgInvitationStatusPending, OrgInvitationStatusAccepted,
		OrgInvitationStatusExpired, OrgInvitationStatusCancelled, OrgInvitationStatusDeclined:
		return true
	default:
		return false
	}
}
