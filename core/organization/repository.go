package organization

import (
	"context"
	"github.com/rs/xid"
)

// Repository defines the organization repository interface
type Repository interface {
	// Organization
	CreateOrganization(ctx context.Context, org *Organization) error
	FindOrganizationByID(ctx context.Context, id xid.ID) (*Organization, error)
	FindOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, id xid.ID) error
	ListOrganizations(ctx context.Context, limit, offset int) ([]*Organization, error)
	CountOrganizations(ctx context.Context) (int, error)

	// Member
	CreateMember(ctx context.Context, member *Member) error
	FindMemberByID(ctx context.Context, id xid.ID) (*Member, error)
	FindMember(ctx context.Context, orgID, userID xid.ID) (*Member, error)
	ListMembers(ctx context.Context, orgID xid.ID, limit, offset int) ([]*Member, error)
	CountMembers(ctx context.Context, orgID xid.ID) (int, error)
	UpdateMember(ctx context.Context, member *Member) error
	DeleteMember(ctx context.Context, id xid.ID) error

	// Team
	CreateTeam(ctx context.Context, team *Team) error
	FindTeamByID(ctx context.Context, id xid.ID) (*Team, error)
	ListTeams(ctx context.Context, orgID xid.ID, limit, offset int) ([]*Team, error)
	CountTeams(ctx context.Context, orgID xid.ID) (int, error)
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
