// Package testing provides mocked Authsome instances and utilities for testing
// external integrations with the authsome authentication framework.
//
// This package is designed for developers building applications that integrate
// with authsome and need to test their code without setting up a full authsome
// instance with database, Redis, etc.
//
// Example usage:
//
//	import (
//	    "testing"
//	    authsometesting "github.com/xraph/authsome/testing"
//	)
//
//	func TestMyHandler(t *testing.T) {
//	    // Create a mock authsome with a test user
//	    mock := authsometesting.NewMock(t)
//	    user := mock.CreateUser("test@example.com", "Test User")
//	    org := mock.GetDefaultOrg()
//	    session := mock.CreateSession(user.ID.String(), org.ID.String())
//
//	    // Set up authenticated context
//	    ctx := mock.WithSession(context.Background(), session.ID.String())
//
//	    // Your test code here
//	    result, err := myService.DoSomething(ctx)
//	    // ... assertions
//	}
package testing

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// Mock provides a mocked Authsome instance for testing.
type Mock struct {
	t *testing.T

	// Core services
	UserService         *MockUserService
	SessionService      *MockSessionService
	OrganizationService *MockOrganizationService

	// Storage
	users        map[xid.ID]*schema.User
	sessions     map[xid.ID]*schema.Session
	apps         map[xid.ID]*schema.App
	environments map[xid.ID]*schema.Environment
	orgs         map[xid.ID]*schema.Organization
	members      map[xid.ID][]*schema.OrganizationMember // Organization members

	// Default entities (created automatically)
	defaultApp *schema.App
	defaultEnv *schema.Environment
	defaultOrg *schema.Organization

	mu sync.RWMutex
}

// NewMock creates a new mock Authsome instance for testing.
// NewMock automatically creates default app, environment, and organization.
func NewMock(t *testing.T) *Mock {
	t.Helper()

	// Create default app
	defaultApp := &schema.App{
		ID:   xid.New(),
		Name: "Test App",
		Slug: "test-app",
	}

	// Create default environment
	defaultEnv := &schema.Environment{
		ID:    xid.New(),
		AppID: defaultApp.ID,
		Name:  "test",
		Slug:  "test",
	}

	// Create default organization
	defaultOrg := &schema.Organization{
		ID:            xid.New(),
		AppID:         defaultApp.ID,
		EnvironmentID: defaultEnv.ID,
		Name:          "Test Organization",
		Slug:          "test-org",
		Metadata:      map[string]any{},
	}

	m := &Mock{
		t:            t,
		users:        make(map[xid.ID]*schema.User),
		sessions:     make(map[xid.ID]*schema.Session),
		apps:         map[xid.ID]*schema.App{defaultApp.ID: defaultApp},
		environments: map[xid.ID]*schema.Environment{defaultEnv.ID: defaultEnv},
		orgs:         map[xid.ID]*schema.Organization{defaultOrg.ID: defaultOrg},
		members:      make(map[xid.ID][]*schema.OrganizationMember),
		defaultApp:   defaultApp,
		defaultEnv:   defaultEnv,
		defaultOrg:   defaultOrg,
	}

	// Initialize mock services
	m.UserService = &MockUserService{mock: m}
	m.SessionService = &MockSessionService{mock: m}
	m.OrganizationService = &MockOrganizationService{mock: m}

	return m
}

// CreateUser creates a test user with the given email and name.
// CreateUser user is automatically added to the default organization.
func (m *Mock) CreateUser(email, name string) *schema.User {
	m.t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	user := &schema.User{
		ID:            xid.New(),
		Email:         email,
		Name:          name,
		EmailVerified: true,
	}

	m.users[user.ID] = user

	// Add to default org
	member := &schema.OrganizationMember{
		ID:             xid.New(),
		OrganizationID: m.defaultOrg.ID,
		UserID:         user.ID,
		Role:           "member",
		Status:         "active",
	}
	m.members[m.defaultOrg.ID] = append(m.members[m.defaultOrg.ID], member)

	return user
}

// CreateUserWithRole creates a test user with a specific role in the default organization.
func (m *Mock) CreateUserWithRole(email, name, role string) *schema.User {
	m.t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	user := &schema.User{
		ID:            xid.New(),
		Email:         email,
		Name:          name,
		EmailVerified: true,
	}

	m.users[user.ID] = user

	// Add to default org with specified role
	member := &schema.OrganizationMember{
		ID:             xid.New(),
		OrganizationID: m.defaultOrg.ID,
		UserID:         user.ID,
		Role:           role,
		Status:         "active",
	}
	m.members[m.defaultOrg.ID] = append(m.members[m.defaultOrg.ID], member)

	return user
}

// CreateOrganization creates a test organization.
func (m *Mock) CreateOrganization(name, slug string) *schema.Organization {
	m.t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	org := &schema.Organization{
		ID:            xid.New(),
		AppID:         m.defaultApp.ID,
		EnvironmentID: m.defaultEnv.ID,
		Name:          name,
		Slug:          slug,
		Metadata:      map[string]any{},
	}

	m.orgs[org.ID] = org

	return org
}

// AddUserToOrg adds a user to an organization with the specified role.
func (m *Mock) AddUserToOrg(userID, orgID xid.ID, role string) *schema.OrganizationMember {
	m.t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	member := &schema.OrganizationMember{
		ID:             xid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		Status:         "active",
	}

	m.members[orgID] = append(m.members[orgID], member)

	return member
}

// CreateSession creates a test session for the given user and organization.
func (m *Mock) CreateSession(userID, orgID xid.ID) *schema.Session {
	m.t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	session := &schema.Session{
		ID:        xid.New(),
		Token:     "session_" + xid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	m.sessions[session.ID] = session

	return session
}

// CreateExpiredSession creates an expired session for testing expiration scenarios.
func (m *Mock) CreateExpiredSession(userID, orgID xid.ID) *schema.Session {
	m.t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	session := &schema.Session{
		ID:        xid.New(),
		Token:     "session_" + xid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	m.sessions[session.ID] = session

	return session
}

// GetDefaultOrg returns the default test organization.
func (m *Mock) GetDefaultOrg() *schema.Organization {
	return m.defaultOrg
}

// GetUser retrieves a user by ID.
func (m *Mock) GetUser(userID xid.ID) (*schema.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[userID]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return user, nil
}

// GetSession retrieves a session by ID.
func (m *Mock) GetSession(sessionID xid.ID) (*schema.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// GetApp retrieves an app by ID.
func (m *Mock) GetApp(appID xid.ID) (*schema.App, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	app, ok := m.apps[appID]
	if !ok {
		return nil, fmt.Errorf("app not found: %s", appID)
	}

	return app, nil
}

// GetEnvironment retrieves an environment by ID.
func (m *Mock) GetEnvironment(envID xid.ID) (*schema.Environment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	env, ok := m.environments[envID]
	if !ok {
		return nil, fmt.Errorf("environment not found: %s", envID)
	}

	return env, nil
}

// GetOrganization retrieves an organization by ID.
func (m *Mock) GetOrganization(orgID xid.ID) (*schema.Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	org, ok := m.orgs[orgID]
	if !ok {
		return nil, fmt.Errorf("organization not found: %s", orgID)
	}

	return org, nil
}

// GetUserOrgs returns all organizations a user is a member of.
func (m *Mock) GetUserOrgs(userID xid.ID) ([]*schema.Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var orgs []*schema.Organization

	for orgID, members := range m.members {
		for _, member := range members {
			if member.UserID == userID {
				if org, ok := m.orgs[orgID]; ok {
					orgs = append(orgs, org)
				}

				break
			}
		}
	}

	return orgs, nil
}

// GetDefaultApp returns the default test app.
func (m *Mock) GetDefaultApp() *schema.App {
	return m.defaultApp
}

// GetDefaultEnvironment returns the default test environment.
func (m *Mock) GetDefaultEnvironment() *schema.Environment {
	return m.defaultEnv
}

// WithUser sets user in context using core/contexts.
func (m *Mock) WithUser(ctx context.Context, userID xid.ID) context.Context {
	return contexts.SetUserID(ctx, userID)
}

// WithApp sets app in context using core/contexts.
func (m *Mock) WithApp(ctx context.Context, appID xid.ID) context.Context {
	return contexts.SetAppID(ctx, appID)
}

// WithEnvironment sets environment in context using core/contexts.
func (m *Mock) WithEnvironment(ctx context.Context, envID xid.ID) context.Context {
	return contexts.SetEnvironmentID(ctx, envID)
}

// WithOrganization sets organization in context using core/contexts.
func (m *Mock) WithOrganization(ctx context.Context, orgID xid.ID) context.Context {
	return contexts.SetOrganizationID(ctx, orgID)
}

// WithSession adds session and user to context.
func (m *Mock) WithSession(ctx context.Context, sessionID xid.ID) context.Context {
	session, err := m.GetSession(sessionID)
	if err != nil {
		m.t.Fatalf("failed to get session: %v", err)
	}

	ctx = contexts.SetUserID(ctx, session.UserID)
	// Store session object separately for retrieval
	return context.WithValue(ctx, sessionContextKey, session)
}

// NewTestContext creates fully authenticated context with all tenancy levels.
func (m *Mock) NewTestContext() context.Context {
	user := m.CreateUser("test@example.com", "Test User")
	session := m.CreateSession(user.ID, m.defaultOrg.ID)

	ctx := context.Background()
	ctx = contexts.SetAppID(ctx, m.defaultApp.ID)
	ctx = contexts.SetEnvironmentID(ctx, m.defaultEnv.ID)
	ctx = contexts.SetOrganizationID(ctx, m.defaultOrg.ID)
	ctx = contexts.SetUserID(ctx, user.ID)
	ctx = context.WithValue(ctx, sessionContextKey, session)

	return ctx
}

// NewTestContextWithUser creates authenticated context for specific user.
func (m *Mock) NewTestContextWithUser(user *schema.User) context.Context {
	session := m.CreateSession(user.ID, m.defaultOrg.ID)

	ctx := context.Background()
	ctx = contexts.SetAppID(ctx, m.defaultApp.ID)
	ctx = contexts.SetEnvironmentID(ctx, m.defaultEnv.ID)
	ctx = contexts.SetOrganizationID(ctx, m.defaultOrg.ID)
	ctx = contexts.SetUserID(ctx, user.ID)
	ctx = context.WithValue(ctx, sessionContextKey, session)

	return ctx
}

// Reset clears all test data. Useful for cleanup between tests.
func (m *Mock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Recreate default app
	m.defaultApp = &schema.App{
		ID:   xid.New(),
		Name: "Test App",
		Slug: "test-app",
	}

	// Recreate default environment
	m.defaultEnv = &schema.Environment{
		ID:    xid.New(),
		AppID: m.defaultApp.ID,
		Name:  "test",
		Slug:  "test",
	}

	// Recreate default org
	m.defaultOrg = &schema.Organization{
		ID:            xid.New(),
		AppID:         m.defaultApp.ID,
		EnvironmentID: m.defaultEnv.ID,
		Name:          "Test Organization",
		Slug:          "test-org",
		Metadata:      map[string]any{},
	}

	m.users = make(map[xid.ID]*schema.User)
	m.sessions = make(map[xid.ID]*schema.Session)
	m.apps = map[xid.ID]*schema.App{m.defaultApp.ID: m.defaultApp}
	m.environments = map[xid.ID]*schema.Environment{m.defaultEnv.ID: m.defaultEnv}
	m.orgs = map[xid.ID]*schema.Organization{m.defaultOrg.ID: m.defaultOrg}
	m.members = make(map[xid.ID][]*schema.OrganizationMember)
}

// MockUserService implements core user service methods for testing.
type MockUserService struct {
	mock *Mock
}

func (s *MockUserService) Create(ctx context.Context, req *user.CreateUserRequest) (*schema.User, error) {
	s.mock.mu.Lock()
	defer s.mock.mu.Unlock()

	user := &schema.User{
		ID:            xid.New(),
		Email:         req.Email,
		Name:          req.Name,
		EmailVerified: false,
	}

	s.mock.users[user.ID] = user

	return user, nil
}

func (s *MockUserService) GetByID(ctx context.Context, userID xid.ID) (*schema.User, error) {
	return s.mock.GetUser(userID)
}

func (s *MockUserService) GetByEmail(ctx context.Context, email string) (*schema.User, error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	for _, user := range s.mock.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found with email: %s", email)
}

func (s *MockUserService) Update(ctx context.Context, userID xid.ID, req *user.UpdateUserRequest) (*schema.User, error) {
	s.mock.mu.Lock()
	defer s.mock.mu.Unlock()

	user, ok := s.mock.users[userID]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	if req.Name != nil {
		user.Name = *req.Name
	}

	if req.Email != nil {
		user.Email = *req.Email
	}

	return user, nil
}

func (s *MockUserService) Delete(ctx context.Context, userID xid.ID) error {
	s.mock.mu.Lock()
	defer s.mock.mu.Unlock()

	delete(s.mock.users, userID)

	return nil
}

// MockSessionService implements core session service methods for testing.
type MockSessionService struct {
	mock *Mock
}

func (s *MockSessionService) Create(ctx context.Context, req *session.CreateSessionRequest) (*schema.Session, error) {
	return s.mock.CreateSession(req.UserID, xid.NilID()), nil
}

func (s *MockSessionService) GetByID(ctx context.Context, sessionID xid.ID) (*schema.Session, error) {
	return s.mock.GetSession(sessionID)
}

func (s *MockSessionService) GetByToken(ctx context.Context, token string) (*schema.Session, error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	for _, session := range s.mock.sessions {
		if session.Token == token {
			return session, nil
		}
	}

	return nil, errs.New(errs.CodeSessionNotFound, "session not found with token", http.StatusNotFound)
}

func (s *MockSessionService) Delete(ctx context.Context, sessionID xid.ID) error {
	s.mock.mu.Lock()
	defer s.mock.mu.Unlock()

	delete(s.mock.sessions, sessionID)

	return nil
}

func (s *MockSessionService) Validate(ctx context.Context, token string) (*schema.Session, error) {
	session, err := s.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errs.New(errs.CodeSessionExpired, "session expired", http.StatusUnauthorized)
	}

	return session, nil
}
