package organization

import (
	"errors"
)

// Errors for organization plugin operations
var (
	ErrOrganizationNotFound       = errors.New("organization not found")
	ErrOrganizationMemberNotFound = errors.New("organization member not found")
	ErrOrganizationTeamNotFound   = errors.New("organization team not found")
	ErrInvalidRole                = errors.New("invalid role")
	ErrInvalidStatus              = errors.New("invalid status")
	ErrSlugAlreadyExists          = errors.New("organization slug already exists")
	ErrMemberAlreadyExists        = errors.New("member already exists in this organization")
	ErrNotOrganizationOwner       = errors.New("only organization owner can perform this action")
	ErrCannotRemoveOwner          = errors.New("cannot remove organization owner")
	ErrMaxOrganizationsReached    = errors.New("maximum organizations limit reached")
)

// Organization member roles
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Organization member statuses
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusPending   = "pending"
)

// Organization invitation statuses
const (
	InvitationStatusPending   = "pending"
	InvitationStatusAccepted  = "accepted"
	InvitationStatusExpired   = "expired"
	InvitationStatusCancelled = "cancelled"
	InvitationStatusDeclined  = "declined"
)

// Team member roles
const (
	TeamRoleLead   = "lead"
	TeamRoleMember = "member"
)

// Note: Entity types (Organization, OrganizationMember, OrganizationTeam, OrganizationInvitation,
// OrganizationTeamMember) are defined in schema/organization.go and used throughout this plugin.
// This ensures a single source of truth for database models and eliminates duplication.
