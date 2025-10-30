package decorators

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/interfaces"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
	"github.com/xraph/authsome/types"
)

// MultiTenantUserService decorates the user service with multi-tenancy support
type MultiTenantUserService struct {
	userService user.ServiceInterface
	orgService  *organization.Service
}

// NewMultiTenantUserService creates a new multi-tenant user service decorator
func NewMultiTenantUserService(userService user.ServiceInterface, orgService *organization.Service) *MultiTenantUserService {
	return &MultiTenantUserService{
		userService: userService,
		orgService:  orgService,
	}
}

// Create creates a new user within an organization context
func (s *MultiTenantUserService) Create(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	// Get organization context
	orgID := s.getOrganizationFromContext(ctx)
	if orgID == "" {
		return nil, fmt.Errorf("organization context required")
	}

	// Validate organization exists
	_, err := s.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization: %w", err)
	}

	// Create user using original service
	newUser, err := s.userService.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add user as member of the organization
	_, err = s.orgService.AddMember(ctx, orgID, newUser.ID.String(), organization.RoleMember)
	if err != nil {
		// TODO: Consider rollback strategy
		return nil, fmt.Errorf("failed to add user to organization: %w", err)
	}

	return newUser, nil
}

// FindByID finds a user by ID within organization context
func (s *MultiTenantUserService) FindByID(ctx context.Context, id xid.ID) (*user.User, error) {
	// Get organization context
	orgID := s.getOrganizationFromContext(ctx)
	if orgID == "" {
		return nil, fmt.Errorf("organization context required")
	}

	// Find user using original service
	foundUser, err := s.userService.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, foundUser.ID.String())
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return foundUser, nil
}

// FindByEmail finds a user by email within organization context
func (s *MultiTenantUserService) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	// Get organization context
	orgID := s.getOrganizationFromContext(ctx)
	if orgID == "" {
		return nil, fmt.Errorf("organization context required")
	}

	// Find user using original service
	foundUser, err := s.userService.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, foundUser.ID.String())
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return foundUser, nil
}

// FindByUsername finds a user by username within organization context
func (s *MultiTenantUserService) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	// Get organization context
	orgID := s.getOrganizationFromContext(ctx)
	if orgID == "" {
		return nil, fmt.Errorf("organization context required")
	}

	// Find user using original service
	foundUser, err := s.userService.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Check if user is member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, foundUser.ID.String())
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, types.ErrUserNotFound
	}

	return foundUser, nil
}

// Update updates a user within organization context
func (s *MultiTenantUserService) Update(ctx context.Context, u *user.User, req *user.UpdateUserRequest) (*user.User, error) {
	// Get organization context
	orgID := s.getOrganizationFromContext(ctx)
	if orgID == "" {
		return nil, fmt.Errorf("organization context required")
	}

	// Check if user is member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, u.ID.String())
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, types.ErrUserNotFound
	}

	// Update user using original service
	return s.userService.Update(ctx, u, req)
}

// Delete deletes a user within organization context
func (s *MultiTenantUserService) Delete(ctx context.Context, id xid.ID) error {
	// Get organization context
	orgID := s.getOrganizationFromContext(ctx)
	if orgID == "" {
		return fmt.Errorf("organization context required")
	}

	// Check if user is member of the organization
	isMember, err := s.orgService.IsUserMember(ctx, orgID, id.String())
	if err != nil {
		return err
	}
	if !isMember {
		return types.ErrUserNotFound
	}

	// Remove user from organization first
	err = s.orgService.RemoveUserFromAllOrganizations(ctx, id.String())
	if err != nil {
		return fmt.Errorf("failed to remove user from organizations: %w", err)
	}

	// Delete user using original service
	return s.userService.Delete(ctx, id)
}

// List lists users within organization context
func (s *MultiTenantUserService) List(ctx context.Context, opts types.PaginationOptions) ([]*user.User, int, error) {
	// Get organization context
	orgID := s.getOrganizationFromContext(ctx)
	if orgID == "" {
		return nil, 0, fmt.Errorf("organization context required")
	}

	// Calculate offset from page and page size
	offset := (opts.Page - 1) * opts.PageSize

	// Get organization members
	members, err := s.orgService.ListMembers(ctx, orgID, opts.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organization members: %w", err)
	}

	// Get user details for each member
	users := make([]*user.User, 0, len(members))
	for _, member := range members {
		userID, err := xid.FromString(member.UserID)
		if err != nil {
			continue // Skip invalid user IDs
		}

		// Use the original service to get user details (bypass organization check)
		u, err := s.userService.FindByID(ctx, userID)
		if err != nil {
			continue // Skip users that can't be found
		}

		users = append(users, u)
	}

	return users, len(users), nil
}

// GetOrganizationContext gets the organization ID from context
func (s *MultiTenantUserService) GetOrganizationContext(ctx context.Context) string {
	return s.getOrganizationFromContext(ctx)
}

// SetOrganizationContext sets the organization ID in context
func (s *MultiTenantUserService) SetOrganizationContext(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, interfaces.OrganizationContextKey, orgID)
}

// getOrganizationFromContext extracts organization ID from context
func (s *MultiTenantUserService) getOrganizationFromContext(ctx context.Context) string {
	if orgID, ok := ctx.Value(interfaces.OrganizationContextKey).(string); ok {
		return orgID
	}
	return ""
}
