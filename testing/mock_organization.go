package testing

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// MockOrganizationService implements organization service methods for testing.
type MockOrganizationService struct {
	mock *Mock
}

// CreateOrganization creates a new organization.
func (s *MockOrganizationService) CreateOrganization(ctx context.Context, req *organization.CreateOrganizationRequest, creatorUserID, appID, environmentID xid.ID) (*organization.Organization, error) {
	s.mock.mu.Lock()
	defer s.mock.mu.Unlock()

	org := &schema.Organization{
		ID:            xid.New(),
		AppID:         appID,
		EnvironmentID: environmentID,
		Name:          req.Name,
		Slug:          req.Slug,
		Metadata:      map[string]any{},
	}

	s.mock.orgs[org.ID] = org

	// Add creator as owner
	member := &schema.OrganizationMember{
		ID:             xid.New(),
		OrganizationID: org.ID,
		UserID:         creatorUserID,
		Role:           "owner",
		Status:         "active",
	}
	s.mock.members[org.ID] = append(s.mock.members[org.ID], member)

	return organization.FromSchemaOrganization(org), nil
}

// FindOrganizationByID retrieves an organization by ID.
func (s *MockOrganizationService) FindOrganizationByID(ctx context.Context, id xid.ID) (*organization.Organization, error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	org, ok := s.mock.orgs[id]
	if !ok {
		return nil, fmt.Errorf("organization not found: %s", id)
	}

	return organization.FromSchemaOrganization(org), nil
}

// GetByID is an alias for FindOrganizationByID for compatibility.
func (s *MockOrganizationService) GetByID(ctx context.Context, id xid.ID) (*organization.Organization, error) {
	return s.FindOrganizationByID(ctx, id)
}

// FindOrganizationBySlug retrieves an organization by slug.
func (s *MockOrganizationService) FindOrganizationBySlug(ctx context.Context, appID, environmentID xid.ID, slug string) (*organization.Organization, error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	for _, org := range s.mock.orgs {
		if org.AppID == appID && org.EnvironmentID == environmentID && org.Slug == slug {
			return organization.FromSchemaOrganization(org), nil
		}
	}

	return nil, fmt.Errorf("organization not found with slug: %s", slug)
}

// GetBySlug is an alias for FindOrganizationBySlug for compatibility.
func (s *MockOrganizationService) GetBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	// Use default app and env
	return s.FindOrganizationBySlug(ctx, s.mock.defaultApp.ID, s.mock.defaultEnv.ID, slug)
}

// ListOrganizations lists organizations with pagination.
func (s *MockOrganizationService) ListOrganizations(ctx context.Context, filter *organization.ListOrganizationsFilter) (*pagination.PageResponse[*organization.Organization], error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	var orgs []*organization.Organization
	for _, org := range s.mock.orgs {
		if !filter.AppID.IsNil() && org.AppID != filter.AppID {
			continue
		}

		if !filter.EnvironmentID.IsNil() && org.EnvironmentID != filter.EnvironmentID {
			continue
		}

		orgs = append(orgs, organization.FromSchemaOrganization(org))
	}

	return &pagination.PageResponse[*organization.Organization]{
		Data: orgs,
		Pagination: &pagination.PageMeta{
			Total:       int64(len(orgs)),
			Limit:       len(orgs),
			Offset:      0,
			CurrentPage: 1,
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
	}, nil
}

// AddMember adds a user as a member to an organization.
func (s *MockOrganizationService) AddMember(ctx context.Context, orgID, userID xid.ID, role string) (*organization.Member, error) {
	s.mock.mu.Lock()
	defer s.mock.mu.Unlock()

	// Verify org exists
	if _, ok := s.mock.orgs[orgID]; !ok {
		return nil, fmt.Errorf("organization not found: %s", orgID)
	}

	// Verify user exists
	if _, ok := s.mock.users[userID]; !ok {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	member := &schema.OrganizationMember{
		ID:             xid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		Status:         "active",
	}

	s.mock.members[orgID] = append(s.mock.members[orgID], member)

	return organization.FromSchemaMember(member), nil
}

// ListMembers lists members of an organization.
func (s *MockOrganizationService) ListMembers(ctx context.Context, filter *organization.ListMembersFilter) (*pagination.PageResponse[*organization.Member], error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	members, ok := s.mock.members[filter.OrganizationID]
	if !ok {
		return &pagination.PageResponse[*organization.Member]{
			Data: []*organization.Member{},
			Pagination: &pagination.PageMeta{
				Total:       0,
				Limit:       0,
				Offset:      0,
				CurrentPage: 1,
				TotalPages:  1,
				HasNext:     false,
				HasPrev:     false,
			},
		}, nil
	}

	var result []*organization.Member
	for _, member := range members {
		result = append(result, organization.FromSchemaMember(member))
	}

	return &pagination.PageResponse[*organization.Member]{
		Data: result,
		Pagination: &pagination.PageMeta{
			Total:       int64(len(result)),
			Limit:       len(result),
			Offset:      0,
			CurrentPage: 1,
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
	}, nil
}

// GetMembers is an alias for ListMembers for compatibility.
func (s *MockOrganizationService) GetMembers(ctx context.Context, orgID xid.ID) (*pagination.PageResponse[*organization.Member], error) {
	return s.ListMembers(ctx, &organization.ListMembersFilter{
		OrganizationID: orgID,
	})
}

// GetUserMemberships retrieves all memberships for a user.
func (s *MockOrganizationService) GetUserMemberships(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*organization.Member], error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	var result []*organization.Member

	for _, members := range s.mock.members {
		for _, member := range members {
			if member.UserID == userID {
				result = append(result, organization.FromSchemaMember(member))
			}
		}
	}

	return &pagination.PageResponse[*organization.Member]{
		Data: result,
		Pagination: &pagination.PageMeta{
			Total:       int64(len(result)),
			Limit:       len(result),
			Offset:      0,
			CurrentPage: 1,
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
	}, nil
}

// GetUserOrganizations retrieves all organizations a user is a member of.
func (s *MockOrganizationService) GetUserOrganizations(ctx context.Context, userID xid.ID) ([]*organization.Organization, error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	var result []*organization.Organization

	orgSet := make(map[xid.ID]bool)

	for orgID, members := range s.mock.members {
		for _, member := range members {
			if member.UserID == userID && !orgSet[orgID] {
				if org, ok := s.mock.orgs[orgID]; ok {
					result = append(result, organization.FromSchemaOrganization(org))
					orgSet[orgID] = true
				}

				break
			}
		}
	}

	return result, nil
}
