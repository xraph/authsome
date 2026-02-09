package adapters

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/user"
)

// =============================================================================
// AUTHSOME USER ADAPTER
// =============================================================================

// AuthsomeUserAdapter adapts authsome's user service to the generic UserProvider interface.
type AuthsomeUserAdapter struct {
	userSvc *user.Service
}

// NewAuthsomeUserAdapter creates a new adapter for authsome's user service.
func NewAuthsomeUserAdapter(userSvc *user.Service) *AuthsomeUserAdapter {
	return &AuthsomeUserAdapter{
		userSvc: userSvc,
	}
}

// GetUser retrieves a single user by ID.
func (a *AuthsomeUserAdapter) GetUser(ctx context.Context, scope *audit.Scope, userID string) (*audit.GenericUser, error) {
	// Parse ID
	uid, err := xid.FromString(userID)
	if err != nil {
		return nil, err
	}

	// Get user from authsome service
	u, err := a.userSvc.FindByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return a.convertUser(u), nil
}

// ListUsers retrieves users matching filter criteria.
func (a *AuthsomeUserAdapter) ListUsers(ctx context.Context, scope *audit.Scope, filter *audit.UserFilter) ([]*audit.GenericUser, error) {
	// Note: Simplified implementation
	// In production, would need to implement user listing with proper filtering
	return []*audit.GenericUser{}, nil
}

// QueryUserMetrics retrieves aggregated metrics about users.
func (a *AuthsomeUserAdapter) QueryUserMetrics(ctx context.Context, scope *audit.Scope, query *audit.MetricsQuery) (*audit.UserMetrics, error) {
	// Note: Simplified implementation
	// In production, would implement proper metrics aggregation
	return &audit.UserMetrics{
		TotalUsers: 0,
		ByRole:     make(map[string]int),
		ByStatus:   make(map[string]int),
	}, nil
}

// convertUser converts authsome User to GenericUser.
func (a *AuthsomeUserAdapter) convertUser(u *user.User) *audit.GenericUser {
	if u == nil {
		return nil
	}

	return &audit.GenericUser{
		ID:               u.ID.String(),
		Email:            u.Email,
		Username:         u.Email, // Authsome uses email as username
		DisplayName:      u.Name,
		MFAEnabled:       false, // TODO: Would need MFA tracking
		MFAMethods:       []string{},
		PasswordChanged:  u.UpdatedAt, // Approximate with UpdatedAt
		LastLogin:        time.Time{}, // TODO: Would need session tracking
		LoginCount:       0,
		FailedLoginCount: 0,
		Status:           "active",
		Roles:            []string{},
		Groups:           []string{},
		Metadata:         make(map[string]any),
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

// =============================================================================
// AUTHSOME ORGANIZATION ADAPTER
// =============================================================================

// AuthsomeOrgAdapter adapts authsome's organization service to the generic OrganizationProvider interface.
type AuthsomeOrgAdapter struct {
	orgSvc *organization.Service
}

// NewAuthsomeOrgAdapter creates a new adapter for authsome's organization service.
func NewAuthsomeOrgAdapter(orgSvc *organization.Service) *AuthsomeOrgAdapter {
	return &AuthsomeOrgAdapter{
		orgSvc: orgSvc,
	}
}

// GetOrganization retrieves organization details.
func (a *AuthsomeOrgAdapter) GetOrganization(ctx context.Context, orgID string) (*audit.GenericOrganization, error) {
	// Parse ID
	oid, err := xid.FromString(orgID)
	if err != nil {
		return nil, err
	}

	// Get organization from authsome service
	org, err := a.orgSvc.FindOrganizationByID(ctx, oid)
	if err != nil {
		return nil, err
	}

	return a.convertOrganization(org), nil
}

// ListOrganizations retrieves organizations matching filter.
func (a *AuthsomeOrgAdapter) ListOrganizations(ctx context.Context, filter *audit.OrgFilter) ([]*audit.GenericOrganization, error) {
	// Note: Simplified implementation
	// In production, would implement proper org listing with filtering
	return []*audit.GenericOrganization{}, nil
}

// convertOrganization converts authsome Organization to GenericOrganization.
func (a *AuthsomeOrgAdapter) convertOrganization(org *organization.Organization) *audit.GenericOrganization {
	if org == nil {
		return nil
	}

	return &audit.GenericOrganization{
		ID:          org.ID.String(),
		Name:        org.Name,
		DisplayName: org.Name, // Authsome doesn't have DisplayName, use Name
		Status:      "active",
		ParentID:    nil,
		Members:     0,
		Metadata:    org.Metadata,
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
	}
}

// =============================================================================
// NULL ADAPTERS - For when providers are not available
// =============================================================================

// NullUserProvider is a no-op user provider.
type NullUserProvider struct{}

// NewNullUserProvider creates a null user provider.
func NewNullUserProvider() *NullUserProvider {
	return &NullUserProvider{}
}

// GetUser returns nil (no user data available).
func (n *NullUserProvider) GetUser(ctx context.Context, scope *audit.Scope, userID string) (*audit.GenericUser, error) {
	return nil, nil
}

// ListUsers returns empty list.
func (n *NullUserProvider) ListUsers(ctx context.Context, scope *audit.Scope, filter *audit.UserFilter) ([]*audit.GenericUser, error) {
	return []*audit.GenericUser{}, nil
}

// QueryUserMetrics returns zero metrics.
func (n *NullUserProvider) QueryUserMetrics(ctx context.Context, scope *audit.Scope, query *audit.MetricsQuery) (*audit.UserMetrics, error) {
	return &audit.UserMetrics{
		ByRole:   make(map[string]int),
		ByStatus: make(map[string]int),
	}, nil
}

// NullOrgProvider is a no-op organization provider.
type NullOrgProvider struct{}

// NewNullOrgProvider creates a null org provider.
func NewNullOrgProvider() *NullOrgProvider {
	return &NullOrgProvider{}
}

// GetOrganization returns nil (no org data available).
func (n *NullOrgProvider) GetOrganization(ctx context.Context, orgID string) (*audit.GenericOrganization, error) {
	return nil, nil
}

// ListOrganizations returns empty list.
func (n *NullOrgProvider) ListOrganizations(ctx context.Context, filter *audit.OrgFilter) ([]*audit.GenericOrganization, error) {
	return []*audit.GenericOrganization{}, nil
}
