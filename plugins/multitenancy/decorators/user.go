package decorators

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/interfaces"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/multitenancy/app"
	"github.com/xraph/authsome/types"
)

// MultiTenantUserService decorates the user service with multi-tenancy support
type MultiTenantUserService struct {
	userService user.ServiceInterface
	appService  *app.Service
}

func (s *MultiTenantUserService) Count(ctx context.Context) (int, error) {
	return s.userService.Count(ctx)
}

// NewMultiTenantUserService creates a new multi-tenant user service decorator
func NewMultiTenantUserService(userService user.ServiceInterface, appService *app.Service) *MultiTenantUserService {
	return &MultiTenantUserService{
		userService: userService,
		appService:  appService,
	}
}

// Create creates a new user within an app context
func (s *MultiTenantUserService) Create(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)

	usersCount, err := s.userService.Count(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[MultiTenancy] Users count: %d, appID: %s\n", usersCount, appID.String())

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
		orgs, err := s.appService.ListApps(ctx, 1, 0)
		if err != nil || len(orgs) == 0 {
			// No organizations exist yet - this is the first user (system owner)
			// Create user without organization membership
			// The post-creation hook will set up their organization
			newUser, err := s.userService.Create(ctx, req)
			if err != nil {
				return nil, err
			}

			fmt.Printf("[MultiTenancy] First user created (system owner): %s\n", newUser.Email)
			return newUser, nil
		}

		// Organizations exist but no context provided - error
		return nil, fmt.Errorf("app context required")
	}

	// Validate organization exists
	_, err = s.appService.GetApp(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization: %w", err)
	}

	// Create user using original service
	newUser, err := s.userService.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add user as member of the organization
	_, err = s.appService.AddMember(ctx, appID, newUser.ID, app.RoleMember)
	if err != nil {
		// TODO: Consider rollback strategy
		return nil, fmt.Errorf("failed to add user to organization: %w", err)
	}

	return newUser, nil
}

// FindByID finds a user by ID within app context
func (s *MultiTenantUserService) FindByID(ctx context.Context, id xid.ID) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return nil, fmt.Errorf("app context required")
	}

	// Find user using original service
	foundUser, err := s.userService.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	isMember, err := s.appService.IsUserMember(ctx, appID, foundUser.ID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return foundUser, nil
}

// FindByEmail finds a user by email within app context
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
		platformOrg, err := s.appService.GetAppBySlug(ctx, "platform")
		if err != nil {
			// Platform org doesn't exist yet - might be during first user setup
			fmt.Printf("[MultiTenancy] No platform org found, allowing user lookup: %s\n", email)
			return foundUser, nil
		}

		// Check if user is platform member
		isMember, err := s.appService.IsMember(ctx, platformOrg.ID, foundUser.ID)
		if err != nil || !isMember {
			// User is not a platform member - require org context
			return nil, fmt.Errorf("app context required")
		}

		fmt.Printf("[MultiTenancy] Platform user login without org context: %s\n", email)
		return foundUser, nil
	}

	// Find user using original service
	foundUser, err := s.userService.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	isMember, err := s.appService.IsUserMember(ctx, appID, foundUser.ID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return foundUser, nil
}

// FindByUsername finds a user by username within app context
func (s *MultiTenantUserService) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return nil, fmt.Errorf("app context required")
	}

	// Find user using original service
	foundUser, err := s.userService.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	isMember, err := s.appService.IsUserMember(ctx, appID, foundUser.ID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return foundUser, nil
}

// Update updates a user within app context
func (s *MultiTenantUserService) Update(ctx context.Context, u *user.User, req *user.UpdateUserRequest) (*user.User, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)

	// Special case: First user update (e.g., email verification during signup)
	// If no app context and no organizations exist yet, allow the update
	if appID.IsNil() {
		// Check if there are any organizations yet
		orgs, err := s.appService.ListApps(ctx, 1, 0)
		if err != nil || len(orgs) == 0 {
			// No organizations exist yet - this is the first user
			// Allow update without organization check (e.g., for email verification)
			fmt.Printf("[MultiTenancy] First user update allowed without org context: %s\n", u.Email)
			return s.userService.Update(ctx, u, req)
		}

		// Organizations exist but no context provided - error
		return nil, fmt.Errorf("app context required")
	}

	// Check if user is member of the organization
	isMember, err := s.appService.IsUserMember(ctx, appID, u.ID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, types.ErrUserNotFound
	}

	// Update user using original service
	return s.userService.Update(ctx, u, req)
}

// Delete deletes a user within app context
func (s *MultiTenantUserService) Delete(ctx context.Context, id xid.ID) error {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return fmt.Errorf("app context required")
	}

	// Check if user is member of the organization
	isMember, err := s.appService.IsUserMember(ctx, appID, id)
	if err != nil {
		return err
	}
	if !isMember {
		return types.ErrUserNotFound
	}

	// Remove user from organization first
	err = s.appService.RemoveUserFromAllApps(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to remove user from organizations: %w", err)
	}

	// Delete user using original service
	return s.userService.Delete(ctx, id)
}

// List lists users within app context
func (s *MultiTenantUserService) List(ctx context.Context, opts types.PaginationOptions) ([]*user.User, int, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return nil, 0, fmt.Errorf("app context required")
	}

	// Calculate offset from page and page size
	offset := (opts.Page - 1) * opts.PageSize

	// Get organization members
	members, err := s.appService.ListMembers(ctx, appID, opts.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organization members: %w", err)
	}

	// Get user details for each member
	users := make([]*user.User, 0, len(members))
	for _, member := range members {
		if member.UserID.IsNil() {
			continue // Skip invalid user IDs
		}

		// Use the original service to get user details (bypass organization check)
		u, err := s.userService.FindByID(ctx, member.UserID)
		if err != nil {
			continue // Skip users that can't be found
		}

		users = append(users, u)
	}

	return users, len(users), nil
}

// GetAppContext gets the organization ID from context
func (s *MultiTenantUserService) GetAppContext(ctx context.Context) xid.ID {
	return s.getAppFromContext(ctx)
}

// SetOrganizationContext sets the organization ID in context
func (s *MultiTenantUserService) SetOrganizationContext(ctx context.Context, appID string) context.Context {
	return context.WithValue(ctx, interfaces.OrganizationContextKey, appID)
}

// Search searches for users within app context
func (s *MultiTenantUserService) Search(ctx context.Context, query string, opts types.PaginationOptions) ([]*user.User, int, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return nil, 0, fmt.Errorf("app context required")
	}

	// Search users using original service
	users, _, err := s.userService.Search(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}

	// Filter users by organization membership
	filteredUsers := make([]*user.User, 0)
	for _, u := range users {
		isMember, err := s.appService.IsUserMember(ctx, appID, u.ID)
		if err != nil {
			continue // Skip on error
		}
		if isMember {
			filteredUsers = append(filteredUsers, u)
		}
	}

	return filteredUsers, len(filteredUsers), nil
}

// CountCreatedToday counts users created today within app context
func (s *MultiTenantUserService) CountCreatedToday(ctx context.Context) (int, error) {
	// Get app context
	appID := s.getAppFromContext(ctx)
	if appID.IsNil() {
		return 0, fmt.Errorf("app context required")
	}

	// For now, delegate to the original service
	// In a full implementation, this should filter by organization
	return s.userService.CountCreatedToday(ctx)
}

// getAppFromContext extracts organization ID from context
func (s *MultiTenantUserService) getAppFromContext(ctx context.Context) xid.ID {
	appID, err := interfaces.GetAppID(ctx)
	if err != nil {
		return xid.NilID()
	}
	return appID
}
