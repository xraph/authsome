package decorators

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/interfaces"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/authsome/types"
)

// MultiTenantAuthService decorates the core auth service with multi-tenancy capabilities
type MultiTenantAuthService struct {
	authService *auth.Service
	orgService  *organization.Service
}

// NewMultiTenantAuthService creates a new multi-tenant auth service decorator
func NewMultiTenantAuthService(authService *auth.Service, orgService *organization.Service) *MultiTenantAuthService {
	return &MultiTenantAuthService{
		authService: authService,
		orgService:  orgService,
	}
}

// SignIn authenticates a user within an organization context
func (s *MultiTenantAuthService) SignIn(ctx context.Context, req *auth.SignInRequest) (*auth.AuthResponse, error) {
	// Get organization from context
	orgID, ok := ctx.Value(interfaces.OrganizationContextKey).(string)
	if !ok {
		return nil, fmt.Errorf("organization context not found")
	}

	// Verify organization exists
	_, err := s.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	// Perform authentication
	response, err := s.authService.SignIn(ctx, req)
	if err != nil {
		return nil, err
	}

	// Verify user is a member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, response.User.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to check organization membership: %w", err)
	}

	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return response, nil
}

// SignUp registers a new user and adds them to the organization
func (s *MultiTenantAuthService) SignUp(ctx context.Context, req *auth.SignUpRequest) (*auth.AuthResponse, error) {
	// Get organization from context
	orgID, ok := ctx.Value(interfaces.OrganizationContextKey).(string)
	if !ok {
		return nil, fmt.Errorf("organization context not found")
	}

	// Verify organization exists
	_, err := s.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	// Perform user registration
	response, err := s.authService.SignUp(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add user to organization as a member
	_, err = s.orgService.AddMember(ctx, orgID, response.User.ID.String(), organization.RoleMember)
	if err != nil {
		// TODO: Consider rolling back user creation if adding to org fails
		return nil, fmt.Errorf("failed to add user to organization: %w", err)
	}

	return response, nil
}

// SignOut signs out a user from the current session
func (s *MultiTenantAuthService) SignOut(ctx context.Context, token string) error {
	// Perform sign out
	return s.authService.SignOut(ctx, &auth.SignOutRequest{Token: token})
}