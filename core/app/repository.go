package app

import (
	"context"

	"github.com/rs/xid"
)

// Repository defines the app repository interface
type Repository interface {
	// App
	CreateApp(ctx context.Context, app *App) error
	FindAppByID(ctx context.Context, id xid.ID) (*App, error)
	FindAppBySlug(ctx context.Context, slug string) (*App, error)
	UpdateApp(ctx context.Context, app *App) error
	DeleteApp(ctx context.Context, id xid.ID) error
	ListApps(ctx context.Context, limit, offset int) ([]*App, error)
	CountApps(ctx context.Context) (int, error)

	// Member
	CreateMember(ctx context.Context, member *Member) error
	FindMemberByID(ctx context.Context, id xid.ID) (*Member, error)
	FindMember(ctx context.Context, appID, userID xid.ID) (*Member, error)
	ListMembers(ctx context.Context, appID xid.ID, limit, offset int) ([]*Member, error)
	CountMembers(ctx context.Context, appID xid.ID) (int, error)
	UpdateMember(ctx context.Context, member *Member) error
	DeleteMember(ctx context.Context, id xid.ID) error

	// Team
	CreateTeam(ctx context.Context, team *Team) error
	FindTeamByID(ctx context.Context, id xid.ID) (*Team, error)
	ListTeams(ctx context.Context, appID xid.ID, limit, offset int) ([]*Team, error)
	CountTeams(ctx context.Context, appID xid.ID) (int, error)
	UpdateTeam(ctx context.Context, team *Team) error
	DeleteTeam(ctx context.Context, id xid.ID) error

	// Team Member
	AddTeamMember(ctx context.Context, tm *TeamMember) error
	RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error
	ListTeamMembers(ctx context.Context, teamID xid.ID, limit, offset int) ([]*TeamMember, error)
	CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error)

	// Invitation
	CreateInvitation(ctx context.Context, inv *Invitation) error
}
