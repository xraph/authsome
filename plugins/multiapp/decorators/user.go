package decorators

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	coreapp "github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
)

// MultiTenantUserService decorates the user service with multi-tenancy support.
type MultiTenantUserService struct {
	userService user.ServiceInterface
	appService  *coreapp.ServiceImpl
}

// CountUsers counts users in the specified app.
func (s *MultiTenantUserService) CountUsers(ctx context.Context, filter *user.CountUsersFilter) (int, error) {
	return s.userService.CountUsers(ctx, filter)
}

// NewMultiTenantUserService creates a new multi-tenant user service decorator.
func NewMultiTenantUserService(userService user.ServiceInterface, appService *coreapp.ServiceImpl) *MultiTenantUserService {
	return &MultiTenantUserService{
		userService: userService,
		appService:  appService,
	}
}

// Create creates a new user within an app context.
func (s *MultiTenantUserService) Create(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)

	// Count users in the app (if appID is available)
	var (
		usersCount int
		err        error
	)

	if !appID.IsNil() {
		usersCount, err = s.userService.CountUsers(ctx, &user.CountUsersFilter{
			AppID: appID,
		})
		if err != nil {
			return nil, err
		}
	}

	if usersCount == 0 {
		// No users exist yet - this is the first user (system owner)
		// Create user without organization membership
		// The post-creation hook will set up their organization
		// newUser, err := s.userService.Create(ctx, req)
		// if err != nil {
		// 	return nil, err
		// }
	}

	// Special case: First user creation (system owner)
	// If no app context is provided, check if this is the very first user
	// The first user will create the platform organization via hooks
	if appID.IsNil() {
		// Check if there are any organizations yet
		response, err := s.appService.ListApps(ctx, &coreapp.ListAppsFilter{
			PaginationParams: pagination.PaginationParams{
				Page:  1,
				Limit: 1,
			},
		})
		if err != nil || len(response.Data) == 0 {
			// No organizations exist yet - this is the first user (system owner)
			// Create user without organization membership
			// The post-creation hook will set up their organization
			newUser, err := s.userService.Create(ctx, req)
			if err != nil {
				return nil, err
			}

			return newUser, nil
		}

		// Organizations exist but no context provided - error
		return nil, errs.BadRequest("app context required")
	}

	// Validate organization exists
	_, err = s.appService.App.FindAppByID(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization: %w", err)
	}

	// Create user using original service
	newUser, err := s.userService.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add user as member of the organization
	_, err = s.appService.Member.CreateMember(ctx, &coreapp.Member{
		AppID:  appID,
		UserID: newUser.ID,
		Role:   coreapp.MemberRoleMember,
		Status: coreapp.MemberStatusActive,
	})
	if err != nil {
		// TODO: Consider rollback strategy
		return nil, fmt.Errorf("failed to add user to organization: %w", err)
	}

	return newUser, nil
}

// FindByID finds a user by ID within app context.
func (s *MultiTenantUserService) FindByID(ctx context.Context, id xid.ID) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return nil, errs.BadRequest("app context required")
	}

	// Find user using original service
	foundUser, err := s.userService.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, foundUser.ID)
	if err != nil {
		return nil, user.UserNotFound(foundUser.ID.String())
	}

	if member.Status != coreapp.MemberStatusActive {
		return nil, user.UserNotFound(foundUser.ID.String())
	}

	return foundUser, nil
}

// FindByEmail finds a user by email within app context.
func (s *MultiTenantUserService) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)

	// Special case: First user lookup (platform owner login)
	// If no app context is provided, check if this user is the platform owner
	if appID.IsNil() {
		// Find user using original service
		foundUser, err := s.userService.FindByEmail(ctx, email)
		if err != nil {
			return nil, err
		}

		// Check if this user is a member of the platform organization
		// Platform owners can login without explicit org context
		platformOrg, err := s.appService.FindAppBySlug(ctx, "platform")
		if err != nil {
			// Platform org doesn't exist yet - might be during first user setup
			return foundUser, nil
		}

		// Check if user is platform member
		member, err := s.appService.Member.FindMember(ctx, platformOrg.ID, foundUser.ID)
		if err != nil || member.Status != coreapp.MemberStatusActive {
			// User is not a platform member - require org context
			return nil, errs.BadRequest("app context required")
		}

		return foundUser, nil
	}

	// Find user using original service
	foundUser, err := s.userService.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, foundUser.ID)
	if err != nil {
		return nil, user.UserNotFound(email)
	}

	if member.Status != coreapp.MemberStatusActive {
		return nil, user.UserNotFound(email)
	}

	return foundUser, nil
}

// FindByAppAndEmail finds a user by app and email.
func (s *MultiTenantUserService) FindByAppAndEmail(ctx context.Context, appID xid.ID, email string) (*user.User, error) {
	// Use the app-scoped search directly
	foundUser, err := s.userService.FindByAppAndEmail(ctx, appID, email)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, foundUser.ID)
	if err != nil {
		return nil, user.UserNotFound(email)
	}

	if member.Status != coreapp.MemberStatusActive {
		return nil, user.UserNotFound(email)
	}

	return foundUser, nil
}

// FindByUsername finds a user by username within app context.
func (s *MultiTenantUserService) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return nil, errs.BadRequest("app context required")
	}

	// Find user using original service
	foundUser, err := s.userService.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, foundUser.ID)
	if err != nil {
		return nil, user.UserNotFound(username)
	}

	if member.Status != coreapp.MemberStatusActive {
		return nil, user.UserNotFound(username)
	}

	return foundUser, nil
}

// Update updates a user within app context.
func (s *MultiTenantUserService) Update(ctx context.Context, u *user.User, req *user.UpdateUserRequest) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)

	// Special case: First user update (e.g., email verification during signup)
	// If no app context and no organizations exist yet, allow the update
	if appID.IsNil() {
		// Check if there are any organizations yet
		response, err := s.appService.ListApps(ctx, &coreapp.ListAppsFilter{
			PaginationParams: pagination.PaginationParams{
				Page:  1,
				Limit: 1,
			},
		})
		if err != nil || len(response.Data) == 0 {
			// No organizations exist yet - this is the first user
			// Allow update without organization check (e.g., for email verification)
			return s.userService.Update(ctx, u, req)
		}

		// Organizations exist but no context provided - error
		return nil, errs.BadRequest("app context required")
	}

	// Check if user is member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, u.ID)
	if err != nil {
		return nil, user.UserNotFound(u.ID.String())
	}

	if member.Status != coreapp.MemberStatusActive {
		return nil, user.UserNotFound(u.ID.String())
	}

	// Update user using original service
	return s.userService.Update(ctx, u, req)
}

// Delete deletes a user within app context.
func (s *MultiTenantUserService) Delete(ctx context.Context, id xid.ID) error {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return errs.BadRequest("app context required")
	}

	// Check if user is member of the organization
	member, err := s.appService.Member.FindMember(ctx, appID, id)
	if err != nil {
		return user.UserNotFound(id.String())
	}

	if member.Status != coreapp.MemberStatusActive {
		return user.UserNotFound(id.String())
	}

	// Note: In a full implementation, we would want to remove user from all apps/organizations
	// before deleting. For now, we rely on database cascade delete or handle at repository level.
	// TODO: Add DeleteMembersByUserID to core/app MemberService if needed

	// Delete user using original service
	return s.userService.Delete(ctx, id)
}

// GetAppContext gets the organization ID from context.
func (s *MultiTenantUserService) GetAppContext(ctx context.Context) xid.ID {
	return s.getAppFromContext(ctx)
}

// SetOrganizationContext sets the organization ID in context.
func (s *MultiTenantUserService) SetOrganizationContext(ctx context.Context, appID string) context.Context {
	id, err := xid.FromString(appID)
	if err != nil {
		return ctx
	}

	return contexts.SetAppID(ctx, id)
}

// ListUsers lists users within app context with search support.
func (s *MultiTenantUserService) ListUsers(ctx context.Context, filter *user.ListUsersFilter) (*pagination.PageResponse[*user.User], error) {
	// Get app context if not already in filter
	if filter.AppID.IsNil() {
		appID := s.getAppFromContext(ctx)
		if appID.IsNil() {
			return nil, errs.BadRequest("app context required")
		}

		filter.AppID = appID
	}

	// List users using original service (with search if provided)
	return s.userService.ListUsers(ctx, filter)
}

// getAppFromContext extracts organization ID from context.
func (s *MultiTenantUserService) getAppFromContext(ctx context.Context) xid.ID {
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return xid.NilID()
	}

	return appID
}

// UpdatePassword updates a user's password directly.
func (s *MultiTenantUserService) UpdatePassword(ctx context.Context, userID xid.ID, hashedPassword string) error {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if !appID.IsNil() {
		// Check if user is member of the organization
		member, err := s.appService.Member.FindMember(ctx, appID, userID)
		if err != nil {
			return user.UserNotFound(userID.String())
		}

		if member.Status != coreapp.MemberStatusActive {
			return user.UserNotFound(userID.String())
		}
	}

	// Update password using original service
	return s.userService.UpdatePassword(ctx, userID, hashedPassword)
}

// SetHookRegistry sets the hook registry for lifecycle events.
func (s *MultiTenantUserService) SetHookRegistry(registry any) {
	s.userService.SetHookRegistry(registry)
}

// GetHookRegistry returns the hook registry.
func (s *MultiTenantUserService) GetHookRegistry() any {
	return s.userService.GetHookRegistry()
}

// SetVerificationRepo sets the verification repository for password resets.
func (s *MultiTenantUserService) SetVerificationRepo(repo any) {
	s.userService.SetVerificationRepo(repo)
}

// GetVerificationRepo returns the verification repository.
func (s *MultiTenantUserService) GetVerificationRepo() any {
	return s.userService.GetVerificationRepo()
}
