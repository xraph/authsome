package testing

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// sessionContextKey is used to store the session object in context.
type contextKey string

const sessionContextKey contextKey = "test_session"

// Context helpers using core/contexts typed keys
// These match the actual AuthSome context system

// GetUserID retrieves the user ID from context using core/contexts.
func GetUserID(ctx context.Context) (xid.ID, bool) {
	return contexts.GetUserID(ctx)
}

// GetAppID retrieves the app ID from context using core/contexts.
func GetAppID(ctx context.Context) (xid.ID, bool) {
	return contexts.GetAppID(ctx)
}

// GetEnvironmentID retrieves the environment ID from context using core/contexts.
func GetEnvironmentID(ctx context.Context) (xid.ID, bool) {
	return contexts.GetEnvironmentID(ctx)
}

// GetOrganizationID retrieves the organization ID from context using core/contexts.
func GetOrganizationID(ctx context.Context) (xid.ID, bool) {
	return contexts.GetOrganizationID(ctx)
}

// GetSession retrieves the session object from context.
func GetSession(ctx context.Context) (*schema.Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*schema.Session)

	return session, ok
}

// Convenience helpers that fetch full entities from Mock

// GetUserFromContext retrieves the full user object from context using the Mock.
func (m *Mock) GetUserFromContext(ctx context.Context) (*schema.User, error) {
	userID, ok := GetUserID(ctx)
	if !ok || userID.IsNil() {
		return nil, errs.New(errs.CodeUserNotFound, "user ID not found in context", http.StatusNotFound)
	}

	return m.GetUser(userID)
}

// GetAppFromContext retrieves the full app object from context using the Mock.
func (m *Mock) GetAppFromContext(ctx context.Context) (*schema.App, error) {
	appID, ok := GetAppID(ctx)
	if !ok || appID.IsNil() {
		return nil, errs.New(errs.CodeNotFound, "app ID not found in context", http.StatusNotFound)
	}

	return m.GetApp(appID)
}

// GetEnvironmentFromContext retrieves the full environment object from context using the Mock.
func (m *Mock) GetEnvironmentFromContext(ctx context.Context) (*schema.Environment, error) {
	envID, ok := GetEnvironmentID(ctx)
	if !ok || envID.IsNil() {
		return nil, errs.New(errs.CodeNotFound, "environment ID not found in context", http.StatusNotFound)
	}

	return m.GetEnvironment(envID)
}

// GetOrganizationFromContext retrieves the full organization object from context using the Mock.
func (m *Mock) GetOrganizationFromContext(ctx context.Context) (*schema.Organization, error) {
	orgID, ok := GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		return nil, errs.New(errs.CodeOrganizationNotFound, "organization ID not found in context", http.StatusNotFound)
	}

	return m.GetOrganization(orgID)
}

// RequireAuth is a helper that mimics authentication middleware behavior.
// It checks if the context has a valid session and returns the user.
func (m *Mock) RequireAuth(ctx context.Context) (*schema.User, error) {
	m.t.Helper()

	// Get session from context
	session, ok := GetSession(ctx)
	if !ok {
		return nil, ErrNotAuthenticated
	}

	// Validate session
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrInvalidSession
	}

	// Get user
	user, err := m.GetUser(session.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Check if user is active
	if user.DeletedAt != nil {
		return nil, ErrUserInactive
	}

	return user, nil
}

// RequireOrgMember checks if the user is a member of the specified organization.
func (m *Mock) RequireOrgMember(ctx context.Context, orgID xid.ID) (*schema.OrganizationMember, error) {
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
		if member.UserID == user.ID {
			return member, nil
		}
	}

	return nil, ErrNotOrgMember
}

// RequireOrgRole checks if the user has the specified role in the organization.
func (m *Mock) RequireOrgRole(ctx context.Context, orgID xid.ID, requiredRole string) (*schema.OrganizationMember, error) {
	m.t.Helper()

	member, err := m.RequireOrgMember(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if member.Role != requiredRole {
		return nil, ErrInsufficientPermissions
	}

	return member, nil
}

// Common test errors.
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
	App         *schema.App
	Environment *schema.Environment
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
	session := cs.mock.CreateSession(user.ID, cs.mock.defaultOrg.ID)
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "authenticated_user",
		Description: "A regular authenticated user",
		User:        user,
		App:         cs.mock.defaultApp,
		Environment: cs.mock.defaultEnv,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// AdminUser returns a scenario with an admin user.
func (cs *CommonScenarios) AdminUser() *Scenario {
	user := cs.mock.CreateUserWithRole("admin@example.com", "Admin User", "admin")
	session := cs.mock.CreateSession(user.ID, cs.mock.defaultOrg.ID)
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "admin_user",
		Description: "An admin user with elevated privileges",
		User:        user,
		App:         cs.mock.defaultApp,
		Environment: cs.mock.defaultEnv,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// UnverifiedUser returns a scenario with an unverified user.
func (cs *CommonScenarios) UnverifiedUser() *Scenario {
	user := cs.mock.CreateUser("unverified@example.com", "Unverified User")
	user.EmailVerified = false
	session := cs.mock.CreateSession(user.ID, cs.mock.defaultOrg.ID)
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "unverified_user",
		Description: "A user with unverified email",
		User:        user,
		App:         cs.mock.defaultApp,
		Environment: cs.mock.defaultEnv,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// MultiOrgUser returns a scenario with a user belonging to multiple organizations.
func (cs *CommonScenarios) MultiOrgUser() *Scenario {
	user := cs.mock.CreateUser("multi@example.com", "Multi-Org User")
	org2 := cs.mock.CreateOrganization("Second Org", "second-org")
	cs.mock.AddUserToOrg(user.ID, org2.ID, "member")
	session := cs.mock.CreateSession(user.ID, cs.mock.defaultOrg.ID)
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "multi_org_user",
		Description: "A user belonging to multiple organizations",
		User:        user,
		App:         cs.mock.defaultApp,
		Environment: cs.mock.defaultEnv,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}

// ExpiredSession returns a scenario with an expired session.
func (cs *CommonScenarios) ExpiredSession() *Scenario {
	user := cs.mock.CreateUser("expired@example.com", "Expired Session User")
	session := cs.mock.CreateExpiredSession(user.ID, cs.mock.defaultOrg.ID)
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "expired_session",
		Description: "A user with an expired session",
		User:        user,
		App:         cs.mock.defaultApp,
		Environment: cs.mock.defaultEnv,
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
		App:         nil,
		Environment: nil,
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
	session := cs.mock.CreateSession(user.ID, cs.mock.defaultOrg.ID)
	ctx := cs.mock.NewTestContextWithUser(user)

	return &Scenario{
		Name:        "inactive_user",
		Description: "A user with inactive account",
		User:        user,
		App:         cs.mock.defaultApp,
		Environment: cs.mock.defaultEnv,
		Org:         cs.mock.defaultOrg,
		Session:     session,
		Context:     ctx,
	}
}
