package organization

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// Focused Service Interfaces (ISP Compliant)
// =============================================================================

// OrganizationOperations defines organization management operations
type OrganizationOperations interface {
	// CreateOrganization creates a new user-created organization
	CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, creatorUserID, appID, environmentID xid.ID) (*Organization, error)

	// FindOrganizationByID retrieves an organization by its ID
	FindOrganizationByID(ctx context.Context, id xid.ID) (*Organization, error)

	// FindOrganizationBySlug retrieves an organization by its slug within an app and environment
	FindOrganizationBySlug(ctx context.Context, appID, environmentID xid.ID, slug string) (*Organization, error)

	// ListOrganizations retrieves a paginated list of organizations within an app and environment
	ListOrganizations(ctx context.Context, filter *ListOrganizationsFilter) (*pagination.PageResponse[*Organization], error)

	// ListUserOrganizations retrieves a paginated list of organizations a user is a member of
	ListUserOrganizations(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Organization], error)

	// UpdateOrganization updates an existing organization
	UpdateOrganization(ctx context.Context, id xid.ID, req *UpdateOrganizationRequest) (*Organization, error)

	// DeleteOrganization deletes an organization (owner only)
	DeleteOrganization(ctx context.Context, id, userID xid.ID) error

	// ForceDeleteOrganization deletes an organization without permission checks
	// Use this for administrative operations or when permission checks would fail
	// (e.g., organization has no members). This should be restricted to admin users.
	ForceDeleteOrganization(ctx context.Context, id xid.ID) error
}

// MemberOperations defines member management operations
type MemberOperations interface {
	// AddMember adds a user as a member of an organization with a specified role
	AddMember(ctx context.Context, orgID, userID xid.ID, role string) (*Member, error)

	// FindMemberByID retrieves a member by their ID
	FindMemberByID(ctx context.Context, id xid.ID) (*Member, error)

	// FindMember retrieves a member by organization ID and user ID
	FindMember(ctx context.Context, orgID, userID xid.ID) (*Member, error)

	// ListMembers retrieves a paginated list of members in an organization
	ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*Member], error)

	// UpdateMember updates a member's role or status
	UpdateMember(ctx context.Context, id xid.ID, req *UpdateMemberRequest, updaterUserID xid.ID) (*Member, error)

	// UpdateMemberRole updates only the role of a member within an organization
	UpdateMemberRole(ctx context.Context, orgID, memberID xid.ID, newRole string, updaterUserID xid.ID) (*Member, error)

	// RemoveMember removes a member from an organization
	RemoveMember(ctx context.Context, id, removerUserID xid.ID) error

	// GetUserMemberships retrieves all organization memberships for a user
	GetUserMemberships(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Member], error)

	// RemoveUserFromAllOrganizations removes a user from all organizations they belong to
	RemoveUserFromAllOrganizations(ctx context.Context, userID xid.ID) error

	// Authorization helpers

	// IsMember checks if a user is a member of an organization
	IsMember(ctx context.Context, orgID, userID xid.ID) (bool, error)

	// IsOwner checks if a user is the owner of an organization
	IsOwner(ctx context.Context, orgID, userID xid.ID) (bool, error)

	// IsAdmin checks if a user is an admin or owner of an organization
	IsAdmin(ctx context.Context, orgID, userID xid.ID) (bool, error)

	// RequireOwner checks if a user is the owner and returns an error if not
	RequireOwner(ctx context.Context, orgID, userID xid.ID) error

	// RequireAdmin checks if a user is an admin or owner and returns an error if not
	RequireAdmin(ctx context.Context, orgID, userID xid.ID) error

	// RBAC Permission methods - use member's role from organization_members as single source of truth

	// CheckPermission checks if a user has permission to perform an action on a resource
	CheckPermission(ctx context.Context, orgID, userID xid.ID, action, resource string) (bool, error)

	// CheckPermissionWithContext checks permission with additional context variables for conditional evaluation
	CheckPermissionWithContext(ctx context.Context, orgID, userID xid.ID, action, resource string, contextVars map[string]string) (bool, error)

	// RequirePermission checks permission and returns an error if denied
	RequirePermission(ctx context.Context, orgID, userID xid.ID, action, resource string) error
}

// TeamOperations defines team management operations
type TeamOperations interface {
	// CreateTeam creates a new team within an organization
	CreateTeam(ctx context.Context, orgID xid.ID, req *CreateTeamRequest, creatorUserID xid.ID) (*Team, error)

	// FindTeamByID retrieves a team by its ID
	FindTeamByID(ctx context.Context, id xid.ID) (*Team, error)

	// FindTeamByName retrieves a team by name within an organization
	FindTeamByName(ctx context.Context, orgID xid.ID, name string) (*Team, error)

	// ListTeams retrieves a paginated list of teams in an organization
	ListTeams(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*Team], error)

	// UpdateTeam updates a team's details
	UpdateTeam(ctx context.Context, id xid.ID, req *UpdateTeamRequest, updaterUserID xid.ID) (*Team, error)

	// DeleteTeam deletes a team
	DeleteTeam(ctx context.Context, id, deleterUserID xid.ID) error

	// Team member operations (part of team aggregate)

	// AddTeamMember adds a member to a team
	AddTeamMember(ctx context.Context, teamID, memberID, adderUserID xid.ID) error

	// RemoveTeamMember removes a member from a team
	RemoveTeamMember(ctx context.Context, teamID, memberID, removerUserID xid.ID) error

	// ListTeamMembers retrieves a paginated list of team members
	ListTeamMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*TeamMember], error)

	// IsTeamMember checks if a member belongs to a team
	IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error)

	// FindTeamMemberByID retrieves a team member by its ID
	FindTeamMemberByID(ctx context.Context, id xid.ID) (*TeamMember, error)

	// FindTeamMember retrieves a team member by team ID and member ID
	FindTeamMember(ctx context.Context, teamID, memberID xid.ID) (*TeamMember, error)

	// ListMemberTeams retrieves all teams that a member belongs to
	ListMemberTeams(ctx context.Context, memberID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Team], error)
}

// InvitationOperations defines invitation management operations
type InvitationOperations interface {
	// InviteMember creates an invitation for a user to join an organization
	InviteMember(ctx context.Context, orgID xid.ID, req *InviteMemberRequest, inviterUserID xid.ID) (*Invitation, error)

	// FindInvitationByID retrieves an invitation by its ID
	FindInvitationByID(ctx context.Context, id xid.ID) (*Invitation, error)

	// FindInvitationByToken retrieves an invitation by its token
	FindInvitationByToken(ctx context.Context, token string) (*Invitation, error)

	// ListInvitations retrieves a paginated list of invitations for an organization
	ListInvitations(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*Invitation], error)

	// AcceptInvitation accepts an invitation and adds the user to the organization
	AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*Member, error)

	// DeclineInvitation declines an invitation
	DeclineInvitation(ctx context.Context, token string) error

	// CancelInvitation cancels a pending invitation (admin/owner only)
	CancelInvitation(ctx context.Context, id, cancellerUserID xid.ID) error

	// ResendInvitation resends an invitation with a new token and updated expiry
	ResendInvitation(ctx context.Context, id, resenderUserID xid.ID) (*Invitation, error)

	// CleanupExpiredInvitations removes all expired invitations
	CleanupExpiredInvitations(ctx context.Context) (int, error)
}

// =============================================================================
// Composite Interface (for backward compatibility)
// =============================================================================

// CompositeOrganizationService defines the complete contract for all organization-related service operations
// This interface combines all focused operations and is useful for backward compatibility
// or when a component needs access to all operations.
// New code should prefer using the focused interfaces (OrganizationOperations, MemberOperations, etc.)
type CompositeOrganizationService interface {
	OrganizationOperations
	MemberOperations
	TeamOperations
	InvitationOperations
}
