package organization

import (
	"fmt"
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// ORGANIZATION-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeOrganizationNotFound         = "ORGANIZATION_NOT_FOUND"
	CodeOrganizationSlugExists       = "ORGANIZATION_SLUG_EXISTS"
	CodeOrganizationAlreadyExists    = "ORGANIZATION_ALREADY_EXISTS"
	CodeMemberNotFound               = "ORGANIZATION_MEMBER_NOT_FOUND"
	CodeMemberAlreadyExists          = "ORGANIZATION_MEMBER_ALREADY_EXISTS"
	CodeMaxMembersReached            = "MAX_ORGANIZATION_MEMBERS_REACHED"
	CodeMaxOrganizationsReached      = "MAX_ORGANIZATIONS_REACHED"
	CodeTeamNotFound                 = "ORGANIZATION_TEAM_NOT_FOUND"
	CodeTeamAlreadyExists            = "ORGANIZATION_TEAM_ALREADY_EXISTS"
	CodeMaxTeamsReached              = "MAX_ORGANIZATION_TEAMS_REACHED"
	CodeTeamMemberNotFound           = "ORGANIZATION_TEAM_MEMBER_NOT_FOUND"
	CodeInvitationNotFound           = "ORGANIZATION_INVITATION_NOT_FOUND"
	CodeInvitationExpired            = "ORGANIZATION_INVITATION_EXPIRED"
	CodeInvitationInvalid            = "ORGANIZATION_INVITATION_INVALID_STATUS"
	CodeInvitationNotPending         = "ORGANIZATION_INVITATION_NOT_PENDING"
	CodeInvalidRole                  = "INVALID_ORGANIZATION_ROLE"
	CodeInvalidStatus                = "INVALID_ORGANIZATION_STATUS"
	CodeCannotRemoveOwner            = "CANNOT_REMOVE_ORGANIZATION_OWNER"
	CodeNotOwner                     = "NOT_ORGANIZATION_OWNER"
	CodeNotAdmin                     = "NOT_ORGANIZATION_ADMIN"
	CodeOrganizationCreationDisabled = "ORGANIZATION_CREATION_DISABLED"
	CodePermissionDenied             = "ORGANIZATION_PERMISSION_DENIED"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// OrganizationNotFound returns an error when an organization is not found.
func OrganizationNotFound() *errs.AuthsomeError {
	return errs.New(CodeOrganizationNotFound, "Organization not found", http.StatusNotFound)
}

func OrganizationSlugExists(slug string) *errs.AuthsomeError {
	return errs.New(CodeOrganizationSlugExists, "Organization slug already exists", http.StatusConflict).
		WithContext("slug", slug)
}

func OrganizationAlreadyExists(identifier string) *errs.AuthsomeError {
	return errs.New(CodeOrganizationAlreadyExists, "Organization already exists", http.StatusConflict).
		WithContext("identifier", identifier)
}

func OrganizationCreationDisabled() *errs.AuthsomeError {
	return errs.New(CodeOrganizationCreationDisabled, "Organization creation is disabled", http.StatusForbidden)
}

// MemberNotFound returns an error when a member is not found.
func MemberNotFound() *errs.AuthsomeError {
	return errs.New(CodeMemberNotFound, "Organization member not found", http.StatusNotFound)
}

func MemberAlreadyExists(userID string) *errs.AuthsomeError {
	return errs.New(CodeMemberAlreadyExists, "User is already a member of this organization", http.StatusConflict).
		WithContext("user_id", userID)
}

func MaxMembersReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxMembersReached, "Maximum members per organization reached", http.StatusForbidden).
		WithContext("limit", limit)
}

func MaxOrganizationsReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxOrganizationsReached, "Maximum organizations per user reached", http.StatusForbidden).
		WithContext("limit", limit)
}

func CannotRemoveOwner() *errs.AuthsomeError {
	return errs.New(CodeCannotRemoveOwner, "Cannot remove or demote organization owner", http.StatusForbidden)
}

// TeamNotFound returns an error when a team is not found.
func TeamNotFound() *errs.AuthsomeError {
	return errs.New(CodeTeamNotFound, "Organization team not found", http.StatusNotFound)
}

func TeamAlreadyExists(name string) *errs.AuthsomeError {
	return errs.New(CodeTeamAlreadyExists, "Team already exists in organization", http.StatusConflict).
		WithContext("name", name)
}

func MaxTeamsReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxTeamsReached, "Maximum teams per organization reached", http.StatusForbidden).
		WithContext("limit", limit)
}

func TeamMemberNotFound() *errs.AuthsomeError {
	return errs.New(CodeTeamMemberNotFound, "Team member not found", http.StatusNotFound)
}

// InvitationNotFound returns an error when an invitation is not found.
func InvitationNotFound() *errs.AuthsomeError {
	return errs.New(CodeInvitationNotFound, "Organization invitation not found", http.StatusNotFound)
}

func InvitationExpired() *errs.AuthsomeError {
	return errs.New(CodeInvitationExpired, "Organization invitation has expired", http.StatusGone)
}

func InvitationInvalidStatus(expected, actual string) *errs.AuthsomeError {
	return errs.New(CodeInvitationInvalid, "Invitation has invalid status for this operation", http.StatusConflict).
		WithContext("expected_status", expected).
		WithContext("actual_status", actual)
}

func InvitationNotPending() *errs.AuthsomeError {
	return errs.New(CodeInvitationNotPending, "Invitation is not in pending status", http.StatusConflict)
}

// InvalidRole returns an error when an invalid role is provided.
func InvalidRole(role string) *errs.AuthsomeError {
	return errs.New(CodeInvalidRole, "Invalid organization member role", http.StatusBadRequest).
		WithContext("role", role).
		WithContext("valid_roles", ValidRoles())
}

// InvalidRoleWithHint creates an invalid role error with a hint message.
func InvalidRoleWithHint(role, hint string) *errs.AuthsomeError {
	return errs.New(CodeInvalidRole, "Invalid organization member role", http.StatusBadRequest).
		WithContext("role", role).
		WithContext("valid_roles", ValidRoles()).
		WithContext("hint", hint)
}

func InvalidStatus(status string) *errs.AuthsomeError {
	return errs.New(CodeInvalidStatus, "Invalid organization member status", http.StatusBadRequest).
		WithContext("status", status).
		WithContext("valid_statuses", ValidStatuses())
}

// Unauthorized returns an error when the user is unauthorized.
func Unauthorized() *errs.AuthsomeError {
	return errs.Unauthorized()
}

func UnauthorizedAction(action string) *errs.AuthsomeError {
	return errs.New(errs.CodeUnauthorized, "Unauthorized to perform this action on organization", http.StatusForbidden).
		WithContext("action", action)
}

func NotOwner() *errs.AuthsomeError {
	return errs.New(CodeNotOwner, "User is not the organization owner", http.StatusForbidden)
}

func NotAdmin() *errs.AuthsomeError {
	return errs.New(CodeNotAdmin, "User is not an organization admin or owner", http.StatusForbidden)
}

// PermissionDenied creates a permission denied error for RBAC checks.
func PermissionDenied(action, resource string) *errs.AuthsomeError {
	return errs.New(CodePermissionDenied, fmt.Sprintf("Permission denied: %s on %s", action, resource), http.StatusForbidden).
		WithContext("action", action).
		WithContext("resource", resource)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrOrganizationNotFound         = &errs.AuthsomeError{Code: CodeOrganizationNotFound}
	ErrOrganizationSlugExists       = &errs.AuthsomeError{Code: CodeOrganizationSlugExists}
	ErrOrganizationAlreadyExists    = &errs.AuthsomeError{Code: CodeOrganizationAlreadyExists}
	ErrOrganizationCreationDisabled = &errs.AuthsomeError{Code: CodeOrganizationCreationDisabled}
	ErrMemberNotFound               = &errs.AuthsomeError{Code: CodeMemberNotFound}
	ErrMemberAlreadyExists          = &errs.AuthsomeError{Code: CodeMemberAlreadyExists}
	ErrMaxMembersReached            = &errs.AuthsomeError{Code: CodeMaxMembersReached}
	ErrMaxOrganizationsReached      = &errs.AuthsomeError{Code: CodeMaxOrganizationsReached}
	ErrCannotRemoveOwner            = &errs.AuthsomeError{Code: CodeCannotRemoveOwner}
	ErrTeamNotFound                 = &errs.AuthsomeError{Code: CodeTeamNotFound}
	ErrTeamAlreadyExists            = &errs.AuthsomeError{Code: CodeTeamAlreadyExists}
	ErrMaxTeamsReached              = &errs.AuthsomeError{Code: CodeMaxTeamsReached}
	ErrTeamMemberNotFound           = &errs.AuthsomeError{Code: CodeTeamMemberNotFound}
	ErrInvitationNotFound           = &errs.AuthsomeError{Code: CodeInvitationNotFound}
	ErrInvitationExpired            = &errs.AuthsomeError{Code: CodeInvitationExpired}
	ErrInvitationInvalid            = &errs.AuthsomeError{Code: CodeInvitationInvalid}
	ErrInvitationNotPending         = &errs.AuthsomeError{Code: CodeInvitationNotPending}
	ErrUnauthorized                 = &errs.AuthsomeError{Code: errs.CodeUnauthorized}
	ErrNotOwner                     = &errs.AuthsomeError{Code: CodeNotOwner}
	ErrNotAdmin                     = &errs.AuthsomeError{Code: CodeNotAdmin}
	ErrInvalidRole                  = &errs.AuthsomeError{Code: CodeInvalidRole}
	ErrInvalidStatus                = &errs.AuthsomeError{Code: CodeInvalidStatus}
	ErrPermissionDenied             = &errs.AuthsomeError{Code: CodePermissionDenied}
)
