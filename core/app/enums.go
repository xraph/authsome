package app

import "github.com/xraph/authsome/schema"

// ============================================================================
// Enum Type Re-exports from Schema Package
// ============================================================================

// Member Role Types
type MemberRole = schema.MemberRole

const (
	MemberRoleOwner  = schema.MemberRoleOwner
	MemberRoleAdmin  = schema.MemberRoleAdmin
	MemberRoleMember = schema.MemberRoleMember
)

// Member Status Types
type MemberStatus = schema.MemberStatus

const (
	MemberStatusActive    = schema.MemberStatusActive
	MemberStatusSuspended = schema.MemberStatusSuspended
	MemberStatusPending   = schema.MemberStatusPending
)

// Invitation Status Types
type InvitationStatus = schema.InvitationStatus

const (
	InvitationStatusPending   = schema.InvitationStatusPending
	InvitationStatusAccepted  = schema.InvitationStatusAccepted
	InvitationStatusExpired   = schema.InvitationStatusExpired
	InvitationStatusCancelled = schema.InvitationStatusCancelled
	InvitationStatusDeclined  = schema.InvitationStatusDeclined
)
