package providers

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// AUTHSOME SERVICE INTERFACES
// =============================================================================

// AuthsomeUserService defines the interface for the AuthSome user service.
type AuthsomeUserService interface {
	// FindByID finds a user by ID
	FindByID(ctx context.Context, id xid.ID) (AuthsomeUser, error)
}

// AuthsomeMemberService defines the interface for organization member operations.
type AuthsomeMemberService interface {
	// GetUserMemberships returns all organizations a user is a member of
	GetUserMembershipsForUser(ctx context.Context, userID xid.ID) ([]AuthsomeMembership, error)
}

// AuthsomeRBACService defines the interface for RBAC operations.
type AuthsomeRBACService interface {
	// GetUserRoles gets the roles for a user in an organization
	GetUserRoles(ctx context.Context, userID, orgID xid.ID) ([]string, error)

	// GetUserPermissions gets the permissions for a user in an organization
	GetUserPermissions(ctx context.Context, userID, orgID xid.ID) ([]string, error)
}

// =============================================================================
// AUTHSOME USER DATA TYPES
// =============================================================================

// AuthsomeUser represents user data from the core user service.
type AuthsomeUser interface {
	GetID() xid.ID
	GetAppID() xid.ID
	GetEmail() string
	GetName() string
	GetEmailVerified() bool
	GetUsername() string
	GetImage() string
	GetCreatedAt() string
}

// AuthsomeMembership represents a user's membership in an organization.
type AuthsomeMembership interface {
	GetOrganizationID() xid.ID
	GetRole() string
	GetStatus() string
}

// =============================================================================
// AUTHSOME USER ATTRIBUTE PROVIDER
// =============================================================================

// AuthsomeUserAttributeProvider provides user attributes from AuthSome services.
type AuthsomeUserAttributeProvider struct {
	userService   AuthsomeUserService
	memberService AuthsomeMemberService
	rbacService   AuthsomeRBACService
	defaultOrgID  *xid.ID // Optional: default organization for context
}

// AuthsomeUserProviderConfig configures the provider.
type AuthsomeUserProviderConfig struct {
	UserService   AuthsomeUserService
	MemberService AuthsomeMemberService
	RBACService   AuthsomeRBACService
	DefaultOrgID  *xid.ID
}

// NewAuthsomeUserAttributeProvider creates a new AuthSome user attribute provider.
func NewAuthsomeUserAttributeProvider(cfg AuthsomeUserProviderConfig) *AuthsomeUserAttributeProvider {
	return &AuthsomeUserAttributeProvider{
		userService:   cfg.UserService,
		memberService: cfg.MemberService,
		rbacService:   cfg.RBACService,
		defaultOrgID:  cfg.DefaultOrgID,
	}
}

// Name returns the provider name.
func (p *AuthsomeUserAttributeProvider) Name() string {
	return "user"
}

// GetAttributes fetches user attributes by user ID
// The key format can be:
//   - "userId" - just the user ID (uses default org for roles)
//   - "userId:orgId" - user ID with specific organization context
func (p *AuthsomeUserAttributeProvider) GetAttributes(ctx context.Context, key string) (map[string]any, error) {
	userID, orgID, err := p.parseKey(key)
	if err != nil {
		return nil, err
	}

	// Fetch user from user service
	if p.userService == nil {
		return nil, errs.InternalServerErrorWithMessage("user service not configured")
	}

	user, err := p.userService.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Build base attributes
	attrs := p.userToAttributes(user)

	// Fetch roles and permissions if org context is available
	if orgID != nil {
		if err := p.enrichWithRBAC(ctx, attrs, userID, *orgID); err != nil {
			// Log but don't fail - some users might not have roles
			_ = err
		}
	} else if p.defaultOrgID != nil {
		if err := p.enrichWithRBAC(ctx, attrs, userID, *p.defaultOrgID); err != nil {
			_ = err
		}
	}

	// Fetch memberships if member service is available
	if p.memberService != nil {
		if err := p.enrichWithMemberships(ctx, attrs, userID); err != nil {
			// Log but don't fail
			_ = err
		}
	}

	return attrs, nil
}

// GetBatchAttributes fetches attributes for multiple users.
func (p *AuthsomeUserAttributeProvider) GetBatchAttributes(ctx context.Context, keys []string) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	for _, key := range keys {
		attrs, err := p.GetAttributes(ctx, key)
		if err != nil {
			// Skip users that can't be fetched
			continue
		}

		result[key] = attrs
	}

	return result, nil
}

// parseKey parses the key format "userId" or "userId:orgId".
func (p *AuthsomeUserAttributeProvider) parseKey(key string) (xid.ID, *xid.ID, error) {
	// Find separator
	sepIdx := -1

	for i := range key {
		if key[i] == ':' {
			sepIdx = i

			break
		}
	}

	var userIDStr, orgIDStr string
	if sepIdx == -1 {
		userIDStr = key
	} else {
		userIDStr = key[:sepIdx]
		orgIDStr = key[sepIdx+1:]
	}

	// Parse user ID
	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return xid.NilID(), nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Parse org ID if present
	var orgID *xid.ID

	if orgIDStr != "" {
		oid, err := xid.FromString(orgIDStr)
		if err != nil {
			return xid.NilID(), nil, fmt.Errorf("invalid org ID: %w", err)
		}

		orgID = &oid
	}

	return userID, orgID, nil
}

// userToAttributes converts an AuthsomeUser to attributes map.
func (p *AuthsomeUserAttributeProvider) userToAttributes(user AuthsomeUser) map[string]any {
	if user == nil {
		return make(map[string]any)
	}

	return map[string]any{
		"id":             user.GetID().String(),
		"app_id":         user.GetAppID().String(),
		"email":          user.GetEmail(),
		"name":           user.GetName(),
		"username":       user.GetUsername(),
		"image":          user.GetImage(),
		"email_verified": user.GetEmailVerified(),
		"created_at":     user.GetCreatedAt(),
		// Initialize empty arrays for roles/permissions
		"roles":       []string{},
		"permissions": []string{},
		"groups":      []string{},
		"org_ids":     []string{},
	}
}

// enrichWithRBAC adds role and permission data to attributes.
func (p *AuthsomeUserAttributeProvider) enrichWithRBAC(ctx context.Context, attrs map[string]any, userID, orgID xid.ID) error {
	if p.rbacService == nil {
		return nil
	}

	// Fetch roles
	roles, err := p.rbacService.GetUserRoles(ctx, userID, orgID)
	if err == nil && len(roles) > 0 {
		attrs["roles"] = roles
	}

	// Fetch permissions
	permissions, err := p.rbacService.GetUserPermissions(ctx, userID, orgID)
	if err == nil && len(permissions) > 0 {
		attrs["permissions"] = permissions
	}

	// Set current org context
	attrs["org_id"] = orgID.String()

	return nil
}

// enrichWithMemberships adds membership data to attributes.
func (p *AuthsomeUserAttributeProvider) enrichWithMemberships(ctx context.Context, attrs map[string]any, userID xid.ID) error {
	memberships, err := p.memberService.GetUserMembershipsForUser(ctx, userID)
	if err != nil {
		return err
	}

	orgIDs := make([]string, 0, len(memberships))
	memberRoles := make(map[string]string) // orgID -> role

	for _, m := range memberships {
		orgIDs = append(orgIDs, m.GetOrganizationID().String())
		memberRoles[m.GetOrganizationID().String()] = m.GetRole()
	}

	attrs["org_ids"] = orgIDs
	attrs["member_roles"] = memberRoles
	attrs["membership_count"] = len(memberships)

	return nil
}

// =============================================================================
// ADAPTER TYPES FOR CORE SERVICES
// =============================================================================

// UserAdapter adapts the core user.User to AuthsomeUser interface.
type UserAdapter struct {
	ID            xid.ID
	AppID         xid.ID
	Email         string
	Name          string
	EmailVerified bool
	Username      string
	Image         string
	CreatedAt     string
}

func (u *UserAdapter) GetID() xid.ID          { return u.ID }
func (u *UserAdapter) GetAppID() xid.ID       { return u.AppID }
func (u *UserAdapter) GetEmail() string       { return u.Email }
func (u *UserAdapter) GetName() string        { return u.Name }
func (u *UserAdapter) GetEmailVerified() bool { return u.EmailVerified }
func (u *UserAdapter) GetUsername() string    { return u.Username }
func (u *UserAdapter) GetImage() string       { return u.Image }
func (u *UserAdapter) GetCreatedAt() string   { return u.CreatedAt }

// MembershipAdapter adapts membership data to AuthsomeMembership interface.
type MembershipAdapter struct {
	OrganizationID xid.ID
	Role           string
	Status         string
}

func (m *MembershipAdapter) GetOrganizationID() xid.ID { return m.OrganizationID }
func (m *MembershipAdapter) GetRole() string           { return m.Role }
func (m *MembershipAdapter) GetStatus() string         { return m.Status }

// =============================================================================
// SERVICE WRAPPER FOR INTEGRATION
// =============================================================================

// UserServiceWrapper wraps the actual core user service.
type UserServiceWrapper struct {
	findByIDFunc func(ctx context.Context, id xid.ID) (AuthsomeUser, error)
}

// NewUserServiceWrapper creates a wrapper for user service.
func NewUserServiceWrapper(findByID func(ctx context.Context, id xid.ID) (AuthsomeUser, error)) *UserServiceWrapper {
	return &UserServiceWrapper{findByIDFunc: findByID}
}

func (w *UserServiceWrapper) FindByID(ctx context.Context, id xid.ID) (AuthsomeUser, error) {
	if w.findByIDFunc == nil {
		return nil, errs.InternalServerErrorWithMessage("user service not configured")
	}

	return w.findByIDFunc(ctx, id)
}

// MemberServiceWrapper wraps the actual member service.
type MemberServiceWrapper struct {
	getMembershipsFunc func(ctx context.Context, userID xid.ID) ([]AuthsomeMembership, error)
}

// NewMemberServiceWrapper creates a wrapper for member service.
func NewMemberServiceWrapper(getMemberships func(ctx context.Context, userID xid.ID) ([]AuthsomeMembership, error)) *MemberServiceWrapper {
	return &MemberServiceWrapper{getMembershipsFunc: getMemberships}
}

func (w *MemberServiceWrapper) GetUserMembershipsForUser(ctx context.Context, userID xid.ID) ([]AuthsomeMembership, error) {
	if w.getMembershipsFunc == nil {
		return nil, errs.InternalServerErrorWithMessage("member service not configured")
	}

	return w.getMembershipsFunc(ctx, userID)
}

// RBACServiceWrapper wraps the actual RBAC service.
type RBACServiceWrapper struct {
	getUserRolesFunc       func(ctx context.Context, userID, orgID xid.ID) ([]string, error)
	getUserPermissionsFunc func(ctx context.Context, userID, orgID xid.ID) ([]string, error)
}

// NewRBACServiceWrapper creates a wrapper for RBAC service.
func NewRBACServiceWrapper(
	getUserRoles func(ctx context.Context, userID, orgID xid.ID) ([]string, error),
	getUserPermissions func(ctx context.Context, userID, orgID xid.ID) ([]string, error),
) *RBACServiceWrapper {
	return &RBACServiceWrapper{
		getUserRolesFunc:       getUserRoles,
		getUserPermissionsFunc: getUserPermissions,
	}
}

func (w *RBACServiceWrapper) GetUserRoles(ctx context.Context, userID, orgID xid.ID) ([]string, error) {
	if w.getUserRolesFunc == nil {
		return nil, nil
	}

	return w.getUserRolesFunc(ctx, userID, orgID)
}

func (w *RBACServiceWrapper) GetUserPermissions(ctx context.Context, userID, orgID xid.ID) ([]string, error) {
	if w.getUserPermissionsFunc == nil {
		return nil, nil
	}

	return w.getUserPermissionsFunc(ctx, userID, orgID)
}
