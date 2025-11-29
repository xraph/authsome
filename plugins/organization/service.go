package organization

import (
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
)

// =============================================================================
// Service Types
// =============================================================================

// Service wraps the core organization service
type Service = organization.Service

// Config holds the organization service configuration
type Config = organization.Config

// NewService creates a new organization service
func NewService(
	orgRepo Repository,
	memberRepo MemberRepository,
	teamRepo TeamRepository,
	inviteRepo InvitationRepository,
	config Config,
	rbacSvc *rbac.Service,
	roleRepo rbac.RoleRepository,
) *organization.Service {
	return organization.NewService(orgRepo, memberRepo, teamRepo, inviteRepo, config, rbacSvc, roleRepo)
}

// =============================================================================
// Entity Types
// =============================================================================

// Organization represents an organization (workspace/tenant)
type Organization = organization.Organization

// Member represents a user's membership in an organization
type Member = organization.Member

// Team represents a team within an organization
type Team = organization.Team

// TeamMember represents a member's assignment to a team
type TeamMember = organization.TeamMember

// Invitation represents an invitation to join an organization
type Invitation = organization.Invitation

// UserInfo contains basic user information for display purposes (embedded in Member/TeamMember)
type UserInfo = organization.UserInfo

// =============================================================================
// Request Types
// =============================================================================

// CreateOrganizationRequest represents the request to create an organization
type CreateOrganizationRequest = organization.CreateOrganizationRequest

// UpdateOrganizationRequest represents the request to update an organization
type UpdateOrganizationRequest = organization.UpdateOrganizationRequest

// CreateTeamRequest represents the request to create a team
type CreateTeamRequest = organization.CreateTeamRequest

// UpdateTeamRequest represents the request to update a team
type UpdateTeamRequest = organization.UpdateTeamRequest

// UpdateMemberRequest represents the request to update a member
type UpdateMemberRequest = organization.UpdateMemberRequest

// InviteMemberRequest represents the request to invite a member
type InviteMemberRequest = organization.InviteMemberRequest

// =============================================================================
// Filter Types (for listing/pagination)
// =============================================================================

// ListOrganizationsFilter represents filters for listing organizations
type ListOrganizationsFilter = organization.ListOrganizationsFilter

// ListMembersFilter represents filters for listing members
type ListMembersFilter = organization.ListMembersFilter

// ListTeamsFilter represents filters for listing teams
type ListTeamsFilter = organization.ListTeamsFilter

// ListTeamMembersFilter represents filters for listing team members
type ListTeamMembersFilter = organization.ListTeamMembersFilter

// ListInvitationsFilter represents filters for listing invitations
type ListInvitationsFilter = organization.ListInvitationsFilter

// =============================================================================
// Interface Types
// =============================================================================

// OrganizationOperations defines operations for managing organizations
type OrganizationOperations = organization.OrganizationOperations

// MemberOperations defines operations for managing organization members
type MemberOperations = organization.MemberOperations

// TeamOperations defines operations for managing teams
type TeamOperations = organization.TeamOperations

// InvitationOperations defines operations for managing invitations
type InvitationOperations = organization.InvitationOperations

// CompositeOrganizationService is the full interface for the organization service
type CompositeOrganizationService = organization.CompositeOrganizationService

// =============================================================================
// Validation Helper Functions
// =============================================================================

// ValidRoles returns all valid member roles
var ValidRoles = organization.ValidRoles

// ValidStatuses returns all valid member statuses
var ValidStatuses = organization.ValidStatuses

// ValidInvitationStatuses returns all valid invitation statuses
var ValidInvitationStatuses = organization.ValidInvitationStatuses

// IsValidRole checks if a role is valid
var IsValidRole = organization.IsValidRole

// IsValidStatus checks if a status is valid
var IsValidStatus = organization.IsValidStatus

// IsValidInvitationStatus checks if an invitation status is valid
var IsValidInvitationStatus = organization.IsValidInvitationStatus

// DefaultConfig returns the default organization configuration
var DefaultConfig = organization.DefaultConfig

// =============================================================================
// Schema Conversion Functions
// =============================================================================

// FromSchemaOrganization converts a schema organization to domain organization
var FromSchemaOrganization = organization.FromSchemaOrganization

// FromSchemaOrganizations converts multiple schema organizations to domain organizations
var FromSchemaOrganizations = organization.FromSchemaOrganizations

// FromSchemaMember converts a schema member to domain member
var FromSchemaMember = organization.FromSchemaMember

// FromSchemaMembers converts multiple schema members to domain members
var FromSchemaMembers = organization.FromSchemaMembers

// FromSchemaTeam converts a schema team to domain team
var FromSchemaTeam = organization.FromSchemaTeam

// FromSchemaTeams converts multiple schema teams to domain teams
var FromSchemaTeams = organization.FromSchemaTeams

// FromSchemaTeamMember converts a schema team member to domain team member
var FromSchemaTeamMember = organization.FromSchemaTeamMember

// FromSchemaTeamMembers converts multiple schema team members to domain team members
var FromSchemaTeamMembers = organization.FromSchemaTeamMembers

// FromSchemaInvitation converts a schema invitation to domain invitation
var FromSchemaInvitation = organization.FromSchemaInvitation

// FromSchemaInvitations converts multiple schema invitations to domain invitations
var FromSchemaInvitations = organization.FromSchemaInvitations
