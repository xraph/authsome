package testing

import (
	"context"
	"time"

	"github.com/xraph/authsome/schema"
)

// GetLoggedInUser retrieves the logged-in user from context.
// This helper mimics what your actual application would do.
func GetLoggedInUser(ctx context.Context) (*schema.User, bool) {
	user, ok := ctx.Value("user").(*schema.User)
	return user, ok
}

// GetLoggedInUserID retrieves the logged-in user ID from context.
func GetLoggedInUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

// GetCurrentOrg retrieves the current organization from context.
func GetCurrentOrg(ctx context.Context) (*schema.Organization, bool) {
	org, ok := ctx.Value("organization").(*schema.Organization)
	return org, ok
}

// GetCurrentOrgID retrieves the current organization ID from context.
func GetCurrentOrgID(ctx context.Context) (string, bool) {
	orgID, ok := ctx.Value("org_id").(string)
	return orgID, ok
}

// GetCurrentSession retrieves the current session from context.
func GetCurrentSession(ctx context.Context) (*schema.Session, bool) {
	session, ok := ctx.Value("session").(*schema.Session)
	return session, ok
}

// GetCurrentSessionID retrieves the current session ID from context.
func GetCurrentSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value("session_id").(string)
	return sessionID, ok
}

// RequireAuth is a helper that mimics authentication middleware behavior.
// It checks if the context has a valid session and returns the user.
func (m *Mock) RequireAuth(ctx context.Context) (*schema.User, error) {
	m.t.Helper()

	sessionID, ok := GetCurrentSessionID(ctx)
	if !ok {
		// Try to get session directly
		session, ok := GetCurrentSession(ctx)
		if !ok {
			return nil, ErrNotAuthenticated
		}
		sessionID = session.ID.String()
	}

	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, ErrInvalidSession
	}

	user, err := m.GetUser(session.UserID.String())
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// RequireOrgMember checks if the user is a member of the specified organization.
func (m *Mock) RequireOrgMember(ctx context.Context, orgID string) (*schema.Member, error) {
	m.t.Helper()

	user, err := m.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// First verify the org exists
	if _, ok := m.orgs[orgID]; !ok {
		return nil, ErrOrgNotFound
	}

	// Get members list (may be empty if no members yet)
	members := m.members[orgID]

	for _, member := range members {
		if member.UserID.String() == user.ID.String() {
			return member, nil
		}
	}

	return nil, ErrNotOrgMember
}

// RequireOrgRole checks if the user has the specified role in the organization.
func (m *Mock) RequireOrgRole(ctx context.Context, orgID, requiredRole string) (*schema.Member, error) {
	m.t.Helper()

	member, err := m.RequireOrgMember(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if string(member.Role) != requiredRole {
		return nil, ErrInsufficientPermissions
	}

	return member, nil
}

// Common test errors
var (
	ErrNotAuthenticated        = &TestError{Code: "not_authenticated", Message: "user is not authenticated"}
	ErrInvalidSession          = &TestError{Code: "invalid_session", Message: "session is invalid or expired"}
	ErrUserNotFound            = &TestError{Code: "user_not_found", Message: "user not found"}
	ErrUserInactive            = &TestError{Code: "user_inactive", Message: "user account is inactive"}
	ErrOrgNotFound             = &TestError{Code: "org_not_found", Message: "organization not found"}
	ErrNotOrgMember            = &TestError{Code: "not_org_member", Message: "user is not a member of this organization"}
	ErrInsufficientPermissions = &TestError{Code: "insufficient_permissions", Message: "user does not have required permissions"}
)

// TestError represents a test error with a code and message.
type TestError struct {
	Code    string
	Message string
}

func (e *TestError) Error() string {
	return e.Message
}

// Scenario represents a common test scenario with pre-configured data.
type Scenario struct {
	Name        string
	Description string
	User        *schema.User
	Org         *schema.Organization
	Session     *schema.Session
	Context     context.Context
}

// CommonScenarios provides pre-configured test scenarios.
type CommonScenarios struct {
	mock *Mock
}

// NewCommonScenarios creates a new set of common test scenarios.
func (m *Mock) NewCommonScenarios() *CommonScenarios {
	return &CommonScenarios{mock: m}
}

// AuthenticatedUser returns a scenario with a basic authenticated user.
func (cs *CommonScenarios) AuthenticatedUser() *Scenario {
	user := cs.mock.CreateUser("user@example.com", "Regular User")
	session := cs.mock.CreateSession(user.ID.String(), cs.mock.defaultOrg.ID.String())
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "authenticated_user",
		Description: "A regular authenticated user",
		User:        user,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// AdminUser returns a scenario with an admin user.
func (cs *CommonScenarios) AdminUser() *Scenario {
	user := cs.mock.CreateUserWithRole("admin@example.com", "Admin User", "admin")
	session := cs.mock.CreateSession(user.ID.String(), cs.mock.defaultOrg.ID.String())
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "admin_user",
		Description: "An admin user with elevated privileges",
		User:        user,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// UnverifiedUser returns a scenario with an unverified user.
func (cs *CommonScenarios) UnverifiedUser() *Scenario {
	user := cs.mock.CreateUser("unverified@example.com", "Unverified User")
	user.EmailVerified = false
	session := cs.mock.CreateSession(user.ID.String(), cs.mock.defaultOrg.ID.String())
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "unverified_user",
		Description: "A user with unverified email",
		User:        user,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// MultiOrgUser returns a scenario with a user belonging to multiple organizations.
func (cs *CommonScenarios) MultiOrgUser() *Scenario {
	user := cs.mock.CreateUser("multi@example.com", "Multi-Org User")
	org2 := cs.mock.CreateOrganization("Second Org", "second-org")
	cs.mock.AddUserToOrg(user.ID.String(), org2.ID.String(), "member")
	session := cs.mock.CreateSession(user.ID.String(), cs.mock.defaultOrg.ID.String())
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "multi_org_user",
		Description: "A user belonging to multiple organizations",
		User:        user,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// ExpiredSession returns a scenario with an expired session.
func (cs *CommonScenarios) ExpiredSession() *Scenario {
	user := cs.mock.CreateUser("expired@example.com", "Expired Session User")
	session := cs.mock.CreateExpiredSession(user.ID.String(), cs.mock.defaultOrg.ID.String())
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "expired_session",
		Description: "A user with an expired session",
		User:        user,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// UnauthenticatedUser returns a scenario with no authentication.
func (cs *CommonScenarios) UnauthenticatedUser() *Scenario {
	return &Scenario{
		Name:        "unauthenticated",
		Description: "No authentication present",
		User:        nil,
		Org:         nil,
		Session:     nil,
		Context:     context.Background(),
	}
}

// InactiveUser returns a scenario with an inactive user account.
func (cs *CommonScenarios) InactiveUser() *Scenario {
	user := cs.mock.CreateUser("inactive@example.com", "Inactive User")
	// Mark user as deleted (soft delete)
	now := time.Now()
	user.DeletedAt = &now
	session := cs.mock.CreateSession(user.ID.String(), cs.mock.defaultOrg.ID.String())
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "inactive_user",
		Description: "A user with inactive account",
		User:        user,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}
