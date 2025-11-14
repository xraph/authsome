package app

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// Focused Repository Interfaces (ISP Compliant)
// =============================================================================

// AppRepository handles app aggregate operations
type AppRepository interface {
	CreateApp(ctx context.Context, app *schema.App) error
	GetPlatformApp(ctx context.Context) (*schema.App, error)
	FindAppByID(ctx context.Context, id xid.ID) (*schema.App, error)
	FindAppBySlug(ctx context.Context, slug string) (*schema.App, error)
	UpdateApp(ctx context.Context, app *schema.App) error
	DeleteApp(ctx context.Context, id xid.ID) error
	ListApps(ctx context.Context, filter *ListAppsFilter) (*pagination.PageResponse[*schema.App], error)
	CountApps(ctx context.Context) (int, error)
}

// MemberRepository handles member aggregate operations
type MemberRepository interface {
	CreateMember(ctx context.Context, member *schema.Member) error
	FindMember(ctx context.Context, appID, userID xid.ID) (*schema.Member, error)
	FindMemberByID(ctx context.Context, id xid.ID) (*schema.Member, error)
	ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*schema.Member], error)
	ListMembersByUser(ctx context.Context, userID xid.ID) ([]*schema.Member, error)
	UpdateMember(ctx context.Context, member *schema.Member) error
	DeleteMember(ctx context.Context, id xid.ID) error
	CountMembers(ctx context.Context, appID xid.ID) (int, error)
}

// TeamRepository handles team aggregate operations (including team members)
type TeamRepository interface {
	CreateTeam(ctx context.Context, team *schema.Team) error
	FindTeamByID(ctx context.Context, id xid.ID) (*schema.Team, error)
	FindTeamByName(ctx context.Context, appID xid.ID, name string) (*schema.Team, error)
	ListTeams(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*schema.Team], error)
	UpdateTeam(ctx context.Context, team *schema.Team) error
	DeleteTeam(ctx context.Context, id xid.ID) error
	CountTeams(ctx context.Context, appID xid.ID) (int, error)

	// Team member operations (part of team aggregate)
	AddTeamMember(ctx context.Context, tm *schema.TeamMember) error
	RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error
	ListTeamMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*schema.TeamMember], error)
	CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error)
	IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error)
	ListMemberTeams(ctx context.Context, filter *ListMemberTeamsFilter) (*pagination.PageResponse[*schema.Team], error)
}

// InvitationRepository handles invitation aggregate operations
type InvitationRepository interface {
	CreateInvitation(ctx context.Context, inv *schema.Invitation) error
	FindInvitationByID(ctx context.Context, id xid.ID) (*schema.Invitation, error)
	FindInvitationByToken(ctx context.Context, token string) (*schema.Invitation, error)
	ListInvitations(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*schema.Invitation], error)
	UpdateInvitation(ctx context.Context, inv *schema.Invitation) error
	DeleteInvitation(ctx context.Context, id xid.ID) error
	DeleteExpiredInvitations(ctx context.Context) (int, error)
}
