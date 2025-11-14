package app

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// Focused ServiceImpl Interfaces (ISP Compliant)
// =============================================================================

// AppOperations defines app management operations
type AppOperations interface {
	// UpdateConfig updates the app service configuration
	UpdateConfig(cfg Config)

	// CreateApp creates a new application based on the provided request and returns the created app or an error.
	CreateApp(ctx context.Context, req *CreateAppRequest) (*App, error)

	// GetPlatformApp retrieves the platform-level app entity, often representing the default or primary tenant.
	GetPlatformApp(ctx context.Context) (*App, error)

	// FindAppByID retrieves an app by its unique identifier. Returns the app if found or an error if not.
	FindAppByID(ctx context.Context, id xid.ID) (*App, error)

	// FindAppBySlug retrieves an app by its unique slug from the repository.
	// Returns the app if found, or an error if not found or if an issue occurred.
	FindAppBySlug(ctx context.Context, slug string) (*App, error)

	// UpdateApp updates an existing app's details identified by its ID using the specified update request parameters.
	UpdateApp(ctx context.Context, id xid.ID, req *UpdateAppRequest) (*App, error)

	// DeleteApp deletes an app identified by the specified ID. Platform app cannot be deleted unless another app is made platform first.
	DeleteApp(ctx context.Context, id xid.ID) error

	// ListApps fetches a paginated list of apps with optional filtering.
	ListApps(ctx context.Context, filter *ListAppsFilter) (*pagination.PageResponse[*App], error)

	// CountApps returns the total number of apps available in the system or database.
	// It takes a context to handle deadlines, cancelation signals, and other request-scoped values.
	CountApps(ctx context.Context) (int, error)

	// SetPlatformApp transfers platform status to the specified app. Only one app can be platform at a time.
	SetPlatformApp(ctx context.Context, newPlatformAppID xid.ID) error

	// IsPlatformApp checks if the specified app is the platform app.
	IsPlatformApp(ctx context.Context, appID xid.ID) (bool, error)
}

// MemberOperations defines member management operations
type MemberOperations interface {
	// CreateMember creates a new member within the specified context and associates it with the provided member details.
	CreateMember(ctx context.Context, member *Member) (*Member, error)

	// FindMemberByID retrieves a member by their unique identifier. Returns the member or an error if not found.
	FindMemberByID(ctx context.Context, id xid.ID) (*Member, error)

	// FindMember retrieves a Member within an app using the provided appID and userID. Returns the Member or an error.
	FindMember(ctx context.Context, appID, userID xid.ID) (*Member, error)

	// ListMembers fetches a paginated list of members for the specified context with optional filtering.
	ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*Member], error)

	// GetUserMemberships retrieves all app memberships for a given user ID. Returns a slice of Member or an error.
	GetUserMemberships(ctx context.Context, userID xid.ID) ([]*Member, error)

	// UpdateMember updates an existing member's details. Takes a context and the member object with updated fields.
	UpdateMember(ctx context.Context, member *Member) error

	// DeleteMember removes a member from the app using the provided member ID. Returns an error if deletion fails.
	DeleteMember(ctx context.Context, id xid.ID) error

	// CountMembers returns the total number of members in a specific app identified by appID.
	CountMembers(ctx context.Context, appID xid.ID) (int, error)

	// IsOwner checks if a user is the owner of an app
	IsOwner(ctx context.Context, appID, userID xid.ID) (bool, error)

	// IsAdmin checks if a user is an admin or owner of an app
	IsAdmin(ctx context.Context, appID, userID xid.ID) (bool, error)

	// RequireOwner checks if a user is the owner of an app and returns an error if not
	RequireOwner(ctx context.Context, appID, userID xid.ID) error

	// RequireAdmin checks if a user is an admin or owner of an app and returns an error if not
	RequireAdmin(ctx context.Context, appID, userID xid.ID) error
}

// TeamOperations defines team management operations
type TeamOperations interface {
	// CreateTeam creates a new team within the given context using the provided team details. Returns an error if creation fails.
	CreateTeam(ctx context.Context, team *Team) error

	// FindTeamByID retrieves a team by its unique identifier. Returns the team or an error if not found.
	FindTeamByID(ctx context.Context, id xid.ID) (*Team, error)

	// FindTeamByName finds a team by name within an app
	FindTeamByName(ctx context.Context, appID xid.ID, name string) (*Team, error)

	// ListTeams fetches a paginated list of teams with optional filtering based on the provided context.
	ListTeams(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*Team], error)

	// UpdateTeam updates an existing team's details. Takes a context and the team object with updated fields.
	UpdateTeam(ctx context.Context, team *Team) error

	// DeleteTeam removes a team identified by the specified ID. Returns an error if deletion fails.
	DeleteTeam(ctx context.Context, id xid.ID) error

	// CountTeams returns the total number of teams in a specific app identified by appID.
	CountTeams(ctx context.Context, appID xid.ID) (int, error)

	// AddTeamMember adds a member to a team. Returns the created TeamMember or an error if the operation fails.
	AddTeamMember(ctx context.Context, tm *TeamMember) (*TeamMember, error)

	// RemoveTeamMember removes a member from a team using the provided teamID and memberID. Returns an error if removal fails.
	RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error

	// ListTeamMembers fetches a paginated list of team members with optional filtering.
	ListTeamMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*TeamMember], error)

	// CountTeamMembers returns the total number of members in a specific team identified by teamID.
	CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error)

	// IsTeamMember checks if a member is part of a team
	IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error)

	// ListMemberTeams lists all teams a member belongs to with pagination
	ListMemberTeams(ctx context.Context, filter *ListMemberTeamsFilter) (*pagination.PageResponse[*Team], error)
}

// InvitationOperations defines invitation management operations
type InvitationOperations interface {
	// CreateInvitation creates an app invitation
	CreateInvitation(ctx context.Context, inv *Invitation) error

	// FindInvitationByID finds an invitation by ID
	FindInvitationByID(ctx context.Context, id xid.ID) (*Invitation, error)

	// FindInvitationByToken finds an invitation by token
	FindInvitationByToken(ctx context.Context, token string) (*Invitation, error)

	// ListInvitations lists invitations for an app with pagination
	ListInvitations(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*Invitation], error)

	// AcceptInvitation accepts an invitation and creates a member
	AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*Member, error)

	// DeclineInvitation declines an invitation
	DeclineInvitation(ctx context.Context, token string) error

	// CancelInvitation cancels an invitation (only admins/owners)
	CancelInvitation(ctx context.Context, id, cancellerUserID xid.ID) error

	// ResendInvitation resends an invitation by creating a new token and updating expiry
	ResendInvitation(ctx context.Context, id, resenderUserID xid.ID) (*Invitation, error)

	// CleanupExpiredInvitations removes expired invitations
	CleanupExpiredInvitations(ctx context.Context) (int, error)
}

// =============================================================================
// Composite Interface (for backward compatibility)
// =============================================================================

// Service defines the complete contract for all app-related service operations
// This interface combines all focused operations and is useful for backward compatibility
// or when a component needs access to all operations.
// New code should prefer using the focused interfaces (AppOperations, MemberOperations, etc.)
type Service interface {

	// UpdateConfig updates the app service configuration
	UpdateConfig(cfg Config)

	// GetPlatformApp retrieves the platform-level app entity, often representing the default or primary tenant.
	GetPlatformApp(ctx context.Context) (*App, error)

	// CreateApp creates a new application based on the provided request and returns the created app or an error.
	CreateApp(ctx context.Context, req *CreateAppRequest) (*App, error)

	// FindAppByID retrieves an app by its unique identifier. Returns the app if found or an error if not.
	FindAppByID(ctx context.Context, id xid.ID) (*App, error)

	// FindAppBySlug retrieves an app by its unique slug from the repository.
	// Returns the app if found, or an error if not found or if an issue occurred.
	FindAppBySlug(ctx context.Context, slug string) (*App, error)

	// UpdateApp updates an existing app's details identified by its ID using the specified update request parameters.
	UpdateApp(ctx context.Context, id xid.ID, req *UpdateAppRequest) (*App, error)

	// DeleteApp deletes an app identified by the specified ID. Returns an error if the deletion fails.
	DeleteApp(ctx context.Context, id xid.ID) error

	// ListApps fetches a paginated list of apps with optional filtering.
	ListApps(ctx context.Context, filter *ListAppsFilter) (*pagination.PageResponse[*App], error)

	// CountApps returns the total number of apps available in the system or database.
	// It takes a context to handle deadlines, cancelation signals, and other request-scoped values.
	CountApps(ctx context.Context) (int, error)

	// CreateMember creates a new member within the specified context and associates it with the provided member details.
	CreateMember(ctx context.Context, member *Member) (*Member, error)

	// FindMemberByID retrieves a member by their unique identifier. Returns the member or an error if not found.
	FindMemberByID(ctx context.Context, id xid.ID) (*Member, error)

	// FindMember retrieves a Member within an app using the provided appID and userID. Returns the Member or an error.
	FindMember(ctx context.Context, appID, userID xid.ID) (*Member, error)

	// ListMembers retrieves a paginated list of members with optional filtering by role and status.
	ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*Member], error)

	// CountMembers counts the number of members associated with a specific application by its appID.
	CountMembers(ctx context.Context, appID xid.ID) (int, error)

	// GetUserMemberships retrieves all apps where the user is a member.
	// Returns a list of members with their associated app details.
	GetUserMemberships(ctx context.Context, userID xid.ID) ([]*Member, error)

	// UpdateMember updates the details of an existing app member in the system.
	UpdateMember(ctx context.Context, member *Member) error

	// DeleteMember removes a member identified by the given ID from the system. Returns an error if the operation fails.
	DeleteMember(ctx context.Context, id xid.ID) error

	// CreateTeam creates a new team and associates it with an app. Returns an error if the operation fails.
	CreateTeam(ctx context.Context, team *Team) error

	// FindTeamByID retrieves a team by its unique identifier.
	// Returns the corresponding team or an error if not found.
	FindTeamByID(ctx context.Context, id xid.ID) (*Team, error)

	// ListTeams retrieves a paginated list of teams associated with a specific app.
	ListTeams(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*Team], error)

	// CountTeams returns the total number of teams associated with a specific app identified by the given appID.
	CountTeams(ctx context.Context, appID xid.ID) (int, error)

	// UpdateTeam updates an existing team with new information provided in the Team object. Returns an error if the update fails.
	UpdateTeam(ctx context.Context, team *Team) error

	// DeleteTeam removes a team with the specified ID from the system. Returns an error if the operation fails.
	DeleteTeam(ctx context.Context, id xid.ID) error

	// AddTeamMember adds a member to a specific team within the system based on provided team member details.
	// Returns the created team member and an error if the operation fails.
	AddTeamMember(ctx context.Context, tm *TeamMember) (*TeamMember, error)

	// RemoveTeamMember removes a specified member from a team identified by the given teamID and memberID. Returns an error if the operation fails.
	RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error

	// ListTeamMembers retrieves a paginated list of members for the specified team.
	ListTeamMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*TeamMember], error)

	// CountTeamMembers returns the total number of members in a specific team identified by teamID.
	CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error)

	// IsUserMember checks if a user is a member of an app with active status.
	// Returns true if the user is an active member, false otherwise.
	IsUserMember(ctx context.Context, appID, userID xid.ID) (bool, error)

	// CreateInvitation creates a new invitation for an application.
	CreateInvitation(ctx context.Context, inv *Invitation) error

	// FindInvitationByID retrieves an invitation by its ID.
	FindInvitationByID(ctx context.Context, id xid.ID) (*Invitation, error)

	// FindInvitationByToken retrieves an invitation by its unique token.
	FindInvitationByToken(ctx context.Context, token string) (*Invitation, error)

	// ListInvitations retrieves a paginated list of invitations for an app with optional filtering.
	ListInvitations(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*Invitation], error)

	// AcceptInvitation accepts an invitation and creates a member.
	// Returns the created member or an error if the invitation is invalid, expired, or already used.
	AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*Member, error)

	// DeclineInvitation declines a pending invitation.
	DeclineInvitation(ctx context.Context, token string) error

	// CancelInvitation cancels a pending invitation (requires admin/owner permissions).
	CancelInvitation(ctx context.Context, id, cancellerUserID xid.ID) error

	// ResendInvitation resends an invitation by generating a new token and updating expiry (requires admin/owner permissions).
	ResendInvitation(ctx context.Context, id, resenderUserID xid.ID) (*Invitation, error)

	// CleanupExpiredInvitations removes all expired invitations from the system.
	// Returns the number of invitations deleted.
	CleanupExpiredInvitations(ctx context.Context) (int, error)

	// Authorization helpers

	// IsOwner checks if a user is the owner of an app.
	IsOwner(ctx context.Context, appID, userID xid.ID) (bool, error)

	// IsAdmin checks if a user is an admin or owner of an app.
	IsAdmin(ctx context.Context, appID, userID xid.ID) (bool, error)

	// RequireOwner checks if a user is the owner of an app and returns an error if not.
	RequireOwner(ctx context.Context, appID, userID xid.ID) error

	// RequireAdmin checks if a user is an admin or owner of an app and returns an error if not.
	RequireAdmin(ctx context.Context, appID, userID xid.ID) error

	// Team query operations

	// IsTeamMember checks if a member is part of a team.
	IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error)

	// ListMemberTeams lists all teams a member belongs to with pagination.
	ListMemberTeams(ctx context.Context, filter *ListMemberTeamsFilter) (*pagination.PageResponse[*Team], error)

	// FindTeamByName finds a team by name within an app.
	FindTeamByName(ctx context.Context, appID xid.ID, name string) (*Team, error)

	// RBAC Permission Checking

	// CheckPermission checks if a user has permission to perform an action on a resource.
	CheckPermission(ctx context.Context, userID, appID xid.ID, action, resourceType, resourceID string) (bool, error)

	// CheckPermissionWithContext checks permission with additional context variables for conditional permissions.
	CheckPermissionWithContext(ctx context.Context, userID, appID xid.ID, action, resourceType, resourceID string, contextVars map[string]string) (bool, error)

	// RequirePermission checks if a user has permission and returns an error if denied.
	RequirePermission(ctx context.Context, userID, appID xid.ID, action, resourceType, resourceID string) error

	// RequirePermissionWithContext checks permission with context variables and returns error if denied.
	RequirePermissionWithContext(ctx context.Context, userID, appID xid.ID, action, resourceType, resourceID string, contextVars map[string]string) error
}
