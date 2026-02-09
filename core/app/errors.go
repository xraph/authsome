package app

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// APP-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeAppNotFound          = "APP_NOT_FOUND"
	CodeAppAlreadyExists     = "APP_ALREADY_EXISTS"
	CodeAppSlugExists        = "APP_SLUG_EXISTS"
	CodePlatformAppImmutable = "PLATFORM_APP_IMMUTABLE"
	CodeMemberNotFound       = "MEMBER_NOT_FOUND"
	CodeMemberAlreadyExists  = "MEMBER_ALREADY_EXISTS"
	CodeMaxMembersReached    = "MAX_MEMBERS_REACHED"
	CodeMaxTeamsReached      = "MAX_TEAMS_REACHED"
	CodeInvalidRole          = "INVALID_ROLE"
	CodeInvalidStatus        = "INVALID_STATUS"
	CodeCannotRemoveOwner    = "CANNOT_REMOVE_OWNER"
	CodeInvitationInvalid    = "INVITATION_INVALID_STATUS"
	CodeTeamMemberNotFound   = "TEAM_MEMBER_NOT_FOUND"
	CodeNotOwner             = "NOT_OWNER"
	CodeNotAdmin             = "NOT_ADMIN"
	CodeInvitationNotPending = "INVITATION_NOT_PENDING"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// App errors.
func AppNotFound() *errs.AuthsomeError {
	return errs.New(CodeAppNotFound, "App not found", http.StatusNotFound)
}

func AppAlreadyExists(identifier string) *errs.AuthsomeError {
	return errs.New(CodeAppAlreadyExists, "App already exists", http.StatusConflict).
		WithContext("identifier", identifier)
}

func AppSlugExists(slug string) *errs.AuthsomeError {
	return errs.New(CodeAppSlugExists, "App slug already exists", http.StatusConflict).
		WithContext("slug", slug)
}

func CannotDeletePlatformApp() *errs.AuthsomeError {
	return errs.New(CodePlatformAppImmutable, "Cannot delete platform app. Transfer platform status to another app first", http.StatusForbidden)
}

func PlatformAppAlreadyExists() *errs.AuthsomeError {
	return errs.New(CodePlatformAppImmutable, "A platform app already exists. Only one platform app is allowed", http.StatusConflict)
}

func PlatformAppImmutable() *errs.AuthsomeError {
	return errs.New(CodePlatformAppImmutable, "Platform app cannot be modified", http.StatusForbidden)
}

// Member errors.
func MemberNotFound() *errs.AuthsomeError {
	return errs.New(CodeMemberNotFound, "Member not found", http.StatusNotFound)
}

func MemberAlreadyExists(userID string) *errs.AuthsomeError {
	return errs.New(CodeMemberAlreadyExists, "Member already exists in this app", http.StatusConflict).
		WithContext("user_id", userID)
}

func MaxMembersReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxMembersReached, "Maximum members per app reached", http.StatusForbidden).
		WithContext("limit", limit)
}

func CannotRemoveOwner() *errs.AuthsomeError {
	return errs.New(CodeCannotRemoveOwner, "Cannot remove or demote app owner", http.StatusForbidden)
}

// Team errors (reuse from errs package where applicable).
func TeamNotFound() *errs.AuthsomeError {
	return errs.TeamNotFound()
}

func TeamAlreadyExists(name string) *errs.AuthsomeError {
	return errs.TeamAlreadyExists(name)
}

func MaxTeamsReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxTeamsReached, "Maximum teams per app reached", http.StatusForbidden).
		WithContext("limit", limit)
}

func TeamMemberNotFound() *errs.AuthsomeError {
	return errs.New(CodeTeamMemberNotFound, "Team member not found", http.StatusNotFound)
}

// Invitation errors (reuse from errs package where applicable).
func InvitationNotFound() *errs.AuthsomeError {
	return errs.InvitationNotFound()
}

func InvitationExpired() *errs.AuthsomeError {
	return errs.InvitationExpired()
}

func InvitationInvalidStatus(expected, actual string) *errs.AuthsomeError {
	return errs.New(CodeInvitationInvalid, "Invitation has invalid status for this operation", http.StatusConflict).
		WithContext("expected_status", expected).
		WithContext("actual_status", actual)
}

func InvitationNotPending() *errs.AuthsomeError {
	return errs.New(CodeInvitationNotPending, "Invitation is not in pending status", http.StatusConflict)
}

// Validation errors.
func InvalidRole(role string) *errs.AuthsomeError {
	return errs.New(CodeInvalidRole, "Invalid member role", http.StatusBadRequest).
		WithContext("role", role)
}

func InvalidStatus(status string) *errs.AuthsomeError {
	return errs.New(CodeInvalidStatus, "Invalid member status", http.StatusBadRequest).
		WithContext("status", status)
}

// Authorization errors.
func Unauthorized() *errs.AuthsomeError {
	return errs.Unauthorized()
}

func UnauthorizedAction(action string) *errs.AuthsomeError {
	return errs.New(errs.CodeUnauthorized, "Unauthorized to perform this action", http.StatusForbidden).
		WithContext("action", action)
}

func NotOwner() *errs.AuthsomeError {
	return errs.New(CodeNotOwner, "User is not the owner", http.StatusForbidden)
}

func NotAdmin() *errs.AuthsomeError {
	return errs.New(CodeNotAdmin, "User is not an admin or owner", http.StatusForbidden)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrAppNotFound          = &errs.AuthsomeError{Code: CodeAppNotFound}
	ErrSlugAlreadyExists    = &errs.AuthsomeError{Code: CodeAppSlugExists}
	ErrMemberNotFound       = &errs.AuthsomeError{Code: CodeMemberNotFound}
	ErrMemberAlreadyExists  = &errs.AuthsomeError{Code: CodeMemberAlreadyExists}
	ErrMaxMembersReached    = &errs.AuthsomeError{Code: CodeMaxMembersReached}
	ErrCannotRemoveOwner    = &errs.AuthsomeError{Code: CodeCannotRemoveOwner}
	ErrTeamNotFound         = &errs.AuthsomeError{Code: errs.CodeTeamNotFound}
	ErrMaxTeamsReached      = &errs.AuthsomeError{Code: CodeMaxTeamsReached}
	ErrTeamMemberNotFound   = &errs.AuthsomeError{Code: CodeTeamMemberNotFound}
	ErrInvitationNotFound   = &errs.AuthsomeError{Code: errs.CodeInvitationNotFound}
	ErrInvitationExpired    = &errs.AuthsomeError{Code: errs.CodeInvitationExpired}
	ErrInvitationInvalid    = &errs.AuthsomeError{Code: CodeInvitationInvalid}
	ErrInvitationNotPending = &errs.AuthsomeError{Code: CodeInvitationNotPending}
	ErrUnauthorized         = &errs.AuthsomeError{Code: errs.CodeUnauthorized}
	ErrNotOwner             = &errs.AuthsomeError{Code: CodeNotOwner}
	ErrNotAdmin             = &errs.AuthsomeError{Code: CodeNotAdmin}
	ErrInvalidRole          = &errs.AuthsomeError{Code: CodeInvalidRole}
	ErrInvalidStatus        = &errs.AuthsomeError{Code: CodeInvalidStatus}
)
