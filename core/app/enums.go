package app

import "github.com/xraph/authsome/schema"

// ============================================================================
// Enum Type Re-exports from Schema Package
// ============================================================================

// MemberRole is a member role type.
type MemberRole = schema.MemberRole

const (
	MemberRoleOwner  = schema.MemberRoleOwner
	MemberRoleAdmin  = schema.MemberRoleAdmin
	MemberRoleMember = schema.MemberRoleMember
)

// MemberStatus is a member status type.
type MemberStatus = schema.MemberStatus

const (
	MemberStatusActive    = schema.MemberStatusActive
	MemberStatusSuspended = schema.MemberStatusSuspended
	MemberStatusPending   = schema.MemberStatusPending
)

// InvitationStatus is an invitation status type.
type InvitationStatus = schema.InvitationStatus

const (
	InvitationStatusPending   = schema.InvitationStatusPending
	InvitationStatusAccepted  = schema.InvitationStatusAccepted
	InvitationStatusExpired   = schema.InvitationStatusExpired
	InvitationStatusCancelled = schema.InvitationStatusCancelled
	InvitationStatusDeclined  = schema.InvitationStatusDeclined
)
