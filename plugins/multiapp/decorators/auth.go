package decorators

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
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
func (s *MultiTenantAuthService) SignIn(ctx context.Context, req *auth.SignInRequest) (*responses.AuthResponse, error) {
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
func (s *MultiTenantAuthService) SignUp(ctx context.Context, req *auth.SignUpRequest) (*responses.AuthResponse, error) {
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
func (s *MultiTenantAuthService) CreateSessionForUser(ctx context.Context, u *user.User, remember bool, ipAddress, userAgent string) (*responses.AuthResponse, error) {
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
func (s *MultiTenantAuthService) GetSession(ctx context.Context, token string) (*responses.AuthResponse, error) {
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

// RefreshSession refreshes an access token using a refresh token within app context
func (s *MultiTenantAuthService) RefreshSession(ctx context.Context, refreshToken string) (*responses.RefreshSessionResponse, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, errs.NotFound("app ID not found in context")
	}

	// Refresh session using core service
	response, err := s.authService.RefreshSession(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Verify user is a member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, response.User.ID)
	if err != nil {
		return nil, errs.InternalServerError("failed to check organization membership", err)
	}
	if member.Status != app.MemberStatusActive {
		return nil, fmt.Errorf("user membership is not active")
	}

	return response, nil
}

// RequestPasswordReset initiates a password reset flow
func (s *MultiTenantAuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	return s.authService.RequestPasswordReset(ctx, email)
}

// ResetPassword completes the password reset flow
func (s *MultiTenantAuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	return s.authService.ResetPassword(ctx, token, newPassword)
}

// ValidateResetToken checks if a reset token is valid
func (s *MultiTenantAuthService) ValidateResetToken(ctx context.Context, token string) (bool, error) {
	return s.authService.ValidateResetToken(ctx, token)
}

// ChangePassword changes a user's password after verifying the old password
func (s *MultiTenantAuthService) ChangePassword(ctx context.Context, userID xid.ID, oldPassword, newPassword string) error {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return errs.NotFound("app ID not found in context")
	}

	// Verify user is a member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, userID)
	if err != nil {
		return errs.InternalServerError("failed to check organization membership", err)
	}
	if member.Status != app.MemberStatusActive {
		return fmt.Errorf("user membership is not active")
	}

	return s.authService.ChangePassword(ctx, userID, oldPassword, newPassword)
}

// RequestEmailChange initiates an email change flow
func (s *MultiTenantAuthService) RequestEmailChange(ctx context.Context, userID xid.ID, newEmail string) (string, error) {
	// Get organization from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return "", errs.NotFound("app ID not found in context")
	}

	// Verify user is a member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, userID)
	if err != nil {
		return "", errs.InternalServerError("failed to check organization membership", err)
	}
	if member.Status != app.MemberStatusActive {
		return "", fmt.Errorf("user membership is not active")
	}

	return s.authService.RequestEmailChange(ctx, userID, newEmail)
}

// ConfirmEmailChange completes the email change flow
func (s *MultiTenantAuthService) ConfirmEmailChange(ctx context.Context, token string) error {
	return s.authService.ConfirmEmailChange(ctx, token)
}

// ValidateEmailChangeToken checks if an email change token is valid
func (s *MultiTenantAuthService) ValidateEmailChangeToken(ctx context.Context, token string) (bool, error) {
	return s.authService.ValidateEmailChangeToken(ctx, token)
}
