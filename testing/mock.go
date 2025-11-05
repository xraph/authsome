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
	"sync"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
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
	users    map[string]*schema.User
	sessions map[string]*schema.Session
	orgs     map[string]*schema.Organization
	members  map[string][]*schema.Member

	// Default organization (created automatically)
	defaultOrg *schema.Organization

	mu sync.RWMutex
}

// NewMock creates a new mock Authsome instance for testing.
// It automatically creates a default organization and sets up basic services.
func NewMock(t *testing.T) *Mock {
	t.Helper()

	defaultOrg := &schema.Organization{
		ID:       xid.New(),
		Name:     "Test Organization",
		Slug:     "test-org",
		Metadata: map[string]interface{}{},
	}

	m := &Mock{
		t:          t,
		users:      make(map[string]*schema.User),
		sessions:   make(map[string]*schema.Session),
		orgs:       map[string]*schema.Organization{defaultOrg.ID.String(): defaultOrg},
		members:    make(map[string][]*schema.Member),
		defaultOrg: defaultOrg,
	}

	// Initialize mock services
	m.UserService = &MockUserService{mock: m}
	m.SessionService = &MockSessionService{mock: m}
	m.OrganizationService = &MockOrganizationService{mock: m}

	return m
}

// CreateUser creates a test user with the given email and name.
// The user is automatically added to the default organization.
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

	m.users[user.ID.String()] = user

	// Add to default org
	member := &schema.Member{
		ID:             xid.New(),
		OrganizationID: m.defaultOrg.ID,
		UserID:         user.ID,
		Role:           "member",
	}
	m.members[m.defaultOrg.ID.String()] = append(m.members[m.defaultOrg.ID.String()], member)

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

	m.users[user.ID.String()] = user

	// Add to default org with specified role
	member := &schema.Member{
		ID:             xid.New(),
		OrganizationID: m.defaultOrg.ID,
		UserID:         user.ID,
		Role:           role,
	}
	m.members[m.defaultOrg.ID.String()] = append(m.members[m.defaultOrg.ID.String()], member)

	return user
}

// CreateOrganization creates a test organization.
func (m *Mock) CreateOrganization(name, slug string) *schema.Organization {
	m.t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	org := &schema.Organization{
		ID:       xid.New(),
		Name:     name,
		Slug:     slug,
		Metadata: map[string]interface{}{},
	}

	m.orgs[org.ID.String()] = org
	return org
}

// AddUserToOrg adds a user to an organization with the specified role.
func (m *Mock) AddUserToOrg(userID, orgID, role string) *schema.Member {
	m.t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse IDs
	uid, _ := xid.FromString(userID)
	oid, _ := xid.FromString(orgID)

	member := &schema.Member{
		ID:             xid.New(),
		OrganizationID: oid,
		UserID:         uid,
		Role:           role,
	}

	m.members[orgID] = append(m.members[orgID], member)
	return member
}

// CreateSession creates a test session for the given user and organization.
func (m *Mock) CreateSession(userID, orgID string) *schema.Session {
	m.t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse IDs
	uid, _ := xid.FromString(userID)

	session := &schema.Session{
		ID:        xid.New(),
		Token:     "session_" + xid.New().String(),
		UserID:    uid,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	m.sessions[session.ID.String()] = session
	return session
}

// CreateExpiredSession creates an expired session for testing expiration scenarios.
func (m *Mock) CreateExpiredSession(userID, orgID string) *schema.Session {
	m.t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse IDs
	uid, _ := xid.FromString(userID)

	session := &schema.Session{
		ID:        xid.New(),
		Token:     "session_" + xid.New().String(),
		UserID:    uid,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	m.sessions[session.ID.String()] = session
	return session
}

// GetDefaultOrg returns the default test organization.
func (m *Mock) GetDefaultOrg() *schema.Organization {
	return m.defaultOrg
}

// GetUser retrieves a user by ID.
func (m *Mock) GetUser(userID string) (*schema.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[userID]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	return user, nil
}

// GetSession retrieves a session by ID.
func (m *Mock) GetSession(sessionID string) (*schema.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

// GetOrganization retrieves an organization by ID.
func (m *Mock) GetOrganization(orgID string) (*schema.Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	org, ok := m.orgs[orgID]
	if !ok {
		return nil, fmt.Errorf("organization not found: %s", orgID)
	}
	return org, nil
}

// GetUserOrgs returns all organizations a user is a member of.
func (m *Mock) GetUserOrgs(userID string) ([]*schema.Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var orgs []*schema.Organization
	for orgID, members := range m.members {
		for _, member := range members {
			if member.UserID.String() == userID {
				if org, ok := m.orgs[orgID]; ok {
					orgs = append(orgs, org)
				}
				break
			}
		}
	}
	return orgs, nil
}

// WithSession adds a session to the context for testing authenticated requests.
// This is useful for testing handlers that require authentication.
func (m *Mock) WithSession(ctx context.Context, sessionID string) context.Context {
	session, err := m.GetSession(sessionID)
	if err != nil {
		m.t.Fatalf("failed to get session: %v", err)
	}

	ctx = context.WithValue(ctx, "session", session)
	ctx = context.WithValue(ctx, "user_id", session.UserID.String())

	return ctx
}

// WithUser adds a user to the context for testing.
func (m *Mock) WithUser(ctx context.Context, userID string) context.Context {
	user, err := m.GetUser(userID)
	if err != nil {
		m.t.Fatalf("failed to get user: %v", err)
	}

	ctx = context.WithValue(ctx, "user", user)
	ctx = context.WithValue(ctx, "user_id", user.ID.String())

	return ctx
}

// WithOrg adds an organization to the context for testing.
func (m *Mock) WithOrg(ctx context.Context, orgID string) context.Context {
	org, err := m.GetOrganization(orgID)
	if err != nil {
		m.t.Fatalf("failed to get organization: %v", err)
	}

	ctx = context.WithValue(ctx, "organization", org)
	ctx = context.WithValue(ctx, "org_id", org.ID.String())

	return ctx
}

// NewTestContext creates a fully authenticated context with user, org, and session.
// This is a convenience method that combines WithSession, WithUser, and WithOrg.
func (m *Mock) NewTestContext() context.Context {
	user := m.CreateUser("test@example.com", "Test User")
	session := m.CreateSession(user.ID.String(), m.defaultOrg.ID.String())

	ctx := context.Background()
	ctx = m.WithSession(ctx, session.ID.String())
	ctx = m.WithUser(ctx, user.ID.String())
	ctx = m.WithOrg(ctx, m.defaultOrg.ID.String())

	return ctx
}

// NewTestContextWithUser creates an authenticated context for a specific user.
func (m *Mock) NewTestContextWithUser(user *schema.User) context.Context {
	session := m.CreateSession(user.ID.String(), m.defaultOrg.ID.String())

	ctx := context.Background()
	ctx = m.WithSession(ctx, session.ID.String())
	ctx = m.WithUser(ctx, user.ID.String())
	ctx = m.WithOrg(ctx, m.defaultOrg.ID.String())

	return ctx
}

// Reset clears all test data. Useful for cleanup between tests.
func (m *Mock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Recreate default org
	m.defaultOrg = &schema.Organization{
		ID:       xid.New(),
		Name:     "Test Organization",
		Slug:     "test-org",
		Metadata: map[string]interface{}{},
	}

	m.users = make(map[string]*schema.User)
	m.sessions = make(map[string]*schema.Session)
	m.orgs = map[string]*schema.Organization{m.defaultOrg.ID.String(): m.defaultOrg}
	m.members = make(map[string][]*schema.Member)
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

	s.mock.users[user.ID.String()] = user
	return user, nil
}

func (s *MockUserService) GetByID(ctx context.Context, userID string) (*schema.User, error) {
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

func (s *MockUserService) Update(ctx context.Context, userID string, req *user.UpdateUserRequest) (*schema.User, error) {
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

func (s *MockUserService) Delete(ctx context.Context, userID string) error {
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
	// For testing, we just use the userID as string
	return s.mock.CreateSession(req.UserID.String(), ""), nil
}

func (s *MockSessionService) GetByID(ctx context.Context, sessionID string) (*schema.Session, error) {
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
	return nil, fmt.Errorf("session not found with token")
}

func (s *MockSessionService) Delete(ctx context.Context, sessionID string) error {
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
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// MockOrganizationService implements core organization service methods for testing.
type MockOrganizationService struct {
	mock *Mock
}

func (s *MockOrganizationService) Create(ctx context.Context, req *organization.CreateOrganizationRequest) (*schema.Organization, error) {
	return s.mock.CreateOrganization(req.Name, req.Slug), nil
}

func (s *MockOrganizationService) GetByID(ctx context.Context, orgID string) (*schema.Organization, error) {
	return s.mock.GetOrganization(orgID)
}

func (s *MockOrganizationService) GetBySlug(ctx context.Context, slug string) (*schema.Organization, error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	for _, org := range s.mock.orgs {
		if org.Slug == slug {
			return org, nil
		}
	}
	return nil, fmt.Errorf("organization not found with slug: %s", slug)
}

func (s *MockOrganizationService) AddMember(ctx context.Context, userID xid.ID, orgID xid.ID, role string) (*schema.Member, error) {
	return s.mock.AddUserToOrg(userID.String(), orgID.String(), role), nil
}

func (s *MockOrganizationService) GetMembers(ctx context.Context, orgID string) ([]*schema.Member, error) {
	s.mock.mu.RLock()
	defer s.mock.mu.RUnlock()

	return s.mock.members[orgID], nil
}

func (s *MockOrganizationService) GetUserOrganizations(ctx context.Context, userID string) ([]*schema.Organization, error) {
	return s.mock.GetUserOrgs(userID)
}
