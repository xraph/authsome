package organization

import (
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
)

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
) *organization.Service {
	return organization.NewService(orgRepo, memberRepo, teamRepo, inviteRepo, config, rbacSvc)
}

// Request/Response types

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
