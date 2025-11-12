package organization

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// OrganizationRepository defines the interface for organization data access
type OrganizationRepository interface {
	Create(ctx context.Context, org *schema.Organization) error
	FindByID(ctx context.Context, id xid.ID) (*schema.Organization, error)
	FindBySlug(ctx context.Context, appID, environmentID xid.ID, slug string) (*schema.Organization, error)
	ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*schema.Organization, error)
	ListByApp(ctx context.Context, appID, environmentID xid.ID, limit, offset int) ([]*schema.Organization, error)
	Update(ctx context.Context, org *schema.Organization) error
	Delete(ctx context.Context, id xid.ID) error
	CountByUser(ctx context.Context, userID xid.ID) (int, error)
	CountByApp(ctx context.Context, appID, environmentID xid.ID) (int, error)
}

// OrganizationMemberRepository defines the interface for member data access
type OrganizationMemberRepository interface {
	Create(ctx context.Context, member *schema.OrganizationMember) error
	FindByID(ctx context.Context, id xid.ID) (*schema.OrganizationMember, error)
	FindByUserAndOrg(ctx context.Context, userID, orgID xid.ID) (*schema.OrganizationMember, error)
	ListByOrganization(ctx context.Context, orgID xid.ID, limit, offset int) ([]*schema.OrganizationMember, error)
	ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*schema.OrganizationMember, error)
	Update(ctx context.Context, member *schema.OrganizationMember) error
	Delete(ctx context.Context, id xid.ID) error
	DeleteByUserAndOrg(ctx context.Context, userID, orgID xid.ID) error
	CountByOrganization(ctx context.Context, orgID xid.ID) (int, error)
	CountByUser(ctx context.Context, userID xid.ID) (int, error)
}

// OrganizationTeamRepository defines the interface for team data access
type OrganizationTeamRepository interface {
	Create(ctx context.Context, team *schema.OrganizationTeam) error
	FindByID(ctx context.Context, id xid.ID) (*schema.OrganizationTeam, error)
	FindByName(ctx context.Context, orgID xid.ID, name string) (*schema.OrganizationTeam, error)
	ListByOrganization(ctx context.Context, orgID xid.ID, limit, offset int) ([]*schema.OrganizationTeam, error)
	Update(ctx context.Context, team *schema.OrganizationTeam) error
	Delete(ctx context.Context, id xid.ID) error
	CountByOrganization(ctx context.Context, orgID xid.ID) (int, error)
	AddMember(ctx context.Context, teamMember *schema.OrganizationTeamMember) error
	RemoveMember(ctx context.Context, teamID, memberID xid.ID) error
	ListMembers(ctx context.Context, teamID xid.ID, limit, offset int) ([]*schema.OrganizationTeamMember, error)
	CountMembers(ctx context.Context, teamID xid.ID) (int, error)
}

// OrganizationInvitationRepository defines the interface for invitation data access
type OrganizationInvitationRepository interface {
	Create(ctx context.Context, invitation *schema.OrganizationInvitation) error
	FindByID(ctx context.Context, id xid.ID) (*schema.OrganizationInvitation, error)
	FindByToken(ctx context.Context, token string) (*schema.OrganizationInvitation, error)
	FindByEmail(ctx context.Context, orgID xid.ID, email string) (*schema.OrganizationInvitation, error)
	ListByOrganization(ctx context.Context, orgID xid.ID, status string, limit, offset int) ([]*schema.OrganizationInvitation, error)
	ListByEmail(ctx context.Context, email string, status string, limit, offset int) ([]*schema.OrganizationInvitation, error)
	Update(ctx context.Context, invitation *schema.OrganizationInvitation) error
	Delete(ctx context.Context, id xid.ID) error
	DeleteExpired(ctx context.Context) (int, error)
	CountByOrganization(ctx context.Context, orgID xid.ID, status string) (int, error)
	CountByEmail(ctx context.Context, email string, status string) (int, error)
}
