package decorators

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/interfaces"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/authsome/types"
)

// Ensure MultiTenantAuthService implements auth.ServiceInterface
var _ auth.ServiceInterface = (*MultiTenantAuthService)(nil)

// MultiTenantAuthService decorates the core auth service with multi-tenancy capabilities
type MultiTenantAuthService struct {
	authService auth.ServiceInterface
	orgService  *organization.Service
}

// NewMultiTenantAuthService creates a new multi-tenant auth service decorator
func NewMultiTenantAuthService(authService auth.ServiceInterface, orgService *organization.Service) *MultiTenantAuthService {
	return &MultiTenantAuthService{
		authService: authService,
		orgService:  orgService,
	}
}

// SignIn authenticates a user within an organization context
func (s *MultiTenantAuthService) SignIn(ctx context.Context, req *auth.SignInRequest) (*auth.AuthResponse, error) {
	// Get organization from context
	orgID, err := interfaces.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	// Verify organization exists
	_, err = s.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	// Perform authentication
	response, err := s.authService.SignIn(ctx, req)
	if err != nil {
		return nil, err
	}

	// Verify user is a member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, response.User.ID)
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
	orgID, err := interfaces.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	// Verify organization exists
	_, err = s.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	// Perform user registration
	response, err := s.authService.SignUp(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add user to organization as a member
	_, err = s.orgService.AddMember(ctx, orgID, response.User.ID, organization.RoleMember)
	if err != nil {
		// TODO: Consider rolling back user creation if adding to org fails
		return nil, fmt.Errorf("failed to add user to organization: %w", err)
	}

	return response, nil
}

// SignOut signs out a user from the current session
func (s *MultiTenantAuthService) SignOut(ctx context.Context, req *auth.SignOutRequest) error {
	// Perform sign out
	return s.authService.SignOut(ctx, req)
}

// CheckCredentials validates user credentials within organization context
func (s *MultiTenantAuthService) CheckCredentials(ctx context.Context, email, password string) (*user.User, error) {
	// Get organization from context
	orgID, err := interfaces.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	// Check credentials using core service
	u, err := s.authService.CheckCredentials(ctx, email, password)
	if err != nil {
		return nil, err
	}

	// Verify user is a member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check organization membership: %w", err)
	}

	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return u, nil
}

// CreateSessionForUser creates a session for a user within organization context
func (s *MultiTenantAuthService) CreateSessionForUser(ctx context.Context, u *user.User, remember bool, ipAddress, userAgent string) (*auth.AuthResponse, error) {
	// Get organization from context
	orgID, err := interfaces.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	// Verify user is a member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, u.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check organization membership: %w", err)
	}

	if !isMember {
		return nil, types.ErrUserNotFound
	}

	// Create session using core service
	return s.authService.CreateSessionForUser(ctx, u, remember, ipAddress, userAgent)
}

// GetSession retrieves a session within organization context
func (s *MultiTenantAuthService) GetSession(ctx context.Context, token string) (*auth.AuthResponse, error) {
	// Get organization from context
	orgID, err := interfaces.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	// Get session using core service
	response, err := s.authService.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}

	// Verify user belongs to the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, response.User.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check organization membership: %w", err)
	}

	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return response, nil
}

// UpdateUser updates a user within organization context
func (s *MultiTenantAuthService) UpdateUser(ctx context.Context, id xid.ID, req *user.UpdateUserRequest) (*user.User, error) {
	// Get organization from context
	orgID, err := interfaces.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	// Verify user is a member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, id)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, types.ErrUserNotFound
	}

	// Update user using core service
	return s.authService.UpdateUser(ctx, id, req)
}
