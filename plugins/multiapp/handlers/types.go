package handlers

import (
	coreapp "github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/responses"
)

// App handler request types
type CreateAppRequest struct {
	coreapp.CreateAppRequest
}

type GetAppRequest struct {
	ID string `path:"id" validate:"required"`
}

type UpdateAppRequest struct {
	ID string `path:"id" validate:"required"`
	coreapp.UpdateAppRequest
}

type DeleteAppRequest struct {
	ID string `path:"id" validate:"required"`
}

type ListAppsRequest struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

// Member handler request types
type AddMemberRequest struct {
	AppID  string `path:"appId" validate:"required"`
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"required"`
}

type RemoveMemberRequest struct {
	AppID    string `path:"appId" validate:"required"`
	MemberID string `path:"memberId" validate:"required"`
}

type ListMembersRequest struct {
	AppID  string `path:"appId" validate:"required"`
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}

type UpdateMemberRoleRequest struct {
	AppID    string `path:"appId" validate:"required"`
	MemberID string `path:"memberId" validate:"required"`
	Role     string `json:"role" validate:"required"`
}

type InviteMemberRequest struct {
	AppID string `path:"appId" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required"`
}

type UpdateMemberRequest struct {
	AppID    string `path:"appId" validate:"required"`
	MemberID string `path:"memberId" validate:"required"`
	Role     string `json:"role"`
}

type GetInvitationRequest struct {
	Token string `path:"token" validate:"required"`
}

type AcceptInvitationRequest struct {
	Token string `path:"token" validate:"required"`
}

type DeclineInvitationRequest struct {
	Token string `path:"token" validate:"required"`
}

// Team handler request types
type CreateTeamRequest struct {
	AppID       string `path:"appId" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type GetTeamRequest struct {
	AppID  string `path:"appId" validate:"required"`
	TeamID string `path:"teamId" validate:"required"`
}

type UpdateTeamRequest struct {
	AppID       string `path:"appId" validate:"required"`
	TeamID      string `path:"teamId" validate:"required"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DeleteTeamRequest struct {
	AppID  string `path:"appId" validate:"required"`
	TeamID string `path:"teamId" validate:"required"`
}

type ListTeamsRequest struct {
	AppID  string `path:"appId" validate:"required"`
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}

type AddTeamMemberRequest struct {
	AppID    string `path:"appId" validate:"required"`
	TeamID   string `path:"teamId" validate:"required"`
	MemberID string `json:"member_id" validate:"required"`
}

type RemoveTeamMemberRequest struct {
	AppID    string `path:"appId" validate:"required"`
	TeamID   string `path:"teamId" validate:"required"`
	MemberID string `path:"memberId" validate:"required"`
}

// Shared response types - use shared responses from core
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
