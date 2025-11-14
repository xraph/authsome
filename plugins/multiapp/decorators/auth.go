package decorators

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
)

// Ensure MultiTenantAuthService implements auth.ServiceInterface
var _ auth.ServiceInterface = (*MultiTenantAuthService)(nil)

// MultiTenantAuthService decorates the core auth service with multi-tenancy capabilities
type MultiTenantAuthService struct {
	authService authsome.AuthService
	appService  *app.ServiceImpl
}

// NewMultiTenantAuthService creates a new multi-tenant auth service decorator
func NewMultiTenantAuthService(authService authsome.AuthService, appService *app.ServiceImpl) *MultiTenantAuthService {
	return &MultiTenantAuthService{
		authService: authService,
		appService:  appService,
	}
}

// SignIn authenticates a user within an app context
func (s *MultiTenantAuthService) SignIn(ctx context.Context, req *auth.SignInRequest) (*auth.AuthResponse, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.NotFound("app ID not found in context")
	}

	// Verify organization exists
	_, err := s.appService.App.FindAppByID(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	// Perform authentication
	response, err := s.authService.SignIn(ctx, req)
	if err != nil {
		return nil, err
	}

	// Verify user is a member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, response.User.ID)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of this app")
	}
	if member.Status != app.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	return response, nil
}

// SignUp registers a new user and adds them to the organization
func (s *MultiTenantAuthService) SignUp(ctx context.Context, req *auth.SignUpRequest) (*auth.AuthResponse, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.NotFound("app ID not found in context")
	}

	// Verify organization exists
	_, err := s.appService.App.FindAppByID(ctx, appID)
	if err != nil {
		return nil, errs.NotFound("app not found")
	}

	// Perform user registration
	response, err := s.authService.SignUp(ctx, req)
	if err != nil {
		return nil, errs.InternalServerError("failed to sign up user", err)
	}

	// Add user to organization as a member
	_, err = s.appService.Member.CreateMember(ctx, &app.Member{
		AppID:  appID,
		UserID: response.User.ID,
		Role:   app.MemberRoleMember,
		Status: app.MemberStatusActive,
	})
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

// CheckCredentials validates user credentials within app context
func (s *MultiTenantAuthService) CheckCredentials(ctx context.Context, email, password string) (*user.User, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.NotFound("app ID not found in context")
	}

	// Check credentials using core service
	u, err := s.authService.CheckCredentials(ctx, email, password)
	if err != nil {
		return nil, errs.InternalServerError("failed to check credentials", err)
	}

	// Verify user is a member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, u.ID)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of this app")
	}
	if member.Status != app.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	return u, nil
}

// CreateSessionForUser creates a session for a user within app context
func (s *MultiTenantAuthService) CreateSessionForUser(ctx context.Context, u *user.User, remember bool, ipAddress, userAgent string) (*auth.AuthResponse, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.NotFound("app ID not found in context")
	}

	// Verify user is a member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, u.ID)
	if err != nil {
		return nil, fmt.Errorf("user is not a member of this app")
	}
	if member.Status != app.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	// Create session using core service
	return s.authService.CreateSessionForUser(ctx, u, remember, ipAddress, userAgent)
}

// GetSession retrieves a session within app context
func (s *MultiTenantAuthService) GetSession(ctx context.Context, token string) (*auth.AuthResponse, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.NotFound("app ID not found in context")
	}

	// Get session using core service
	response, err := s.authService.GetSession(ctx, token)
	if err != nil {
		return nil, errs.InternalServerError("failed to check organization membership", err)
	}

	// Verify user belongs to the organization
	member, err := s.appService.Member.FindMember(ctx, appID, response.User.ID)
	if err != nil {
		return nil, errs.InternalServerError("failed to check organization membership", err)
	}
	if member.Status != app.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	return response, nil
}

// UpdateUser updates a user within app context
func (s *MultiTenantAuthService) UpdateUser(ctx context.Context, id xid.ID, req *user.UpdateUserRequest) (*user.User, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.NotFound("app ID not found in context")
	}

	// Verify user is a member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, id)
	if err != nil {
		return nil, errs.InternalServerError("failed to check organization membership", err)
	}
	if member.Status != app.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	// Update user using core service
	return s.authService.UpdateUser(ctx, id, req)
}
