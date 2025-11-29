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
	ErrPermissionDenied        = organization.ErrPermissionDenied
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

// =============================================================================
// Error Constructor Functions
// =============================================================================

// OrganizationNotFound returns an error indicating organization was not found
var OrganizationNotFound = organization.OrganizationNotFound

// OrganizationSlugExists returns an error indicating organization slug already exists
var OrganizationSlugExists = organization.OrganizationSlugExists

// OrganizationAlreadyExists returns an error indicating organization already exists
var OrganizationAlreadyExists = organization.OrganizationAlreadyExists

// OrganizationCreationDisabled returns an error indicating organization creation is disabled
var OrganizationCreationDisabled = organization.OrganizationCreationDisabled

// MemberNotFound returns an error indicating member was not found
var MemberNotFound = organization.MemberNotFound

// MemberAlreadyExists returns an error indicating member already exists
var MemberAlreadyExists = organization.MemberAlreadyExists

// MaxMembersReached returns an error indicating maximum members limit reached
var MaxMembersReached = organization.MaxMembersReached

// MaxOrganizationsReached returns an error indicating maximum organizations limit reached
var MaxOrganizationsReached = organization.MaxOrganizationsReached

// CannotRemoveOwner returns an error indicating owner cannot be removed
var CannotRemoveOwner = organization.CannotRemoveOwner

// TeamNotFound returns an error indicating team was not found
var TeamNotFound = organization.TeamNotFound

// TeamAlreadyExists returns an error indicating team already exists
var TeamAlreadyExists = organization.TeamAlreadyExists

// MaxTeamsReached returns an error indicating maximum teams limit reached
var MaxTeamsReached = organization.MaxTeamsReached

// TeamMemberNotFound returns an error indicating team member was not found
var TeamMemberNotFound = organization.TeamMemberNotFound

// InvitationNotFound returns an error indicating invitation was not found
var InvitationNotFound = organization.InvitationNotFound

// InvitationExpired returns an error indicating invitation has expired
var InvitationExpired = organization.InvitationExpired

// InvitationInvalidStatus returns an error indicating invalid invitation status
var InvitationInvalidStatus = organization.InvitationInvalidStatus

// InvitationNotPending returns an error indicating invitation is not pending
var InvitationNotPending = organization.InvitationNotPending

// InvalidRole returns an error indicating invalid role
var InvalidRole = organization.InvalidRole

// InvalidStatus returns an error indicating invalid status
var InvalidStatus = organization.InvalidStatus

// Unauthorized returns an unauthorized error
var Unauthorized = organization.Unauthorized

// UnauthorizedAction returns an error for unauthorized action
var UnauthorizedAction = organization.UnauthorizedAction

// NotOwner returns an error indicating user is not owner
var NotOwner = organization.NotOwner

// NotAdmin returns an error indicating user is not admin
var NotAdmin = organization.NotAdmin

// PermissionDenied returns an error indicating permission denied
var PermissionDenied = organization.PermissionDenied

// Note: Entity types (Organization, OrganizationMember, OrganizationTeam, OrganizationInvitation,
// OrganizationTeamMember) are defined in schema/organization.go and used throughout this plugin.
// This ensures a single source of truth for database models and eliminates duplication.
