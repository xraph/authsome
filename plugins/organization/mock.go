package organization

import (
	"context"
	"sync"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
)

// MockService provides a mock implementation of organization.Service for testing
type MockService struct {
	mu sync.RWMutex

	// Storage
	organizations map[xid.ID]*organization.Organization
	members       map[xid.ID]*organization.Member
	teams         map[xid.ID]*organization.Team
	invitations   map[xid.ID]*organization.Invitation

	// Error injection for testing
	CreateOrgError        error
	FindOrgError          error
	AddMemberError        error
	InviteMemberError     error
	AcceptInvitationError error
}

// NewMockService creates a new mock organization service
func NewMockService() *MockService {
	return &MockService{
		organizations: make(map[xid.ID]*organization.Organization),
		members:       make(map[xid.ID]*organization.Member),
		teams:         make(map[xid.ID]*organization.Team),
		invitations:   make(map[xid.ID]*organization.Invitation),
	}
}

// CreateOrganization mocks creating an organization
func (m *MockService) CreateOrganization(ctx context.Context, req *organization.CreateOrganizationRequest, creatorUserID, appID, environmentID xid.ID) (*organization.Organization, error) {
	if m.CreateOrgError != nil {
		return nil, m.CreateOrgError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	org := &organization.Organization{
		ID:            xid.New(),
		AppID:         appID,
		EnvironmentID: environmentID,
		Name:          req.Name,
		Slug:          req.Slug,
		CreatedBy:     creatorUserID,
	}
	if req.Logo != nil {
		org.Logo = *req.Logo
	}
	org.Metadata = req.Metadata

	m.organizations[org.ID] = org
	return org, nil
}

// FindOrganizationByID mocks finding an organization by ID
func (m *MockService) FindOrganizationByID(ctx context.Context, id xid.ID) (*organization.Organization, error) {
	if m.FindOrgError != nil {
		return nil, m.FindOrgError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	org, exists := m.organizations[id]
	if !exists {
		return nil, organization.OrganizationNotFound()
	}
	return org, nil
}

// FindOrganizationBySlug mocks finding an organization by slug
func (m *MockService) FindOrganizationBySlug(ctx context.Context, appID, environmentID xid.ID, slug string) (*organization.Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, org := range m.organizations {
		if org.AppID == appID && org.EnvironmentID == environmentID && org.Slug == slug {
			return org, nil
		}
	}
	return nil, organization.OrganizationNotFound()
}

// ListOrganizations mocks listing organizations
func (m *MockService) ListOrganizations(ctx context.Context, filter *organization.ListOrganizationsFilter) (*pagination.PageResponse[*organization.Organization], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var orgs []*organization.Organization
	for _, org := range m.organizations {
		if org.AppID == filter.AppID && org.EnvironmentID == filter.EnvironmentID {
			orgs = append(orgs, org)
		}
	}

	return pagination.NewPageResponse(orgs, int64(len(orgs)), &filter.PaginationParams), nil
}

// ListUserOrganizations mocks listing user organizations
func (m *MockService) ListUserOrganizations(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*organization.Organization], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var orgs []*organization.Organization
	for _, member := range m.members {
		if member.UserID == userID {
			if org, exists := m.organizations[member.OrganizationID]; exists {
				orgs = append(orgs, org)
			}
		}
	}

	return pagination.NewPageResponse(orgs, int64(len(orgs)), filter), nil
}

// UpdateOrganization mocks updating an organization
func (m *MockService) UpdateOrganization(ctx context.Context, id xid.ID, req *organization.UpdateOrganizationRequest) (*organization.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	org, exists := m.organizations[id]
	if !exists {
		return nil, organization.OrganizationNotFound()
	}

	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Logo != nil {
		org.Logo = *req.Logo
	}
	if req.Metadata != nil {
		org.Metadata = req.Metadata
	}

	return org, nil
}

// DeleteOrganization mocks deleting an organization
func (m *MockService) DeleteOrganization(ctx context.Context, id, userID xid.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.organizations, id)
	return nil
}

// AddMember mocks adding a member
func (m *MockService) AddMember(ctx context.Context, orgID, userID xid.ID, role string) (*organization.Member, error) {
	if m.AddMemberError != nil {
		return nil, m.AddMemberError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	member := &organization.Member{
		ID:             xid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		Status:         organization.StatusActive,
	}

	m.members[member.ID] = member
	return member, nil
}

// FindMemberByID mocks finding a member by ID
func (m *MockService) FindMemberByID(ctx context.Context, id xid.ID) (*organization.Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	member, exists := m.members[id]
	if !exists {
		return nil, organization.MemberNotFound()
	}
	return member, nil
}

// FindMember mocks finding a member by org and user ID
func (m *MockService) FindMember(ctx context.Context, orgID, userID xid.ID) (*organization.Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, member := range m.members {
		if member.OrganizationID == orgID && member.UserID == userID {
			return member, nil
		}
	}
	return nil, organization.MemberNotFound()
}

// ListMembers mocks listing members
func (m *MockService) ListMembers(ctx context.Context, filter *organization.ListMembersFilter) (*pagination.PageResponse[*organization.Member], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var members []*organization.Member
	for _, member := range m.members {
		if member.OrganizationID == filter.OrganizationID {
			members = append(members, member)
		}
	}

	return pagination.NewPageResponse(members, int64(len(members)), &filter.PaginationParams), nil
}

// UpdateMember mocks updating a member
func (m *MockService) UpdateMember(ctx context.Context, id xid.ID, req *organization.UpdateMemberRequest, updaterUserID xid.ID) (*organization.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	member, exists := m.members[id]
	if !exists {
		return nil, organization.MemberNotFound()
	}

	if req.Role != nil {
		member.Role = *req.Role
	}
	if req.Status != nil {
		member.Status = *req.Status
	}

	return member, nil
}

// RemoveMember mocks removing a member
func (m *MockService) RemoveMember(ctx context.Context, id, removerUserID xid.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.members, id)
	return nil
}

// GetUserMemberships mocks getting user memberships
func (m *MockService) GetUserMemberships(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*organization.Member], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var members []*organization.Member
	for _, member := range m.members {
		if member.UserID == userID {
			members = append(members, member)
		}
	}

	return pagination.NewPageResponse(members, int64(len(members)), filter), nil
}

// RemoveUserFromAllOrganizations mocks removing user from all orgs
func (m *MockService) RemoveUserFromAllOrganizations(ctx context.Context, userID xid.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, member := range m.members {
		if member.UserID == userID {
			delete(m.members, id)
		}
	}
	return nil
}

// IsMember mocks checking membership
func (m *MockService) IsMember(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, member := range m.members {
		if member.OrganizationID == orgID && member.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

// IsOwner mocks checking ownership
func (m *MockService) IsOwner(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, member := range m.members {
		if member.OrganizationID == orgID && member.UserID == userID && member.Role == organization.RoleOwner {
			return true, nil
		}
	}
	return false, nil
}

// IsAdmin mocks checking admin status
func (m *MockService) IsAdmin(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, member := range m.members {
		if member.OrganizationID == orgID && member.UserID == userID {
			return member.Role == organization.RoleOwner || member.Role == organization.RoleAdmin, nil
		}
	}
	return false, nil
}

// RequireOwner mocks requiring owner status
func (m *MockService) RequireOwner(ctx context.Context, orgID, userID xid.ID) error {
	isOwner, _ := m.IsOwner(ctx, orgID, userID)
	if !isOwner {
		return organization.NotOwner()
	}
	return nil
}

// RequireAdmin mocks requiring admin status
func (m *MockService) RequireAdmin(ctx context.Context, orgID, userID xid.ID) error {
	isAdmin, _ := m.IsAdmin(ctx, orgID, userID)
	if !isAdmin {
		return organization.NotAdmin()
	}
	return nil
}

// Stub implementations for team and invitation operations
// These can be expanded as needed for testing

func (m *MockService) CreateTeam(ctx context.Context, orgID xid.ID, req *organization.CreateTeamRequest, creatorUserID xid.ID) (*organization.Team, error) {
	return nil, nil
}

func (m *MockService) FindTeamByID(ctx context.Context, id xid.ID) (*organization.Team, error) {
	return nil, organization.TeamNotFound()
}

func (m *MockService) FindTeamByName(ctx context.Context, orgID xid.ID, name string) (*organization.Team, error) {
	return nil, organization.TeamNotFound()
}

func (m *MockService) ListTeams(ctx context.Context, filter *organization.ListTeamsFilter) (*pagination.PageResponse[*organization.Team], error) {
	return pagination.NewPageResponse([]*organization.Team{}, 0, &filter.PaginationParams), nil
}

func (m *MockService) UpdateTeam(ctx context.Context, id xid.ID, req *organization.UpdateTeamRequest, updaterUserID xid.ID) (*organization.Team, error) {
	return nil, organization.TeamNotFound()
}

func (m *MockService) DeleteTeam(ctx context.Context, id, deleterUserID xid.ID) error {
	return nil
}

func (m *MockService) AddTeamMember(ctx context.Context, teamID, memberID, adderUserID xid.ID) error {
	return nil
}

func (m *MockService) RemoveTeamMember(ctx context.Context, teamID, memberID, removerUserID xid.ID) error {
	return nil
}

func (m *MockService) ListTeamMembers(ctx context.Context, filter *organization.ListTeamMembersFilter) (*pagination.PageResponse[*organization.TeamMember], error) {
	return pagination.NewPageResponse([]*organization.TeamMember{}, 0, &filter.PaginationParams), nil
}

func (m *MockService) IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error) {
	return false, nil
}

func (m *MockService) InviteMember(ctx context.Context, orgID xid.ID, req *organization.InviteMemberRequest, inviterUserID xid.ID) (*organization.Invitation, error) {
	if m.InviteMemberError != nil {
		return nil, m.InviteMemberError
	}
	return nil, nil
}

func (m *MockService) FindInvitationByID(ctx context.Context, id xid.ID) (*organization.Invitation, error) {
	return nil, organization.InvitationNotFound()
}

func (m *MockService) FindInvitationByToken(ctx context.Context, token string) (*organization.Invitation, error) {
	return nil, organization.InvitationNotFound()
}

func (m *MockService) ListInvitations(ctx context.Context, filter *organization.ListInvitationsFilter) (*pagination.PageResponse[*organization.Invitation], error) {
	return pagination.NewPageResponse([]*organization.Invitation{}, 0, &filter.PaginationParams), nil
}

func (m *MockService) AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*organization.Member, error) {
	if m.AcceptInvitationError != nil {
		return nil, m.AcceptInvitationError
	}
	return nil, nil
}

func (m *MockService) DeclineInvitation(ctx context.Context, token string) error {
	return nil
}

func (m *MockService) CancelInvitation(ctx context.Context, id, cancellerUserID xid.ID) error {
	return nil
}

func (m *MockService) ResendInvitation(ctx context.Context, id, resenderUserID xid.ID) (*organization.Invitation, error) {
	return nil, nil
}

func (m *MockService) CleanupExpiredInvitations(ctx context.Context) (int, error) {
	return 0, nil
}

// Type assertion to ensure MockService implements CompositeOrganizationService
var _ organization.CompositeOrganizationService = (*MockService)(nil)
