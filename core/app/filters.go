package app

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// ListAppsFilter represents filter parameters for listing apps.
type ListAppsFilter struct {
	pagination.PaginationParams

	IsPlatform *bool `json:"isPlatform,omitempty" query:"is_platform"`
}

// ListMembersFilter represents filter parameters for listing members.
type ListMembersFilter struct {
	pagination.PaginationParams

	AppID  xid.ID               `json:"appId"            query:"app_id"`
	Role   *schema.MemberRole   `json:"role,omitempty"   query:"role"`
	Status *schema.MemberStatus `json:"status,omitempty" query:"status"`
}

// ListTeamsFilter represents filter parameters for listing teams.
type ListTeamsFilter struct {
	pagination.PaginationParams

	AppID xid.ID `json:"appId" query:"app_id"`
}

// ListTeamMembersFilter represents filter parameters for listing team members.
type ListTeamMembersFilter struct {
	pagination.PaginationParams

	TeamID xid.ID `json:"teamId" query:"team_id"`
}

// ListInvitationsFilter represents filter parameters for listing invitations.
type ListInvitationsFilter struct {
	pagination.PaginationParams

	AppID  xid.ID                   `json:"appId"            query:"app_id"`
	Status *schema.InvitationStatus `json:"status,omitempty" query:"status"`
	Email  *string                  `json:"email,omitempty"  query:"email"`
}

// ListMemberTeamsFilter represents filter parameters for listing teams a member belongs to.
type ListMemberTeamsFilter struct {
	pagination.PaginationParams

	MemberID xid.ID `json:"memberId" query:"member_id"`
}
