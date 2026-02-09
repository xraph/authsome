package organization

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// OrganizationRepository defines the interface for organization data access.
type OrganizationRepository interface {
	// Create creates a new organization
	Create(ctx context.Context, org *Organization) error

	// FindByID retrieves an organization by its ID
	FindByID(ctx context.Context, id xid.ID) (*Organization, error)

	// FindBySlug retrieves an organization by its slug within an app and environment
	FindBySlug(ctx context.Context, appID, envID xid.ID, slug string) (*Organization, error)

	// ListByApp retrieves a paginated list of organizations within an app and environment
	ListByApp(ctx context.Context, filter *ListOrganizationsFilter) (*pagination.PageResponse[*Organization], error)

	// ListByUser retrieves a paginated list of organizations a user is a member of
	ListByUser(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Organization], error)

	// Update updates an existing organization
	Update(ctx context.Context, org *Organization) error

	// Delete deletes an organization by ID
	Delete(ctx context.Context, id xid.ID) error

	// CountByUser counts the number of organizations a user has created or is a member of
	CountByUser(ctx context.Context, userID xid.ID) (int, error)
}

// MemberRepository defines the interface for organization member data access.
type MemberRepository interface {
	// Create creates a new organization member
	Create(ctx context.Context, member *Member) error

	// FindByID retrieves a member by their ID
	FindByID(ctx context.Context, id xid.ID) (*Member, error)

	// FindByUserAndOrg retrieves a member by user ID and organization ID
	FindByUserAndOrg(ctx context.Context, userID, orgID xid.ID) (*Member, error)

	// ListByOrganization retrieves a paginated list of members in an organization
	ListByOrganization(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*Member], error)

	// ListByUser retrieves a paginated list of organization memberships for a user
	ListByUser(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Member], error)

	// Update updates an existing member
	Update(ctx context.Context, member *Member) error

	// Delete deletes a member by ID
	Delete(ctx context.Context, id xid.ID) error

	// DeleteByUserAndOrg deletes a member by user ID and organization ID
	DeleteByUserAndOrg(ctx context.Context, userID, orgID xid.ID) error

	// CountByOrganization counts the number of members in an organization
	CountByOrganization(ctx context.Context, orgID xid.ID) (int, error)
}

// TeamRepository defines the interface for organization team data access.
type TeamRepository interface {
	// Create creates a new team
	Create(ctx context.Context, team *Team) error

	// FindByID retrieves a team by its ID
	FindByID(ctx context.Context, id xid.ID) (*Team, error)

	// FindByName retrieves a team by name within an organization
	FindByName(ctx context.Context, orgID xid.ID, name string) (*Team, error)

	// ListByOrganization retrieves a paginated list of teams in an organization
	ListByOrganization(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*Team], error)

	// Update updates an existing team
	Update(ctx context.Context, team *Team) error

	// Delete deletes a team by ID
	Delete(ctx context.Context, id xid.ID) error

	// CountByOrganization counts the number of teams in an organization
	CountByOrganization(ctx context.Context, orgID xid.ID) (int, error)

	// AddMember adds a member to a team (part of team aggregate)
	AddMember(ctx context.Context, tm *TeamMember) error

	// RemoveMember removes a member from a team
	RemoveMember(ctx context.Context, teamID, memberID xid.ID) error

	// ListMembers retrieves a paginated list of team members
	ListMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*TeamMember], error)

	// CountMembers counts the number of members in a team
	CountMembers(ctx context.Context, teamID xid.ID) (int, error)

	// IsTeamMember checks if a member belongs to a team
	IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error)

	// FindTeamMemberByID retrieves a team member by its ID
	FindTeamMemberByID(ctx context.Context, id xid.ID) (*TeamMember, error)

	// FindTeamMember retrieves a team member by team ID and member ID
	FindTeamMember(ctx context.Context, teamID, memberID xid.ID) (*TeamMember, error)

	// ListMemberTeams retrieves all teams that a member belongs to
	ListMemberTeams(ctx context.Context, memberID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Team], error)
}

// InvitationRepository defines the interface for organization invitation data access.
type InvitationRepository interface {
	// Create creates a new invitation
	Create(ctx context.Context, inv *Invitation) error

	// FindByID retrieves an invitation by its ID
	FindByID(ctx context.Context, id xid.ID) (*Invitation, error)

	// FindByToken retrieves an invitation by its token
	FindByToken(ctx context.Context, token string) (*Invitation, error)

	// ListByOrganization retrieves a paginated list of invitations for an organization
	ListByOrganization(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*Invitation], error)

	// Update updates an existing invitation
	Update(ctx context.Context, inv *Invitation) error

	// Delete deletes an invitation by ID
	Delete(ctx context.Context, id xid.ID) error

	// DeleteExpired deletes all expired invitations and returns the count
	DeleteExpired(ctx context.Context) (int, error)
}
