package organization

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for organization operations.
type Store interface {
	// Organization CRUD
	CreateOrganization(ctx context.Context, o *Organization) error
	GetOrganization(ctx context.Context, orgID id.OrgID) (*Organization, error)
	GetOrganizationBySlug(ctx context.Context, appID id.AppID, slug string) (*Organization, error)
	UpdateOrganization(ctx context.Context, o *Organization) error
	DeleteOrganization(ctx context.Context, orgID id.OrgID) error
	ListOrganizations(ctx context.Context, appID id.AppID) ([]*Organization, error)
	ListUserOrganizations(ctx context.Context, userID id.UserID) ([]*Organization, error)

	// Membership
	CreateMember(ctx context.Context, m *Member) error
	GetMember(ctx context.Context, memberID id.MemberID) (*Member, error)
	GetMemberByUserAndOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) (*Member, error)
	UpdateMember(ctx context.Context, m *Member) error
	DeleteMember(ctx context.Context, memberID id.MemberID) error
	ListMembers(ctx context.Context, orgID id.OrgID) ([]*Member, error)

	// Invitations
	CreateInvitation(ctx context.Context, inv *Invitation) error
	GetInvitation(ctx context.Context, invID id.InvitationID) (*Invitation, error)
	GetInvitationByToken(ctx context.Context, token string) (*Invitation, error)
	UpdateInvitation(ctx context.Context, inv *Invitation) error
	ListInvitations(ctx context.Context, orgID id.OrgID) ([]*Invitation, error)

	// Teams
	CreateTeam(ctx context.Context, t *Team) error
	GetTeam(ctx context.Context, teamID id.TeamID) (*Team, error)
	UpdateTeam(ctx context.Context, t *Team) error
	DeleteTeam(ctx context.Context, teamID id.TeamID) error
	ListTeams(ctx context.Context, orgID id.OrgID) ([]*Team, error)
}
