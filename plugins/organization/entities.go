package organization

import (
	"github.com/xraph/authsome/core/organization"
)

// Errors for organization plugin operations
var (
	ErrOrganizationNotFound    = organization.ErrOrganizationNotFound
	ErrMemberNotFound          = organization.ErrMemberNotFound
	ErrTeamNotFound            = organization.ErrTeamNotFound
	ErrInvalidRole             = organization.ErrInvalidRole
	ErrInvalidStatus           = organization.ErrInvalidStatus
	ErrOrganizationSlugExists  = organization.ErrOrganizationSlugExists
	ErrMemberAlreadyExists     = organization.ErrMemberAlreadyExists
	ErrNotOwner                = organization.ErrNotOwner
	ErrNotAdmin                = organization.ErrNotAdmin
	ErrCannotRemoveOwner       = organization.ErrCannotRemoveOwner
	ErrMaxOrganizationsReached = organization.ErrMaxOrganizationsReached
)

// Organization member roles
const (
	RoleOwner  = organization.RoleOwner
	RoleAdmin  = organization.RoleAdmin
	RoleMember = organization.RoleMember
)

// Organization member statuses
const (
	StatusActive    = organization.StatusActive
	StatusSuspended = organization.StatusSuspended
	StatusPending   = organization.StatusPending
)

// Organization invitation statuses
const (
	InvitationStatusPending   = organization.InvitationStatusPending
	InvitationStatusAccepted  = organization.InvitationStatusAccepted
	InvitationStatusExpired   = organization.InvitationStatusExpired
	InvitationStatusCancelled = organization.InvitationStatusCancelled
	InvitationStatusDeclined  = organization.InvitationStatusDeclined
)

// Note: Entity types (Organization, OrganizationMember, OrganizationTeam, OrganizationInvitation,
// OrganizationTeamMember) are defined in schema/organization.go and used throughout this plugin.
// This ensures a single source of truth for database models and eliminates duplication.
